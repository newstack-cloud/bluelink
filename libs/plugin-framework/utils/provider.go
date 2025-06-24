package utils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ProviderContextFromVarMaps creates a provider.Context from the given provider config and context variables.
// This is primarily useful for creating a blueprint framework provider.Context from config maps derived
// from a deserialised protobuf message.
func ProviderContextFromVarMaps(
	providerConfigVars map[string]*core.ScalarValue,
	contextVars map[string]*core.ScalarValue,
) provider.Context {
	return &providerContextFromVarMaps{
		providerConfigVars: providerConfigVars,
		contextVars:        contextVars,
	}
}

type providerContextFromVarMaps struct {
	providerConfigVars map[string]*core.ScalarValue
	contextVars        map[string]*core.ScalarValue
}

func (p *providerContextFromVarMaps) ProviderConfigVariable(name string) (*core.ScalarValue, bool) {
	v, ok := p.providerConfigVars[name]
	return v, ok
}

func (p *providerContextFromVarMaps) ProviderConfigVariables() map[string]*core.ScalarValue {
	return p.providerConfigVars
}

func (p *providerContextFromVarMaps) ContextVariable(name string) (*core.ScalarValue, bool) {
	v, ok := p.contextVars[name]
	return v, ok
}

func (p *providerContextFromVarMaps) ContextVariables() map[string]*core.ScalarValue {
	return p.contextVars
}
