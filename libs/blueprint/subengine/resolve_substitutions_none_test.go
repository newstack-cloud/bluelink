package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type SubstitutionNoneValueResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	noneValueHandlingFixtureName = "none-value-handling"
)

func (s *SubstitutionNoneValueResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(
		map[string]string{
			noneValueHandlingFixtureName: "__testdata/sub-resolver/none-value-handling-blueprint.yml",
		},
		&s.Suite,
	)
}

func (s *SubstitutionNoneValueResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

// Test_resolves_none_values_in_metadata_for_change_staging verifies that none values
// are properly handled in metadata resolution - fields omitted, arrays filtered, etc.
func (s *SubstitutionNoneValueResolverTestSuite) Test_resolves_none_values_in_metadata_for_change_staging() {
	blueprint := s.specFixtureSchemas[noneValueHandlingFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := noneValueHandlingTestParams()
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
		params,
	)

	result, err := subResolver.ResolveInMappingNode(
		context.TODO(),
		"metadata",
		blueprint.Metadata,
		&ResolveMappingNodeTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Use snapshot testing to verify the complete resolved structure
	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

// Test_resolves_none_values_in_resource_for_change_staging verifies that none values
// in resource specs and metadata are properly handled
func (s *SubstitutionNoneValueResolverTestSuite) Test_resolves_none_values_in_resource_for_change_staging() {
	blueprint := s.specFixtureSchemas[noneValueHandlingFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := noneValueHandlingTestParams()
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
		params,
	)

	result, err := subResolver.ResolveInResource(
		context.TODO(),
		"testTable",
		blueprint.Resources.Values["testTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Use snapshot testing to verify the complete resolved structure
	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

// Test_resolves_none_values_in_exports_for_change_staging verifies that exports
// resolving to none are properly handled
func (s *SubstitutionNoneValueResolverTestSuite) Test_resolves_none_values_in_exports_for_change_staging() {
	blueprint := s.specFixtureSchemas[noneValueHandlingFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := noneValueHandlingTestParams()
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
		params,
	)

	// Resolve all exports
	results := make(map[string]interface{})
	for exportName, exportDef := range blueprint.Exports.Values {
		result, err := subResolver.ResolveInExport(
			context.TODO(),
			exportName,
			exportDef,
			&ResolveExportTargetInfo{
				ResolveFor: ResolveForChangeStaging,
			},
		)
		s.Require().NoError(err)
		results[exportName] = result
	}

	// Use snapshot testing to verify the complete resolved exports
	err := testhelpers.Snapshot(results)
	s.Require().NoError(err)
}

func noneValueHandlingTestParams() core.BlueprintParams {
	environment := "test-env"
	blueprintVars := map[string]*core.ScalarValue{
		"environment": {
			StringValue: &environment,
		},
	}
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func TestSubstitutionNoneValueResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionNoneValueResolverTestSuite))
}
