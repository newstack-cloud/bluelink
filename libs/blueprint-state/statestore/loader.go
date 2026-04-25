package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Load prepares state for use according to cfg.Mode.
//
// Under ModeLazy state starts empty; entries materialise on demand via
// state's EntityLoader (set at construction via WithEntityLoader). Load
// is a no-op in this mode.
//
// Under ModeEager Load walks every key under prefix in storage, classifies
// each by its canonical KeyBuilder filename, unmarshals into state's maps,
// re-wires the instance parent/child pointer graph, and rebuilds the
// name-lookup cache. Missing storage (empty dir / empty bucket) is not an
// error — state is left with its initialised-but-empty maps.
func Load(ctx context.Context, state *State, storage Storage, cfg Config, prefix string) error {
	if cfg.Mode != ModeEager {
		return nil
	}
	return loadEager(ctx, state, storage, prefix)
}

// LoadMode picks how State behaves on a lookup that misses the cache.
type LoadMode int

const (
	// ModeEager populates the full State at construction time. Cache misses
	// imply the entity does not exist. Appropriate for memfile (local,
	// bounded data) and postgres-style backends where the database is the
	// live source of truth.
	ModeEager LoadMode = iota

	// ModeLazy leaves State empty at construction and materialises entities
	// on demand via EntityLoader. Appropriate for objectstore backends where
	// the state set can be large and a given process only touches a small
	// working subset.
	ModeLazy
)

// EntityLoader fetches a single entity from Storage when a State cache miss
// occurs under ModeLazy. The concrete implementation lives in each backend:
// memfile supplies a no-op loader (ModeEager); objectstore supplies one that
// reads and unmarshals objects from its Service.
//
// Every method returns (entity, found, error):
//   - entity non-nil, found true, err nil  — entity materialised
//   - entity nil, found false, err nil     — entity does not exist
//   - entity nil, found false, err non-nil — transient error; caller decides
type EntityLoader interface {
	LoadInstance(ctx context.Context, id string) (*state.InstanceState, bool, error)
	LoadInstanceIDByName(ctx context.Context, name string) (string, bool, error)
	LoadResource(ctx context.Context, id string) (*state.ResourceState, bool, error)
	LoadResourceDrift(ctx context.Context, id string) (*state.ResourceDriftState, bool, error)
	LoadLink(ctx context.Context, id string) (*state.LinkState, bool, error)
	LoadLinkDrift(ctx context.Context, id string) (*state.LinkDriftState, bool, error)
	LoadEvent(ctx context.Context, id string) (*manage.Event, bool, error)
	LoadChangeset(ctx context.Context, id string) (*manage.Changeset, bool, error)
	LoadValidation(ctx context.Context, id string) (*manage.BlueprintValidation, bool, error)
	LoadReconciliation(ctx context.Context, id string) (*manage.ReconciliationResult, bool, error)
	LoadCleanupOperation(ctx context.Context, id string) (*manage.CleanupOperation, bool, error)
}

// noopLoader is the loader used under ModeEager — it never materialises
// anything because the State is assumed to hold the full picture.
type noopLoader struct{}

func (noopLoader) LoadInstance(context.Context, string) (*state.InstanceState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadInstanceIDByName(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (noopLoader) LoadResource(context.Context, string) (*state.ResourceState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadResourceDrift(context.Context, string) (*state.ResourceDriftState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadLink(context.Context, string) (*state.LinkState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadLinkDrift(context.Context, string) (*state.LinkDriftState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadEvent(context.Context, string) (*manage.Event, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadChangeset(context.Context, string) (*manage.Changeset, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadValidation(context.Context, string) (*manage.BlueprintValidation, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadReconciliation(context.Context, string) (*manage.ReconciliationResult, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadCleanupOperation(context.Context, string) (*manage.CleanupOperation, bool, error) {
	return nil, false, nil
}
