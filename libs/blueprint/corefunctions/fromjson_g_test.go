package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type FromJSON_G_FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&FromJSON_G_FunctionTestSuite{})

func (s *FromJSON_G_FunctionTestSuite) SetUpTest(c *C) {
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

func (s *FromJSON_G_FunctionTestSuite) Test_returns_function_runtime_info_with_partial_args(c *C) {
	fromJSON_G_Func := NewFromJSON_G_Function()
	s.callStack.Push(&function.Call{
		FunctionName: "fromjson_g",
	})
	output, err := fromJSON_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"/clusterConfig/hosts/1/hostname",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.FunctionInfo, DeepEquals, provider.FunctionRuntimeInfo{
		FunctionName: "fromjson",
		PartialArgs:  []any{"/clusterConfig/hosts/1/hostname"},
		ArgsOffset:   1,
	})
}
