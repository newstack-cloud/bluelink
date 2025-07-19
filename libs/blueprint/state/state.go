package state

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// Container provides an interface for services
// that persist blueprint instance state.
// Various methods are provided to deal with extracting and saving information
// for a blueprint instance.
// Instead of operating at the instance level and manipulating the entire state,
// methods are provided to deal with sub-components of the instance state
// such as resources, links, metadata and exports.
// Depending on the implementation, it can be more efficient to deal with
// sub-components of the instance state separately.
// Fr example, `Instances().Get` may be a view of the instance state
// that could be an expensive operation to perform involving multiple
// queries to a database or an expensive join operation.
// The state persistence method is entirely up to the application
// making use of this library.
type Container interface {
	// Instances provides functionality to manage state for blueprint instances.
	Instances() InstancesContainer
	// Resources provides functionality to manage state for resources in blueprint instances.
	Resources() ResourcesContainer
	// Links provides functionality to manage state for links in blueprint instances.
	Links() LinksContainer
	// Children provides functionality to manage state for child blueprints in relation
	// to their parent blueprint instances.
	Children() ChildrenContainer
	// Metadata provides functionality to manage metadata for blueprint instances.
	Metadata() MetadataContainer
	// Exports provides functionality to manage exported fields for blueprint instances.
	Exports() ExportsContainer
}

// InstancesContainer provides an interface for functionality related
// to persisting and retrieving the top-level state of a blueprint instance.
type InstancesContainer interface {
	// Get deals with retrieving the state for a given blueprint
	// instance ID.
	Get(ctx context.Context, instanceID string) (InstanceState, error)
	// LookupIDByName deals with retrieving the ID of a blueprint instance
	// given a unique name.
	// This is useful for situations where the user makes requests to carry
	// out an action, only providing a name that they have defined.
	// This exists to allow the majority of the deployment/change staging
	// system to work based off of an instance ID
	// while allowing the user to provide their own name to carry out an
	// action on an instance.
	LookupIDByName(ctx context.Context, instanceName string) (string, error)
	// Save deals with persisting a blueprint instance.
	Save(ctx context.Context, instanceState InstanceState) error
	// UpdateStatus deals with updating the status of the latest blueprint
	// instance deployment.
	UpdateStatus(
		ctx context.Context,
		instanceID string,
		statusInfo InstanceStatusInfo,
	) error
	// Remove deals with removing the state for a given blueprint instance.
	// This is not for destroying the actual deployed resources, just removing the state.
	Remove(ctx context.Context, instanceID string) (InstanceState, error)
}

// ResourcesContainer provides an interface for functionality related
// to persisting and retrieving resource state in a blueprint instance.
type ResourcesContainer interface {
	// Get deals with retrieving resource state for a given globally unique
	// resource ID.
	Get(ctx context.Context, resourceID string) (ResourceState, error)
	// GetByName deals with retrieving the state for a given resource
	// in the provided blueprint instance by its logical name.
	GetByName(ctx context.Context, instanceID string, resourceName string) (ResourceState, error)
	// Save deals with persisting a resource in a blueprint instance.
	// This both creates a resource with the given unique ID and attaches
	// it to the blueprint instance ID in the provided resource state structure.
	Save(
		ctx context.Context,
		resourceState ResourceState,
	) error
	// UpdateStatus deals with updating the status of the latest deployment of a given resource.
	UpdateStatus(
		ctx context.Context,
		resourceID string,
		statusInfo ResourceStatusInfo,
	) error
	// Remove deals with removing the state of a resource from the system.
	Remove(ctx context.Context, resourceID string) (ResourceState, error)
	// GetDrift deals with retrieving the current drift state for a given resource.
	GetDrift(ctx context.Context, resourceID string) (ResourceDriftState, error)
	// SaveDrift deals with persisting the drift state for a given resource.
	SaveDrift(ctx context.Context, driftState ResourceDriftState) error
	// RemoveDrift deals with removing the drift state for a given resource.
	RemoveDrift(ctx context.Context, resourceID string) (ResourceDriftState, error)
}

// LinksContainer provides an interface for functionality related
// to persisting and retrieving link state in a blueprint instance.
type LinksContainer interface {
	// Get deals with retrieving the link state for a given globally unique
	// link ID.
	Get(ctx context.Context, linkID string) (LinkState, error)
	// GetByName deals with retrieving the state for a given link
	// in the provided blueprint instance by its logical name ({resourceA}::{resourceB}).
	GetByName(ctx context.Context, instanceID string, linkName string) (LinkState, error)
	// ListWithResourceDataMappings deals with retrieving all links in a given blueprint instance
	// that have resource data mappings that map to a spec field in the resource with the given
	// name in the blueprint instance.
	ListWithResourceDataMappings(
		ctx context.Context,
		instanceID string,
		resourceName string,
	) ([]LinkState, error)
	// Save deals with persisting a link in the system and attaching it to the blueprint instance
	// for the instance ID in the provided link state structure.
	Save(ctx context.Context, linkState LinkState) error
	// UpdateStatus deals with updating the status of the latest deployment of a given link.
	UpdateStatus(
		ctx context.Context,
		linkID string,
		statusInfo LinkStatusInfo,
	) error
	// Remove deals with removing the state of a link from
	// the system
	Remove(ctx context.Context, linkID string) (LinkState, error)
}

// ChildrenContainer provides an interface for functionality related
// to persisting and retrieving child blueprint state in relation to
// a parent blueprint instance.
type ChildrenContainer interface {
	// Get deals with retrieving the state for a given child blueprint
	// in the provided blueprint instance.
	Get(ctx context.Context, instanceID string, childName string) (InstanceState, error)
	// Attach a blueprint instance as a child of the specified parent instance.
	// Both the parent and child blueprint instances must already exist.
	Attach(
		ctx context.Context,
		parentInstanceID string,
		childInstanceID string,
		childName string,
	) error
	// SaveDependencies deals with persisting the dependencies of a child blueprint
	// in relation to other elements in the parent blueprint instance.
	SaveDependencies(
		ctx context.Context,
		instanceID string,
		childName string,
		dependencies *DependencyInfo,
	) error
	// Detach deals with removing the relationship between a child blueprint
	// and its parent blueprint instance.
	// This will not remove the child blueprint instance itself,
	// instances.Remove should be used to completely remove the child blueprint instance.
	Detach(ctx context.Context, instanceID string, childName string) error
}

// MetadataContainer provides an interface for functionality related
// to persisting and retrieving metadata for a blueprint instance.
type MetadataContainer interface {
	// Get deals with retrieving metadata for a given blueprint instance.
	Get(ctx context.Context, instanceID string) (map[string]*core.MappingNode, error)
	// Save deals with persisting metadata for a given blueprint instance.
	Save(ctx context.Context, instanceID string, metadata map[string]*core.MappingNode) error
	// Remove deals with removing metadata from a given blueprint instance.
	Remove(ctx context.Context, instanceID string) (map[string]*core.MappingNode, error)
}

// ExportsContainer provides an interface for functionality related
// to persisting and retrieving exported fields for a blueprint instance.
type ExportsContainer interface {
	// GetAll deals with retrieving exported fields for a given blueprint instance.
	GetAll(ctx context.Context, instanceID string) (map[string]*ExportState, error)
	// Get deals with retrieving an exported field for a given blueprint instance.
	Get(ctx context.Context, instanceID string, exportName string) (ExportState, error)
	// SaveAll deals with persisting exported fields for a given blueprint instance.
	SaveAll(ctx context.Context, instanceID string, exports map[string]*ExportState) error
	// Save deals with persisting an exported field for a given blueprint instance.
	Save(ctx context.Context, instanceID string, exportName string, export ExportState) error
	// RemoveAll deals with removing all exported fields for a given blueprint instance.
	RemoveAll(ctx context.Context, instanceID string) (map[string]*ExportState, error)
	// Remove deals with removing an exported field for a given blueprint instance.
	Remove(ctx context.Context, instanceID string, exportName string) (ExportState, error)
}

// Element provides a convenience interface for elements in blueprint state
// to extract common information such as an ID, logical name and type.
type Element interface {
	ID() string
	LogicalName() string
	Kind() ElementKind
}

// ElementKind represents the kind of an element in a blueprint instance.
type ElementKind string

const (
	// ChildElement represents a child blueprint in a blueprint instance.
	ChildElement ElementKind = "child"
	// ResourceElement represents a resource in a blueprint instance.
	ResourceElement ElementKind = "resource"
	// LinkElement represents a link in a blueprint instance.
	LinkElement ElementKind = "link"
)

// ResourceState provides the current state of a resource
// in a blueprint instance.
// This includes the status, the Raw data from the upstream resouce provider
// along with reasons for failure when a resource is in a failure state.
type ResourceState struct {
	// A globally unique identifier for the resource.
	ResourceID string `json:"id"`
	// The logical name of the resource in the blueprint.
	Name string `json:"name"`
	// The type of the resource as defined in the source blueprint.
	Type string `json:"type"`
	// The name of the resource template in the source blueprint
	// that the resource is derived from.
	// This will be empty if the resource is not derived from a resource template.
	TemplateName  string                     `json:"templateName,omitempty"`
	InstanceID    string                     `json:"instanceId"`
	Status        core.ResourceStatus        `json:"status"`
	PreciseStatus core.PreciseResourceStatus `json:"preciseStatus"`
	// LastStatusUpdateTimestamp holds the unix timestamp when the resource deployment
	// status was last updated.
	LastStatusUpdateTimestamp int `json:"lastStatusUpdateTimestamp,omitempty"`
	// LastDeployedTimestamp holds the unix timestamp when the resource was last deployed.
	LastDeployedTimestamp int `json:"lastDeployedTimestamp"`
	// LastDeployAttempTimestamp holds the unix timestamp when an attempt was last made to deploy the resource.
	LastDeployAttemptTimestamp int `json:"lastDeployAttemptTimestamp"`
	// SpecData holds the resolved resource spec
	// for the currently deployed version of the resource along with computed
	// fields derived from the deployed resource in the provider.
	SpecData *core.MappingNode `json:"specData"`
	// Description holds a human-friendly description of the resource derived
	// from a source blueprint.
	Description string `json:"description,omitempty"`
	// Metadata holds metadata for the resource that is derived from a source blueprint
	// that includes additional information that allows for extensions built on top of the
	// blueprint framework along with the storage of labels, annotations and a human-friendly
	// display name for the resource.
	Metadata *ResourceMetadataState `json:"metadata,omitempty"`
	// DependsOnResources holds a list of resource names that the resource depends on,
	// this dependency is derived from "hard" links, references and the use of the dependsOn
	// property in the source blueprint.
	DependsOnResources []string `json:"dependsOnResources,omitempty"`
	// DependsOnChildren holds a list of child blueprint names that the resource depends on.
	// This dependency is derived from references in the source blueprint.
	DependsOnChildren []string `json:"dependsOnChildren,omitempty"`
	// Holds the latest reasons for failures in deploying a resource,
	// this only ever holds the results of the latest deployment attempt.
	FailureReasons []string `json:"failureReasons"`
	// Drifted indicates whether or not the resource state has drifted
	// due to changes in the upstream provider.
	Drifted bool `json:"drifted,omitempty"`
	// LastDriftDetectedTimestamp holds the unix timestamp when drift was last detected.
	LastDriftDetectedTimestamp *int `json:"lastDriftDetectedTimestamp,omitempty"`
	// Durations holds duration information for the latest deployment of the resource.
	Durations *ResourceCompletionDurations `json:"durations,omitempty"`
}

func (r *ResourceState) ID() string {
	return r.ResourceID
}

func (r *ResourceState) LogicalName() string {
	return r.Name
}

func (r *ResourceState) Kind() ElementKind {
	return ResourceElement
}

// ResourceMetadataState holds metadata for a resource
// that is derived from a source blueprint.
type ResourceMetadataState struct {
	DisplayName string                       `json:"displayName,omitempty"`
	Annotations map[string]*core.MappingNode `json:"annotations,omitempty"`
	Labels      map[string]string            `json:"labels,omitempty"`
	Custom      *core.MappingNode            `json:"custom,omitempty"`
}

// ResourceDriftState holds information about how a resource has drifted
// from the current state persisted in the blueprint framework.
type ResourceDriftState struct {
	// A globally unique identifier for the resource the drift state is for.
	ResourceID string `json:"resourceId"`
	// The logical name of the resource in the blueprint the drift state is for.
	ResourceName string `json:"resourceName"`
	// SpecData holds the resource spec
	// for the drifted version of the resource derived
	// from the upstream provider.
	SpecData *core.MappingNode `json:"specData"`
	// Difference holds the changes that have been detected
	// in the resource state in the upstream provider.
	// This holds a representation of changes from the current
	// state to the drifted state.
	Difference *ResourceDriftChanges `json:"difference"`
	// Timestamp holds the unix timestamp of when the drift
	// was detected.
	Timestamp *int `json:"timestamp,omitempty"`
}

// ResourceDriftChanges holds the changes that have been detected
// in the resource state in the upstream provider.
type ResourceDriftChanges struct {
	ModifiedFields  []*ResourceDriftFieldChange `json:"modifiedFields"`
	NewFields       []*ResourceDriftFieldChange `json:"newFields"`
	RemovedFields   []string                    `json:"removedFields"`
	UnchangedFields []string                    `json:"unchangedFields"`
}

// ResourceDriftFieldChange represents a change in a field value
// of a resource that is used in drift detection.
type ResourceDriftFieldChange struct {
	// FieldPath holds the path of the field in the resource spec.
	// For example, "spec.template.spec.containers[0].image".
	FieldPath string `json:"fieldPath"`
	// StateValue holds the value of the field in the current state
	// persisted in the blueprint framework.
	StateValue *core.MappingNode `json:"stateValue"`
	// DriftedValue holds the value of the field in the drifted state
	// in the upstream provider.
	DriftedValue *core.MappingNode `json:"driftedValue"`
}

// ResourceStatusInfo holds information about the status of a resource
// that is primarily used when updating the status of a resource.
type ResourceStatusInfo struct {
	Status                     core.ResourceStatus          `json:"status"`
	PreciseStatus              core.PreciseResourceStatus   `json:"preciseStatus"`
	LastDeployedTimestamp      *int                         `json:"lastDeployedTimestamp,omitempty"`
	LastDeployAttemptTimestamp *int                         `json:"lastDeployAttemptTimestamp,omitempty"`
	LastStatusUpdateTimestamp  *int                         `json:"lastStatusUpdateTimestamp,omitempty"`
	Durations                  *ResourceCompletionDurations `json:"durations,omitempty"`
	FailureReasons             []string                     `json:"failureReasons,omitempty"`
}

// InstanceState stores the state of a blueprint instance
// including resources, metadata, exported fields and child blueprints.
type InstanceState struct {
	InstanceID string `json:"id"`
	// InstanceName holds a user-defined globally unique name for the blueprint instance.
	// In multi-tenant systems this expected to be prefixed by an account or organisation
	// name or ID.
	InstanceName string              `json:"name"`
	Status       core.InstanceStatus `json:"status"`
	// LastStatusUpdateTimestamp holds the unix timestamp when the blueprint instance deployment
	// status was last updated.
	LastStatusUpdateTimestamp int `json:"lastStatusUpdateTimestamp,omitempty"`
	// LastDeployedTimestamp holds the unix timestamp when the blueprint instance was last deployed.
	LastDeployedTimestamp int `json:"lastDeployedTimestamp"`
	// LastDeployAttempTimestamp holds the unix timestamp when an attempt
	// was last made to deploy the blueprint instance.
	LastDeployAttemptTimestamp int `json:"lastDeployAttemptTimestamp"`
	// A mapping of logical resource definition name
	// to the resource IDs
	// that are created from the resource definition.
	ResourceIDs map[string]string `json:"resourceIds"`
	// A mapping of resource IDs to the resource state.
	Resources map[string]*ResourceState `json:"resources"`
	// A mapping of logical link definition names
	// to the state of each link in the blueprint instance.
	Links map[string]*LinkState `json:"links"`
	// Metadata is used internally to store additional non-structured information
	// that is relevant to the blueprint framework but can also be used to store
	// additional information that is relevant to the application/tool
	// making use of the framework.
	Metadata        map[string]*core.MappingNode `json:"metadata"`
	Exports         map[string]*ExportState      `json:"exports"`
	ChildBlueprints map[string]*InstanceState    `json:"childBlueprints"`
	// ChildDependencies holds a mapping of child blueprint names to their dependencies.
	ChildDependencies map[string]*DependencyInfo `json:"childDependencies,omitempty"`
	// Durations holds duration information for the latest deployment of the blueprint instance.
	Durations *InstanceCompletionDuration `json:"durations,omitempty"`
}

// ExportState holds state that is persisted for an export
// in a blueprint instance.
type ExportState struct {
	// Value holds the resolved exported value.
	Value *core.MappingNode `json:"value"`
	// Type holds the type of the exported value.
	Type schema.ExportType `json:"type"`
	// Description holds a human-friendly description of the export.
	Description string `json:"description,omitempty"`
	// Field holds the path of a field in a blueprint element
	// that should be exported.
	Field string `json:"field"`
}

// InstanceStatusInfo holds information about the status of a blueprint instance
// that is primarily used when updating the status of a blueprint instance.
type InstanceStatusInfo struct {
	Status                     core.InstanceStatus         `json:"status"`
	FailureReasons             []string                    `json:"failureReasons,omitempty"`
	LastDeployedTimestamp      *int                        `json:"lastDeployedTimestamp,omitempty"`
	LastDeployAttemptTimestamp *int                        `json:"lastDeployAttemptTimestamp,omitempty"`
	LastStatusUpdateTimestamp  *int                        `json:"lastStatusUpdateTimestamp,omitempty"`
	Durations                  *InstanceCompletionDuration `json:"durations,omitempty"`
}

// ChildBlueprint holds the state of a child blueprint
// along with its logical name in the parent blueprint.
type ChildBlueprint struct {
	ChildName string         `json:"childName"`
	State     *InstanceState `json:"state"`
}

func WrapChildBlueprintInstance(childName string, instance *InstanceState) *ChildBlueprint {
	return &ChildBlueprint{
		ChildName: childName,
		State:     instance,
	}
}

func (b *ChildBlueprint) ID() string {
	return b.State.InstanceID
}

func (b *ChildBlueprint) LogicalName() string {
	return b.ChildName
}

func (b *ChildBlueprint) Kind() ElementKind {
	return ChildElement
}

// ChildDependencyInfo holds information about the dependencies
// of a child blueprint or resource.
type DependencyInfo struct {
	// DependsOnResources holds a list of resource IDs that the
	// child blueprint or resource depends on.
	DependsOnResources []string `json:"dependsOnResources,omitempty"`
	// DependsOnChildren holds a list of child blueprint names that the
	// child blueprint or resource depends on.
	DependsOnChildren []string `json:"dependsOnChildren,omitempty"`
}

// LinkState provides a way to store some state for links between
// resources.
// This is useful for holding state about intermediary resources
// managed by a provider's implementation of a link in addition
// to tracking the deployment status for a link.
type LinkState struct {
	// A globally unique identifier for the link.
	LinkID string `json:"id"`
	// The logical name of the link in the blueprint.
	// This is a combination of the logical names of the 2 resources that are linked.
	// For example, if a link is between a VPC and a subnet,
	// the link name would be "vpc::subnet".
	Name          string                 `json:"name"`
	InstanceID    string                 `json:"instanceId"`
	Status        core.LinkStatus        `json:"status"`
	PreciseStatus core.PreciseLinkStatus `json:"preciseStatus"`
	// LastStatusUpdateTimestamp holds the unix timestamp when the link deployment
	// status was last updated.
	LastStatusUpdateTimestamp int `json:"lastStatusUpdateTimestamp,omitempty"`
	// LastDeployedTimestamp holds the unix timestamp when the link was last deployed.
	LastDeployedTimestamp int `json:"lastDeployedTimestamp"`
	// LastDeployAttempTimestamp holds the unix timestamp when an attempt was last made to deploy the link.
	LastDeployAttemptTimestamp int `json:"lastDeployAttemptTimestamp"`
	// IntermediaryResourceStates holds the state of intermediary resources
	// that are created by the provider's implementation of a link.
	IntermediaryResourceStates []*LinkIntermediaryResourceState `json:"intermediaryResourceStates"`
	// Data is the map that holds the structure of
	// the "raw" link data to hold information about a link that is not
	// stored directly in the resources that are linked and is not
	// stored in intermediary resources.
	// This should hold information that may include values that are populated
	// in one or both of the resources in the link relationship along with
	// intermediary resources.
	Data map[string]*core.MappingNode `json:"data"`
	// ResourceDataMappings provides mappings of resource spec fields
	// to the link data fields created when updating intermediary resources
	// in a link relationship.
	// The format is:
	// {resourceName}::{fieldPath} -> {linkDataFieldPath}
	// e.g. "orderServiceRole::spec.policy.name" -> "orderServiceRole.policy"
	// This is useful for applying link data projections to resources to take
	// link changes into account when checking for drift.
	//
	// {resourceName} represents the logical name of the resource in single blueprint instance.
	ResourceDataMappings map[string]string `json:"resourceDataMappings,omitempty"`
	// Holds the latest reasons for failures in deploying a link,
	// this only ever holds the results of the latest deployment attempt.
	FailureReasons []string `json:"failureReasons"`
	// Durations holds duration information for the latest deployment of the link.
	Durations *LinkCompletionDurations `json:"durations,omitempty"`
}

func (l *LinkState) ID() string {
	return l.LinkID
}

func (l *LinkState) LogicalName() string {
	return l.Name
}

func (l *LinkState) Kind() ElementKind {
	return LinkElement
}

// LinkIntermediaryResourceState holds information about the state
// of an intermediary resources created for a link.
type LinkIntermediaryResourceState struct {
	// A globally unique identifier for the resource.
	ResourceID    string                     `json:"id"`
	ResourceType  string                     `json:"type"`
	InstanceID    string                     `json:"instanceId"`
	Status        core.ResourceStatus        `json:"status"`
	PreciseStatus core.PreciseResourceStatus `json:"preciseStatus"`
	// LastDeployedTimestamp holds the unix timestamp when the resource was last deployed.
	LastDeployedTimestamp int `json:"lastDeployedTimestamp"`
	// LastDeployAttempTimestamp holds the unix timestamp when an attempt was last made to deploy the resource.
	LastDeployAttemptTimestamp int `json:"lastDeployAttemptTimestamp"`
	// ResourceSpecData holds the resolved resource spec
	// for the currently deployed version of the resource along with computed
	// fields derived from the deployed resource in the provider.
	ResourceSpecData *core.MappingNode `json:"resourceSpecData"`
	FailureReasons   []string          `json:"failureReasons,omitempty"`
}

// LinkStatusInfo holds information about the status of a link
// that is primarily used when updating the status of a link.
type LinkStatusInfo struct {
	Status                     core.LinkStatus          `json:"status"`
	PreciseStatus              core.PreciseLinkStatus   `json:"preciseStatus"`
	LastDeployedTimestamp      *int                     `json:"lastDeployedTimestamp,omitempty"`
	LastDeployAttemptTimestamp *int                     `json:"lastDeployAttemptTimestamp,omitempty"`
	LastStatusUpdateTimestamp  *int                     `json:"lastStatusUpdateTimestamp,omitempty"`
	Durations                  *LinkCompletionDurations `json:"durations,omitempty"`
	FailureReasons             []string                 `json:"failureReasons,omitempty"`
}

// ResourceCompletionDurations holds duration information
// for the deployment of a resource change.
type ResourceCompletionDurations struct {
	// ConfigCompleteDuration is the duration in milliseconds for the resource to be configured.
	// This will only be present if the resource has reached config complete status.
	ConfigCompleteDuration *float64 `json:"configCompleteDuration,omitempty"`
	// TotalDuration is the duration in milliseconds for the resource change to reach the final
	// status.
	TotalDuration *float64 `json:"totalDuration,omitempty"`
	// AttemptDurations holds a list of durations in milliseconds
	// for each attempt to deploy the resource.
	// Attempt durations are in order as per the "Attempt" field in a status update message.
	AttemptDurations []float64 `json:"attemptDurations,omitempty"`
}

// LinkCompletionDurations holds duration information
// for the deployment of a link change.
type LinkCompletionDurations struct {
	// ResourceAUpdate is the duration information for the update of resource A in the link.
	// This will only be present if the link has reached resource A updated status.
	ResourceAUpdate *LinkComponentCompletionDurations `json:"resourceAUpdate,omitempty"`
	// ResourceBUpdate is the duration information for the update of resource B in the link.
	// This will only be present if the link has reached resource B updated status.
	ResourceBUpdate *LinkComponentCompletionDurations `json:"resourceBUpdate,omitempty"`
	// IntermediaryResources is the duration information for the update, creation or removal
	// of intermediary resources in the link.
	// This will only be present if the link has reached intermediary resources updated status.
	IntermediaryResources *LinkComponentCompletionDurations `json:"intermediaryResources,omitempty"`
	// TotalDuration is the duration in milliseconds for the link change to reach the final
	// status.
	TotalDuration *float64 `json:"totalDuration,omitempty"`
}

type LinkComponentCompletionDurations struct {
	// TotalDuration is the duration in milliseconds for the link component
	// change to reach the final status.
	TotalDuration *float64 `json:"totalDuration,omitempty"`
	// AttemptDurations holds a list of durations in milliseconds
	// for each attempt to deploy the link component.
	// Attempt durations are in order as per the "Attempt" field in a status update message.
	AttemptDurations []float64 `json:"attemptDurations,omitempty"`
}

// InstanceCompletionDuration holds duration information
// for the deployment of a blueprint instance.
type InstanceCompletionDuration struct {
	// PrepareDuration is the duration in milliseconds for the preparation phase
	// of a blueprint instance deployment to be completed.
	PrepareDuration *float64 `json:"prepareDuration,omitempty"`
	// TotalDuration is the duration in milliseconds for the blueprint instance to reach the final
	// status.
	TotalDuration *float64 `json:"totalDuration,omitempty"`
}
