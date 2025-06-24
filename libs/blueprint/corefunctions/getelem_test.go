package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type GetElemFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&GetElemFunctionTestSuite{})

func (s *GetElemFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"_getelem_exec": NewGetElemExecFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *GetElemFunctionTestSuite) Test_retrieves_element_from_array(c *C) {
	getElemFunc := NewGetElemFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "getelem",
	})

	getElemFuncOutput, err := getElemFunc.Call(
		context.TODO(),
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					int64(2),
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	c.Assert(err, IsNil)

	// Execute the attribute retrieval.
	args := []any{
		[]interface{}{
			"item-1",
			"item-2",
			"item-3",
			"item-4",
		},
	}
	args = append(args, getElemFuncOutput.FunctionInfo.PartialArgs...)
	result, err := s.callContext.registry.Call(
		context.TODO(),
		getElemFuncOutput.FunctionInfo.FunctionName,
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args:    args,
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	c.Assert(err, IsNil)

	c.Assert(result.ResponseData, Equals, "item-3")
}

func (s *GetElemFunctionTestSuite) Test_returns_func_call_error_for_index_out_of_bounds(c *C) {
	getAttrFunc := NewGetElemFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "getelem",
	})

	getAttrFuncOutput, err := getAttrFunc.Call(
		context.TODO(),
		&provider.FunctionCallInput{
			Arguments: &functionCallArgsMock{
				args: []any{
					int64(20),
				},
				callCtx: s.callContext,
			},
			CallContext: s.callContext,
		},
	)
	c.Assert(err, IsNil)

	// Execute the composition.
	args := []any{
		[]interface{}{
			"item-101",
			"item-102",
			"item-103",
			"item-104",
		},
	}
	args = append(args, getAttrFuncOutput.FunctionInfo.PartialArgs...)
	_, err = s.callContext.registry.Call(
		context.TODO(),
		getAttrFuncOutput.FunctionInfo.FunctionName,
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
		"index 20 out of bounds",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "_getelem_exec",
		},
		{
			FunctionName: "getelem",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
