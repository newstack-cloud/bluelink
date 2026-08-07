package container

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Records how many links were inside their resource A update at the same time.
//
// The first link to arrive waits for a sibling to join it, so an overlap is observed
// rather than inferred from how long anything took. The wait is bounded: a link that is
// genuinely alone in its batch carries on rather than hanging the suite.
type linkConcurrencyRecorder struct {
	mu       sync.Mutex
	inFlight int
	peak     int
	joined   chan struct{}
	once     sync.Once
}

func (r *linkConcurrencyRecorder) enter() {
	r.mu.Lock()
	r.inFlight++
	if r.inFlight > r.peak {
		r.peak = r.inFlight
	}
	joined := r.inFlight >= 2
	r.mu.Unlock()

	if joined {
		r.once.Do(func() { close(r.joined) })
		return
	}

	select {
	case <-r.joined:
	case <-time.After(2 * time.Second):
	}
}

func (r *linkConcurrencyRecorder) leave() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.inFlight -= 1
}

func (r *linkConcurrencyRecorder) peakInFlight() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.peak
}

// Wraps a link so its resource A update is observed, leaving the rest of the
// implementation as the suite already has it.
type concurrencyObservedLink struct {
	provider.Link
	recorder *linkConcurrencyRecorder
}

func (l *concurrencyObservedLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	l.recorder.enter()
	defer l.recorder.leave()

	return l.Link.UpdateResourceA(ctx, input)
}

func observedLinkAWSProvider(
	stateContainer state.Container,
	recorder *linkConcurrencyRecorder,
) provider.Provider {
	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	)
	mock := awsProvider.(*internal.ProviderMock)

	observed := map[string]provider.Link{}
	for linkType, linkImpl := range mock.Links {
		observed[linkType] = &concurrencyObservedLink{Link: linkImpl, recorder: recorder}
	}
	mock.Links = observed

	return mock
}

// Links made ready together are deployed at the same time rather than one after
// another.
//
// Deploying them in series used to stand in for the resource locks that keep two links
// off the same resource. Those locks do that job now, so links that share nothing have
// no reason to wait for one another, and a blueprint whose resources each fan out to
// several links spent most of its deployment idle.
func (s *ContainerDeployTestSuite) Test_links_ready_together_are_deployed_concurrently() {
	recorder := &linkConcurrencyRecorder{joined: make(chan struct{})}
	stateContainer := memstate.NewMemoryStateContainer()

	loader := NewDefaultLoader(
		map[string]provider.Provider{
			"aws":     observedLinkAWSProvider(stateContainer, recorder),
			"example": newTestExampleProvider(),
			"core": providerhelpers.NewCoreProvider(
				stateContainer.Links(),
				core.BlueprintInstanceIDFromContext,
				os.Getwd,
				provider.NewFileSourceRegistry(),
				core.SystemClock{},
			),
		},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)

	params := blueprint1DeployParams( /* includeInvoices */ true)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint1.yml",
		params,
	)
	s.Require().NoError(err)

	deployChanges, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		blueprintContainer,
		params,
	)
	s.Require().NoError(err)

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "LinkConcurrencyInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

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
	s.Require().NoError(err)
	s.Require().NotNil(finishedMessage)

	s.Assert().GreaterOrEqual(
		recorder.peakInFlight(),
		2,
		"links made ready together should have been deployed at the same time",
	)
}
