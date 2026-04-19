package transformerserverv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// WrapTransformerWithDerivedCanLinkTo wraps a transformer to automatically derive
// ResourceCanLinkTo from abstract link types defined for a given transformer.
// This eliminates the need for plugin developers to manually maintain
// ResourceCanLinkTo lists.
func WrapTransformerWithDerivedCanLinkTo(
	t transform.SpecTransformer,
) transform.SpecTransformer {
	return &transformerWithDerivedCanLinkTo{
		SpecTransformer: t,
	}
}

type transformerWithDerivedCanLinkTo struct {
	transform.SpecTransformer
}

func (t *transformerWithDerivedCanLinkTo) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	resource, err := t.SpecTransformer.AbstractResource(ctx, resourceType)
	if err != nil {
		return nil, err
	}
	return &abstractResourceWithDerivedCanLinkTo{
		AbstractResource: resource,
		resourceType:     resourceType,
		transformer:      t.SpecTransformer,
	}, nil
}

type abstractResourceWithDerivedCanLinkTo struct {
	transform.AbstractResource
	resourceType string
	transformer  transform.SpecTransformer
}

func (r *abstractResourceWithDerivedCanLinkTo) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	linkTypes, err := r.transformer.ListAbstractLinkTypes(ctx)
	if err != nil {
		return nil, err
	}

	return &transform.AbstractResourceCanLinkToOutput{
		CanLinkTo: utils.DeriveLinkableTypes(r.resourceType, linkTypes),
	}, nil
}
