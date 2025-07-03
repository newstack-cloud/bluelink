package container

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinkDeployer provides an interface for a service that deploys a link between two
// resources as a part of the deployment process for a blueprint instance.
// This can be used for creating, updating and deleting a link between two resources.
// "Deploying" a link in the context of destruction means detaching information
// saved in the 2 resources related to the link and the removal of any intermediary
// resources created by a provider link implementation.
type LinkDeployer interface {
	Deploy(
		ctx context.Context,
		linkElement state.Element,
		instanceID string,
		InstanceName string,
		linkUpdateType provider.LinkUpdateType,
		linkImplementation provider.Link,
		deployCtx *DeployContext,
		retryPolicy *provider.RetryPolicy,
	) error
}

// LinkDeployResult contains the result of deploying a link between two resources
// in a blueprint instance.
// LinkData contains the merged data from the link update operations on the two resources
// and intermediary resources.
type LinkDeployResult struct {
	IntermediaryResourceStates []*state.LinkIntermediaryResourceState
	LinkData                   *core.MappingNode
	ResourceDataMappings       map[string]string
}

// NewDefaultLinkDeployer creates a new instance of the default implementation
// of the service that deploys a link between two resources as a part of the deployment process
// for a blueprint instance.
func NewDefaultLinkDeployer(clock core.Clock, stateContainer state.Container) LinkDeployer {
	return &defaultLinkDeployer{
		clock:          clock,
		stateContainer: stateContainer,
	}
}

type defaultLinkDeployer struct {
	clock          core.Clock
	stateContainer state.Container
}

func (d *defaultLinkDeployer) Deploy(
	ctx context.Context,
	linkElement state.Element,
	instanceID string,
	instanceName string,
	linkUpdateType provider.LinkUpdateType,
	linkImplementation provider.Link,
	deployCtx *DeployContext,
	retryPolicy *provider.RetryPolicy,
) error {
	linkDependencyInfo := extractLinkDirectDependencies(
		linkElement.LogicalName(),
	)

	resourceAInfo := getResourceInfoFromStateForLinkDeployment(
		deployCtx.InstanceStateSnapshot,
		linkDependencyInfo.resourceAName,
	)
	resourceBInfo := getResourceInfoFromStateForLinkDeployment(
		deployCtx.InstanceStateSnapshot,
		linkDependencyInfo.resourceBName,
	)

	if linkUpdateType == provider.LinkUpdateTypeCreate {
		deployCtx.Logger.Info(
			"persisting skeleton state for new link",
			core.StringLogField("linkId", linkElement.ID()),
		)
		links := d.stateContainer.Links()
		err := links.Save(
			ctx,
			state.LinkState{
				LinkID:        linkElement.ID(),
				Name:          linkElement.LogicalName(),
				InstanceID:    instanceID,
				Status:        core.LinkStatusUnknown,
				PreciseStatus: core.PreciseLinkStatusUnknown,
			},
		)
		if err != nil {
			return err
		}
	}

	linkInfo := &deploymentElementInfo{
		element:    linkElement,
		instanceID: instanceID,
	}
	linkCtx := provider.NewLinkContextFromParams(deployCtx.ParamOverrides)
	resourceAOutput, stop, err := d.updateLinkResourceA(
		ctx,
		linkImplementation,
		&provider.LinkUpdateResourceInput{
			ResourceInfo:      resourceAInfo,
			OtherResourceInfo: resourceBInfo,
			InstanceName:      instanceName,
			LinkUpdateType:    linkUpdateType,
			LinkContext:       linkCtx,
		},
		linkInfo,
		provider.CreateRetryContext(retryPolicy),
		deployCtx,
	)
	if err != nil {
		return err
	}
	if stop {
		return nil
	}

	resourceBOutput, stop, err := d.updateLinkResourceB(
		ctx,
		linkImplementation,
		&provider.LinkUpdateResourceInput{
			ResourceInfo:      resourceBInfo,
			OtherResourceInfo: resourceAInfo,
			InstanceName:      instanceName,
			LinkUpdateType:    linkUpdateType,
			LinkContext:       linkCtx,
		},
		linkInfo,
		provider.CreateRetryContext(retryPolicy),
		deployCtx,
	)
	if err != nil {
		return err
	}
	if stop {
		return nil
	}

	err = d.updateLinkIntermediaryResources(
		ctx,
		linkImplementation,
		&provider.LinkUpdateIntermediaryResourcesInput{
			ResourceAInfo:         resourceAInfo,
			ResourceBInfo:         resourceBInfo,
			InstanceName:          instanceName,
			LinkUpdateType:        linkUpdateType,
			LinkContext:           linkCtx,
			ResourceDeployService: deployCtx.ResourceRegistry,
		},
		linkInfo,
		provider.CreateRetryContext(retryPolicy),
		&linkUpdateResourceOutputs{
			resourceAOutput: resourceAOutput,
			resourceBOutput: resourceBOutput,
		},
		deployCtx,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *defaultLinkDeployer) updateLinkResourceA(
	ctx context.Context,
	linkImplementation provider.Link,
	input *provider.LinkUpdateResourceInput,
	linkInfo *deploymentElementInfo,
	updateResourceARetryInfo *provider.RetryContext,
	deployCtx *DeployContext,
) (*provider.LinkUpdateResourceOutput, bool, error) {
	updateResourceAStartTime := d.clock.Now()
	deployCtx.Channels.LinkUpdateChan <- d.createLinkUpdatingResourceAMessage(
		linkInfo,
		deployCtx,
		updateResourceARetryInfo,
		input.LinkUpdateType,
	)

	deployCtx.Logger.Info(
		"calling link plugin implementation to update resource A",
		core.IntegerLogField("attempt", int64(updateResourceARetryInfo.Attempt)),
	)

	resourceAOutput, err := linkImplementation.UpdateResourceA(ctx, input)
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during resource A update",
				core.IntegerLogField("attempt", int64(updateResourceARetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleUpdateLinkResourceARetry(
				ctx,
				linkInfo,
				linkImplementation,
				provider.RetryContextWithStartTime(
					updateResourceARetryInfo,
					updateResourceAStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: []string{retryErr.ChildError.Error()},
					input:          input,
				},
				deployCtx,
			)
		}

		var linkUpdateResourceAError *provider.LinkUpdateResourceAError
		if provider.AsLinkUpdateResourceAError(err, &linkUpdateResourceAError) {
			deployCtx.Logger.Debug(
				"terminal error occurred during resource A update",
				core.IntegerLogField("attempt", int64(updateResourceARetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			stop, err := d.handleUpdateResourceATerminalFailure(
				linkInfo,
				provider.RetryContextWithStartTime(
					updateResourceARetryInfo,
					updateResourceAStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: linkUpdateResourceAError.FailureReasons,
					input:          input,
				},
				deployCtx,
			)
			return nil, stop, err
		}

		deployCtx.Logger.Warn(
			unknownErrorWarningText("link resource A update"),
			core.IntegerLogField("attempt", int64(updateResourceARetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed to be fatal
		// and the deployment process will be stopped without reporting a failure state.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return nil, true, err
	}

	deployCtx.Channels.LinkUpdateChan <- d.createLinkResourceAUpdatedMessage(
		linkInfo,
		deployCtx,
		provider.RetryContextWithStartTime(
			updateResourceARetryInfo,
			updateResourceAStartTime,
		),
		input.LinkUpdateType,
	)

	return resourceAOutput, false, nil
}

func (d *defaultLinkDeployer) handleUpdateLinkResourceARetry(
	ctx context.Context,
	linkInfo *deploymentElementInfo,
	linkImplementation provider.Link,
	updateResourceARetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (*provider.LinkUpdateResourceOutput, bool, error) {
	currentAttemptDuration := d.clock.Since(updateResourceARetryInfo.AttemptStartTime)
	nextRetryInfo := provider.RetryContextWithNextAttempt(updateResourceARetryInfo, currentAttemptDuration)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkResourceAUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons: updateInfo.failureReasons,
		// Attempt and retry information included the status update is specific to
		// updating resource A, each component of a link change will have its own
		// number of attempts and retry information.
		CurrentStageAttempt:  updateResourceARetryInfo.Attempt,
		CanRetryCurrentStage: !nextRetryInfo.ExceededMaxRetries,
		UpdateTimestamp:      d.clock.Now().Unix(),
		// Attempt durations will be accumulated and sent in the status updates
		// for each subsequent retry.
		// Total duration will be calculated if retry limit is exceeded.
		Durations: determineLinkUpdateResourceARetryFailureDurations(
			nextRetryInfo,
		),
	}

	if !nextRetryInfo.ExceededMaxRetries {
		waitTimeMS := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		time.Sleep(time.Duration(waitTimeMS) * time.Millisecond)
		return d.updateLinkResourceA(
			ctx,
			linkImplementation,
			updateInfo.input,
			linkInfo,
			nextRetryInfo,
			deployCtx,
		)
	}

	deployCtx.Logger.Debug(
		"link resource A update failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil, true, nil
}

func (d *defaultLinkDeployer) handleUpdateResourceATerminalFailure(
	linkInfo *deploymentElementInfo,
	updateResourceARetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (bool, error) {
	currentAttemptDuration := d.clock.Since(updateResourceARetryInfo.AttemptStartTime)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkResourceAUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons:      updateInfo.failureReasons,
		CurrentStageAttempt: updateResourceARetryInfo.Attempt,
		UpdateTimestamp:     d.clock.Now().Unix(),
		Durations: determineLinkUpdateResourceAFinishedDurations(
			updateResourceARetryInfo,
			currentAttemptDuration,
		),
	}

	return true, nil
}

func (d *defaultLinkDeployer) createLinkUpdatingResourceAMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateResourceARetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus: determinePreciseLinkUpdatingResourceAStatus(
			deployCtx.Rollback,
		),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateResourceARetryInfo.Attempt,
	}
}

func (d *defaultLinkDeployer) createLinkResourceAUpdatedMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateResourceARetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	durations := determineLinkUpdateResourceAFinishedDurations(
		updateResourceARetryInfo,
		d.clock.Since(updateResourceARetryInfo.AttemptStartTime),
	)
	linkName := linkInfo.element.LogicalName()
	deployCtx.State.SetLinkDurationInfo(linkName, durations)

	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkName,
		// We are still in the process of updating the link,
		// resource B and intermediary resources still need to be updated.
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus:       determinePreciseLinkResourceAUpdatedStatus(deployCtx.Rollback),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateResourceARetryInfo.Attempt,
		Durations:           durations,
	}
}

func (d *defaultLinkDeployer) updateLinkResourceB(
	ctx context.Context,
	linkImplementation provider.Link,
	input *provider.LinkUpdateResourceInput,
	linkInfo *deploymentElementInfo,
	updateResourceBRetryInfo *provider.RetryContext,
	deployCtx *DeployContext,
) (*provider.LinkUpdateResourceOutput, bool, error) {
	updateResourceBStartTime := d.clock.Now()
	deployCtx.Channels.LinkUpdateChan <- d.createLinkUpdatingResourceBMessage(
		linkInfo,
		deployCtx,
		updateResourceBRetryInfo,
		input.LinkUpdateType,
	)

	deployCtx.Logger.Info(
		"calling link plugin implementation to update resource B",
		core.IntegerLogField("attempt", int64(updateResourceBRetryInfo.Attempt)),
	)

	resourceBOutput, err := linkImplementation.UpdateResourceB(ctx, input)
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during resource B update",
				core.IntegerLogField("attempt", int64(updateResourceBRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleUpdateLinkResourceBRetry(
				ctx,
				linkInfo,
				linkImplementation,
				provider.RetryContextWithStartTime(
					updateResourceBRetryInfo,
					updateResourceBStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: []string{retryErr.ChildError.Error()},
					input:          input,
				},
				deployCtx,
			)
		}

		var linkUpdateResourceBError *provider.LinkUpdateResourceBError
		if provider.AsLinkUpdateResourceBError(err, &linkUpdateResourceBError) {
			deployCtx.Logger.Debug(
				"terminal error occurred during resource B update",
				core.IntegerLogField("attempt", int64(updateResourceBRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			stop, err := d.handleUpdateResourceBTerminalFailure(
				linkInfo,
				provider.RetryContextWithStartTime(
					updateResourceBRetryInfo,
					updateResourceBStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: linkUpdateResourceBError.FailureReasons,
					input:          input,
				},
				deployCtx,
			)
			return nil, stop, err
		}

		deployCtx.Logger.Warn(
			unknownErrorWarningText("link resource B update"),
			core.IntegerLogField("attempt", int64(updateResourceBRetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed to be fatal
		// and the deployment process will be stopped without reporting a failure state.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return nil, true, err
	}

	deployCtx.Channels.LinkUpdateChan <- d.createLinkResourceBUpdatedMessage(
		linkInfo,
		deployCtx,
		provider.RetryContextWithStartTime(
			updateResourceBRetryInfo,
			updateResourceBStartTime,
		),
		input.LinkUpdateType,
	)

	return resourceBOutput, false, nil
}

func (d *defaultLinkDeployer) handleUpdateLinkResourceBRetry(
	ctx context.Context,
	linkInfo *deploymentElementInfo,
	linkImplementation provider.Link,
	updateResourceBRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (*provider.LinkUpdateResourceOutput, bool, error) {
	currentAttemptDuration := d.clock.Since(updateResourceBRetryInfo.AttemptStartTime)
	nextRetryInfo := provider.RetryContextWithNextAttempt(updateResourceBRetryInfo, currentAttemptDuration)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkResourceBUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons: updateInfo.failureReasons,
		// Attempt and retry information included the status update is specific to
		// updating resource B, each component of a link change will have its own
		// number of attempts and retry information.
		CurrentStageAttempt:  updateResourceBRetryInfo.Attempt,
		CanRetryCurrentStage: !nextRetryInfo.ExceededMaxRetries,
		UpdateTimestamp:      d.clock.Now().Unix(),
		// Attempt durations will be accumulated and sent in the status updates
		// for each subsequent retry.
		// Total duration will be calculated if retry limit is exceeded.
		Durations: determineLinkUpdateResourceBRetryFailureDurations(
			nextRetryInfo,
		),
	}

	if !nextRetryInfo.ExceededMaxRetries {
		waitTimeMS := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		time.Sleep(time.Duration(waitTimeMS) * time.Millisecond)
		return d.updateLinkResourceB(
			ctx,
			linkImplementation,
			updateInfo.input,
			linkInfo,
			nextRetryInfo,
			deployCtx,
		)
	}

	deployCtx.Logger.Debug(
		"link resource B update failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil, true, nil
}

func (d *defaultLinkDeployer) handleUpdateResourceBTerminalFailure(
	linkInfo *deploymentElementInfo,
	updateResourceBRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (bool, error) {
	currentAttemptDuration := d.clock.Since(updateResourceBRetryInfo.AttemptStartTime)
	linkName := linkInfo.element.LogicalName()
	accumDurationInfo := deployCtx.State.GetLinkDurationInfo(linkName)
	durations := determineLinkUpdateResourceBFinishedDurations(
		updateResourceBRetryInfo,
		currentAttemptDuration,
		accumDurationInfo,
	)
	deployCtx.State.SetLinkDurationInfo(linkName, durations)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkResourceBUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons:      updateInfo.failureReasons,
		CurrentStageAttempt: updateResourceBRetryInfo.Attempt,
		UpdateTimestamp:     d.clock.Now().Unix(),
		Durations:           durations,
	}

	return true, nil
}

func (d *defaultLinkDeployer) createLinkUpdatingResourceBMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateResourceBRetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus: determinePreciseLinkUpdatingResourceBStatus(
			deployCtx.Rollback,
		),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateResourceBRetryInfo.Attempt,
	}
}

func (d *defaultLinkDeployer) createLinkResourceBUpdatedMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateResourceBRetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	linkName := linkInfo.element.LogicalName()
	accumDurationInfo := deployCtx.State.GetLinkDurationInfo(linkName)
	durations := determineLinkUpdateResourceBFinishedDurations(
		updateResourceBRetryInfo,
		d.clock.Since(updateResourceBRetryInfo.AttemptStartTime),
		accumDurationInfo,
	)
	deployCtx.State.SetLinkDurationInfo(linkName, durations)
	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		// We are still in the process of updating the link,
		// intermediary resources still need to be updated.
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus:       determinePreciseLinkResourceBUpdatedStatus(deployCtx.Rollback),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateResourceBRetryInfo.Attempt,
		Durations:           durations,
	}
}

func (d *defaultLinkDeployer) updateLinkIntermediaryResources(
	ctx context.Context,
	linkImplementation provider.Link,
	input *provider.LinkUpdateIntermediaryResourcesInput,
	linkInfo *deploymentElementInfo,
	updateIntermediariesRetryInfo *provider.RetryContext,
	resourceOutputs *linkUpdateResourceOutputs,
	deployCtx *DeployContext,
) error {
	updateIntermediariesStartTime := d.clock.Now()
	deployCtx.Channels.LinkUpdateChan <- d.createLinkUpdatingIntermediaryResourcesMessage(
		linkInfo,
		deployCtx,
		updateIntermediariesRetryInfo,
		input.LinkUpdateType,
	)

	deployCtx.Logger.Info(
		"calling link plugin implementation to update intermediary resources",
		core.IntegerLogField("attempt", int64(updateIntermediariesRetryInfo.Attempt)),
	)

	intermediaryResourcesOutput, err := linkImplementation.UpdateIntermediaryResources(ctx, input)
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during intermediary resources update",
				core.IntegerLogField("attempt", int64(updateIntermediariesRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleUpdateLinkIntermediaryResourcesRetry(
				ctx,
				linkInfo,
				linkImplementation,
				provider.RetryContextWithStartTime(
					updateIntermediariesRetryInfo,
					updateIntermediariesStartTime,
				),
				&linkUpdateIntermediaryResourcesInfo{
					failureReasons: []string{retryErr.ChildError.Error()},
					input:          input,
				},
				resourceOutputs,
				deployCtx,
			)
		}

		var linkUpdateIntermediariesError *provider.LinkUpdateIntermediaryResourcesError
		if provider.AsLinkUpdateIntermediaryResourcesError(err, &linkUpdateIntermediariesError) {
			deployCtx.Logger.Debug(
				"terminal error occurred during intermediary resources update",
				core.IntegerLogField("attempt", int64(updateIntermediariesRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleUpdateIntermediaryResourcesTerminalFailure(
				linkInfo,
				provider.RetryContextWithStartTime(
					updateIntermediariesRetryInfo,
					updateIntermediariesStartTime,
				),
				&linkUpdateIntermediaryResourcesInfo{
					failureReasons: linkUpdateIntermediariesError.FailureReasons,
					input:          input,
				},
				deployCtx,
			)
		}

		deployCtx.Logger.Warn(
			unknownErrorWarningText("link intermediary resources update"),
			core.IntegerLogField("attempt", int64(updateIntermediariesRetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed to be fatal
		// and the deployment process will be stopped without reporting a failure state.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return err
	}

	// We need to store the link deploy result before sending the status update
	// to ensure consistency in the temporary state of the link.
	// This makes sure that the link deploy result is available in the ephemeral state
	// when the status update handler persists the results to the state container.
	result := createLinkDeployResult(
		resourceOutputs.resourceAOutput,
		resourceOutputs.resourceBOutput,
		intermediaryResourcesOutput,
	)
	deployCtx.State.SetLinkDeployResult(linkInfo.element.LogicalName(), result)

	deployCtx.Channels.LinkUpdateChan <- d.createLinkIntermediariesUpdatedMessage(
		linkInfo,
		deployCtx,
		updateIntermediariesRetryInfo,
		input.LinkUpdateType,
	)

	return nil
}

func (d *defaultLinkDeployer) createLinkIntermediariesUpdatedMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateIntermediariesRetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	linkName := linkInfo.element.LogicalName()
	accumDurationInfo := deployCtx.State.GetLinkDurationInfo(linkName)
	durations := determineLinkUpdateIntermediariesFinishedDurations(
		updateIntermediariesRetryInfo,
		d.clock.Since(updateIntermediariesRetryInfo.AttemptStartTime),
		accumDurationInfo,
	)
	deployCtx.State.SetLinkDurationInfo(linkName, durations)

	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		// Updating intermediary resources is the last step in the link update process.
		Status: determineLinkOperationSuccessfullyFinishedStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus: determinePreciseLinkIntermediariesUpdatedStatus(
			deployCtx.Rollback,
		),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateIntermediariesRetryInfo.Attempt,
		Durations:           durations,
	}
}

func (d *defaultLinkDeployer) handleUpdateLinkIntermediaryResourcesRetry(
	ctx context.Context,
	linkInfo *deploymentElementInfo,
	linkImplementation provider.Link,
	updateIntermediaryResourcesRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateIntermediaryResourcesInfo,
	resourceOutputs *linkUpdateResourceOutputs,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(
		updateIntermediaryResourcesRetryInfo.AttemptStartTime,
	)
	nextRetryInfo := provider.RetryContextWithNextAttempt(
		updateIntermediaryResourcesRetryInfo,
		currentAttemptDuration,
	)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkIntermediariesUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons: updateInfo.failureReasons,
		// Attempt and retry information included the status update is specific to
		// updating intermediary resources, each component of a link change will have its own
		// number of attempts and retry information.
		CurrentStageAttempt:  updateIntermediaryResourcesRetryInfo.Attempt,
		CanRetryCurrentStage: !nextRetryInfo.ExceededMaxRetries,
		UpdateTimestamp:      d.clock.Now().Unix(),
		// Attempt durations will be accumulated and sent in the status updates
		// for each subsequent retry.
		// Total duration will be calculated if retry limit is exceeded.
		Durations: determineLinkUpdateIntermediariesRetryFailureDurations(
			nextRetryInfo,
		),
	}

	if !nextRetryInfo.ExceededMaxRetries {
		waitTimeMS := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		time.Sleep(time.Duration(waitTimeMS) * time.Millisecond)
		return d.updateLinkIntermediaryResources(
			ctx,
			linkImplementation,
			updateInfo.input,
			linkInfo,
			nextRetryInfo,
			resourceOutputs,
			deployCtx,
		)
	}

	deployCtx.Logger.Debug(
		"link intermediary resources update failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil
}

func (d *defaultLinkDeployer) handleUpdateIntermediaryResourcesTerminalFailure(
	linkInfo *deploymentElementInfo,
	updateIntermediariesRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateIntermediaryResourcesInfo,
	deployCtx *DeployContext,
) error {
	currentAttemptDuration := d.clock.Since(
		updateIntermediariesRetryInfo.AttemptStartTime,
	)
	linkName := linkInfo.element.LogicalName()
	accumDurationInfo := deployCtx.State.GetLinkDurationInfo(linkName)
	durations := determineLinkUpdateIntermediariesFinishedDurations(
		updateIntermediariesRetryInfo,
		currentAttemptDuration,
		accumDurationInfo,
	)
	deployCtx.State.SetLinkDurationInfo(linkName, durations)

	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkIntermediariesUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons:      updateInfo.failureReasons,
		CurrentStageAttempt: updateIntermediariesRetryInfo.Attempt,
		UpdateTimestamp:     d.clock.Now().Unix(),
		Durations:           durations,
	}

	return nil
}

func (d *defaultLinkDeployer) createLinkUpdatingIntermediaryResourcesMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateIntermediariesRetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus: determinePreciseLinkUpdatingIntermediariesStatus(
			deployCtx.Rollback,
		),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateIntermediariesRetryInfo.Attempt,
	}
}

func getResourceInfoFromStateForLinkDeployment(
	instanceState *state.InstanceState,
	resourceName string,
) *provider.ResourceInfo {
	resourceState := getResourceStateByName(instanceState, resourceName)
	if resourceState == nil {
		return nil
	}

	return &provider.ResourceInfo{
		ResourceID:           resourceState.ResourceID,
		ResourceName:         resourceName,
		InstanceID:           instanceState.InstanceID,
		CurrentResourceState: resourceState,
	}
}

func createLinkDeployResult(
	resourceAOutput *provider.LinkUpdateResourceOutput,
	resourceBOutput *provider.LinkUpdateResourceOutput,
	intermediaryResourcesOutput *provider.LinkUpdateIntermediaryResourcesOutput,
) *LinkDeployResult {
	resourceAOutputLinkData := getResourceOutputLinkData(resourceAOutput)
	resourceBOutputLinkData := getResourceOutputLinkData(resourceBOutput)
	intermediaryResourcesOutputLinkData := getIntermediaryResourcesOutputLinkData(
		intermediaryResourcesOutput,
	)
	resADataMappings := getOutputResourceDataMappings(resourceAOutput)
	resBDataMappings := getOutputResourceDataMappings(resourceBOutput)
	intermediaryResourcesDataMappings := getIntermediaryResourcesOutputDataMappings(
		intermediaryResourcesOutput,
	)
	intermediaryResourceStates := getIntermediaryResourcesOutputStates(
		intermediaryResourcesOutput,
	)

	return &LinkDeployResult{
		IntermediaryResourceStates: intermediaryResourceStates,
		LinkData: core.MergeMaps(
			resourceAOutputLinkData,
			resourceBOutputLinkData,
			intermediaryResourcesOutputLinkData,
		),
		ResourceDataMappings: core.MergeNativeMaps(
			resADataMappings,
			resBDataMappings,
			intermediaryResourcesDataMappings,
		),
	}
}

func getResourceOutputLinkData(output *provider.LinkUpdateResourceOutput) *core.MappingNode {
	if output == nil {
		return nil
	}

	return output.LinkData
}

func getOutputResourceDataMappings(
	output *provider.LinkUpdateResourceOutput,
) map[string]string {
	if output == nil {
		return nil
	}

	return output.ResourceDataMappings
}

func getIntermediaryResourcesOutputLinkData(
	output *provider.LinkUpdateIntermediaryResourcesOutput,
) *core.MappingNode {
	if output == nil {
		return nil
	}

	return output.LinkData
}

func getIntermediaryResourcesOutputDataMappings(
	output *provider.LinkUpdateIntermediaryResourcesOutput,
) map[string]string {
	if output == nil {
		return nil
	}

	return output.ResourceDataMappings
}

func getIntermediaryResourcesOutputStates(
	output *provider.LinkUpdateIntermediaryResourcesOutput,
) []*state.LinkIntermediaryResourceState {
	if output == nil {
		return nil
	}

	return output.IntermediaryResourceStates
}

func unknownErrorWarningText(operation string) string {
	return "an unknown error occurred during " + operation + ", " +
		"plugins should wrap all errors in the appropriate provider error"
}

type linkUpdateResourceOutputs struct {
	resourceAOutput *provider.LinkUpdateResourceOutput
	resourceBOutput *provider.LinkUpdateResourceOutput
}
