package gcs

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// ClientOptions configures optional client-level overrides used when
// constructing a GCS client for the objectstore Service. The caller still
// owns credential resolution through Application Default Credentials or
// explicit credentials options.
type ClientOptions struct {
	// Endpoint overrides the default GCS endpoint — useful for
	// fake-gcs-server and any GCS-compatible gateway. Empty means use the
	// default GCS endpoint.
	Endpoint string
	// WithoutAuthentication disables the default authentication flow. Set
	// this when targeting fake-gcs-server or another emulator that does
	// not expect credentials.
	WithoutAuthentication bool
	// Extra lets callers append provider-specific option.ClientOption
	// values (e.g. option.WithCredentialsFile, option.WithHTTPClient) for
	// cases the struct fields do not cover.
	Extra []option.ClientOption
}

// NewClient builds a *storage.Client from the given options. Mirrors the
// shape used by libs/blueprint-resolvers/gcs so downstream callers can share
// credential-loading conventions.
func NewClient(ctx context.Context, opts ClientOptions) (*storage.Client, error) {
	clientOpts := make([]option.ClientOption, 0, len(opts.Extra)+2)
	if opts.Endpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(opts.Endpoint))
	}
	if opts.WithoutAuthentication {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}
	clientOpts = append(clientOpts, opts.Extra...)
	return storage.NewClient(ctx, clientOpts...)
}
