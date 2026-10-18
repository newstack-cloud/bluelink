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

func (c *AuditContainer) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	matched, err := c.persister.QueryAuditRecords(ctx, query)
	if err != nil {
		return nil, err
	}

	records := make([]*manage.AuditRecord, len(matched))
	for i, record := range matched {
		copied := copyAuditRecord(record)
		records[i] = &copied
	}
	return records, nil
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
