package statestore

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strconv"
	"strings"
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
	lastAuditRecordChunk    int
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
		lastAuditRecordChunk:    maxChunkNumber(state.auditRecordIndex),
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

func (p *Persister) auditRecordSpec() catSpec {
	return catSpec{
		index:        p.state.auditRecordIndex,
		lastChunkPtr: &p.lastAuditRecordChunk,
		chunkKey:     p.keys.AuditRecordChunk,
		indexKey:     p.keys.AuditRecordIndex(),
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
// before any main-record I/O. If the instance object could not be written, the
// name record is rolled back on a best-effort basis.
//
// The caller must hold the State write lock for state cache updates.
func (p *Persister) CreateInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.reserveNameRecord(ctx, instance); err != nil {
		p.state.forgetCachedInstances()
		return err
	}
	instanceWritten, err := p.createInstanceRecords(ctx, instance)
	if err != nil {
		// Once the instance object exists the name record belongs to it.
		if !instanceWritten {
			_ = p.maybeRemoveNameRecord(ctx, instance.InstanceName)
		}
		p.state.forgetCachedInstances()
		return err
	}
	return nil
}

// UpdateInstance rewrites an existing instance in place, for a change that adds or removes
// no resources or links (see UpdateInstanceEntities for one that does). Instance names
// cannot change, so the name record is left as it is.
//
// The caller must hold the State write lock for state cache updates.
func (p *Persister) UpdateInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.rewriteInstanceRecord(ctx, instance)
	if err != nil {
		p.state.forgetCachedInstances()
	}
	return err
}

// RemoveInstance removes an instance's persisted record, its index entry,
// its name-lookup record (if any) and, under Config.WriteEntityRecords, the ID
// records of its resources and links.
func (p *Persister) RemoveInstance(ctx context.Context, instance *state.InstanceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.WriteEntityRecords {
		return p.removeInstanceWithRecords(ctx, instance)
	}

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

	if err := writeIndex(ctx, p.storage, p.keys.EventIndex(), p.state.eventIndex); err != nil {
		// Rolled back, since the caller abandons the save on this error and does not
		// commit the event to the in-memory partition. An index entry left pointing
		// into a partition that does not hold the event resolves a stream's starting
		// position against the wrong offset.
		delete(p.state.eventIndex, eventToSave.ID)
		return err
	}

	return nil
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
) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return cleanupCategory[*manage.Changeset](
		ctx, p, p.config.Changesets, p.changesetSpec(),
		p.keys.ChangesetsPrefix(), thresholdDate, "changesets",
	)
}

// CreateAuditRecord persists a newly-emitted audit record. There is no update
// counterpart as an audit trail the acting principal can amend is not an audit trail.
func (p *Persister) CreateAuditRecord(ctx context.Context, r *manage.AuditRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.config.AuditRecords.Layout {
	case LayoutPerEntity:
		return writeChunk(ctx, p.storage, p.keys.AuditRecordObject(r.Timestamp, r.ID), r)
	case LayoutChunked:
		spec := p.auditRecordSpec()
		// Appending unconditionally would put the same ID in the trail twice, since
		// nothing here updates an existing entry. A record already stored under this ID
		// stands.
		if _, recorded := spec.index[r.ID]; recorded {
			return nil
		}
		return createInChunk(
			ctx, p.storage, p.maxGuideFileSize, spec,
			r, r.ID,
		)
	}
	return errUnknownLayout("audit records", p.config.AuditRecords.Layout)
}

// QueryAuditRecords returns the audit records matching query, oldest-first by
// timestamp and then by ID.
//
// Records are read from the chunk files rather than from the in-memory lookup,
// which under LoadMode ModeLazy holds only the subset of records that have been
// read back individually. A partial audit trail presented as a complete one is
// worse than a slower query.
func (p *Persister) QueryAuditRecords(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.AuditRecords.Layout != LayoutChunked {
		return nil, errAuditQueryUnsupported(p.config.AuditRecords.Layout)
	}

	spec := p.auditRecordSpec()
	matched, err := loadAndFilterChunked(
		ctx, p.storage, *spec.lastChunkPtr, spec.chunkKey,
		func(r *manage.AuditRecord) bool { return AuditRecordMatches(r, query) },
	)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(matched, compareAuditRecords)
	return applyAuditQueryLimit(matched, query), nil
}

// NarrowAuditRecords filters, orders and limits records according to query.
//
// Backends that enumerate their own records rather than reading them from chunk
// files apply the query through here, so that no read path drifts from the
// filtering and ordering manage.Audit documents.
func NarrowAuditRecords(
	records []*manage.AuditRecord,
	query *manage.AuditRecordQuery,
) []*manage.AuditRecord {
	matched := make([]*manage.AuditRecord, 0, len(records))
	for _, record := range records {
		if AuditRecordMatches(record, query) {
			matched = append(matched, record)
		}
	}
	slices.SortFunc(matched, compareAuditRecords)
	return applyAuditQueryLimit(matched, query)
}

func applyAuditQueryLimit(
	records []*manage.AuditRecord,
	query *manage.AuditRecordQuery,
) []*manage.AuditRecord {
	limit := AuditQueryLimit(query)
	if len(records) > limit {
		return records[:limit]
	}
	return records
}

// AuditQueryLimit is how many records the query asks for, resolved.
//
// Exported alongside AuditRecordMatches so that a backend reading records one at a time
// knows when it has enough and can stop, rather than reading the full trail and discarding
// most of it.
func AuditQueryLimit(query *manage.AuditRecordQuery) int {
	if query.Limit <= 0 {
		return manage.DefaultAuditQueryLimit
	}

	return query.Limit
}

// AuditRecordMatches reports whether one record satisfies the query.
//
// Exported for a backend that holds one object per record and reads them one at a time,
// so that what counts as a match is decided here rather than in each backend.
func AuditRecordMatches(record *manage.AuditRecord, query *manage.AuditRecordQuery) bool {
	if query.From != 0 && record.Timestamp < query.From {
		return false
	}

	if !isAfterAuditCursor(record, query) {
		return false
	}

	if query.To != 0 && record.Timestamp >= query.To {
		return false
	}

	if query.Action != "" && record.Action != query.Action {
		return false
	}

	if query.PrincipalSubject != "" &&
		record.PrincipalSubject != query.PrincipalSubject {
		return false
	}

	return query.Outcome == "" || record.Outcome == query.Outcome
}

// The cursor is a (timestamp, ID) pair rather than a timestamp, so that a second
// holding more records than the query limit can still be read past.
func isAfterAuditCursor(
	record *manage.AuditRecord,
	query *manage.AuditRecordQuery,
) bool {
	// Keyed on the timestamp alone so that this agrees with the SQL backend, which
	// cannot distinguish an unset cursor from one at the epoch.
	if query.AfterTimestamp == 0 {
		return true
	}
	if record.Timestamp != query.AfterTimestamp {
		return record.Timestamp > query.AfterTimestamp
	}
	return record.ID > query.AfterID
}

// ID breaks the tie for records sharing a timestamp, per the ordering manage.Audit
// documents.
func compareAuditRecords(a *manage.AuditRecord, b *manage.AuditRecord) int {
	if a.Timestamp != b.Timestamp {
		return cmp.Compare(a.Timestamp, b.Timestamp)
	}
	return cmp.Compare(a.ID, b.ID)
}

// CleanupAuditRecords removes audit records older than thresholdDate and
// re-persists the remaining entries.
func (p *Persister) CleanupAuditRecords(
	ctx context.Context,
	thresholdDate time.Time,
) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.AuditRecords.Layout == LayoutPerEntity {
		return cleanupAuditRecordObjects(
			ctx, p.storage, p.keys.AuditRecordsPrefix(), thresholdDate.Unix(),
		)
	}

	return cleanupCategory[*manage.AuditRecord](
		ctx, p, p.config.AuditRecords, p.auditRecordSpec(),
		p.keys.AuditRecordsPrefix(), thresholdDate, "audit records",
	)
}

// Removes the records that have passed their retention period, deciding which from the
// key rather than from the record.
//
// A trail is one object per record under the per-entity layout, which is the only caller of this.
// Without this, the generic per-entity cleanup
// would read every record in the trail to find the ones to delete. The key carries the
// timestamp, so a pass deletes without reading any of them.
//
// A key that is not a record key stops the pass rather than being skipped.
// Deleting around an object this code cannot identify, or leaving it to accumulate unnoticed, are
// both worse than saying so.
func cleanupAuditRecordObjects(
	ctx context.Context,
	storage Storage,
	prefix string,
	threshold int64,
) ([]string, error) {
	keys, err := storage.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	removed := []string{}
	for _, key := range keys {
		timestamp, recordID, ok := ParseTimestampedKey(key)
		if !ok {
			return removed, NotAnAuditRecordKeyError(key)
		}
		if timestamp > threshold {
			continue
		}

		if err := deleteIgnoreNotFound(ctx, storage, key); err != nil {
			return removed, err
		}
		removed = append(removed, recordID)
	}

	return removed, nil
}

// Removes the records of one category that have passed their retention period, by
// whichever route the category's configured layout is actually stored under, and
// returns the IDs it removed.
//
// A free function rather than a method because Go methods cannot take type parameters,
// and the entity type is what lets one implementation serve every category.
func cleanupCategory[T manage.Entity](
	ctx context.Context,
	p *Persister,
	category CategoryConfig,
	spec catSpec,
	prefix string,
	thresholdDate time.Time,
	categoryName string,
) ([]string, error) {
	threshold := thresholdDate.Unix()

	switch category.Layout {
	case LayoutChunked:
		return cleanupChunked[T](ctx, p.storage, p.maxGuideFileSize, spec, threshold)
	case LayoutPerEntity:
		return cleanupPerEntity[T](ctx, p.storage, prefix, threshold)
	}

	return nil, errUnknownLayout(categoryName, category.Layout)
}

// CleanupValidations removes blueprint validations older than thresholdDate
// and re-persists the remaining entries.
func (p *Persister) CleanupValidations(
	ctx context.Context,
	thresholdDate time.Time,
) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return cleanupCategory[*manage.BlueprintValidation](
		ctx, p, p.config.Validations, p.validationSpec(),
		p.keys.ValidationsPrefix(), thresholdDate, "validations",
	)
}

// CleanupReconciliationResults removes reconciliation results older than
// thresholdDate and re-persists the remaining entries.
func (p *Persister) CleanupReconciliationResults(
	ctx context.Context,
	thresholdDate time.Time,
) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return cleanupCategory[*manage.ReconciliationResult](
		ctx, p, p.config.ReconciliationResults, p.reconciliationSpec(),
		p.keys.ReconciliationResultsPrefix(), thresholdDate, "reconciliation results",
	)
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

// EventsLayout reports how the events category is stored, which is the one thing the
// events container cannot hide behind this persister, under LayoutPerEntity its
// in-memory partitions are a cache rather than the record, so reads have to come from
// storage.
func (p *Persister) EventsLayout() Layout {
	return p.config.Events.Layout
}

// CreateEventObject persists one event as its own object, under its channel's prefix.
func (p *Persister) CreateEventObject(ctx context.Context, event *manage.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	partitionName := partitionNameForChannel(event.ChannelType, event.ChannelID)

	return writeChunk(
		ctx,
		p.storage,
		p.keys.EventObject(partitionName, event.Timestamp, event.ID),
		event,
	)
}

// EventsInChannel returns every stored event for one channel, oldest first.
//
// Read from storage rather than from the in-memory partition, so that a channel another
// engine instance has been writing to comes back whole.
func (p *Persister) EventsInChannel(
	ctx context.Context,
	partitionName string,
) ([]*manage.Event, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keys, err := p.orderedEventKeys(ctx, p.keys.EventChannelPrefix(partitionName))
	if err != nil {
		return nil, err
	}

	events := make([]*manage.Event, 0, len(keys))
	for _, key := range keys {
		var event *manage.Event
		if err := readChunk(ctx, p.storage, key, &event); err != nil {
			return nil, err
		}

		// Left nil by a key removed between the listing and the read.
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// LastEventIDInChannel returns the ID of the most recent stored event for one channel,
// or an empty string when the channel has none.
func (p *Persister) LastEventIDInChannel(
	ctx context.Context,
	partitionName string,
) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keys, err := p.orderedEventKeys(ctx, p.keys.EventChannelPrefix(partitionName))
	if err != nil || len(keys) == 0 {
		return "", err
	}

	return entityIDFromKey(keys[len(keys)-1]), nil
}

// CleanupEventObjects removes the events older than thresholdDate across every channel,
// returning the IDs it removed.
//
// The timestamp is in the key, so no events are read, the listing says which objects are
// past their retention period.
func (p *Persister) CleanupEventObjects(
	ctx context.Context,
	thresholdDate time.Time,
) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keys, err := p.storage.List(ctx, p.keys.EventsPrefix())
	if err != nil {
		return nil, err
	}

	threshold := thresholdDate.Unix()
	removed := []string{}
	for _, key := range keys {
		timestamp, eventID, ok := ParseTimestampedKey(key)
		if !ok || timestamp > threshold {
			continue
		}

		if err := deleteIgnoreNotFound(ctx, p.storage, key); err != nil {
			return removed, err
		}
		removed = append(removed, eventID)
	}

	return removed, nil
}

// FindEventObject returns the stored event with the given ID, searching across every
// channel, and whether it was found.
//
// A key carries the event's own ID but not which channel it belongs to, so this costs a
// listing. Nothing in the deploy engine resolves an event by ID alone, a stream is
// always asked for by channel, so the cost falls on an operator's lookup rather than on
// anything on the request path.
func (p *Persister) FindEventObject(
	ctx context.Context,
	eventID string,
) (*manage.Event, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keys, err := p.storage.List(ctx, p.keys.EventsPrefix())
	if err != nil {
		return nil, false, err
	}

	for _, key := range keys {
		if entityIDFromKey(key) != eventID {
			continue
		}

		var event *manage.Event
		if err := readChunk(ctx, p.storage, key, &event); err != nil {
			return nil, false, err
		}

		return event, event != nil, nil
	}

	return nil, false, nil
}

// Keys that do not parse are skipped rather than failing the listing. The prefix is the
// engine's own, but a listing is not the place to refuse to proceed over one stray
// object.
func (p *Persister) orderedEventKeys(ctx context.Context, prefix string) ([]string, error) {
	keys, err := p.storage.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	ordered := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, _, ok := ParseTimestampedKey(key); ok {
			ordered = append(ordered, key)
		}
	}
	// Storage.List makes no promise about order, and the key's zero-padded timestamp
	// is what puts events back into the order they were produced in.
	slices.Sort(ordered)

	return ordered, nil
}

// LoadMode reports how State was populated, which is what decides whether what it holds
// for a category is the whole of it or a subset materialised on demand.
func (p *Persister) LoadMode() LoadMode {
	return p.config.Mode
}

// ParseTimestampedKey reads back a key written as
// "<zero-padded timestamp>-<entity id>.json", returning the timestamp, the entity ID and
// whether the key is one of that shape at all.
//
// The timestamp part has to be exactly the padded width. An identifier is commonly a
// UUID, whose first hyphen-separated segment is eight hex characters and is all digits
// often enough to be parsed as a number, so without the length check a key written
// before the timestamp was part of it would be misread as a timestamped one. Reporting
// false for such a key is what lets a caller recognise it and fall back to reading the
// entity for its timestamp.
func ParseTimestampedKey(key string) (int64, string, bool) {
	name := strings.TrimSuffix(path.Base(key), ".json")
	timestampPart, entityID, found := strings.Cut(name, "-")
	if !found || entityID == "" || len(timestampPart) != timestampKeyDigits {
		return 0, "", false
	}

	timestamp, err := strconv.ParseInt(timestampPart, 10, 64)
	if err != nil {
		return 0, "", false
	}

	return timestamp, entityID, true
}

func entityIDFromKey(key string) string {
	_, entityID, _ := ParseTimestampedKey(key)
	return entityID
}
