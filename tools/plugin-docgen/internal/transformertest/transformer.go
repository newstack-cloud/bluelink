package transformertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func NewTransformer() transform.SpecTransformer {
	return &transformerv1.TransformerPluginDefinition{
		TransformName:               "celerity-2025-04-01",
		TransformerConfigDefinition: transformerConfigDefinition(),
		AbstractResources: map[string]transform.AbstractResource{
			"celerity/handler":   handlerAbstractResource(),
			"celerity/datastore": datastoreAbstractResource(),
		},
	}
}

func transformerConfigDefinition() *core.ConfigDefinition {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{
			"apiKey": {
				Type:        core.ScalarTypeString,
				Label:       "API key",
				Description: "The API key to talk to the underlying infrastructure.",
				Required:    true,
				Secret:      true,
				Examples: []*core.ScalarValue{
					core.ScalarFromString("sk_10ea49b2a109eaab43f4c3d2b0e1a5e"),
				},
			},
		},
	}
}
