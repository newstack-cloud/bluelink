package languageservices

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
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

	// Convert LSP position to source.Position (1-based)
	pos := source.Position{
		Line:   int(params.Position.Line + 1),
		Column: int(params.Position.Character + 1),
	}

	// Get node context using DocumentContext with completion leeway
	nodeCtx := docCtx.GetNodeContext(pos, CompletionColumnLeeway)

	// Determine completion context using type-safe detection
	completionCtx := docmodel.DetermineCompletionContext(nodeCtx)

	// Get the effective blueprint for completion items
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
	case docmodel.CompletionContextDataSourceFilterField:
		return s.getDataSourceFilterFieldCompletionItemsFromContext(ctx, nodeCtx, blueprint)
	case docmodel.CompletionContextDataSourceFilterOperator:
		return s.getDataSourceFilterOperatorCompletionItemsFromContext(position, nodeCtx)
	case docmodel.CompletionContextExportType:
		return s.getExportTypeCompletionItems()
	case docmodel.CompletionContextResourceSpecField:
		// Schema-based completions disabled for JSONC
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceSpecFieldCompletionItems(ctx, position, blueprint, completionCtx, format)
	case docmodel.CompletionContextResourceMetadataField:
		// Schema-based completions disabled for JSONC
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceMetadataFieldCompletionItems(position, completionCtx)
	case docmodel.CompletionContextResourceDefinitionField:
		// Schema-based completions disabled for JSONC - focus on YAML for v0
		if format == docmodel.FormatJSONC {
			return []*lsp.CompletionItem{}, nil
		}
		return s.getResourceDefinitionFieldCompletionItems(position, completionCtx)
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
		return []*lsp.CompletionItem{}, nil // elem refs don't have completion suggestions
	case docmodel.CompletionContextStringSubPartialPath:
		return []*lsp.CompletionItem{}, nil // partial paths don't have completion suggestions
	case docmodel.CompletionContextStringSubPotentialResourceProp:
		return s.getStringSubPotentialResourcePropCompletionItems(ctx, position, blueprint, completionCtx, nodeCtx)
	case docmodel.CompletionContextStringSub:
		return s.getStringSubCompletionItems(ctx, position, blueprint)
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
	default:
		return []*lsp.CompletionItem{}, nil
	}
}

// getDataSourceFilterFieldCompletionItemsFromContext adapts filter field completion to use NodeContext.
func (s *CompletionService) getDataSourceFilterFieldCompletionItemsFromContext(
	ctx *common.LSPContext,
	nodeCtx *docmodel.NodeContext,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {
	if nodeCtx == nil || blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName, ok := nodeCtx.GetDataSourceName()
	if !ok || dataSourceName == "" {
		return []*lsp.CompletionItem{}, nil
	}

	dataSource, hasDataSource := blueprint.DataSources.Values[dataSourceName]
	if !hasDataSource || dataSource.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	filterFieldsOutput, err := s.dataSourceRegistry.GetFilterFields(
		ctx.Context,
		string(dataSource.Type.Value),
		&provider.DataSourceGetFilterFieldsInput{},
	)
	if err != nil {
		return nil, err
	}

	completionItems := []*lsp.CompletionItem{}
	filterFieldDetail := "Data source filter field"
	for filterField := range filterFieldsOutput.FilterFields {
		enumKind := lsp.CompletionItemKindEnum
		completionItems = append(completionItems, &lsp.CompletionItem{
			Label:      filterField,
			Detail:     &filterFieldDetail,
			Kind:       &enumKind,
			InsertText: &filterField,
			Data:       map[string]any{"completionType": "dataSourceFilterField"},
		})
	}

	return completionItems, nil
}

// getDataSourceFilterOperatorCompletionItemsFromContext adapts filter operator completion to use NodeContext.
func (s *CompletionService) getDataSourceFilterOperatorCompletionItemsFromContext(
	position *lsp.Position,
	nodeCtx *docmodel.NodeContext,
) ([]*lsp.CompletionItem, error) {
	filterOperatorDetail := "Data source filter operator"
	enumKind := lsp.CompletionItemKindEnum

	// Get operator element position from schema node (preferred) or fallback to current position
	var operatorElementPosition *source.Position
	if nodeCtx != nil && nodeCtx.SchemaNode != nil && nodeCtx.SchemaNode.Range != nil {
		operatorElementPosition = nodeCtx.SchemaNode.Range.Start
	} else {
		// Fallback to current position
		operatorElementPosition = &source.Position{
			Line:   int(position.Line + 1),
			Column: int(position.Character + 1),
		}
	}

	// Determine if cursor is right after "operator:" field declaration
	isPrecededByOperator := nodeCtx != nil && nodeCtx.IsPrecededByOperatorField()

	filterOpItems := []*lsp.CompletionItem{}
	for _, filterOperator := range schema.DataSourceFilterOperators {
		filterOperatorStr := fmt.Sprintf("\"%s\"", string(filterOperator))
		edit := lsp.TextEdit{
			NewText: filterOperatorStr,
			Range: getOperatorInsertRange(
				position,
				filterOperatorStr,
				isPrecededByOperator,
				operatorElementPosition,
			),
		}
		filterOpItems = append(
			filterOpItems,
			&lsp.CompletionItem{
				Label:    filterOperatorStr,
				Detail:   &filterOperatorDetail,
				Kind:     &enumKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "dataSourceFilterOperator"},
			},
		)
	}

	return filterOpItems, nil
}

// getStringSubPotentialResourcePropCompletionItems handles completion for potential standalone
// resource property patterns like ${myResource. or ${myResource[
// It validates the potential resource name against the blueprint and delegates to resource property completion.
func (s *CompletionService) getStringSubPotentialResourcePropCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	completionCtx *docmodel.CompletionContext,
	nodeCtx *docmodel.NodeContext,
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

	// Delegate to the existing resource property completion logic
	return s.getStringSubResourcePropCompletionItemsFromContext(ctx, position, blueprint, nodeCtx)
}

// getStringSubResourcePropCompletionItemsFromContext adapts resource property completion to use NodeContext.
func (s *CompletionService) getStringSubResourcePropCompletionItemsFromContext(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	nodeCtx *docmodel.NodeContext,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	// Get text before cursor for analysis
	textBefore := ""
	if nodeCtx != nil {
		textBefore = nodeCtx.TextBefore
	}

	// Try to get resource property from schema element first
	var resourceProp *substitutions.SubstitutionResourceProperty
	if nodeCtx != nil && nodeCtx.SchemaElement != nil {
		resourceProp, _ = nodeCtx.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	}

	// Fallback: parse resource property from text when schema element is unavailable
	if resourceProp == nil {
		resourceProp = parseResourcePropertyFromText(textBefore, blueprint)
	}

	if resourceProp == nil {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	// Check if we're directly after the resource name (need top-level props like spec, metadata)
	// Handles both: resources.myResource. and ${myResource.
	if strings.HasSuffix(textBefore, fmt.Sprintf(".%s.", resourceProp.ResourceName)) ||
		strings.HasSuffix(textBefore, fmt.Sprintf("${%s.", resourceProp.ResourceName)) ||
		strings.HasSuffix(textBefore, fmt.Sprintf("${%s[", resourceProp.ResourceName)) {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	// Check if we're in the spec path (works for both resources.x.spec. and ${x.spec.)
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "spec" {
		return s.getResourceSpecPropCompletionItems(ctx, position, blueprint, resourceProp)
	}

	// Check if we're in the metadata path (works for both resources.x.metadata. and ${x.metadata.)
	if len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "metadata" {
		return s.getResourceMetadataPropCompletionItemsForPath(position, blueprint, resourceProp, nodeCtx)
	}

	return []*lsp.CompletionItem{}, nil
}

// parseResourcePropertyFromText extracts resource property path from text.
// Used as fallback when SchemaElement is not available during editing.
func parseResourcePropertyFromText(
	textBefore string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	// Find the start of the substitution
	subStart := strings.LastIndex(textBefore, "${")
	if subStart == -1 {
		return nil
	}

	subText := textBefore[subStart+2:] // Text after ${

	// Try parsing with resources. prefix first
	if strings.HasPrefix(subText, "resources.") {
		return parseResourcePropertyWithPrefix(subText, blueprint)
	}

	// Try parsing as standalone resource name (without resources. prefix)
	return parseStandaloneResourceProperty(subText, blueprint)
}

func parseResourcePropertyWithPrefix(
	subText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	// Parse: resources.resourceName.path1.path2.
	remaining := strings.TrimPrefix(subText, "resources.")
	return parseResourcePath(remaining, blueprint)
}

func parseStandaloneResourceProperty(
	subText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	// For standalone resource names, the first part before . or [ is the resource name
	// Example: myResource.metadata.annotations[
	return parseResourcePath(subText, blueprint)
}

func parseResourcePath(
	pathText string,
	blueprint *schema.Blueprint,
) *substitutions.SubstitutionResourceProperty {
	if blueprint.Resources == nil {
		return nil
	}

	// Split by . but also handle [ for bracket notation
	// First, replace [ with . to normalize, then split
	normalized := strings.ReplaceAll(pathText, "[", ".")
	parts := strings.Split(normalized, ".")
	if len(parts) == 0 {
		return nil
	}

	// Clean up resource name (remove quotes and brackets)
	resourceName := strings.Trim(parts[0], "\"'[]")
	if resourceName == "" {
		return nil
	}

	// Verify resource exists
	if blueprint.Resources.Values[resourceName] == nil {
		return nil
	}

	// Build path from remaining parts (excluding trailing empty string from ".")
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

// getStringSubDataSourcePropCompletionItemsFromContext adapts data source property completion to use CompletionContext.
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

	dataSourceItems := []*lsp.CompletionItem{}

	// Get data source name from completion context
	dataSourceName := completionCtx.DataSourceName
	if dataSourceName == "" {
		return dataSourceItems, nil
	}

	dataSource := getDataSource(blueprint, dataSourceName)
	if dataSource == nil || dataSource.Exports == nil {
		return dataSourceItems, nil
	}

	for exportName := range dataSource.Exports.Values {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: exportName,
			Range:   insertRange,
		}
		dataSourceItems = append(
			dataSourceItems,
			&lsp.CompletionItem{
				Label:    exportName,
				Detail:   &detail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "dataSourceProperty"},
			},
		)
	}

	return dataSourceItems, nil
}

// getResourceSpecFieldCompletionItems returns completion items for resource spec fields
// when editing directly in the YAML/JSONC definition (not in substitutions).
func (s *CompletionService) getResourceSpecFieldCompletionItems(
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

	// Navigate the schema tree based on the spec path depth
	// Path is like: /resources/{name}/spec/field1/field2/...
	// GetSpecPath() returns segments after /resources/{name}/spec/
	specPath := completionCtx.NodeCtx.ASTPath.GetSpecPath()
	currentSchema := specDefOutput.SpecDefinition.Schema

	// Navigate to the correct schema depth
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

	// Get the typed prefix for filtering completions
	typedPrefix := ""
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
	}

	return resourceDefAttributesSchemaCompletionItemsWithPrefix(
		currentSchema.Attributes,
		position,
		"Resource spec field",
		typedPrefix,
	), nil
}

// coreResourceDefinitionFieldInfo holds description information for core resource fields.
type coreResourceDefinitionFieldInfo struct {
	name        string
	description string
}

// Core blueprint resource definition fields with descriptions
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

// Variable definition fields
var coreVariableDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The variable type (string, integer, float, boolean, or a custom type)."},
	{name: "description", description: "A human-readable description of the variable's purpose."},
	{name: "secret", description: "When true, the variable value is treated as sensitive and masked in logs."},
	{name: "default", description: "The default value used when no value is provided."},
	{name: "allowedValues", description: "An array of valid values that constrain input validation."},
}

// Value definition fields
var coreValueDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The value type (string, integer, float, boolean, array, object)."},
	{name: "value", description: "The computed value, which may include substitutions."},
	{name: "description", description: "A human-readable description of the value's purpose."},
	{name: "secret", description: "When true, the value is treated as sensitive and masked in logs."},
}

// DataSource definition fields
var coreDataSourceDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The data source type identifier (e.g., `aws/vpc`)."},
	{name: "metadata", description: "Metadata including `displayName`, `annotations`, and `custom` fields."},
	{name: "filter", description: "Filter criteria to select specific data source instances."},
	{name: "exports", description: "Field definitions for data exported from this data source."},
	{name: "description", description: "A human-readable description of the data source's purpose."},
}

// DataSource filter definition fields (inside filter)
var coreDataSourceFilterDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "field", description: "The name of the data source field to filter on."},
	{name: "operator", description: "The comparison operator (=, !=, in, not in, contains, starts with, etc.)."},
	{name: "search", description: "The value(s) to search for, which may include substitutions."},
}

// DataSource export definition fields (inside exports/{name})
var coreDataSourceExportDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The export field type (string, integer, float, boolean, array)."},
	{name: "aliasFor", description: "The original field name from the provider if different from the export name."},
	{name: "description", description: "A human-readable description of the exported field."},
}

// DataSource metadata fields (inside metadata)
var coreDataSourceMetadataFields = []coreResourceDefinitionFieldInfo{
	{name: "displayName", description: "A human-readable name for the data source."},
	{name: "annotations", description: "Key-value pairs for configuring data source behavior."},
	{name: "custom", description: "Custom metadata fields for provider-specific configuration."},
}

// Include definition fields
var coreIncludeDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "path", description: "The path to the child blueprint (local file or remote URL)."},
	{name: "variables", description: "Variables to pass to the child blueprint."},
	{name: "metadata", description: "Extra metadata for include resolver plugins."},
	{name: "description", description: "A human-readable description of the included blueprint."},
}

// Export definition fields
var coreExportDefinitionFields = []coreResourceDefinitionFieldInfo{
	{name: "type", description: "The export type (string, integer, float, boolean, array, object)."},
	{name: "field", description: "A substitution reference to the value being exported."},
	{name: "description", description: "A human-readable description of the exported value."},
}

// Blueprint top-level fields
var coreBlueprintTopLevelFields = []coreResourceDefinitionFieldInfo{
	{name: "version", description: "The blueprint specification version (e.g., `2021-12-18`)."},
	{name: "transform", description: "One or more transforms to apply to the blueprint."},
	{name: "variables", description: "Input variables that parameterize the blueprint."},
	{name: "values", description: "Computed values derived from variables and other sources."},
	{name: "include", description: "Child blueprints to include in this blueprint."},
	{name: "resources", description: "The infrastructure resources defined in this blueprint."},
	{name: "datasources", description: "External data sources to look up existing infrastructure."},
	{name: "exports", description: "Values exported from this blueprint for external access."},
	{name: "metadata", description: "Blueprint-level metadata for organization and documentation."},
}

// getResourceDefinitionFieldCompletionItems returns completion items for core resource fields
// like type, description, spec, metadata, etc. when editing at the resource definition level.
// Note: Only used for YAML - JSONC schema completions are disabled.
func (s *CompletionService) getResourceDefinitionFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	// Get the typed prefix for filtering completions
	typedPrefix := ""
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
	}

	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)

	detail := "Resource field"
	fieldKind := lsp.CompletionItemKindField

	items := make([]*lsp.CompletionItem, 0, len(coreResourceDefinitionFields))
	for _, fieldInfo := range coreResourceDefinitionFields {
		// Filter by prefix if one is provided
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

		// Add documentation
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreExportDefinitionFields,
		position,
		typedPrefix,
		"Export field",
		"exportDefinitionField",
	), nil
}

// getBlueprintTopLevelFieldCompletionItems returns completion items for blueprint top-level fields.
func (s *CompletionService) getBlueprintTopLevelFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
	}
	return createDefinitionFieldCompletionItems(
		coreBlueprintTopLevelFields,
		position,
		typedPrefix,
		"Blueprint field",
		"blueprintTopLevelField",
	), nil
}

func (s *CompletionService) getResourceTypeCompletionItems(
	ctx *common.LSPContext,
) ([]*lsp.CompletionItem, error) {

	resourceTypes, err := s.resourceRegistry.ListResourceTypes(ctx.Context)
	if err != nil {
		return nil, err
	}

	completionItems := []*lsp.CompletionItem{}
	resourceTypeDetail := "Resource type"
	for _, resourceType := range resourceTypes {
		enumKind := lsp.CompletionItemKindEnum
		completionItems = append(completionItems, &lsp.CompletionItem{
			Label:      resourceType,
			Detail:     &resourceTypeDetail,
			Kind:       &enumKind,
			InsertText: &resourceType,
			Data:       map[string]any{"completionType": "resourceType"},
		})
	}

	return completionItems, nil
}

func (s *CompletionService) getDataSourceTypeCompletionItems(
	ctx *common.LSPContext,
) ([]*lsp.CompletionItem, error) {

	dataSourceTypes, err := s.dataSourceRegistry.ListDataSourceTypes(ctx.Context)
	if err != nil {
		return nil, err
	}

	completionItems := []*lsp.CompletionItem{}
	dataSourceTypeDetail := "Data source type"
	for _, dataSourceType := range dataSourceTypes {
		enumKind := lsp.CompletionItemKindEnum
		completionItems = append(completionItems, &lsp.CompletionItem{
			Label:      dataSourceType,
			Detail:     &dataSourceTypeDetail,
			Kind:       &enumKind,
			InsertText: &dataSourceType,
			Data:       map[string]any{"completionType": "dataSourceType"},
		})
	}

	return completionItems, nil
}

func (s *CompletionService) getVariableTypeCompletionItems(
	ctx *common.LSPContext,
) ([]*lsp.CompletionItem, error) {
	variableTypeDetail := "Variable type"
	enumKind := lsp.CompletionItemKindEnum

	typeItems := []*lsp.CompletionItem{}
	for _, coreType := range schema.CoreVariableTypes {
		coreTypeStr := string(coreType)
		typeItems = append(
			typeItems,
			&lsp.CompletionItem{
				Label:      coreTypeStr,
				Detail:     &variableTypeDetail,
				Kind:       &enumKind,
				InsertText: &coreTypeStr,
				Data:       map[string]any{"completionType": "variableType"},
			},
		)
	}

	customTypes, err := s.customVarTypeRegistry.ListCustomVariableTypes(ctx.Context)
	if err != nil {
		s.logger.Error("Failed to list custom variable types, returning core types only", zap.Error(err))
		return typeItems, nil
	}

	for _, customType := range customTypes {
		typeItems = append(typeItems, &lsp.CompletionItem{
			Label:      customType,
			Detail:     &variableTypeDetail,
			Kind:       &enumKind,
			InsertText: &customType,
			Data:       map[string]any{"completionType": "variableType"},
		})
	}

	return typeItems, nil
}

func (s *CompletionService) getValueTypeCompletionItems() ([]*lsp.CompletionItem, error) {
	valueTypeDetail := "Value type"
	enumKind := lsp.CompletionItemKindEnum

	typeItems := []*lsp.CompletionItem{}
	for _, valueType := range schema.ValueTypes {
		valueTypeStr := string(valueType)
		typeItems = append(
			typeItems,
			&lsp.CompletionItem{
				Label:      valueTypeStr,
				Detail:     &valueTypeDetail,
				Kind:       &enumKind,
				InsertText: &valueTypeStr,
				Data:       map[string]any{"completionType": "valueType"},
			},
		)
	}

	return typeItems, nil
}

func (s *CompletionService) getDataSourceFieldTypeCompletionItems() ([]*lsp.CompletionItem, error) {
	fieldTypeDetail := "Data source field type"
	enumKind := lsp.CompletionItemKindEnum

	typeItems := []*lsp.CompletionItem{}
	for _, fieldType := range schema.DataSourceFieldTypes {
		fieldTypeStr := string(fieldType)
		typeItems = append(
			typeItems,
			&lsp.CompletionItem{
				Label:      fieldTypeStr,
				Detail:     &fieldTypeDetail,
				Kind:       &enumKind,
				InsertText: &fieldTypeStr,
				Data:       map[string]any{"completionType": "dataSourceFieldType"},
			},
		)
	}

	return typeItems, nil
}

func (s *CompletionService) getExportTypeCompletionItems() ([]*lsp.CompletionItem, error) {
	exportTypeDetail := "Export type"
	enumKind := lsp.CompletionItemKindEnum

	typeItems := []*lsp.CompletionItem{}
	for _, exportType := range schema.ExportTypes {
		exportTypeStr := string(exportType)
		typeItems = append(
			typeItems,
			&lsp.CompletionItem{
				Label:      exportTypeStr,
				Detail:     &exportTypeDetail,
				Kind:       &enumKind,
				InsertText: &exportTypeStr,
				Data:       map[string]any{"completionType": "exportType"},
			},
		)
	}

	return typeItems, nil
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
		insertRange := getItemInsertRange(
			position,
		)
		edit := lsp.TextEdit{
			NewText: varName,
			Range:   insertRange,
		}
		varItems = append(
			varItems,
			&lsp.CompletionItem{
				Label:    varName,
				Detail:   &variableDetail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "variable"},
			},
		)
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
		insertRange := getItemInsertRange(
			position,
		)
		edit := lsp.TextEdit{
			NewText: resourceName,
			Range:   insertRange,
		}
		resourceItems = append(
			resourceItems,
			&lsp.CompletionItem{
				Label:    resourceName,
				Detail:   &resourceDetail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "resource"},
			},
		)
	}

	return resourceItems, nil
}

func (s *CompletionService) getResourceSpecPropCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
	resourceProp *substitutions.SubstitutionResourceProperty,
) ([]*lsp.CompletionItem, error) {

	resource := getResource(blueprint, resourceProp.ResourceName)
	if resource == nil || resource.Type == nil {
		return []*lsp.CompletionItem{}, nil
	}

	specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
		ctx.Context,
		resource.Type.Value,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		// Return a hint completion item explaining why completions aren't available
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

	currentSchema := specDefOutput.SpecDefinition.Schema
	pathAfterSpec := resourceProp.Path[1:]
	i := 0
	for currentSchema != nil && i < len(pathAfterSpec) {
		if currentSchema.Type != provider.ResourceDefinitionsSchemaTypeObject {
			currentSchema = nil
		} else {
			currentSchema = currentSchema.Attributes[pathAfterSpec[i].FieldName]
		}
		i += 1
	}

	if currentSchema == nil || currentSchema.Attributes == nil {
		return []*lsp.CompletionItem{}, nil
	}

	return resourceDefAttributesSchemaCompletionItems(
		currentSchema.Attributes,
		position,
		"Resource spec property",
	), nil
}

// noCompletionsHint returns a single completion item that displays a hint message.
// This is used when completions aren't available (e.g., resource type not found).
func noCompletionsHint(position *lsp.Position, message string) []*lsp.CompletionItem {
	hintKind := lsp.CompletionItemKindText
	detail := "No completions available"
	return []*lsp.CompletionItem{
		{
			Label:  message,
			Kind:   &hintKind,
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range:   getItemInsertRange(position),
				NewText: "",
			},
			Data: map[string]any{
				"completionType": "hint",
			},
		},
	}
}

// resourceDefAttributesSchemaCompletionItems creates completion items for resource attributes
// used in substitution references (inside ${...}). These completions should NOT include
// colons or quotes since they're property access paths, not key-value definitions.
func resourceDefAttributesSchemaCompletionItems(
	attributes map[string]*provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	attrDetail string,
) []*lsp.CompletionItem {
	return resourceDefAttributesSchemaCompletionItemsForSubstitution(attributes, position, attrDetail, "")
}

// resourceDefAttributesSchemaCompletionItemsForSubstitution creates completion items for
// substitution references (inside ${...}). These completions use just the field name
// without colons or quotes since they're property access paths.
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

		// For substitutions, just use the field name (no colon or quotes)
		edit := lsp.TextEdit{
			NewText: attrName,
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
		// Filter by prefix if one is provided
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

		// Build a detailed label showing the type if available
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

		// Add documentation from the schema if available
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

// resourcePropInfo holds label and description for resource properties.
type resourcePropInfo struct {
	label       string
	description string
}

func createResourcePropCompletionItemsWithDescriptions(
	position *lsp.Position,
	props []resourcePropInfo,
	detail string,
) []*lsp.CompletionItem {
	insertRange := getItemInsertRange(position)
	fieldKind := lsp.CompletionItemKindField
	items := make([]*lsp.CompletionItem, 0, len(props))

	for _, prop := range props {
		item := &lsp.CompletionItem{
			Label:  prop.label,
			Detail: &detail,
			Kind:   &fieldKind,
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

var resourceTopLevelProps = []resourcePropInfo{
	{label: "metadata", description: "Resource metadata including `displayName`, `labels`, `annotations`, and `custom` fields."},
	{label: "spec", description: "The resource specification containing provider-specific configuration and computed fields."},
	{label: "state", description: "The current deployment state of the resource from the external provider."},
}

func getResourceTopLevelPropCompletionItems(position *lsp.Position) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithDescriptions(position, resourceTopLevelProps, "Resource property")
}

var resourceMetadataProps = []resourcePropInfo{
	{label: "displayName", description: "A human-readable name for the resource, used in UI displays."},
	{label: "labels", description: "Key-value pairs for organizing and selecting resources. Used by `linkSelector`."},
	{label: "annotations", description: "Key-value pairs for storing additional metadata. Unlike labels, annotations are not used for selection."},
	{label: "custom", description: "Custom metadata fields specific to your use case."},
}

func getResourceMetadataPropCompletionItems(position *lsp.Position) []*lsp.CompletionItem {
	return createResourcePropCompletionItemsWithDescriptions(position, resourceMetadataProps, "Resource metadata property")
}

// getResourceMetadataFieldCompletionItems returns completion items for metadata fields
// when editing directly in YAML resource definitions (with colons for key-value syntax).
func (s *CompletionService) getResourceMetadataFieldCompletionItems(
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, error) {
	typedPrefix := ""
	if completionCtx.NodeCtx != nil {
		typedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
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
func (s *CompletionService) getResourceMetadataPropCompletionItemsForPath(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	resourceProp *substitutions.SubstitutionResourceProperty,
	nodeCtx *docmodel.NodeContext,
) ([]*lsp.CompletionItem, error) {
	if len(resourceProp.Path) <= 1 {
		return getResourceMetadataPropCompletionItems(position), nil
	}

	resource := getResource(blueprint, resourceProp.ResourceName)
	if resource == nil || resource.Metadata == nil {
		return []*lsp.CompletionItem{}, nil
	}

	secondSegment := resourceProp.Path[1].FieldName
	keys, detail := getMetadataKeysAndDetail(resource.Metadata, secondSegment)
	if keys == nil {
		return []*lsp.CompletionItem{}, nil
	}

	quoteType := docmodel.QuoteTypeNone
	if nodeCtx != nil {
		quoteType = nodeCtx.GetEnclosingQuoteType()
	}

	return createMetadataKeyCompletionItems(position, keys, detail, quoteType), nil
}

// createMetadataKeyCompletionItems creates completion items for metadata keys.
// Keys containing special characters use bracket notation with contextual quotes:
// - Inside double-quoted strings: ['key.with.dots'] (single quotes)
// - Otherwise: ["key.with.dots"] (double quotes)
func createMetadataKeyCompletionItems(
	position *lsp.Position,
	keys []string,
	detail string,
	quoteType docmodel.QuoteType,
) []*lsp.CompletionItem {
	fieldKind := lsp.CompletionItemKindField
	items := make([]*lsp.CompletionItem, 0, len(keys))

	for _, key := range keys {
		var textEdit lsp.TextEdit
		if needsBracketNotation(key) {
			// Replace the preceding "." with bracket notation
			// Use single quotes inside double-quoted strings to avoid escaping
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

// formatBracketNotation formats a key with bracket notation using appropriate quotes.
// Uses single quotes when inside double-quoted strings to avoid escaping.
func formatBracketNotation(key string, quoteType docmodel.QuoteType) string {
	if quoteType == docmodel.QuoteTypeDouble {
		return fmt.Sprintf(`['%s']`, key)
	}
	return fmt.Sprintf(`["%s"]`, key)
}

// needsBracketNotation returns true if the key contains characters that require
// bracket notation in substitution paths (e.g., dots, brackets, quotes).
func needsBracketNotation(key string) bool {
	for _, c := range key {
		if c == '.' || c == '[' || c == ']' || c == '"' || c == ' ' {
			return true
		}
	}
	return false
}

// getBracketNotationInsertRange returns a range that starts 1 character before
// the cursor position, to replace the preceding "." with bracket notation.
func getBracketNotationInsertRange(position *lsp.Position) *lsp.Range {
	startChar := position.Character
	if startChar > 0 {
		startChar--
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
	default:
		return nil, ""
	}
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
		insertRange := getItemInsertRange(
			position,
		)
		edit := lsp.TextEdit{
			NewText: dataSourceName,
			Range:   insertRange,
		}
		dataSourceItems = append(
			dataSourceItems,
			&lsp.CompletionItem{
				Label:    dataSourceName,
				Detail:   &detail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "dataSource"},
			},
		)
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
		insertRange := getItemInsertRange(
			position,
		)
		edit := lsp.TextEdit{
			NewText: valueName,
			Range:   insertRange,
		}
		valueItems = append(
			valueItems,
			&lsp.CompletionItem{
				Label:    valueName,
				Detail:   &detail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "value"},
			},
		)
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
		insertRange := getItemInsertRange(
			position,
		)
		edit := lsp.TextEdit{
			NewText: includeName,
			Range:   insertRange,
		}
		includeItems = append(
			includeItems,
			&lsp.CompletionItem{
				Label:    includeName,
				Detail:   &detail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data:     map[string]any{"completionType": "child"},
			},
		)
	}

	return includeItems, nil
}

func (s *CompletionService) getStringSubCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {

	items := []*lsp.CompletionItem{}

	// Sort priority order:
	// 1. Resources
	// 2. Variables
	// 3. Functions
	// 4. Data sources
	// 5. Values
	// 6. Child blueprints

	resourceItems := s.getResourceCompletionItems(position, blueprint /* sortPrefix */, "1-")
	items = append(items, resourceItems...)

	variableItems := s.getVariableCompletionItems(position, blueprint /* sortPrefix */, "2-")
	items = append(items, variableItems...)

	functionItems := s.getFunctionCompletionItems(ctx, position /* sortPrefix */, "3-")
	items = append(items, functionItems...)

	dataSourceItems := s.getDataSourceCompletionItems(position, blueprint /* sortPrefix */, "4-")
	items = append(items, dataSourceItems...)

	valueItems := s.getValueCompletionItems(position, blueprint /* sortPrefix */, "5-")
	items = append(items, valueItems...)

	childItems := s.getChildCompletionItems(position, blueprint /* sortPrefix */, "6-")
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
		resourceItems = append(
			resourceItems,
			&lsp.CompletionItem{
				Label:    resourceText,
				Detail:   &resourceDetail,
				Kind:     &resourceKind,
				TextEdit: edit,
				SortText: &sortText,
				Data:     map[string]any{"completionType": "resource"},
			},
		)

		// Add standalone version: {name} (without resources. prefix)
		standaloneEdit := lsp.TextEdit{
			NewText: resourceName,
			Range:   insertRange,
		}
		standaloneSortText := fmt.Sprintf("%s1-%s", sortPrefix, resourceName)
		resourceItems = append(
			resourceItems,
			&lsp.CompletionItem{
				Label:    resourceName,
				Detail:   &standaloneResourceDetail,
				Kind:     &resourceKind,
				TextEdit: standaloneEdit,
				SortText: &standaloneSortText,
				Data:     map[string]any{"completionType": "resourceStandalone"},
			},
		)
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

	for variableName := range blueprint.Variables.Values {
		variableText := fmt.Sprintf("variables.%s", variableName)
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: variableText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, variableName)
		variableItems = append(
			variableItems,
			&lsp.CompletionItem{
				Label:    variableText,
				Detail:   &variableDetail,
				Kind:     &variableKind,
				TextEdit: edit,
				SortText: &sortText,
				Data:     map[string]any{"completionType": "variable"},
			},
		)
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
	functions, err := s.functionRegistry.ListFunctions(ctx.Context)
	if err != nil {
		s.logger.Error("Failed to list functions", zap.Error(err))
		return functionItems
	}

	for _, function := range functions {
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: fmt.Sprintf("%s($0)", function),
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, function)
		functionItems = append(
			functionItems,
			&lsp.CompletionItem{
				Label:            function,
				Detail:           &functionDetail,
				Kind:             &functionKind,
				InsertTextFormat: &lsp.InsertTextFormatSnippet,
				TextEdit:         edit,
				SortText:         &sortText,
				Data:             map[string]any{"completionType": "function"},
			},
		)
	}

	return functionItems
}

func (s *CompletionService) getDataSourceCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	dataSourceDetail := "Data source"
	dataSourceKind := lsp.CompletionItemKindValue

	dataSourceItems := []*lsp.CompletionItem{}

	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return dataSourceItems
	}

	for dataSourceName := range blueprint.DataSources.Values {
		dataSourceText := fmt.Sprintf("datasources.%s", dataSourceName)
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: dataSourceText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, dataSourceName)
		dataSourceItems = append(
			dataSourceItems,
			&lsp.CompletionItem{
				Label:    dataSourceText,
				Detail:   &dataSourceDetail,
				Kind:     &dataSourceKind,
				TextEdit: edit,
				SortText: &sortText,
				Data:     map[string]any{"completionType": "dataSource"},
			},
		)
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

	for valueName := range blueprint.Values.Values {
		valueText := fmt.Sprintf("values.%s", valueName)
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: valueText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, valueName)
		valueItems = append(
			valueItems,
			&lsp.CompletionItem{
				Label:    valueText,
				Detail:   &valueDetail,
				Kind:     &valueKind,
				TextEdit: edit,
				SortText: &sortText,
				Data:     map[string]any{"completionType": "value"},
			},
		)
	}

	return valueItems
}

func (s *CompletionService) getChildCompletionItems(
	position *lsp.Position,
	blueprint *schema.Blueprint,
	sortPrefix string,
) []*lsp.CompletionItem {
	childDetail := "Child"
	childKind := lsp.CompletionItemKindValue

	childItems := []*lsp.CompletionItem{}

	if blueprint.Include == nil || len(blueprint.Include.Values) == 0 {
		return childItems
	}

	for childName := range blueprint.Include.Values {
		childText := fmt.Sprintf("children.%s", childName)
		insertRange := getItemInsertRange(position)
		edit := lsp.TextEdit{
			NewText: childText,
			Range:   insertRange,
		}
		sortText := fmt.Sprintf("%s%s", sortPrefix, childName)
		childItems = append(
			childItems,
			&lsp.CompletionItem{
				Label:    childText,
				Detail:   &childDetail,
				Kind:     &childKind,
				TextEdit: edit,
				SortText: &sortText,
				Data:     map[string]any{"completionType": "child"},
			},
		)
	}

	return childItems
}

func getItemInsertRange(
	position *lsp.Position,
) *lsp.Range {

	return &lsp.Range{
		Start: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
		End: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
	}
}

// getItemInsertRangeWithPrefix returns a range that includes characters already typed.
// This allows the completion to replace what the user has typed so far.
func getItemInsertRangeWithPrefix(
	position *lsp.Position,
	prefixLen int,
) *lsp.Range {
	startChar := position.Character - lsp.UInteger(prefixLen)

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

func getOperatorInsertRange(
	position *lsp.Position,
	insertText string,
	isPrecededByOperator bool,
	operatorElementPosition *source.Position,
) *lsp.Range {
	charCount := utf8.RuneCountInString(insertText)

	// If cursor is right after "operator:" field, insert at cursor position
	if isPrecededByOperator {
		return &lsp.Range{
			Start: lsp.Position{
				Line:      position.Line,
				Character: position.Character,
			},
			End: lsp.Position{
				Line:      position.Line,
				Character: position.Character + lsp.UInteger(charCount),
			},
		}
	}

	// Otherwise, replace from the operator element's start position
	start := lsp.Position{
		Line:      lsp.UInteger(operatorElementPosition.Line) - 1,
		Character: lsp.UInteger(operatorElementPosition.Column) - 1,
	}

	return &lsp.Range{
		Start: start,
		End: lsp.Position{
			Line:      start.Line,
			Character: start.Character + lsp.UInteger(charCount),
		},
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
