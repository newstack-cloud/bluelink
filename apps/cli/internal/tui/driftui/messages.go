package driftui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// DriftDetectedMsg is sent when drift is detected during staging or deployment.
// For staging: triggered by the DriftDetected streaming event.
// For deployment: triggered by a 409 DriftBlockedResponse.
type DriftDetectedMsg struct {
	// ReconciliationResult contains the full drift/interrupted state detection result.
	ReconciliationResult *container.ReconciliationCheckResult
	// Message explains what was detected.
	Message string
	// InstanceID is the ID of the blueprint instance.
	InstanceID string
	// ChangesetID is the ID of the changeset (for continuing deployment after reconciliation).
	ChangesetID string
	// InstanceState is the current instance state (for displaying computed fields).
	InstanceState *state.InstanceState
}

// ReconciliationCompleteMsg is sent after successful reconciliation.
type ReconciliationCompleteMsg struct {
	// InstanceID is the ID of the blueprint instance that was reconciled.
	InstanceID string
	// ResourcesUpdated is the number of resources that were successfully updated.
	ResourcesUpdated int
	// LinksUpdated is the number of links that were successfully updated.
	LinksUpdated int
}

// ReconciliationErrorMsg is sent when reconciliation fails.
type ReconciliationErrorMsg struct {
	// Err is the error that occurred during reconciliation.
	Err error
}

// DriftContext determines the contextual hints shown in the footer.
type DriftContext string

const (
	// DriftContextStage is used when drift is detected during the stage command.
	// Hint: "Use --skip-drift-check to skip drift detection"
	DriftContextStage DriftContext = "stage"
	// DriftContextDeployStage is used when drift is detected during the staging phase of deploy --stage.
	// Hint: "Use --skip-drift-check to skip drift detection"
	DriftContextDeployStage DriftContext = "deploy_stage"
	// DriftContextDeploy is used when drift is detected during deployment (409 response).
	// Hint: "Use --force to override drift check"
	DriftContextDeploy DriftContext = "deploy"
)

// HintForContext returns the appropriate hint text for the given drift context.
func HintForContext(ctx DriftContext) string {
	switch ctx {
	case DriftContextStage, DriftContextDeployStage:
		return "Run bluelink stage --skip-drift-check to skip drift detection"
	case DriftContextDeploy:
		return "Run bluelink deploy --force to override drift check"
	default:
		return ""
	}
}
