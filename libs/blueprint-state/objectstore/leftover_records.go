package objectstore

import (
	"context"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// Deletes what interrupted multi-object writes leave in the bucket that nothing else would
// ever delete:
//   - resource and link ID records whose instance is gone, is being removed, or no longer
//     holds the resource or link;
//   - resource and link drift whose resource or link no longer exists;
//   - name records naming an instance being removed, or an abandoned reservation;
//   - instances whose removal was interrupted after they were marked.
//
// Nothing is judged until it is older than leftoverRecordGracePeriod, because a record
// legitimately precedes the instance write that makes it valid. Every delete is conditional on
// the version that was judged, so anything rewritten since is left alone and two engines
// sweeping at once cannot delete each other's work.
//
// This is the objectstore backend's housekeeping (manage.Housekeeping).
type leftoverRecords struct {
	svc   Service
	keys  statestore.KeyBuilder
	clock func() commoncore.Clock
}

// How old a record has to be before the sweep judges it. A record younger than this may
// belong to a write still in progress and deleting the ID record of a create between its record
// and instance writes would leave a resource that cannot be found by ID. So this is part of
// the write protocol rather than a setting, kept far longer than any write sequence takes,
// retries and backoff included, and longer than abandonedNameReservationAge.
const leftoverRecordGracePeriod = time.Hour

var _ manage.Housekeeping = (*leftoverRecords)(nil)

func (l *leftoverRecords) Run(ctx context.Context) (int64, error) {
	run := &sweepRun{
		leftoverRecords: l,
		threshold:       l.clock().Now().Add(-leftoverRecordGracePeriod),
		instances:       map[string]*sweptInstance{},
	}

	// Records go before the instances they name, so no record is left naming an instance
	// object that no longer exists. Drift is judged through its record, so it follows them.
	steps := []func(context.Context) error{
		func(ctx context.Context) error {
			return run.sweepEntityRecords(ctx, l.keys.ResourceRecordsPrefix(), resourceEntity)
		},
		func(ctx context.Context) error {
			return run.sweepEntityRecords(ctx, l.keys.LinkRecordsPrefix(), linkEntity)
		},
		func(ctx context.Context) error {
			return run.sweepDrift(ctx, l.keys.ResourceDriftPrefix(), l.keys.ResourceRecord, resourceEntity)
		},
		func(ctx context.Context) error {
			return run.sweepDrift(ctx, l.keys.LinkDriftPrefix(), l.keys.LinkRecord, linkEntity)
		},
		run.sweepNameRecords,
		run.sweepMarkedInstances,
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return run.deleted, err
		}
	}
	return run.deleted, nil
}

type entityKind int

const (
	resourceEntity entityKind = iota
	linkEntity
)

type sweepRun struct {
	*leftoverRecords
	threshold time.Time
	// Each instance is read once per run, however many records name it.
	instances map[string]*sweptInstance
	deleted   int64
}

type sweptInstance struct {
	found       bool
	removing    bool
	etag        string
	resourceIDs map[string]bool
	linkIDs     map[string]bool
}

func (r *sweepRun) sweepEntityRecords(ctx context.Context, prefix string, kind entityKind) error {
	objects, err := r.listJudgeable(ctx, prefix)
	if err != nil {
		return err
	}

	for _, object := range objects {
		var record statestore.EntityRecord
		etag, unchanged, err := r.readListedVersion(ctx, object, &record)
		if err != nil {
			return err
		}
		if !unchanged {
			continue
		}

		held, err := r.isHeld(ctx, record.InstanceID, idFromKey(prefix, object.Key), kind)
		if err != nil {
			return err
		}
		if !held {
			if err := r.deleteAtVersion(ctx, object.Key, etag); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *sweepRun) sweepDrift(
	ctx context.Context,
	prefix string,
	recordKey func(entityID string) string,
	kind entityKind,
) error {
	objects, err := r.listJudgeable(ctx, prefix)
	if err != nil {
		return err
	}

	for _, object := range objects {
		entityID := idFromKey(prefix, object.Key)
		var record statestore.EntityRecord
		found, err := readEntity(ctx, r.svc, recordKey(entityID), &record)
		if err != nil {
			return err
		}

		held := false
		if found {
			if held, err = r.isHeld(ctx, record.InstanceID, entityID, kind); err != nil {
				return err
			}
		}
		if !held {
			if err := r.deleteAtVersion(ctx, object.Key, object.ETag); err != nil {
				return err
			}
		}
	}
	return nil
}

// A record naming a live instance is never leftover. One naming an instance with no object is
// only leftover once its reservation is abandoned, as a reservation precedes the instance write.
func (r *sweepRun) sweepNameRecords(ctx context.Context) error {
	objects, err := r.listJudgeable(ctx, r.keys.InstanceNamesPrefix())
	if err != nil {
		return err
	}

	now := r.clock().Now()
	for _, object := range objects {
		var record statestore.InstanceNameRecord
		etag, unchanged, err := r.readListedVersion(ctx, object, &record)
		if err != nil {
			return err
		}
		if !unchanged {
			continue
		}

		instance, err := r.instance(ctx, record.ID)
		if err != nil {
			return err
		}
		if instance.removing || (!instance.found && isAbandoned(&record, now)) {
			if err := r.deleteAtVersion(ctx, object.Key, etag); err != nil {
				return err
			}
		}
	}
	return nil
}

// A marked instance's records were written before its marker, so once the marker is old
// enough to judge, so are they, and the steps before this one have deleted them.
func (r *sweepRun) sweepMarkedInstances(ctx context.Context) error {
	prefix := r.keys.InstancesPrefix()
	objects, err := r.listJudgeable(ctx, prefix)
	if err != nil {
		return err
	}

	for _, object := range objects {
		instance, err := r.instance(ctx, idFromKey(prefix, object.Key))
		if err != nil {
			return err
		}

		if instance.removing && instance.etag == object.ETag {
			if err := r.deleteAtVersion(ctx, object.Key, instance.etag); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *sweepRun) isHeld(ctx context.Context, instanceID string, entityID string, kind entityKind) (bool, error) {
	instance, err := r.instance(ctx, instanceID)
	if err != nil || !instance.found || instance.removing {
		return false, err
	}

	if kind == resourceEntity {
		return instance.resourceIDs[entityID], nil
	}

	return instance.linkIDs[entityID], nil
}

func (r *sweepRun) instance(ctx context.Context, instanceID string) (*sweptInstance, error) {
	if instance, ok := r.instances[instanceID]; ok {
		return instance, nil
	}

	var persisted statestore.PersistedInstanceState
	etag, found, err := readVersionedEntity(ctx, r.svc, r.keys.Instance(instanceID), &persisted)
	if err != nil {
		return nil, err
	}

	instance := &sweptInstance{
		found:       found,
		removing:    persisted.Removing,
		etag:        etag,
		resourceIDs: map[string]bool{},
		linkIDs:     map[string]bool{},
	}
	ids := persisted.EntityIDs()
	for _, resourceID := range ids.Resources {
		instance.resourceIDs[resourceID] = true
	}

	for _, linkID := range ids.Links {
		instance.linkIDs[linkID] = true
	}

	r.instances[instanceID] = instance

	return instance, nil
}

// A store that does not report when an object was written leaves it with a zero time, which
// is treated as too recent to judge rather than as old.
func (r *sweepRun) listJudgeable(ctx context.Context, prefix string) ([]*ObjectInfo, error) {
	objects, err := r.svc.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	judgeable := make([]*ObjectInfo, 0, len(objects))
	for _, object := range objects {
		if !object.LastModified.IsZero() && object.LastModified.Before(r.threshold) {
			judgeable = append(judgeable, object)
		}
	}
	return judgeable, nil
}

// The age that made an object judgeable belongs to the version that was listed, so a version
// written since is too recent to judge.
func (r *sweepRun) readListedVersion(ctx context.Context, object *ObjectInfo, into any) (string, bool, error) {
	etag, found, err := readVersionedEntity(ctx, r.svc, object.Key, into)
	if err != nil || !found {
		return "", false, err
	}
	return etag, etag == object.ETag, nil
}

func (r *sweepRun) deleteAtVersion(ctx context.Context, key string, etag string) error {
	err := r.svc.Delete(ctx, key, &DeleteOptions{IfMatch: etag})
	if err == nil {
		r.deleted += 1
		return nil
	}
	if isObjectNotFound(err) || isPreconditionFailed(err) {
		return nil
	}
	return err
}

func idFromKey(prefix string, key string) string {
	return strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".json")
}
