package statestore

import (
	"fmt"
	"path"
	"strings"
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

// EventIndex returns the storage key for the global event index, which
// maps event IDs to (partition, indexInPartition).
func (k KeyBuilder) EventIndex() string {
	return k.join("event_index.json")
}

// CleanupOperation returns the per-entity storage key for a cleanup operation.
func (k KeyBuilder) CleanupOperation(operationID string) string {
	return k.join("cleanup_operations", operationID+".json")
}

// join prepends the prefix and normalises leading slashes so empty prefixes
// don't produce absolute-looking object keys.
func (k KeyBuilder) join(parts ...string) string {
	all := make([]string, 0, len(parts)+1)
	if k.prefix != "" {
		all = append(all, k.prefix)
	}
	all = append(all, parts...)
	return strings.TrimPrefix(path.Join(all...), "/")
}
