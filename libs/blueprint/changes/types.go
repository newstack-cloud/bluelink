package changes

import "github.com/newstack-cloud/bluelink/libs/blueprint/provider"

// MetadataChanges holds information about changes to blueprint-wide metadata.
type MetadataChanges struct {
	// NewFields contains new fields that will be added to the blueprint-wide metadata.
	NewFields []provider.FieldChange `json:"newFields"`
	// ModifiedFields contains changes to existing fields in the blueprint-wide metadata.
	ModifiedFields []provider.FieldChange `json:"modifiedFields"`
	// UnchangedFields contains the names of fields that will not be changed.
	UnchangedFields []string `json:"unchangedFields"`
	// RemovedFields contains the names of fields that will no longer be present.
	RemovedFields []string `json:"removedFields"`
}

// BlueprintChanges provides a set of changes that will be made
// to a blueprint instance when deploying a new version of the source blueprint.
// This contains a mapping of resource name
// to the changes that will come into effect upon deploying
// the currently loaded version of a blueprint for a given
// instance ID.
// This also contains a mapping of child blueprint names to the changes
// that will come into effect upon deploying the child blueprint.
// Changes takes the type parameter interface{} as we can't know the exact
// range of resource types for a blueprint at compile time.
// We must check the resource types associated with a set of changes
// at runtime.
type BlueprintChanges struct {
	// NewResources contains the resources that will be created
	// when deploying the changes.
	NewResources map[string]provider.Changes `json:"newResources"`
	// ResourceChanges contains the changes that will be made to
	// existing resources when deploying the changes.
	ResourceChanges map[string]provider.Changes `json:"resourceChanges"`
	// RemovedResources contains the name of the resources that will be removed
	// when deploying the changes.
	RemovedResources []string `json:"removedResources"`
	// RemovedLinks contains the name of the links that will be removed
	// when deploying the changes.
	// These will be in the format "resourceAName::resourceBName".
	RemovedLinks []string `json:"removedLinks"`
	// NewChildren contains the child blueprints that will be created
	// when deploying the changes.
	NewChildren map[string]NewBlueprintDefinition `json:"newChildren"`
	// ChildChanges contains the changes that will be made to the child blueprints
	// when deploying the changes.
	ChildChanges map[string]BlueprintChanges `json:"childChanges"`
	// RecreateChildren contains the name of the child blueprints that will be recreated
	// when deploying the changes.
	// The reason for this will primarily be due to a dependency of a child blueprint
	// being removed from the latest version of the host blueprint.
	RecreateChildren []string `json:"recreateChildren"`
	// RemovedChildren contains the name of the child blueprints that will be removed
	// when deploying the changes.
	RemovedChildren []string `json:"removedChildren"`
	// NewExports contains new fields that will be exported from the blueprint.
	NewExports map[string]provider.FieldChange `json:"newExports"`
	// ExportChanges contains changes to exported fields.
	ExportChanges map[string]provider.FieldChange `json:"exportChanges"`
	// UnchangedExports contains the names of fields that will not be changed.
	UnchangedExports []string `json:"unchangedExports"`
	// RemovedExports contains the names of fields that will no longer be exported.
	RemovedExports []string `json:"removedExports"`
	// MetadataChanges contains changes to blueprint-wide metadata.
	MetadataChanges MetadataChanges `json:"metadataChanges"`
	// ResolveOnDeploy contains paths to properties in blueprint elements
	// that contain substitutions that can not be resolved until the blueprint
	// is deployed.
	// This includes properties in resources, data sources, blueprint-wide metadata
	// and exported fields.
	ResolveOnDeploy []string `json:"resolveOnDeploy"`
}

// IntermediaryBlueprintChanges holds changes to a blueprint that are not yet finalised
// but are stored in temporary state for the duration of the change staging process.
// This differs from blueprint changes in that it holds pointers to change items
// that makes it more efficient to update the changes as the staging process progresses.
type IntermediaryBlueprintChanges struct {
	NewResources     map[string]*provider.Changes
	ResourceChanges  map[string]*provider.Changes
	RemovedResources []string
	RemovedLinks     []string
	NewChildren      map[string]*NewBlueprintDefinition
	ChildChanges     map[string]*BlueprintChanges
	RemovedChildren  []string
	NewExports       map[string]*provider.FieldChange
	ExportChanges    map[string]*provider.FieldChange
	RemovedExports   []string
	MetadataChanges  *MetadataChanges
	UnchangedExports []string
	ResolveOnDeploy  []string
}

// NewBlueprintDefinition provides a definition for a new child blueprint
// that will be created when deploying a blueprint instance.
type NewBlueprintDefinition struct {
	NewResources map[string]provider.Changes       `json:"newResources"`
	NewChildren  map[string]NewBlueprintDefinition `json:"newChildren"`
	NewExports   map[string]provider.FieldChange   `json:"newExports"`
}
