// Package deployui provides drift item types for the deploy command.
// All core drift types are defined in the driftui package and re-exported here
// for backwards compatibility.
package deployui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
)

// Type aliases for backwards compatibility
type (
	DriftItemType        = driftui.DriftItemType
	DriftItem            = driftui.DriftItem
	DriftDetailsRenderer = driftui.DriftDetailsRenderer
	DriftSectionGrouper  = driftui.DriftSectionGrouper
	DriftFooterRenderer  = driftui.DriftFooterRenderer
)

// Constant aliases for backwards compatibility
const (
	DriftItemTypeResource = driftui.DriftItemTypeResource
	DriftItemTypeLink     = driftui.DriftItemTypeLink
	DriftItemTypeChild    = driftui.DriftItemTypeChild
)

// BuildDriftItems creates DriftItems from a ReconciliationCheckResult.
var BuildDriftItems = driftui.BuildDriftItems

// SortDriftItems sorts items alphabetically by name.
var SortDriftItems = driftui.SortDriftItems

// HumanReadableDriftType converts a ReconciliationType to a short uppercase label.
var HumanReadableDriftType = driftui.HumanReadableDriftType

// HumanReadableAction converts a ReconciliationAction to a human-readable label.
var HumanReadableAction = driftui.HumanReadableAction

// HumanReadableDriftTypeLabel converts a ReconciliationType to a human-readable label.
var HumanReadableDriftTypeLabel = driftui.HumanReadableDriftTypeLabel
