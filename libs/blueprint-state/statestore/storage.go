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

// ConditionalStorage is implemented by a Storage whose objects carry a version (an ETag)
// that a write or delete can be made conditional on. It lets processes that share the
// same storage detect that another process changed an object since it was read, instead of
// overwriting the change.
//
// Optional because not every store versions its objects. The Persister type-asserts for it
// and, without it, writes unconditionally as it would for a single process.
type ConditionalStorage interface {
	// ReadVersion returns the contents of the object at key and its current version.
	// Returns ErrNotFound if the key does not exist.
	ReadVersion(ctx context.Context, key string) (data []byte, version string, err error)

	// WriteIfMatch stores data at key only if the object's current version is version,
	// and returns the new version. An empty version means the write only succeeds if no
	// object exists at key. Returns ErrPreconditionFailed otherwise.
	WriteIfMatch(ctx context.Context, key string, data []byte, version string) (newVersion string, err error)

	// DeleteIfMatch removes the object at key only if its current version is version.
	// Returns ErrPreconditionFailed if the version does not match, and ErrNotFound if the
	// key does not exist.
	DeleteIfMatch(ctx context.Context, key string, version string) error
}

// ErrPreconditionFailed is returned by ConditionalStorage implementations when the object
// at a key is not at the version a conditional write or delete required.
var ErrPreconditionFailed = errors.New("statestore: precondition failed")
