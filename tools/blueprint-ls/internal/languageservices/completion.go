package languageservices

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
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

var (
	// Pattern for matching "type:" before the cursor position
	// in determining the type of completion items to provide.
	typePattern = regexp.MustCompile(`type:\s*$`)

	// Pattern for matching "field:" before the cursor position
	// in determining the type of completion items to provide.
	fieldPattern = regexp.MustCompile(`field:\s*$`)

	// Pattern for matching "operator:" before the cursor position
	// in determining the type of completion items to provide.
	operatorPattern = regexp.MustCompile(`operator:\s*$`)

	// Pattern for matching "variables." before the cursor position
	// in determining the type of completion items to provide.
	variableRefPattern = regexp.MustCompile(`variables\.$`)

	// Pattern for matching "resources." before the cursor position
	// in determining the type of completion items to provide.
	resourceRefPattern = regexp.MustCompile(`resources\.$`)

	// Pattern for matching resource property references before the
	// cursor position in determining the type of completion items to provide.
	resourcePropertyPattern = regexp.MustCompile(`resources\.([A-Za-z0-9_-]|"|\.|\[|\])+\.$`)

	// Pattern for matching resource property references without the "resources" namespace
	// before the cursor position in determining the type of completion items to provide.
	resourceWithoutNamespacePropPattern = regexp.MustCompile(`([A-Za-z0-9_-]|"|\.|\[|\])+\.$`)

	// Pattern for matching "datasources." before the cursor position
	// in determining the type of completion items to provide.
	dataSourceRefPattern = regexp.MustCompile(`datasources\.$`)

	// Pattern for matching data source property references before the
	// cursor position in determining the type of completion items to provide.
	dataSourcePropertyPattern = regexp.MustCompile(`datasources\.(([A-Za-z0-9_-]|"|\.|\[|\])+)\.$`)

	// Pattern for matching "values." before the cursor position
	// in determining the type of completion items to provide.
	valueRefPattern = regexp.MustCompile(`values\.$`)

	// Pattern for matching "children." before the cursor position
	// in determining the type of completion items to provide.
	childRefPattern = regexp.MustCompile(`children\.$`)

	// Pattern for matching "elem." before the cursor position
	// in determining the type of completion items to provide.
	elemRefPattern = regexp.MustCompile(`elem\.$`)

	// Pattern for matching "${" before the cursor position
	// in determining the type of completion items to provide.
	subOpenPattern = regexp.MustCompile(`\${$`)

	// Pattern for matching for a cursor position inside a string substitution.
	inSubPattern = regexp.MustCompile(`\${[^\}]*$`)
)

var (
	// String path types in a tree node path that may contain "${" indicating
	// the intention of adding a substitution.
	stringPathTypes = []string{
		"scalar",
		"string",
	}
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

// GetCompletionItems returns completion items for a given position in a document.
func (s *CompletionService) GetCompletionItems(
	ctx *common.LSPContext,
	content string,
	tree *schema.TreeNode,
	blueprint *schema.Blueprint,
	params *lsp.TextDocumentPositionParams,
) ([]*lsp.CompletionItem, error) {

	accuratePosition := true
	// The last element in the collected list is the element with the shortest
	// range that contains the position.
	collected := []*schema.TreeNode{}
	collectElementsAtPosition(tree, params.Position, s.logger, &collected, CompletionColumnLeeway)

	// If no elements are collected other than the root node,
	// collect elements on the line of the cursor position.
	if len(collected) == 1 {
		accuratePosition = false
		collected = []*schema.TreeNode{}
		collectElementsOnLine(tree, params.Position, s.logger, &collected)
	}

	return s.getCompletionItems(
		ctx,
		content,
		blueprint,
		&params.Position,
		collected,
		accuratePosition,
	)
}

func (s *CompletionService) getCompletionItems(
	ctx *common.LSPContext,
	content string,
	blueprint *schema.Blueprint,
	position *lsp.Position,
	collected []*schema.TreeNode,
	accuratePosition bool,
) ([]*lsp.CompletionItem, error) {

	if len(collected) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	// Work backwards through the collected elements to find the first element
	// of a type that supports completion items.
	var node *schema.TreeNode
	var elementType string
	i := len(collected) - 1
	for node == nil && i >= 0 {
		pathParts := strings.Split(collected[i].Path, "/")
		node, elementType = s.matchCompletionElement(
			collected,
			i,
			pathParts,
			content,
			position,
			accuratePosition,
		)
		i -= 1
	}

	switch elementType {
	case "resourceType":
		return s.getResourceTypeCompletionItems(ctx)
	case "dataSourceType":
		return s.getDataSourceTypeCompletionItems(ctx)
	case "variableType":
		return s.getVariableTypeCompletionItems(ctx)
	case "valueType":
		return s.getValueTypeCompletionItems()
	case "dataSourceFieldType":
		return s.getDataSourceFieldTypeCompletionItems()
	case "dataSourceFilterField":
		return s.getDataSourceFilterFieldCompletionItems(ctx, node, blueprint)
	case "dataSourceFilterOperator":
		return s.getDataSourceFilterOperatorCompletionItems(position, content, node)
	case "exportType":
		return s.getExportTypeCompletionItems()
	case "stringSubVariableRef":
		return s.getStringSubVariableCompletionItems(position, blueprint)
	case "stringSubResourceProperty":
		return s.getStringSubResourcePropCompletionItems(ctx, position, content, blueprint, node)
	case "stringSubDataSourceRef":
		return s.getStringSubDataSourceCompletionItems(position, blueprint)
	case "stringSubDataSourceProperty":
		return s.getStringSubDataSourcePropCompletionItems(position, content, blueprint, node)
	case "stringSubValueRef":
		return s.getStringSubValueCompletionItems(position, blueprint)
	case "stringSubChildRef":
		return s.getStringSubChildCompletionItems(position, blueprint)
	case "stringSubResourceRef":
		return s.getStringSubResourceCompletionItems(position, blueprint)
	case "stringSub":
		return s.getStringSubCompletionItems(ctx, position, blueprint)
	default:
		return []*lsp.CompletionItem{}, nil
	}
}

func (s *CompletionService) matchCompletionElement(
	collected []*schema.TreeNode,
	index int,
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
	accuratePosition bool,
) (*schema.TreeNode, string) {
	if !accuratePosition {
		return s.matchCompletionElementOnLine(collected, index, pathParts, sourceContent, position)
	}

	if s.isResourceType(pathParts, sourceContent, position) {
		return collected[index], "resourceType"
	}

	if s.isDataSourceType(pathParts, sourceContent, position) {
		return collected[index], "dataSourceType"
	}

	if s.isVariableType(pathParts, sourceContent, position) {
		return collected[index], "variableType"
	}

	if s.isValueType(pathParts, sourceContent, position) {
		return collected[index], "valueType"
	}

	// Check potential new fields first to avoid getting the wrong completion type
	// in data source filter fields and operators.
	if s.isNewDataSourceFilterField(pathParts, sourceContent, position) {
		return collected[index], "dataSourceFilterField"
	}

	if s.isNewDataSourceFilterOperator(pathParts, sourceContent, position) {
		return collected[index], "dataSourceFilterOperator"
	}

	if s.isDataSourceFieldType(pathParts, sourceContent, position) {
		return collected[index], "dataSourceFieldType"
	}

	if s.isDataSourceFilterField(pathParts) {
		return collected[index], "dataSourceFilterField"
	}

	if s.isDataSourceFilterOperator(pathParts) {
		return collected[index], "dataSourceFilterOperator"
	}

	if s.isExportType(pathParts, sourceContent, position) {
		return collected[index], "exportType"
	}

	if elementType, isStringSubElem := s.checkStringSubElement(
		pathParts,
		sourceContent,
		position,
		collected[index],
	); isStringSubElem {
		return collected[index], elementType
	}

	return nil, ""
}

func (s *CompletionService) checkStringSubElement(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
	node *schema.TreeNode,
) (string, bool) {

	if s.isStringSubVariable(pathParts, sourceContent, position) {
		return "stringSubVariableRef", true
	}

	if s.isStringSubDataSource(pathParts, sourceContent, position) {
		return "stringSubDataSourceRef", true
	}

	if s.isStringSubDataSourceProperty(pathParts, sourceContent, position) {
		return "stringSubDataSourceProperty", true
	}

	if s.isStringSubValue(pathParts, sourceContent, position) {
		return "stringSubValueRef", true
	}

	if s.isStringSubChild(pathParts, sourceContent, position) {
		return "stringSubChildRef", true
	}

	if s.isStringSubElem(pathParts, sourceContent, position) {
		return "elemRef", true
	}

	// Resource and resource property references should be checked after all other possible references
	// as a resource reference can match on an identifier pattern which would lead to incorrect completion
	// suggestions if checked before other references.
	if s.isStringSubResource(pathParts, sourceContent, position) {
		return "stringSubResourceRef", true
	}

	if s.isStringSubResourceProperty(pathParts, sourceContent, position, node) {
		return "stringSubResourceProperty", true
	}

	if s.isStringSub(pathParts, sourceContent, position) {
		return "stringSub", true
	}

	return "", false
}

func (s *CompletionService) isResourceType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingResourceType := len(pathParts) == 4 &&
		pathParts[1] == "resources" &&
		pathParts[3] == "type"

	isInNewResourceType := len(pathParts) == 3 &&
		pathParts[1] == "resources" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingResourceType || isInNewResourceType
}

func (s *CompletionService) isDataSourceType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingDataSourceType := len(pathParts) == 4 &&
		pathParts[1] == "datasources" &&
		pathParts[3] == "type"

	isInNewDataSourceType := len(pathParts) == 3 &&
		pathParts[1] == "datasources" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingDataSourceType || isInNewDataSourceType
}

func (s *CompletionService) isVariableType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingVarType := len(pathParts) == 4 &&
		pathParts[1] == "variables" &&
		pathParts[3] == "type"

	isInNewVarType := len(pathParts) == 3 &&
		pathParts[1] == "variables" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingVarType || isInNewVarType
}

func (s *CompletionService) isValueType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingValType := len(pathParts) == 4 &&
		pathParts[1] == "values" &&
		pathParts[3] == "type"

	isInNewValType := len(pathParts) == 3 &&
		pathParts[1] == "values" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingValType || isInNewValType
}

func (s *CompletionService) isDataSourceFieldType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingExportType := len(pathParts) == 5 &&
		pathParts[1] == "datasources" &&
		pathParts[4] == "type"

	isInNewExportType := len(pathParts) == 4 &&
		pathParts[1] == "datasources" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingExportType || isInNewExportType
}

func (s *CompletionService) isNewDataSourceFilterField(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return len(pathParts) >= 5 &&
		pathParts[1] == "datasources" &&
		pathParts[3] == "filters" &&
		s.isPrecededBy(position, fieldPattern, sourceContent)
}

func (s *CompletionService) isDataSourceFilterField(
	pathParts []string,
) bool {
	return len(pathParts) == 7 &&
		pathParts[1] == "datasources" &&
		pathParts[3] == "filters" &&
		pathParts[5] == "filter" &&
		pathParts[6] == "field"
}

func (s *CompletionService) isNewDataSourceFilterOperator(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return len(pathParts) >= 6 &&
		pathParts[1] == "datasources" &&
		pathParts[3] == "filters" &&
		pathParts[5] == "filter" &&
		s.isPrecededBy(position, operatorPattern, sourceContent)
}

func (s *CompletionService) isDataSourceFilterOperator(
	pathParts []string,
) bool {
	return len(pathParts) == 7 &&
		pathParts[1] == "datasources" &&
		pathParts[3] == "filters" &&
		pathParts[5] == "filter" &&
		pathParts[6] == "operator"
}

func (s *CompletionService) isExportType(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	isInExistingExportType := len(pathParts) == 4 &&
		pathParts[1] == "exports" &&
		pathParts[3] == "type"

	isInNewExportType := len(pathParts) == 3 &&
		pathParts[1] == "exports" &&
		s.isPrecededBy(position, typePattern, sourceContent)

	return isInExistingExportType || isInNewExportType
}

func (s *CompletionService) isStringSubVariable(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, variableRefPattern, sourceContent)
}

func (s *CompletionService) isStringSubResource(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, resourceRefPattern, sourceContent)
}

func (s *CompletionService) isStringSubResourceProperty(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
	node *schema.TreeNode,
) bool {
	namespacedResourcePropMatch := slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, resourcePropertyPattern, sourceContent)

	_, isResourceProp := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	resourcePropMatchWithoutNamespace := slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, resourceWithoutNamespacePropPattern, sourceContent) &&
		isResourceProp

	return namespacedResourcePropMatch || resourcePropMatchWithoutNamespace
}

func (s *CompletionService) isStringSubDataSource(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, dataSourceRefPattern, sourceContent)
}

func (s *CompletionService) isStringSubDataSourceProperty(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, dataSourcePropertyPattern, sourceContent)
}

func (s *CompletionService) isStringSubValue(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, valueRefPattern, sourceContent)
}

func (s *CompletionService) isStringSubChild(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, childRefPattern, sourceContent)
}

func (s *CompletionService) isStringSubElem(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, elemRefPattern, sourceContent)
}

func (s *CompletionService) isStringSub(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return (slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, inSubPattern, sourceContent)) ||
		(slices.Contains(stringPathTypes, pathParts[len(pathParts)-1]) &&
			s.isPrecededBy(position, subOpenPattern, sourceContent))
}

func (s *CompletionService) matchCompletionElementOnLine(
	collected []*schema.TreeNode,
	index int,
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) (*schema.TreeNode, string) {
	if s.isStringSubDataSourcePropertyOnLine(pathParts, sourceContent, position) {
		return collected[index], "stringSubDataSourceProperty"
	}

	return nil, ""
}

func (s *CompletionService) isStringSubDataSourcePropertyOnLine(
	pathParts []string,
	sourceContent string,
	position *lsp.Position,
) bool {
	return slices.Contains(pathParts, "stringSubs") &&
		s.isPrecededBy(position, dataSourcePropertyPattern, sourceContent)
}

func (s *CompletionService) isPrecededBy(
	position *lsp.Position,
	expectedBeforePattern *regexp.Regexp,
	sourceContent string,
) bool {
	if position.Line == 0 {
		return false
	}

	index := position.IndexIn(sourceContent, s.state.GetPositionEncodingKind())
	sourceContentBefore := sourceContent[:index]
	return expectedBeforePattern.Match([]byte(sourceContentBefore))
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
			Data:       map[string]interface{}{"completionType": "resourceType"},
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
			Data:       map[string]interface{}{"completionType": "dataSourceType"},
		})
	}

	return completionItems, nil
}

func (s *CompletionService) getDataSourceFilterFieldCompletionItems(
	ctx *common.LSPContext,
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) ([]*lsp.CompletionItem, error) {

	if node == nil || blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceName := getDataSourceName(node)
	if dataSourceName == "" {
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
			Data:       map[string]interface{}{"completionType": "dataSourceFilterField"},
		})
	}

	return completionItems, nil
}

func getDataSourceName(node *schema.TreeNode) string {
	if node == nil {
		return ""
	}

	pathParts := strings.Split(node.Path, "/")
	if len(pathParts) > 2 && pathParts[1] == "datasources" {
		return pathParts[2]
	}

	return ""
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
				Data:       map[string]interface{}{"completionType": "variableType"},
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
			Data:       map[string]interface{}{"completionType": "variableType"},
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
				Data:       map[string]interface{}{"completionType": "valueType"},
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
				Data:       map[string]interface{}{"completionType": "dataSourceFieldType"},
			},
		)
	}

	return typeItems, nil
}

func (s *CompletionService) getDataSourceFilterOperatorCompletionItems(
	position *lsp.Position,
	sourceContent string,
	operatorElementNode *schema.TreeNode,
) ([]*lsp.CompletionItem, error) {
	filterOperatorDetail := "Data source filter operator"
	enumKind := lsp.CompletionItemKindEnum
	positionEncodingKind := s.state.GetPositionEncodingKind()

	operatorElementPosition := operatorElementNode.Range.Start

	filterOpItems := []*lsp.CompletionItem{}
	for _, filterOperator := range schema.DataSourceFilterOperators {
		filterOperatorStr := fmt.Sprintf("\"%s\"", string(filterOperator))
		// Due to filters containing white space characters, which are trigger characters
		// for completion, we need to do a custom edit instead of the default insertText
		// approach.
		edit := lsp.TextEdit{
			NewText: filterOperatorStr,
			Range: getOperatorInsertRange(
				position,
				filterOperatorStr,
				sourceContent,
				positionEncodingKind,
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
				Data:       map[string]interface{}{"completionType": "exportType"},
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
				Data:     map[string]interface{}{"completionType": "variable"},
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
				Data:     map[string]interface{}{"completionType": "resource"},
			},
		)
	}

	return resourceItems, nil
}

func (s *CompletionService) getStringSubResourcePropCompletionItems(
	ctx *common.LSPContext,
	position *lsp.Position,
	content string,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
) ([]*lsp.CompletionItem, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	resourceItems := []*lsp.CompletionItem{}
	s.logger.Debug("Node path", zap.String("path", node.Path))
	s.logger.Debug("Node schema element", zap.Any("range", node.SchemaElement))

	resourceProp, isResourceProp := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	if !isResourceProp {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	index := position.IndexIn(content, s.state.GetPositionEncodingKind())
	sourceContentBefore := content[:index]

	if strings.HasSuffix(sourceContentBefore, fmt.Sprintf(".%s.", resourceProp.ResourceName)) {
		return getResourceTopLevelPropCompletionItems(position), nil
	}

	if strings.HasSuffix(sourceContentBefore, ".spec.") ||
		(len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "spec") {
		return s.getResourceSpecPropCompletionItems(ctx, position, blueprint, resourceProp)
	}

	if strings.HasSuffix(sourceContentBefore, ".metadata.") ||
		(len(resourceProp.Path) >= 1 && resourceProp.Path[0].FieldName == "metadata") {
		return getResourceMetadataPropCompletionItems(position), nil
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
		return nil, err
	}

	if specDefOutput.SpecDefinition == nil || specDefOutput.SpecDefinition.Schema == nil {
		return []*lsp.CompletionItem{}, nil
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

	return resourceDefAttributesSchemaCompletionItems(
		currentSchema.Attributes,
		position,
		"Resource spec property",
	), nil
}

func resourceDefAttributesSchemaCompletionItems(
	attributes map[string]*provider.ResourceDefinitionsSchema,
	position *lsp.Position,
	attrDetail string,
) []*lsp.CompletionItem {
	completionItems := []*lsp.CompletionItem{}
	for attrName := range attributes {
		fieldKind := lsp.CompletionItemKindField

		edit := lsp.TextEdit{
			NewText: attrName,
			Range:   getItemInsertRange(position),
		}

		completionItems = append(
			completionItems,
			&lsp.CompletionItem{
				Label:    attrName,
				Detail:   &attrDetail,
				Kind:     &fieldKind,
				TextEdit: edit,
				Data: map[string]interface{}{
					"completionType": "resourceProperty",
				},
			},
		)
	}

	return completionItems
}

func getResourceTopLevelPropCompletionItems(position *lsp.Position) []*lsp.CompletionItem {
	detail := "Resource property"

	insertRange := getItemInsertRange(position)

	metadataText := "metadata"
	metadataEdit := lsp.TextEdit{
		NewText: metadataText,
		Range:   insertRange,
	}

	specText := "spec"
	specEdit := lsp.TextEdit{
		NewText: specText,
		Range:   insertRange,
	}

	stateText := "state"
	stateEdit := lsp.TextEdit{
		NewText: stateText,
		Range:   insertRange,
	}

	fieldKind := lsp.CompletionItemKindField
	return []*lsp.CompletionItem{
		{
			Label:    metadataText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: metadataEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
		{
			Label:    specText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: specEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
		{
			Label:    stateText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: stateEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
	}
}

func getResourceMetadataPropCompletionItems(
	position *lsp.Position,
) []*lsp.CompletionItem {
	detail := "Resource metadata property"

	insertRange := getItemInsertRange(position)

	displayNameText := "displayName"
	displayNameEdit := lsp.TextEdit{
		NewText: displayNameText,
		Range:   insertRange,
	}

	labelsText := "labels"
	labelsEdit := lsp.TextEdit{
		NewText: labelsText,
		Range:   insertRange,
	}

	annotationsText := "annotations"
	annotationsEdit := lsp.TextEdit{
		NewText: annotationsText,
		Range:   insertRange,
	}

	customText := "custom"
	customEdit := lsp.TextEdit{
		NewText: customText,
		Range:   insertRange,
	}

	fieldKind := lsp.CompletionItemKindField
	return []*lsp.CompletionItem{
		{
			Label:    displayNameText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: displayNameEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
		{
			Label:    labelsText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: labelsEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
		{
			Label:    annotationsText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: annotationsEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
		{
			Label:    customText,
			Detail:   &detail,
			Kind:     &fieldKind,
			TextEdit: customEdit,
			Data: map[string]interface{}{
				"completionType": "resourceProperty",
			},
		},
	}
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
				Data:     map[string]interface{}{"completionType": "dataSource"},
			},
		)
	}

	return dataSourceItems, nil
}

func (s *CompletionService) getStringSubDataSourcePropCompletionItems(
	position *lsp.Position,
	content string,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
) ([]*lsp.CompletionItem, error) {
	detail := "Data source exported field"
	fieldKind := lsp.CompletionItemKindField

	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []*lsp.CompletionItem{}, nil
	}

	dataSourceItems := []*lsp.CompletionItem{}

	dataSourceName, isDataSourceProp := s.deriveDataSourceName(node, position, content)
	if !isDataSourceProp {
		return dataSourceItems, nil
	}

	dataSource := getDataSource(blueprint, dataSourceName)
	if dataSource == nil || dataSource.Exports == nil {
		return dataSourceItems, nil
	}

	for exportName := range dataSource.Exports.Values {
		insertRange := getItemInsertRange(
			position,
		)
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

func (s *CompletionService) deriveDataSourceName(node *schema.TreeNode, position *lsp.Position, content string) (string, bool) {
	if node == nil {
		return "", false
	}

	dsProp, isDSProp := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
	if isDSProp {
		return dsProp.DataSourceName, true
	}

	// If the node path does not contain the data source name, we need to derive it
	// from the content.
	index := position.IndexIn(content, s.state.GetPositionEncodingKind())
	sourceContentBefore := content[:index]
	matches := dataSourcePropertyPattern.FindStringSubmatch(sourceContentBefore)
	if len(matches) >= 2 {
		// The name is the second capture group.
		return matches[1], true
	}

	return "", false
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
				Data:     map[string]interface{}{"completionType": "value"},
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
				Data:     map[string]interface{}{"completionType": "child"},
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
	resourceKind := lsp.CompletionItemKindValue

	resourceItems := []*lsp.CompletionItem{}

	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return resourceItems
	}

	for resourceName := range blueprint.Resources.Values {
		resourceText := fmt.Sprintf("resources.%s", resourceName)
		insertRange := getItemInsertRange(position)
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
				Data:     map[string]interface{}{"completionType": "resource"},
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
				Data:     map[string]interface{}{"completionType": "variable"},
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
				Data:             map[string]interface{}{"completionType": "function"},
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
				Data:     map[string]interface{}{"completionType": "dataSource"},
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
				Data:     map[string]interface{}{"completionType": "value"},
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
				Data:     map[string]interface{}{"completionType": "child"},
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

func getOperatorInsertRange(
	position *lsp.Position,
	insertText string,
	sourceContent string,
	positionEncodingKind lsp.PositionEncodingKind,
	operatorElementPosition *source.Position,
) *lsp.Range {
	index := position.IndexIn(sourceContent, positionEncodingKind)
	sourceContentBefore := sourceContent[:index]
	charCount := utf8.RuneCountInString(insertText)

	if operatorPattern.Match([]byte(sourceContentBefore)) {
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
