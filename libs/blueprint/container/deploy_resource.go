package container

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

const (
	resourceStabilisingTimeoutFailureMessage = "Resource failed to stabilise within the configured timeout"
)

// ResourceDeployer provides an interface for a service that deploys
// a resource as a part of the deployment process for a blueprint instance.
type ResourceDeployer interface {
	Deploy(
		ctx context.Context,
		instanceID string,
		chainLinkNode *links.ChainLinkNode,
		changes *changes.BlueprintChanges,
		deployCtx *DeployContext,
	)
}

// ResourceSubstitutionResolver provides an interface for a service that
// is responsible for resolving substitutions in a resource definition.
type ResourceSubstitutionResolver interface {
	// ResolveInResource resolves substitutions in a resource.
	ResolveInResource(
		ctx context.Context,
		resourceName string,
		resource *schema.Resource,
		resolveTargetInfo *subengine.ResolveResourceTargetInfo,
	) (*subengine.ResolveInResourceResult, error)
}

// NewDefaultResourceDeployer creates a new instance of the default
// implementation of the service that deploys a resource as a part of
// the deployment process for a blueprint instance.
func NewDefaultResourceDeployer(
	clock core.Clock,
	idGenerator core.IDGenerator,
	defaultRetryPolicy *provider.RetryPolicy,
	stabilityPollingConfig *ResourceStabilityPollingConfig,
	substitutionResolver ResourceSubstitutionResolver,
	resourceCache *core.Cache[*provider.ResolvedResource],
	stateContainer state.Container,
) ResourceDeployer {
	return &defaultResourceDeployer{
		clock:                  clock,
		idGenerator:            idGenerator,
		defaultRetryPolicy:     defaultRetryPolicy,
		substitutionResolver:   substitutionResolver,
		stabilityPollingConfig: stabilityPollingConfig,
		resourceCache:          resourceCache,
		stateContainer:         stateContainer,
	}
}

type defaultResourceDeployer struct {
	clock                  core.Clock
	idGenerator            core.IDGenerator
	defaultRetryPolicy     *provider.RetryPolicy
	stabilityPollingConfig *ResourceStabilityPollingConfig
	substitutionResolver   ResourceSubstitutionResolver
	resourceCache          *core.Cache[*provider.ResolvedResource]
	stateContainer         state.Container
}

func (d *defaultResourceDeployer) Deploy(
	ctx context.Context,
	instanceID string,
	chainLinkNode *links.ChainLinkNode,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) {
	resourceChangeInfo := getResourceChangeInfo(chainLinkNode.ResourceName, changes)
	if resourceChangeInfo == nil {
		deployCtx.Channels.ErrChan <- errMissingResourceChanges(
			chainLinkNode.ResourceName,
		)
		return
	}
	partiallyResolvedResource := getResolvedResourceFromChanges(resourceChangeInfo.changes)
	if partiallyResolvedResource == nil {
		deployCtx.Channels.ErrChan <- errMissingPartiallyResolvedResource(
			chainLinkNode.ResourceName,
		)
		return
	}
	resolvedResource, err := d.resolveResourceForDeployment(
		ctx,
		partiallyResolvedResource,
		chainLinkNode,
		changes.ResolveOnDeploy,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	resourceID, err := d.getResourceID(resourceChangeInfo.changes)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	if resourceChangeInfo.isNew {
		resources := d.stateContainer.Resources()
		err := resources.Save(ctx, state.ResourceState{
			ResourceID:    resourceID,
			Name:          chainLinkNode.ResourceName,
			InstanceID:    instanceID,
			Type:          resolvedResource.Type.Value,
			Status:        core.ResourceStatusUnknown,
			PreciseStatus: core.PreciseResourceStatusUnknown,
		})
		if err != nil {
			deployCtx.Channels.ErrChan <- err
			return
		}
	}

	resourceImplementation, err := getProviderResourceImplementation(
		ctx,
		chainLinkNode.ResourceName,
		resolvedResource.Type.Value,
		deployCtx.ResourceProviders,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	policy, err := getRetryPolicy(
		ctx,
		deployCtx.ResourceProviders,
		chainLinkNode.ResourceName,
		d.defaultRetryPolicy,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	// The resource state is made available in a change set at the time
	// changes were staged, this is primarily to provide a convenient way
	// to surface the current state to users during the "planning" phase.
	// As there can be a significant delay between change staging and deployment,
	// we'll replace the current state in the change set with the latest snapshot
	// of the resource state.
	resourceState := getResourceStateByName(
		deployCtx.InstanceStateSnapshot,
		chainLinkNode.ResourceName,
	)

	err = d.deployResource(
		ctx,
		&resourceDeployInfo{
			instanceID:   instanceID,
			instanceName: deployCtx.InstanceStateSnapshot.InstanceName,
			resourceID:   resourceID,
			resourceName: chainLinkNode.ResourceName,
			resourceImpl: resourceImplementation,
			changes: prepareResourceChangesForDeployment(
				resourceChangeInfo.changes,
				resolvedResource,
				resourceState,
				resourceID,
				instanceID,
			),
			isNew: resourceChangeInfo.isNew,
		},
		resolvedResource.Type.Value,
		deployCtx,
		provider.CreateRetryContext(policy),
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
	}
}

func (d *defaultResourceDeployer) deployResource(
	ctx context.Context,
	resourceInfo *resourceDeployInfo,
	resourceType string,
	deployCtx *DeployContext,
	resourceRetryInfo *provider.RetryContext,
) error {
	resourceDeploymentStartTime := d.clock.Now()
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceDeployingStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceDeployingStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		UpdateTimestamp: d.clock.Now().Unix(),
		Attempt:         resourceRetryInfo.Attempt,
	}

	// Check for context cancellation before calling the plugin.
	// This allows us to exit early if the deployment has been interrupted.
	// The drain mechanism will send INTERRUPTED status via markInFlightElementsAsInterrupted.
	select {
	case <-ctx.Done():
		deployCtx.Logger.Debug("context cancelled before resource deployment")
		return ctx.Err()
	default:
	}

	deployCtx.Logger.Info(
		"calling resource plugin implementation to deploy resource",
		core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
	)

	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	output, err := resourceInfo.resourceImpl.Deploy(
		ctx,
		&provider.ResourceDeployInput{
			InstanceID:   resourceInfo.instanceID,
			InstanceName: resourceInfo.instanceName,
			ResourceID:   resourceInfo.resourceID,
			Changes:      resourceInfo.changes,
			ProviderContext: provider.NewProviderContextFromParamsWithOptions(
				providerNamespace,
				deployCtx.ParamOverrides,
				&provider.ProviderContextOptions{
					TaggingConfig: createResourceTaggingConfig(
						deployCtx.TaggingConfig,
						providerNamespace,
						deployCtx.ProviderMetadataLookup,
					),
				},
			),
		},
	)
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during resource deployment",
				core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleDeployResourceRetry(
				ctx,
				resourceInfo,
				provider.RetryContextWithStartTime(
					resourceRetryInfo,
					resourceDeploymentStartTime,
				),
				[]string{retryErr.ChildError.Error()},
				deployCtx,
			)
		}

		var resourceDeployErr *provider.ResourceDeployError
		if provider.AsResourceDeployError(err, &resourceDeployErr) {
			deployCtx.Logger.Debug(
				"terminal error occurred during resource deployment",
				core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleDeployResourceTerminalFailure(
				resourceInfo,
				provider.RetryContextWithStartTime(
					resourceRetryInfo,
					resourceDeploymentStartTime,
				),
				resourceDeployErr.FailureReasons,
				deployCtx,
			)
		}

		deployCtx.Logger.Warn(
			"an unknown error occurred during resource deployment, "+
				"plugins should wrap all errors in the appropriate provider error",
			core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed
		// to be fatal and the deployment process will be stopped without reporting
		// a failure status.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return err
	}

	deployCtx.Logger.Debug(
		"merging user-defined resource spec with computed field values returned by the plugin",
	)
	resolvedResource := getResolvedResourceFromChanges(resourceInfo.changes)
	mergedSpecState, err := specmerge.MergeResourceSpec(
		resolvedResource,
		resourceInfo.resourceName,
		output.ComputedFieldValues,
		resourceInfo.changes.ComputedFields,
	)
	if err != nil {
		return err
	}

	deployCtx.State.SetResourceData(
		resourceInfo.resourceName,
		&CollectedResourceData{
			Spec: mergedSpecState,
			Metadata: resolvedMetadataToState(
				extractResolvedMetadataFromResourceInfo(resourceInfo),
			),
			ComputedFields: resourceInfo.changes.ComputedFields,
		},
	)
	// At this point, we mark the resource as "config complete", a separate coroutine
	// is invoked asynchronously to poll the resource for stability.
	// Once the resource is stable, a status update will be sent with the appropriate
	// "deployed" status.
	configCompleteMsg := ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceConfigCompleteStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceConfigCompleteStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		UpdateTimestamp: d.clock.Now().Unix(),
		Attempt:         resourceRetryInfo.Attempt,
		Durations: determineResourceDeployConfigCompleteDurations(
			resourceRetryInfo,
			d.clock.Since(resourceDeploymentStartTime),
		),
	}
	deployCtx.Channels.ResourceUpdateChan <- configCompleteMsg
	deployCtx.State.SetResourceDurationInfo(
		resourceInfo.resourceName,
		configCompleteMsg.Durations,
	)

	go d.pollForResourceStability(
		ctx,
		resourceInfo,
		resourceRetryInfo,
		deployCtx,
	)

	return nil
}

func (d *defaultResourceDeployer) pollForResourceStability(
	ctx context.Context,
	resourceInfo *resourceDeployInfo,
	resourceRetryInfo *provider.RetryContext,
	deployCtx *DeployContext,
) {
	pollingStabilisationStartTime := d.clock.Now()

	ctxWithPollingTimeout, cancel := context.WithTimeout(
		ctx,
		d.stabilityPollingConfig.PollingTimeout,
	)
	defer cancel()

	for {
		select {
		case <-ctxWithPollingTimeout.Done():
			deployCtx.Channels.ResourceUpdateChan <- d.createResourceStabiliseTimeoutMessage(
				resourceInfo,
				resourceRetryInfo,
				pollingStabilisationStartTime,
				deployCtx,
			)
			return
		case <-time.After(d.stabilityPollingConfig.PollingInterval):
			resourceData := deployCtx.State.GetResourceData(resourceInfo.resourceName)
			resolvedResource := getResolvedResourceFromChanges(resourceInfo.changes)
			providerNamespace := provider.ExtractProviderFromItemType(
				changes.GetResourceTypeFromResolved(resolvedResource),
			)
			deployCtx.Logger.Debug(
				"checking if resource has stabilised with resource plugin implementation",
			)
			hasStabilisedRetryCtx := provider.CreateRetryContext(
				resourceRetryInfo.Policy,
			)
			output, err := d.hasStabilised(
				ctxWithPollingTimeout,
				resourceInfo.resourceImpl,
				&provider.ResourceHasStabilisedInput{
					InstanceID:       resourceInfo.instanceID,
					InstanceName:     resourceInfo.instanceName,
					ResourceID:       resourceInfo.resourceID,
					ResourceSpec:     resourceData.Spec,
					ResourceMetadata: resourceData.Metadata,
					ProviderContext: provider.NewProviderContextFromParamsWithOptions(
						providerNamespace,
						deployCtx.ParamOverrides,
						&provider.ProviderContextOptions{
							TaggingConfig: createResourceTaggingConfig(
								deployCtx.TaggingConfig,
								providerNamespace,
								deployCtx.ProviderMetadataLookup,
							),
						},
					),
				},
				hasStabilisedRetryCtx,
				deployCtx.Logger,
			)
			if err != nil {
				deployCtx.Logger.Debug(
					"error occurred while checking resource for stability",
					core.ErrorLogField("error", err),
				)
				deployCtx.Channels.ErrChan <- err
				return
			}

			if output.Stabilised {
				deployCtx.Channels.ResourceUpdateChan <- d.createResourceStabilisedMessage(
					resourceInfo,
					resourceRetryInfo,
					pollingStabilisationStartTime,
					deployCtx,
				)
				return
			}
		}
	}
}

func (d *defaultResourceDeployer) hasStabilised(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceHasStabilisedInput,
	retryCtx *provider.RetryContext,
	logger core.Logger,
) (*provider.ResourceHasStabilisedOutput, error) {
	stabiliseCheckStartTime := d.clock.Now()
	hasStabilisedOutput, err := resource.HasStabilised(ctx, input)
	if err != nil {
		if provider.IsRetryableError(err) {
			logger.Debug(
				"retryable error occurred while checking if resource has stabilised",
				core.IntegerLogField("attempt", int64(retryCtx.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleHasStabilisedRetry(
				ctx,
				resource,
				input,
				provider.RetryContextWithStartTime(
					retryCtx,
					stabiliseCheckStartTime,
				),
				logger,
			)
		}

		return nil, err
	}

	return hasStabilisedOutput, nil
}

func (d *defaultResourceDeployer) handleHasStabilisedRetry(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceHasStabilisedInput,
	retryCtx *provider.RetryContext,
	logger core.Logger,
) (*provider.ResourceHasStabilisedOutput, error) {
	currentAttemptDuration := d.clock.Since(
		retryCtx.AttemptStartTime,
	)
	nextRetryCtx := provider.RetryContextWithNextAttempt(retryCtx, currentAttemptDuration)

	if !nextRetryCtx.ExceededMaxRetries {
		waitTimeMs := provider.CalculateRetryWaitTimeMS(nextRetryCtx.Policy, nextRetryCtx.Attempt)
		if err := sleepWithContext(ctx, time.Duration(waitTimeMs)*time.Millisecond); err != nil {
			logger.Debug("context cancelled during stability check retry wait")
			return nil, err
		}
		return d.hasStabilised(
			ctx,
			resource,
			input,
			nextRetryCtx,
			logger,
		)
	}

	logger.Debug(
		"resource stabilisation check failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryCtx.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryCtx.Policy.MaxRetries)),
	)

	return nil, nil
}

func (d *defaultResourceDeployer) createResourceStabiliseTimeoutMessage(
	resourceInfo *resourceDeployInfo,
	resourceRetryInfo *provider.RetryContext,
	pollingStabilisationStartTime time.Time,
	deployCtx *DeployContext,
) ResourceDeployUpdateMessage {
	configCompleteDurationInfo := deployCtx.State.GetResourceDurationInfo(
		resourceInfo.resourceName,
	)

	return ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		FailureReasons:  []string{resourceStabilisingTimeoutFailureMessage},
		Attempt:         resourceRetryInfo.Attempt,
		CanRetry:        false,
		UpdateTimestamp: d.clock.Now().Unix(),
		Durations: addTotalToResourceCompletionDurations(
			configCompleteDurationInfo,
			d.clock.Since(pollingStabilisationStartTime),
		),
	}
}

func (d *defaultResourceDeployer) createResourceStabilisedMessage(
	resourceInfo *resourceDeployInfo,
	resourceRetryInfo *provider.RetryContext,
	pollingStabilisationStartTime time.Time,
	deployCtx *DeployContext,
) ResourceDeployUpdateMessage {
	configCompleteDurationInfo := deployCtx.State.GetResourceDurationInfo(
		resourceInfo.resourceName,
	)

	return ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceDeployedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceDeployedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		Attempt:         resourceRetryInfo.Attempt,
		CanRetry:        false,
		UpdateTimestamp: d.clock.Now().Unix(),
		Durations: addTotalToResourceCompletionDurations(
			configCompleteDurationInfo,
			d.clock.Since(pollingStabilisationStartTime),
		),
	}
}

func (d *defaultResourceDeployer) handleDeployResourceRetry(
	ctx context.Context,
	resourceInfo *resourceDeployInfo,
	resourceRetryInfo *provider.RetryContext,
	failureReasons []string,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(
		resourceRetryInfo.AttemptStartTime,
	)
	nextRetryInfo := provider.RetryContextWithNextAttempt(resourceRetryInfo, currentAttemptDuration)
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		Attempt:         resourceRetryInfo.Attempt,
		FailureReasons:  failureReasons,
		CanRetry:        !nextRetryInfo.ExceededMaxRetries,
		UpdateTimestamp: d.clock.Now().Unix(),
		// Attempt durations will be accumulated and sent in the status updates
		// for each subsequent retry.
		// Total duration will be calculated if retry limit is exceeded.
		Durations: determineResourceRetryFailureDurations(
			nextRetryInfo,
		),
	}

	if !nextRetryInfo.ExceededMaxRetries {
		waitTimeMs := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		if err := sleepWithContext(ctx, time.Duration(waitTimeMs)*time.Millisecond); err != nil {
			deployCtx.Logger.Debug("context cancelled during retry wait")
			return err
		}
		resolvedResource := getResolvedResourceFromChanges(resourceInfo.changes)
		resourceType := changes.GetResourceTypeFromResolved(resolvedResource)
		return d.deployResource(
			ctx,
			resourceInfo,
			resourceType,
			deployCtx,
			nextRetryInfo,
		)
	}

	deployCtx.Logger.Debug(
		"resource deployment failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil
}

func (d *defaultResourceDeployer) handleDeployResourceTerminalFailure(
	resourceInfo *resourceDeployInfo,
	resourceRetryInfo *provider.RetryContext,
	failureReasons []string,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(resourceRetryInfo.AttemptStartTime)
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:   resourceInfo.instanceID,
		ResourceID:   resourceInfo.resourceID,
		ResourceName: resourceInfo.resourceName,
		Group:        deployCtx.CurrentGroupIndex,
		Status: determineResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		PreciseStatus: determinePreciseResourceDeployFailedStatus(
			deployCtx.Rollback,
			resourceInfo.isNew,
		),
		FailureReasons:  failureReasons,
		Attempt:         resourceRetryInfo.Attempt,
		CanRetry:        false,
		UpdateTimestamp: d.clock.Now().Unix(),
		Durations: determineResourceDeployFinishedDurations(
			resourceRetryInfo,
			currentAttemptDuration,
			/* configCompleteDuration */ nil,
		),
	}

	return nil
}

func (d *defaultResourceDeployer) resolveResourceForDeployment(
	ctx context.Context,
	partiallyResolvedResource *provider.ResolvedResource,
	node *links.ChainLinkNode,
	resolveOnDeploy []string,
) (*provider.ResolvedResource, error) {
	if !resourceHasFieldsToResolve(node.ResourceName, resolveOnDeploy) {
		// Even when no fields need to be resolved, cache the partially resolved
		// resource so it can be used when resolving exports and other elements
		// that reference this resource.
		d.resourceCache.Set(node.ResourceName, partiallyResolvedResource)
		return partiallyResolvedResource, nil
	}

	resolveResourceResult, err := d.substitutionResolver.ResolveInResource(
		ctx,
		node.ResourceName,
		node.Resource,
		&subengine.ResolveResourceTargetInfo{
			ResolveFor:        subengine.ResolveForDeployment,
			PartiallyResolved: partiallyResolvedResource,
		},
	)
	if err != nil {
		return nil, err
	}

	// Cache the resolved resource so that it can be used in resolving other elements
	// that reference fields in the current resource.
	d.resourceCache.Set(
		node.ResourceName,
		resolveResourceResult.ResolvedResource,
	)

	return resolveResourceResult.ResolvedResource, nil
}

// ResourceStabilityPollingConfig represents the configuration for
// polling resources for stability.
type ResourceStabilityPollingConfig struct {
	// PollingInterval is the interval at which the resource will be polled
	// for stability.
	PollingInterval time.Duration
	// PollingTimeout is the maximum amount of time that the resource will be
	// polled for stability.
	PollingTimeout time.Duration
}

// DefaultResourceStabilityPollingConfig is a reasonable default configuration
// for polling resources for stability.
var DefaultResourceStabilityPollingConfig = &ResourceStabilityPollingConfig{
	PollingInterval: 5 * time.Second,
	PollingTimeout:  30 * time.Minute,
}

func (d *defaultResourceDeployer) getResourceID(changes *provider.Changes) (string, error) {
	if changes.AppliedResourceInfo.ResourceID == "" {
		return d.idGenerator.GenerateID()
	}

	return changes.AppliedResourceInfo.ResourceID, nil
}

// createResourceTaggingConfig creates a TaggingConfig for a specific resource
// by combining the base tagging config with provider-specific metadata.
func createResourceTaggingConfig(
	baseConfig *provider.TaggingConfig,
	providerNamespace string,
	metadataLookup func(string) (string, string),
) *provider.TaggingConfig {
	if baseConfig == nil {
		return nil
	}

	pluginID, pluginVersion := lookupProviderMetadata(providerNamespace, metadataLookup)

	return &provider.TaggingConfig{
		Enabled:               baseConfig.Enabled,
		Prefix:                baseConfig.Prefix,
		DeployEngineVersion:   baseConfig.DeployEngineVersion,
		ProviderPluginID:      pluginID,
		ProviderPluginVersion: pluginVersion,
	}
}

// lookupProviderMetadata retrieves the plugin ID and version for a provider namespace.
func lookupProviderMetadata(
	providerNamespace string,
	metadataLookup func(string) (string, string),
) (pluginID, pluginVersion string) {
	if metadataLookup == nil {
		return "", ""
	}
	return metadataLookup(providerNamespace)
}
