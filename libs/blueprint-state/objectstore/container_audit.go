package objectstore

import (
	"context"
	"slices"

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
// Per-entity is the right layout here. A chunked audit category would have every engine
// instance appending to the same object on every mutating request, which is the heaviest
// write contention of any category in the store.
type auditContainer struct {
	*statestore.AuditContainer
	svc  Service
	keys statestore.KeyBuilder
}

// Query reads the trail, narrowing it by key before reading any record.
//
// A record's key leads with its timestamp, so the whole of a query's time range and
// cursor is answered from the key, only the records that could match are read, and
// reading stops once the query's limit is satisfied. What is left for the record itself
// is the action, the principal and the outcome.
func (c *auditContainer) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	keys, err := c.candidateKeys(ctx, query)
	if err != nil {
		return nil, err
	}

	// Lexicographic order is (timestamp, ID) order, since the timestamp is zero padded to
	// a fixed width, which is the order manage.Audit documents.
	slices.Sort(keys)

	limit := statestore.AuditQueryLimit(query)
	matched := make([]*manage.AuditRecord, 0, min(limit, len(keys)))
	for _, key := range keys {
		record, found, err := c.readRecord(ctx, key)
		if err != nil {
			return nil, err
		}
		if !found || !statestore.AuditRecordMatches(record, query) {
			continue
		}

		matched = append(matched, record)
		if len(matched) == limit {
			break
		}
	}

	return matched, nil
}

// The keys that could hold a matching record.
//
// A key that is not a record key is reported rather than skipped. Every object the engine
// writes under this prefix carries a timestamp, so one that does not means the trail is
// not what this code can read, and quietly leaving it out would answer a query with part
// of a trail as though it were the whole of it.
func (c *auditContainer) candidateKeys(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]string, error) {
	objects, err := c.svc.List(ctx, c.queryPrefix(query))
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		timestamp, recordID, ok := statestore.ParseTimestampedKey(object.Key)
		if !ok {
			return nil, statestore.NotAnAuditRecordKeyError(object.Key)
		}
		if keyInQueryRange(timestamp, recordID, query) {
			keys = append(keys, object.Key)
		}
	}

	return keys, nil
}

// Narrows the listing to the slice of the trail the query's bounds can touch.
//
// A bounded query is the common read, the audit endpoints page through a time range. The
// timestamp leads the key, so the digits the ends of that range agree on are a literal
// key prefix, and a query bounded to an hour lists an hour of the trail rather than all
// of it. See statestore.SharedTimestampKeyPrefix for why that cannot exclude a match.
//
// An unbounded query lists the whole trail where without an upper bound there is
// no shared prefix to narrow by.
func (c *auditContainer) queryPrefix(query *manage.AuditRecordQuery) string {
	prefix := c.keys.AuditRecordsPrefix()
	if query.To == 0 {
		return prefix
	}

	// The cursor is a lower bound too, so paging within a range narrows as the page
	// advances rather than listing from the start of the range each time.
	low := max(query.From, query.AfterTimestamp)
	// To is exclusive, so the newest key the query can match is the second before it.
	high := query.To - 1
	if high < low {
		return prefix
	}

	return prefix + statestore.SharedTimestampKeyPrefix(low, high)
}

// Whether a key's timestamp and ID clear the query's time range and cursor.
//
// These are exactly the parts of statestore.AuditRecordMatches that depend on the
// timestamp and the ID, which is what makes them answerable without a read. The bounds
// match it deliberately. From is inclusive, To is exclusive, and the cursor compares the
// ID only for a record sharing the cursor's second.
func keyInQueryRange(
	timestamp int64,
	recordID string,
	query *manage.AuditRecordQuery,
) bool {
	if query.From != 0 && timestamp < query.From {
		return false
	}
	if query.To != 0 && timestamp >= query.To {
		return false
	}

	if query.AfterTimestamp == 0 {
		return true
	}
	if timestamp != query.AfterTimestamp {
		return timestamp > query.AfterTimestamp
	}

	return recordID > query.AfterID
}

func (c *auditContainer) readRecord(
	ctx context.Context,
	key string,
) (*manage.AuditRecord, bool, error) {
	var record manage.AuditRecord
	found, err := readEntity(ctx, c.svc, key, &record)
	if err != nil || !found {
		return nil, false, err
	}

	return &record, true, nil
}
