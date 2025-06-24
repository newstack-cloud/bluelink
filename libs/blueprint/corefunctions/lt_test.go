package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type LtFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&LtFunctionTestSuite{})

func (s *LtFunctionTestSuite) SetUpTest(c *C) {
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

func (s *LtFunctionTestSuite) Test_less_than_case_1(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	output, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(501),
				int64(502),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, true)
}

func (s *LtFunctionTestSuite) Test_less_than_case_2(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	output, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(505),
				float64(501.5),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, false)
}

func (s *LtFunctionTestSuite) Test_less_than_case_3(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	output, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				float64(500.1),
				float64(500.4),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, true)
}

func (s *LtFunctionTestSuite) Test_less_than_case_4(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	output, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				float64(514.5),
				int64(508),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, false)
}

func (s *LtFunctionTestSuite) Test_returns_func_error_for_invalid_input_a(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	_, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// An int64 or float64 is expected here, not a boolean.
				true,
				int64(6204),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "expected the left-hand side of the comparison to be a number, got bool")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "lt",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *LtFunctionTestSuite) Test_returns_func_error_for_invalid_input_b(c *C) {
	ltFunc := NewLtFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lt",
	})
	_, err := ltFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(6104),
				// An int64 or float64 is expected here, not a map[string]interface{}.
				map[string]interface{}{
					"value": 6314,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "expected the right-hand side of the comparison to be a number, got map[string]interface {}")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "lt",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
