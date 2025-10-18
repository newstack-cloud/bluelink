package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type CIDRSubnetFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&CIDRSubnetFunctionTestSuite{})

func (s *CIDRSubnetFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
		},
		callStack: s.callStack,
	}
}

func (s *CIDRSubnetFunctionTestSuite) Test_calculates_ipv4_subnet_basic(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	output, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"192.168.0.0/16",
				8,
				1,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "192.168.1.0/24")
}

func (s *CIDRSubnetFunctionTestSuite) Test_calculates_ipv4_subnet_with_larger_newbits(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	output, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"10.0.0.0/8",
				4,
				2,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "10.32.0.0/12")
}

func (s *CIDRSubnetFunctionTestSuite) Test_calculates_ipv4_subnet_first_subnet(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	output, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"192.168.0.0/16",
				8,
				0,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "192.168.0.0/24")
}

func (s *CIDRSubnetFunctionTestSuite) Test_calculates_ipv6_subnet(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	output, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"fd00::/64",
				8,
				1,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	// For /64 to /72 (adding 8 bits), netnum=1 sets bits 64-71 to 00000001
	// This affects the first 8 bits of the 5th hextet, making it 0x0100
	c.Assert(output.ResponseData, Equals, "fd00::100:0:0:0/72")
}

func (s *CIDRSubnetFunctionTestSuite) Test_returns_error_for_invalid_cidr(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	_, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"invalid-cidr",
				8,
				1,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, "invalid CIDR block.*")
}

func (s *CIDRSubnetFunctionTestSuite) Test_returns_error_for_negative_newbits(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	_, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"192.168.0.0/16",
				-1,
				1,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, "newbits must be non-negative.*")
}

func (s *CIDRSubnetFunctionTestSuite) Test_returns_error_for_netnum_out_of_range(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	_, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"192.168.0.0/16",
				8,
				256, // Only 256 subnets (0-255) are possible with newbits=8
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, "netnum .* out of range.*")
}

func (s *CIDRSubnetFunctionTestSuite) Test_returns_error_for_prefix_overflow(c *C) {
	cidrsubnetFunc := NewCIDRSubnetFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "cidrsubnet",
	})
	_, err := cidrsubnetFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"192.168.0.0/16",
				20, // 16 + 20 = 36, which exceeds 32 for IPv4
				1,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, "new prefix length .* exceeds maximum.*")
}
