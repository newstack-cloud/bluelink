package statestore

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// PersistedInstanceState is the on-disk / in-object-store JSON representation of
// a blueprint instance. It differs from state.InstanceState in that child
// blueprints are stored as ID references rather than being embedded — the
// loader re-wires the object graph on startup.
//
// This shape is part of statestore's public surface because backends that
// implement ETag-based atomic claims (objectstore) JSON-unmarshal directly
// into it when performing a compare-and-swap against a single instance object.
type PersistedInstanceState struct {
	InstanceID                 string                          `json:"id"`
	InstanceName               string                          `json:"name"`
	Status                     core.InstanceStatus             `json:"status"`
	LastStatusUpdateTimestamp  int                             `json:"lastStatusUpdateTimestamp,omitempty"`
	LastDeployedTimestamp      int                             `json:"lastDeployedTimestamp"`
	LastDeployAttemptTimestamp int                             `json:"lastDeployAttemptTimestamp"`
	ResourceIDs                map[string]string               `json:"resourceIds"`
	Resources                  map[string]*state.ResourceState `json:"resources"`
	Links                      map[string]*state.LinkState     `json:"links"`
	Metadata                   map[string]*core.MappingNode    `json:"metadata"`
	Exports                    map[string]*state.ExportState   `json:"exports"`
	// A mapping of child blueprint names to their blueprint instance IDs.
	ChildBlueprints   map[string]string                 `json:"childBlueprints"`
	ChildDependencies map[string]*state.DependencyInfo  `json:"childDependencies,omitempty"`
	Durations         *state.InstanceCompletionDuration `json:"durations,omitempty"`
	Version           int64                             `json:"version"`
	// Removing marks an instance whose removal has started under Config.WriteEntityRecords.
	// A marked instance reads as not found, and removing it again resumes the removal.
	Removing bool `json:"removing,omitempty"`
}

// EntityRecord is the JSON shape of a resource or link ID record written under
// Config.WriteEntityRecords. It is a hint where whoever follows it checks that the instance
// it names still holds the entity.
type EntityRecord struct {
	InstanceID string `json:"instanceId"`
}

// InstanceEntityIDs lists resource and link IDs held by an instance, or added or removed by a
// change to it, so their ID records can be written before and deleted after the instance.
type InstanceEntityIDs struct {
	Resources []string
	Links     []string
}

// EntityIDs returns the IDs of the resources and links the persisted instance holds, which
// are the ID records it needs under Config.WriteEntityRecords.
func (p *PersistedInstanceState) EntityIDs() InstanceEntityIDs {
	ids := InstanceEntityIDs{}
	for resourceID := range p.Resources {
		ids.Resources = append(ids.Resources, resourceID)
	}
	for _, link := range p.Links {
		ids.Links = append(ids.Links, link.LinkID)
	}
	return ids
}

// WithoutEntities returns a copy of the persisted instance without its resources and links.
// Under Config.WriteEntityRecords an instance is first written in this form, so that the
// instance object exists before any ID record names it.
func (p *PersistedInstanceState) WithoutEntities() *PersistedInstanceState {
	shell := *p
	shell.ResourceIDs = map[string]string{}
	shell.Resources = map[string]*state.ResourceState{}
	shell.Links = map[string]*state.LinkState{}
	return &shell
}

// NewPersistedInstanceState returns the persistence representation of an
// in-memory instance. Child blueprints are flattened to an ID-reference map —
// callers are responsible for persisting each child blueprint separately.
func NewPersistedInstanceState(instance *state.InstanceState) *PersistedInstanceState {
	childRefs := map[string]string{}
	for childName, child := range instance.ChildBlueprints {
		childRefs[childName] = child.InstanceID
	}

	return &PersistedInstanceState{
		InstanceID:                 instance.InstanceID,
		InstanceName:               instance.InstanceName,
		Status:                     instance.Status,
		LastStatusUpdateTimestamp:  instance.LastStatusUpdateTimestamp,
		LastDeployedTimestamp:      instance.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: instance.LastDeployAttemptTimestamp,
		ResourceIDs:                instance.ResourceIDs,
		Resources:                  instance.Resources,
		Links:                      instance.Links,
		Metadata:                   instance.Metadata,
		Exports:                    instance.Exports,
		ChildDependencies:          instance.ChildDependencies,
		ChildBlueprints:            childRefs,
		Durations:                  instance.Durations,
		Version:                    instance.Version,
	}
}

// InstanceNameRecord is the minimal JSON shape written to instances_by_name/
// when Config.WriteNameRecords is true. Gives a single read to resolve a
// name to an id under ModeLazy.
type InstanceNameRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ReservedAt is the Unix time the name was reserved, set by backends that reserve
	// a name before writing the instance. It lets a reservation whose instance was never
	// written, because the process died in between, be recognised as abandoned.
	ReservedAt int64 `json:"reservedAt,omitempty"`
}

// ToInstanceState returns a live state.InstanceState populated from the
// persisted shape. Child blueprints are left as an empty map — the loader
// re-wires the parent/child object graph after all instances are loaded.
func (p *PersistedInstanceState) ToInstanceState() *state.InstanceState {
	return &state.InstanceState{
		InstanceID:                 p.InstanceID,
		InstanceName:               p.InstanceName,
		Status:                     p.Status,
		LastStatusUpdateTimestamp:  p.LastStatusUpdateTimestamp,
		LastDeployedTimestamp:      p.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: p.LastDeployAttemptTimestamp,
		ResourceIDs:                p.ResourceIDs,
		Resources:                  p.Resources,
		Links:                      p.Links,
		Metadata:                   p.Metadata,
		Exports:                    p.Exports,
		ChildDependencies:          p.ChildDependencies,
		ChildBlueprints:            map[string]*state.InstanceState{},
		Durations:                  p.Durations,
		Version:                    p.Version,
	}
}
