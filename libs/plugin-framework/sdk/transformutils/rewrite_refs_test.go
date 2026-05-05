package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type RewriteRefsTestSuite struct {
	suite.Suite
}

func (s *RewriteRefsTestSuite) Test_RewriteResourcePropertyRefs_invokes_rewriter_for_resource_property() {
	called := 0
	replacement := ValueRef("myHandler_lambda_arn")

	rewriter := func(ref *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
		called++
		s.Assert().Equal("myHandler", ref.ResourceName)
		return replacement
	}

	visitor := RewriteResourcePropertyRefs(rewriter)
	original := MakeRef("myHandler", []*substitutions.SubstitutionPathItem{
		Field("spec"), Field("arn"),
	})

	got := visitor(original)

	s.Assert().Equal(1, called)
	s.Assert().Same(replacement, got)
}

func (s *RewriteRefsTestSuite) Test_RewriteResourcePropertyRefs_returns_original_when_rewriter_returns_nil() {
	rewriter := func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
		return nil
	}

	visitor := RewriteResourcePropertyRefs(rewriter)
	original := MakeRef("myHandler", []*substitutions.SubstitutionPathItem{
		Field("spec"), Field("memory"),
	})

	got := visitor(original)

	s.Assert().Same(original, got)
}

func (s *RewriteRefsTestSuite) Test_RewriteResourcePropertyRefs_passes_through_non_resource_property_substitutions() {
	called := 0
	rewriter := func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
		called++
		return ValueRef("should_not_appear")
	}

	visitor := RewriteResourcePropertyRefs(rewriter)

	cases := []struct {
		name string
		sub  *substitutions.Substitution
	}{
		{name: "variable", sub: variableSub("env")},
		{name: "value reference", sub: ValueRef("region")},
		{
			name: "function expression",
			sub: &substitutions.Substitution{
				Function: &substitutions.SubstitutionFunctionExpr{
					FunctionName: substitutions.SubstitutionFunctionName("upper"),
				},
			},
		},
		{name: "string literal", sub: stringLiteralSub("hello")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := visitor(tc.sub)
			s.Assert().Same(tc.sub, got)
		})
	}

	s.Assert().Equal(0, called, "rewriter should never be invoked for non-resource-property subs")
}

func (s *RewriteRefsTestSuite) Test_RewriteResourcePropertyRefs_handles_nil_substitution() {
	rewriter := func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
		s.FailNow("rewriter should not be invoked for a nil substitution")
		return nil
	}

	visitor := RewriteResourcePropertyRefs(rewriter)

	s.Assert().Nil(visitor(nil))
}

func (s *RewriteRefsTestSuite) Test_ChainResourcePropertyRewriters_first_match_wins() {
	first := ValueRef("first_match")
	second := ValueRef("second_match")

	chain := ChainResourcePropertyRewriters(
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return first },
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return second },
	)

	got := chain(&substitutions.SubstitutionResourceProperty{ResourceName: "x"})

	s.Assert().Same(first, got)
}

func (s *RewriteRefsTestSuite) Test_ChainResourcePropertyRewriters_falls_through_to_next_on_nil() {
	second := ValueRef("second_match")

	chain := ChainResourcePropertyRewriters(
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return nil },
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return second },
	)

	got := chain(&substitutions.SubstitutionResourceProperty{ResourceName: "x"})

	s.Assert().Same(second, got)
}

func (s *RewriteRefsTestSuite) Test_ChainResourcePropertyRewriters_returns_nil_when_no_rewriter_matches() {
	chain := ChainResourcePropertyRewriters(
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return nil },
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution { return nil },
	)

	got := chain(&substitutions.SubstitutionResourceProperty{ResourceName: "x"})

	s.Assert().Nil(got)
}

func (s *RewriteRefsTestSuite) Test_ChainResourcePropertyRewriters_empty_chain_returns_nil() {
	chain := ChainResourcePropertyRewriters()

	got := chain(&substitutions.SubstitutionResourceProperty{ResourceName: "x"})

	s.Assert().Nil(got)
}

func (s *RewriteRefsTestSuite) Test_PathMatches() {
	cases := []struct {
		name     string
		ref      *substitutions.SubstitutionResourceProperty
		segments []string
		expected bool
	}{
		{
			name:     "nil ref with no segments matches",
			ref:      nil,
			segments: nil,
			expected: true,
		},
		{
			name:     "nil ref with segments does not match",
			ref:      nil,
			segments: []string{"spec"},
			expected: false,
		},
		{
			name:     "empty path with no segments matches",
			ref:      &substitutions.SubstitutionResourceProperty{},
			segments: nil,
			expected: true,
		},
		{
			name:     "empty path with segments does not match",
			ref:      &substitutions.SubstitutionResourceProperty{},
			segments: []string{"spec"},
			expected: false,
		},
		{
			name:     "single segment prefix",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec"},
			expected: true,
		},
		{
			name:     "full field path as prefix",
			ref:      resourcePropertyRef("h", Field("spec"), Field("vpc"), Field("securityGroups"), Index(0)),
			segments: []string{"spec", "vpc", "securityGroups"},
			expected: true,
		},
		{
			name:     "skips array index between fields",
			ref:      resourcePropertyRef("h", Field("spec"), Field("environmentVariables"), Index(2), Field("value")),
			segments: []string{"spec", "environmentVariables", "value"},
			expected: true,
		},
		{
			name:     "skips consecutive array indices for multi-dimensional arrays",
			ref:      resourcePropertyRef("h", Field("spec"), Field("matrix"), Index(0), Index(1), Field("value")),
			segments: []string{"spec", "matrix", "value"},
			expected: true,
		},
		{
			name:     "trailing array index is allowed for prefix match",
			ref:      resourcePropertyRef("h", Field("spec"), Field("tags"), Index(0)),
			segments: []string{"spec", "tags"},
			expected: true,
		},
		{
			name:     "field name mismatch fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec", "cpu"},
			expected: false,
		},
		{
			name:     "segments longer than path fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec", "memory", "value"},
			expected: false,
		},
		{
			name:     "non-prefix start fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("vpc"), Field("securityGroups")),
			segments: []string{"vpc", "securityGroups"},
			expected: false,
		},
		{
			name:     "empty segments always match a non-nil ref",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: nil,
			expected: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := PathMatches(tc.ref, tc.segments...)
			s.Assert().Equal(tc.expected, got)
		})
	}
}

func (s *RewriteRefsTestSuite) Test_PathExact() {
	cases := []struct {
		name     string
		ref      *substitutions.SubstitutionResourceProperty
		segments []string
		expected bool
	}{
		{
			name:     "nil ref with no segments matches",
			ref:      nil,
			segments: nil,
			expected: true,
		},
		{
			name:     "nil ref with segments does not match",
			ref:      nil,
			segments: []string{"spec"},
			expected: false,
		},
		{
			name:     "exact field-only path",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec", "memory"},
			expected: true,
		},
		{
			name:     "trailing array index is ignored",
			ref:      resourcePropertyRef("h", Field("spec"), Field("tags"), Index(0)),
			segments: []string{"spec", "tags"},
			expected: true,
		},
		{
			name:     "consecutive trailing array indices are ignored",
			ref:      resourcePropertyRef("h", Field("spec"), Field("matrix"), Index(0), Index(1)),
			segments: []string{"spec", "matrix"},
			expected: true,
		},
		{
			name:     "mid-path array index is ignored",
			ref:      resourcePropertyRef("h", Field("spec"), Field("environmentVariables"), Index(2), Field("value")),
			segments: []string{"spec", "environmentVariables", "value"},
			expected: true,
		},
		{
			name:     "extra trailing field beyond segments fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory"), Field("value")),
			segments: []string{"spec", "memory"},
			expected: false,
		},
		{
			name:     "missing trailing field fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec", "memory", "value"},
			expected: false,
		},
		{
			name:     "field mismatch fails",
			ref:      resourcePropertyRef("h", Field("spec"), Field("memory")),
			segments: []string{"spec", "cpu"},
			expected: false,
		},
		{
			name:     "empty segments do not match a non-empty path",
			ref:      resourcePropertyRef("h", Field("spec")),
			segments: nil,
			expected: false,
		},
		{
			name:     "empty segments match a path of only array indices",
			ref:      resourcePropertyRef("h", Index(0), Index(1)),
			segments: nil,
			expected: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := PathExact(tc.ref, tc.segments...)
			s.Assert().Equal(tc.expected, got)
		})
	}
}

func (s *RewriteRefsTestSuite) Test_RetargetRef_returns_nil_for_nil_ref() {
	s.Assert().Nil(RetargetRef(nil, "newName"))
}

func (s *RewriteRefsTestSuite) Test_RetargetRef_preserves_path_and_metadata() {
	templateIndex := int64(3)
	sourceMeta := &source.Meta{Position: source.Position{Line: 12, Column: 4}}
	ref := &substitutions.SubstitutionResourceProperty{
		ResourceName:              "myHandler",
		ResourceEachTemplateIndex: &templateIndex,
		Path: []*substitutions.SubstitutionPathItem{
			Field("spec"), Field("vpc"),
			Field("securityGroups"), Index(0),
		},
		SourceMeta: sourceMeta,
	}

	got := RetargetRef(ref, "myHandler_lambda")

	s.Require().NotNil(got)
	s.Require().NotNil(got.ResourceProperty)
	s.Assert().Equal("myHandler_lambda", got.ResourceProperty.ResourceName)
	s.Assert().Same(ref.Path[0], got.ResourceProperty.Path[0])
	s.Assert().Same(ref.ResourceEachTemplateIndex, got.ResourceProperty.ResourceEachTemplateIndex)
	s.Assert().Same(sourceMeta, got.ResourceProperty.SourceMeta)
	// The original ref should be untouched.
	s.Assert().Equal("myHandler", ref.ResourceName)
}

func (s *RewriteRefsTestSuite) Test_RewriteFields() {
	cases := []struct {
		name      string
		ref       *substitutions.SubstitutionResourceProperty
		newName   string
		newFields []string
		expected  []*substitutions.SubstitutionPathItem
	}{
		{
			name:      "one-to-one rename",
			ref:       resourcePropertyRef("h", Field("spec"), Field("memory")),
			newName:   "h_lambda",
			newFields: []string{"spec", "memorySize"},
			expected:  []*substitutions.SubstitutionPathItem{Field("spec"), Field("memorySize")},
		},
		{
			name: "preserves mid-path array index",
			ref: resourcePropertyRef(
				"h",
				Field("spec"), Field("routes"),
				Index(0), Field("method"),
			),
			newName:   "h_api",
			newFields: []string{"spec", "paths", "httpMethod"},
			expected: []*substitutions.SubstitutionPathItem{
				Field("spec"), Field("paths"),
				Index(0), Field("httpMethod"),
			},
		},
		{
			name: "preserves multi-dimensional array indices",
			ref: resourcePropertyRef(
				"h",
				Field("spec"), Field("rules"),
				Index(0), Field("targets"),
				Index(1), Field("arn"),
			),
			newName:   "h_alb",
			newFields: []string{"spec", "rules", "destinations", "arn"},
			expected: []*substitutions.SubstitutionPathItem{
				Field("spec"), Field("rules"),
				Index(0), Field("destinations"),
				Index(1), Field("arn"),
			},
		},
		{
			name: "fewer newFields than source fields keeps remainder unchanged",
			ref: resourcePropertyRef(
				"h",
				Field("spec"), Field("a"),
				Field("b"), Field("c"),
			),
			newName:   "h2",
			newFields: []string{"spec", "renamed"},
			expected: []*substitutions.SubstitutionPathItem{
				Field("spec"), Field("renamed"),
				Field("b"), Field("c"),
			},
		},
		{
			name:      "extra newFields are appended as fields after the source path",
			ref:       resourcePropertyRef("h", Field("spec"), Field("a")),
			newName:   "h2",
			newFields: []string{"spec", "first", "added1", "added2"},
			expected: []*substitutions.SubstitutionPathItem{
				Field("spec"), Field("first"),
				Field("added1"), Field("added2"),
			},
		},
		{
			name:      "empty newFields keeps the original path",
			ref:       resourcePropertyRef("h", Field("spec"), Field("memory")),
			newName:   "h2",
			newFields: nil,
			expected:  []*substitutions.SubstitutionPathItem{Field("spec"), Field("memory")},
		},
		{
			name:      "empty source path appends newFields as field-only items",
			ref:       resourcePropertyRef("h"),
			newName:   "h2",
			newFields: []string{"spec", "memory"},
			expected:  []*substitutions.SubstitutionPathItem{Field("spec"), Field("memory")},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := RewriteFields(tc.ref, tc.newName, tc.newFields...)
			s.Require().NotNil(got)
			s.Require().NotNil(got.ResourceProperty)
			s.Assert().Equal(tc.newName, got.ResourceProperty.ResourceName)
			s.Assert().Equal(tc.expected, got.ResourceProperty.Path)
		})
	}
}

func (s *RewriteRefsTestSuite) Test_RewriteFields_returns_nil_for_nil_ref() {
	s.Assert().Nil(RewriteFields(nil, "newName", "spec", "memory"))
}

func (s *RewriteRefsTestSuite) Test_RewriteFields_preserves_resource_template_index() {
	templateIndex := int64(7)
	ref := &substitutions.SubstitutionResourceProperty{
		ResourceName:              "h",
		ResourceEachTemplateIndex: &templateIndex,
		Path: []*substitutions.SubstitutionPathItem{
			Field("spec"), Field("memory"),
		},
	}

	got := RewriteFields(ref, "h_lambda", "spec", "memorySize")

	s.Require().NotNil(got)
	s.Require().NotNil(got.ResourceProperty)
	s.Assert().Same(ref.ResourceEachTemplateIndex, got.ResourceProperty.ResourceEachTemplateIndex)
}

func (s *RewriteRefsTestSuite) Test_MakeRef() {
	cases := []struct {
		name     string
		resource string
		path     []*substitutions.SubstitutionPathItem
	}{
		{
			name:     "empty path",
			resource: "myValue",
			path:     nil,
		},
		{
			name:     "field-only path",
			resource: "myHandler_lambda",
			path:     []*substitutions.SubstitutionPathItem{Field("spec"), Field("memorySize")},
		},
		{
			name:     "path with array indices",
			resource: "myQueue_sqs",
			path: []*substitutions.SubstitutionPathItem{
				Field("spec"), Field("redrivePolicy"),
				Index(0), Field("deadLetterTargetArn"),
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := MakeRef(tc.resource, tc.path)
			s.Require().NotNil(got)
			s.Require().NotNil(got.ResourceProperty)
			s.Assert().Equal(tc.resource, got.ResourceProperty.ResourceName)
			s.Assert().Equal(tc.path, got.ResourceProperty.Path)
			s.Assert().Nil(got.ResourceProperty.ResourceEachTemplateIndex)
			s.Assert().Nil(got.ResourceProperty.SourceMeta)
		})
	}
}

func (s *RewriteRefsTestSuite) Test_Field() {
	got := Field("memorySize")

	s.Require().NotNil(got)
	s.Assert().Equal("memorySize", got.FieldName)
	s.Assert().Nil(got.ArrayIndex)
}

func (s *RewriteRefsTestSuite) Test_Index() {
	cases := []struct {
		name     string
		input    int
		expected int64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 42, expected: 42},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := Index(tc.input)
			s.Require().NotNil(got)
			s.Assert().Equal("", got.FieldName)
			s.Require().NotNil(got.ArrayIndex)
			s.Assert().Equal(tc.expected, *got.ArrayIndex)
		})
	}
}

func (s *RewriteRefsTestSuite) Test_ValueRef() {
	cases := []struct {
		name      string
		valueName string
		path      []*substitutions.SubstitutionPathItem
	}{
		{
			name:      "flat form with no path",
			valueName: "ordersHandler_lambda_arn",
			path:      nil,
		},
		{
			name:      "with nested field path",
			valueName: "ordersDb_connection",
			path:      []*substitutions.SubstitutionPathItem{Field("host")},
		},
		{
			name:      "with array index and field path",
			valueName: "api_endpoints",
			path:      []*substitutions.SubstitutionPathItem{Index(0), Field("url")},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := ValueRef(tc.valueName, tc.path...)
			s.Require().NotNil(got)
			s.Require().NotNil(got.ValueReference)
			s.Assert().Equal(tc.valueName, got.ValueReference.ValueName)
			s.Assert().Equal(tc.path, got.ValueReference.Path)
		})
	}
}

func resourcePropertyRef(
	name string,
	items ...*substitutions.SubstitutionPathItem,
) *substitutions.SubstitutionResourceProperty {
	return &substitutions.SubstitutionResourceProperty{
		ResourceName: name,
		Path:         items,
	}
}

func variableSub(name string) *substitutions.Substitution {
	return &substitutions.Substitution{
		Variable: &substitutions.SubstitutionVariable{VariableName: name},
	}
}

func stringLiteralSub(value string) *substitutions.Substitution {
	v := value
	return &substitutions.Substitution{StringValue: &v}
}

func TestRewriteRefsTestSuite(t *testing.T) {
	suite.Run(t, new(RewriteRefsTestSuite))
}
