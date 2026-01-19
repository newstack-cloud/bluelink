package jsonout

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// StageOutput represents a successful staging result.
type StageOutput struct {
	Success      bool                      `json:"success"`
	ChangesetID  string                    `json:"changesetId"`
	InstanceID   string                    `json:"instanceId,omitempty"`
	InstanceName string                    `json:"instanceName,omitempty"`
	Changes      *changes.BlueprintChanges `json:"changes"`
	Summary      ChangeSummary             `json:"summary"`
}

// StageDriftOutput represents drift detected during staging.
type StageDriftOutput struct {
	Success        bool                                `json:"success"`
	DriftDetected  bool                                `json:"driftDetected"`
	InstanceID     string                              `json:"instanceId"`
	InstanceName   string                              `json:"instanceName,omitempty"`
	Message        string                              `json:"message"`
	Reconciliation *container.ReconciliationCheckResult `json:"reconciliation"`
}

// ErrorOutput represents a structured error output.
type ErrorOutput struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail provides detailed error information.
type ErrorDetail struct {
	Type        string            `json:"type"` // "validation", "stream", "client", "internal"
	Message     string            `json:"message"`
	StatusCode  int               `json:"statusCode,omitempty"`
	Diagnostics []Diagnostic      `json:"diagnostics,omitempty"`
	Validation  []ValidationError `json:"validation,omitempty"`
}

// ChangeSummary contains summary counts organized by element type.
type ChangeSummary struct {
	Resources ResourceSummary `json:"resources"`
	Children  ChildSummary    `json:"children"`
	Links     LinkSummary     `json:"links"`
	Exports   ExportSummary   `json:"exports"`
}

// ResourceSummary contains action counts for resources.
type ResourceSummary struct {
	Total    int `json:"total"`
	Create   int `json:"create"`
	Update   int `json:"update"`
	Delete   int `json:"delete"`
	Recreate int `json:"recreate"`
}

// ChildSummary contains action counts for child blueprints.
type ChildSummary struct {
	Total  int `json:"total"`
	Create int `json:"create"`
	Update int `json:"update"`
	Delete int `json:"delete"`
}

// LinkSummary contains action counts for links.
type LinkSummary struct {
	Total  int `json:"total"`
	Create int `json:"create"`
	Update int `json:"update"`
	Delete int `json:"delete"`
}

// ExportSummary contains action counts for exports.
type ExportSummary struct {
	Total     int `json:"total"`
	New       int `json:"new"`
	Modified  int `json:"modified"`
	Removed   int `json:"removed"`
	Unchanged int `json:"unchanged"`
}

// Diagnostic represents a single diagnostic message.
type Diagnostic struct {
	Level            string            `json:"level"`
	Message          string            `json:"message"`
	Line             int               `json:"line,omitempty"`
	Column           int               `json:"column,omitempty"`
	Code             string            `json:"code,omitempty"`
	Category         string            `json:"category,omitempty"`
	SuggestedActions []SuggestedAction `json:"suggestedActions,omitempty"`
}

// SuggestedAction represents an actionable suggestion for resolving an error.
type SuggestedAction struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Commands    []string `json:"commands,omitempty"`
	Links       []ActionLink `json:"links,omitempty"`
}

// ActionLink represents a link to documentation or resources.
type ActionLink struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url"`
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Type     string `json:"type,omitempty"`
}

// DeployOutput represents a successful deployment result.
type DeployOutput struct {
	Success          bool                              `json:"success"`
	InstanceID       string                            `json:"instanceId"`
	InstanceName     string                            `json:"instanceName,omitempty"`
	ChangesetID      string                            `json:"changesetId"`
	Status           string                            `json:"status"`
	InstanceState    *state.InstanceState              `json:"instanceState,omitempty"`
	PreRollbackState *container.PreRollbackStateMessage `json:"preRollbackState,omitempty"`
	Summary          DeploySummary                     `json:"summary"`
}

// DeploySummary contains deployment result summary.
type DeploySummary struct {
	Successful           int                             `json:"successful"`
	Failed               int                             `json:"failed"`
	Interrupted          int                             `json:"interrupted"`
	SkippedRollbackItems []container.SkippedRollbackItem `json:"skippedRollbackItems,omitempty"`
	Elements             []DeployedElement               `json:"elements"`
}

// DeployedElement represents an element in the deployment.
type DeployedElement struct {
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`                     // "resource", "child", "link"
	Status         string   `json:"status"`
	Action         string   `json:"action,omitempty"`         // "created", "updated", "destroyed", etc.
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// DeployDriftOutput represents drift detected during deployment.
type DeployDriftOutput struct {
	Success        bool                                 `json:"success"`
	DriftDetected  bool                                 `json:"driftDetected"`
	InstanceID     string                               `json:"instanceId"`
	InstanceName   string                               `json:"instanceName,omitempty"`
	ChangesetID    string                               `json:"changesetId,omitempty"`
	Message        string                               `json:"message"`
	Reconciliation *container.ReconciliationCheckResult `json:"reconciliation"`
}

// DestroyOutput represents a successful destroy result.
type DestroyOutput struct {
	Success         bool                 `json:"success"`
	InstanceID      string               `json:"instanceId"`
	InstanceName    string               `json:"instanceName,omitempty"`
	ChangesetID     string               `json:"changesetId"`
	Status          string               `json:"status"`
	InstanceState   *state.InstanceState `json:"instanceState,omitempty"`
	PreDestroyState *state.InstanceState `json:"preDestroyState,omitempty"`
	Summary         DestroySummary       `json:"summary"`
}

// DestroySummary contains destroy operation summary.
type DestroySummary struct {
	Destroyed   int                `json:"destroyed"`
	Failed      int                `json:"failed"`
	Interrupted int                `json:"interrupted"`
	Elements    []DestroyedElement `json:"elements"`
}

// DestroyedElement represents an element in the destroy result.
type DestroyedElement struct {
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Type           string   `json:"type"` // "resource", "child", "link"
	Status         string   `json:"status"`
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// DestroyDriftOutput represents drift detected during destroy.
type DestroyDriftOutput struct {
	Success        bool                                 `json:"success"`
	DriftDetected  bool                                 `json:"driftDetected"`
	InstanceID     string                               `json:"instanceId"`
	InstanceName   string                               `json:"instanceName,omitempty"`
	Message        string                               `json:"message"`
	Reconciliation *container.ReconciliationCheckResult `json:"reconciliation"`
}

// ListInstancesOutput represents the result of listing instances.
type ListInstancesOutput struct {
	Success    bool                    `json:"success"`
	Instances  []ListInstanceItem      `json:"instances"`
	TotalCount int                     `json:"totalCount"`
	Search     string                  `json:"search,omitempty"`
}

// ListInstanceItem represents a single instance in the list output.
type ListInstanceItem struct {
	InstanceID            string `json:"instanceId"`
	InstanceName          string `json:"instanceName"`
	Status                string `json:"status"`
	LastDeployedTimestamp int64  `json:"lastDeployedTimestamp"`
}

// StateImportOutput represents a state import result.
type StateImportOutput struct {
	Success        bool   `json:"success"`
	Mode           string `json:"mode"`
	InstancesCount int    `json:"instancesCount,omitempty"`
	FilesExtracted int    `json:"filesExtracted,omitempty"`
	Message        string `json:"message"`
}
