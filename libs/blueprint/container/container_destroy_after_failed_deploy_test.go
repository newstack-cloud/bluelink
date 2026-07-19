package container

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/require"
)

// Regression test for destroys after failed deploys ending in
// `DESTROY FAILED: ... 0 failures N elements were interrupted (state unknown)`.
// A deploy that fails via the error channel leaves partial state behind
// (skeleton resource rows, links that were never created, elements that never
// started). A subsequent destroy staged via StageChanges(Destroy: true) must
// run to completion for the elements that do have state and finish with a
// Destroyed status.
func TestDestroyAfterFailedDeployEndsInDestroyedStatus(t *testing.T) {
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &rawFailDeployResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
					failResourceName:        "badFunction",
				},
				"aws/events2/rule": &eventsRule2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
			},
			Links: map[string]provider.Link{
				"aws/events2/rule::aws/lambda2/function": &testNoPriorityRuleLambda2Link{},
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
		"__testdata/container/deploy/blueprint8.yml",
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
			InstanceName: "DestroyAfterFailedDeployInstance",
			Changes:      deployChanges,
			Rollback:     false,
			DrainTimeout: 500 * time.Millisecond,
		},
		deployChannels,
		params,
	)
	require.NoError(t, err)

	// The deploy is expected to fail, either via a finish message with a
	// failed status or via the error channel (an unwrapped provider error is
	// treated as fatal and surfaces on the error channel).
	deployFailedViaErrChan := false
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for !deployFailedViaErrChan && finishedMessage == nil {
		select {
		case <-deployChannels.ResourceUpdateChan:
		case <-deployChannels.ChildUpdateChan:
		case <-deployChannels.LinkUpdateChan:
		case msg := <-deployChannels.FinishChan:
			finishedMessage = &msg
		case <-deployChannels.DeploymentUpdateChan:
		case <-deployChannels.ErrChan:
			deployFailedViaErrChan = true
		case <-time.After(defaultDrainTimeout):
			require.FailNow(t, timeoutMessage)
		}
	}
	if finishedMessage != nil {
		require.NotEqual(t, core.InstanceStatusDeployed, finishedMessage.Status)
	}

	instances := stateContainer.Instances()
	instanceID, err := instances.LookupIDByName(
		context.Background(),
		"DestroyAfterFailedDeployInstance",
	)
	require.NoError(t, err)

	// Wait for the in-progress claim to be released (the release runs
	// asynchronously when a deploy exits via the error channel).
	require.Eventually(t, func() bool {
		instanceState, err := instances.Get(context.Background(), instanceID)
		return err == nil && instanceState.Status == core.InstanceStatusDeployFailed
	}, 5*time.Second, 10*time.Millisecond)

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

// A lambda2 function resource that fails deployment for a specific resource
// with an error that is not wrapped in a provider error type,
// causing the deployment process to exit via the error channel.
type rawFailDeployResource struct {
	*internal.Lambda2FunctionResource
	failResourceName string
}

func (r *rawFailDeployResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	if input.Changes.AppliedResourceInfo.ResourceName == r.failResourceName {
		return nil, errors.New("unexpected failure deploying resource")
	}
	return r.Lambda2FunctionResource.Deploy(ctx, input)
}
