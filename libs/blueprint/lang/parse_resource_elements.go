package lang

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

func (p *parser) parseResourceDeclEntry(r *schema.Resource) error {
	switch p.peek().Type {
	case TokenKeywordMetadata:
		return p.parseResourceMetadataBlock(r)
	case TokenKeywordSelect:
		return p.parseSelectStatement(r)
	case TokenKeywordSpec:
		return p.parseSpecBlock(r)
	case TokenKeywordForeach:
		return p.parseForeachStatement(r)
	case TokenIdent:
		return p.parseResourceFieldAssignment(r)
	default:
		return p.errf(
			p.peek().Start,
			"expected 'metadata', 'select', 'spec', 'foreach', or a field assignment in resource declaration, got %s",
			p.peek().Type,
		)
	}
}

func (p *parser) parseResourceMetadataBlock(r *schema.Resource) error {
	meta := &schema.Metadata{
		FieldsSourceMeta: map[string]*source.Meta{},
	}

	blockMeta, err := p.parseMetadataKeywordBlock(func() error {
		return p.parseResourceMetadataField(meta)
	})
	if err != nil {
		return err
	}

	meta.SourceMeta = blockMeta
	r.Metadata = meta
	r.FieldsSourceMeta["metadata"] = blockMeta
	return nil
}

func (p *parser) parseResourceMetadataField(m *schema.Metadata) error {
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
	case "labels":
		obj, err := p.parseObjectLiteralAssignment(field)
		if err != nil {
			return err
		}
		m.Labels, err = objectExprToStringMap(obj)
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
		return p.errf(fieldMeta.Position, "unknown field %q in resource metadata", field)
	}
}

func (p *parser) parseSelectStatement(r *schema.Resource) error {
	open := p.advance() // 'select'

	if _, err := p.expect(TokenKeywordBy); err != nil {
		return err
	}
	if _, err := p.expect(TokenKeywordLabel); err != nil {
		return err
	}

	selector := &schema.LinkSelector{
		ByLabel: &schema.StringMap{
			Values:     map[string]string{},
			SourceMeta: map[string]*source.Meta{},
		},
		FieldsSourceMeta: map[string]*source.Meta{},
	}

	blockMeta, err := p.parseBraceBlock(func() error {
		return p.parseSelectByLabelEntry(selector)
	})
	if err != nil {
		return err
	}

	selector.SourceMeta = &source.Meta{
		Position:    open.Start,
		EndPosition: blockMeta.EndPosition,
	}
	r.LinkSelector = selector
	r.FieldsSourceMeta["linkSelector"] = selector.SourceMeta
	return nil
}

func (p *parser) parseSelectByLabelEntry(s *schema.LinkSelector) error {
	key, keyMeta, err := p.parseObjectKey()
	if err != nil {
		return err
	}

	if _, err := p.expect(TokenAssign); err != nil {
		return err
	}

	if key == "exclude" {
		e, err := p.parseExpr()
		if err != nil {
			return err
		}

		list, err := exprToResourceNameList(e, "exclude")
		if err != nil {
			return err
		}

		s.Exclude = list
		s.FieldsSourceMeta[key] = keyMeta
		return nil
	}

	if p.peek().Type != TokenStringStart {
		return p.errf(
			p.peek().Start,
			"label value for %q must be a string literal, got %s",
			key, p.peek().Type,
		)
	}

	value, _, err := p.collectStringLiteral(false)
	if err != nil {
		return err
	}

	s.ByLabel.Values[key] = value
	s.ByLabel.SourceMeta[key] = keyMeta
	return nil
}

func (p *parser) parseSpecBlock(r *schema.Resource) error {
	open := p.advance() // 'spec'

	spec, err := p.parseFreeFormMapBlock()
	if err != nil {
		return err
	}

	spec.SourceMeta.Position = open.Start
	r.Spec = spec
	r.FieldsSourceMeta["spec"] = spec.SourceMeta
	return nil
}

func (p *parser) parseForeachStatement(r *schema.Resource) error {
	open := p.advance() // 'foreach'

	e, err := p.parseExpr()
	if err != nil {
		return err
	}

	sos, err := exprToStringOrSubstitutions(e)
	if err != nil {
		return err
	}

	r.Each = sos
	r.FieldsSourceMeta["each"] = &source.Meta{
		Position:    open.Start,
		EndPosition: e.meta().EndPosition,
	}
	return nil
}

func (p *parser) parseResourceFieldAssignment(r *schema.Resource) error {
	field, fieldMeta, err := p.parseFieldAssignment()
	if err != nil {
		return err
	}

	var valueEnd *source.Position
	switch field {
	case "description":
		r.Description, err = p.parseInterpolatedString()
		if err == nil && r.Description.SourceMeta != nil {
			valueEnd = r.Description.SourceMeta.EndPosition
		}
	case "condition":
		e, exprErr := p.parseExpr()
		if exprErr != nil {
			return exprErr
		}

		r.Condition, err = exprToCondition(e)
		if err == nil && r.Condition.SourceMeta != nil {
			valueEnd = r.Condition.SourceMeta.EndPosition
		}
	case "dependsOn":
		e, exprErr := p.parseExpr()
		if exprErr != nil {
			return exprErr
		}

		var list *schema.StringList
		list, err = exprToResourceNameList(e, "dependsOn")
		if err == nil {
			r.DependsOn = &schema.DependsOnList{StringList: *list}
			if n := len(list.SourceMeta); n > 0 && list.SourceMeta[n-1] != nil {
				valueEnd = list.SourceMeta[n-1].EndPosition
			}
		}
	case "removalPolicy":
		var value string
		var valueMeta *source.Meta
		value, valueMeta, err = p.parseRemovalPolicyValue()
		if err == nil {
			r.RemovalPolicy = &schema.RemovalPolicyWrapper{
				Value:      schema.RemovalPolicy(value),
				SourceMeta: valueMeta,
			}
			if valueMeta != nil {
				valueEnd = valueMeta.EndPosition
			}
		}
	default:
		return p.errf(fieldMeta.Position, "unknown field %q in resource declaration", field)
	}

	if err != nil {
		return err
	}
	r.FieldsSourceMeta[field] = mergeEnd(fieldMeta, valueEnd)
	return nil
}

// mergeEnd extends fieldMeta's range to end at valueEnd, so a FieldsSourceMeta
// entry covers the whole `field = value` span rather than just the field-name
// token. Falls back to fieldMeta when no end is available (e.g. an empty
// dependsOn list).
func mergeEnd(fieldMeta *source.Meta, valueEnd *source.Position) *source.Meta {
	if fieldMeta == nil || valueEnd == nil {
		return fieldMeta
	}
	return &source.Meta{
		Position:    fieldMeta.Position,
		EndPosition: valueEnd,
	}
}

func (p *parser) parseRemovalPolicyValue() (string, *source.Meta, error) {
	if p.peek().Type != TokenStringStart {
		return "", nil, p.errf(
			p.peek().Start,
			"removalPolicy must be a literal string, got %s",
			p.peek().Type,
		)
	}

	value, meta, err := p.collectStringLiteral(false)
	if err != nil {
		return "", nil, err
	}

	for _, valid := range schema.ValidRemovalPolicies {
		if value == string(valid) {
			return value, meta, nil
		}
	}

	return "", nil, p.errf(
		meta.Position,
		"removalPolicy must be one of \"delete\" or \"retain\", got %q",
		value,
	)
}

// Lowers a single-resource-name or array-of-resource-names
// expression to a *schema.StringList for use by `dependsOn` and `select.exclude`.
// Each element must be either a string literal or a bare single-segment
// reference (resolves to a refExpr whose substitution is a ResourceProperty
// with no path accessors).
func exprToResourceNameList(e expr, field string) (*schema.StringList, error) {
	if arr, ok := e.(*arrayExpr); ok {
		values := make([]string, 0, len(arr.elems))
		sourceMetas := make([]*source.Meta, 0, len(arr.elems))
		for _, el := range arr.elems {
			name, meta, err := extractResourceName(el, field)
			if err != nil {
				return nil, err
			}
			values = append(values, name)
			sourceMetas = append(sourceMetas, meta)
		}

		return &schema.StringList{
			Values:     values,
			SourceMeta: sourceMetas,
		}, nil
	}

	name, meta, err := extractResourceName(e, field)
	if err != nil {
		return nil, err
	}

	return &schema.StringList{
		Values:     []string{name},
		SourceMeta: []*source.Meta{meta},
	}, nil
}

func extractResourceName(e expr, field string) (string, *source.Meta, error) {
	if scalar, ok := e.(*scalarExpr); ok && scalar.value.StringValue != nil {
		return *scalar.value.StringValue, scalar.value.SourceMeta, nil
	}

	if ref, ok := e.(*refExpr); ok {
		if rp := ref.sub.ResourceProperty; rp != nil &&
			len(rp.Path) == 0 && rp.ResourceEachTemplateIndex == nil {
			return rp.ResourceName, ref.meta(), nil
		}
	}

	return "", nil, &ParseError{
		Message: "entries in " + field +
			" must be either a string literal or a bare resource name",
		SourceMeta: e.meta(),
	}
}
