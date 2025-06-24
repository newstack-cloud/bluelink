package plugintestsuites

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

const (
	trimSuffixFunctionName = "trim_suffix"
	alterListFunction      = "alter_list"
	alterMapFunction       = "alter_map"
	alterObjectFunction    = "alter_object"
	composeFunction        = "compose"
	mapFunction            = "map"
)

func (s *ProviderPluginV1Suite) Test_function_get_definition_1() {
	function, err := s.provider.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.TrimSuffixFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_2() {
	function, err := s.provider.Function(
		context.Background(),
		alterListFunction,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.AlterListFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_3() {
	function, err := s.provider.Function(
		context.Background(),
		alterMapFunction,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.AlterMapFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_4() {
	function, err := s.provider.Function(
		context.Background(),
		alterObjectFunction,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.AlterObjectFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_5() {
	function, err := s.provider.Function(
		context.Background(),
		composeFunction,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.ComposeFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_6() {
	function, err := s.provider.Function(
		context.Background(),
		mapFunction,
	)
	s.Require().NoError(err)

	output, err := function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionGetDefinitionOutput{
			Definition: testprovider.MapFunctionDefinition(),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_defintion_fails_for_unexpected_host() {
	function, err := s.providerWrongHost.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	_, err = function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetFunctionDefinition,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_function_get_definition_reports_expected_error_for_failure() {
	function, err := s.failingProvider.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	_, err = function.GetDefinition(
		context.Background(),
		functionGetDefinitionInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving function definition")
}

func (s *ProviderPluginV1Suite) Test_function_call() {
	fn, err := s.provider.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	callStack := function.NewStack()
	// Add an initial call to ensure parent calls are sent
	// to the provider.
	callStack.Push(
		&function.Call{
			FunctionName: "parent_function",
			Location: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 10,
				},
				EndPosition: &source.Position{
					Line:   2,
					Column: 20,
				},
			},
		},
	)
	registryForCall := s.funcRegistry.ForCallContext(callStack)
	callContext := &testutils.FunctionCallContextMock{
		CallCtxRegistry: registryForCall,
		CallStack:       callStack,
		CallCtxParams:   testutils.CreateEmptyConcreteParams(),
	}
	output, err := fn.Call(
		context.Background(),
		&provider.FunctionCallInput{
			Arguments: callContext.NewCallArgs(
				"Hello, universe!",
				", universe!",
			),
			CallContext: callContext,
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.FunctionCallOutput{
			ResponseData: "Hello",
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_function_call_fails_for_unexpected_host() {
	fn, err := s.providerWrongHost.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	callStack := function.NewStack()
	registryForCall := s.funcRegistry.ForCallContext(callStack)
	callContext := &testutils.FunctionCallContextMock{
		CallCtxRegistry: registryForCall,
		CallStack:       callStack,
		CallCtxParams:   testutils.CreateEmptyConcreteParams(),
	}
	_, err = fn.Call(
		context.Background(),
		&provider.FunctionCallInput{
			Arguments: callContext.NewCallArgs(
				"Hello there!",
				" there!",
			),
			CallContext: callContext,
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderCallFunction,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_function_call_reports_expected_error_for_failure() {
	fn, err := s.failingProvider.Function(
		context.Background(),
		trimSuffixFunctionName,
	)
	s.Require().NoError(err)

	callStack := function.NewStack()
	registryForCall := s.funcRegistry.ForCallContext(callStack)
	callContext := &testutils.FunctionCallContextMock{
		CallCtxRegistry: registryForCall,
		CallStack:       callStack,
		CallCtxParams:   testutils.CreateEmptyConcreteParams(),
	}
	_, err = fn.Call(
		context.Background(),
		&provider.FunctionCallInput{
			Arguments: callContext.NewCallArgs(
				"Input with suffix!",
				" with suffix!",
			),
			CallContext: callContext,
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred calling function")
}
