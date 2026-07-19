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

// Regression test for a new outbound link between a modified resource and a
// new resource never being scheduled during an update deploy.
//
// The scenario: an instance is first deployed from a blueprint containing a
// rule linked to a function (nightlyRule::syncFunction). An updated blueprint
// modifies both existing resources and adds a new resource (jobsFunction)
// together with a new link syncFunction::jobsFunction. Change staging places
// the new link in ResourceChanges["syncFunction"].NewOutboundLinks and
// deliberately excludes the unchanged nightlyRule::syncFunction link from
// OutboundLinkChanges.
//
// Previously, the deploy-side pending link tracking scheduled every link
// adjacent to a completing resource regardless of the staged change set, so
// the unchanged link was deployed without being counted as an element to
// deploy. The extra completion let the deployment event loop exit as soon as
// the last counted element (the gated jobsFunction) completed, which was before the
// new link had a chance to start. The new link was then classified as failed
// with zero link update messages emitted for it, producing a DeployFailed
// finish message with `failed to deploy "link(syncFunction::jobsFunction)"`
// as the only diagnostic.
func TestUpdateDeploySchedulesNewLinkBetweenModifiedAndNewResource(t *testing.T) {
	releaseGate := make(chan struct{})
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/events2/rule": &eventsRule2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
				"aws/lambda2/function": &jobsLinkingLambda2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
				"aws/jobs2/function": &jobs2FunctionResource{
					gatedDeployResource: &gatedDeployResource{
						Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
						gatedResourceName:       "jobsFunction",
						gate:                    releaseGate,
					},
				},
			},
			Links: map[string]provider.Link{
				"aws/events2/rule::aws/lambda2/function":   &testNoPriorityRuleLambda2Link{},
				"aws/lambda2/function::aws/jobs2/function": &testLambda2ToJobs2Link{},
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

	instanceID := deployInitialInstanceForNewLinkTest(t, loader, params)

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-new-link-update.yml",
		params,
	)
	require.NoError(t, err)

	updateChangeStagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: instanceID,
		},
		updateChangeStagingChannels,
		params,
	)
	require.NoError(t, err)

	updateChanges, err := consumeStagedChangesForTest(updateChangeStagingChannels)
	require.NoError(t, err)

	// Guard the shape of the staged changes that this regression depends on:
	// both existing resources are modified, the new link lives in the modified
	// source resource's NewOutboundLinks, the link target is a new resource
	// and the unchanged existing link is excluded from OutboundLinkChanges.
	syncFunctionChanges, hasSyncFunctionChanges := updateChanges.ResourceChanges["syncFunction"]
	require.True(t, hasSyncFunctionChanges, "syncFunction must be staged as a modified resource")
	require.Contains(t, syncFunctionChanges.NewOutboundLinks, "jobsFunction")
	require.Contains(t, updateChanges.ResourceChanges, "nightlyRule")
	require.Contains(t, updateChanges.NewResources, "jobsFunction")
	nightlyRuleChanges := updateChanges.ResourceChanges["nightlyRule"]
	require.NotContains(t, nightlyRuleChanges.OutboundLinkChanges, "syncFunction")

	updateChannels := CreateDeployChannels()
	err = updatedContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: instanceID,
			Changes:    updateChanges,
			Rollback:   false,
		},
		updateChannels,
		params,
	)
	require.NoError(t, err)

	newLinkMessages, finishedMessage, deployErr := consumeUpdateDeployForNewLinkTest(
		updateChannels,
		releaseGate,
	)
	require.NoError(t, deployErr)
	require.NotNil(t, finishedMessage)
	require.Equal(
		t,
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)
	require.Empty(t, finishedMessage.FailureReasons)
	require.NotEmpty(
		t,
		newLinkMessages,
		"link update messages must be emitted for the new link between the "+
			"modified resource and the new resource",
	)
	lastLinkMessage := newLinkMessages[len(newLinkMessages)-1]
	require.Equal(t, core.LinkStatusCreated, lastLinkMessage.Status)

	linkState, err := stateContainer.Links().GetByName(
		context.Background(),
		instanceID,
		"syncFunction::jobsFunction",
	)
	require.NoError(t, err)
	require.Equal(t, core.LinkStatusCreated, linkState.Status)
}

// Consumes deploy channels until the finish message arrives, holding back the
// gated jobsFunction resource until the rest of the update has settled.
// The gate is released as soon as the unchanged nightlyRule::syncFunction
// link reports completion (reproducing the uncounted extra completion) or,
// when that link is not redeployed at all, shortly after both modified
// resources have completed.
func consumeUpdateDeployForNewLinkTest(
	channels *DeployChannels,
	releaseGate chan struct{},
) ([]LinkDeployUpdateMessage, *DeploymentFinishedMessage, error) {
	newLinkMessages := []LinkDeployUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var deployErr error

	completedResources := map[string]bool{}
	gateReleased := false
	releaseGateOnce := func() {
		if !gateReleased {
			close(releaseGate)
			gateReleased = true
		}
	}
	defer releaseGateOnce()

	var settleTimer <-chan time.Time
	for deployErr == nil && finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			if msg.PreciseStatus == core.PreciseResourceStatusUpdated {
				completedResources[msg.ResourceName] = true
			}
			if completedResources["nightlyRule"] &&
				completedResources["syncFunction"] &&
				settleTimer == nil {
				settleTimer = time.After(500 * time.Millisecond)
			}
		case <-channels.ChildUpdateChan:
		case msg := <-channels.LinkUpdateChan:
			if msg.LinkName == "syncFunction::jobsFunction" {
				newLinkMessages = append(newLinkMessages, msg)
			}
			if msg.LinkName == "nightlyRule::syncFunction" &&
				msg.Status == core.LinkStatusUpdated {
				releaseGateOnce()
			}
		case <-settleTimer:
			releaseGateOnce()
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case deployErr = <-channels.ErrChan:
		case <-time.After(defaultDrainTimeout):
			deployErr = errors.New(timeoutMessage)
		}
	}

	return newLinkMessages, finishedMessage, deployErr
}

func deployInitialInstanceForNewLinkTest(
	t *testing.T,
	loader Loader,
	params core.BlueprintParams,
) string {
	t.Helper()

	initialContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-new-link-initial.yml",
		params,
	)
	require.NoError(t, err)

	initialChanges, err := stageChangesForTest(
		context.Background(),
		initialContainer,
		params,
	)
	require.NoError(t, err)

	initialChannels := CreateDeployChannels()
	err = initialContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "NewLinkUpdateInstance",
			Changes:      initialChanges,
			Rollback:     false,
		},
		initialChannels,
		params,
	)
	require.NoError(t, err)

	initialFinished := consumeUntilFinishForTest(t, initialChannels, "initial deploy")
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		initialFinished.Status,
		fmt.Sprintf("initial deploy failed: %v", initialFinished.FailureReasons),
	)

	return initialFinished.InstanceID
}

// A lambda2 function that can link to the jobs2 and old2 function types used
// to model links from a modified resource to new and removed resources.
type jobsLinkingLambda2Resource struct {
	*internal.Lambda2FunctionResource
}

func (r *jobsLinkingLambda2Resource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/jobs2/function", "aws/old2/function"},
	}, nil
}

// A function resource type for the new link target, with a deploy that can be
// gated by the test so the new resource is always the last element to
// complete in the update deploy.
type jobs2FunctionResource struct {
	*gatedDeployResource
}

func (r *jobs2FunctionResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/jobs2/function",
	}, nil
}

// A link between a lambda2 function and a jobs2 function with the same
// no-priority soft shape as the other test links.
type testLambda2ToJobs2Link struct {
	testNoPriorityRuleLambda2Link
}

func (l *testLambda2ToJobs2Link) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda2/function::aws/jobs2/function",
	}, nil
}
