package languageservices

import (
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
) {
	s.resourceRegistry = resourceRegistry
	s.dataSourceRegistry = dataSourceRegistry
	s.customVarTypeRegistry = customVarTypeRegistry
	s.functionRegistry = functionRegistry
}

// GetCompletionItems returns completion items for a given position in a document.
// Uses DocumentContext for position resolution with tree-sitter based AST analysis.
func (s *CompletionService) GetCompletionItems(
	ctx *common.LSPContext,
	docCtx *docmodel.DocumentContext,
	params *lsp.TextDocumentPositionParams,
) ([]*lsp.CompletionItem, error) {
	if docCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	pos := source.Position{
		Line:   int(params.Position.Line + 1),
		Column: int(params.Position.Character + 1),
	}

	nodeCtx := docCtx.GetNodeContext(pos, CompletionColumnLeeway)
	completionCtx := docmodel.DetermineCompletionContext(nodeCtx)
	blueprint := docCtx.GetEffectiveSchema()
	if blueprint == nil {
		return []*lsp.CompletionItem{}, nil
	}

	return s.getCompletionItemsByContext(ctx, blueprint, &params.Position, completionCtx, nodeCtx, docCtx.Format)
}

// getCompletionItemsByContext returns completion items based on the detected CompletionContext.
// This replaces the string-based element type matching with type-safe CompletionContextKind.
func (s *CompletionService) getCompletionItemsByContext(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
	nodeCtx *docmodel.NodeContext,
	format docmodel.DocumentFormat,
) ([]*lsp.CompletionItem, error) {
	if completionCtx == nil {
		return []*lsp.CompletionItem{}, nil
	}

	switch completionCtx.Kind {
	// Registry-based type completions (completion_types.go)
	case docmodel.CompletionContextResourceType:
		return s.getResourceTypeCompletionItems(ctx)
	case docmodel.CompletionContextDataSourceType:
		return s.getDataSourceTypeCompletionItems(ctx)
	case docmodel.CompletionContextVariableType:
		return s.getVariableTypeCompletionItems(ctx)
	case docmodel.CompletionContextValueType:
		return s.getValueTypeCompletionItems()
	case docmodel.CompletionContextDataSourceFieldType:
		return s.getDataSourceFieldTypeCompletionItems()
	case docmodel.CompletionContextExportType:
		return s.getExportTypeCompletionItems()
	case docmodel.CompletionContextVersionField:
		return s.getVersionCompletionItems(position, completionCtx, format)
	case docmodel.CompletionContextTransformField:
		return s.getTransformCompletionItems(ctx, position, completionCtx, format)
	case docmodel.CompletionContextCustomVariableTypeValue:
		return s.getCustomVariableTypeOptionsCompletionItems(ctx, position, blueprint, completionCtx, format)

	// Data source completions (completion_datasource.go)
	case docmodel.CompletionContextDataSourceFilterField:
		return s.getDataSourceFilterFieldCompletionItemsFromContext(ctx, nodeCtx, blueprint)
	case docmodel.CompletionContextDataSourceFilterOperator:
		return s.getDataSourceFilterOperatorCompletionItemsFromContext(position, nodeCtx, format)
	case docmodel.CompletionContextDataSourceExportAliasForValue:
		return s.getDataSourceExportAliasForCompletionItems(ctx, nodeCtx, blueprint, position, completionCtx, format)
	case docmodel.CompletionContextDataSourceExportName:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getDataSourceExportNameCompletionItems(ctx, nodeCtx, blueprint, position, completionCtx, format)

	// Schema/definition field completions (completion_schema.go)
	case docmodel.CompletionContextResourceSpecField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceSpecFieldCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextResourceSpecFieldValue:
		return s.getResourceSpecFieldValueCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextResourceMetadataField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceMetadataFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextResourceDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextVariableDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getVariableDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextValueDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getValueDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getDataSourceDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceFilterDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getDataSourceFilterDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceExportDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getDataSourceExportDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextDataSourceMetadataField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getDataSourceMetadataFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextIncludeDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getIncludeDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextExportDefinitionField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getExportDefinitionFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextBlueprintTopLevelField:
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getBlueprintTopLevelFieldCompletionItems(position, completionCtx)

	// String substitution completions (completion_stringsub.go)
	case docmodel.CompletionContextStringSubVariableRef:
		return s.getStringSubVariableCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubResourceRef:
		return s.getStringSubResourceCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubResourceProperty:
		return s.getStringSubResourcePropCompletionItemsFromContext(ctx, position, blueprint, nodeCtx)
	case docmodel.CompletionContextStringSubDataSourceRef:
		return s.getStringSubDataSourceCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubDataSourceProperty:
		return s.getStringSubDataSourcePropCompletionItemsFromContext(position, blueprint, completionCtx)
	case docmodel.CompletionContextStringSubValueRef:
		return s.getStringSubValueCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubChildRef:
		return s.getStringSubChildCompletionItems(position, blueprint)
	case docmodel.CompletionContextStringSubElemRef:
		return []*lsp.CompletionItem{}, nil
	case docmodel.CompletionContextStringSubPartialPath:
		return []*lsp.CompletionItem{}, nil
	case docmodel.CompletionContextStringSubPotentialResourceProp:
		return s.getStringSubPotentialResourcePropCompletionItems(ctx, position, blueprint, completionCtx, nodeCtx)
	case docmodel.CompletionContextStringSub:
		return s.getStringSubCompletionItems(ctx, position, blueprint)

	default:
		return []*lsp.CompletionItem{}, nil
	}
}
