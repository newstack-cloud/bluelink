package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type MD5FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&MD5FunctionTestSuite{})

func (s *MD5FunctionTestSuite) SetUpTest(c *C) {
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

func (s *MD5FunctionTestSuite) Test_computes_md5_hash_of_string(c *C) {
	md5Func := NewMD5Function()
	s.callStack.Push(&function.Call{
		FunctionName: "md5",
	})
	output, err := md5Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"Hello World"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// MD5 of "Hello World"
	c.Assert(output.ResponseData, Equals, "b10a8db164e0754105b7a99be72e3fe5")
}

func (s *MD5FunctionTestSuite) Test_computes_md5_hash_of_bytes(c *C) {
	md5Func := NewMD5Function()
	s.callStack.Push(&function.Call{
		FunctionName: "md5",
	})
	output, err := md5Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]byte("test data"),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// MD5 of "test data"
	c.Assert(output.ResponseData, Equals, "eb733a00c0c9d336e65691a37ab54293")
}

func (s *MD5FunctionTestSuite) Test_computes_md5_hash_of_empty_string(c *C) {
	md5Func := NewMD5Function()
	s.callStack.Push(&function.Call{
		FunctionName: "md5",
	})
	output, err := md5Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{""},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// MD5 of empty string
	c.Assert(output.ResponseData, Equals, "d41d8cd98f00b204e9800998ecf8427e")
}

func (s *MD5FunctionTestSuite) Test_returns_error_for_invalid_argument_type(c *C) {
	md5Func := NewMD5Function()
	s.callStack.Push(&function.Call{
		FunctionName: "md5",
	})
	_, err := md5Func.Call(context.TODO(), &provider.FunctionCallInput{
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
