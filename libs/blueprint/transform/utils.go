package transform

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// ExtractTransformerFromItemType extracts the transformer namespace from an
// abstract resource type.
func ExtractTransformerFromItemType(itemType string) string {
	parts := strings.Split(itemType, "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

type transformerCtxFromParams struct {
	transformerNamespace string
	blueprintParams      core.BlueprintParams
}

// NewTransformerContextFromParams creates a new transformer context
// from a set of blueprint parameters for the current environment.
// The transformer context will then be passed into transformer plugins
// to allow them to access configuration values and context variables.
func NewTransformerContextFromParams(
	transformerNamespace string,
	blueprintParams core.BlueprintParams,
) Context {
	return &transformerCtxFromParams{
		transformerNamespace: transformerNamespace,
		blueprintParams:      blueprintParams,
	}
}

func (p *transformerCtxFromParams) TransformerConfigVariable(name string) (*core.ScalarValue, bool) {
	transformerConfig := p.blueprintParams.TransformerConfig(p.transformerNamespace)
	if transformerConfig == nil {
		return nil, false
	}

	configValue, ok := transformerConfig[name]
	return configValue, ok
}

func (p *transformerCtxFromParams) ContextVariable(name string) (*core.ScalarValue, bool) {
	contextVar := p.blueprintParams.ContextVariable(name)
	if contextVar == nil {
		return nil, false
	}
	return contextVar, true
}

func (p *transformerCtxFromParams) TransformerConfigVariables() map[string]*core.ScalarValue {
	return p.blueprintParams.TransformerConfig(p.transformerNamespace)
}

func (p *transformerCtxFromParams) ContextVariables() map[string]*core.ScalarValue {
	return p.blueprintParams.AllContextVariables()
}
