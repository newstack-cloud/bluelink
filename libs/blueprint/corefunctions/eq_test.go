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

type EqFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	suite.Suite
}

func (s *EqFunctionTestSuite) SetupTest() {
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

func (s *EqFunctionTestSuite) Test_equals_1() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is a string",
				"This is a string",
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

func (s *EqFunctionTestSuite) Test_equals_2() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is a string",
				"This is a different string",
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

func (s *EqFunctionTestSuite) Test_equals_3() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]interface{}{
					"key": "value",
				},
				map[string]interface{}{
					"key": "value",
				},
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

func (s *EqFunctionTestSuite) Test_equals_4() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]interface{}{
					"key1": "value1",
				},
				map[string]interface{}{
					"key1": "value1",
					"key2": []interface{}{1, 2, 3},
				},
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

func (s *EqFunctionTestSuite) Test_equals_comparable_1() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				comparableInt(9),
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

func (s *EqFunctionTestSuite) Test_equals_comparable_2() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	output, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				comparableInt(9),
				comparableInt(15),
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

func (s *EqFunctionTestSuite) Test_returns_func_error_for_comparison_type_mismatch() {
	eqFunc := NewEqFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "eq",
	})
	_, err := eqFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is a string",
				// An integer cannot be compared to a string.
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
		"expected both values to be of the same type, got string and int",
		funcErr.Message,
	)
	s.Assert().Equal([]*function.Call{{FunctionName: "eq"}}, funcErr.CallStack)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidArgumentType, funcErr.Code)
}

func TestEqFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(EqFunctionTestSuite))
}
