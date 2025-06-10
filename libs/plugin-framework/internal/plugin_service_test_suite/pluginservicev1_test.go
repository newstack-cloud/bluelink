// Integrated test suite for the v1 plugin service.
// This suite is designed to test the full lifecycle of the host plugin
// service that allows plugins to register and interact with each other.
package pluginservicetestsuite

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/newstack-cloud/celerity/libs/blueprint/core"
	"github.com/newstack-cloud/celerity/libs/blueprint/function"
	"github.com/newstack-cloud/celerity/libs/blueprint/provider"
	"github.com/newstack-cloud/celerity/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/celerity/libs/blueprint/transform"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/internal/testutils"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/utils"
	"github.com/stretchr/testify/suite"
)

const (
	testHostID      = "test-host-id"
	testWrongHostID = "wrong-host-id"
)

type PluginServiceV1Suite struct {
	pluginService pluginservicev1.ServiceClient
	funcRegistry  provider.FunctionRegistry
	provider      provider.Provider

	closePluginService func()
	closeProvider      func()
	suite.Suite
}

func (s *PluginServiceV1Suite) SetupTest() {
	pluginManager := pluginservicev1.NewManager(
		map[pluginservicev1.PluginType]string{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    "1.0",
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: "1.0",
		},
		s.createPluginInstance,
		testHostID,
	)
	providers := map[string]provider.Provider{}
	s.funcRegistry = provider.NewFunctionRegistry(
		providers,
	)
	pluginService, closePluginService := testutils.StartPluginServiceServer(
		testHostID,
		pluginManager,
		s.funcRegistry,
		resourcehelpers.NewRegistry(
			providers,
			map[string]transform.SpecTransformer{},
			/* stabilisationPollingInterval */ 1*time.Millisecond,
			core.NewDefaultParams(
				map[string]map[string]*core.ScalarValue{},
				map[string]map[string]*core.ScalarValue{},
				map[string]*core.ScalarValue{},
				map[string]*core.ScalarValue{},
			),
		),
	)
	s.pluginService = pluginService
	s.closePluginService = closePluginService

	// Initialise a provider to carry out end-to-end tests of the plugin service
	// as a gateway to allow calls to provider plugin elements such as functions
	// and resources through the plugin service.
	// Typically these calls would be made across multiple providers, but for the sake
	// of simplicity, we will use a single provider plugin to test the plugin service.
	// In the future, if functionality is added to optimise calls between elements in the same
	// provider (that bypass the plugin service),
	// these tests will need to be altered to get the same coverage of the plugin service.
	providerClient, closeProvider := testprovider.StartPluginServer(
		pluginService,
		/* failingPlugin */ false,
	)
	s.closeProvider = closeProvider
	s.provider = providerserverv1.WrapProviderClient(providerClient, testHostID)
	namespace, err := s.provider.Namespace(context.Background())
	s.Require().NoError(err)
	providers[namespace] = s.provider
}

func (s *PluginServiceV1Suite) TearDownTest() {
	s.closeProvider()
	// We must close the plugin service after the provider plugin
	// so it can deregister itself.
	s.closePluginService()
}

func (s *PluginServiceV1Suite) Test_fails_to_deregister_plugin_with_wrong_host_id() {
	response, err := s.pluginService.Deregister(
		context.TODO(),
		&pluginservicev1.PluginDeregistrationRequest{
			PluginType: pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
			HostId:     testWrongHostID,
			InstanceId: "1",
		},
	)
	s.Require().NoError(err)
	s.Assert().False(response.Success)
	expectedMessage := fmt.Sprintf(
		"failed to deregister plugin due to error: host id mismatch, expected %q, got %q",
		testHostID,
		testWrongHostID,
	)
	s.Assert().Equal(
		expectedMessage,
		response.Message,
	)
}

func (s *PluginServiceV1Suite) Test_function_call_between_plugin_functions() {
	trimFunc, err := s.provider.Function(
		context.Background(),
		"trim_space_and_suffix",
	)
	s.Require().NoError(err)

	callStack := function.NewStack()
	registryForCall := s.funcRegistry.ForCallContext(callStack)
	callContext := &testutils.FunctionCallContextMock{
		CallCtxRegistry: registryForCall,
		CallStack:       callStack,
		CallCtxParams:   testutils.CreateEmptyConcreteParams(),
	}
	resp, err := trimFunc.Call(
		context.Background(),
		&provider.FunctionCallInput{
			Arguments:   callContext.NewCallArgs("   localhost:3000 ", ":3000"),
			CallContext: callContext,
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		"localhost",
		resp.ResponseData,
	)
}

func (s *PluginServiceV1Suite) Test_fails_with_timeout_for_cyclic_function_calls() {
	callSelfFunc, err := s.provider.Function(context.Background(), "call_self")
	s.Require().NoError(err)

	callStack := function.NewStack()
	registryForCall := s.funcRegistry.ForCallContext(callStack)
	callContext := &testutils.FunctionCallContextMock{
		CallCtxRegistry: registryForCall,
		CallStack:       callStack,
		CallCtxParams:   testutils.CreateEmptyConcreteParams(),
	}
	_, err = callSelfFunc.Call(
		context.Background(),
		&provider.FunctionCallInput{
			Arguments:   callContext.NewCallArgs(),
			CallContext: callContext,
		},
	)
	s.Require().Error(err)
	s.Assert().Contains(
		err.Error(),
		"context deadline exceeded",
	)
}

func (s *PluginServiceV1Suite) Test_get_function_definition() {
	hostInfoContainer := pluginutils.NewHostInfoContainer()
	hostInfoContainer.SetID(testHostID)
	registryWrapper := pluginservicev1.FunctionRegistryFromClient(
		s.pluginService,
		hostInfoContainer,
	)
	resp, err := registryWrapper.GetDefinition(
		context.Background(),
		"trim_space_and_suffix",
		&provider.FunctionGetDefinitionInput{
			Params: testutils.CreateEmptyTestParams(),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.TrimSpaceAndSuffixFunctionDefinition(),
		},
		resp,
	)
}

func (s *PluginServiceV1Suite) Test_list_functions_in_registry() {
	hostInfoContainer := pluginutils.NewHostInfoContainer()
	hostInfoContainer.SetID(testHostID)
	registryWrapper := pluginservicev1.FunctionRegistryFromClient(
		s.pluginService,
		hostInfoContainer,
	)
	functions, err := registryWrapper.ListFunctions(
		context.Background(),
	)
	s.Require().NoError(err)
	expectedFunctions := utils.GetKeys(testprovider.Functions())
	slices.Sort(functions)
	slices.Sort(expectedFunctions)
	s.Assert().Equal(
		expectedFunctions,
		functions,
	)
}

func (s *PluginServiceV1Suite) Test_check_has_function_in_registry() {
	hostInfoContainer := pluginutils.NewHostInfoContainer()
	hostInfoContainer.SetID(testHostID)
	registryWrapper := pluginservicev1.FunctionRegistryFromClient(
		s.pluginService,
		hostInfoContainer,
	)
	hasFunction, err := registryWrapper.HasFunction(
		context.Background(),
		"trim_space_and_suffix",
	)
	s.Require().NoError(err)
	s.Assert().True(hasFunction)
}

func (s *PluginServiceV1Suite) Test_link_deploy_intermediary_resource_call() {
	link, err := s.provider.Link(
		context.Background(),
		"aws/lambda/function",
		"aws/dynamodb/table",
	)
	s.Require().NoError(err)

	output, err := link.UpdateIntermediaryResources(
		context.Background(),
		linkUpdateIntermediaryResourcesInput(
			provider.LinkUpdateTypeCreate,
		),
	)
	s.Require().NoError(err)

	s.Assert().Equal(&provider.LinkUpdateIntermediaryResourcesOutput{}, output)
}

func (s *PluginServiceV1Suite) Test_link_destroy_intermediary_resource_call() {
	link, err := s.provider.Link(
		context.Background(),
		"aws/lambda/function",
		"aws/dynamodb/table",
	)
	s.Require().NoError(err)

	output, err := link.UpdateIntermediaryResources(
		context.Background(),
		linkUpdateIntermediaryResourcesInput(
			provider.LinkUpdateTypeDestroy,
		),
	)
	s.Require().NoError(err)

	s.Assert().Equal(&provider.LinkUpdateIntermediaryResourcesOutput{}, output)
}

func (s *PluginServiceV1Suite) createPluginInstance(
	info *pluginservicev1.PluginInstanceInfo,
	hostID string,
) (any, func(), error) {
	return nil, nil, nil
}

func TestPluginServiceV1Suite(t *testing.T) {
	suite.Run(t, new(PluginServiceV1Suite))
}
