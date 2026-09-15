package container

import (
	"context"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type LinkSchedulerDrainTestSuite struct {
	suite.Suite
}

// Draining must not wait on a link that never returns.
//
// A link deployment that does not come back can happen when a provider plugin that
// dies mid-call leaves the worker parked on a transport that will never answer. Draining
// is reached after a terminal failure, by which point the deployment's own drain deadline
// has already been spent, so an unbounded wait here is the last thing standing between a
// failed deployment and one that never finishes. The deployment reports nothing further,
// does not roll back, and the stall detector is suppressed because it is already draining.
//
// Giving up on a worker is safe. The caller marks whatever is still in flight as
// interrupted, which is a truthful account of a link whose outcome is unknown, and far
// better than reporting nothing at all.
func (s *LinkSchedulerDrainTestSuite) Test_drains_while_a_link_never_returns() {
	blocked := make(chan struct{})
	defer close(blocked)
	started := make(chan struct{}, 1)

	scheduler := newLinkScheduler(
		NewLinkSlots(DefaultMaxConcurrentLinks),
		func(string) provider.LinkModifies { return provider.LinkModifiesBoth },
		func(context.Context, *LinkPendingCompletion, *state.InstanceState) error {
			select {
			case started <- struct{}{}:
			default:
			}
			// Never returns, which is what a call into a dead plugin looks like.
			<-blocked
			return nil
		},
		func(context.Context) (*state.InstanceState, error) {
			return &state.InstanceState{}, nil
		},
		func(string) {},
		func(string) []string { return nil },
		func(string) bool { return false },
		func(error) bool { return true },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)
	scheduler.Submit([]*LinkPendingCompletion{s.pendingLink("stuck")})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		s.Require().Fail("the link was never dispatched, so nothing is in flight to drain")
	}

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelDrain()

	returned := make(chan struct{})
	go func() {
		scheduler.Drain(drainCtx)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		s.Fail("draining waited on a link that never returns, so the deployment cannot finish")
	}
}

// A drain with nothing stuck still waits for its workers, so bounding the wait must not
// turn draining into a race that abandons links about to report.
func (s *LinkSchedulerDrainTestSuite) Test_waits_for_a_link_that_does_return() {
	release := make(chan struct{})
	finished := make(chan struct{})
	started := make(chan struct{}, 1)

	scheduler := newLinkScheduler(
		NewLinkSlots(DefaultMaxConcurrentLinks),
		func(string) provider.LinkModifies { return provider.LinkModifiesBoth },
		func(context.Context, *LinkPendingCompletion, *state.InstanceState) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			close(finished)
			return nil
		},
		func(context.Context) (*state.InstanceState, error) {
			return &state.InstanceState{}, nil
		},
		func(string) {},
		func(string) []string { return nil },
		func(string) bool { return false },
		func(error) bool { return true },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)
	scheduler.Submit([]*LinkPendingCompletion{s.pendingLink("slow")})

	// The link has to be running before the drain starts, or there is nothing in flight
	// and the wait this test is about is never reached.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		s.Require().Fail("the link was never dispatched")
	}

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDrain()

	drained := make(chan struct{})
	go func() {
		scheduler.Drain(drainCtx)
		close(drained)
	}()

	// Draining must still be waiting while the link is running.
	select {
	case <-drained:
		s.Fail("draining returned before the link it was waiting for had finished")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		s.Require().Fail("the link never finished")
	}

	select {
	case <-drained:
	case <-time.After(2 * time.Second):
		s.Fail("draining did not return after its link finished")
	}
}

func (s *LinkSchedulerDrainTestSuite) pendingLink(name string) *LinkPendingCompletion {
	return &LinkPendingCompletion{
		resourceANode: &links.ChainLinkNode{ResourceName: name + "A"},
		resourceBNode: &links.ChainLinkNode{ResourceName: name + "B"},
		linkPending:   true,
	}
}

func TestLinkSchedulerDrainTestSuite(t *testing.T) {
	suite.Run(t, new(LinkSchedulerDrainTestSuite))
}
