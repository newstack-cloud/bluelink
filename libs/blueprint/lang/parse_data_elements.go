package lang

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseDataDeclEntry(ds *schema.DataSource) error {
	switch p.peek().tokenType {
	case tokenKeywordMetadata:
		return p.parseDataSourceMetadataBlock(ds)
	case tokenKeywordFilter:
		return p.parseFilterStatement(ds)
	case tokenKeywordExport:
		return p.parseDataSourceExportStatement(ds)
	case tokenIdent:
		return p.parseDataSourceFieldAssignment(ds)
	default:
		return p.errf(
			p.peek().pos,
			"expected 'metadata', 'filter', 'export', or a field assignment in data declaration, got %s",
			p.peek().tokenType,
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
			Position:    open.pos,
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
	if p.peek().tokenType != tokenStringStart {
		return nil, p.errf(
			p.peek().pos,
			"expected a string literal naming the filter field, got %s",
			p.peek().tokenType,
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
	start := p.peek().pos
	negated := false
	if p.peek().tokenType == tokenKeywordNot {
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
	switch p.peek().tokenType {
	case tokenKeywordIn:
		return "in", p.advance().endPos, nil
	case tokenKeywordContains:
		return "contains", p.advance().endPos, nil
	case tokenKeywordHas:
		p.advance()
		keyTkn, err := p.expect(tokenKeywordKey)
		if err != nil {
			return "", source.Position{}, err
		}

		return "has key", keyTkn.endPos, nil
	case tokenKeywordStarts:
		p.advance()
		withTkn, err := p.expect(tokenKeywordWith)
		if err != nil {
			return "", source.Position{}, err
		}

		return "starts with", withTkn.endPos, nil
	case tokenKeywordEnds:
		p.advance()
		withTkn, err := p.expect(tokenKeywordWith)
		if err != nil {
			return "", source.Position{}, err
		}

		return "ends with", withTkn.endPos, nil
	}

	if negated {
		return "", source.Position{}, p.errf(
			notPos,
			"'not' is only valid before 'in', 'has key', 'contains', 'starts with', or 'ends with'",
		)
	}

	switch p.peek().tokenType {
	case tokenEq:
		return "=", p.advance().endPos, nil
	case tokenNeq:
		return "!=", p.advance().endPos, nil
	case tokenGt:
		return ">", p.advance().endPos, nil
	case tokenLt:
		return "<", p.advance().endPos, nil
	case tokenGte:
		return ">=", p.advance().endPos, nil
	case tokenLte:
		return "<=", p.advance().endPos, nil
	}

	return "", source.Position{}, p.errf(
		p.peek().pos,
		"expected a filter operator, got %s",
		p.peek().tokenType,
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

	if p.peek().tokenType == tokenStar {
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
	if p.peek().tokenType == tokenKeywordAs {
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

	if _, err := p.expect(tokenColon); err != nil {
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

	if p.peek().tokenType == tokenLeftBrace {
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
	switch p.peek().tokenType {
	case tokenKeywordString, tokenKeywordInteger, tokenKeywordFloat,
		tokenKeywordBoolean, tokenKeywordArray:
		tkn := p.advance()
		return schema.DataSourceFieldType(tkn.value), sourceMetaFromToken(tkn), nil
	default:
		return "", nil, p.errf(
			p.peek().pos,
			"expected a data source export type (string, integer, float, boolean, or array), got %s",
			p.peek().tokenType,
		)
	}
}
