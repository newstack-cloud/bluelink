package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type SubstitutionExportResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInExportFixtureName = "resolve-in-export"
)

func (s *SubstitutionExportResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(
		map[string]string{
			resolveInExportFixtureName: "__testdata/sub-resolver/resolve-in-export-blueprint.yml",
		},
		&s.Suite,
	)
}

func (s *SubstitutionExportResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionExportResolverTestSuite) Test_resolves_substitutions_in_export_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInExportFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInExportTestParams()
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

	result, err := subResolver.ResolveInExport(
		context.TODO(),
		"environment",
		blueprint.Exports.Values["environment"],
		&ResolveExportTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func resolveInExportTestParams() core.BlueprintParams {
	environment := "production-env"
	enableOrderTableTrigger := true
	region := "us-west-2"
	deployOrdersTableToRegions := "[\"us-west-2\",\"us-east-1\"]"
	blueprintVars := map[string]*core.ScalarValue{
		"environment": {
			StringValue: &environment,
		},
		"region": {
			StringValue: &region,
		},
		"deployOrdersTableToRegions": {
			StringValue: &deployOrdersTableToRegions,
		},
		"enableOrderTableTrigger": {
			BoolValue: &enableOrderTableTrigger,
		},
	}
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func TestSubstitutionExportResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionExportResolverTestSuite))
}
