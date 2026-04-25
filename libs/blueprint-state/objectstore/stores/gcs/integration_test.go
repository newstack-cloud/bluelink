package gcs_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	gcsstore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/gcs"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
)

func integrationEndpoint() string {
	return os.Getenv("OBJECTSTORE_GCS_ENDPOINT")
}

func integrationProjectID() string {
	return os.Getenv("OBJECTSTORE_GCS_PROJECT_ID")
}

func newTestBucket(t *testing.T, ctx context.Context) (*gcsstore.Service, *storage.Client, string) {
	t.Helper()

	client, err := gcsstore.NewClient(ctx, gcsstore.ClientOptions{
		Endpoint:              integrationEndpoint(),
		WithoutAuthentication: true,
	})
	require.NoError(t, err)

	bucket := "bluelink-state-it-" + uuid.NewString()
	err = client.Bucket(bucket).Create(ctx, integrationProjectID(), nil)
	require.NoError(t, err, "create bucket")

	t.Cleanup(func() {
		emptyAndDelete(t, client, bucket)
	})

	return gcsstore.NewService(client, bucket), client, bucket
}

func emptyAndDelete(t *testing.T, client *storage.Client, bucket string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Logf("list bucket %s for teardown: %v", bucket, err)
			return
		}
		_ = client.Bucket(bucket).Object(attrs.Name).Delete(ctx)
	}
	if err := client.Bucket(bucket).Delete(ctx); err != nil {
		t.Logf("delete bucket %s: %v", bucket, err)
	}
}

func TestGCSService_put_get_round_trips(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	etag, err := svc.Put(ctx, "hello.json", []byte(`{"msg":"hi"}`), nil)
	require.NoError(t, err)
	require.NotEmpty(t, etag)

	data, gotETag, err := svc.Get(ctx, "hello.json")
	require.NoError(t, err)
	require.Equal(t, []byte(`{"msg":"hi"}`), data)
	require.Equal(t, etag, gotETag)
}

func TestGCSService_get_returns_object_not_found_for_missing_key(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	_, _, err := svc.Get(ctx, "missing.json")
	requireReason(t, err, objectstore.ErrorReasonCodeObjectNotFound)
}

func TestGCSService_put_ifNoneMatch_rejects_duplicate_create(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	_, err := svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	require.NoError(t, err)

	_, err = svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	requireReason(t, err, objectstore.ErrorReasonCodePreconditionFailed)
}

func TestGCSService_put_ifMatch_succeeds_with_current_token_and_fails_with_stale(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	firstToken, err := svc.Put(ctx, "cas.json", []byte(`{"v":1}`), nil)
	require.NoError(t, err)

	secondToken, err := svc.Put(
		ctx, "cas.json", []byte(`{"v":2}`),
		&objectstore.PutOptions{IfMatch: firstToken},
	)
	require.NoError(t, err)
	require.NotEqual(t, firstToken, secondToken)

	// Using the stale first token must now fail with precondition-failed.
	_, err = svc.Put(
		ctx, "cas.json", []byte(`{"v":3}`),
		&objectstore.PutOptions{IfMatch: firstToken},
	)
	requireReason(t, err, objectstore.ErrorReasonCodePreconditionFailed)
}

func TestGCSService_head_list_delete_round_trip(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

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

// Exercises the full objectstore.StateContainer wiring against fake-gcs-server,
// mirroring the mockservice-backed smoke tests. Proves the generation CAS
// guard on InitialiseAndClaim serialises concurrent callers.
func TestStateContainer_initialise_and_claim_against_gcs_concurrent_callers_only_one_wins(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

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
