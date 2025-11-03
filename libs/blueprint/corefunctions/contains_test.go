package corefunctions

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ContainsFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	suite.Suite
}

func (s *ContainsFunctionTestSuite) SetupTest() {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *ContainsFunctionTestSuite) Test_string_contains_substring() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	output, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is a string that contains a substring",
				"contains",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputBool, isBool := output.ResponseData.(bool)
	s.Assert().True(isBool)
	s.Assert().True(outputBool)
}

func (s *ContainsFunctionTestSuite) Test_array_contains_element_primitive() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	output, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{1, 3, 5, 9, 10},
				9,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputBool, isBool := output.ResponseData.(bool)
	s.Assert().True(isBool)
	s.Assert().True(outputBool)
}

func (s *ContainsFunctionTestSuite) Test_array_contains_element_comparable() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	output, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{comparableInt(1), comparableInt(5), comparableInt(9), comparableInt(10)},
				comparableInt(9),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputBool, isBool := output.ResponseData.(bool)
	s.Assert().True(isBool)
	s.Assert().True(outputBool)
}

func (s *ContainsFunctionTestSuite) Test_returns_func_error_for_invalid_input_string_search() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	_, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com/config/2",
				// A string is expected here as the first argument "haystack" is a string
				// and the second argument "needle" should also be a string.
				785043,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal(
		"Invalid input type. Expected string for item to search for in a string search space, received int",
		funcErr.Message,
	)
	s.Assert().Equal(
		[]*function.Call{
			{
				FunctionName: "contains",
			},
		},
		funcErr.CallStack,
	)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidInput, funcErr.Code)
}

func (s *ContainsFunctionTestSuite) Test_returns_func_error_for_invalid_input_search_space() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	_, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// A string or an array is expected for the search space.
				struct{ Value string }{},
				"needle",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal(
		"Invalid input type. Expected string or array for "+
			"search space, received struct { Value string }",
		funcErr.Message,
	)
	s.Assert().Equal(
		[]*function.Call{
			{
				FunctionName: "contains",
			},
		},
		funcErr.CallStack,
	)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidInput, funcErr.Code)
}

func (s *ContainsFunctionTestSuite) Test_returns_false_for_none_value() {
	containsFunc := NewContainsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "contains",
	})
	output, err := containsFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				core.GetNoneMarker(),
				"substring",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputBool, isBool := output.ResponseData.(bool)
	s.Assert().True(isBool)
	s.Assert().False(outputBool)
}

func TestContainsFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(ContainsFunctionTestSuite))
}
