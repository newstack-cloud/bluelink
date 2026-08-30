package container

import (
	"context"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// DefaultMaxConcurrentLinks is the number of links a deployment will deploy at the same
// time when nothing else is configured.
//
// The budget is shared by every blueprint instance in a deployment rather than held per
// instance.
//
// It doesn't only exist to reduce impact of rate limits for an upstream API.
// The deploy event loop drains every link status message in the instance tree on a single goroutine,
// plugins are separate processes with their own connection pools, and each in-flight link holds file
// descriptors. Retries and backoff are handled separately in provider plugins.
const DefaultMaxConcurrentLinks = 20

// LinkSlots bounds how many links a deployment deploys at the same time.
//
// A scheduler must never block acquiring a slot, with a shared budget the slot it waits
// for may be released by another blueprint instance's scheduler, which has no way to wake
// it. TryAcquire and Wait keep the caller in control of when it blocks, and waiters are
// served in the order they arrived so a busy parent cannot starve a child.
type LinkSlots struct {
	mu      sync.Mutex
	free    int
	waiters []chan struct{}
}

// NewLinkSlots creates a budget of the given size, shared by every blueprint instance in
// a deployment.
func NewLinkSlots(maxConcurrent int) *LinkSlots {
	if maxConcurrent < 1 {
		maxConcurrent = DefaultMaxConcurrentLinks
	}

	return &LinkSlots{free: maxConcurrent}
}

// TryAcquire takes a slot if one is free, reporting whether it did.
func (s *LinkSlots) TryAcquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.free == 0 {
		return false
	}
	s.free -= 1

	return true
}

// Wait registers for the next free slot and returns a channel that is closed when one may
// be available. The caller must call TryAcquire again on wake, since another waiter may
// have taken the slot first.
func (s *LinkSlots) Wait() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	waiter := make(chan struct{})
	if s.free > 0 {
		close(waiter)
		return waiter
	}
	s.waiters = append(s.waiters, waiter)

	return waiter
}

// Release returns a slot and wakes the longest-waiting scheduler.
func (s *LinkSlots) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.free += 1
	if len(s.waiters) == 0 {
		return
	}

	waiter := s.waiters[0]
	s.waiters = s.waiters[1:]
	close(waiter)
}

type linkCompletion struct {
	linkName string
	holds    []string
}

// Deploys the links of one blueprint instance, replacing the batch of links that used to
// be assembled and run for each resource completion.
//
// One goroutine owns the pending set for the lifetime of a deployment,
// so a global lock is no longer needed to protect the set from concurrent
// access by multiple resource completion goroutines.
type linkScheduler struct {
	slots             *LinkSlots
	modifies          func(linkName string) provider.LinkModifies
	deployLink        func(context.Context, *LinkPendingCompletion, *state.InstanceState) error
	loadInstanceState func(context.Context) (*state.InstanceState, error)
	markSettled       func(linkName string)
	awaitingProviders func(linkName string) []string
	// Reports an error to whoever is listening, returning false if it could not be
	// delivered without waiting. The scheduler never waits on it.
	trySendError func(err error) bool

	errMu      sync.Mutex
	heldErrors []error

	pendingMu    sync.Mutex
	pendingCount int

	submitCh   chan []*LinkPendingCompletion
	completeCh chan *linkCompletion
	drainCh    chan chan []*LinkPendingCompletion
	stopped    chan struct{}

	inFlight sync.WaitGroup
}

func newLinkScheduler(
	slots *LinkSlots,
	modifies func(linkName string) provider.LinkModifies,
	deployLink func(context.Context, *LinkPendingCompletion, *state.InstanceState) error,
	loadInstanceState func(context.Context) (*state.InstanceState, error),
	markSettled func(linkName string),
	awaitingProviders func(linkName string) []string,
	trySendError func(err error) bool,
) *linkScheduler {
	return &linkScheduler{
		slots:             slots,
		modifies:          modifies,
		deployLink:        deployLink,
		loadInstanceState: loadInstanceState,
		markSettled:       markSettled,
		awaitingProviders: awaitingProviders,
		trySendError:      trySendError,
		submitCh:          make(chan []*LinkPendingCompletion, 1),
		completeCh:        make(chan *linkCompletion, 1),
		drainCh:           make(chan chan []*LinkPendingCompletion),
		stopped:           make(chan struct{}),
	}
}

// Start launches the dispatcher, which runs until the context is cancelled or Drain is
// called.
func (s *linkScheduler) Start(ctx context.Context) {
	go s.dispatch(ctx)
}

// Submit hands links that have become ready to the dispatcher.
//
// It never blocks on a stopped scheduler, so a resource completing after a terminal
// failure does not wedge the deployment event loop that reported it.
func (s *linkScheduler) Submit(links []*LinkPendingCompletion) {
	if len(links) == 0 {
		return
	}

	select {
	case s.submitCh <- links:
	case <-s.stopped:
	}
}

// Drain stops dispatching, waits for links already running to finish, and returns the
// links that never started along with any errors that could not be delivered while the
// deployment was running, so the caller can report both.
func (s *linkScheduler) Drain(ctx context.Context) ([]*LinkPendingCompletion, []error) {
	leftover := make(chan []*LinkPendingCompletion, 1)

	select {
	case s.drainCh <- leftover:
	case <-s.stopped:
		return nil, s.takeHeldErrors()
	case <-ctx.Done():
		return nil, s.takeHeldErrors()
	}

	s.inFlight.Wait()

	// A worker that finished after the dispatcher returned has no one left to forward
	// its error, so this is the last chance to collect it.
	s.forwardErrors()

	select {
	case pending := <-leftover:
		return pending, s.takeHeldErrors()
	case <-ctx.Done():
		return nil, s.takeHeldErrors()
	}
}

// HasPendingLinks reports whether any link is waiting to be dispatched, which is work the
// deployment can still make progress on even when nothing is in flight.
func (s *linkScheduler) HasPendingLinks() bool {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	return s.pendingCount > 0
}

func (s *linkScheduler) setPendingCount(count int) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	s.pendingCount = count
}

// Holds an error until it can be delivered without waiting.
//
// A worker must never block reporting one. The error channel is unbuffered, and whoever
// reads it may be busy or may have stopped, so a worker parked on a send holds its slot
// and its resource affinity for the rest of the deployment and never reports completion.
// Holding the error costs nothing and loses nothing, the dispatcher tries to deliver it
// on every pass, and Drain hands back whatever never went so the caller can report it.
func (s *linkScheduler) recordError(err error) {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	s.heldErrors = append(s.heldErrors, err)
}

func (s *linkScheduler) forwardErrors() {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	undelivered := s.heldErrors[:0]
	for _, err := range s.heldErrors {
		if !s.trySendError(err) {
			undelivered = append(undelivered, err)
		}
	}
	s.heldErrors = undelivered
}

func (s *linkScheduler) takeHeldErrors() []error {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	held := s.heldErrors
	s.heldErrors = nil

	return held
}

func (s *linkScheduler) dispatch(ctx context.Context) {
	defer close(s.stopped)

	pending := []*LinkPendingCompletion{}
	writing := map[string]bool{}
	var slotWait <-chan struct{}

	for {
		select {
		case links := <-s.submitCh:
			pending = append(pending, links...)
		case completed := <-s.completeCh:
			s.markSettled(completed.linkName)
			for _, resourceName := range completed.holds {
				delete(writing, resourceName)
			}
			s.slots.Release()
		case leftover := <-s.drainCh:
			leftover <- pending
			return
		case <-slotWait:
			slotWait = nil
		case <-ctx.Done():
			return
		}

		s.forwardErrors()
		pending = s.dispatchReady(ctx, pending, writing)
		s.setPendingCount(len(pending))
		slotWait = s.waitForSlotIfBlocked(pending, writing, slotWait)
	}
}

// A slot is only waited on when something is held back for want of one. Waiting while
// every pending link is blocked on a capability provider instead would wake the
// dispatcher for a slot it has no use for.
func (s *linkScheduler) waitForSlotIfBlocked(
	pending []*LinkPendingCompletion,
	writing map[string]bool,
	current <-chan struct{},
) <-chan struct{} {
	if current != nil {
		return current
	}

	for _, link := range pending {
		if s.readyToDispatch(link, writing) {
			return s.slots.Wait()
		}
	}

	return nil
}

func (s *linkScheduler) dispatchReady(
	ctx context.Context,
	pending []*LinkPendingCompletion,
	writing map[string]bool,
) []*LinkPendingCompletion {
	held := []*LinkPendingCompletion{}
	var instanceState *state.InstanceState

	for _, link := range pending {
		if !s.readyToDispatch(link, writing) {
			held = append(held, link)
			continue
		}

		if !s.slots.TryAcquire() {
			held = append(held, link)
			continue
		}

		if instanceState == nil {
			loaded, err := s.loadInstanceState(ctx)
			if err != nil {
				s.slots.Release()
				s.recordError(err)
				return append(held, link)
			}
			instanceState = loaded
		}

		s.run(ctx, link, instanceState, writing)
	}

	return held
}

// Resource affinity, not correctness. Two links on one resource are kept apart by the
// resource locks the link deployer holds around each phase; holding the second back here
// stops it taking a slot only to block on a lock while links on other resources queue
// behind it. Without it, throughput on a blueprint whose links fan out from a few
// resources drops by more than half, because the slots are spent on links that cannot
// move.
//
// Only the sides a link declared it writes are keyed on. A link that reads a resource is
// not held back because another link is busy with it, since the deployer takes no
// exclusive lock on a side that is only read. A link that declares nothing is treated as
// writing both, which is what the framework did before the declaration existed.
//
// It sees the link's own two resources and not the ones a link implementation locks from
// the inside, such as an execution role several links write policy to, so it is a
// scheduling improvement rather than a guarantee.
func (s *linkScheduler) readyToDispatch(
	link *LinkPendingCompletion,
	writing map[string]bool,
) bool {
	linkName := pendingLinkName(link)
	if len(s.awaitingProviders(linkName)) > 0 {
		return false
	}

	writtenResources := s.writtenResources(link)
	for _, resourceName := range writtenResources {
		if writing[resourceName] {
			return false
		}
	}

	return true
}

func (s *linkScheduler) writtenResources(link *LinkPendingCompletion) []string {
	modifies := s.modifies(pendingLinkName(link))
	written := []string{}
	if modifies.WritesResourceA() {
		written = append(written, link.resourceANode.ResourceName)
	}
	if modifies.WritesResourceB() {
		written = append(written, link.resourceBNode.ResourceName)
	}

	return written
}

func (s *linkScheduler) run(
	ctx context.Context,
	link *LinkPendingCompletion,
	instanceState *state.InstanceState,
	writing map[string]bool,
) {
	holds := s.writtenResources(link)
	for _, resourceName := range holds {
		writing[resourceName] = true
	}

	linkName := pendingLinkName(link)
	s.inFlight.Add(1)

	go func() {
		defer s.inFlight.Done()

		err := s.deployLink(ctx, link, instanceState)

		// Completion is reported first, and on every outcome including a failure.
		//
		// Reporting an error can block, the error channel is unbuffered and whoever
		// reads it may have stopped. Letting that gate the completion means one
		// unread error holds this link's slot and its resource affinity for the rest
		// of the deployment, and leaves every link waiting on a capability this one
		// provides pending with nothing to say why.
		select {
		case s.completeCh <- &linkCompletion{linkName: linkName, holds: holds}:
		case <-s.stopped:
		}

		if err != nil {
			s.recordError(err)
		}
	}()
}
