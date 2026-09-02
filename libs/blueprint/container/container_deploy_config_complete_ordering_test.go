package container

import (
	"context"
	"errors"
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

// A resource's spec must reach the state container before the resource is marked
// config complete, since that mark is what releases dependants to read it.
//
// A dependant resolves computed properties, which are absent from the blueprint,
// by reading the dependency's persisted state. A new resource has a placeholder
// row saved before its provider is invoked, carrying no spec at all, so a read
// taken before the spec is persisted does not fall back to a default, it fails
// with the property missing from state.
//
// Readiness evaluations run concurrently in goroutines the deploy event loop
// does not wait on, so marking the element before persisting its spec publishes
// "ready to be read from" while there is still nothing to read. This test parks
// one of those evaluations, then holds the producer's save open, so the window
// is entered deliberately rather than by chance.
func TestDeployPersistsResourceSpecBeforeMarkingConfigComplete(t *testing.T) {
	evaluatorParked := make(chan struct{})
	releaseEvaluator := make(chan struct{})
	producerGate := make(chan struct{})
	producerSaving := make(chan struct{})
	releaseProducerSave := make(chan struct{})

	consumer := &orderingConsumerResource{
		unlinkedConsumerResource: &unlinkedConsumerResource{
			Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
		},
		parked:  evaluatorParked,
		release: releaseEvaluator,
	}
	stateContainer := newGatedSaveStateContainer(
		memstate.NewMemoryStateContainer(),
		"orderingProducer",
		producerSaving,
		releaseProducerSave,
	)
	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &gatedDeployResource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
					gatedResourceName:       "orderingProducer",
					gate:                    producerGate,
				},
				"aws/consumer2/record": consumer,
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
		"__testdata/container/deploy/blueprint-config-complete-ordering.yml",
		params,
	)
	require.NoError(t, err)

	deployChanges, err := stageChangesForTest(
		context.Background(),
		blueprintContainer,
		params,
	)
	require.NoError(t, err)

	// Only park a readiness evaluation once the deployment itself is under way,
	// so that change staging's own calls are left alone.
	consumer.arm()

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "ConfigCompleteOrderingInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	require.NoError(t, err)

	sequenceDone := make(chan struct{})
	go func() {
		defer close(sequenceDone)

		// The trigger completing spawns the evaluation, which parks in the
		// consumer's stabilised dependencies call, immediately before it reads
		// whether the producer is ready.
		select {
		case <-evaluatorParked:
		case <-time.After(defaultDrainTimeout):
			return
		}

		// With an evaluation held in that position, let the producer run through
		// to its config complete handling.
		close(producerGate)

		select {
		case <-producerSaving:
		case <-time.After(defaultDrainTimeout):
			return
		}

		// The producer's spec is now in flight to the state container and has not
		// landed. Releasing the evaluation here is the moment we want to capture, it
		// reads the producer's readiness while the spec is still absent.
		consumer.releaseIfHeld()

		// Give the released evaluation time to act on what it read before the
		// save is allowed to complete. This bounds how long the window is held
		// open, it is not what makes the outcome deterministic.
		time.Sleep(200 * time.Millisecond)
		stateContainer.releaseIfHeld()
	}()

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

	consumer.releaseIfHeld()
	stateContainer.releaseIfHeld()
	<-sequenceDone

	require.NoError(t, err)
	require.NotNil(t, finishedMessage)
	require.Equal(
		t,
		core.InstanceStatusDeployed,
		finishedMessage.Status,
		fmt.Sprintf("deployment failed: %v", finishedMessage.FailureReasons),
	)
}

// Parks the first readiness evaluation that reaches it once armed, holding the
// evaluation immediately before it checks whether its dependencies are ready.
type orderingConsumerResource struct {
	*unlinkedConsumerResource
	parked   chan struct{}
	release  chan struct{}
	mu       sync.Mutex
	armed    bool
	held     bool
	released bool
}

func (r *orderingConsumerResource) arm() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.armed = true
}

func (r *orderingConsumerResource) releaseIfHeld() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.held && !r.released {
		r.released = true
		close(r.release)
	}
}

func (r *orderingConsumerResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	r.mu.Lock()
	shouldPark := r.armed && !r.held
	if shouldPark {
		r.held = true
	}
	r.mu.Unlock()

	if shouldPark {
		close(r.parked)
		select {
		case <-r.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

// Holds the save that carries the named resource's spec, so a dependant can be
// released to read the resource while the spec has not yet been persisted.
type gatedSaveStateContainer struct {
	state.Container
	resources *gatedSaveResources
}

func newGatedSaveStateContainer(
	inner state.Container,
	gatedResourceName string,
	saving chan struct{},
	release chan struct{},
) *gatedSaveStateContainer {
	return &gatedSaveStateContainer{
		Container: inner,
		resources: &gatedSaveResources{
			ResourcesContainer: inner.Resources(),
			gatedResourceName:  gatedResourceName,
			saving:             saving,
			release:            release,
		},
	}
}

func (c *gatedSaveStateContainer) Resources() state.ResourcesContainer {
	return c.resources
}

func (c *gatedSaveStateContainer) releaseIfHeld() {
	c.resources.releaseIfHeld()
}

type gatedSaveResources struct {
	state.ResourcesContainer
	gatedResourceName string
	saving            chan struct{}
	release           chan struct{}
	mu                sync.Mutex
	held              bool
	released          bool
}

func (r *gatedSaveResources) releaseIfHeld() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.held && !r.released {
		r.released = true
		close(r.release)
	}
}

func (r *gatedSaveResources) Save(
	ctx context.Context,
	resourceState state.ResourceState,
) error {
	// The placeholder row saved before the provider runs carries no spec, so
	// gating on the spec being present picks out the save that publishes it.
	carriesSpec := resourceState.Name == r.gatedResourceName &&
		resourceState.SpecData != nil

	r.mu.Lock()
	shouldHold := carriesSpec && !r.held
	if shouldHold {
		r.held = true
	}
	r.mu.Unlock()

	if shouldHold {
		close(r.saving)
		select {
		case <-r.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return r.ResourcesContainer.Save(ctx, resourceState)
}
