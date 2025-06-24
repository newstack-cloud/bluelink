package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type LeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&LeFunctionTestSuite{})

func (s *LeFunctionTestSuite) SetUpTest(c *C) {
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

func (s *LeFunctionTestSuite) Test_less_than_equal_case_1(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	output, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(702),
				int64(702),
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

func (s *LeFunctionTestSuite) Test_less_than_equal_case_2(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	output, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(603),
				float64(601.159),
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

func (s *LeFunctionTestSuite) Test_less_than_equal_case_3(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	output, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
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

func (s *LeFunctionTestSuite) Test_less_than_equal_case_4(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	output, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				float64(724.5),
				int64(619),
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

func (s *LeFunctionTestSuite) Test_returns_func_error_for_invalid_input_a(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	_, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// An int64 or float64 is expected here, not a boolean.
				true,
				int64(1859),
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
			FunctionName: "le",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *LeFunctionTestSuite) Test_returns_func_error_for_invalid_input_b(c *C) {
	leFunc := NewLeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "le",
	})
	_, err := leFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(3200),
				// An int64 or float64 is expected here, not a map[string]interface{}.
				map[string]interface{}{
					"value": 8293,
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
			FunctionName: "le",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
