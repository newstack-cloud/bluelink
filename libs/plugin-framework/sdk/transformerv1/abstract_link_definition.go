package transformerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// AbstractLinkDefinition declares a link between two abstract resource types.
// It is the abstract resource analogue of providerv1.LinkDefinition, minus
// the execution logic as abstract links don't deploy anything;
// they are expanded into concrete resource properties and links during
// transformation.
type AbstractLinkDefinition struct {
	// The type of the source abstract resource in the link relationship.
	// (e.g. "celerity/api").
	ResourceTypeA string

	// The type of the destination abstract resource in the link relationship.
	// (e.g. "celerity/handler").
	ResourceTypeB string

	// A summary of the link type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextSummary string

	// A summary of the link type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedSummary string

	// A description of the link type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the link type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A mapping of annotation names prefixed by resource type that
	// can be used to fine tune the behaviour of a link in a blueprint spec.
	// The format should be as follows:
	// {resourceType}::{annotationName} -> LinkAnnotationDefinition
	// e.g. "celerity/handler::celerity.handler.http.method" -> LinkAnnotationDefinition
	AnnotationDefinitions map[string]*provider.LinkAnnotationDefinition

	// CardinalityA: how many B's each A may link to.
	//   Example: celerity/handler → celerity/api, CardinalityA = {Min: 0, Max: 1}
	//     means a handler may link to at most one api.
	CardinalityA provider.LinkCardinality

	// CardinalityB: how many A's may link to each B.
	//   Example: celerity/api ← celerity/handler, CardinalityB = {Min: 1, Max: 0}
	//     means every api must have at least one handler linking to it
	//     (Max: 0 = unlimited).
	CardinalityB provider.LinkCardinality

	// Custom validation function for constraints that can't be expressed declaratively.
	// Called once per resolved edge matching this definition. Runs after
	// the declarative checks; additional diagnostics are concatenated.
	ValidateFunc func(
		ctx context.Context,
		input *AbstractLinkValidateInput,
	) (*AbstractLinkValidateOutput, error)
}

// AbstractLinkValidateInput is the input to the
// custom validate function for an AbstractLinkDefinition.
type AbstractLinkValidateInput struct {
	Edge               *linktypes.ResolvedLink
	LinkGraph          linktypes.DeclaredLinkGraph
	TransformerContext transform.Context
}

// AbstractLinkValidateOutput is the output of the
// custom validate function for an AbstractLinkDefinition.
type AbstractLinkValidateOutput struct {
	Diagnostics []*core.Diagnostic
}

func (d *AbstractLinkDefinition) GetType(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeInput,
) (*transform.AbstractLinkGetTypeOutput, error) {
	return &transform.AbstractLinkGetTypeOutput{
		Type:          core.LinkType(d.ResourceTypeA, d.ResourceTypeB),
		ResourceTypeA: d.ResourceTypeA,
		ResourceTypeB: d.ResourceTypeB,
	}, nil
}

func (d *AbstractLinkDefinition) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeDescriptionInput,
) (*transform.AbstractLinkGetTypeDescriptionOutput, error) {
	return &transform.AbstractLinkGetTypeDescriptionOutput{
		MarkdownDescription:  d.FormattedDescription,
		PlainTextDescription: d.PlainTextDescription,
		MarkdownSummary:      d.FormattedSummary,
		PlainTextSummary:     d.PlainTextSummary,
	}, nil
}

func (d *AbstractLinkDefinition) GetAnnotationDefinitions(
	ctx context.Context,
	input *transform.AbstractLinkGetAnnotationDefinitionsInput,
) (*transform.AbstractLinkGetAnnotationDefinitionsOutput, error) {
	return &transform.AbstractLinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: d.AnnotationDefinitions,
	}, nil
}

func (d *AbstractLinkDefinition) GetCardinality(
	ctx context.Context,
	input *transform.AbstractLinkGetCardinalityInput,
) (*transform.AbstractLinkGetCardinalityOutput, error) {
	return &transform.AbstractLinkGetCardinalityOutput{
		CardinalityA: d.CardinalityA,
		CardinalityB: d.CardinalityB,
	}, nil
}
