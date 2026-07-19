package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/require"
)

// Regression test for destruction of successfully deployed instances ending in
// `DESTROY FAILED: ... 0 failures N elements were interrupted (state unknown)`.
// It drives the same flow an external caller does: deploy a new instance,
// stage changes for a destroy and then destroy the instance, consuming all
// deploy channels until the finish message arrives.
func TestDeployThenDestroyEndsInDestroyedStatus(t *testing.T) {
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ true,
			/* skipRetryFailuresForLinkNames */ []string{
				"saveOrderFunction::ordersTable_0",
				"saveOrderFunction::ordersTable_2",
			},
			stateContainer,
		),
		"example": newTestExampleProvider(),
		"core": providerhelpers.NewCoreProvider(
			stateContainer.Links(),
			core.BlueprintInstanceIDFromContext,
			os.Getwd,
			provider.NewFileSourceRegistry(),
			core.SystemClock{},
		),
	}
	loader := NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(&ResourceStabilityPollingConfig{
			PollingInterval: 10 * time.Millisecond,
			PollingTimeout:  1 * time.Second,
		}),
		WithLoaderLogger(core.NewNopLogger()),
	)
	params := blueprint1DeployParams( /* includeInvoices */ true)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint2.jsonc",
		params,
	)
	require.NoError(t, err)

	deployChanges, err := stageChangesForTest(
		context.Background(),
		blueprintContainer,
		params,
	)
	require.NoError(t, err)

	deployChannels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "DeployDestroyCycleInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	require.NoError(t, err)

	deployFinished := consumeUntilFinishForTest(t, deployChannels, "deploy")
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		deployFinished.Status,
		fmt.Sprintf("deploy failed: %v", deployFinished.FailureReasons),
	)
	instanceID := deployFinished.InstanceID

	destroyChangeStagingChannels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: instanceID,
			Destroy:    true,
		},
		destroyChangeStagingChannels,
		params,
	)
	require.NoError(t, err)

	destroyChanges, err := consumeStagedChangesForTest(destroyChangeStagingChannels)
	require.NoError(t, err)

	destroyChannels := CreateDeployChannels()
	blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID: instanceID,
			Changes:    destroyChanges,
			Rollback:   false,
		},
		destroyChannels,
		params,
	)

	destroyFinished := consumeUntilFinishForTest(t, destroyChannels, "destroy")
	require.Equal(
		t,
		core.InstanceStatusDestroyed,
		destroyFinished.Status,
		fmt.Sprintf("destroy failed: %v", destroyFinished.FailureReasons),
	)
	require.Empty(t, destroyFinished.FailureReasons)
}

// Regression test for the destroy drain finish message swallowing the error
// that triggered the drain. An error that is not attributed to a specific
// element (e.g. an unwrapped error from a provider link implementation)
// previously produced `... due to 0 failures` + `N elements were interrupted
// (state unknown)` with no trace of the underlying error, making the failure
// impossible to diagnose from the finish message. The finish message must
// include the drain-triggering error and name the interrupted elements.
func TestDestroyDrainFinishMessageIncludesTriggeringError(t *testing.T) {
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &internal.Lambda2FunctionResource{},
				"aws/events2/rule": &eventsRule2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
			},
			Links: map[string]provider.Link{
				"aws/events2/rule::aws/lambda2/function": &failOnDestroyRuleLambda2Link{},
			},
			CustomVariableTypes: map[string]provider.CustomVariableType{},
			DataSources:         map[string]provider.DataSource{},
		},
	}
	loader := NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(&ResourceStabilityPollingConfig{
			PollingInterval: 10 * time.Millisecond,
			PollingTimeout:  1 * time.Second,
		}),
		WithLoaderLogger(core.NewNopLogger()),
	)
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint9.yml",
		params,
	)
	require.NoError(t, err)

	deployChanges, err := stageChangesForTest(
		context.Background(),
		blueprintContainer,
		params,
	)
	require.NoError(t, err)

	deployChannels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "DestroyDrainErrorInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	require.NoError(t, err)

	deployFinished := consumeUntilFinishForTest(t, deployChannels, "deploy")
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		deployFinished.Status,
		fmt.Sprintf("deploy failed: %v", deployFinished.FailureReasons),
	)
	instanceID := deployFinished.InstanceID

	destroyChangeStagingChannels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: instanceID,
			Destroy:    true,
		},
		destroyChangeStagingChannels,
		params,
	)
	require.NoError(t, err)

	destroyChanges, err := consumeStagedChangesForTest(destroyChangeStagingChannels)
	require.NoError(t, err)

	destroyChannels := CreateDeployChannels()
	blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID:   instanceID,
			Changes:      destroyChanges,
			Rollback:     false,
			DrainTimeout: 500 * time.Millisecond,
		},
		destroyChannels,
		params,
	)

	destroyFinished := consumeUntilFinishForTest(t, destroyChannels, "destroy")
	require.NotEqual(t, core.InstanceStatusDestroyed, destroyFinished.Status)
	allFailureReasons := strings.Join(destroyFinished.FailureReasons, "\n")
	require.Contains(
		t,
		allFailureReasons,
		failOnDestroyLinkErrorMessage,
		"the destroy finish message must include the error that triggered the drain",
	)
	require.Contains(
		t,
		allFailureReasons,
		"eventsRule::goodFunction",
		"the destroy finish message must name the interrupted element",
	)
}

const failOnDestroyLinkErrorMessage = "unexpected failure removing the link relationship"

// A link that deploys successfully but fails with an unwrapped error when
// the link is being destroyed, emulating a provider link implementation
// returning an error that is not wrapped in a provider error type.
type failOnDestroyRuleLambda2Link struct {
	testNoPriorityRuleLambda2Link
}

func (l *failOnDestroyRuleLambda2Link) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	if input.LinkUpdateType == provider.LinkUpdateTypeDestroy {
		return nil, errors.New(failOnDestroyLinkErrorMessage)
	}
	return l.testNoPriorityRuleLambda2Link.UpdateResourceA(ctx, input)
}

func consumeUntilFinishForTest(
	t *testing.T,
	channels *DeployChannels,
	operation string,
) *DeploymentFinishedMessage {
	t.Helper()
	var err error
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil && finishedMessage == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(defaultDrainTimeout):
			err = errors.New(timeoutMessage)
		}
	}
	require.NoError(t, err, "unexpected error during "+operation)
	require.NotNil(t, finishedMessage)
	return finishedMessage
}

func consumeStagedChangesForTest(
	channels *ChangeStagingChannels,
) (*changes.BlueprintChanges, error) {
	for {
		select {
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case <-channels.ResourceChangesChan:
		case changeSet := <-channels.CompleteChan:
			return &changeSet, nil
		case err := <-channels.ErrChan:
			return nil, err
		case <-time.After(defaultDrainTimeout):
			return nil, errors.New(timeoutMessage)
		}
	}
}
