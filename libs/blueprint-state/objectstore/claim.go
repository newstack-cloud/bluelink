package objectstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ClaimForDeployment performs an atomic compare-and-swap on a single instance
// object in the Service. On success the instance's version is bumped, its
// status is set to newStatus, and the in-memory State is refreshed.
//
// The CAS bypasses statestore.Storage deliberately — statestore.Storage has
// no ETag awareness, and the conditional write is what makes this safe for
// concurrent executions such as CI/CD runs sharing a bucket storage backend.
//
// Returns state.InstanceNotFoundError when the instance object does not
// exist, and state.ErrVersionConflict when the persisted version does not
// match expectedVersion (either caught cheaply before the write, or via a
// 412 on IfMatch).
func ClaimForDeployment(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
	instanceID string,
	expectedVersion int64,
	newStatus core.InstanceStatus,
) (int64, error) {
	key := keys.Instance(instanceID)

	persisted, etag, err := readPersistedInstance(ctx, svc, key, instanceID)
	if err != nil {
		return 0, err
	}

	if persisted.Version != expectedVersion {
		return persisted.Version, state.ErrVersionConflict
	}

	persisted.Version++
	persisted.Status = newStatus
	persisted.LastStatusUpdateTimestamp = int(time.Now().Unix())

	data, err := json.Marshal(persisted)
	if err != nil {
		return 0, err
	}

	_, putErr := svc.Put(ctx, key, data, &PutOptions{IfMatch: etag})
	if putErr != nil {
		if isPreconditionFailed(putErr) {
			return currentPersistedVersion(ctx, svc, key, instanceID)
		}
		return 0, putErr
	}

	st.SetInstanceInMemory(persisted.ToInstanceState())
	return persisted.Version, nil
}

func readPersistedInstance(
	ctx context.Context,
	svc Service,
	key, instanceID string,
) (*statestore.PersistedInstanceState, string, error) {
	data, etag, err := svc.Get(ctx, key)
	if err != nil {
		if isObjectNotFound(err) {
			return nil, "", state.InstanceNotFoundError(instanceID)
		}
		return nil, "", err
	}
	var persisted statestore.PersistedInstanceState
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, "", err
	}
	return &persisted, etag, nil
}

func currentPersistedVersion(
	ctx context.Context,
	svc Service,
	key, instanceID string,
) (int64, error) {
	persisted, _, err := readPersistedInstance(ctx, svc, key, instanceID)
	if err != nil {
		return 0, err
	}
	return persisted.Version, state.ErrVersionConflict
}

func isPreconditionFailed(err error) bool {
	var sErr *Error
	if !errors.As(err, &sErr) {
		return false
	}
	return sErr.ReasonCode == ErrorReasonCodePreconditionFailed
}
