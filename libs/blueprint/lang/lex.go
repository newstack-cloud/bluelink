package lang

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// Lex modes drive the dispatch in nextToken. The mode stack is pushed/popped
// as the lexer enters and leaves string literals and ${..} interpolations.
type lexMode int

const (
	// Outside any string literal or ${..} interpolation.
	modeNormal lexMode = iota
	// Inside "..." (single-line string)
	modeSinglestring
	// Inside """...""" (multi-line string)
	modeMultistring
	// Inside ${...} interpolation, expression grammar applies.
	modeInterpolation
)

type lexer struct {
	src string
	// Current byte offset into src.
	pos int
	// 1-based current line
	line int
	// 1-based current column, rune-counted
	col int
	// Stack of lex modes, with the current mode
	// at the top (end of the slice).
	modes []lexMode
	// Indentation prefix stripped from each line of the current multi-line
	// string. Set on entering modeMultistring (the whitespace preceding the
	// closing """ on its own line) and cleared on exit. Empty when not inside
	// a multi-line string, or when the closing """ sits at column 1.
	multilineStrip string
	// Brace depth inside each in-progress ${..} interpolation. Pushed (with
	// value 1) when entering modeInterpolation, decremented on `}` and popped
	// when it reaches 0 — that's the `}` that closes the interpolation.
	// `{` inside the interpolation increments the top so a `}` closing an
	// object literal doesn't prematurely close the interpolation.
	interpBraceDepth []int
	// Collects multiple errors during lexing.
	diags *diagnostics
}

func newLexer(src string) *lexer {
	return &lexer{
		src:   strings.ReplaceAll(src, "\r\n", "\n"),
		line:  1,
		col:   1,
		modes: []lexMode{modeNormal},
		diags: &diagnostics{max: 10, source: src},
	}
}

func (l *lexer) nextToken() *Token {
	for !l.atEOF() {
		startPos := l.currentPos()
		tkn, err := l.nextTokenInner()
		if err != nil {
			l.diags.add(err)
			if l.currentPos() == startPos {
				// Ensure progress, prevent infinite loop.
				l.consume()
			}
			continue
		}
		return tkn
	}

	return l.eofToken()
}

func (l *lexer) nextTokenInner() (*Token, error) {
	mode := l.currentMode()

	switch mode {
	case modeNormal, modeInterpolation:
		return l.nextExprToken()
	case modeSinglestring:
		return l.nextStringContent(false)
	case modeMultistring:
		return l.nextStringContent(true)
	}

	return nil, l.errf("invalid lexer mode: %d", mode)
}

func (l *lexer) nextExprToken() (*Token, error) {
	l.skipSpacesAndTabs()
	if l.atEOF() {
		return l.eofToken(), nil
	}

	char := l.peek()
	switch {
	case char == '\n':
		return l.lexNewline(), nil
	case char == '#':
		return l.lexComment(), nil
	case isIdentStartChar(char):
		return l.lexIdentOrKeyword(), nil
	case char == '"':
		return l.openString()
	case isDigit(char) || char == '-':
		return l.lexNumber()
	}

	return l.lexPunctOrOperator()
}

func (l *lexer) skipSpacesAndTabs() {
	l.skipWhile(func(r rune) bool {
		return r == ' ' || r == '\t'
	})
}

func (l *lexer) eofToken() *Token {
	return makeToken(TokenEOF, "", l.currentPos(), l)
}

func (l *lexer) lexIdentOrKeyword() *Token {
	start := l.currentPos()
	word := l.takeWhile(isIdentChar)
	return classifyIdentOrKeyword(word, start, l)
}

func (l *lexer) lexNumber() (*Token, error) {
	start := l.currentPos()
	sb := strings.Builder{}
	if l.consumeChar('-') {
		sb.WriteRune('-')
	}
	intPart := l.takeWhile(isDigit)
	if intPart == "" {
		return nil, l.errf("expected digit")
	}
	sb.WriteString(intPart)

	// Float? both sides of '.' required as per spec.
	if l.peek() == '.' && isDigit(rune(l.peekByte(1))) {
		l.consume() // Consume the '.'
		sb.WriteRune('.')
		fracPart := l.takeWhile(isDigit)
		if fracPart == "" {
			return nil, l.errf("expected digit after decimal point")
		}
		sb.WriteString(fracPart)
		return makeToken(TokenFloatLiteral, sb.String(), start, l), nil
	}

	return makeToken(TokenIntLiteral, sb.String(), start, l), nil
}

func (l *lexer) lexPunctOrOperator() (*Token, error) {
	start := l.currentPos()
	// Try multi-char operators first, in longest-to-shortest order.
	switch {
	case l.consumePrefix("=="):
		return makeToken(TokenEq, "==", start, l), nil
	case l.consumePrefix("!="):
		return makeToken(TokenNeq, "!=", start, l), nil
	case l.consumePrefix("<="):
		return makeToken(TokenLte, "<=", start, l), nil
	case l.consumePrefix(">="):
		return makeToken(TokenGte, ">=", start, l), nil
	case l.consumePrefix("&&"):
		return makeToken(TokenAnd, "&&", start, l), nil
	case l.consumePrefix("||"):
		return makeToken(TokenOr, "||", start, l), nil
	}

	// Now try single-char tokens.
	char := l.consume()
	switch char {
	case '[':
		return makeToken(TokenLeftBracket, "[", start, l), nil
	case ']':
		return makeToken(TokenRightBracket, "]", start, l), nil
	case '(':
		return makeToken(TokenLeftParen, "(", start, l), nil
	case ')':
		return makeToken(TokenRightParen, ")", start, l), nil
	case '{':
		if l.currentMode() == modeInterpolation {
			l.interpBraceDepth[len(l.interpBraceDepth)-1]++
		}
		return makeToken(TokenLeftBrace, "{", start, l), nil
	case '}':
		if l.currentMode() == modeInterpolation {
			top := len(l.interpBraceDepth) - 1
			l.interpBraceDepth[top]--
			if l.interpBraceDepth[top] == 0 {
				l.interpBraceDepth = l.interpBraceDepth[:top]
				l.popMode()
				return makeToken(TokenInterpolationEnd, "}", start, l), nil
			}
		}
		return makeToken(TokenRightBrace, "}", start, l), nil
	case ':':
		return makeToken(TokenColon, ":", start, l), nil
	case '=':
		return makeToken(TokenAssign, "=", start, l), nil
	case ',':
		return makeToken(TokenComma, ",", start, l), nil
	case '.':
		return makeToken(TokenPeriod, ".", start, l), nil
	case '<':
		return makeToken(TokenLt, "<", start, l), nil
	case '>':
		return makeToken(TokenGt, ">", start, l), nil
	case '*':
		return makeToken(TokenStar, "*", start, l), nil
	case '/':
		return makeToken(TokenSlash, "/", start, l), nil
	case '!':
		return makeToken(TokenNot, "!", start, l), nil
	}

	return nil, l.errfAt(start, "unexpected character: %q", string(char))
}

func (l *lexer) lexComment() *Token {
	start := l.currentPos()
	l.consume() // Consume the '#'
	commentText := l.takeWhile(func(r rune) bool {
		return r != '\n'
	})
	return makeToken(TokenComment, commentText, start, l)
}

func (l *lexer) lexNewline() *Token {
	start := l.currentPos()
	l.consume() // Consume the '\n'
	return makeToken(TokenNewline, "\n", start, l)
}

func (l *lexer) openString() (*Token, error) {
	start := l.currentPos()

	if l.consumePrefix("\"\"\"") {
		if l.peek() != '\n' {
			return nil, l.errf("opening \"\"\" must be followed by a line break")
		}
		l.consume()

		strip, err := l.scanMultilineStripIndent()
		if err != nil {
			return nil, err
		}
		l.multilineStrip = strip

		if err := l.skipMultilineStrip(); err != nil {
			return nil, err
		}

		l.pushMode(modeMultistring)
		return makeToken(TokenStringStart, "\"\"\"", start, l), nil
	}

	l.consume()
	l.pushMode(modeSinglestring)
	return makeToken(TokenStringStart, "\"", start, l), nil
}

func (l *lexer) nextStringContent(isMultiline bool) (*Token, error) {
	start := l.currentPos()
	var sb strings.Builder

	for !l.atEOF() {
		if l.atStringClose(isMultiline) {
			if sb.Len() > 0 {
				value := sb.String()
				if isMultiline {
					// The line break immediately before the closing """
					// is not part of the value (spec).
					value = strings.TrimSuffix(value, "\n")
					return makeToken(TokenMultilineStringLiteral, value, start, l), nil
				}
				return makeToken(TokenStringLiteral, value, start, l), nil
			}
			return l.closeString(start, isMultiline), nil
		}

		if !isMultiline && l.peek() == '\n' {
			return nil, l.errf("unterminated string literal: newline in single-line string")
		}

		if l.hasPrefix("${") {
			if sb.Len() > 0 {
				return makeToken(stringContentTokenType(isMultiline), sb.String(), start, l), nil
			}
			l.consumePrefix("${")
			l.pushMode(modeInterpolation)
			l.interpBraceDepth = append(l.interpBraceDepth, 1)
			return makeToken(TokenInterpolationStart, "${", start, l), nil
		}

		// Only \" is recognised as an escape (spec); every other char is
		// taken literally. In multi-line strings " is allowed unescaped.
		if !isMultiline && l.hasPrefix("\\\"") {
			l.consumePrefix("\\\"")
			sb.WriteRune('"')
			continue
		}

		ch := l.consume()
		sb.WriteRune(ch)

		if isMultiline && ch == '\n' {
			if err := l.skipMultilineStrip(); err != nil {
				return nil, err
			}
		}
	}

	return nil, l.errf("unexpected end of input in string literal")
}

func (l *lexer) atStringClose(isMultiline bool) bool {
	if isMultiline {
		return l.hasPrefix("\"\"\"")
	}
	return l.peek() == '"'
}

func (l *lexer) closeString(start source.Position, isMultiline bool) *Token {
	closer := "\""
	if isMultiline {
		closer = "\"\"\""
		l.multilineStrip = ""
	}
	l.consumePrefix(closer)
	l.popMode()
	return makeToken(TokenStringEnd, closer, start, l)
}

func stringContentTokenType(isMultiline bool) TokenType {
	if isMultiline {
		return TokenMultilineStringLiteral
	}
	return TokenStringLiteral
}

func (l *lexer) hasPrefix(prefix string) bool {
	return strings.HasPrefix(l.src[l.pos:], prefix)
}

// Textual scan only — the first """ found is treated as the closer, even if
// it sits inside a ${..} interpolation. Nested multi-line strings inside
// interpolation expressions are not currently supported.
func (l *lexer) scanMultilineStripIndent() (string, error) {
	pos := l.pos
	lineStart := pos
	for pos < len(l.src) {
		if pos+3 <= len(l.src) && l.src[pos:pos+3] == "\"\"\"" {
			indent := l.src[lineStart:pos]
			for _, r := range indent {
				if r != ' ' && r != '\t' {
					return "", l.errf("closing \"\"\" must be on its own line")
				}
			}
			return indent, nil
		}
		if l.src[pos] == '\n' {
			lineStart = pos + 1
		}
		pos += 1
	}
	return "", l.errf("unterminated multi-line string")
}

func (l *lexer) skipMultilineStrip() error {
	for _, c := range l.multilineStrip {
		if l.atEOF() || l.peek() == '\n' {
			// Blank (or EOF) line shorter than the strip is allowed.
			return nil
		}
		if l.peek() != c {
			return l.errf("multi-line string line not indented correctly")
		}
		l.consume()
	}
	return nil
}

func isIdentStartChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		char == '_'
}

func isIdentChar(char rune) bool {
	return isIdentStartChar(char) ||
		(char >= '0' && char <= '9') ||
		char == '-'
}

func isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

func isLetter(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}

	char, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return char
}

// Allows lookahead for chars beyond the next one that take up a single byte each,
// safe for ASCII-only lookahead patterns like ${ and """.
func (l *lexer) peekByte(offset int) byte {
	pos := l.pos + offset
	if pos >= len(l.src) {
		return 0
	}

	return l.src[pos]
}

func (l *lexer) consumeChar(char rune) bool {
	if l.peek() == char {
		l.consume()
		return true
	}

	return false
}

func (l *lexer) consumePrefix(prefix string) bool {
	if l.pos+len(prefix) > len(l.src) {
		return false
	}

	if l.src[l.pos:l.pos+len(prefix)] != prefix {
		return false
	}

	// Consume the prefix, rune by rune to correctly
	// update line and column numbers.
	end := l.pos + len(prefix)
	for l.pos < end {
		l.consume()
	}

	return true
}

func (l *lexer) skipWhile(predicate func(rune) bool) {
	for !l.atEOF() && predicate(l.peek()) {
		l.consume()
	}
}

func (l *lexer) takeWhile(predicate func(rune) bool) string {
	start := l.pos
	l.skipWhile(predicate)
	return l.src[start:l.pos]
}

func (l *lexer) consume() rune {
	if l.pos >= len(l.src) {
		return 0
	}

	char, size := utf8.DecodeRuneInString(l.src[l.pos:])
	l.pos += size
	if char == '\n' {
		l.line += 1
		l.col = 1
	} else {
		l.col += 1
	}

	return char
}

func (l *lexer) atEOF() bool {
	return l.pos >= len(l.src)
}

func (l *lexer) currentPos() source.Position {
	return source.Position{
		Line:   l.line,
		Column: l.col,
	}
}

func (l *lexer) pushMode(mode lexMode) {
	l.modes = append(l.modes, mode)
}

func (l *lexer) popMode() {
	if len(l.modes) == 0 {
		return
	}

	l.modes = l.modes[:len(l.modes)-1]
}

func (l *lexer) currentMode() lexMode {
	if len(l.modes) == 0 {
		// This should never happen, but if it does,
		// treat it as normal mode.
		return modeNormal
	}
	return l.modes[len(l.modes)-1]
}

func (l *lexer) errf(format string, args ...any) error {
	return l.errfAt(l.currentPos(), format, args...)
}

func (l *lexer) errfAt(pos source.Position, format string, args ...any) error {
	return &LexError{
		Message:    fmt.Sprintf(format, args...),
		SourceMeta: &source.Meta{Position: pos},
	}
}

func classifyIdentOrKeyword(word string, pos source.Position, l *lexer) *Token {
	switch word {
	case "true", "false":
		return makeToken(TokenBoolLiteral, word, pos, l)
	case "none":
		return makeToken(TokenNoneLiteral, word, pos, l)
	}

	if TokenType, isKeyword := keywordTable[word]; isKeyword {
		return makeToken(TokenType, word, pos, l)
	}

	return makeToken(TokenIdent, word, pos, l)
}

func makeToken(
	TokenType TokenType,
	value string,
	pos source.Position,
	l *lexer,
) *Token {
	return &Token{
		Type:  TokenType,
		Value: value,
		Start: pos,
		End:   l.currentPos(),
	}
}

var keywordTable = map[string]TokenType{
	"variable":    TokenKeywordVariable,
	"value":       TokenKeywordValue,
	"data":        TokenKeywordData,
	"resource":    TokenKeywordResource,
	"include":     TokenKeywordInclude,
	"export":      TokenKeywordExport,
	"metadata":    TokenKeywordMetadata,
	"spec":        TokenKeywordSpec,
	"select":      TokenKeywordSelect,
	"filter":      TokenKeywordFilter,
	"foreach":     TokenKeywordForeach,
	"as":          TokenKeywordAs,
	"by":          TokenKeywordBy,
	"label":       TokenKeywordLabel,
	"version":     TokenKeywordVersion,
	"transform":   TokenKeywordTransform,
	"not":         TokenKeywordNot,
	"in":          TokenKeywordIn,
	"has":         TokenKeywordHas,
	"key":         TokenKeywordKey,
	"contains":    TokenKeywordContains,
	"starts":      TokenKeywordStarts,
	"with":        TokenKeywordWith,
	"ends":        TokenKeywordEnds,
	"string":      TokenKeywordString,
	"integer":     TokenKeywordInteger,
	"float":       TokenKeywordFloat,
	"boolean":     TokenKeywordBoolean,
	"array":       TokenKeywordArray,
	"object":      TokenKeywordObject,
	"variables":   TokenKeywordVariables,
	"values":      TokenKeywordValues,
	"datasources": TokenKeywordDatasources,
	"resources":   TokenKeywordResources,
	"children":    TokenKeywordChildren,
	"elem":        TokenKeywordElem,
	"i":           TokenKeywordI,
}
