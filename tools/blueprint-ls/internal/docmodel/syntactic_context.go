package docmodel

// SyntacticPosition represents where the cursor is syntactically within
// the document structure. This is independent of the document format
// (YAML vs JSONC) and represents the semantic editing position.
type SyntacticPosition int

const (
	// SyntacticPositionUnknown indicates the position could not be determined.
	SyntacticPositionUnknown SyntacticPosition = iota

	// SyntacticPositionKeyField indicates the cursor is at a position where
	// a mapping key (field name) should be entered.
	// Examples:
	//   YAML block: "  |" (indented empty line under a mapping)
	//   YAML flow:  "{|" or "{key: value, |"
	//   JSONC:      "{|" or '{"key": "value", |'
	SyntacticPositionKeyField

	// SyntacticPositionValueField indicates the cursor is at a position where
	// a value should be entered (after a colon).
	// Examples:
	//   YAML block: "key: |"
	//   YAML flow:  "{key: |"
	//   JSONC:      '{"key": |'
	SyntacticPositionValueField

	// SyntacticPositionSequenceItem indicates the cursor is at a position where
	// a sequence/array item should be entered.
	// Examples:
	//   YAML block: "- |" (after sequence marker)
	//   YAML flow:  "[|" or "[item, |"
	//   JSONC:      "[|" or '["item", |'
	SyntacticPositionSequenceItem

	// SyntacticPositionStringContent indicates the cursor is inside a string value.
	// This is relevant for substitution completions and array item completions.
	// Examples:
	//   YAML: '"${resources.|}"'
	//   JSONC: '["partial|"]'
	SyntacticPositionStringContent

	// SyntacticPositionEmptyContainer indicates the cursor is inside an empty
	// container (array or object) with no content yet.
	// Examples:
	//   YAML flow: "[]" with cursor between brackets
	//   JSONC:     "[]" or "{}" with cursor between brackets
	SyntacticPositionEmptyContainer
)

// String returns a human-readable name for the syntactic position.
func (p SyntacticPosition) String() string {
	switch p {
	case SyntacticPositionUnknown:
		return "unknown"
	case SyntacticPositionKeyField:
		return "keyField"
	case SyntacticPositionValueField:
		return "valueField"
	case SyntacticPositionSequenceItem:
		return "sequenceItem"
	case SyntacticPositionStringContent:
		return "stringContent"
	case SyntacticPositionEmptyContainer:
		return "emptyContainer"
	default:
		return "unknown"
	}
}

// SyntacticStyle represents the syntactic style being used at the cursor position.
// This distinguishes between indentation-based YAML (block style), delimiter-based
// YAML (flow style), and JSON/JSONC syntax.
//
// Key insight: JSON is a subset of YAML 1.2. Flow-style YAML uses the same
// delimiter syntax as JSON ({}, [], commas), so they can share detection logic.
type SyntacticStyle int

const (
	// SyntacticStyleUnknown indicates the style could not be determined.
	SyntacticStyleUnknown SyntacticStyle = iota

	// SyntacticStyleBlockYAML indicates indentation-based YAML syntax.
	// Structure is determined by indentation levels and markers like "- ".
	// Examples:
	//   resources:
	//     myResource:
	//       type: aws/lambda/function
	SyntacticStyleBlockYAML

	// SyntacticStyleFlowYAML indicates inline/flow-style YAML syntax.
	// Structure is determined by delimiters: {}, [], commas.
	// Examples:
	//   resources: {myResource: {type: "aws/lambda/function"}}
	//   exclude: [item1, item2]
	SyntacticStyleFlowYAML

	// SyntacticStyleJSONC indicates JSON or JSONC syntax.
	// Like flow YAML, structure is determined by delimiters.
	// Examples:
	//   {"resources": {"myResource": {"type": "aws/lambda/function"}}}
	SyntacticStyleJSONC
)

// String returns a human-readable name for the syntactic style.
func (s SyntacticStyle) String() string {
	switch s {
	case SyntacticStyleUnknown:
		return "unknown"
	case SyntacticStyleBlockYAML:
		return "blockYAML"
	case SyntacticStyleFlowYAML:
		return "flowYAML"
	case SyntacticStyleJSONC:
		return "jsonc"
	default:
		return "unknown"
	}
}

// IsFlowStyle returns true if the style uses delimiter-based syntax (flow YAML or JSONC).
// This is useful because flow YAML and JSONC share the same structural delimiters
// ({}, [], commas) and can use the same position detection logic.
func (s SyntacticStyle) IsFlowStyle() bool {
	return s == SyntacticStyleFlowYAML || s == SyntacticStyleJSONC
}

// IsBlockStyle returns true if the style uses indentation-based syntax (block YAML).
func (s SyntacticStyle) IsBlockStyle() bool {
	return s == SyntacticStyleBlockYAML
}

// QuoteType indicates the type of quotes enclosing a string value.
type QuoteType int

const (
	// QuoteTypeNone indicates no quotes or unknown quote type.
	QuoteTypeNone QuoteType = iota

	// QuoteTypeSingle indicates single quotes ('value').
	QuoteTypeSingle

	// QuoteTypeDouble indicates double quotes ("value").
	QuoteTypeDouble
)

// SyntacticContext captures the syntactic editing context at a cursor position.
// It provides a unified interface for completion generation regardless of
// the underlying document format.
type SyntacticContext struct {
	// Position indicates what kind of syntactic position the cursor is at.
	Position SyntacticPosition

	// Style indicates the syntactic style (block YAML, flow YAML, or JSONC).
	Style SyntacticStyle

	// TypedPrefix is the text being typed at the current position.
	// This is used for filtering completion suggestions.
	TypedPrefix string

	// QuoteType indicates the enclosing quote style if inside a string.
	QuoteType QuoteType

	// InSubstitution is true if the cursor is inside a ${...} substitution.
	InSubstitution bool

	// SubstitutionText is the text inside the current substitution, if any.
	SubstitutionText string
}

// IsAtKeyPosition returns true if the cursor is at a key/field name position.
func (ctx *SyntacticContext) IsAtKeyPosition() bool {
	return ctx.Position == SyntacticPositionKeyField
}

// IsAtValuePosition returns true if the cursor is at a value position.
func (ctx *SyntacticContext) IsAtValuePosition() bool {
	return ctx.Position == SyntacticPositionValueField
}

// IsAtSequenceItemPosition returns true if the cursor is at a sequence item position.
func (ctx *SyntacticContext) IsAtSequenceItemPosition() bool {
	return ctx.Position == SyntacticPositionSequenceItem
}

// IsAtStringContent returns true if the cursor is inside a string value.
func (ctx *SyntacticContext) IsAtStringContent() bool {
	return ctx.Position == SyntacticPositionStringContent
}

// IsAtEmptyContainer returns true if the cursor is inside an empty container.
func (ctx *SyntacticContext) IsAtEmptyContainer() bool {
	return ctx.Position == SyntacticPositionEmptyContainer
}
