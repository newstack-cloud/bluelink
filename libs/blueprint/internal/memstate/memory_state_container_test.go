package memstate

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testInstanceID        = "test-instance-1"
	testInstanceName      = "TestInstance1"
	nonExistentInstanceID = "non-existent-instance"
)

func newTestContainerWithInstance(t *testing.T) state.InstancesContainer {
	t.Helper()
	container := NewMemoryStateContainer()
	err := container.Instances().Save(context.Background(), state.InstanceState{
		InstanceID:   testInstanceID,
		InstanceName: testInstanceName,
	})
	require.NoError(t, err)
	return container.Instances()
}

func TestClaimForDeployment_succeeds_with_matching_version(t *testing.T) {
	instances := newTestContainerWithInstance(t)

	newVersion, err := instances.ClaimForDeployment(
		context.Background(),
		testInstanceID,
		0,
		core.InstanceStatusDeploying,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), newVersion)

	savedState, err := instances.Get(context.Background(), testInstanceID)
	require.NoError(t, err)
	assert.Equal(t, core.InstanceStatusDeploying, savedState.Status)
	assert.Equal(t, int64(1), savedState.Version)
}

func TestClaimForDeployment_returns_conflict_on_stale_version(t *testing.T) {
	instances := newTestContainerWithInstance(t)

	_, err := instances.ClaimForDeployment(
		context.Background(),
		testInstanceID,
		0,
		core.InstanceStatusDeploying,
	)
	require.NoError(t, err)

	currentVersion, err := instances.ClaimForDeployment(
		context.Background(),
		testInstanceID,
		0,
		core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrVersionConflict))
	assert.Equal(t, int64(1), currentVersion)
}

func TestClaimForDeployment_reports_instance_not_found(t *testing.T) {
	container := NewMemoryStateContainer()
	instances := container.Instances()

	_, err := instances.ClaimForDeployment(
		context.Background(),
		nonExistentInstanceID,
		0,
		core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	stateErr, ok := err.(*state.Error)
	require.True(t, ok)
	assert.Equal(t, state.ErrInstanceNotFound, stateErr.Code)
}

func TestClaimForDeployment_concurrent_goroutines_only_one_wins(t *testing.T) {
	instances := newTestContainerWithInstance(t)

	const workers = 10
	var wg sync.WaitGroup
	var successes, conflicts int64

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_, err := instances.ClaimForDeployment(
				context.Background(),
				testInstanceID,
				0,
				core.InstanceStatusDeploying,
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

	savedState, err := instances.Get(context.Background(), testInstanceID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), savedState.Version)
}

func TestInitialiseAndClaim_creates_new_instance_at_version_1(t *testing.T) {
	container := NewMemoryStateContainer()
	instances := container.Instances()

	version, err := instances.InitialiseAndClaim(
		context.Background(),
		state.InstanceState{
			InstanceID:   testInstanceID,
			InstanceName: testInstanceName,
		},
		core.InstanceStatusPreparing,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)

	savedState, err := instances.Get(context.Background(), testInstanceID)
	require.NoError(t, err)
	assert.Equal(t, core.InstanceStatusPreparing, savedState.Status)
	assert.Equal(t, int64(1), savedState.Version)
	assert.Equal(t, testInstanceName, savedState.InstanceName)
}

func TestInitialiseAndClaim_returns_already_exists_for_existing_instance(t *testing.T) {
	instances := newTestContainerWithInstance(t)

	_, err := instances.InitialiseAndClaim(
		context.Background(),
		state.InstanceState{
			InstanceID:   testInstanceID,
			InstanceName: "ignored-because-conflict",
		},
		core.InstanceStatusPreparing,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrInstanceAlreadyExists))

	savedState, err := instances.Get(context.Background(), testInstanceID)
	require.NoError(t, err)
	assert.NotEqual(t, "ignored-because-conflict", savedState.InstanceName)
	assert.Equal(t, int64(0), savedState.Version)
}

func TestInitialiseAndClaim_concurrent_goroutines_only_one_wins(t *testing.T) {
	container := NewMemoryStateContainer()
	instances := container.Instances()
	const raceID = "race-instance"

	const workers = 10
	var wg sync.WaitGroup
	var successes, conflicts int64

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_, err := instances.InitialiseAndClaim(
				context.Background(),
				state.InstanceState{
					InstanceID:   raceID,
					InstanceName: raceID,
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

	savedState, err := instances.Get(context.Background(), raceID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), savedState.Version)
}
