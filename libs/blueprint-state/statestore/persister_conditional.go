package statestore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Instance objects are written conditionally when the storage versions its objects and each
// instance is its own object. Chunked instances share an object, so a version would guard the
// whole chunk rather than one instance.
func (p *Persister) conditionalInstanceStorage() (ConditionalStorage, bool) {
	if p.config.Instances.Layout != LayoutPerEntity {
		return nil, false
	}

	conditional, ok := p.storage.(ConditionalStorage)
	return conditional, ok
}

// Writes an instance object under the per-entity layout. With conditional storage, a new
// instance is only written where none exists and an existing one only over the version that
// was cached, so a change another process made since is reported (as an instanceChangedError)
// rather than overwritten. Expects the State write lock to be held.
func (p *Persister) putInstanceObject(
	ctx context.Context,
	persisted *PersistedInstanceState,
	isNew bool,
) error {
	key := p.keys.Instance(persisted.InstanceID)
	conditional, ok := p.conditionalInstanceStorage()
	if !ok {
		return writeChunk(ctx, p.storage, key, persisted)
	}

	version := ""
	if !isNew {
		cachedVersion, known, stale := p.state.cachedInstanceVersion(persisted.InstanceID)
		if stale {
			return &instanceChangedError{instanceID: persisted.InstanceID}
		}
		if !known {
			// Every instance cached from versioned storage has its version, so this is not
			// expected. Writing anyway matches storage without versions, and leaving no version
			// recorded has revalidation reload the instance.
			return writeChunk(ctx, p.storage, key, persisted)
		}
		version = cachedVersion
	}

	data, err := json.Marshal(persisted)
	if err != nil {
		return err
	}

	newVersion, err := conditional.WriteIfMatch(ctx, key, data, version)
	if errors.Is(err, ErrPreconditionFailed) {
		return &instanceChangedError{instanceID: persisted.InstanceID}
	}
	if err != nil {
		return err
	}

	p.state.recordInstanceVersion(persisted.InstanceID, newVersion)
	return nil
}

// Deletes an instance object, with conditional storage only while it is still the version
// this process last wrote. A different version means another process finished this removal
// and created a new instance with the same ID, which is left alone. Expects the State write
// lock to be held.
func (p *Persister) deleteInstanceObject(ctx context.Context, instanceID string) error {
	key := p.keys.Instance(instanceID)
	conditional, ok := p.conditionalInstanceStorage()
	version, known, stale := p.state.cachedInstanceVersion(instanceID)
	if ok && stale {
		return &instanceChangedError{instanceID: instanceID}
	}
	delete(p.state.instanceVersions, instanceID)
	if !ok || !known {
		return deleteIgnoreNotFound(ctx, p.storage, key)
	}

	err := conditional.DeleteIfMatch(ctx, key, version)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrPreconditionFailed) {
		return nil
	}
	return err
}

// As deleteInstanceObject, a name record that changed after it was read belongs to another
// instance by the time it would be deleted, so it is left alone.
func (p *Persister) deleteNameRecordIfOwned(
	ctx context.Context,
	conditional ConditionalStorage,
	key string,
	instanceID string,
) error {
	data, version, err := conditional.ReadVersion(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var record InstanceNameRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return err
	}
	if record.ID != instanceID {
		return nil
	}

	err = conditional.DeleteIfMatch(ctx, key, version)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrPreconditionFailed) {
		return nil
	}
	return err
}

// After a failed instance write, the cache holds a change that storage does not. A write rejected
// because another process changed the instance only needs that instance reloaded, which the
// mutator does before applying its change again. Any other failure drops the cached instances
// (see forgetCachedInstances). Expects the State write lock to be held.
func (p *Persister) forgetAfterFailedWrite(err error) {
	if changed, ok := errors.AsType[*instanceChangedError](err); ok {
		p.state.markInstanceStale(changed.instanceID)
		return
	}
	p.state.forgetCachedInstances()
}

func isInstanceChanged(err error) bool {
	var changed *instanceChangedError
	return errors.As(err, &changed)
}

func newRemovalMarker(instance *state.InstanceState) *PersistedInstanceState {
	marked := NewPersistedInstanceState(instance)
	marked.Removing = true
	return marked
}
