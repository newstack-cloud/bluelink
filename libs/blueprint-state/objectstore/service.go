package objectstore

import "context"

// Service is an interface for interacting with object storage services.
type Service interface {
	// Get retrieves the object data and its ETag for the given key.
	Get(ctx context.Context, key string) (data []byte, etag string, err error)
	// Put stores the object data with the given key and options, returning the new ETag.
	Put(ctx context.Context, key string, data []byte, opts *PutOptions) (etag string, err error)
	// Delete removes the object associated with the given key.
	Delete(ctx context.Context, key string) error
	// Head retrieves the size and ETag of the object for the given key without fetching the data.
	Head(ctx context.Context, key string) (*ObjectInfo, error)
	// List returns a list of ObjectInfo for objects that match the given prefix.
	List(ctx context.Context, prefix string) ([]*ObjectInfo, error)
}

// ListerFrom is implemented by a Service whose backing store can begin a listing at a
// key instead of returning every key under a prefix.
//
// Separate from Service because stores differ. S3 takes a start key (StartAfter) and
// Google Cloud Storage takes a start offset, while Azure Blob Storage's flat listing
// offers only an opaque continuation marker and so cannot. A caller type-asserts for it
// and falls back to List, rather than every store having to pretend to support it.
type ListerFrom interface {
	// ListFrom returns the objects under prefix whose keys sort strictly after
	// startAfter, in the same way List returns those under prefix.
	//
	// Exclusive of startAfter itself, matching S3. A caller wanting a key included
	// passes something that sorts just below it: for keys of the form "<prefix><n>-<id>",
	// passing "<prefix><n>" includes every key for n, since a string sorts before any
	// string it prefixes.
	ListFrom(ctx context.Context, prefix string, startAfter string) ([]*ObjectInfo, error)
}

// PutOptions defines options for the Put operation in the object storage service.
type PutOptions struct {
	// IfNoneMatch is used to specify that the Put operation should only succeed if the object does not exist (ETag "*")
	// or if the ETag does not match the provided value.
	IfNoneMatch string
	// IfMatch is used to specify that the Put operation should only succeed
	// if the ETag matches the provided value.
	IfMatch string
}

// ObjectInfo contains metadata about an object stored in the object storage service.
type ObjectInfo struct {
	// The key (or name) of the object in the storage service.
	Key string
	// The size of the object in bytes.
	Size int64
	// The ETag of the object, which is typically a hash of the object's content.
	ETag string
}
