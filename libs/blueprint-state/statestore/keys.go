package statestore

import (
	"fmt"
	"path"
)

// KeyBuilder produces the canonical storage keys for each persisted category,
// prefixed by a storage-level root (memfile's state directory or objectstore's
// bucket-level prefix). Each method emits one fixed key format — layout
// dispatch (chunked vs per-entity) currently lives in Persister, not here.
//
// When a category grows scope-aware keys (e.g. ScopePerInstance prefixing
// resource chunks with "instances/<id>/"), those method variants belong on
// KeyBuilder so the scope decision lives in one place.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder returns a KeyBuilder with the given storage-level prefix.
// The prefix may be empty (memfile passes its state directory here;
// objectstore passes its bucket-level prefix).
func NewKeyBuilder(prefix string) KeyBuilder {
	return KeyBuilder{prefix: prefix}
}

// Instance returns the per-entity storage key for a single instance record.
func (k KeyBuilder) Instance(instanceID string) string {
	return k.join("instances", instanceID+".json")
}

// InstanceByName returns the storage key for an instance's name-lookup
// record — a tiny object containing {id, name} written alongside the main
// instance object under Config.WriteNameRecords. Enables O(1) name → id
// resolution under ModeLazy without enumerating the instances prefix.
func (k KeyBuilder) InstanceByName(name string) string {
	return k.join("instances_by_name", name+".json")
}

// InstanceChunk returns the chunked storage key for the instance chunk file
// at the given chunk number.
func (k KeyBuilder) InstanceChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("instances_c%d.json", chunkNumber))
}

// InstanceIndex returns the storage key for the instance chunk-index file.
func (k KeyBuilder) InstanceIndex() string {
	return k.join("instance_index.json")
}

// ResourceDrift returns the per-entity storage key for a resource-drift record.
func (k KeyBuilder) ResourceDrift(resourceID string) string {
	return k.join("resource_drift", resourceID+".json")
}

// ResourceDriftChunk returns the chunked storage key for a resource-drift
// chunk file at the given chunk number.
func (k KeyBuilder) ResourceDriftChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("resource_drift_c%d.json", chunkNumber))
}

// ResourceDriftIndex returns the storage key for the resource-drift chunk-index file.
func (k KeyBuilder) ResourceDriftIndex() string {
	return k.join("resource_drift_index.json")
}

// LinkDrift returns the per-entity storage key for a link-drift record.
func (k KeyBuilder) LinkDrift(linkID string) string {
	return k.join("link_drift", linkID+".json")
}

// LinkDriftChunk returns the chunked storage key for a link-drift chunk file
// at the given chunk number.
func (k KeyBuilder) LinkDriftChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("link_drift_c%d.json", chunkNumber))
}

// LinkDriftIndex returns the storage key for the link-drift chunk-index file.
func (k KeyBuilder) LinkDriftIndex() string {
	return k.join("link_drift_index.json")
}

// Changeset returns the per-entity storage key for a changeset.
func (k KeyBuilder) Changeset(changesetID string) string {
	return k.join("changesets", changesetID+".json")
}

// ChangesetChunk returns the chunked storage key for a changeset chunk file
// at the given chunk number.
func (k KeyBuilder) ChangesetChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("changesets_c%d.json", chunkNumber))
}

// ChangesetIndex returns the storage key for the changeset chunk-index file.
func (k KeyBuilder) ChangesetIndex() string {
	return k.join("changeset_index.json")
}

// Validation returns the per-entity storage key for a blueprint validation.
func (k KeyBuilder) Validation(validationID string) string {
	return k.join("validations", validationID+".json")
}

// ValidationChunk returns the chunked storage key for a blueprint-validation
// chunk file at the given chunk number.
func (k KeyBuilder) ValidationChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("blueprint_validations_c%d.json", chunkNumber))
}

// ValidationIndex returns the storage key for the blueprint-validation chunk-index file.
func (k KeyBuilder) ValidationIndex() string {
	return k.join("blueprint_validation_index.json")
}

// AuditRecord returns the per-entity storage key for an audit record.
func (k KeyBuilder) AuditRecord(recordID string) string {
	return k.join("audit_records", recordID+".json")
}

// AuditRecordsPrefix returns the storage key prefix every per-entity
// audit-record key sits under, for backends that enumerate the trail.
func (k KeyBuilder) AuditRecordsPrefix() string {
	return k.categoryPrefix("audit_records")
}

// ValidationsPrefix returns the storage key prefix every per-entity
// blueprint-validation key sits under.
func (k KeyBuilder) ValidationsPrefix() string {
	return k.categoryPrefix("validations")
}

// ChangesetsPrefix returns the storage key prefix every per-entity
// changeset key sits under.
func (k KeyBuilder) ChangesetsPrefix() string {
	return k.categoryPrefix("changesets")
}

// ReconciliationResultsPrefix returns the storage key prefix every per-entity
// reconciliation-result key sits under.
func (k KeyBuilder) ReconciliationResultsPrefix() string {
	return k.categoryPrefix("reconciliation_results")
}

// The directory each category's per-entity keys sit in, so a prefix cannot drift from
// the keys it is meant to enumerate.
//
// The trailing separator is formatted in rather than joined on, because path.Join strips
// one and it is what makes this a prefix rather than an object key.
func (k KeyBuilder) categoryPrefix(category string) string {
	return fmt.Sprintf("%s/", k.join(category))
}

// AuditRecordChunk returns the chunked storage key for an audit-record
// chunk file at the given chunk number.
func (k KeyBuilder) AuditRecordChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("audit_records_c%d.json", chunkNumber))
}

// AuditRecordIndex returns the storage key for the audit-record chunk-index file.
func (k KeyBuilder) AuditRecordIndex() string {
	return k.join("audit_record_index.json")
}

// ReconciliationResult returns the per-entity storage key for a reconciliation result.
func (k KeyBuilder) ReconciliationResult(resultID string) string {
	return k.join("reconciliation_results", resultID+".json")
}

// ReconciliationResultChunk returns the chunked storage key for a
// reconciliation-result chunk file at the given chunk number.
func (k KeyBuilder) ReconciliationResultChunk(chunkNumber int) string {
	return k.join(fmt.Sprintf("reconciliation_results_c%d.json", chunkNumber))
}

// ReconciliationResultIndex returns the storage key for the reconciliation-result chunk-index file.
func (k KeyBuilder) ReconciliationResultIndex() string {
	return k.join("reconciliation_result_index.json")
}

// EventPartition returns the storage key for a single channel's event
// partition file. Events are grouped into per-channel partitions rather
// than chunked globally, so the partition name (e.g. "changesets_<id>")
// fully identifies the file.
func (k KeyBuilder) EventPartition(partitionName string) string {
	return k.join(fmt.Sprintf("events__%s.json", partitionName))
}

// EventObject returns the per-entity storage key for a single event, under the
// prefix for its channel.
//
// The timestamp is zero-padded so that keys sort lexicographically in the order events
// were produced. A channel can then be read, resumed and cleaned up from a listing
// alone, without opening a single event, and no two writers share a file to overwrite.
func (k KeyBuilder) EventObject(partitionName string, timestamp int64, eventID string) string {
	return k.join("events", partitionName, fmt.Sprintf("%011d-%s.json", timestamp, eventID))
}

// EventChannelPrefix returns the storage key prefix every per-entity event key for one
// channel sits under.
func (k KeyBuilder) EventChannelPrefix(partitionName string) string {
	return fmt.Sprintf("%s/", k.join("events", partitionName))
}

// EventsPrefix returns the storage key prefix every per-entity event key sits under,
// across all channels.
func (k KeyBuilder) EventsPrefix() string {
	return k.categoryPrefix("events")
}

// EventIndex returns the storage key for the global event index, which
// maps event IDs to (partition, indexInPartition).
func (k KeyBuilder) EventIndex() string {
	return k.join("event_index.json")
}

// CleanupOperation returns the per-entity storage key for a cleanup operation.
func (k KeyBuilder) CleanupOperation(operationID string) string {
	return k.join("cleanup_operations", operationID+".json")
}

// EncryptionSalt returns the storage key for the salt a passphrase-derived
// key-encryption key is built from.
//
// Stored alongside state rather than beside the engine's configuration because
// the salt belongs to the data: the same passphrase with a different salt derives
// a different key and cannot read anything previously written. Moving or backing
// up a state directory therefore carries the salt with it.
//
// Not secret, and written unencrypted, it is an input to deriving the key, not
// key material. The eager loader skips keys it does not recognise, so this sits in
// the store without being mistaken for state.
func (k KeyBuilder) EncryptionSalt() string {
	return k.join("encryption_salt.json")
}

// Prefix returns the storage-level root every key is built under, for a caller that walks or
// lists the whole store rather than addressing one object.
func (k KeyBuilder) Prefix() string {
	return k.prefix
}

// Prepends the prefix to the given parts. The leading "/" of an
// absolute prefix is preserved so memfile state directories like
// "/var/lib/bluelink/state" survive intact.
func (k KeyBuilder) join(parts ...string) string {
	all := make([]string, 0, len(parts)+1)
	if k.prefix != "" {
		all = append(all, k.prefix)
	}
	all = append(all, parts...)
	return path.Join(all...)
}
