package languageservices

import (
	"fmt"
	"os"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// GotoDefinitionService provides go-to-definition functionality for blueprint documents.
type GotoDefinitionService struct {
	state         *State
	childResolver *ChildBlueprintResolver
	logger        *zap.Logger
}

// NewGotoDefinitionService creates a new service for go-to-definition support.
func NewGotoDefinitionService(
	state *State,
	childResolver *ChildBlueprintResolver,
	logger *zap.Logger,
) *GotoDefinitionService {
	return &GotoDefinitionService{
		state:         state,
		childResolver: childResolver,
		logger:        logger,
	}
}

// GetDefinitionsFromContext returns definition links using the DocumentContext.
func (s *GotoDefinitionService) GetDefinitionsFromContext(
	docCtx *docmodel.DocumentContext,
	params *lsp.TextDocumentPositionParams,
) ([]lsp.LocationLink, error) {
	if docCtx == nil || docCtx.SchemaTree == nil || docCtx.Blueprint == nil {
		return []lsp.LocationLink{}, nil
	}

	pos := source.Position{
		Line:   int(params.Position.Line + 1),
		Column: int(params.Position.Character + 1),
	}

	collected := docCtx.CollectSchemaNodesAtPosition(pos, CompletionColumnLeeway)
	if len(collected) == 0 {
		return []lsp.LocationLink{}, nil
	}

	return s.findDefinitionLink(
		params.TextDocument.URI,
		docCtx.Blueprint,
		docCtx.SchemaTree,
		collected,
	)
}

// findDefinitionLink searches the collected nodes for a definition target.
func (s *GotoDefinitionService) findDefinitionLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	tree *schema.TreeNode,
	collected []*schema.TreeNode,
) ([]lsp.LocationLink, error) {
	for i := len(collected) - 1; i >= 0; i-- {
		node := collected[i]
		kind := docmodel.KindFromSchemaElement(node.SchemaElement)

		switch kind {
		case docmodel.SchemaElementResourceRef:
			return s.buildResourceRefLink(docURI, blueprint, node, tree)
		case docmodel.SchemaElementDataSourceRef:
			return s.buildDataSourceRefLink(docURI, blueprint, node, tree)
		case docmodel.SchemaElementVariableRef:
			return s.buildVariableRefLink(docURI, blueprint, node, tree)
		case docmodel.SchemaElementValueRef:
			return s.buildValueRefLink(docURI, blueprint, node, tree)
		case docmodel.SchemaElementChildRef:
			return s.buildChildRefLink(docURI, blueprint, node, tree)
		}
	}

	// Fallback: check for plain-string contexts (export field values,
	// linkSelector.exclude items, dependsOn items).
	return s.findPlainStringDefinitionLink(docURI, blueprint, tree, collected)
}

func (s *GotoDefinitionService) buildResourceRefLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	resourceProp, ok := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		rootNode,
		fmt.Sprintf("/resources/%s", resourceProp.ResourceName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, node.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildDataSourceRefLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	dataSourceProp, ok := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		rootNode,
		fmt.Sprintf("/datasources/%s", dataSourceProp.DataSourceName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, node.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildVariableRefLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Variables == nil || len(blueprint.Variables.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	varProp, ok := node.SchemaElement.(*substitutions.SubstitutionVariable)
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		rootNode,
		fmt.Sprintf("/variables/%s", varProp.VariableName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, node.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildValueRefLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Values == nil || len(blueprint.Values.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	valProp, ok := node.SchemaElement.(*substitutions.SubstitutionValueReference)
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		rootNode,
		fmt.Sprintf("/values/%s", valProp.ValueName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, node.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildChildRefLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Include == nil || len(blueprint.Include.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	childProp, ok := node.SchemaElement.(*substitutions.SubstitutionChild)
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		rootNode,
		fmt.Sprintf("/includes/%s", childProp.ChildName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, node.Range, targetNode.Range), nil
}

// findSchemaNodeByPath searches for a schema node by path string.
func findSchemaNodeByPath(node *schema.TreeNode, targetPath string) *schema.TreeNode {
	if node == nil {
		return nil
	}

	if node.Path == targetPath {
		return node
	}

	for _, child := range node.Children {
		if found := findSchemaNodeByPath(child, targetPath); found != nil {
			return found
		}
	}

	return nil
}

// buildLocationLink creates a LocationLink slice from origin and target ranges.
func buildLocationLink(docURI lsp.URI, originRange, targetRange *source.Range) []lsp.LocationLink {
	origin := rangeToLSPRange(originRange)
	target := rangeToLSPRange(targetRange)
	if target == nil {
		return []lsp.LocationLink{}
	}

	return []lsp.LocationLink{
		{
			OriginSelectionRange: origin,
			TargetURI:            docURI,
			TargetRange:          *target,
			TargetSelectionRange: *target,
		},
	}
}

type plainStringContextType int

const (
	plainStringContextExportField plainStringContextType = iota
	plainStringContextResourceRef
	plainStringContextIncludePath
)

type plainStringContext struct {
	contextType plainStringContextType
	leafNode    *schema.TreeNode
	parentNode  *schema.TreeNode
}

// exportFieldNamespaceTargets maps export field namespace prefixes
// to their corresponding schema tree path prefixes.
var exportFieldNamespaceTargets = map[string]string{
	"resources":   "/resources",
	"datasources": "/datasources",
	"variables":   "/variables",
	"values":      "/values",
	"children":    "/includes",
}

func (s *GotoDefinitionService) findPlainStringDefinitionLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	tree *schema.TreeNode,
	collected []*schema.TreeNode,
) ([]lsp.LocationLink, error) {
	ctx := classifyPlainStringContext(collected)
	if ctx == nil {
		return []lsp.LocationLink{}, nil
	}

	switch ctx.contextType {
	case plainStringContextExportField:
		return s.buildExportFieldDefinitionLink(docURI, ctx, tree)
	case plainStringContextResourceRef:
		return s.buildResourceNameDefinitionLink(docURI, blueprint, ctx.leafNode, tree)
	case plainStringContextIncludePath:
		return s.buildIncludePathDefinitionLink(docURI, ctx.parentNode, ctx.leafNode)
	}

	return []lsp.LocationLink{}, nil
}

func classifyPlainStringContext(collected []*schema.TreeNode) *plainStringContext {
	if len(collected) < 2 {
		return nil
	}

	leaf := collected[len(collected)-1]
	if docmodel.KindFromSchemaElement(leaf.SchemaElement) != docmodel.SchemaElementUnknown {
		return nil
	}

	for i := len(collected) - 2; i >= 0; i-- {
		node := collected[i]
		kind := docmodel.KindFromSchemaElement(node.SchemaElement)

		if kind == docmodel.SchemaElementInclude && hasPathDescendant(collected[i+1:]) {
			return &plainStringContext{
				contextType: plainStringContextIncludePath,
				leafNode:    leaf,
				parentNode:  node,
			}
		}

		if kind == docmodel.SchemaElementExport && hasFieldDescendant(collected[i+1:]) {
			return &plainStringContext{
				contextType: plainStringContextExportField,
				leafNode:    leaf,
				parentNode:  node,
			}
		}

		if kind == docmodel.SchemaElementStringList && isResourceRefStringList(node.Path) {
			return &plainStringContext{
				contextType: plainStringContextResourceRef,
				leafNode:    leaf,
				parentNode:  node,
			}
		}
	}

	return nil
}

func hasFieldDescendant(descendants []*schema.TreeNode) bool {
	for _, node := range descendants {
		if node.Label == "field" {
			return true
		}
	}
	return false
}

func hasPathDescendant(descendants []*schema.TreeNode) bool {
	for _, node := range descendants {
		if node.Label == "path" {
			return true
		}
	}
	return false
}

func isResourceRefStringList(path string) bool {
	return strings.HasSuffix(path, "/linkSelector/exclude") ||
		strings.HasSuffix(path, "/dependsOn")
}

func (s *GotoDefinitionService) buildExportFieldDefinitionLink(
	docURI lsp.URI,
	ctx *plainStringContext,
	tree *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	export, ok := ctx.parentNode.SchemaElement.(*schema.Export)
	if !ok || export == nil || export.Field == nil || export.Field.StringValue == nil {
		return []lsp.LocationLink{}, nil
	}

	segments := parseFieldPathSegments(*export.Field.StringValue)
	if len(segments) < 2 {
		return []lsp.LocationLink{}, nil
	}

	targetPathPrefix, ok := exportFieldNamespaceTargets[segments[0].value]
	if !ok {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		tree, fmt.Sprintf("%s/%s", targetPathPrefix, segments[1].value),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, ctx.leafNode.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildResourceNameDefinitionLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	leafNode *schema.TreeNode,
	tree *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	resourceName, ok := leafNode.SchemaElement.(string)
	if !ok || resourceName == "" {
		return []lsp.LocationLink{}, nil
	}

	targetNode := findSchemaNodeByPath(
		tree, fmt.Sprintf("/resources/%s", resourceName),
	)
	if targetNode == nil {
		return []lsp.LocationLink{}, nil
	}

	return buildLocationLink(docURI, leafNode.Range, targetNode.Range), nil
}

func (s *GotoDefinitionService) buildIncludePathDefinitionLink(
	docURI lsp.URI,
	includeNode *schema.TreeNode,
	leafNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if s.childResolver == nil {
		return []lsp.LocationLink{}, nil
	}

	include, ok := includeNode.SchemaElement.(*schema.Include)
	if !ok || include == nil {
		return []lsp.LocationLink{}, nil
	}

	resolvedPath := s.childResolver.ResolveIncludePath(string(docURI), include)
	if resolvedPath == "" {
		return []lsp.LocationLink{}, nil
	}

	if _, err := os.Stat(resolvedPath); err != nil {
		return []lsp.LocationLink{}, nil
	}

	targetURI := lsp.URI(fileURIFromPath(resolvedPath))
	origin := rangeToLSPRange(leafNode.Range)
	startOfFile := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 0, Character: 0},
	}

	return []lsp.LocationLink{
		{
			OriginSelectionRange: origin,
			TargetURI:            targetURI,
			TargetRange:          startOfFile,
			TargetSelectionRange: startOfFile,
		},
	}, nil
}
