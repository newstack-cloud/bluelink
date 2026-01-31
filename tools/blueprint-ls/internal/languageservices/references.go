package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// ElementCategory identifies what kind of blueprint element is being referenced.
type ElementCategory int

const (
	ElementCategoryResource ElementCategory = iota
	ElementCategoryVariable
	ElementCategoryValue
	ElementCategoryDataSource
	ElementCategoryChild
)

// ResolvedElement represents the identity of a blueprint element
// resolved from a cursor position.
type ResolvedElement struct {
	Category       ElementCategory
	Name           string
	DefinitionPath string
}

// FindReferencesService provides find-all-references functionality
// for blueprint documents.
type FindReferencesService struct {
	state  *State
	logger *zap.Logger
}

// NewFindReferencesService creates a new service for find-all-references support.
func NewFindReferencesService(
	state *State,
	logger *zap.Logger,
) *FindReferencesService {
	return &FindReferencesService{
		state:  state,
		logger: logger,
	}
}

var categoryToDefinitionPrefix = map[ElementCategory]string{
	ElementCategoryResource:   "/resources",
	ElementCategoryVariable:   "/variables",
	ElementCategoryValue:      "/values",
	ElementCategoryDataSource: "/datasources",
	ElementCategoryChild:      "/includes",
}

var namespaceToCategoryMap = map[string]ElementCategory{
	"resources":   ElementCategoryResource,
	"variables":   ElementCategoryVariable,
	"values":      ElementCategoryValue,
	"datasources": ElementCategoryDataSource,
	"children":    ElementCategoryChild,
}

var categoryToNamespaceMap = map[ElementCategory]string{
	ElementCategoryResource:   "resources",
	ElementCategoryVariable:   "variables",
	ElementCategoryValue:      "values",
	ElementCategoryDataSource: "datasources",
	ElementCategoryChild:      "children",
}

// GetReferencesFromContext returns all reference locations for the element
// at the given cursor position.
func (s *FindReferencesService) GetReferencesFromContext(
	docCtx *docmodel.DocumentContext,
	params *lsp.ReferencesParams,
) ([]lsp.Location, error) {
	if docCtx == nil || docCtx.SchemaTree == nil || docCtx.Blueprint == nil {
		return []lsp.Location{}, nil
	}

	pos := source.Position{
		Line:   int(params.Position.Line + 1),
		Column: int(params.Position.Character + 1),
	}

	collected := docCtx.CollectSchemaNodesAtPosition(pos, CompletionColumnLeeway)
	if len(collected) == 0 {
		return []lsp.Location{}, nil
	}

	resolved := resolveElementAtPosition(collected)
	if resolved == nil {
		return []lsp.Location{}, nil
	}

	docURI := params.TextDocument.URI
	locations := []lsp.Location{}

	if params.Context.IncludeDeclaration {
		locations = appendDefinitionLocation(
			locations, docURI, docCtx.SchemaTree, resolved,
		)
	}

	refNodes := collectAllReferences(docCtx.SchemaTree, resolved)
	for _, ref := range refNodes {
		lspRange := rangeToLSPRange(ref.Range)
		if lspRange == nil {
			continue
		}
		locations = append(locations, lsp.Location{
			URI:   docURI,
			Range: lspRange,
		})
	}

	return locations, nil
}

func appendDefinitionLocation(
	locations []lsp.Location,
	docURI lsp.URI,
	tree *schema.TreeNode,
	resolved *ResolvedElement,
) []lsp.Location {
	defNode := findSchemaNodeByPath(tree, resolved.DefinitionPath)
	if defNode == nil {
		return locations
	}

	lspRange := rangeToLSPRange(defNode.Range)
	if lspRange == nil {
		return locations
	}

	return append(locations, lsp.Location{
		URI:   docURI,
		Range: lspRange,
	})
}

func resolveElementAtPosition(collected []*schema.TreeNode) *ResolvedElement {
	if resolved := resolveFromSubstitutionRef(collected); resolved != nil {
		return resolved
	}

	if resolved := resolveFromPlainStringContext(collected); resolved != nil {
		return resolved
	}

	return resolveFromDefinition(collected)
}

func resolveFromSubstitutionRef(collected []*schema.TreeNode) *ResolvedElement {
	for i := len(collected) - 1; i >= 0; i-- {
		node := collected[i]
		kind := docmodel.KindFromSchemaElement(node.SchemaElement)

		switch kind {
		case docmodel.SchemaElementResourceRef:
			prop, ok := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
			if !ok {
				continue
			}
			return buildResolvedElement(ElementCategoryResource, prop.ResourceName)

		case docmodel.SchemaElementVariableRef:
			v, ok := node.SchemaElement.(*substitutions.SubstitutionVariable)
			if !ok {
				continue
			}
			return buildResolvedElement(ElementCategoryVariable, v.VariableName)

		case docmodel.SchemaElementValueRef:
			v, ok := node.SchemaElement.(*substitutions.SubstitutionValueReference)
			if !ok {
				continue
			}
			return buildResolvedElement(ElementCategoryValue, v.ValueName)

		case docmodel.SchemaElementDataSourceRef:
			v, ok := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
			if !ok {
				continue
			}
			return buildResolvedElement(ElementCategoryDataSource, v.DataSourceName)

		case docmodel.SchemaElementChildRef:
			v, ok := node.SchemaElement.(*substitutions.SubstitutionChild)
			if !ok {
				continue
			}
			return buildResolvedElement(ElementCategoryChild, v.ChildName)
		}
	}
	return nil
}

func resolveFromPlainStringContext(collected []*schema.TreeNode) *ResolvedElement {
	ctx := classifyPlainStringContext(collected)
	if ctx == nil {
		return nil
	}

	switch ctx.contextType {
	case plainStringContextResourceRef:
		resourceName, ok := ctx.leafNode.SchemaElement.(string)
		if !ok || resourceName == "" {
			return nil
		}
		return buildResolvedElement(ElementCategoryResource, resourceName)

	case plainStringContextExportField:
		return resolveFromExportFieldContext(ctx)
	}

	return nil
}

func resolveFromExportFieldContext(ctx *plainStringContext) *ResolvedElement {
	export, ok := ctx.parentNode.SchemaElement.(*schema.Export)
	if !ok || export == nil || export.Field == nil || export.Field.StringValue == nil {
		return nil
	}

	segments := parseFieldPathSegments(*export.Field.StringValue)
	if len(segments) < 2 {
		return nil
	}

	category, ok := namespaceToCategoryMap[segments[0].value]
	if !ok {
		return nil
	}

	return buildResolvedElement(category, segments[1].value)
}

func resolveFromDefinition(collected []*schema.TreeNode) *ResolvedElement {
	for i := len(collected) - 1; i >= 0; i-- {
		node := collected[i]
		kind := docmodel.KindFromSchemaElement(node.SchemaElement)

		switch kind {
		case docmodel.SchemaElementResource:
			return buildResolvedElement(ElementCategoryResource, node.Label)
		case docmodel.SchemaElementVariable:
			return buildResolvedElement(ElementCategoryVariable, node.Label)
		case docmodel.SchemaElementValue:
			return buildResolvedElement(ElementCategoryValue, node.Label)
		case docmodel.SchemaElementDataSource:
			return buildResolvedElement(ElementCategoryDataSource, node.Label)
		case docmodel.SchemaElementInclude:
			return buildResolvedElement(ElementCategoryChild, node.Label)
		}
	}
	return nil
}

func buildResolvedElement(category ElementCategory, name string) *ResolvedElement {
	prefix := categoryToDefinitionPrefix[category]
	return &ResolvedElement{
		Category:       category,
		Name:           name,
		DefinitionPath: prefix + "/" + name,
	}
}

func collectAllReferences(
	tree *schema.TreeNode,
	target *ResolvedElement,
) []*schema.TreeNode {
	var results []*schema.TreeNode
	walkSchemaTree(tree, func(node *schema.TreeNode) {
		if ref := matchSubstitutionRef(node, target); ref != nil {
			results = append(results, ref)
			return
		}

		if ref := matchExportFieldRef(node, target); ref != nil {
			results = append(results, ref)
			return
		}

		if matchStringListRef(node, target) {
			results = append(results, node)
		}
	})
	return results
}

func walkSchemaTree(node *schema.TreeNode, visit func(*schema.TreeNode)) {
	if node == nil {
		return
	}
	visit(node)
	for _, child := range node.Children {
		walkSchemaTree(child, visit)
	}
}

func matchSubstitutionRef(
	node *schema.TreeNode,
	target *ResolvedElement,
) *schema.TreeNode {
	kind := docmodel.KindFromSchemaElement(node.SchemaElement)

	switch target.Category {
	case ElementCategoryResource:
		if kind != docmodel.SchemaElementResourceRef {
			return nil
		}
		prop, ok := node.SchemaElement.(*substitutions.SubstitutionResourceProperty)
		if ok && prop.ResourceName == target.Name {
			return node
		}

	case ElementCategoryVariable:
		if kind != docmodel.SchemaElementVariableRef {
			return nil
		}
		v, ok := node.SchemaElement.(*substitutions.SubstitutionVariable)
		if ok && v.VariableName == target.Name {
			return node
		}

	case ElementCategoryValue:
		if kind != docmodel.SchemaElementValueRef {
			return nil
		}
		v, ok := node.SchemaElement.(*substitutions.SubstitutionValueReference)
		if ok && v.ValueName == target.Name {
			return node
		}

	case ElementCategoryDataSource:
		if kind != docmodel.SchemaElementDataSourceRef {
			return nil
		}
		v, ok := node.SchemaElement.(*substitutions.SubstitutionDataSourceProperty)
		if ok && v.DataSourceName == target.Name {
			return node
		}

	case ElementCategoryChild:
		if kind != docmodel.SchemaElementChildRef {
			return nil
		}
		v, ok := node.SchemaElement.(*substitutions.SubstitutionChild)
		if ok && v.ChildName == target.Name {
			return node
		}
	}

	return nil
}

// matchExportFieldRef checks if an export node's field value references
// the target element, returning the field child node for precise location.
func matchExportFieldRef(
	node *schema.TreeNode,
	target *ResolvedElement,
) *schema.TreeNode {
	if docmodel.KindFromSchemaElement(node.SchemaElement) != docmodel.SchemaElementExport {
		return nil
	}

	export, ok := node.SchemaElement.(*schema.Export)
	if !ok || export == nil || export.Field == nil || export.Field.StringValue == nil {
		return nil
	}

	segments := parseFieldPathSegments(*export.Field.StringValue)
	if len(segments) < 2 {
		return nil
	}

	expectedNamespace := categoryToNamespaceMap[target.Category]
	if segments[0].value != expectedNamespace || segments[1].value != target.Name {
		return nil
	}

	// Return the field child node for a more precise reference location.
	fieldNode := findFieldChildNode(node)
	if fieldNode != nil {
		return fieldNode
	}
	return node
}

func findFieldChildNode(exportNode *schema.TreeNode) *schema.TreeNode {
	for _, child := range exportNode.Children {
		if child.Label == "field" {
			return child
		}
	}
	return nil
}

func matchStringListRef(
	node *schema.TreeNode,
	target *ResolvedElement,
) bool {
	if target.Category != ElementCategoryResource {
		return false
	}

	stringVal, ok := node.SchemaElement.(string)
	if !ok || stringVal != target.Name {
		return false
	}

	return isResourceRefStringListPath(node.Path)
}

func isResourceRefStringListPath(path string) bool {
	return strings.Contains(path, "/linkSelector/exclude/") ||
		strings.Contains(path, "/dependsOn/")
}
