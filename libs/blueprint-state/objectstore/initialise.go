package objectstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// InitialiseAndClaim performs an atomic create-if-absent of a new instance
// object at version 1 with the given status. The conditional Put with
// IfNoneMatch: "*" is what serialises concurrent first-deploys of the same
// instance ID sharing a bucket.
//
// When the instance has a name, its name-lookup record is reserved first (see
// reserveName), so LookupInstanceIDByName can resolve under ModeLazy without
// enumerating the instances prefix. A name held by another live instance is
// mapped to state.ErrInstanceAlreadyExists; the caller can distinguish by a
// subsequent read if needed. If the instance object cannot be written at all,
// a reservation this call made is given back.
//
// An instance created with resources or links is first written without them,
// then their ID records, then the whole instance, so no resource or link is
// ever stored without its record. An earlier instance with this ID whose
// removal was interrupted reads as not found everywhere else, so its removal
// is finished first (see finishRemoval) and the ID used again.
//
// Returns state.ErrInstanceAlreadyExists when the instance object already
// exists (412 on IfNoneMatch).
func InitialiseAndClaim(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
	clock commoncore.Clock,
	instanceState state.InstanceState,
	newStatus core.InstanceStatus,
) (int64, error) {
	persisted := statestore.NewPersistedInstanceState(&instanceState)
	persisted.Status = newStatus
	persisted.Version = 1
	persisted.LastStatusUpdateTimestamp = int(clock.Now().Unix())

	if err := finishRemoval(ctx, svc, keys, instanceState.InstanceID); err != nil {
		return 0, err
	}
	reserved, err := reserveName(
		ctx, svc, keys, clock, instanceState.InstanceName, instanceState.InstanceID,
	)
	if err != nil {
		return 0, err
	}

	instanceWritten, err := createInstanceObject(ctx, svc, keys, persisted)
	if err != nil {
		if reserved && !instanceWritten {
			giveBackName(ctx, svc, keys, instanceState, err)
		}
		return 0, err
	}

	if _, err := st.CachePersistedInstance(ctx, persisted); err != nil {
		return 0, err
	}
	return persisted.Version, nil
}

// This is best effort. When an instance with this ID already exists under the reserved name, the
// record is right even though this call did not create the instance, a concurrent caller
// creating the same instance did.
func giveBackName(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	instanceState state.InstanceState,
	createErr error,
) {
	if errors.Is(createErr, state.ErrInstanceAlreadyExists) {
		var existing statestore.PersistedInstanceState
		found, err := readEntity(ctx, svc, keys.Instance(instanceState.InstanceID), &existing)
		if err != nil || (found && existing.InstanceName == instanceState.InstanceName) {
			return
		}
	}
	_ = svc.Delete(ctx, keys.InstanceByName(instanceState.InstanceName), nil)
}

// Reports whether the instance object was written. The caller gives back the name it
// reserved only when it was not written.
// Once the object exists, even as the shell without its
// resources and links, the name record is how the instance is found by name, so a failure
// in a later step must leave the record in place.
func createInstanceObject(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	persisted *statestore.PersistedInstanceState,
) (bool, error) {
	entities := persisted.EntityIDs()
	hasEntities := len(entities.Resources) > 0 || len(entities.Links) > 0
	first := persisted
	if hasEntities {
		first = persisted.WithoutEntities()
	}

	etag, err := putNewInstanceObject(ctx, svc, keys, first)
	if err != nil {
		return false, err
	}
	if !hasEntities {
		persisted.ETag = etag
		return true, nil
	}

	if err := putEntityRecords(ctx, svc, keys, persisted.InstanceID, entities); err != nil {
		return true, err
	}
	persisted.ETag, err = putJSON(
		ctx, svc, keys.Instance(persisted.InstanceID), persisted, &PutOptions{IfMatch: etag},
	)
	return true, err
}

// Creates the instance object only where none exists. An interrupted removal of the same ID
// was already finished by InitialiseAndClaim, so an object here is a live instance or one
// another process created or started removing since, and either way this ID is taken.
func putNewInstanceObject(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	persisted *statestore.PersistedInstanceState,
) (string, error) {
	etag, err := putJSON(ctx, svc, keys.Instance(persisted.InstanceID), persisted, &PutOptions{IfNoneMatch: "*"})
	if isPreconditionFailed(err) {
		return "", state.ErrInstanceAlreadyExists
	}

	return etag, err
}

func putEntityRecords(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	instanceID string,
	entities statestore.InstanceEntityIDs,
) error {
	record := &statestore.EntityRecord{InstanceID: instanceID}
	for _, resourceID := range entities.Resources {
		if _, err := putJSON(ctx, svc, keys.ResourceRecord(resourceID), record, nil); err != nil {
			return err
		}
	}

	for _, linkID := range entities.Links {
		if _, err := putJSON(ctx, svc, keys.LinkRecord(linkID), record, nil); err != nil {
			return err
		}
	}
	return nil
}

func putJSON(ctx context.Context, svc Service, key string, value any, opts *PutOptions) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return svc.Put(ctx, key, data, opts)
}
