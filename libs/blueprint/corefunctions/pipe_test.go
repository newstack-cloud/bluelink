package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type PipeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&PipeFunctionTestSuite{})

func (s *PipeFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"_pipe_exec":    NewPipeExecFunction(),
				"getattr":       NewGetAttrFunction(),
				"_getattr_exec": NewGetAttrExecFunction(),
				"to_upper":      NewToUpperFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *PipeFunctionTestSuite) Test_pipes_functions_together_and_executes_piped_function_correctly(c *C) {
	pipeFunc := NewPipeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "pipe",
	})

	getAttrFuncOutput, err := s.callContext.registry.Call(
		context.TODO(),
		"getattr",
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					"elementId",
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	c.Assert(err, IsNil)

	output, err := pipeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]provider.FunctionRuntimeInfo{
					getAttrFuncOutput.FunctionInfo,
					{
						FunctionName: "to_upper",
					},
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)

	// Execute the piped function.
	args := []any{
		map[string]interface{}{
			"elementId": "test-id-67392",
			"label":     "Test Label 67392",
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
	c.Assert(err, IsNil)

	c.Assert(result.ResponseData, Equals, "TEST-ID-67392")
}

func (s *PipeFunctionTestSuite) Test_piped_execution_fails_for_invalid_args_offset(c *C) {
	pipeFunc := NewPipeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "pipe",
	})

	getAttrFuncOutput, err := s.callContext.registry.Call(
		context.TODO(),
		"getattr",
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					"elementId",
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	c.Assert(err, IsNil)

	// 25 is not a valid args offset, _pipe_exec expects 1 or 0.
	getAttrFuncOutput.FunctionInfo.ArgsOffset = 25
	output, err := pipeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]provider.FunctionRuntimeInfo{
					getAttrFuncOutput.FunctionInfo,
					{
						FunctionName: "to_upper",
					},
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)

	// Execute the composition.
	args := []any{
		map[string]interface{}{
			"elementId": "test-id-67392",
			"label":     "Test Label 67392",
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
	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(
		funcErr.Message,
		Equals,
		"invalid args offset defined for the partially applied function \"_getattr_exec\"",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "_pipe_exec",
		},
		{
			FunctionName: "pipe",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgsOffset)
}
