package core

import (
	"fmt"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// UnpackJSONOptions contains optional configuration for JSON unpacking.
type UnpackJSONOptions struct {
	// ParentNode is the parent JSON node, used to provide position info for errors.
	ParentNode *json.Node
}

// UnpackJSONOption is a function that configures UnpackJSONOptions.
type UnpackJSONOption func(*UnpackJSONOptions)

// WithParentNode sets the parent node for error position info.
func WithParentNode(node *json.Node) UnpackJSONOption {
	return func(opts *UnpackJSONOptions) {
		opts.ParentNode = node
	}
}

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
	opts ...UnpackJSONOption,
) error {
	options := &UnpackJSONOptions{}
	for _, opt := range opts {
		opt(options)
	}

	node, ok := nodeMap[key]
	if !ok && required {
		err := fmt.Errorf("required field %q is missing in %s", key, parentPath)
		if options.ParentNode != nil {
			position := source.PositionFromJSONNode(options.ParentNode, linePositions)
			line := position.Line
			col := position.Column
			return &Error{
				ReasonCode:   ErrorCoreReasonCodeMissingMappingNode,
				Err:          err,
				SourceLine:   &line,
				SourceColumn: &col,
			}
		}
		return err
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
