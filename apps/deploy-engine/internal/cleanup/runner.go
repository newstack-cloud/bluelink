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
	// Categories this engine is mid-run on. Checking the recorded operations alone
	// would not keep two of this engine's own runs apart, since reading them is not
	// atomic with recording one.
	mu       sync.Mutex
	inFlight map[manage.CleanupType]struct{}
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
	runner := &Runner{
		operations:  operations,
		idGenerator: idGenerator,
		clock:       clock,
		timeout:     DefaultTimeout,
		logger:      logger,
		inFlight:    map[manage.CleanupType]struct{}{},
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
// by a request outlives the request that asked for it.
func (r *Runner) Start(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	operation, err := r.begin(ctx, task)
	if err != nil {
		return nil, err
	}

	started := *operation
	go r.execute(context.Background(), operation, task)

	return &started, nil
}

// Run records a cleanup operation and carries the task out before returning,
// reporting the operation as it ended.
//
// The task is bounded by both the runner's timeout and the given context, so a
// caller that is shutting down can cut a run short.
func (r *Runner) Run(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	operation, err := r.begin(ctx, task)
	if err != nil {
		return nil, err
	}

	r.execute(ctx, operation, task)

	return operation, nil
}

func (r *Runner) begin(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
	if err := r.claim(ctx, task.Type); err != nil {
		return nil, err
	}

	operation, err := r.record(ctx, task)
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

	return nil
}

func (r *Runner) release(cleanupType manage.CleanupType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.inFlight, cleanupType)
}

func (r *Runner) record(ctx context.Context, task Task) (*manage.CleanupOperation, error) {
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

func (r *Runner) execute(
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
