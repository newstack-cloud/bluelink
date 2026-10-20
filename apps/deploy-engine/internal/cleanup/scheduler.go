package cleanup

import (
	"context"
	"errors"
	"sync"
	"time"

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

func (s *Scheduler) runPass(ctx context.Context) {
	for _, task := range s.tasks {
		if ctx.Err() != nil {
			return
		}

		s.runTask(ctx, task)
	}
}

func (s *Scheduler) runTask(ctx context.Context, task Task) {
	logger := s.logger.WithFields(core.StringLogField("cleanupType", string(task.Type)))

	// Every instance sharing a state backend runs its own scheduler, so without this,
	// they would each clean up the same records on their own interval. The work is
	// idempotent, being a delete by threshold, so the duplication is wasteful rather
	// than harmful.
	recent, err := s.runner.CompletedWithin(ctx, task.Type, s.interval)
	if err != nil {
		logger.Error(
			"failed to check when this category was last cleaned up, skipping this pass",
			core.ErrorLogField("error", err),
		)
		return
	}
	if recent {
		logger.Debug("category was cleaned up recently, skipping this pass")
		return
	}

	operation, err := s.runner.Run(ctx, task)
	if err != nil {
		// Something else is already cleaning this category, which is the outcome this
		// pass wanted rather than a problem with it.
		if errors.Is(err, ErrCleanupInProgress) {
			logger.Debug("a cleanup for this category is already in progress, skipping this pass")
			return
		}
		logger.Error(
			"failed to start a scheduled cleanup",
			core.ErrorLogField("error", err),
		)
		return
	}

	logger.Info(
		"scheduled cleanup finished",
		core.StringLogField("status", string(operation.Status)),
		core.IntegerLogField("itemsDeleted", operation.ItemsDeleted),
	)
}
