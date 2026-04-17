// Provides functionality for a declared link graph, which is a view of declared
// links within a blueprint that can be used to analyse links and their
// relationships without needing to resolve them.
// This is especially important in enabling rich validation of links between
// abstract resource types defined in transformer plugins.
package links

import (
	"context"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
)

type resourceEntry struct {
	schema         *schema.Resource
	classification linktypes.ResourceClass
}

type defaultDeclaredLinkGraph struct {
	edges     []*linktypes.ResolvedLink
	outgoing  map[string][]*linktypes.ResolvedLink
	incoming  map[string][]*linktypes.ResolvedLink
	resources map[string]resourceEntry
}

func (g *defaultDeclaredLinkGraph) Edges() []*linktypes.ResolvedLink {
	return g.edges
}

func (g *defaultDeclaredLinkGraph) EdgesFrom(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.outgoing[resourceName]; ok {
		return edges
	}

	return []*linktypes.ResolvedLink{}
}

func (g *defaultDeclaredLinkGraph) EdgesTo(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.incoming[resourceName]; ok {
		return edges
	}

	return []*linktypes.ResolvedLink{}
}

func (g *defaultDeclaredLinkGraph) Resource(name string) (*schema.Resource, linktypes.ResourceClass, bool) {
	if entry, ok := g.resources[name]; ok {
		return entry.schema, entry.classification, true
	}
	return nil, "", false
}

// EnumerateDeclaredLinks takes a blueprint and resource registry and produces a DeclaredLinkGraph
// representing the declared links within the blueprint.
// This is intended to be used for analysis and validation of links that is inclusive of abstract resources
// defined in transformers without needing to resolve concrete link implementations.
func EnumerateDeclaredLinks(
	ctx context.Context,
	spec *schema.Blueprint,
	resourceRegistry resourcehelpers.Registry,
) (linktypes.DeclaredLinkGraph, error) {
	groupedResources := GroupResourcesBySelector(
		speccore.BlueprintSpecFromSchema(spec),
	)

	graph := &defaultDeclaredLinkGraph{
		edges:     []*linktypes.ResolvedLink{},
		outgoing:  make(map[string][]*linktypes.ResolvedLink),
		incoming:  make(map[string][]*linktypes.ResolvedLink),
		resources: make(map[string]resourceEntry),
	}

	// Track source -> target -> *linktypes.ResolvedLink to avoid creating duplicate edges in cases where
	// multiple selector keys match between the same source and target resources.
	pairIndex := map[string]map[string]*linktypes.ResolvedLink{}

	for selectorKey, selectGroup := range groupedResources {
		err := collectDeclaredLinks(
			ctx, selectGroup, selectorKey, graph, pairIndex, resourceRegistry,
		)
		if err != nil {
			return nil, err
		}
	}

	return graph, nil
}

func collectDeclaredLinks(
	ctx context.Context,
	selectGroup *SelectGroup,
	selectorKey string,
	graph *defaultDeclaredLinkGraph,
	pairIndex map[string]map[string]*linktypes.ResolvedLink,
	resourceRegistry resourcehelpers.Registry,
) error {
	for _, sourceResourceInfo := range selectGroup.SelectorResources {
		if err := addResourceToGraph(
			ctx, sourceResourceInfo, graph, resourceRegistry,
		); err != nil {
			return err
		}

		for _, targetResourceInfo := range selectGroup.CandidateResourcesForSelection {
			if sourceResourceInfo.Name == targetResourceInfo.Name {
				// Skip self-referencing links where the source and target are the same resource.
				// These can occur when a resource has labels that match its own link selector,
				// but should not be treated as valid links in the graph.
				// This matches the behaviour for concrete link chain resolution in SpecLinkInfo.
				continue
			}

			if err := addResourceToGraph(
				ctx, targetResourceInfo, graph, resourceRegistry,
			); err != nil {
				return err
			}

			targets, ok := pairIndex[sourceResourceInfo.Name]
			if !ok {
				targets = map[string]*linktypes.ResolvedLink{}
				pairIndex[sourceResourceInfo.Name] = targets
			}

			if existing, seen := targets[targetResourceInfo.Name]; seen {
				if !slices.Contains(existing.SelectorKeys, selectorKey) {
					existing.SelectorKeys = append(existing.SelectorKeys, selectorKey)
				}
				continue
			}

			edge := addEdgeToGraph(graph, sourceResourceInfo, targetResourceInfo, selectorKey)
			targets[targetResourceInfo.Name] = edge
		}
	}

	return nil
}

func addResourceToGraph(
	ctx context.Context,
	resourceInfo *ResourceWithNameAndSelectors,
	graph *defaultDeclaredLinkGraph,
	resourceRegistry resourcehelpers.Registry,
) error {
	if _, exists := graph.resources[resourceInfo.Name]; !exists {
		sourceResourceType := schema.GetResourceType(resourceInfo.Resource)
		sourceClassification := linktypes.ResourceClassConcrete
		isAbstract, err := resourceRegistry.IsAbstractResourceType(
			ctx,
			sourceResourceType,
		)
		if err != nil {
			return err
		}

		if isAbstract {
			sourceClassification = linktypes.ResourceClassAbstract
		}

		graph.resources[resourceInfo.Name] = resourceEntry{
			schema:         resourceInfo.Resource,
			classification: sourceClassification,
		}
	}

	return nil
}

func addEdgeToGraph(
	graph *defaultDeclaredLinkGraph,
	sourceResourceInfo *ResourceWithNameAndSelectors,
	targetResourceInfo *ResourceWithNameAndSelectors,
	selectorKey string,
) *linktypes.ResolvedLink {
	edge := &linktypes.ResolvedLink{
		Source:       sourceResourceInfo.Name,
		Target:       targetResourceInfo.Name,
		SourceType:   schema.GetResourceType(sourceResourceInfo.Resource),
		TargetType:   schema.GetResourceType(targetResourceInfo.Resource),
		SelectorKeys: []string{selectorKey},
	}
	graph.edges = append(graph.edges, edge)
	graph.outgoing[sourceResourceInfo.Name] = append(
		graph.outgoing[sourceResourceInfo.Name],
		edge,
	)
	graph.incoming[targetResourceInfo.Name] = append(
		graph.incoming[targetResourceInfo.Name],
		edge,
	)
	return edge
}
