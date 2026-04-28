package subwalk

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type WalkTestSuite struct {
	suite.Suite
}

func (s *WalkTestSuite) Test_walks_returns_nil_for_nil_string_or_substitutions() {
	out := WalkStringOrSubstitutions(nil, identityVisitor)
	s.Assert().Nil(out)
}

func (s *WalkTestSuite) Test_walks_returns_nil_for_empty_string_or_substitutions() {
	in := &substitutions.StringOrSubstitutions{}
	out := WalkStringOrSubstitutions(in, identityVisitor)
	s.Assert().Nil(out)
}

func (s *WalkTestSuite) Test_walks_preserves_plain_string_values() {
	plain := "hello"
	in := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{StringValue: &plain},
		},
	}

	out := WalkStringOrSubstitutions(in, identityVisitor)

	s.Require().NotNil(out)
	s.Require().Len(out.Values, 1)
	s.Require().NotNil(out.Values[0].StringValue)
	s.Assert().Equal("hello", *out.Values[0].StringValue)
	s.Assert().Nil(out.Values[0].SubstitutionValue)
}

func (s *WalkTestSuite) Test_walks_visits_each_top_level_substitution_once() {
	in := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{SubstitutionValue: variableSub("a")},
			{SubstitutionValue: variableSub("b")},
		},
	}

	visited := []string{}
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub.Variable != nil {
			visited = append(visited, sub.Variable.VariableName)
		}
		return nil
	}

	out := WalkStringOrSubstitutions(in, visitor)

	s.Assert().Equal([]string{"a", "b"}, visited)
	s.Require().NotNil(out)
	s.Require().Len(out.Values, 2)
	// When the visitor returns nil, the original substitution pointer is preserved.
	s.Assert().Same(in.Values[0].SubstitutionValue, out.Values[0].SubstitutionValue)
	s.Assert().Same(in.Values[1].SubstitutionValue, out.Values[1].SubstitutionValue)
}

func (s *WalkTestSuite) Test_walks_replaces_substitution_when_visitor_returns_replacement() {
	in := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{SubstitutionValue: variableSub("original")},
		},
	}

	replacement := variableSub("replaced")
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		return replacement
	}

	out := WalkStringOrSubstitutions(in, visitor)

	s.Require().NotNil(out)
	s.Require().Len(out.Values, 1)
	s.Assert().Same(replacement, out.Values[0].SubstitutionValue)
	// The input is not mutated.
	s.Assert().Equal("original", in.Values[0].SubstitutionValue.Variable.VariableName)
}

func (s *WalkTestSuite) Test_walks_visits_nested_function_arguments_in_post_order() {
	innerVar := variableSub("inner")
	outer := &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: "trim",
			Arguments: []*substitutions.SubstitutionFunctionArg{
				{Value: innerVar},
			},
		},
	}
	in := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{SubstitutionValue: outer},
		},
	}

	visited := []string{}
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		switch {
		case sub.Variable != nil:
			visited = append(visited, "var:"+sub.Variable.VariableName)
		case sub.Function != nil:
			visited = append(visited, "fn:"+string(sub.Function.FunctionName))
		}
		return nil
	}

	WalkStringOrSubstitutions(in, visitor)

	// Inner substitution should be visited before the wrapping function.
	s.Assert().Equal([]string{"var:inner", "fn:trim"}, visited)
}

func (s *WalkTestSuite) Test_walks_rewrites_nested_function_argument_without_mutating_input() {
	innerVar := variableSub("inner")
	originalFn := &substitutions.SubstitutionFunctionExpr{
		FunctionName: "trim",
		Arguments: []*substitutions.SubstitutionFunctionArg{
			{Name: "value", Value: innerVar},
		},
	}
	originalSub := &substitutions.Substitution{Function: originalFn}
	in := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{SubstitutionValue: originalSub},
		},
	}

	replacement := variableSub("replaced")
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub.Variable != nil && sub.Variable.VariableName == "inner" {
			return replacement
		}
		return nil
	}

	out := WalkStringOrSubstitutions(in, visitor)

	s.Require().NotNil(out)
	s.Require().Len(out.Values, 1)
	rewritten := out.Values[0].SubstitutionValue
	s.Require().NotNil(rewritten)
	s.Require().NotNil(rewritten.Function)
	s.Require().Len(rewritten.Function.Arguments, 1)
	s.Assert().Same(replacement, rewritten.Function.Arguments[0].Value)
	s.Assert().Equal("value", rewritten.Function.Arguments[0].Name)

	// Original input pointers are untouched.
	s.Assert().NotSame(originalSub, rewritten)
	s.Assert().NotSame(originalFn, rewritten.Function)
	s.Assert().Same(innerVar, originalFn.Arguments[0].Value)
}

func (s *WalkTestSuite) Test_walks_returns_nil_for_nil_mapping_node() {
	out := WalkMappingNode(nil, identityVisitor)
	s.Assert().Nil(out)
}

func (s *WalkTestSuite) Test_walks_preserves_scalar_only_mapping_nodes() {
	node := core.MappingNodeFromString("plain-scalar")

	out := WalkMappingNode(node, identityVisitor)

	s.Require().NotNil(out)
	s.Assert().NotSame(node, out)
	s.Require().NotNil(out.Scalar)
	s.Assert().Equal(node.Scalar, out.Scalar)
}

func (s *WalkTestSuite) Test_walks_visits_substitutions_inside_mapping_node_fields() {
	leaf := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{SubstitutionValue: variableSub("nested")},
			},
		},
	}
	node := &core.MappingNode{
		Fields: map[string]*core.MappingNode{"key": leaf},
	}

	visited := []string{}
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub.Variable != nil {
			visited = append(visited, sub.Variable.VariableName)
		}
		return nil
	}

	out := WalkMappingNode(node, visitor)

	s.Assert().Equal([]string{"nested"}, visited)
	s.Require().NotNil(out)
	s.Require().Contains(out.Fields, "key")
	s.Assert().NotSame(node, out)
	s.Assert().NotSame(leaf, out.Fields["key"])
}

func (s *WalkTestSuite) Test_walks_visits_substitutions_inside_mapping_node_items() {
	itemA := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{SubstitutionValue: variableSub("a")},
			},
		},
	}
	itemB := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{SubstitutionValue: variableSub("b")},
			},
		},
	}
	node := &core.MappingNode{Items: []*core.MappingNode{itemA, itemB}}

	visited := []string{}
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub.Variable != nil {
			visited = append(visited, sub.Variable.VariableName)
		}
		return nil
	}

	out := WalkMappingNode(node, visitor)

	s.Assert().Equal([]string{"a", "b"}, visited)
	s.Require().NotNil(out)
	s.Require().Len(out.Items, 2)
	s.Assert().NotSame(node, out)
}

func (s *WalkTestSuite) Test_walks_rewrites_substitutions_in_mapping_node_tree() {
	target := variableSub("target")
	node := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"outer": {
				Items: []*core.MappingNode{
					{
						StringWithSubstitutions: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{SubstitutionValue: target},
							},
						},
					},
				},
			},
		},
	}

	replacement := variableSub("replaced")
	visitor := func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub.Variable != nil && sub.Variable.VariableName == "target" {
			return replacement
		}
		return nil
	}

	out := WalkMappingNode(node, visitor)

	rewritten := out.Fields["outer"].Items[0].StringWithSubstitutions.Values[0].SubstitutionValue
	s.Assert().Same(replacement, rewritten)
	// Original tree is untouched.
	original := node.Fields["outer"].Items[0].StringWithSubstitutions.Values[0].SubstitutionValue
	s.Assert().Same(target, original)
}

func identityVisitor(_ *substitutions.Substitution) *substitutions.Substitution {
	return nil
}

func variableSub(name string) *substitutions.Substitution {
	return &substitutions.Substitution{
		Variable: &substitutions.SubstitutionVariable{VariableName: name},
	}
}

func TestWalkTestSuite(t *testing.T) {
	suite.Run(t, new(WalkTestSuite))
}
