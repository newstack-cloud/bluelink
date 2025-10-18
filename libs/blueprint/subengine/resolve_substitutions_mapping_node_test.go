package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type SubstitutionMappingNodeResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInMappingNodeFixtureName = "resolve-in-mapping-node"
)

func (s *SubstitutionMappingNodeResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(
		map[string]string{
			resolveInMappingNodeFixtureName: "__testdata/sub-resolver/resolve-in-mapping-node-blueprint.yml",
		},
		&s.Suite,
	)
}

func (s *SubstitutionMappingNodeResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionMappingNodeResolverTestSuite) Test_resolves_substitutions_in_mapping_node_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInMappingNodeFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInMappingNodeTestParams()
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

	// The resource must be resolved before the metadata can be resolved.
	// During change staging, the blueprint container will make sure that resources
	// are ordered correctly so that by the time any resource is referenced, it has
	// already been resolved.
	resolvedOrdersTableResult, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.resourceCache.Set("ordersTable", resolvedOrdersTableResult.ResolvedResource)

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

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionMappingNodeResolverTestSuite) Test_resolves_substitutions_in_mapping_node_for_deployment() {
	blueprint := s.specFixtureSchemas[resolveInMappingNodeFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInMappingNodeTestParams()
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
	// ordersTable.spec.id is used in the referenced data source and should be resolved using the resource
	// state.
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

	// Make sure the current instance ID can be retrieved from the context when fetching
	// state from the state container to resolve the resource spec id reference.
	ctx := context.WithValue(
		context.Background(),
		core.BlueprintInstanceIDKey,
		testInstanceID,
	)

	// coreInfra.region is used in the metadata and should be resolved using the child blueprint
	// state.
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

	// The resource must be resolved before the metadata can be resolved.
	// This is required in addition to the resource state to access metadata properties.
	resolvedOrdersTableResult, err := subResolver.ResolveInResource(
		ctx,
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForDeployment,
		},
	)
	s.Require().NoError(err)
	s.resourceCache.Set("ordersTable", resolvedOrdersTableResult.ResolvedResource)

	result, err := subResolver.ResolveInMappingNode(
		ctx,
		"metadata",
		blueprint.Metadata,
		&ResolveMappingNodeTargetInfo{
			ResolveFor:        ResolveForDeployment,
			PartiallyResolved: partiallyResolvedMappingNode(),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func partiallyResolvedMappingNode() *core.MappingNode {
	build := "esbuild"
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"build": {
				Scalar: &core.ScalarValue{
					StringValue: &build,
				},
			},
			"networkingSummary": {
				Fields: map[string]*core.MappingNode{
					"coreInfraRegion": (*core.MappingNode)(nil),
					"vpc":             (*core.MappingNode)(nil),
				},
			},
		},
	}
}

func resolveInMappingNodeTestParams() core.BlueprintParams {
	environment := "production-env"
	enableOrderTableTrigger := true
	region := "us-west-2"
	deployOrdersTableToRegions := "[\"us-west-2\",\"us-east-1\"]"
	relatedInfo := "[{\"id\":\"test-info-1\"},{\"id\":\"test-info-2\"}]"
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
		"relatedInfo": {
			StringValue: &relatedInfo,
		},
	}
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func TestSubstitutionMappingNodeResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionMappingNodeResolverTestSuite))
}
