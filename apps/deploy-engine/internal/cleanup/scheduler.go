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

// Scheduler carries out the configured cleanup tasks on an interval.
//
// This is what makes the retention periods take effect; core.ScheduledCleanupConfig
// carries why it is off by default.
type Scheduler struct {
	runner   *Runner
	tasks    []Task
	interval time.Duration
	clock    commoncore.Clock
	logger   core.Logger
}

// NewScheduler creates a scheduler that runs the given tasks every interval.
func NewScheduler(
	runner *Runner,
	tasks []Task,
	interval time.Duration,
	clock commoncore.Clock,
	logger core.Logger,
) *Scheduler {
	return &Scheduler{
		runner:   runner,
		tasks:    tasks,
		interval: interval,
		clock:    clock,
		logger:   logger,
	}
}

// Start begins running the tasks and returns the function that stops them.
//
// The returned function waits for an in-flight pass to finish, so that shutdown does
// not leave a cleanup half-done with its operation record still reading "running".
func (s *Scheduler) Start() func() {
	ctx, cancel := context.WithCancel(context.Background())
	var waitGroup sync.WaitGroup

	waitGroup.Go(func() {
		s.run(ctx)
	})

	return func() {
		cancel()
		waitGroup.Wait()
	}
}

// The first pass waits a full interval rather than running at startup. A deployment
// that restarts often would otherwise clean up on every start, and a restart is the
// point at which the engine has the least reason to be confident about what else is
// running.
func (s *Scheduler) run(ctx context.Context) {
	s.logger.Info(
		"scheduled state cleanup is running",
		core.StringLogField("interval", s.interval.String()),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("scheduled state cleanup stopping")
			return
		case <-ticker.C:
			s.runPass(ctx)
		}
	}
}

// One span for the pass with a span per task inside it, so that a pass which spent its
// time on a single category shows that rather than only how long the whole of it took.
func (s *Scheduler) runPass(ctx context.Context) {
	ctx, span := telemetry.CurrentTracer().StartRoot(
		ctx,
		cleanupPassSpanName,
		// Nothing asked for a scheduled pass, so there is nothing to link it to.
		telemetry.SpanContext{},
	)
	defer span.End()

	for _, task := range s.tasks {
		if ctx.Err() != nil {
			return
		}

		s.runTask(ctx, task)
	}
}

func (s *Scheduler) runTask(ctx context.Context, task Task) {
	ctx, span := telemetry.CurrentTracer().Start(
		ctx,
		cleanupTaskSpanName,
		telemetry.StringAttr("cleanupType", string(task.Type)),
	)
	defer span.End()

	logger := s.logger.WithFields(taskLogFields(task, span)...)

	// Every instance sharing a state backend runs its own scheduler, so without this,
	// they would each clean up the same records on their own interval. The work is
	// idempotent, being a delete by threshold, so the duplication is wasteful rather
	// than harmful.
	recent, err := s.runner.CompletedWithin(ctx, task.Type, s.interval)
	if err != nil {
		span.RecordError(err)
		logger.Error(
			"failed to check when this category was last cleaned up, skipping this pass",
			core.ErrorLogField("error", err),
		)
		return
	}
	if recent {
		span.SetAttributes(telemetry.StringAttr("outcome", "skippedCleanedUpRecently"))
		logger.Debug("category was cleaned up recently, skipping this pass")
		return
	}

	operation, err := s.runner.Run(ctx, task)
	if err != nil {
		// Something else is already cleaning this category, which is the outcome this
		// pass wanted rather than a problem with it.
		if errors.Is(err, ErrCleanupInProgress) {
			span.SetAttributes(telemetry.StringAttr("outcome", "skippedAlreadyInProgress"))
			logger.Debug("a cleanup for this category is already in progress, skipping this pass")
			return
		}
		span.RecordError(err)
		logger.Error(
			"failed to start a scheduled cleanup",
			core.ErrorLogField("error", err),
		)
		return
	}

	recordCleanupOutcome(span, logger, operation)
}

func recordCleanupOutcome(
	span telemetry.Span,
	logger core.Logger,
	operation *manage.CleanupOperation,
) {
	recordCleanupSpanOutcome(span, operation)

	logger.Info(
		"scheduled cleanup finished",
		core.StringLogField("status", string(operation.Status)),
		core.IntegerLogField("itemsDeleted", operation.ItemsDeleted),
	)
}

// Shared with the runs the cleanup endpoints start, which record the same outcome but
// must not log it as a scheduled one.
func recordCleanupSpanOutcome(span telemetry.Span, operation *manage.CleanupOperation) {
	span.SetAttributes(
		telemetry.StringAttr("status", string(operation.Status)),
		telemetry.IntAttr("itemsDeleted", operation.ItemsDeleted),
	)

	// A failed removal is reported through the operation rather than as an error from
	// the run, so the span has to be marked failed from the status or a backend
	// filtering failed spans will miss it.
	if operation.Status == manage.CleanupOperationStatusFailed {
		span.RecordError(cleanupFailedError(operation))
	}
}

func cleanupFailedError(operation *manage.CleanupOperation) error {
	if operation.ErrorMessage == "" {
		return errors.New("the cleanup failed without saying why")
	}

	return errors.New(operation.ErrorMessage)
}

// The span's identity goes onto the logger so that the entries a pass writes lead to
// the trace of it. Omitted where no backend has the span, see
// telemetry.SpanContext.Recorded.
func taskLogFields(task Task, span telemetry.Span) []core.LogField {
	fields := []core.LogField{
		core.StringLogField("cleanupType", string(task.Type)),
	}

	spanCtx := span.Context()
	if spanCtx.Recorded() {
		fields = append(
			fields,
			core.StringLogField(telemetry.TraceIDField, spanCtx.TraceID),
			core.StringLogField(telemetry.SpanIDField, spanCtx.SpanID),
		)
	}

	return fields
}

// Named for the work rather than the category, with the category an attribute, so that
// a backend groups every pass over every category together.
const (
	cleanupPassSpanName = "cleanup.pass"
	cleanupTaskSpanName = "cleanup.task"
	cleanupRunSpanName  = "cleanup.run"
)
