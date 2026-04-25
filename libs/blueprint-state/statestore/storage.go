package statestore

import (
	"context"
	"errors"
)

// Storage is a context aware persistence state container implementations
// can provide to a statestore backed implementation.
// Keys are opaque forward-slash-separated strings; The storage implementation
// maps them to files on a local file system or objects in a remote object store.
type Storage interface {
	// Read returns the full contents of the object at the given key.
	// Returns ErrNotFound if the key does not exist.
	Read(ctx context.Context, key string) ([]byte, error)

	// Write stores data at the given key, overwriting any existing data.
	Write(ctx context.Context, key string, data []byte) error

	// Delete removes the object at key.
	// Returns ErrNotFound if the key does not exist.
	Delete(ctx context.Context, key string) error

	// List returns all keys under the given prefix, recursively.
	// An empty prefix lists every key. Order is down to the implementation.
	List(ctx context.Context, prefix string) ([]string, error)

	// Exists reports whether key is present without reading its contents.
	Exists(ctx context.Context, key string) (bool, error)
}

// ErrNotFound is returned by Storage implementations when a target key does
// not exist. Backends map their native not-found signals onto this sentinel.
var ErrNotFound = errors.New("statestore: key not found")
