package enginev1

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// Stops the cleanups started by the cleanup endpoints, bounded by the same timeout as
// the deployment drain, so that a run against a backend that does not give up on being
// cancelled cannot hold up process exit.
func stopRequestedCleanupFunc(
	runner *cleanup.Runner,
	drainTimeout time.Duration,
) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()
		runner.Shutdown(ctx)
	}
}

// Starts the cleanup the engine carries out on an interval, and returns a callback
// to stop the scheduler.
//
// An operator is warned when it is off, because the retention periods otherwise read as
// a promise the engine does not keep. See core.ScheduledCleanupConfig.
func setupScheduledCleanup(
	config *core.Config,
	runner *cleanup.Runner,
	state *stateServices,
	clock commoncore.Clock,
	logger bpcore.Logger,
) func() {
	scheduledConfig := config.Maintenance.ScheduledCleanup
	if !scheduledConfig.Enabled {
		logger.Warn(
			"the deploy engine is not cleaning up state on a schedule, so the configured " +
				"retention periods will not be applied unless the cleanup endpoints are " +
				"called by something else; set maintenance.scheduled_cleanup.enabled to " +
				"true to have the engine apply them itself",
		)

		return func() {}
	}

	scheduler := cleanup.NewScheduler(
		runner,
		cleanupTasks(&config.Maintenance, state),
		schedulerInterval(scheduledConfig.Interval, logger),
		clock,
		logger.Named("scheduledCleanup"),
	)

	return scheduler.Start()
}

// A non-positive interval would take the engine down at startup rather than be
// rejected, since that is what a ticker does with one, so it falls back to the default
// and says so.
func schedulerInterval(configured int, logger bpcore.Logger) time.Duration {
	if configured > 0 {
		return time.Duration(configured) * time.Second
	}

	logger.Warn(
		"maintenance.scheduled_cleanup.interval must be a positive number of seconds, " +
			"falling back to the default of 24 hours",
	)

	return defaultScheduledCleanupInterval
}

const defaultScheduledCleanupInterval = 24 * time.Hour

// Every category that has a retention period. Instances, resources, links and drift
// records are absent because they have no retention period,
// they are the deployed state rather than a record of working through it.
func cleanupTasks(
	maintenance *core.MaintenanceConfig,
	state *stateServices,
) []cleanup.Task {
	return []cleanup.Task{
		cleanup.ValidationsTask(
			state.validation,
			retentionPeriod(maintenance.BlueprintValidationRetentionPeriod),
		),
		cleanup.ChangesetsTask(
			state.changesets,
			retentionPeriod(maintenance.ChangesetRetentionPeriod),
		),
		cleanup.ReconciliationResultsTask(
			state.reconciliationResults,
			retentionPeriod(maintenance.ReconciliationResultsRetentionPeriod),
		),
		cleanup.EventsTask(
			state.events,
			retentionPeriod(maintenance.EventsRetentionPeriod),
		),
		cleanup.AuditRecordsTask(
			state.audit,
			retentionPeriod(maintenance.AuditRecordRetentionPeriod),
		),
	}
}

func retentionPeriod(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
