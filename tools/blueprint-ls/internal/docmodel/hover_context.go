package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// HoverContext provides context for hover information at a position.
type HoverContext struct {
	// ElementKind is the type-safe classification of the schema element
	ElementKind SchemaElementKind

	// SchemaElement is the raw schema element (for type assertions)
	SchemaElement any

	// TreeNode is the schema tree node containing position information
	TreeNode *schema.TreeNode

	// AncestorNodes are the ancestor tree nodes from root to the target node
	AncestorNodes []*schema.TreeNode

	// DescendantNodes are tree nodes deeper than the target node
	// that were collected at the hover position but are not themselves hoverable.
	DescendantNodes []*schema.TreeNode

	// CursorPosition is the 1-based cursor position used for hover.
	// This is set by the caller after DetermineHoverContext returns.
	CursorPosition source.Position
}

// DetermineHoverContext analyzes collected tree nodes to find the hover target.
// It works backwards through the collected elements to find the first element
// that supports hover content.
func DetermineHoverContext(collected []*schema.TreeNode) *HoverContext {
	if len(collected) == 0 {
		return nil
	}

	for i := len(collected) - 1; i >= 0; i-- {
		node := collected[i]
		kind := KindFromSchemaElement(node.SchemaElement)

		if kind.SupportsHover() {
			var descendants []*schema.TreeNode
			if i+1 < len(collected) {
				descendants = collected[i+1:]
			}
			return &HoverContext{
				ElementKind:     kind,
				SchemaElement:   node.SchemaElement,
				TreeNode:        node,
				AncestorNodes:   collected[:i+1],
				DescendantNodes: descendants,
			}
		}
	}

	return nil
}

// SupportsHover returns true if this kind supports hover content.
func (k SchemaElementKind) SupportsHover() bool {
	switch k {
	case SchemaElementFunctionCall,
		SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementChildRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef,
		SchemaElementPathItem,
		SchemaElementResourceType,
		SchemaElementDataSourceType,
		SchemaElementDataSourceFieldType,
		SchemaElementDataSourceFilterOperator,
		// Named elements
		SchemaElementResource,
		SchemaElementVariable,
		SchemaElementValue,
		SchemaElementDataSource,
		SchemaElementInclude,
		// Top-level sections
		SchemaElementResources,
		SchemaElementVariables,
		SchemaElementValues,
		SchemaElementDataSources,
		SchemaElementIncludes,
		// Structural elements
		SchemaElementMappingNode,
		SchemaElementDataSourceFieldExport,
		SchemaElementDataSourceFieldExportMap,
		SchemaElementDataSourceFilters,
		SchemaElementDataSourceFilter,
		SchemaElementDataSourceFilterSearch,
		SchemaElementMetadata,
		SchemaElementDataSourceMetadata,
		SchemaElementLinkSelector,
		SchemaElementStringMap,
		SchemaElementStringOrSubstitutionsMap,
		SchemaElementStringList:
		return true
	}
	return false
}
