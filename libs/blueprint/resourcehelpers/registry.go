package resourcehelpers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Registry provides a way to retrieve resource plugins
// across multiple providers and transformers for tasks such as resource spec validation.
type Registry interface {
	// GetSpecDefinition returns the definition of a resource spec
	// in the registry.
	GetSpecDefinition(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceGetSpecDefinitionInput,
	) (*provider.ResourceGetSpecDefinitionOutput, error)

	// GetTypeDescription returns the description of a resource type
	// in the registry.
	GetTypeDescription(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceGetTypeDescriptionInput,
	) (*provider.ResourceGetTypeDescriptionOutput, error)

	// HasResourceType checks if a resource type is available in the registry.
	HasResourceType(ctx context.Context, resourceType string) (bool, error)

	// ListResourceTypes returns a list of all resource types available in the registry.
	ListResourceTypes(ctx context.Context) ([]string, error)

	// CustomValidate allows for custom validation of a resource of a given type.
	CustomValidate(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceValidateInput,
	) (*provider.ResourceValidateOutput, error)

	// Deploy deals with the deployment of a resource of a given type.
	// The caller can specify whether or not to wait until the resource is considered
	// stable.
	Deploy(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceDeployServiceInput,
	) (*provider.ResourceDeployOutput, error)

	// Destroy deals with the destruction of a resource of a given type.
	Destroy(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceDestroyInput,
	) error

	// StabilisedDependencies lists the resource types that are required to be stable
	// when a resource that is a dependency of the given resource type is being deployed.
	GetStabilisedDependencies(
		ctx context.Context,
		resourceType string,
		input *provider.ResourceStabilisedDependenciesInput,
	) (*provider.ResourceStabilisedDependenciesOutput, error)

	// LookupResourceInState retrieves a resource of a given type
	// from the blueprint state.
	LookupResourceInState(
		ctx context.Context,
		input *provider.ResourceLookupInput,
	) (*state.ResourceState, error)

	// HasResourceInState checks if a resource of a given type
	// exists in the blueprint state.
	HasResourceInState(
		ctx context.Context,
		input *provider.ResourceLookupInput,
	) (bool, error)

	// AcquireResourceLock acquires a lock on a resource
	// in the blueprint state to ensure that no other operations
	// are modifying the resource at the same time.
	// This is useful for links that need to update existing resources
	// in the same blueprint as a part of the intermediary resources update phase.
	// The blueprint container will ensure that the lock is released after the
	// update intermediary resources phase is complete for the current link.
	// The lock will be released if the link update fails or by the resource registry
	// if a lock timeout occurs.
	AcquireResourceLock(
		ctx context.Context,
		input *provider.AcquireResourceLockInput,
	) error

	// ReleaseResourceLock releases a lock on a resource of a given type
	// in the blueprint state.
	// This is to be used by the deployment orchestrator to release the lock
	// after the link update phase is complete or the link update fails.
	ReleaseResourceLock(
		ctx context.Context,
		instanceID string,
		resourceName string,
	)

	// ReleaseResourceLocks releases all resource locks
	// that have been acquired for the given instance ID.
	ReleaseResourceLocks(ctx context.Context, instanceID string)

	// ReleaseResourceLocksAcquiredBy releases all resource locks
	// that have been acquired by a specific caller (e.g. a link).
	// This is useful for releasing locks proactively instead of waiting
	// for the lock timeout to occur.
	ReleaseResourceLocksAcquiredBy(ctx context.Context, instanceID string, acquiredBy string)

	// WithParams creates a new registry derived from the current registry
	// with the given parameters.
	WithParams(
		params core.BlueprintParams,
	) Registry

	// ListTransformers returns a list of all transformer names available in the registry.
	ListTransformers(ctx context.Context) ([]string, error)
}

type resourceLock struct {
	// The ID of the blueprint instance that the lock is acquired in.
	instanceID string
	// The name of the resource that the lock is acquired on.
	resourceName string
	// The time when the lock was acquired.
	// This is used to determine if the lock has timed out.
	lockTime time.Time
	// Optional field to track who acquired the lock.
	acquiredBy string
}

const (
	// DefaultResourceLockTimeout is the default timeout for acquiring a resource lock.
	DefaultResourceLockTimeout = 3 * time.Minute
	// DefaultResourceLockCheckInterval is the default interval at which the resource lock
	// will be checked for availability when acquiring a lock.
	DefaultResourceLockCheckInterval = 100 * time.Millisecond
)

type registryFromProviders struct {
	providers                    map[string]provider.Provider
	transformers                 map[string]transform.SpecTransformer
	resourceCache                *core.Cache[provider.Resource]
	abstractResourceCache        *core.Cache[transform.AbstractResource]
	resourceTypes                []string
	params                       core.BlueprintParams
	stabilisationPollingInterval time.Duration
	stateContainer               state.Container
	resourceLocks                map[string]*resourceLock
	resourceLockTimeout          time.Duration
	resourceLockCheckInterval    time.Duration
	clock                        core.Clock
	mu                           *sync.Mutex
	// Use a separate mutex for resource locks to avoid contention
	// by blocking unrelated operations.
	resourceLocksMu *sync.Mutex
}

// RegistryOption is a function that modifies the registryFromProviders
// to allow for additional configuration options when creating a new registry.
type RegistryOption func(*registryFromProviders)

// WithResourceLockTimeout sets the timeout for acquiring a resource lock.
// If not provided, the default timeout is 180 seconds (3 minutes).
func WithResourceLockTimeout(timeout time.Duration) RegistryOption {
	return func(r *registryFromProviders) {
		r.resourceLockTimeout = timeout
	}
}

// WithResourceLockCheckInterval sets the interval at which the resource lock
// will be checked for availability when acquiring a lock.
// If not provided, the default interval is 100 milliseconds.
func WithResourceLockCheckInterval(interval time.Duration) RegistryOption {
	return func(r *registryFromProviders) {
		r.resourceLockCheckInterval = interval
	}
}

// WithClock sets the clock to be used by the registry.
func WithClock(clock core.Clock) RegistryOption {
	return func(r *registryFromProviders) {
		r.clock = clock
	}
}

// NewRegistry creates a new resource registry from a map of providers,
// matching against providers based on the resource type prefix.
func NewRegistry(
	providers map[string]provider.Provider,
	transformers map[string]transform.SpecTransformer,
	stabilisationPollingInterval time.Duration,
	stateContainer state.Container,
	params core.BlueprintParams,
	opts ...RegistryOption,
) Registry {
	registry := &registryFromProviders{
		providers:                    providers,
		transformers:                 transformers,
		stabilisationPollingInterval: stabilisationPollingInterval,
		stateContainer:               stateContainer,
		params:                       params,
		resourceCache:                core.NewCache[provider.Resource](),
		abstractResourceCache:        core.NewCache[transform.AbstractResource](),
		resourceTypes:                []string{},
		resourceLocks:                map[string]*resourceLock{},
		resourceLockTimeout:          DefaultResourceLockTimeout,
		resourceLockCheckInterval:    DefaultResourceLockCheckInterval,
		clock:                        core.SystemClock{},
		mu:                           &sync.Mutex{},
		resourceLocksMu:              &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(registry)
	}

	return registry
}

func (r *registryFromProviders) GetSpecDefinition(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		abstractResourceImpl, abstractErr := r.getAbstractResourceType(ctx, resourceType)
		if abstractErr != nil {
			return nil, errMultipleRunErrors([]error{err, abstractErr})
		}

		transformerNamespace := transform.ExtractTransformerFromItemType(resourceType)
		output, err := abstractResourceImpl.GetSpecDefinition(
			ctx,
			&transform.AbstractResourceGetSpecDefinitionInput{
				TransformerContext: transform.NewTransformerContextFromParams(
					transformerNamespace,
					r.params,
				),
			},
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetSpecDefinitionOutput{
			SpecDefinition: output.SpecDefinition,
		}, nil
	}

	return resourceImpl.GetSpecDefinition(ctx, input)
}

func (r *registryFromProviders) GetTypeDescription(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		abstractResourceImpl, abstractErr := r.getAbstractResourceType(ctx, resourceType)
		if abstractErr != nil {
			return nil, errMultipleRunErrors([]error{err, abstractErr})
		}

		transformerNamespace := transform.ExtractTransformerFromItemType(resourceType)
		output, err := abstractResourceImpl.GetTypeDescription(
			ctx,
			&transform.AbstractResourceGetTypeDescriptionInput{
				TransformerContext: transform.NewTransformerContextFromParams(
					transformerNamespace,
					r.params,
				),
			},
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetTypeDescriptionOutput{
			MarkdownDescription:  output.MarkdownDescription,
			PlainTextDescription: output.PlainTextDescription,
		}, nil
	}

	return resourceImpl.GetTypeDescription(ctx, input)
}

func (r *registryFromProviders) ListResourceTypes(ctx context.Context) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.resourceTypes) > 0 {
		return r.resourceTypes, nil
	}

	resourceTypes := []string{}
	for _, provider := range r.providers {
		types, err := provider.ListResourceTypes(ctx)
		if err != nil {
			return nil, err
		}

		resourceTypes = append(resourceTypes, types...)
	}

	for _, transformer := range r.transformers {
		abstractResourceTypes, err := transformer.ListAbstractResourceTypes(ctx)
		if err != nil {
			return nil, err
		}

		resourceTypes = append(resourceTypes, abstractResourceTypes...)
	}

	r.resourceTypes = resourceTypes

	return resourceTypes, nil
}

func (r *registryFromProviders) HasResourceType(ctx context.Context, resourceType string) (bool, error) {
	hasResourceType, err := r.hasProviderResourceType(ctx, resourceType)
	if err != nil {
		return false, err
	}

	hasAbstractResourceType, err := r.hasAbstractResourceType(ctx, resourceType)
	if err != nil {
		return false, err
	}

	return hasResourceType || hasAbstractResourceType, nil
}

func (r *registryFromProviders) hasProviderResourceType(ctx context.Context, resourceType string) (bool, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		if runErr, isRunErr := err.(*errors.RunError); isRunErr {
			if runErr.ReasonCode == ErrorReasonCodeProviderResourceTypeNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return resourceImpl != nil, nil
}

func (r *registryFromProviders) hasAbstractResourceType(ctx context.Context, resourceType string) (bool, error) {
	abstractResourceImpl, err := r.getAbstractResourceType(ctx, resourceType)
	if err != nil {
		if runErr, isRunErr := err.(*errors.RunError); isRunErr {
			if runErr.ReasonCode == ErrorReasonCodeAbstractResourceTypeNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return abstractResourceImpl != nil, nil
}

func (r *registryFromProviders) CustomValidate(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		abstractResourceImpl, abstractErr := r.getAbstractResourceType(ctx, resourceType)
		if abstractErr != nil {
			return nil, errMultipleRunErrors([]error{err, abstractErr})
		}

		transformerNamespace := transform.ExtractTransformerFromItemType(resourceType)
		output, err := abstractResourceImpl.CustomValidate(ctx, &transform.AbstractResourceValidateInput{
			SchemaResource: input.SchemaResource,
			TransformerContext: transform.NewTransformerContextFromParams(
				transformerNamespace,
				r.params,
			),
		})
		if err != nil {
			return nil, err
		}
		return &provider.ResourceValidateOutput{
			Diagnostics: output.Diagnostics,
		}, nil
	}

	return resourceImpl.CustomValidate(ctx, input)
}

func (r *registryFromProviders) Deploy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDeployServiceInput,
) (*provider.ResourceDeployOutput, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		return nil, err
	}

	output, err := resourceImpl.Deploy(ctx, input.DeployInput)
	if err != nil {
		return nil, err
	}

	if input.WaitUntilStable {
		err = r.waitForStabilisedDependencies(
			ctx,
			resourceImpl,
			input.DeployInput,
			output,
		)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

func (r *registryFromProviders) waitForStabilisedDependencies(
	ctx context.Context,
	resourceImpl provider.Resource,
	deployInput *provider.ResourceDeployInput,
	deployOutput *provider.ResourceDeployOutput,
) error {
	resolvedResource := getResolvedResourceFromChanges(deployInput.Changes)
	resourceName := getResourceNameFromChanges(deployInput.Changes)
	expectedComputedFields := getComputedFieldsFromChanges(deployInput.Changes)
	resourceSpec, err := specmerge.MergeResourceSpec(
		resolvedResource,
		resourceName,
		deployOutput.ComputedFieldValues,
		expectedComputedFields,
	)
	if err != nil {
		return err
	}

	// The provided context must have a timeout set by the caller,
	// unlike with the resource deployer in the container package,
	// the resource registry is not configured with a polling timeout
	// so without a deadline set on the context, the polling will continue indefinitely.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.stabilisationPollingInterval):
			output, err := resourceImpl.HasStabilised(
				ctx,
				&provider.ResourceHasStabilisedInput{
					InstanceID:   deployInput.InstanceID,
					ResourceID:   deployInput.ResourceID,
					ResourceSpec: resourceSpec,
					// Use the resolved resource and not the current resource state,
					// as the new metadata is what is relevant for the stabilisation check.
					ResourceMetadata: metadataStateFromResolvedResource(resolvedResource),
					ProviderContext:  deployInput.ProviderContext,
				},
			)
			if err != nil {
				return err
			}
			if output.Stabilised {
				return nil
			}
		}
	}
}

func (r *registryFromProviders) Destroy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDestroyInput,
) error {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		return err
	}

	return resourceImpl.Destroy(ctx, input)
}

func (r *registryFromProviders) GetStabilisedDependencies(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	resourceImpl, err := r.getResourceType(ctx, resourceType)
	if err != nil {
		return nil, err
	}

	return resourceImpl.GetStabilisedDependencies(ctx, input)
}

func (r *registryFromProviders) LookupResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (*state.ResourceState, error) {
	resourceImpl, err := r.getResourceType(ctx, input.ResourceType)
	if err != nil {
		return nil, err
	}

	definition, err := resourceImpl.GetSpecDefinition(
		ctx,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: input.ProviderContext,
		},
	)
	if err != nil {
		return nil, err
	}

	if definition == nil || definition.SpecDefinition == nil {
		return nil, errEmptyResourceSpecDefinition(input.ResourceType)
	}

	idField := definition.SpecDefinition.IDField
	instance, err := r.stateContainer.Instances().Get(ctx, input.InstanceID)
	if err != nil {
		return nil, err
	}

	return extractResourceByExternalID(
		idField,
		input.ExternalID,
		input.ResourceType,
		&instance,
	), nil
}

func extractResourceByExternalID(
	idField string,
	externalID string,
	resourceType string,
	instance *state.InstanceState,
) *state.ResourceState {
	if instance == nil {
		return nil
	}

	for _, resource := range instance.Resources {
		fieldPath := substitutions.RenderFieldPath("$", idField)
		idFieldValue, _ := core.GetPathValue(
			fieldPath,
			resource.SpecData,
			core.MappingNodeMaxTraverseDepth,
		)
		if idFieldValue != nil &&
			core.StringValue(idFieldValue) == externalID &&
			resource.Type == resourceType {
			return resource
		}
	}

	return nil
}

func (r *registryFromProviders) HasResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (bool, error) {
	resourceState, err := r.LookupResourceInState(ctx, input)
	if err != nil {
		return false, err
	}

	if resourceState == nil {
		return false, nil
	}

	return true, nil
}

func (r *registryFromProviders) AcquireResourceLock(
	ctx context.Context,
	input *provider.AcquireResourceLockInput,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			r.resourceLocksMu.Lock()
			lockKey := createResourceLockKey(input.InstanceID, input.ResourceName)
			if r.checkLock(lockKey) {
				r.resourceLocks[lockKey] = &resourceLock{
					instanceID:   input.InstanceID,
					resourceName: input.ResourceName,
					lockTime:     r.clock.Now(),
					acquiredBy:   input.AcquiredBy,
				}
				r.resourceLocksMu.Unlock()
				return nil
			}
			r.resourceLocksMu.Unlock()
			time.Sleep(r.resourceLockCheckInterval)
		}
	}
}

// The resource locks mutex must be held when calling this method.
func (r *registryFromProviders) checkLock(lockKey string) bool {
	if lock, exists := r.resourceLocks[lockKey]; exists {
		// If the lock exists, check if it has timed out.
		if r.clock.Now().Sub(lock.lockTime) < r.resourceLockTimeout {
			// Lock is still held, cannot acquire.
			return false
		}
		// Lock has timed out, remove it.
		delete(r.resourceLocks, lockKey)
	}

	return true
}

func (r *registryFromProviders) ReleaseResourceLock(
	ctx context.Context,
	instanceID string,
	resourceName string,
) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	lockKey := createResourceLockKey(instanceID, resourceName)
	delete(r.resourceLocks, lockKey)
}

func (r *registryFromProviders) ReleaseResourceLocks(ctx context.Context, instanceID string) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	for lockKey := range r.resourceLocks {
		if lockKeyHasInstanceID(lockKey, instanceID) {
			delete(r.resourceLocks, lockKey)
		}
	}
}

func (r *registryFromProviders) ReleaseResourceLocksAcquiredBy(
	ctx context.Context,
	instanceID string,
	acquiredBy string,
) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	for lockKey, lock := range r.resourceLocks {
		if lockKeyHasInstanceID(lockKey, instanceID) && lock.acquiredBy == acquiredBy {
			delete(r.resourceLocks, lockKey)
		}
	}
}

func createResourceLockKey(instanceID, resourceName string) string {
	return fmt.Sprintf("%s:%s", instanceID, resourceName)
}

func lockKeyHasInstanceID(lockKey, instanceID string) bool {
	instanceIDPrefix := fmt.Sprintf("%s:", instanceID)
	return strings.HasPrefix(lockKey, instanceIDPrefix)
}

func (r *registryFromProviders) WithParams(
	params core.BlueprintParams,
) Registry {
	return &registryFromProviders{
		providers:                    r.providers,
		transformers:                 r.transformers,
		resourceCache:                r.resourceCache,
		abstractResourceCache:        r.abstractResourceCache,
		resourceTypes:                r.resourceTypes,
		stabilisationPollingInterval: r.stabilisationPollingInterval,
		stateContainer:               r.stateContainer,
		resourceLocks:                r.resourceLocks,
		resourceLockTimeout:          r.resourceLockTimeout,
		resourceLockCheckInterval:    r.resourceLockCheckInterval,
		clock:                        r.clock,
		params:                       params,
		// The same locks must be used across all registry instances derived
		// from the same base registry, derived registries exist to allow
		// the attachment of parameters to the registry.
		mu:              r.mu,
		resourceLocksMu: r.resourceLocksMu,
	}
}

func (r *registryFromProviders) ListTransformers(ctx context.Context) ([]string, error) {
	transformerNames := make([]string, 0, len(r.transformers))
	for name := range r.transformers {
		transformerNames = append(transformerNames, name)
	}
	return transformerNames, nil
}

func (r *registryFromProviders) getResourceType(ctx context.Context, resourceType string) (provider.Resource, error) {
	resource, cached := r.resourceCache.Get(resourceType)
	if cached {
		return resource, nil
	}

	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	provider, ok := r.providers[providerNamespace]
	if !ok {
		return nil, errResourceTypeProviderNotFound(providerNamespace, resourceType)
	}
	resourceImpl, err := provider.Resource(ctx, resourceType)
	if err != nil || resourceImpl == nil {
		return nil, errProviderResourceTypeNotFound(resourceType, providerNamespace)
	}
	r.resourceCache.Set(resourceType, resourceImpl)

	return resourceImpl, nil
}

func (r *registryFromProviders) getAbstractResourceType(ctx context.Context, resourceType string) (transform.AbstractResource, error) {
	resource, cached := r.abstractResourceCache.Get(resourceType)
	if cached {
		return resource, nil
	}

	var abstractResource transform.AbstractResource
	// Transformers do not have namespaces that correspond to resource type prefixes
	// so we need to iterate through all transformers to find the correct one.
	// This shouldn't be a problem as in practice, a small number of transformers
	// will be used at a time.
	for _, transformer := range r.transformers {
		var err error
		abstractResource, err = transformer.AbstractResource(ctx, resourceType)
		if err == nil && abstractResource != nil {
			break
		}
	}

	if abstractResource == nil {
		return nil, errAbstactResourceTypeNotFound(resourceType)
	}

	r.abstractResourceCache.Set(resourceType, abstractResource)

	return abstractResource, nil
}

func getResolvedResourceFromChanges(changes *provider.Changes) *provider.ResolvedResource {
	if changes == nil {
		return nil
	}

	return changes.AppliedResourceInfo.ResourceWithResolvedSubs
}

func getResourceNameFromChanges(changes *provider.Changes) string {
	if changes == nil {
		return ""
	}

	return changes.AppliedResourceInfo.ResourceName
}

func getComputedFieldsFromChanges(changes *provider.Changes) []string {
	if changes == nil {
		return []string{}
	}

	return changes.ComputedFields
}

func metadataStateFromResolvedResource(
	resolvedResource *provider.ResolvedResource,
) *state.ResourceMetadataState {
	if resolvedResource == nil || resolvedResource.Metadata == nil {
		return nil
	}

	metadata := resolvedResource.Metadata
	return &state.ResourceMetadataState{
		DisplayName: core.StringValue(metadata.DisplayName),
		Annotations: fieldsFromMappingNode(metadata.Annotations),
		Labels:      getValuesFromStringMap(metadata.Labels),
		Custom:      metadata.Custom,
	}
}

func fieldsFromMappingNode(
	mappingNode *core.MappingNode,
) map[string]*core.MappingNode {
	if mappingNode == nil {
		return map[string]*core.MappingNode{}
	}

	return mappingNode.Fields
}

func getValuesFromStringMap(
	stringMap *schema.StringMap,
) map[string]string {
	if stringMap == nil {
		return map[string]string{}
	}

	return stringMap.Values
}
