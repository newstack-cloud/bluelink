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
