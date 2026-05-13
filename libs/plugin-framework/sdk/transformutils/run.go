package transformutils

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Run holds per-pipeline-call state.
// The framework allocates one Run per RunTransformPipeline call,
// threads it as an explicit *Run parameter to every phase, and exposes
// typed Provide/Use helpers for arbitrary plugin-defined run-scoped values.
// Run is not shared across pipeline calls; each call allocates a fresh instance,
// so concurrent runs are isolated from each other.
type Run struct {
	// TransformContext is the per-run transformer context.
	// Phases use it for config and context variables.
	TransformContext transform.Context

	mu      sync.RWMutex
	storage map[reflect.Type]any
}

// Provide stores a typed value on the run, keyed by Go type.
// This is intended to be called from OnRun, but a phase can also provide
// values for later phases in the same run if needed.
//
// Two values with the same Go type collide on the same key.
// If a plugin needs to distinguish (e.g. two []string configs), use newtype
// wrappers (`type BuildManifestPath string`) to disambiguate.
func Provide[T any](run *Run, value T) {
	run.mu.Lock()
	defer run.mu.Unlock()

	if run.storage == nil {
		run.storage = make(map[reflect.Type]any)
	}
	run.storage[reflect.TypeFor[T]()] = value
}

// Use retrieves a typed value previously stored by Provide.
// Returns (zero, false) of no value of type T has been provided on this run.
func Use[T any](run *Run) (T, bool) {
	run.mu.RLock()
	defer run.mu.RUnlock()

	if run.storage == nil {
		var zero T
		return zero, false
	}

	value, ok := run.storage[reflect.TypeFor[T]()]
	if !ok {
		var zero T
		return zero, false
	}
	return value.(T), true
}

// MustUse is a helper to retrieve the same value as Use but
// panics if no value of type T has been provided on this run.
func MustUse[T any](run *Run) T {
	value, ok := Use[T](run)
	if !ok {
		panic(
			fmt.Sprintf(
				"MustUse: no value of type %s found on run",
				reflect.TypeFor[T]().String(),
			),
		)
	}
	return value
}
