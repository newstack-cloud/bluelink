package subwalk

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// SubstitutionVisitor is called for each substitution encountered during traversal.
// Returning a non-nil *Substitution replaces the original (rewrite); returning nil keeps it.
type SubstitutionVisitor func(sub *substitutions.Substitution) *substitutions.Substitution

// WalkStringOrSubstitutions traverses a single StringOrSubstitutions value,
// calling the visitor for each Substitution found (including nested ones
// inside function arguments).
func WalkStringOrSubstitutions(
	stringOrSubs *substitutions.StringOrSubstitutions,
	visitor SubstitutionVisitor,
) *substitutions.StringOrSubstitutions {
	if stringOrSubs == nil || len(stringOrSubs.Values) == 0 {
		return nil
	}

	finalStringOrSubs := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{},
	}

	for _, stringOrSub := range stringOrSubs.Values {
		if stringOrSub.StringValue != nil {
			// Plain string, keep as is.
			finalStringOrSubs.Values = append(
				finalStringOrSubs.Values,
				&substitutions.StringOrSubstitution{
					StringValue: stringOrSub.StringValue,
					SourceMeta:  stringOrSub.SourceMeta,
				},
			)
		}

		if stringOrSub.SubstitutionValue != nil {
			subVal := walkSubstitution(stringOrSub.SubstitutionValue, visitor)
			if subVal == nil {
				subVal = stringOrSub.SubstitutionValue
			}
			finalStringOrSubs.Values = append(
				finalStringOrSubs.Values,
				&substitutions.StringOrSubstitution{
					SubstitutionValue: subVal,
					SourceMeta:        stringOrSub.SourceMeta,
				},
			)
		}
	}

	return finalStringOrSubs
}

func walkSubstitution(subVal *substitutions.Substitution, visitor SubstitutionVisitor) *substitutions.Substitution {
	current := subVal

	if subVal.Function != nil {
		newArgs := make([]*substitutions.SubstitutionFunctionArg, len(subVal.Function.Arguments))
		changed := false
		for i, arg := range subVal.Function.Arguments {
			if arg == nil || arg.Value == nil {
				newArgs[i] = arg
				continue
			}

			rewrittenArg := walkSubstitution(arg.Value, visitor)
			if rewrittenArg != arg.Value {
				newArgs[i] = &substitutions.SubstitutionFunctionArg{
					Name:       arg.Name,
					Value:      rewrittenArg,
					SourceMeta: arg.SourceMeta,
				}
				changed = true
			} else {
				newArgs[i] = arg
			}
		}

		if changed {
			newFn := *subVal.Function
			newFn.Arguments = newArgs
			newSub := *subVal
			newSub.Function = &newFn
			current = &newSub
		}
	}

	if replacement := visitor(current); replacement != nil {
		return replacement
	}

	return current
}

// WalkMappingNode recursively traverses a core.MappingNode tree,
// finding all embedded StringOrSubstitutions values and calling the visitor.
// MappingNodes store resource specs as untyped trees — substitution references
// like ${resources.X.spec.Y} live inside scalar leaves.
func WalkMappingNode(
	node *core.MappingNode,
	visitor SubstitutionVisitor,
) *core.MappingNode {
	if node == nil {
		return nil
	}

	newFields := map[string]*core.MappingNode{}
	if node.Fields != nil {
		newFields = make(map[string]*core.MappingNode, len(node.Fields))
		for k, v := range node.Fields {
			newFields[k] = WalkMappingNode(v, visitor)
		}
	}

	newItems := []*core.MappingNode{}
	if node.Items != nil {
		newItems = make([]*core.MappingNode, len(node.Items))
		for i, item := range node.Items {
			newItems[i] = WalkMappingNode(item, visitor)
		}
	}

	newStrWithSubs := WalkStringOrSubstitutions(node.StringWithSubstitutions, visitor)

	out := *node
	out.Fields = newFields
	out.Items = newItems
	out.StringWithSubstitutions = newStrWithSubs
	return &out
}
