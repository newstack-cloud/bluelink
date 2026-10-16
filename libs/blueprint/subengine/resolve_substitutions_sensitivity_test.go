package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
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

func (s *SubstitutionSensitivityTestSuite) blueprint() *schema.Blueprint {
	return s.specFixtureSchemas[resolveInResourceSensitivityFixtureName]
}

func (s *SubstitutionSensitivityTestSuite) newResolver() SubstitutionResolver {
	spec := internal.NewBlueprintSpecMock(s.blueprint())

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

	return NewDefaultSubstitutionResolver(
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
}

// Resolved and cached the way the container does it, in dependency order, so a reference to
// it reads a resource whose recorded paths came from resolution rather than from the fixture.
func (s *SubstitutionSensitivityTestSuite) cacheResolvedResource(resourceName string) {
	result, err := s.newResolver().ResolveInResource(
		context.TODO(),
		resourceName,
		s.blueprint().Resources.Values[resourceName],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result.ResolvedResource)

	s.resourceCache.Set(resourceName, result.ResolvedResource)
}

func (s *SubstitutionSensitivityTestSuite) resolveOrdersTable() *ResolveInResourceResult {
	s.cacheResolvedResource("interpolatedTable")

	result, err := s.newResolver().ResolveInResource(
		context.TODO(),
		"ordersTable",
		s.blueprint().Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.ResolvedResource)

	return result
}

func (s *SubstitutionSensitivityTestSuite) resolveCoreInfraInclude() *ResolveInIncludeResult {
	result, err := s.newResolver().ResolveInInclude(
		context.TODO(),
		"coreInfra",
		s.blueprint().Include.Values["coreInfra"],
		&ResolveIncludeTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	return result
}

// The exported value is resolved apart from the export's own fields, through the same entry
// point the container uses, so that is where the mark for the value has to appear.
func (s *SubstitutionSensitivityTestSuite) resolveExportValue(exportName string) *ResolveResult {
	export := s.blueprint().Exports.Values[exportName]
	s.Require().NotNil(export)

	fieldAsSub, err := substitutions.ParseSubstitution(
		"exports",
		core.StringValueFromScalar(export.Field),
		/* parentSourceStart */ &source.Meta{Position: source.Position{}},
		/* outputLineInfo */ false,
		/* ignoreParentColumn */ true,
	)
	s.Require().NoError(err)

	result, err := s.newResolver().ResolveSubstitution(
		context.TODO(),
		&substitutions.StringOrSubstitution{SubstitutionValue: fieldAsSub},
		core.ExportElementID(exportName),
		"field",
		&ResolveTargetInfo{ResolveFor: ResolveForChangeStaging},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

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

// A secret passed down to a child blueprint leaves this blueprint's control, so the
// include's own variable is the last place it can be marked.
func (s *SubstitutionSensitivityTestSuite) Test_a_secret_passed_to_a_child_blueprint_marks_the_variable() {
	result := s.resolveCoreInfraInclude()

	s.Assert().Contains(result.SensitivePaths, "children.coreInfra.variables.dbPassword")
	s.Assert().Contains(
		result.SensitivePaths,
		"children.coreInfra.variables.parameterPassword",
		"a variable reading a field another resource declares sensitive was not marked",
	)
	s.Assert().NotContains(
		result.SensitivePaths,
		"children.coreInfra.variables.region",
		"a variable holding a literal was marked",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_a_secret_in_include_metadata_is_marked() {
	result := s.resolveCoreInfraInclude()

	s.Assert().Contains(result.SensitivePaths, "children.coreInfra.metadata.auditKey")
	s.Assert().NotContains(result.SensitivePaths, "children.coreInfra.metadata.sourceType")
}

// Resolution records against the element it is resolving for, so an export's mark names the
// export rather than the variable the value was read from. The path matters as the mark that
// reaches state is matched by path, not by the result holding any mark at all.
func (s *SubstitutionSensitivityTestSuite) Test_exporting_a_secret_variable_marks_the_exported_value() {
	s.Assert().Contains(
		s.resolveExportValue("dbPassword").SensitivePaths,
		"exports.dbPassword.field",
	)
	s.Assert().Empty(
		s.resolveExportValue("environment").SensitivePaths,
		"an export of a variable declaring nothing was marked",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_exporting_a_sensitive_resource_field_marks_the_exported_value() {
	s.Assert().Contains(
		s.resolveExportValue("parameterPassword").SensitivePaths,
		"exports.parameterPassword.field",
	)
}

// The record is shared with everything resolved on the way, so an export reading a plain
// field of a value whose sibling field holds a secret finds that secret's path in its own
// result. Marking from any path at all would hide a value that is not secret.
func (s *SubstitutionSensitivityTestSuite) Test_an_export_is_not_marked_by_a_secret_in_a_sibling_field() {
	paths := s.resolveExportValue("connectionHost").SensitivePaths

	s.Require().NotEmpty(
		paths,
		"the fixture records nothing on the way, so it cannot show the hazard",
	)
	s.Assert().NotContains(paths, "exports.connectionHost.field")
}

func (s *SubstitutionSensitivityTestSuite) Test_referencing_an_interpolated_secret_in_another_resource_marks_the_field() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.interpolatedTableName")
	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.interpolatedTableRegion",
		"a sibling field of the one holding the secret was marked",
	)
}

// Coverage is a path test, not a string prefix test. "spec.tableName" holding a secret says
// nothing about "spec.tableNameSuffix", and a bare strings.HasPrefix would mark it.
func (s *SubstitutionSensitivityTestSuite) Test_a_mark_does_not_cover_a_field_whose_name_extends_it() {
	result := s.resolveOrdersTable()

	s.Assert().NotContains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.interpolatedTableNameSuffix",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_an_array_reference_matches_on_the_resolved_index() {
	result := s.resolveOrdersTable()

	// References for `spec.replicaSecret` and `spec.replicaPlain` are array items, where only one of them
	// holds a sensitive value.
	// See __testdata/sub-resolver/resolve-in-resource-sensitivity-blueprint.yml
	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.replicaSecret")
	s.Assert().NotContains(result.ResolvedResource.SensitiveSpecPaths, "spec.replicaPlain")
}

// Values are memoised, so only the first read resolves one. A value that declares nothing
// but interpolates a secret has to carry what its resolution recorded, or every read after
// the first is served from the cache and marked nothing.
func (s *SubstitutionSensitivityTestSuite) Test_a_cached_value_still_marks_a_field_reading_its_recorded_secret() {
	result := s.resolveOrdersTable()

	s.Assert().Contains(result.ResolvedResource.SensitiveSpecPaths, "spec.connectionPassword")
	s.Assert().Contains(
		result.ResolvedResource.SensitiveSpecPaths,
		"spec.connectionPasswordAgain",
		"the second read was served from the value cache and lost its marking",
	)
}

func (s *SubstitutionSensitivityTestSuite) Test_reading_a_plain_field_of_a_value_holding_an_adjacent_secret_is_not_marked() {
	result := s.resolveOrdersTable()

	s.Assert().NotContains(result.ResolvedResource.SensitiveSpecPaths, "spec.connectionHost")
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
