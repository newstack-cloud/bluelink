package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Link provides the interface for the implementation of a link between two resources.
type Link interface {
	// StageChanges must detail the changes that will be made when a deployment of the loaded blueprint
	// for the link between two resources.
	// Unlike resources, links do not map to a specification for a single deployable unit,
	// so link implementations must specify the changes that will be made across multiple resources.
	StageChanges(
		ctx context.Context,
		input *LinkStageChangesInput,
	) (*LinkStageChangesOutput, error)
	// UpdateResourceA deals with applying the changes to the first of the two linked resources
	// for the creation or removal of a link between two resources.
	// The value of the `LinkData` field returned in the output will be combined
	// with the LinkData output from updating resource B and intermediary resources
	// to form the final LinkData that will be persisted in the state of the blueprint instance.
	// Parameters are passed into UpdateResourceA for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	UpdateResourceA(ctx context.Context, input *LinkUpdateResourceInput) (*LinkUpdateResourceOutput, error)
	// UpdateResourceB deals with applying the changes to the second of the two linked resources
	// for the creation or removal of a link between two resources.
	// The value of the `LinkData` field returned in the output will be combined
	// with the LinkData output from updating resource A and intermediary resources
	// to form the final LinkData that will be persisted in the state of the blueprint instance.
	// Parameters are passed into UpdateResourceB for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	UpdateResourceB(ctx context.Context, input *LinkUpdateResourceInput) (*LinkUpdateResourceOutput, error)
	// UpdateIntermediaryResources deals with creating, updating or deleting intermediary resources
	// that are required for the link between two resources.
	//
	// An intermediary resource can be an existing resource in the blueprint that will be modified
	// by the link, or a new resource that will be completely managed by the link implementation,
	// where the link implementation will be responsible for creating, updating and deleting
	// the intermediary resource as part of the link lifecycle.
	// The ResourceDeployService provided in the input must only be used for intermediary resources
	// fully managed by the link implementation.
	// For existing resources in the blueprint, direct service calls to the upstream provider
	// for the resource should be made and the link data will be overlayed or merged with the resource state
	// of the top-level resource in the blueprint to account for changes made by the link
	// when checking for drift, change staging (or planning) is applied only to resource and link state
	// in the blueprint framework model and not to the current state of the upstream provider.
	//
	// This is called for both the creation and removal of a link between two resources.
	// The value of the `LinkData` field returned in the output will be combined
	// with the LinkData output from updating resource A and B
	// to form the final LinkData that will be persisted in the state of the blueprint instance.
	// Parameters are passed into UpdateIntermediaryResources for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	UpdateIntermediaryResources(
		ctx context.Context,
		input *LinkUpdateIntermediaryResourcesInput,
	) (*LinkUpdateIntermediaryResourcesOutput, error)
	// GetPriorityResource retrieves the resource in the relationship
	// that must be deployed first. This will be empty for links where one resource does not
	// need to be deployed before the other.
	GetPriorityResource(ctx context.Context, input *LinkGetPriorityResourceInput) (*LinkGetPriorityResourceOutput, error)
	// GetType deals with retrieving the type of the link in relation to the two resource
	// types it provides a relationship between.
	GetType(ctx context.Context, input *LinkGetTypeInput) (*LinkGetTypeOutput, error)
	// GetTypeDescription deals with retrieving the description for a link type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetTypeDescription(ctx context.Context, input *LinkGetTypeDescriptionInput) (*LinkGetTypeDescriptionOutput, error)
	// GetAnnotationDefinitions retrieves the annotation definitions for the link type.
	// Annotations provide a way to fine tune the behaviour of a link in a blueprint spec
	// in the linked resource metadata sections.
	GetAnnotationDefinitions(ctx context.Context, input *LinkGetAnnotationDefinitionsInput) (*LinkGetAnnotationDefinitionsOutput, error)
	// GetKind tells us whether the link is a "hard" or "soft" link.
	// A hard link is where the priority resource type must be created first.
	// A soft link is where it does not matter which resource type in the relationship
	// is created first.
	GetKind(ctx context.Context, input *LinkGetKindInput) (*LinkGetKindOutput, error)
}

// LinkStageChangesInput provides the input required to
// stage changes for a link between two resources.
type LinkStageChangesInput struct {
	ResourceAChanges *Changes
	ResourceBChanges *Changes
	CurrentLinkState *state.LinkState
	LinkContext      LinkContext
}

// LinkStageChangesOutput provides the output from staging changes
// for a link between two resources.
type LinkStageChangesOutput struct {
	Changes *LinkChanges
}

// LinkUpdateResourceInput provides the input required to
// update a resource in a link relationship
// with data that will contribute to "activating" or "de-activating" the link.
type LinkUpdateResourceInput struct {
	Changes           *LinkChanges
	ResourceInfo      *ResourceInfo
	OtherResourceInfo *ResourceInfo
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName   string
	LinkUpdateType LinkUpdateType
	LinkContext    LinkContext
}

// LinkUpdateType represents the type of update that is being carried out
// for a link between two resources.
type LinkUpdateType int

const (
	// LinkUpdateTypeCreate is used when a link is being created.
	LinkUpdateTypeCreate LinkUpdateType = iota
	// LinkUpdateTypeDestroy is used when a link is being destroyed.
	LinkUpdateTypeDestroy
	// LinkUpdateTypeUpdate is used when a link is being updated.
	LinkUpdateTypeUpdate
)

// LinkUpdateResourceOutput provides the output from updating
// a resource in a link relationship.
type LinkUpdateResourceOutput struct {
	LinkData *core.MappingNode
	// ResourceDataMappings provides mappings of resource spec fields
	// to the link data fields created when updating one of the two
	// resources in a link relationship.
	// The format is:
	// {resourceName}::{fieldPath} -> {linkDataFieldPath}
	// e.g. "orderServiceRole::spec.policy.name" -> "orderServiceRole.policy"
	// This is useful for applying link data projections to resources to take
	// link changes into account when checking for drift.
	//
	// {resourceName} represents the logical name of the resource in single blueprint instance.
	ResourceDataMappings map[string]string
}

// LinkUpdateIntermediaryResourcesInput provides the input required to
// update intermediary resources in a link relationship.
type LinkUpdateIntermediaryResourcesInput struct {
	ResourceAInfo *ResourceInfo
	ResourceBInfo *ResourceInfo
	Changes       *LinkChanges
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName   string
	LinkUpdateType LinkUpdateType
	LinkContext    LinkContext
	// ResourceDeployService allows a link implementation to hook into
	// the framework's existing mechanism to manage resource deployments,
	// this is useful as it allows link implementations to use the same
	// resources used in blueprints.
	ResourceDeployService ResourceDeployService
	// ResourceLookupService allows a link implementation to look up resources
	// in the blueprint state.
	// This is useful for link implementations that need to check if an intermediary
	// resource exists in the blueprint for intermediary resources that are not fully managed
	// by the link implementation.
	// For example, in AWS, an IAM role would be an intermediary resource
	// between an AWS Lambda function and an AWS DynamoDB table,
	// where the IAM role is not fully managed by the link implementation,
	// but is an existing resource in the blueprint that will be modified by the link.
	ResourceLookupService ResourceLookupService
}

type LinkUpdateIntermediaryResourcesOutput struct {
	IntermediaryResourceStates []*state.LinkIntermediaryResourceState
	LinkData                   *core.MappingNode
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
	ResourceDataMappings map[string]string
}

// LinkGetPriorityResourceInput provides the input for retrieving
// the priority resource type in a link relationship.
type LinkGetPriorityResourceInput struct {
	LinkContext LinkContext
}

// LinkPriorityResourceOutput provides the output for retrieving
// the priority resource in a link relationship.
type LinkGetPriorityResourceOutput struct {
	PriorityResource     LinkPriorityResource
	PriorityResourceType string
}

// LinkPriorityResource holds the type of resource that must be deployed first
// in a link relationship.
type LinkPriorityResource int

const (
	// LinkPriorityResourceNone is used when there is no priority resource in the link relationship.
	LinkPriorityResourceNone LinkPriorityResource = iota
	// LinkPriorityResourceA is used when the priority resource is the first resource in the link relationship.
	LinkPriorityResourceA
	// LinkPriorityResourceB is used when the priority resource is the second resource in the link relationship.
	LinkPriorityResourceB
)

// LinkGetKindInput provides the input for retrieving the kind of link.
type LinkGetKindInput struct {
	LinkContext LinkContext
}

// LinkGetKindOutput provides the output for retrieving the kind of link.
type LinkGetKindOutput struct {
	Kind LinkKind
}

// LinkGetTypeOutput provides the output for retrieving the type of link
// with respect to the two resource types it provides a relationship between.
type LinkGetTypeInput struct {
	LinkContext LinkContext
}

// LinkGetTypeOutput provides the output for retrieving the type of link
// with respect to the two resource types it provides a relationship between.
type LinkGetTypeOutput struct {
	Type string
}

// LinkGetTypeDescriptionInput provides the input for retrieving the description
// of a link type in a blueprint spec.
type LinkGetTypeDescriptionInput struct {
	LinkContext LinkContext
}

// LinkGetTypeDescriptionOutput provides the output for retrieving the description
// of a link type in a blueprint spec.
type LinkGetTypeDescriptionOutput struct {
	MarkdownDescription  string
	PlainTextDescription string
	// A short summary of the link type that can be formatted
	// in markdown, this is useful for listing link types in documentation.
	MarkdownSummary string
	// A short summary of the link type in plain text,
	// this is useful for listing link types in documentation.
	PlainTextSummary string
}

// LinkGetAnnotationDefinitionsInput provides the input for retrieving
// the annotation definitions for the link type.
type LinkGetAnnotationDefinitionsInput struct {
	LinkContext LinkContext
}

// LinkGetAnnotationDefinitionsOutput provides the output for retrieving
// the annotation definitions for the link type.
type LinkGetAnnotationDefinitionsOutput struct {
	// A mapping of annotation names prefixed by resource type that
	// can be used to fine tune the behaviour of a link in a blueprint spec.
	// The format should be as follows:
	// {resourceType}::{annotationName} -> LinkAnnotationDefinition
	// e.g. "aws/lambda/function::aws.lambda.dynamodb.accessType" -> LinkAnnotationDefinition
	AnnotationDefinitions map[string]*LinkAnnotationDefinition
}

// LinkAnnotationDefinition provides a way to define annotations
// for a link type.
// Annotations that have dynamic keys should use the `<resourceName>` syntax
// in the key name, e.g. "aws.lambda.dynamodb.<tableName>.accessType".
// The value that "<resourceName>" represents must be the name of a resource
// that is linked to the resource type where the annotation is defined.
// Schema validation will match based on the pattern for dynamic keys.
// Only a single "<resourceName>" placeholder is allowed for a dynamic annotation key.
// Dynamic keys are used to target specific resources when there are multiple resources
// of the same type linked to the resource where the annotation is defined.
// Default values are ignored for link annotation field definitions that have
// dynamic field names, the default value should be defined in an equivalent
// annotation that is not targeted at a specific resource name (e.g. "aws.lambda.dynamodb.accessType").
type LinkAnnotationDefinition struct {
	Name          string              `json:"name"`
	Label         string              `json:"label"`
	Type          core.ScalarType     `json:"type"`
	Description   string              `json:"description"`
	DefaultValue  *core.ScalarValue   `json:"defaultValue,omitempty"`
	AllowedValues []*core.ScalarValue `json:"allowedValues,omitempty"`
	Examples      []*core.ScalarValue `json:"examples,omitempty"`
	Required      bool                `json:"required"`
	// ValidateFunc is a custom validation function that allows for validation
	// of annotation values when provided as literals.
	// When substitutions are used as annotation values (e.g. `${variables.myVar}`),
	// the validation function will not be called, as the value will not be resolved
	// to a concrete value at the validation stage.
	// The function should return a slice of diagnostics, where if at least one diagnostic
	// has a level of Error, overall validation will fail.
	ValidateFunc func(
		key string,
		annotationValue *core.ScalarValue,
	) []*core.Diagnostic `json:"-"`
}

// LinkKind provides a way to categorise links to help determine the order
// in which resources need to be deployed when a blueprint instance is being deployed.
type LinkKind string

const (
	// LinkKindHard is the type of link where the priority resource type
	// must be created before the other resource type in the relationship.
	LinkKindHard LinkKind = "hard"
	// LinkKindSoft is the type of link where it does not matter
	// which of the two resource types in the relationship is created
	// first.
	LinkKindSoft LinkKind = "soft"
)

// LinkChanges provides a set of modified fields for a link between two resources.
// The link field changes represent a set of changes that will be made to the
// resources in the link relationship, these changes should be modelled as per the
// structure of the linkData that is persisted in the state of a blueprint instance.
// The linkData model should be organised by the resource type with a structure
// that is a close approximation of the actual changes that will be made to each
// resource during deployment in the upstream provider.
type LinkChanges struct {
	ModifiedFields  []*FieldChange `json:"modifiedFields"`
	NewFields       []*FieldChange `json:"newFields"`
	RemovedFields   []string       `json:"removedFields"`
	UnchangedFields []string       `json:"unchangedFields"`
	// FieldChangesKnownOnDeploy holds a list of field names
	// for which changes will be known when the host blueprint is deployed.
	FieldChangesKnownOnDeploy []string `json:"fieldChangesKnownOnDeploy"`
}

// LinkContext provides access to information about providers
// and configuration in the current environment.
// Links can live in intermediary provider plugins that can represent a link
// between resources in different providers, for this reason, the LinkContext
// provides access to configuration for all providers loaded into the current
// environment.
type LinkContext interface {
	// ProviderConfigVariable retrieves a configuration value that was loaded
	// for the specified provider.
	ProviderConfigVariable(providerNamespace string, varName string) (*core.ScalarValue, bool)
	// ProviderConfigVariables retrieves all configuration values that were loaded
	// for the specified provider.
	ProviderConfigVariables(providerNamespace string) map[string]*core.ScalarValue
	// AllProviderConfigVariables retrieves all configuration values that were loaded
	// for all providers.
	AllProviderConfigVariables() map[string]map[string]*core.ScalarValue
	// ContextVariable retrieves a context-wide variable
	// for the current environment, this differs from values extracted
	// from context.Context, as these context variables are specific
	// to the components that implement the interfaces of the blueprint library
	// and can be shared between processes over a network or similar.
	ContextVariable(name string) (*core.ScalarValue, bool)
	// ContextVariables returns all context variables
	// for the current environment.
	ContextVariables() map[string]*core.ScalarValue
}
