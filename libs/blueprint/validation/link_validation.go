package validation

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ValidateLinkConstraints validates cardinality constraints for concrete
// provider links along with custom validation functionality defined by the
// provider developer.
func ValidateLinkConstraints(
	ctx context.Context,
	linkChains []*links.ChainLinkNode,
	declaredLinkGraph linktypes.DeclaredLinkGraph,
	spec speccore.BlueprintSpec,
	params core.BlueprintParams,
) ([]*core.Diagnostic, error) {
	visited := map[string]bool{}
	linkImplInfo, err := collectLinkImplInfo(ctx, linkChains, params, visited)
	if err != nil {
		return nil, err
	}

	diagnostics := ValidateLinkCardinality(declaredLinkGraph, linkImplInfo.cardinalityRules)

	customDiags, err := runCustomLinkValidation(ctx, linkImplInfo.linkInstances, spec, params)
	if err != nil {
		return diagnostics, err
	}
	diagnostics = append(diagnostics, customDiags...)

	return diagnostics, nil
}

// linkImplCollected holds information extracted from link implementations
// during chain traversal.
type linkImplCollected struct {
	cardinalityRules map[string]provider.LinkGetCardinalityOutput
	linkInstances    []*linkInstanceInfo
}

// linkInstanceInfo holds information about a specific link instance
// in the blueprint for custom validation.
type linkInstanceInfo struct {
	linkImpl      provider.Link
	linkType      string
	resourceAName string
	resourceBName string
}

func collectLinkImplInfo(
	ctx context.Context,
	linkChains []*links.ChainLinkNode,
	params core.BlueprintParams,
	visited map[string]bool,
) (*linkImplCollected, error) {
	result := &linkImplCollected{
		cardinalityRules: map[string]provider.LinkGetCardinalityOutput{},
		linkInstances:    []*linkInstanceInfo{},
	}

	for _, node := range linkChains {
		err := collectLinkImplInfoFromNode(ctx, node, params, visited, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func collectLinkImplInfoFromNode(
	ctx context.Context,
	node *links.ChainLinkNode,
	params core.BlueprintParams,
	visited map[string]bool,
	result *linkImplCollected,
) error {
	if visited[node.ResourceName] {
		return nil
	}
	visited[node.ResourceName] = true

	linkCtx := provider.NewLinkContextFromParams(params)

	for targetName, linkImpl := range node.LinkImplementations {
		linkTypeOutput, err := linkImpl.GetType(ctx, &provider.LinkGetTypeInput{
			LinkContext: linkCtx,
		})
		if err != nil {
			return err
		}

		linkType := linkTypeOutput.Type
		result.linkInstances = append(result.linkInstances, &linkInstanceInfo{
			linkImpl:      linkImpl,
			linkType:      linkType,
			resourceAName: node.ResourceName,
			resourceBName: targetName,
		})

		if _, exists := result.cardinalityRules[linkType]; !exists {
			cardinalityOutput, err := linkImpl.GetCardinality(
				ctx,
				&provider.LinkGetCardinalityInput{
					LinkContext: linkCtx,
				},
			)
			if err != nil {
				return err
			}

			if cardinalityOutput != nil {
				result.cardinalityRules[linkType] = *cardinalityOutput
			}
		}
	}

	for _, child := range node.LinksTo {
		err := collectLinkImplInfoFromNode(ctx, child, params, visited, result)
		if err != nil {
			return err
		}
	}

	return nil
}

func runCustomLinkValidation(
	ctx context.Context,
	linkInstances []*linkInstanceInfo,
	spec speccore.BlueprintSpec,
	params core.BlueprintParams,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	for _, instance := range linkInstances {
		resourceA := spec.ResourceSchema(instance.resourceAName)
		resourceB := spec.ResourceSchema(instance.resourceBName)
		if resourceA == nil || resourceB == nil {
			continue
		}

		annotations := collectAnnotationsForLink(resourceA, resourceB)

		output, err := instance.linkImpl.ValidateLink(
			ctx,
			&provider.LinkValidateInput{
				ResourceASpec: resourceA.Spec,
				ResourceBSpec: resourceB.Spec,
				ResourceAName: instance.resourceAName,
				ResourceBName: instance.resourceBName,
				ResourceAType: schema.GetResourceType(resourceA),
				ResourceBType: schema.GetResourceType(resourceB),
				Annotations:   annotations,
				LinkContext:   provider.NewLinkContextFromParams(params),
			},
		)
		if err != nil {
			return diagnostics, err
		}

		if output != nil {
			diagnostics = append(diagnostics, output.Diagnostics...)
		}
	}

	return diagnostics, nil
}

// collectAnnotationsForLink collects annotations from both resources
// that can be resolved at validation time (static string values only).
// Annotations containing substitutions are not included as they
// can not be resolved until deployment time.
func collectAnnotationsForLink(
	resourceA *schema.Resource,
	resourceB *schema.Resource,
) map[string]*core.ScalarValue {
	annotations := map[string]*core.ScalarValue{}
	collectStaticAnnotations(resourceA, annotations)
	collectStaticAnnotations(resourceB, annotations)
	return annotations
}

func collectStaticAnnotations(
	resource *schema.Resource,
	annotations map[string]*core.ScalarValue,
) {
	if resource.Metadata == nil || resource.Metadata.Annotations == nil {
		return
	}

	for key, strOrSubs := range resource.Metadata.Annotations.Values {
		if substitutions.IsNilStringSubs(strOrSubs) {
			continue
		}

		if len(strOrSubs.Values) == 1 && strOrSubs.Values[0].StringValue != nil {
			value := *strOrSubs.Values[0].StringValue
			annotations[key] = &core.ScalarValue{
				StringValue: &value,
			}
		}
	}
}

// ValidateLinkCardinality validates that the links in the link graph
// adhere to the cardinality rules defined by their corresponding
// link types.
// This can be used for both abstract transformer links and concrete provider links.
func ValidateLinkCardinality(
	linkGraph linktypes.DeclaredLinkGraph,
	linkCardinalityRules map[string]provider.LinkGetCardinalityOutput,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	// Count of outgoing edges from each source resource.
	// (linkType, sourceResourceName) -> count of outgoing edges.
	sourceCounts := map[resourceLinkKey]int{}
	// Count of incoming edges to each target resource.
	// (linkType, targetResourceName) -> count of incoming edges.
	targetCounts := map[resourceLinkKey]int{}

	for _, edge := range linkGraph.Edges() {
		linkKey := core.LinkType(edge.SourceType, edge.TargetType)

		sourceCountKey := resourceLinkKey{
			linkType:     linkKey,
			resourceName: edge.Source,
		}
		sourceCounts[sourceCountKey] += 1

		targetCountKey := resourceLinkKey{
			linkType:     linkKey,
			resourceName: edge.Target,
		}
		targetCounts[targetCountKey] += 1
	}

	for sourceKey, outgoingCount := range sourceCounts {
		linkType := sourceKey.linkType
		sourceResourceName := sourceKey.resourceName

		rules, exists := linkCardinalityRules[linkType]
		if !exists {
			// If there are no cardinality rules for this link type, skip validation.
			continue
		}

		sourceType, targetType, _ := core.SplitLinkType(linkType)
		diagnostics = append(
			diagnostics,
			validateCardinalityOutgoing(
				sourceResourceName,
				sourceType,
				targetType,
				outgoingCount,
				rules,
			)...,
		)
	}

	for targetKey, incomingCount := range targetCounts {
		linkType := targetKey.linkType
		targetResourceName := targetKey.resourceName

		rules, exists := linkCardinalityRules[linkType]
		if !exists {
			// If there are no cardinality rules for this link type, skip validation.
			continue
		}

		sourceType, targetType, _ := core.SplitLinkType(linkType)
		diagnostics = append(
			diagnostics,
			validateCardinalityIncoming(
				targetResourceName,
				sourceType,
				targetType,
				incomingCount,
				rules,
			)...,
		)
	}

	return diagnostics
}

type resourceLinkKey struct {
	linkType     string
	resourceName string
}

func validateCardinalityOutgoing(
	sourceResourceName string,
	sourceType string,
	targetType string,
	sourceOutgoingCount int,
	rules provider.LinkGetCardinalityOutput,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	// <=0 means there is no maximum cardinality.
	if rules.CardinalityA.Max > 0 && sourceOutgoingCount > rules.CardinalityA.Max {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"Resource %q of type %q has %d outgoing links to resources of type %q,"+
					" exceeding the maximum of %d defined by the link type %q.",
				sourceResourceName,
				sourceType,
				sourceOutgoingCount,
				targetType,
				rules.CardinalityA.Max,
				core.LinkType(sourceType, targetType),
			),
		})
	}

	// <=0 means there is no minimum cardinality.
	if rules.CardinalityA.Min > 0 && sourceOutgoingCount < rules.CardinalityA.Min {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"Resource %q of type %q has %d outgoing links to resources of type %q,"+
					" below the minimum of %d defined by the link type %q.",
				sourceResourceName,
				sourceType,
				sourceOutgoingCount,
				targetType,
				rules.CardinalityA.Min,
				core.LinkType(sourceType, targetType),
			),
		})
	}

	return diagnostics
}

func validateCardinalityIncoming(
	targetResourceName string,
	sourceType string,
	targetType string,
	targetIncomingCount int,
	rules provider.LinkGetCardinalityOutput,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	// <=0 means there is no maximum cardinality.
	if rules.CardinalityB.Max > 0 && targetIncomingCount > rules.CardinalityB.Max {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"Resource %q of type %q has %d incoming links from resources of type %q,"+
					" exceeding the maximum of %d defined by the link type %q.",
				targetResourceName,
				targetType,
				targetIncomingCount,
				sourceType,
				rules.CardinalityB.Max,
				core.LinkType(sourceType, targetType),
			),
		})
	}

	// <=0 means there is no minimum cardinality.
	if rules.CardinalityB.Min > 0 && targetIncomingCount < rules.CardinalityB.Min {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"Resource %q of type %q has %d incoming links from resources of type %q,"+
					" below the minimum of %d defined by the link type %q.",
				targetResourceName,
				targetType,
				targetIncomingCount,
				sourceType,
				rules.CardinalityB.Min,
				core.LinkType(sourceType, targetType),
			),
		})
	}

	return diagnostics
}
