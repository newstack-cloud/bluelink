package lang

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// Consumes a tokenStringStart..tokenStringEnd run restricted to a single-line
// literal with no interpolation, returning the content and its full span. For
// positions that require a single-line compile-time literal: version,
// transform, include path, removalPolicy, and quoted names / object keys.
func (p *parser) parsePlainStringLiteral() (string, *source.Meta, error) {
	return p.collectStringLiteral(false)
}

// Reads a string literal as a static string: it appends single-line and, when
// allowMultiline is set, multi-line content, and rejects interpolation since
// the caller needs a plain string rather than a StringOrSubstitutions. A scalar
// string value (a variable's default or description) sets allowMultiline;
// positions restricted to a single-line literal do not.
func (p *parser) collectStringLiteral(allowMultiline bool) (string, *source.Meta, error) {
	start, err := p.expect(tokenStringStart)
	if err != nil {
		return "", nil, err
	}

	if start.value == `"""` && !allowMultiline {
		return "", nil, p.errf(start.pos, "multi-line strings are not allowed in this position")
	}

	var sb strings.Builder
	for {
		switch p.peek().tokenType {
		case tokenStringLiteral, tokenMultilineStringLiteral:
			sb.WriteString(p.advance().value)
		case tokenInterpolationStart:
			tkn := p.peek()
			return "", nil, p.errf(tkn.pos, "interpolation is not allowed in this position")
		case tokenStringEnd:
			end := p.advance()
			return sb.String(), &source.Meta{
				Position:    start.pos,
				EndPosition: &end.endPos,
			}, nil
		default:
			tkn := p.peek()
			return "", nil, p.errf(tkn.pos, "unterminated string literal")
		}
	}
}
