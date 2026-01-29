package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

const (
	// CompletionColumnLeeway is the number of columns to allow for leeway
	// when determining if a position is within a range.
	// This accounts for the case when a completion trigger character such
	// as "." is not a change that leads to succesfully parsing the source,
	// meaning the range end positions in the schema tree are not updated.
	CompletionColumnLeeway = 2
)

// CompletionService is a service that provides functionality
// for completion suggestions.
type CompletionService struct {
	resourceRegistry      resourcehelpers.Registry
	dataSourceRegistry    provider.DataSourceRegistry
	customVarTypeRegistry provider.CustomVariableTypeRegistry
	functionRegistry      provider.FunctionRegistry
	linkRegistry          provider.LinkRegistry
	annotationDefCache    *core.Cache[map[string]*provider.LinkAnnotationDefinition]
	state                 *State
	logger                *zap.Logger
}

// NewCompletionService creates a new service for completion suggestions.
func NewCompletionService(
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	customVarTypeRegistry provider.CustomVariableTypeRegistry,
	functionRegistry provider.FunctionRegistry,
	state *State,
	logger *zap.Logger,
) *CompletionService {
	return &CompletionService{
		resourceRegistry:      resourceRegistry,
		dataSourceRegistry:    dataSourceRegistry,
		customVarTypeRegistry: customVarTypeRegistry,
		functionRegistry:      functionRegistry,
		annotationDefCache:    core.NewCache[map[string]*provider.LinkAnnotationDefinition](),
		state:                 state,
		logger:                logger,
	}
}

// UpdateRegistries updates the registries used by the completion service.
// This is called after plugin loading to include plugin-provided types.
func (s *CompletionService) UpdateRegistries(
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	customVarTypeRegistry provider.CustomVariableTypeRegistry,
	functionRegistry provider.FunctionRegistry,
	linkRegistry provider.LinkRegistry,
) {
	s.resourceRegistry = resourceRegistry
	s.dataSourceRegistry = dataSourceRegistry
	s.customVarTypeRegistry = customVarTypeRegistry
	s.functionRegistry = functionRegistry
	s.linkRegistry = linkRegistry
}

// CompletionResult contains the completion items and metadata for the response.
type CompletionResult struct {
	Items []*lsp.CompletionItem
	// IsIncomplete indicates whether the completion list should be marked as incomplete.
	// When true, the client will re-request completions on each keystroke for server-side filtering.
	// This is needed for substitution contexts where client-side filtering doesn't work reliably.
	IsIncomplete bool
}

// GetCompletionItems returns completion items for a given position in a document.
// Uses DocumentContext for position resolution with tree-sitter based AST analysis.
// Returns a CompletionResult with items and metadata (IsIncomplete flag for substitution contexts).
func (s *CompletionService) GetCompletionItems(
	ctx *common.LSPContext,
	docCtx *docmodel.DocumentContext,
	params *lsp.TextDocumentPositionParams,
) (*CompletionResult, error) {
	if docCtx == nil {
		return &CompletionResult{Items: []*lsp.CompletionItem{}}, nil
	}

	pos := source.Position{
		Line:   int(params.Position.Line + 1),
		Column: int(params.Position.Character + 1),
	}

	cursorCtx := docCtx.GetCursorContext(pos, CompletionColumnLeeway)
	completionCtx := docmodel.DetermineCompletionContext(cursorCtx)
	blueprint := docCtx.GetEffectiveSchema()
	if blueprint == nil {
		return &CompletionResult{Items: []*lsp.CompletionItem{}}, nil
	}

	items, err := s.getCompletionItemsByContext(ctx, blueprint, &params.Position, completionCtx, cursorCtx, docCtx.Format)
	if err != nil {
		return nil, err
	}

	if completionCtx.Kind.IsSubstitutionContext() {
		items = adjustSubstitutionCompletionItems(items, cursorCtx, &params.Position)
	}

	return &CompletionResult{
		Items:        items,
		IsIncomplete: false,
	}, nil
}

// getCompletionItemsByContext returns completion items based on the detected CompletionContext.
// This replaces the string-based element type matching with type-safe CompletionContextKind.
func (s *CompletionService) getCompletionItemsByContext(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	cursorCtx *docmodel.CursorContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if completionCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	// Centralized format capability check.
	// Key completions (field names) are disabled for JSONC because editors handle them.
	// Value completions work for all formats.
	if !completionCtx.Kind.IsEnabledForFormat(format) {
		return []*lsp.CompletionItem{}, nil
	}

	switch completionCtx.Kind {
	// Registry-based type completions (completion_types.go)
	case docmodel.CompletionContextResourceType:
		return s.getResourceTypeCompletionItems(ctx, position, completionCtx, format)
	case docmodel.CompletionContextDataSourceType:
		return s.getDataSourceTypeCompletionItems(ctx, position, completionCtx, format)
	case docmodel.CompletionContextVariableType:
		return s.getVariableTypeCompletionItems(ctx, position, completionCtx, format)
	case docmodel.CompletionContextValueType:
		return s.getValueTypeCompletionItems(position, completionCtx, format)
	case docmodel.CompletionContextDataSourceFieldType:
		return s.getDataSourceFieldTypeCompletionItems(position, completionCtx, format)
	case docmodel.CompletionContextExportType:
		return s.getExportTypeCompletionItems(position, completionCtx, format)
	case docmodel.CompletionContextVersionField:
		return s.getVersionCompletionItems(position, completionCtx, format)
	case docmodel.CompletionContextTransformField:
		return s.getTransformCompletionItems(ctx, position, completionCtx, format)
	case docmodel.CompletionContextCustomVariableTypeValue:
		return s.getCustomVariableTypeOptionsCompletionItems(ctx, position, blueprint, completionCtx, format)

	// Data source completions (completion_datasource.go)
	case docmodel.CompletionContextDataSourceFilterField:
		return s.getDataSourceFilterFieldCompletionItemsFromContext(ctx, cursorCtx, blueprint, position, completionCtx, format)
	case docmodel.CompletionContextDataSourceFilterOperator:
		return s.getDataSourceFilterOperatorCompletionItemsFromContext(position, cursorCtx, format)
	case docmodel.CompletionContextDataSourceExportAliasForValue:
		return s.getDataSourceExportAliasForCompletionItems(ctx, cursorCtx, blueprint, position, completionCtx, format)
	case docmodel.CompletionContextDataSourceExportName:
		return s.getDataSourceExportNameCompletionItems(ctx, cursorCtx, blueprint, position, completionCtx, format)

	// Schema/definition field completions (completion_schema.go)
	case docmodel.CompletionContextResourceSpecField:
		return s.getResourceSpecFieldCompletionItems(ctx, position, blueprint, completionCtx)
	case docmodel.CompletionContextResourceSpecFieldValue:
		return s.getResourceSpecFieldValueCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextResourceMetadataField:
		return s.getResourceMetadataFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextResourceAnnotationKey:
		return s.getResourceAnnotationKeyCompletionItems(ctx, position, blueprint, completionCtx)
	case docmodel.CompletionContextResourceLabelKey:
		return s.getResourceLabelKeyCompletionItems(position, blueprint, completionCtx)
	case docmodel.CompletionContextResourceAnnotationValue:
		return s.getResourceAnnotationValueCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextResourceDefinitionField:
		return s.getResourceDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextVariableDefinitionField:
		return s.getVariableDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextValueDefinitionField:
		return s.getValueDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceDefinitionField:
		return s.getDataSourceDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceFilterDefinitionField:
		return s.getDataSourceFilterDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceExportDefinitionField:
		return s.getDataSourceExportDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceAnnotationKey:
		return s.getDataSourceAnnotationKeyCompletionItems(position, blueprint, completionCtx)
	case docmodel.CompletionContextDataSourceAnnotationValue:
		return s.getDataSourceAnnotationValueCompletionItems(position)
	case docmodel.CompletionContextDataSourceMetadataField:
		return s.getDataSourceMetadataFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextIncludeDefinitionField:
		return s.getIncludeDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextExportDefinitionField:
		return s.getExportDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextLinkSelectorField:
		return s.getLinkSelectorFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextLinkSelectorExcludeValue:
		return s.getLinkSelectorExcludeValueCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextBlueprintTopLevelField:
		return s.getBlueprintTopLevelFieldCompletionItems(position, completionCtx)

	// String substitution completions (completion_stringsub.go)
	case docmodel.CompletionContextStringSubVariableRef:
		return s.getStringSubVariableCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubResourceRef:
		return s.getStringSubResourceCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubResourceProperty:
		return s.getStringSubResourcePropCompletionItemsFromContext(ctx, position, blueprint, cursorCtx)
	case docmodel.CompletionContextStringSubDataSourceRef:
		return s.getStringSubDataSourceCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubDataSourceProperty:
		return s.getStringSubDataSourcePropCompletionItemsFromContext(position, blueprint, completionCtx)
	case docmodel.CompletionContextStringSubValueRef:
		return s.getStringSubValueCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubValueProperty:
		return s.getStringSubValuePropertyCompletionItems(position, blueprint, cursorCtx)
	case docmodel.CompletionContextStringSubChildRef:
		return s.getStringSubChildCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubChildProperty:
		return s.getStringSubChildPropertyCompletionItems()
	case docmodel.CompletionContextStringSubElemRef:
		return []*lsp.CompletionItem{}, nil
	case docmodel.CompletionContextStringSubPartialPath:
		return []*lsp.CompletionItem{}, nil
	case docmodel.CompletionContextStringSubPotentialResourceProp:
		return s.getStringSubPotentialResourcePropCompletionItems(ctx, position, blueprint, completionCtx, cursorCtx)
	case docmodel.CompletionContextStringSub:
		return s.getStringSubCompletionItems(ctx, position, blueprint)

	// Export field reference completions (completion_exportfield.go)
	case docmodel.CompletionContextExportFieldTopLevel:
		return s.getExportFieldTopLevelCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldResourceRef:
		return s.getExportFieldResourceRefCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldResourceProperty:
		return s.getExportFieldResourcePropertyCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldVariableRef:
		return s.getExportFieldVariableRefCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldValueRef:
		return s.getExportFieldValueRefCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldValueProperty:
		return s.getExportFieldValuePropertyCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldChildRef:
		return s.getExportFieldChildRefCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldChildProperty:
		return s.getExportFieldChildPropertyCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldDataSourceRef:
		return s.getExportFieldDataSourceRefCompletionItems(position, blueprint, completionCtx, format)
	case docmodel.CompletionContextExportFieldDataSourceProperty:
		return s.getExportFieldDataSourcePropertyCompletionItems(position, blueprint, completionCtx, format)

	default:
		return []*lsp.CompletionItem{}, nil
	}
}

// adjustSubstitutionCompletionItems rewrites substitution completion items to use
// full-path TextEdit ranges. This enables reliable client-side filtering by anchoring
// the TextEdit range from right after "${" to the cursor position, with NewText set
// to the full path and FilterText set to "${" + full path.
func adjustSubstitutionCompletionItems(
	items []*lsp.CompletionItem,
	cursorCtx *docmodel.CursorContext,
	position *lsp.Position,
) []*lsp.CompletionItem {
	if cursorCtx == nil || len(items) == 0 {
		return items
	}

	textBefore := cursorCtx.TextBefore
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return items
	}

	subText := textBefore[subStart+2:]
	subStartCol := position.Character - lsp.UInteger(len(subText))

	fullPathRange := &lsp.Range{
		Start: lsp.Position{Line: position.Line, Character: subStartCol},
		End:   lsp.Position{Line: position.Line, Character: position.Character},
	}

	pathPrefix := substitutionPathPrefix(subText)

	for _, item := range items {
		originalNewText := extractNewText(item.TextEdit)
		// For bracket notation (e.g., ["key.with.dots"]), strip trailing dot from prefix
		// to produce correct paths like annotations["key"] instead of annotations.["key"]
		itemPrefix := pathPrefix
		if strings.HasPrefix(originalNewText, "[") && strings.HasSuffix(itemPrefix, ".") {
			itemPrefix = itemPrefix[:len(itemPrefix)-1]
		}
		fullPath := itemPrefix + originalNewText
		filterText := "${" + fullPath
		item.TextEdit = lsp.TextEdit{
			NewText: fullPath,
			Range:   fullPathRange,
		}
		item.FilterText = &filterText
	}

	return items
}

// extractNewText extracts the NewText from a CompletionItem's TextEdit field.
// TextEdit is typed as `any` and can be TextEdit or InsertReplaceEdit.
func extractNewText(textEdit any) string {
	if te, ok := textEdit.(lsp.TextEdit); ok {
		return te.NewText
	}
	return ""
}

// substitutionPathPrefix extracts the path prefix from substitution text.
// If the text ends with . or [, the entire text is the prefix.
// Otherwise, text up to and including the last . or [ is the prefix.
func substitutionPathPrefix(subText string) string {
	if len(subText) == 0 {
		return ""
	}

	lastChar := subText[len(subText)-1]
	if lastChar == '.' || lastChar == '[' {
		return subText
	}

	lastDot := strings.LastIndex(subText, ".")
	lastBracket := strings.LastIndex(subText, "[")

	lastSep := max(lastBracket, lastDot)

	if lastSep >= 0 {
		return subText[:lastSep+1]
	}

	return ""
}
