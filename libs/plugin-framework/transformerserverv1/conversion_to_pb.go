package transformerserverv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
)

func toPBTransformerContext(transformerCtx transform.Context) (*TransformerContext, error) {
	transformerConfigVars, err := convertv1.ToPBScalarMap(transformerCtx.TransformerConfigVariables())
	if err != nil {
		return nil, err
	}

	contextVars, err := convertv1.ToPBScalarMap(transformerCtx.ContextVariables())
	if err != nil {
		return nil, err
	}

	return &TransformerContext{
		TransformerConfigVariables: transformerConfigVars,
		ContextVariables:           contextVars,
	}, nil
}

func toPBDeclaredLinkGraph(graph linktypes.DeclaredLinkGraph) (*DeclaredLinkGraph, error) {
	if graph == nil {
		return nil, nil
	}

	edges := graph.Edges()
	pbEdges := make([]*ResolvedLink, len(edges))
	for i, edge := range edges {
		pbEdges[i] = toPBResolvedLink(edge)
	}

	resourceNames := collectResourceNamesFromEdges(edges)
	pbResources := make(map[string]*DeclaredLinkGraphEntry)
	for _, name := range resourceNames {
		_, class, found := graph.Resource(name)
		if !found {
			continue
		}

		pbResources[name] = &DeclaredLinkGraphEntry{
			ResourceClass: string(class),
		}
	}

	return &DeclaredLinkGraph{
		Edges:     pbEdges,
		Resources: pbResources,
	}, nil
}

func toPBResolvedLink(edge *linktypes.ResolvedLink) *ResolvedLink {
	return &ResolvedLink{
		Source:       edge.Source,
		Target:       edge.Target,
		SourceType:   edge.SourceType,
		TargetType:   edge.TargetType,
		SelectorKeys: edge.SelectorKeys,
	}
}

func collectResourceNamesFromEdges(edges []*linktypes.ResolvedLink) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, edge := range edges {
		if _, ok := seen[edge.Source]; !ok {
			seen[edge.Source] = struct{}{}
			names = append(names, edge.Source)
		}
		if _, ok := seen[edge.Target]; !ok {
			seen[edge.Target] = struct{}{}
			names = append(names, edge.Target)
		}
	}
	return names
}
