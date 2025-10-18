package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type Base64EncodeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&Base64EncodeFunctionTestSuite{})

func (s *Base64EncodeFunctionTestSuite) SetUpTest(c *C) {
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

func (s *Base64EncodeFunctionTestSuite) Test_base64_encodes_utf8_string(c *C) {
	base64EncodeFunc := NewBase64EncodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64encode",
	})
	output, err := base64EncodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is an example string.",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "VGhpcyBpcyBhbiBleGFtcGxlIHN0cmluZy4=")
}

func (s *Base64EncodeFunctionTestSuite) Test_base64_encodes_byte_array(c *C) {
	base64EncodeFunc := NewBase64EncodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64encode",
	})
	output, err := base64EncodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("Example byte array."),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "RXhhbXBsZSBieXRlIGFycmF5Lg==")
}

func (s *Base64EncodeFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	base64EncodeFunc := NewBase64EncodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64encode",
	})
	_, err := base64EncodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Only a string or byte array is allowed.
				8594032,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "input argument at index 0 must be a byte array or string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "base64encode",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
