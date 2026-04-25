package s3_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	s3store "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/s3"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/require"
)

func integrationEndpoint() string {
	return os.Getenv("OBJECTSTORE_S3_ENDPOINT")
}
func integrationRegion() string {
	return os.Getenv("OBJECTSTORE_S3_REGION")
}

func integrationAccessKey() string {
	return os.Getenv("OBJECTSTORE_S3_ACCESS_KEY_ID")
}

func integrationSecretKey() string {
	return os.Getenv("OBJECTSTORE_S3_SECRET_ACCESS_KEY")
}

func newTestBucket(t *testing.T, ctx context.Context) (*s3store.Service, *s3sdk.Client, string) {
	t.Helper()

	conf, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(integrationRegion()),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(integrationAccessKey(), integrationSecretKey(), ""),
		),
	)
	require.NoError(t, err)

	client := s3store.NewClient(conf, s3store.ClientOptions{
		Endpoint:     integrationEndpoint(),
		UsePathStyle: true,
	})

	bucket := "bluelink-state-it-" + uuid.NewString()
	_, err = client.CreateBucket(ctx, &s3sdk.CreateBucketInput{
		Bucket: awssdk.String(bucket),
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(integrationRegion()),
		},
	})
	require.NoError(t, err, "create bucket")

	t.Cleanup(func() {
		emptyAndDelete(t, client, bucket)
	})

	return s3store.NewService(client, bucket), client, bucket
}

func emptyAndDelete(t *testing.T, client *s3sdk.Client, bucket string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paginator := s3sdk.NewListObjectsV2Paginator(client, &s3sdk.ListObjectsV2Input{
		Bucket: awssdk.String(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			t.Logf("list bucket %s for teardown: %v", bucket, err)
			return
		}
		for _, obj := range page.Contents {
			_, _ = client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
				Bucket: awssdk.String(bucket),
				Key:    obj.Key,
			})
		}
	}
	_, err := client.DeleteBucket(ctx, &s3sdk.DeleteBucketInput{Bucket: awssdk.String(bucket)})
	if err != nil {
		t.Logf("delete bucket %s: %v", bucket, err)
	}
}

func TestS3Service_put_get_round_trips(t *testing.T) {
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

func TestS3Service_get_returns_object_not_found_for_missing_key(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	_, _, err := svc.Get(ctx, "missing.json")
	requireReason(t, err, objectstore.ErrorReasonCodeObjectNotFound)
}

func TestS3Service_put_ifNoneMatch_rejects_duplicate_create(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

	_, err := svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	require.NoError(t, err)

	_, err = svc.Put(ctx, "once.json", []byte(`{}`), &objectstore.PutOptions{IfNoneMatch: "*"})
	requireReason(t, err, objectstore.ErrorReasonCodePreconditionFailed)
}

func TestS3Service_put_ifMatch_succeeds_with_current_etag_and_fails_with_stale(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestBucket(t, ctx)

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

func TestS3Service_head_list_delete_round_trip(t *testing.T) {
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

// Exercises the full objectstore.StateContainer wiring against real S3,
// mirroring the mockservice-backed smoke tests. Proves the ETag CAS guard
// on InitialiseAndClaim serialises concurrent callers.
func TestStateContainer_initialise_and_claim_against_s3_concurrent_callers_only_one_wins(t *testing.T) {
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
