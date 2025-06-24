package utils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// TransformerContextFromVarMaps creates a transform.Context from the given transformer config and context variables.
// This is primarily useful for creating a blueprint framework transform.Context from config maps derived
// from a deserialised protobuf message.
func TransformerContextFromVarMaps(
	transformerConfigVars map[string]*core.ScalarValue,
	contextVars map[string]*core.ScalarValue,
) transform.Context {
	return &transformerContextFromVarMaps{
		transformerConfigVars: transformerConfigVars,
		contextVars:           contextVars,
	}
}

type transformerContextFromVarMaps struct {
	transformerConfigVars map[string]*core.ScalarValue
	contextVars           map[string]*core.ScalarValue
}

func (p *transformerContextFromVarMaps) TransformerConfigVariable(name string) (*core.ScalarValue, bool) {
	v, ok := p.transformerConfigVars[name]
	return v, ok
}

func (p *transformerContextFromVarMaps) TransformerConfigVariables() map[string]*core.ScalarValue {
	return p.transformerConfigVars
}

func (p *transformerContextFromVarMaps) ContextVariable(name string) (*core.ScalarValue, bool) {
	v, ok := p.contextVars[name]
	return v, ok
}

func (p *transformerContextFromVarMaps) ContextVariables() map[string]*core.ScalarValue {
	return p.contextVars
}
