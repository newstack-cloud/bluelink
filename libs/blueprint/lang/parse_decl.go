package lang

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseVariableDecl(bp *schema.Blueprint) error {
	if bp.Variables == nil {
		bp.Variables = &schema.VariableMap{
			Values:     map[string]*schema.Variable{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // consume 'variable' keyword
	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	variable := &schema.Variable{}
	bp.Variables.Values[name] = variable
	bp.Variables.SourceMeta[name] = meta

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	var varType string
	var varTypeMeta *source.Meta
	switch p.peek().Type {
	case TokenKeywordString, TokenKeywordInteger, TokenKeywordFloat, TokenKeywordBoolean:
		tkn := p.advance()
		varType = tkn.Value
		varTypeMeta = sourceMetaFromToken(tkn)
	default:
		elementType, meta, err := p.parseElementType()
		if err != nil {
			return err
		}
		varType = elementType
		varTypeMeta = meta
	}

	variable.Type = &schema.VariableTypeWrapper{
		Value:      schema.VariableType(varType),
		SourceMeta: varTypeMeta,
	}

	_, err = p.parseBraceBlock(func() error {
		return p.parseVariableField(variable)
	})
	return err
}

func (p *parser) parseVariableField(v *schema.Variable) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	switch field {
	case "default":
		v.Default, err = p.parseScalarLiteral()
	case "description":
		v.Description, err = p.parseStringLiteral()
	case "secret":
		v.Secret, err = p.parseBoolLiteral()
	case "allowedValues":
		if v.Type != nil && v.Type.Value == schema.VariableTypeBoolean {
			return p.errf(
				fieldMeta.Position,
				"'allowedValues' is not valid on a boolean variable",
			)
		}
		v.AllowedValues, err = p.parseScalarLiteralArray()
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in variable declaration", field)
	}

	return err
}

func (p *parser) parseValueDecl(bp *schema.Blueprint) error {
	if bp.Values == nil {
		bp.Values = &schema.ValueMap{
			Values:     map[string]*schema.Value{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // consume 'value' keyword
	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	value := &schema.Value{}
	bp.Values.Values[name] = value
	bp.Values.SourceMeta[name] = meta

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	var valType string
	var valTypeMeta *source.Meta
	switch p.peek().Type {
	case TokenKeywordString, TokenKeywordInteger, TokenKeywordFloat,
		TokenKeywordBoolean, TokenKeywordArray, TokenKeywordObject:
		tkn := p.advance()
		valType = tkn.Value
		valTypeMeta = sourceMetaFromToken(tkn)
	default:
		return p.errf(
			p.peek().Start,
			"expected a valid value type, got %s",
			p.peek().Type,
		)
	}

	value.Type = &schema.ValueTypeWrapper{
		Value:      schema.ValueType(valType),
		SourceMeta: valTypeMeta,
	}

	_, err = p.parseBraceBlock(func() error {
		return p.parseValueField(value)
	})
	return err
}

func (p *parser) parseValueField(v *schema.Value) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	var valueExpr expr
	switch field {
	case "value":
		valueExpr, err = p.parseExpr()
		if err != nil {
			return err
		}
		v.Value, err = exprToMappingNode(valueExpr)
	case "description":
		v.Description, err = p.parseInterpolatedString()
	case "secret":
		v.Secret, err = p.parseBoolLiteral()
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in value declaration", field)
	}

	return err
}

func (p *parser) parseDataDecl(bp *schema.Blueprint) error {
	if bp.DataSources == nil {
		bp.DataSources = &schema.DataSourceMap{
			Values:     map[string]*schema.DataSource{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // 'data'

	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	typeStr, typeMeta, err := p.parseElementType()
	if err != nil {
		return err
	}

	ds := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{
			Value:      typeStr,
			SourceMeta: typeMeta,
		},
		FieldsSourceMeta: map[string]*source.Meta{},
	}
	bp.DataSources.Values[name] = ds
	bp.DataSources.SourceMeta[name] = meta

	_, err = p.parseBraceBlock(func() error {
		return p.parseDataDeclEntry(ds)
	})
	return err
}

func (p *parser) parseResourceDecl(bp *schema.Blueprint) error {
	if bp.Resources == nil {
		bp.Resources = &schema.ResourceMap{
			Values:     map[string]*schema.Resource{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // 'resource'

	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	typeStr, typeMeta, err := p.parseElementType()
	if err != nil {
		return err
	}

	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{
			Value:      typeStr,
			SourceMeta: typeMeta,
		},
		FieldsSourceMeta: map[string]*source.Meta{},
	}
	bp.Resources.Values[name] = resource
	bp.Resources.SourceMeta[name] = meta

	_, err = p.parseBraceBlock(func() error {
		return p.parseResourceDeclEntry(resource)
	})
	if err != nil {
		return err
	}

	if resource.Spec == nil {
		return p.errf(
			meta.Position,
			"resource %q is missing the required 'spec { ... }' block",
			name,
		)
	}
	return nil
}

func (p *parser) parseIncludeDecl(bp *schema.Blueprint) error {
	if bp.Include == nil {
		bp.Include = &schema.IncludeMap{
			Values:     map[string]*schema.Include{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // 'include'

	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	path, err := p.parseInterpolatedString()
	if err != nil {
		return err
	}

	include := &schema.Include{
		Path:             path,
		FieldsSourceMeta: map[string]*source.Meta{},
	}
	bp.Include.Values[name] = include
	bp.Include.SourceMeta[name] = meta

	_, err = p.parseBraceBlock(func() error {
		return p.parseIncludeField(include)
	})
	return err
}

func (p *parser) parseIncludeField(inc *schema.Include) error {
	field, fieldMeta, err := p.parseFieldKey()
	if err != nil {
		return err
	}

	switch field {
	case "description":
		if _, err := p.expect(TokenAssign); err != nil {
			return err
		}

		inc.Description, err = p.parseInterpolatedString()
		if err != nil {
			return err
		}

		var valueEnd *source.Position
		if inc.Description.SourceMeta != nil {
			valueEnd = inc.Description.SourceMeta.EndPosition
		}
		inc.FieldsSourceMeta[field] = mergeEnd(fieldMeta, valueEnd)
		return nil
	case "variables":
		inc.Variables, err = p.parseFreeFormMapBlock()
		if err != nil {
			return err
		}

		var valueEnd *source.Position
		if inc.Variables.SourceMeta != nil {
			valueEnd = inc.Variables.SourceMeta.EndPosition
		}
		inc.FieldsSourceMeta[field] = mergeEnd(fieldMeta, valueEnd)
		return nil
	case "metadata":
		inc.Metadata, err = p.parseFreeFormMapBlock()
		if err != nil {
			return err
		}

		var valueEnd *source.Position
		if inc.Metadata.SourceMeta != nil {
			valueEnd = inc.Metadata.SourceMeta.EndPosition
		}
		inc.FieldsSourceMeta[field] = mergeEnd(fieldMeta, valueEnd)
		return nil
	default:
		return p.errf(
			fieldMeta.Position,
			"unknown field %q in include declaration",
			field,
		)
	}
}

func (p *parser) parseExportDecl(bp *schema.Blueprint) error {
	if bp.Exports == nil {
		bp.Exports = &schema.ExportMap{
			Values:     map[string]*schema.Export{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	p.advance() // 'export'

	name, meta, err := p.parseElementName()
	if err != nil {
		return err
	}

	export := &schema.Export{}
	bp.Exports.Values[name] = export
	bp.Exports.SourceMeta[name] = meta

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	exportType, typeMeta, err := p.parseExportType()
	if err != nil {
		return err
	}
	export.Type = &schema.ExportTypeWrapper{
		Value:      schema.ExportType(exportType),
		SourceMeta: typeMeta,
	}

	_, err = p.parseBraceBlock(func() error {
		return p.parseExportField(export)
	})
	return err
}

func (p *parser) parseExportType() (string, *source.Meta, error) {
	switch p.peek().Type {
	case TokenKeywordString, TokenKeywordInteger, TokenKeywordFloat,
		TokenKeywordBoolean, TokenKeywordArray, TokenKeywordObject:
		tkn := p.advance()
		return tkn.Value, sourceMetaFromToken(tkn), nil
	default:
		return "", nil, p.errf(
			p.peek().Start,
			"expected a valid export type, got %s",
			p.peek().Type,
		)
	}
}

func (p *parser) parseExportField(e *schema.Export) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	switch field {
	case "field":
		e.Field, err = p.parseExportFieldValue()
	case "description":
		e.Description, err = p.parseInterpolatedString()
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in export declaration", field)
	}

	return err
}

func (p *parser) parseExportFieldValue() (*core.ScalarValue, error) {
	if p.peek().Type == TokenStringStart {
		value, meta, err := p.collectStringLiteral(false)
		if err != nil {
			return nil, err
		}
		return &core.ScalarValue{
			StringValue: &value,
			SourceMeta:  meta,
		}, nil
	}

	ref, err := p.parseReferenceOrCall()
	if err != nil {
		return nil, err
	}

	refSub, ok := ref.(*refExpr)
	if !ok {
		return nil, p.errf(
			ref.meta().Position,
			"export 'field' must be a reference path, not a function call or computed expression",
		)
	}

	pathStr, err := substitutions.SubstitutionToString("export field", refSub.sub)
	if err != nil {
		return nil, err
	}

	return &core.ScalarValue{
		StringValue: &pathStr,
		SourceMeta:  refSub.sub.SourceMeta,
	}, nil
}

func (p *parser) parseFieldAssignment() (string, *source.Meta, error) {
	field, meta, err := p.parseFieldKey()
	if err != nil {
		return "", nil, err
	}

	if _, err := p.expect(TokenAssign); err != nil {
		return "", nil, err
	}

	return field, meta, nil
}

// Parses a provider/transformer element type such as
// aws/ec2/instance: two or more "/"-joined segments.
func (p *parser) parseElementType() (string, *source.Meta, error) {
	sb := strings.Builder{}

	first, err := p.parseTypeSegment()
	if err != nil {
		return "", nil, err
	}
	sb.WriteString(first.Value)
	start := first.Start
	end := first.End

	if _, err := p.expect(TokenSlash); err != nil {
		return "", nil, err
	}
	sb.WriteByte('/')

	second, err := p.parseTypeSegment()
	if err != nil {
		return "", nil, err
	}
	sb.WriteString(second.Value)
	end = second.End

	for p.match(TokenSlash) {
		sb.WriteByte('/')
		seg, err := p.parseTypeSegment()
		if err != nil {
			return "", nil, err
		}

		sb.WriteString(seg.Value)
		end = seg.End
	}

	return sb.String(), &source.Meta{
		Position:    start,
		EndPosition: &end,
	}, nil
}

// Parses one segment of an element type. A segment lexes as an
// identifier, or as a keyword token when it collides with a reserved word
// (e.g. "data", "object"), so it is accepted by token kind and then validated
// against the stricter type-segment grammar (letters and digits only, no '_' or
// '-'), which is narrower than an identifier.
func (p *parser) parseTypeSegment() (*Token, error) {
	tkn := p.peek()
	if !isSegmentToken(tkn.Type) {
		return nil, p.errf(
			tkn.Start,
			"expected an element type segment, got %s",
			tkn.Type,
		)
	}
	p.advance()
	if !isValidTypeSegment(tkn.Value) {
		return nil, p.errf(
			tkn.Start,
			"invalid element type segment %q: a segment must be a letter followed by letters or digits",
			tkn.Value,
		)
	}
	return tkn, nil
}

func isSegmentToken(tt TokenType) bool {
	if tt == TokenIdent || tt == TokenBoolLiteral || tt == TokenNoneLiteral {
		return true
	}
	_, isKeyword := keywordWords[tt]
	return isKeyword
}

func isValidTypeSegment(s string) bool {
	for i, char := range s {
		if i == 0 && !isLetter(char) {
			return false
		}
		if i > 0 && !isLetter(char) && !isDigit(char) {
			return false
		}
	}
	return s != ""
}
