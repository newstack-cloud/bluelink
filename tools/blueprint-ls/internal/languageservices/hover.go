package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/helpinfo"
	common "github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// HoverService is a service that provides functionality for hover messages.
type HoverService struct {
	funcRegistry       provider.FunctionRegistry
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
	signatureService   *SignatureService
	logger             *zap.Logger
}

// NewHoverService creates a new service for hover messages.
func NewHoverService(
	funcRegistry provider.FunctionRegistry,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	signatureService *SignatureService,
	logger *zap.Logger,
) *HoverService {
	return &HoverService{
		funcRegistry,
		resourceRegistry,
		dataSourceRegistry,
		signatureService,
		logger,
	}
}

// HoverContent represents the content for a hover message.
type HoverContent struct {
	Value string
	Range *lsp.Range
}

// GetHoverContent returns the hover content for the given position in the
// source document.
func (s *HoverService) GetHoverContent(
	ctx *common.LSPContext,
	tree *schema.TreeNode,
	blueprint *schema.Blueprint,
	params *lsp.TextDocumentPositionParams,
) (*HoverContent, error) {

	// The last element in the collected list is the element with the shortest
	// range that contains the position.
	collected := []*schema.TreeNode{}
	collectElementsAtPosition(tree, params.Position, s.logger, &collected, 0 /* columnLeeway */)

	return s.getHoverElementContent(
		ctx,
		blueprint,
		collected,
	)
}

func (s *HoverService) getHoverElementContent(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	collected []*schema.TreeNode,
) (*HoverContent, error) {
	hoverCtx := docmodel.DetermineHoverContext(collected)
	if hoverCtx == nil {
		return &HoverContent{}, nil
	}

	return s.getHoverContentByKind(ctx, blueprint, hoverCtx)
}

func (s *HoverService) getHoverContentByKind(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	hoverCtx *docmodel.HoverContext,
) (*HoverContent, error) {
	switch hoverCtx.ElementKind {
	case docmodel.SchemaElementFunctionCall:
		return s.getFunctionCallHoverContent(ctx, hoverCtx.TreeNode)
	case docmodel.SchemaElementVariableRef:
		return getVarRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementValueRef:
		return getValRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementChildRef:
		return getChildRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementResourceRef:
		return s.getResourceRefHoverContent(ctx, hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementDataSourceRef:
		return getDataSourceRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementElemRef:
		return getElemRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementElemIndexRef:
		return getElemIndexRefHoverContent(hoverCtx.TreeNode, blueprint)
	case docmodel.SchemaElementResourceType:
		return s.getResourceTypeHoverContent(ctx, hoverCtx.TreeNode)
	case docmodel.SchemaElementDataSourceType:
		return s.getDataSourceTypeHoverContent(ctx, hoverCtx.TreeNode)
	default:
		return &HoverContent{}, nil
	}
}

func (s *HoverService) getFunctionCallHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
) (*HoverContent, error) {

	subFunc, isSubFunc := node.SchemaElement.(*substitutions.SubstitutionFunctionExpr)
	if !isSubFunc {
		return &HoverContent{}, nil
	}

	signatureInfo, err := s.signatureService.SignatureInfoFromFunction(subFunc, ctx)
	if err != nil {
		return &HoverContent{}, err
	}

	content := helpinfo.CustomRenderSignatures(signatureInfo)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getResourceRefHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	resRef, isResRef := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	if !isResRef {
		return &HoverContent{}, nil
	}

	resource := getResource(blueprint, resRef.ResourceName)
	if resource == nil || resource.Type == nil {
		return &HoverContent{}, nil
	}

	if len(resRef.Path) == 0 {
		return getBasicResourceHoverContent(node.Label, resource), nil
	}

	if len(resRef.Path) > 1 && resRef.Path[0].FieldName == "spec" {
		s.logger.Debug(
			"Fetching spec definition for hover content",
			zap.String("resourceType", resource.Type.Value),
		)
		specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
			ctx.Context,
			resource.Type.Value,
			&provider.ResourceGetSpecDefinitionInput{},
		)
		if err != nil {
			return &HoverContent{}, nil
		}

		return getResourceWithSpecHoverContent(
			node,
			resource,
			resRef,
			specDefOutput.SpecDefinition,
		)
	}

	return &HoverContent{}, nil
}

func (s *HoverService) getResourceTypeHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
) (*HoverContent, error) {
	resType, isResType := node.SchemaElement.(*schema.ResourceTypeWrapper)
	if !isResType || resType == nil {
		return &HoverContent{}, nil
	}

	s.logger.Debug(
		"Fetching resource type definition for hover content",
		zap.String("resourceType", resType.Value),
	)
	descriptionOutput, err := s.resourceRegistry.GetTypeDescription(
		ctx.Context,
		resType.Value,
		&provider.ResourceGetTypeDescriptionInput{},
	)
	if err != nil {
		s.logger.Debug(
			"Failed to fetch type description for resource type hover content",
			zap.Error(err),
		)
		return &HoverContent{}, nil
	}

	description := descriptionOutput.MarkdownDescription
	if description == "" {
		description = descriptionOutput.PlainTextDescription
	}

	return &HoverContent{
		Value: description,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getDataSourceTypeHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
) (*HoverContent, error) {
	dataSourceType, isDataSourceType := node.SchemaElement.(*schema.DataSourceTypeWrapper)
	if !isDataSourceType || dataSourceType == nil {
		return &HoverContent{}, nil
	}

	s.logger.Debug(
		"Fetching data source type definition for hover content",
		zap.String("dataSourceType", dataSourceType.Value),
	)
	descriptionOutput, err := s.dataSourceRegistry.GetTypeDescription(
		ctx.Context,
		dataSourceType.Value,
		&provider.DataSourceGetTypeDescriptionInput{},
	)
	if err != nil {
		s.logger.Debug(
			"Failed to fetch type description for data source type hover content",
			zap.Error(err),
		)
		return &HoverContent{}, nil
	}

	description := descriptionOutput.MarkdownDescription
	if description == "" {
		description = descriptionOutput.PlainTextDescription
	}

	return &HoverContent{
		Value: description,
		Range: rangeToLSPRange(node.Range),
	}, nil
}


func getVarRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	varRef, isVarRef := node.SchemaElement.(*substitutions.SubstitutionVariable)
	if !isVarRef {
		return &HoverContent{}, nil
	}

	variable := getVariable(blueprint, varRef.VariableName)
	if variable == nil {
		return &HoverContent{}, nil
	}

	content := helpinfo.RenderVariableInfo(node.Label, variable)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getValRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	valRef, isValRef := node.SchemaElement.(*substitutions.SubstitutionValueReference)
	if !isValRef {
		return &HoverContent{}, nil
	}

	value := getValue(blueprint, valRef.ValueName)
	if value == nil {
		return &HoverContent{}, nil
	}

	content := helpinfo.RenderValueInfo(node.Label, value)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getChildRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	childRef, isChildRef := node.SchemaElement.(*substitutions.SubstitutionChild)
	if !isChildRef {
		return &HoverContent{}, nil
	}

	child := getChild(blueprint, childRef.ChildName)
	if child == nil {
		return &HoverContent{}, nil
	}

	content := helpinfo.RenderChildInfo(node.Label, child)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getBasicResourceHoverContent(
	resourceName string,
	resource *schema.Resource,
) *HoverContent {
	content := helpinfo.RenderBasicResourceInfo(resourceName, resource)

	return &HoverContent{
		Value: content,
		Range: nil,
	}
}

func getResourceWithSpecHoverContent(
	node *schema.TreeNode,
	resource *schema.Resource,
	resRef *substitutions.SubstitutionResourceProperty,
	specDef *provider.ResourceSpecDefinition,
) (*HoverContent, error) {
	if specDef == nil {
		return &HoverContent{}, nil
	}

	specFieldSchema, err := findResourceFieldSchema(specDef.Schema, resRef.Path[1:])
	if err != nil {
		return &HoverContent{}, err
	}

	content := helpinfo.RenderResourceDefinitionFieldInfo(
		node.Label,
		resource,
		resRef,
		specFieldSchema,
	)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	dataSourceRef, isDataSourceRef := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
	if !isDataSourceRef {
		return &HoverContent{}, nil
	}

	dataSource := getDataSource(blueprint, dataSourceRef.DataSourceName)
	if dataSource == nil {
		return &HoverContent{}, nil
	}

	dataSourceField := getDataSourceField(dataSource, dataSourceRef.FieldName)
	if dataSourceField == nil {
		return &HoverContent{
			Value: helpinfo.RenderBasicDataSourceInfo(node.Label, dataSource),
			Range: rangeToLSPRange(node.Range),
		}, nil
	}

	content := helpinfo.RenderDataSourceFieldInfo(node.Label, dataSource, dataSourceRef, dataSourceField)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getElemRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	elemRef, isElemRef := node.SchemaElement.(*substitutions.SubstitutionElemReference)
	if !isElemRef {
		return &HoverContent{}, nil
	}

	resourceName := extractResourceNameFromElemRef(node.Path)
	resource := getResource(blueprint, resourceName)
	if resource == nil {
		return &HoverContent{}, nil
	}

	content := helpinfo.RenderElemRefInfo(resourceName, resource, elemRef)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getElemIndexRefHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {

	resourceName := extractResourceNameFromElemRef(node.Path)
	resource := getResource(blueprint, resourceName)
	if resource == nil {
		return &HoverContent{}, nil
	}

	content := helpinfo.RenderElemIndexRefInfo(resourceName, resource)

	return &HoverContent{
		Value: content,
		Range: rangeToLSPRange(node.Range),
	}, nil
}

func getVariable(blueprint *schema.Blueprint, name string) *schema.Variable {
	if blueprint.Variables == nil || blueprint.Variables.Values == nil {
		return nil
	}

	variable, hasVariable := blueprint.Variables.Values[name]
	if !hasVariable {
		return nil
	}

	return variable
}

func getValue(blueprint *schema.Blueprint, name string) *schema.Value {
	if blueprint.Values == nil || blueprint.Values.Values == nil {
		return nil
	}

	value, hasValue := blueprint.Values.Values[name]
	if !hasValue {
		return nil
	}

	return value
}

func getChild(blueprint *schema.Blueprint, name string) *schema.Include {
	if blueprint.Include == nil || blueprint.Include.Values == nil {
		return nil
	}

	child, hasChild := blueprint.Include.Values[name]
	if !hasChild {
		return nil
	}

	return child
}

func getResource(blueprint *schema.Blueprint, name string) *schema.Resource {
	if blueprint.Resources == nil || blueprint.Resources.Values == nil {
		return nil
	}

	resource, hasResource := blueprint.Resources.Values[name]
	if !hasResource {
		return nil
	}

	return resource
}

func getDataSource(blueprint *schema.Blueprint, name string) *schema.DataSource {
	if blueprint.DataSources == nil || blueprint.DataSources.Values == nil {
		return nil
	}

	dataSource, hasDataSource := blueprint.DataSources.Values[name]
	if !hasDataSource {
		return nil
	}

	return dataSource
}

func getDataSourceField(dataSource *schema.DataSource, name string) *schema.DataSourceFieldExport {
	if dataSource.Exports == nil || dataSource.Exports.Values == nil {
		return nil
	}

	field, hasField := dataSource.Exports.Values[name]
	if !hasField {
		return nil
	}

	return field
}

func findResourceFieldSchema(
	defSchema *provider.ResourceDefinitionsSchema,
	path []*substitutions.SubstitutionPathItem,
) (*provider.ResourceDefinitionsSchema, error) {
	if len(path) == 0 {
		return nil, nil
	}

	currentSchema := defSchema
	i := 0
	for currentSchema != nil && i < len(path) {
		pathItem := path[i]

		objectFieldSchema := checkResourceObjectFieldSchema(currentSchema, pathItem)
		if objectFieldSchema != nil {
			currentSchema = objectFieldSchema
		}

		mapFieldSchema := checkResourceMapFieldSchema(currentSchema, pathItem)
		if mapFieldSchema != nil {
			currentSchema = mapFieldSchema
		}

		arrayItemSchema := checkResourceArrayItemSchema(currentSchema, pathItem)
		if arrayItemSchema != nil {
			currentSchema = arrayItemSchema
		}

		if objectFieldSchema == nil && mapFieldSchema == nil && arrayItemSchema == nil {
			// Avoid associating the field with parent schemas,
			// this will create confusing docs/help information that suggests
			// a given field has a type that it does not.
			currentSchema = nil
		}

		i += 1
	}

	return currentSchema, nil
}

func checkResourceObjectFieldSchema(
	schema *provider.ResourceDefinitionsSchema,
	pathItem *substitutions.SubstitutionPathItem,
) *provider.ResourceDefinitionsSchema {

	if pathItem.FieldName != "" &&
		schema.Type == provider.ResourceDefinitionsSchemaTypeObject {
		fieldSchema, hasField := schema.Attributes[pathItem.FieldName]
		if !hasField {
			return nil
		} else {
			return fieldSchema
		}
	}

	return nil
}

func checkResourceMapFieldSchema(
	schema *provider.ResourceDefinitionsSchema,
	pathItem *substitutions.SubstitutionPathItem,
) *provider.ResourceDefinitionsSchema {

	if pathItem.FieldName != "" &&
		schema.Type == provider.ResourceDefinitionsSchemaTypeMap {
		if schema.MapValues == nil {
			return nil
		} else {
			return schema.MapValues
		}
	}

	return nil
}

func checkResourceArrayItemSchema(
	schema *provider.ResourceDefinitionsSchema,
	pathItem *substitutions.SubstitutionPathItem,
) *provider.ResourceDefinitionsSchema {

	if pathItem.ArrayIndex != nil &&
		schema.Type == provider.ResourceDefinitionsSchemaTypeArray {
		if schema.Items == nil {
			return nil
		} else {
			return schema.Items
		}
	}

	return nil
}

func extractResourceNameFromElemRef(
	elemRefPath string,
) string {
	pathParts := strings.Split(elemRefPath, "/")
	if len(pathParts) < 4 || (len(pathParts) > 1 && pathParts[1] != "resources") {
		// "/resources/<resourceName>/.*?(elemRef | elemIndexRef)"
		// must contain at least 4 parts to be a valid elemRef
		// path. "" "resources" "<resourceName>" ... ( "elemRef" | "elemIndexRef" )
		return ""
	}

	return pathParts[2]
}
