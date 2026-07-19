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
	resolveInResourceFixtureName                    = "resolve-in-resource"
	resolveInResourcePartialAnnotationsFixtureName  = "resolve-in-resource-partial-annotations"
	resolveInResourceTemplateFixtureName            = "resolve-in-resource-2"
	resolveInResourceComputedWhenOmittedFixtureName = "resolve-in-resource-computed-when-omitted"
	resolveInResourceComputedFromStateFixtureName   = "resolve-in-resource-computed-from-state"
	testInstanceID                                  = "cb826a32-1052-4fde-aa6e-d449b9f50066"
	testChildInstanceID                             = "05253564-cd77-4b92-81bc-e75f9478ec4d"
)

func (s *SubstitutionResourceResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(map[string]string{
		resolveInResourceFixtureName:                    "__testdata/sub-resolver/resolve-in-resource-blueprint.yml",
		resolveInResourcePartialAnnotationsFixtureName:  "__testdata/sub-resolver/resolve-in-resource-partial-annotations-blueprint.yml",
		resolveInResourceTemplateFixtureName:            "__testdata/sub-resolver/resolve-in-resource-2-blueprint.yml",
		resolveInResourceComputedWhenOmittedFixtureName: "__testdata/sub-resolver/resolve-in-resource-computed-when-omitted-blueprint.yml",
		resolveInResourceComputedFromStateFixtureName:   "__testdata/sub-resolver/resolve-in-resource-computed-from-state-blueprint.yml",
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

// Regression test for phantom modified-field changes on restaging: a
// reference to another resource's computed field must be resolved from the
// referenced resource's existing state during update change staging so that
// change staging compares concrete values, while the property is still
// marked to be resolved on deploy.
func (s *SubstitutionResourceResolverTestSuite) Test_resolves_computed_field_reference_from_existing_state_for_update_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourceComputedFromStateFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
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

	err := s.stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID: testInstanceID,
	})
	s.Require().NoError(err)

	resourceID := "test-orders-table-309428320"
	err = s.stateContainer.Resources().Save(
		context.Background(),
		state.ResourceState{
			ResourceID: resourceID,
			InstanceID: testInstanceID,
			Name:       "ordersTable",
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"id": {
						Scalar: &core.ScalarValue{
							StringValue: &resourceID,
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Make sure the current instance ID can be retrieved from the context when
	// fetching state from the state container to resolve the computed field reference.
	ctx := context.WithValue(
		context.Background(),
		core.BlueprintInstanceIDKey,
		testInstanceID,
	)

	result, err := subResolver.ResolveInResource(
		ctx,
		"saveOrderFunction",
		blueprint.Resources.Values["saveOrderFunction"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// The property must still be re-resolved during deployment as the
	// computed field of the referenced resource can change as part of
	// the deployment.
	s.Assert().Contains(
		result.ResolveOnDeploy,
		"resources.saveOrderFunction.spec.tableId",
	)

	// The concrete value from the referenced resource's existing state must be
	// used for the resolved property so change staging can compare concrete
	// values instead of planning phantom modifications with unknown new values.
	resolvedTableID := result.ResolvedResource.Spec.Fields["tableId"]
	s.Require().NotNil(resolvedTableID)
	s.Assert().Equal(resourceID, core.StringValue(resolvedTableID))

	// The same applies to a computed field reference embedded in an
	// interpolated string.
	s.Assert().Contains(
		result.ResolveOnDeploy,
		"resources.saveOrderFunction.spec.tableEndpoint",
	)
	resolvedTableEndpoint := result.ResolvedResource.Spec.Fields["tableEndpoint"]
	s.Require().NotNil(resolvedTableEndpoint)
	s.Assert().Equal(
		"https://"+resourceID+".example.com",
		core.StringValue(resolvedTableEndpoint),
	)
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

func (s *SubstitutionResourceResolverTestSuite) Test_reference_to_omitted_computed_when_omitted_field_resolves_on_deploy() {
	blueprint := s.specFixtureSchemas[resolveInResourceComputedWhenOmittedFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
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
		resolveInResourceTestParams(),
	)

	resolvedTableResult, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.resourceCache.Set("ordersTable", resolvedTableResult.ResolvedResource)

	// The table's `tableName` is computed-when-omitted and not defined in the
	// blueprint, so a reference to it must be deferred to deployment instead of
	// failing change staging with a missing property error.
	result, err := subResolver.ResolveInResource(
		context.TODO(),
		"saveOrderFunction",
		blueprint.Resources.Values["saveOrderFunction"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Contains(result.ResolveOnDeploy, "resources.saveOrderFunction.spec.handler")
}

func (s *SubstitutionResourceResolverTestSuite) Test_reference_to_set_computed_when_omitted_field_resolves_at_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourceComputedWhenOmittedFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
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
		resolveInResourceTestParams(),
	)

	resolvedTableResult, err := subResolver.ResolveInResource(
		context.TODO(),
		"namedOrdersTable",
		blueprint.Resources.Values["namedOrdersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.resourceCache.Set("namedOrdersTable", resolvedTableResult.ResolvedResource)

	result, err := subResolver.ResolveInResource(
		context.TODO(),
		"saveNamedOrderFunction",
		blueprint.Resources.Values["saveNamedOrderFunction"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Empty(result.ResolveOnDeploy)
	handler := result.ResolvedResource.Spec.Fields["handler"]
	s.Require().NotNil(handler)
	s.Equal("explicit-orders", core.StringValue(handler))
}

func TestSubstitutionResourceResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionResourceResolverTestSuite))
}
