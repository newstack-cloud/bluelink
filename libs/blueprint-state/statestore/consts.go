package statestore

const (
	// DefaultMaxGuideFileSize is the default maximum size of a state chunk file
	// in bytes. Used by the persister as a guide for chunk rollover — if a
	// single record exceeds this size, it will not be split across multiple
	// files. Actual file sizes are often slightly larger than this guide.
	DefaultMaxGuideFileSize int64 = 1048576

	// DefaultMaxEventPartitionSize is the default maximum size of an event
	// partition file in bytes. When saving a new event would push the partition
	// past this size, the persister returns an error for the save operation.
	DefaultMaxEventPartitionSize int64 = 10485760
)
