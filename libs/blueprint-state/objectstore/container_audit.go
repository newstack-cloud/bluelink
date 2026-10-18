package objectstore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
)

// auditContainer serves the audit trail from a per-entity object layout.
//
// Get, Save and Cleanup come from the embedded statestore container, which knows the
// per-entity layout.
// The statestore Query is overridden as object storage uses per-entity layout
// for audit records instead of the index and chunk approach in statestore containers
// to allow for more efficient queries and allow for independent writes across
// different instances of a deploy engine running against the same storage (e.g. parallel CI runs).
//
// The records are one object each, so the trail has to be enumerated under its key
// prefix.
//
// Per-entity is the right layout here even at the cost of that enumeration. A
// chunked audit category would have every engine instance appending to the same
// object on every mutating request, which is the heaviest write contention of
// any category in the store.
type auditContainer struct {
	*statestore.AuditContainer
	svc  Service
	keys statestore.KeyBuilder
}

func (c *auditContainer) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	records, err := c.listRecords(ctx)
	if err != nil {
		return nil, err
	}
	return statestore.NarrowAuditRecords(records, query), nil
}

// Records are fetched one object at a time, since a key carries only the record
// ID and so cannot be matched against a query's time bounds without being read.
// Audit reads are operational rather than on the request path, which is what
// makes that affordable; a trail large enough for it to hurt wants the
// timestamp in the key.
func (c *auditContainer) listRecords(ctx context.Context) ([]*manage.AuditRecord, error) {
	objects, err := c.svc.List(ctx, c.keys.AuditRecordsPrefix())
	if err != nil {
		return nil, err
	}

	records := make([]*manage.AuditRecord, 0, len(objects))
	for _, object := range objects {
		var record manage.AuditRecord
		found, err := readEntity(ctx, c.svc, object.Key, &record)
		if err != nil {
			return nil, err
		}
		if found {
			records = append(records, &record)
		}
	}
	return records, nil
}
