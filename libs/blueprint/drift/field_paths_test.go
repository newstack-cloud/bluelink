package drift

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type FieldPathsTestSuite struct {
	suite.Suite
}

func (s *FieldPathsTestSuite) Test_split_field_path_segments() {
	testCases := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "single segment",
			path:     "tags",
			expected: []string{"tags"},
		},
		{
			name:     "dotted path",
			path:     "vpcConfig.subnetIds",
			expected: []string{"vpcConfig", "subnetIds"},
		},
		{
			name:     "index selector stays with its segment",
			path:     "vpcConfig.subnetIds[2]",
			expected: []string{"vpcConfig", "subnetIds[2]"},
		},
		{
			name:     "dot inside a filter selector does not separate segments",
			path:     `policy.statement[@.sid="AllowInvoke"].effect`,
			expected: []string{"policy", `statement[@.sid="AllowInvoke"]`, "effect"},
		},
		{
			name:     "dot inside a quoted selector value does not separate segments",
			path:     `environment.variables[@.name="TABLE.NAME"].value`,
			expected: []string{"environment", `variables[@.name="TABLE.NAME"]`, "value"},
		},
		{
			name:     "selector nested under an array stays in one segment",
			path:     `policy.statement[@.resources[0]="arn"].effect`,
			expected: []string{"policy", `statement[@.resources[0]="arn"]`, "effect"},
		},
		{
			name:     "consecutive selectors stay with their segment",
			path:     "policy.statement[0][1].action",
			expected: []string{"policy", "statement[0][1]", "action"},
		},
		{
			name:     "empty path yields a single empty segment",
			path:     "",
			expected: []string{""},
		},
		{
			name:     "trailing separator yields a trailing empty segment",
			path:     "policy.",
			expected: []string{"policy", ""},
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			s.Equal(testCase.expected, SplitFieldPathSegments(testCase.path))
		})
	}
}

func (s *FieldPathsTestSuite) Test_split_segment_selectors() {
	testCases := []struct {
		name          string
		segment       string
		expectedName  string
		expectedCount int
	}{
		{
			name:          "field with no selector",
			segment:       "statement",
			expectedName:  "statement",
			expectedCount: 0,
		},
		{
			name:          "field with an index selector",
			segment:       "statement[0]",
			expectedName:  "statement",
			expectedCount: 1,
		},
		{
			name:          "field with a filter selector",
			segment:       `statement[@.sid="AllowInvoke"]`,
			expectedName:  "statement",
			expectedCount: 1,
		},
		{
			name:          "field with consecutive selectors",
			segment:       "statement[0][1]",
			expectedName:  "statement",
			expectedCount: 2,
		},
		{
			name:          "selector nested under an array counts once",
			segment:       `statement[@.resources[0]="arn"]`,
			expectedName:  "statement",
			expectedCount: 1,
		},
		{
			name:          "nested selector followed by a further selection",
			segment:       `statement[@.condition[0].key="k"][2]`,
			expectedName:  "statement",
			expectedCount: 2,
		},
		{
			name:          "empty segment",
			segment:       "",
			expectedName:  "",
			expectedCount: 0,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			name, count := SplitSegmentSelectors(testCase.segment)
			s.Equal(testCase.expectedName, name)
			s.Equal(testCase.expectedCount, count)
		})
	}
}

func (s *FieldPathsTestSuite) Test_schema_for_resource_field_path() {
	specSchema := testResourceSpecSchema()

	testCases := []struct {
		name     string
		path     string
		expected *provider.ResourceDefinitionsSchema
	}{
		{
			name:     "spec prefix alone resolves to the root schema",
			path:     "spec",
			expected: specSchema,
		},
		{
			name:     "spec prefix with a trailing separator resolves to the root schema",
			path:     "spec.",
			expected: specSchema,
		},
		{
			name:     "top level field",
			path:     "spec.policy",
			expected: specSchema.Attributes["policy"],
		},
		{
			name:     "path without the spec prefix addresses the same field",
			path:     "policy",
			expected: specSchema.Attributes["policy"],
		},
		{
			name:     "array field without a selector resolves to the array",
			path:     "spec.policy.statement",
			expected: specSchema.Attributes["policy"].Attributes["statement"],
		},
		{
			name:     "index selector descends into the array items",
			path:     "spec.policy.statement[0]",
			expected: specSchema.Attributes["policy"].Attributes["statement"].Items,
		},
		{
			name:     "filter selector descends into the array items",
			path:     `spec.policy.statement[@.sid="AllowInvoke"]`,
			expected: specSchema.Attributes["policy"].Attributes["statement"].Items,
		},
		{
			name: "selector nested under an array descends a single level",
			path: `spec.policy.statement[@.resources[0]="arn"].effect`,
			expected: specSchema.Attributes["policy"].
				Attributes["statement"].Items.Attributes["effect"],
		},
		{
			name: "field of a selected array item",
			path: `spec.policy.statement[@.sid="AllowInvoke"].resources[0]`,
			expected: specSchema.Attributes["policy"].
				Attributes["statement"].Items.Attributes["resources"].Items,
		},
		{
			name:     "unordered array field",
			path:     "spec.vpcConfig.subnetIds",
			expected: specSchema.Attributes["vpcConfig"].Attributes["subnetIds"],
		},
		{
			name:     "unknown field does not resolve",
			path:     "spec.policy.missing",
			expected: nil,
		},
		{
			name:     "selector on a field that holds no items does not resolve",
			path:     "spec.policy.version[0]",
			expected: nil,
		},
		{
			name:     "selector beyond the depth of an array does not resolve",
			path:     "spec.vpcConfig.subnetIds[0][1]",
			expected: nil,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			s.Equal(testCase.expected, SchemaForResourceFieldPath(specSchema, testCase.path))
		})
	}
}

func (s *FieldPathsTestSuite) Test_schema_for_resource_field_path_without_a_schema() {
	s.Nil(SchemaForResourceFieldPath(nil, "spec.policy"))
}

func testResourceSpecSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"policy": {
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"version": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"statement": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeObject,
							Attributes: map[string]*provider.ResourceDefinitionsSchema{
								"sid": {
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
								"effect": {
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
								"resources": {
									Type: provider.ResourceDefinitionsSchemaTypeArray,
									Items: &provider.ResourceDefinitionsSchema{
										Type: provider.ResourceDefinitionsSchemaTypeString,
									},
								},
							},
						},
					},
				},
			},
			"vpcConfig": {
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"subnetIds": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						IgnoreItemOrder: true,
					},
				},
			},
		},
	}
}

func TestFieldPathsTestSuite(t *testing.T) {
	suite.Run(t, new(FieldPathsTestSuite))
}
