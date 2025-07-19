package corefunctions

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type DateTimeFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	clock       core.Clock
	suite.Suite
}

func (s *DateTimeFunctionTestSuite) SetupTest() {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
	s.clock = &mockclock.StaticClock{}
}

func (s *DateTimeFunctionTestSuite) Test_gets_current_time_unix_format() {
	dateTimeFunc := NewDateTimeFunction(s.clock)
	s.callStack.Push(&function.Call{
		FunctionName: "datetime",
	})
	output, err := dateTimeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"unix",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputStr, isStr := output.ResponseData.(string)
	s.Assert().True(isStr)
	s.Assert().Equal(fmt.Sprintf("%d", mockclock.CurrentTimeUnixMock), outputStr)
}

func (s *DateTimeFunctionTestSuite) Test_gets_current_time_rfc3339_format() {
	dateTimeFunc := NewDateTimeFunction(s.clock)
	s.callStack.Push(&function.Call{
		FunctionName: "datetime",
	})
	output, err := dateTimeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"rfc3339",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputStr, isStr := output.ResponseData.(string)
	s.Assert().True(isStr)
	s.Assert().Equal("2023-09-07T14:43:44Z", outputStr)
}

func (s *DateTimeFunctionTestSuite) Test_gets_current_time_tag_format() {
	dateTimeFunc := NewDateTimeFunction(s.clock)
	s.callStack.Push(&function.Call{
		FunctionName: "datetime",
	})
	output, err := dateTimeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"tag",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputStr, isStr := output.ResponseData.(string)
	s.Assert().True(isStr)
	s.Assert().Equal("2023-09-07--14-43-44", outputStr)
}

func (s *DateTimeFunctionTestSuite) Test_gets_current_time_tagcompact_format() {
	dateTimeFunc := NewDateTimeFunction(s.clock)
	s.callStack.Push(&function.Call{
		FunctionName: "datetime",
	})
	output, err := dateTimeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"tagcompact",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputStr, isStr := output.ResponseData.(string)
	s.Assert().True(isStr)
	s.Assert().Equal("20230907144344", outputStr)
}

func (s *DateTimeFunctionTestSuite) Test_returns_func_error_when_unsupported_format_is_provided() {
	dateTimeFunc := NewDateTimeFunction(s.clock)
	s.callStack.Push(&function.Call{
		FunctionName: "datetime",
	})
	_, err := dateTimeFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// rfc3339nano is not a supported format
				"rfc3339nano",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal(
		fmt.Sprintf(
			"the requested date/time format is not supported by the \"datetime\" function, "+
				"supported formats include: %s", strings.Join(SupportedDateTimeFormats, ", "),
		),
		funcErr.Message,
	)
	s.Assert().Equal(
		[]*function.Call{
			{
				FunctionName: "datetime",
			},
		},
		funcErr.CallStack,
	)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidInput, funcErr.Code)
}

func TestDateTimeFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(DateTimeFunctionTestSuite))
}
