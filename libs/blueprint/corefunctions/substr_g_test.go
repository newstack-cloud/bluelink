package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type Substr_G_FunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&Substr_G_FunctionTestSuite{})

func (s *Substr_G_FunctionTestSuite) SetUpTest(c *C) {
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

func (s *Substr_G_FunctionTestSuite) Test_returns_function_runtime_info_with_partial_args_with_end(c *C) {
	Substr_G_Func := NewSubstr_G_Function()
	s.callStack.Push(&function.Call{
		FunctionName: "substr_g",
	})
	output, err := Substr_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(10),
				int64(15),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.FunctionInfo, DeepEquals, provider.FunctionRuntimeInfo{
		FunctionName: "substr",
		PartialArgs:  []any{int64(10), int64(15)},
		ArgsOffset:   1,
	})
}

func (s *Substr_G_FunctionTestSuite) Test_returns_function_runtime_info_with_partial_args_without_end(c *C) {
	Substr_G_Func := NewSubstr_G_Function()
	s.callStack.Push(&function.Call{
		FunctionName: "substr_g",
	})
	output, err := Substr_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				int64(20),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.FunctionInfo, DeepEquals, provider.FunctionRuntimeInfo{
		FunctionName: "substr",
		PartialArgs:  []any{int64(20)},
		ArgsOffset:   1,
	})
}
