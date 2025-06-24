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

type ComposeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	suite.Suite
}

func (s *ComposeFunctionTestSuite) SetupTest() {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"_compose_exec": NewComposeExecFunction(),
				"getattr":       NewGetAttrFunction(),
				"_getattr_exec": NewGetAttrExecFunction(),
				"to_upper":      NewToUpperFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *ComposeFunctionTestSuite) Test_composes_functions_together_and_executes_composed_function_correctly() {
	composeFunc := NewComposeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "compose",
	})

	getAttrFuncOutput, err := s.callContext.registry.Call(
		context.TODO(),
		"getattr",
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					"id",
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	s.Require().NoError(err)

	output, err := composeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]provider.FunctionRuntimeInfo{
					{
						FunctionName: "to_upper",
					},
					getAttrFuncOutput.FunctionInfo,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)

	// Execute the composition.
	args := []any{
		map[string]interface{}{
			"id":    "test-id-10392",
			"label": "Test Label 10392",
		},
	}
	args = append(args, output.FunctionInfo.PartialArgs...)
	result, err := s.callContext.registry.Call(
		context.TODO(),
		output.FunctionInfo.FunctionName,
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args:    args,
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal("TEST-ID-10392", result.ResponseData)
}

func (s *ComposeFunctionTestSuite) Test_composition_execution_fails_for_invalid_args_offset() {
	composeFunc := NewComposeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "compose",
	})

	getAttrFuncOutput, err := s.callContext.registry.Call(
		context.TODO(),
		"getattr",
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					"id",
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	s.Require().NoError(err)

	// 30 is not a valid args offset, _compose_exec expects 1 or 0.
	getAttrFuncOutput.FunctionInfo.ArgsOffset = 30
	output, err := composeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]provider.FunctionRuntimeInfo{
					{
						FunctionName: "to_upper",
					},
					getAttrFuncOutput.FunctionInfo,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)

	// Execute the composition.
	args := []any{
		map[string]interface{}{
			"id":    "test-id-32392",
			"label": "Test Label 32392",
		},
	}
	args = append(args, output.FunctionInfo.PartialArgs...)
	_, err = s.callContext.registry.Call(
		context.TODO(),
		output.FunctionInfo.FunctionName,
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args:    args,
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal("invalid args offset defined for the partially applied function \"_getattr_exec\"", funcErr.Message)
	s.Assert().Equal([]*function.Call{
		{
			FunctionName: "_compose_exec",
		},
		{
			FunctionName: "compose",
		},
	}, funcErr.CallStack)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidArgsOffset, funcErr.Code)
}

func TestComposeFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(ComposeFunctionTestSuite))
}
