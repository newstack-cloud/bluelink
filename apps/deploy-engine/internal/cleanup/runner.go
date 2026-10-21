// Package cleanup carries out the removal of state that has passed its retention
// period, and records each removal as a cleanup operation that can be polled.
//
// The same runner serves both drivers, the HTTP endpoints that start a cleanup on
// request, and the scheduler that starts one on an interval. A category cleaned by
// one is indistinguishable from the same category cleaned by the other.
package cleanup

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// DefaultTimeout bounds a single cleanup run.
//
// Ten minutes is generous enough for a deploy engine with a lot of use, and the work
// happens outside the request that asked for it, so the bound is on the work rather
// than on anyone waiting.
const DefaultTimeout = 10 * time.Minute

// How long the final status write is given. Short, because it is one small write and it
// happens after the work it describes has already finished.
const operationUpdateTimeout = 30 * time.Second

// ErrCleanupInProgress reports that the category is already being cleaned up, so a
// second run was not started.
//
// Two runs over one category are not merely redundant as they both walk the same records,
// and a backend that removes them one at a time has the second run deleting what the
// first already took.
//
// Within one engine, where the scheduler and the cleanup endpoints share a Runner, this
// is exact. Across engines sharing a state backend it is best effort, the only evidence
// another engine is mid-run is the operation it recorded, and reading that is not atomic
// with recording one here.
var ErrCleanupInProgress = errors.New("a cleanup for this category is already in progress")

// ErrShuttingDown reports that the engine is shutting down, so no cleanup was started.
//
// Shutdown waits for the runs already under way, which it can only do if no further run
// begins while it is waiting. Refusing is also the honest answer to the caller, since a
// run accepted here would be cut short moments later.
var ErrShuttingDown = errors.New("the engine is shutting down, so no cleanup was started")

// Task describes one category of state to clean up.
type Task struct {
	// Type names the category, and is what a cleanup operation is recorded under.
	Type manage.CleanupType
	// RetentionPeriod is how far back the category is kept. Records older than this
	// are removed.
	RetentionPeriod time.Duration
	// Remove deletes the records older than thresholdDate, returning how many it
	// removed. This is the category's own store method.
	Remove func(ctx context.Context, thresholdDate time.Time) (int64, error)
	// LogDescription names the category in log messages, as "blueprint validations".
	LogDescription string
	// LogName is the name the run's logger is given, as "validationCleanup".
	LogName string
}

// Runner starts cleanup runs and keeps their operation records up to date.
type Runner struct {
	operations  manage.CleanupOperations
	idGenerator core.IDGenerator
	clock       commoncore.Clock
	timeout     time.Duration
	logger      core.Logger
	// The context the runs started by Runner.Start are given,
	// in place of the context of the request that asked for one.
	// Cancelled by Shutdown.
	detachedCtx    context.Context
	cancelDetached context.CancelFunc
	// Every claimed run, so that Shutdown waits for the work rather than letting the
	// state service close from under it.
	runs sync.WaitGroup
	// Categories this engine is mid-run on. Checking the recorded operations alone
	// would not keep two of this engine's own runs apart, since reading them is not
	// atomic with recording one.
	mu       sync.Mutex
	inFlight map[manage.CleanupType]struct{}
	// Guarded by mu as well. Set once Shutdown has begun,
	// so that no further run joins the set it waits on.
	shuttingDown bool
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithTimeout sets how long a single cleanup run is given, in place of DefaultTimeout.
func WithTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) { r.timeout = timeout }
}

// NewRunner creates a runner that records its operations through the given store.
func NewRunner(
	operations manage.CleanupOperations,
	idGenerator core.IDGenerator,
	clock commoncore.Clock,
	logger core.Logger,
	opts ...RunnerOption,
) *Runner {
	detachedCtx, cancelDetached := context.WithCancel(context.Background())
	runner := &Runner{
		operations:     operations,
		idGenerator:    idGenerator,
		clock:          clock,
		timeout:        DefaultTimeout,
		logger:         logger,
		detachedCtx:    detachedCtx,
		cancelDetached: cancelDetached,
		inFlight:       map[manage.CleanupType]struct{}{},
	}
	for _, opt := range opts {
		opt(runner)
	}

	return runner
}

// Start records a cleanup operation and carries the task out in the background,
// returning the operation as recorded.
//
// The returned operation is a copy taken before the work begins, so a caller can
// serialise it while the run is still updating its own copy.
//
// The work is given a context of its own rather than the caller's, a cleanup started
// by a request outlives the request that asked for it. That context belongs to the
// runner, so Shutdown can cut the run short and wait for it.
func (r *Runner) Start(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	operation, err := r.recordOperation(ctx, task)
	if err != nil {
		return nil, err
	}

	started := *operation

	// A cleanup a request started is a trace of its own, naming the request that asked
	// for it. The run's context comes from the runner rather than the request, so we retain
	// the link span from the request context.
	runCtx, span := telemetry.CurrentTracer().StartRoot(
		r.detachedCtx,
		cleanupRunSpanName,
		telemetry.LinkFrom(ctx),
		telemetry.StringAttr("cleanupType", string(task.Type)),
	)

	// The run's own count is released when the work ends, before the span is finished,
	// so without a count of its own the span can be lost to shutdown.
	r.runs.Go(func() {
		defer span.End()
		r.carryOutTask(runCtx, operation, task)
		recordCleanupSpanOutcome(span, operation)
	})

	return &started, nil
}

// Run records a cleanup operation and carries the task out before returning,
// reporting the operation as it ended.
//
// The task is bounded by both the runner's timeout and the given context, so a
// caller that is shutting down can cut a run short.
func (r *Runner) Run(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	operation, err := r.recordOperation(ctx, task)
	if err != nil {
		return nil, err
	}

	r.carryOutTask(ctx, operation, task)

	return operation, nil
}

// Takes the category and records the run that is about to happen, so that Start and Run
// both have an operation to report before any state is removed. The removal itself is
// carryOutTask.
func (r *Runner) recordOperation(
	ctx context.Context,
	task Task,
) (*manage.CleanupOperation, error) {
	if err := r.claim(ctx, task.Type); err != nil {
		return nil, err
	}

	operation, err := r.saveOperation(ctx, task)
	if err != nil {
		r.release(task.Type)
		return nil, err
	}

	return operation, nil
}

// Takes the category, so that no second run over it begins until this one is released.
//
// The store is consulted under the same lock as the local set, so two of this engine's
// own requests cannot both read "free" and both go on to record a run.
func (r *Runner) claim(ctx context.Context, cleanupType manage.CleanupType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.shuttingDown {
		return ErrShuttingDown
	}

	if _, taken := r.inFlight[cleanupType]; taken {
		return ErrCleanupInProgress
	}

	inProgress, err := r.inProgress(ctx, cleanupType)
	if err != nil {
		return err
	}
	if inProgress {
		return ErrCleanupInProgress
	}

	r.inFlight[cleanupType] = struct{}{}
	// Counted under the same lock as the shutdown check, so a run cannot join the set
	// Shutdown waits on after the wait has begun.
	r.runs.Add(1)

	return nil
}

// Paired with the claim rather than with the work, so the count also covers a claim
// that never became a run because recording it failed. A run started through Start is
// counted once more on top of this, see there for why.
func (r *Runner) release(cleanupType manage.CleanupType) {
	defer r.runs.Done()

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.inFlight, cleanupType)
}

// Shutdown cuts short the cleanups Start left running and waits for every claimed run
// to finish, or for ctx to be done, whichever comes first.
//
// A cleanup started by a request outlives that request, so without this the state
// service could close while a run was still deleting through it. A run cut short
// records itself as failed, which leaves the category for the next pass to pick up
// rather than reporting it as cleaned.
func (r *Runner) Shutdown(ctx context.Context) {
	// Cancelled before the lock is taken, since a claim in progress holds that lock
	// across a read of the recorded operations, and a run already under way should not
	// wait on that read to be cut short.
	r.cancelDetached()

	r.mu.Lock()
	r.shuttingDown = true
	r.mu.Unlock()

	finished := make(chan struct{})
	go func() {
		r.runs.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-ctx.Done():
		r.logger.Warn(
			"stopped waiting for cleanups still in flight, "+
				"the operations they recorded may be left reading as running",
			core.ErrorLogField("error", ctx.Err()),
		)
	}
}

func (r *Runner) saveOperation(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	operationID, err := r.idGenerator.GenerateID()
	if err != nil {
		return nil, err
	}

	startedAt := r.clock.Now()
	operation := &manage.CleanupOperation{
		ID:            operationID,
		CleanupType:   task.Type,
		Status:        manage.CleanupOperationStatusRunning,
		StartedAt:     startedAt.Unix(),
		ThresholdDate: startedAt.Add(-task.RetentionPeriod).Unix(),
	}

	if err := r.operations.Save(ctx, operation); err != nil {
		return nil, err
	}

	return operation, nil
}

func (r *Runner) carryOutTask(
	ctx context.Context,
	operation *manage.CleanupOperation,
	task Task,
) {
	defer r.release(task.Type)

	logger := r.logger.Named(task.LogName)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	thresholdDate := time.Unix(operation.ThresholdDate, 0)
	itemsDeleted, err := task.Remove(ctxWithTimeout, thresholdDate)

	operation.EndedAt = r.clock.Now().Unix()
	operation.ItemsDeleted = itemsDeleted

	if err != nil {
		logger.Error(
			"failed to clean up old "+task.LogDescription,
			core.ErrorLogField("error", err),
		)
		operation.Status = manage.CleanupOperationStatusFailed
		operation.ErrorMessage = err.Error()
	} else {
		operation.Status = manage.CleanupOperationStatusCompleted
	}

	// Updated through a context of its own, since the one the run used may be the
	// reason it ended and an operation left reading "running" forever is worse than
	// one that reports having been cut short.
	updateCtx, updateCancel := context.WithTimeout(context.Background(), operationUpdateTimeout)
	defer updateCancel()

	if updateErr := r.operations.Update(updateCtx, operation); updateErr != nil {
		logger.Error(
			"failed to update cleanup operation",
			core.ErrorLogField("error", updateErr),
		)
	}
}

// LatestRun returns the most recent cleanup operation recorded for the category, and
// whether there is one at all.
//
// A category nobody has ever cleaned up is not an error, so it comes back as absent
// rather than as the store's not-found error.
func (r *Runner) LatestRun(
	ctx context.Context,
	cleanupType manage.CleanupType,
) (*manage.CleanupOperation, bool, error) {
	latest, err := r.operations.GetLatestByType(ctx, cleanupType)
	if err != nil {
		notFoundErr := &manage.CleanupOperationNotFound{}
		if errors.As(err, &notFoundErr) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return latest, true, nil
}

// CompletedWithin reports whether the category's most recent run finished successfully
// less than window ago. Used to leave alone a category another engine has just cleaned.
//
// This only accounts for runs that completed. A run that failed, or that was cut short when its
// engine shut down, says nothing about whether the category still needs cleaning, so it
// must not hold off the next attempt for a whole window.
func (r *Runner) CompletedWithin(
	ctx context.Context,
	cleanupType manage.CleanupType,
	window time.Duration,
) (bool, error) {
	latest, found, err := r.LatestRun(ctx, cleanupType)
	if err != nil || !found {
		return false, err
	}

	if latest.Status != manage.CleanupOperationStatusCompleted {
		return false, nil
	}

	return r.startedWithin(latest, window), nil
}

// An operation left reading "running" by an instance that died would otherwise block the
// category for ever, so one that has been running for longer than a run is allowed to
// take is treated as gone rather than as in progress.
func (r *Runner) inProgress(ctx context.Context, cleanupType manage.CleanupType) (bool, error) {
	latest, found, err := r.LatestRun(ctx, cleanupType)
	if err != nil || !found {
		return false, err
	}

	if latest.Status != manage.CleanupOperationStatusRunning {
		return false, nil
	}

	return r.startedWithin(latest, r.timeout), nil
}

func (r *Runner) startedWithin(
	operation *manage.CleanupOperation,
	window time.Duration,
) bool {
	return r.clock.Now().Sub(time.Unix(operation.StartedAt, 0)) < window
}
