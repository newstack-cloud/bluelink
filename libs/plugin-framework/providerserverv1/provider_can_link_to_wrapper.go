package providerserverv1

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

const linkTypeSeparator = "::"

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
		CanLinkTo: deriveLinkableTypes(r.resourceType, r.allLinkTypes),
	}, nil
}

// deriveLinkableTypes extracts all resource types that the given resource can link to.
func deriveLinkableTypes(resourceType string, allLinkTypes []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, linkType := range allLinkTypes {
		target := extractLinkTarget(resourceType, linkType)
		if target == "" {
			continue
		}
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		result = append(result, target)
	}

	return result
}

// extractLinkTarget returns the other resource type if resourceType participates in the link.
// Returns empty string if resourceType is not part of the link.
func extractLinkTarget(resourceType, linkType string) string {
	parts := strings.SplitN(linkType, linkTypeSeparator, 2)
	if len(parts) != 2 {
		return ""
	}

	if parts[0] == resourceType {
		return parts[1]
	}
	if parts[1] == resourceType {
		return parts[0]
	}
	return ""
}
