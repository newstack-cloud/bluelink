package statestore

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// UpdateInstanceEntities rewrites an existing instance whose change adds or removes
// resources or links. Under Config.WriteEntityRecords the records for added entities are
// written before the instance and those for removed entities are deleted after it, so an
// interruption at any point leaves only records whose instance does not hold their entity,
// which lookups reject, and never an entity without a record.
//
// The caller must hold the State write lock for the state cache updates.
func (p *Persister) UpdateInstanceEntities(
	ctx context.Context,
	instance *state.InstanceState,
	added InstanceEntityIDs,
	removed InstanceEntityIDs,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.updateInstanceEntities(ctx, instance, added, removed)
	if err != nil {
		p.state.forgetCachedInstances()
	}

	return err
}

func (p *Persister) updateInstanceEntities(
	ctx context.Context,
	instance *state.InstanceState,
	added InstanceEntityIDs,
	removed InstanceEntityIDs,
) error {
	if err := p.writeEntityRecords(ctx, instance.InstanceID, added); err != nil {
		return err
	}

	if err := p.rewriteInstanceRecord(ctx, instance); err != nil {
		return err
	}

	return p.deleteEntityRecords(ctx, removed)
}

// Records are only written for an instance whose object already exists, so the first write
// of an instance that holds resources or links is a shell without them. Reports whether the
// instance object was written at all, since only a failure before that should give the name
// reservation back.
func (p *Persister) createInstanceRecords(
	ctx context.Context,
	instance *state.InstanceState,
) (instanceWritten bool, err error) {
	entities := entityIDsOf(instance)
	if !p.config.WriteEntityRecords || entities.isEmpty() {
		return false, p.writeInstanceRecord(ctx, instance)
	}

	if err := p.writeInstanceRecord(ctx, instanceShell(instance)); err != nil {
		return false, err
	}

	if err := p.writeEntityRecords(ctx, instance.InstanceID, entities); err != nil {
		return true, err
	}

	return true, p.rewriteInstanceRecord(ctx, instance)
}

// The marker goes first and the instance object last, so every record is deleted while its
// instance still exists, and an interrupted removal leaves a marked instance that reads as
// not found and that removing again finishes.
func (p *Persister) removeInstanceWithRecords(ctx context.Context, instance *state.InstanceState) error {
	if p.config.Instances.Layout != LayoutPerEntity {
		return errEntityRecordsNeedPerEntityInstances
	}

	marked := NewPersistedInstanceState(instance)
	marked.Removing = true
	if err := writeChunk(
		ctx, p.storage, p.keys.Instance(instance.InstanceID), marked,
	); err != nil {
		return err
	}

	if err := p.deleteEntityRecords(ctx, entityIDsOf(instance)); err != nil {
		return err
	}

	if err := p.maybeRemoveNameRecord(ctx, instance.InstanceName); err != nil {
		return err
	}

	return p.deleteInstanceRecord(ctx, instance)
}

var errEntityRecordsNeedPerEntityInstances = errors.New(
	"statestore: WriteEntityRecords requires the per-entity layout for instances",
)

func (p *Persister) writeEntityRecords(ctx context.Context, instanceID string, ids InstanceEntityIDs) error {
	if !p.config.WriteEntityRecords {
		return nil
	}
	record := &EntityRecord{InstanceID: instanceID}
	for _, key := range p.entityRecordKeys(ids) {
		if err := writeChunk(ctx, p.storage, key, record); err != nil {
			return err
		}
	}
	return nil
}

// Deleted without reading them first, a resource or link ID belongs to the instance it was
// created in for its whole life, so a record for an entity this instance held cannot name
// another instance.
func (p *Persister) deleteEntityRecords(ctx context.Context, ids InstanceEntityIDs) error {
	if !p.config.WriteEntityRecords {
		return nil
	}

	for _, key := range p.entityRecordKeys(ids) {
		if err := deleteIgnoreNotFound(ctx, p.storage, key); err != nil {
			return err
		}
	}

	return nil
}

func (p *Persister) entityRecordKeys(ids InstanceEntityIDs) []string {
	keys := make([]string, 0, len(ids.Resources)+len(ids.Links))

	for _, resourceID := range ids.Resources {
		keys = append(keys, p.keys.ResourceRecord(resourceID))
	}

	for _, linkID := range ids.Links {
		keys = append(keys, p.keys.LinkRecord(linkID))
	}

	return keys
}

func instanceShell(instance *state.InstanceState) *state.InstanceState {
	shell := *instance
	shell.ResourceIDs = map[string]string{}
	shell.Resources = map[string]*state.ResourceState{}
	shell.Links = map[string]*state.LinkState{}
	return &shell
}

func (ids InstanceEntityIDs) isEmpty() bool {
	return len(ids.Resources) == 0 && len(ids.Links) == 0
}

func entityIDsOf(instance *state.InstanceState) InstanceEntityIDs {
	ids := InstanceEntityIDs{}

	for resourceID := range instance.Resources {
		ids.Resources = append(ids.Resources, resourceID)
	}

	for _, link := range instance.Links {
		ids.Links = append(ids.Links, link.LinkID)
	}

	return ids
}

func diffEntityIDs(previous, current InstanceEntityIDs) (added, removed InstanceEntityIDs) {
	added.Resources, removed.Resources = diffIDs(previous.Resources, current.Resources)
	added.Links, removed.Links = diffIDs(previous.Links, current.Links)
	return added, removed
}

func diffIDs(previous, current []string) (added, removed []string) {
	previousSet := make(map[string]bool, len(previous))
	for _, id := range previous {
		previousSet[id] = true
	}

	currentSet := make(map[string]bool, len(current))
	for _, id := range current {
		currentSet[id] = true
		if !previousSet[id] {
			added = append(added, id)
		}
	}

	for _, id := range previous {
		if !currentSet[id] {
			removed = append(removed, id)
		}
	}

	return added, removed
}
