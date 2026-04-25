package objectstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion that *StateContainer satisfies the blueprint
// state.Container interface — catches signature drift in per-category
// sub-containers without needing a runtime test.
var _ state.Container = (*objectstore.StateContainer)(nil)

func TestLoadStateContainer_on_empty_bucket_initialises_cleanly(t *testing.T) {
	svc := mockservice.New()

	container, err := objectstore.LoadStateContainer(
		context.Background(),
		svc,
		"bluelink-state/",
		core.NewNopLogger(),
	)
	require.NoError(t, err)
	require.NotNil(t, container)

	_, err = container.Instances().Get(context.Background(), "missing-instance")
	require.Error(t, err)
	stateErr, ok := err.(*state.Error)
	require.True(t, ok, "expected *state.Error, got %T", err)
	assert.Equal(t, state.ErrInstanceNotFound, stateErr.Code)
}

func TestLoadStateContainer_initialise_and_claim_roundtrips_through_service(t *testing.T) {
	svc := mockservice.New()
	prefix := "bluelink-state/"
	const instanceID = "round-trip-instance"
	const instanceName = "RoundTripInstance"

	container, err := objectstore.LoadStateContainer(
		context.Background(),
		svc,
		prefix,
		core.NewNopLogger(),
	)
	require.NoError(t, err)

	version, err := container.Instances().InitialiseAndClaim(
		context.Background(),
		state.InstanceState{
			InstanceID:   instanceID,
			InstanceName: instanceName,
		},
		core.InstanceStatusPreparing,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)

	// A fresh container re-loading from the same bucket must find the
	// persisted record — proves the round-trip through Service/Storage.
	reloaded, err := objectstore.LoadStateContainer(
		context.Background(),
		svc,
		prefix,
		core.NewNopLogger(),
	)
	require.NoError(t, err)

	got, err := reloaded.Instances().Get(context.Background(), instanceID)
	require.NoError(t, err)
	assert.Equal(t, instanceName, got.InstanceName)
	assert.Equal(t, core.InstanceStatusPreparing, got.Status)
	assert.Equal(t, int64(1), got.Version)
}

func TestLoadStateContainer_lazy_loads_instance_by_name_via_name_record(t *testing.T) {
	svc := mockservice.New()
	prefix := "bluelink-state/"
	const instanceID = "lazy-by-name-instance"
	const instanceName = "LazyByNameInstance"

	container, err := objectstore.LoadStateContainer(
		context.Background(), svc, prefix, core.NewNopLogger(),
	)
	require.NoError(t, err)

	_, err = container.Instances().InitialiseAndClaim(
		context.Background(),
		state.InstanceState{InstanceID: instanceID, InstanceName: instanceName},
		core.InstanceStatusPreparing,
	)
	require.NoError(t, err)

	// A fresh container starts with an empty in-memory State. Resolving by
	// name must therefore go through the ServiceLoader — proves both the
	// name record was written by InitialiseAndClaim and the lazy loader
	// can resolve it.
	reloaded, err := objectstore.LoadStateContainer(
		context.Background(), svc, prefix, core.NewNopLogger(),
	)
	require.NoError(t, err)

	gotID, err := reloaded.Instances().LookupIDByName(context.Background(), instanceName)
	require.NoError(t, err)
	assert.Equal(t, instanceID, gotID)
}

func TestLoadStateContainer_claim_for_deployment_via_container(t *testing.T) {
	svc := mockservice.New()
	const instanceID = "claim-via-container"

	container, err := objectstore.LoadStateContainer(
		context.Background(),
		svc,
		"bluelink-state/",
		core.NewNopLogger(),
	)
	require.NoError(t, err)

	// Bootstrap at V1 via InitialiseAndClaim.
	_, err = container.Instances().InitialiseAndClaim(
		context.Background(),
		state.InstanceState{InstanceID: instanceID, InstanceName: instanceID},
		core.InstanceStatusPreparing,
	)
	require.NoError(t, err)

	// Follow-up ClaimForDeployment at expected=1 should succeed and bump to V2.
	newVersion, err := container.Instances().ClaimForDeployment(
		context.Background(),
		instanceID,
		1,
		core.InstanceStatusDeploying,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), newVersion)

	// Stale claim at expected=1 loses.
	_, err = container.Instances().ClaimForDeployment(
		context.Background(),
		instanceID,
		1,
		core.InstanceStatusDeploying,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrVersionConflict))
}
