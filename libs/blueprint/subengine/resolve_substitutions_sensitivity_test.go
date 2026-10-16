package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

// Every fixture here interpolates a secret into a field that declares nothing. A fixture
// whose target field was declared sensitive would pass whether or not sensitivity propagates
// at all, which is the trap worth designing the fixtures around.
type SubstitutionSensitivityTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

func (s *SubstitutionSensitivityTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(map[string]string{
		resolveInResourceSensitivityFixtureName: "__testdata/sub-resolver/resolve-in-resource-sensitivity-blueprint.yml",
	}, &s.Suite)
}

func (s *SubstitutionSensitivityTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionSensitivityTestSuite) resolveOrdersTable() *ResolveInResourceResult {
	blueprint := s.specFixtureSchemas[resolveInResourceSensitivityFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)

	// The referenced resource has to be resolved before anything referring to it, which the
	// container guarantees by dependency order; here it is primed directly.
	s.resourceCache.Set("parameterTree", &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{Value: "example/sensitiveContainer"},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"name": core.MappingNodeFromString("orders-parameters"),
				"secureValues": {
					Fields: map[string]*core.MappingNode{
						"dbPassword": core.MappingNodeFromString("super-secret-password"),
					},
				},
				"credentials": {
					Fields: map[string]*core.MappingNode{
						"username": core.MappingNodeFromString("orders-service"),
						"nested": {
							Fields: map[string]*core.MappingNode{
								"token": core.MappingNodeFromString("nested-token"),
							},
						},
					},
				},
			},
		},
	})

	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		sensitivityTestParams(),
	)

	result, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.ResolvedResource)

	return result
}

func (s *SubstitutionSensitivityTestSuite) Test_a_secret_variable_marks_the_field_it_is_interpolated_into() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.connectionString",
		"a secret interpolated into an undeclared field was not recorded",
	)
	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.tableName",
		"a field assembled from a plain variable was recorded as sensitive",
	)
}

// Sensitivity is recorded at the path a secret reached, so an array item holding one marks
// that item rather than the array, which would mark its siblings along with it.
func (s *SubstitutionSensitivityTestSuite) Test_a_secret_in_an_array_item_marks_that_item_alone() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.replicaRegions[1]")
	s.Assert().NotContains(result.ResolvedResource.SensitiveSpecPaths, "spec.replicaRegions")
	s.Assert().NotContains(result.ResolvedResource.SensitiveSpecPaths, "spec.replicaRegions[0]")
}

func (s *SubstitutionSensitivityTestSuite) Test_marking_a_field_leaves_its_value_alone() {
	result := s.resolveOrdersTable()

	connectionString := result.ResolvedResource.Spec.Fields["connectionString"]
	s.Require().NotNil(connectionString)
	s.Assert().Equal(
		"postgres://user:super-secret-password@host/orders",
		core.StringValue(connectionString),
	)
}

// Items resolving to none are dropped from the resolved array, so a path built from the
// source index would address a different item than the one holding the secret. Change
// staging walks the resolved array, so the index has to be the resolved one.
func (s *SubstitutionSensitivityTestSuite) Test_an_index_counts_the_items_that_survived_resolution() {
	result := s.resolveOrdersTable()

	backupRegions := result.ResolvedResource.Spec.Fields["backupRegions"]
	s.Require().NotNil(backupRegions)
	s.Require().Len(backupRegions.Items, 1, "the none item should have been dropped")

	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.backupRegions[0]",
		"the mark points at an index the resolved array does not hold",
	)
	s.Assert().NotContains(result.ResolvedResource.SensitiveSpecPaths, "spec.backupRegions[1]")
}

func (s *SubstitutionSensitivityTestSuite) Test_referencing_a_sensitive_field_marks_the_referencing_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.referencedPassword")
	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.referencedUsername")
	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.referencedNestedToken",
		"sensitivity was kept only from the nearest ancestor rather than accumulated",
	)
	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.referencedName",
		"a reference to a field declaring nothing was marked",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_referencing_a_sensitive_data_source_export_marks_the_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.vpcPrivateKey")
	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.vpcId",
		"an export the plugin declares nothing about was marked",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_an_aliased_export_of_a_sensitive_field_marks_the_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.aliasedKey")
}

func (s *SubstitutionSensitivityTestSuite) Test_an_export_named_after_a_sensitive_field_marks_the_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.secretArn",
		"the declaration was read through the alias rather than the field a read serves",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_a_deferred_data_source_still_marks_the_field_reading_it() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.deferredKey",
		"a data source deferred to deployment left its reader unmarked at change staging",
	)
}

// A value declared secret is a declared source in the same way a secret variable is, and
// resolution memoises values, so the second field to reference one must be marked from the
// cached value rather than missed.
func (s *SubstitutionSensitivityTestSuite) Test_a_secret_value_marks_every_field_referencing_it() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.primaryToken")
	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.secondaryToken",
		"the second reference was served from the value cache and lost its marking",
	)
}

// A secret reaching the resource outside its spec is reported on the result, since only spec
// paths travel on the resolved resource. Dropping it from both was the first version's bug.
func (s *SubstitutionSensitivityTestSuite) Test_a_secret_outside_the_spec_is_reported_on_the_result() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(
		result.SensitivePaths,
		"resources.ordersTable.metadata.custom.auditKey",
	)
	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"metadata.custom.auditKey",
	)
}

// Spec paths travel on the resolved resource alone, so the result must not repeat them.
func (s *SubstitutionSensitivityTestSuite) Test_the_result_does_not_repeat_the_resources_spec_paths() {
	result := s.resolveOrdersTable()

	for _, path := range result.SensitivePaths {
		s.Assert().NotContains(path, "ordersTable.spec.", "%s is reported twice", path)
	}
}

// Arguments resolve through the same dispatch under the same context, so a secret passed to
// a function marks the field the call lands in without the function needing to be understood.
// Asserted rather than assumed: it is the property that makes an opaque function safe.
func (s *SubstitutionSensitivityTestSuite) Test_a_secret_passed_through_a_function_still_marks_its_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.trimmedToken")
}

func sensitivityTestParams() core.BlueprintParams {
	environment := "production"
	dbPassword := "super-secret-password"

	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			"environment": {StringValue: &environment},
			"dbPassword":  {StringValue: &dbPassword},
		},
	)
}

func TestSubstitutionSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionSensitivityTestSuite))
}
