package lang

import (
	"strconv"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func (p *parser) parseReferenceOrCall() (expr, error) {
	tkn := p.peek()
	switch tkn.tokenType {
	case tokenKeywordVariables:
		// variables.<name>
		return p.parseVariableReference()
	case tokenKeywordValues:
		// values.<name>{ accessor }
		return p.parseValueReference()
	case tokenKeywordResources:
		// resources.<name>[ idx ]?.<spec|metadata>...
		return p.parseResourceReference()
	case tokenKeywordDatasources:
		// datasources.<name>.<field>[ idx ]?
		return p.parseDataSourceReference()
	case tokenKeywordChildren:
		// children.<name>.<output>{ accessor }
		return p.parseChildReference()
	case tokenKeywordElem:
		// elem{ accessor }
		return p.parseElemReference()
	case tokenKeywordI:
		// i - bare element index
		return p.parseElemIndexReference()
	case tokenIdent:
		// `name(` → function call;
		// `name`, `name.`, `name[` → bare  resource reference.
		if p.peekAt(1).tokenType == tokenLeftParen {
			return p.parseFunctionCall()
		}
		return p.parseBareResourceReference()
	default:
		// Some core functions (object, list, string, etc.) share names with
		// reserved words. Allow them as function names when followed by '('.
		// The namespace-keyword cases above take precedence and already returned.
		if isKeyword(tkn.tokenType) && p.peekAt(1).tokenType == tokenLeftParen {
			return p.parseFunctionCall()
		}
		return nil, p.errf(tkn.pos, "expected an expression, got %s", tkn.tokenType)
	}
}

func (p *parser) parseVariableReference() (expr, error) {
	keyword := p.advance() // consume 'variables' keyword

	name, nameMeta, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	tkn := p.peek()
	if tkn.tokenType == tokenPeriod || tkn.tokenType == tokenLeftBracket {
		return nil, p.errf(
			tkn.pos,
			"variables resolve to scalars and cannot have nested values",
		)
	}

	sourceMeta := &source.Meta{
		Position:    keyword.pos,
		EndPosition: nameMeta.EndPosition,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			Variable: &substitutions.SubstitutionVariable{
				VariableName: name,
				SourceMeta:   sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseValueReference() (expr, error) {
	keyword := p.advance() // 'values'

	name, nameMeta, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	path, err := p.parseAccessorChain()
	if err != nil {
		return nil, err
	}

	endPos := nameMeta.EndPosition
	if n := len(path); n > 0 && path[n-1].SourceMeta != nil {
		endPos = path[n-1].SourceMeta.EndPosition
	}
	sourceMeta := &source.Meta{
		Position:    keyword.pos,
		EndPosition: endPos,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			ValueReference: &substitutions.SubstitutionValueReference{
				ValueName:  name,
				Path:       path,
				SourceMeta: sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseChildReference() (expr, error) {
	keyword := p.advance() // 'children'

	name, nameMeta, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	path, err := p.parseAccessorChain()
	if err != nil {
		return nil, err
	}

	if len(path) == 0 {
		return nil, p.errf(
			nameMeta.Position,
			"child reference requires an output name: 'children.%s.<output>'",
			name,
		)
	}

	sourceMeta := &source.Meta{
		Position:    keyword.pos,
		EndPosition: path[len(path)-1].SourceMeta.EndPosition,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			Child: &substitutions.SubstitutionChild{
				ChildName:  name,
				Path:       path,
				SourceMeta: sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseDataSourceReference() (expr, error) {
	keyword := p.advance() // 'datasources'

	dsName, _, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	fieldName, fieldMeta, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	var arrIdx *int64
	endMeta := fieldMeta
	if p.peek().tokenType == tokenLeftBracket {
		item, err := p.parseBracketAccessor()
		if err != nil {
			return nil, err
		}
		// parseBracketAccessor accepts ["name"] / [N] / []; the grammar here
		// restricts the bracket form to *index accessor* only, so reject the
		// quoted-name form.
		if item.FieldName != "" {
			return nil, p.errf(
				item.SourceMeta.Position,
				"data source reference accepts only an index accessor here, not a quoted name",
			)
		}
		arrIdx = item.ArrayIndex
		endMeta = item.SourceMeta
	}

	tkn := p.peek()
	if tkn.tokenType == tokenPeriod || tkn.tokenType == tokenLeftBracket {
		return nil, p.errf(
			tkn.pos,
			"data source references end after the field name (and optional index); got %s",
			tkn.tokenType,
		)
	}

	sourceMeta := &source.Meta{
		Position:    keyword.pos,
		EndPosition: endMeta.EndPosition,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			DataSourceProperty: &substitutions.SubstitutionDataSourceProperty{
				DataSourceName:    dsName,
				FieldName:         fieldName,
				PrimitiveArrIndex: arrIdx,
				SourceMeta:        sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseElemReference() (expr, error) {
	keyword := p.advance() // 'elem'

	path, err := p.parseAccessorChain()
	if err != nil {
		return nil, err
	}

	endPos := &keyword.endPos
	if n := len(path); n > 0 && path[n-1].SourceMeta != nil {
		endPos = path[n-1].SourceMeta.EndPosition
	}
	sourceMeta := &source.Meta{
		Position:    keyword.pos,
		EndPosition: endPos,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			ElemReference: &substitutions.SubstitutionElemReference{
				Path:       path,
				SourceMeta: sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseElemIndexReference() (expr, error) {
	keyword := p.advance() // 'i'

	tkn := p.peek()
	if tkn.tokenType == tokenPeriod || tkn.tokenType == tokenLeftBracket {
		tkn := p.peek()
		return nil, p.errf(
			tkn.pos,
			"'i' is the iteration index and cannot have accessors; got %s",
			tkn.tokenType,
		)
	}

	sourceMeta := sourceMetaFromToken(keyword)
	return &refExpr{
		sub: &substitutions.Substitution{
			ElemIndexReference: &substitutions.SubstitutionElemIndexReference{
				SourceMeta: sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseResourceReference() (expr, error) {
	keyword := p.advance() // 'resources'

	name, nameMeta, err := p.parseNameAccessor()
	if err != nil {
		return nil, err
	}

	return p.finishResourceReference(keyword.pos, name, nameMeta)
}

func (p *parser) parseBareResourceReference() (expr, error) {
	nameTkn, err := p.expect(tokenIdent)
	if err != nil {
		return nil, err
	}

	return p.finishResourceReference(
		nameTkn.pos,
		nameTkn.value,
		sourceMetaFromToken(nameTkn),
	)
}

func (p *parser) finishResourceReference(
	startPos source.Position,
	resourceName string,
	nameMeta *source.Meta,
) (expr, error) {
	eachIdx, eachMeta, err := p.parseOptionalEachIndex()
	if err != nil {
		return nil, err
	}

	path, err := p.parseAccessorChain()
	if err != nil {
		return nil, err
	}

	endPos := nameMeta.EndPosition
	if eachMeta != nil {
		endPos = eachMeta.EndPosition
	}
	if n := len(path); n > 0 && path[n-1].SourceMeta != nil {
		endPos = path[n-1].SourceMeta.EndPosition
	}
	sourceMeta := &source.Meta{
		Position:    startPos,
		EndPosition: endPos,
	}

	return &refExpr{
		sub: &substitutions.Substitution{
			ResourceProperty: &substitutions.SubstitutionResourceProperty{
				ResourceName:              resourceName,
				ResourceEachTemplateIndex: eachIdx,
				Path:                      path,
				SourceMeta:                sourceMeta,
			},
			SourceMeta: sourceMeta,
		},
	}, nil
}

func (p *parser) parseOptionalEachIndex() (*int64, *source.Meta, error) {
	if p.peek().tokenType != tokenLeftBracket {
		return nil, nil, nil
	}

	// A `["quotedName"]` immediately after the resource name is not an
	// each-template index — it's the quoted-name form of a regular field
	// accessor. Leave it for parseAccessorChain to consume.
	if p.peekAt(1).tokenType == tokenStringStart {
		return nil, nil, nil
	}

	item, err := p.parseBracketAccessor()
	if err != nil {
		return nil, nil, err
	}

	return item.ArrayIndex, item.SourceMeta, nil
}

func (p *parser) parseAccessorChain() ([]*substitutions.SubstitutionPathItem, error) {
	var path []*substitutions.SubstitutionPathItem

	for {
		var item *substitutions.SubstitutionPathItem
		var err error

		switch p.peek().tokenType {
		case tokenPeriod:
			item, err = p.parseDotAccessor()
		case tokenLeftBracket:
			item, err = p.parseBracketAccessor()
		default:
			return path, nil
		}
		if err != nil {
			return nil, err
		}

		path = append(path, item)
	}
}

func (p *parser) parseDotAccessor() (*substitutions.SubstitutionPathItem, error) {
	dot := p.advance()

	nameTkn := p.peek()
	if nameTkn.tokenType != tokenIdent && !isKeyword(nameTkn.tokenType) {
		return nil, p.errf(
			nameTkn.pos,
			"expected a field name, got %s",
			nameTkn.tokenType,
		)
	}
	p.advance()

	return &substitutions.SubstitutionPathItem{
		FieldName: nameTkn.value,
		SourceMeta: &source.Meta{
			Position:    dot.pos,
			EndPosition: &nameTkn.endPos,
		},
	}, nil
}

// ["name"]  →  FieldName
// [N]       →  ArrayIndex
// []        →  alias for [0], shorthand for accessing the first element
func (p *parser) parseBracketAccessor() (*substitutions.SubstitutionPathItem, error) {
	open := p.advance()

	item := &substitutions.SubstitutionPathItem{}
	switch p.peek().tokenType {
	case tokenStringStart:
		name, _, err := p.parseQuotedName()
		if err != nil {
			return nil, err
		}
		item.FieldName = name
	case tokenIntLiteral:
		intTkn := p.advance()
		idx, _ := strconv.ParseInt(intTkn.value, 10, 64)
		if idx < 0 {
			return nil, p.errf(
				intTkn.pos,
				"array index must be a non-negative integer, got %d",
				idx,
			)
		}
		item.ArrayIndex = &idx
	case tokenRightBracket:
		// '[]' defaults to index 0 as per spec, there's no AST representation for
		// an empty index (nil means absent), so the YAML/JWCC parser encodes it
		// as 0 at parse time and we match.
		zero := int64(0)
		item.ArrayIndex = &zero
	default:
		tkn := p.peek()
		return nil, p.errf(
			tkn.pos,
			"expected a quoted name, an index or ']', got %s",
			tkn.tokenType,
		)
	}

	closeTkn, err := p.expect(tokenRightBracket)
	if err != nil {
		return nil, err
	}

	item.SourceMeta = &source.Meta{
		Position:    open.pos,
		EndPosition: &closeTkn.endPos,
	}

	return item, nil
}

// One `name accessor`: .name or ["quotedName"]. Distinct from parseAccessorChain
// (which loops) and parseBracketAccessor (which also accepts indexes). Used by
// every reference parser to read its required leading name(s).
func (p *parser) parseNameAccessor() (string, *source.Meta, error) {
	switch p.peek().tokenType {
	case tokenPeriod:
		dot := p.advance()
		nameTkn, err := p.expect(tokenIdent)
		if err != nil {
			return "", nil, err
		}

		return nameTkn.value, &source.Meta{
			Position:    dot.pos,
			EndPosition: &nameTkn.endPos,
		}, nil
	case tokenLeftBracket:
		open := p.advance()
		name, _, err := p.parseQuotedName()
		if err != nil {
			return "", nil, err
		}

		closeTkn, err := p.expect(tokenRightBracket)
		if err != nil {
			return "", nil, err
		}

		return name, &source.Meta{
			Position:    open.pos,
			EndPosition: &closeTkn.endPos,
		}, nil
	default:
		tkn := p.peek()
		return "", nil, p.errf(
			tkn.pos,
			"expected '.<name>' or '[\"<name>\"]', got %s",
			tkn.tokenType,
		)
	}
}

func (p *parser) parseFunctionCall() (expr, error) {
	nameTkn := p.peek()
	if nameTkn.tokenType != tokenIdent && !isKeyword(nameTkn.tokenType) {
		return nil, p.errf(
			nameTkn.pos,
			"expected a function name, got %s",
			nameTkn.tokenType,
		)
	}
	p.advance()

	if _, err := p.expect(tokenLeftParen); err != nil {
		return nil, err
	}

	args, err := p.parseCallArgs()
	if err != nil {
		return nil, err
	}

	closeTkn, err := p.expect(tokenRightParen)
	if err != nil {
		return nil, err
	}

	path, err := p.parseAccessorChain()
	if err != nil {
		return nil, err
	}

	endPos := &closeTkn.endPos
	if n := len(path); n > 0 && path[n-1].SourceMeta != nil {
		endPos = path[n-1].SourceMeta.EndPosition
	}

	return &callExpr{
		name: nameTkn.value,
		args: args,
		path: path,
		m: &source.Meta{
			Position:    nameTkn.pos,
			EndPosition: endPos,
		},
	}, nil
}

// Zero or more comma-separated arguments, with an optional trailing comma.
// Stops at ')'.
func (p *parser) parseCallArgs() ([]callArg, error) {
	var args []callArg
	if p.peek().tokenType == tokenRightParen {
		return args, nil
	}

	for {
		arg, err := p.parseCallArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if !p.match(tokenComma) {
			return args, nil
		}

		// Trailing comma support: if next is ')', stop.
		if p.peek().tokenType == tokenRightParen {
			return args, nil
		}
	}
}

// One function argument: either `name = expression` (named) or `expression`
// (positional). The disambiguation is one token of lookahead past an ident
// for '=' — an ident alone is a bare resource reference (positional).
func (p *parser) parseCallArg() (callArg, error) {
	start := p.peek()
	if start.tokenType == tokenIdent && p.peekAt(1).tokenType == tokenAssign {
		nameTkn := p.advance()
		p.advance() // '='
		value, err := p.parseExpr()
		if err != nil {
			return callArg{}, err
		}

		meta := &source.Meta{Position: nameTkn.pos}
		if valueMeta := value.meta(); valueMeta != nil {
			meta.EndPosition = valueMeta.EndPosition
		}
		return callArg{
			name:  nameTkn.value,
			value: value,
			meta:  meta,
		}, nil
	}

	value, err := p.parseExpr()
	if err != nil {
		return callArg{}, err
	}

	return callArg{
		value: value,
		meta:  value.meta(),
	}, nil
}
