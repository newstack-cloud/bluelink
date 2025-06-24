// Provides an implementation of a serverless transformer
// for testing purposes.

package internal

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

const ServerlessTransformName = "serverless-2024"

type ServerlessTransformer struct{}

func (t *ServerlessTransformer) GetTransformName(ctx context.Context) (string, error) {
	return ServerlessTransformName, nil
}

func (t *ServerlessTransformer) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{},
	}, nil
}

func (t *ServerlessTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	// Converts an "aws/serverless/function" resource to an "aws/lambda/function" resource.
	transformed := transformServerlessFunctions(input.InputBlueprint)
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: transformed,
	}, nil
}

func transformServerlessFunctions(
	blueprint *schema.Blueprint,
) *schema.Blueprint {
	transform := removeServerlessTransform(blueprint.Transform)
	transformed := &schema.Blueprint{
		Version:   blueprint.Version,
		Transform: transform,
		Resources: &schema.ResourceMap{
			Values:     map[string]*schema.Resource{},
			SourceMeta: map[string]*source.Meta{},
		},
	}
	if blueprint.Resources == nil {
		return transformed
	}

	for resourceName, resource := range blueprint.Resources.Values {
		if resource.Type == nil || resource.Type.Value != "aws/serverless/function" {
			transformed.Resources.Values[resourceName] = resource
			transformed.Resources.SourceMeta[resourceName] = blueprint.Resources.SourceMeta[resourceName]
		} else {
			transformed.Resources.Values[resourceName] = expandServerlessFunction(resource)
		}
	}

	return transformed
}

func expandServerlessFunction(
	resource *schema.Resource,
) *schema.Resource {
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{
			Value: "aws/lambda/function",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"handler": {
					Scalar: &core.ScalarValue{
						StringValue: resource.Spec.Fields["handler"].Scalar.StringValue,
					},
				},
			},
		},
	}
}

func removeServerlessTransform(
	transform *schema.TransformValueWrapper,
) *schema.TransformValueWrapper {
	if transform == nil {
		return nil
	}

	values := []string{}
	sourceMeta := []*source.Meta{}
	for i, value := range transform.Values {
		if value != ServerlessTransformName {
			values = append(values, value)
			sourceMeta = append(sourceMeta, transform.SourceMeta[i])
		}
	}

	return &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values:     values,
			SourceMeta: sourceMeta,
		},
	}
}

func (t *ServerlessTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	if resourceType == "aws/serverless/function" {
		return &serverlessFunctionResource{}, nil
	}

	return nil, nil
}

func (t *ServerlessTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{"aws/serverless/function"}, nil
}

type serverlessFunctionResource struct{}

func (r *serverlessFunctionResource) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{}, nil
}

func (r *serverlessFunctionResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"handler": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *serverlessFunctionResource) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{}, nil
}

func (r *serverlessFunctionResource) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{}, nil
}

func (r *serverlessFunctionResource) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{}, nil
}

func (r *serverlessFunctionResource) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (r *serverlessFunctionResource) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}
