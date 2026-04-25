package gcs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// Service is the GCS-backed objectstore.Service implementation. It maps SDK
// errors to objectstore.Error with a ReasonCode so the shared state layer
// can reason about not-found vs. precondition-failed vs. auth / rate limit
// without depending on GCS types.
//
// The ETag string in the objectstore.Service contract carries the object's
// GCS generation (stringified int64) for this backend. Callers treat the
// value as opaque, passing it back through PutOptions.IfMatch to drive
// conditional writes via storage.Conditions{GenerationMatch: n}.
type Service struct {
	client *storage.Client
	bucket string
}

// NewService constructs a GCS Service bound to the given bucket. The caller
// owns the storage.Client lifecycle (credentials, endpoint etc.).
func NewService(client *storage.Client, bucket string) *Service {
	return &Service{client: client, bucket: bucket}
}

func (s *Service) Get(ctx context.Context, key string) ([]byte, string, error) {
	reader, err := s.client.Bucket(s.bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, "", mapErr(err, "get "+key)
	}
	defer reader.Close()

	data, readErr := io.ReadAll(reader)
	if readErr != nil {
		return nil, "", readErr
	}
	return data, formatGeneration(reader.Attrs.Generation), nil
}

func (s *Service) Put(
	ctx context.Context,
	key string,
	data []byte,
	opts *objectstore.PutOptions,
) (string, error) {
	obj, err := conditionalObject(s.client.Bucket(s.bucket).Object(key), opts)
	if err != nil {
		return "", err
	}

	writer := obj.NewWriter(ctx)
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		_ = writer.Close()
		return "", mapErr(err, "put "+key)
	}
	if err := writer.Close(); err != nil {
		return "", mapErr(err, "put "+key)
	}
	return formatGeneration(writer.Attrs().Generation), nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	if err := s.client.Bucket(s.bucket).Object(key).Delete(ctx); err != nil {
		return mapErr(err, "delete "+key)
	}

	return nil
}

func (s *Service) Head(ctx context.Context, key string) (*objectstore.ObjectInfo, error) {
	attrs, err := s.client.Bucket(s.bucket).Object(key).Attrs(ctx)
	if err != nil {
		return nil, mapErr(err, "head "+key)
	}

	return &objectstore.ObjectInfo{
		Key:  key,
		Size: attrs.Size,
		ETag: formatGeneration(attrs.Generation),
	}, nil
}

func (s *Service) List(ctx context.Context, prefix string) ([]*objectstore.ObjectInfo, error) {
	var infos []*objectstore.ObjectInfo
	it := s.client.Bucket(s.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr(err, "list "+prefix)
		}
		infos = append(infos, &objectstore.ObjectInfo{
			Key:  attrs.Name,
			Size: attrs.Size,
			ETag: formatGeneration(attrs.Generation),
		})
	}

	return infos, nil
}

// conditionalObject applies IfMatch / IfNoneMatch semantics from the
// objectstore.PutOptions onto the ObjectHandle. GCS uses numeric generation
// CAS rather than string ETag CAS, so IfMatch values are parsed as int64
// generations; the "*" IfNoneMatch sentinel maps to DoesNotExist: true.
func conditionalObject(
	obj *storage.ObjectHandle,
	opts *objectstore.PutOptions,
) (*storage.ObjectHandle, error) {
	if opts == nil {
		return obj, nil
	}
	conds := storage.Conditions{}
	hasCondition := false
	if opts.IfNoneMatch == "*" {
		conds.DoesNotExist = true
		hasCondition = true
	}
	if opts.IfMatch != "" {
		gen, err := strconv.ParseInt(opts.IfMatch, 10, 64)
		if err != nil {
			return nil, err
		}
		conds.GenerationMatch = gen
		hasCondition = true
	}
	if !hasCondition {
		return obj, nil
	}
	return obj.If(conds), nil
}

// Classifies an SDK error onto the ReasonCode taxonomy so the shared
// state layer can match on a stable surface. Not-found comes from the
// typed storage.ErrObjectNotExist sentinel; precondition / auth / rate
// limit come from the HTTP status on a wrapped *googleapi.Error.
func mapErr(err error, ctx string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, storage.ErrObjectNotExist) || errors.Is(err, storage.ErrBucketNotExist) {
		return objectstore.NewObjectNotFound(ctx)
	}

	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		switch {
		case gerr.Code == http.StatusNotFound:
			return objectstore.NewObjectNotFound(ctx)
		case gerr.Code == http.StatusPreconditionFailed:
			return objectstore.NewPreconditionFailed(ctx)
		case isAuthFailedCode(gerr.Code):
			return objectstore.NewAuthFailed(ctx)
		case isRateLimitedCode(gerr.Code):
			return objectstore.NewRateLimited(ctx)
		}
	}
	return err
}

func isAuthFailedCode(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

func isRateLimitedCode(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable
}

func formatGeneration(generation int64) string {
	return strconv.FormatInt(generation, 10)
}
