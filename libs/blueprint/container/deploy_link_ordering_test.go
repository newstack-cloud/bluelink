package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/require"
)

// Two links that share a resource must not be deployed in either order.
//
// The shape is the one the AWS flex VPC stack has: a VPC places a function, and the same
// function reaches a queue. The placement link sets the function's vpcConfig; the access
// link reads that live config to decide whether the function needs a VPC endpoint, and
// silently creates nothing if the function is not attached yet. Placement therefore has to
// be deployed first, or the deployment succeeds having built a function that cannot reach
// the queue at runtime.
//
//	netVPC ──places──► netFunction ──accesses──► netQueue
//
// The placement link declares that it provides "network-attached" on the function, and the
// access link declares that it requires it. That declaration is the whole of the signal;
// nothing is inferred from which side of either link the function sits on.
//
// The order links are reported ready in is timing-dependent: it follows whichever of
// netVPC and netQueue finished deploying first.
func linkOrderingChain() (vpc, function, queue *links.ChainLinkNode) {
	vpc = &links.ChainLinkNode{ResourceName: "netVPC"}
	function = &links.ChainLinkNode{ResourceName: "netFunction"}
	queue = &links.ChainLinkNode{ResourceName: "netQueue"}

	vpc.LinksTo = []*links.ChainLinkNode{function}
	function.LinkedFrom = []*links.ChainLinkNode{vpc}
	function.LinksTo = []*links.ChainLinkNode{queue}
	queue.LinkedFrom = []*links.ChainLinkNode{function}

	return vpc, function, queue
}

// The graph the chain above resolves to, built the way a deployment builds it so the
// declarations and the ordering they produce are exercised together.
func placementOrderingGraph(t *testing.T) *LinkCapabilityGraph {
	t.Helper()

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		capabilityChain(t, map[[2]string]*capabilityTestLink{
			{"netVPC", "netFunction"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
			{"netFunction", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
		}),
		capabilityChangeSet(
			[2]string{"netVPC", "netFunction"},
			[2]string{"netFunction", "netQueue"},
		),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	return graph
}

// A graph for links that declare nothing, so nothing is ordered.
func unorderedGraph(t *testing.T) *LinkCapabilityGraph {
	t.Helper()

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		capabilityChain(t, map[[2]string]*capabilityTestLink{
			{"netFunction", "netQueue"}: {},
		}),
		capabilityChangeSet([2]string{"netFunction", "netQueue"}),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)
	require.True(t, graph.Empty(), "nothing was declared, so nothing should be ordered")

	return graph
}

// A graph in which the access link waits for two placement links, built the same way.
func twoProviderOrderingGraph(t *testing.T) *LinkCapabilityGraph {
	t.Helper()

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		capabilityChain(t, map[[2]string]*capabilityTestLink{
			{"netVPC", "netFunction"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
			{"netVPC2", "netFunction"}:  providesNetworkAttached(provider.LinkPriorityResourceB),
			{"netFunction", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
		}),
		capabilityChangeSet(
			[2]string{"netVPC", "netFunction"},
			[2]string{"netVPC2", "netFunction"},
			[2]string{"netFunction", "netQueue"},
		),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	return graph
}

// Both links become ready in the same batch. The consumer has to wait even though it is
// sitting right next to the providing link in the same slice.
func Test_access_link_waits_for_the_placement_link_in_the_same_batch(t *testing.T) {
	vpc, function, queue := linkOrderingChain()
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkCapabilityGraph(placementOrderingGraph(t))

	require.Empty(t, deployState.UpdateLinkDeploymentState(queue))
	require.Empty(t, deployState.UpdateLinkDeploymentState(vpc))
	ready := deployState.UpdateLinkDeploymentState(function)
	require.Len(t, ready, 2)

	require.Empty(
		t,
		deployState.AwaitingCapabilityProviders("netVPC::netFunction"),
		"the placement link requires nothing, so it should never wait",
	)
	require.Equal(
		t,
		[]string{"netVPC::netFunction"},
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"the access link should wait while the placement link is still undeployed",
	)

	deployState.MarkLinkSettled("netVPC::netFunction")

	require.Empty(
		t,
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"the access link should be released once the placement link has been deployed",
	)
}

// A case that failed against a real AWS account. The queue finished after the function,
// so the access link arrived in a batch of its own, running concurrently with the batch
// holding the placement link.
//
// Waiting is keyed on whether the provider has finished deploying rather than on whether
// it happens to be ready, so a provider that is not even ready yet still holds its
// consumers back.
func Test_access_link_waits_for_a_placement_link_in_another_batch(t *testing.T) {
	vpc, function, queue := linkOrderingChain()
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkCapabilityGraph(placementOrderingGraph(t))

	require.Empty(t, deployState.UpdateLinkDeploymentState(vpc))

	placementBatch := deployState.UpdateLinkDeploymentState(function)
	require.Len(t, placementBatch, 1, "only the placement link should be ready")

	accessBatch := deployState.UpdateLinkDeploymentState(queue)
	require.Len(t, accessBatch, 1, "only the access link should be ready")

	require.NotEmpty(
		t,
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"the access link should wait for a placement link its own batch does not hold",
	)

	deployState.MarkLinkSettled("netVPC::netFunction")

	require.Empty(t, deployState.AwaitingCapabilityProviders("netFunction::netQueue"))
}

// A function that was never placed in a VPC has no placement link, so the requirement its
// access links declare matches no provider. They must deploy immediately rather than wait
// for something that does not exist, which is why a requirement is satisfied by absence
// unless it is explicitly marked as one the link cannot function without.
func Test_a_requirement_nothing_provides_does_not_hold_a_link_back(t *testing.T) {
	_, function, queue := linkOrderingChain()
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkCapabilityGraph(unorderedGraph(t))

	require.Empty(t, deployState.UpdateLinkDeploymentState(function))
	ready := deployState.UpdateLinkDeploymentState(queue)
	require.Len(t, ready, 1)

	require.Empty(
		t,
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"an access link for an unplaced function has nothing to wait for",
	)
}

// A deployment in which no link declares anything is the common case, and must not be
// ordered at all.
func Test_links_are_unordered_when_no_capabilities_are_declared(t *testing.T) {
	vpc, function, queue := linkOrderingChain()
	deployState := NewDefaultDeploymentState()

	require.Empty(t, deployState.UpdateLinkDeploymentState(queue))
	require.Empty(t, deployState.UpdateLinkDeploymentState(vpc))
	ready := deployState.UpdateLinkDeploymentState(function)
	require.Len(t, ready, 2)

	for _, linkName := range []string{"netVPC::netFunction", "netFunction::netQueue"} {
		require.Emptyf(
			t,
			deployState.AwaitingCapabilityProviders(linkName),
			"%s declared nothing and should not wait", linkName,
		)
	}
}

// Two links between the same pair of resources, pointing in opposite directions.
//
// This is an ordinary blueprint with a function that writes to
// a table and is also triggered by that table's stream produces both
// "function -> table" and "table -> function", and the AWS provider registers a link for
// each.
//
// Under the previous positional rule this pair deadlocked, because each link held the other's
// A-side resource on its B side and position alone said they configured each other.
// Neither declares a capability, so there is nothing here to order and no stall to guard
// against.
func Test_opposing_links_between_the_same_pair_do_not_wait_on_each_other(t *testing.T) {
	function := &links.ChainLinkNode{ResourceName: "appFunction"}
	table := &links.ChainLinkNode{ResourceName: "appTable"}

	function.LinksTo = []*links.ChainLinkNode{table}
	table.LinkedFrom = []*links.ChainLinkNode{function}
	table.LinksTo = []*links.ChainLinkNode{function}
	function.LinkedFrom = []*links.ChainLinkNode{table}

	deployState := NewDefaultDeploymentState()
	require.Empty(t, deployState.UpdateLinkDeploymentState(function))
	ready := deployState.UpdateLinkDeploymentState(table)
	require.Len(t, ready, 2, "both directions should become ready together")

	for _, linkName := range []string{"appFunction::appTable", "appTable::appFunction"} {
		require.Emptyf(
			t,
			deployState.AwaitingCapabilityProviders(linkName),
			"%s should be free to deploy", linkName,
		)
	}
}

// A link with several providers is released only by the last of them, not the first.
func Test_a_link_waits_for_every_provider_of_the_capabilities_it_requires(t *testing.T) {
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkCapabilityGraph(twoProviderOrderingGraph(t))

	deployState.MarkLinkSettled("netVPC::netFunction")
	require.Equal(
		t,
		[]string{"netVPC2::netFunction"},
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
	)

	deployState.MarkLinkSettled("netVPC2::netFunction")
	require.Empty(t, deployState.AwaitingCapabilityProviders("netFunction::netQueue"))
}
