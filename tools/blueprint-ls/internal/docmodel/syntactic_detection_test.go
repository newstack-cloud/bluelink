package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type SyntacticDetectionSuite struct {
	suite.Suite
}

func (s *SyntacticDetectionSuite) TestDetectSyntacticStyle_JSONC() {
	// JSONC format always returns JSONC style regardless of ancestors
	tests := []struct {
		name      string
		ancestors []*UnifiedNode
	}{
		{
			name:      "empty ancestors",
			ancestors: nil,
		},
		{
			name: "with block mapping ancestors (ignored for JSONC)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
			},
		},
		{
			name: "with flow mapping ancestors (still JSONC)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "flow_mapping"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			style := DetectSyntacticStyle(tt.ancestors, FormatJSONC)
			s.Assert().Equal(SyntacticStyleJSONC, style)
		})
	}
}

func (s *SyntacticDetectionSuite) TestDetectSyntacticStyle_YAML_BlockStyle() {
	tests := []struct {
		name      string
		ancestors []*UnifiedNode
	}{
		{
			name:      "empty ancestors defaults to block",
			ancestors: nil,
		},
		{
			name: "document only",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
			},
		},
		{
			name: "explicit block mapping",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "block_mapping_pair"},
			},
		},
		{
			name: "explicit block sequence",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_sequence"},
				{TSKind: "block_sequence_item"},
			},
		},
		{
			name: "nested block mapping",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "block_mapping_pair"},
				{TSKind: "block_mapping"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			style := DetectSyntacticStyle(tt.ancestors, FormatYAML)
			s.Assert().Equal(SyntacticStyleBlockYAML, style)
		})
	}
}

func (s *SyntacticDetectionSuite) TestDetectSyntacticStyle_YAML_FlowStyle() {
	tests := []struct {
		name      string
		ancestors []*UnifiedNode
	}{
		{
			name: "flow mapping",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "flow_mapping"},
			},
		},
		{
			name: "flow sequence",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "flow_sequence"},
			},
		},
		{
			name: "flow pair",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "flow_mapping"},
				{TSKind: "flow_pair"},
			},
		},
		{
			name: "flow sequence item",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "flow_sequence"},
				{TSKind: "flow_sequence_item"},
			},
		},
		{
			name: "flow nested inside block - flow wins",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "block_mapping_pair"},
				{TSKind: "flow_sequence"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			style := DetectSyntacticStyle(tt.ancestors, FormatYAML)
			s.Assert().Equal(SyntacticStyleFlowYAML, style)
		})
	}
}

func (s *SyntacticDetectionSuite) TestDetectSyntacticPosition_BlockYAML() {
	tests := []struct {
		name       string
		textBefore string
		ancestors  []*UnifiedNode
		expected   SyntacticPosition
	}{
		{
			name:       "empty line - key position",
			textBefore: "",
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "whitespace only - key position",
			textBefore: "    ",
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "typing key name",
			textBefore: "  run",
			ancestors:  nil,
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "after colon - value position",
			textBefore: "fieldName: ",
			ancestors:  nil,
			expected:   SyntacticPositionValueField,
		},
		{
			name:       "after colon typing value",
			textBefore: "fieldName: someValue",
			ancestors:  nil,
			expected:   SyntacticPositionValueField,
		},
		{
			name:       "sequence item with dash-space",
			textBefore: "  - item",
			ancestors:  nil,
			expected:   SyntacticPositionSequenceItem,
		},
		{
			name:       "sequence item dash only",
			textBefore: "  -",
			ancestors:  nil,
			expected:   SyntacticPositionSequenceItem,
		},
		{
			name:       "colon at end",
			textBefore: "name:",
			ancestors:  nil,
			expected:   SyntacticPositionValueField,
		},
		{
			name:       "newline after mapping - key position",
			textBefore: "spec:\n    ",
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "empty in sequence context",
			textBefore: "",
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionSequenceItem,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			pos := DetectSyntacticPosition(
				SyntacticStyleBlockYAML,
				nil,
				tt.ancestors,
				source.Position{Line: 1, Column: 1},
				tt.textBefore,
			)
			s.Assert().Equal(tt.expected, pos, "textBefore: %q", tt.textBefore)
		})
	}
}

func (s *SyntacticDetectionSuite) TestDetectSyntacticPosition_FlowStyle() {
	tests := []struct {
		name       string
		style      SyntacticStyle
		textBefore string
		ancestors  []*UnifiedNode
		expected   SyntacticPosition
	}{
		// Empty container cases
		{
			name:       "inside empty array - JSONC",
			style:      SyntacticStyleJSONC,
			textBefore: `"exclude": [`,
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionSequenceItem,
		},
		{
			name:       "inside empty object - JSONC",
			style:      SyntacticStyleJSONC,
			textBefore: `"spec": {`,
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "inside empty array - flow YAML",
			style:      SyntacticStyleFlowYAML,
			textBefore: "exclude: [",
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionSequenceItem,
		},

		// After delimiters
		{
			name:       "after opening brace",
			style:      SyntacticStyleJSONC,
			textBefore: `{ `,
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "after opening bracket",
			style:      SyntacticStyleJSONC,
			textBefore: `[ `,
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionSequenceItem,
		},
		{
			name:       "after comma in mapping",
			style:      SyntacticStyleJSONC,
			textBefore: `"name": "test", `,
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionKeyField,
		},
		{
			name:       "after comma in sequence",
			style:      SyntacticStyleJSONC,
			textBefore: `["a", `,
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionSequenceItem,
		},
		{
			name:       "after colon",
			style:      SyntacticStyleJSONC,
			textBefore: `"type": `,
			ancestors:  []*UnifiedNode{{Kind: NodeKindMapping}},
			expected:   SyntacticPositionValueField,
		},

		// Inside string in array
		{
			name:       "inside quoted string in array - JSONC",
			style:      SyntacticStyleJSONC,
			textBefore: `["`,
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionStringContent,
		},
		{
			name:       "inside quoted string after value - JSONC",
			style:      SyntacticStyleJSONC,
			textBefore: `["a", "`,
			ancestors:  []*UnifiedNode{{Kind: NodeKindSequence}},
			expected:   SyntacticPositionStringContent,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			pos := DetectSyntacticPosition(
				tt.style,
				nil,
				tt.ancestors,
				source.Position{Line: 1, Column: 1},
				tt.textBefore,
			)
			s.Assert().Equal(tt.expected, pos, "style: %v, textBefore: %q", tt.style, tt.textBefore)
		})
	}
}

func (s *SyntacticDetectionSuite) TestExtractCurrentLine() {
	tests := []struct {
		name       string
		textBefore string
		expected   string
	}{
		{
			name:       "single line",
			textBefore: "hello world",
			expected:   "hello world",
		},
		{
			name:       "after newline",
			textBefore: "line1\nline2",
			expected:   "line2",
		},
		{
			name:       "multiple newlines",
			textBefore: "line1\nline2\nline3",
			expected:   "line3",
		},
		{
			name:       "ends with newline",
			textBefore: "line1\n",
			expected:   "",
		},
		{
			name:       "empty",
			textBefore: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := extractCurrentLine(tt.textBefore)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func (s *SyntacticDetectionSuite) TestIsInsideEmptyContainer() {
	tests := []struct {
		name       string
		textBefore string
		expected   bool
	}{
		{
			name:       "empty array",
			textBefore: "exclude: [",
			expected:   true,
		},
		{
			name:       "empty array with space",
			textBefore: "exclude: [ ",
			expected:   true,
		},
		{
			name:       "empty object",
			textBefore: "spec: {",
			expected:   true,
		},
		{
			name:       "empty object with space",
			textBefore: "spec: { ",
			expected:   true,
		},
		{
			name:       "array with content",
			textBefore: `["item"`,
			expected:   false,
		},
		{
			name:       "object with content",
			textBefore: `{"key": "value"`,
			expected:   false,
		},
		{
			name:       "no container",
			textBefore: "just some text",
			expected:   false,
		},
		{
			name:       "JSONC style empty array",
			textBefore: `"exclude": [`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := isInsideEmptyContainer(tt.textBefore)
			s.Assert().Equal(tt.expected, result, "textBefore: %q", tt.textBefore)
		})
	}
}

func (s *SyntacticDetectionSuite) TestIsInsideArrayString() {
	tests := []struct {
		name       string
		textBefore string
		expected   bool
	}{
		{
			name:       "just opened quote",
			textBefore: `["`,
			expected:   true,
		},
		{
			name:       "typing inside string",
			textBefore: `["some`,
			expected:   true,
		},
		{
			name:       "after first item, in second string",
			textBefore: `["a", "b`,
			expected:   true,
		},
		{
			name:       "no quote",
			textBefore: `[`,
			expected:   false,
		},
		{
			name:       "closed string",
			textBefore: `["item"`,
			expected:   false,
		},
		{
			name:       "after comma, before quote",
			textBefore: `["a", `,
			expected:   false,
		},
		{
			name:       "no bracket",
			textBefore: `"item"`,
			expected:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := isInsideArrayString(tt.textBefore)
			s.Assert().Equal(tt.expected, result, "textBefore: %q", tt.textBefore)
		})
	}
}

func (s *SyntacticDetectionSuite) TestExtractTypedPrefix() {
	tests := []struct {
		name       string
		position   SyntacticPosition
		style      SyntacticStyle
		textBefore string
		expected   string
	}{
		// Key field cases
		{
			name:       "typing key - block YAML",
			position:   SyntacticPositionKeyField,
			style:      SyntacticStyleBlockYAML,
			textBefore: "  run",
			expected:   "run",
		},
		{
			name:       "typing key with hyphen",
			position:   SyntacticPositionKeyField,
			style:      SyntacticStyleBlockYAML,
			textBefore: "  my-field",
			expected:   "my-field",
		},
		{
			name:       "empty key position",
			position:   SyntacticPositionKeyField,
			style:      SyntacticStyleBlockYAML,
			textBefore: "  ",
			expected:   "",
		},

		// Value field cases
		{
			name:       "typing value after colon",
			position:   SyntacticPositionValueField,
			style:      SyntacticStyleBlockYAML,
			textBefore: "type: aws/lam",
			expected:   "aws/lam",
		},
		{
			name:       "empty value position",
			position:   SyntacticPositionValueField,
			style:      SyntacticStyleBlockYAML,
			textBefore: "type: ",
			expected:   "",
		},

		// Sequence item cases - block YAML
		{
			name:       "sequence item - block YAML",
			position:   SyntacticPositionSequenceItem,
			style:      SyntacticStyleBlockYAML,
			textBefore: "  - item",
			expected:   "item",
		},
		{
			name:       "empty sequence item - block YAML",
			position:   SyntacticPositionSequenceItem,
			style:      SyntacticStyleBlockYAML,
			textBefore: "  - ",
			expected:   "",
		},

		// Sequence item cases - flow/JSONC
		{
			name:       "sequence item after bracket - JSONC",
			position:   SyntacticPositionSequenceItem,
			style:      SyntacticStyleJSONC,
			textBefore: `["item`,
			expected:   "item",
		},
		{
			name:       "empty sequence item after bracket",
			position:   SyntacticPositionSequenceItem,
			style:      SyntacticStyleJSONC,
			textBefore: `[`,
			expected:   "",
		},

		// String content cases
		{
			name:       "inside string - typing",
			position:   SyntacticPositionStringContent,
			style:      SyntacticStyleJSONC,
			textBefore: `["proc`,
			expected:   "proc",
		},
		{
			name:       "inside empty string",
			position:   SyntacticPositionStringContent,
			style:      SyntacticStyleJSONC,
			textBefore: `["`,
			expected:   "",
		},

		// Empty container
		{
			name:       "empty container",
			position:   SyntacticPositionEmptyContainer,
			style:      SyntacticStyleJSONC,
			textBefore: `{`,
			expected:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := ExtractTypedPrefix(tt.position, tt.style, tt.textBefore)
			s.Assert().Equal(tt.expected, result, "position: %v, textBefore: %q", tt.position, tt.textBefore)
		})
	}
}

func (s *SyntacticDetectionSuite) TestIsWordCharacter() {
	wordChars := []byte{'a', 'z', 'A', 'Z', '0', '9', '_', '-', '.'}
	for _, c := range wordChars {
		s.Assert().True(isWordCharacter(c), "expected '%c' to be a word char", c)
	}

	nonWordChars := []byte{' ', '\t', ':', '$', '{', '}', '[', ']', '"', '\'', ','}
	for _, c := range nonWordChars {
		s.Assert().False(isWordCharacter(c), "expected '%c' not to be a word char", c)
	}
}

func (s *SyntacticDetectionSuite) TestFindNearestContainerKind() {
	tests := []struct {
		name      string
		ancestors []*UnifiedNode
		expected  NodeKind
	}{
		{
			name:      "empty ancestors - default mapping",
			ancestors: nil,
			expected:  NodeKindMapping,
		},
		{
			name: "sequence as nearest",
			ancestors: []*UnifiedNode{
				{Kind: NodeKindMapping},
				{Kind: NodeKindSequence},
			},
			expected: NodeKindSequence,
		},
		{
			name: "mapping as nearest",
			ancestors: []*UnifiedNode{
				{Kind: NodeKindSequence},
				{Kind: NodeKindMapping},
			},
			expected: NodeKindMapping,
		},
		{
			name: "skip non-container kinds",
			ancestors: []*UnifiedNode{
				{Kind: NodeKindSequence},
				{Kind: NodeKindScalar},
				{Kind: NodeKindKey},
			},
			expected: NodeKindSequence,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := findNearestContainerKind(tt.ancestors)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func TestSyntacticDetectionSuite(t *testing.T) {
	suite.Run(t, new(SyntacticDetectionSuite))
}
