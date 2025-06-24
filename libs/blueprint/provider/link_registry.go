package provider

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// LinkRegistry provides an interface for a registry of link implementations
// that span multiple providers.
type LinkRegistry interface {
	// Link retrieves a link plugin to handle a link between two resource types
	// in a blueprint from one of the registered providers.
	Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (Link, error)
	// Provider retrieves the provider that implements the link between two resource types.
	Provider(resourceTypeA string, resourceTypeB string) (Provider, error)
}

type linkRegistryFromProviders struct {
	providers         map[string]Provider
	linkProviderCache *core.Cache[Provider]
}

// NewLinkRegistry creates a new LinkRegistry from a map of providers,
// each provider will be checked for a given link implementation
// on the first request to retrieve a link for a given resource type pair.
// As links are not tied to the providers of each of the resource types in the link,
// a trial-and-error approach is used to find the correct provider for a link.
func NewLinkRegistry(
	providers map[string]Provider,
) LinkRegistry {
	return &linkRegistryFromProviders{
		providers:         providers,
		linkProviderCache: core.NewCache[Provider](),
	}
}

func (r *linkRegistryFromProviders) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (Link, error) {
	provider, err := r.getProviderForLink(ctx, resourceTypeA, resourceTypeB)
	if err != nil {
		return nil, err
	}

	return provider.Link(ctx, resourceTypeA, resourceTypeB)
}

func (r *linkRegistryFromProviders) Provider(
	resourceTypeA string,
	resourceTypeB string,
) (Provider, error) {
	return r.getProviderForLink(context.Background(), resourceTypeA, resourceTypeB)
}

func (r *linkRegistryFromProviders) getProviderForLink(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (Provider, error) {
	linkType := fmt.Sprintf("%s::%s", resourceTypeA, resourceTypeB)
	provider, found := r.linkProviderCache.Get(linkType)
	if found {
		return provider, nil
	}

	for _, provider := range r.providers {
		link, err := provider.Link(ctx, resourceTypeA, resourceTypeB)
		if err == nil && link != nil {
			r.linkProviderCache.Set(linkType, provider)
			return provider, nil
		}
	}

	return nil, errLinkImplementationNotFound(resourceTypeA, resourceTypeB)
}
