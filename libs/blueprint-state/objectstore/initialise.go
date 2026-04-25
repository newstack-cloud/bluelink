package objectstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// InitialiseAndClaim performs an atomic create-if-absent of a new instance
// object at version 1 with the given status. The conditional Put with
// IfNoneMatch: "*" is what serialises concurrent first-deploys of the same
// instance ID sharing a bucket.
//
// When the instance has a name the companion name-lookup record is
// written alongside it so LookupInstanceIDByName can resolve under
// ModeLazy without enumerating the instances prefix. A concurrent
// different-ID-same-name collision on the name record is also mapped to
// state.ErrInstanceAlreadyExists; the caller can distinguish by a
// subsequent read if needed.
//
// Returns state.ErrInstanceAlreadyExists when the instance object already
// exists (412 on IfNoneMatch).
func InitialiseAndClaim(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
	instanceState state.InstanceState,
	newStatus core.InstanceStatus,
) (int64, error) {
	persisted := statestore.NewPersistedInstanceState(&instanceState)
	persisted.Status = newStatus
	persisted.Version = 1
	persisted.LastStatusUpdateTimestamp = int(time.Now().Unix())

	data, err := json.Marshal(persisted)
	if err != nil {
		return 0, err
	}

	key := keys.Instance(instanceState.InstanceID)
	_, putErr := svc.Put(ctx, key, data, &PutOptions{IfNoneMatch: "*"})
	if putErr != nil {
		if isPreconditionFailed(putErr) {
			return 0, state.ErrInstanceAlreadyExists
		}
		return 0, putErr
	}

	if err := writeNameRecord(ctx, svc, keys, &instanceState); err != nil {
		return 0, err
	}

	st.SetInstanceInMemory(persisted.ToInstanceState())
	return persisted.Version, nil
}

func writeNameRecord(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	inst *state.InstanceState,
) error {
	if inst.InstanceName == "" {
		return nil
	}
	record := &statestore.InstanceNameRecord{
		ID:   inst.InstanceID,
		Name: inst.InstanceName,
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = svc.Put(ctx, keys.InstanceByName(inst.InstanceName), data, &PutOptions{IfNoneMatch: "*"})
	if err != nil {
		if isPreconditionFailed(err) {
			return state.ErrInstanceAlreadyExists
		}
		return err
	}
	return nil
}
