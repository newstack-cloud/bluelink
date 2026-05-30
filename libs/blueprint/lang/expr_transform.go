package lang

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func exprToMappingNode(e expr) (*core.MappingNode, error) {
	switch n := e.(type) {
	case *scalarExpr:
		return &core.MappingNode{
			Scalar:     n.value,
			SourceMeta: n.value.SourceMeta,
		}, nil
	case *arrayExpr:
		return arrayExprToMappingNode(n)
	case *objectExpr:
		return objectExprToMappingNode(n)
	case *refExpr:
		return substitutionToMappingNode(n.sub), nil
	case *callExpr:
		return wrapAsMappingNode(callExprToSubstitution(n))
	case *opExpr:
		return wrapAsMappingNode(opExprToSubstitution(n))
	case *interpolationExpr:
		return interpolationExprToMappingNode(n)
	default:
		return nil, errUnknownExprVariant(e)
	}
}

func exprToSubstitution(e expr) (*substitutions.Substitution, error) {
	switch n := e.(type) {
	case *scalarExpr:
		return scalarToSubstitution(n.value), nil
	case *refExpr:
		return n.sub, nil
	case *callExpr:
		return callExprToSubstitution(n)
	case *opExpr:
		return opExprToSubstitution(n)
	case *arrayExpr:
		return arrayExprToSubstitution(n)
	case *objectExpr:
		return objectExprToSubstitution(n)
	case *interpolationExpr:
		return interpolationExprToSubstitution(n)
	default:
		return nil, errUnknownExprVariant(e)
	}
}

func wrapAsMappingNode(sub *substitutions.Substitution, err error) (*core.MappingNode, error) {
	if err != nil {
		return nil, err
	}
	return substitutionToMappingNode(sub), nil
}

func substitutionToMappingNode(sub *substitutions.Substitution) *core.MappingNode {
	stringOrSubs := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: sub,
				SourceMeta:        sub.SourceMeta,
			},
		},
		SourceMeta: sub.SourceMeta,
	}

	return &core.MappingNode{
		StringWithSubstitutions: stringOrSubs,
		SourceMeta:              sub.SourceMeta,
	}
}

func scalarToSubstitution(s *core.ScalarValue) *substitutions.Substitution {
	out := &substitutions.Substitution{
		SourceMeta: s.SourceMeta,
	}

	switch {
	case s.StringValue != nil:
		out.StringValue = s.StringValue
	case s.IntValue != nil:
		iv := int64(*s.IntValue)
		out.IntValue = &iv
	case s.FloatValue != nil:
		out.FloatValue = s.FloatValue
	case s.BoolValue != nil:
		out.BoolValue = s.BoolValue
	}

	return out
}

func arrayExprToMappingNode(e *arrayExpr) (*core.MappingNode, error) {
	items := make([]*core.MappingNode, 0, len(e.elems))
	for _, el := range e.elems {
		mn, err := exprToMappingNode(el)
		if err != nil {
			return nil, err
		}
		items = append(items, mn)
	}

	return &core.MappingNode{
		Items:      items,
		SourceMeta: e.m,
	}, nil
}

func objectExprToMappingNode(e *objectExpr) (*core.MappingNode, error) {
	fields := make(map[string]*core.MappingNode, len(e.entries))
	fieldsMeta := make(map[string]*source.Meta, len(e.entries))
	for _, entry := range e.entries {
		mn, err := exprToMappingNode(entry.value)
		if err != nil {
			return nil, err
		}
		fields[entry.key] = mn
		fieldsMeta[entry.key] = entry.meta
	}

	return &core.MappingNode{
		Fields:           fields,
		FieldsSourceMeta: fieldsMeta,
		SourceMeta:       e.m,
	}, nil
}

func interpolationExprToMappingNode(e *interpolationExpr) (*core.MappingNode, error) {
	values, err := interpolationPartsToSOSValues(e.parts)
	if err != nil {
		return nil, err
	}

	return &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values:     values,
			SourceMeta: e.m,
		},
		SourceMeta: e.m,
	}, nil
}

// Lowers an expression into a *substitutions.StringOrSubstitutions, the
// schema shape used by field-level positions that the canonical schema
// expresses as "a string with optional substitutions" — filter search values
// in data sources, resource conditions, dependsOn entries, etc.
func exprToStringOrSubstitutions(e expr) (*substitutions.StringOrSubstitutions, error) {
	switch n := e.(type) {
	case *scalarExpr:
		if n.value.StringValue != nil {
			s := *n.value.StringValue
			return &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{{
					StringValue: &s,
					SourceMeta:  n.value.SourceMeta,
				}},
				SourceMeta: n.value.SourceMeta,
			}, nil
		}
		sub := scalarToSubstitution(n.value)
		return &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{{
				SubstitutionValue: sub,
				SourceMeta:        n.value.SourceMeta,
			}},
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
		sub, err := exprToSubstitution(e)
		if err != nil {
			return nil, err
		}
		return &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{{
				SubstitutionValue: sub,
				SourceMeta:        e.meta(),
			}},
			SourceMeta: e.meta(),
		}, nil
	}
}

func objectExprToStringMap(e *objectExpr) (*schema.StringMap, error) {
	values := make(map[string]string, len(e.entries))
	sourceMeta := make(map[string]*source.Meta, len(e.entries))
	for _, entry := range e.entries {
		scalar, ok := entry.value.(*scalarExpr)
		if !ok || scalar.value.StringValue == nil {
			return nil, &ParseError{
				Message:    fmt.Sprintf("expected a string value for key %q", entry.key),
				SourceMeta: entry.value.meta(),
			}
		}

		values[entry.key] = *scalar.value.StringValue
		sourceMeta[entry.key] = entry.meta
	}

	return &schema.StringMap{
		Values:     values,
		SourceMeta: sourceMeta,
	}, nil
}

func objectExprToStringOrSubstitutionsMap(
	e *objectExpr,
) (*schema.StringOrSubstitutionsMap, error) {
	values := make(map[string]*substitutions.StringOrSubstitutions, len(e.entries))
	sourceMeta := make(map[string]*source.Meta, len(e.entries))
	for _, entry := range e.entries {
		sos, err := exprToStringOrSubstitutions(entry.value)
		if err != nil {
			return nil, err
		}

		values[entry.key] = sos
		sourceMeta[entry.key] = entry.meta
	}

	return &schema.StringOrSubstitutionsMap{
		Values:     values,
		SourceMeta: sourceMeta,
	}, nil
}

// Lowers an expression to a *schema.Condition. A bare
// boolean expression becomes the StringValue form (a substitution-wrapped
// expression). An object literal with exactly one of `and`, `or`, or `not`
// becomes the structural form, recursively lowering nested conditions. The
// exactly-one-of-three check mirrors what *Condition.UnmarshalYAML enforces;
// without it, ill-formed condition objects would round-trip through the blueprint
// lang parser unchecked.
func exprToCondition(e expr) (*schema.Condition, error) {
	obj, ok := e.(*objectExpr)
	if !ok {
		sos, err := exprToStringOrSubstitutions(e)
		if err != nil {
			return nil, err
		}

		return &schema.Condition{
			StringValue: sos,
			SourceMeta:  e.meta(),
		}, nil
	}

	return objectExprToCondition(obj)
}

func objectExprToCondition(obj *objectExpr) (*schema.Condition, error) {
	cond := &schema.Condition{SourceMeta: obj.m}
	if len(obj.entries) != 1 {
		return nil, &ParseError{
			Message:    "condition object must have exactly one of 'and', 'or', or 'not'",
			SourceMeta: obj.m,
		}
	}

	entry := obj.entries[0]
	switch entry.key {
	case "and":
		conds, err := conditionListFromExpr(entry.value)
		if err != nil {
			return nil, err
		}

		cond.And = conds
	case "or":
		conds, err := conditionListFromExpr(entry.value)
		if err != nil {
			return nil, err
		}

		cond.Or = conds
	case "not":
		notCond, err := exprToCondition(entry.value)
		if err != nil {
			return nil, err
		}

		cond.Not = notCond
	default:
		return nil, &ParseError{
			Message: fmt.Sprintf(
				"unknown condition object field %q: expected 'and', 'or', or 'not'", entry.key,
			),
			SourceMeta: entry.meta,
		}
	}
	return cond, nil
}

func conditionListFromExpr(e expr) ([]*schema.Condition, error) {
	arr, ok := e.(*arrayExpr)
	if !ok {
		return nil, &ParseError{
			Message:    "'and' and 'or' condition entries must be an array",
			SourceMeta: e.meta(),
		}
	}

	out := make([]*schema.Condition, 0, len(arr.elems))
	for _, el := range arr.elems {
		c, err := exprToCondition(el)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	return out, nil
}

func interpolationPartsToSOSValues(
	parts []interpolationPart,
) ([]*substitutions.StringOrSubstitution, error) {
	out := make([]*substitutions.StringOrSubstitution, 0, len(parts))
	for _, p := range parts {
		v, err := interpolationPartToSOSValue(p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}

	return out, nil
}

func interpolationPartToSOSValue(p interpolationPart) (*substitutions.StringOrSubstitution, error) {
	switch part := p.(type) {
	case *stringPart:
		s := part.value
		return &substitutions.StringOrSubstitution{
			StringValue: &s,
			SourceMeta:  part.m,
		}, nil
	case *substitutionPart:
		sub, err := exprToSubstitution(part.value)
		if err != nil {
			return nil, err
		}

		return &substitutions.StringOrSubstitution{
			SubstitutionValue: sub,
			SourceMeta:        part.m,
		}, nil
	default:
		return nil, &ParseError{
			Message: fmt.Sprintf("internal: unknown interpolation part variant %T", p),
		}
	}
}

func callExprToSubstitution(e *callExpr) (*substitutions.Substitution, error) {
	args, err := callArgsToSubstitutionArgs(e.args)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionName(e.name),
			Arguments:    args,
			Path:         e.path,
			SourceMeta:   e.m,
		},
		SourceMeta: e.m,
	}, nil
}

func callArgsToSubstitutionArgs(args []callArg) ([]*substitutions.SubstitutionFunctionArg, error) {
	out := make([]*substitutions.SubstitutionFunctionArg, 0, len(args))
	for _, a := range args {
		sub, err := exprToSubstitution(a.value)
		if err != nil {
			return nil, err
		}

		out = append(out, &substitutions.SubstitutionFunctionArg{
			Name:       a.name,
			Value:      sub,
			SourceMeta: a.meta,
		})
	}

	return out, nil
}

func opExprToSubstitution(e *opExpr) (*substitutions.Substitution, error) {
	args, err := positionalSubstitutionArgs(e.args)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: e.fn,
			Arguments:    args,
			SourceMeta:   e.m,
		},
		SourceMeta: e.m,
	}, nil
}

func positionalSubstitutionArgs(exprs []expr) ([]*substitutions.SubstitutionFunctionArg, error) {
	out := make([]*substitutions.SubstitutionFunctionArg, 0, len(exprs))
	for _, e := range exprs {
		sub, err := exprToSubstitution(e)
		if err != nil {
			return nil, err
		}

		out = append(out, &substitutions.SubstitutionFunctionArg{
			Value: sub,
		})
	}
	return out, nil
}

func arrayExprToSubstitution(e *arrayExpr) (*substitutions.Substitution, error) {
	args, err := positionalSubstitutionArgs(e.elems)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionList,
			Arguments:    args,
			SourceMeta:   e.m,
		},
		SourceMeta: e.m,
	}, nil
}

func objectExprToSubstitution(e *objectExpr) (*substitutions.Substitution, error) {
	args := make([]*substitutions.SubstitutionFunctionArg, 0, len(e.entries))
	for _, entry := range e.entries {
		sub, err := exprToSubstitution(entry.value)
		if err != nil {
			return nil, err
		}

		args = append(args, &substitutions.SubstitutionFunctionArg{
			Name:       entry.key,
			Value:      sub,
			SourceMeta: entry.meta,
		})
	}

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionObject,
			Arguments:    args,
			SourceMeta:   e.m,
		},
		SourceMeta: e.m,
	}, nil
}

func interpolationExprToSubstitution(e *interpolationExpr) (*substitutions.Substitution, error) {
	if text, ok := joinPlainStringParts(e.parts); ok {
		return &substitutions.Substitution{
			StringValue: &text,
			SourceMeta:  e.m,
		}, nil
	}

	return interpolationToJoinSubstitution(e)
}

func joinPlainStringParts(parts []interpolationPart) (string, bool) {
	var sb strings.Builder
	for _, p := range parts {
		sp, ok := p.(*stringPart)
		if !ok {
			return "", false
		}
		sb.WriteString(sp.value)
	}

	return sb.String(), true
}

// Desugars a multi-part interpolated string into join(list(parts...), "") so it
// can stand as a single Substitution in operand/arg position.
func interpolationToJoinSubstitution(e *interpolationExpr) (*substitutions.Substitution, error) {
	partSubs := make([]*substitutions.SubstitutionFunctionArg, 0, len(e.parts))
	for _, p := range e.parts {
		sub, err := interpolationPartToSubstitution(p)
		if err != nil {
			return nil, err
		}
		partSubs = append(partSubs, &substitutions.SubstitutionFunctionArg{
			Value: sub,
		})
	}

	listSub := &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionList,
			Arguments:    partSubs,
		},
	}
	empty := ""

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionJoin,
			Arguments: []*substitutions.SubstitutionFunctionArg{
				{
					Value: listSub,
				},
				{
					Value: &substitutions.Substitution{
						StringValue: &empty,
					},
				},
			},
			SourceMeta: e.m,
		},
		SourceMeta: e.m,
	}, nil
}

func interpolationPartToSubstitution(p interpolationPart) (*substitutions.Substitution, error) {
	switch part := p.(type) {
	case *stringPart:
		s := part.value
		return &substitutions.Substitution{
			StringValue: &s,
			SourceMeta:  part.m,
		}, nil
	case *substitutionPart:
		return exprToSubstitution(part.value)
	default:
		return nil, &ParseError{
			Message: fmt.Sprintf("internal: unknown interpolation part variant %T", p),
		}
	}
}

func errUnknownExprVariant(e expr) error {
	return &ParseError{
		Message:    fmt.Sprintf("internal: unknown expression variant %T", e),
		SourceMeta: e.meta(),
	}
}
