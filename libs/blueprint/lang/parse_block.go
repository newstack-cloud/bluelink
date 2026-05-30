package lang

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseMetadataBlock(bp *schema.Blueprint) error {
	open := p.advance() // 'metadata'

	if bp.Metadata != nil {
		return p.errf(
			open.pos,
			"top-level 'metadata' block already declared at %d:%d",
			bp.Metadata.SourceMeta.Position.Line,
			bp.Metadata.SourceMeta.Position.Column,
		)
	}

	mapping, err := p.parseFreeFormMapBlock()
	if err != nil {
		return err
	}

	mapping.SourceMeta.Position = open.pos
	bp.Metadata = mapping

	return nil
}

// Parses a `{ key = expr ... }` block as a free-form
// object MappingNode. The returned MappingNode's SourceMeta spans the
// `{...}` only; callers that want to extend the range to include a leading
// keyword or field name should overwrite SourceMeta.Position themselves.
func (p *parser) parseFreeFormMapBlock() (*core.MappingNode, error) {
	fields := map[string]*core.MappingNode{}
	fieldsSourceMeta := map[string]*source.Meta{}

	blockMeta, err := p.parseBraceBlock(func() error {
		return p.parseFreeFormMapEntry(fields, fieldsSourceMeta)
	})
	if err != nil {
		return nil, err
	}

	return &core.MappingNode{
		Fields:           fields,
		FieldsSourceMeta: fieldsSourceMeta,
		SourceMeta:       blockMeta,
	}, nil
}

func (p *parser) parseFreeFormMapEntry(
	fields map[string]*core.MappingNode,
	fieldsSourceMeta map[string]*source.Meta,
) error {
	key, keyMeta, err := p.parseObjectKey()
	if err != nil {
		return err
	}

	if _, err := p.expect(tokenAssign); err != nil {
		return err
	}

	valueExpr, err := p.parseExpr()
	if err != nil {
		return err
	}

	valNode, err := exprToMappingNode(valueExpr)
	if err != nil {
		return err
	}

	fields[key] = valNode
	fieldsSourceMeta[key] = keyMeta

	return nil
}

// Wraps the shared envelope used by every `metadata { ... }`
// sub-block (resource, data source, future include sub-metadata): advances
// the `metadata` keyword, runs parseBraceBlock with the caller-supplied
// per-entry callback, and returns the keyword-anchored block span. The
// caller binds the returned span to its target schema struct.
func (p *parser) parseMetadataKeywordBlock(parseField func() error) (*source.Meta, error) {
	open := p.advance() // 'metadata'

	blockMeta, err := p.parseBraceBlock(parseField)
	if err != nil {
		return nil, err
	}

	return &source.Meta{
		Position:    open.pos,
		EndPosition: blockMeta.EndPosition,
	}, nil
}

// Reads the `= <interpolated-string>` for a `displayName` field.
func (p *parser) parseDisplayNameField() (*substitutions.StringOrSubstitutions, error) {
	if _, err := p.expect(tokenAssign); err != nil {
		return nil, err
	}
	return p.parseInterpolatedString()
}

// Reads `= { key = expr, ... }` for an `annotations` field.
func (p *parser) parseAnnotationsField() (*schema.StringOrSubstitutionsMap, error) {
	obj, err := p.parseObjectLiteralAssignment("annotations")
	if err != nil {
		return nil, err
	}
	return objectExprToStringOrSubstitutionsMap(obj)
}

// Reads `= <expr>` for a `custom` field; the value may be any expression
// that lowers to a MappingNode (typically an object literal).
func (p *parser) parseCustomField() (*core.MappingNode, error) {
	if _, err := p.expect(tokenAssign); err != nil {
		return nil, err
	}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return exprToMappingNode(e)
}

// Consumes `= <expr>` after a field name and requires the
// expression to be an object literal, returning it for caller-specific
// lowering. Used by fields whose schema shape (StringMap,
// StringOrSubstitutionsMap, etc.) only makes sense for an object literal.
func (p *parser) parseObjectLiteralAssignment(field string) (*objectExpr, error) {
	if _, err := p.expect(tokenAssign); err != nil {
		return nil, err
	}

	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	obj, ok := e.(*objectExpr)
	if !ok {
		return nil, p.errf(
			e.meta().Position,
			"%q must be an object literal",
			field,
		)
	}

	return obj, nil
}

// Runs the brace + separator mechanics shared by every
// declaration block and by object literals, calling parseEntry once per entry.
func (p *parser) parseBraceBlock(parseEntry func() error) (*source.Meta, error) {
	open, err := p.expect(tokenLeftBrace)
	if err != nil {
		return nil, err
	}

	// Leading newlines after '{'
	p.consumeSeparators()

	for p.peek().tokenType != tokenRightBrace &&
		p.peek().tokenType != tokenEOF {
		if err := parseEntry(); err != nil {
			return nil, err
		}

		if !p.consumeSeparators() {
			// No separator -> next must be '}'
			break
		}
	}

	close, err := p.expect(tokenRightBrace)
	if err != nil {
		return nil, err
	}

	return &source.Meta{
		Position:    open.pos,
		EndPosition: &close.endPos,
	}, nil
}
