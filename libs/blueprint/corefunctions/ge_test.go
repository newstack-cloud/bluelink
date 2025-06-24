package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type GeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&GeFunctionTestSuite{})

func (s *GeFunctionTestSuite) SetUpTest(c *C) {
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

func (s *GeFunctionTestSuite) Test_greater_than_equal_case_1(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	output, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(689),
				int64(689),
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

func (s *GeFunctionTestSuite) Test_greater_than_equal_case_2(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	output, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(600),
				float64(602.8),
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

func (s *GeFunctionTestSuite) Test_greater_than_equal_case_3(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	output, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				float64(500.5),
				float64(500.2),
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

func (s *GeFunctionTestSuite) Test_greater_than_equal_case_4(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	output, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				float64(324.5),
				int64(609),
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

func (s *GeFunctionTestSuite) Test_returns_func_error_for_invalid_input_a(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	_, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// An int64 or float64 is expected here, not a boolean.
				true,
				int64(7859),
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
			FunctionName: "ge",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *GeFunctionTestSuite) Test_returns_func_error_for_invalid_input_b(c *C) {
	geFunc := NewGeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "ge",
	})
	_, err := geFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(3402),
				// An int64 or float64 is expected here, not a map[string]interface{}.
				map[string]interface{}{
					"value": 8493,
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
			FunctionName: "ge",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
