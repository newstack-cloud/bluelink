package transformerv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

func crossBoundryLinkDiagnostic(
	edge *linktypes.ResolvedLink,
	linkGraph linktypes.DeclaredLinkGraph,
) *core.Diagnostic {
	srcResource, srcResClass, _ := linkGraph.Resource(edge.Source)
	_, tgtResClass, _ := linkGraph.Resource(edge.Target)
	// Highlight the link selector in the source resource to make it clear that the outgoing
	// link from the source resource is what's causing the issue.
	location := getResourceLinkSelectorLocation(srcResource)
	return &core.Diagnostic{
		Level: core.DiagnosticLevelError,
		Message: fmt.Sprintf(
			"The %q resource can not link to the %q resource as %q is %s and %q is %s. Links between abstract and concrete resources are not supported.",
			edge.Source,
			edge.Target,
			edge.Source,
			srcResClass,
			edge.Target,
			tgtResClass,
		),
		Range: core.DiagnosticRangeFromSourceMeta(location, nil),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeCrossAbstractConcreteBoundaryLink,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:  string(errors.ActionTypeCheckResourceType),
					Title: "Check resource types in link",
					Description: fmt.Sprintf(
						"Check the resource types of %q and %q and ensure they are both abstract or both concrete.",
						edge.Source,
						edge.Target,
					),
					Priority: 1,
				},
			},
		},
	}
}

func noSuchAbstractLinkDefinitionDiagnostic(
	edge *linktypes.ResolvedLink,
	linkGraph linktypes.DeclaredLinkGraph,
) *core.Diagnostic {
	srcResource, _, _ := linkGraph.Resource(edge.Source)
	// Highlight the link selector in the source resource to make it clear that the outgoing
	// link from the source resource is what's causing the issue.
	location := getResourceLinkSelectorLocation(srcResource)

	return &core.Diagnostic{
		Level: core.DiagnosticLevelError,
		Message: fmt.Sprintf(
			"No abstract link definition found for link type: %s -> %s",
			edge.SourceType,
			edge.TargetType,
		),
		Range: core.DiagnosticRangeFromSourceMeta(location, nil),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryTransformer,
			ReasonCode: ErrorReasonCodeNoSuchAbstractLinkDefinition,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:  string(errors.ActionTypeCheckConfiguration),
					Title: "Check transformer configuration",
					Description: fmt.Sprintf(
						"Check the transformer plugin configuration to ensure it defines an abstract link for the link type: %s -> %s",
						edge.SourceType,
						edge.TargetType,
					),
					Priority: 1,
				},
			},
		},
	}
}

func missingAnnotationResourceDiagnostic(annotationResourceName string) *core.Diagnostic {
	return &core.Diagnostic{
		Level: core.DiagnosticLevelError,
		Message: fmt.Sprintf(
			"Missing resource %q required for link annotation",
			annotationResourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryTransformer,
			ReasonCode: ErrorReasonCodeMissingAnnotationResource,
		},
	}
}

func getResourceLinkSelectorLocation(resource *schema.Resource) *source.Meta {
	if resource == nil || resource.LinkSelector == nil {
		return nil
	}

	return resource.LinkSelector.SourceMeta
}
