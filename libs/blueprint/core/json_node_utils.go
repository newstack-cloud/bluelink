package core

import (
	"fmt"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// UnpackValueFromJSONMapNode unpacks a value from a JSON map node
// into the target struct.
func UnpackValueFromJSONMapNode(
	nodeMap map[string]json.Node,
	key string,
	target JSONNodeExtractable,
	linePositions []int,
	parentPath string,
	parentIsRoot bool,
	required bool,
) error {
	node, ok := nodeMap[key]
	if !ok && required {
		return fmt.Errorf("required field missing %s in %s", key, parentPath)
	}

	if node.Value == nil && !required {
		return nil
	}

	path := CreateJSONNodePath(key, parentPath, parentIsRoot)
	return target.FromJSONNode(&node, linePositions, path)
}

// UnpackValuesFromJSONMapNode unpacks a slice of values from a JSON map node
// into the target slice.
func UnpackValuesFromJSONMapNode[Target JSONNodeExtractable](
	nodeMap map[string]json.Node,
	key string,
	target *[]Target,
	linePositions []int,
	parentPath string,
	parentIsRoot bool,
	required bool,
) error {
	node, ok := nodeMap[key]
	if !ok && required {
		return fmt.Errorf("missing %s in %s", key, parentPath)
	}

	if node.Value == nil && !required {
		return nil
	}

	fieldPath := CreateJSONNodePath(key, parentPath, parentIsRoot)
	nodeSlice, ok := node.Value.([]json.Node)
	if !ok {
		position := source.PositionFromOffset(node.KeyEnd, linePositions)
		return errInvalidMappingNode(&position)
	}

	for i, node := range nodeSlice {
		key := fmt.Sprintf("%d", i)
		path := CreateJSONNodePath(key, fieldPath, parentIsRoot)
		var item Target
		err := item.FromJSONNode(&node, linePositions, path)
		if err != nil {
			return err
		}
		*target = append(*target, item)
	}

	return nil
}

// LinePositionsFromSource returns the line positions of the source string.
// It returns a slice of integers representing the start positions of each line.
// This will always include a line ending even if the source string
// does not end with a newline character to be able to obtain the length
// of the last line in the source string.
func LinePositionsFromSource(source string) []int {
	linePositions := []int{0}
	for i, c := range source {
		if c == '\n' {
			linePositions = append(linePositions, i)
		}
	}

	if source[len(source)-1] != '\n' {
		// Ensure that an end offset can be determined
		// when the source document does not end with a newline
		// character.
		linePositions = append(linePositions, len(source))
	}

	return linePositions
}

// CreateJSONNodePath creates a JSON node path from the given key and parent path.
func CreateJSONNodePath(key string, parentPath string, parentIsRoot bool) string {
	if parentPath == "" || parentIsRoot {
		return key
	}

	return fmt.Sprintf("%s.%s", parentPath, key)
}
