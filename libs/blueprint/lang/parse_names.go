package lang

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// parseElementName reads a declaration name. Grammar: name | quoted name.
// A reserved word is rejected in the bare form: a declaration header requires a
// non-keyword identifier, so a name colliding with a keyword must be quoted.
func (p *parser) parseElementName() (string, *source.Meta, error) {
	tkn := p.peek()
	switch tkn.tokenType {
	case tokenIdent:
		p.advance()
		return tkn.value, sourceMetaFromToken(tkn), nil

	case tokenStringStart:
		return p.parseQuotedName()

	default:
		// Consume the offending token so a reserved word in name position is
		// not re-dispatched as a fresh declaration by the recovery in Parse.
		bad := p.advance()
		if isKeyword(bad.tokenType) {
			return "", nil, p.errf(
				bad.pos,
				"%s is reserved and cannot be used as a bare name; wrap it in double quotes to use it as a name",
				bad.tokenType,
			)
		}
		return "", nil, p.errf(bad.pos, "expected element name, got %s", bad.tokenType)
	}
}

// Reads the left of a `field = value` assignment in a block.
// Grammar: name | quoted name. Unlike an element name it accepts a reserved
// word as a bare key (e.g. the `value` field of a value declaration); the
// quoted form is the restricted, referenceable name set.
func (p *parser) parseFieldKey() (string, *source.Meta, error) {
	tkn := p.peek()
	switch {
	case tkn.tokenType == tokenIdent || isKeyword(tkn.tokenType):
		p.advance()
		return tkn.value, sourceMetaFromToken(tkn), nil
	case tkn.tokenType == tokenStringStart:
		return p.parseQuotedName()
	default:
		return "", nil, p.errf(tkn.pos, "expected a field name, got %s", tkn.tokenType)
	}
}

// Reads the left of a `key = value` entry in an object literal.
// Grammar: name | object key string. The quoted form allows any character
// (e.g. "aws:SourceArn") since object keys are data, not referenceable names;
// this is wider than a field key's quoted name.
func (p *parser) parseObjectKey() (string, *source.Meta, error) {
	tkn := p.peek()
	switch {
	case tkn.tokenType == tokenIdent || isKeyword(tkn.tokenType):
		p.advance()
		return tkn.value, sourceMetaFromToken(tkn), nil
	case tkn.tokenType == tokenStringStart:
		return p.parsePlainStringLiteral()
	default:
		return "", nil, p.errf(tkn.pos, "expected an object key, got %s", tkn.tokenType)
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

func isKeyword(tt tokenType) bool {
	_, ok := keywordWords[tt]
	return ok
}
