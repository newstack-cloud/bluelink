package container

import (
	"context"
	"slices"
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Matching on the resource instance rather than the resource type is what keeps two
// resources of the same type in one blueprint from ordering each other: a link that
// places one function in a VPC provides the capability on that function alone.
type capabilityRef struct {
	name     string
	resource string
}

// LinkCapabilityGraph holds the deployment ordering implied by the capabilities
// that links declare, as a mapping from a link to the links it must follow.
//
// Ordering runs forwards on deploy and backwards on destroy. A link that requires
// a capability is deployed after every link that provides it, and destroyed
// before them. For example, in the context of a VPC, an access link must revoke
// its rules while the function is still attached to the network,
// so tearing placement down first would strand them.
type LinkCapabilityGraph struct {
	waitsFor map[string][]string
	// What each link declared it writes, resolved from the same GetCapabilities call
	// that produced the ordering. A link with no entry is treated as writing both
	// resources, which is the behaviour that predates the declaration.
	modifies map[string]provider.LinkModifies
	// How far down the ordering each link sits, zero for a link that requires nothing
	// provided in this deployment, otherwise one more than the deepest link it waits for.
	//
	// Well defined because the ordering is acyclic, which is checked before this is
	// resolved. It is what lets a resource's contributions be applied in layers rather
	// than in one write, so a link can require a capability on a resource it also
	// contributes to.
	depth map[string]int
	// The resource updates each link waits for, for a capability that is established by
	// a contribution rather than by a write. Empty for every link whose providers write
	// their resources directly, which is every link today.
	waitsForLayers map[string][]ContributionLayer
}

// ContributionLayer identifies one update of a resource carrying the contributions of the
// links at a given depth and above it in the capability ordering.
//
// A resource whose contributors all sit at one depth, which is every resource in a
// deployment with no capability ordering over it, has a single layer and is written once.
type ContributionLayer struct {
	ResourceName string
	Depth        int
}

// Depth returns how far down the capability ordering the named link sits, zero for a link
// that requires nothing provided in this deployment.
func (g *LinkCapabilityGraph) Depth(linkName string) int {
	if g == nil {
		return 0
	}

	return g.depth[linkName]
}

// WaitsForLayers returns the resource updates the named link must see land before it runs,
// for capabilities its providers establish by contributing rather than by writing.
func (g *LinkCapabilityGraph) WaitsForLayers(linkName string) []ContributionLayer {
	if g == nil {
		return nil
	}

	return g.waitsForLayers[linkName]
}

// WaitsFor returns the logical names of the links that must finish deploying
// before the named link runs. The result is empty for a link that declared no
// requirements, and for one whose requirements are not provided by any links
// in the current deployment.
func (g *LinkCapabilityGraph) WaitsFor(linkName string) []string {
	if g == nil {
		return nil
	}

	return g.waitsFor[linkName]
}

// Modifies returns which resources in its relationship the named link declared it
// writes, defaulting to both for a link that declared nothing.
//
// The deployer takes an exclusive lock only on a declared side, and the scheduler only
// holds a link back for a side it writes, so a resource that many links merely read stops
// serialising them against one another.
func (g *LinkCapabilityGraph) Modifies(linkName string) provider.LinkModifies {
	if g == nil {
		return provider.LinkModifiesBoth
	}

	return g.modifies[linkName]
}

// Empty reports whether no link in the deployment declared an ordering
// constraint that another link satisfies, in which case link deployment is
// unconstrained.
func (g *LinkCapabilityGraph) Empty() bool {
	return g == nil || len(g.waitsFor) == 0
}

// The chain direction and a link implementation's own A/B orientation usually agree, but
// a capability selector names a side of the implementation, so resourceA and resourceB
// are recorded in the implementation's terms rather than the chain's.
type linkCapabilityPair struct {
	name      string
	resourceA string
	resourceB string
}

type capabilityIndex struct {
	providers    map[capabilityRef][]string
	requirements map[string][]capabilityRef
}

func newCapabilityIndex() *capabilityIndex {
	return &capabilityIndex{
		providers:    map[capabilityRef][]string{},
		requirements: map[string][]capabilityRef{},
	}
}

func (i *capabilityIndex) addProvided(
	pair *linkCapabilityPair,
	capabilities []provider.LinkCapability,
) {
	for _, capability := range capabilities {
		if ref, ok := resolveCapabilityRef(capability, pair); ok {
			i.providers[ref] = append(i.providers[ref], pair.name)
		}
	}
}

func (i *capabilityIndex) addRequired(
	pair *linkCapabilityPair,
	capabilities []provider.LinkCapability,
) {
	for _, capability := range capabilities {
		if ref, ok := resolveCapabilityRef(capability, pair); ok {
			i.requirements[pair.name] = append(i.requirements[pair.name], ref)
		}
	}
}

func (i *capabilityIndex) edges() map[string][]string {
	edges := map[string][]string{}
	for linkName, refs := range i.requirements {
		waiting := i.providersFor(linkName, refs)
		if len(waiting) > 0 {
			edges[linkName] = waiting
		}
	}

	return edges
}

// The resource updates each requiring link has to see land, for the capabilities whose
// providers establish them by contributing to the resource rather than by writing it.
//
// A provider that writes its resource directly has established the capability by the time
// it settles, which is what the link ordering already waits for, so it adds no layer.
func (i *capabilityIndex) contributionLayers(
	depth map[string]int,
	contributionTargets map[string][]string,
) map[string][]ContributionLayer {
	layers := map[string][]ContributionLayer{}
	for linkName, refs := range i.requirements {
		for _, ref := range refs {
			for _, providerName := range i.providers[ref] {
				if providerName == linkName {
					continue
				}

				if !slices.Contains(contributionTargets[providerName], ref.resource) {
					continue
				}

				layer := ContributionLayer{
					ResourceName: ref.resource,
					Depth:        depth[providerName],
				}
				if !slices.Contains(layers[linkName], layer) {
					layers[linkName] = append(layers[linkName], layer)
				}
			}
		}
	}

	for linkName := range layers {
		sort.Slice(layers[linkName], func(a, b int) bool {
			left, right := layers[linkName][a], layers[linkName][b]
			if left.ResourceName != right.ResourceName {
				return left.ResourceName < right.ResourceName
			}
			return left.Depth < right.Depth
		})
	}

	return layers
}

func (i *capabilityIndex) providersFor(linkName string, refs []capabilityRef) []string {
	waiting := []string{}
	for _, ref := range refs {
		for _, providerName := range i.providers[ref] {
			// A link that both provides and requires a capability establishes its
			// own precondition and has nothing to wait for.
			if providerName == linkName || slices.Contains(waiting, providerName) {
				continue
			}
			waiting = append(waiting, providerName)
		}
	}
	sort.Strings(waiting)

	return waiting
}

// BuildLinkCapabilityGraph resolves the capabilities declared by every link in
// the deployment into the ordering they imply.
func BuildLinkCapabilityGraph(
	ctx context.Context,
	nodes []*DeploymentNode,
	blueprintChanges *changes.BlueprintChanges,
	params core.BlueprintParams,
) (*LinkCapabilityGraph, error) {
	pairs, err := collectLinkCapabilityPairs(nodes)
	if err != nil {
		return nil, err
	}

	linkCtx := provider.NewLinkContextFromParams(params)
	index := newCapabilityIndex()
	modifies := map[string]provider.LinkModifies{}
	for _, pair := range pairs {
		output, err := getLinkCapabilities(ctx, pair.impl, linkCtx)
		if err != nil {
			return nil, err
		}

		if linkInChangeSet(pair.pending, blueprintChanges) {
			index.addProvided(pair.capabilityPair, output.Provides)
		}
		index.addRequired(pair.capabilityPair, output.Requires)
		modifies[pendingLinkName(pair.pending)] = output.Modifies
	}

	waitsFor := index.edges()
	if cycle := findCapabilityCycle(waitsFor); len(cycle) > 0 {
		return nil, errLinkCapabilityCycle(cycle)
	}

	// Resolved after the cycle check, since both the depth of a link and the layers it
	// waits for are only well defined over an acyclic ordering.
	depth := capabilityDepths(waitsFor)

	return &LinkCapabilityGraph{
		waitsFor: waitsFor,
		modifies: modifies,
		depth:    depth,
		waitsForLayers: index.contributionLayers(
			depth,
			BuildLinkContributionTargets(blueprintChanges),
		),
	}, nil
}

// The depth of every link in an acyclic ordering which is zero
// for a link that waits for nothing,
// otherwise one more than the deepest link it waits for.
//
// Longest path rather than shortest, so that a link sits below every provider it depends
// on rather than only the nearest one. A link reached from several depths takes the
// greatest, which is what keeps its layer after all of theirs.
func capabilityDepths(waitsFor map[string][]string) map[string]int {
	depths := map[string]int{}

	var resolve func(linkName string, visiting map[string]bool) int
	resolve = func(linkName string, visiting map[string]bool) int {
		if known, resolved := depths[linkName]; resolved {
			return known
		}

		// The ordering is checked for cycles before this runs, so this guards against a
		// caller resolving depths over an unchecked graph rather than against a graph
		// this package would build.
		if visiting[linkName] {
			return 0
		}
		visiting[linkName] = true

		deepest := 0
		for _, providerName := range waitsFor[linkName] {
			if below := resolve(providerName, visiting) + 1; below > deepest {
				deepest = below
			}
		}

		delete(visiting, linkName)
		depths[linkName] = deepest

		return deepest
	}

	linkNames := make([]string, 0, len(waitsFor))
	for linkName := range waitsFor {
		linkNames = append(linkNames, linkName)
	}
	sort.Strings(linkNames)

	for _, linkName := range linkNames {
		resolve(linkName, map[string]bool{})
	}

	return depths
}

// BuildLinkCapabilityRemovalEdges resolves the capabilities declared by the links being
// removed into the order they have to come apart in, as a mapping from a link to the
// links it must be destroyed before.
//
// Teardown runs the deploy ordering backwards. For example, an access link has to revoke its rules
// while the function is still attached to the network: if placement detaches first, the
// access link sees an unattached function, revokes nothing, and leaves rules holding the
// security group, which holds the VPC.
func BuildLinkCapabilityRemovalEdges(
	ctx context.Context,
	linkElements []*LinkIDInfo,
	currentState *state.InstanceState,
	linkRegistry provider.LinkRegistry,
	params core.BlueprintParams,
) (map[string][]string, error) {
	linkCtx := provider.NewLinkContextFromParams(params)
	index := newCapabilityIndex()

	for _, linkElement := range linkElements {
		pair, linkImpl := resolveRemovedLink(ctx, linkElement.LinkName, currentState, linkRegistry)
		if linkImpl == nil {
			continue
		}

		output, err := getLinkCapabilities(ctx, linkImpl, linkCtx)
		if err != nil {
			return nil, err
		}

		index.addProvided(pair, output.Provides)
		index.addRequired(pair, output.Requires)
	}

	return index.edges(), nil
}

// A nil link implementation means the link cannot be ordered against anything and is
// skipped, which happens when its name is malformed, its resources are no longer in
// state, or no implementation is registered for its resource types.
//
// None of those cause failures here. Ordering runs during the preparation phase, so
// returning an error would abandon the whole removal before anything is torn down, over a
// link that may not even declare a capability. The destroyer resolves the same
// implementation when it comes to remove the link and reports the failure there, against
// the link it actually belongs to.
func resolveRemovedLink(
	ctx context.Context,
	linkName string,
	currentState *state.InstanceState,
	linkRegistry provider.LinkRegistry,
) (*linkCapabilityPair, provider.Link) {
	names := extractLinkDirectDependencies(linkName)
	if names == nil {
		return nil, nil
	}

	resourceTypeA, resourceTypeB, err := getResourceTypesForLink(linkName, currentState)
	if err != nil {
		return nil, nil
	}

	linkImpl, err := linkRegistry.Link(ctx, resourceTypeA, resourceTypeB)
	if err != nil {
		return nil, nil
	}

	return &linkCapabilityPair{
		name:      linkName,
		resourceA: names.resourceAName,
		resourceB: names.resourceBName,
	}, linkImpl
}

func getLinkCapabilities(
	ctx context.Context,
	linkImpl provider.Link,
	linkCtx provider.LinkContext,
) (*provider.LinkGetCapabilitiesOutput, error) {
	output, err := linkImpl.GetCapabilities(
		ctx,
		&provider.LinkGetCapabilitiesInput{
			LinkContext: linkCtx,
		},
	)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &provider.LinkGetCapabilitiesOutput{}, nil
	}

	return output, nil
}

type deploymentLinkCapabilityPair struct {
	capabilityPair *linkCapabilityPair
	pending        *LinkPendingCompletion
	impl           provider.Link
}

// Links are deduplicated by logical name, since a link is reachable from both of the
// resources it connects, and sorted so that the resulting edges do not depend on map
// iteration order.
func collectLinkCapabilityPairs(nodes []*DeploymentNode) ([]*deploymentLinkCapabilityPair, error) {
	seen := map[string]bool{}
	pairs := []*deploymentLinkCapabilityPair{}

	for _, node := range nodes {
		chainNode := node.ChainLinkNode
		if chainNode == nil {
			continue
		}

		nodePairs, err := collectNodeLinkCapabilityPairs(chainNode, seen)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, nodePairs...)
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].capabilityPair.name < pairs[j].capabilityPair.name
	})

	return pairs, nil
}

func collectNodeLinkCapabilityPairs(
	chainNode *links.ChainLinkNode,
	seen map[string]bool,
) ([]*deploymentLinkCapabilityPair, error) {
	pairs := []*deploymentLinkCapabilityPair{}
	for _, linkedTo := range chainNode.LinksTo {
		pair, err := newDeploymentLinkCapabilityPair(chainNode, linkedTo)
		if err != nil {
			return nil, err
		}
		if seen[pair.capabilityPair.name] {
			continue
		}
		seen[pair.capabilityPair.name] = true
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

func newDeploymentLinkCapabilityPair(
	fromNode *links.ChainLinkNode,
	toNode *links.ChainLinkNode,
) (*deploymentLinkCapabilityPair, error) {
	linkImpl, implResourceA, err := getLinkImplementation(fromNode, toNode)
	if err != nil {
		return nil, err
	}

	resourceA, resourceB := fromNode.ResourceName, toNode.ResourceName
	if implResourceA == toNode.ResourceName {
		resourceA, resourceB = toNode.ResourceName, fromNode.ResourceName
	}

	return &deploymentLinkCapabilityPair{
		capabilityPair: &linkCapabilityPair{
			name:      core.LogicalLinkName(fromNode.ResourceName, toNode.ResourceName),
			resourceA: resourceA,
			resourceB: resourceB,
		},
		pending: &LinkPendingCompletion{
			resourceANode: fromNode,
			resourceBNode: toNode,
		},
		impl: linkImpl,
	}, nil
}

func resolveCapabilityRef(
	capability provider.LinkCapability,
	pair *linkCapabilityPair,
) (capabilityRef, bool) {
	if capability.Name == "" {
		return capabilityRef{}, false
	}

	switch capability.Resource {
	case provider.LinkPriorityResourceA:
		return capabilityRef{
			name:     capability.Name,
			resource: pair.resourceA,
		}, true
	case provider.LinkPriorityResourceB:
		return capabilityRef{
			name:     capability.Name,
			resource: pair.resourceB,
		}, true
	default:
		// A capability that names neither side has nothing to be matched on, and is
		// treated as undeclared rather than applied to both.
		return capabilityRef{}, false
	}
}

// A cycle means the links disagree about which of them establishes the other's
// precondition, and there is no order that satisfies both. That is an error in how the
// links were written rather than something to resolve arbitrarily.
func findCapabilityCycle(waitsFor map[string][]string) []string {
	walk := &capabilityCycleWalk{
		waitsFor:    waitsFor,
		stateByLink: map[string]int{},
	}

	linkNames := make([]string, 0, len(waitsFor))
	for linkName := range waitsFor {
		linkNames = append(linkNames, linkName)
	}
	sort.Strings(linkNames)

	for _, linkName := range linkNames {
		if cycle := walk.visit(linkName); len(cycle) > 0 {
			return cycle
		}
	}

	return nil
}

const (
	capabilityWalkUnvisited = iota
	capabilityWalkOnStack
	capabilityWalkDone
)

type capabilityCycleWalk struct {
	waitsFor    map[string][]string
	stateByLink map[string]int
	stack       []string
}

func (w *capabilityCycleWalk) visit(linkName string) []string {
	switch w.stateByLink[linkName] {
	case capabilityWalkDone:
		return nil
	case capabilityWalkOnStack:
		// Trimmed back to where this link was first entered so that the reported
		// cycle holds only the links actually involved.
		start := slices.Index(w.stack, linkName)
		return append(slices.Clone(w.stack[start:]), linkName)
	}

	w.stateByLink[linkName] = capabilityWalkOnStack
	w.stack = append(w.stack, linkName)
	for _, next := range w.waitsFor[linkName] {
		if cycle := w.visit(next); len(cycle) > 0 {
			return cycle
		}
	}
	w.stack = w.stack[:len(w.stack)-1]
	w.stateByLink[linkName] = capabilityWalkDone

	return nil
}
