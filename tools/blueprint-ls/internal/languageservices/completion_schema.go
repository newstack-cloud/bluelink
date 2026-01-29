package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// coreResourceDefinitionFieldInfo holds description information for core resource fields.
type coreResourceDefinitionFieldInfo struct {
	name        string
	description string
}

// Core blueprint resource definition fields with descriptions.
var coreResourceDefinitionFields = []coreResourceDefinitionFieldInfo{
	{
		name:        "type",
		description: "The resource type identifier (e.g., `aws/dynamodb/table`). This determines which provider handles the resource.",
	},
	{
		name:        "description",
		description: "A human-readable description of the resource's purpose in the blueprint.",
	},
	{
		name:        "spec",
		description: "The resource specification containing provider-specific configuration fields.",
	},
	{
		name:        "metadata",
		description: "Optional metadata including `displayName`, `labels`, `annotations`, and `custom` fields.",
	},
	{
		name:        "condition",
		description: "A condition expression that determines whether this resource should be deployed. Supports `and`, `or`, and `not` operators.",
	},
	{
		name:        "each",
		description: "Creates multiple instances of this resource from an array or map. Use `elem` to reference the current item.",
	},
	{
		name:        "linkSelector",
		description: "Defines criteria for automatic link creation with other resources based on labels.",
	},
	{
		name:        "dependsOn",
		description: "Explicitly declares dependencies on other resources that must be deployed first.",
	},
}

// Variable definition fields.
var coreVariableDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The variable type (string, integer, float, boolean, or a custom type)."},
	{name: "description", description: "A human-readable description of the variable's purpose."},
	{name: "secret", description: "When true, the variable value is treated as sensitive and masked in logs."},
	{name: "default", description: "The default value used when no value is provided."},
	{name: "allowedValues", description: "An array of valid values that constrain input validation."},
}

// Value definition fields.
var coreValueDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The value type (string, integer, float, boolean, array, object)."},
	{name: "value", description: "The computed value, which may include substitutions."},
	{name: "description", description: "A human-readable description of the value's purpose."},
	{name: "secret", description: "When true, the value is treated as sensitive and masked in logs."},
}

// DataSource definition fields.
var coreDataSourceDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The data source type identifier (e.g., `aws/vpc`)."},
	{name: "metadata", description: "Metadata including `displayName`, `annotations`, and `custom` fields."},
	{name: "filter", description: "Filter criteria to select specific data source instances."},
	{name: "exports", description: "Field definitions for data exported from this data source."},
	{name: "description", description: "A human-readable description of the data source's purpose."},
}

// DataSource filter definition fields (inside filter).
var coreDataSourceFilterDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "field", description: "The name of the data source field to filter on."},
	{name: "operator", description: "The comparison operator (=, !=, in, not in, contains, starts with, etc.)."},
	{name: "search", description: "The value(s) to search for, which may include substitutions."},
}

// DataSource export definition fields (inside exports/{name}).
var coreDataSourceExportDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The export field type (string, integer, float, boolean, array)."},
	{name: "aliasFor", description: "The original field name from the provider if different from the export name."},
	{name: "description", description: "A human-readable description of the exported field."},
}

// DataSource metadata fields (inside metadata).
var coreDataSourceMetadataFields = []coreResourceDefinitionFieldInfo{
	{name: "displayName", description: "A human-readable name for the data source."},
	{name: "annotations", description: "Key-value pairs for configuring data source behavior."},
	{name: "custom", description: "Custom metadata fields for provider-specific configuration."},
}

// Include definition fields.
var coreIncludeDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "path", description: "The path to the child blueprint (local file or remote URL)."},
	{name: "variables", description: "Variables to pass to the child blueprint."},
	{name: "metadata", description: "Extra metadata for include resolver plugins."},
	{name: "description", description: "A human-readable description of the included blueprint."},
}

// Export definition fields.
var coreExportDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The export type (string, integer, float, boolean, array, object)."},
	{name: "field", description: "A substitution reference to the value being exported."},
	{name: "description", description: "A human-readable description of the exported value."},
}

// LinkSelector definition fields.
var coreLinkSelectorDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "byLabel", description: "Map of label key-value pairs to match resources for linking."},
	{name: "exclude", description: "List of resource names to exclude from link matching."},
}

// Blueprint top-level fields.
var coreBlueprintTopLevelFields = []coreResourceDefinitionFieldInfo{
	{name: "version", description: "The blueprint specification version (e.g., `2025-11-02`)."},
	{name: "transform", description: "One or more transforms to apply to the blueprint."},
	{name: "variables", description: "Input variables that parameterize the blueprint."},
	{name: "values", description: "Computed values derived from variables and other sources."},
	{name: "include", description: "Child blueprints to include in this blueprint."},
	{name: "resources", description: "The infrastructure resources defined in this blueprint."},
	{name: "datasources", description: "External data sources to look up existing infrastructure."},
	{name: "exports", description: "Values exported from this blueprint for external access."},
	{name: "metadata", description: "Blueprint-level metadata for organization and documentation."},
}

// resourcePropInfo holds label and description for resource properties.
type resourcePropInfo struct {
	label       string
	description string
}

var resourceTopLevelProps = []resourcePropInfo{
	{label: "metadata", description: "Resource metadata including `displayName`, `labels`, `annotations`, and `custom` fields."},
	{label: "spec", description: "The resource specification containing provider-specific configuration and computed fields."},
	{label: "state", description: "The current deployment state of the resource from the external provider."},
}

var resourceMetadataProps = []resourcePropInfo{
	{label: "displayName", description: "A human-readable name for the resource, used in UI displays."},
	{label: "labels", description: "Key-value pairs for organizing and selecting resources. Used by `linkSelector`."},
	{label: "annotations", description: "Key-value pairs for storing additional metadata. Unlike labels, annotations are not used for selection."},
	{label: "custom", description: "Custom metadata fields specific to your use case."},
}

// getResourceSpecFieldCompletionItems returns completion items for resource spec fields
// when editing directly in the YAML/JSONC definition (not in substitutions).
func (s *CompletionService) getResourceSpecFieldCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	resourceName := completionCtx.ResourceName
	if resourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	resource := getResource(blueprint, resourceName)
	if resource == nil || resource.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
		ctx.Context,
		resource.Type.Value,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return noCompletionsHint(
			position,
			fmt.Sprintf("Resource type '%s' not found - install the provider to enable completions", resource.Type.Value),
		), nil
	}

	if specDefOutput.SpecDefinition == nil || specDefOutput.SpecDefinition.Schema == nil {
		return noCompletionsHint(
			position,
			fmt.Sprintf("No spec schema available for resource type '%s'", resource.Type.Value),
		), nil
	}

	specPath := completionCtx.CursorCtx.StructuralPath.GetSpecPath()
	currentSchema := specDefOutput.SpecDefinition.Schema

	for i := 0; i < len(specPath) && currentSchema != nil; i++ {
		segment := specPath[i]
		if segment.Kind != docmodel.PathSegmentField {
			continue
		}
		if currentSchema.Type != provider.ResourceDefinitionsSchemaTypeObject {
			currentSchema = nil
			break
		}
		currentSchema = currentSchema.Attributes[segment.FieldName]
	}

	if currentSchema == nil || currentSchema.Attributes == nil {
		return []*lsp.CompletionItem{}, nil
	}

	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}

	return resourceDefAttributesSchemaCompletionItemsWithPrefix(
		currentSchema.Attributes,
		position,
		"Resource spec field",
		typedPrefix,
	), nil
}

// getResourceSpecFieldValueCompletionItems returns completion items for resource spec field values
// when the field has AllowedValues defined in the provider schema.
func (s *CompletionService) getResourceSpecFieldValueCompletionItems(
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
	if resource == nil || resource.Type == nil {
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

	specPath := completionCtx.CursorCtx.StructuralPath.GetSpecPath()

	// If specPath is empty but we have an extracted field name from TextBefore,
	// use that instead. This handles the case where the cursor is after "fieldName: "
	// but outside the AST node's range.
	if len(specPath) == 0 && completionCtx.CursorCtx.ExtractedFieldName != "" {
		specPath = []docmodel.PathSegment{
			{Kind: docmodel.PathSegmentField, FieldName: completionCtx.CursorCtx.ExtractedFieldName},
		}
	}

	if len(specPath) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	fieldSchema := navigateToFieldSchema(specDefOutput.SpecDefinition.Schema, specPath)
	if fieldSchema == nil {
		return []*lsp.CompletionItem{}, nil
	}

	if len(fieldSchema.AllowedValues) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	typedPrefix := ""
	textBefore := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
		textBefore = completionCtx.CursorCtx.TextBefore
	}

	return allowedValuesCompletionItems(fieldSchema, position, typedPrefix, textBefore, format), nil
}

// getResourceDefinitionFieldCompletionItems returns completion items for core resource fields
// like type, description, spec, metadata, etc. when editing at the resource definition level.
func (s *CompletionService) getResourceDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}

	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)

	detail := "Resource field"
	fieldKind := lsp.CompletionItemKindField

	items := make([]*lsp.CompletionItem, 0, len(coreResourceDefinitionFields))
	for _, fieldInfo := range coreResourceDefinitionFields {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(fieldInfo.name), prefixLower) {
			continue
		}

		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		insertText := fieldInfo.name + ": "
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      fieldInfo.name,
			Detail:     &detail,
			Kind:       &fieldKind,
			TextEdit:   edit,
			FilterText: &fieldInfo.name,
			Data: map[string]any{
				"completionType": "resourceDefinitionField",
			},
		}

		if fieldInfo.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: fieldInfo.description,
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// createDefinitionFieldCompletionItems is a reusable helper for generating
// definition field completion items with YAML syntax.
func createDefinitionFieldCompletionItems(
	fields []coreResourceDefinitionFieldInfo,
	position *lsp.Position,
	typedPrefix string,
	detailText string,
	completionType string,
) []*lsp.CompletionItem {
	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)
	fieldKind := lsp.CompletionItemKindField

	items := make([]*lsp.CompletionItem, 0, len(fields))
	for _, fieldInfo := range fields {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(fieldInfo.name), prefixLower) {
			continue
		}

		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		insertText := fieldInfo.name + ": "
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      fieldInfo.name,
			Detail:     &detailText,
			Kind:       &fieldKind,
			TextEdit:   edit,
			FilterText: &fieldInfo.name,
			Data:       map[string]any{"completionType": completionType},
		}

		if fieldInfo.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: fieldInfo.description,
			}
		}

		items = append(items, item)
	}

	return items
}

// getVariableDefinitionFieldCompletionItems returns completion items for variable definition fields.
func (s *CompletionService) getVariableDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreVariableDefinitionFields,
		position,
		typedPrefix,
		"Variable field",
		"variableDefinitionField",
	), nil
}

// getValueDefinitionFieldCompletionItems returns completion items for value definition fields.
func (s *CompletionService) getValueDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreValueDefinitionFields,
		position,
		typedPrefix,
		"Value field",
		"valueDefinitionField",
	), nil
}

// getDataSourceDefinitionFieldCompletionItems returns completion items for data source definition fields.
func (s *CompletionService) getDataSourceDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreDataSourceDefinitionFields,
		position,
		typedPrefix,
		"Data source field",
		"dataSourceDefinitionField",
	), nil
}

// getDataSourceFilterDefinitionFieldCompletionItems returns completion items for fields inside a data source filter.
func (s *CompletionService) getDataSourceFilterDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreDataSourceFilterDefinitionFields,
		position,
		typedPrefix,
		"Filter field",
		"dataSourceFilterDefinitionField",
	), nil
}

// getDataSourceExportDefinitionFieldCompletionItems returns completion items for fields inside a data source export.
func (s *CompletionService) getDataSourceExportDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreDataSourceExportDefinitionFields,
		position,
		typedPrefix,
		"Export field",
		"dataSourceExportDefinitionField",
	), nil
}

// getDataSourceMetadataFieldCompletionItems returns completion items for fields inside data source metadata.
func (s *CompletionService) getDataSourceMetadataFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreDataSourceMetadataFields,
		position,
		typedPrefix,
		"Metadata field",
		"dataSourceMetadataField",
	), nil
}

// getIncludeDefinitionFieldCompletionItems returns completion items for include definition fields.
func (s *CompletionService) getIncludeDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreIncludeDefinitionFields,
		position,
		typedPrefix,
		"Include field",
		"includeDefinitionField",
	), nil
}

// getExportDefinitionFieldCompletionItems returns completion items for export definition fields.
func (s *CompletionService) getExportDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreExportDefinitionFields,
		position,
		typedPrefix,
		"Export field",
		"exportDefinitionField",
	), nil
}

// getLinkSelectorFieldCompletionItems returns completion items for linkSelector fields.
func (s *CompletionService) getLinkSelectorFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreLinkSelectorDefinitionFields,
		position,
		typedPrefix,
		"LinkSelector field",
		"linkSelectorField",
	), nil
}

// getLinkSelectorExcludeValueCompletionItems returns resource names as completions
// for values in the linkSelector.exclude list.
func (s *CompletionService) getLinkSelectorExcludeValueCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Resources == nil {
		return []*lsp.CompletionItem{}, nil
	}

	currentResourceName := completionCtx.ResourceName
	typedPrefix := ""
	textBefore := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
		textBefore = completionCtx.CursorCtx.TextBefore
	}

	// For JSONC, handle the case where cursor is inside a quoted string
	filterPrefix := typedPrefix
	hasLeadingQuote := false
	prefixLen := len(typedPrefix)

	if format == docmodel.FormatJSONC {
		filterPrefix, hasLeadingQuote = stripLeadingQuote(typedPrefix)
		// If typedPrefix is empty but textBefore ends with a quote,
		// we're inside an empty or partially typed string in a JSONC array
		if typedPrefix == "" && len(textBefore) > 0 && textBefore[len(textBefore)-1] == '"' {
			hasLeadingQuote = true
			prefixLen = 1 // Include the opening quote in the replacement range
		}
	}

	hasLeadingSpace := hasLeadingWhitespace(textBefore, prefixLen)
	prefixLower := strings.ToLower(filterPrefix)
	detail := "Resource name"
	valueKind := lsp.CompletionItemKindValue

	var items []*lsp.CompletionItem
	for resourceName := range blueprint.Resources.Values {
		if resourceName == currentResourceName {
			continue
		}
		if len(filterPrefix) > 0 && !strings.HasPrefix(strings.ToLower(resourceName), prefixLower) {
			continue
		}

		insertText := formatValueForInsert(resourceName, format, hasLeadingQuote, hasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		item := &lsp.CompletionItem{
			Label:  resourceName,
			Detail: &detail,
			Kind:   &valueKind,
			TextEdit: lsp.TextEdit{
				NewText: insertText,
				Range:   insertRange,
			},
			FilterText: &resourceName,
			Data: map[string]any{
				"completionType": "linkSelectorExcludeValue",
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// getBlueprintTopLevelFieldCompletionItems returns completion items for blueprint top-level fields.
func (s *CompletionService) getBlueprintTopLevelFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreBlueprintTopLevelFields,
		position,
		typedPrefix,
		"Blueprint field",
		"blueprintTopLevelField",
	), nil
}

// getResourceMetadataFieldCompletionItems returns completion items for metadata fields
// when editing directly in YAML resource definitions (with colons for key-value syntax).
func (s *CompletionService) getResourceMetadataFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}

	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)

	detail := "Metadata field"
	fieldKind := lsp.CompletionItemKindField

	items := make([]*lsp.CompletionItem, 0, len(resourceMetadataProps))
	for _, propInfo := range resourceMetadataProps {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(propInfo.label), prefixLower) {
			continue
		}

		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		insertText := propInfo.label + ": "
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      propInfo.label,
			Detail:     &detail,
			Kind:       &fieldKind,
			TextEdit:   edit,
			FilterText: &propInfo.label,
			Data: map[string]any{
				"completionType": "metadataField",
			},
		}

		if propInfo.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: propInfo.description,
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// getResourceMetadataPropCompletionItemsForPath returns metadata property completions
// based on the current path depth. At "metadata." it returns top-level metadata props.
// At deeper paths like "metadata.annotations." it returns the keys from the resource's metadata.
// For "metadata.custom." and deeper paths, navigates the MappingNode tree.
func (s *CompletionService) getResourceMetadataPropCompletionItemsForPath(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	resourceProp *substitutions.SubstitutionResourceProperty,
	cursorCtx *docmodel.CursorContext,
	partialPrefix string,
) ([]*lsp.CompletionItem, error) {
	if len(resourceProp.Path) <= 1 {
		return getResourceMetadataPropCompletionItemsWithPrefix(position, partialPrefix), nil
	}

	resource := getResource(blueprint, resourceProp.ResourceName)
	if resource == nil || resource.Metadata == nil {
		return []*lsp.CompletionItem{}, nil
	}

	secondSegment := resourceProp.Path[1].FieldName

	// For custom metadata, support deep navigation into the MappingNode tree.
	if secondSegment == "custom" && len(resourceProp.Path) > 2 {
		return getCustomMetadataDeepCompletionItems(
			position, resource.Metadata.Custom, resourceProp.Path[2:], cursorCtx,
		)
	}

	keys, detail := getMetadataKeysAndDetail(resource.Metadata, secondSegment)
	if keys == nil {
		return []*lsp.CompletionItem{}, nil
	}

	quoteType := docmodel.QuoteTypeNone
	isBracketTrigger := false
	if cursorCtx != nil {
		quoteType = cursorCtx.GetEnclosingQuoteType()
		isBracketTrigger = strings.HasSuffix(cursorCtx.TextBefore, "[")
	}

	return createMetadataKeyCompletionItems(position, keys, detail, quoteType, isBracketTrigger), nil
}

// getCustomMetadataDeepCompletionItems navigates into a custom MappingNode tree
// and returns field keys or array indices at the current depth.
func getCustomMetadataDeepCompletionItems(
	position *lsp.Position,
	customNode *core.MappingNode,
	remainingPath []*substitutions.SubstitutionPathItem,
	cursorCtx *docmodel.CursorContext,
) ([]*lsp.CompletionItem, error) {
	if customNode == nil {
		return []*lsp.CompletionItem{}, nil
	}

	// Navigate through the path to reach the target node.
	current := customNode
	for _, pathItem := range remainingPath {
		if current == nil {
			return []*lsp.CompletionItem{}, nil
		}
		if pathItem.ArrayIndex != nil {
			current = navigateMappingNodeByIndex(current, int(*pathItem.ArrayIndex))
		} else if pathItem.FieldName != "" {
			current = navigateMappingNode(current, []string{pathItem.FieldName})
		}
	}

	return mappingNodeCompletionItems(position, current, cursorCtx)
}

// mappingNodeCompletionItems returns field keys or array indices for a MappingNode.
func mappingNodeCompletionItems(
	position *lsp.Position,
	node *core.MappingNode,
	cursorCtx *docmodel.CursorContext,
) ([]*lsp.CompletionItem, error) {
	if node == nil || isMappingNodeTerminal(node) {
		return []*lsp.CompletionItem{}, nil
	}

	quoteType := docmodel.QuoteTypeNone
	isBracketTrigger := false
	if cursorCtx != nil {
		quoteType = cursorCtx.GetEnclosingQuoteType()
		isBracketTrigger = strings.HasSuffix(cursorCtx.TextBefore, "[")
	}

	if isMappingNodeObject(node) {
		keys := getMappingNodeFieldKeys(node)
		return createMetadataKeyCompletionItems(position, keys, "Custom metadata field", quoteType, isBracketTrigger), nil
	}

	if isMappingNodeArray(node) {
		return createMappingNodeArrayIndexCompletionItems(position, node), nil
	}

	return []*lsp.CompletionItem{}, nil
}

// createMappingNodeArrayIndexCompletionItems returns index completions for a MappingNode array.
func createMappingNodeArrayIndexCompletionItems(
	position *lsp.Position,
	node *core.MappingNode,
) []*lsp.CompletionItem {
	length := getMappingNodeArrayLength(node)
	if length == 0 {
		return []*lsp.CompletionItem{}
	}

	items := make([]*lsp.CompletionItem, 0, length)
	fieldKind := lsp.CompletionItemKindValue
	detail := "Array index"
	for i := range length {
		label := fmt.Sprintf("%d", i)
		items = append(items, &lsp.CompletionItem{
			Label:  label,
			Detail: &detail,
			Kind:   &fieldKind,
			TextEdit: lsp.TextEdit{
				NewText: label,
				Range:   getItemInsertRange(position),
			},
			Data: map[string]any{"completionType": "arrayIndex"},
		})
	}
	return items
}

func getResourceTopLevelPropCompletionItems(position *lsp.Position) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithDescriptions(position, resourceTopLevelProps, "Resource property")
}

func getResourceTopLevelPropCompletionItemsWithPrefix(position *lsp.Position, prefix string) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithPrefix(position, resourceTopLevelProps, "Resource property", prefix)
}

func getResourceMetadataPropCompletionItemsWithPrefix(position *lsp.Position, prefix string) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithPrefix(position, resourceMetadataProps, "Resource metadata property", prefix)
}

func createResourcePropCompletionItemsWithDescriptions(
	position *lsp.Position,
	props []resourcePropInfo,
	detail string,
) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithPrefix(position, props, detail, "")
}

// createResourcePropCompletionItemsWithPrefix creates completion items for resource properties.
// prefix is the partial segment being typed (e.g., "sp") for filtering.
// pathPrefix is the path typed so far (e.g., "resources.myTable.") for building FilterText.
func createResourcePropCompletionItemsWithPrefix(
	position *lsp.Position,
	props []resourcePropInfo,
	detail string,
	prefix string,
) []*lsp.CompletionItem {
	prefixLen := len(prefix)
	prefixLower := strings.ToLower(prefix)
	insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
	fieldKind := lsp.CompletionItemKindField
	items := make([]*lsp.CompletionItem, 0, len(props))

	for _, prop := range props {
		// Filter by prefix if provided
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(prop.label), prefixLower) {
			continue
		}

		filterText := prop.label

		item := &lsp.CompletionItem{
			Label:      prop.label,
			Detail:     &detail,
			Kind:       &fieldKind,
			FilterText: &filterText,
			TextEdit: lsp.TextEdit{
				NewText: prop.label,
				Range:   insertRange,
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		}

		if prop.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: prop.description,
			}
		}

		items = append(items, item)
	}

	return items
}

// createMetadataKeyCompletionItems creates completion items for metadata keys.
// Keys containing special characters use bracket notation with contextual quotes:
// - Inside double-quoted strings: ['key.with.dots'] (single quotes)
// - Otherwise: ["key.with.dots"] (double quotes)
// When isBracketTrigger is true (user typed `[`), all keys are wrapped as `"key"]`.
func createMetadataKeyCompletionItems(
	position *lsp.Position,
	keys []string,
	detail string,
	quoteType docmodel.QuoteType,
	isBracketTrigger bool,
) []*lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField
	items := make([]*lsp.CompletionItem, 0, len(keys))

	for _, key := range keys {
		var textEdit lsp.TextEdit
		if isBracketTrigger {
			// User typed "[", insert "key"] to complete the bracket access.
			insertText := formatMapKeyForBracketInsertion(key, quoteType)
			textEdit = lsp.TextEdit{
				NewText: insertText,
				Range:   getItemInsertRange(position),
			}
		} else if needsBracketNotation(key) {
			bracketNotation := formatBracketNotation(key, quoteType)
			textEdit = lsp.TextEdit{
				NewText: bracketNotation,
				Range:   getBracketNotationInsertRange(position),
			}
		} else {
			textEdit = lsp.TextEdit{
				NewText: key,
				Range:   getItemInsertRange(position),
			}
		}

		items = append(items, &lsp.CompletionItem{
			Label:    key,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: textEdit,
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		})
	}

	return items
}

func getMetadataKeysAndDetail(
	metadata *schema.Metadata,
	segment string,
) ([]string, string) {
	switch segment {
	case "annotations":
		if metadata.Annotations == nil {
			return nil, ""
		}
		return mapKeys(metadata.Annotations.Values), "Annotation key"
	case "labels":
		if metadata.Labels == nil {
			return nil, ""
		}
		return mapKeys(metadata.Labels.Values), "Label key"
	case "custom":
		if metadata.Custom == nil {
			return nil, ""
		}
		return getMappingNodeFieldKeys(metadata.Custom), "Custom metadata field"
	default:
		return nil, ""
	}
}

// resourceDefAttributesSchemaCompletionItemsForSubstitution creates completion items for
// substitution references (inside ${...}). These completions use just the field name
// without colons or quotes since they're property access paths.
// pathPrefix is the path typed so far (e.g., "resources.myTable.spec.") for building FilterText.
// typedPrefix is the partial segment being typed (e.g., "ar") for filtering and range calculation.
func resourceDefAttributesSchemaCompletionItemsForSubstitution(
	attributes map[string]*provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	attrDetail string,
	typedPrefix string,
) []*lsp.CompletionItem {
	completionItems := []*lsp.CompletionItem{}
	prefixLen := len(typedPrefix)
	prefixLower := strings.ToLower(typedPrefix)

	for attrName, attrSchema := range attributes {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(attrName), prefixLower) {
			continue
		}

		fieldKind := lsp.CompletionItemKindField
		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)

		edit := lsp.TextEdit{
			NewText: attrName,
			Range:   insertRange,
		}

		detail := attrDetail
		if attrSchema != nil && attrSchema.Type != "" {
			detail = fmt.Sprintf("%s (%s)", attrDetail, attrSchema.Type)
		}

		filterText := attrName

		item := &lsp.CompletionItem{
			Label:      attrName,
			Detail:     &detail,
			Kind:       &fieldKind,
			TextEdit:   edit,
			FilterText: &filterText,
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		}

		if attrSchema != nil {
			if attrSchema.FormattedDescription != "" {
				item.Documentation = lsp.MarkupContent{
					Kind:  lsp.MarkupKindMarkdown,
					Value: attrSchema.FormattedDescription,
				}
			} else if attrSchema.Description != "" {
				item.Documentation = attrSchema.Description
			}
		}

		completionItems = append(completionItems, item)
	}

	return completionItems
}

// resourceDefAttributesSchemaCompletionItemsWithPrefix creates completion items for resource
// spec fields based on the provider schema. Note: Only used for YAML - JSONC is disabled.
func resourceDefAttributesSchemaCompletionItemsWithPrefix(
	attributes map[string]*provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	attrDetail string,
	typedPrefix string,
) []*lsp.CompletionItem {
	completionItems := []*lsp.CompletionItem{}
	prefixLen := len(typedPrefix)
	prefixLower := strings.ToLower(typedPrefix)

	for attrName, attrSchema := range attributes {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(attrName), prefixLower) {
			continue
		}

		fieldKind := lsp.CompletionItemKindField
		insertRange := getItemInsertRangeWithPrefix(position, prefixLen)
		insertText := attrName + ": "
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		detail := attrDetail
		if attrSchema != nil && attrSchema.Type != "" {
			detail = fmt.Sprintf("%s (%s)", attrDetail, attrSchema.Type)
		}

		item := &lsp.CompletionItem{
			Label:      attrName,
			Detail:     &detail,
			Kind:       &fieldKind,
			TextEdit:   edit,
			FilterText: &attrName,
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		}

		if attrSchema != nil {
			if attrSchema.FormattedDescription != "" {
				item.Documentation = lsp.MarkupContent{
					Kind:  lsp.MarkupKindMarkdown,
					Value: attrSchema.FormattedDescription,
				}
			} else if attrSchema.Description != "" {
				item.Documentation = attrSchema.Description
			}
		}

		completionItems = append(completionItems, item)
	}

	return completionItems
}
