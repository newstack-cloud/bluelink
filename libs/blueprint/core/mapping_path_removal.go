package core

import (
	"math"
	"slices"
)

// RemovePathValue removes the value at a path from a MappingNode, reporting whether there
// was anything there to remove.
//
// It is the counterpart to InjectPathValue. Where injection creates the structure on the
// way to a value, removal takes away the structure that is left holding nothing once the
// value is gone, an object with no fields and an array with no items are removed along
// with it, up as far as the path was followed. Without that, removing a value would leave
// behind a shape that was only ever there to carry it.
//
// A container left holding only the attributes a selector matched it on is removed too.
// Injection builds such an item from the selector's own conditions when nothing matches
// yet, so an item holding nothing else is one that was created to carry the value being
// removed. Removal is then the exact inverse of injection, and a spec that has had
// contributions applied and taken away again is the spec it started as.
//
// A path supports the same accessors as GetPathValue and InjectPathValue.
//
// Example:
//
//	core.RemovePathValue("$.config.endpoints[0]", node, 3)
func RemovePathValue(
	path string,
	removeFrom *MappingNode,
	maxTraverseDepth int,
) (bool, error) {
	parsedPath, err := parsePath(
		path,
		/* allowPatterns */ false,
	)
	if err != nil {
		return false, err
	}

	if len(parsedPath) == 0 || removeFrom == nil {
		return false, nil
	}

	if maxTraverseDepth < len(parsedPath) {
		return false, nil
	}

	// The node each path item was reached from, so that containers left empty by the
	// removal can be removed in turn, deepest first.
	parents := []*MappingNode{removeFrom}
	current := removeFrom
	for i := range len(parsedPath) - 1 {
		current = childForPathItem(current, parsedPath[i])
		if current == nil {
			return false, nil
		}
		parents = append(parents, current)
	}

	removed := removeAtPathItem(current, parsedPath[len(parsedPath)-1])
	if !removed {
		return false, nil
	}

	removeEmptyContainers(parents, parsedPath)

	return true, nil
}

func removeEmptyContainers(parents []*MappingNode, parsedPath []*pathItem) {
	for i := len(parents) - 1; i > 0; i -= 1 {
		reachedBy := parsedPath[i-1]
		if !isEmptyContainer(parents[i]) && !isSelectorCarrier(parents[i], reachedBy) {
			return
		}

		if !removeAtPathItem(parents[i-1], reachedBy) {
			return
		}
	}
}

// Reports whether a node holds nothing beyond the attributes the path item's selector
// matches it on, which is what injection creates when it has to make the item itself.
func isSelectorCarrier(node *MappingNode, item *pathItem) bool {
	if node == nil || item.arrayItemSelector == nil || item.arrayItemSelector.isScalar() {
		return false
	}

	if node.Fields == nil || len(node.Fields) != len(item.arrayItemSelector.conditions) {
		return false
	}

	return objectHasPropertyWithValue(item.arrayItemSelector)(node)
}

func isEmptyContainer(node *MappingNode) bool {
	if node == nil {
		return false
	}

	if node.Fields != nil {
		return len(node.Fields) == 0
	}

	if node.Items != nil {
		return len(node.Items) == 0
	}

	return false
}

// Resolves the node a path item points at, without modifying anything.
func childForPathItem(node *MappingNode, item *pathItem) *MappingNode {
	if item.fieldName != "" && node.Fields != nil {
		return node.Fields[item.fieldName]
	}

	if item.arrayIndex != nil && node.Items != nil {
		child, exists := itemAtIndex(node.Items, *item.arrayIndex)
		if !exists {
			return nil
		}

		return child
	}

	if item.arrayItemSelector != nil && node.Items != nil {
		index := slices.IndexFunc(
			node.Items,
			objectHasPropertyWithValue(item.arrayItemSelector),
		)
		if index < 0 {
			return nil
		}

		return node.Items[index]
	}

	return nil
}

// Takes a path item's target out of the node holding it, reporting whether it was there.
func removeAtPathItem(node *MappingNode, item *pathItem) bool {
	if item.fieldName != "" && node.Fields != nil {
		if _, exists := node.Fields[item.fieldName]; !exists {
			return false
		}

		delete(node.Fields, item.fieldName)
		return true
	}

	if item.arrayIndex != nil && node.Items != nil {
		index := int(math.Min(float64(*item.arrayIndex), float64(len(node.Items))))
		if index < 0 || index >= len(node.Items) {
			return false
		}

		node.Items = slices.Delete(node.Items, index, index+1)
		return true
	}

	if item.arrayItemSelector != nil && node.Items != nil {
		index := slices.IndexFunc(
			node.Items,
			objectHasPropertyWithValue(item.arrayItemSelector),
		)
		if index < 0 {
			return false
		}

		node.Items = slices.Delete(node.Items, index, index+1)
		return true
	}

	return false
}
