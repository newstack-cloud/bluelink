package languageservices

import (
	"fmt"
	"strconv"
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

	// Extract partial segment for filtering (text after last . or [)
	partialPrefix := extractSubstitutionPartialSegment(textBefore)

	// Always use text-based parsing for navigation.
	// SchemaElement may contain stale data after reaching terminal properties.
	resourceProp := parseResourcePropertyFromTextForNavigation(textBefore, blueprint)

	if resourceProp == nil {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	// Check if in spec path
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "spec" {
		return s.getResourceSpecPropCompletionItemsWithPrefix(ctx, position, blueprint, resourceProp, partialPrefix)
	}

	// Check if in metadata path
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "metadata" {
		return s.getResourceMetadataPropCompletionItemsForPath(position, blueprint, resourceProp, cursorCtx, partialPrefix)
	}

	// If path is empty, we're at the resource level - return top-level props (spec, metadata, state)
	// This handles both trailing dot (e.g., "${resources.myTable.") and partial prefix (e.g., "${resources.myTable.sp")
	return getResourceTopLevelPropCompletionItemsWithPrefix(position, partialPrefix), nil
}

// extractSubstitutionPartialSegment extracts the partial segment being typed from substitution text.
// Returns the text after the last '.' or '[', or empty if text ends with '.' or '['.
func extractSubstitutionPartialSegment(textBefore string) string {
	// Find the substitution start
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return ""
	}

	subText := textBefore[subStart+2:]

	// If text ends with . or [, there's no partial segment
	if strings.HasSuffix(subText, ".") || strings.HasSuffix(subText, "[") {
		return ""
	}

	// Find the last . or [ position
	lastDot := strings.LastIndex(subText, ".")
	lastBracket := strings.LastIndex(subText, "[")

	// Use whichever is later
	lastSep := max(lastBracket, lastDot)

	if lastSep >= 0 {
		return subText[lastSep+1:]
	}

	return ""
}

// parseResourcePropertyFromTextForNavigation parses resource property excluding partial segment.
// This is used for schema navigation where partial segments would fail.
func parseResourcePropertyFromTextForNavigation(
	textBefore string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	// If text ends with . or [, parse normally
	if strings.HasSuffix(textBefore, ".") || strings.HasSuffix(textBefore, "[") {
		return parseResourcePropertyFromText(textBefore, blueprint)
	}

	// Otherwise, find the last separator and parse up to that point
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return nil
	}

	subText := textBefore[subStart+2:]

	// Find last separator
	lastDot := strings.LastIndex(subText, ".")
	lastBracket := strings.LastIndex(subText, "[")

	lastSep := max(lastBracket, lastDot)

	if lastSep < 0 {
		return nil
	}

	// Parse text up to and including the separator
	truncatedText := textBefore[:subStart+2+lastSep+1]
	return parseResourcePropertyFromText(truncatedText, blueprint)
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

	// Parse path handling bracket notation for array indices
	// Only bracket notation [0] creates array indices, not dot-notation numerics
	segments := parseSubstitutionPathSegments(pathText)
	if len(segments) == 0 {
		return nil
	}

	resourceName := segments[0].value
	if resourceName == "" {
		return nil
	}

	if blueprint.Resources.Values[resourceName] == nil {
		return nil
	}

	var pathItems []*substitutions.SubstitutionPathItem
	for i := 1; i < len(segments); i += 1 {
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
		pathItems = append(pathItems, pathItem)
	}

	return &substitutions.SubstitutionResourceProperty{
		ResourceName: resourceName,
		Path:         pathItems,
	}
}

// substitutionPathSegment represents a parsed segment of a substitution path.
type substitutionPathSegment struct {
	value      string
	isIndex    bool
	indexValue int64
}

// parseSubstitutionPathSegments parses a substitution path into segments.
// Only bracket notation [0] creates array indices.
func parseSubstitutionPathSegments(path string) []substitutionPathSegment {
	var segments []substitutionPathSegment
	i := 0
	n := len(path)

	for i < n {
		// Skip dots
		if path[i] == '.' {
			i += 1
			continue
		}

		// Check for bracket notation
		if path[i] == '[' {
			seg := parseSubstitutionBracketSegment(path, &i)
			segments = append(segments, seg)
			continue
		}

		// Regular field name - read until dot or bracket
		start := i
		for i < n && path[i] != '.' && path[i] != '[' {
			i += 1
		}
		if start < i {
			fieldName := path[start:i]
			segments = append(segments, substitutionPathSegment{value: fieldName, isIndex: false})
		}
	}

	return segments
}

// parseSubstitutionBracketSegment parses a bracket notation segment.
func parseSubstitutionBracketSegment(path string, i *int) substitutionPathSegment {
	n := len(path)
	*i += 1 // Skip opening '['

	if *i >= n {
		return substitutionPathSegment{}
	}

	// Check for quoted key
	if path[*i] == '"' || path[*i] == '\'' {
		quote := path[*i]
		*i += 1 // Skip opening quote
		start := *i
		for *i < n && path[*i] != quote {
			*i += 1
		}
		value := path[start:*i]
		if *i < n {
			*i += 1 // Skip closing quote
		}
		if *i < n && path[*i] == ']' {
			*i += 1 // Skip closing ']'
		}
		return substitutionPathSegment{value: value, isIndex: false}
	}

	// Numeric index
	start := *i
	for *i < n && path[*i] != ']' {
		*i += 1
	}
	indexStr := path[start:*i]
	if *i < n {
		*i += 1 // Skip closing ']'
	}

	if idx, err := strconv.ParseInt(indexStr, 10, 64); err == nil {
		return substitutionPathSegment{value: indexStr, isIndex: true, indexValue: idx}
	}
	return substitutionPathSegment{value: indexStr, isIndex: false}
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

// getStringSubValuePropertyCompletionItems returns completion items for value property paths.
// Navigates the Value.Value MappingNode tree to provide field keys or array indices.
func (s *CompletionService) getStringSubValuePropertyCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	cursorCtx *docmodel.CursorContext,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Values == nil || len(blueprint.Values.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	textBefore := ""
	if cursorCtx != nil {
		textBefore = cursorCtx.TextBefore
	}

	valueName, pathSegments := parseValuePropertyPath(textBefore)
	if valueName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	valueDef := blueprint.Values.Values[valueName]
	if valueDef == nil || valueDef.Value == nil {
		return []*lsp.CompletionItem{}, nil
	}

	current := valueDef.Value
	for _, seg := range pathSegments {
		if current == nil {
			return []*lsp.CompletionItem{}, nil
		}
		current = navigateMappingNode(current, []string{seg})
	}

	return mappingNodeCompletionItems(position, current, cursorCtx)
}

// parseValuePropertyPath extracts the value name and remaining path segments from substitution text.
// For "${values.myValue.field1.field2.", returns ("myValue", ["field1", "field2"]).
func parseValuePropertyPath(textBefore string) (string, []string) {
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return "", nil
	}

	subText := textBefore[subStart+2:]

	// Strip "values." prefix
	if !strings.HasPrefix(subText, "values.") {
		return "", nil
	}
	remaining := subText[len("values."):]

	// Split by dots - first segment is value name, rest are path segments
	parts := strings.Split(remaining, ".")
	if len(parts) == 0 {
		return "", nil
	}

	valueName := parts[0]
	if valueName == "" {
		return "", nil
	}

	// Remaining parts (excluding empty trailing element from trailing dot and partial segment)
	var pathSegments []string
	for i := 1; i < len(parts); i += 1 {
		if parts[i] != "" {
			// If this is the last part and text doesn't end with ".", it's a partial segment - skip it
			if i == len(parts)-1 && !strings.HasSuffix(subText, ".") {
				continue
			}
			pathSegments = append(pathSegments, parts[i])
		}
	}

	return valueName, pathSegments
}

// getStringSubChildPropertyCompletionItems returns completion items for child blueprint export names.
// Resolves the child blueprint from the include definition and suggests its exports.
func (s *CompletionService) getStringSubChildPropertyCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	docURI string,
) ([]*lsp.CompletionItem, error) {
	childInfo := s.resolveChildInfo(blueprint, completionCtx.ChildName, docURI)
	if childInfo == nil {
		return []*lsp.CompletionItem{}, nil
	}

	fieldKind := lsp.CompletionItemKindField
	items := make([]*lsp.CompletionItem, 0, len(childInfo.Exports))
	for _, export := range childInfo.Exports {
		detail := "Child export"
		if export.Type != "" {
			detail = string(export.Type)
		}

		insertRange := getItemInsertRange(position)
		item := &lsp.CompletionItem{
			Label:  export.Name,
			Detail: &detail,
			Kind:   &fieldKind,
			TextEdit: lsp.TextEdit{
				NewText: export.Name,
				Range:   insertRange,
			},
		}
		if export.Description != "" {
			item.Documentation = export.Description
		}
		items = append(items, item)
	}

	return items, nil
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
		defOutput, err := s.functionRegistry.GetDefinition(
			ctx.Context,
			functionName,
			&provider.FunctionGetDefinitionInput{},
		)
		if err != nil {
			continue
		}

		if defOutput.Definition.Internal {
			continue
		}

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

// getResourceSpecPropCompletionItemsWithPrefix returns completion items with prefix filtering.
// prefix is the partial segment being typed (e.g., "ar").
// pathPrefix is the path typed so far (e.g., "resources.myTable.spec.") for building FilterText.
func (s *CompletionService) getResourceSpecPropCompletionItemsWithPrefix(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	resourceProp *substitutions.SubstitutionResourceProperty,
	prefix string,
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

	currentSchema, isAtArray := navigateSchemaForSubstitution(specDefOutput.SpecDefinition.Schema, resourceProp)

	// If at an array, suggest array indices based on the resource spec
	if isAtArray {
		return getSubstitutionArrayIndexCompletionItems(position, resource, resourceProp), nil
	}

	if currentSchema == nil || currentSchema.Attributes == nil {
		return []*lsp.CompletionItem{}, nil
	}

	return resourceDefAttributesSchemaCompletionItemsForSubstitution(
		currentSchema.Attributes, position, "Resource spec property", prefix,
	), nil
}

// navigateSchemaForSubstitution navigates to the correct schema level for substitution completions.
// Returns (schema, isAtArray) where isAtArray is true if we ended at an array and should show index suggestions.
func navigateSchemaForSubstitution(
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

	for i := pathStart; i < len(resourceProp.Path); i += 1 {
		pathItem := resourceProp.Path[i]

		// Handle array index navigation
		if pathItem.ArrayIndex != nil {
			if currentSchema.Type == provider.ResourceDefinitionsSchemaTypeArray && currentSchema.Items != nil {
				currentSchema = currentSchema.Items
				continue
			}
			return nil, false
		}

		if pathItem.FieldName == "" {
			break
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
		if currentSchema.Items != nil {
			return currentSchema.Items, true
		}
		return currentSchema, true
	}

	return currentSchema, false
}

// getSubstitutionArrayIndexCompletionItems returns array index suggestions for substitutions.
func getSubstitutionArrayIndexCompletionItems(
	position *lsp.Position,
	resource *schema.Resource,
	resourceProp *substitutions.SubstitutionResourceProperty,
) []*lsp.CompletionItem {
	items := []*lsp.CompletionItem{}

	// Try to determine array length from the resource spec
	arrayLen := getArrayLengthFromResourceProp(resource, resourceProp)

	if arrayLen == 0 {
		// If we can't determine the length, offer index 0 as a starting point
		arrayLen = 1
	}

	fieldKind := lsp.CompletionItemKindValue
	for i := 0; i < arrayLen; i += 1 {
		indexStr := fmt.Sprintf("%d", i)
		detail := fmt.Sprintf("Array index %d", i)
		items = append(items, &lsp.CompletionItem{
			Label:  indexStr,
			Detail: &detail,
			Kind:   &fieldKind,
			TextEdit: lsp.TextEdit{
				NewText: indexStr,
				Range:   getItemInsertRange(position),
			},
			Data: map[string]any{"completionType": "arrayIndex"},
		})
	}

	return items
}

// getArrayLengthFromResourceProp tries to determine array length from resource spec using path.
func getArrayLengthFromResourceProp(
	resource *schema.Resource,
	resourceProp *substitutions.SubstitutionResourceProperty,
) int {
	if resource == nil || resource.Spec == nil || resourceProp == nil {
		return 0
	}

	// Navigate through the spec to find the array
	current := resource.Spec
	for _, pathItem := range resourceProp.Path {
		if pathItem.FieldName == "spec" {
			continue
		}
		if pathItem.FieldName == "" {
			continue
		}

		if current == nil || current.Fields == nil {
			return 0
		}

		next, exists := current.Fields[pathItem.FieldName]
		if !exists {
			return 0
		}
		current = next
	}

	// At this point, current should be the array node
	if current != nil && current.Items != nil {
		return len(current.Items)
	}

	return 0
}

func getDataSource(blueprint *schema.Blueprint, dataSourceName string) *schema.DataSource {
	if blueprint == nil || blueprint.DataSources == nil {
		return nil
	}
	return blueprint.DataSources.Values[dataSourceName]
}
