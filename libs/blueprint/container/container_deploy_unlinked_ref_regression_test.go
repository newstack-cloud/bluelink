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

// A resource that references a computed property of a resource it has no link
// to must wait for that resource to be created, with the ordering coming from
// the reference alone.
//
// Observed against a real deployment where an ElastiCache subnet group whose spec
// carries ${resources.<vpc>.spec.privateSubnetIds} was resolved while the VPC
// was still being created, several minutes from finishing because of its NAT
// gateway. A computed property is absent from the blueprint spec and is only
// readable from state once the producer has been deployed, so resolution
// failed with "missing property ... in spec definition" and took the whole
// deployment down.
//
// Unlike the linked case covered elsewhere, there is no link between these two
// resources at all, so nothing but the reference can establish the ordering.
func TestDeployWaitsForUnlinkedReferencedResource(t *testing.T) {
	releaseGate := make(chan struct{})
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &gatedDeployResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
					gatedResourceName:       "gatedProducer",
					gate:                    releaseGate,
				},
				"aws/consumer2/record": &unlinkedConsumerResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
			},
			Links:               map[string]provider.Link{},
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
		"__testdata/container/deploy/blueprint-stabilise-ref.yml",
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
			InstanceName: "UnlinkedRefInstance",
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
				// The fast function completing prompts a readiness evaluation
				// for the consumer while gatedProducer has not been created.
				// Releasing here lets the deployment finish if, and only if,
				// the consumer correctly waited.
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
		// Always release the gate so the deployment goroutines are not leaked
		// when the test fails before the release point.
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

// A resource expanded from a template must also wait for a resource it
// references but has no link to.
//
// Deployment nodes are grouped from the expanded blueprint, where a templated
// resource appears as consumer_0 and consumer_1. Direct dependencies were
// populated from the container's own collector instead, which only knows the
// unexpanded "consumer" and therefore has no entry for either node. With no
// entry, no references are seen, no dependencies are recorded, and nothing
// blocks the nodes from deploying before the resource they read from.
func TestDeployWaitsForReferencedResourceFromTemplatedResource(t *testing.T) {
	releaseGate := make(chan struct{})
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &gatedDeployResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
					gatedResourceName:       "gatedProducer",
					gate:                    releaseGate,
				},
				"aws/consumer2/record": &unlinkedConsumerResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
			},
			Links:               map[string]provider.Link{},
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
		"__testdata/container/deploy/blueprint-template-ref.yml",
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
			InstanceName: "TemplatedRefInstance",
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

// A resource that references another resource without linking to it.
//
// It reuses the lambda2 function implementation for behaviour that is not
// specific to the type, and deliberately declares no link targets so that the
// only relationship with the referenced resource is the reference itself.
type unlinkedConsumerResource struct {
	*internal.Lambda2FunctionResource
}

func (r *unlinkedConsumerResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/consumer2/record",
	}, nil
}

func (r *unlinkedConsumerResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *unlinkedConsumerResource) GetSpecDefinition(
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
					"sourceId": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"otherId": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}
