package transformerv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// TransformerPluginDefinition is a template to be used when creating transformer plugins.
// It provides a structure that allows you to define the abstract resources supported
// by the transformer plugin.
// This doesn't have to be used but is a useful way to define the plugin's capabilities,
// there are multiple convenience functions to create new plugins.
// This implements the `transform.SpecTransformer` interface and can be used in the same way
// as any other transformer implementation to create a transformer plugin.
type TransformerPluginDefinition struct {

	// The transform name string that is to be used in the
	// `transform` field of a blueprint.
	TransformName string

	// Configuration definition for the transformer plugin.
	TransformerConfigDefinition *core.ConfigDefinition

	// A mapping of asbtract resource types to their
	// implementations.
	AbstractResources map[string]transform.AbstractResource

	// A set of link definitions between abstract resource types
	// owned by this transformer. Each entry is keyed by
	// "{ResourceTypeA}::{ResourceTypeB}" for fast edge-class lookup.
	// (e.g. "celerity/handler::celerity/api")
	AbstractLinks map[string]*AbstractLinkDefinition

	// A function to transform a blueprint.
	// If this function is not set, the default implementation
	// will return the input blueprint as the transformed blueprint.
	TransformFunc func(
		ctx context.Context,
		input *transform.SpecTransformerTransformInput,
	) (*transform.SpecTransformerTransformOutput, error)
}

func (p *TransformerPluginDefinition) GetTransformName(
	ctx context.Context,
) (string, error) {
	return p.TransformName, nil
}

func (p *TransformerPluginDefinition) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return p.TransformerConfigDefinition, nil
}

func (p *TransformerPluginDefinition) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	if p.TransformFunc != nil {
		return p.TransformFunc(ctx, input)
	}

	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

func (p *TransformerPluginDefinition) ListAbstractLinkTypes(
	ctx context.Context,
) ([]string, error) {
	return utils.GetKeys(p.AbstractLinks), nil
}

func (p *TransformerPluginDefinition) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	diagnostics := []*core.Diagnostic{}

	for _, edge := range input.LinkGraph.Edges() {
		if crossesAbstractConcreteBoundary(edge, input.LinkGraph) {
			diagnostics = append(
				diagnostics,
				crossBoundryLinkDiagnostic(edge, input.LinkGraph),
			)
			continue
		}

		key := core.LinkType(edge.SourceType, edge.TargetType)
		definition, ok := p.AbstractLinks[key]
		if !ok {
			diagnostics = append(
				diagnostics,
				noSuchAbstractLinkDefinitionDiagnostic(edge, input.LinkGraph),
			)
			continue
		}

		annotationDiagnostics, err := validateLinkAnnotations(edge, definition, input)
		diagnostics = append(diagnostics, annotationDiagnostics...)
		if err != nil {
			return &transform.SpecTransformerValidateLinksOutput{
				Diagnostics: diagnostics,
			}, err
		}

		if definition.ValidateFunc != nil {
			customDiagnostics, err := definition.ValidateFunc(ctx, &AbstractLinkValidateInput{
				Edge:               edge,
				LinkGraph:          input.LinkGraph,
				TransformerContext: input.TransformerContext,
			})
			diagnostics = append(diagnostics, customDiagnostics.Diagnostics...)
			if err != nil {
				return &transform.SpecTransformerValidateLinksOutput{
					Diagnostics: diagnostics,
				}, err
			}
		}
	}

	diagnostics = append(
		diagnostics,
		validation.ValidateLinkCardinality(
			input.LinkGraph,
			abstractLinkDefsToCardinalityInfo(p.AbstractLinks),
		)...,
	)

	return &transform.SpecTransformerValidateLinksOutput{
		Diagnostics: diagnostics,
	}, nil
}

func (p *TransformerPluginDefinition) AbstractResource(
	ctx context.Context,
	abstractResourceType string,
) (transform.AbstractResource, error) {
	resource, ok := p.AbstractResources[abstractResourceType]
	if !ok {
		return nil, errAbstractResourceTypeNotFound(abstractResourceType)
	}
	return resource, nil
}

func (p *TransformerPluginDefinition) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return utils.GetKeys(p.AbstractResources), nil
}

func (p *TransformerPluginDefinition) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	def, ok := p.AbstractLinks[linkType]
	if !ok {
		return nil, errAbstractLinkNotFound(linkType, p.TransformName)
	}
	return def, nil
}

func validateLinkAnnotations(
	edge *linktypes.ResolvedLink,
	definition *AbstractLinkDefinition,
	input *transform.SpecTransformerValidateLinksInput,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	for definitionKey, annotationDef := range definition.AnnotationDefinitions {
		// Annotations can be on either the source or target resource depending on the definition.
		annotationResourceName, otherResourceName := getAnnotationResourceNames(
			definitionKey,
			annotationDef,
			edge,
		)

		// Replace the "other" resource placeholder in the annotation definition
		// key with the actual resource name to get the real annotation key to look up on the resource.
		renderedKey := validation.ReplaceAnnotationPlaceholderWithResourceName(
			definitionKey,
			otherResourceName,
		)

		resource, _, hasResource := input.LinkGraph.Resource(annotationResourceName)
		if !hasResource {
			diagnostics = append(
				diagnostics,
				missingAnnotationResourceDiagnostic(annotationResourceName),
			)
			continue
		}

		resourceAnnotations := getResourceAnnotations(resource)

		// Get all annotations that match this definition.
		// For dynamic definitions (with placeholders), this uses pattern matching
		// to find all annotations that match the pattern, not just those
		// that exactly match the rendered definition name.
		matchingAnnotations, err := validation.GetAllMatchingAnnotations(
			definitionKey,
			resourceAnnotations,
		)
		if err != nil {
			return nil, err
		}

		// Check if required annotation is missing
		if len(matchingAnnotations) == 0 && annotationDef.Required {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"The %q annotation is required for the %q resource in relation to the %q resource, but is missing or null.",
					renderedKey,
					annotationResourceName,
					otherResourceName,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					getResourceMetadataBlockLocation(resource),
					nil,
				),
			})
			continue
		}

		matchDiagnostics := validateMatchingAnnotations(
			definitionKey,
			matchingAnnotations,
			annotationDef,
			annotationResourceName,
		)
		diagnostics = append(diagnostics, matchDiagnostics...)
	}

	return diagnostics, nil
}

func validateMatchingAnnotations(
	definitionKey string,
	matchingAnnotations []*validation.ResourceAnnotationInfo,
	annotationDef *provider.LinkAnnotationDefinition,
	annotationResourceName string,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	for _, resourceAnnotationInfo := range matchingAnnotations {
		if substitutions.IsNilStringSubs(resourceAnnotationInfo.Annotation) {
			continue
		}

		validateDiagnostics := validation.ValidateAnnotationValue(
			definitionKey,
			annotationDef,
			annotationResourceName,
			resourceAnnotationInfo,
		)
		diagnostics = append(diagnostics, validateDiagnostics...)
	}

	return diagnostics
}

func getAnnotationResourceNames(
	annotationKey string,
	annotationDef *provider.LinkAnnotationDefinition,
	edge *linktypes.ResolvedLink,
) (string, string) {
	if annotationDef.AppliesTo == provider.LinkAnnotationResourceB ||
		strings.HasPrefix(annotationKey, fmt.Sprintf("%s::", edge.TargetType)) {
		return edge.Target, edge.Source
	}

	return edge.Source, edge.Target
}

func getResourceAnnotations(resource *schema.Resource) *schema.StringOrSubstitutionsMap {
	if resource.Metadata == nil || resource.Metadata.Annotations == nil {
		return &schema.StringOrSubstitutionsMap{
			Values: map[string]*substitutions.StringOrSubstitutions{},
		}
	}

	return resource.Metadata.Annotations
}

func getResourceMetadataBlockLocation(resource *schema.Resource) *source.Meta {
	if resource.Metadata == nil {
		return nil
	}

	return resource.Metadata.SourceMeta
}

func crossesAbstractConcreteBoundary(
	edge *linktypes.ResolvedLink,
	linkGraph linktypes.DeclaredLinkGraph,
) bool {
	_, srcResClass, srcExists := linkGraph.Resource(edge.Source)
	_, tgtResClass, tgtExists := linkGraph.Resource(edge.Target)
	if !srcExists || !tgtExists {
		return false
	}

	return (srcResClass == linktypes.ResourceClassAbstract &&
		tgtResClass == linktypes.ResourceClassConcrete) ||
		(srcResClass == linktypes.ResourceClassConcrete &&
			tgtResClass == linktypes.ResourceClassAbstract)
}

func abstractLinkDefsToCardinalityInfo(
	abstractLinkDefs map[string]*AbstractLinkDefinition,
) map[string]provider.LinkGetCardinalityOutput {
	cardinalityMap := map[string]provider.LinkGetCardinalityOutput{}

	for linkType, def := range abstractLinkDefs {
		cardinalityMap[linkType] = provider.LinkGetCardinalityOutput{
			CardinalityA: def.CardinalityA,
			CardinalityB: def.CardinalityB,
		}
	}

	return cardinalityMap
}
