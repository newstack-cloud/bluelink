package container

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourceDestroyer provides an interface for a service that destroys a resource
// as a part of the deployment process for a blueprint instance.
type ResourceDestroyer interface {
	Destroy(
		ctx context.Context,
		resourceElement state.Element,
		instanceID string,
		deployCtx *DeployContext,
	)
}

// NewDefaultResourceDestroyer creates a new instance of the default implementation
// of the service that destroys a resource as a part of the deployment process for a blueprint instance.
func NewDefaultResourceDestroyer(
	clock core.Clock,
	defaultRetryPolicy *provider.RetryPolicy,
) ResourceDestroyer {
	return &defaultResourceDestroyer{
		clock:              clock,
		defaultRetryPolicy: defaultRetryPolicy,
	}
}

type defaultResourceDestroyer struct {
	clock              core.Clock
	defaultRetryPolicy *provider.RetryPolicy
}

func (d *defaultResourceDestroyer) Destroy(
	ctx context.Context,
	resourceElement state.Element,
	instanceID string,
	deployCtx *DeployContext,
) {
	resourceState := getResourceStateByName(
		deployCtx.InstanceStateSnapshot,
		resourceElement.LogicalName(),
	)
	if resourceState == nil {
		deployCtx.Channels.ErrChan <- errResourceNotFoundInState(
			resourceElement.LogicalName(),
			instanceID,
		)
		return
	}

	deployCtx.Logger.Info("loading resource plugin implementation for destruction")
	resourceImplementation, err := getProviderResourceImplementation(
		ctx,
		resourceElement.LogicalName(),
		resourceState.Type,
		deployCtx.ResourceProviders,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	deployCtx.Logger.Info("loading provider retry policy for resource destruction")
	policy, err := getRetryPolicy(
		ctx,
		deployCtx.ResourceProviders,
		resourceElement.LogicalName(),
		d.defaultRetryPolicy,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	err = d.destroyResource(
		ctx,
		&deploymentElementInfo{
			element:    resourceElement,
			instanceID: instanceID,
		},
		deployCtx.InstanceStateSnapshot.InstanceName,
		resourceImplementation,
		deployCtx,
		provider.CreateRetryContext(policy),
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
	}
}

func (d *defaultResourceDestroyer) destroyResource(
	ctx context.Context,
	resourceInfo *deploymentElementInfo,
	instanceName string,
	resourceImplementation provider.Resource,
	deployCtx *DeployContext,
	resourceRetryInfo *provider.RetryContext,
) error {
	resourceRemovalStartTime := d.clock.Now()
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:      resourceInfo.instanceID,
		ResourceID:      resourceInfo.element.ID(),
		ResourceName:    resourceInfo.element.LogicalName(),
		Group:           deployCtx.CurrentGroupIndex,
		Status:          determineResourceDestroyingStatus(deployCtx.Rollback),
		PreciseStatus:   determinePreciseResourceDestroyingStatus(deployCtx.Rollback),
		UpdateTimestamp: d.clock.Now().Unix(),
		Attempt:         resourceRetryInfo.Attempt,
	}

	deployCtx.Logger.Info(
		"calling resource plugin implementation to destroy resource",
		core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
	)

	resourceState := getResourceStateByName(
		deployCtx.InstanceStateSnapshot,
		resourceInfo.element.LogicalName(),
	)
	providerNamespace := provider.ExtractProviderFromItemType(resourceState.Type)
	err := resourceImplementation.Destroy(ctx, &provider.ResourceDestroyInput{
		InstanceID:    resourceInfo.instanceID,
		InstanceName:  instanceName,
		ResourceID:    resourceInfo.element.ID(),
		ResourceState: resourceState,
		ProviderContext: provider.NewProviderContextFromParams(
			providerNamespace,
			deployCtx.ParamOverrides,
		),
	})
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during resource destruction",
				core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleDestroyResourceRetry(
				ctx,
				resourceInfo,
				instanceName,
				resourceImplementation,
				provider.RetryContextWithStartTime(resourceRetryInfo, resourceRemovalStartTime),
				[]string{retryErr.ChildError.Error()},
				deployCtx,
			)
		}

		var resourceDestroyErr *provider.ResourceDestroyError
		if provider.AsResourceDestroyError(err, &resourceDestroyErr) {
			deployCtx.Logger.Debug(
				"terminal error occurred during resource destruction",
				core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleDestroyResourceTerminalFailure(
				resourceInfo,
				provider.RetryContextWithStartTime(resourceRetryInfo, resourceRemovalStartTime),
				resourceDestroyErr.FailureReasons,
				deployCtx,
			)
		}

		deployCtx.Logger.Warn(
			"an unknown error occurred during resource destruction, "+
				"plugins should wrap all errors in the appropriate provider error",
			core.IntegerLogField("attempt", int64(resourceRetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed to be fatal
		// and the deployment process will be stopped without reporting a failure state.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return err
	}

	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:      resourceInfo.instanceID,
		ResourceID:      resourceInfo.element.ID(),
		ResourceName:    resourceInfo.element.LogicalName(),
		Group:           deployCtx.CurrentGroupIndex,
		Status:          determineResourceDestroyedStatus(deployCtx.Rollback),
		PreciseStatus:   determinePreciseResourceDestroyedStatus(deployCtx.Rollback),
		UpdateTimestamp: d.clock.Now().Unix(),
		Attempt:         resourceRetryInfo.Attempt,
		Durations: determineResourceDeployFinishedDurations(
			resourceRetryInfo,
			d.clock.Since(resourceRemovalStartTime),
			/* configCompleteDuration */ nil,
		),
	}

	return nil
}

func (d *defaultResourceDestroyer) handleDestroyResourceRetry(
	ctx context.Context,
	resourceInfo *deploymentElementInfo,
	instanceName string,
	resourceImplementation provider.Resource,
	resourceRetryInfo *provider.RetryContext,
	failureReasons []string,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(
		resourceRetryInfo.AttemptStartTime,
	)
	nextRetryInfo := provider.RetryContextWithNextAttempt(resourceRetryInfo, currentAttemptDuration)
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:      resourceInfo.instanceID,
		ResourceID:      resourceInfo.element.ID(),
		ResourceName:    resourceInfo.element.LogicalName(),
		Group:           deployCtx.CurrentGroupIndex,
		Status:          determineResourceDestroyFailedStatus(deployCtx.Rollback),
		PreciseStatus:   determinePreciseResourceDestroyFailedStatus(deployCtx.Rollback),
		FailureReasons:  failureReasons,
		Attempt:         resourceRetryInfo.Attempt,
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
		waitTimeMS := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		time.Sleep(time.Duration(waitTimeMS) * time.Millisecond)
		return d.destroyResource(
			ctx,
			resourceInfo,
			instanceName,
			resourceImplementation,
			deployCtx,
			nextRetryInfo,
		)
	}

	deployCtx.Logger.Debug(
		"resource destruction failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil
}

func (d *defaultResourceDestroyer) handleDestroyResourceTerminalFailure(
	resourceInfo *deploymentElementInfo,
	resourceRetryInfo *provider.RetryContext,
	failureReasons []string,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(
		resourceRetryInfo.AttemptStartTime,
	)
	deployCtx.Channels.ResourceUpdateChan <- ResourceDeployUpdateMessage{
		InstanceID:      resourceInfo.instanceID,
		ResourceID:      resourceInfo.element.ID(),
		ResourceName:    resourceInfo.element.LogicalName(),
		Group:           deployCtx.CurrentGroupIndex,
		Status:          determineResourceDestroyFailedStatus(deployCtx.Rollback),
		PreciseStatus:   determinePreciseResourceDestroyFailedStatus(deployCtx.Rollback),
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
