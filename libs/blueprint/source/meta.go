package source

import (
	"strings"
	"unicode/utf8"

	"github.com/coreos/go-json"
	"gopkg.in/yaml.v3"
)

// ColumnAccuracy indicates how accurate the column position is.
type ColumnAccuracy int

const (
	// ColumnAccuracyExact indicates that column numbers are accurate.
	ColumnAccuracyExact ColumnAccuracy = 1
	// ColumnAccuracyApproximate indicates that column numbers are approximate.
	// This is the case for YAML block literals where the column cannot be precisely determined.
	ColumnAccuracyApproximate ColumnAccuracy = 2
)

// Meta represents information about the deserialised source of
// a blueprint value including the line and column
// where a blueprint element begins that can be used by tools such
// as linters to provide more detailed diagnostics to users creating
// blueprints from source in some supported formats.
type Meta struct {
	Position
	EndPosition    *Position       `json:"endPosition,omitempty"`
	ColumnAccuracy *ColumnAccuracy `json:"columnAccuracy,omitempty"`
}

// Position represents a position in the source code of a blueprint.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func (p *Position) GetLine() int {
	return p.Line
}

func (p *Position) GetColumn() int {
	return p.Column
}

// PositionFromSourceMeta returns the line and column from the provided source meta.
// This is primarily useful for attaching position information to errors.
func PositionFromSourceMeta(sourceMeta *Meta) (line *int, column *int) {
	if sourceMeta == nil {
		return nil, nil
	}

	return &sourceMeta.Line, &sourceMeta.Column
}

// PositionRange holds start and end positions with column accuracy information.
type PositionRange struct {
	Line           *int
	Column         *int
	EndLine        *int
	EndColumn      *int
	ColumnAccuracy *ColumnAccuracy
}

// PositionRangeFromSourceMeta extracts position range info from source meta.
// Returns an empty PositionRange (with nil fields) if sourceMeta is nil.
func PositionRangeFromSourceMeta(sourceMeta *Meta) *PositionRange {
	if sourceMeta == nil {
		return &PositionRange{}
	}

	pr := &PositionRange{
		Line:           &sourceMeta.Line,
		Column:         &sourceMeta.Column,
		ColumnAccuracy: sourceMeta.ColumnAccuracy,
	}

	if sourceMeta.EndPosition != nil {
		pr.EndLine = &sourceMeta.EndPosition.Line
		pr.EndColumn = &sourceMeta.EndPosition.Column
	}

	return pr
}

// EndSourcePositionFromYAMLScalarNode returns the precise
// end position of a YAML scalar node.
func EndSourcePositionFromYAMLScalarNode(node *yaml.Node) *Position {
	if node.Kind != yaml.ScalarNode {
		return nil
	}

	if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
		charCount := utf8.RuneCountInString(node.Value) + 2
		return &Position{
			Line:   node.Line,
			Column: node.Column + charCount,
		}
	}

	// 0 indicates plain style
	if node.Style == 0 {
		return &Position{
			Line:   node.Line,
			Column: node.Column + utf8.RuneCountInString(node.Value),
		}
	}

	lines := strings.Split(strings.ReplaceAll(node.Value, "\r\n", "\n"), "\n")
	lineCountInBlock := len(lines) - 1
	columnOnLastLine := node.Column

	if lineCountInBlock > 0 {
		columnOnLastLine += utf8.RuneCountInString(lines[lineCountInBlock-1])
	}

	return &Position{
		Line:   node.Line + lineCountInBlock,
		Column: columnOnLastLine - 1,
	}
}

// Range represents a range in the source code of a blueprint.
// Start and End could also hold additional information provided
// in the Meta struct.
type Range struct {
	Start *Position
	End   *Position
}

// ExtractSourcePositionFromJSONNode extracts the position
// in source document from a given JSON node and line positions.
func ExtractSourcePositionFromJSONNode(
	node *json.Node,
	linePositions []int,
) *Meta {
	startOffset := getJSONNodeStartOffset(node)
	position := PositionFromOffset(startOffset, linePositions)
	// coreos/go-json counts the end offset as the index of the last
	// character in the node, so we need to add 1 to get the end position.
	endOffset := node.End + 1
	endPosition := PositionFromOffset(endOffset, linePositions)
	return &Meta{
		Position:    position,
		EndPosition: &endPosition,
	}
}

// ExtractSourcePositionForJSONNodeMapField extracts the position
// in source document from a given JSON node and line positions.
func ExtractSourcePositionForJSONNodeMapField(
	node *json.Node,
	linePositions []int,
) *Meta {
	position := PositionFromOffset(node.KeyStart, linePositions)
	// coreos/go-json counts the end offset as the index of the last
	// character in the node, so we need to add 1 to get the end position.
	endOffset := node.End + 1
	endPosition := PositionFromOffset(endOffset, linePositions)
	return &Meta{
		Position:    position,
		EndPosition: &endPosition,
	}
}

func getJSONNodeStartOffset(node *json.Node) int {
	startOffset := node.KeyEnd
	if startOffset == 0 {
		// Not a value for a map, take the start position of the value
		// instead of the end of a key in a map.
		startOffset = node.Start
	}

	return startOffset
}

// PositionFromJSONNode returns the position of a JSON node
// in the source code based on the node and an ordered list of line offsets.
func PositionFromJSONNode(node *json.Node, linePositions []int) Position {
	startOffset := getJSONNodeStartOffset(node)
	return PositionFromOffset(startOffset, linePositions)
}

// PositionFromOffset returns the position of a character in the source
// code based on the offset and an ordered list of line offsets.
// This treats the offset of a new line character as the end of the line
// and not the first column of the next line.
func PositionFromOffset(offset int, linePositions []int) Position {
	line := 0
	for i, lineOffset := range linePositions {
		if offset < lineOffset {
			break
		}
		line = i
	}

	column := offset - linePositions[line]
	if column == 0 {
		prevLineOffset := 0
		if line > 0 {
			prevLineOffset = linePositions[line-1]
		}
		lineLength := linePositions[line] - prevLineOffset
		return Position{
			Line:   line,
			Column: lineLength,
		}
	}

	return Position{
		Line:   line + 1,
		Column: column,
	}
}

// GetLastColumnOnLine returns the last column on a given line
// based on the provided line positions.
// The second return value is the raw offset for the end of the line.
func GetLastColumnOnLine(line int, linePositions []int) (int, int) {
	lineIndex := line - 1
	if lineIndex < 0 || lineIndex >= len(linePositions)-1 {
		return -1, -1
	}

	lineStartOffset := linePositions[lineIndex]
	nextLineStartOffset := linePositions[lineIndex+1]
	column := nextLineStartOffset - lineStartOffset
	// The refined raw offset is the last character on the same line.
	return column, nextLineStartOffset - 1
}
