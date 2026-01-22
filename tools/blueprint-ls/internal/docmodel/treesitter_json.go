package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/tailscale/hujson"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
)

// JSONParser wraps tree-sitter for parsing JSON/JSONC documents.
// Must call Close() when done to free C memory.
type JSONParser struct {
	parser *tree_sitter.Parser
}

// NewJSONParser creates a new JSON parser using tree-sitter.
func NewJSONParser() (*JSONParser, error) {
	parser := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_json.Language())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}
	return &JSONParser{parser: parser}, nil
}

// Parse parses JSON content and returns the tree-sitter tree.
func (p *JSONParser) Parse(content []byte) *tree_sitter.Tree {
	return p.parser.Parse(content, nil)
}

// ParseIncremental parses JSON content with an existing tree for incremental updates.
func (p *JSONParser) ParseIncremental(content []byte, oldTree *tree_sitter.Tree) *tree_sitter.Tree {
	return p.parser.Parse(content, oldTree)
}

// Close releases the parser resources.
func (p *JSONParser) Close() {
	p.parser.Close()
}

// ParseJSONCToUnified parses JSONC content and converts to a UnifiedNode tree.
// It uses hujson to strip comments, then tree-sitter for error-recovering parsing.
func ParseJSONCToUnified(content string) (*UnifiedNode, error) {
	standardized, err := hujson.Standardize([]byte(content))
	if err != nil {
		// hujson failed on broken input, try tree-sitter with original content
		standardized = []byte(content)
	}

	parser, parseErr := NewJSONParser()
	if parseErr != nil {
		return nil, parseErr
	}
	defer parser.Close()

	tree := parser.Parse(standardized)
	if tree == nil {
		return nil, err
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, err
	}

	return convertJSONTreeSitterNode(rootNode, nil, standardized), nil
}

func convertJSONTreeSitterNode(
	tsNode *tree_sitter.Node,
	parent *UnifiedNode,
	content []byte,
) *UnifiedNode {
	if tsNode == nil {
		return nil
	}

	startPoint := tsNode.StartPosition()
	endPoint := tsNode.EndPosition()

	unified := &UnifiedNode{
		Kind:    mapJSONNodeKind(tsNode.Kind()),
		IsError: tsNode.IsError() || tsNode.IsMissing(),
		TSKind:  tsNode.Kind(),
		Parent:  parent,
		Index:   -1,
		Range: source.Range{
			Start: &source.Position{
				Line:   int(startPoint.Row) + 1,
				Column: int(startPoint.Column) + 1,
			},
			End: &source.Position{
				Line:   int(endPoint.Row) + 1,
				Column: int(endPoint.Column) + 1,
			},
		},
	}

	if unified.Kind == NodeKindScalar && content != nil {
		unified.Value = extractJSONScalarValue(tsNode, content)
		unified.Tag = jsonTagMap[tsNode.Kind()]
	}

	childCount := tsNode.ChildCount()
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}

		childUnified := convertJSONTreeSitterNode(child, unified, content)
		if childUnified != nil {
			unified.Children = append(unified.Children, childUnified)
		}
	}

	postProcessJSONNode(unified)

	return unified
}

var jsonNodeKindMap = map[string]NodeKind{
	"document":       NodeKindDocument,
	"object":         NodeKindMapping,
	"array":          NodeKindSequence,
	"pair":           NodeKindMapping,
	"string":         NodeKindScalar,
	"number":         NodeKindScalar,
	"true":           NodeKindScalar,
	"false":          NodeKindScalar,
	"null":           NodeKindScalar,
	"string_content": NodeKindScalar,
	"ERROR":          NodeKindError,
}

func mapJSONNodeKind(kind string) NodeKind {
	if k, ok := jsonNodeKindMap[kind]; ok {
		return k
	}
	return NodeKindScalar
}

var jsonTagMap = map[string]string{
	"string":         "string",
	"string_content": "string",
	"number":         "number",
	"true":           "boolean",
	"false":          "boolean",
	"null":           "null",
}

func extractJSONScalarValue(tsNode *tree_sitter.Node, content []byte) string {
	if tsNode.Kind() != "string" {
		return tsNode.Utf8Text(content)
	}

	// For strings, get the content without quotes
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child != nil && child.Kind() == "string_content" {
			return child.Utf8Text(content)
		}
	}

	// Fallback: extract and remove quotes manually
	text := tsNode.Utf8Text(content)
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

func postProcessJSONNode(node *UnifiedNode) {
	if node == nil {
		return
	}

	switch node.TSKind {
	case "object":
		processJSONObject(node)
	case "array":
		processJSONArray(node)
	case "pair":
		processJSONPair(node)
	}
}

func processJSONObject(node *UnifiedNode) {
	var newChildren []*UnifiedNode

	for _, child := range node.Children {
		if child.TSKind == "pair" {
			valueNode := findJSONValueNode(child)
			if valueNode != nil {
				valueNode.Parent = node
				newChildren = append(newChildren, valueNode)
			}
		} else if child.TSKind != "{" && child.TSKind != "}" && child.TSKind != "," {
			newChildren = append(newChildren, child)
		}
	}

	node.Children = newChildren
}

var jsonArraySkipTokens = map[string]bool{
	"[": true,
	"]": true,
	",": true,
}

func processJSONArray(node *UnifiedNode) {
	index := 0
	var newChildren []*UnifiedNode

	for _, child := range node.Children {
		if jsonArraySkipTokens[child.TSKind] {
			continue
		}
		child.Index = index
		index++
		newChildren = append(newChildren, child)
	}

	node.Children = newChildren
}

func processJSONPair(node *UnifiedNode) {
	var keyNode, valueNode *UnifiedNode

	for _, child := range node.Children {
		switch child.TSKind {
		case "string":
			if keyNode == nil {
				keyNode = child
			} else if valueNode == nil {
				// Second string is the value (e.g., "type": "value")
				valueNode = child
			}
		case ":", ",":
			continue
		default:
			if keyNode != nil && valueNode == nil {
				valueNode = child
			}
		}
	}

	if keyNode == nil || valueNode == nil {
		return
	}

	valueNode.FieldName = extractKeyValue(keyNode)
	valueNode.KeyRange = &source.Range{
		Start: keyNode.Range.Start,
		End:   keyNode.Range.End,
	}
}

func extractKeyValue(keyNode *UnifiedNode) string {
	if keyNode.Value != "" {
		return keyNode.Value
	}

	for _, c := range keyNode.Children {
		if c.TSKind == "string_content" && c.Value != "" {
			return c.Value
		}
	}

	return ""
}

func findJSONValueNode(pair *UnifiedNode) *UnifiedNode {
	foundKey := false
	for _, child := range pair.Children {
		if child.TSKind == "string" && !foundKey {
			foundKey = true
			continue
		}
		if child.TSKind == ":" {
			continue
		}
		if foundKey {
			return child
		}
	}
	return nil
}
