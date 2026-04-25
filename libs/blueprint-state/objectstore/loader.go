package objectstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ServiceLoader resolves one entity per call from the objectstore Service,
// materialising it into statestore.State only on cache miss. This is the
// seam that lets objectstore run under statestore.ModeLazy — processes
// only touch the working subset of state rather than bulk-loading the
// entire bucket on startup.
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
) (*state.InstanceState, bool, error) {
	var persisted statestore.PersistedInstanceState
	found, err := readEntity(ctx, l.svc, l.keys.Instance(id), &persisted)
	if err != nil || !found {
		return nil, found, err
	}
	return persisted.ToInstanceState(), true, nil
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

// LoadResource and LoadLink intentionally return (nil, false, nil).
// Resources and links are persisted inline with the parent instance record,
// so their in-memory caches are warmed transitively via LoadInstance. A
// direct resource-or-link-by-ID lookup without a prior instance load won't
// resolve — callers must load the parent instance first.
func (l *ServiceLoader) LoadResource(
	_ context.Context,
	_ string,
) (*state.ResourceState, bool, error) {
	return nil, false, nil
}

func (l *ServiceLoader) LoadLink(_ context.Context, _ string) (*state.LinkState, bool, error) {
	return nil, false, nil
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
