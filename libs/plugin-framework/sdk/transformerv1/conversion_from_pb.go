package transformerv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

func fromPBCustomValidateAbstractResourceRequest(
	req *transformerserverv1.CustomValidateAbstractResourceRequest,
) (*transform.AbstractResourceValidateInput, error) {
	if req == nil {
		return nil, nil
	}

	schemaResource, err := serialisation.FromResourcePB(req.SchemaResource)
	if err != nil {
		return nil, err
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &transform.AbstractResourceValidateInput{
		SchemaResource:     schemaResource,
		TransformerContext: transformerCtx,
	}, nil
}

func fromPBValidateLinksRequest(
	req *transformerserverv1.ValidateLinksRequest,
) (*transform.SpecTransformerValidateLinksInput, error) {
	if req == nil {
		return nil, nil
	}

	blueprint, err := serialisation.FromSchemaPB(req.Blueprint)
	if err != nil {
		return nil, err
	}

	linkGraph, err := fromPBDeclaredLinkGraph(req.LinkGraph, blueprint)
	if err != nil {
		return nil, err
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &transform.SpecTransformerValidateLinksInput{
		Blueprint:          blueprint,
		LinkGraph:          linkGraph,
		TransformerContext: transformerCtx,
	}, nil
}

func fromPBDeclaredLinkGraph(
	pbGraph *transformerserverv1.DeclaredLinkGraph,
	blueprint *schema.Blueprint,
) (linktypes.DeclaredLinkGraph, error) {
	if pbGraph == nil {
		return nil, nil
	}

	edges := make([]*linktypes.ResolvedLink, len(pbGraph.Edges))
	for i, pbEdge := range pbGraph.Edges {
		edges[i] = fromPBResolvedLink(pbEdge)
	}

	resources := make(map[string]resourceEntry)
	for name, pbEntry := range pbGraph.Resources {
		resource, ok := getResource(blueprint, name)
		if !ok {
			return nil, fmt.Errorf("resource %q not found in blueprint", name)
		}

		resources[name] = resourceEntry{
			schema:         resource,
			classification: linktypes.ResourceClass(pbEntry.ResourceClass),
		}
	}

	return newReconstructedLinkGraph(edges, resources), nil
}

func fromPBResolvedLink(pbEdge *transformerserverv1.ResolvedLink) *linktypes.ResolvedLink {
	return &linktypes.ResolvedLink{
		Source:       pbEdge.Source,
		Target:       pbEdge.Target,
		SourceType:   pbEdge.SourceType,
		TargetType:   pbEdge.TargetType,
		SelectorKeys: pbEdge.SelectorKeys,
	}
}

func fromPBTransformerContext(
	pbTransformerCtx *transformerserverv1.TransformerContext,
) (transform.Context, error) {
	if pbTransformerCtx == nil {
		return nil, nil
	}

	transformerConfigVars, err := convertv1.FromPBScalarMap(
		pbTransformerCtx.TransformerConfigVariables,
	)
	if err != nil {
		return nil, err
	}

	contextVars, err := convertv1.FromPBScalarMap(pbTransformerCtx.ContextVariables)
	if err != nil {
		return nil, err
	}

	return utils.TransformerContextFromVarMaps(transformerConfigVars, contextVars), nil
}

func getResource(blueprint *schema.Blueprint, name string) (*schema.Resource, bool) {
	if blueprint == nil || blueprint.Resources == nil {
		return nil, false
	}

	for resourceName, resource := range blueprint.Resources.Values {
		if resourceName == name {
			return resource, true
		}
	}

	return nil, false
}
