package router

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

type routerChildResolver struct {
	routes          map[string]includes.ChildResolver
	defaultResolver includes.ChildResolver
}

type ResolverOption func(*routerChildResolver)

// WithRoute adds a new route to the router resolver
func WithRoute(sourceType string, resolver includes.ChildResolver) ResolverOption {
	return func(r *routerChildResolver) {
		r.routes[sourceType] = resolver
	}
}

// NewResolver creates a new instance of a ChildResolver
// that routes child blueprint resolution to the appropriate resolver
// based on the include metadata `sourceType` field.
// The default resolver is used when no sourceType is provided,
// in most cases this should be a file system resolver.
func NewResolver(
	defaultResolver includes.ChildResolver,
	opts ...ResolverOption,
) includes.ChildResolver {
	router := &routerChildResolver{
		routes:          map[string]includes.ChildResolver{},
		defaultResolver: defaultResolver,
	}

	for _, opt := range opts {
		opt(router)
	}

	return router
}

func (r *routerChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	sourceType := core.StringValue(include.Metadata.Fields["sourceType"])
	if sourceType == "" {
		return r.defaultResolver.Resolve(ctx, includeName, include, params)
	}

	resolver, ok := r.routes[sourceType]
	if !ok {
		return nil, includes.ErrInvalidMetadata(
			includeName,
			fmt.Sprintf("no resolver found for sourceType: %s", sourceType),
		)
	}

	return resolver.Resolve(ctx, includeName, include, params)
}
