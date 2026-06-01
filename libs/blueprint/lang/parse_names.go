package lang

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// Reads a declaration name. Grammar: name | quoted name.
// A reserved word is rejected in the bare form: a declaration header requires a
// non-keyword identifier, so a name colliding with a keyword must be quoted.
func (p *parser) parseElementName() (string, *source.Meta, error) {
	tkn := p.peek()
	switch tkn.Type {
	case TokenIdent:
		p.advance()
		return tkn.Value, sourceMetaFromToken(tkn), nil

	case TokenStringStart:
		return p.parseQuotedName()

	default:
		// Consume the offending token so a reserved word in name position is
		// not re-dispatched as a fresh declaration by the recovery in Parse.
		bad := p.advance()
		if isKeyword(bad.Type) {
			return "", nil, p.errf(
				bad.Start,
				"%s is reserved and cannot be used as a bare name; wrap it in double quotes to use it as a name",
				bad.Type,
			)
		}
		return "", nil, p.errf(bad.Start, "expected element name, got %s", bad.Type)
	}
}

// Reads the left of a `field = value` assignment in a block.
// Grammar: name | quoted name. Unlike an element name it accepts a reserved
// word as a bare key (e.g. the `value` field of a value declaration); the
// quoted form is the restricted, referenceable name set.
func (p *parser) parseFieldKey() (string, *source.Meta, error) {
	tkn := p.peek()
	switch {
	case tkn.Type == TokenIdent || isKeyword(tkn.Type):
		p.advance()
		return tkn.Value, sourceMetaFromToken(tkn), nil
	case tkn.Type == TokenStringStart:
		return p.parseQuotedName()
	default:
		return "", nil, p.errf(tkn.Start, "expected a field name, got %s", tkn.Type)
	}
}

// Reads the left of a `key = value` entry in an object literal.
// Grammar: name | object key string. The quoted form allows any character
// (e.g. "aws:SourceArn") since object keys are data, not referenceable names;
// this is wider than a field key's quoted name.
func (p *parser) parseObjectKey() (string, *source.Meta, error) {
	tkn := p.peek()
	switch {
	case tkn.Type == TokenIdent || isKeyword(tkn.Type):
		p.advance()
		return tkn.Value, sourceMetaFromToken(tkn), nil
	case tkn.Type == TokenStringStart:
		return p.parsePlainStringLiteral()
	default:
		return "", nil, p.errf(tkn.Start, "expected an object key, got %s", tkn.Type)
	}
}

// Reads a `quoted name`: double-quoted text restricted to
// letter | digit | "_" | "-" | ".", the referenceable name set shared by
// element names and field keys.
func (p *parser) parseQuotedName() (string, *source.Meta, error) {
	name, meta, err := p.parsePlainStringLiteral()
	if err != nil {
		return "", nil, err
	}
	if !isValidQuotedName(name) {
		return "", nil, p.errf(
			meta.Position,
			"invalid quoted name %q: only letters, digits, '_', '-' and '.' are allowed",
			name,
		)
	}
	return name, meta, nil
}

func isValidQuotedName(s string) bool {
	if s == "" {
		return false
	}

	for _, char := range s {
		if !isIdentChar(char) && char != '.' {
			return false
		}
	}

	return true
}

func isKeyword(tt TokenType) bool {
	_, ok := keywordWords[tt]
	return ok
}
