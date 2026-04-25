package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

// ClientOptions configures optional client-level overrides used when
// constructing an S3 client for the objectstore Service. The caller still
// owns credential resolution through aws.Config (default credential chain,
// static credentials, assume-role, etc.).
type ClientOptions struct {
	// Endpoint overrides the default S3 endpoint — useful for LocalStack,
	// MinIO, or any S3-compatible gateway. Empty means use the default.
	Endpoint string
	// UsePathStyle switches the addressing style from virtual-hosted
	// (default) to path-style. Required for LocalStack and many
	// S3-compatible gateways that don't implement virtual-hosted buckets.
	UsePathStyle bool
}

// NewClient builds an s3.Client from an aws.Config and the given overrides.
// Mirrors the shape used by libs/blueprint-resolvers/s3 so downstream
// callers can share credential-loading conventions.
func NewClient(conf aws.Config, opts ClientOptions) *s3sdk.Client {
	return s3sdk.NewFromConfig(conf, func(o *s3sdk.Options) {
		o.UsePathStyle = opts.UsePathStyle
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
		}
	})
}
