package transformerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// TransformerPluginDefinition is a template to be used when creating transformer plugins.
// It provides a structure that allows you to define the abstract resources supported
// by the transformer plugin.
// This doesn't have to be used but is a useful way to define the plugin's capabilities,
// there are multiple convenience functions to create new plugins.
// This implements the `transform.SpecTransformer` interface and can be used in the same way
// as any other transformer implementation to create a transformer plugin.
type TransformerPluginDefinition struct {

	// The transform name string that is to be used in the
	// `transform` field of a blueprint.
	TransformName string

	// Configuration definition for the transformer plugin.
	TransformerConfigDefinition *core.ConfigDefinition

	// A mapping of asbtract resource types to their
	// implementations.
	AbstractResources map[string]transform.AbstractResource

	// A function to transform a blueprint.
	// If this function is not set, the default implementation
	// will return the input blueprint as the transformed blueprint.
	TransformFunc func(
		ctx context.Context,
		input *transform.SpecTransformerTransformInput,
	) (*transform.SpecTransformerTransformOutput, error)
}

func (p *TransformerPluginDefinition) GetTransformName(
	ctx context.Context,
) (string, error) {
	return p.TransformName, nil
}

func (p *TransformerPluginDefinition) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return p.TransformerConfigDefinition, nil
}

func (p *TransformerPluginDefinition) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	if p.TransformFunc != nil {
		return p.TransformFunc(ctx, input)
	}

	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

func (p *TransformerPluginDefinition) AbstractResource(
	ctx context.Context,
	abstractResourceType string,
) (transform.AbstractResource, error) {
	resource, ok := p.AbstractResources[abstractResourceType]
	if !ok {
		return nil, errAbstractResourceTypeNotFound(abstractResourceType)
	}
	return resource, nil
}

func (p *TransformerPluginDefinition) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return utils.GetKeys(p.AbstractResources), nil
}
