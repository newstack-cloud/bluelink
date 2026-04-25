package statestore

// Layout determines how entities are organised on disk or in a cloud object store.
type Layout int

const (
	// LayoutChunked packs many entities into JSON chunk files with a shared index file.
	// Small per-file I/O overhead. fewer writes per mutation.
	LayoutChunked Layout = iota

	// LayoutPerEntity stores each entity as its own JSON object keyed by ID.
	// Predictable keys enable per-entity ETag CAS at the storage layer for backends
	// that support it, and avoids cross-run chunk contention for globally shared
	// entity types (e.g. changesets, validations).
	LayoutPerEntity
)

type Scope int

const (
	// ScopeGlobal puts all entities for the category in one namespace,
	// shared across all blueprint instances.
	ScopeGlobal Scope = iota

	// ScopePerInstance prefixes all keys with "instances/{instanceID}/".
	// Two runs deploying different instances touch disjoint namespaces
	// meaning no chunk contention even without CAS on chunk writes.
	ScopePerInstance

	// ScopePerChannel prefixes keys with the event channel identifier
	// (e.g. "channels/{channelType}/{channelID}/"). This is the
	// partitioning that existed in the `memfile` implementation
	// before the introduction of the `statestore` package.
	ScopePerChannel
)

// CategoryConfig defines the layout and scope for a category of state entities.
type CategoryConfig struct {
	Layout Layout
	Scope  Scope
}

// Config defines the layout and scope for each category of state entities in a state container.
type Config struct {
	// Mode selects eager vs lazy cache population for State. Defaults to
	// ModeEager (zero value) for backwards compatibility with memfile.
	Mode LoadMode

	// WriteNameRecords toggles per-entity name-lookup record writes alongside
	// instance creates/updates/removes. Required for ModeLazy so
	// LookupInstanceIDByName can avoid enumerating all instances. Memfile
	// (ModeEager) leaves this false and resolves names from its in-memory map.
	WriteNameRecords bool

	Instances             CategoryConfig
	Resources             CategoryConfig
	Links                 CategoryConfig
	ResourceDrift         CategoryConfig
	LinkDrift             CategoryConfig
	Events                CategoryConfig
	Changesets            CategoryConfig
	Validations           CategoryConfig
	ReconciliationResults CategoryConfig
	CleanupOperations     CategoryConfig
}
