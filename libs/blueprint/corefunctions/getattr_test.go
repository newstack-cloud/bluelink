package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type GetAttrFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&GetAttrFunctionTestSuite{})

func (s *GetAttrFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"_getattr_exec": NewGetAttrExecFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *GetAttrFunctionTestSuite) Test_retrieves_attribute_from_provided_map(c *C) {
	getAttrFunc := NewGetAttrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "getattr",
	})

	getAttrFuncOutput, err := getAttrFunc.Call(
		context.TODO(),
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

	// Execute the attribute retrieval.
	args := []any{
		map[string]interface{}{
			"elementId": "test-id-23392",
			"label":     "Test Label 23392",
		},
	}
	args = append(args, getAttrFuncOutput.FunctionInfo.PartialArgs...)
	result, err := s.callContext.registry.Call(
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
	c.Assert(err, IsNil)

	c.Assert(result.ResponseData, Equals, "test-id-23392")
}

func (s *GetAttrFunctionTestSuite) Test_returns_func_call_error_for_missing_attribute(c *C) {
	getAttrFunc := NewGetAttrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "getattr",
	})

	getAttrFuncOutput, err := getAttrFunc.Call(
		context.TODO(),
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

	// Execute the composition.
	args := []any{
		map[string]interface{}{
			"id":    "test-id-32392",
			"label": "Test Label 32392",
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
		"attribute \"elementId\" not found in object/mapping",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "_getattr_exec",
		},
		{
			FunctionName: "getattr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
