package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
)

// Service is the S3-backed objectstore.Service implementation. It maps SDK
// errors to objectstore.Error with a ReasonCode so the shared state layer
// can reason about not-found vs. precondition-failed vs. auth / rate limit
// without depending on AWS types.
type Service struct {
	client *s3sdk.Client
	bucket string
}

// NewService constructs an S3 Service bound to the given bucket. The caller
// owns the s3.Client lifecycle (credentials, region, endpoint, path-style
// addressing etc.)
func NewService(client *s3sdk.Client, bucket string) *Service {
	return &Service{client: client, bucket: bucket}
}

func (s *Service) Get(ctx context.Context, key string) ([]byte, string, error) {
	out, err := s.client.GetObject(ctx, &s3sdk.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", mapErr(err, "get "+key)
	}
	defer out.Body.Close()

	data, readErr := io.ReadAll(out.Body)
	if readErr != nil {
		return nil, "", readErr
	}

	return data, derefString(out.ETag), nil
}

func (s *Service) Put(
	ctx context.Context,
	key string,
	data []byte,
	opts *objectstore.PutOptions,
) (string, error) {
	input := &s3sdk.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}

	if opts != nil {
		if opts.IfMatch != "" {
			input.IfMatch = aws.String(opts.IfMatch)
		}
		if opts.IfNoneMatch != "" {
			input.IfNoneMatch = aws.String(opts.IfNoneMatch)
		}
	}

	out, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", mapErr(err, "put "+key)
	}

	return derefString(out.ETag), nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return mapErr(err, "delete "+key)
	}

	return nil
}

func (s *Service) Head(ctx context.Context, key string) (*objectstore.ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3sdk.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapErr(err, "head "+key)
	}

	return &objectstore.ObjectInfo{
		Key:  key,
		Size: derefInt64(out.ContentLength),
		ETag: derefString(out.ETag),
	}, nil
}

func (s *Service) List(ctx context.Context, prefix string) ([]*objectstore.ObjectInfo, error) {
	var infos []*objectstore.ObjectInfo
	paginator := s3sdk.NewListObjectsV2Paginator(s.client, &s3sdk.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, mapErr(err, "list "+prefix)
		}
		for _, obj := range page.Contents {
			infos = append(infos, &objectstore.ObjectInfo{
				Key:  derefString(obj.Key),
				Size: derefInt64(obj.Size),
				ETag: derefString(obj.ETag),
			})
		}
	}

	return infos, nil
}

// Classifies an SDK error onto the ReasonCode taxonomy so the shared
// state layer can match on a stable surface. Precondition-failed catches
// both "PreconditionFailed" (IfMatch stale) and IfNoneMatch: "*" collision
// (which S3 historically returned as PreconditionFailed; newer responses
// use ConditionalRequestConflict).
func mapErr(err error, ctx string) error {
	if err == nil {
		return nil
	}

	var notFoundKey *types.NoSuchKey
	var notFound *types.NotFound
	if errors.As(err, &notFoundKey) || errors.As(err, &notFound) {
		return objectstore.NewObjectNotFound(ctx)
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		switch {
		case isNotFoundCode(code):
			return objectstore.NewObjectNotFound(ctx)
		case isPreconditionFailedCode(code):
			return objectstore.NewPreconditionFailed(ctx)
		case isAuthFailedCode(code):
			return objectstore.NewAuthFailed(ctx)
		case isRateLimitedCode(code):
			return objectstore.NewRateLimited(ctx)
		}
	}

	return err
}

func isNotFoundCode(code string) bool {
	return code == "NoSuchKey" || code == "NotFound" || code == "NoSuchBucket"
}

func isPreconditionFailedCode(code string) bool {
	return code == "PreconditionFailed" || code == "ConditionalRequestConflict"
}

func isAuthFailedCode(code string) bool {
	return code == "AccessDenied" ||
		code == "InvalidAccessKeyId" ||
		code == "SignatureDoesNotMatch" ||
		code == "ExpiredToken"
}

func isRateLimitedCode(code string) bool {
	return code == "SlowDown" ||
		code == "TooManyRequests" ||
		strings.EqualFold(code, "Throttling") ||
		strings.EqualFold(code, "ThrottlingException")
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
