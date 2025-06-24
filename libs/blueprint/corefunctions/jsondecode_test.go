package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type JSONDecodeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&JSONDecodeFunctionTestSuite{})

func (s *JSONDecodeFunctionTestSuite) SetUpTest(c *C) {
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

func (s *JSONDecodeFunctionTestSuite) Test_decodes_json_object(c *C) {
	jsonDecodeFunc := NewJSONDecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "jsondecode",
	})
	output, err := jsonDecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"host": "example.com"}`,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputMap, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)
	c.Assert(outputMap, DeepEquals, map[string]interface{}{
		"host": "example.com",
	})
}

func (s *JSONDecodeFunctionTestSuite) Test_decodes_json_primitive(c *C) {
	jsonDecodeFunc := NewJSONDecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "jsondecode",
	})
	output, err := jsonDecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`"This is a string"`,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "This is a string")
}

func (s *JSONDecodeFunctionTestSuite) Test_decodes_json_array(c *C) {
	jsonDecodeFunc := NewJSONDecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "jsondecode",
	})
	output, err := jsonDecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`[{"host": "example1.com"}, {"host": "example2.com"}]`,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(outputSlice, DeepEquals, []interface{}{
		map[string]interface{}{
			"host": "example1.com",
		},
		map[string]interface{}{
			"host": "example2.com",
		},
	})
}

func (s *JSONDecodeFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	jsonDecodeFunc := NewJSONDecodeFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "jsondecode",
	})
	_, err := jsonDecodeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Invalid json string, missing closing object brace.
				`{"host": "example.com"`,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "unable to decode json string: unexpected end of JSON input")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "jsondecode",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
