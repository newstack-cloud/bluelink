package transform

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// SpecTransformer is the common interface
// used for spec transformations that takes a blueprint spec
// with a specific transform and applies it to expand
// the blueprint spec into it's final form.
// This is primarily for allowing users to define more concise specifications
// where a lot of detail can be abstracted away.
//
// A spec transformer is responsible for providing a way to validate
// abstract resources against a schema prior to transforming the blueprint.
//
// Spec transformers are called straight after a schema has been successfully
// parsed and variables have been validated.
type SpecTransformer interface {
	// GetTransformName retrieves the name of the transformer
	// that is used to identify the transformer in blueprints,
	// this is what is set in the `transform` field of a blueprint.
	GetTransformName(ctx context.Context) (string, error)
	// ConfigDefinition retrieves a detailed definition of the
	// configuration that is required for the transformer.
	ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error)
	// Transform a blueprint by expanding abstract resources
	// into their final form along with any other transformations
	// that are required.
	Transform(ctx context.Context, input *SpecTransformerTransformInput) (*SpecTransformerTransformOutput, error)
	// AbstractResources returns the abstract resource implementation
	// for a given resource type.
	AbstractResource(ctx context.Context, resourceType string) (AbstractResource, error)
	// ListAbstractResourceTypes retrieves a list of all the abstract resource types
	// that are provided by the provider.
	// This is primarily used in tools and documentation to provide a list of
	// available abstract resource types.
	ListAbstractResourceTypes(ctx context.Context) ([]string, error)
}

// AbstractResource is the interface for an abstract resource
// that a spec transformer can contain which includes logic for validating
// an abstract resource before transformation.
type AbstractResource interface {
	// CustomValidate a schema for an abstract resource that will be transformed.
	CustomValidate(ctx context.Context, input *AbstractResourceValidateInput) (*AbstractResourceValidateOutput, error)
	// GetSpecDefinition retrieves the spec definition for an abstract resource,
	// this is the first line of validation for a resource in a blueprint and is also
	// useful for validating references to an abstract resource instance
	// in a blueprint and for providing definitions for docs and tooling.
	GetSpecDefinition(
		ctx context.Context,
		input *AbstractResourceGetSpecDefinitionInput,
	) (*AbstractResourceGetSpecDefinitionOutput, error)
	// CanLinkTo specifices the list of resource types the current resource type
	// can link to.
	// For abstract resources, links do not have a one-to-one mapping to a link plugin implementation,
	// the transformer should expand these links to the concrete resources for which there will be
	// a link plugin implementation.
	CanLinkTo(ctx context.Context, input *AbstractResourceCanLinkToInput) (*AbstractResourceCanLinkToOutput, error)
	// IsCommonTerminal specifies whether this resource is expected to have a common use-case
	// as a terminal resource that does not link out to other resources.
	// This is useful for providing useful warnings to users about their blueprints
	// without overloading them with warnings for all resources that don't have any outbound
	// links that could have.
	IsCommonTerminal(
		ctx context.Context,
		input *AbstractResourceIsCommonTerminalInput,
	) (*AbstractResourceIsCommonTerminalOutput, error)
	// GetType deals with retrieving the namespaced type for an abstract
	// resource in a blueprint spec.
	GetType(ctx context.Context, input *AbstractResourceGetTypeInput) (*AbstractResourceGetTypeOutput, error)
	// GetTypeDescription deals with retrieving the description for a resource type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetTypeDescription(
		ctx context.Context,
		input *AbstractResourceGetTypeDescriptionInput,
	) (*AbstractResourceGetTypeDescriptionOutput, error)
	// GetExamples deals with retrieving a list examples for an abstract resource type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetExamples(ctx context.Context, input *AbstractResourceGetExamplesInput) (*AbstractResourceGetExamplesOutput, error)
}

// SpecTransformerTransformInput provides the input required to transform
// a blueprint.
type SpecTransformerTransformInput struct {
	InputBlueprint     *schema.Blueprint
	TransformerContext Context
}

// SpecTransformerTransformOutput provides the output from transforming a blueprint
// which includes the expanded blueprint.
type SpecTransformerTransformOutput struct {
	TransformedBlueprint *schema.Blueprint
}

// AbstractResourceValidateInput provides the input required to validate
// an abstract resource before transformation.
type AbstractResourceValidateInput struct {
	SchemaResource     *schema.Resource
	TransformerContext Context
}

// AbstractResourceValidateOutput provides the output from validating an abstract resource
// which includes a list of diagnostics that detail issues with the abstract resource.
type AbstractResourceValidateOutput struct {
	Diagnostics []*core.Diagnostic
}

// AbstractResourceGetSpecDefinitionInput provides the input from providing a spec definition
// for an abstract resource.
type AbstractResourceGetSpecDefinitionInput struct {
	TransformerContext Context
}

// AbstractResourceGetSpecDefinitionOutput provides the output from providing a spec definition
// for an abstract resource.
type AbstractResourceGetSpecDefinitionOutput struct {
	SpecDefinition *provider.ResourceSpecDefinition
}

// AbstractResourceCanLinkToInput provides the input data needed for a resource to
// determine what types of resources it can link to.
type AbstractResourceCanLinkToInput struct {
	TransformerContext Context
}

// AbstractResourceCanLinkToOutput provides the output data from determining what types of resources
// a given resource can link to.
type AbstractResourceCanLinkToOutput struct {
	CanLinkTo []string
}

// AbstractResourceIsCommonTerminalInput provides the input data needed for a resource to
// determine if it is a common terminal resource.
type AbstractResourceIsCommonTerminalInput struct {
	TransformerContext Context
}

// AbstractResourceIsCommonTerminalOutput provides the output data from determining if a resource
// is a common terminal resource.
type AbstractResourceIsCommonTerminalOutput struct {
	IsCommonTerminal bool
}

// AbstractResourceGetTypeInput provides the input data needed for an abstract resource to
// determine the type of a resource in a blueprint spec.
type AbstractResourceGetTypeInput struct {
	TransformerContext Context
}

// AbstractResourceGetTypeOutput provides the output data from determining the type of an
// abstract resource in a blueprint spec.
type AbstractResourceGetTypeOutput struct {
	Type string
	// A human-readable label for the abstract resource type.
	Label string
}

// AbstractResourceGetTypeDescriptionInput provides the input data needed for a resource to
// retrieve a description of the type of an abstract resource in a blueprint spec.
type AbstractResourceGetTypeDescriptionInput struct {
	TransformerContext Context
}

// AbstractResourceGetTypeDescriptionOutput provides the output data from retrieving a description
// of the type of am abstract resource in a blueprint spec.
type AbstractResourceGetTypeDescriptionOutput struct {
	MarkdownDescription  string
	PlainTextDescription string
	// A short summary of the abstract resource type that can be formatted
	// in markdown, this is useful for listing abstract resource types in documentation.
	MarkdownSummary string
	// A short summary of the abstract resource type in plain text,
	// this is useful for listing abstract resource types in documentation.
	PlainTextSummary string
}

// AbstractResourceGetExamplesInput provides the input data needed for a resource to
// retrieve examples for an abstract resource type in a blueprint spec.
type AbstractResourceGetExamplesInput struct {
	TransformerContext Context
}

// AbstractResourceGetExamplesOutput provides the output data from retrieving examples
// for an abstract resource type in a blueprint spec.
type AbstractResourceGetExamplesOutput struct {
	MarkdownExamples  []string
	PlainTextExamples []string
}

// Context provides access to information about the current transformer
// and environment that a transformer plugin is running in.
// This is not to be confused with the conventional Go context.Context
// used for setting deadlines, cancelling requests and storing request-scoped
// values in a Go program.
type Context interface {
	// TransformerConfigVariable retrieves a configuration value that was loaded
	// for the current provider.
	TransformerConfigVariable(name string) (*core.ScalarValue, bool)
	// TransformerConfigVariables retrieves all the configuration values that were loaded
	// for the current transformer.
	// This is useful to export all the configuration values to be sent to plugins
	// that are running in a different process.
	TransformerConfigVariables() map[string]*core.ScalarValue
	// ContextVariable retrieves a context-wide variable
	// for the current environment, this differs from values extracted
	// from context.Context, as these context variables are specific
	// to the components that implement the interfaces of the blueprint library
	// and can be shared between processes over a network or similar.
	ContextVariable(name string) (*core.ScalarValue, bool)
	// ContextVariables retrieves all the context-wide variables
	// for the current environment.
	// This is useful to export all the context-wide variables to be sent to plugins
	// that are running in a different process.
	ContextVariables() map[string]*core.ScalarValue
}
