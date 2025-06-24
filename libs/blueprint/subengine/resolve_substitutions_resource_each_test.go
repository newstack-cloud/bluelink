package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/stretchr/testify/suite"
)

type SubstitutionResourceEachResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInResourceEachFixtureName      = "resolve-resource-each"
	resolveInResourceEachFail1FixtureName = "resolve-resource-each-fail-1"
	resolveInResourceEachFail2FixtureName = "resolve-resource-each-fail-2"
	resolveInResourceEachFail3FixtureName = "resolve-resource-each-fail-3"
)

func (s *SubstitutionResourceEachResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(map[string]string{
		resolveInResourceEachFixtureName:      "__testdata/sub-resolver/resolve-resource-each-blueprint.yml",
		resolveInResourceEachFail1FixtureName: "__testdata/sub-resolver/resolve-resource-each-fail-1-blueprint.yml",
		resolveInResourceEachFail2FixtureName: "__testdata/sub-resolver/resolve-resource-each-fail-2-blueprint.yml",
		resolveInResourceEachFail3FixtureName: "__testdata/sub-resolver/resolve-resource-each-fail-3-blueprint.yml",
	}, &s.Suite)
}

func (s *SubstitutionResourceEachResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionResourceEachResolverTestSuite) Test_resolves_substitutions_in_resource_each_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourceEachFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveResourceEachTestParams()
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

	result, err := subResolver.ResolveResourceEach(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		ResolveForChangeStaging,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	region1 := "us-west-2"
	region2 := "us-east-1"
	region3 := "eu-west-2"
	s.Assert().Equal(
		[]*core.MappingNode{
			{
				Scalar: &core.ScalarValue{
					StringValue: &region1,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &region2,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &region3,
				},
			},
		},
		result,
	)
}

func (s *SubstitutionResourceEachResolverTestSuite) Test_fails_when_resource_each_depends_on_resource() {
	blueprint := s.specFixtureSchemas[resolveInResourceEachFail1FixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveResourceEachTestParams()
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

	result, err := subResolver.ResolveResourceEach(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		ResolveForChangeStaging,
	)
	s.Assert().Error(err)
	s.Assert().Nil(result)
	runErr, isRunErr := err.(*errors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(ErrorReasonCodeDisallowedElementType, runErr.ReasonCode)
	s.Assert().Equal(
		"run error: [resources.ordersTable]: element type \"resource\" can not be a dependency for the \"each\" property, "+
			"a dependency can be either a direct or indirect reference to an element in a blueprint, "+
			"be sure to check the full trail of references",
		runErr.Error(),
	)
}

func (s *SubstitutionResourceEachResolverTestSuite) Test_fails_when_resource_each_depends_on_child_blueprint() {
	blueprint := s.specFixtureSchemas[resolveInResourceEachFail2FixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveResourceEachTestParams()
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

	result, err := subResolver.ResolveResourceEach(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		ResolveForChangeStaging,
	)
	s.Assert().Error(err)
	s.Assert().Nil(result)
	runErr, isRunErr := err.(*errors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(ErrorReasonCodeDisallowedElementType, runErr.ReasonCode)
	s.Assert().Equal(
		"run error: [resources.ordersTable]: element type \"child\" can not be a dependency for the \"each\" property, "+
			"a dependency can be either a direct or indirect reference to an element in a blueprint, "+
			"be sure to check the full trail of references",
		runErr.Error(),
	)
}

func (s *SubstitutionResourceEachResolverTestSuite) Test_fails_when_resource_each_resolves_to_a_value_that_is_not_a_list() {
	blueprint := s.specFixtureSchemas[resolveInResourceEachFail3FixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveResourceEachTestParams()
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

	result, err := subResolver.ResolveResourceEach(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		ResolveForChangeStaging,
	)
	s.Assert().Error(err)
	s.Assert().Nil(result)
	runErr, isRunErr := err.(*errors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(ErrorReasonCodeResourceEachInvalidType, runErr.ReasonCode)
	s.Assert().Equal(
		"run error: [resources.ordersTable]: `each` property in resource template"+
			" \"ordersTable\" must yield an array, string found",
		runErr.Error(),
	)
}

func resolveResourceEachTestParams() core.BlueprintParams {
	environment := "production-env"
	enableOrderTableTrigger := true
	region := "us-west-2"
	deployOrdersTableToRegions := "[\"us-west-2\",\"us-east-1\",\"eu-west-2\"]"
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

func TestSubstitutionResourceEachResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionResourceEachResolverTestSuite))
}
