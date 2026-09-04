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

type LinkSchedulerSubmitTestSuite struct {
	suite.Suite
}

func (s *LinkSchedulerSubmitTestSuite) Test_submitting_does_not_wait_for_a_busy_dispatcher() {
	held := make(chan struct{})
	defer close(held)
	busy := make(chan struct{}, 1)

	scheduler := newLinkScheduler(
		NewLinkSlots(DefaultMaxConcurrentLinks),
		func(string) provider.LinkModifies { return provider.LinkModifiesBoth },
		func(context.Context, *LinkPendingCompletion, *state.InstanceState) error {
			return nil
		},
		func(context.Context) (*state.InstanceState, error) {
			return &state.InstanceState{}, nil
		},
		func(string) {},
		func(string) []string { return nil },
		func(string) bool {
			select {
			case busy <- struct{}{}:
			default:
			}
			<-held
			return false
		},
		func(error) bool { return true },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	scheduler.Submit([]*LinkPendingCompletion{s.pendingLink("first")})

	select {
	case <-busy:
	case <-time.After(2 * time.Second):
		s.Require().Fail("the dispatcher never picked up the first link, so it is not busy")
	}

	// Enough submits to fill any buffer the dispatcher is not draining.
	returned := make(chan struct{})
	go func() {
		for index := range 8 {
			scheduler.Submit([]*LinkPendingCompletion{
				s.pendingLink(string(rune('a' + index))),
			})
		}
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		s.Fail("submitting blocked while the dispatcher was busy")
	}
}

// A scheduler that has stopped collects nothing, and a resource completing after a
// terminal failure still has to be able to report the links it released.
func (s *LinkSchedulerSubmitTestSuite) Test_submitting_does_not_wait_for_a_stopped_dispatcher() {
	scheduler := newLinkScheduler(
		NewLinkSlots(DefaultMaxConcurrentLinks),
		func(string) provider.LinkModifies { return provider.LinkModifiesBoth },
		func(context.Context, *LinkPendingCompletion, *state.InstanceState) error {
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
	scheduler.Start(ctx)
	cancel()

	returned := make(chan struct{})
	go func() {
		for index := range 4 {
			scheduler.Submit([]*LinkPendingCompletion{
				s.pendingLink(string(rune('a' + index))),
			})
		}
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		s.Fail("submitting blocked on a stopped scheduler")
	}
}

func (s *LinkSchedulerSubmitTestSuite) pendingLink(name string) *LinkPendingCompletion {
	return &LinkPendingCompletion{
		resourceANode: &links.ChainLinkNode{ResourceName: name + "A"},
		resourceBNode: &links.ChainLinkNode{ResourceName: name + "B"},
		linkPending:   true,
	}
}

func TestLinkSchedulerSubmitTestSuite(t *testing.T) {
	suite.Run(t, new(LinkSchedulerSubmitTestSuite))
}
