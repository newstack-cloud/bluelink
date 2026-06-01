package lang

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseDataDeclEntry(ds *schema.DataSource) error {
	switch p.peek().Type {
	case TokenKeywordMetadata:
		return p.parseDataSourceMetadataBlock(ds)
	case TokenKeywordFilter:
		return p.parseFilterStatement(ds)
	case TokenKeywordExport:
		return p.parseDataSourceExportStatement(ds)
	case TokenIdent:
		return p.parseDataSourceFieldAssignment(ds)
	default:
		return p.errf(
			p.peek().Start,
			"expected 'metadata', 'filter', 'export', or a field assignment in data declaration, got %s",
			p.peek().Type,
		)
	}
}

func (p *parser) parseDataSourceFieldAssignment(ds *schema.DataSource) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	switch field {
	case "description":
		ds.Description, err = p.parseInterpolatedString()
		if err != nil {
			return err
		}

		var valueEnd *source.Position
		if ds.Description.SourceMeta != nil {
			valueEnd = ds.Description.SourceMeta.EndPosition
		}
		ds.FieldsSourceMeta[field] = mergeEnd(fieldMeta, valueEnd)
		return nil
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in data declaration", field)
	}
}

func (p *parser) parseDataSourceMetadataBlock(ds *schema.DataSource) error {
	meta := &schema.DataSourceMetadata{
		FieldsSourceMeta: map[string]*source.Meta{},
	}

	blockMeta, err := p.parseMetadataKeywordBlock(func() error {
		return p.parseDataSourceMetadataField(meta)
	})
	if err != nil {
		return err
	}

	meta.SourceMeta = blockMeta
	ds.DataSourceMetadata = meta
	ds.FieldsSourceMeta["metadata"] = blockMeta
	return nil
}

func (p *parser) parseDataSourceMetadataField(m *schema.DataSourceMetadata) error {
	field, fieldMeta, err := p.parseFieldKey()
	if err != nil {
		return err
	}

	switch field {
	case "displayName":
		m.DisplayName, err = p.parseDisplayNameField()
		if err != nil {
			return err
		}
		m.FieldsSourceMeta[field] = fieldMeta
		return nil
	case "annotations":
		m.Annotations, err = p.parseAnnotationsField()
		if err != nil {
			return err
		}
		m.FieldsSourceMeta[field] = fieldMeta
		return nil
	case "custom":
		m.Custom, err = p.parseCustomField()
		if err != nil {
			return err
		}
		m.FieldsSourceMeta[field] = fieldMeta
		return nil
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in data source metadata", field)
	}
}

func (p *parser) parseFilterStatement(ds *schema.DataSource) error {
	open := p.advance() // 'filter'

	field, err := p.parseFilterField()
	if err != nil {
		return err
	}

	op, err := p.parseFilterOperator()
	if err != nil {
		return err
	}

	search, err := p.parseFilterSearch()
	if err != nil {
		return err
	}

	endPos := search.SourceMeta.Position
	if search.SourceMeta.EndPosition != nil {
		endPos = *search.SourceMeta.EndPosition
	}

	filter := &schema.DataSourceFilter{
		Field:    field,
		Operator: op,
		Search:   search,
		SourceMeta: &source.Meta{
			Position:    open.Start,
			EndPosition: &endPos,
		},
	}

	if ds.Filter == nil {
		ds.Filter = &schema.DataSourceFilters{}
	}
	ds.Filter.Filters = append(ds.Filter.Filters, filter)
	return nil
}

func (p *parser) parseFilterField() (*core.ScalarValue, error) {
	if p.peek().Type != TokenStringStart {
		return nil, p.errf(
			p.peek().Start,
			"expected a string literal naming the filter field, got %s",
			p.peek().Type,
		)
	}

	value, meta, err := p.collectStringLiteral(false)
	if err != nil {
		return nil, err
	}

	return &core.ScalarValue{
		StringValue: &value,
		SourceMeta:  meta,
	}, nil
}

func (p *parser) parseFilterOperator() (*schema.DataSourceFilterOperatorWrapper, error) {
	start := p.peek().Start
	negated := false
	if p.peek().Type == TokenKeywordNot {
		p.advance()
		negated = true
	}

	opStr, endPos, err := p.parseFilterOperatorBody(negated, start)
	if err != nil {
		return nil, err
	}
	if negated {
		opStr = "not " + opStr
	}

	return &schema.DataSourceFilterOperatorWrapper{
		Value: schema.DataSourceFilterOperator(opStr),
		SourceMeta: &source.Meta{
			Position:    start,
			EndPosition: &endPos,
		},
	}, nil
}

func (p *parser) parseFilterOperatorBody(
	negated bool,
	notPos source.Position,
) (string, source.Position, error) {
	switch p.peek().Type {
	case TokenKeywordIn:
		return "in", p.advance().End, nil
	case TokenKeywordContains:
		return "contains", p.advance().End, nil
	case TokenKeywordHas:
		p.advance()
		keyTkn, err := p.expect(TokenKeywordKey)
		if err != nil {
			return "", source.Position{}, err
		}

		return "has key", keyTkn.End, nil
	case TokenKeywordStarts:
		p.advance()
		withTkn, err := p.expect(TokenKeywordWith)
		if err != nil {
			return "", source.Position{}, err
		}

		return "starts with", withTkn.End, nil
	case TokenKeywordEnds:
		p.advance()
		withTkn, err := p.expect(TokenKeywordWith)
		if err != nil {
			return "", source.Position{}, err
		}

		return "ends with", withTkn.End, nil
	}

	if negated {
		return "", source.Position{}, p.errf(
			notPos,
			"'not' is only valid before 'in', 'has key', 'contains', 'starts with', or 'ends with'",
		)
	}

	switch p.peek().Type {
	case TokenEq:
		return "=", p.advance().End, nil
	case TokenNeq:
		return "!=", p.advance().End, nil
	case TokenGt:
		return ">", p.advance().End, nil
	case TokenLt:
		return "<", p.advance().End, nil
	case TokenGte:
		return ">=", p.advance().End, nil
	case TokenLte:
		return "<=", p.advance().End, nil
	}

	return "", source.Position{}, p.errf(
		p.peek().Start,
		"expected a filter operator, got %s",
		p.peek().Type,
	)
}

func (p *parser) parseFilterSearch() (*schema.DataSourceFilterSearch, error) {
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if arr, ok := e.(*arrayExpr); ok {
		values := make([]*substitutions.StringOrSubstitutions, 0, len(arr.elems))
		for _, el := range arr.elems {
			stringOrSubs, err := exprToStringOrSubstitutions(el)
			if err != nil {
				return nil, err
			}

			values = append(values, stringOrSubs)
		}

		return &schema.DataSourceFilterSearch{
			Values:     values,
			SourceMeta: arr.m,
		}, nil
	}

	stringOrSubs, err := exprToStringOrSubstitutions(e)
	if err != nil {
		return nil, err
	}

	return &schema.DataSourceFilterSearch{
		Values:     []*substitutions.StringOrSubstitutions{stringOrSubs},
		SourceMeta: e.meta(),
	}, nil
}

// Parses one `export …` statement inside a data declaration. Four
// forms are accepted: `export *`, `export <name>: <type>`,
// `export <name>: <type> { … }`, and `export <source> as <exposed>: <type>
// [{ … }]`.
func (p *parser) parseDataSourceExportStatement(ds *schema.DataSource) error {
	p.advance() // 'export'

	if ds.Exports == nil {
		ds.Exports = &schema.DataSourceFieldExportMap{
			Values:     map[string]*schema.DataSourceFieldExport{},
			SourceMeta: map[string]*source.Meta{},
		}
	}

	if p.peek().Type == TokenStar {
		p.advance()
		ds.Exports.ExportAll = true
		return nil
	}

	sourceName, nameMeta, err := p.parseElementName()
	if err != nil {
		return err
	}

	exposedName := sourceName
	exposedMeta := nameMeta
	var aliasFor *core.ScalarValue
	if p.peek().Type == TokenKeywordAs {
		p.advance()
		aliasName, aliasMeta, err := p.parseElementName()
		if err != nil {
			return err
		}
		src := sourceName
		aliasFor = &core.ScalarValue{
			StringValue: &src,
			SourceMeta:  nameMeta,
		}
		exposedName = aliasName
		exposedMeta = aliasMeta
	}

	if _, err := p.expect(TokenColon); err != nil {
		return err
	}

	fieldType, typeMeta, err := p.parseDataSourceFieldType()
	if err != nil {
		return err
	}

	exportEntry := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{
			Value:      fieldType,
			SourceMeta: typeMeta,
		},
		AliasFor:   aliasFor,
		SourceMeta: exposedMeta,
	}

	if p.peek().Type == TokenLeftBrace {
		_, err := p.parseBraceBlock(func() error {
			return p.parseDataSourceExportField(exportEntry)
		})
		if err != nil {
			return err
		}
	}

	ds.Exports.Values[exposedName] = exportEntry
	ds.Exports.SourceMeta[exposedName] = exposedMeta
	return nil
}

func (p *parser) parseDataSourceExportField(e *schema.DataSourceFieldExport) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	switch field {
	case "description":
		e.Description, err = p.parseInterpolatedString()
		return err
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in data source export", field)
	}
}

func (p *parser) parseDataSourceFieldType() (schema.DataSourceFieldType, *source.Meta, error) {
	switch p.peek().Type {
	case TokenKeywordString, TokenKeywordInteger, TokenKeywordFloat,
		TokenKeywordBoolean, TokenKeywordArray:
		tkn := p.advance()
		return schema.DataSourceFieldType(tkn.Value), sourceMetaFromToken(tkn), nil
	default:
		return "", nil, p.errf(
			p.peek().Start,
			"expected a data source export type (string, integer, float, boolean, or array), got %s",
			p.peek().Type,
		)
	}
}
