package transformerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// AbstractResourceDefinition is a template to be used for defining abstract resources
// when creating transformer plugins.
// It provides a structure that allows you to define a schema and behaviour
// of a resource.
// This implements the `transform.AbstractResource` interface and can be used in the same way
// as any other resource implementation used in a transformer plugin.
type AbstractResourceDefinition struct {
	// The type of the abstract resource.
	// Example: "celerity/handler".
	Type string

	// A human-readable label for the abstract resource type.
	// This will be used in documentation and tooling.
	Label string

	// A summary of the abstract resource type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextSummary string

	// A summary of the abstract resource type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedSummary string

	// A description of the abstract resource type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the abstract resource type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A list of plain text examples that can be used to
	// demonstrate how to use the abstract resource.
	// This will be used in documentation and tooling.
	PlainTextExamples []string

	// A list of markdown examples that can be used to
	// demonstrate how to use the abstract resource.
	// This will be used in documentation and tooling.
	FormattedExamples []string

	// The schema of the abstract resource specification that comes under the `spec` field
	// of a resource in a blueprint.
	Schema *provider.ResourceDefinitionsSchema

	// Holds the name of the field in the abstract resource schema
	// that holds the third-party
	// ID of the abstract resource.
	// This is used to resolve references to a resource in a blueprint
	// where only the name of the resource is specified.
	// For example, references such as `resources.processOrderHandler` or
	// `processOrderHandler` should resolve to the ID of the resource in the blueprint.
	// The ID field must be a top-level property of the resource spec schema.
	//
	// For abstract resources, this should point to an underlying ID value
	// in the primary concrete resource that the abstract resource expands to.
	IDField string

	// Specifies whether this abstract resource is expected to have a common use-case
	// as a terminal resource that does not link out to other resources.
	// This is useful for providing useful warnings to users about their blueprints
	// without overloading them with warnings for all resources that don't have any outbound
	// links that could have.
	// If CommonTerminalFunc is provided, this static value will not be used.
	CommonTerminal bool

	// A function that can be used to dynamically determine whether an abstract resource is likely
	// to be a common terminal resource (does not link out to other resources).
	CommonTerminalFunc func(
		ctx context.Context,
		input *transform.AbstractResourceIsCommonTerminalInput,
	) (*transform.AbstractResourceIsCommonTerminalOutput, error)

	// A static list of abstract resource types that this abstract resource can link to.
	// If ResourceCanLinkToFunc is provided, this static list will not be used.
	ResourceCanLinkTo []string

	// A function that can be used to dynamically determine what
	// abstract resource types this resource can link to.
	ResourceCanLinkToFunc func(
		ctx context.Context,
		input *transform.AbstractResourceCanLinkToInput,
	) (*transform.AbstractResourceCanLinkToOutput, error)

	// A function to apply custom validation for an abstract resource
	// that goes beyond validating against the resource spec schema which
	// the blueprint framework takes care of.
	// When not provided, the plugin will return a result with an empty list
	// of diagnostics.
	CustomValidateFunc func(
		ctx context.Context,
		input *transform.AbstractResourceValidateInput,
	) (*transform.AbstractResourceValidateOutput, error)
}

func (r *AbstractResourceDefinition) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	if r.CustomValidateFunc == nil {
		return &transform.AbstractResourceValidateOutput{
			Diagnostics: []*core.Diagnostic{},
		}, nil
	}

	return r.CustomValidateFunc(ctx, input)
}

func (r *AbstractResourceDefinition) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema:  r.Schema,
			IDField: r.IDField,
		},
	}, nil
}

func (r *AbstractResourceDefinition) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	if r.ResourceCanLinkToFunc != nil {
		return r.ResourceCanLinkToFunc(ctx, input)
	}

	return &transform.AbstractResourceCanLinkToOutput{
		CanLinkTo: r.ResourceCanLinkTo,
	}, nil
}

func (r *AbstractResourceDefinition) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	if r.CommonTerminalFunc != nil {
		return r.CommonTerminalFunc(ctx, input)
	}

	return &transform.AbstractResourceIsCommonTerminalOutput{
		IsCommonTerminal: r.CommonTerminal,
	}, nil
}

func (r *AbstractResourceDefinition) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type:  r.Type,
		Label: r.Label,
	}, nil
}

func (r *AbstractResourceDefinition) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{
		PlainTextDescription: r.PlainTextDescription,
		MarkdownDescription:  r.FormattedDescription,
		PlainTextSummary:     r.PlainTextSummary,
		MarkdownSummary:      r.FormattedSummary,
	}, nil
}

func (r *AbstractResourceDefinition) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{
		MarkdownExamples:  r.FormattedExamples,
		PlainTextExamples: r.PlainTextExamples,
	}, nil
}
