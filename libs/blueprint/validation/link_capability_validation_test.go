package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/require"
)

const testNetworkAttached = "test.vpc/network-attached"

func placementCapabilities(vpcName, functionName string) *LinkInstanceCapabilities {
	return &LinkInstanceCapabilities{
		LinkType:      "test/vpc::test/function",
		ResourceAName: vpcName,
		ResourceBName: functionName,
		Provides: []provider.LinkCapability{
			{Name: testNetworkAttached, Resource: provider.LinkPriorityResourceB},
		},
	}
}

func accessCapabilities(
	functionName string,
	targetName string,
	mustExist bool,
) *LinkInstanceCapabilities {
	return &LinkInstanceCapabilities{
		LinkType:      "test/function::test/queue",
		ResourceAName: functionName,
		ResourceBName: targetName,
		Requires: []provider.LinkCapability{
			{
				Name:      testNetworkAttached,
				Resource:  provider.LinkPriorityResourceA,
				MustExist: mustExist,
			},
		},
	}
}

// A link that cannot function without a capability is invalid when nothing in the
// blueprint establishes it, and saying so at validation time means the deployment fails
// before anything has been provisioned.
func Test_a_required_capability_with_no_provider_is_an_error(t *testing.T) {
	diagnostics := ValidateLinkCapabilities([]*LinkInstanceCapabilities{
		accessCapabilities("netFunction", "netQueue", true),
	})

	require.Len(t, diagnostics, 1)
	require.Equal(t, core.DiagnosticLevelError, diagnostics[0].Level)
	require.Contains(t, diagnostics[0].Message, testNetworkAttached)
	require.Contains(t, diagnostics[0].Message, "netFunction")
}

// The default, and the case that matters most: an access link for a function that was
// never placed in a VPC is perfectly valid, and must deploy rather than fail.
func Test_a_required_capability_without_must_exist_is_satisfied_by_absence(t *testing.T) {
	diagnostics := ValidateLinkCapabilities([]*LinkInstanceCapabilities{
		accessCapabilities("netFunction", "netQueue", false),
	})

	require.Empty(t, diagnostics)
}

func Test_a_required_capability_with_a_provider_is_valid(t *testing.T) {
	diagnostics := ValidateLinkCapabilities([]*LinkInstanceCapabilities{
		placementCapabilities("netVPC", "netFunction"),
		accessCapabilities("netFunction", "netQueue", true),
	})

	require.Empty(t, diagnostics)
}

// Presence is judged per resource instance, so a placement link for a different function
// does not satisfy this function's requirement.
func Test_a_provider_on_another_resource_does_not_satisfy_the_requirement(t *testing.T) {
	diagnostics := ValidateLinkCapabilities([]*LinkInstanceCapabilities{
		placementCapabilities("netVPC", "otherFunction"),
		accessCapabilities("netFunction", "netQueue", true),
	})

	require.Len(t, diagnostics, 1)
	require.Contains(t, diagnostics[0].Message, "netFunction")
}
