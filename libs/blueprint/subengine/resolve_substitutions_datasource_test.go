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

type SubstitutionDataSourceResolverTestSuite struct {
	SubResolverTestContainer
	suite.Suite
}

const (
	resolveInDataSourceFixtureName  = "resolve-in-datasource"
	resolveInDataSourceFixture2Name = "resolve-in-datasource-2"
)

func (s *SubstitutionDataSourceResolverTestSuite) SetupSuite() {
	s.populateSpecFixtureSchemas(map[string]string{
		resolveInDataSourceFixtureName:  "__testdata/sub-resolver/resolve-in-datasource-blueprint.yml",
		resolveInDataSourceFixture2Name: "__testdata/sub-resolver/resolve-in-datasource-blueprint-2.yml",
	}, &s.Suite)
}

func (s *SubstitutionDataSourceResolverTestSuite) SetupTest() {
	s.populateDependencies()
}

func (s *SubstitutionDataSourceResolverTestSuite) Test_resolves_substitutions_in_datasource_for_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInDataSourceFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInDataSourceTestParams()
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

	result, err := subResolver.ResolveInDataSource(
		context.TODO(),
		"network",
		blueprint.DataSources.Values["network"],
		&ResolveDataSourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionDataSourceResolverTestSuite) Test_resolves_substitutions_in_datasource_that_exports_all_fields() {
	blueprint := s.specFixtureSchemas[resolveInDataSourceFixture2Name]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInDataSourceTestParams()
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

	result, err := subResolver.ResolveInDataSource(
		context.TODO(),
		"network",
		blueprint.DataSources.Values["network"],
		&ResolveDataSourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *SubstitutionDataSourceResolverTestSuite) Test_resolves_substitutions_in_datasource_for_deployment() {
	blueprint := s.specFixtureSchemas[resolveInDataSourceFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInDataSourceTestParams()
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
	// ordersTable.spec.id is used in the resource and should be resolved using the resource
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

	// The resource must be resolved before the data source can be resolved.
	// During change staging, the blueprint container will make sure that resources
	// are ordered correctly so that by the time any resource is referenced, it has
	// already been resolved.
	s.resourceCache.Set("ordersTable", &provider.ResolvedResource{})

	result, err := subResolver.ResolveInDataSource(
		ctx,
		"network",
		blueprint.DataSources.Values["network"],
		&ResolveDataSourceTargetInfo{
			ResolveFor:        ResolveForDeployment,
			PartiallyResolved: partiallyResolvedDataSource(),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func partiallyResolvedDataSource() *provider.ResolvedDataSource {
	displayName := "Networking"
	filterField := "tags"
	subnetIdsDescription := "The IDs of the subnets."
	vpcDescription := "The ID of the VPC.\n"
	vpcField := "vpcId"
	dataSourceDescription := "Networking resources for the application."
	return &provider.ResolvedDataSource{
		Type: &schema.DataSourceTypeWrapper{
			Value: "aws/vpc",
		},
		DataSourceMetadata: &provider.ResolvedDataSourceMetadata{
			DisplayName: &core.MappingNode{
				Scalar: &core.ScalarValue{
					StringValue: &displayName,
				},
			},
		},
		Filter: &provider.ResolvedDataSourceFilters{
			Filters: []*provider.ResolvedDataSourceFilter{
				{
					Field: &core.ScalarValue{
						StringValue: &filterField,
					},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorNotContains,
					},
				},
			},
		},
		Exports: map[string]*provider.ResolvedDataSourceFieldExport{
			"subnetIds": {
				Type: &schema.DataSourceFieldTypeWrapper{
					Value: schema.DataSourceFieldTypeArray,
				},
				Description: &core.MappingNode{
					Scalar: &core.ScalarValue{
						StringValue: &subnetIdsDescription,
					},
				},
			},
			"vpc": {
				Type: &schema.DataSourceFieldTypeWrapper{
					Value: schema.DataSourceFieldTypeString,
				},
				AliasFor: &core.ScalarValue{
					StringValue: &vpcField,
				},
				Description: &core.MappingNode{
					Scalar: &core.ScalarValue{
						StringValue: &vpcDescription,
					},
				},
			},
		},
		Description: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &dataSourceDescription,
			},
		},
	}
}

func resolveInDataSourceTestParams() core.BlueprintParams {
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

func TestSubstitutionDataSourceResolverTestSuite(t *testing.T) {
	suite.Run(t, new(SubstitutionDataSourceResolverTestSuite))
}
