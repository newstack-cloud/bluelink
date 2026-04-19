package transformerv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

type resourceEntry struct {
	schema         *schema.Resource
	classification linktypes.ResourceClass
}

// reconstructedLinkGraph implements linktypes.DeclaredLinkGraph
// from a deserialised protobuf representation.
// This is used on the SDK/plugin side to reconstruct the link graph
// that was sent from the host for link validation.
type reconstructedLinkGraph struct {
	edges     []*linktypes.ResolvedLink
	outgoing  map[string][]*linktypes.ResolvedLink
	incoming  map[string][]*linktypes.ResolvedLink
	resources map[string]resourceEntry
}

func newReconstructedLinkGraph(
	edges []*linktypes.ResolvedLink,
	resources map[string]resourceEntry,
) *reconstructedLinkGraph {
	outgoing := make(map[string][]*linktypes.ResolvedLink)
	incoming := make(map[string][]*linktypes.ResolvedLink)
	for _, edge := range edges {
		outgoing[edge.Source] = append(outgoing[edge.Source], edge)
		incoming[edge.Target] = append(incoming[edge.Target], edge)
	}
	return &reconstructedLinkGraph{
		edges:     edges,
		outgoing:  outgoing,
		incoming:  incoming,
		resources: resources,
	}
}

func (g *reconstructedLinkGraph) Edges() []*linktypes.ResolvedLink {
	return g.edges
}

func (g *reconstructedLinkGraph) EdgesFrom(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.outgoing[resourceName]; ok {
		return edges
	}
	return []*linktypes.ResolvedLink{}
}

func (g *reconstructedLinkGraph) EdgesTo(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.incoming[resourceName]; ok {
		return edges
	}
	return []*linktypes.ResolvedLink{}
}

func (g *reconstructedLinkGraph) Resource(name string) (*schema.Resource, linktypes.ResourceClass, bool) {
	if entry, ok := g.resources[name]; ok {
		return entry.schema, entry.classification, true
	}
	return nil, "", false
}
