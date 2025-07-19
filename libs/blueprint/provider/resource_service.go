package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourceService is an interface for a service that provides a way for links
// to deploy link-managed resources, look up resources in the blueprint state
// and acquire locks when updating existing resources in the same blueprint
// as a part of the intermediary resources update phase.
//
// This is a subset of a resource registry that exposes only the methods
// that will typically be used by a link implementation.
type ResourceService interface {
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

	// Deploy deals with the deployment of a resource of a given type.
	// Callers can specify whether or not to wait for the resource to stabilise
	// before returning.
	Deploy(
		ctx context.Context,
		resourceType string,
		input *ResourceDeployServiceInput,
	) (*ResourceDeployOutput, error)

	// Destroy deals with the destruction of a resource of a given type.
	Destroy(
		ctx context.Context,
		resourceType string,
		input *ResourceDestroyInput,
	) error

	// AcquireResourceLock acquires a lock on a resource of a given type
	// in the blueprint state to ensure that no other operations
	// are modifying the resource at the same time.
	// This is useful for links that need to update existing resources
	// in the same blueprint as a part of the intermediary resources update phase.
	// The blueprint container will ensure that the lock is released after the
	// update intermediary resources phase is complete for the current link.
	// The lock will be released if the link update fails or a lock timeout occurs.
	AcquireResourceLock(
		ctx context.Context,
		input *AcquireResourceLockInput,
	) error
}

// ResourceLookupInput is the input for the lookup methods of the ResourceService.
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

// ResourceDeployServiceInput is the input for the Deploy method of the ResourceService
// that enhances the ResourceDeployInput with a flag to allow the caller
// to specify whether or not to wait for the resource to stabilise before returning.
type ResourceDeployServiceInput struct {
	// DeployInput is the input for the resource deployment that is passed into the `Deploy`
	// method of a `provider.Resource` implementation.
	DeployInput *ResourceDeployInput
	// WaitUntilStable specifies whether or not to
	// wait for the resource to stabilise before returning.
	WaitUntilStable bool
}

// AcquireResourceLockInput is the input for the AcquireResourceLock method of the ResourceService.
type AcquireResourceLockInput struct {
	// InstanceID is the ID of the blueprint instance to acquire the lock in.
	InstanceID string
	// ResourceName is the name of the resource as defined in the blueprint
	// to acquire the lock on.
	ResourceName string
	// AcquiredBy is the identifier of the caller that is acquiring the lock.
	// This is typically populated by the deployment orchestrator to identify
	// a link that has acquired the lock, this doesn't need to be set by the caller
	// in a provider link implementation.
	// This is helpful in releasing locks proactively instead of waiting for the
	// lock timeout to occur.
	AcquiredBy string
}
