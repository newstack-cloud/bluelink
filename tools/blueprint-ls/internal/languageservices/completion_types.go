package languageservices

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// Known blueprint specification versions for completion suggestions.
var knownBlueprintVersions = []string{
	validation.Version2025_11_02,
}

func (s *CompletionService) getResourceTypeCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	resourceTypes, err := s.resourceRegistry.ListResourceTypes(ctx.Context)
	if err != nil {
		return nil, err
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)
	completionItems := []*lsp.CompletionItem{}
	resourceTypeDetail := "Resource type"

	for _, resourceType := range resourceTypes {
		if !filterByPrefix(resourceType, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(resourceType, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		enumKind := lsp.CompletionItemKindEnum
		completionItems = append(completionItems, &lsp.CompletionItem{
			Label:      resourceType,
			Detail:     &resourceTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &resourceType,
			Data:       map[string]any{"completionType": "resourceType"},
		})
	}

	return completionItems, nil
}

func (s *CompletionService) getDataSourceTypeCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	dataSourceTypes, err := s.dataSourceRegistry.ListDataSourceTypes(ctx.Context)
	if err != nil {
		return nil, err
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)
	completionItems := []*lsp.CompletionItem{}
	dataSourceTypeDetail := "Data source type"

	for _, dataSourceType := range dataSourceTypes {
		if !filterByPrefix(dataSourceType, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(dataSourceType, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		enumKind := lsp.CompletionItemKindEnum
		completionItems = append(completionItems, &lsp.CompletionItem{
			Label:      dataSourceType,
			Detail:     &dataSourceTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &dataSourceType,
			Data:       map[string]any{"completionType": "dataSourceType"},
		})
	}

	return completionItems, nil
}

func (s *CompletionService) getVariableTypeCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractCompletionPrefix(completionCtx, format)
	variableTypeDetail := "Variable type"
	enumKind := lsp.CompletionItemKindEnum
	typeItems := []*lsp.CompletionItem{}

	// Add core variable types
	for _, coreType := range schema.CoreVariableTypes {
		coreTypeStr := string(coreType)
		if !filterByPrefix(coreTypeStr, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(coreTypeStr, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      coreTypeStr,
			Detail:     &variableTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &coreTypeStr,
			Data:       map[string]any{"completionType": "variableType"},
		})
	}

	// Add custom variable types from registry
	customTypes, err := s.customVarTypeRegistry.ListCustomVariableTypes(ctx.Context)
	if err != nil {
		s.logger.Error("Failed to list custom variable types, returning core types only", zap.Error(err))
		return typeItems, nil
	}

	for _, customType := range customTypes {
		if !filterByPrefix(customType, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(customType, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      customType,
			Detail:     &variableTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &customType,
			Data:       map[string]any{"completionType": "variableType"},
		})
	}

	return typeItems, nil
}

func (s *CompletionService) getValueTypeCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractCompletionPrefix(completionCtx, format)
	valueTypeDetail := "Value type"
	enumKind := lsp.CompletionItemKindEnum
	typeItems := []*lsp.CompletionItem{}

	for _, valueType := range schema.ValueTypes {
		valueTypeStr := string(valueType)
		if !filterByPrefix(valueTypeStr, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(valueTypeStr, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      valueTypeStr,
			Detail:     &valueTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &valueTypeStr,
			Data:       map[string]any{"completionType": "valueType"},
		})
	}

	return typeItems, nil
}

func (s *CompletionService) getDataSourceFieldTypeCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractCompletionPrefix(completionCtx, format)
	fieldTypeDetail := "Data source field type"
	enumKind := lsp.CompletionItemKindEnum
	typeItems := []*lsp.CompletionItem{}

	for _, fieldType := range schema.DataSourceFieldTypes {
		fieldTypeStr := string(fieldType)
		if !filterByPrefix(fieldTypeStr, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(fieldTypeStr, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      fieldTypeStr,
			Detail:     &fieldTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &fieldTypeStr,
			Data:       map[string]any{"completionType": "dataSourceFieldType"},
		})
	}

	return typeItems, nil
}

func (s *CompletionService) getExportTypeCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractCompletionPrefix(completionCtx, format)
	exportTypeDetail := "Export type"
	enumKind := lsp.CompletionItemKindEnum
	typeItems := []*lsp.CompletionItem{}

	for _, exportType := range schema.ExportTypes {
		exportTypeStr := string(exportType)
		if !filterByPrefix(exportTypeStr, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(exportTypeStr, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      exportTypeStr,
			Detail:     &exportTypeDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &exportTypeStr,
			Data:       map[string]any{"completionType": "exportType"},
		})
	}

	return typeItems, nil
}

func (s *CompletionService) getVersionCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	prefixInfo := extractCompletionPrefix(completionCtx, format)

	versionDetail := "Blueprint spec version"
	enumKind := lsp.CompletionItemKindEnumMember
	items := make([]*lsp.CompletionItem, 0, len(knownBlueprintVersions))

	for _, version := range knownBlueprintVersions {
		if !filterByPrefix(version, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(version, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		items = append(items, &lsp.CompletionItem{
			Label:      version,
			Detail:     &versionDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &version,
			Data:       map[string]any{"completionType": "version"},
		})
	}

	return items, nil
}

func (s *CompletionService) getTransformCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	transformers, err := s.resourceRegistry.ListTransformers(ctx.Context)
	if err != nil {
		return []*lsp.CompletionItem{}, nil // Silently fail - registry not available
	}

	if len(transformers) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)

	transformDetail := "Blueprint transformer"
	enumKind := lsp.CompletionItemKindEnumMember
	items := make([]*lsp.CompletionItem, 0, len(transformers))

	for _, transformName := range transformers {
		if !filterByPrefix(transformName, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(transformName, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
		edit := lsp.TextEdit{
			NewText: insertText,
			Range:   insertRange,
		}

		items = append(items, &lsp.CompletionItem{
			Label:      transformName,
			Detail:     &transformDetail,
			Kind:       &enumKind,
			TextEdit:   edit,
			FilterText: &transformName,
			Data:       map[string]any{"completionType": "transform"},
		})
	}

	return items, nil
}

func (s *CompletionService) getCustomVariableTypeOptionsCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Variables == nil {
		return []*lsp.CompletionItem{}, nil
	}

	variableName, ok := completionCtx.NodeCtx.ASTPath.GetVariableName()
	if !ok {
		return []*lsp.CompletionItem{}, nil
	}

	variable, exists := blueprint.Variables.Values[variableName]
	if !exists || variable == nil || variable.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	variableType := string(variable.Type.Value)

	// Check if this is a core type (no options for core types)
	for _, coreType := range schema.CoreVariableTypes {
		if string(coreType) == variableType {
			return []*lsp.CompletionItem{}, nil
		}
	}

	optionsOutput, err := s.customVarTypeRegistry.GetOptions(
		ctx.Context,
		variableType,
		&provider.CustomVariableTypeOptionsInput{},
	)
	if err != nil || optionsOutput == nil || len(optionsOutput.Options) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	prefixInfo := extractCompletionPrefix(completionCtx, format)

	detail := fmt.Sprintf("Option (%s)", variableType)
	enumKind := lsp.CompletionItemKindEnumMember
	items := make([]*lsp.CompletionItem, 0, len(optionsOutput.Options))

	for label, option := range optionsOutput.Options {
		if option == nil || option.Value == nil {
			continue
		}

		valueStr := option.Value.ToString()
		if !filterByPrefix(valueStr, prefixInfo) {
			continue
		}

		insertText := formatValueForInsert(valueStr, format, prefixInfo.HasLeadingQuote, prefixInfo.HasLeadingSpace)
		insertRange := getItemInsertRangeWithPrefix(position, prefixInfo.PrefixLen)
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
			Data: map[string]any{
				"completionType": "customVariableTypeOption",
			},
		}

		// Add documentation from option if available
		addCustomVariableTypeOptionDocumentation(item, label, valueStr, option)
		items = append(items, item)
	}

	return items, nil
}

func addCustomVariableTypeOptionDocumentation(
	item *lsp.CompletionItem,
	label string,
	valueStr string,
	option *provider.CustomVariableTypeOption,
) {
	if option.MarkdownDescription != "" {
		docValue := option.MarkdownDescription
		if label != valueStr && label != "" {
			docValue = fmt.Sprintf("## %s\n\n%s", label, option.MarkdownDescription)
		}
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: docValue,
		}
	} else if option.Description != "" {
		if label != valueStr && label != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: fmt.Sprintf("## %s\n\n%s", label, option.Description),
			}
		} else {
			item.Documentation = option.Description
		}
	} else if label != valueStr {
		item.Documentation = label
	}
}

// ResolveCompletionItem resolves extra information such as detailed
// descriptions for a completion item.
func (s *CompletionService) ResolveCompletionItem(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
	completionType string,
) (*lsp.CompletionItem, error) {
	switch completionType {
	case "resourceType":
		return s.resolveResourceTypeCompletionItem(ctx, item)
	case "dataSourceType":
		return s.resolveDataSourceTypeCompletionItem(ctx, item)
	case "variableType":
		return s.resolveVariableTypeCompletionItem(ctx, item)
	case "function":
		return s.resolveFunctionCompletionItem(ctx, item)
	default:
		return item, nil
	}
}

func (s *CompletionService) resolveResourceTypeCompletionItem(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
) (*lsp.CompletionItem, error) {
	resourceType := item.Label
	descriptionOutput, err := s.resourceRegistry.GetTypeDescription(
		ctx.Context,
		resourceType,
		&provider.ResourceGetTypeDescriptionInput{},
	)
	if err != nil {
		return nil, err
	}

	if descriptionOutput.MarkdownDescription != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: descriptionOutput.MarkdownDescription,
		}
	} else if descriptionOutput.PlainTextDescription != "" {
		item.Documentation = descriptionOutput.PlainTextDescription
	}

	return item, nil
}

func (s *CompletionService) resolveDataSourceTypeCompletionItem(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
) (*lsp.CompletionItem, error) {
	dataSourceType := item.Label
	descriptionOutput, err := s.dataSourceRegistry.GetTypeDescription(
		ctx.Context,
		dataSourceType,
		&provider.DataSourceGetTypeDescriptionInput{},
	)
	if err != nil {
		return nil, err
	}

	if descriptionOutput.MarkdownDescription != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: descriptionOutput.MarkdownDescription,
		}
	} else if descriptionOutput.PlainTextDescription != "" {
		item.Documentation = descriptionOutput.PlainTextDescription
	}

	return item, nil
}

func (s *CompletionService) resolveVariableTypeCompletionItem(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
) (*lsp.CompletionItem, error) {
	variableType := item.Label
	if slices.Contains(schema.CoreVariableTypes, schema.VariableType(variableType)) {
		return item, nil
	}

	descriptionOutput, err := s.customVarTypeRegistry.GetDescription(
		ctx.Context,
		variableType,
		&provider.CustomVariableTypeGetDescriptionInput{},
	)
	if err != nil {
		return nil, err
	}

	if descriptionOutput.MarkdownDescription != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: descriptionOutput.MarkdownDescription,
		}
	} else if descriptionOutput.PlainTextDescription != "" {
		item.Documentation = descriptionOutput.PlainTextDescription
	}

	return item, nil
}

func (s *CompletionService) resolveFunctionCompletionItem(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
) (*lsp.CompletionItem, error) {
	functionName := item.Label

	defOutput, err := s.functionRegistry.GetDefinition(
		ctx.Context,
		functionName,
		&provider.FunctionGetDefinitionInput{},
	)
	if err != nil {
		return nil, err
	}

	if defOutput.Definition.FormattedDescription != "" {
		item.Documentation = lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: defOutput.Definition.FormattedDescription,
		}
	} else if defOutput.Definition.Description != "" {
		item.Documentation = defOutput.Definition.Description
	}

	return item, nil
}

// getResource retrieves a resource from a blueprint by name.
func getResource(blueprint *schema.Blueprint, resourceName string) *schema.Resource {
	if blueprint == nil || blueprint.Resources == nil {
		return nil
	}
	return blueprint.Resources.Values[resourceName]
}

// navigateToFieldSchema traverses the schema tree to find the schema for the last path segment.
func navigateToFieldSchema(
	rootSchema *provider.ResourceDefinitionsSchema,
	specPath []docmodel.PathSegment,
) *provider.ResourceDefinitionsSchema {
	if rootSchema == nil || len(specPath) == 0 {
		return nil
	}

	currentSchema := rootSchema
	for _, segment := range specPath {
		switch segment.Kind {
		case docmodel.PathSegmentField:
			if currentSchema.Attributes == nil {
				return nil
			}
			attrSchema, exists := currentSchema.Attributes[segment.FieldName]
			if !exists {
				return nil
			}
			currentSchema = attrSchema

		case docmodel.PathSegmentIndex:
			if currentSchema.Items == nil {
				return nil
			}
			currentSchema = currentSchema.Items
		}
	}

	return currentSchema
}

// allowedValuesCompletionItems creates completion items for AllowedValues in a schema field.
func allowedValuesCompletionItems(
	fieldSchema *provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	typedPrefix string,
	textBefore string,
	format docmodel.DocumentFormat,
) []*lsp.CompletionItem {
	if fieldSchema == nil || len(fieldSchema.AllowedValues) == 0 {
		return []*lsp.CompletionItem{}
	}

	filterPrefix, hasLeadingQuote := stripLeadingQuote(typedPrefix)
	if format != docmodel.FormatJSONC {
		hasLeadingQuote = false
	}
	prefixLen := len(typedPrefix)
	hasLeadingSpace := hasLeadingWhitespace(textBefore, prefixLen)
	prefixLower := strings.ToLower(filterPrefix)

	detail := fmt.Sprintf("Allowed value (%s)", fieldSchema.Type)
	enumKind := lsp.CompletionItemKindEnumMember
	items := make([]*lsp.CompletionItem, 0, len(fieldSchema.AllowedValues))

	for _, allowedValue := range fieldSchema.AllowedValues {
		if allowedValue == nil || allowedValue.Scalar == nil {
			continue
		}

		valueStr := allowedValue.Scalar.ToString()
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
			Data:       map[string]any{"completionType": "resourceSpecAllowedValue"},
		}

		if fieldSchema.Description != "" {
			item.Documentation = fieldSchema.Description
		}

		items = append(items, item)
	}

	return items
}
