package statestore

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// AuditContainer implements manage.Audit against a shared
// statestore.State and Persister.
type AuditContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewAuditContainer(st *State, persister *Persister, logger core.Logger) *AuditContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &AuditContainer{state: st, persister: persister, logger: logger}
}

func (c *AuditContainer) Get(
	ctx context.Context,
	id string,
) (*manage.AuditRecord, error) {
	record, ok, err := c.state.LookupAuditRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, manage.AuditRecordNotFoundError(id)
	}

	c.state.RLock()
	defer c.state.RUnlock()
	copied := copyAuditRecord(record)
	return &copied, nil
}

func (c *AuditContainer) Save(
	ctx context.Context,
	record *manage.AuditRecord,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	// A record ID is stored at most once, so an ID already held needs no write.
	if _, recorded := c.state.auditRecords[record.ID]; recorded {
		return nil
	}

	c.logger.Debug(
		"persisting audit record",
		core.StringLogField("auditRecordId", record.ID),
		core.StringLogField("auditAction", record.Action),
	)

	// Persisted before the in-memory write,
	// so a failed persist leaves nothing behind.
	if err := c.persister.CreateAuditRecord(ctx, record); err != nil {
		return err
	}
	c.state.auditRecords[record.ID] = record

	return nil
}

// Query reads the trail, from memory where memory holds the whole of it.
//
// Under ModeEager the trail was read into State at startup and every write since has gone
// through Save, so re-reading it from storage would be work for an answer already held in memory.
// Under any other mode State holds whatever has been materialised on demand, and
// answering from that would present part of a trail as the whole of it, which an audit trail must
// not do.
//
// Reading from memory also means a record another process wrote since startup is not
// seen. That is already true of every other category a ModeEager container serves, and
// that mode is for a state directory with a single writer.
func (c *AuditContainer) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	if c.persister.LoadMode() == ModeEager {
		return c.queryHeldRecords(query), nil
	}

	matched, err := c.persister.QueryAuditRecords(ctx, query)
	if err != nil {
		return nil, err
	}

	return copyAuditRecords(matched), nil
}

func (c *AuditContainer) queryHeldRecords(
	query *manage.AuditRecordQuery,
) []*manage.AuditRecord {
	c.state.RLock()
	defer c.state.RUnlock()

	held := make([]*manage.AuditRecord, 0, len(c.state.auditRecords))
	for _, record := range c.state.auditRecords {
		held = append(held, record)
	}

	// Narrowed rather than filtered here, so that what counts as a match, the order and
	// the limit are the same wherever the records came from.
	return copyAuditRecords(NarrowAuditRecords(held, query))
}

// Copied while the caller still holds the read lock, since the records are the ones State
// is holding rather than copies of them.
func copyAuditRecords(records []*manage.AuditRecord) []*manage.AuditRecord {
	copied := make([]*manage.AuditRecord, len(records))
	for i, record := range records {
		recordCopy := copyAuditRecord(record)
		copied[i] = &recordCopy
	}

	return copied
}

func (c *AuditContainer) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.state.Lock()
	defer c.state.Unlock()

	// The count is what storage removed, not the change in the size of the in-memory
	// lookup, which only holds the whole trail under ModeEager.
	removedIDs, err := c.persister.CleanupAuditRecords(ctx, thresholdDate)
	if err != nil {
		return 0, err
	}
	for _, id := range removedIDs {
		delete(c.state.auditRecords, id)
	}

	return int64(len(removedIDs)), nil
}

// Copied wholesale rather than field by field. Every field is a value type, so this
// stays complete when one is added.
// A reference-typed field means writing a real copy here.
func copyAuditRecord(r *manage.AuditRecord) manage.AuditRecord {
	return *r
}
