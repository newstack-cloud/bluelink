// Package stateutil provides shared utilities for fetching and working with
// blueprint instance state across TUI components.
package stateutil

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
)

// FetchInstanceState attempts to fetch the instance state for an existing deployment.
// Returns nil if the instance doesn't exist or if there's an error (new deployment case).
func FetchInstanceState(eng engine.DeployEngine, instanceID, instanceName string) *state.InstanceState {
	identifier := instanceID
	if identifier == "" {
		identifier = instanceName
	}
	if identifier == "" {
		return nil
	}

	instanceState, err := eng.GetBlueprintInstance(context.TODO(), identifier)
	if err != nil {
		return nil
	}

	return instanceState
}

// FindResourceState finds a resource state by logical name.
// It uses the instance state's ResourceIDs map to look up the resource ID,
// then retrieves the state from Resources.
func FindResourceState(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil ||
		instanceState.ResourceIDs == nil ||
		instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// FindLinkState finds a link state by logical name.
func FindLinkState(instanceState *state.InstanceState, name string) *state.LinkState {
	if instanceState == nil || instanceState.Links == nil {
		return nil
	}
	return instanceState.Links[name]
}

// FindChildInstanceState finds the instance state for a child blueprint by name.
func FindChildInstanceState(instanceState *state.InstanceState, childName string) *state.InstanceState {
	if instanceState == nil || instanceState.ChildBlueprints == nil {
		return nil
	}
	return instanceState.ChildBlueprints[childName]
}
