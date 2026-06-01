package lang

import (
	"fmt"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type parser struct {
	lex *lexer
	// Lookeahead ring. The grammar needs at most 2 tokens of lookahead.
	// (declaration-header dispatch, cmp/eq operator chains, "select by label").
	buf []*Token
	// Tracks nesting inside ( ) and [ ]. When > 0, newline
	// tokens are insignificant and skipped during fetch.
	groupingDepth int
	diags         *diagnostics
}

func (p *parser) parse() (*schema.Blueprint, error) {
	blueprint := &schema.Blueprint{}
	for {
		p.skipNewlines()
		if p.peek().Type == TokenEOF {
			break
		}
		if err := p.parseTopLevelItem(blueprint); err != nil {
			p.diags.add(err)
			p.synchronise()
		}
	}

	if blueprint.Version == nil {
		p.diags.add(&ParseError{
			Message: "missing required 'version' directive",
		})
	}

	return blueprint, p.diags.asError()
}

func (p *parser) parseTopLevelItem(bp *schema.Blueprint) error {
	switch p.peek().Type {
	case TokenKeywordVersion:
		return p.parseVersionDirective(bp)
	case TokenKeywordTransform:
		return p.parseTransformDirective(bp)
	case TokenKeywordVariable:
		return p.parseVariableDecl(bp)
	case TokenKeywordValue:
		return p.parseValueDecl(bp)
	case TokenKeywordData:
		return p.parseDataDecl(bp)
	case TokenKeywordResource:
		return p.parseResourceDecl(bp)
	case TokenKeywordInclude:
		return p.parseIncludeDecl(bp)
	case TokenKeywordMetadata:
		return p.parseMetadataBlock(bp)
	case TokenKeywordExport:
		return p.parseExportDecl(bp)
	default:
		tkn := p.peek()
		return p.errf(tkn.Start, "unexpected %s at top level", tkn.Type)
	}
}

func (p *parser) parseVersionDirective(bp *schema.Blueprint) error {
	p.advance() // consume 'version' keyword
	value, meta, err := p.parsePlainStringLiteral()
	if err != nil {
		return err
	}

	if bp.Version != nil {
		return p.errf(
			meta.Position,
			"version directive first declared at %d:%d, duplicated",
			bp.Version.SourceMeta.Position.Line,
			bp.Version.SourceMeta.Position.Column,
		)
	}

	bp.Version = &core.ScalarValue{
		StringValue: &value,
		SourceMeta:  meta,
	}

	return nil
}

func (p *parser) parseTransformDirective(bp *schema.Blueprint) error {
	keyword := p.advance() // consume 'transform' keyword
	if bp.Transform != nil {
		return p.errf(keyword.Start, "transform directive already declared")
	}

	switch p.peek().Type {
	case TokenStringStart:
		return p.parseSingleTransformDirective(bp)
	case TokenLeftBracket:
		return p.parseMultipleTransformsDirective(bp)
	default:
		tkn := p.peek()
		return p.errf(tkn.Start, "expected string literal for transform directive, got %s", tkn.Type)
	}
}

func (p *parser) parseSingleTransformDirective(bp *schema.Blueprint) error {
	value, meta, err := p.parsePlainStringLiteral()
	if err != nil {
		return err
	}

	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values:     []string{value},
			SourceMeta: []*source.Meta{meta},
		},
	}

	return nil
}

func (p *parser) parseMultipleTransformsDirective(bp *schema.Blueprint) error {
	p.advance() // consume '['

	var values []string
	var metas []*source.Meta

	for {
		if p.peek().Type == TokenRightBracket {
			break
		}
		value, meta, err := p.parsePlainStringLiteral()
		if err != nil {
			return err
		}
		values = append(values, value)
		metas = append(metas, meta)

		if !p.match(TokenComma) {
			break
		}
	}

	closeBracket, err := p.expect(TokenRightBracket)
	if err != nil {
		return err
	}

	if len(values) == 0 {
		return p.errf(closeBracket.Start, "expected at least one transform in list")
	}

	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values:     values,
			SourceMeta: metas,
		},
	}

	return nil
}

func (p *parser) advance() *Token {
	p.skipGroupedNewLines()
	p.fill(1)
	tkn := p.buf[0]
	p.buf = p.buf[1:]

	switch tkn.Type {
	case TokenLeftParen, TokenLeftBracket:
		p.groupingDepth += 1
	case TokenRightParen, TokenRightBracket:
		if p.groupingDepth > 0 {
			p.groupingDepth -= 1
		}
	}

	return tkn
}

// Look past new lines without consuming, to test for a continuation operator.
func (p *parser) peekPastNewlines() *Token {
	i := 0
	for {
		p.fill(i + 1)
		if p.buf[i].Type != TokenNewline {
			return p.buf[i]
		}
		i += 1
	}
}

func (p *parser) peek() *Token {
	p.skipGroupedNewLines()
	p.fill(1)
	return p.buf[0]
}

func (p *parser) skipGroupedNewLines() {
	if p.groupingDepth == 0 {
		return
	}

	for {
		p.fill(1)
		if p.buf[0].Type != TokenNewline {
			return
		}
		p.buf = p.buf[1:]
	}
}

// On a parse error inside a declaration, skip tokens until the next
// declaration header or top-level boundary, then resume.
func (p *parser) synchronise() {
	// Reset grouping depth to avoid skipping significant
	// newlines due to an unclosed ( or [ in the broken section.
	p.groupingDepth = 0

	for {
		switch p.peek().Type {
		case TokenEOF,
			TokenKeywordVariable, TokenKeywordValue, TokenKeywordData,
			TokenKeywordResource, TokenKeywordInclude, TokenKeywordExport,
			TokenKeywordVersion, TokenKeywordTransform, TokenKeywordMetadata:
			return
		}
		p.advance()
	}
}

func (p *parser) fill(n int) {
	for len(p.buf) < n {
		tkn := p.lex.nextToken()
		if tkn.Type == TokenComment {
			continue
		}
		p.buf = append(p.buf, tkn)
	}
}

func newParser(src string) *parser {
	lex := newLexer(src)
	return &parser{
		lex: lex,
		// The parser shares the lexer's diagnostics sink so lex and parse errors
		// accumulate together and surface in a single pass.
		diags: lex.diags,
	}
}

func (p *parser) peekAt(n int) *Token {
	p.skipGroupedNewLines()
	p.fill(n + 1)
	return p.buf[n]
}

func (p *parser) match(tt TokenType) bool {
	if p.peek().Type == tt {
		p.advance()
		return true
	}
	return false
}

func (p *parser) matchAcrossNewlines(tts ...TokenType) (*Token, bool) {
	next := p.peekPastNewlines()

	if slices.Contains(tts, next.Type) {
		p.skipNewlines() // commit: newline was a continuation, not a separator
		op := p.advance()
		p.skipNewlines() // tolerate operator-at-end-of-line
		return op, true
	}

	return nil, false
}

func (p *parser) expect(tt TokenType) (*Token, error) {
	tkn := p.peek()
	if tkn.Type != tt {
		return nil, p.errf(tkn.Start, "expected %s, got %s", tt, tkn.Type)
	}
	return p.advance(), nil
}

func (p *parser) consumeSeparators() bool {
	consumed := false
	for {
		tt := p.peek().Type
		if tt != TokenComma && tt != TokenNewline {
			return consumed
		}
		p.advance()
		consumed = true
	}
}

func (p *parser) skipNewlines() {
	for {
		p.fill(1)
		if p.buf[0].Type != TokenNewline {
			return
		}
		p.buf = p.buf[1:]
	}
}

func (p *parser) errf(pos source.Position, format string, args ...any) error {
	return &ParseError{
		Message:    fmt.Sprintf(format, args...),
		SourceMeta: &source.Meta{Position: pos},
	}
}

func sourceMetaFromToken(tkn *Token) *source.Meta {
	return &source.Meta{
		Position:    tkn.Start,
		EndPosition: &tkn.End,
	}
}
