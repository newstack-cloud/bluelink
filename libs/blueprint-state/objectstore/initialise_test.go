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

func TestInitialiseAndClaim_creates_new_instance_at_version_1(t *testing.T) {
	svc := mockservice.New()
	keys := statestore.NewKeyBuilder("")
	st := statestore.NewState()

	version, err := objectstore.InitialiseAndClaim(
		context.Background(), svc, keys, st,
		state.InstanceState{
			InstanceID:   testInstanceID,
			InstanceName: "TestInstance",
		},
		core.InstanceStatusPreparing,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)

	// In-memory state synced.
	inst, ok := st.Instance(testInstanceID)
	require.True(t, ok)
	assert.Equal(t, core.InstanceStatusPreparing, inst.Status)
	assert.Equal(t, int64(1), inst.Version)

	// Persisted object reflects the new record.
	data, _, err := svc.Get(context.Background(), keys.Instance(testInstanceID))
	require.NoError(t, err)
	var persisted statestore.PersistedInstanceState
	require.NoError(t, json.Unmarshal(data, &persisted))
	assert.Equal(t, int64(1), persisted.Version)
	assert.Equal(t, core.InstanceStatusPreparing, persisted.Status)
}

func TestInitialiseAndClaim_returns_already_exists_for_existing_instance(t *testing.T) {
	svc, keys, st := setupClaimTest(t, 3)

	_, err := objectstore.InitialiseAndClaim(
		context.Background(), svc, keys, st,
		state.InstanceState{
			InstanceID:   testInstanceID,
			InstanceName: "ignored-because-conflict",
		},
		core.InstanceStatusPreparing,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrInstanceAlreadyExists))

	// Existing record untouched.
	data, _, err := svc.Get(context.Background(), keys.Instance(testInstanceID))
	require.NoError(t, err)
	var persisted statestore.PersistedInstanceState
	require.NoError(t, json.Unmarshal(data, &persisted))
	assert.Equal(t, int64(3), persisted.Version)
	assert.NotEqual(t, "ignored-because-conflict", persisted.InstanceName)
}

func TestInitialiseAndClaim_concurrent_callers_only_one_wins(t *testing.T) {
	svc := mockservice.New()
	keys := statestore.NewKeyBuilder("")
	st := statestore.NewState()

	const workers = 10
	var wg sync.WaitGroup
	var successes, conflicts int64

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_, err := objectstore.InitialiseAndClaim(
				context.Background(), svc, keys, st,
				state.InstanceState{
					InstanceID:   testInstanceID,
					InstanceName: "TestInstance",
				},
				core.InstanceStatusPreparing,
			)
			if err == nil {
				atomic.AddInt64(&successes, 1)
				return
			}
			if errors.Is(err, state.ErrInstanceAlreadyExists) {
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
