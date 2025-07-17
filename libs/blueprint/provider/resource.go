package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Resource provides the interface for a resource
// that a provider can contain which includes logic for validating,
// transforming, linking and deploying a resource.
type Resource interface {
	// CustomValidate provides support for custom validation that goes beyond
	// the spec schema validation provided by the resource's spec definition.
	CustomValidate(ctx context.Context, input *ResourceValidateInput) (*ResourceValidateOutput, error)
	// GetSpecDefinition retrieves the spec definition for a resource,
	// this is the first line of validation for a resource in a blueprint and is also
	// useful for validating references to a resource instance
	// in a blueprint and for providing definitions for docs and tooling.
	// The spec defines both the schema for the resource spec fields that can be defined
	// by users in a blueprint and computed fields that are derived from the deployed
	// resource in the external provider (e.g. Lambda ARN in AWS).
	GetSpecDefinition(ctx context.Context, input *ResourceGetSpecDefinitionInput) (*ResourceGetSpecDefinitionOutput, error)
	// CanLinkTo specifices the list of resource types the current resource type
	// can link to.
	CanLinkTo(ctx context.Context, input *ResourceCanLinkToInput) (*ResourceCanLinkToOutput, error)
	// GetStabilisedDependencies retrieves a list of resource types that must be stabilised
	// before the current resource can be deployed when another resource of one of the specified types
	// is a dependency of the current resource in a blueprint.
	GetStabilisedDependencies(ctx context.Context, input *ResourceStabilisedDependenciesInput) (*ResourceStabilisedDependenciesOutput, error)
	// IsCommonTerminal specifies whether this resource is expected to have a common use-case
	// as a terminal resource that does not link out to other resources.
	// This is useful for providing useful warnings to users about their blueprints
	// without overloading them with warnings for all resources that don't have any outbound
	// links that could have.
	IsCommonTerminal(ctx context.Context, input *ResourceIsCommonTerminalInput) (*ResourceIsCommonTerminalOutput, error)
	// GetType deals with retrieving the namespaced type for a resource in a blueprint spec.
	GetType(ctx context.Context, input *ResourceGetTypeInput) (*ResourceGetTypeOutput, error)
	// GetTypeDescription deals with retrieving the description for a resource type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetTypeDescription(ctx context.Context, input *ResourceGetTypeDescriptionInput) (*ResourceGetTypeDescriptionOutput, error)
	// GetExamples deals with retrieving a list of examples for a resource type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetExamples(ctx context.Context, input *ResourceGetExamplesInput) (*ResourceGetExamplesOutput, error)
	// Deploy deals with deploying a resource with the upstream resource provider.
	// The behaviour of deploy is to create or update the resource configuration and return the resource
	// spec state once the configuration has been created or updated.
	// Deploy should not wait for the resource to be in a stable state before returning,
	// the framework will call the HasStabilised method periodically when waiting for a resource
	// to stabilise.
	// Parameters are passed into Deploy for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	Deploy(ctx context.Context, input *ResourceDeployInput) (*ResourceDeployOutput, error)
	// HasStabilised deals with checking if a resource has stabilised after being deployed.
	// This is important for resources that require a stable state before other resources can be deployed.
	// This is only used when creating or updating a resource, not when destroying a resource.
	HasStabilised(ctx context.Context, input *ResourceHasStabilisedInput) (*ResourceHasStabilisedOutput, error)
	// GetExternalState deals with getting the state of the resource from the resource provider.
	// (e.g. AWS or Google Cloud)
	GetExternalState(ctx context.Context, input *ResourceGetExternalStateInput) (*ResourceGetExternalStateOutput, error)
	// Destroy deals with destroying a resource instance if its current
	// state is successfully deployed or cleaning up a corrupt or partially deployed
	// resource instance.
	// The resource instance should be completely removed from the external provider
	// as a result of this operation; this is essential for when
	// another element to be removed from a blueprint
	// requires a resource to be completely removed from the external provider.
	// There is no "config complete" equivalent for destroying a resource and
	// "HasStabilised" is designed to be used for resources being created or
	// updated.
	Destroy(ctx context.Context, input *ResourceDestroyInput) error
}

// ResourceInfo provides all the information needed for a resource
// including the blueprint schema data with annotations, labels
// and the spec as a core mapping node.
type ResourceInfo struct {
	// ResourceID holds the ID of a resource when in the context
	// of a blueprint instance when deploying or staging changes.
	// Sometimes staging changes is independent of an instance and is used to compare
	// two versions of a blueprint in which
	// case the resource ID will be empty.
	ResourceID string `json:"resourceId"`
	// ResourceName holds the name of the resource in the blueprint spec.
	// This is useful for new resources that do not have any current resource state.
	ResourceName string `json:"resourceName"`
	// InstanceID holds the ID of the blueprint instance
	// that the current resource belongs to.
	// This could be empty if the resource is being staged
	// for an initial deployment.
	InstanceID string `json:"instanceId"`
	// CurrentResourceState holds the current state of the resource
	// for which changes are being staged.
	// The ResourceInfo struct is passed into resource implementation plugins
	// for deployment.
	// A resource implementation should be guarded from the details of where
	// the state is stored and how it is retrieved,
	// the state should be provided to resource plugins by the blueprint
	// engine.
	// If this is a nil pointer, it means that the resource is new and does not have
	// a current state.
	CurrentResourceState *state.ResourceState `json:"currentResourceState"`
	// ResourceWithResolvedSubs holds a version of a resource for which all ${..}
	// substitutions have been applied.
	ResourceWithResolvedSubs *ResolvedResource `json:"resourceWithResolvedSubs"`
}

// ResolvedResource provides a version of a resource for which all ${..}
// substitutions have been applied.
// Mapping nodes replace StringOrSubstitutions from the blueprint schema representation
// of the resource.
type ResolvedResource struct {
	Type         *schema.ResourceTypeWrapper `json:"type"`
	Description  *core.MappingNode           `json:"description,omitempty"`
	Metadata     *ResolvedResourceMetadata   `json:"metadata,omitempty"`
	Condition    *ResolvedResourceCondition  `json:"condition,omitempty"`
	LinkSelector *schema.LinkSelector        `json:"linkSelector,omitempty"`
	Spec         *core.MappingNode           `json:"spec"`
}

// ResolvedResourceMetadata provides a resolved version of the metadata
// for a resource where all substitutions have been applied.
type ResolvedResourceMetadata struct {
	DisplayName *core.MappingNode `json:"displayName,omitempty"`
	Annotations *core.MappingNode `json:"annotations,omitempty"`
	Labels      *schema.StringMap `json:"labels,omitempty"`
	Custom      *core.MappingNode `json:"custom,omitempty"`
}

// ResolvedResourceCondition provides a resolved version of the condition
// for a resource where all substitutions have been applied.
type ResolvedResourceCondition struct {
	// A list of conditions that must all be true.
	And []*ResolvedResourceCondition `json:"and,omitempty"`
	// A list of conditions where at least one must be true.
	Or []*ResolvedResourceCondition `json:"or,omitempty"`
	// A condition that will be negated.
	Not *ResolvedResourceCondition `json:"not,omitempty"`
	// A condition expression that is expected
	// to be a substitution that resolves to a boolean.
	StringValue *core.MappingNode `json:"-"`
}

func (c *ResolvedResourceCondition) UnmarshalJSON(data []byte) error {
	if strings.HasPrefix(string(data), "\"") {
		stringVal := &core.MappingNode{}
		if err := json.Unmarshal(data, &stringVal); err == nil {
			c.StringValue = stringVal
			return nil
		} else {
			return err
		}
	}

	type conditionAlias ResolvedResourceCondition
	var alias conditionAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	c.And = alias.And
	c.Or = alias.Or
	c.Not = alias.Not

	if (len(c.And) > 0 && len(c.Or) > 0) ||
		(len(c.Or) > 0 && c.Not != nil) ||
		(len(c.And) > 0 && c.Not != nil) {
		return fmt.Errorf(
			"an invalid resource condition has been provided, only one of \"and\", \"or\" or \"not\" can be set",
		)
	}

	return nil
}

func (c *ResolvedResourceCondition) MarshalJSON() ([]byte, error) {
	if c.StringValue != nil {
		return json.Marshal(c.StringValue)
	}

	type conditionAlias ResolvedResourceCondition
	var alias conditionAlias
	alias.And = c.And
	alias.Or = c.Or
	alias.Not = c.Not
	return json.Marshal(alias)
}

// ResourceValidateParams provides the input data needed for a resource to
// be validated.
type ResourceValidateInput struct {
	SchemaResource  *schema.Resource
	ProviderContext Context
}

// ResourceValidateOutput provides the output data from validating a resource
// which includes a list of diagnostics that detail issues with the resource.
type ResourceValidateOutput struct {
	Diagnostics []*core.Diagnostic
}

// ResourceGetSpecDefinitionInput provides the input data needed for a resource to
// provide a spec definition.
type ResourceGetSpecDefinitionInput struct {
	ProviderContext Context
}

// ResourceGetSpecDefinitionOutput provides the output data from providing a spec definition
// for a resource.
type ResourceGetSpecDefinitionOutput struct {
	SpecDefinition *ResourceSpecDefinition
}

// ResourceCanLinkToInput provides the input data needed for a resource to
// determine what types of resources it can link to.
type ResourceCanLinkToInput struct {
	ProviderContext Context
}

// ResourceCanLinkToOutput provides the output data from determining what types of resources
// a given resource can link to.
type ResourceCanLinkToOutput struct {
	CanLinkTo []string
}

// ResourceStabilisedDependenciesInput provides the input data needed for a resource to
// determine what resource types must be stabilised before the current resource can be deployed.
type ResourceStabilisedDependenciesInput struct {
	ProviderContext Context
}

// ResourceStabilisedDependenciesOutput provides the output data from determining what resource types
// must be stabilised before the current resource can be deployed.
type ResourceStabilisedDependenciesOutput struct {
	StabilisedDependencies []string
}

// ResourceIsCommonTerminalInput provides the input data needed for a resource to
// determine if it is a common terminal resource.
type ResourceIsCommonTerminalInput struct {
	ProviderContext Context
}

// ResourceIsCommonTerminalOutput provides the output data from determining if a resource
// is a common terminal resource.
type ResourceIsCommonTerminalOutput struct {
	IsCommonTerminal bool
}

// ResourceDeployInput provides the input data needed for a resource to
// be deployed.
type ResourceDeployInput struct {
	InstanceID string
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName    string
	ResourceID      string
	Changes         *Changes
	ProviderContext Context
}

// ResourceGetTypeInput provides the input data needed for a resource to
// determine the type of a resource in a blueprint spec.
type ResourceGetTypeInput struct {
	ProviderContext Context
}

// ResourceGetTypeOutput provides the output data from determining the type of a resource
// in a blueprint spec.
type ResourceGetTypeOutput struct {
	Type string
	// A human-readable label for the resource type.
	Label string
}

// ResourceGetTypeDescriptionInput provides the input data needed for a resource to
// retrieve a description of the type of a resource in a blueprint spec.
type ResourceGetTypeDescriptionInput struct {
	ProviderContext Context
}

// ResourceGetTypeDescriptionOutput provides the output data from retrieving a description
// of the type of a resource in a blueprint spec.
type ResourceGetTypeDescriptionOutput struct {
	MarkdownDescription  string
	PlainTextDescription string
	// A short summary of the resource type that can be formatted
	// in markdown, this is useful for listing resource types in documentation.
	MarkdownSummary string
	// A short summary of the resource type in plain text,
	// this is useful for listing resource types in documentation.
	PlainTextSummary string
}

// ResourceGetExamplesInput provides the input data needed for a resource to
// retrieve examples for a resource type in a blueprint spec.
type ResourceGetExamplesInput struct {
	ProviderContext Context
}

// ResourceGetExamplesOutput provides the output data from retrieving examples
// for a resource type in a blueprint spec.
type ResourceGetExamplesOutput struct {
	MarkdownExamples  []string
	PlainTextExamples []string
}

// ResourceDeployOutput provides the output data from deploying a resource.
// This should contain any computed fields that are known after the resource
// has been deployed.
type ResourceDeployOutput struct {
	// ComputedFieldValues holds a mapping of computed field paths
	// to their values.
	// Examples of computed fields are the ARN of an AWS Lambda function
	// or the ID of a Google Cloud Storage bucket.
	// Some examples of valid computed field paths are:
	// - `spec.arn`
	// - `spec.id`
	// - `spec.arns[0]`
	// - `spec.identifiers["id.v1"]`
	// - `spec["identifiers.1"].arn`
	//
	// The computed fields will be injected into the final resource state that
	// will be persisted as a part of the blueprint instance state.
	ComputedFieldValues map[string]*core.MappingNode
}

// ResourceHasStabilisedInput provides the input data needed for a resource to
// determine if it has stabilised after being deployed.
type ResourceHasStabilisedInput struct {
	InstanceID string
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName     string
	ResourceID       string
	ResourceSpec     *core.MappingNode
	ResourceMetadata *state.ResourceMetadataState
	ProviderContext  Context
}

// ResourceHasStabilisedOutput provides the output data from determining if a resource
// has stabilised after being deployed.
type ResourceHasStabilisedOutput struct {
	Stabilised bool
}

// ResourceGetExternalStateInput provides the input data needed for a resource to
// get the external state of a resource.
type ResourceGetExternalStateInput struct {
	InstanceID string
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName            string
	ResourceID              string
	CurrentResourceSpec     *core.MappingNode
	CurrentResourceMetadata *state.ResourceMetadataState
	ProviderContext         Context
}

// ResourceGetExternalStateOutput provides the output data from
// retrieving the external state of a resource.
type ResourceGetExternalStateOutput struct {
	ResourceSpecState *core.MappingNode
}

// ResourceDestroyInput provides the input data needed to delete
// a resource.
type ResourceDestroyInput struct {
	InstanceID string
	// Additional user-defined blueprint instance name
	// that can be used in ID/unique name generation and for debugging.
	InstanceName    string
	ResourceID      string
	ResourceState   *state.ResourceState
	ProviderContext Context
}

// Changes provides a set of modified fields along with a version
// of the resource schema (includes metadata labels and annotations) and spec
// that has already had all it's variables substituted.
type Changes struct {
	// AppliedResourceInfo provides a new version of the spec
	// and schema for which variable substitution has been applied
	// so the deploy phase has everything it needs to deploy the resource.
	AppliedResourceInfo ResourceInfo  `json:"appliedResourceInfo"`
	MustRecreate        bool          `json:"mustRecreate"`
	ModifiedFields      []FieldChange `json:"modifiedFields"`
	NewFields           []FieldChange `json:"newFields"`
	RemovedFields       []string      `json:"removedFields"`
	UnchangedFields     []string      `json:"unchangedFields"`
	// ComputedFields holds a list of field paths that are computed
	// at deploy time. This is primarily useful to give fast access to
	// information about which fields are computed without having to inspect
	// the spec schema in link implementations.
	ComputedFields []string `json:"computedFields"`
	// FieldChangesKnownOnDeploy holds a list of field names
	// for which changes will be known when the host blueprint is deployed.
	FieldChangesKnownOnDeploy []string `json:"fieldChangesKnownOnDeploy"`
	// ConditionKnownOnDeploy specifies whether the condition
	// for the resource will be known when the host blueprint is deployed.
	// When a condition makes use of items in the blueprint that are not resolved
	// until deployment, whether the resource will be deployed or not
	// cannot be known during the change staging phase.
	ConditionKnownOnDeploy bool `json:"conditionKnownOnDeploy"`
	// NewOutboundLinks holds a mapping of the linked to resource name
	// to the link changes representing the new links that will be created.
	NewOutboundLinks map[string]LinkChanges `json:"newOutboundLinks"`
	// OutboundLinkChanges holds a mapping
	// of the linked to resource name to any changes
	// that will be made to existing links.
	// The key is of the form `{resourceA}::{resoureB}`
	OutboundLinkChanges map[string]LinkChanges `json:"outboundLinkChanges"`
	// RemovedOutboundLinks holds a list of link identifiers
	// that will be removed.
	// The form of the link identifier is `{resourceA}::{resoureB}`
	RemovedOutboundLinks []string `json:"removedOutboundLinks"`
}

// ResourceSpecDefinition provides a definition for a resource spec
// that is used for state management, validation, docs and tooling.
type ResourceSpecDefinition struct {
	// Schema holds the schema for the resource
	// specification.
	Schema *ResourceDefinitionsSchema
	// IDField holds the name of the field in the resource spec
	// that holds the ID of the resource.
	// This is used to resolve references to a resource in a blueprint
	// where only the name of the resource is specified.
	// For example, references such as `resources.processOrderFunction` or `processOrderFunction`
	// should resolve to the ID of the resource in the blueprint.
	// The ID field must be a top-level property of the resource spec schema.
	IDField string
	// DestroyBeforeCreate specifies whether the resource must be destroyed
	// before it can be created again.
	// In most cases, this should be false or not set, as where possible,
	// resources should be replaced with minimal disruption, as per a create-before-destroy
	// strategy.
	// This is useful for resources that will often need to be recreated with the same
	// user-defined unique name that are safe to destroy before creating a replacement.
	// This doesn't necessarily mean the resource will always be recreated successfully,
	// as some providers may not allow the use of the same name for a resource until a certain
	// period of time has passed since the resource was destroyed.
	//
	// To keep things simple for the practitioner, this is not something that can be set
	// in a blueprint spec, it is up to the developers of resource providers to set this
	// based on the nature of the resource and how critical it will be deemed for most use cases.
	//
	// As far as the core blueprint framework is concerned, this is for documentation purposes
	// and is a hint for tools and SDKs built on top of the framework (such as the plugin framework SDK)
	// on how they should handle resource deployments.
	DestroyBeforeCreate bool
}

// ResourceDefinitionsSchema provides a schema that can be used to validate
// a resource spec or output state.
type ResourceDefinitionsSchema struct {
	// Type holds the type of the resource definition.
	Type ResourceDefinitionsSchemaType
	// Label holds a human-readable label for the resource definition.
	Label string
	// Description holds a human-readable description for the resource definition
	// without any formatting.
	Description string
	// FormattedDescription holds a human-readable description for the resource definition
	// that is formatted with markdown.
	FormattedDescription string
	// Attributes holds a mapping of the attribute types for a resource definition
	// schema object.
	Attributes map[string]*ResourceDefinitionsSchema
	// Items holds the schema for the items in a resource definition schema array.
	Items *ResourceDefinitionsSchema
	// MapValues holds the schema for the values in a resource definition schema map.
	// Keys are always strings.
	MapValues *ResourceDefinitionsSchema
	// OneOf holds a list of possible schemas for a resource definition item.
	// This is to be used with the "union" type.
	OneOf []*ResourceDefinitionsSchema
	// Required holds a list of required attributes for a resource definition schema object.
	Required []string
	// Nullable specifies whether the resource definition schema can be null.
	Nullable bool
	// AllowedValues holds a list of allowed values for a resource definition schema.
	// This is useful for enum-like values and can be used with the "string",
	// "integer" and "float" types.
	// Unions and enums are mutually exclusive, when designing your schemas,
	// you should use either unions for type constraints or allowed values for
	// value constraints.
	// In the validation implementation, the allowed values constraint will only be applied
	// for values that contain ${..} substitutions, warnings will be produced when the value
	// contains substitutions as there is no way to know the final value during the validation phase.
	// Allowed values take precedence over other value constrants such as minimum, maximum and pattern.
	AllowedValues []*core.MappingNode
	// Minimum holds the minimum value that can be used for an element in a resource spec that uses
	// this schema.
	// This is only used for "integer" and "float" types.
	// This constraint can not be forced when ${..} substitutions are used in a value,
	// warnings will be produced when the value contains substitutions
	// as there is no way to know the final value during the validation phase.
	// Allowed values will take precedence over this constraint.
	Minimum *core.ScalarValue
	// Maximum holds the maximum value that can be used for an element in a resource spec that uses
	// this schema.
	// This is only used for "integer" and "float" types.
	// This constraint can not be forced when ${..} substitutions are used in a value,
	// warnings will be produced when the value contains substitutions
	// as there is no way to know the final value during the validation phase.
	// Allowed values will take precedence over this constraint.
	Maximum *core.ScalarValue
	// MinLength can provide a minimum length for a string value,
	// minimum amount of items in an array or minimum number of keys in a map.
	// This is only used for "string", "array" and "map" types.
	//
	// For strings, this constraint represents the minimum number of characters (runes)
	// in the string and not the number of bytes.
	//
	// This constraint can not be forced when ${..} substitutions are used in a string value,
	// warnings will be produced when a string value contains substitutions
	// as there is no way to know the final value during the validation phase.
	// 0 means that there is no minimum length constraint.
	// Allowed values will take precedence over this constraint for values of type "string".
	MinLength int
	// MaxLength can provide a maximum length for a string value,
	// maximum amount of items in an array or maximum number of keys in a map.
	// This is only used for "string", "array" and "map" types.
	//
	// For strings, this constraint represents the maximum number of characters (runes)
	// in the string and not the number of bytes.
	//
	// This constraint can not be forced when ${..} substitutions are used in a string value,
	// warnings will be produced when the vstring alue contains substitutions
	// as there is no way to know the final value during the validation phase.
	// 0 means that there is no maximum length constraint.
	// Allowed values will take precedence over this constraint for values of type "string".
	MaxLength int
	// Pattern holds a regular expression pattern that can be used to validate
	// a string value in a resource spec that uses this schema.
	// This is only used for "string" types.
	// The pattern must be a valid Go regular expression, see: https://pkg.go.dev/regexp/syntax.
	// Allowed values will take precedence over this constraint.
	Pattern string
	// ValidateFunc is a custom validation function that can be used to validate
	// a resource spec value that uses this schema.
	// This function will be called during the validation phase of a resource
	// and can be used to perform additional validation that is not covered
	// by the schema definition.
	// This can also be useful for targeted conditional validation based on other
	// known values in the resource as defined in the source blueprint document.
	// The function should return a list of diagnostics that will be used to
	// report validation errors and warnings, validation will fail if at least one
	// diagnostic is of level `core.DiagnosticLevelError`.
	//
	// All first-class validation properties such as `AllowedValues`, `Minimum`, `Maximum`,
	// `MinLength`, `MaxLength` and `Pattern` will be applied before the custom validation function
	// is called, so the custom validation function should not be used to validate
	// and will only be called if a non-nil value is provided for the value and the value
	// is known at the validation stage.
	// This will not be called for computed values or fields that contain ${..} substitutions
	// as the value is not known at the initial validation stage.
	ValidateFunc func(
		path string,
		value *core.MappingNode,
		resource *schema.Resource,
	) []*core.Diagnostic
	// Default holds the default value for a resource spec schema,
	// this will be populated in the `Resource.Spec.*` mapping node
	// if the resource spec is missing a value
	// for a specific attribute or item in the spec.
	// The default value will not be used if the attribute value in a given resource spec is nil
	// and the schema is nullable, a nil pointer should not be used
	// for an empty value, pointers should be set when you want to explicitly
	// set a value to nil.
	// The default value will not be used for computed values in a resource spec.
	Default *core.MappingNode
	// Examples holds a list of examples for the resource definition element.
	// Examples are useful for documentation and tooling.
	Examples []*core.MappingNode
	// Computed specifies whether the value is computed by the provider
	// and should not be set by the user.
	// Computed values are expected to be populated by resource implementations
	// in a provider in the deployment process.
	Computed bool
	// MustRecreate specifies whether the resource must be recreated
	// if a change to the field is detected in the resource state.
	// This is only used for user-provided values, it will be ignored
	// for computed values.
	MustRecreate bool
	// Sensitive specifies whether the value is sensitive and should be kept
	// out of logs and other outputs.
	Sensitive bool
	// IgnoreDrift specifies whether drift checking behaviour in the core blueprint
	// framework drift checker should ignore changes to this field when checking
	// for drift in the resource state.
	// This is mostly useful for fields that are only used during the creation of a resource,
	// such as a private key for a certificate.
	// This only applies to user-defined values defined in a blueprint resource spec,
	// computed values are not used in the change set generation process used for checking drift.
	//
	// For fields that have a union type, this must be set at the union level
	// and not at the individual schema level in order to ignore the field's changes
	// to be considered as drift.
	// Nested types in a union can be ignored for drift checking.
	IgnoreDrift bool
}

// ResourceDefinitionsSchemaType holds the type of a resource schema.
type ResourceDefinitionsSchemaType string

const (
	// ResourceDefinitionsSchemaTypeString is for a schema string.
	ResourceDefinitionsSchemaTypeString ResourceDefinitionsSchemaType = "string"
	// ResourceDefinitionsSchemaTypeInteger is for a schema integer.
	ResourceDefinitionsSchemaTypeInteger ResourceDefinitionsSchemaType = "integer"
	// ResourceDefinitionsSchemaTypeFloat is for a schema float.
	ResourceDefinitionsSchemaTypeFloat ResourceDefinitionsSchemaType = "float"
	// ResourceDefinitionsSchemaTypeBoolean is for a schema boolean.
	ResourceDefinitionsSchemaTypeBoolean ResourceDefinitionsSchemaType = "boolean"
	// ResourceDefinitionsSchemaTypeMap is for a schema map.
	ResourceDefinitionsSchemaTypeMap ResourceDefinitionsSchemaType = "map"
	// ResourceDefinitionsSchemaTypeObject is for a schema object.
	ResourceDefinitionsSchemaTypeObject ResourceDefinitionsSchemaType = "object"
	// ResourceDefinitionsSchemaTypeArray is for a schema array.
	ResourceDefinitionsSchemaTypeArray ResourceDefinitionsSchemaType = "array"
	// ResourceDefinitionsSchemaTypeUnion is for an element that can be one of
	// multiple schemas.
	ResourceDefinitionsSchemaTypeUnion ResourceDefinitionsSchemaType = "union"
)
