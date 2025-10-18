package corefunctions

import (
	"context"
	"regexp"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type UUIDFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&UUIDFunctionTestSuite{})

func (s *UUIDFunctionTestSuite) SetUpTest(c *C) {
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

func (s *UUIDFunctionTestSuite) Test_generates_valid_uuid_v4(c *C) {
	uuidFunc := NewUUIDFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "uuid",
	})
	output, err := uuidFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)

	// Verify it's a valid UUID v4 format
	uuidStr, ok := output.ResponseData.(string)
	c.Assert(ok, Equals, true)

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// where y is one of [8, 9, a, b]
	uuidV4Regex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	c.Assert(uuidV4Regex.MatchString(uuidStr), Equals, true)
}

func (s *UUIDFunctionTestSuite) Test_generates_unique_uuids(c *C) {
	uuidFunc := NewUUIDFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "uuid",
	})

	// Generate two UUIDs
	output1, err := uuidFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	output2, err := uuidFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	// Verify they are different
	uuid1, _ := output1.ResponseData.(string)
	uuid2, _ := output2.ResponseData.(string)
	c.Assert(uuid1 == uuid2, Equals, false)
}
