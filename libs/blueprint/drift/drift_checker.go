package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/two-hundred/celerity/libs/blueprint/changes"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/schema"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	commoncore "github.com/two-hundred/celerity/libs/common/core"
)

// Checker is an interface for behaviour
// that can be used to check if resources within
// a blueprint have drifted from the current state
// persisted with the blueprint framework.
// This is useful to detect situations where resources
// in an upstream provider (e.g. an AWS account) have been modified
// manually or by other means, and the blueprint state
// is no longer in sync with the actual state of the
// resources.
// A checker is only responsible for checking and persisting
// drift, the course of action to resolve the drift is
// left to the user.
type Checker interface {
	// CheckDrift checks the drift of all resources in the blueprint
	// with the given instance ID.
	// This will always check the drift with the upstream provider,
	// the state container can be used to retrieve the last known
	// drift state that was previously checked.
	// In most cases, this method will persist the results of the
	// drift check with the configured state container.
	// This returns a map of resource IDs to their drift state ONLY
	// if the resource has drifted from the last known state.
	CheckDrift(
		ctx context.Context,
		instanceID string,
		params core.BlueprintParams,
	) (map[string]*state.ResourceDriftState, error)
	// CheckResourceDrift checks the drift of a single resource
	// with the given instance ID and resource ID.
	// This will always check the drift with the upstream provider,
	// the state container can be used to retrieve the last known
	// drift state that was previously checked.
	// In most cases, this method will persist the results of the
	// drift check with the configured state container.
	// This will return nil if the resource has not drifted from
	// the last known state.
	CheckResourceDrift(
		ctx context.Context,
		instanceID string,
		resourceID string,
		params core.BlueprintParams,
	) (*state.ResourceDriftState, error)
}

type defaultChecker struct {
	stateContainer  state.Container
	providers       map[string]provider.Provider
	changeGenerator changes.ResourceChangeGenerator
	clock           core.Clock
	logger          core.Logger
}

// NewDefaultChecker creates a new instance
// of the default drift checker implementation.
func NewDefaultChecker(
	stateContainer state.Container,
	providers map[string]provider.Provider,
	changeGenerator changes.ResourceChangeGenerator,
	clock core.Clock,
	logger core.Logger,
) Checker {
	return &defaultChecker{
		stateContainer,
		providers,
		changeGenerator,
		clock,
		logger,
	}
}

func (c *defaultChecker) CheckDrift(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
) (map[string]*state.ResourceDriftState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)
	instances := c.stateContainer.Instances()

	instanceLogger.Info(
		fmt.Sprintf("Fetching instance state for instance %s", instanceID),
	)
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		instanceLogger.Debug(
			fmt.Sprintf("Failed to fetch instance state for instance %s", instanceID),
		)
		return nil, err
	}

	driftResults := map[string]*state.ResourceDriftState{}
	for _, resource := range instanceState.Resources {
		resourceLogger := instanceLogger.WithFields(
			core.StringLogField("resourceId", resource.ResourceID),
		)
		resourceLogger.Debug(
			fmt.Sprintf("Checking drift for resource %s", resource.ResourceID),
		)
		resourceDrift, err := c.checkResourceDrift(ctx, resource, params, resourceLogger)
		if err != nil {
			instanceLogger.Debug(
				fmt.Sprintf("Failed to check drift for resource %s", resource.ResourceID),
				core.StringLogField("resourceId", resource.ResourceID),
				core.ErrorLogField("error", err),
			)
			return nil, err
		}

		// A nil resource drift means that the resource has not drifted.
		if resourceDrift != nil {
			driftResults[resource.ResourceID] = resourceDrift
		}
	}

	return driftResults, nil
}

func (c *defaultChecker) CheckResourceDrift(
	ctx context.Context,
	instanceID string,
	resourceID string,
	params core.BlueprintParams,
) (*state.ResourceDriftState, error) {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("resourceId", resourceID),
	)
	resources := c.stateContainer.Resources()

	resourceLogger.Info(
		fmt.Sprintf("Fetching state for resource %s", resourceID),
	)
	resourceState, err := resources.Get(ctx, resourceID)
	if err != nil {
		resourceLogger.Debug(
			fmt.Sprintf("Failed to fetch state for resource %s", resourceID),
		)
		return nil, err
	}

	return c.checkResourceDrift(ctx, &resourceState, params, resourceLogger)
}

func (c *defaultChecker) checkResourceDrift(
	ctx context.Context,
	resource *state.ResourceState,
	params core.BlueprintParams,
	resourceLogger core.Logger,
) (*state.ResourceDriftState, error) {
	resourceLogger.Debug(
		"Loading resource plugin implementation for resource type",
		core.StringLogField("resourceType", resource.ResourceType),
	)
	providerNamespace := provider.ExtractProviderFromItemType(resource.ResourceType)
	resourceImpl, resourceProvider, err := c.getResourceImplementation(ctx, providerNamespace, resource.ResourceType)
	if err != nil {
		resourceLogger.Debug(
			"Failed to load resource plugin implementation for resource type",
			core.StringLogField("resourceType", resource.ResourceType),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	resourceLogger.Debug(
		"Loading retry policy for resource provider",
	)
	policy, err := c.getRetryPolicy(
		ctx,
		resourceProvider,
		provider.DefaultRetryPolicy,
	)
	if err != nil {
		resourceLogger.Debug(
			"Failed to load retry policy for resource provider",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	resourceLogger.Info(
		"Retrieving external state for the resource from the provider",
	)
	retryCtx := provider.CreateRetryContext(policy)
	externalStateOutput, err := c.getResourceExternalState(
		ctx,
		resourceImpl,
		&provider.ResourceGetExternalStateInput{
			InstanceID:      resource.InstanceID,
			ResourceID:      resource.ResourceID,
			ProviderContext: provider.NewProviderContextFromParams(providerNamespace, params),
		},
		retryCtx,
		resourceLogger,
	)
	if err != nil {
		return nil, err
	}

	if externalStateOutput == nil {
		resourceLogger.Debug(
			"External state for the resource is nil, moving on",
		)
		return nil, nil
	}

	driftedResourceInfo := createDriftedResourceInfo(
		resource,
		externalStateOutput,
	)
	resourceChanges, err := c.changeGenerator.GenerateChanges(
		ctx,
		driftedResourceInfo,
		resourceImpl,
		/* resolveOnDeploy */ []string{},
		params,
	)
	if err != nil {
		return nil, err
	}

	if !hasChanges(resourceChanges) {
		resourceLogger.Debug(
			"No changes detected indicating that the resource has not drifted" +
				", updating resource state as not drifted",
		)
		_, err = c.stateContainer.Resources().RemoveDrift(
			ctx,
			resource.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	resourceLogger.Debug(
		"Changes have been detected indicating that the resource has drifted" +
			", updating resource state to reflect this",
	)

	currentTime := int(c.clock.Now().Unix())
	driftState := state.ResourceDriftState{
		ResourceID:       resource.ResourceID,
		ResourceName:     resource.ResourceName,
		ResourceSpecData: resource.ResourceSpecData,
		Difference:       toResourceDriftChanges(resourceChanges),
		Timestamp:        &currentTime,
	}

	err = c.stateContainer.Resources().SaveDrift(
		ctx,
		driftState,
	)
	if err != nil {
		return nil, err
	}

	return &driftState, nil
}

func (c *defaultChecker) getResourceExternalState(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceGetExternalStateInput,
	retryCtx *provider.RetryContext,
	resourceLogger core.Logger,
) (*provider.ResourceGetExternalStateOutput, error) {
	getExternalStateStartTime := c.clock.Now()
	externalStateOutput, err := resource.GetExternalState(ctx, input)
	if err != nil {
		if provider.IsRetryableError(err) {
			resourceLogger.Debug(
				"retryable error occurred during external resource state retrieval",
				core.IntegerLogField("attempt", int64(retryCtx.Attempt)),
				core.ErrorLogField("error", err),
			)
			return c.handleGetResourceExternalStateRetry(
				ctx,
				resource,
				input,
				provider.RetryContextWithStartTime(
					retryCtx,
					getExternalStateStartTime,
				),
				resourceLogger,
			)
		}

		return nil, err
	}

	return externalStateOutput, nil
}

func (c *defaultChecker) handleGetResourceExternalStateRetry(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceGetExternalStateInput,
	retryCtx *provider.RetryContext,
	resourceLogger core.Logger,
) (*provider.ResourceGetExternalStateOutput, error) {
	currentAttemptDuration := c.clock.Since(
		retryCtx.AttemptStartTime,
	)
	nextRetryCtx := provider.RetryContextWithNextAttempt(retryCtx, currentAttemptDuration)

	if !nextRetryCtx.ExceededMaxRetries {
		waitTimeMs := provider.CalculateRetryWaitTimeMS(nextRetryCtx.Policy, nextRetryCtx.Attempt)
		time.Sleep(time.Duration(waitTimeMs) * time.Millisecond)
		return c.getResourceExternalState(
			ctx,
			resource,
			input,
			nextRetryCtx,
			resourceLogger,
		)
	}

	resourceLogger.Debug(
		"resource external state retrieval failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryCtx.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryCtx.Policy.MaxRetries)),
	)

	return nil, nil
}

func (c *defaultChecker) getResourceImplementation(
	ctx context.Context,
	providerNamespace string,
	resourceType string,
) (provider.Resource, provider.Provider, error) {
	provider, ok := c.providers[providerNamespace]
	if !ok {
		return nil, nil, fmt.Errorf("provider %s not found", providerNamespace)
	}

	resourceImpl, err := provider.Resource(ctx, resourceType)
	if err != nil {
		return nil, nil, err
	}

	return resourceImpl, provider, nil
}

func (c *defaultChecker) getRetryPolicy(
	ctx context.Context,
	resourceProvider provider.Provider,
	defaultRetryPolicy *provider.RetryPolicy,
) (*provider.RetryPolicy, error) {
	retryPolicy, err := resourceProvider.RetryPolicy(ctx)
	if err != nil {
		return nil, err
	}

	if retryPolicy == nil {
		return defaultRetryPolicy, nil
	}

	return retryPolicy, nil
}

func createDriftedResourceInfo(
	resource *state.ResourceState,
	externalStateOutput *provider.ResourceGetExternalStateOutput,
) *provider.ResourceInfo {
	resourceFromExternalState := externalStateOutput.ResourceSpecState
	return &provider.ResourceInfo{
		ResourceID:           resource.ResourceID,
		ResourceName:         resource.ResourceName,
		CurrentResourceState: resource,
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{
				Value: resource.ResourceType,
			},
			Description: core.MappingNodeFromString(resource.Description),
			Metadata:    createResolvedResourceMetadata(resource),
			Spec:        resourceFromExternalState,
		},
	}
}

func createResolvedResourceMetadata(
	resource *state.ResourceState,
) *provider.ResolvedResourceMetadata {
	if resource.Metadata == nil {
		return nil
	}

	return &provider.ResolvedResourceMetadata{
		DisplayName: core.MappingNodeFromString(
			resource.Metadata.DisplayName,
		),
		Annotations: &core.MappingNode{
			Fields: resource.Metadata.Annotations,
		},
		Labels: &schema.StringMap{
			Values: resource.Metadata.Labels,
		},
		Custom: resource.Metadata.Custom,
	}
}

func hasChanges(changes *provider.Changes) bool {
	return len(changes.ModifiedFields) > 0 ||
		len(changes.NewFields) > 0 ||
		len(changes.RemovedFields) > 0
}

func toResourceDriftChanges(changes *provider.Changes) *state.ResourceDriftChanges {
	return &state.ResourceDriftChanges{
		ModifiedFields:  toResourceDriftFieldChanges(changes.ModifiedFields),
		NewFields:       toResourceDriftFieldChanges(changes.NewFields),
		RemovedFields:   changes.RemovedFields,
		UnchangedFields: changes.UnchangedFields,
	}
}

func toResourceDriftFieldChanges(
	fieldChanges []provider.FieldChange,
) []*state.ResourceDriftFieldChange {
	return commoncore.Map(
		fieldChanges,
		func(fieldChange provider.FieldChange, _ int) *state.ResourceDriftFieldChange {
			return &state.ResourceDriftFieldChange{
				FieldPath:    fieldChange.FieldPath,
				StateValue:   fieldChange.PrevValue,
				DriftedValue: fieldChange.NewValue,
			}
		},
	)
}
