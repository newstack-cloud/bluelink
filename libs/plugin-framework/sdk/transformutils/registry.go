package transformutils

import (
	"fmt"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Target represents a deployment target such as "aws-serverless" or "azure".
type Target string

// Aggregator is a function that produces an emit plan
// from a list of resources resolved for a particular abstract resource type.
type Aggregator func([]ResolvedResource) *EmitPlan

// Emitter produces concrete output for one resolved primary for a specific target.
type Emitter func(
	r ResolvedResource,
	resPropRewriter ResourcePropertyRewriter,
	transformCtx transform.Context,
) (*EmitResult, error)

// RewriteFactory produces a list of rewriters from on resolved primary.
// Returns multiple for compound primaries (e.g. a ResolvedService folding
// in N handlers + M apis, contributes to N+M rewriters).
type RewriteFactory func(r ResolvedResource) []ResourcePropertyRewriter

// Resolver is a target-agnostic resolver for a single abstract resource type.
// These resolvers are keyed by abstract resource type in a transformer registry
// and are not tied to any specific target.
type Resolver func(
	name string,
	resource *schema.Resource,
	linkGraph linktypes.DeclaredLinkGraph,
	blueprint *schema.Blueprint,
) (ResolvedResource, error)

type TransformerRegistry struct {
	// Keyed by abstract resource type.
	resolvers    map[string]Resolver
	aggregators  map[Target]Aggregator
	emitters     map[Target]map[reflect.Type]Emitter
	rewriters    map[Target]map[reflect.Type]RewriteFactory
	capabilities map[Target]map[reflect.Type]Capabilities
}

func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		resolvers:    make(map[string]Resolver),
		aggregators:  make(map[Target]Aggregator),
		emitters:     make(map[Target]map[reflect.Type]Emitter),
		rewriters:    make(map[Target]map[reflect.Type]RewriteFactory),
		capabilities: make(map[Target]map[reflect.Type]Capabilities),
	}
}

// RegisterResolver registers a resolver function for a given abstract resource type.
func (r *TransformerRegistry) RegisterResolver(
	abstractResourceType string,
	resolver Resolver,
) {
	r.resolvers[abstractResourceType] = resolver
}

// RegisterAggregator registers an aggregator function for a given deployment target.
func (r *TransformerRegistry) RegisterAggregator(
	target Target,
	aggregator Aggregator,
) {
	r.aggregators[target] = aggregator
}

// RegisterEmit is a generic helper to register an emitter for a specific target,
// ensuring type safety on the resolved type.
//
// The registry is keyed by the pointer type *T (PR), not the value type T,
// because aggregators put *T values into EmitPlan.Primaries — so
// reflect.TypeOf(primary) at emit-lookup time is the pointer type, and the
// registration key must match.
func RegisterEmit[T any, PR ResolvedPtr[T]](
	reg *TransformerRegistry,
	target Target,
	fn func(r PR, resPropRewriter ResourcePropertyRewriter, transformCtx transform.Context) (*EmitResult, error),
) {
	if reg.emitters[target] == nil {
		reg.emitters[target] = make(map[reflect.Type]Emitter)
	}

	reg.emitters[target][reflect.TypeFor[PR]()] = func(
		r ResolvedResource,
		resPropRewriter ResourcePropertyRewriter,
		transformCtx transform.Context,
	) (*EmitResult, error) {
		casted, ok := r.(PR)
		if !ok {
			return nil, fmt.Errorf("expected resource of type %T but got %T", (*PR)(nil), r)
		}
		return fn(casted, resPropRewriter, transformCtx)
	}
}

// TypedEmitter is a generic helper to produce an EmitterRegistration for a specific target,
// ensuring type safety on the resolved type. This is intended for use in AbstractResourceDefinition.Emitters maps.
func TypedEmitter[T any, PR ResolvedPtr[T]](
	fn func(
		r PR,
		resPropRewriter ResourcePropertyRewriter,
		transformCtx transform.Context,
	) (*EmitResult, error),
) EmitterRegistration {
	return func(registry *TransformerRegistry, target Target) {
		RegisterEmit(registry, target, fn)
	}
}

// TypedRewriter is a generic helper to produce a RewriterRegistration for a specific target,
// ensuring type safety on the resolved type. This is intended for use in AbstractResourceDefinition.Rewriters maps.
func TypedRewriter[T any, PR ResolvedPtr[T]](
	fn func(r PR) []ResourcePropertyRewriter,
) RewriterRegistration {
	return func(registry *TransformerRegistry, target Target) {
		RegisterRewriter(registry, target, fn)
	}
}

// RewriterFromPropertyMap produces a RewriterRegistration that wraps a PropertyMap
// in a single-rewriter factory. The concreteName function derives the concrete
// resource name for one resolved primary; this is the per-target piece of
// information that the PropertyMap itself can't carry (different targets name
// the same abstract resource differently — _sqs, _lambda, _topic, etc.).
//
// Side effect: also registers a capability-matrix entry for (target, *T)
// derived from the same PropertyMap, so pre-emit reference validation and
// authoring docs see the supported abstract paths without authors having to
// do anything else. The capability matrix and the rewriter share the same
// PropertyMap as a single source of truth.
//
// Use this as the typical declarative path for AbstractResourceDefinition.Rewriters
// when the rewriting is fully described by a PropertyMap. For compound primaries
// or multi-rewriter contributions, use TypedRewriter with a hand-written factory
// instead — capability matrix entries for those cases must be supplied via the
// Registry overlay.
func RewriterFromPropertyMap[T any, PR ResolvedPtr[T]](
	pm *PropertyMap,
	concreteName func(r PR) string,
) RewriterRegistration {
	return func(registry *TransformerRegistry, target Target) {
		RegisterRewriter(registry, target, func(r PR) []ResourcePropertyRewriter {
			return []ResourcePropertyRewriter{
				pm.Rewriter(r.ResourceName(), concreteName(r)),
			}
		})
		registry.registerCapabilities(
			target,
			reflect.TypeFor[PR](),
			CapabilitiesFromPropertyMap(pm),
		)
	}
}

// RegisterRewriter is a generic helper to register a rewrite factory for a specific target,
// ensuring type safety on the resolved type. Keyed by *T (PR) for the same
// reason as RegisterEmit — primaries are pointer-typed at lookup time.
func RegisterRewriter[T any, PR ResolvedPtr[T]](
	reg *TransformerRegistry,
	target Target,
	fn func(r PR) []ResourcePropertyRewriter,
) {
	if reg.rewriters[target] == nil {
		reg.rewriters[target] = make(map[reflect.Type]RewriteFactory)
	}

	reg.rewriters[target][reflect.TypeFor[PR]()] = func(
		r ResolvedResource,
	) []ResourcePropertyRewriter {
		casted, ok := r.(PR)
		if !ok {
			return nil
		}
		return fn(casted)
	}
}

// ResolverFor looks up the resolver for a given abstract resource type.
func (r *TransformerRegistry) ResolverFor(abstractResourceType string) (Resolver, bool) {
	resolver, found := r.resolvers[abstractResourceType]
	return resolver, found
}

// AggregatorFor looks up the aggregator for a given target.
func (r *TransformerRegistry) AggregatorFor(target Target) (Aggregator, bool) {
	aggregator, found := r.aggregators[target]
	return aggregator, found
}

// EmitterFor looks up the emitter for a given target and
// resolved resource type.
func (r *TransformerRegistry) EmitterFor(
	target Target,
	resolvedType reflect.Type,
) (Emitter, bool) {
	emittersForTarget, found := r.emitters[target]
	if !found {
		return nil, false
	}

	emitter, found := emittersForTarget[resolvedType]
	return emitter, found
}

// RewriteFactoryFor looks up the rewrite factory for a given target and
// resolved resource type.
func (r *TransformerRegistry) RewriteFactoryFor(
	target Target,
	resolvedType reflect.Type,
) (RewriteFactory, bool) {
	rewritersForTarget, found := r.rewriters[target]
	if !found {
		return nil, false
	}

	rewriter, found := rewritersForTarget[resolvedType]
	return rewriter, found
}

// CapabilitiesFor looks up the capability matrix entry for a given target
// and resolved resource type. Used by pre-emit reference validation
// (Pillar 4) and tooling that renders authoring docs.
func (r *TransformerRegistry) CapabilitiesFor(
	target Target,
	resolvedType reflect.Type,
) (Capabilities, bool) {
	capsForTarget, found := r.capabilities[target]
	if !found {
		return Capabilities{}, false
	}

	caps, found := capsForTarget[resolvedType]
	return caps, found
}

// HasAggregators reports whether the registry has at least one aggregator
// registered. Used by the framework's first-use validation to confirm that
// a pipeline-mode plugin has at least one aggregator across the union of
// TransformerPluginDefinition.Aggregators and the Registry overlay.
func (r *TransformerRegistry) HasAggregators() bool {
	return len(r.aggregators) > 0
}

// registerCapabilities writes a capability-matrix entry for the (target,
// resolvedType) pair. Currently called only from RewriterFromPropertyMap
// (which has T statically); there is no public RegisterCapabilities helper
// because the auto-derive path is the only intended population route.
// Authors with non-PropertyMap rewriting populate via the Registry overlay
// using their own pre-built TransformerRegistry.
func (r *TransformerRegistry) registerCapabilities(
	target Target,
	resolvedType reflect.Type,
	caps *Capabilities,
) {
	if caps == nil {
		return
	}

	if r.capabilities[target] == nil {
		r.capabilities[target] = make(map[reflect.Type]Capabilities)
	}

	r.capabilities[target][resolvedType] = *caps
}

// IsEmpty reports whether the registry holds zero registrations across all
// dimensions (resolvers, aggregators, emitters, rewriters, capabilities).
// Used by the framework to decide whether a non-nil overlay registry
// actually contributes anything to the pipeline-detection signal.
func (r *TransformerRegistry) IsEmpty() bool {
	if len(r.resolvers) > 0 || len(r.aggregators) > 0 {
		return false
	}

	for _, byType := range r.emitters {
		if len(byType) > 0 {
			return false
		}
	}

	for _, byType := range r.rewriters {
		if len(byType) > 0 {
			return false
		}
	}

	for _, byType := range r.capabilities {
		if len(byType) > 0 {
			return false
		}
	}

	return true
}

// MergeFrom merges every registration from other into r. Collisions on any
// dimension (resolver abstract type, aggregator target, (target, resolved-
// type) emitter / rewriter / capability) panic with a "duplicate
// registration" message — this matches the original design's "Registry overlay
// collides with auto-derived registration" contract and keeps misconfig
// loud rather than silently overwriting one source with another.
//
// A nil receiver panics; a nil other is a no-op.
func (r *TransformerRegistry) MergeFrom(other *TransformerRegistry) {
	if other == nil {
		return
	}

	for abstractType, resolver := range other.resolvers {
		if _, exists := r.resolvers[abstractType]; exists {
			panic(
				fmt.Sprintf(
					"duplicate registration: resolver for abstract resource type %q",
					abstractType,
				),
			)
		}
		r.resolvers[abstractType] = resolver
	}

	for target, aggregator := range other.aggregators {
		if _, exists := r.aggregators[target]; exists {
			panic(
				fmt.Sprintf(
					"duplicate registration: aggregator for target %q",
					target,
				),
			)
		}
		r.aggregators[target] = aggregator
	}

	mergeByTypeMap(
		r.emitters, other.emitters,
		func() map[reflect.Type]Emitter { return make(map[reflect.Type]Emitter) },
		"emitter",
	)
	mergeByTypeMap(
		r.rewriters, other.rewriters,
		func() map[reflect.Type]RewriteFactory { return make(map[reflect.Type]RewriteFactory) },
		"rewriter",
	)
	mergeByTypeMap(
		r.capabilities, other.capabilities,
		func() map[reflect.Type]Capabilities { return make(map[reflect.Type]Capabilities) },
		"capabilities",
	)
}

// Merges one of the (Target → reflect.Type → V) registry
// dimensions, panicking on per-(target, type) collision with a message
// that names the dimension. Pulled out of MergeFrom because the same
// nested-map walk applies to emitters, rewriters, and capabilities.
func mergeByTypeMap[V any](
	dst, src map[Target]map[reflect.Type]V,
	newInner func() map[reflect.Type]V,
	dimensionName string,
) {
	for target, byType := range src {
		if dst[target] == nil {
			dst[target] = newInner()
		}
		for resolvedType, value := range byType {
			if _, exists := dst[target][resolvedType]; exists {
				panic(fmt.Sprintf(
					"duplicate registration: %s for (target=%q, type=%s)",
					dimensionName, target, resolvedType,
				))
			}
			dst[target][resolvedType] = value
		}
	}
}
