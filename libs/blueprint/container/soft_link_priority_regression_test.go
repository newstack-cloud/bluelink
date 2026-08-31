package container

import (
	"context"
	"fmt"
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

// A resource that depends on another through a soft link must be deployed, wherever the
// two happen to be grouped.
//
// Ordering only knows about links that are hard, since only those are recorded as
// references, while a dependency comes from the link's priority resource whatever its
// kind. A soft link with a priority is therefore a dependency nothing orders, and the
// resource that depends can be grouped before the one it depends on. It is held at the
// start of the deployment because it is waiting, and dispatch after a completion used to
// look only at later groups, so nothing ever reached back to it.
func TestDeploySoftLinkDependencyGroupedBeforeItsDependency(t *testing.T) {
	stateContainer := memstate.NewMemoryStateContainer()
	loader := newSoftLinkPriorityLoader(stateContainer)
	params := newLinkProjectionParams()

	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-soft-link-priority.yml",
		params,
	)
	require.NoError(t, err)

	stagingChannels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{},
		stagingChannels,
		params,
	)
	require.NoError(t, err)

	stagedChanges, err := consumeStagedChangesForTest(stagingChannels)
	require.NoError(t, err)
	require.Contains(t, stagedChanges.NewResources, "httpApi")

	deployChannels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "SoftLinkPriorityInstance",
			Changes:      stagedChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	require.NoError(t, err)

	finishedMessage := consumeUntilFinishForTest(t, deployChannels, "deploy")
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		finishedMessage.Status,
		fmt.Sprintf("deploy failed: %v", finishedMessage.FailureReasons),
	)

	instance, err := stateContainer.Instances().Get(
		context.Background(),
		finishedMessage.InstanceID,
	)
	require.NoError(t, err)
	require.Contains(
		t,
		instance.ResourceIDs,
		"httpApi",
		"the resource waiting on a soft link dependency was never deployed",
	)
}

func newSoftLinkPriorityLoader(stateContainer state.Container) Loader {
	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	).(*internal.ProviderMock)
	awsProvider.Resources["aws/gateway2/api"] = &gateway2APIResource{
		Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
	}
	awsProvider.Links["aws/gateway2/api::aws/lambda/function"] = &softPriorityLink{}

	return NewDefaultLoader(
		map[string]provider.Provider{"aws": awsProvider},
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
}

// A gateway that links to functions, standing in for any resource that depends on the
// other side of a soft link.
type gateway2APIResource struct {
	*internal.Lambda2FunctionResource
}

func (r *gateway2APIResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/gateway2/api",
	}, nil
}

func (r *gateway2APIResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/lambda/function"},
	}, nil
}

func (r *gateway2APIResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"apiName": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

// A soft link whose priority resource is the one being linked to.
//
// The priority makes the linking resource depend on the resource it links to. The kind
// keeps that dependency out of the reference graph, so nothing orders the two.
type softPriorityLink struct {
	*testApiGatewayLambdaLink
}

func (l *softPriorityLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/gateway2/api::aws/lambda/function",
	}, nil
}
