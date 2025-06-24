package transformerv1

import (
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

func fromPBTransformerContext(
	pbTransformerCtx *transformerserverv1.TransformerContext,
) (transform.Context, error) {
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
