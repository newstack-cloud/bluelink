package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type SHA1FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&SHA1FunctionTestSuite{})

func (s *SHA1FunctionTestSuite) SetUpTest(c *C) {
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

func (s *SHA1FunctionTestSuite) Test_computes_sha1_hash_of_string(c *C) {
	sha1Func := NewSHA1Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha1",
	})
	output, err := sha1Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"Hello World"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-1 of "Hello World"
	c.Assert(output.ResponseData, Equals, "0a4d55a8d778e5022fab701977c5d840bbc486d0")
}

func (s *SHA1FunctionTestSuite) Test_computes_sha1_hash_of_bytes(c *C) {
	sha1Func := NewSHA1Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha1",
	})
	output, err := sha1Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("test data"),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-1 of "test data"
	c.Assert(output.ResponseData, Equals, "f48dd853820860816c75d54d0f584dc863327a7c")
}

func (s *SHA1FunctionTestSuite) Test_computes_sha1_hash_of_empty_string(c *C) {
	sha1Func := NewSHA1Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha1",
	})
	output, err := sha1Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{""},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// SHA-1 of empty string
	c.Assert(output.ResponseData, Equals, "da39a3ee5e6b4b0d3255bfef95601890afd80709")
}

func (s *SHA1FunctionTestSuite) Test_returns_error_for_invalid_argument_type(c *C) {
	sha1Func := NewSHA1Function()
	s.callStack.Push(&function.Call{
		FunctionName: "sha1",
	})
	_, err := sha1Func.Call(context.TODO(), &provider.FunctionCallInput{
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
