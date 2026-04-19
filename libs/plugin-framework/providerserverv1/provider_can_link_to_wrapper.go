package providerserverv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// WrapProviderWithDerivedCanLinkTo wraps a provider to automatically derive
// ResourceCanLinkTo from aggregated link types across all providers.
// This eliminates the need for plugin developers to manually maintain
// ResourceCanLinkTo lists.
func WrapProviderWithDerivedCanLinkTo(
	p provider.Provider,
	allLinkTypes []string,
) provider.Provider {
	return &providerWithDerivedCanLinkTo{
		Provider:     p,
		allLinkTypes: allLinkTypes,
	}
}

type providerWithDerivedCanLinkTo struct {
	provider.Provider
	allLinkTypes []string
}

func (p *providerWithDerivedCanLinkTo) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	resource, err := p.Provider.Resource(ctx, resourceType)
	if err != nil {
		return nil, err
	}
	return &resourceWithDerivedCanLinkTo{
		Resource:     resource,
		resourceType: resourceType,
		allLinkTypes: p.allLinkTypes,
	}, nil
}

type resourceWithDerivedCanLinkTo struct {
	provider.Resource
	resourceType string
	allLinkTypes []string
}

func (r *resourceWithDerivedCanLinkTo) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: utils.DeriveLinkableTypes(r.resourceType, r.allLinkTypes),
	}, nil
}
