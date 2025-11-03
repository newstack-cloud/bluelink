package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type SHA256FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&SHA256FunctionTestSuite{})

func (s *SHA256FunctionTestSuite) SetUpTest(c *C) {
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

func (s *SHA256FunctionTestSuite) Test_computes_sha256_hash_of_string(c *C) {
	sha256Func := NewSHA256Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha256",
	})
	output, err := sha256Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"Hello World"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-256 of "Hello World"
	c.Assert(output.ResponseData, Equals, "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e")
}

func (s *SHA256FunctionTestSuite) Test_computes_sha256_hash_of_bytes(c *C) {
	sha256Func := NewSHA256Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha256",
	})
	output, err := sha256Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("test data"),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-256 of "test data"
	c.Assert(output.ResponseData, Equals, "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9")
}

func (s *SHA256FunctionTestSuite) Test_computes_sha256_hash_of_empty_string(c *C) {
	sha256Func := NewSHA256Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha256",
	})
	output, err := sha256Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{""},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-256 of empty string
	c.Assert(output.ResponseData, Equals, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}

func (s *SHA256FunctionTestSuite) Test_returns_error_for_invalid_argument_type(c *C) {
	sha256Func := NewSHA256Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha256",
	})
	_, err := sha256Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{12345},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Matches, "input argument at index 0 must be a string or byte array.*")
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *SHA256FunctionTestSuite) Test_propagates_none_value(c *C) {
	sha256Func := NewSHA256Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha256",
	})
	output, err := sha256Func.Call(context.TODO(), &provider.FunctionCallInput{
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
