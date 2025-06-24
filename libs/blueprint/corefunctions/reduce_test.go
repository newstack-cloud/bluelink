package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type ReduceFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&ReduceFunctionTestSuite{})

func (s *ReduceFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"extract_crucial_network_info":         newExtractCrucialNetworkInfoFunction(),
				"invalid_extract_crucial_network_info": newInvalidExtractCrucialNetworkInfoFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *ReduceFunctionTestSuite) Test_reduces_list_to_crucial_network_info(c *C) {

	filterFunc := NewReduceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "filter",
	})
	output, err := filterFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// initial.
				map[string]interface{}{
					"allowedAddresses": []interface{}{},
				},
				// network list.
				[]interface{}{
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"178.168.1.3",
						},
					},
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"178.255.0.255",
							"192.168.0.2",
							"192.168.0.9",
							"192.168.0.4",
							"178.168.0.3",
						},
					},
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"178.255.0.128",
							"192.168.0.2",
							"192.168.0.9",
							"192.168.0.4",
							"178.255.0.255",
						},
					},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "extract_crucial_network_info",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputMap, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)
	mergedAllowedAddresses, isSlice := outputMap["allowedAddresses"].([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(mergedAllowedAddresses, DeepEquals, []interface{}{
		"192.168.0.1",
		"192.168.0.2",
		"192.168.0.3",
		"192.168.0.4",
		"178.168.1.3",
		"178.255.0.255",
		"192.168.0.9",
		"178.168.0.3",
		"178.255.0.128",
	})
}

func (s *ReduceFunctionTestSuite) Test_returns_func_error_for_invalid_item_in_list(c *C) {
	reduceFunc := NewReduceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "reduce",
	})
	_, err := reduceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// initial.
				map[string]interface{}{
					"allowedAddresses": []interface{}{},
				},
				// Network list.
				[]interface{}{
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"178.168.1.3",
						},
					},

					// 5 is not a valid object, the extract_crucial_network_info function should return an error.
					5,
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "extract_crucial_network_info",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 1 is of type int, but target is of type map")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "extract_crucial_network_info",
		},
		{
			FunctionName: "reduce",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *ReduceFunctionTestSuite) Test_returns_func_error_for_invalid_args_offset(c *C) {
	reduceFunc := NewReduceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "reduce",
	})
	_, err := reduceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// initial.
				map[string]interface{}{
					"allowedAddresses": []interface{}{},
				},
				// Network list.
				[]interface{}{
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"178.168.1.3",
						},
					},
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.1.0",
							"192.168.2.0",
							"192.168.3.0",
							"192.168.4.0",
							"178.168.1.3",
						},
					},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "extract_crucial_network_info",
					PartialArgs:  []any{},
					// 20 is not a valid args offset for the extract_crucial_network_info function.
					ArgsOffset: 20,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(
		funcErr.Message,
		Equals,
		"invalid args offset defined for the partially applied \"extract_crucial_network_info\""+
			" function, this is an issue with the function used to create the function value passed into reduce",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "reduce",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgsOffset)
}

func (s *ReduceFunctionTestSuite) Test_returns_func_error_for_invalid_return_value(c *C) {
	reduceFunc := NewReduceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "reduce",
	})
	_, err := reduceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// initial.
				map[string]interface{}{
					"allowedAddresses": []interface{}{},
				},
				// Network list.
				[]interface{}{
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"178.168.1.3",
						},
					},
					map[string]interface{}{
						"allowedAddresses": []interface{}{
							"192.168.1.0",
							"192.168.2.0",
							"192.168.3.0",
							"192.168.4.0",
							"178.168.1.3",
						},
					},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "invalid_extract_crucial_network_info",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(
		funcErr.Message,
		Equals,
		"expected the \"invalid_extract_crucial_network_info\" reducer function to return a value of type "+
			"map[string]interface {}, but got a value of type int",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "reduce",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidReturnType)
}

func newExtractCrucialNetworkInfoFunction() provider.Function {
	return &extractCrucialNetworkInfoFunction{}
}

type extractCrucialNetworkInfoFunction struct{}

func (e *extractCrucialNetworkInfoFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		// Definition is an empty stub as it is not used in the tests.
		Definition: &function.Definition{},
	}, nil
}

func (e *extractCrucialNetworkInfoFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var crucialNetworkInfo map[string]interface{}
	// Network is expected to be a simple object that contains an allow list
	// of IP addresses that should be combined in the final crucial network info
	// object.
	var currentNetwork map[string]interface{}
	if err := input.Arguments.GetMultipleVars(ctx, &crucialNetworkInfo, &currentNetwork); err != nil {
		return nil, err
	}

	currentAllowedIPs, ok := currentNetwork["allowedAddresses"].([]interface{})
	if !ok {
		return &provider.FunctionCallOutput{
			ResponseData: crucialNetworkInfo,
		}, nil
	}

	accumAllowedIPs, ok := crucialNetworkInfo["allowedAddresses"].([]interface{})
	if !ok {
		return nil, function.NewFuncCallError(
			"allowedAddresses in the accumulated crucial network info"+
				" must be set and must be a list",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	for _, ip := range currentAllowedIPs {
		if ipStr, ok := ip.(string); ok {
			if !contains(accumAllowedIPs, ipStr) {
				crucialNetworkInfo["allowedAddresses"] = append(
					crucialNetworkInfo["allowedAddresses"].([]interface{}),
					ipStr,
				)
			}
		}
	}

	return &provider.FunctionCallOutput{
		ResponseData: crucialNetworkInfo,
	}, nil
}

func newInvalidExtractCrucialNetworkInfoFunction() provider.Function {
	return &invalidExtractCrucialNetworkInfoFunction{}
}

type invalidExtractCrucialNetworkInfoFunction struct{}

func (e *invalidExtractCrucialNetworkInfoFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		// Definition is an empty stub as it is not used in the tests.
		Definition: &function.Definition{},
	}, nil
}

func (e *invalidExtractCrucialNetworkInfoFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	return &provider.FunctionCallOutput{
		// Integer is not a valid return type for the reduce function in the tests.
		ResponseData: 1024,
	}, nil
}

func contains[Type comparable](slice []interface{}, item Type) bool {
	for _, sliceItem := range slice {
		compareWith, isType := sliceItem.(Type)
		if isType && compareWith == item {
			return true
		}
	}
	return false
}
