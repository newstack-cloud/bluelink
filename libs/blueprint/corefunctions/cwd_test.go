package corefunctions

import (
	"context"
	"fmt"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type CWDFunctionTestSuite struct {
	callStack         function.Stack
	callContext       *functionCallContextMock
	getWorkingDir     WorkingDirResolver
	getWorkingDirFail WorkingDirResolver
	suite.Suite
}

func (s *CWDFunctionTestSuite) SetupTest() {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
	s.getWorkingDir = func() (string, error) {
		return "/home/user", nil
	}
	s.getWorkingDirFail = func() (string, error) {
		return "", fmt.Errorf("failed to resolve working directory")
	}
}

func (s *CWDFunctionTestSuite) Test_gets_current_working_directory() {
	cwdFunc := NewCWDFunction(s.getWorkingDir)
	s.callStack.Push(&function.Call{
		FunctionName: "cwd",
	})
	output, err := cwdFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputStr, isStr := output.ResponseData.(string)
	s.Assert().True(isStr)
	s.Assert().Equal("/home/user", outputStr)
}

func (s *CWDFunctionTestSuite) Test_returns_func_error_on_failure_to_get_current_working_directory() {
	cwdFunc := NewCWDFunction(s.getWorkingDirFail)
	s.callStack.Push(&function.Call{
		FunctionName: "cwd",
	})
	_, err := cwdFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal(
		"unable to get current working directory: failed to resolve working directory",
		funcErr.Message,
	)
	s.Assert().Equal(
		[]*function.Call{
			{
				FunctionName: "cwd",
			},
		},
		funcErr.CallStack,
	)
	s.Assert().Equal(function.FuncCallErrorCodeFunctionCall, funcErr.Code)
}

func TestCWDFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(CWDFunctionTestSuite))
}
