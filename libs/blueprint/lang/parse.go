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
	buf []*token
	// Tracks nesting inside ( ) and [ ]. When > 0, newline
	// tokens are insignificant and skipped during fetch.
	groupingDepth int
	diags         *diagnostics
}

func (p *parser) parse() (*schema.Blueprint, error) {
	blueprint := &schema.Blueprint{}
	for {
		p.skipNewlines()
		if p.peek().tokenType == tokenEOF {
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
	switch p.peek().tokenType {
	case tokenKeywordVersion:
		return p.parseVersionDirective(bp)
	case tokenKeywordTransform:
		return p.parseTransformDirective(bp)
	case tokenKeywordVariable:
		return p.parseVariableDecl(bp)
	case tokenKeywordValue:
		return p.parseValueDecl(bp)
	case tokenKeywordData:
		return p.parseDataDecl(bp)
	case tokenKeywordResource:
		return p.parseResourceDecl(bp)
	case tokenKeywordInclude:
		return p.parseIncludeDecl(bp)
	case tokenKeywordMetadata:
		return p.parseMetadataBlock(bp)
	case tokenKeywordExport:
		return p.parseExportDecl(bp)
	default:
		tkn := p.peek()
		return p.errf(tkn.pos, "unexpected %s at top level", tkn.tokenType)
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
		return p.errf(keyword.pos, "transform directive already declared")
	}

	switch p.peek().tokenType {
	case tokenStringStart:
		return p.parseSingleTransformDirective(bp)
	case tokenLeftBracket:
		return p.parseMultipleTransformsDirective(bp)
	default:
		tkn := p.peek()
		return p.errf(tkn.pos, "expected string literal for transform directive, got %s", tkn.tokenType)
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
		if p.peek().tokenType == tokenRightBracket {
			break
		}
		value, meta, err := p.parsePlainStringLiteral()
		if err != nil {
			return err
		}
		values = append(values, value)
		metas = append(metas, meta)

		if !p.match(tokenComma) {
			break
		}
	}

	closeBracket, err := p.expect(tokenRightBracket)
	if err != nil {
		return err
	}

	if len(values) == 0 {
		return p.errf(closeBracket.pos, "expected at least one transform in list")
	}

	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values:     values,
			SourceMeta: metas,
		},
	}

	return nil
}

func (p *parser) advance() *token {
	p.skipGroupedNewLines()
	p.fill(1)
	tkn := p.buf[0]
	p.buf = p.buf[1:]

	switch tkn.tokenType {
	case tokenLeftParen, tokenLeftBracket:
		p.groupingDepth += 1
	case tokenRightParen, tokenRightBracket:
		if p.groupingDepth > 0 {
			p.groupingDepth -= 1
		}
	}

	return tkn
}

// Look past new lines without consuming, to test for a continuation operator.
func (p *parser) peekPastNewlines() *token {
	i := 0
	for {
		p.fill(i + 1)
		if p.buf[i].tokenType != tokenNewline {
			return p.buf[i]
		}
		i += 1
	}
}

func (p *parser) peek() *token {
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
		if p.buf[0].tokenType != tokenNewline {
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
		switch p.peek().tokenType {
		case tokenEOF,
			tokenKeywordVariable, tokenKeywordValue, tokenKeywordData,
			tokenKeywordResource, tokenKeywordInclude, tokenKeywordExport,
			tokenKeywordVersion, tokenKeywordTransform, tokenKeywordMetadata:
			return
		}
		p.advance()
	}
}

func (p *parser) fill(n int) {
	for len(p.buf) < n {
		tkn := p.lex.nextToken()
		if tkn.tokenType == tokenComment {
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

func (p *parser) peekAt(n int) *token {
	p.skipGroupedNewLines()
	p.fill(n + 1)
	return p.buf[n]
}

func (p *parser) match(tt tokenType) bool {
	if p.peek().tokenType == tt {
		p.advance()
		return true
	}
	return false
}

func (p *parser) matchAcrossNewlines(tts ...tokenType) (*token, bool) {
	next := p.peekPastNewlines()

	if slices.Contains(tts, next.tokenType) {
		p.skipNewlines() // commit: newline was a continuation, not a separator
		op := p.advance()
		p.skipNewlines() // tolerate operator-at-end-of-line
		return op, true
	}

	return nil, false
}

func (p *parser) expect(tt tokenType) (*token, error) {
	tkn := p.peek()
	if tkn.tokenType != tt {
		return nil, p.errf(tkn.pos, "expected %s, got %s", tt, tkn.tokenType)
	}
	return p.advance(), nil
}

func (p *parser) consumeSeparators() bool {
	consumed := false
	for {
		tt := p.peek().tokenType
		if tt != tokenComma && tt != tokenNewline {
			return consumed
		}
		p.advance()
		consumed = true
	}
}

func (p *parser) skipNewlines() {
	for {
		p.fill(1)
		if p.buf[0].tokenType != tokenNewline {
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

func sourceMetaFromToken(tkn *token) *source.Meta {
	return &source.Meta{
		Position:    tkn.pos,
		EndPosition: &tkn.endPos,
	}
}
