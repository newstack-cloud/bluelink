package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type Base64DecodeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&Base64DecodeFunctionTestSuite{})

func (s *Base64DecodeFunctionTestSuite) SetUpTest(c *C) {
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

func (s *Base64DecodeFunctionTestSuite) Test_base64_decodes_to_byte_array(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	output, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"VGhpcyBpcyBhbiBleGFtcGxlIHN0cmluZy4=",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, []byte("This is an example string."))
}

func (s *Base64DecodeFunctionTestSuite) Test_base64_decodes_binary_data(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	output, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"RXhhbXBsZSBieXRlIGFycmF5Lg==",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, []byte("Example byte array."))
}

func (s *Base64DecodeFunctionTestSuite) Test_base64_decodes_empty_string(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	output, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, []byte(""))
}

func (s *Base64DecodeFunctionTestSuite) Test_returns_func_error_for_invalid_base64(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	_, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is not valid base64!@#$%",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Matches, "unable to decode base64 string:.*")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "base64decode",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *Base64DecodeFunctionTestSuite) Test_returns_func_error_for_invalid_input_type(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	_, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Only a string is allowed.
				12345,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 0 is of type int, but target is of type string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "base64decode",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *Base64DecodeFunctionTestSuite) Test_base64_decode_roundtrip_with_encode(c *C) {
	base64EncodeFunc := NewBase64EncodeFunction()
	base64DecodeFunc := NewBase64DecodeFunction()

	originalData := []byte("Test roundtrip data with special chars: !@#$%^&*()")

	// Encode
	s.callStack.Push(&function.Call{
		FunctionName: "base64encode",
	})
	encodeOutput, err := base64EncodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				originalData,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	encodedString, ok := encodeOutput.ResponseData.(string)
	c.Assert(ok, Equals, true)

	// Decode
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	decodeOutput, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				encodedString,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(decodeOutput.ResponseData, DeepEquals, originalData)
}

func (s *Base64DecodeFunctionTestSuite) Test_propagates_none_value(c *C) {
	base64DecodeFunc := NewBase64DecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "base64decode",
	})
	output, err := base64DecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				core.GetNoneMarker(),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(core.IsNoneMarker(output.ResponseData), Equals, true)
}
