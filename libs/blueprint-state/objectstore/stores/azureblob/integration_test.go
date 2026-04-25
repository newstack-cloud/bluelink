package azureblob_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	azstore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/azureblob"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/require"
)

func integrationServiceURL() string {
	return os.Getenv("OBJECTSTORE_AZURE_SERVICE_URL")
}

func integrationAccountName() string {
	return os.Getenv("OBJECTSTORE_AZURE_ACCOUNT_NAME")
}

func integrationAccountKey() string {
	return os.Getenv("OBJECTSTORE_AZURE_ACCOUNT_KEY")
}

func newTestContainer(t *testing.T, ctx context.Context) (*azstore.Service, *azblob.Client, string) {
	t.Helper()

	client, err := azstore.NewClient(azstore.ClientOptions{
		ServiceURL:  integrationServiceURL(),
		AccountName: integrationAccountName(),
		AccountKey:  integrationAccountKey(),
	})
	require.NoError(t, err)

	// Azure container names must be lowercase, 3-63 chars, hyphens only.
	name := "bluelink-state-it-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(name) > 63 {
		name = name[:63]
	}
	_, err = client.CreateContainer(ctx, name, nil)
	require.NoError(t, err, "create container")

	t.Cleanup(func() {
		dropContainer(t, client, name)
	})

	return azstore.NewService(client, name), client, name
}

func dropContainer(t *testing.T, client *azblob.Client, container string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.DeleteContainer(ctx, container, nil); err != nil {
		t.Logf("delete container %s: %v", container, err)
	}
}

func TestAzureService_put_get_round_trips(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	etag, err := svc.Put(ctx, "hello.json", []byte(`{"msg":"hi"}`), nil)
	require.NoError(t, err)
	require.NotEmpty(t, etag)

	data, gotETag, err := svc.Get(ctx, "hello.json")
	require.NoError(t, err)
	require.Equal(t, []byte(`{"msg":"hi"}`), data)
	require.Equal(t, etag, gotETag)
}

func TestAzureService_get_returns_object_not_found_for_missing_key(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	_, _, err := svc.Get(ctx, "missing.json")
	requireReason(t, err, objectstore.ErrorReasonCodeObjectNotFound)
}

func TestAzureService_put_ifNoneMatch_rejects_duplicate_create(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	_, err := svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	require.NoError(t, err)

	_, err = svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	requireReason(t, err, objectstore.ErrorReasonCodePreconditionFailed)
}

func TestAzureService_put_ifMatch_succeeds_with_current_etag_and_fails_with_stale(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	firstETag, err := svc.Put(ctx, "cas.json", []byte(`{"v":1}`), nil)
	require.NoError(t, err)

	secondETag, err := svc.Put(
		ctx, "cas.json", []byte(`{"v":2}`),
		&objectstore.PutOptions{IfMatch: firstETag},
	)
	require.NoError(t, err)
	require.NotEqual(t, firstETag, secondETag)

	// Using the stale first ETag must now fail with precondition-failed.
	_, err = svc.Put(
		ctx, "cas.json", []byte(`{"v":3}`),
		&objectstore.PutOptions{IfMatch: firstETag},
	)
	requireReason(t, err, objectstore.ErrorReasonCodePreconditionFailed)
}

func TestAzureService_head_list_delete_round_trip(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	_, err := svc.Put(ctx, "dir/a.json", []byte(`a`), nil)
	require.NoError(t, err)
	_, err = svc.Put(ctx, "dir/b.json", []byte(`bb`), nil)
	require.NoError(t, err)

	info, err := svc.Head(ctx, "dir/a.json")
	require.NoError(t, err)
	require.Equal(t, int64(1), info.Size)
	require.NotEmpty(t, info.ETag)

	listed, err := svc.List(ctx, "dir/")
	require.NoError(t, err)
	require.Len(t, listed, 2)

	require.NoError(t, svc.Delete(ctx, "dir/a.json"))

	_, err = svc.Head(ctx, "dir/a.json")
	requireReason(t, err, objectstore.ErrorReasonCodeObjectNotFound)
}

// Exercises the full objectstore.StateContainer wiring against Azurite,
// mirroring the mockservice-backed smoke tests. Proves the ETag CAS
// guard on InitialiseAndClaim serialises concurrent callers.
func TestStateContainer_initialise_and_claim_against_azure_concurrent_callers_only_one_wins(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestContainer(t, ctx)

	container, err := objectstore.LoadStateContainer(
		ctx, svc, "bluelink-state/",
		core.NewNopLogger(),
	)
	require.NoError(t, err)

	const raceID = "race-instance"
	const workers = 10
	var wg sync.WaitGroup
	var successes, conflicts int64

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_, err := container.Instances().InitialiseAndClaim(
				ctx,
				state.InstanceState{InstanceID: raceID, InstanceName: raceID},
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

	require.Equal(t, int64(1), atomic.LoadInt64(&successes))
	require.Equal(t, int64(workers-1), atomic.LoadInt64(&conflicts))

	reloaded, err := objectstore.LoadStateContainer(
		ctx, svc, "bluelink-state/",
		core.NewNopLogger(),
	)
	require.NoError(t, err)

	got, err := reloaded.Instances().Get(ctx, raceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Version)
	require.Equal(t, core.InstanceStatusPreparing, got.Status)
}

func requireReason(t *testing.T, err error, expected objectstore.ErrorReasonCode) {
	t.Helper()
	var sErr *objectstore.Error
	if !errors.As(err, &sErr) {
		t.Fatalf("expected *objectstore.Error, got %T: %v", err, err)
	}
	require.Equal(t, expected, sErr.ReasonCode)
}
