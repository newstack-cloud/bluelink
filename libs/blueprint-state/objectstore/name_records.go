package objectstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// How long a name reservation whose instance object was never written is assumed to still
// be in progress. Writing the object follows the reservation immediately, so this only has
// to outlast a store's retries and backoff on that write.
const abandonedNameReservationAge = 10 * time.Minute

type instanceLiveness int

const (
	instanceAbsent instanceLiveness = iota
	instanceLive
	// The instance object is still there, but its removal has started.
	instanceRemoving
)

type nameReservation struct {
	key        string
	record     []byte
	instanceID string
	now        time.Time
}

// Reserves an instance name with a conditional write, so two engines creating instances
// under one name are serialised at the bucket. Reports whether this call wrote the record,
// which is what decides whether a failed create should give it back.
//
// A name already held is free to take when its record names this instance, names one whose
// removal was interrupted (that removal is finished first, see finishRemoval), or was
// reserved for an instance that was never written and is older than
// abandonedNameReservationAge.
func reserveName(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	clock commoncore.Clock,
	name string,
	instanceID string,
) (bool, error) {
	if name == "" {
		return false, nil
	}

	now := clock.Now()
	record, err := json.Marshal(&statestore.InstanceNameRecord{
		ID:         instanceID,
		Name:       name,
		ReservedAt: now.Unix(),
	})
	if err != nil {
		return false, err
	}

	reservation := &nameReservation{
		key:        keys.InstanceByName(name),
		record:     record,
		instanceID: instanceID,
		now:        now,
	}

	// A second attempt covers a record that is gone by the time it is read, deleted by a
	// removal finishing (possibly this call's) or by a create giving its reservation back.
	for range 2 {
		_, err = svc.Put(ctx, reservation.key, record, &PutOptions{IfNoneMatch: "*"})
		if err == nil {
			return true, nil
		}
		if !isPreconditionFailed(err) {
			return false, err
		}
		reserved, err := takeOverName(ctx, svc, keys, reservation)
		if !errors.Is(err, errNameFreed) && !isObjectNotFound(err) {
			return reserved, err
		}
	}

	return false, state.ErrInstanceAlreadyExists
}

var errNameFreed = errors.New("objectstore: name record freed by finishing a removal")

func takeOverName(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	reservation *nameReservation,
) (bool, error) {
	data, etag, err := svc.Get(ctx, reservation.key)
	if err != nil {
		return false, err
	}

	var held statestore.InstanceNameRecord
	if err := json.Unmarshal(data, &held); err != nil {
		return false, err
	}

	liveness, err := readInstanceLiveness(ctx, svc, keys, held.ID)
	if err != nil {
		return false, err
	}

	switch {
	case liveness == instanceLive:
		return false, state.ErrInstanceAlreadyExists
	case liveness == instanceRemoving:
		// Finishing the removal deletes this record, so the reservation is then retried.
		if err := finishRemoval(ctx, svc, keys, held.ID); err != nil {
			return false, err
		}
		return false, errNameFreed
	case held.ID == reservation.instanceID:
		return false, nil
	case !isAbandoned(&held, reservation.now):
		return false, state.ErrInstanceAlreadyExists
	}

	if _, err := svc.Put(ctx, reservation.key, reservation.record, &PutOptions{IfMatch: etag}); err != nil {
		if isPreconditionFailed(err) {
			return false, state.ErrInstanceAlreadyExists
		}
		return false, err
	}

	return true, nil
}

// Every reservation this package writes carries ReservedAt, so a record without one cannot
// belong to a create in progress.
func isAbandoned(held *statestore.InstanceNameRecord, now time.Time) bool {
	if held.ReservedAt == 0 {
		return true
	}

	return now.Sub(time.Unix(held.ReservedAt, 0)) >= abandonedNameReservationAge
}

func readInstanceLiveness(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	instanceID string,
) (instanceLiveness, error) {
	var persisted statestore.PersistedInstanceState
	found, err := readEntity(ctx, svc, keys.Instance(instanceID), &persisted)
	if err != nil || !found {
		return instanceAbsent, err
	}

	if persisted.Removing {
		return instanceRemoving, nil
	}

	return instanceLive, nil
}
