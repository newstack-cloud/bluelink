package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type FromJSONFunctionTestSuite struct {
	callStack       function.Stack
	callContext     *functionCallContextMock
	rfcExamplesJSON string
}

var _ = Suite(&FromJSONFunctionTestSuite{})

func (s *FromJSONFunctionTestSuite) SetUpSuite(c *C) {
	s.rfcExamplesJSON = `
	{
		"foo": ["bar", "baz"],
		"": 0,
		"a/b": 1,
		"c%d": 2,
		"e^f": 3,
		"g|h": 4,
		"i\\j": 5,
		"k\"l": 6,
		" ": 7,
		"m~n": 8
	}
	`
}

func (s *FromJSONFunctionTestSuite) SetUpTest(c *C) {
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

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_1(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	docMap, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)
	c.Assert(docMap, DeepEquals, map[string]interface{}{
		"foo":  []interface{}{"bar", "baz"},
		"":     float64(0),
		"a/b":  float64(1),
		"c%d":  float64(2),
		"e^f":  float64(3),
		"g|h":  float64(4),
		"i\\j": float64(5),
		"k\"l": float64(6),
		" ":    float64(7),
		"m~n":  float64(8),
	})
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_2(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/foo",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	slice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(slice, DeepEquals, []interface{}{"bar", "baz"})
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_3(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/foo/0",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	str, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(str, Equals, "bar")
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_4(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(0))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_5(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/a~1b",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(1))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_6(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/c%d",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(2))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_7(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/e^f",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(3))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_8(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/g|h",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(4))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_9(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/i\\j",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(5))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_10(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/k\"l",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(6))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_11(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/ ",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(7))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_example_12(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"/m~0n",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(8))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_1(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	docMap, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)
	c.Assert(docMap, DeepEquals, map[string]interface{}{
		"foo":  []interface{}{"bar", "baz"},
		"":     float64(0),
		"a/b":  float64(1),
		"c%d":  float64(2),
		"e^f":  float64(3),
		"g|h":  float64(4),
		"i\\j": float64(5),
		"k\"l": float64(6),
		" ":    float64(7),
		"m~n":  float64(8),
	})
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_2(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/foo",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	slice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(slice, DeepEquals, []interface{}{"bar", "baz"})
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_3(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/foo/0",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	str, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(str, Equals, "bar")
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_4(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(0))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_5(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/a~1b",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(1))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_6(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/c%d",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(2))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_7(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/e^f",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(3))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_8(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/g|h",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(4))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_9(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/i\\j",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(5))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_10(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/k\"l",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(6))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_11(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/ ",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(7))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_uri_frag_json_pointer_example_12(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				s.rfcExamplesJSON,
				"#/m~0n",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	float, isFlaot := output.ResponseData.(float64)
	c.Assert(isFlaot, Equals, true)
	c.Assert(float, Equals, float64(8))
}

func (s *FromJSONFunctionTestSuite) Test_extracts_value_with_valid_json_pointer_for_nested_prop(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	output, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }}`,
				"/clusterConfig/hosts/1/hostname",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "example2.com")
}

func (s *FromJSONFunctionTestSuite) Test_returns_func_error_for_invalid_json_pointer_array_index_out_of_bounds(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	_, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }}`,
				// index 3 is out of bounds.
				"/clusterConfig/hosts/3/hostname",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "unable to extract value from json string using pointer \"/clusterConfig/hosts/3/hostname\"")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "fromjson",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *FromJSONFunctionTestSuite) Test_returns_func_error_for_invalid_json_pointer_invalid_map_key(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	_, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }}`,
				// hostIp does not exist in the host object.
				"/clusterConfig/hosts/1/hostIp",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "unable to extract value from json string using pointer \"/clusterConfig/hosts/1/hostIp\"")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "fromjson",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *FromJSONFunctionTestSuite) Test_returns_func_error_for_invalid_json_pointer_reached_leaf_node(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	_, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }}`,
				// hostname is a string, not an object.
				"/clusterConfig/hosts/1/hostname/tld",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "unable to extract value from json string using pointer \"/clusterConfig/hosts/1/hostname/tld\"")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "fromjson",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *FromJSONFunctionTestSuite) Test_returns_func_error_for_invalid_json_pointer_dash_index(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	_, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }}`,
				// "-" indicates index out of bounds as per the interpretation of the spec
				// being that "-" is the nonexistent element after the last in the array.
				"/clusterConfig/hosts/-/hostname",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "unable to extract value from json string using pointer \"/clusterConfig/hosts/-/hostname\"")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "fromjson",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *FromJSONFunctionTestSuite) Test_returns_func_error_for_invalid_json_string_input(c *C) {
	fromJSONFunc := NewFromJSONFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson",
	})
	_, err := fromJSONFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Missing closing "}".
				`{"clusterConfig": { "hosts": [{ "hostname": "example.com" }, { "hostname": "example2.com" }] }`,
				"/clusterConfig/hosts/0/hostname",
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
			FunctionName: "fromjson",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
