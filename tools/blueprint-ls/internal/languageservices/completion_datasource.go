package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// getDataSourceFilterFieldCompletionItemsFromContext adapts filter field completion to use CursorContext.
func (s *CompletionService) getDataSourceFilterFieldCompletionItemsFromContext(
	ctx *common.LSPContext,
	cursorCtx *docmodel.CursorContext,
	blueprint *schema.Blueprint,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if cursorCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName, ok := cursorCtx.GetDataSourceName()
	if !ok || dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	// Try to find the data source type from the current blueprint or fall back to last valid schema.
	dataSourceType := getDataSourceTypeForCompletion(blueprint, cursorCtx.DocumentCtx, dataSourceName)
	if dataSourceType == "" {
		return []*lsp.CompletionItem{}, nil
	}

	filterFieldsOutput, err := s.dataSourceRegistry.GetFilterFields(
		ctx.Context,
		dataSourceType,
		&provider.DataSourceGetFilterFieldsInput{},
	)
	if err != nil {
		return nil, err
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)
	completionItems := []*lsp.CompletionItem{}
	filterFieldDetail := "Data source filter field"
	enumKind := lsp.CompletionItemKindEnum

	for filterField := range filterFieldsOutput.FilterFields {
		if !filterByPrefix(filterField, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(filterField, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      filterField,
			Detail:     &filterFieldDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &filterField,
			Data:       map[string]any{"completionType": "dataSourceFilterField"},
		}
		completionItems = append(completionItems, item)
	}

	return completionItems, nil
}

// getDataSourceFilterOperatorCompletionItemsFromContext adapts filter operator completion to use CursorContext.
func (s *CompletionService) getDataSourceFilterOperatorCompletionItemsFromContext(
	position *lsp.Position,
	cursorCtx *docmodel.CursorContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	filterOperatorDetail := "Data source filter operator"
	enumKind := lsp.CompletionItemKindEnum

	// Get operator element position from schema node (preferred) or fallback to current position
	var operatorElementPosition *source.Position
	if cursorCtx != nil && cursorCtx.SchemaNode != nil && cursorCtx.SchemaNode.Range != nil {
		operatorElementPosition = cursorCtx.SchemaNode.Range.Start
	} else {
		operatorElementPosition = &source.Position{
			Line:   int(position.Line + 1),
			Column: int(position.Character + 1),
		}
	}

	// Determine if cursor is right after "operator:" field declaration
	isPrecededByOperator := cursorCtx != nil && cursorCtx.IsPrecededByOperatorField()

	// Check if user has already typed a leading quote (only relevant for JSONC)
	hasLeadingQuote := false
	if format == docmodel.FormatJSONC && cursorCtx != nil {
		typedPrefix := cursorCtx.GetTypedPrefix()
		_, hasLeadingQuote = stripLeadingQuote(typedPrefix)
	}

	filterOpItems := []*lsp.CompletionItem{}
	for _, filterOperator := range schema.DataSourceFilterOperators {
		filterOperatorStr := formatFilterOperator(string(filterOperator), format, hasLeadingQuote)
		edit := lsp.TextEdit{
			NewText: filterOperatorStr,
			Range: getOperatorInsertRange(
				position,
				filterOperatorStr,
				isPrecededByOperator,
				operatorElementPosition,
			),
		}
		filterOpItems = append(filterOpItems, &lsp.CompletionItem{
			Label:    fmt.Sprintf("\"%s\"", string(filterOperator)),
			Detail:   &filterOperatorDetail,
			Kind:     &enumKind,
			TextEdit: edit,
			Data:     map[string]any{"completionType": "dataSourceFilterOperator"},
		})
	}

	return filterOpItems, nil
}

func formatFilterOperator(op string, format docmodel.DocumentFormat, hasLeadingQuote bool) string {
	if hasLeadingQuote {
		// JSONC: User already typed opening quote, complete with value + closing quote
		return op + `"`
	}
	if format == docmodel.FormatJSONC {
		// JSONC without leading quote: add space + quoted value
		return fmt.Sprintf(` "%s"`, op)
	}
	// YAML: quoted value
	return fmt.Sprintf(`"%s"`, op)
}

// getDataSourceExportAliasForCompletionItems provides completion suggestions for the aliasFor field
// in data source exports. The aliasFor value references a field name from the data source's spec definition.
func (s *CompletionService) getDataSourceExportAliasForCompletionItems(
	ctx *common.LSPContext,
	cursorCtx *docmodel.CursorContext,
	blueprint *schema.Blueprint,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if cursorCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName, ok := cursorCtx.GetDataSourceName()
	if !ok || dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceType := getDataSourceTypeForCompletion(blueprint, cursorCtx.DocumentCtx, dataSourceName)
	if dataSourceType == "" {
		return []*lsp.CompletionItem{}, nil
	}

	specDef, err := s.dataSourceRegistry.GetSpecDefinition(
		ctx.Context,
		dataSourceType,
		&provider.DataSourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return nil, err
	}

	if specDef == nil || specDef.SpecDefinition == nil || specDef.SpecDefinition.Fields == nil {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)
	completionItems := []*lsp.CompletionItem{}
	fieldDetail := "Data source field"
	enumKind := lsp.CompletionItemKindField

	for fieldName, fieldSchema := range specDef.SpecDefinition.Fields {
		if !filterByPrefix(fieldName, prefixInfo) {
			continue
		}

		detail := fieldDetail
		if fieldSchema != nil && fieldSchema.Type != "" {
			detail = fmt.Sprintf("%s (%s)", fieldDetail, fieldSchema.Type)
		}

		insertText := formatValueForInsert(fieldName, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		item := &lsp.CompletionItem{
			Label:      fieldName,
			Detail:     &detail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &fieldName,
			Data:       map[string]any{"completionType": "dataSourceExportAliasFor"},
		}

		if fieldSchema != nil {
			addDataSourceFieldDocumentation(item, fieldSchema)
		}

		completionItems = append(completionItems, item)
	}

	return completionItems, nil
}

// getDataSourceExportNameCompletionItems provides completion suggestions for export names
// (the key when creating a new export). Suggests field names from the data source's spec definition.
// YAML-only as JSONC key completions are disabled.
func (s *CompletionService) getDataSourceExportNameCompletionItems(
	ctx *common.LSPContext,
	cursorCtx *docmodel.CursorContext,
	blueprint *schema.Blueprint,
	_ *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	_ docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if cursorCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName, ok := cursorCtx.GetDataSourceName()
	if !ok || dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceType := getDataSourceTypeForCompletion(blueprint, cursorCtx.DocumentCtx, dataSourceName)
	if dataSourceType == "" {
		return []*lsp.CompletionItem{}, nil
	}

	specDef, err := s.dataSourceRegistry.GetSpecDefinition(
		ctx.Context,
		dataSourceType,
		&provider.DataSourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return nil, err
	}

	if specDef == nil || specDef.SpecDefinition == nil || specDef.SpecDefinition.Fields == nil {
		return []*lsp.CompletionItem{}, nil
	}

	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}
	prefixLower := strings.ToLower(typedPrefix)

	completionItems := []*lsp.CompletionItem{}
	fieldDetail := "Data source field (export name suggestion)"
	enumKind := lsp.CompletionItemKindField

	for fieldName, fieldSchema := range specDef.SpecDefinition.Fields {
		if len(typedPrefix) > 0 && !strings.HasPrefix(strings.ToLower(fieldName), prefixLower) {
			continue
		}

		detail := fieldDetail
		if fieldSchema != nil && fieldSchema.Type != "" {
			detail = fmt.Sprintf("%s (%s)", fieldDetail, fieldSchema.Type)
		}

		// For YAML key completions, append colon
		insertText := fieldName + ":"

		item := &lsp.CompletionItem{
			Label:      fieldName,
			Detail:     &detail,
			Kind:       &enumKind,
			InsertText: &insertText,
			FilterText: &fieldName,
			Data:       map[string]any{"completionType": "dataSourceExportName"},
		}

		if fieldSchema != nil {
			addDataSourceFieldDocumentation(item, fieldSchema)
		}

		completionItems = append(completionItems, item)
	}

	return completionItems, nil
}

func addDataSourceFieldDocumentation(item *lsp.CompletionItem, fieldSchema *provider.DataSourceSpecSchema) {
	if fieldSchema.FormattedDescription != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: fieldSchema.FormattedDescription,
		}
	} else if fieldSchema.Description != "" {
		item.Documentation = fieldSchema.Description
	}
}

// getDataSourceTypeForCompletion attempts to find the data source type for completions.
// It first checks the current blueprint, then falls back to the last valid schema.
func getDataSourceTypeForCompletion(
	blueprint *schema.Blueprint,
	docCtx *docmodel.DocumentContext,
	dataSourceName string,
) string {
	// Try current blueprint first
	if blueprint != nil && blueprint.DataSources != nil {
		if ds, ok := blueprint.DataSources.Values[dataSourceName]; ok && ds.Type != nil {
			return string(ds.Type.Value)
		}
	}

	// Fall back to last valid schema
	if docCtx != nil && docCtx.LastValidSchema != nil && docCtx.LastValidSchema.DataSources != nil {
		if ds, ok := docCtx.LastValidSchema.DataSources.Values[dataSourceName]; ok && ds.Type != nil {
			return string(ds.Type.Value)
		}
	}

	return ""
}
