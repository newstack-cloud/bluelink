package statestore

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"regexp"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Walks every object under a given prefix in storage, classifies each key
// by its file-name pattern, unmarshals it, and populates the matching map
// on st. Missing storage (empty directory / empty bucket) is not an error —
// st is left with its initialised-but-empty maps.
//
// After all categories are loaded, the parent/child blueprint pointer graph
// is re-wired using ChildBlueprints ID references captured during instance
// loading, and the name-lookup cache is rebuilt.
func loadEager(ctx context.Context, st *State, storage Storage, prefix string) error {
	keys, err := storage.List(ctx, prefix)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	parentChildMapping := map[string][]childInstanceRef{}
	for _, key := range keys {
		filename := keyFilename(prefix, key)
		if filename == "" {
			continue
		}
		if err := loadKey(ctx, st, storage, key, filename, parentChildMapping); err != nil {
			return err
		}
	}

	rewireParentChildInstances(st, parentChildMapping)
	st.RebuildNameLookup()
	return nil
}

// Strips the prefix and any leading slash so the returned value
// is the category-local filename ready for pattern matching. Returns "" for
// keys outside the prefix (defensive; shouldn't happen with a well-behaved
// Storage.List implementation).
func keyFilename(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if !strings.HasPrefix(key, prefix) {
		return ""
	}
	rel := strings.TrimPrefix(key, prefix)
	return strings.TrimPrefix(rel, "/")
}

type childInstanceRef struct {
	childName       string
	childInstanceID string
}

var (
	instancesFilePattern             = regexp.MustCompile(`^instances_c(\d+)\.json$`)
	resourceDriftFilePattern         = regexp.MustCompile(`^resource_drift_c(\d+)\.json$`)
	linkDriftFilePattern             = regexp.MustCompile(`^link_drift_c(\d+)\.json$`)
	eventPartitionFilePattern        = regexp.MustCompile(`^events__(.*?)\.json$`)
	changesetsFilePattern            = regexp.MustCompile(`^changesets_c(\d+)\.json$`)
	blueprintValidationsFilePattern  = regexp.MustCompile(`^blueprint_validations_c(\d+)\.json$`)
	reconciliationResultsFilePattern = regexp.MustCompile(`^reconciliation_results_c(\d+)\.json$`)
)

func loadKey(
	ctx context.Context,
	st *State,
	storage Storage,
	key, filename string,
	parentChildMapping map[string][]childInstanceRef,
) error {
	switch {
	case instancesFilePattern.MatchString(filename):
		return loadInstanceChunk(ctx, st, storage, key, parentChildMapping)
	case filename == "instance_index.json":
		return loadIndex(ctx, storage, key, &st.instanceIndex)
	case resourceDriftFilePattern.MatchString(filename):
		return loadResourceDriftChunk(ctx, st, storage, key)
	case filename == "resource_drift_index.json":
		return loadIndex(ctx, storage, key, &st.resourceDriftIndex)
	case linkDriftFilePattern.MatchString(filename):
		return loadLinkDriftChunk(ctx, st, storage, key)
	case filename == "link_drift_index.json":
		return loadIndex(ctx, storage, key, &st.linkDriftIndex)
	case eventPartitionFilePattern.MatchString(filename):
		return loadEventPartition(ctx, st, storage, key, filename)
	case filename == "event_index.json":
		return loadEventIndex(ctx, storage, key, &st.eventIndex)
	case changesetsFilePattern.MatchString(filename):
		return loadChangesetChunk(ctx, st, storage, key)
	case filename == "changeset_index.json":
		return loadIndex(ctx, storage, key, &st.changesetIndex)
	case blueprintValidationsFilePattern.MatchString(filename):
		return loadValidationChunk(ctx, st, storage, key)
	case filename == "blueprint_validation_index.json":
		return loadIndex(ctx, storage, key, &st.validationIndex)
	case reconciliationResultsFilePattern.MatchString(filename):
		return loadReconciliationChunk(ctx, st, storage, key)
	case filename == "reconciliation_result_index.json":
		return loadIndex(ctx, storage, key, &st.reconciliationIndex)
	case strings.HasPrefix(filename, "cleanup_operations/") && strings.HasSuffix(filename, ".json"):
		return loadCleanupOperation(ctx, st, storage, key)
	case strings.HasPrefix(filename, "instances/") && strings.HasSuffix(filename, ".json"):
		return loadInstancePerEntity(ctx, st, storage, key, parentChildMapping)
	case strings.HasPrefix(filename, "instances_by_name/"):
		// Skip — name records are rebuilt from the instances map after load.
		return nil
	}
	return nil
}

func loadInstancePerEntity(
	ctx context.Context,
	st *State,
	storage Storage,
	key string,
	parentChildMapping map[string][]childInstanceRef,
) error {
	var p PersistedInstanceState
	if err := readJSON(ctx, storage, key, &p); err != nil {
		return err
	}
	if p.InstanceID == "" {
		return nil
	}

	st.instances[p.InstanceID] = p.ToInstanceState()
	maps.Copy(st.resources, p.Resources)

	for _, link := range p.Links {
		st.links[link.LinkID] = link
	}

	if len(p.ChildBlueprints) > 0 {
		refs := make([]childInstanceRef, 0, len(p.ChildBlueprints))
		for childName, childID := range p.ChildBlueprints {
			refs = append(
				refs,
				childInstanceRef{
					childName:       childName,
					childInstanceID: childID,
				},
			)
		}
		parentChildMapping[p.InstanceID] = refs
	}

	return nil
}

func loadInstanceChunk(
	ctx context.Context,
	st *State,
	storage Storage,
	key string,
	parentChildMapping map[string][]childInstanceRef,
) error {
	var persisted []*PersistedInstanceState

	if err := readJSON(ctx, storage, key, &persisted); err != nil {
		return err
	}

	for _, p := range persisted {
		st.instances[p.InstanceID] = p.ToInstanceState()
		maps.Copy(st.resources, p.Resources)

		for _, link := range p.Links {
			st.links[link.LinkID] = link
		}

		if len(p.ChildBlueprints) > 0 {
			refs := buildInstanceChunkRefs(p)
			parentChildMapping[p.InstanceID] = refs
		}
	}

	return nil
}

func buildInstanceChunkRefs(p *PersistedInstanceState) []childInstanceRef {
	refs := make([]childInstanceRef, 0, len(p.ChildBlueprints))
	for childName, childID := range p.ChildBlueprints {
		refs = append(
			refs,
			childInstanceRef{
				childName:       childName,
				childInstanceID: childID,
			},
		)
	}

	return refs
}

func loadResourceDriftChunk(ctx context.Context, st *State, storage Storage, key string) error {
	var entries []*state.ResourceDriftState
	if err := readJSON(ctx, storage, key, &entries); err != nil {
		return err
	}
	for _, d := range entries {
		st.resourceDrift[d.ResourceID] = d
	}
	return nil
}

func loadLinkDriftChunk(ctx context.Context, st *State, storage Storage, key string) error {
	var entries []*state.LinkDriftState
	if err := readJSON(ctx, storage, key, &entries); err != nil {
		return err
	}
	for _, d := range entries {
		st.linkDrift[d.LinkID] = d
	}
	return nil
}

func loadEventPartition(
	ctx context.Context,
	st *State,
	storage Storage,
	key, filename string,
) error {
	var events []*manage.Event
	if err := readJSON(ctx, storage, key, &events); err != nil {
		return err
	}
	for _, e := range events {
		st.events[e.ID] = e
	}
	matches := eventPartitionFilePattern.FindStringSubmatch(filename)
	if len(matches) > 1 {
		st.partitionEvents[matches[1]] = events
	}
	return nil
}

func loadChangesetChunk(ctx context.Context, st *State, storage Storage, key string) error {
	var entries []*manage.Changeset
	if err := readJSON(ctx, storage, key, &entries); err != nil {
		return err
	}
	for _, cs := range entries {
		st.changesets[cs.ID] = cs
	}
	return nil
}

func loadValidationChunk(ctx context.Context, st *State, storage Storage, key string) error {
	var entries []*manage.BlueprintValidation
	if err := readJSON(ctx, storage, key, &entries); err != nil {
		return err
	}
	for _, v := range entries {
		st.validations[v.ID] = v
	}
	return nil
}

func loadReconciliationChunk(ctx context.Context, st *State, storage Storage, key string) error {
	var entries []*manage.ReconciliationResult
	if err := readJSON(ctx, storage, key, &entries); err != nil {
		return err
	}
	for _, r := range entries {
		st.reconciliations[r.ID] = r
	}
	return nil
}

func loadCleanupOperation(ctx context.Context, st *State, storage Storage, key string) error {
	var op manage.CleanupOperation
	if err := readJSON(ctx, storage, key, &op); err != nil {
		return err
	}
	st.cleanupOps[op.ID] = &op
	return nil
}

func loadEventIndex(ctx context.Context, storage Storage, key string, dst *map[string]*EventIndexLocation) error {
	index := map[string]*EventIndexLocation{}
	if err := readJSON(ctx, storage, key, &index); err != nil {
		return err
	}
	*dst = index
	return nil
}

func loadIndex(ctx context.Context, storage Storage, key string, dst *map[string]*IndexLocation) error {
	index := map[string]*IndexLocation{}
	if err := readJSON(ctx, storage, key, &index); err != nil {
		return err
	}
	*dst = index
	return nil
}

// Reads the object at key and unmarshals into dst.
// Missing objects are treated as empty (dst left untouched); other read errors surface.
func readJSON(ctx context.Context, storage Storage, key string, dst any) error {
	data, err := storage.Read(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dst)
}

// Fills in each parent instance's ChildBlueprints
// map with pointers to the fully-loaded child instances, using the ID
// references captured during chunk load. Dangling references (child listed
// in a parent but not loaded) are silently skipped.
func rewireParentChildInstances(st *State, parentChildMapping map[string][]childInstanceRef) {
	for parentID, refs := range parentChildMapping {
		parent, ok := st.instances[parentID]
		if !ok {
			continue
		}
		if parent.ChildBlueprints == nil {
			parent.ChildBlueprints = map[string]*state.InstanceState{}
		}
		for _, ref := range refs {
			if child, ok := st.instances[ref.childInstanceID]; ok {
				parent.ChildBlueprints[ref.childName] = child
			}
		}
	}
}
