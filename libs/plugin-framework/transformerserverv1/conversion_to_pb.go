package transformerserverv1

import (
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
