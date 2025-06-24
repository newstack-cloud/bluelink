package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// GotoDefinitionService is a service that provides the functionality
// to go to the definition of a symbol in a blueprint.
type GotoDefinitionService struct {
	state  *State
	logger *zap.Logger
}

// NewGotoDefinitionService creates a new service for go to definition
// support.
func NewGotoDefinitionService(
	state *State,
	logger *zap.Logger,
) *GotoDefinitionService {
	return &GotoDefinitionService{
		state:  state,
		logger: logger,
	}
}

// GetDefinitions returns the definition links for the symbol at the given
// position.
func (s *GotoDefinitionService) GetDefinitions(
	content string,
	tree *schema.TreeNode,
	blueprint *schema.Blueprint,
	params *lsp.TextDocumentPositionParams,
) ([]lsp.LocationLink, error) {

	// The last element in the collected list is the element with the shortest
	// range that contains the position.
	collected := []*schema.TreeNode{}
	collectElementsAtPosition(tree, params.Position, s.logger, &collected, CompletionColumnLeeway)

	return s.getDefinitionLinks(
		params.TextDocument.URI,
		blueprint,
		collected,
		tree,
	)
}

func (s *GotoDefinitionService) getDefinitionLinks(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	collected []*schema.TreeNode,
	tree *schema.TreeNode,
) ([]lsp.LocationLink, error) {

	if len(collected) == 0 {
		return []lsp.LocationLink{}, nil
	}

	// Work backwards through the collected elements to find the first element
	// of a type that supports definition links.
	var node *schema.TreeNode
	var elementType string
	i := len(collected) - 1
	for node == nil && i >= 0 {
		pathParts := strings.Split(collected[i].Path, "/")
		node, elementType = s.matchDefinitionElement(
			collected,
			i,
			pathParts,
		)
		i -= 1
	}

	switch elementType {
	case "resourceRef":
		return s.getResourceRefLocationLink(docURI, blueprint, node, tree)
	case "datasourceRef":
		return s.getDataSourceRefLocationLink(docURI, blueprint, node, tree)
	case "varRef":
		return s.getVarRefLocationLink(docURI, blueprint, node, tree)
	case "valRef":
		return s.getValRefLocationLink(docURI, blueprint, node, tree)
	case "childRef":
		return s.getChildRefLocationLink(docURI, blueprint, node, tree)
	default:
		return []lsp.LocationLink{}, nil
	}
}

func (s *GotoDefinitionService) matchDefinitionElement(
	collected []*schema.TreeNode,
	index int,
	pathParts []string,
) (*schema.TreeNode, string) {

	if s.isResourceRef(pathParts) {
		return collected[index], "resourceRef"
	}

	if s.isDataSourceRef(pathParts) {
		return collected[index], "datasourceRef"
	}

	if s.isVariableRef(pathParts) {
		return collected[index], "varRef"
	}

	if s.isValueRef(pathParts) {
		return collected[index], "valRef"
	}

	if s.isChildRef(pathParts) {
		return collected[index], "childRef"
	}

	return nil, ""
}

func (s *GotoDefinitionService) isResourceRef(
	pathParts []string,
) bool {
	return len(pathParts) > 2 &&
		pathParts[len(pathParts)-2] == "resourceRef"
}

func (s *GotoDefinitionService) isDataSourceRef(
	pathParts []string,
) bool {
	return len(pathParts) > 2 &&
		pathParts[len(pathParts)-2] == "datasourceRef"
}

func (s *GotoDefinitionService) isVariableRef(
	pathParts []string,
) bool {
	return len(pathParts) > 2 &&
		pathParts[len(pathParts)-2] == "varRef"
}

func (s *GotoDefinitionService) isValueRef(
	pathParts []string,
) bool {
	return len(pathParts) > 2 &&
		pathParts[len(pathParts)-2] == "valRef"
}

func (s *GotoDefinitionService) isChildRef(
	pathParts []string,
) bool {
	return len(pathParts) > 2 &&
		pathParts[len(pathParts)-2] == "childRef"
}

func (s *GotoDefinitionService) getResourceRefLocationLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Resources == nil || len(blueprint.Resources.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	locationLinks := []lsp.LocationLink{}

	resourceProp, isResourceProp := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
	if !isResourceProp {
		return locationLinks, nil
	}

	resourceNode := findNodeByPath(
		rootNode,
		fmt.Sprintf("/resources/%s", resourceProp.ResourceName),
		s.logger,
	)
	if resourceNode == nil {
		return locationLinks, nil
	}

	targetRange := rangeToLSPRange(resourceNode.Range)

	return []lsp.LocationLink{
		{
			OriginSelectionRange: rangeToLSPRange(node.Range),
			TargetURI:            docURI,
			TargetRange:          *targetRange,
			TargetSelectionRange: *targetRange,
		},
	}, nil
}

func (s *GotoDefinitionService) getDataSourceRefLocationLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.DataSources == nil || len(blueprint.DataSources.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	locationLinks := []lsp.LocationLink{}

	dataSourceProp, isDataSourceProp := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
	if !isDataSourceProp {
		return locationLinks, nil
	}

	dataSourceNode := findNodeByPath(
		rootNode,
		fmt.Sprintf("/datasources/%s", dataSourceProp.DataSourceName),
		s.logger,
	)
	if dataSourceNode == nil {
		return locationLinks, nil
	}

	targetRange := rangeToLSPRange(dataSourceNode.Range)

	return []lsp.LocationLink{
		{
			OriginSelectionRange: rangeToLSPRange(node.Range),
			TargetURI:            docURI,
			TargetRange:          *targetRange,
			TargetSelectionRange: *targetRange,
		},
	}, nil
}

func (s *GotoDefinitionService) getVarRefLocationLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Variables == nil || len(blueprint.Variables.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	locationLinks := []lsp.LocationLink{}

	varProp, isVarProp := node.SchemaElement.(*substitutions.SubstitutionVariable)
	if !isVarProp {
		return locationLinks, nil
	}

	varNode := findNodeByPath(
		rootNode,
		fmt.Sprintf("/variables/%s", varProp.VariableName),
		s.logger,
	)
	if varNode == nil {
		return locationLinks, nil
	}

	targetRange := rangeToLSPRange(varNode.Range)

	return []lsp.LocationLink{
		{
			OriginSelectionRange: rangeToLSPRange(node.Range),
			TargetURI:            docURI,
			TargetRange:          *targetRange,
			TargetSelectionRange: *targetRange,
		},
	}, nil
}

func (s *GotoDefinitionService) getValRefLocationLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Values == nil || len(blueprint.Values.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	locationLinks := []lsp.LocationLink{}

	valProp, isValProp := node.SchemaElement.(*substitutions.SubstitutionValueReference)
	if !isValProp {
		return locationLinks, nil
	}

	valNode := findNodeByPath(
		rootNode,
		fmt.Sprintf("/values/%s", valProp.ValueName),
		s.logger,
	)
	if valNode == nil {
		return locationLinks, nil
	}

	targetRange := rangeToLSPRange(valNode.Range)

	return []lsp.LocationLink{
		{
			OriginSelectionRange: rangeToLSPRange(node.Range),
			TargetURI:            docURI,
			TargetRange:          *targetRange,
			TargetSelectionRange: *targetRange,
		},
	}, nil
}

func (s *GotoDefinitionService) getChildRefLocationLink(
	docURI lsp.URI,
	blueprint *schema.Blueprint,
	node *schema.TreeNode,
	rootNode *schema.TreeNode,
) ([]lsp.LocationLink, error) {
	if blueprint.Include == nil || len(blueprint.Include.Values) == 0 {
		return []lsp.LocationLink{}, nil
	}

	locationLinks := []lsp.LocationLink{}

	childProp, isChildProp := node.SchemaElement.(*substitutions.SubstitutionChild)
	if !isChildProp {
		return locationLinks, nil
	}

	childNode := findNodeByPath(
		rootNode,
		fmt.Sprintf("/includes/%s", childProp.ChildName),
		s.logger,
	)
	if childNode == nil {
		return locationLinks, nil
	}

	targetRange := rangeToLSPRange(childNode.Range)

	return []lsp.LocationLink{
		{
			OriginSelectionRange: rangeToLSPRange(node.Range),
			TargetURI:            docURI,
			TargetRange:          *targetRange,
			TargetSelectionRange: *targetRange,
		},
	}, nil
}
