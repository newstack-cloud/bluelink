package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/require"
)

const testNetworkAttached = "test.vpc/network-attached"

// A link that declares nothing but the capabilities under test. Everything else on the
// interface is ignored, since building the capability graph never reaches it.
type capabilityTestLink struct {
	provides []provider.LinkCapability
	requires []provider.LinkCapability
}

func (l *capabilityTestLink) GetCapabilities(
	ctx context.Context,
	input *provider.LinkGetCapabilitiesInput,
) (*provider.LinkGetCapabilitiesOutput, error) {
	return &provider.LinkGetCapabilitiesOutput{
		Provides: l.provides,
		Requires: l.requires,
	}, nil
}

func (l *capabilityTestLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *capabilityTestLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	return &provider.LinkProduceResourceContributionsOutput{}, nil
}

func (l *capabilityTestLink) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	return &provider.LinkUpdateLinkedResourcesOutput{}, nil
}

func (l *capabilityTestLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *capabilityTestLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{}, nil
}

func (l *capabilityTestLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{}, nil
}

func (l *capabilityTestLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *capabilityTestLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{}, nil
}

func (l *capabilityTestLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{Kind: provider.LinkKindSoft}, nil
}

func (l *capabilityTestLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *capabilityTestLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{}, nil
}

func (l *capabilityTestLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}

// Wires a set of "from -> to" links into chain nodes, attaching the given
// link implementation to each, and returns the deployment nodes to build a graph from.
func capabilityChain(
	t *testing.T,
	linkImpls map[[2]string]*capabilityTestLink,
) []*DeploymentNode {
	t.Helper()

	nodes := map[string]*links.ChainLinkNode{}
	nodeFor := func(name string) *links.ChainLinkNode {
		if node, exists := nodes[name]; exists {
			return node
		}
		node := &links.ChainLinkNode{
			ResourceName:        name,
			LinkImplementations: map[string]provider.Link{},
		}
		nodes[name] = node
		return node
	}

	for pair, impl := range linkImpls {
		from, to := nodeFor(pair[0]), nodeFor(pair[1])
		from.LinksTo = append(from.LinksTo, to)
		to.LinkedFrom = append(to.LinkedFrom, from)
		from.LinkImplementations[pair[1]] = impl
	}

	deploymentNodes := []*DeploymentNode{}
	for _, node := range nodes {
		deploymentNodes = append(deploymentNodes, &DeploymentNode{ChainLinkNode: node})
	}

	return deploymentNodes
}

// A change set in which every one of the given links is being created.
func capabilityChangeSet(linkPairs ...[2]string) *changes.BlueprintChanges {
	newResources := map[string]provider.Changes{}
	for _, pair := range linkPairs {
		resourceChanges, exists := newResources[pair[0]]
		if !exists {
			resourceChanges = provider.Changes{
				NewOutboundLinks: map[string]provider.LinkChanges{},
			}
		}
		resourceChanges.NewOutboundLinks[pair[1]] = provider.LinkChanges{}
		newResources[pair[0]] = resourceChanges
	}

	return &changes.BlueprintChanges{NewResources: newResources}
}

func providesNetworkAttached(on provider.LinkPriorityResource) *capabilityTestLink {
	return &capabilityTestLink{
		provides: []provider.LinkCapability{
			{Name: testNetworkAttached, Resource: on},
		},
	}
}

func requiresNetworkAttached(on provider.LinkPriorityResource) *capabilityTestLink {
	return &capabilityTestLink{
		requires: []provider.LinkCapability{
			{Name: testNetworkAttached, Resource: on},
		},
	}
}

// The flex VPC shape: placement provides "network-attached" on the function it places,
// and the access link requires it on the function it grants access from.
func Test_capability_graph_orders_an_access_link_after_the_placement_link(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"netVPC", "netFunction"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
		{"netFunction", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
	})

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet([2]string{"netVPC", "netFunction"}, [2]string{"netFunction", "netQueue"}),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	require.Equal(
		t,
		[]string{"netVPC::netFunction"},
		graph.WaitsFor("netFunction::netQueue"),
	)
	require.Empty(t, graph.WaitsFor("netVPC::netFunction"))
}

// Capabilities are matched on the resource instance, not the resource type, so a second
// function placed by a second link does not order the first function's access link.
func Test_capability_graph_matches_on_the_resource_instance(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"netVPC", "functionOne"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
		{"netVPC", "functionTwo"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
		{"functionOne", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
	})

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet(
			[2]string{"netVPC", "functionOne"},
			[2]string{"netVPC", "functionTwo"},
			[2]string{"functionOne", "netQueue"},
		),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	require.Equal(
		t,
		[]string{"netVPC::functionOne"},
		graph.WaitsFor("functionOne::netQueue"),
		"only the link that placed this function should order its access link",
	)
}

// A function that was never placed in a VPC has no placement link at all. Its access link
// declares the requirement all the same, matches no provider, and must deploy immediately.
func Test_capability_graph_leaves_a_requirement_with_no_provider_unordered(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"netFunction", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
	})

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet([2]string{"netFunction", "netQueue"}),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	require.True(t, graph.Empty())
	require.Empty(t, graph.WaitsFor("netFunction::netQueue"))
}

// A provider left out of the change set is treated as already having established its
// guarantee on a previous deployment. Creating an edge to it would leave the requirer
// waiting for a link that never runs, which hangs the deployment rather than failing it.
func Test_capability_graph_creates_no_edge_to_a_provider_outside_the_change_set(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"netVPC", "netFunction"}:   providesNetworkAttached(provider.LinkPriorityResourceB),
		{"netFunction", "netQueue"}: requiresNetworkAttached(provider.LinkPriorityResourceA),
	})

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet([2]string{"netFunction", "netQueue"}),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	require.Empty(
		t,
		graph.WaitsFor("netFunction::netQueue"),
		"the placement link is not being deployed, so nothing can wait for it",
	)
}

// Links that require what each other provide disagree about which of them establishes the
// other's precondition. No order satisfies both, so this is reported rather than resolved
// by picking one.
func Test_capability_graph_reports_a_cycle(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"resourceA", "resourceB"}: {
			provides: []provider.LinkCapability{
				{Name: "test/first", Resource: provider.LinkPriorityResourceA},
			},
			requires: []provider.LinkCapability{
				{Name: "test/second", Resource: provider.LinkPriorityResourceA},
			},
		},
		{"resourceA", "resourceC"}: {
			provides: []provider.LinkCapability{
				{Name: "test/second", Resource: provider.LinkPriorityResourceA},
			},
			requires: []provider.LinkCapability{
				{Name: "test/first", Resource: provider.LinkPriorityResourceA},
			},
		},
	})

	_, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet(
			[2]string{"resourceA", "resourceB"},
			[2]string{"resourceA", "resourceC"},
		),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.Error(t, err)

	runErr, isRunErr := err.(*errors.RunError)
	require.True(t, isRunErr)
	require.Equal(t, ErrorReasonCodeLinkCapabilityCycleDetected, runErr.ReasonCode)
	require.Contains(t, runErr.Error(), "resourceA::resourceB")
	require.Contains(t, runErr.Error(), "resourceA::resourceC")
}

// A link that establishes its own precondition has nothing to wait for, and must not be
// made to wait for itself.
func Test_capability_graph_ignores_a_link_that_provides_what_it_requires(t *testing.T) {
	nodes := capabilityChain(t, map[[2]string]*capabilityTestLink{
		{"netVPC", "netFunction"}: {
			provides: []provider.LinkCapability{
				{Name: testNetworkAttached, Resource: provider.LinkPriorityResourceB},
			},
			requires: []provider.LinkCapability{
				{Name: testNetworkAttached, Resource: provider.LinkPriorityResourceB},
			},
		},
	})

	graph, err := BuildLinkCapabilityGraph(
		context.Background(),
		nodes,
		capabilityChangeSet([2]string{"netVPC", "netFunction"}),
		core.NewDefaultParams(nil, nil, nil, nil),
	)
	require.NoError(t, err)

	require.True(t, graph.Empty())
}
