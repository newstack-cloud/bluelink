package objectstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ServiceLoader reads entities from the objectstore Service on a State cache miss,
// and lists categories for whole-category queries. This lets objectstore run under
// statestore.ModeLazy without bulk-loading the bucket at startup.
type ServiceLoader struct {
	svc  Service
	keys statestore.KeyBuilder
}

func NewServiceLoader(svc Service, keys statestore.KeyBuilder) *ServiceLoader {
	return &ServiceLoader{svc: svc, keys: keys}
}

func (l *ServiceLoader) LoadInstance(
	ctx context.Context,
	id string,
) (*statestore.PersistedInstanceState, bool, error) {
	var persisted statestore.PersistedInstanceState
	found, err := readEntity(ctx, l.svc, l.keys.Instance(id), &persisted)
	if err != nil || !found {
		return nil, found, err
	}
	return &persisted, true, nil
}

func (l *ServiceLoader) LoadInstanceIDByName(
	ctx context.Context,
	name string,
) (string, bool, error) {
	var record statestore.InstanceNameRecord
	found, err := readEntity(ctx, l.svc, l.keys.InstanceByName(name), &record)
	if err != nil || !found {
		return "", found, err
	}
	return record.ID, true, nil
}

// LoadResourceInstanceID reads a resource's ID record. Resources are persisted inside
// their instance's object, so the record only says which instance to load.
func (l *ServiceLoader) LoadResourceInstanceID(
	ctx context.Context,
	resourceID string,
) (string, bool, error) {
	return l.loadEntityRecord(ctx, l.keys.ResourceRecord(resourceID))
}

// LoadLinkInstanceID reads a link's ID record, which says which instance to load.
func (l *ServiceLoader) LoadLinkInstanceID(
	ctx context.Context,
	linkID string,
) (string, bool, error) {
	return l.loadEntityRecord(ctx, l.keys.LinkRecord(linkID))
}

// ListInstanceIDs lists the IDs of every instance object in the bucket.
func (l *ServiceLoader) ListInstanceIDs(ctx context.Context) ([]string, error) {
	return l.listEntityIDs(ctx, l.keys.InstancesPrefix())
}

// ListReconciliationResultIDs lists the IDs of every reconciliation result in the bucket.
func (l *ServiceLoader) ListReconciliationResultIDs(ctx context.Context) ([]string, error) {
	return l.listEntityIDs(ctx, l.keys.ReconciliationResultsPrefix())
}

// ListCleanupOperationIDs lists the IDs of every cleanup operation in the bucket.
func (l *ServiceLoader) ListCleanupOperationIDs(ctx context.Context) ([]string, error) {
	return l.listEntityIDs(ctx, l.keys.CleanupOperationsPrefix())
}

// Keys directly under prefix are "<prefix><id>.json"; anything nested deeper is not an
// entity of the category.
func (l *ServiceLoader) listEntityIDs(ctx context.Context, prefix string) ([]string, error) {
	objects, err := l.svc.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(objects))
	for _, object := range objects {
		name := strings.TrimPrefix(object.Key, prefix)
		id, isJSON := strings.CutSuffix(name, ".json")
		if isJSON && id != "" && !strings.Contains(id, "/") {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func (l *ServiceLoader) loadEntityRecord(ctx context.Context, key string) (string, bool, error) {
	var record statestore.EntityRecord
	found, err := readEntity(ctx, l.svc, key, &record)
	if err != nil || !found {
		return "", found, err
	}

	return record.InstanceID, true, nil
}

func (l *ServiceLoader) LoadResourceDrift(
	ctx context.Context,
	id string,
) (*state.ResourceDriftState, bool, error) {
	var drift state.ResourceDriftState
	found, err := readEntity(ctx, l.svc, l.keys.ResourceDrift(id), &drift)
	if err != nil || !found {
		return nil, found, err
	}
	return &drift, true, nil
}

func (l *ServiceLoader) LoadLinkDrift(
	ctx context.Context,
	id string,
) (*state.LinkDriftState, bool, error) {
	var drift state.LinkDriftState
	found, err := readEntity(ctx, l.svc, l.keys.LinkDrift(id), &drift)
	if err != nil || !found {
		return nil, found, err
	}
	return &drift, true, nil
}

// LoadEvent returns (nil, false, nil). Events are partition-keyed rather
// than per-ID; resolving a single event by ID would require reading the
// relevant partition. Event lookup is not a hot path under ModeLazy; if it
// becomes one, add an event-ID index.
func (l *ServiceLoader) LoadEvent(_ context.Context, _ string) (*manage.Event, bool, error) {
	return nil, false, nil
}

func (l *ServiceLoader) LoadChangeset(
	ctx context.Context,
	id string,
) (*manage.Changeset, bool, error) {
	var cs manage.Changeset
	found, err := readEntity(ctx, l.svc, l.keys.Changeset(id), &cs)
	if err != nil || !found {
		return nil, found, err
	}
	return &cs, true, nil
}

func (l *ServiceLoader) LoadValidation(
	ctx context.Context,
	id string,
) (*manage.BlueprintValidation, bool, error) {
	var v manage.BlueprintValidation
	found, err := readEntity(ctx, l.svc, l.keys.Validation(id), &v)
	if err != nil || !found {
		return nil, found, err
	}
	return &v, true, nil
}

// LoadAuditRecord reads one record by ID.
//
// The key leads with the record's timestamp, which a caller holding only an ID does not
// have, so the key is found by listing the trail's prefix rather than constructed. That
// is a listing of keys and a single read, not a read of the trail.
func (l *ServiceLoader) LoadAuditRecord(
	ctx context.Context,
	id string,
) (*manage.AuditRecord, bool, error) {
	objects, err := l.svc.List(ctx, l.keys.AuditRecordsPrefix())
	if err != nil {
		return nil, false, err
	}

	for _, object := range objects {
		_, recordID, ok := statestore.ParseTimestampedKey(object.Key)
		if !ok {
			return nil, false, statestore.NotAnAuditRecordKeyError(object.Key)
		}
		if recordID != id {
			continue
		}

		var r manage.AuditRecord
		found, err := readEntity(ctx, l.svc, object.Key, &r)
		if err != nil || !found {
			return nil, found, err
		}

		return &r, true, nil
	}

	return nil, false, nil
}

func (l *ServiceLoader) LoadReconciliation(
	ctx context.Context,
	id string,
) (*manage.ReconciliationResult, bool, error) {
	var r manage.ReconciliationResult
	found, err := readEntity(ctx, l.svc, l.keys.ReconciliationResult(id), &r)
	if err != nil || !found {
		return nil, found, err
	}
	return &r, true, nil
}

func (l *ServiceLoader) LoadCleanupOperation(
	ctx context.Context,
	id string,
) (*manage.CleanupOperation, bool, error) {
	var op manage.CleanupOperation
	found, err := readEntity(ctx, l.svc, l.keys.CleanupOperation(id), &op)
	if err != nil || !found {
		return nil, found, err
	}
	return &op, true, nil
}

func readEntity(ctx context.Context, svc Service, key string, dst any) (bool, error) {
	data, _, err := svc.Get(ctx, key)
	if err != nil {
		if isObjectNotFound(err) {
			return false, nil
		}

		var sErr *Error
		if errors.As(err, &sErr) && sErr.ReasonCode == ErrorReasonCodeObjectNotFound {
			return false, nil
		}

		return false, err
	}

	if len(data) == 0 {
		return false, nil
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return false, err
	}

	return true, nil
}
