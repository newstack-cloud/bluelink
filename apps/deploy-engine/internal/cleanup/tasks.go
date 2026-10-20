package cleanup

import (
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
