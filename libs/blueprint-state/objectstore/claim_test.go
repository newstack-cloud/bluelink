package objectstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testInstanceID        = "test-instance-1"
	nonExistentInstanceID = "missing-instance"
)

func setupClaimTest(t *testing.T, version int64) (*mockservice.Service, statestore.KeyBuilder, *statestore.State) {
	t.Helper()
	svc := mockservice.New()
	keys := statestore.NewKeyBuilder("")
	st := statestore.NewState()

	persisted := &statestore.PersistedInstanceState{
		InstanceID:   testInstanceID,
		InstanceName: "TestInstance",
		Status:       core.InstanceStatusDeployed,
		Version:      version,
	}
	data, err := json.Marshal(persisted)
	require.NoError(t, err)
	_, err = svc.Put(context.Background(), keys.Instance(testInstanceID), data, nil)
	require.NoError(t, err)

	return svc, keys, st
}

func TestClaimForDeployment_succeeds_with_matching_version(t *testing.T) {
	svc, keys, st := setupClaimTest(t, 0)

	newVersion, err := objectstore.ClaimForDeployment(
		context.Background(), svc, keys, st,
		testInstanceID, 0, core.InstanceStatusDeploying,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), newVersion)

	// In-memory state synced.
	inst, ok := st.Instance(testInstanceID)
	require.True(t, ok)
	assert.Equal(t, core.InstanceStatusDeploying, inst.Status)
	assert.Equal(t, int64(1), inst.Version)

	// Persisted object reflects the new version.
	data, _, err := svc.Get(context.Background(), keys.Instance(testInstanceID))
	require.NoError(t, err)
	var persisted statestore.PersistedInstanceState
	require.NoError(t, json.Unmarshal(data, &persisted))
	assert.Equal(t, int64(1), persisted.Version)
	assert.Equal(t, core.InstanceStatusDeploying, persisted.Status)
}

func TestClaimForDeployment_returns_conflict_on_stale_version(t *testing.T) {
	svc, keys, st := setupClaimTest(t, 5)

	currentVersion, err := objectstore.ClaimForDeployment(
		context.Background(), svc, keys, st,
		testInstanceID, 3, core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrVersionConflict))
	assert.Equal(t, int64(5), currentVersion)
}

func TestClaimForDeployment_reports_instance_not_found(t *testing.T) {
	svc := mockservice.New()
	keys := statestore.NewKeyBuilder("")
	st := statestore.NewState()

	_, err := objectstore.ClaimForDeployment(
		context.Background(), svc, keys, st,
		nonExistentInstanceID, 0, core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	stateErr, ok := err.(*state.Error)
	require.True(t, ok)
	assert.Equal(t, state.ErrInstanceNotFound, stateErr.Code)
}

func TestClaimForDeployment_412_returns_conflict_with_fresh_version(t *testing.T) {
	svc, keys, st := setupClaimTest(t, 0)

	// Simulate a concurrent writer overwriting the instance after our Get
	// but before our Put — the mockservice advances the ETag so IfMatch
	// will fail.
	svc.SetHooks(mockservice.Hooks{
		BeforePut: func(ctx context.Context, key string, data []byte, opts *objectstore.PutOptions) error {
			if opts != nil && opts.IfMatch != "" {
				// Before letting our CAS proceed, advance the stored object once.
				bumped := &statestore.PersistedInstanceState{
					InstanceID: testInstanceID, Version: 42,
					Status: core.InstanceStatusDeployed,
				}
				bumpedData, _ := json.Marshal(bumped)
				// Clear hook to avoid infinite recursion; write unconditionally.
				svc.SetHooks(mockservice.Hooks{})
				_, err := svc.Put(ctx, key, bumpedData, nil)
				return err
			}
			return nil
		},
	})

	currentVersion, err := objectstore.ClaimForDeployment(
		context.Background(), svc, keys, st,
		testInstanceID, 0, core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrVersionConflict))
	assert.Equal(t, int64(42), currentVersion, "should report the version established by the concurrent writer")
}

func TestClaimForDeployment_concurrent_callers_only_one_wins(t *testing.T) {
	svc, keys, st := setupClaimTest(t, 0)

	const workers = 10
	var wg sync.WaitGroup
	var successes, conflicts int64

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_, err := objectstore.ClaimForDeployment(
				context.Background(), svc, keys, st,
				testInstanceID, 0, core.InstanceStatusDeploying,
			)
			if err == nil {
				atomic.AddInt64(&successes, 1)
				return
			}
			if errors.Is(err, state.ErrVersionConflict) {
				atomic.AddInt64(&conflicts, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&successes))
	assert.Equal(t, int64(workers-1), atomic.LoadInt64(&conflicts))

	data, _, err := svc.Get(context.Background(), keys.Instance(testInstanceID))
	require.NoError(t, err)
	var persisted statestore.PersistedInstanceState
	require.NoError(t, json.Unmarshal(data, &persisted))
	assert.Equal(t, int64(1), persisted.Version)
}
