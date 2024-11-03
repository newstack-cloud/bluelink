package corefunctions

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/internal"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	. "gopkg.in/check.v1"
)

type LinkFunctionTestSuite struct {
	callStack           function.Stack
	callContext         *functionCallContextMock
	stateRetriever      *linkStateRetrieverMock
	instanceIDRetriever func(context.Context) (string, error)
}

var _ = Suite(&LinkFunctionTestSuite{})

func (s *LinkFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &blueprintParamsMock{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}

	resourceARN := "arn:aws:iam::123456789012:policy/test-execute-function-policy"
	awsLambdaHTTP := true
	awsLambdaHTTPMethod := "POST"
	awsLambdaHTTPPath := "/orders"
	s.stateRetriever = &linkStateRetrieverMock{
		linkState: map[string]state.LinkState{
			"test-blueprint-1::orderApi::createOrderFunction": {
				IntermediaryResourceStates: []*state.ResourceState{
					{
						ResourceID: "test-execute-function-policy",
						Status:     core.ResourceStatusCreated,
						ResourceSpecData: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"arn": {
									Literal: &core.ScalarValue{
										StringValue: &resourceARN,
									},
								},
							},
						},
						FailureReasons: []string{},
					},
				},
				LinkData: map[string]*core.MappingNode{
					"aws.lambda.http": {
						Literal: &core.ScalarValue{BoolValue: &awsLambdaHTTP},
					},
					"aws.lambda.http.method": {
						Literal: &core.ScalarValue{
							StringValue: &awsLambdaHTTPMethod,
						},
					},
					"aws.lambda.http.path": {
						Literal: &core.ScalarValue{
							StringValue: &awsLambdaHTTPPath,
						},
					},
				},
			},
		},
	}
	s.instanceIDRetriever = func(ctx context.Context) (string, error) {
		return "test-blueprint-1", nil
	}
}

func (s *LinkFunctionTestSuite) Test_gets_link_state(c *C) {
	linkFunc := NewLinkFunction(s.stateRetriever, s.instanceIDRetriever)
	s.callStack.Push(&function.Call{
		FunctionName: "link",
	})
	output, err := linkFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"orderApi",
				map[string]interface{}{
					"name": "createOrderFunction",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputState, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)

	resourceARN := "arn:aws:iam::123456789012:policy/test-execute-function-policy"
	isAWSLambdaHTTP := true
	awsLambdaHTTPMethod := "POST"
	awsLambdaHTTPPath := "/orders"
	c.Assert(outputState, DeepEquals, map[string]interface{}{
		"intermediaryResourceStates": []interface{}{
			map[string]interface{}{
				"resourceID": "test-execute-function-policy",
				"status":     core.ResourceStatusCreated,
				"resourceSpecData": &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"arn": {
							Literal: &core.ScalarValue{
								StringValue: &resourceARN,
							},
						},
					},
				},
				"failureReasons": []interface{}{},
			},
		},
		"linkData": map[string]*core.MappingNode{
			"aws.lambda.http": {
				Literal: &core.ScalarValue{BoolValue: &isAWSLambdaHTTP},
			},
			"aws.lambda.http.method": {
				Literal: &core.ScalarValue{
					StringValue: &awsLambdaHTTPMethod,
				},
			},
			"aws.lambda.http.path": {
				Literal: &core.ScalarValue{
					StringValue: &awsLambdaHTTPPath,
				},
			},
		},
	})
}

func (s *LinkFunctionTestSuite) Test_returns_func_error_for_missing_link_state(c *C) {
	linkFunc := NewLinkFunction(s.stateRetriever, s.instanceIDRetriever)
	s.callStack.Push(&function.Call{
		FunctionName: "link",
	})
	_, err := linkFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"orderApi",
				"listOrdersFunction",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "failed to retrieve link state for \"orderApi\" and \"listOrdersFunction\": link state not found")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "link",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeFunctionCall)
}

func (s *LinkFunctionTestSuite) Test_returns_func_error_for_invalid_resource_name_argument(c *C) {
	linkFunc := NewLinkFunction(s.stateRetriever, s.instanceIDRetriever)
	s.callStack.Push(&function.Call{
		FunctionName: "link",
	})
	_, err := linkFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Not a valid resource name or ref object.
				map[string]interface{}{
					"OTHER_NAME": "orderApi",
				},
				"listOrdersFunction",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument 0 must be a string or a resource reference.")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "link",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
