package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// getStringSubPotentialResourcePropCompletionItems handles completion for potential standalone
// resource property patterns like ${myResource. or ${myResource[
func (s *CompletionService) getStringSubPotentialResourcePropCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	cursorCtx *docmodel.CursorContext,
) ([]*lsp.CompletionItem, error) {
	potentialName := completionCtx.PotentialResourceName
	if potentialName == "" {
		return s.getStringSubCompletionItems(ctx, position, blueprint)
	}

	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return s.getStringSubCompletionItems(ctx, position, blueprint)
	}

	// Validate that the potential name is an actual resource in the blueprint
	if _, exists := blueprint.Resources.Values[potentialName]; !exists {
		return s.getStringSubCompletionItems(ctx, position, blueprint)
	}

	// Delegate to resource property completion logic
	return s.getStringSubResourcePropCompletionItemsFromContext(ctx, position, blueprint, cursorCtx)
}

// getStringSubResourcePropCompletionItemsFromContext adapts resource property completion using CursorContext.
func (s *CompletionService) getStringSubResourcePropCompletionItemsFromContext(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	cursorCtx *docmodel.CursorContext,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	textBefore := ""
	if cursorCtx != nil {
		textBefore = cursorCtx.TextBefore
	}

	// Try to get resource property from schema element first
	var resourceProp *substitutions.SubstitutionResourceProperty
	if cursorCtx != nil && cursorCtx.SchemaElement != nil {
		resourceProp, _ = cursorCtx.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	}

	// Fallback: parse resource property from text
	if resourceProp == nil {
		resourceProp = parseResourcePropertyFromText(textBefore, blueprint)
	}

	if resourceProp == nil {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	// Check if directly after the resource name (need top-level props)
	if strings.HasSuffix(textBefore, fmt.Sprintf(".%s.", resourceProp.ResourceName)) ||
		strings.HasSuffix(textBefore, fmt.Sprintf("${%s.", resourceProp.ResourceName)) ||
		strings.HasSuffix(textBefore, fmt.Sprintf("${%s[", resourceProp.ResourceName)) {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	// Check if in spec path
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "spec" {
		return s.getResourceSpecPropCompletionItems(ctx, position, blueprint, resourceProp)
	}

	// Check if in metadata path
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "metadata" {
		return s.getResourceMetadataPropCompletionItemsForPath(position, blueprint, resourceProp, cursorCtx)
	}

	return []*lsp.CompletionItem{}, nil
}

// parseResourcePropertyFromText extracts resource property path from text.
func parseResourcePropertyFromText(
	textBefore string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return nil
	}

	subText := textBefore[subStart+2:] // Text after ${

	// Try parsing with resources. prefix first
	if strings.HasPrefix(subText, "resources.") {
		return parseResourcePropertyWithPrefix(subText, blueprint)
	}

	// Try parsing as standalone resource name
	return parseStandaloneResourceProperty(subText, blueprint)
}

func parseResourcePropertyWithPrefix(
	subText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	remaining := strings.TrimPrefix(subText, "resources.")
	return parseResourcePath(remaining, blueprint)
}

func parseStandaloneResourceProperty(
	subText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	return parseResourcePath(subText, blueprint)
}

func parseResourcePath(
	pathText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	if blueprint.Resources == nil {
		return nil
	}

	// Normalize: replace [ with . then split
	normalized := strings.ReplaceAll(pathText, "[", ".")
	parts := strings.Split(normalized, ".")
	if len(parts) == 0 {
		return nil
	}

	resourceName := strings.Trim(parts[0], "\"'[]")
	if resourceName == "" {
		return nil
	}

	if blueprint.Resources.Values[resourceName] == nil {
		return nil
	}

	var pathItems []*substitutions.SubstitutionPathItem
	for i := 1; i < len(parts); i++ {
		fieldName := strings.Trim(parts[i], "\"'[]")
		if fieldName == "" {
			continue
		}
		pathItems = append(pathItems, &substitutions.SubstitutionPathItem{
			FieldName: fieldName,
		})
	}

	return &substitutions.SubstitutionResourceProperty{
		ResourceName: resourceName,
		Path:         pathItems,
	}
}

// getStringSubDataSourcePropCompletionItemsFromContext adapts data source property completion.
func (s *CompletionService) getStringSubDataSourcePropCompletionItemsFromContext(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	detail := "Data source exported field"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName := completionCtx.DataSourceName
	if dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	dataSource := getDataSource(blueprint, dataSourceName)
	if dataSource == nil || dataSource.Exports == nil {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceItems := []*lsp.CompletionItem{}
	for exportName := range dataSource.Exports.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: exportName,
			Range:   insertRange,
		}
		dataSourceItems = append(dataSourceItems, &lsp.CompletionItem{
			Label:    exportName,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "dataSourceProperty"},
		})
	}

	return dataSourceItems, nil
}

func (s *CompletionService) getStringSubVariableCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	variableDetail := "Variable"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.Variables == nil || len(blueprint.Variables.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	varItems := []*lsp.CompletionItem{}
	for varName := range blueprint.Variables.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: varName,
			Range:   insertRange,
		}
		varItems = append(varItems, &lsp.CompletionItem{
			Label:    varName,
			Detail:   &variableDetail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "variable"},
		})
	}

	return varItems, nil
}

func (s *CompletionService) getStringSubResourceCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	resourceDetail := "Resource"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	resourceItems := []*lsp.CompletionItem{}
	for resourceName := range blueprint.Resources.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: resourceName,
			Range:   insertRange,
		}
		resourceItems = append(resourceItems, &lsp.CompletionItem{
			Label:    resourceName,
			Detail:   &resourceDetail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "resource"},
		})
	}

	return resourceItems, nil
}

func (s *CompletionService) getStringSubDataSourceCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	detail := "Data source"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceItems := []*lsp.CompletionItem{}
	for dataSourceName := range blueprint.DataSources.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: dataSourceName,
			Range:   insertRange,
		}
		dataSourceItems = append(dataSourceItems, &lsp.CompletionItem{
			Label:    dataSourceName,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "dataSource"},
		})
	}

	return dataSourceItems, nil
}

func (s *CompletionService) getStringSubValueCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	detail := "Value"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.Values == nil || len(blueprint.Values.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	valueItems := []*lsp.CompletionItem{}
	for valueName := range blueprint.Values.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: valueName,
			Range:   insertRange,
		}
		valueItems = append(valueItems, &lsp.CompletionItem{
			Label:    valueName,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "value"},
		})
	}

	return valueItems, nil
}

func (s *CompletionService) getStringSubChildCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	detail := "Child blueprint"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.Include == nil || len(blueprint.Include.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	includeItems := []*lsp.CompletionItem{}
	for includeName := range blueprint.Include.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: includeName,
			Range:   insertRange,
		}
		includeItems = append(includeItems, &lsp.CompletionItem{
			Label:    includeName,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "child"},
		})
	}

	return includeItems, nil
}

// getStringSubCompletionItems returns completion items for all string substitution types.
func (s *CompletionService) getStringSubCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	items := []*lsp.CompletionItem{}

	// Priority order: Resources, Variables, Functions, Data sources, Values, Child blueprints
	resourceItems := s.getResourceCompletionItems(position, blueprint, "1-")
	items = append(items, resourceItems...)

	variableItems := s.getVariableCompletionItems(position, blueprint, "2-")
	items = append(items, variableItems...)

	functionItems := s.getFunctionCompletionItems(ctx, position, "3-")
	items = append(items, functionItems...)

	dataSourceItems := s.getDataSourceCompletionItems(position, blueprint, "4-")
	items = append(items, dataSourceItems...)

	valueItems := s.getValueCompletionItems(position, blueprint, "5-")
	items = append(items, valueItems...)

	childItems := s.getChildCompletionItems(position, blueprint, "6-")
	items = append(items, childItems...)

	return items, nil
}

func (s *CompletionService) getResourceCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	resourceDetail := "Resource"
	standaloneResourceDetail := "Resource (standalone)"
	resourceKind := lsp.CompletionItemKindValue

	resourceItems := []*lsp.CompletionItem{}

	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return resourceItems
	}

	insertRange := getItemInsertRange(position)

	for resourceName := range blueprint.Resources.Values {
		// Add prefixed version: resources.{name}
		resourceText := fmt.Sprintf("resources.%s", resourceName)
		edit := lsp.TextEdit{
			NewText: resourceText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, resourceName)
		resourceItems = append(resourceItems, &lsp.CompletionItem{
			Label:    resourceText,
			Detail:   &resourceDetail,
			Kind:     &resourceKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "resource"},
		})

		// Add standalone version: {name}
		standaloneEdit := lsp.TextEdit{
			NewText: resourceName,
			Range:   insertRange,
		}
		standaloneSortText := fmt.Sprintf("%s1-%s", sortPrefix, resourceName)
		resourceItems = append(resourceItems, &lsp.CompletionItem{
			Label:    resourceName,
			Detail:   &standaloneResourceDetail,
			Kind:     &resourceKind,
			TextEdit: standaloneEdit,
			SortText: &standaloneSortText,
			Data:     map[string]any{"completionType": "resourceStandalone"},
		})
	}

	return resourceItems
}

func (s *CompletionService) getVariableCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	variableDetail := "Variable"
	variableKind := lsp.CompletionItemKindVariable

	variableItems := []*lsp.CompletionItem{}

	if blueprint.Variables == nil || len(blueprint.Variables.Values) == 0 {
		return variableItems
	}

	insertRange := getItemInsertRange(position)

	for variableName := range blueprint.Variables.Values {
		variableText := fmt.Sprintf("variables.%s", variableName)
		edit := lsp.TextEdit{
			NewText: variableText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, variableName)
		variableItems = append(variableItems, &lsp.CompletionItem{
			Label:    variableText,
			Detail:   &variableDetail,
			Kind:     &variableKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "variable"},
		})
	}

	return variableItems
}

func (s *CompletionService) getFunctionCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	sortPrefix string,
) []*lsp.CompletionItem {
	functionDetail := "Function"
	functionKind := lsp.CompletionItemKindFunction

	functionItems := []*lsp.CompletionItem{}

	functionNames, err := s.functionRegistry.ListFunctions(ctx.Context)
	if err != nil {
		return functionItems
	}

	insertRange := getItemInsertRange(position)

	for _, functionName := range functionNames {
		functionText := fmt.Sprintf("%s(", functionName)
		edit := lsp.TextEdit{
			NewText: functionText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, functionName)
		functionItems = append(functionItems, &lsp.CompletionItem{
			Label:    functionName,
			Detail:   &functionDetail,
			Kind:     &functionKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "function"},
		})
	}

	return functionItems
}

func (s *CompletionService) getDataSourceCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	dataSourceDetail := "Data source"
	dataSourceKind := lsp.CompletionItemKindField

	dataSourceItems := []*lsp.CompletionItem{}

	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return dataSourceItems
	}

	insertRange := getItemInsertRange(position)

	for dataSourceName := range blueprint.DataSources.Values {
		dataSourceText := fmt.Sprintf("datasources.%s", dataSourceName)
		edit := lsp.TextEdit{
			NewText: dataSourceText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, dataSourceName)
		dataSourceItems = append(dataSourceItems, &lsp.CompletionItem{
			Label:    dataSourceText,
			Detail:   &dataSourceDetail,
			Kind:     &dataSourceKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "dataSource"},
		})
	}

	return dataSourceItems
}

func (s *CompletionService) getValueCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	valueDetail := "Value"
	valueKind := lsp.CompletionItemKindValue

	valueItems := []*lsp.CompletionItem{}

	if blueprint.Values == nil || len(blueprint.Values.Values) == 0 {
		return valueItems
	}

	insertRange := getItemInsertRange(position)

	for valueName := range blueprint.Values.Values {
		valueText := fmt.Sprintf("values.%s", valueName)
		edit := lsp.TextEdit{
			NewText: valueText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, valueName)
		valueItems = append(valueItems, &lsp.CompletionItem{
			Label:    valueText,
			Detail:   &valueDetail,
			Kind:     &valueKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "value"},
		})
	}

	return valueItems
}

func (s *CompletionService) getChildCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	childDetail := "Child blueprint"
	childKind := lsp.CompletionItemKindField

	childItems := []*lsp.CompletionItem{}

	if blueprint.Include == nil || len(blueprint.Include.Values) == 0 {
		return childItems
	}

	insertRange := getItemInsertRange(position)

	for childName := range blueprint.Include.Values {
		childText := fmt.Sprintf("children.%s", childName)
		edit := lsp.TextEdit{
			NewText: childText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, childName)
		childItems = append(childItems, &lsp.CompletionItem{
			Label:    childText,
			Detail:   &childDetail,
			Kind:     &childKind,
			TextEdit: edit,
			SortText: &sortText,
			Data:     map[string]any{"completionType": "child"},
		})
	}

	return childItems
}

// getResourceSpecPropCompletionItems returns completion items for resource spec properties
// based on the provider's spec definition schema.
func (s *CompletionService) getResourceSpecPropCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	resourceProp *substitutions.SubstitutionResourceProperty,
) ([]*lsp.CompletionItem, error) {
	resource := getResource(blueprint, resourceProp.ResourceName)
	if resource == nil || resource.Type == nil {
		return noCompletionsHint(position, "Resource not found in blueprint"), nil
	}

	specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
		ctx.Context,
		resource.Type.Value,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return noCompletionsHint(position, fmt.Sprintf("Provider '%s' not available", resource.Type.Value)), nil
	}

	if specDefOutput.SpecDefinition == nil || specDefOutput.SpecDefinition.Schema == nil {
		return noCompletionsHint(position, fmt.Sprintf("No spec schema for '%s'", resource.Type.Value)), nil
	}

	// Navigate to the correct schema depth based on path
	// Path is like: [{spec} {fieldName} ...]
	currentSchema := specDefOutput.SpecDefinition.Schema

	// Skip the first "spec" element if present
	pathStart := 0
	if len(resourceProp.Path) > 0 && resourceProp.Path[0].FieldName == "spec" {
		pathStart = 1
	}

	for i := pathStart; i < len(resourceProp.Path); i++ {
		pathItem := resourceProp.Path[i]
		if pathItem.FieldName != "" && currentSchema.Attributes != nil {
			attrSchema, exists := currentSchema.Attributes[pathItem.FieldName]
			if !exists {
				return []*lsp.CompletionItem{}, nil
			}
			currentSchema = attrSchema
		}
	}

	if currentSchema == nil || currentSchema.Attributes == nil {
		return []*lsp.CompletionItem{}, nil
	}

	return resourceDefAttributesSchemaCompletionItems(currentSchema.Attributes, position, "Resource spec property"), nil
}

func getDataSource(blueprint *schema.Blueprint, dataSourceName string) *schema.DataSource {
	if blueprint == nil || blueprint.DataSources == nil {
		return nil
	}
	return blueprint.DataSources.Values[dataSourceName]
}
