package statestore

import (
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// State is the in-memory representation of all state data managed by the state store.
type State struct {
	mu *sync.RWMutex

	// loader materialises entities on cache miss under ModeLazy. Always
	// non-nil: under ModeEager it's a noopLoader that returns (nil, false, nil).
	loader EntityLoader

	// nameLookup caches instance name → id resolutions. Populated lazily from
	// loader.LoadInstanceIDByName under ModeLazy; kept in sync by
	// SetInstanceInMemory / RemoveInstanceFromMemory under ModeEager.
	nameLookup map[string]string

	instances       map[string]*state.InstanceState
	resources       map[string]*state.ResourceState
	resourceDrift   map[string]*state.ResourceDriftState
	links           map[string]*state.LinkState
	linkDrift       map[string]*state.LinkDriftState
	events          map[string]*manage.Event
	partitionEvents map[string][]*manage.Event
	changesets      map[string]*manage.Changeset
	validations     map[string]*manage.BlueprintValidation
	reconciliations map[string]*manage.ReconciliationResult
	cleanupOps      map[string]*manage.CleanupOperation

	instanceIndex       map[string]*IndexLocation
	resourceChunkIndex  map[string]*IndexLocation
	linkChunkIndex      map[string]*IndexLocation
	resourceDriftIndex  map[string]*IndexLocation
	linkDriftIndex      map[string]*IndexLocation
	eventIndex          map[string]*EventIndexLocation
	changesetIndex      map[string]*IndexLocation
	validationIndex     map[string]*IndexLocation
	reconciliationIndex map[string]*IndexLocation
	cleanupOpIndex      map[string]*IndexLocation
}

// IndexLocation records the position of an entity within a chunk file.
// Used for LayoutChunked categories where the index file maps entity IDs
// to (chunk number, position-in-chunk) pairs.
type IndexLocation struct {
	ChunkNumber  int `json:"chunkNumber"`
	IndexInChunk int `json:"indexInChunk"`
}

// EventIndexLocation records the position of an event within its channel's
// partition file. Used by the Events category under ScopePerChannel.
type EventIndexLocation struct {
	Partition        string `json:"partition"`
	IndexInPartition int    `json:"indexInPartition"`
}

// Lock helpers for callers that need to hold the State lock across a sequence
// of operations. Prefer the focused accessors below where possible.
func (s *State) Lock()    { s.mu.Lock() }
func (s *State) Unlock()  { s.mu.Unlock() }
func (s *State) RLock()   { s.mu.RLock() }
func (s *State) RUnlock() { s.mu.RUnlock() }

// Instance returns the in-memory instance pointer. The caller must already
// hold at least an RLock on the State; the returned pointer is shared and
// must not be mutated without holding the write lock. For durable updates,
// go through the Persister.
func (s *State) Instance(instanceID string) (*state.InstanceState, bool) {
	inst, ok := s.instances[instanceID]
	return inst, ok
}

// EachInstance iterates over all instances under an RLock held internally.
// The visitor must not mutate the instance. Return false to stop early.
func (s *State) EachInstance(visit func(*state.InstanceState) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, inst := range s.instances {
		if !visit(inst) {
			return
		}
	}
}

// SetInstanceInMemory replaces (or inserts) an instance in the in-memory map
// without triggering any persistence. Also keeps the name-lookup cache in sync.
// Intended for:
//   - the loader, populating State from Storage on startup, and
//   - backend flows that handled durability out of band (e.g. objectstore's
//     ClaimForDeployment via a direct ETag CAS on the Service).
//
// Acquires the write lock internally.
func (s *State) SetInstanceInMemory(instance *state.InstanceState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.instances[instance.InstanceID]; ok && prev.InstanceName != instance.InstanceName {
		delete(s.nameLookup, prev.InstanceName)
	}
	s.instances[instance.InstanceID] = instance
	if instance.InstanceName != "" {
		s.nameLookup[instance.InstanceName] = instance.InstanceID
	}
}

// RemoveInstanceFromMemory removes an instance from the in-memory map and
// its name-lookup entry. Acquires the write lock internally.
func (s *State) RemoveInstanceFromMemory(instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.instances[instanceID]; ok && prev.InstanceName != "" {
		delete(s.nameLookup, prev.InstanceName)
	}
	delete(s.instances, instanceID)
}

// RebuildNameLookup regenerates the name → id cache from the current
// instances map. Intended for ModeEager backends that populate State's
// instances pointer-share-style (e.g. memfile's loadStateFromDir) without
// going through SetInstanceInMemory.
func (s *State) RebuildNameLookup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nameLookup = make(map[string]string, len(s.instances))
	for id, inst := range s.instances {
		if inst.InstanceName != "" {
			s.nameLookup[inst.InstanceName] = id
		}
	}
}

// StateOption configures a State at construction.
type StateOption func(*State)

// WithEntityLoader injects the loader used to materialise entities on cache
// miss under ModeLazy. Memfile passes a noopLoader (or omits the option);
// objectstore supplies its Service-backed loader.
func WithEntityLoader(l EntityLoader) StateOption {
	return func(s *State) { s.loader = l }
}

// WithSharedMutex replaces State's internal RWMutex with the given pointer.
// Used by memfile during its incremental migration so legacy sub-containers
// and statestore.Persister lock against the same primitive.
func WithSharedMutex(mu *sync.RWMutex) StateOption {
	return func(s *State) { s.mu = mu }
}

// WithSharedInstances replaces State's instance map with the given pointer.
// The caller retains ownership; changes made via either reference are visible
// through the other.
func WithSharedInstances(instances map[string]*state.InstanceState) StateOption {
	return func(s *State) { s.instances = instances }
}

// WithSharedInstanceIndex replaces State's instance chunk-index map with the
// given pointer. Same sharing semantics as WithSharedInstances.
func WithSharedInstanceIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.instanceIndex = index }
}

// WithSharedResources replaces State's resource map with the given pointer.
func WithSharedResources(resources map[string]*state.ResourceState) StateOption {
	return func(s *State) { s.resources = resources }
}

// WithSharedLinks replaces State's link map with the given pointer.
func WithSharedLinks(links map[string]*state.LinkState) StateOption {
	return func(s *State) { s.links = links }
}

// WithSharedResourceDrift replaces State's resource-drift map with the given
// pointer. Same sharing semantics as WithSharedInstances.
func WithSharedResourceDrift(drift map[string]*state.ResourceDriftState) StateOption {
	return func(s *State) { s.resourceDrift = drift }
}

// WithSharedResourceDriftIndex replaces State's resource-drift chunk-index
// map with the given pointer. Same sharing semantics as WithSharedInstances.
func WithSharedResourceDriftIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.resourceDriftIndex = index }
}

// WithSharedLinkDrift replaces State's link-drift map.
func WithSharedLinkDrift(drift map[string]*state.LinkDriftState) StateOption {
	return func(s *State) { s.linkDrift = drift }
}

// WithSharedLinkDriftIndex replaces State's link-drift chunk-index map.
func WithSharedLinkDriftIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.linkDriftIndex = index }
}

// WithSharedChangesets replaces State's changesets map.
func WithSharedChangesets(changesets map[string]*manage.Changeset) StateOption {
	return func(s *State) { s.changesets = changesets }
}

// WithSharedChangesetIndex replaces State's changeset chunk-index map.
func WithSharedChangesetIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.changesetIndex = index }
}

// WithSharedValidations replaces State's validations map.
func WithSharedValidations(validations map[string]*manage.BlueprintValidation) StateOption {
	return func(s *State) { s.validations = validations }
}

// WithSharedValidationIndex replaces State's validation chunk-index map.
func WithSharedValidationIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.validationIndex = index }
}

// WithSharedReconciliations replaces State's reconciliation results map.
func WithSharedReconciliations(results map[string]*manage.ReconciliationResult) StateOption {
	return func(s *State) { s.reconciliations = results }
}

// WithSharedReconciliationIndex replaces State's reconciliation chunk-index map.
func WithSharedReconciliationIndex(index map[string]*IndexLocation) StateOption {
	return func(s *State) { s.reconciliationIndex = index }
}

// WithSharedEvents replaces State's events map.
func WithSharedEvents(events map[string]*manage.Event) StateOption {
	return func(s *State) { s.events = events }
}

// WithSharedPartitionEvents replaces State's partition events map.
func WithSharedPartitionEvents(partitionEvents map[string][]*manage.Event) StateOption {
	return func(s *State) { s.partitionEvents = partitionEvents }
}

// WithSharedEventIndex replaces State's event index map.
func WithSharedEventIndex(index map[string]*EventIndexLocation) StateOption {
	return func(s *State) { s.eventIndex = index }
}

// WithSharedCleanupOps replaces State's cleanup-operations map.
func WithSharedCleanupOps(ops map[string]*manage.CleanupOperation) StateOption {
	return func(s *State) { s.cleanupOps = ops }
}

// NewState returns a new, empty State instance. Options let callers share
// specific maps or the mutex with an external owner (memfile's legacy state
// during the incremental migration).
func NewState(opts ...StateOption) *State {
	s := &State{
		mu:                  &sync.RWMutex{},
		loader:              noopLoader{},
		nameLookup:          map[string]string{},
		instances:           map[string]*state.InstanceState{},
		resources:           map[string]*state.ResourceState{},
		resourceDrift:       map[string]*state.ResourceDriftState{},
		links:               map[string]*state.LinkState{},
		linkDrift:           map[string]*state.LinkDriftState{},
		events:              map[string]*manage.Event{},
		partitionEvents:     map[string][]*manage.Event{},
		changesets:          map[string]*manage.Changeset{},
		validations:         map[string]*manage.BlueprintValidation{},
		reconciliations:     map[string]*manage.ReconciliationResult{},
		cleanupOps:          map[string]*manage.CleanupOperation{},
		instanceIndex:       map[string]*IndexLocation{},
		resourceChunkIndex:  map[string]*IndexLocation{},
		linkChunkIndex:      map[string]*IndexLocation{},
		resourceDriftIndex:  map[string]*IndexLocation{},
		linkDriftIndex:      map[string]*IndexLocation{},
		eventIndex:          map[string]*EventIndexLocation{},
		changesetIndex:      map[string]*IndexLocation{},
		validationIndex:     map[string]*IndexLocation{},
		reconciliationIndex: map[string]*IndexLocation{},
		cleanupOpIndex:      map[string]*IndexLocation{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
