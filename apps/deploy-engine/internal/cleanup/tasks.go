package cleanup

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

// One constructor per category, so that a category's type and the name it is logged
// under are decided in one place. Both drivers (HTTP endpoints and scheduler intervals)
// build their tasks through these.

// EventsTask cleans up events older than the retention period.
func EventsTask(store manage.Events, retentionPeriod time.Duration) Task {
	return Task{
		Type:            manage.CleanupTypeEvents,
		RetentionPeriod: retentionPeriod,
		Remove:          store.Cleanup,
		LogDescription:  "events",
		LogName:         "eventsCleanup",
	}
}

// ValidationsTask cleans up blueprint validations older than the retention period.
func ValidationsTask(store manage.Validation, retentionPeriod time.Duration) Task {
	return Task{
		Type:            manage.CleanupTypeValidations,
		RetentionPeriod: retentionPeriod,
		Remove:          store.Cleanup,
		LogDescription:  "blueprint validations",
		LogName:         "validationCleanup",
	}
}

// ChangesetsTask cleans up change sets older than the retention period.
func ChangesetsTask(store manage.Changesets, retentionPeriod time.Duration) Task {
	return Task{
		Type:            manage.CleanupTypeChangesets,
		RetentionPeriod: retentionPeriod,
		Remove:          store.Cleanup,
		LogDescription:  "change sets",
		LogName:         "changesetCleanup",
	}
}

// ReconciliationResultsTask cleans up reconciliation results older than the retention
// period.
func ReconciliationResultsTask(
	store manage.ReconciliationResults,
	retentionPeriod time.Duration,
) Task {
	return Task{
		Type:            manage.CleanupTypeReconciliationResults,
		RetentionPeriod: retentionPeriod,
		Remove:          store.Cleanup,
		LogDescription:  "reconciliation results",
		LogName:         "reconciliationResultsCleanup",
	}
}

// StateHousekeepingTask carries out the upkeep the state backend needs on its own storage.
//
// What that involves, and how old anything has to be before it is touched, is the backend's
// to decide, so the task has no retention period. The threshold the runner records for it is
// the time the run started.
func StateHousekeepingTask(housekeeping manage.Housekeeping) Task {
	return Task{
		Type: manage.CleanupTypeStateHousekeeping,
		Remove: func(ctx context.Context, _ time.Time) (int64, error) {
			return housekeeping.Run(ctx)
		},
		LogDescription: "state housekeeping",
		LogName:        "stateHousekeeping",
	}
}

// AuditRecordsTask cleans up audit records older than the retention period.
func AuditRecordsTask(store manage.Audit, retentionPeriod time.Duration) Task {
	return Task{
		Type:            manage.CleanupTypeAuditRecords,
		RetentionPeriod: retentionPeriod,
		Remove:          store.Cleanup,
		LogDescription:  "audit records",
		LogName:         "auditRecordsCleanup",
	}
}
