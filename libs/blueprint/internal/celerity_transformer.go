// Provides an implementation of a transformer with abstract resource types
// under a prefix that does not match any provider namespace,
// for testing purposes.

package internal

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

const (
	CelerityTransformName            = "celerity-2026"
	CelerityHandlerResourceType      = "celerity/handler"
	celerityHandlerExpandsToResource = "aws/lambda/function"
)

// CelerityTransformer is a test transformer with abstract resource types under
// the "celerity" prefix, a namespace that no test provider is registered for.
// This exercises resource type resolution paths that must fall back from
// providers to transformers, unlike ServerlessTransformer whose
// "aws/serverless/function" abstract type shares a prefix with the "aws" provider.
type CelerityTransformer struct{}

func (t *CelerityTransformer) GetTransformName(ctx context.Context) (string, error) {
	return CelerityTransformName, nil
}

func (t *CelerityTransformer) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{},
	}, nil
}

func (t *CelerityTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: transformCelerityHandlers(input.InputBlueprint),
	}, nil
}

func (t *CelerityTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func transformCelerityHandlers(
	blueprint *schema.Blueprint,
) *schema.Blueprint {
	transformed := &schema.Blueprint{
		Version:   blueprint.Version,
		Transform: removeTransform(blueprint.Transform, CelerityTransformName),
		Resources: &schema.ResourceMap{
			Values:     map[string]*schema.Resource{},
			SourceMeta: map[string]*source.Meta{},
		},
	}
	if blueprint.Resources == nil {
		return transformed
	}

	for resourceName, resource := range blueprint.Resources.Values {
		if resource.Type == nil || resource.Type.Value != CelerityHandlerResourceType {
			transformed.Resources.Values[resourceName] = resource
			transformed.Resources.SourceMeta[resourceName] = blueprint.Resources.SourceMeta[resourceName]
		} else {
			transformed.Resources.Values[resourceName] = expandCelerityHandler(resource)
		}
	}

	return transformed
}

func expandCelerityHandler(
	resource *schema.Resource,
) *schema.Resource {
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{
			Value: celerityHandlerExpandsToResource,
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

func (t *CelerityTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	if resourceType == CelerityHandlerResourceType {
		return &serverlessFunctionResource{}, nil
	}

	return nil, nil
}

func (t *CelerityTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{CelerityHandlerResourceType}, nil
}

func (t *CelerityTransformer) ListAbstractLinkTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{}, nil
}

func (t *CelerityTransformer) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	return nil, errors.New("no links defined for celerity transformer")
}
