package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// FileSource is an interface that provides the capability to read
// files from various sources (local filesystem, S3, GCS, etc.).
type FileSource interface {
	// Scheme returns the URI scheme this file source handles (e.g., "file", "s3", "gs").
	// An empty string indicates the default handler for paths without a scheme.
	Scheme() string

	// ReadFile reads the file at the given path and returns its contents.
	// The path may include the scheme prefix (e.g., "s3://bucket/key") or be a plain path.
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

// FileSourceRegistry manages file sources for different URI schemes.
type FileSourceRegistry interface {
	// Register registers a file source for a specific URI scheme.
	// The scheme should be lowercase (e.g., "s3", "gs", "http", "https").
	// Use an empty string for the default handler (local filesystem).
	Register(source FileSource) error

	// Get returns the file source for the given path.
	// It parses the URI scheme and returns the appropriate handler.
	// Falls back to the default handler if no scheme is found.
	Get(path string) (FileSource, error)

	// ReadFile is a convenience method that gets the appropriate source
	// and reads the file in one call.
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

// NewFileSourceRegistry creates a new file source registry with
// the default local filesystem handler.
func NewFileSourceRegistry() FileSourceRegistry {
	registry := &fileSourceRegistry{
		sources: make(map[string]FileSource),
	}
	// Register default local filesystem handler
	registry.Register(&LocalFileSource{})
	return registry
}

type fileSourceRegistry struct {
	sources map[string]FileSource
}

func (r *fileSourceRegistry) Register(source FileSource) error {
	scheme := source.Scheme()
	if _, exists := r.sources[scheme]; exists {
		return fmt.Errorf("file source for scheme %q is already registered", scheme)
	}
	r.sources[scheme] = source
	return nil
}

func (r *fileSourceRegistry) Get(path string) (FileSource, error) {
	scheme := extractScheme(path)

	source, exists := r.sources[scheme]
	if !exists {
		if scheme == "" {
			return nil, fmt.Errorf("no default file source handler registered")
		}
		return nil, fmt.Errorf("no file source handler registered for scheme %q", scheme)
	}

	return source, nil
}

func (r *fileSourceRegistry) ReadFile(ctx context.Context, path string) ([]byte, error) {
	source, err := r.Get(path)
	if err != nil {
		return nil, err
	}

	return source.ReadFile(ctx, path)
}

// extractScheme extracts the URI scheme from a path.
// Returns empty string if no scheme is found (local file path).
func extractScheme(path string) string {
	// Check if it looks like a URI with a scheme
	if strings.Contains(path, "://") {
		u, err := url.Parse(path)
		if err == nil && u.Scheme != "" {
			return strings.ToLower(u.Scheme)
		}
	}
	// No scheme found, this is a local path
	return ""
}

// LocalFileSource is the default file source for reading local files.
type LocalFileSource struct{}

func (s *LocalFileSource) Scheme() string {
	// Empty string indicates this is the default handler
	return ""
}

func (s *LocalFileSource) ReadFile(ctx context.Context, path string) ([]byte, error) {
	path = strings.TrimPrefix(path, "file://")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read local file: %w", err)
	}

	return data, nil
}
