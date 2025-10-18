package corefunctions

import (
	"context"
	"os"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type FileFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	tempDir     string
}

var _ = Suite(&FileFunctionTestSuite{})

func (s *FileFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "file-function-test-*")
	c.Assert(err, IsNil)
	s.tempDir = tempDir
}

func (s *FileFunctionTestSuite) TearDownTest(c *C) {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *FileFunctionTestSuite) Test_reads_text_file(c *C) {
	// Create test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	content := []byte("Hello, World!")
	err := os.WriteFile(testFile, content, 0644)
	c.Assert(err, IsNil)

	fileFunc := NewFileFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "file",
	})
	output, err := fileFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{testFile},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, content)
}

func (s *FileFunctionTestSuite) Test_reads_binary_file(c *C) {
	// Create test binary file
	testFile := filepath.Join(s.tempDir, "test.bin")
	content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	err := os.WriteFile(testFile, content, 0644)
	c.Assert(err, IsNil)

	fileFunc := NewFileFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "file",
	})
	output, err := fileFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{testFile},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, content)
}

func (s *FileFunctionTestSuite) Test_reads_empty_file(c *C) {
	// Create empty test file
	testFile := filepath.Join(s.tempDir, "empty.txt")
	err := os.WriteFile(testFile, []byte{}, 0644)
	c.Assert(err, IsNil)

	fileFunc := NewFileFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "file",
	})
	output, err := fileFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{testFile},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, []byte{})
}

func (s *FileFunctionTestSuite) Test_returns_error_for_nonexistent_file(c *C) {
	fileFunc := NewFileFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "file",
	})
	_, err := fileFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"/nonexistent/path/to/file.txt"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Matches, "unable to read file at path.*")
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeFunctionCall)
}

func (s *FileFunctionTestSuite) Test_returns_error_for_invalid_argument_type(c *C) {
	fileFunc := NewFileFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "file",
	})
	_, err := fileFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{12345},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 0 is of type int, but target is of type string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "file",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
