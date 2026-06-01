package docmodel

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// Parses a resource `{ ... }` body, dispatching the named
// sub-blocks (spec, metadata), the select-by-label selector and the inline
// statements (foreach) and treating everything else as a `field = value` entry.
func (b *bpBuilder) parseResourceBody(node *UnifiedNode) {
	if b.kind() != lang.TokenLeftBrace {
		return
	}
	b.advance()

	for !b.atEnd() {
		b.skipSeparators()
		switch b.kind() {
		case lang.TokenRightBrace:
			setNodeEnd(node, b.advance())
			return
		case lang.TokenEOF:
			return
		case lang.TokenKeywordSpec:
			b.appendChild(node, b.parseNamedFieldsBlock("spec", node))
		case lang.TokenKeywordMetadata:
			b.appendChild(node, b.parseNamedFieldsBlock("metadata", node))
		case lang.TokenKeywordSelect:
			b.appendChild(node, b.parseSelectBlock(node))
		case lang.TokenKeywordForeach:
			b.appendChild(node, b.parseForeachField(node))
		default:
			b.appendChild(node, b.parseField(node))
		}
	}
}

// Parses a data source `{ ... }` body: filter and export
// statements, plus metadata/spec sub-blocks and any remaining `field = value`.
func (b *bpBuilder) parseDataBody(node *UnifiedNode) {
	if b.kind() != lang.TokenLeftBrace {
		return
	}
	b.advance()

	var filters, exports *UnifiedNode
	for !b.atEnd() {
		b.skipSeparators()
		switch b.kind() {
		case lang.TokenRightBrace:
			setNodeEnd(node, b.advance())
			return
		case lang.TokenEOF:
			return
		case lang.TokenKeywordFilter:
			filters = b.appendFilter(node, filters)
		case lang.TokenKeywordExport:
			exports = b.appendExport(node, exports)
		case lang.TokenKeywordMetadata:
			b.appendChild(node, b.parseNamedFieldsBlock("metadata", node))
		case lang.TokenKeywordSpec:
			b.appendChild(node, b.parseNamedFieldsBlock("spec", node))
		default:
			b.appendChild(node, b.parseField(node))
		}
	}
}

// Parses a generic `{ key = value ... }` block into node's
// children. Used for variable/value/export bodies, spec, metadata and any
// nested object-shaped sub-block.
func (b *bpBuilder) parseFieldsBlock(node *UnifiedNode) {
	if b.kind() != lang.TokenLeftBrace {
		return
	}
	b.advance()

	for !b.atEnd() {
		b.skipSeparators()
		if b.kind() == lang.TokenRightBrace {
			setNodeEnd(node, b.advance())
			return
		}
		if b.kind() == lang.TokenEOF {
			// Unclosed block (in-progress edit): extend the range to EOF so the
			// cursor on a trailing blank line still resolves inside this block.
			setNodeEnd(node, b.cur())
			return
		}
		b.appendChild(node, b.parseField(node))
	}
}

// Parses `<keyword> { fields }` into a named mapping node
// (e.g. `spec { ... }` -> FieldName "spec").
func (b *bpBuilder) parseNamedFieldsBlock(name string, parent *UnifiedNode) *UnifiedNode {
	keyword := b.advance()
	node := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: name,
		KeyRange:  rangePtr(tokRange(keyword, keyword)),
		Parent:    parent,
		Range:     tokRange(keyword, keyword),
		TSKind:    "bp_block",
	}
	b.parseFieldsBlock(node)
	return node
}

// Parses `select by label { ... }` into the canonical
// linkSelector mapping: label entries live under a `byLabel` child, while an
// `exclude` entry is surfaced directly on linkSelector.
func (b *bpBuilder) parseSelectBlock(parent *UnifiedNode) *UnifiedNode {
	keyword := b.advance() // select
	if b.kind() == lang.TokenKeywordBy {
		b.advance()
	}
	if b.kind() == lang.TokenKeywordLabel {
		b.advance()
	}

	linkSelector := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: "linkSelector",
		KeyRange:  rangePtr(tokRange(keyword, keyword)),
		Parent:    parent,
		Range:     tokRange(keyword, keyword),
		TSKind:    "bp_link_selector",
	}
	byLabel := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: "byLabel",
		Parent:    linkSelector,
		TSKind:    "bp_block",
	}
	linkSelector.Children = append(linkSelector.Children, byLabel)

	if b.kind() == lang.TokenLeftBrace {
		b.advance()
		for !b.atEnd() {
			b.skipSeparators()
			if b.kind() == lang.TokenRightBrace {
				setNodeEnd(linkSelector, b.advance())
				break
			}

			if b.kind() == lang.TokenEOF {
				break
			}

			field := b.parseField(byLabel)
			if field.FieldName == "exclude" {
				field.Parent = linkSelector
				linkSelector.Children = append(linkSelector.Children, field)
			} else {
				byLabel.Children = append(byLabel.Children, field)
			}
		}
	}

	return linkSelector
}

// Parses the inline `foreach <expr>` statement into a scalar
// field named "foreach".
func (b *bpBuilder) parseForeachField(parent *UnifiedNode) *UnifiedNode {
	keyword := b.advance() // foreach
	b.skipComments()
	val := b.consumeExprScalar(parent)
	b.nameNode(val, "foreach", tokRange(keyword, keyword))
	return val
}

// Parses a `filter <field> <operator> <search>` statement, adding a
// mapping (field/operator/search) to the data source's `filters` sequence.
func (b *bpBuilder) appendFilter(dataNode, filters *UnifiedNode) *UnifiedNode {
	if filters == nil {
		filters = &UnifiedNode{
			Kind:      NodeKindSequence,
			Index:     -1,
			FieldName: "filters",
			Parent:    dataNode,
			TSKind:    "bp_filters",
		}
		dataNode.Children = append(dataNode.Children, filters)
	}

	keyword := b.advance() // filter
	filter := &UnifiedNode{
		Kind:   NodeKindMapping,
		Index:  len(filters.Children),
		Parent: filters,
		Range:  tokRange(keyword, keyword),
		TSKind: "bp_filter",
	}

	fieldVal, fieldRange := b.parseKey()
	fieldChild := scalarNode(fieldVal, "!!str", fieldRange, filter)
	b.nameNode(fieldChild, "field", fieldRange)
	filter.Children = append(filter.Children, fieldChild)

	if opVal, opRange, ok := b.parseFilterOperator(); ok {
		opChild := scalarNode(opVal, "!!str", opRange, filter)
		b.nameNode(opChild, "operator", opRange)
		filter.Children = append(filter.Children, opChild)
	}

	b.skipComments()
	if b.isValueStart() {
		search := b.parseValueExpr(filter)
		b.nameNode(search, "search", search.Range)
		filter.Children = append(filter.Children, search)
	}

	filters.Children = append(filters.Children, filter)
	return filters
}

// Consumes the operator run of a data-source filter (symbol
// operators like == / != and word operators like `has key`, `starts with`).
func (b *bpBuilder) parseFilterOperator() (string, source.Range, bool) {
	var parts []string
	var start, last lang.Token
	count := 0
	for isFilterOperatorToken(b.kind()) {
		tkn := b.advance()
		if count == 0 {
			start = tkn
		}
		parts = append(parts, tkn.Value)
		last = tkn
		count++
	}

	if count == 0 {
		return "", source.Range{}, false
	}

	return strings.Join(parts, " "), posRange(start.Start, last.End), true
}

// Parses a data-source `export <field> [as <name>]: <type> [ { ... } ]`
// (or `export *`) statement into the canonical `exports` mapping.
func (b *bpBuilder) appendExport(dataNode, exports *UnifiedNode) *UnifiedNode {
	if exports == nil {
		exports = &UnifiedNode{
			Kind:      NodeKindMapping,
			Index:     -1,
			FieldName: "exports",
			Parent:    dataNode,
			TSKind:    "bp_exports",
		}
		dataNode.Children = append(dataNode.Children, exports)
	}

	keyword := b.advance() // export
	if b.kind() == lang.TokenStar {
		star := b.advance()
		wildcard := scalarNode("*", "!!str", tokRange(star, star), exports)
		b.nameNode(wildcard, "*", tokRange(star, star))
		exports.Children = append(exports.Children, wildcard)
		return exports
	}

	field, fieldRange := b.parseKey()
	exportName, nameRange := field, fieldRange
	aliasFor := ""
	if b.kind() == lang.TokenKeywordAs {
		b.advance()
		exportName, nameRange = b.parseKey()
		aliasFor = field
	}

	export := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: exportName,
		KeyRange:  rangePtr(nameRange),
		Parent:    exports,
		Range:     mergeRange(tokRange(keyword, keyword), nameRange),
		TSKind:    "bp_export",
	}

	if b.kind() == lang.TokenColon {
		b.advance()
		typeVal, typeRange := b.parseTypeRef()
		typeChild := scalarNode(typeVal, "!!str", typeRange, export)
		b.nameNode(typeChild, "type", typeRange)
		export.Children = append(export.Children, typeChild)
	}

	if aliasFor != "" {
		aliasChild := scalarNode(aliasFor, "!!str", fieldRange, export)
		b.nameNode(aliasChild, "aliasFor", fieldRange)
		export.Children = append(export.Children, aliasChild)
	}

	b.parseFieldsBlock(export)
	finalizeNodeRange(export, nameRange)
	exports.Children = append(exports.Children, export)
	return exports
}

// Parses a `key = value` entry, returning the value node with its
// FieldName/KeyRange set to the key.
func (b *bpBuilder) parseField(parent *UnifiedNode) *UnifiedNode {
	key, keyRange := b.parseKey()
	b.skipComments()
	if b.kind() == lang.TokenAssign {
		b.advance()
	}
	b.skipComments()

	val := b.parseValueExpr(parent)
	b.nameNode(val, key, keyRange)
	return val
}

// Parses the value after `=` (or a positional value): a string,
// array literal, object literal, or a bare expression captured as a scalar.
func (b *bpBuilder) parseValueExpr(parent *UnifiedNode) *UnifiedNode {
	b.skipComments()
	switch b.kind() {
	case lang.TokenStringStart:
		value, rng := b.consumeString()
		return scalarNode(value, "!!str", rng, parent)
	case lang.TokenLeftBracket:
		return b.parseArray(parent)
	case lang.TokenLeftBrace:
		return b.parseObject(parent)
	default:
		return b.consumeExprScalar(parent)
	}
}

// Parses a `{ key = value, ... }` object literal into a mapping.
func (b *bpBuilder) parseObject(parent *UnifiedNode) *UnifiedNode {
	open := b.advance() // {
	node := &UnifiedNode{
		Kind:   NodeKindMapping,
		Index:  -1,
		Parent: parent,
		Range:  tokRange(open, open),
		TSKind: "bp_object",
	}

	for !b.atEnd() {
		b.skipSeparators()
		if b.kind() == lang.TokenRightBrace {
			setNodeEnd(node, b.advance())
			break
		}
		if b.kind() == lang.TokenEOF {
			// Unclosed object literal (in-progress edit): extend the range to EOF
			// so the cursor on a trailing blank line still resolves inside it.
			setNodeEnd(node, b.cur())
			break
		}
		node.Children = append(node.Children, b.parseField(node))
	}

	return node
}

// Parses a `[ expr, ... ]` array literal into an indexed sequence.
func (b *bpBuilder) parseArray(parent *UnifiedNode) *UnifiedNode {
	open := b.advance() // [
	node := &UnifiedNode{
		Kind:   NodeKindSequence,
		Index:  -1,
		Parent: parent,
		Range:  tokRange(open, open),
		TSKind: "bp_array",
	}

	index := 0
	for !b.atEnd() {
		b.skipSeparators()
		if b.kind() == lang.TokenRightBracket {
			setNodeEnd(node, b.advance())
			break
		}
		if b.kind() == lang.TokenEOF {
			break
		}
		element := b.parseValueExpr(node)
		element.Index = index
		index += 1
		node.Children = append(node.Children, element)
	}

	return node
}

// Consumes a TokenStringStart..TokenStringEnd run, returning the
// literal text (interpolations rendered with their ${} delimiters) and the full
// source range.
func (b *bpBuilder) consumeString() (string, source.Range) {
	start := b.advance() // string start
	var sb strings.Builder
	last := start
	for !b.atEnd() && b.kind() != lang.TokenStringEnd {
		tkn := b.advance()
		switch tkn.Type {
		case lang.TokenInterpolationStart:
			sb.WriteString("${")
		case lang.TokenInterpolationEnd:
			sb.WriteString("}")
		default:
			sb.WriteString(tkn.Value)
		}
		last = tkn
	}

	if b.kind() == lang.TokenStringEnd {
		last = b.advance()
	}

	return sb.String(), posRange(start.Start, last.End)
}

// Captures a bare expression (reference, function call,
// operators, literal) as a single scalar node. It honours the blueprint
// language's multi-line rule: a newline continues the expression when the token
// before or after it is a binary operator.
func (b *bpBuilder) consumeExprScalar(parent *UnifiedNode) *UnifiedNode {
	start := b.cur()
	var sb strings.Builder
	last := start
	depth := 0
	count := 0
	litTag := ""
	var prevType lang.TokenType

	for !b.atEnd() {
		if depth == 0 && b.isExprTerminator(prevType) {
			break
		}

		tkn := b.advance()
		switch tkn.Type {
		case lang.TokenLeftParen, lang.TokenLeftBracket, lang.TokenLeftBrace:
			depth += 1
		case lang.TokenRightParen, lang.TokenRightBracket, lang.TokenRightBrace:
			if depth > 0 {
				depth -= 1
			}
		}

		if tkn.Type == lang.TokenNewline {
			// Continuation (or in-group) newline: keep scanning, but do not add
			// it to the captured value text.
			continue
		}

		sb.WriteString(tkn.Value)
		last = tkn
		if tag := literalTag(tkn.Type); tag != "" {
			litTag = tag
		}
		prevType = tkn.Type
		count += 1
	}

	if count == 0 {
		return scalarNode("", "!!str", tokRange(start, start), parent)
	}

	tag := "!!str"
	if count == 1 && litTag != "" {
		tag = litTag
	}

	return scalarNode(sb.String(), tag, posRange(start.Start, last.End), parent)
}

// Reports whether the cursor (at grouping depth 0) ends the
// current bare expression. A newline terminates unless it is a continuation
// (the previous or next significant token is a binary operator).
func (b *bpBuilder) isExprTerminator(prevType lang.TokenType) bool {
	switch b.kind() {
	case lang.TokenComma, lang.TokenRightBrace, lang.TokenRightBracket,
		lang.TokenRightParen, lang.TokenComment, lang.TokenEOF:
		return true
	case lang.TokenNewline:
		return !isContinuationOp(prevType) && !isContinuationOp(b.peekPastNewlines())
	default:
		return false
	}
}

// Returns the type of the next token after any newlines and
// comments, without consuming anything.
func (b *bpBuilder) peekPastNewlines() lang.TokenType {
	for j := b.i; j < len(b.tokens); j++ {
		t := b.tokens[j].Type
		if t != lang.TokenNewline && t != lang.TokenComment {
			return t
		}
	}
	return lang.TokenEOF
}

// Parses a key/name token (a bare identifier/keyword or a quoted string)
// and returns its text and source range. A bare dotted run (e.g.
// aws.lambda.dynamodb.accessType) is absorbed into a single key, since the lexer
// emits it as separate ident/period tokens and a single token would only capture
// the first segment.
func (b *bpBuilder) parseKey() (string, source.Range) {
	b.skipComments()
	if b.kind() == lang.TokenStringStart {
		return b.consumeString()
	}

	first := b.advance()
	var sb strings.Builder
	sb.WriteString(first.Value)
	last := first

	for b.kind() == lang.TokenPeriod {
		dot := b.advance()
		sb.WriteString(dot.Value)
		last = dot
		if !isKeySegmentToken(b.kind()) {
			break
		}
		seg := b.advance()
		sb.WriteString(seg.Value)
		last = seg
	}

	return sb.String(), posRange(first.Start, last.End)
}

// isKeySegmentToken reports whether a token following a period can continue a
// dotted key: anything that is not a structural delimiter, separator or
// terminator (so identifiers, keywords and literals all qualify).
func isKeySegmentToken(t lang.TokenType) bool {
	switch t {
	case lang.TokenNewline, lang.TokenComma, lang.TokenComment, lang.TokenEOF,
		lang.TokenAssign, lang.TokenColon, lang.TokenPeriod,
		lang.TokenLeftBrace, lang.TokenRightBrace,
		lang.TokenLeftBracket, lang.TokenRightBracket,
		lang.TokenLeftParen, lang.TokenRightParen:
		return false
	default:
		return true
	}
}

// Parses a name token (an alias for parseKey) used at declaration-name positions.
func (b *bpBuilder) parseName() (string, source.Range) {
	return b.parseKey()
}

// Parses an element-type reference (`aws/lambda/function`) or a
// builtin type keyword up to the block opening or end of line.
func (b *bpBuilder) parseTypeRef() (string, source.Range) {
	b.skipComments()

	var sb strings.Builder
	var start, last lang.Token
	count := 0
	for !b.atEnd() {
		switch b.kind() {
		case lang.TokenLeftBrace, lang.TokenNewline, lang.TokenAssign,
			lang.TokenRightBrace, lang.TokenComma, lang.TokenEOF:
			if count == 0 {
				return "", tokRange(b.cur(), b.cur())
			}
			return sb.String(), posRange(start.Start, last.End)
		}
		tkn := b.advance()
		if count == 0 {
			start = tkn
		}
		sb.WriteString(tkn.Value)
		last = tkn
		count += 1
	}

	if count == 0 {
		return "", tokRange(b.cur(), b.cur())
	}

	return sb.String(), posRange(start.Start, last.End)
}

func (b *bpBuilder) isValueStart() bool {
	switch b.kind() {
	case lang.TokenNewline, lang.TokenComma, lang.TokenComment,
		lang.TokenRightBrace, lang.TokenEOF:
		return false
	default:
		return true
	}
}

func isFilterOperatorToken(t lang.TokenType) bool {
	switch t {
	case lang.TokenEq, lang.TokenNeq, lang.TokenLt, lang.TokenGt,
		lang.TokenLte, lang.TokenGte,
		lang.TokenKeywordNot, lang.TokenKeywordIn, lang.TokenKeywordHas,
		lang.TokenKeywordKey, lang.TokenKeywordContains, lang.TokenKeywordStarts,
		lang.TokenKeywordWith, lang.TokenKeywordEnds:
		return true
	default:
		return false
	}
}

func isContinuationOp(t lang.TokenType) bool {
	switch t {
	case lang.TokenEq, lang.TokenNeq, lang.TokenLt, lang.TokenGt,
		lang.TokenLte, lang.TokenGte, lang.TokenAnd, lang.TokenOr, lang.TokenPeriod:
		return true
	default:
		return false
	}
}

func literalTag(t lang.TokenType) string {
	switch t {
	case lang.TokenIntLiteral:
		return "!!int"
	case lang.TokenFloatLiteral:
		return "!!float"
	case lang.TokenBoolLiteral:
		return "!!bool"
	case lang.TokenNoneLiteral:
		return "!!null"
	default:
		return ""
	}
}

func scalarNode(value, tag string, rng source.Range, parent *UnifiedNode) *UnifiedNode {
	return &UnifiedNode{
		Kind:   NodeKindScalar,
		Index:  -1,
		Value:  value,
		Tag:    tag,
		Range:  rng,
		Parent: parent,
	}
}

func (b *bpBuilder) nameNode(node *UnifiedNode, name string, keyRange source.Range) {
	node.FieldName = name
	kr := keyRange
	node.KeyRange = &kr
	if keyRange.Start != nil {
		if node.Range.Start == nil || posBefore(*keyRange.Start, *node.Range.Start) {
			s := *keyRange.Start
			node.Range.Start = &s
		}
	}
}

func setNodeEnd(node *UnifiedNode, tkn lang.Token) {
	e := tkn.End
	node.Range.End = &e
}

// Ensures a declaration node spans from its name to the end of
// its last child when the closing brace was not captured (e.g. truncated input).
func finalizeNodeRange(node *UnifiedNode, nameRange source.Range) {
	if node.Range.Start == nil {
		node.Range.Start = nameRange.Start
	}

	if len(node.Children) == 0 {
		return
	}

	last := node.Children[len(node.Children)-1]
	if last.Range.End != nil &&
		(node.Range.End == nil || posBefore(*node.Range.End, *last.Range.End)) {
		e := *last.Range.End
		node.Range.End = &e
	}
}

// Fills any node still lacking a range from the union of its
// children's ranges (children first, so unions propagate upward).
func fillMissingRanges(node *UnifiedNode) {
	for _, child := range node.Children {
		fillMissingRanges(child)
	}

	if len(node.Children) == 0 {
		return
	}

	var start, end *source.Position
	for _, child := range node.Children {
		if child.Range.Start != nil &&
			(start == nil || posBefore(*child.Range.Start, *start)) {
			s := *child.Range.Start
			start = &s
		}
		if child.Range.End != nil &&
			(end == nil || posBefore(*end, *child.Range.End)) {
			e := *child.Range.End
			end = &e
		}
	}

	if node.Range.Start == nil {
		node.Range.Start = start
	}

	if node.Range.End == nil {
		node.Range.End = end
	}
}

func tokRange(start, end lang.Token) source.Range {
	s := start.Start
	e := end.End
	return source.Range{Start: &s, End: &e}
}

func posRange(start, end source.Position) source.Range {
	s := start
	e := end
	return source.Range{Start: &s, End: &e}
}

func mergeRange(first, second source.Range) source.Range {
	out := source.Range{}
	if first.Start != nil {
		s := *first.Start
		out.Start = &s
	}
	if second.End != nil {
		e := *second.End
		out.End = &e
	}
	if out.Start == nil {
		out.Start = second.Start
	}
	if out.End == nil {
		out.End = first.End
	}
	return out
}

func rangePtr(r source.Range) *source.Range {
	return &r
}

func posBefore(a, b source.Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}
