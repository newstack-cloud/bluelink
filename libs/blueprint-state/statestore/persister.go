package statestore

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Persister manages durable persistence of state data via the Storage seam.
// Sub-container callers hold State.Lock for the in-memory mutation; Persister
// acquires its own mu internally for chunk-file I/O and index updates.
type Persister struct {
	state                 *State
	storage               Storage
	config                Config
	keys                  KeyBuilder
	logger                core.Logger
	maxGuideFileSize      int64
	maxEventPartitionSize int64

	lastInstanceChunk       int
	lastResourceDriftChunk  int
	lastLinkDriftChunk      int
	lastChangesetChunk      int
	lastValidationChunk     int
	lastReconciliationChunk int

	nameRecordReserver NameRecordReserver

	mu sync.Mutex
}

// NameRecordReserver is an optional hook called by CreateInstance before any
// main-record I/O to atomically reserve an instance name. Backends that
// support conditional writes (objectstore via IfNoneMatch: "*") supply an
// implementation; single-process backends leave it nil and the plain
// write path applies. Returning ErrInstanceNameTaken from the reserver
// aborts the CreateInstance call with that same error surfaced to the caller.
type NameRecordReserver func(ctx context.Context, name, instanceID string) error

// PersisterOption configures a Persister at construction.
type PersisterOption func(*Persister)

// WithNameRecordReserver installs an atomic name-record reservation hook.
// Only meaningful when Config.WriteNameRecords is also true.
func WithNameRecordReserver(r NameRecordReserver) PersisterOption {
	return func(p *Persister) { p.nameRecordReserver = r }
}

// WithMaxGuideFileSize sets the chunk-rollover guide (bytes).
// Default: DefaultMaxGuideFileSize (1 MiB). A single record exceeding this
// size will not be split across chunks; actual file sizes may be larger.
func WithMaxGuideFileSize(maxGuideFileSize int64) PersisterOption {
	return func(p *Persister) {
		p.maxGuideFileSize = maxGuideFileSize
	}
}

// WithMaxEventPartitionSize sets the max bytes in a single event partition.
// Default: DefaultMaxEventPartitionSize (10 MiB). Saving an event that would
// push the partition past this size returns an error.
func WithMaxEventPartitionSize(maxEventPartitionSize int64) PersisterOption {
	return func(p *Persister) {
		p.maxEventPartitionSize = maxEventPartitionSize
	}
}

// WithLogger sets the persister's logger. Defaults to a no-op logger.
func WithLogger(logger core.Logger) PersisterOption {
	return func(p *Persister) {
		p.logger = logger
	}
}

// NewPersister constructs a Persister bound to the given State and Storage.
// prefix is prepended to every storage key (memfile: stateDir; objectstore:
// bucket-level prefix). It may be empty.
func NewPersister(
	state *State,
	storage Storage,
	conf Config,
	prefix string,
	opts ...PersisterOption,
) *Persister {
	p := &Persister{
		state:                   state,
		storage:                 storage,
		config:                  conf,
		keys:                    NewKeyBuilder(prefix),
		logger:                  core.NewNopLogger(),
		maxGuideFileSize:        DefaultMaxGuideFileSize,
		maxEventPartitionSize:   DefaultMaxEventPartitionSize,
		lastInstanceChunk:       maxChunkNumber(state.instanceIndex),
		lastResourceDriftChunk:  maxChunkNumber(state.resourceDriftIndex),
		lastLinkDriftChunk:      maxChunkNumber(state.linkDriftIndex),
		lastChangesetChunk:      maxChunkNumber(state.changesetIndex),
		lastValidationChunk:     maxChunkNumber(state.validationIndex),
		lastReconciliationChunk: maxChunkNumber(state.reconciliationIndex),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Persister) instanceSpec() catSpec {
	return catSpec{
		index:        p.state.instanceIndex,
		lastChunkPtr: &p.lastInstanceChunk,
		chunkKey:     p.keys.InstanceChunk,
		indexKey:     p.keys.InstanceIndex(),
	}
}

func (p *Persister) resourceDriftSpec() catSpec {
	return catSpec{
		index:        p.state.resourceDriftIndex,
		lastChunkPtr: &p.lastResourceDriftChunk,
		chunkKey:     p.keys.ResourceDriftChunk,
		indexKey:     p.keys.ResourceDriftIndex(),
	}
}

func (p *Persister) linkDriftSpec() catSpec {
	return catSpec{
		index:        p.state.linkDriftIndex,
		lastChunkPtr: &p.lastLinkDriftChunk,
		chunkKey:     p.keys.LinkDriftChunk,
		indexKey:     p.keys.LinkDriftIndex(),
	}
}

func (p *Persister) changesetSpec() catSpec {
	return catSpec{
		index:        p.state.changesetIndex,
		lastChunkPtr: &p.lastChangesetChunk,
		chunkKey:     p.keys.ChangesetChunk,
		indexKey:     p.keys.ChangesetIndex(),
	}
}

func (p *Persister) validationSpec() catSpec {
	return catSpec{
		index:        p.state.validationIndex,
		lastChunkPtr: &p.lastValidationChunk,
		chunkKey:     p.keys.ValidationChunk,
		indexKey:     p.keys.ValidationIndex(),
	}
}

func (p *Persister) reconciliationSpec() catSpec {
	return catSpec{
		index:        p.state.reconciliationIndex,
		lastChunkPtr: &p.lastReconciliationChunk,
		chunkKey:     p.keys.ReconciliationResultChunk,
		indexKey:     p.keys.ReconciliationResultIndex(),
	}
}

// CreateInstance persists a newly-added instance. When Config.WriteNameRecords
// is true, the name-lookup record is reserved FIRST (via NameRecordReserver if
// installed, else a plain write) so duplicate-name creates are rejected
// before any main-record I/O. On main-record failure the name record is
// best-effort rolled back.
func (p *Persister) CreateInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.reserveNameRecord(ctx, instance); err != nil {
		return err
	}
	if err := p.writeInstanceRecord(ctx, instance); err != nil {
		_ = p.maybeRemoveNameRecord(ctx, instance.InstanceName)
		return err
	}
	return nil
}

// UpdateInstance rewrites an existing instance in place. If the instance's
// name has changed and Config.WriteNameRecords is true, the stale record is
// left in place (self-corrected on lookup) and a fresh record is written
// for the current name.
func (p *Persister) UpdateInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.rewriteInstanceRecord(ctx, instance); err != nil {
		return err
	}
	return p.maybeRefreshNameRecord(ctx, instance)
}

// RemoveInstance removes an instance's persisted record, its index entry,
// and its name-lookup record (if any).
func (p *Persister) RemoveInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.deleteInstanceRecord(ctx, instance); err != nil {
		return err
	}
	return p.maybeRemoveNameRecord(ctx, instance.InstanceName)
}

func (p *Persister) writeInstanceRecord(ctx context.Context, instance *state.InstanceState) error {
	switch p.config.Instances.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Instance(instance.InstanceID), NewPersistedInstanceState(instance))
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.instanceSpec(),
			NewPersistedInstanceState(instance), instance.InstanceID,
		)
	}
	return errUnknownLayout("instances", p.config.Instances.Layout)
}

func (p *Persister) rewriteInstanceRecord(ctx context.Context, instance *state.InstanceState) error {
	switch p.config.Instances.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Instance(instance.InstanceID), NewPersistedInstanceState(instance))
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.instanceSpec(),
			NewPersistedInstanceState(instance), instance.InstanceID,
			state.InstanceNotFoundError(instance.InstanceID),
		)
	}
	return errUnknownLayout("instances", p.config.Instances.Layout)
}

func (p *Persister) deleteInstanceRecord(ctx context.Context, instance *state.InstanceState) error {
	switch p.config.Instances.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.Instance(instance.InstanceID))
	case LayoutChunked:
		return removeFromChunk[*PersistedInstanceState](
			ctx, p.storage, p.instanceSpec(), instance.InstanceID,
		)
	}
	return errUnknownLayout("instances", p.config.Instances.Layout)
}

// reserveNameRecord installs the name-lookup record. Uses the configured
// NameRecordReserver when present (objectstore's IfNoneMatch-gated write);
// otherwise falls back to an unconditional write (memfile single-process).
// No-op when WriteNameRecords is false or the instance has no name.
func (p *Persister) reserveNameRecord(ctx context.Context, instance *state.InstanceState) error {
	if !p.config.WriteNameRecords || instance.InstanceName == "" {
		return nil
	}
	if p.nameRecordReserver != nil {
		return p.nameRecordReserver(ctx, instance.InstanceName, instance.InstanceID)
	}
	record := &InstanceNameRecord{ID: instance.InstanceID, Name: instance.InstanceName}
	return writeChunk(ctx, p.storage, p.keys.InstanceByName(instance.InstanceName), record)
}

func (p *Persister) maybeRefreshNameRecord(ctx context.Context, instance *state.InstanceState) error {
	if !p.config.WriteNameRecords || instance.InstanceName == "" {
		return nil
	}
	// If the name changed, the previous record is orphaned but harmless —
	// LookupInstanceIDByName verifies record→instance consistency by reading
	// the instance it points at and rejecting stale results. We still
	// write the new record so the current name resolves cheaply.
	record := &InstanceNameRecord{ID: instance.InstanceID, Name: instance.InstanceName}
	return writeChunk(ctx, p.storage, p.keys.InstanceByName(instance.InstanceName), record)
}

func (p *Persister) maybeRemoveNameRecord(ctx context.Context, name string) error {
	if !p.config.WriteNameRecords || name == "" {
		return nil
	}
	return deleteIgnoreNotFound(ctx, p.storage, p.keys.InstanceByName(name))
}

func (p *Persister) CreateResourceDrift(ctx context.Context, drift *state.ResourceDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ResourceDrift.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.ResourceDrift(drift.ResourceID), drift)
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.resourceDriftSpec(),
			drift, drift.ResourceID,
		)
	}
	return errUnknownLayout("resource_drift", p.config.ResourceDrift.Layout)
}

func (p *Persister) UpdateResourceDrift(ctx context.Context, drift *state.ResourceDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ResourceDrift.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.ResourceDrift(drift.ResourceID), drift)
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.resourceDriftSpec(),
			drift, drift.ResourceID,
			state.ResourceNotFoundError(drift.ResourceID),
		)
	}
	return errUnknownLayout("resource_drift", p.config.ResourceDrift.Layout)
}

func (p *Persister) RemoveResourceDrift(ctx context.Context, drift *state.ResourceDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ResourceDrift.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.ResourceDrift(drift.ResourceID))
	case LayoutChunked:
		return removeFromChunk[*state.ResourceDriftState](
			ctx, p.storage, p.resourceDriftSpec(), drift.ResourceID,
		)
	}
	return errUnknownLayout("resource_drift", p.config.ResourceDrift.Layout)
}

func (p *Persister) CreateLinkDrift(ctx context.Context, drift *state.LinkDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.LinkDrift.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.LinkDrift(drift.LinkID), drift)
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.linkDriftSpec(),
			drift, drift.LinkID,
		)
	}
	return errUnknownLayout("link_drift", p.config.LinkDrift.Layout)
}

func (p *Persister) UpdateLinkDrift(ctx context.Context, drift *state.LinkDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.LinkDrift.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.LinkDrift(drift.LinkID), drift)
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.linkDriftSpec(),
			drift, drift.LinkID,
			state.LinkNotFoundError(drift.LinkID),
		)
	}
	return errUnknownLayout("link_drift", p.config.LinkDrift.Layout)
}

func (p *Persister) RemoveLinkDrift(ctx context.Context, drift *state.LinkDriftState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.LinkDrift.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.LinkDrift(drift.LinkID))
	case LayoutChunked:
		return removeFromChunk[*state.LinkDriftState](
			ctx, p.storage, p.linkDriftSpec(), drift.LinkID,
		)
	}
	return errUnknownLayout("link_drift", p.config.LinkDrift.Layout)
}

func (p *Persister) CreateChangeset(ctx context.Context, cs *manage.Changeset) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Changesets.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Changeset(cs.ID), cs)
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.changesetSpec(),
			cs, cs.ID,
		)
	}
	return errUnknownLayout("changesets", p.config.Changesets.Layout)
}

func (p *Persister) UpdateChangeset(ctx context.Context, cs *manage.Changeset) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Changesets.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Changeset(cs.ID), cs)
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.changesetSpec(),
			cs, cs.ID,
			errMalformedState("changeset index entry missing for "+cs.ID),
		)
	}
	return errUnknownLayout("changesets", p.config.Changesets.Layout)
}

func (p *Persister) RemoveChangeset(ctx context.Context, changesetID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Changesets.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.Changeset(changesetID))
	case LayoutChunked:
		return removeFromChunk[*manage.Changeset](
			ctx, p.storage, p.changesetSpec(), changesetID,
		)
	}
	return errUnknownLayout("changesets", p.config.Changesets.Layout)
}

func (p *Persister) CreateValidation(ctx context.Context, v *manage.BlueprintValidation) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Validations.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Validation(v.ID), v)
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.validationSpec(),
			v, v.ID,
		)
	}
	return errUnknownLayout("validations", p.config.Validations.Layout)
}

func (p *Persister) UpdateValidation(ctx context.Context, v *manage.BlueprintValidation) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Validations.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.Validation(v.ID), v)
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.validationSpec(),
			v, v.ID,
			errMalformedState("validation index entry missing for "+v.ID),
		)
	}
	return errUnknownLayout("validations", p.config.Validations.Layout)
}

func (p *Persister) RemoveValidation(ctx context.Context, validationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.Validations.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.Validation(validationID))
	case LayoutChunked:
		return removeFromChunk[*manage.BlueprintValidation](
			ctx, p.storage, p.validationSpec(), validationID,
		)
	}
	return errUnknownLayout("validations", p.config.Validations.Layout)
}

func (p *Persister) CreateReconciliationResult(ctx context.Context, r *manage.ReconciliationResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ReconciliationResults.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.ReconciliationResult(r.ID), r)
	case LayoutChunked:
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, p.reconciliationSpec(),
			r, r.ID,
		)
	}
	return errUnknownLayout("reconciliation_results", p.config.ReconciliationResults.Layout)
}

func (p *Persister) UpdateReconciliationResult(ctx context.Context, r *manage.ReconciliationResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ReconciliationResults.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.ReconciliationResult(r.ID), r)
	case LayoutChunked:
		return updateInChunk(
			ctx, p.storage, p.reconciliationSpec(),
			r, r.ID,
			errMalformedState("reconciliation result index entry missing for "+r.ID),
		)
	}
	return errUnknownLayout("reconciliation_results", p.config.ReconciliationResults.Layout)
}

func (p *Persister) RemoveReconciliationResult(ctx context.Context, resultID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.ReconciliationResults.Layout {
	case LayoutPerEntity:
		return deleteIgnoreNotFound(ctx, p.storage, p.keys.ReconciliationResult(resultID))
	case LayoutChunked:
		return removeFromChunk[*manage.ReconciliationResult](
			ctx, p.storage, p.reconciliationSpec(), resultID,
		)
	}
	return errUnknownLayout("reconciliation_results", p.config.ReconciliationResults.Layout)
}

// SaveEventPartition writes the full partition for a channel and records
// the event's position in the event index. Returns
// ErrorReasonCodeMaxEventPartitionSizeExceeded if the serialised partition
// would exceed the configured maximum.
func (p *Persister) SaveEventPartition(
	ctx context.Context,
	partitionName string,
	partition []*manage.Event,
	eventToSave *manage.Event,
	indexInPartition int,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	partitionData, err := json.Marshal(partition)
	if err != nil {
		return err
	}

	if int64(len(partitionData)) > p.maxEventPartitionSize {
		return errMaxEventPartitionSizeExceeded(partitionName, p.maxEventPartitionSize)
	}

	if err := p.storage.Write(ctx, p.keys.EventPartition(partitionName), partitionData); err != nil {
		return err
	}

	p.state.eventIndex[eventToSave.ID] = &EventIndexLocation{
		Partition:        toPartitionFileBaseName(partitionName),
		IndexInPartition: indexInPartition,
	}

	return writeIndex(ctx, p.storage, p.keys.EventIndex(), p.state.eventIndex)
}

// UpdateEventPartitionsForRemovals persists the post-cleanup state of event
// partitions. It deletes files for fully-removed partitions, overwrites the
// remaining partitions, and drops the removed event IDs from the event index.
func (p *Persister) UpdateEventPartitionsForRemovals(
	ctx context.Context,
	partitions map[string][]*manage.Event,
	removedPartitions []string,
	removedEvents []string,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, partitionName := range removedPartitions {
		if err := deleteIgnoreNotFound(ctx, p.storage, p.keys.EventPartition(partitionName)); err != nil {
			return err
		}
	}

	for partitionName, partition := range partitions {
		partitionData, err := json.Marshal(partition)
		if err != nil {
			return err
		}
		if err := p.storage.Write(ctx, p.keys.EventPartition(partitionName), partitionData); err != nil {
			return err
		}
	}

	for _, eventID := range removedEvents {
		delete(p.state.eventIndex, eventID)
	}

	return writeIndex(ctx, p.storage, p.keys.EventIndex(), p.state.eventIndex)
}

// GetEventIndexEntry returns the indexed location of an event (or nil if not
// found). Acquires the persister mutex so readers see a consistent view of
// the event index while a write is in progress.
func (p *Persister) GetEventIndexEntry(eventID string) *EventIndexLocation {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.state.eventIndex[eventID]
	if !ok {
		return nil
	}
	return entry
}

func toPartitionFileBaseName(partitionName string) string {
	return "events__" + partitionName
}

// CleanupChangesets removes changesets older than thresholdDate and
// re-persists the remaining entries. Returns a lookup of retained
// changesets keyed by ID so callers can refresh their in-memory view.
func (p *Persister) CleanupChangesets(
	ctx context.Context,
	thresholdDate time.Time,
) (map[string]*manage.Changeset, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	threshold := thresholdDate.Unix()
	spec := p.changesetSpec()
	kept, err := loadAndFilterChunked(
		ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey,
		func(cs *manage.Changeset) bool { return cs.Created > threshold },
	)
	if err != nil {
		return nil, err
	}

	if err := resetChunkedCategory(ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey, spec.indexKey); err != nil {
		return nil, err
	}
	clear(spec.index)
	*spec.lastChunkPtr = 0

	lookup := make(map[string]*manage.Changeset, len(kept))
	for _, cs := range kept {
		if err := createInChunk(ctx, p.storage, p.maxGuideFileSize, spec, cs, cs.ID); err != nil {
			return nil, err
		}
		lookup[cs.ID] = cs
	}
	return lookup, nil
}

// CleanupValidations removes blueprint validations older than thresholdDate
// and re-persists the remaining entries.
func (p *Persister) CleanupValidations(
	ctx context.Context,
	thresholdDate time.Time,
) (map[string]*manage.BlueprintValidation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	threshold := thresholdDate.Unix()
	spec := p.validationSpec()
	kept, err := loadAndFilterChunked(
		ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey,
		func(v *manage.BlueprintValidation) bool { return v.Created > threshold },
	)
	if err != nil {
		return nil, err
	}

	if err := resetChunkedCategory(ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey, spec.indexKey); err != nil {
		return nil, err
	}
	clear(spec.index)
	*spec.lastChunkPtr = 0

	lookup := make(map[string]*manage.BlueprintValidation, len(kept))
	for _, v := range kept {
		if err := createInChunk(ctx, p.storage, p.maxGuideFileSize, spec, v, v.ID); err != nil {
			return nil, err
		}
		lookup[v.ID] = v
	}
	return lookup, nil
}

// CleanupReconciliationResults removes reconciliation results older than
// thresholdDate and re-persists the remaining entries.
func (p *Persister) CleanupReconciliationResults(
	ctx context.Context,
	thresholdDate time.Time,
) (map[string]*manage.ReconciliationResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	threshold := thresholdDate.Unix()
	spec := p.reconciliationSpec()
	kept, err := loadAndFilterChunked(
		ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey,
		func(r *manage.ReconciliationResult) bool { return r.Created > threshold },
	)
	if err != nil {
		return nil, err
	}

	if err := resetChunkedCategory(ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey, spec.indexKey); err != nil {
		return nil, err
	}
	clear(spec.index)
	*spec.lastChunkPtr = 0

	lookup := make(map[string]*manage.ReconciliationResult, len(kept))
	for _, r := range kept {
		if err := createInChunk(ctx, p.storage, p.maxGuideFileSize, spec, r, r.ID); err != nil {
			return nil, err
		}
		lookup[r.ID] = r
	}
	return lookup, nil
}

// CreateCleanupOperation persists a newly-recorded cleanup operation.
// Only LayoutPerEntity is supported — cleanup operations are low-volume
// globally-shared entities where per-object keys give the simplest story
// for concurrent CI/CD runs.
func (p *Persister) CreateCleanupOperation(ctx context.Context, op *manage.CleanupOperation) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.CleanupOperations.Layout != LayoutPerEntity {
		return errUnknownLayout("cleanup_operations", p.config.CleanupOperations.Layout)
	}
	return writeChunk(ctx, p.storage, p.keys.CleanupOperation(op.ID), op)
}

// UpdateCleanupOperation rewrites an existing cleanup operation.
func (p *Persister) UpdateCleanupOperation(ctx context.Context, op *manage.CleanupOperation) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.CleanupOperations.Layout != LayoutPerEntity {
		return errUnknownLayout("cleanup_operations", p.config.CleanupOperations.Layout)
	}
	return writeChunk(ctx, p.storage, p.keys.CleanupOperation(op.ID), op)
}

// RemoveCleanupOperation deletes a cleanup operation's persisted record.
// A missing target is not an error (matches other Remove semantics).
func (p *Persister) RemoveCleanupOperation(ctx context.Context, operationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.CleanupOperations.Layout != LayoutPerEntity {
		return errUnknownLayout("cleanup_operations", p.config.CleanupOperations.Layout)
	}
	return deleteIgnoreNotFound(ctx, p.storage, p.keys.CleanupOperation(operationID))
}

func deleteIgnoreNotFound(ctx context.Context, storage Storage, key string) error {
	if err := storage.Delete(ctx, key); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return nil
}
