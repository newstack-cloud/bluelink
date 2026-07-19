package container

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/require"
)

// Regression test for resources in changes.RemovedResources never being
// removed during an update deploy.
//
// The scenario mirrors a live run: an instance is first deployed from a
// blueprint containing a function linked to an "old" function
// (syncFunction::oldFunction). An updated blueprint drops oldFunction
// entirely while modifying syncFunction and adding a new resource
// (jobsFunction) together with a new link syncFunction::jobsFunction.
// Change staging places oldFunction in RemovedResources and the old link in
// RemovedLinks. The update deploy must destroy the removed resource with its
// provider implementation and prune it (and the removed link) from the
// persisted instance state before deploying the rest of the changes.
func TestUpdateDeployRemovesResourcesDroppedFromBlueprint(t *testing.T) {
	preReleasedGate := make(chan struct{})
	close(preReleasedGate)
	oldFunctionResource := &old2FunctionResource{
		Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
	}
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &jobsLinkingLambda2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
				"aws/old2/function": oldFunctionResource,
				"aws/jobs2/function": &jobs2FunctionResource{
					gatedDeployResource: &gatedDeployResource{
						Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
						gatedResourceName:       "jobsFunction",
						gate:                    preReleasedGate,
					},
				},
			},
			Links: map[string]provider.Link{
				"aws/lambda2/function::aws/old2/function":  &testLambda2ToOld2Link{},
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

	instanceID := deployInitialInstanceForRemovalTest(t, loader, params)

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-remove-on-update-update.yml",
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
	// the dropped resource and its link are staged for removal, the source
	// function is modified and the new resource and link are staged as new.
	require.Contains(t, updateChanges.RemovedResources, "oldFunction")
	require.Contains(t, updateChanges.RemovedLinks, "syncFunction::oldFunction")
	require.Contains(t, updateChanges.ResourceChanges, "syncFunction")
	require.Contains(t, updateChanges.NewResources, "jobsFunction")

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

	finishedMessage := consumeUntilFinishForTest(t, updateChannels, "update deploy")
	require.Equal(
		t,
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)
	require.Empty(t, finishedMessage.FailureReasons)

	require.Equal(
		t,
		[]string{"oldFunction"},
		oldFunctionResource.destroyedResourceNames(),
		"the provider Destroy implementation must be called for the removed resource",
	)

	postDeployState, err := stateContainer.Instances().Get(
		context.Background(),
		instanceID,
	)
	require.NoError(t, err)
	postDeployResourceNames := resourceNamesFromInstanceState(&postDeployState)
	require.NotContains(
		t,
		postDeployResourceNames,
		"oldFunction",
		"the removed resource must be pruned from the persisted instance state",
	)
	require.Contains(t, postDeployResourceNames, "syncFunction")
	require.Contains(t, postDeployResourceNames, "jobsFunction")
	require.NotContains(t, postDeployState.Links, "syncFunction::oldFunction")
	require.Contains(t, postDeployState.Links, "syncFunction::jobsFunction")
}

func deployInitialInstanceForRemovalTest(
	t *testing.T,
	loader Loader,
	params core.BlueprintParams,
) string {
	t.Helper()

	initialContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-remove-on-update-initial.yml",
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
			InstanceName: "RemoveOnUpdateInstance",
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

func resourceNamesFromInstanceState(instanceState *state.InstanceState) []string {
	resourceNames := []string{}
	for _, resourceState := range instanceState.Resources {
		resourceNames = append(resourceNames, resourceState.Name)
	}
	return resourceNames
}

// A function resource type for the removed resource that records the names of
// the resources it has been asked to destroy.
type old2FunctionResource struct {
	*internal.Lambda2FunctionResource
	mu        sync.Mutex
	destroyed []string
}

func (r *old2FunctionResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/old2/function",
	}, nil
}

func (r *old2FunctionResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	resourceName := ""
	if input.ResourceState != nil {
		resourceName = input.ResourceState.Name
	}
	r.destroyed = append(r.destroyed, resourceName)
	return nil
}

func (r *old2FunctionResource) destroyedResourceNames() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	namesCopy := make([]string, len(r.destroyed))
	copy(namesCopy, r.destroyed)
	return namesCopy
}

// A link between a lambda2 function and an old2 function with the same
// no-priority soft shape as the other test links.
type testLambda2ToOld2Link struct {
	testNoPriorityRuleLambda2Link
}

func (l *testLambda2ToOld2Link) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda2/function::aws/old2/function",
	}, nil
}
