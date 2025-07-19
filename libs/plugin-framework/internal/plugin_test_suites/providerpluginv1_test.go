// Integrated test suite for v1 provider plugins.
// This suite is designed to test the full lifecycle of a v1 provider plugin
// including registration and interaction with the host service,
// this is an integrated test that comes close to an end-to-end test,
// the only difference is that the network listener is in-process
// meaning that the host service and provider plugin are running in the same process
// for the automated tests.
package plugintestsuites

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/stretchr/testify/suite"
)

const (
	testHostID      = "test-host-id"
	testWrongHostID = "wrong-host-id"
)

type ProviderPluginV1Suite struct {
	pluginService     pluginservicev1.ServiceClient
	provider          provider.Provider
	providerWrongHost provider.Provider
	failingProvider   provider.Provider
	funcRegistry      provider.FunctionRegistry
	stateContainer    state.Container

	closePluginService   func()
	closeProvider        func()
	closeFailingProvider func()
	suite.Suite
}

func (s *ProviderPluginV1Suite) SetupSuite() {
	providers := map[string]provider.Provider{}
	pluginManager := pluginservicev1.NewManager(
		map[pluginservicev1.PluginType]string{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    "1.0",
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: "1.0",
		},
		s.createPluginInstance,
		testHostID,
	)
	s.funcRegistry = provider.NewFunctionRegistry(
		providers,
	)
	s.stateContainer = testutils.NewMemoryStateContainer()
	err := s.populateResourceStateToLookup()
	s.Require().NoError(err)

	resourceRegistry := resourcehelpers.NewRegistry(
		providers,
		map[string]transform.SpecTransformer{},
		/* stabilisationPollingInterval */ 1*time.Millisecond,
		s.stateContainer,
		core.NewDefaultParams(
			map[string]map[string]*core.ScalarValue{},
			map[string]map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
		),
	)
	pluginService, closePluginService := testutils.StartPluginServiceServer(
		testHostID,
		pluginManager,
		s.funcRegistry,
		/* resourceService */ resourceRegistry,
	)
	s.pluginService = pluginService
	s.closePluginService = closePluginService

	providerClient, closeProvider := testprovider.StartPluginServer(
		pluginService,
		/* failingPlugin */ false,
	)
	s.closeProvider = closeProvider
	s.provider = providerserverv1.WrapProviderClient(providerClient, testHostID)
	namespace, err := s.provider.Namespace(context.Background())
	s.Require().NoError(err)
	providers[namespace] = s.provider

	failingProviderClient, closeFailingProvider := testprovider.StartPluginServer(
		pluginService,
		/* failingPlugin */ true,
	)
	s.closeFailingProvider = closeFailingProvider
	s.failingProvider = providerserverv1.WrapProviderClient(failingProviderClient, testHostID)

	s.providerWrongHost = providerserverv1.WrapProviderClient(providerClient, testWrongHostID)
}

func (s *ProviderPluginV1Suite) populateResourceStateToLookup() error {
	input := linkUpdateIntermediaryResourcesInput()
	return s.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:   input.ResourceAInfo.InstanceID,
			InstanceName: input.InstanceName,
			ResourceIDs: map[string]string{
				input.ResourceAInfo.ResourceName: input.ResourceAInfo.ResourceID,
			},
			Resources: map[string]*state.ResourceState{
				input.ResourceAInfo.ResourceName: input.ResourceAInfo.CurrentResourceState,
			},
		},
	)
}

func (s *ProviderPluginV1Suite) Test_get_provider_namespace() {
	namespace, err := s.provider.Namespace(context.Background())
	s.Require().NoError(err)
	s.Require().Equal("aws", namespace)
}

func (s *ProviderPluginV1Suite) Test_get_provider_namespace_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.Namespace(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetNamespace,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_get_provider_namespace_reports_expected_error_for_failure() {
	_, err := s.failingProvider.Namespace(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving namespace")
}

func (s *ProviderPluginV1Suite) Test_get_provider_config_definition() {
	configDefinition, err := s.provider.ConfigDefinition(context.Background())
	s.Require().NoError(err)
	testutils.AssertConfigDefinitionEquals(
		testprovider.TestProviderConfigDefinition(),
		configDefinition,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_get_provider_config_definition_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ConfigDefinition(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetConfigDefinition,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_get_provider_config_definition_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ConfigDefinition(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving config definition")
}

func (s *ProviderPluginV1Suite) Test_list_resource_types() {
	resourceTypes, err := s.provider.ListResourceTypes(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal([]string{"aws/lambda/function"}, resourceTypes)
}

func (s *ProviderPluginV1Suite) Test_list_resource_types_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ListResourceTypes(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderListResourceTypes,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_list_resource_types_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ListResourceTypes(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing resource types")
}

func (s *ProviderPluginV1Suite) Test_list_link_types() {
	linkTypes, err := s.provider.ListLinkTypes(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal([]string{"aws/lambda/function::aws/dynamodb/table"}, linkTypes)
}

func (s *ProviderPluginV1Suite) Test_list_link_types_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ListLinkTypes(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderListLinkTypes,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_list_link_types_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ListLinkTypes(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing link types")
}

func (s *ProviderPluginV1Suite) Test_list_data_source_types() {
	dataSourceTypes, err := s.provider.ListDataSourceTypes(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal([]string{"aws/vpc"}, dataSourceTypes)
}

func (s *ProviderPluginV1Suite) Test_list_data_source_types_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ListDataSourceTypes(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderListDataSourceTypes,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_list_data_source_types_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ListDataSourceTypes(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing data source types")
}

func (s *ProviderPluginV1Suite) Test_list_custom_variable_types() {
	customVarTypes, err := s.provider.ListCustomVariableTypes(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal([]string{"aws/ec2/instanceType"}, customVarTypes)
}

func (s *ProviderPluginV1Suite) Test_list_custom_variable_types_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ListCustomVariableTypes(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderListCustomVariableTypes,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_list_custom_variable_types_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ListCustomVariableTypes(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing custom variable types")
}

func (s *ProviderPluginV1Suite) Test_list_functions() {
	functions, err := s.provider.ListFunctions(context.Background())
	s.Require().NoError(err)
	expected := utils.GetKeys(testprovider.Functions())
	slices.Sort(functions)
	slices.Sort(expected)
	s.Assert().Equal(expected, functions)
}

func (s *ProviderPluginV1Suite) Test_list_functions_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.ListFunctions(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderListFunctions,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_list_functions_reports_expected_error_for_failure() {
	_, err := s.failingProvider.ListFunctions(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing functions")
}

func (s *ProviderPluginV1Suite) Test_get_retry_policy() {
	retryPolicy, err := s.provider.RetryPolicy(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal(testprovider.TestProviderRetryPolicy(), retryPolicy)
}

func (s *ProviderPluginV1Suite) Test_get_retry_policy_fails_for_unexpected_host() {
	_, err := s.providerWrongHost.RetryPolicy(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetRetryPolicy,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_get_retry_policy_reports_expected_error_for_failure() {
	_, err := s.failingProvider.RetryPolicy(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving retry policy")
}

func (s *ProviderPluginV1Suite) createPluginInstance(
	info *pluginservicev1.PluginInstanceInfo,
	hostID string,
) (any, func(), error) {
	// This is required for the manager that backs the plugin service that allows
	// the plugin to register itself with the host service.
	// For the provider v1 plugin test suite, we are not testing the management
	// of plugins, the provider plugin to be tested is instantiated as a part of
	// the test suite setup.
	return nil, nil, nil
}

func (s *ProviderPluginV1Suite) TearDownSuite() {
	s.closeProvider()
	s.closeFailingProvider()
	// We must close the plugin service after the provider plugin
	// so it can deregister itself.
	s.closePluginService()
}

func TestProviderPluginV1Suite(t *testing.T) {
	suite.Run(t, new(ProviderPluginV1Suite))
}
