package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type SubstitutionResourceResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInResourceFixtureName                   = "resolve-in-resource"
	resolveInResourcePartialAnnotationsFixtureName = "resolve-in-resource-partial-annotations"
	resolveInResourceTemplateFixtureName           = "resolve-in-resource-2"
	testInstanceID                                 = "cb826a32-1052-4fde-aa6e-d449b9f50066"
	testChildInstanceID                            = "05253564-cd77-4b92-81bc-e75f9478ec4d"
)

func (s *SubstitutionResourceResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(map[string]string{
		resolveInResourceFixtureName:                   "__testdata/sub-resolver/resolve-in-resource-blueprint.yml",
		resolveInResourcePartialAnnotationsFixtureName: "__testdata/sub-resolver/resolve-in-resource-partial-annotations-blueprint.yml",
		resolveInResourceTemplateFixtureName:           "__testdata/sub-resolver/resolve-in-resource-2-blueprint.yml",
	}, &s.Suite)
}

func (s *SubstitutionResourceResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionResourceResolverTestSuite) Test_resolves_substitutions_in_resource_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourceFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
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

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionResourceResolverTestSuite) Test_resolves_substitutions_in_resource_for_change_staging_with_partial_annotations() {
	blueprint := s.specFixtureSchemas[resolveInResourcePartialAnnotationsFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
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

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionResourceResolverTestSuite) Test_resolves_substitutions_in_resource_for_deployment() {
	blueprint := s.specFixtureSchemas[resolveInResourceFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
	)
	// coreInfra.region is used in the resource and should be resolved using the child blueprint
	// state.
	err := s.stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID: testInstanceID,
	})
	s.Require().NoError(err)

	childBlueprintRegion := "eu-west-1"
	err = s.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID: testChildInstanceID,
			Exports: map[string]*state.ExportState{
				"region": {
					Value: &core.MappingNode{
						Scalar: &core.ScalarValue{
							StringValue: &childBlueprintRegion,
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	err = s.stateContainer.Children().Attach(
		context.Background(),
		testInstanceID,
		testChildInstanceID,
		"coreInfra",
	)
	s.Require().NoError(err)

	// Make sure the current instance ID can be retrieved from the context when fetching
	// state from the state container to resolve the child blueprint export reference.
	ctx := context.WithValue(
		context.Background(),
		core.BlueprintInstanceIDKey,
		testInstanceID,
	)
	result, err := subResolver.ResolveInResource(
		ctx,
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor:        ResolveForDeployment,
			PartiallyResolved: partiallyResolvedResource(),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionResourceResolverTestSuite) Test_resolves_substitutions_in_resource_with_template_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourceTemplateFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
	)

	ordersTable1Name := "production-Orders-1"
	s.resourceTemplateInputElemCache.Set("ordersTable", []*core.MappingNode{
		{
			Fields: map[string]*core.MappingNode{
				"name": {
					Scalar: &core.ScalarValue{
						StringValue: &ordersTable1Name,
					},
				},
			},
		},
	})
	result, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable_0",
		convertToTemplateResourceInstance(
			blueprint.Resources.Values["ordersTable"],
		),
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func partiallyResolvedResource() *provider.ResolvedResource {
	description := "Table that stores orders for an application."
	displayName := "production-env Orders Table"
	trigger := true
	x := 100
	y := 200
	condition1 := true
	condition2 := true
	condition3 := false
	tableName := "production-Orders"
	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "aws/dynamodb/table",
		},
		Description: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &description,
			},
		},
		Metadata: &provider.ResolvedResourceMetadata{
			DisplayName: &core.MappingNode{
				Scalar: &core.ScalarValue{
					StringValue: &displayName,
				},
			},
			Annotations: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"aws.dynamodb.trigger": {
						Scalar: &core.ScalarValue{
							BoolValue: &trigger,
						},
					},
				},
			},
			Labels: &schema.StringMap{
				Values: map[string]string{
					"app": "orders",
				},
			},
			Custom: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"label": {
						Scalar: &core.ScalarValue{
							StringValue: &displayName,
						},
					},
					"x": {
						Scalar: &core.ScalarValue{
							IntValue: &x,
						},
					},
					"y": {
						Scalar: &core.ScalarValue{
							IntValue: &y,
						},
					},
				},
			},
		},
		Condition: &provider.ResolvedResourceCondition{
			And: []*provider.ResolvedResourceCondition{
				{
					StringValue: &core.MappingNode{
						Scalar: &core.ScalarValue{
							BoolValue: &condition1,
						},
					},
				},
				{
					Or: []*provider.ResolvedResourceCondition{
						{
							StringValue: &core.MappingNode{
								Scalar: &core.ScalarValue{
									BoolValue: &condition2,
								},
							},
						},
						{
							Not: &provider.ResolvedResourceCondition{
								StringValue: &core.MappingNode{
									Scalar: &core.ScalarValue{
										BoolValue: &condition3,
									},
								},
							},
						},
					},
				},
			},
		},
		LinkSelector: &schema.LinkSelector{
			ByLabel: &schema.StringMap{
				Values: map[string]string{
					"app": "orders",
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"region": (*core.MappingNode)(nil),
				"tableName": {
					Scalar: &core.ScalarValue{
						StringValue: &tableName,
					},
				},
			},
		},
	}
}

func resolveInResourceTestParams() core.BlueprintParams {
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

func TestSubstitutionResourceResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionResourceResolverTestSuite))
}
