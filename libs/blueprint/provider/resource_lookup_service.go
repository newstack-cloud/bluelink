package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourceLookupService is an interface for a service that looks up resources
// in a blueprint instance.
// This is a subset of a resource registry that enables looking up resources
// in limited contexts that don't need the full functionality of a resource registry.
type ResourceLookupService interface {
	// LookupResourceInState retrieves a resource of a given type
	// from the blueprint state.
	LookupResourceInState(
		ctx context.Context,
		input *ResourceLookupInput,
	) (*state.ResourceState, error)

	// HasResourceInState checks if a resource of a given type
	// exists in the blueprint state.
	HasResourceInState(
		ctx context.Context,
		input *ResourceLookupInput,
	) (bool, error)
}

// ResourceLookupInput is the input for the methods of the ResourceLookupService.
type ResourceLookupInput struct {
	// InstanceID is the ID of the blueprint instance to look up the resource in.
	InstanceID string
	// ResourceType is the type of the resource to look up.
	// For example, "aws/iam/role" or "gcloud/compute/instance".
	ResourceType string
	// ExternalID is the external identifier of the resource in the provider.
	// This is defined in a resource type spec definition as the `IDField`.
	ExternalID      string
	ProviderContext Context
}
