package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type UTF8FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&UTF8FunctionTestSuite{})

func (s *UTF8FunctionTestSuite) SetUpTest(c *C) {
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

func (s *UTF8FunctionTestSuite) Test_converts_bytes_to_utf8_string(c *C) {
	utf8Func := NewUTF8Function()
	s.callStack.Push(&function.Call{
		FunctionName: "utf8",
	})
	output, err := utf8Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("Hello, World!"),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "Hello, World!")
}

func (s *UTF8FunctionTestSuite) Test_converts_empty_bytes_to_empty_string(c *C) {
	utf8Func := NewUTF8Function()
	s.callStack.Push(&function.Call{
		FunctionName: "utf8",
	})
	output, err := utf8Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte{},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "")
}

func (s *UTF8FunctionTestSuite) Test_converts_utf8_bytes_with_special_chars(c *C) {
	utf8Func := NewUTF8Function()
	s.callStack.Push(&function.Call{
		FunctionName: "utf8",
	})
	output, err := utf8Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("Hello 世界! 🌍"),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "Hello 世界! 🌍")
}

func (s *UTF8FunctionTestSuite) Test_returns_error_for_invalid_argument_type(c *C) {
	utf8Func := NewUTF8Function()
	s.callStack.Push(&function.Call{
		FunctionName: "utf8",
	})
	_, err := utf8Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"not bytes"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 0 is of type string, but target is of type slice")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "utf8",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
