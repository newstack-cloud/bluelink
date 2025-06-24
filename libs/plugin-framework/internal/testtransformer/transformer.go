package testtransformer

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

// NewTransformer creates a new instance of the test Celerity transformer
// that contains the supported abstract resources for the stub Celerity transformer.
// This is purely for testing purposes.
func NewTransformer() transform.SpecTransformer {
	return &transformerv1.TransformerPluginDefinition{
		TransformName:               "celerity-2025-04-01",
		TransformerConfigDefinition: TestTransformerConfigDefinition(),
		AbstractResources: map[string]transform.AbstractResource{
			"celerity/handler": abstractResourceHandler(),
		},
		TransformFunc: transformBlueprint,
	}
}

// TestTransformerConfigDefinition creates the config definition for the test
// Celerity transformer.
func TestTransformerConfigDefinition() *core.ConfigDefinition {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{
			"deployTarget": {
				Type:        core.ScalarTypeString,
				Label:       "Deploy Target",
				Description: "The target environment to deploy Celerity resources to.",
				Examples: []*core.ScalarValue{
					core.ScalarFromString("aws"),
				},
				Required: true,
			},
		},
	}
}

func transformBlueprint(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: &schema.Blueprint{
			Version: input.InputBlueprint.Version,
			// Replace transforms with an empty list after
			// applying the transformation.
			Transform: &schema.TransformValueWrapper{
				StringList: schema.StringList{
					Values: []string{},
				},
			},
			Variables:   input.InputBlueprint.Variables,
			Include:     input.InputBlueprint.Include,
			Values:      input.InputBlueprint.Values,
			Resources:   input.InputBlueprint.Resources,
			DataSources: input.InputBlueprint.DataSources,
			Exports:     input.InputBlueprint.Exports,
			Metadata:    withNewMetadataField(input.InputBlueprint.Metadata),
		},
	}, nil
}

func withNewMetadataField(
	metadata *core.MappingNode,
) *core.MappingNode {
	if metadata == nil || metadata.Fields == nil {
		metadata = &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	metadata.Fields["test"] = core.MappingNodeFromString(
		"testTransformedMetadataValue",
	)

	return metadata
}
