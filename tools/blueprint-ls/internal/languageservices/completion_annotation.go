package languageservices

import (
	"context"
	"fmt"
	"slices"
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
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
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

// annotationDefWithContext wraps an annotation definition with the target resource type
// that should be used when expanding <resourceName> placeholders.
type annotationDefWithContext struct {
	definition         *provider.LinkAnnotationDefinition
	targetResourceType string // The "other" resource type for placeholder expansion
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

	var linkedResources []linkedResourceInfo

	for otherName, otherResource := range blueprint.Resources.Values {
		if otherName == resourceName || otherResource.Type == nil {
			continue
		}

		otherType := otherResource.Type.Value
		registeredCurrentIsA, linked := s.getLinkDirection(ctx, resourceType, otherType)
		if !linked {
			continue
		}

		// Check if there's an actual link relationship via selectors/labels.
		// A registered link type alone isn't enough - there must be a selector/label match.
		// A link requires one resource to have a linkSelector matching another's labels.
		currentSelectsOther := hasMatchingSelector(currentResource, otherResource, otherName)
		otherSelectsCurrent := hasMatchingSelector(otherResource, currentResource, resourceName)

		if !currentSelectsOther && !otherSelectsCurrent {
			continue
		}

		currentIsA := determineCurrentIsA(
			currentResource, otherResource,
			resourceName, otherName,
			registeredCurrentIsA,
		)
		linkedResources = append(linkedResources, linkedResourceInfo{
			name:         otherName,
			resourceType: otherType,
			currentIsA:   currentIsA,
		})
	}

	return linkedResources
}

// determineCurrentIsA determines whether the current resource is A in the link relationship.
// The selector/selected relationship takes precedence for determining direction.
// Falls back to the registered link direction when no explicit selector relationship exists.
func determineCurrentIsA(
	currentResource, otherResource *schema.Resource,
	currentName, otherName string,
	registeredCurrentIsA bool,
) bool {
	// Check selector relationships for ALL links (not just same-type)
	// The resource with the linkSelector is A, the selected resource is B
	currentSelectsOther := hasMatchingSelector(currentResource, otherResource, otherName)
	otherSelectsCurrent := hasMatchingSelector(otherResource, currentResource, currentName)

	if currentSelectsOther && !otherSelectsCurrent {
		// Only current selects other, so current is A
		return true
	}
	if otherSelectsCurrent && !currentSelectsOther {
		// Only other selects current, so current is B
		return false
	}

	// Both select each other, or neither explicitly selects (e.g., label matching only)
	// Fall back to the registered link direction
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

// collectAnnotationDefinitions collects all annotation definitions relevant to the resource type,
// along with the target resource type for placeholder expansion.
func (s *CompletionService) collectAnnotationDefinitions(
	ctx context.Context,
	blueprint *schema.Blueprint,
	currentResourceType string,
	linkedResources []linkedResourceInfo,
) map[string]*annotationDefWithContext {
	allDefs := make(map[string]*annotationDefWithContext)

	if blueprint.Resources == nil {
		return allDefs
	}

	for _, linked := range linkedResources {
		// Determine the link lookup order based on currentIsA.
		// If current is A, the link should be Link(currentType, linkedType).
		// If current is B, the link should be Link(linkedType, currentType).
		// We only look up the link that matches our actual position in the relationship,
		// not both directions.
		var typeA, typeB string
		if linked.currentIsA {
			typeA = currentResourceType
			typeB = linked.resourceType
		} else {
			typeA = linked.resourceType
			typeB = currentResourceType
		}
		s.collectDefsFromLinkPair(ctx, typeA, typeB, currentResourceType, linked.currentIsA, allDefs)
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
	allDefs map[string]*annotationDefWithContext,
) {
	link, err := s.linkRegistry.Link(ctx, typeA, typeB)
	if err != nil || link == nil {
		return
	}

	defs, err := s.getAnnotationDefinitionsForLink(ctx, link, typeA, typeB)
	if err != nil {
		return
	}

	// Determine the "other" resource type for placeholder expansion
	// If current is typeA, the target for <resourceName> is typeB, and vice versa
	targetResourceType := typeB
	if currentResourceType == typeB {
		targetResourceType = typeA
	}

	for key, def := range defs {
		if _, exists := allDefs[key]; exists {
			continue
		}
		if !annotationAppliesToCurrentResource(key, def, currentIsA, typeA, typeB) {
			continue
		}
		allDefs[key] = &annotationDefWithContext{
			definition:         def,
			targetResourceType: targetResourceType,
		}
	}
}

// annotationAppliesToCurrentResource determines if an annotation definition applies
// to the current resource based on AppliesTo and the resource type in the key.
// When AppliesTo is ResourceAny (the default), the function infers the target position
// from the annotation key's resource type prefix and the link registration order.
func annotationAppliesToCurrentResource(
	key string,
	def *provider.LinkAnnotationDefinition,
	currentIsA bool,
	typeA, typeB string,
) bool {
	switch def.AppliesTo {
	case provider.LinkAnnotationResourceA:
		return currentIsA
	case provider.LinkAnnotationResourceB:
		return !currentIsA
	default: // LinkAnnotationResourceAny
		// Infer which position the annotation targets from its key prefix.
		// For example, "aws/lambda/function::someAnnotation" targets the resource
		// whose type is "aws/lambda/function" in the link relationship.
		keyResourceType := extractResourceTypeFromKey(key)
		if keyResourceType == "" {
			// No resource type prefix - allow for any resource
			return true
		}

		// For same-type links, we can't infer position from the key prefix
		// since both A and B have the same type. Allow the annotation for
		// any resource whose type matches.
		if typeA == typeB {
			return keyResourceType == typeA
		}

		// Determine if the annotation's resource type is A or B in the link
		if keyResourceType == typeA {
			// Annotation targets resource A, so current must be A
			return currentIsA
		}
		if keyResourceType == typeB {
			// Annotation targets resource B, so current must be B
			return !currentIsA
		}

		// Key resource type doesn't match either type in this link
		return false
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
	annotationDefs map[string]*annotationDefWithContext,
	linkedResources []linkedResourceInfo,
	position *lsp.Position,
	typedPrefix string,
) []*lsp.CompletionItem {
	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)
	fieldKind := lsp.CompletionItemKindField

	// Build a map of resource type -> resource names for filtering
	linkedNamesByType := make(map[string][]string)
	for _, lr := range linkedResources {
		linkedNamesByType[lr.resourceType] = append(linkedNamesByType[lr.resourceType], lr.name)
	}

	seen := make(map[string]bool)
	var items []*lsp.CompletionItem

	for _, defCtx := range annotationDefs {
		// Only use linked resource names that match the target type for this annotation
		targetNames := linkedNamesByType[defCtx.targetResourceType]
		expandedNames := expandAnnotationName(defCtx.definition.Name, targetNames)

		for _, annotationName := range expandedNames {
			if seen[annotationName] {
				continue
			}
			seen[annotationName] = true

			if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(annotationName), prefixLower) {
				continue
			}

			item := createAnnotationCompletionItem(defCtx.definition, annotationName, position, prefixLen, fieldKind)
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

// hasMatchingSelector checks if the first resource has a linkSelector that matches the second.
// The candidateName parameter is used to check if the candidate is in the exclude list.
func hasMatchingSelector(selector *schema.Resource, candidate *schema.Resource, candidateName string) bool {
	if selector.LinkSelector == nil || selector.LinkSelector.ByLabel == nil {
		return false
	}

	// Check if candidate is in exclude list
	if selector.LinkSelector.Exclude != nil {
		if slices.Contains(selector.LinkSelector.Exclude.Values, candidateName) {
			return false
		}
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

// getResourceAnnotationValueCompletionItems returns completion items for annotation values
// based on AllowedValues in the LinkAnnotationDefinition.
func (s *CompletionService) getResourceAnnotationValueCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
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

	// Get the annotation key from the path or extracted field name
	annotationKey := ""
	if completionCtx.CursorCtx != nil {
		if key, ok := completionCtx.CursorCtx.StructuralPath.GetAnnotationKey(); ok {
			annotationKey = key
		} else if completionCtx.CursorCtx.ExtractedFieldName != "" {
			annotationKey = completionCtx.CursorCtx.ExtractedFieldName
		}
	}

	if annotationKey == "" {
		return []*lsp.CompletionItem{}, nil
	}

	// Find the annotation definition for this key
	currentResourceType := resource.Type.Value
	linkedResources := s.findLinkedResources(ctx.Context, blueprint, resourceName, currentResourceType)
	annotationDefs := s.collectAnnotationDefinitions(ctx.Context, blueprint, currentResourceType, linkedResources)

	annotationDef := s.findAnnotationDefinitionByKey(annotationDefs, annotationKey, linkedResources)
	if annotationDef == nil || len(annotationDef.AllowedValues) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	// Create completion items from AllowedValues
	typedPrefix := ""
	textBefore := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
		textBefore = completionCtx.CursorCtx.TextBefore
	}

	return s.createAnnotationValueCompletionItems(
		annotationDef,
		position,
		typedPrefix,
		textBefore,
		format,
	), nil
}

// findAnnotationDefinitionByKey finds an annotation definition that matches the given key.
// It handles both static keys and expanded dynamic keys with resource name placeholders.
func (s *CompletionService) findAnnotationDefinitionByKey(
	annotationDefs map[string]*annotationDefWithContext,
	annotationKey string,
	linkedResources []linkedResourceInfo,
) *provider.LinkAnnotationDefinition {
	// Build a map of resource type -> resource names for filtering
	linkedNamesByType := make(map[string][]string)
	for _, lr := range linkedResources {
		linkedNamesByType[lr.resourceType] = append(linkedNamesByType[lr.resourceType], lr.name)
	}

	for _, defCtx := range annotationDefs {
		// Only use linked resource names that match the target type for this annotation
		targetNames := linkedNamesByType[defCtx.targetResourceType]
		expandedNames := expandAnnotationName(defCtx.definition.Name, targetNames)
		if slices.Contains(expandedNames, annotationKey) {
			return defCtx.definition
		}
	}
	return nil
}

// createAnnotationValueCompletionItems creates completion items from AllowedValues.
func (s *CompletionService) createAnnotationValueCompletionItems(
	def *provider.LinkAnnotationDefinition,
	position *lsp.Position,
	typedPrefix string,
	textBefore string,
	format docmodel.DocumentFormat,
) []*lsp.CompletionItem {
	filterPrefix, hasLeadingQuote := stripLeadingQuote(typedPrefix)
	if format != docmodel.FormatJSONC {
		hasLeadingQuote = false
	}
	prefixLen := len(typedPrefix)
	hasLeadingSpace := hasLeadingWhitespace(textBefore, prefixLen)
	prefixLower := strings.ToLower(filterPrefix)

	detail := fmt.Sprintf("Allowed value (%s)", string(def.Type))
	enumKind := lsp.CompletionItemKindEnumMember
	items := make([]*lsp.CompletionItem, 0, len(def.AllowedValues))

	for _, allowedValue := range def.AllowedValues {
		if allowedValue == nil {
			continue
		}

		valueStr := allowedValue.ToString()
		if len(filterPrefix) > 0 && !strings.HasPrefix(strings.ToLower(valueStr), prefixLower) {
			continue
		}

		insertText := formatValueForInsert(valueStr, format, hasLeadingQuote, hasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      valueStr,
			Detail:     &detail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &valueStr,
			Data:       map[string]any{"completionType": "annotationValue"},
		}

		if def.Description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: def.Description,
			}
		}

		items = append(items, item)
	}

	return items
}
