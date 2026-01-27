package languageservices

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// getResourceAnnotationKeyCompletionItems returns completion items for annotation keys
// based on link annotation definitions relevant to the current resource.
func (s *CompletionService) getResourceAnnotationKeyCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	if s.linkRegistry == nil {
		return []*lsp.CompletionItem{}, nil
	}

	resourceName := completionCtx.ResourceName
	if resourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	resource := getResource(blueprint, resourceName)
	if resource == nil || resource.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	currentResourceType := resource.Type.Value
	linkedResources := s.findLinkedResources(ctx.Context, blueprint, resourceName, currentResourceType)
	annotationDefs := s.collectAnnotationDefinitions(ctx.Context, blueprint, currentResourceType, linkedResources)

	typedPrefix := ""
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
	}

	return s.createAnnotationKeyCompletionItems(annotationDefs, linkedResources, position, typedPrefix), nil
}

// linkedResourceInfo holds information about a resource that may be linked to the current resource.
type linkedResourceInfo struct {
	name         string
	resourceType string
	// currentIsA indicates whether the current resource is "A" in the link relationship.
	// When true, current resource is A and this linked resource is B.
	// When false, current resource is B and this linked resource is A.
	currentIsA bool
}

// findLinkedResources finds all resources that could be linked to/from the given resource.
func (s *CompletionService) findLinkedResources(
	ctx context.Context,
	blueprint *schema.Blueprint,
	resourceName string,
	resourceType string,
) []linkedResourceInfo {
	if blueprint.Resources == nil {
		return nil
	}

	currentResource := blueprint.Resources.Values[resourceName]
	if currentResource == nil {
		return nil
	}

	currentLabels := getResourceLabels(currentResource)
	var linkedResources []linkedResourceInfo

	for otherName, otherResource := range blueprint.Resources.Values {
		if otherName == resourceName || otherResource.Type == nil {
			continue
		}

		otherType := otherResource.Type.Value
		registeredCurrentIsA, linked := s.getLinkDirection(ctx, resourceType, otherType)
		if linked {
			currentIsA := determineCurrentIsA(
				currentResource, otherResource,
				resourceType, otherType,
				registeredCurrentIsA,
			)
			linkedResources = append(linkedResources, linkedResourceInfo{
				name:         otherName,
				resourceType: otherType,
				currentIsA:   currentIsA,
			})
			continue
		}

		if hasMatchingLabels(otherResource, currentLabels) ||
			hasMatchingSelector(currentResource, otherResource) {
			currentIsA := hasMatchingSelector(currentResource, otherResource)
			linkedResources = append(linkedResources, linkedResourceInfo{
				name:         otherName,
				resourceType: otherType,
				currentIsA:   currentIsA,
			})
		}
	}

	return linkedResources
}

// determineCurrentIsA determines whether the current resource is A in the link relationship.
// For different-type links, uses the registered link direction.
// For same-type links, uses the selector/selected relationship to determine direction.
func determineCurrentIsA(
	currentResource, otherResource *schema.Resource,
	currentType, otherType string,
	registeredCurrentIsA bool,
) bool {
	// For different-type links, use the registered link direction
	if currentType != otherType {
		return registeredCurrentIsA
	}

	// For same-type links, determine based on selector relationship
	if hasMatchingSelector(currentResource, otherResource) {
		// Current is selecting other, so current is A
		return true
	}
	if hasMatchingSelector(otherResource, currentResource) {
		// Other is selecting current, so current is B
		return false
	}

	// No clear direction from selectors, default based on registration order
	return registeredCurrentIsA
}

// getLinkDirection checks if a link exists between two resource types and returns
// whether the current resource (first parameter) is A in the link relationship.
// Returns (currentIsA, linked).
func (s *CompletionService) getLinkDirection(ctx context.Context, currentType, otherType string) (bool, bool) {
	link, err := s.linkRegistry.Link(ctx, currentType, otherType)
	if err == nil && link != nil {
		return true, true // current is A
	}

	link, err = s.linkRegistry.Link(ctx, otherType, currentType)
	if err == nil && link != nil {
		return false, true // current is B
	}

	return false, false
}

// collectAnnotationDefinitions collects all annotation definitions relevant to the resource type.
func (s *CompletionService) collectAnnotationDefinitions(
	ctx context.Context,
	blueprint *schema.Blueprint,
	currentResourceType string,
	linkedResources []linkedResourceInfo,
) map[string]*provider.LinkAnnotationDefinition {
	allDefs := make(map[string]*provider.LinkAnnotationDefinition)

	if blueprint.Resources == nil {
		return allDefs
	}

	for _, linked := range linkedResources {
		s.collectDefsFromLinkPair(ctx, currentResourceType, linked.resourceType, currentResourceType, linked.currentIsA, allDefs)
		// Only try the reverse ordering when types differ - when types are the same,
		// both calls would find the same link and we already have the correct currentIsA.
		if currentResourceType != linked.resourceType {
			s.collectDefsFromLinkPair(ctx, linked.resourceType, currentResourceType, currentResourceType, !linked.currentIsA, allDefs)
		}
	}

	return allDefs
}

// collectDefsFromLinkPair collects annotation definitions from a specific link type pair.
// currentResourceType is the type of the resource being edited.
// currentIsA indicates whether the current resource is A in the link relationship for filtering.
func (s *CompletionService) collectDefsFromLinkPair(
	ctx context.Context,
	typeA, typeB string,
	currentResourceType string,
	currentIsA bool,
	allDefs map[string]*provider.LinkAnnotationDefinition,
) {
	link, err := s.linkRegistry.Link(ctx, typeA, typeB)
	if err != nil || link == nil {
		return
	}

	defs, err := s.getAnnotationDefinitionsForLink(ctx, link, typeA, typeB)
	if err != nil {
		return
	}

	for key, def := range defs {
		if _, exists := allDefs[key]; exists {
			continue
		}
		if !annotationAppliesToCurrentResource(key, def, currentResourceType, currentIsA) {
			continue
		}
		allDefs[key] = def
	}
}

// annotationAppliesToCurrentResource determines if an annotation definition applies
// to the current resource based on AppliesTo and the resource type in the key.
func annotationAppliesToCurrentResource(
	key string,
	def *provider.LinkAnnotationDefinition,
	currentResourceType string,
	currentIsA bool,
) bool {
	switch def.AppliesTo {
	case provider.LinkAnnotationResourceA:
		return currentIsA
	case provider.LinkAnnotationResourceB:
		return !currentIsA
	default: // LinkAnnotationResourceAny
		keyResourceType := extractResourceTypeFromKey(key)
		return keyResourceType == currentResourceType
	}
}

// extractResourceTypeFromKey extracts the resource type prefix from an annotation definition key.
// Key format: {resourceType}::{annotationName}
func extractResourceTypeFromKey(key string) string {
	idx := strings.Index(key, "::")
	if idx == -1 {
		return ""
	}
	return key[:idx]
}

// getAnnotationDefinitionsForLink retrieves annotation definitions from a link,
// using the cache to avoid repeated calls.
func (s *CompletionService) getAnnotationDefinitionsForLink(
	ctx context.Context,
	link provider.Link,
	typeA, typeB string,
) (map[string]*provider.LinkAnnotationDefinition, error) {
	linkKey := fmt.Sprintf("%s::%s", typeA, typeB)

	if cached, found := s.annotationDefCache.Get(linkKey); found {
		return cached, nil
	}

	emptyParams := core.NewDefaultParams(nil, nil, nil, nil)
	linkCtx := provider.NewLinkContextFromParams(emptyParams)
	output, err := link.GetAnnotationDefinitions(ctx, &provider.LinkGetAnnotationDefinitionsInput{
		LinkContext: linkCtx,
	})
	if err != nil {
		return nil, err
	}

	if output == nil || output.AnnotationDefinitions == nil {
		return nil, nil
	}

	s.annotationDefCache.Set(linkKey, output.AnnotationDefinitions)
	return output.AnnotationDefinitions, nil
}

// createAnnotationKeyCompletionItems creates completion items for annotation keys.
func (s *CompletionService) createAnnotationKeyCompletionItems(
	annotationDefs map[string]*provider.LinkAnnotationDefinition,
	linkedResources []linkedResourceInfo,
	position *lsp.Position,
	typedPrefix string,
) []*lsp.CompletionItem {
	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)
	fieldKind := lsp.CompletionItemKindField

	linkedNames := make([]string, len(linkedResources))
	for i, lr := range linkedResources {
		linkedNames[i] = lr.name
	}

	seen := make(map[string]bool)
	var items []*lsp.CompletionItem

	for _, def := range annotationDefs {
		expandedNames := expandAnnotationName(def.Name, linkedNames)

		for _, annotationName := range expandedNames {
			if seen[annotationName] {
				continue
			}
			seen[annotationName] = true

			if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(annotationName), prefixLower) {
				continue
			}

			item := createAnnotationCompletionItem(def, annotationName, position, prefixLen, fieldKind)
			items = append(items, item)
		}
	}

	return items
}

// createAnnotationCompletionItem creates a single completion item for an annotation key.
func createAnnotationCompletionItem(
	def *provider.LinkAnnotationDefinition,
	annotationName string,
	position *lsp.Position,
	prefixLen int,
	fieldKind lsp.CompletionItemKind,
) *lsp.CompletionItem {
	insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
	insertText := annotationName + ": "
	detail := "Link annotation"

	item := &lsp.CompletionItem{
		Label:      annotationName,
		Detail:     &detail,
		Kind:       &fieldKind,
		FilterText: &annotationName,
		TextEdit: lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		},
		Data: map[string]any{
			"completionType": "annotationKey",
		},
	}

	if def.Description != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: def.Description,
		}
	}

	return item
}

// expandAnnotationName expands annotation names with <resourceName> placeholders.
// Returns the original name if no placeholder, or expanded names for each linked resource.
func expandAnnotationName(name string, linkedResourceNames []string) []string {
	openIdx := strings.Index(name, "<")
	closeIdx := strings.Index(name, ">")
	if openIdx == -1 || closeIdx == -1 || closeIdx < openIdx {
		return []string{name}
	}

	if len(linkedResourceNames) == 0 {
		return nil
	}

	expanded := make([]string, 0, len(linkedResourceNames))
	for _, linkedName := range linkedResourceNames {
		expandedName := name[:openIdx] + linkedName + name[closeIdx+1:]
		expanded = append(expanded, expandedName)
	}

	return expanded
}

// getResourceLabels extracts labels from a resource's metadata.
func getResourceLabels(resource *schema.Resource) map[string]string {
	if resource.Metadata == nil || resource.Metadata.Labels == nil {
		return nil
	}
	return resource.Metadata.Labels.Values
}

// hasMatchingLabels checks if a resource has labels that match any in the given label map.
func hasMatchingLabels(resource *schema.Resource, labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}

	resourceLabels := getResourceLabels(resource)
	if len(resourceLabels) == 0 {
		return false
	}

	for key, value := range labels {
		if resourceValue, ok := resourceLabels[key]; ok && resourceValue == value {
			return true
		}
	}
	return false
}

// hasMatchingSelector checks if the first resource has a linkSelector that matches the second.
func hasMatchingSelector(selector *schema.Resource, candidate *schema.Resource) bool {
	if selector.LinkSelector == nil || selector.LinkSelector.ByLabel == nil {
		return false
	}

	candidateLabels := getResourceLabels(candidate)
	if len(candidateLabels) == 0 {
		return false
	}

	for key, selectorValue := range selector.LinkSelector.ByLabel.Values {
		if candidateValue, ok := candidateLabels[key]; ok && candidateValue == selectorValue {
			return true
		}
	}
	return false
}
