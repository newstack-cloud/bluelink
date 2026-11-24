package helpersv1

import (
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

var preciseResourceStatusLabels = map[string]core.PreciseResourceStatus{
	"UNKNOWN":                          core.PreciseResourceStatusUnknown,
	"UPDATING LINK CONTRIBUTIONS":      core.PreciseResourceStatusUpdatingLinkContributions,
	"LINK CONTRIBUTIONS UPDATED":       core.PreciseResourceStatusLinkContributionsUpdated,
	"LINK CONTRIBUTIONS UPDATE FAILED": core.PreciseResourceStatusLinkContributionsUpdateFailed,
	"CREATING":                         core.PreciseResourceStatusCreating,
	"CONFIG COMPLETE":                  core.PreciseResourceStatusConfigComplete,
	"CREATED":                          core.PreciseResourceStatusCreated,
	"CREATE FAILED":                    core.PreciseResourceStatusCreateFailed,
	"CREATE ROLLING BACK":              core.PreciseResourceStatusCreateRollingBack,
	"CREATE ROLLBACK FAILED":           core.PreciseResourceStatusCreateRollbackFailed,
	"CREATE ROLLBACK COMPLETE":         core.PreciseResourceStatusCreateRollbackComplete,
	"DESTROYING":                       core.PreciseResourceStatusDestroying,
	"DESTROYED":                        core.PreciseResourceStatusDestroyed,
	"DESTROY FAILED":                   core.PreciseResourceStatusDestroyFailed,
	"DESTROY ROLLING BACK":             core.PreciseResourceStatusDestroyRollingBack,
	"DESTROY ROLLBACK FAILED":          core.PreciseResourceStatusDestroyRollbackFailed,
	"DESTROY ROLLBACK CONFIG COMPLETE": core.PreciseResourceStatusDestroyRollbackConfigComplete,
	"DESTROY ROLLBACK COMPLETE":        core.PreciseResourceStatusDestroyRollbackComplete,
	"UPDATING":                         core.PreciseResourceStatusUpdating,
	"UPDATE CONFIG COMPLETE":           core.PreciseResourceStatusUpdateConfigComplete,
	"UPDATED":                          core.PreciseResourceStatusUpdated,
	"UPDATE FAILED":                    core.PreciseResourceStatusUpdateFailed,
	"UPDATE ROLLING BACK":              core.PreciseResourceStatusUpdateRollingBack,
	"UPDATE ROLLBACK FAILED":           core.PreciseResourceStatusUpdateRollbackFailed,
	"UPDATE ROLLBACK CONFIG COMPLETE":  core.PreciseResourceStatusUpdateRollbackConfigComplete,
	"UPDATE ROLLBACK COMPLETE":         core.PreciseResourceStatusUpdateRollbackComplete,
	"CREATE INTERRUPTED":               core.PreciseResourceStatusCreateInterrupted,
	"UPDATE INTERRUPTED":               core.PreciseResourceStatusUpdateInterrupted,
	"DESTROY INTERRUPTED":              core.PreciseResourceStatusDestroyInterrupted,
	"RETAINED":                         core.PreciseResourceStatusRetained,
}

var preciseLinkStatusLabels = map[string]core.PreciseLinkStatus{
	"UNKNOWN":                                         core.PreciseLinkStatusUnknown,
	"UPDATING LINKED RESOURCES":                       core.PreciseLinkStatusUpdatingLinkedResources,
	"LINKED RESOURCES UPDATED":                        core.PreciseLinkStatusLinkedResourcesUpdated,
	"LINKED RESOURCES UPDATE FAILED":                  core.PreciseLinkStatusLinkedResourcesUpdateFailed,
	"LINKED RESOURCES UPDATE ROLLING BACK":            core.PreciseLinkStatusLinkedResourcesUpdateRollingBack,
	"LINKED RESOURCES UPDATE ROLLBACK FAILED":         core.PreciseLinkStatusLinkedResourcesUpdateRollbackFailed,
	"LINKED RESOURCES UPDATE ROLLBACK COMPLETE":       core.PreciseLinkStatusLinkedResourcesUpdateRollbackComplete,
	"UPDATING INTERMEDIARY RESOURCES":                 core.PreciseLinkStatusUpdatingIntermediaryResources,
	"INTERMEDIARY RESOURCES UPDATED":                  core.PreciseLinkStatusIntermediaryResourcesUpdated,
	"INTERMEDIARY RESOURCES UPDATE FAILED":            core.PreciseLinkStatusIntermediaryResourceUpdateFailed,
	"INTERMEDIARY RESOURCES UPDATE ROLLING BACK":      core.PreciseLinkStatusIntermediaryResourceUpdateRollingBack,
	"INTERMEDIARY RESOURCES UPDATE ROLLBACK FAILED":   core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed,
	"INTERMEDIARY RESOURCES UPDATE ROLLBACK COMPLETE": core.PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete,
	"LINKED RESOURCES UPDATE INTERRUPTED":             core.PreciseLinkStatusLinkedResourcesUpdateInterrupted,
	"INTERMEDIARY RESOURCES UPDATE INTERRUPTED":       core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted,
}

// ParsePreciseResourceStatus parses a precise resource status from its label, ignoring case
// (e.g. "created"), or from its integer (e.g. "3"), the form a reconciliation check returns.
func ParsePreciseResourceStatus(value string) (core.PreciseResourceStatus, bool) {
	return parsePreciseStatus(value, preciseResourceStatusLabels)
}

// ParsePreciseLinkStatus parses a precise link status from its label, ignoring case, or from
// its integer.
func ParsePreciseLinkStatus(value string) (core.PreciseLinkStatus, bool) {
	return parsePreciseStatus(value, preciseLinkStatusLabels)
}

func parsePreciseStatus[Status ~int](value string, labels map[string]Status) (Status, bool) {
	if status, isLabel := labels[strings.ToUpper(value)]; isLabel {
		return status, true
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	for _, status := range labels {
		if int(status) == number {
			return status, true
		}
	}
	return 0, false
}
