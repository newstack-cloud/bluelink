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
// Every method returns (value, found, error): the value and true when found, the zero
// value and false when it does not exist, and the zero value with an error on a transient
// failure, which the caller decides how to handle.
//
// LoadInstance returns the instance as persisted, with child blueprints as ID references,
// which State resolves when it caches the instance.
type EntityLoader interface {
	LoadInstance(ctx context.Context, id string) (*PersistedInstanceState, bool, error)
	LoadInstanceIDByName(ctx context.Context, name string) (string, bool, error)
	// LoadResourceInstanceID and LoadLinkInstanceID resolve an entity's ID record to the
	// instance said to hold it. State loads that instance and checks it still holds the entity.
	LoadResourceInstanceID(ctx context.Context, resourceID string) (string, bool, error)
	LoadLinkInstanceID(ctx context.Context, linkID string) (string, bool, error)
	LoadResourceDrift(ctx context.Context, id string) (*state.ResourceDriftState, bool, error)
	LoadLinkDrift(ctx context.Context, id string) (*state.LinkDriftState, bool, error)
	LoadEvent(ctx context.Context, id string) (*manage.Event, bool, error)
	LoadChangeset(ctx context.Context, id string) (*manage.Changeset, bool, error)
	LoadValidation(ctx context.Context, id string) (*manage.BlueprintValidation, bool, error)
	LoadReconciliation(ctx context.Context, id string) (*manage.ReconciliationResult, bool, error)
	LoadCleanupOperation(ctx context.Context, id string) (*manage.CleanupOperation, bool, error)
	LoadAuditRecord(ctx context.Context, id string) (*manage.AuditRecord, bool, error)
}

// EntityLister is implemented by an EntityLoader that can enumerate the entities of a
// category in storage. Under ModeLazy, queries that span a whole category (listing
// instances, finding the latest reconciliation result or cleanup operation) list the
// category and load each entity before answering from the cache, so they cover what
// other processes wrote rather than only what this one has loaded.
type EntityLister interface {
	ListInstanceIDs(ctx context.Context) ([]string, error)
	ListReconciliationResultIDs(ctx context.Context) ([]string, error)
	ListCleanupOperationIDs(ctx context.Context) ([]string, error)
}

// noopLoader is the loader used under ModeEager — it never materialises
// anything because the State is assumed to hold the full picture.
type noopLoader struct{}

func (noopLoader) LoadInstance(context.Context, string) (*PersistedInstanceState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadInstanceIDByName(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (noopLoader) LoadResourceInstanceID(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (noopLoader) LoadResourceDrift(context.Context, string) (*state.ResourceDriftState, bool, error) {
	return nil, false, nil
}

func (noopLoader) LoadLinkInstanceID(context.Context, string) (string, bool, error) {
	return "", false, nil
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

func (noopLoader) LoadAuditRecord(context.Context, string) (*manage.AuditRecord, bool, error) {
	return nil, false, nil
}
