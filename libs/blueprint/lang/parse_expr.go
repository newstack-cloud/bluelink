package lang

import (
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseExpr() (expr, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		// `||` may sit at a line boundary, when the newline is a continuation,
		// not a separator.
		if _, ok := p.matchAcrossNewlines(TokenOr); !ok {
			return left, nil
		}

		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}

		left = &opExpr{
			fn:   substitutions.SubstitutionFunctionOr,
			args: []expr{left, right},
			m:    exprSpan(left, right),
		}
	}
}

func (p *parser) parseAnd() (expr, error) {
	left, err := p.parseEq()
	if err != nil {
		return nil, err
	}

	for {
		if _, ok := p.matchAcrossNewlines(TokenAnd); !ok {
			return left, nil
		}

		right, err := p.parseEq()
		if err != nil {
			return nil, err
		}

		left = &opExpr{
			fn:   substitutions.SubstitutionFunctionAnd,
			args: []expr{left, right},
			m:    exprSpan(left, right),
		}
	}
}

func (p *parser) parseEq() (expr, error) {
	left, err := p.parseComp()
	if err != nil {
		return nil, err
	}

	for {
		op, ok := p.matchAcrossNewlines(TokenEq, TokenNeq)
		if !ok {
			return left, nil
		}

		right, err := p.parseComp()
		if err != nil {
			return nil, err
		}

		switch op.Type {
		case TokenEq:
			left = &opExpr{
				fn:   substitutions.SubstitutionFunctionEq,
				args: []expr{left, right},
				m:    exprSpan(left, right),
			}
		case TokenNeq:
			// not(eq(left, right)) - we don't have a native "not equal" operator
			// in the substitutions language, but we can desugar it to a "not" of an "eq"
			left = &opExpr{
				fn: substitutions.SubstitutionFunctionNot,
				args: []expr{
					&opExpr{
						fn:   substitutions.SubstitutionFunctionEq,
						args: []expr{left, right},
						m:    exprSpan(left, right),
					},
				},
				m: exprSpan(left, right),
			}
		default:
			return nil, p.errf(
				op.Start,
				"unexpected equality operator %s",
				op.Type,
			)
		}
	}
}

func (p *parser) parseComp() (expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		op, ok := p.matchAcrossNewlines(TokenLt, TokenLte, TokenGt, TokenGte)
		if !ok {
			return left, nil
		}

		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		var fn substitutions.SubstitutionFunctionName
		switch op.Type {
		case TokenLt:
			fn = substitutions.SubstitutionFunctionLT
		case TokenLte:
			fn = substitutions.SubstitutionFunctionLE
		case TokenGt:
			fn = substitutions.SubstitutionFunctionGT
		case TokenGte:
			fn = substitutions.SubstitutionFunctionGE
		default:
			return nil, p.errf(
				op.Start,
				"unexpected comparison operator %s",
				op.Type,
			)
		}

		left = &opExpr{
			fn:   fn,
			args: []expr{left, right},
			m:    exprSpan(left, right),
		}
	}
}

func (p *parser) parseUnary() (expr, error) {
	if p.match(TokenNot) {
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		return &opExpr{
			fn:   substitutions.SubstitutionFunctionNot,
			args: []expr{operand},
			m:    operand.meta(),
		}, nil
	}

	return p.parsePrimary()
}

func (p *parser) parsePrimary() (expr, error) {
	tkn := p.peek()
	switch tkn.Type {
	case TokenFloatLiteral, TokenIntLiteral, TokenBoolLiteral:
		return p.parseScalarExpr()
	case TokenStringStart:
		return p.parseStringExpr()
	case TokenNoneLiteral:
		return p.parseNoneExpr()
	case TokenLeftBracket:
		return p.parseArrayExpr()
	case TokenLeftBrace:
		return p.parseObjectExpr()
	case TokenLeftParen:
		return p.parseGroup()
	default:
		return p.parseReferenceOrCall()
	}
}

func (p *parser) parseScalarExpr() (expr, error) {
	lit, err := p.parseScalarLiteral()
	if err != nil {
		return nil, err
	}

	return &scalarExpr{
		value: lit,
	}, nil
}

func (p *parser) parseStringExpr() (expr, error) {
	start, err := p.expect(TokenStringStart)
	if err != nil {
		return nil, err
	}

	var parts []interpolationPart
	for {
		tkn := p.peek()
		switch tkn.Type {
		case TokenStringLiteral, TokenMultilineStringLiteral:
			p.advance()
			parts = append(
				parts,
				&stringPart{
					value: tkn.Value,
					m:     sourceMetaFromToken(tkn),
				},
			)
		case TokenInterpolationStart:
			part, err := p.parseInterpolationPart()
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case TokenStringEnd:
			end := p.advance()
			span := &source.Meta{
				Position:    start.Start,
				EndPosition: &end.End,
			}
			return finishStringExpr(parts, span), nil
		default:
			return nil, p.errf(tkn.Start, "unterminated string literal")
		}
	}
}

func (p *parser) parseInterpolationPart() (interpolationPart, error) {
	open := p.advance() // '${'
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	closeTkn, err := p.expect(TokenInterpolationEnd)
	if err != nil {
		return nil, err
	}
	return &substitutionPart{
		value: value,
		m: &source.Meta{
			Position:    open.Start,
			EndPosition: &closeTkn.End,
		},
	}, nil
}

func finishStringExpr(parts []interpolationPart, span *source.Meta) expr {
	text, plain := joinStringParts(parts)
	if plain {
		return &scalarExpr{
			value: &core.ScalarValue{
				StringValue: &text,
				SourceMeta:  span,
			},
		}
	}
	return &interpolationExpr{
		parts: parts,
		m:     span,
	}
}

func joinStringParts(parts []interpolationPart) (string, bool) {
	var sb strings.Builder
	for _, part := range parts {
		sp, ok := part.(*stringPart)
		if !ok {
			return "", false
		}
		sb.WriteString(sp.value)
	}
	return sb.String(), true
}

func (p *parser) parseNoneExpr() (expr, error) {
	tkn, err := p.expect(TokenNoneLiteral)
	if err != nil {
		return nil, err
	}

	return &noneExpr{
		m: sourceMetaFromToken(tkn),
	}, nil
}

func (p *parser) parseArrayExpr() (expr, error) {
	var elems []expr
	span, err := p.parseArray(func() error {
		expr, err := p.parseExpr()
		if err != nil {
			return err
		}

		elems = append(elems, expr)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &arrayExpr{
		elems: elems,
		m:     span,
	}, nil
}

func (p *parser) parseObjectExpr() (expr, error) {
	var entries []objectField
	span, err := p.parseBraceBlock(func() error {
		key, keyMeta, err := p.parseObjectKey()
		if err != nil {
			return err
		}

		if _, err := p.expect(TokenAssign); err != nil {
			return err
		}

		value, err := p.parseExpr()
		if err != nil {
			return err
		}

		entries = append(entries, objectField{
			key:   key,
			value: value,
			meta:  keyMeta,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &objectExpr{
		entries: entries,
		m:       span,
	}, nil
}

func (p *parser) parseGroup() (expr, error) {
	// consume '(' and bump groupingDepth
	p.advance()

	inner, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenRightParen); err != nil {
		return nil, err
	}

	return inner, nil
}

func (p *parser) parseScalarLiteralArray() ([]*core.ScalarValue, error) {
	var values []*core.ScalarValue

	_, err := p.parseArray(func() error {
		val, err := p.parseScalarLiteral()
		if err != nil {
			return err
		}

		values = append(values, val)
		return nil
	})

	return values, err
}

func (p *parser) parseArray(parseElement func() error) (*source.Meta, error) {
	open, err := p.expect(TokenLeftBracket)
	if err != nil {
		return nil, err
	}

	p.consumeSeparators()
	for p.peek().Type != TokenRightBracket &&
		p.peek().Type != TokenEOF {
		if err := parseElement(); err != nil {
			return nil, err
		}
		if !p.consumeSeparators() {
			break
		}
	}

	closeTkn, err := p.expect(TokenRightBracket)
	if err != nil {
		return nil, err
	}

	return &source.Meta{
		Position:    open.Start,
		EndPosition: &closeTkn.End,
	}, nil
}

// Parses a string literal that may contain ${..} interpolations, returning the
// canonical *StringOrSubstitutions used by stringy schema fields such as
// schema.Value.Description.
func (p *parser) parseInterpolatedString() (*substitutions.StringOrSubstitutions, error) {
	e, err := p.parseStringExpr()
	if err != nil {
		return nil, err
	}

	switch n := e.(type) {
	case *scalarExpr:
		// parseStringExpr only produces scalarExpr for a plain (no-${..}) string,
		// so the value is always a StringValue scalar.
		s := *n.value.StringValue
		return &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &s,
					SourceMeta:  n.value.SourceMeta,
				},
			},
			SourceMeta: n.value.SourceMeta,
		}, nil
	case *interpolationExpr:
		values, err := interpolationPartsToSOSValues(n.parts)
		if err != nil {
			return nil, err
		}
		return &substitutions.StringOrSubstitutions{
			Values:     values,
			SourceMeta: n.m,
		}, nil
	default:
		return nil, p.errf(
			e.meta().Position,
			"internal: parseStringExpr produced an unexpected variant",
		)
	}
}

func (p *parser) parseScalarLiteral() (*core.ScalarValue, error) {
	tkn := p.peek()
	switch tkn.Type {
	case TokenStringStart:
		return p.parseStringLiteral()
	case TokenFloatLiteral:
		return p.parseFloatLiteral()
	case TokenIntLiteral:
		return p.parseIntLiteral()
	case TokenBoolLiteral:
		return p.parseBoolLiteral()
	default:
		return nil, p.errf(tkn.Start, "expected scalar literal, got %s", tkn.Type)
	}
}

func (p *parser) parseStringLiteral() (*core.ScalarValue, error) {
	value, meta, err := p.collectStringLiteral(true)
	if err != nil {
		return nil, err
	}

	return &core.ScalarValue{
		StringValue: &value,
		SourceMeta:  meta,
	}, nil
}

func (p *parser) parseFloatLiteral() (*core.ScalarValue, error) {
	tkn, err := p.expect(TokenFloatLiteral)
	if err != nil {
		return nil, err
	}

	// The lexer guarantees this will succeed since it only produces a TokenFloatLiteral
	// if the text matches a valid float format.
	floatVal, _ := strconv.ParseFloat(tkn.Value, 64)

	return &core.ScalarValue{
		FloatValue: &floatVal,
		SourceMeta: sourceMetaFromToken(tkn),
	}, nil
}

func (p *parser) parseIntLiteral() (*core.ScalarValue, error) {
	tkn, err := p.expect(TokenIntLiteral)
	if err != nil {
		return nil, err
	}

	// The lexer guarantees this will succeed since it only produces a TokenIntLiteral
	// if the text matches a valid int format.
	int64Val, _ := strconv.ParseInt(tkn.Value, 10, 64)
	intVal := int(int64Val)

	return &core.ScalarValue{
		IntValue:   &intVal,
		SourceMeta: sourceMetaFromToken(tkn),
	}, nil
}

func (p *parser) parseBoolLiteral() (*core.ScalarValue, error) {
	tkn, err := p.expect(TokenBoolLiteral)
	if err != nil {
		return nil, err
	}

	boolVal := tkn.Value == "true"

	return &core.ScalarValue{
		BoolValue:  &boolVal,
		SourceMeta: sourceMetaFromToken(tkn),
	}, nil
}

func exprSpan(left, right expr) *source.Meta {
	span := &source.Meta{}

	if lm := left.meta(); lm != nil {
		span.Position = lm.Position
	}

	if rm := right.meta(); rm != nil {
		span.EndPosition = rm.EndPosition
	}

	return span
}
