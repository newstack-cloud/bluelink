package specmerge

import (
	"sort"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type CollectComputedFieldsTestSuite struct {
	suite.Suite
}

func (s *CollectComputedFieldsTestSuite) Test_collects_computed_fields_across_nested_schema() {
	schema := &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			// Not computed - user supplied.
			"handler": {Type: provider.ResourceDefinitionsSchemaTypeString},
			// Computed scalar.
			"id": {Type: provider.ResourceDefinitionsSchemaTypeString, Computed: true},
			// Computed field nested under an object.
			"config": {
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"arn": {Type: provider.ResourceDefinitionsSchemaTypeString, Computed: true},
				},
			},
			// Computed field nested under an array - "[0]" placeholder.
			"endpoints": {
				Type: provider.ResourceDefinitionsSchemaTypeArray,
				Items: &provider.ResourceDefinitionsSchema{
					Type: provider.ResourceDefinitionsSchemaTypeObject,
					Attributes: map[string]*provider.ResourceDefinitionsSchema{
						"url": {Type: provider.ResourceDefinitionsSchemaTypeString, Computed: true},
					},
				},
			},
			// Computed field nested under a map - "[\"<key>\"]" placeholder.
			"identifiers": {
				Type: provider.ResourceDefinitionsSchemaTypeMap,
				MapValues: &provider.ResourceDefinitionsSchema{
					Type:     provider.ResourceDefinitionsSchemaTypeString,
					Computed: true,
				},
			},
			// Computed field inside a union branch.
			"status": {
				Type: provider.ResourceDefinitionsSchemaTypeUnion,
				OneOf: []*provider.ResourceDefinitionsSchema{
					{Type: provider.ResourceDefinitionsSchemaTypeString, Computed: true},
				},
			},
		},
	}

	fields := CollectComputedFields(schema, "spec")
	sort.Strings(fields)

	s.Assert().Equal(
		[]string{
			"spec.config.arn",
			"spec.endpoints[0].url",
			"spec.id",
			"spec.identifiers[\"<key>\"]",
			"spec.status",
		},
		fields,
	)
}

func (s *CollectComputedFieldsTestSuite) Test_returns_empty_for_nil_schema() {
	s.Assert().Empty(CollectComputedFields(nil, "spec"))
}

func (s *CollectComputedFieldsTestSuite) Test_returns_empty_when_no_computed_fields() {
	schema := &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"handler": {Type: provider.ResourceDefinitionsSchemaTypeString},
		},
	}
	s.Assert().Empty(CollectComputedFields(schema, "spec"))
}

func TestCollectComputedFieldsTestSuite(t *testing.T) {
	suite.Run(t, new(CollectComputedFieldsTestSuite))
}
