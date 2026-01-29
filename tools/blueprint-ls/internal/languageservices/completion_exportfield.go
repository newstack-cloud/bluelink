package languageservices

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// Export field top-level namespace info for completion suggestions.
type exportFieldNamespace struct {
	name        string
	description string
}

var exportFieldNamespaces = []exportFieldNamespace{
	{"resources", "Reference a resource's spec or metadata fields"},
	{"variables", "Reference a variable value"},
	{"values", "Reference a computed value"},
	{"children", "Reference a child blueprint's exported values"},
	{"datasources", "Reference a data source's exported fields"},
}

// getExportFieldTopLevelCompletionItems returns completion items for export field top-level namespaces.
func (s *CompletionService) getExportFieldTopLevelCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := []*lsp.CompletionItem{}

	for _, ns := range exportFieldNamespaces {
		if !hasNamespaceContent(blueprint, ns.name) {
			continue
		}
		if !filterByPrefix(ns.name, prefixInfo) {
			continue
		}
		item := buildExportFieldCompletionItem(
			ns.name+".",
			ns.name,
			"Export field namespace",
			ns.description,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// hasNamespaceContent checks if a namespace has any content in the blueprint.
func hasNamespaceContent(blueprint *schema.Blueprint, namespace string) bool {
	switch namespace {
	case "resources":
		return blueprint.Resources != nil && len(blueprint.Resources.Values) > 0
	case "variables":
		return blueprint.Variables != nil && len(blueprint.Variables.Values) > 0
	case "values":
		return blueprint.Values != nil && len(blueprint.Values.Values) > 0
	case "children":
		return blueprint.Include != nil && len(blueprint.Include.Values) > 0
	case "datasources":
		return blueprint.DataSources != nil && len(blueprint.DataSources.Values) > 0
	}
	return false
}

// getExportFieldResourceRefCompletionItems returns resource names for export field references.
func (s *CompletionService) getExportFieldResourceRefCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Resources == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(blueprint.Resources.Values))

	for resourceName, resource := range blueprint.Resources.Values {
		if !filterByPrefix(resourceName, prefixInfo) {
			continue
		}
		desc := getResourceDescription(resource)
		item := buildExportFieldCompletionItem(
			resourceName+".",
			resourceName,
			"Resource",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getResourceDescription extracts the description from a resource definition.
func getResourceDescription(resource *schema.Resource) string {
	if resource == nil || resource.Description == nil {
		return ""
	}
	return getStringOrSubstitutionsValue(resource.Description)
}

// getExportFieldResourcePropertyCompletionItems returns spec/metadata properties for resources.
func (s *CompletionService) getExportFieldResourcePropertyCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	resourceName := completionCtx.ResourceName
	if resourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	resource := getResource(blueprint, resourceName)
	if resource == nil {
		return []*lsp.CompletionItem{}, nil
	}

	// Determine path depth to decide what to suggest
	// resources.{name}. has 2 dots, resources.{name}.spec. has 3 dots
	fieldValue := extractExportFieldValueFromContext(completionCtx)
	depth := countDots(fieldValue)

	// resources.{name}. (depth 2) -> suggest "spec", "metadata"
	if depth == 2 {
		return getResourceTopLevelPropCompletionItemsForExportField(position, format, completionCtx)
	}

	// resources.{name}.spec. (depth >= 3) -> suggest spec fields from registry
	if depth >= 3 && containsSegment(fieldValue, "spec") {
		return s.getExportFieldResourceSpecCompletionItems(ctx, position, resource, fieldValue, format, completionCtx)
	}

	// resources.{name}.metadata. (depth == 3) -> suggest metadata fields
	// resources.{name}.metadata.{field}. (depth == 4) -> suggest keys from that field
	// resources.{name}.metadata.custom.{...}. (depth > 4) -> navigate custom MappingNode tree
	if containsSegment(fieldValue, "metadata") {
		if depth == 3 {
			return getExportFieldResourceMetadataCompletionItems(position, format, completionCtx)
		}
		if depth >= 4 {
			return getExportFieldResourceMetadataKeysCompletionItems(position, resource, fieldValue, format, completionCtx)
		}
		return []*lsp.CompletionItem{}, nil
	}

	return []*lsp.CompletionItem{}, nil
}

// getResourceTopLevelPropCompletionItemsForExportField returns spec/metadata completion items.
func getResourceTopLevelPropCompletionItemsForExportField(
	position *lsp.Position,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := []*lsp.CompletionItem{}

	props := []exportFieldNamespace{
		{"spec", "The resource specification containing provider-specific configuration and computed fields"},
		{"metadata", "Resource metadata including displayName, labels, annotations, and custom fields"},
	}

	for _, prop := range props {
		if !filterByPrefix(prop.name, prefixInfo) {
			continue
		}
		item := buildExportFieldCompletionItem(
			prop.name+".",
			prop.name,
			"Property",
			prop.description,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getExportFieldResourceSpecCompletionItems returns resource spec field completions from the registry.
func (s *CompletionService) getExportFieldResourceSpecCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	resource *schema.Resource,
	fieldValue string,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	if resource.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
		ctx.Context,
		resource.Type.Value,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return []*lsp.CompletionItem{}, nil
	}

	if specDefOutput.SpecDefinition == nil || specDefOutput.SpecDefinition.Schema == nil {
		return []*lsp.CompletionItem{}, nil
	}

	// Navigate to correct schema depth based on path
	resourceProp := parseExportFieldResourcePath(fieldValue)
	currentSchema, isAtArray := navigateToSchemaForExportFieldWithArrayInfo(specDefOutput.SpecDefinition.Schema, resourceProp)

	// If we're at an array, show index suggestions based on resource spec
	if isAtArray {
		return getArrayIndexCompletionItems(position, resource, fieldValue, format, completionCtx)
	}

	if currentSchema == nil || currentSchema.Attributes == nil {
		return []*lsp.CompletionItem{}, nil
	}

	return buildExportFieldSchemaCompletionItems(currentSchema.Attributes, position, format, completionCtx)
}

// parseExportFieldResourcePath parses the field value into a resource property path.
// Handles both dot notation (items.0.field) and bracket notation (items[0].field, annotations["key"]).
func parseExportFieldResourcePath(fieldValue string) *substitutions.SubstitutionResourceProperty {
	// Field value is like "resources.myResource.spec." or "resources.myResource.spec.nested."
	segments := parseFieldPathSegments(fieldValue)
	if len(segments) < 3 {
		return nil
	}

	resourceName := segments[1].value
	var path []*substitutions.SubstitutionPathItem
	for i := 2; i < len(segments); i++ {
		seg := segments[i]
		if seg.value == "" {
			continue
		}
		pathItem := &substitutions.SubstitutionPathItem{}
		if seg.isIndex {
			idx := seg.indexValue
			pathItem.ArrayIndex = &idx
		} else {
			pathItem.FieldName = seg.value
		}
		path = append(path, pathItem)
	}

	return &substitutions.SubstitutionResourceProperty{
		ResourceName: resourceName,
		Path:         path,
	}
}

// pathSegment represents a parsed segment of a field path.
type pathSegment struct {
	value      string
	isIndex    bool
	indexValue int64
}

// parseFieldPathSegments parses a field path into segments, handling bracket notation.
// Supports: "a.b.c", "a[0].b", "a["key"].b", "a['key'].b"
// Note: Numeric segments after dots (like "a.0.b") are treated as field names, not indices.
// Only bracket notation [0] creates array index segments.
func parseFieldPathSegments(path string) []pathSegment {
	var segments []pathSegment
	i := 0
	n := len(path)

	for i < n {
		// Skip leading dots
		if path[i] == '.' {
			i++
			continue
		}

		// Check for bracket notation
		if path[i] == '[' {
			seg := parseBracketSegment(path, &i)
			segments = append(segments, seg)
			continue
		}

		// Regular field name - read until dot or bracket
		// Numeric segments via dot notation are field names, not indices
		start := i
		for i < n && path[i] != '.' && path[i] != '[' {
			i++
		}
		if start < i {
			fieldName := path[start:i]
			segments = append(segments, pathSegment{value: fieldName, isIndex: false})
		}
	}

	return segments
}

// parseBracketSegment parses a bracket notation segment like [0], ["key"], or ['key'].
func parseBracketSegment(path string, i *int) pathSegment {
	n := len(path)
	*i++ // Skip opening '['

	if *i >= n {
		return pathSegment{}
	}

	// Check for quoted key
	if path[*i] == '"' || path[*i] == '\'' {
		quote := path[*i]
		*i++ // Skip opening quote
		start := *i
		for *i < n && path[*i] != quote {
			*i++
		}
		value := path[start:*i]
		if *i < n {
			*i++ // Skip closing quote
		}
		if *i < n && path[*i] == ']' {
			*i++ // Skip closing ']'
		}
		return pathSegment{value: value, isIndex: false}
	}

	// Numeric index
	start := *i
	for *i < n && path[*i] != ']' {
		*i++
	}
	indexStr := path[start:*i]
	if *i < n {
		*i++ // Skip closing ']'
	}

	if idx, err := strconv.ParseInt(indexStr, 10, 64); err == nil {
		return pathSegment{value: indexStr, isIndex: true, indexValue: idx}
	}
	return pathSegment{value: indexStr, isIndex: false}
}

// navigateToSchemaForExportFieldWithArrayInfo navigates to the correct schema level and reports if we're at an array.
// Returns (schema, isAtArray) where isAtArray is true if we ended at an array and should show index suggestions.
func navigateToSchemaForExportFieldWithArrayInfo(
	schema *provider.ResourceDefinitionsSchema,
	resourceProp *substitutions.SubstitutionResourceProperty,
) (*provider.ResourceDefinitionsSchema, bool) {
	if resourceProp == nil || len(resourceProp.Path) == 0 {
		return schema, false
	}

	currentSchema := schema

	// Skip "spec" segment if present
	pathStart := 0
	if len(resourceProp.Path) > 0 && resourceProp.Path[0].FieldName == "spec" {
		pathStart = 1
	}

	for i := pathStart; i < len(resourceProp.Path); i++ {
		pathItem := resourceProp.Path[i]

		// Handle array index navigation - navigate into array items
		if pathItem.ArrayIndex != nil {
			if currentSchema.Type == provider.ResourceDefinitionsSchemaTypeArray && currentSchema.Items != nil {
				currentSchema = currentSchema.Items
				continue
			}
			return nil, false
		}

		// Skip empty field names
		if pathItem.FieldName == "" {
			continue
		}

		// Handle array types - navigate through Items to get element schema
		if currentSchema.Type == provider.ResourceDefinitionsSchemaTypeArray && currentSchema.Items != nil {
			currentSchema = currentSchema.Items
		}

		// Handle map types - navigate through MapValues to get value schema
		if currentSchema.Type == provider.ResourceDefinitionsSchemaTypeMap && currentSchema.MapValues != nil {
			currentSchema = currentSchema.MapValues
		}

		// Now navigate by field name in attributes
		if currentSchema.Attributes == nil {
			return nil, false
		}
		attrSchema, exists := currentSchema.Attributes[pathItem.FieldName]
		if !exists {
			return nil, false
		}
		currentSchema = attrSchema
	}

	// If we ended on an array, report that we should show index completions
	if currentSchema.Type == provider.ResourceDefinitionsSchemaTypeArray {
		// Return the Items schema for further navigation, but signal we're at an array
		if currentSchema.Items != nil {
			return currentSchema.Items, true
		}
		return currentSchema, true
	}

	return currentSchema, false
}

// getArrayIndexCompletionItems returns index suggestions for arrays in the resource spec.
func getArrayIndexCompletionItems(
	position *lsp.Position,
	resource *schema.Resource,
	fieldValue string,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := []*lsp.CompletionItem{}

	// Try to determine array length from the resource spec
	arrayLen := getArrayLengthFromResourceSpec(resource, fieldValue)

	if arrayLen == 0 {
		// If we can't determine the length, offer index 0 as a starting point
		arrayLen = 1
	}

	// Check if we're triggered by bracket notation (fieldValue ends with '[')
	isBracketTrigger := strings.HasSuffix(fieldValue, "[")

	for i := 0; i < arrayLen; i++ {
		indexStr := fmt.Sprintf("%d", i)
		if !filterByPrefix(indexStr, prefixInfo) {
			continue
		}

		var insertText string
		if isBracketTrigger {
			// Bracket trigger: insert "0]." to complete the bracket and continue path
			insertText = indexStr + "]."
		} else {
			// Dot trigger (after navigating into array): insert "0."
			insertText = indexStr + "."
		}

		item := buildExportFieldCompletionItem(
			insertText,
			indexStr,
			"Array index",
			fmt.Sprintf("Element at index %d", i),
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getArrayLengthFromResourceSpec tries to determine the array length from the resource's spec.
func getArrayLengthFromResourceSpec(resource *schema.Resource, fieldValue string) int {
	if resource == nil || resource.Spec == nil {
		return 0
	}

	// Parse the path to find the array location
	pathParts := splitFieldPath(fieldValue)
	if len(pathParts) < 4 { // resources.{name}.spec.{arrayField}
		return 0
	}

	// Navigate through the spec to find the array
	// pathParts[3:] contains the spec path (after resources.{name}.spec.)
	current := resource.Spec
	for i := 3; i < len(pathParts); i++ {
		part := pathParts[i]
		if part == "" {
			continue
		}

		// Try to navigate into the current node
		if current == nil {
			return 0
		}

		// Check if current is a mapping node with fields
		if current.Fields != nil {
			if next, exists := current.Fields[part]; exists {
				current = next
			} else {
				return 0
			}
		} else {
			return 0
		}
	}

	// At this point, current should be the array node
	if current != nil && current.Items != nil {
		return len(current.Items)
	}

	return 0
}

// buildExportFieldSchemaCompletionItems creates completion items from schema attributes.
func buildExportFieldSchemaCompletionItems(
	attributes map[string]*provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(attributes))

	for attrName, attrSchema := range attributes {
		if !filterByPrefix(attrName, prefixInfo) {
			continue
		}
		desc := ""
		if attrSchema != nil {
			desc = attrSchema.Description
		}
		// Add trailing dot if this attribute has nested content:
		// - object with attributes
		// - array (will navigate to items)
		// - map (will navigate to values)
		insertText := attrName
		if attrSchema != nil {
			hasNested := len(attrSchema.Attributes) > 0 ||
				attrSchema.Type == provider.ResourceDefinitionsSchemaTypeArray ||
				attrSchema.Type == provider.ResourceDefinitionsSchemaTypeMap ||
				attrSchema.Type == provider.ResourceDefinitionsSchemaTypeObject
			if hasNested {
				insertText = attrName + "."
			}
		}
		item := buildExportFieldCompletionItem(
			insertText,
			attrName,
			"Spec field",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getExportFieldResourceMetadataCompletionItems returns metadata field completions for resources.
func getExportFieldResourceMetadataCompletionItems(
	position *lsp.Position,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := []*lsp.CompletionItem{}

	// Metadata fields with their types:
	// - displayName: string (terminal)
	// - labels: map of strings (progressive)
	// - annotations: map of strings (progressive)
	// - custom: free-form object (progressive)
	type metadataFieldInfo struct {
		name        string
		description string
		isTerminal  bool
	}
	metadataFields := []metadataFieldInfo{
		{"displayName", "A human-readable name for the resource", true},
		{"labels", "Key-value pairs for organizing and categorizing resources", false},
		{"annotations", "Key-value pairs for configuring resource behavior", false},
		{"custom", "Custom metadata fields for provider-specific configuration", false},
	}

	for _, field := range metadataFields {
		if !filterByPrefix(field.name, prefixInfo) {
			continue
		}
		insertText := field.name
		if !field.isTerminal {
			insertText = field.name + "."
		}
		item := buildExportFieldCompletionItem(
			insertText,
			field.name,
			"Metadata field",
			field.description,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getExportFieldResourceMetadataKeysCompletionItems returns keys from a metadata field.
// This extracts keys from labels, annotations, or custom fields defined on the resource.
// For custom metadata, supports deep navigation into the MappingNode tree.
// For keys containing dots or special characters, bracket notation is used (e.g., ["kubernetes.io/name"]).
func getExportFieldResourceMetadataKeysCompletionItems(
	position *lsp.Position,
	resource *schema.Resource,
	fieldValue string,
	format docmodel.DocumentFormat,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	if resource == nil || resource.Metadata == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := []*lsp.CompletionItem{}

	// Determine which metadata field we're in (labels, annotations, or custom)
	pathParts := splitFieldPath(fieldValue)
	if len(pathParts) < 4 {
		return items, nil
	}

	metadataField := pathParts[3] // resources.{name}.metadata.{field}

	isBracketTrigger := strings.HasSuffix(fieldValue, "[")

	switch metadataField {
	case "labels":
		if resource.Metadata.Labels != nil {
			items = appendMetadataKeyCompletionItems(
				items, mapKeys(resource.Metadata.Labels.Values),
				"Label key", position, format, prefixInfo, isBracketTrigger,
			)
		}
	case "annotations":
		if resource.Metadata.Annotations != nil {
			items = appendMetadataKeyCompletionItems(
				items, mapKeys(resource.Metadata.Annotations.Values),
				"Annotation key", position, format, prefixInfo, isBracketTrigger,
			)
		}
	case "custom":
		if resource.Metadata.Custom != nil {
			items = getExportFieldCustomMetadataCompletionItems(
				resource.Metadata.Custom, pathParts, position, format, prefixInfo,
			)
		}
	}

	return items, nil
}

// getExportFieldCustomMetadataCompletionItems navigates the custom MappingNode tree
// based on the path depth and returns field keys or array indices.
func getExportFieldCustomMetadataCompletionItems(
	customNode *core.MappingNode,
	pathParts []string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) []*lsp.CompletionItem {
	// pathParts: [resources, {name}, metadata, custom, {field1}, {field2}, ...]
	// Navigate from index 4 onwards into the MappingNode tree.
	current := customNode
	for i := 4; i < len(pathParts); i++ {
		part := pathParts[i]
		if part == "" {
			continue
		}
		if current == nil {
			return []*lsp.CompletionItem{}
		}
		current = navigateMappingNode(current, []string{part})
	}

	if current == nil || isMappingNodeTerminal(current) {
		return []*lsp.CompletionItem{}
	}

	if isMappingNodeObject(current) {
		keys := getMappingNodeFieldKeys(current)
		return appendMappingNodeKeyExportFieldItems(
			nil, keys, "Custom field", position, format, prefixInfo, current,
		)
	}

	if isMappingNodeArray(current) {
		return getExportFieldMappingNodeArrayIndexItems(current, position, format, prefixInfo)
	}

	return []*lsp.CompletionItem{}
}

// appendMappingNodeKeyExportFieldItems adds export field completion items for MappingNode keys.
// Keys that have nested children (objects/arrays) get a trailing dot for progressive completion.
func appendMappingNodeKeyExportFieldItems(
	items []*lsp.CompletionItem,
	keys []string,
	detail string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
	parentNode *core.MappingNode,
) []*lsp.CompletionItem {
	for _, key := range keys {
		if !filterByPrefix(key, prefixInfo) {
			continue
		}

		insertText := key
		// Add trailing dot if the child node has further navigable content.
		if parentNode != nil && parentNode.Fields != nil {
			if childNode, ok := parentNode.Fields[key]; ok {
				if isMappingNodeObject(childNode) || isMappingNodeArray(childNode) {
					insertText = key + "."
				}
			}
		}

		if needsBracketNotation(key) {
			item := buildBracketNotationCompletionItem(key, detail, position, format, prefixInfo)
			items = append(items, item)
		} else {
			item := buildExportFieldCompletionItem(
				insertText, key, detail, "", position, format, prefixInfo,
			)
			items = append(items, item)
		}
	}
	return items
}

// getExportFieldMappingNodeArrayIndexItems returns index completions for a MappingNode array
// in an export field context.
func getExportFieldMappingNodeArrayIndexItems(
	node *core.MappingNode,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) []*lsp.CompletionItem {
	length := getMappingNodeArrayLength(node)
	if length == 0 {
		return []*lsp.CompletionItem{}
	}

	items := make([]*lsp.CompletionItem, 0, length)
	for i := 0; i < length; i++ {
		indexStr := fmt.Sprintf("%d", i)
		if !filterByPrefix(indexStr, prefixInfo) {
			continue
		}
		insertText := indexStr + "."
		item := buildExportFieldCompletionItem(
			insertText, indexStr, "Array index",
			fmt.Sprintf("Element at index %d", i),
			position, format, prefixInfo,
		)
		items = append(items, item)
	}
	return items
}

// appendMetadataKeyCompletionItems adds completion items for metadata keys.
// Uses bracket notation for keys containing dots or special characters.
// When isBracketTrigger is true, all keys are wrapped as `"key"]` for bracket access.
func appendMetadataKeyCompletionItems(
	items []*lsp.CompletionItem,
	keys []string,
	detail string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
	isBracketTrigger bool,
) []*lsp.CompletionItem {
	for _, key := range keys {
		if !filterByPrefix(key, prefixInfo) {
			continue
		}
		if isBracketTrigger {
			item := buildBracketInsertionCompletionItem(key, detail, position, prefixInfo)
			items = append(items, item)
		} else {
			item := buildExportFieldKeyCompletionItem(key, detail, position, format, prefixInfo)
			items = append(items, item)
		}
	}
	return items
}

// buildBracketInsertionCompletionItem creates a completion item for bracket insertion context.
// Inserts `"key"]` when the user has typed `[`.
func buildBracketInsertionCompletionItem(
	key string,
	detail string,
	position *lsp.Position,
	prefixInfo completionPrefixInfo,
) *lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField
	insertText := formatMapKeyForBracketInsertion(key, docmodel.QuoteTypeDouble)
	insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)

	return &lsp.CompletionItem{
		Label:  key,
		Detail: &detail,
		Kind:   &fieldKind,
		TextEdit: lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		},
		Data: map[string]any{"completionType": "exportField"},
	}
}

// buildExportFieldKeyCompletionItem creates a completion item for a key that may need bracket notation.
func buildExportFieldKeyCompletionItem(
	key string,
	detail string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) *lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField

	if needsBracketNotation(key) {
		return buildBracketNotationCompletionItem(key, detail, position, format, prefixInfo)
	}

	// Normal key - use standard completion item
	insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
	formattedText, useSnippet := formatExportFieldValueForInsert(key, format, prefixInfo)

	item := &lsp.CompletionItem{
		Label:  key,
		Detail: &detail,
		Kind:   &fieldKind,
		TextEdit: lsp.TextEdit{
			NewText: formattedText,
			Range:   insertRange,
		},
		Data: map[string]any{"completionType": "exportField"},
	}

	if useSnippet {
		snippetFormat := lsp.InsertTextFormatSnippet
		item.InsertTextFormat = &snippetFormat
	}

	return item
}

// buildBracketNotationCompletionItem creates a completion item using bracket notation.
// For keys with dots or special characters, uses ["key"] syntax and replaces the trailing dot.
func buildBracketNotationCompletionItem(
	key string,
	detail string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) *lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField

	// Use double quotes for bracket notation (standard in both YAML and JSONC export fields)
	bracketNotation := formatBracketNotation(key, docmodel.QuoteTypeDouble)

	// Calculate the insert range - need to replace the trailing "." the user typed
	insertRange := getBracketNotationInsertRangeForExportField(position, prefixInfo)

	// Format for JSONC if needed
	insertText := bracketNotation
	useSnippet := false

	if format == docmodel.FormatJSONC && !prefixInfo.HasTrailingQuote && !prefixInfo.HasLeadingQuote {
		// For JSONC, wrap the entire value in quotes if not already in a string
		if prefixInfo.HasLeadingSpace {
			insertText = `"` + bracketNotation + `"`
		} else {
			insertText = ` "` + bracketNotation + `"`
		}
	}

	item := &lsp.CompletionItem{
		Label:  key,
		Detail: &detail,
		Kind:   &fieldKind,
		TextEdit: lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		},
		Data: map[string]any{"completionType": "exportField"},
	}

	if useSnippet {
		snippetFormat := lsp.InsertTextFormatSnippet
		item.InsertTextFormat = &snippetFormat
	}

	return item
}

// getBracketNotationInsertRangeForExportField returns the range for bracket notation in export fields.
// This replaces the trailing "." and any typed prefix.
func getBracketNotationInsertRangeForExportField(
	position *lsp.Position,
	prefixInfo completionPrefixInfo,
) *lsp.Range {
	// Start from current position, minus prefix length, minus 1 for the trailing "."
	startChar := position.Character - lsp.UInteger(prefixInfo.PrefixLen)
	if startChar > 0 {
		startChar-- // Replace the trailing "." as well
	}

	return &lsp.Range{
		Start: lsp.Position{
			Line:      position.Line,
			Character: startChar,
		},
		End: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
	}
}

// getExportFieldValuePropertyCompletionItems returns value property completions for export fields.
// Navigates the Value.Value MappingNode tree based on the typed path.
func (s *CompletionService) getExportFieldValuePropertyCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Values == nil {
		return []*lsp.CompletionItem{}, nil
	}

	fieldValue := extractExportFieldValueFromContext(completionCtx)
	pathParts := splitFieldPath(fieldValue)
	// pathParts: [values, {name}, {field1}, {field2}, ...]
	if len(pathParts) < 2 {
		return []*lsp.CompletionItem{}, nil
	}

	valueName := pathParts[1]
	valueDef := blueprint.Values.Values[valueName]
	if valueDef == nil || valueDef.Value == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)

	// Navigate from index 2 onwards into the MappingNode tree.
	current := valueDef.Value
	for i := 2; i < len(pathParts); i++ {
		part := pathParts[i]
		if part == "" {
			continue
		}
		if current == nil {
			return []*lsp.CompletionItem{}, nil
		}
		current = navigateMappingNode(current, []string{part})
	}

	if current == nil || isMappingNodeTerminal(current) {
		return []*lsp.CompletionItem{}, nil
	}

	if isMappingNodeObject(current) {
		keys := getMappingNodeFieldKeys(current)
		return appendMappingNodeKeyExportFieldItems(
			nil, keys, "Value field", position, format, prefixInfo, current,
		), nil
	}

	if isMappingNodeArray(current) {
		return getExportFieldMappingNodeArrayIndexItems(current, position, format, prefixInfo), nil
	}

	return []*lsp.CompletionItem{}, nil
}

// getExportFieldVariableRefCompletionItems returns variable names for export field references.
func (s *CompletionService) getExportFieldVariableRefCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Variables == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(blueprint.Variables.Values))

	for varName, variable := range blueprint.Variables.Values {
		if !filterByPrefix(varName, prefixInfo) {
			continue
		}
		desc := getVariableDescription(variable)
		item := buildExportFieldCompletionItem(
			varName,
			varName,
			"Variable",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getVariableDescription extracts the description from a variable definition.
func getVariableDescription(variable *schema.Variable) string {
	if variable == nil || variable.Description == nil || variable.Description.StringValue == nil {
		return ""
	}
	return *variable.Description.StringValue
}

// getExportFieldValueRefCompletionItems returns value names for export field references.
func (s *CompletionService) getExportFieldValueRefCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Values == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(blueprint.Values.Values))

	for valueName, value := range blueprint.Values.Values {
		if !filterByPrefix(valueName, prefixInfo) {
			continue
		}
		desc := getValueDescription(value)
		// Add trailing dot if the value has navigable MappingNode content.
		insertText := valueName
		if value.Value != nil && (isMappingNodeObject(value.Value) || isMappingNodeArray(value.Value)) {
			insertText = valueName + "."
		}
		item := buildExportFieldCompletionItem(
			insertText,
			valueName,
			"Value",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getValueDescription extracts the description from a value definition.
func getValueDescription(value *schema.Value) string {
	if value == nil || value.Description == nil {
		return ""
	}
	return getStringOrSubstitutionsValue(value.Description)
}

// getExportFieldChildRefCompletionItems returns child blueprint names for export field references.
func (s *CompletionService) getExportFieldChildRefCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Include == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(blueprint.Include.Values))

	for childName, include := range blueprint.Include.Values {
		if !filterByPrefix(childName, prefixInfo) {
			continue
		}
		desc := getIncludeDescription(include)
		item := buildExportFieldCompletionItem(
			childName+".",
			childName,
			"Child blueprint",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getIncludeDescription extracts the description from an include definition.
func getIncludeDescription(include *schema.Include) string {
	if include == nil || include.Description == nil {
		return ""
	}
	return getStringOrSubstitutionsValue(include.Description)
}

// getExportFieldChildPropertyCompletionItems returns child blueprint export names.
// Note: This requires resolved child blueprint exports, which may not be available.
func (s *CompletionService) getExportFieldChildPropertyCompletionItems(
	_ *lsp.Position,
	_ *schema.Blueprint,
	_ *docmodel.CompletionContext,
	_ docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	// Child blueprint exports would require resolving the child blueprint,
	// which is not available in the current context. Return empty for now.
	// This could be enhanced in the future if child blueprint resolution is available.
	return []*lsp.CompletionItem{}, nil
}

// getExportFieldDataSourceRefCompletionItems returns data source names for export field references.
func (s *CompletionService) getExportFieldDataSourceRefCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.DataSources == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(blueprint.DataSources.Values))

	for dsName, dataSource := range blueprint.DataSources.Values {
		if !filterByPrefix(dsName, prefixInfo) {
			continue
		}
		desc := getDataSourceDescription(dataSource)
		item := buildExportFieldCompletionItem(
			dsName+".",
			dsName,
			"Data source",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getDataSourceDescription extracts the description from a data source definition.
func getDataSourceDescription(dataSource *schema.DataSource) string {
	if dataSource == nil || dataSource.Description == nil {
		return ""
	}
	return getStringOrSubstitutionsValue(dataSource.Description)
}

// getExportFieldDataSourcePropertyCompletionItems returns data source export names.
func (s *CompletionService) getExportFieldDataSourcePropertyCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	dataSourceName := completionCtx.DataSourceName
	if dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	dataSource := getDataSource(blueprint, dataSourceName)
	if dataSource == nil || dataSource.Exports == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractExportFieldPrefixInfo(completionCtx, format)
	items := make([]*lsp.CompletionItem, 0, len(dataSource.Exports.Values))

	for exportName, export := range dataSource.Exports.Values {
		if !filterByPrefix(exportName, prefixInfo) {
			continue
		}
		desc := getDataSourceExportDescription(export)
		item := buildExportFieldCompletionItem(
			exportName,
			exportName,
			"Data source export",
			desc,
			position,
			format,
			prefixInfo,
		)
		items = append(items, item)
	}

	return items, nil
}

// getDataSourceExportDescription extracts the description from a data source export.
func getDataSourceExportDescription(export *schema.DataSourceFieldExport) string {
	if export == nil || export.Description == nil {
		return ""
	}
	return getStringOrSubstitutionsValue(export.Description)
}

// buildExportFieldCompletionItem creates a completion item for export field references.
func buildExportFieldCompletionItem(
	insertText string,
	label string,
	detail string,
	description string,
	position *lsp.Position,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) *lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField
	insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
	isProgressive := strings.HasSuffix(insertText, ".")

	// Format value for insertion based on format
	formattedInsertText, useSnippet := formatExportFieldValueForInsert(insertText, format, prefixInfo)

	// Build FilterText to include the full path prefix for proper client-side filtering.
	// This helps VSCode match completions when its word detection includes the full path.
	filterText := prefixInfo.TypedPrefix + label

	item := &lsp.CompletionItem{
		Label:      label,
		Detail:     &detail,
		Kind:       &fieldKind,
		FilterText: &filterText,
		TextEdit: lsp.TextEdit{
			NewText: formattedInsertText,
			Range:   insertRange,
		},
		Data: map[string]any{"completionType": "exportField"},
	}

	// Use snippet format for JSONC progressive completions to position cursor before closing quote
	if useSnippet {
		snippetFormat := lsp.InsertTextFormatSnippet
		item.InsertTextFormat = &snippetFormat
	}

	// For progressive completions (paths ending with "."), add a command to retrigger
	// the completion menu automatically after insertion.
	if isProgressive {
		item.Command = &lsp.Command{
			Title:   "Trigger Suggestions",
			Command: "editor.action.triggerSuggest",
		}
	}

	if description != "" {
		item.Documentation = description
	}

	return item
}

// formatExportFieldValueForInsert formats the value for insertion in the document.
// Returns the formatted text and whether snippet format should be used.
// For JSONC with progressive completions (values ending in "."), uses snippet format
// with $0 cursor position before the closing quote.
func formatExportFieldValueForInsert(
	value string,
	format docmodel.DocumentFormat,
	prefixInfo completionPrefixInfo,
) (string, bool) {
	isProgressive := strings.HasSuffix(value, ".")

	if format == docmodel.FormatJSONC {
		// If there's already a trailing quote (user is inside an existing string),
		// don't add quotes - just insert the value
		if prefixInfo.HasTrailingQuote {
			// Inside existing string - just insert value, use snippet for cursor position
			if isProgressive {
				return value + "$0", true
			}
			return value, false
		}

		// For JSONC progressive completions, use snippet to position cursor before closing quote
		if isProgressive {
			if prefixInfo.HasLeadingQuote {
				// User already typed opening quote: insert value + cursor + closing quote
				return value + `$0"`, true
			}
			if prefixInfo.HasLeadingSpace {
				// Space before cursor: insert quoted value with cursor before close
				return `"` + value + `$0"`, true
			}
			// Insert with leading space and cursor before close
			return ` "` + value + `$0"`, true
		}

		// Non-progressive: standard quoting without snippet
		if prefixInfo.HasLeadingQuote {
			return value + `"`, false
		}
		if prefixInfo.HasLeadingSpace {
			return `"` + value + `"`, false
		}
		return ` "` + value + `"`, false
	}

	// For YAML, no special handling needed
	return value, false
}

// extractExportFieldValueFromContext extracts the field value from completion context.
func extractExportFieldValueFromContext(completionCtx *docmodel.CompletionContext) string {
	if completionCtx == nil || completionCtx.CursorCtx == nil {
		return ""
	}
	return extractExportFieldValueHelper(completionCtx.TextBefore)
}

// extractExportFieldValueHelper extracts the value portion after "field:" from text.
func extractExportFieldValueHelper(textBefore string) string {
	// Same logic as in completion_context.go but accessible here
	idx := findFieldValueStart(textBefore)
	if idx == -1 {
		return ""
	}
	return textBefore[idx:]
}

// findFieldValueStart finds the starting index of the field value in text.
func findFieldValueStart(textBefore string) int {
	// Try YAML pattern: field:
	if idx := lastIndexOfPattern(textBefore, "field:"); idx != -1 {
		start := idx + len("field:")
		// Skip whitespace after colon
		for start < len(textBefore) && (textBefore[start] == ' ' || textBefore[start] == '\t') {
			start += 1
		}
		return start
	}

	// Try JSONC pattern: "field":
	if idx := lastIndexOfPattern(textBefore, "\"field\":"); idx != -1 {
		start := idx + len("\"field\":")
		// Skip whitespace after colon
		for start < len(textBefore) && (textBefore[start] == ' ' || textBefore[start] == '\t') {
			start += 1
		}
		// Skip leading quote if present
		if start < len(textBefore) && textBefore[start] == '"' {
			start += 1
		}
		return start
	}

	return -1
}

// lastIndexOfPattern finds the last occurrence of a pattern in text.
func lastIndexOfPattern(text, pattern string) int {
	for i := len(text) - len(pattern); i >= 0; i-- {
		if text[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}

// countDots counts the number of dots in a string.
func countDots(s string) int {
	count := 0
	for _, c := range s {
		if c == '.' {
			count += 1
		}
	}
	return count
}

// containsSegment checks if the field value contains a specific path segment.
func containsSegment(fieldValue, segment string) bool {
	parts := splitFieldPath(fieldValue)
	for _, part := range parts {
		if part == segment {
			return true
		}
	}
	return false
}

// splitFieldPath splits a field value path by dots.
func splitFieldPath(fieldValue string) []string {
	if fieldValue == "" {
		return nil
	}
	var parts []string
	start := 0
	for i, c := range fieldValue {
		if c == '.' {
			parts = append(parts, fieldValue[start:i])
			start = i + 1
		}
	}
	if start < len(fieldValue) {
		parts = append(parts, fieldValue[start:])
	}
	return parts
}

// extractExportFieldPrefixInfo extracts prefix information for export field completions.
// Unlike regular completions, export fields use progressive path completion where we need
// to filter by only the partial segment being typed (after the last dot or bracket), not the entire path.
func extractExportFieldPrefixInfo(
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) completionPrefixInfo {
	basePrefixInfo := extractCompletionPrefix(completionCtx, format)

	// Get the field value (path typed so far)
	fieldValue := extractExportFieldValueFromContext(completionCtx)

	// Extract just the partial segment for filtering
	// For bracket notation, use content after the last '[' if it's after the last '.'
	// For dot notation, use content after the last '.'
	partialSegment := extractPartialSegment(fieldValue)

	// Extract the path prefix (everything before the partial segment) for FilterText
	pathPrefix := extractPathPrefix(fieldValue)

	// Create modified prefix info with just the partial segment for filtering
	return completionPrefixInfo{
		TypedPrefix:      pathPrefix, // Use pathPrefix for building FilterText
		TextBefore:       basePrefixInfo.TextBefore,
		TextAfter:        basePrefixInfo.TextAfter,
		FilterPrefix:     partialSegment,
		HasLeadingQuote:  basePrefixInfo.HasLeadingQuote,
		HasLeadingSpace:  basePrefixInfo.HasLeadingSpace,
		HasTrailingQuote: basePrefixInfo.HasTrailingQuote,
		PrefixLen:        len(partialSegment), // Only the partial segment length for replace range
		PrefixLower:      strings.ToLower(partialSegment),
	}
}

// extractPathPrefix extracts the path prefix from a field value (everything before the partial segment).
// For "resources.myTable.spec.ar", returns "resources.myTable.spec.".
// For "resources.myTable.spec.", returns "resources.myTable.spec.".
func extractPathPrefix(fieldValue string) string {
	if fieldValue == "" {
		return ""
	}

	lastDot := strings.LastIndex(fieldValue, ".")
	lastBracket := strings.LastIndex(fieldValue, "[")

	// If bracket is after the last dot, include up to and including the bracket
	if lastBracket > lastDot {
		return fieldValue[:lastBracket+1]
	}

	// Otherwise include up to and including the last dot
	if lastDot >= 0 {
		return fieldValue[:lastDot+1]
	}

	return ""
}

// extractPartialSegment extracts the partial segment being typed from a field path.
// Handles both dot notation (after '.') and bracket notation (after '[').
func extractPartialSegment(fieldValue string) string {
	if fieldValue == "" {
		return ""
	}

	lastDot := strings.LastIndex(fieldValue, ".")
	lastBracket := strings.LastIndex(fieldValue, "[")

	// If bracket is after the last dot, use content after bracket
	// This handles cases like "items[" or "items[0" where we're in bracket notation
	if lastBracket > lastDot {
		return fieldValue[lastBracket+1:]
	}

	// Otherwise use content after the last dot
	if lastDot >= 0 {
		return fieldValue[lastDot+1:]
	}

	return fieldValue
}

// getStringOrSubstitutionsValue extracts a plain string value from StringOrSubstitutions.
func getStringOrSubstitutionsValue(sos *substitutions.StringOrSubstitutions) string {
	if sos == nil {
		return ""
	}
	// Try to get the plain string value if available
	if len(sos.Values) == 1 && sos.Values[0].StringValue != nil {
		return *sos.Values[0].StringValue
	}
	return ""
}
