package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
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
			return &HoverContext{
				ElementKind:   kind,
				SchemaElement: node.SchemaElement,
				TreeNode:      node,
				AncestorNodes: collected[:i+1],
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
		SchemaElementDataSourceType:
		return true
	}
	return false
}
