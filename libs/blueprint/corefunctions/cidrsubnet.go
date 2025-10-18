package corefunctions

import (
	"context"
	"fmt"
	"math/big"
	"net"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// CIDRSubnetFunction provides the implementation of
// a function that calculates subnet addresses within a given CIDR block.
type CIDRSubnetFunction struct {
	definition *function.Definition
}

// NewCIDRSubnetFunction creates a new instance of the CIDRSubnetFunction with
// a complete function definition.
func NewCIDRSubnetFunction() provider.Function {
	return &CIDRSubnetFunction{
		definition: &function.Definition{
			Description: "A function that calculates subnet addresses within a given CIDR block.",
			FormattedDescription: "A function that calculates subnet addresses within a given CIDR block.\n\n" +
				"**Examples:**\n\n" +
				"Calculating subnets:\n" +
				"```\n${cidrsubnet(\"192.168.0.0/16\", 8, 1)}  # Returns \"192.168.1.0/24\"\n" +
				"${cidrsubnet(\"10.0.0.0/8\", 4, 2)}      # Returns \"10.2.0.0/12\"\n```\n\n" +
				"Dynamic subnet allocation:\n" +
				"```\n${cidrsubnet(variables.vpcCidr, 4, variables.subnetIndex)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "prefix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The CIDR block to calculate subnets within (e.g., \"192.168.0.0/16\").",
				},
				&function.ScalarParameter{
					Label: "newbits",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Description: "The number of additional bits to add to the prefix length.",
				},
				&function.ScalarParameter{
					Label: "netnum",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Description: "The subnet number to calculate (0-based index).",
				},
			},
			Return: &function.ValueTypeDefinitionScalar{
				Type: function.ValueTypeString,
			},
		},
	}
}

func (f *CIDRSubnetFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *CIDRSubnetFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var prefix string
	if err := input.Arguments.GetVar(ctx, 0, &prefix); err != nil {
		return nil, err
	}

	var newbits int
	if err := input.Arguments.GetVar(ctx, 1, &newbits); err != nil {
		return nil, err
	}

	var netnum int
	if err := input.Arguments.GetVar(ctx, 2, &netnum); err != nil {
		return nil, err
	}

	// Parse the CIDR block
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR block %q: %w", prefix, err)
	}

	// Get the current prefix length
	prefixLen, _ := ipNet.Mask.Size()
	newPrefixLen := prefixLen + newbits

	// Validate newbits
	if newbits < 0 {
		return nil, fmt.Errorf("newbits must be non-negative, got %d", newbits)
	}

	// Determine if this is an IPv4 or IPv6 address
	ipv4 := ipNet.IP.To4() != nil
	maxBits := 32
	if !ipv4 {
		maxBits = 128
	}

	if newPrefixLen > maxBits {
		return nil, fmt.Errorf("new prefix length %d exceeds maximum of %d for %s", newPrefixLen, maxBits, map[bool]string{true: "IPv4", false: "IPv6"}[ipv4])
	}

	// Calculate the maximum number of subnets
	maxSubnets := 1 << newbits
	if netnum < 0 || netnum >= maxSubnets {
		return nil, fmt.Errorf("netnum %d out of range (must be 0-%d)", netnum, maxSubnets-1)
	}

	// Convert IP to big.Int for manipulation
	ip := ipNet.IP
	if ipv4 {
		ip = ip.To4()
	} else {
		ip = ip.To16()
	}

	ipInt := new(big.Int).SetBytes(ip)

	// Calculate the subnet increment
	// The increment is the number of addresses per subnet (2^(maxBits-newPrefixLen))
	// We need to add netnum * increment to the base network address
	bitsToShift := maxBits - newPrefixLen
	increment := new(big.Int).Lsh(big.NewInt(1), uint(bitsToShift))
	offset := new(big.Int).Mul(big.NewInt(int64(netnum)), increment)

	// Add the offset to get the new subnet address
	newIPInt := new(big.Int).Add(ipInt, offset)

	// Convert back to IP
	var newIP net.IP
	ipBytes := newIPInt.Bytes()

	if ipv4 {
		// For IPv4, ensure we have exactly 4 bytes
		newIP = make(net.IP, 4)
		if len(ipBytes) <= 4 {
			copy(newIP[4-len(ipBytes):], ipBytes)
		} else {
			return nil, fmt.Errorf("calculated IP address exceeds IPv4 range")
		}
	} else {
		// For IPv6, ensure we have exactly 16 bytes
		newIP = make(net.IP, 16)
		if len(ipBytes) <= 16 {
			copy(newIP[16-len(ipBytes):], ipBytes)
		} else {
			return nil, fmt.Errorf("calculated IP address exceeds IPv6 range")
		}
	}

	// Create the new CIDR notation
	result := fmt.Sprintf("%s/%d", newIP.String(), newPrefixLen)

	return &provider.FunctionCallOutput{
		ResponseData: result,
	}, nil
}
