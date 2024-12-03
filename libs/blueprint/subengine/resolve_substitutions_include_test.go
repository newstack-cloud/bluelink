package subengine

import (
	"context"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/suite"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/internal"
)

type SubstitutionIncludeResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInIncludeFixtureName = "resolve-in-include"
)

func (s *SubstitutionIncludeResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(
		map[string]string{
			resolveInIncludeFixtureName: "__testdata/sub-resolver/resolve-in-include-blueprint.yml",
		},
		&s.Suite,
	)
}

func (s *SubstitutionIncludeResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionIncludeResolverTestSuite) Test_resolves_substitutions_in_include_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInIncludeFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInIncludeTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		spec,
		params,
	)

	result, err := subResolver.ResolveInInclude(
		context.TODO(),
		"coreInfra",
		blueprint.Include.Values["coreInfra"],
		&ResolveIncludeTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = cupaloy.Snapshot(result)
	s.Require().NoError(err)
}

func resolveInIncludeTestParams() *internal.Params {
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
	return internal.NewParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func TestSubstitutionIncludeResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionIncludeResolverTestSuite))
}
