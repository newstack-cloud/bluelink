package container

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/require"
)

// Regression test for a deploy-time "resource not found" state error that
// occurs when a dependant references a computed property of a resource it is
// also linked to via a link with no priority resource
// (provider.LinkPriorityResourceNone).
//
// The dependency computation for deployment ordering short-circuited on the
// link relationship: when the link priority did not make the linked-to
// resource a dependency, the reference between the two resources was never
// considered. The dependant (eventsRule) was therefore considered free of any
// dependency on the linked resource (syncFunction) and was deployed as soon
// as another dependency (fastFunction) completed. Resolving
// ${resources.syncFunction.spec.id} (a computed property) then read the
// resource's state before it had been persisted, surfacing
// `StateError: resource "instance:<id>:resource:syncFunction" not found`
// through the deploy error channel.
func TestDeployWaitsForReferencedLinkedResourceWithNoPriority(t *testing.T) {
	releaseGate := make(chan struct{})
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &gatedDeployResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
					gatedResourceName:       "slowBase",
					gate:                    releaseGate,
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
		"__testdata/container/deploy/blueprint7.yml",
		params,
	)
	require.NoError(t, err)

	deployChanges, err := stageChangesForTest(
		context.Background(),
		blueprintContainer,
		params,
	)
	require.NoError(t, err)

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "LinkRefNoPriorityInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	require.NoError(t, err)

	finishedMessage := (*DeploymentFinishedMessage)(nil)
	gateReleased := false
	for err == nil && finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			if msg.ResourceName == "fastFunction" &&
				msg.PreciseStatus == core.PreciseResourceStatusCreated &&
				!gateReleased {
				// fastFunction has fully completed which triggers an
				// evaluation of the events rule's readiness. Release the
				// gate so syncFunction (and then eventsRule) can complete.
				close(releaseGate)
				gateReleased = true
			}
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
	if !gateReleased {
		// Always release the gate so the deployment goroutines
		// are not leaked when the test fails before the release point.
		close(releaseGate)
	}
	require.NoError(t, err)
	require.NotNil(t, finishedMessage)
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		finishedMessage.Status,
		fmt.Sprintf("deployment failed: %v", finishedMessage.FailureReasons),
	)
}

func stageChangesForTest(
	ctx context.Context,
	blueprintContainer BlueprintContainer,
	params core.BlueprintParams,
) (*changes.BlueprintChanges, error) {
	changeStagingChannels := createChangeStagingChannels()
	err := blueprintContainer.StageChanges(
		ctx,
		&StageChangesInput{},
		changeStagingChannels,
		params,
	)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-changeStagingChannels.ChildChangesChan:
		case <-changeStagingChannels.LinkChangesChan:
		case <-changeStagingChannels.ResourceChangesChan:
		case changeSet := <-changeStagingChannels.CompleteChan:
			return &changeSet, nil
		case err := <-changeStagingChannels.ErrChan:
			return nil, err
		case <-time.After(defaultDrainTimeout):
			return nil, errors.New(timeoutMessage)
		}
	}
}

// A minimal events rule resource used to reproduce deployment ordering
// behaviour for a resource that links to a lambda2 function.
// It reuses the lambda2 function test implementation for behaviour that
// is not specific to the rule resource type.
type eventsRule2Resource struct {
	*internal.Lambda2FunctionResource
}

func (r *eventsRule2Resource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/events2/rule",
	}, nil
}

func (r *eventsRule2Resource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/lambda2/function"},
	}, nil
}

func (r *eventsRule2Resource) GetSpecDefinition(
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
					"targetArn": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"otherArn": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

// A link between an events rule and a lambda2 function that mirrors the shape
// of provider links that model the relationship inline in the rule's spec:
// the link is soft with no priority resource, the ordering between the two
// resources is expected to come from the reference in the rule's spec.
type testNoPriorityRuleLambda2Link struct{}

func (l *testNoPriorityRuleLambda2Link) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResourceNone,
	}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/events2/rule::aws/lambda2/function",
	}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindSoft,
	}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testNoPriorityRuleLambda2Link) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{}, nil
}

func (l *testNoPriorityRuleLambda2Link) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}
