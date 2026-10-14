package sensitivity

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/stretchr/testify/suite"
)

const (
	testResourceType   = "aws/ssm/parameterTree"
	testDataSourceType = "aws/ssm/parameter"
)

type SensitivityIndexSuite struct {
	suite.Suite
}

// A spec shaped like the cases that matter, secrets declared on containers rather than on
// the leaves holding the values, which is how a plugin author declares them.
func (s *SensitivityIndexSuite) specSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"treeName": {Type: provider.ResourceDefinitionsSchemaTypeString},
			"password": {
				Type:      provider.ResourceDefinitionsSchemaTypeString,
				Sensitive: true,
			},
			// A map of secrets, marked on the map itself.
			"secureValues": {
				Type:      provider.ResourceDefinitionsSchemaTypeMap,
				Sensitive: true,
				MapValues: &provider.ResourceDefinitionsSchema{
					Type: provider.ResourceDefinitionsSchemaTypeString,
				},
			},
			// A map marked on its values rather than on the map, so resolving a value
			// means descending into the map's value schema rather than inheriting.
			"taggedValues": {
				Type: provider.ResourceDefinitionsSchemaTypeMap,
				MapValues: &provider.ResourceDefinitionsSchema{
					Type:      provider.ResourceDefinitionsSchemaTypeString,
					Sensitive: true,
				},
			},
			// An object of secrets, marked on the object.
			"credentials": {
				Type:      provider.ResourceDefinitionsSchemaTypeObject,
				Sensitive: true,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"username": {Type: provider.ResourceDefinitionsSchemaTypeString},
					"computedToken": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
				},
			},
			// An array whose items hold a secret, marked on the leaf.
			"parameters": {
				Type: provider.ResourceDefinitionsSchemaTypeArray,
				Items: &provider.ResourceDefinitionsSchema{
					Type: provider.ResourceDefinitionsSchemaTypeObject,
					Attributes: map[string]*provider.ResourceDefinitionsSchema{
						"name": {Type: provider.ResourceDefinitionsSchemaTypeString},
						"value": {
							Type:      provider.ResourceDefinitionsSchemaTypeString,
							Sensitive: true,
						},
					},
				},
			},
			// An array marked at the array level, so every item is a secret.
			"secureParameters": {
				Type:      provider.ResourceDefinitionsSchemaTypeArray,
				Sensitive: true,
				Items: &provider.ResourceDefinitionsSchema{
					Type: provider.ResourceDefinitionsSchemaTypeObject,
					Attributes: map[string]*provider.ResourceDefinitionsSchema{
						"value": {Type: provider.ResourceDefinitionsSchemaTypeString},
					},
				},
			},
			// A union of which one variant is a secret.
			"connection": {
				Type: provider.ResourceDefinitionsSchemaTypeUnion,
				OneOf: []*provider.ResourceDefinitionsSchema{
					{
						Type: provider.ResourceDefinitionsSchemaTypeObject,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"host": {Type: provider.ResourceDefinitionsSchemaTypeString},
						},
					},
					{
						Type: provider.ResourceDefinitionsSchemaTypeObject,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"connectionString": {
								Type:      provider.ResourceDefinitionsSchemaTypeString,
								Sensitive: true,
							},
						},
					},
				},
			},
			"specVersion": {Type: provider.ResourceDefinitionsSchemaTypeString},
			// Named so that trimming the root as a string prefix, rather than matching it
			// as a segment, would fail to resolve it and so report it as not sensitive.
			"specSecret": {
				Type:      provider.ResourceDefinitionsSchemaTypeString,
				Sensitive: true,
			},
		},
	}
}

func (s *SensitivityIndexSuite) index(registry *fakeResourceRegistry) *Index {
	return NewIndex(registry, &fakeDataSourceRegistry{
		fields: map[string]*provider.DataSourceSpecSchema{
			"arn": {Type: provider.DataSourceSpecTypeString},
			"value": {
				Type:      provider.DataSourceSpecTypeString,
				Sensitive: true,
			},
		},
	})
}

func (s *SensitivityIndexSuite) isResFieldSensitive(fieldPath string) bool {
	sensitive, err := s.index(&fakeResourceRegistry{schema: s.specSchema()}).
		ResourceField(context.Background(), testResourceType, fieldPath)
	s.Require().NoError(err)

	return sensitive
}

func (s *SensitivityIndexSuite) Test_a_field_declared_sensitive_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.password"))
}

func (s *SensitivityIndexSuite) Test_a_field_declared_nothing_is_not_sensitive() {
	s.Assert().False(s.isResFieldSensitive("spec.treeName"))
}

// The case sensitivity inheritance exists for: a plugin author marks the container they
// think of as secret, and the values are in its entries.
func (s *SensitivityIndexSuite) Test_a_value_in_a_sensitive_map_is_sensitive() {
	for _, fieldPath := range []string{
		"spec.secureValues.dbPassword",
		`spec.secureValues["db/password"]`,
		"spec.secureValues",
	} {
		s.Assert().True(s.isResFieldSensitive(fieldPath), "%s came out unmarked", fieldPath)
	}
}

// Inheritance is not the only route: a map whose values are declared sensitive has to be
// descended into, in both the dotted and the bracketed form a key can be addressed with.
func (s *SensitivityIndexSuite) Test_a_map_whose_values_are_sensitive_is_descended_into() {
	for _, fieldPath := range []string{
		"spec.taggedValues.dbPassword",
		`spec.taggedValues["db/password"]`,
	} {
		s.Assert().True(s.isResFieldSensitive(fieldPath), "%s came out unmarked", fieldPath)
	}

	s.Assert().False(s.isResFieldSensitive("spec.taggedValues"))
}

func (s *SensitivityIndexSuite) Test_an_attribute_of_a_sensitive_object_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.credentials.username"))
}

// A value a provider computed from a secret is still a secret, so computed fields are not
// exempt from inheritance.
func (s *SensitivityIndexSuite) Test_a_computed_attribute_of_a_sensitive_object_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.credentials.computedToken"))
}

func (s *SensitivityIndexSuite) Test_a_sensitive_field_of_an_array_item_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.parameters[0].value"))
	s.Assert().False(s.isResFieldSensitive("spec.parameters[0].name"))
}

func (s *SensitivityIndexSuite) Test_everything_under_a_sensitive_array_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.secureParameters[2].value"))
}

// Change staging treats a union as sensitive when any variant is, since which variant a
// value took is not known from the path alone.
func (s *SensitivityIndexSuite) Test_a_union_with_a_sensitive_variant_is_sensitive() {
	s.Assert().True(s.isResFieldSensitive("spec.connection.connectionString"))
	s.Assert().False(s.isResFieldSensitive("spec.connection.host"))
}

func (s *SensitivityIndexSuite) Test_the_spec_prefix_is_optional() {
	s.Assert().True(s.isResFieldSensitive("secureValues.dbPassword"))
	s.Assert().False(s.isResFieldSensitive("treeName"))
}

// A field whose name starts with the root's name is not the root. The sensitive one is what
// makes this discriminating: a path that fails to resolve reports "not sensitive", so a
// non-sensitive field would answer the same either way.
func (s *SensitivityIndexSuite) Test_a_field_named_like_the_root_is_not_the_root() {
	s.Assert().True(s.isResFieldSensitive("specSecret"))
	s.Assert().True(s.isResFieldSensitive("spec.specSecret"))
	s.Assert().False(s.isResFieldSensitive("specVersion"))
	s.Assert().False(s.isResFieldSensitive("spec.specVersion"))
}

// A path that leaves the schema keeps what it inherited. Resolution fails for ordinary
// reasons, and treating that as "not sensitive" would unmark the values inside a container
// someone took the trouble to mark.
func (s *SensitivityIndexSuite) Test_an_unresolvable_path_keeps_the_sensitivity_it_inherited() {
	s.Assert().True(s.isResFieldSensitive("spec.credentials.added.later"))
	s.Assert().False(s.isResFieldSensitive("spec.treeName.added.later"))
	s.Assert().False(s.isResFieldSensitive("spec.absent"))
}

func (s *SensitivityIndexSuite) Test_a_resource_type_with_no_schema_is_not_sensitive() {
	index := s.index(&fakeResourceRegistry{})

	sensitive, err := index.ResourceField(context.Background(), testResourceType, "spec.password")
	s.Require().NoError(err)
	s.Assert().False(sensitive)
}

// The registry call can cross a process boundary to a plugin, and a schema does not change
// while that plugin is running.
func (s *SensitivityIndexSuite) Test_a_schema_is_fetched_once_per_resource_type() {
	registry := &fakeResourceRegistry{schema: s.specSchema()}
	index := s.index(registry)

	for range 3 {
		_, err := index.ResourceField(context.Background(), testResourceType, "spec.password")
		s.Require().NoError(err)
	}

	s.Assert().Equal(1, registry.specDefinitionCalls)
}

func (s *SensitivityIndexSuite) Test_reports_a_registry_failure() {
	registry := &fakeResourceRegistry{err: errors.New("plugin is not responding")}

	_, err := s.index(registry).ResourceField(context.Background(), testResourceType, "spec.password")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "plugin is not responding")
}

func (s *SensitivityIndexSuite) Test_a_sensitive_data_source_field_is_sensitive() {
	index := s.index(&fakeResourceRegistry{schema: s.specSchema()})

	sensitive, err := index.DataSourceField(context.Background(), testDataSourceType, "value")
	s.Require().NoError(err)
	s.Assert().True(sensitive)

	sensitive, err = index.DataSourceField(context.Background(), testDataSourceType, "arn")
	s.Require().NoError(err)
	s.Assert().False(sensitive)
}

func (s *SensitivityIndexSuite) Test_an_unknown_data_source_field_is_not_sensitive() {
	index := s.index(&fakeResourceRegistry{schema: s.specSchema()})

	sensitive, err := index.DataSourceField(context.Background(), testDataSourceType, "absent")
	s.Require().NoError(err)
	s.Assert().False(sensitive)
}

func TestSensitivityIndexSuite(t *testing.T) {
	suite.Run(t, new(SensitivityIndexSuite))
}

// Stands in for the resource registry, which in a running engine reaches a plugin over gRPC.
type fakeResourceRegistry struct {
	resourcehelpers.Registry
	schema              *provider.ResourceDefinitionsSchema
	specDefinitionCalls int
	err                 error
}

func (r *fakeResourceRegistry) GetSpecDefinition(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	r.specDefinitionCalls++
	if r.err != nil {
		return nil, r.err
	}

	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{Schema: r.schema},
	}, nil
}

type fakeDataSourceRegistry struct {
	provider.DataSourceRegistry
	fields map[string]*provider.DataSourceSpecSchema
}

func (r *fakeDataSourceRegistry) GetSpecDefinition(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{Fields: r.fields},
	}, nil
}
