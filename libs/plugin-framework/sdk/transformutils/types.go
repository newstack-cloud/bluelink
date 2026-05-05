package transformutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// EmitPlan represents the output of the aggregation phase that is fed into
// the concrete output emission phase.
type EmitPlan struct {
	// Primaries are the main resolved resources that will be emitted as concrete output.
	Primaries []ResolvedResource
	// SharedParents allow aggregates to declare concrete output resources
	// that don't correspond to any single abstract input.
	// This lets  pre-primary emits populate shared parents incrementally so
	// partial-fold targets (e.g. Azure Function, Cloud Run, VPC Connector)
	// can produce per-resource outputs and shared resource output in the same
	// pass without giving up the the per-primary 1:1 mapping that a full fold
	// implementation would otherwise require.
	SharedParents []SharedParent
}

// SharedParent represents a declared concrete output resource that
// doesn't correspond to a single abstract input resource.
type SharedParent struct {
	Key          string
	ResourceName string
	ResourceType string
	Annotations  *core.MappingNode
	SeedSpec     *core.MappingNode
}

// EmitResult represents the output of the emission phase for a single resolved resource.
type EmitResult struct {
	Resources     map[string]*schema.Resource
	DerivedValues map[string]*schema.Value
	// Mapping of shared parent key to the contributions
	// from all resources that share this parent.
	SharedParentContributions map[string]*core.MappingNode
	Diagnostics               []*core.Diagnostic
}

// ResolvedResource is the target-agnostic output of the resolve phase
// that is fed into aggregation, rewriting and emission.
// Concrete resource types (e.g. *ResolvedQueue, *ResolvedHandler) live
// in their resource packages and implement this interface.
type ResolvedResource interface {
	ResourceName() string
	ResourceType() string
}

// ResolvedPtr is a generic helper interface to ensure
// type safety on resolved resource types.
// Concrete resolved types implement ResolvedResource with
// pointer receivers, so the constraint is expressed via
// ResolvedPtr[T]; "PR is *T and *T satisfies ResolvedResource".
type ResolvedPtr[T any] interface {
	*T
	ResolvedResource
}

// EmitterRegistration is a deferred registration closure
// that captures the concrete resolved type.
// This is necessary to allow authors to write their emitters with concrete resolved types
// while maintaing a valid homogenous map value type for emitter storage
// in registries.
type EmitterRegistration func(registry *TransformerRegistry, t Target)

// RewriterRegistration is a deferred registration closure that captures the concrete resolved type for rewriters.
// This is necessary to allow authors to write their rewriters with concrete resolved types
// while maintaing a valid homogenous map value type for rewriter storage
// in registries.
type RewriterRegistration func(registry *TransformerRegistry, t Target)
