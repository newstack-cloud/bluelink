package container

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
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
	// Contributions holds what the link needs the resources it contributes to include, for
	// the merged update of each of those resources to carry.
	Contributions []*provider.ResourceContribution
	// ContributionRecords holds how each contribution is applied and whether it outlives
	// the link, keyed the same way as ResourceDataMappings.
	ContributionRecords map[string]state.ContributionRecord
}

// NewDefaultLinkDeployer creates a new instance of the default implementation
// of the service that deploys a link between two resources as a part of the deployment process
// for a blueprint instance.
func NewDefaultLinkDeployer(
	clock core.Clock,
	stateContainer state.Container,
	settlePollingConfig *LinkSettlePollingConfig,
) LinkDeployer {
	if settlePollingConfig == nil {
		settlePollingConfig = DefaultLinkSettlePollingConfig
	}

	return &defaultLinkDeployer{
		clock:               clock,
		stateContainer:      stateContainer,
		settlePollingConfig: settlePollingConfig,
	}
}

type defaultLinkDeployer struct {
	clock               core.Clock
	stateContainer      state.Container
	settlePollingConfig *LinkSettlePollingConfig
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
		getResolvedResourceFromInputChanges(deployCtx.InputChanges, linkDependencyInfo.resourceAName),
	)
	resourceBInfo := getResourceInfoFromStateForLinkDeployment(
		deployCtx.InstanceStateSnapshot,
		linkDependencyInfo.resourceBName,
		getResolvedResourceFromInputChanges(deployCtx.InputChanges, linkDependencyInfo.resourceBName),
	)

	var currentLinkState *state.LinkState
	if linkUpdateType == provider.LinkUpdateTypeCreate {
		deployCtx.Logger.Info(
			"persisting skeleton state for new link",
			core.StringLogField("linkId", linkElement.ID()),
		)
		links := d.stateContainer.Links()
		linkState := state.LinkState{
			LinkID:        linkElement.ID(),
			Name:          linkElement.LogicalName(),
			InstanceID:    instanceID,
			Status:        core.LinkStatusUnknown,
			PreciseStatus: core.PreciseLinkStatusUnknown,
		}
		err := links.Save(
			ctx,
			linkState,
		)
		if err != nil {
			return err
		}
		currentLinkState = &linkState
	} else {
		currentLinkState = getLinkStateByName(
			deployCtx.InstanceStateSnapshot,
			linkElement.LogicalName(),
		)
	}

	linkInfo := &deploymentElementInfo{
		element:    linkElement,
		instanceID: instanceID,
	}
	linkCtx := provider.NewLinkContextFromParams(deployCtx.ParamOverrides)
	linkResourceService := newLinkScopedResourceService(
		deployCtx.ResourceRegistry,
		linkElement.ID(),
	)
	contributionsOutput, err := linkImplementation.ProduceResourceContributions(
		ctx,
		&provider.LinkProduceResourceContributionsInput{
			ResourceAInfo:    resourceAInfo,
			ResourceBInfo:    resourceBInfo,
			LinkID:           linkElement.ID(),
			InstanceName:     instanceName,
			LinkUpdateType:   linkUpdateType,
			CurrentLinkState: currentLinkState,
			LinkContext:      linkCtx,
			ResourceService:  linkResourceService,
		},
	)
	if err != nil {
		return err
	}

	linkedResourcesOutput, stop, err := d.updateLinkedResources(
		ctx,
		linkImplementation,
		&provider.LinkUpdateLinkedResourcesInput{
			ResourceAInfo:    resourceAInfo,
			ResourceBInfo:    resourceBInfo,
			LinkID:           linkElement.ID(),
			InstanceName:     instanceName,
			LinkUpdateType:   linkUpdateType,
			CurrentLinkState: currentLinkState,
			LinkContext:      linkCtx,
			ResourceService:  linkResourceService,
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
			ResourceAInfo:    resourceAInfo,
			ResourceBInfo:    resourceBInfo,
			LinkID:           linkElement.ID(),
			InstanceName:     instanceName,
			LinkUpdateType:   linkUpdateType,
			CurrentLinkState: currentLinkState,
			LinkContext:      linkCtx,
			ResourceService:  linkResourceService,
		},
		linkInfo,
		provider.CreateRetryContext(retryPolicy),
		&linkUpdateResourceOutputs{
			linkedResourcesOutput: linkedResourcesOutput,
			contributionsOutput:   contributionsOutput,
		},
		deployCtx,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *defaultLinkDeployer) updateLinkedResources(
	ctx context.Context,
	linkImplementation provider.Link,
	input *provider.LinkUpdateLinkedResourcesInput,
	linkInfo *deploymentElementInfo,
	updateRetryInfo *provider.RetryContext,
	deployCtx *DeployContext,
) (*provider.LinkUpdateLinkedResourcesOutput, bool, error) {
	updateStartTime := d.clock.Now()
	deployCtx.Channels.LinkUpdateChan <- d.createLinkUpdatingLinkedResourcesMessage(
		linkInfo,
		deployCtx,
		updateRetryInfo,
		input.LinkUpdateType,
	)

	written := d.resourcesWrittenByLink(linkInfo, deployCtx, input)
	err := d.acquireLocksForWrittenResources(ctx, linkInfo, written, deployCtx)
	if err != nil {
		return nil, true, err
	}

	// Check for context cancellation before calling the plugin.
	select {
	case <-ctx.Done():
		// Release the locks we just acquired before returning.
		deployCtx.ResourceRegistry.ReleaseResourceLocksAcquiredBy(
			ctx,
			linkInfo.instanceID,
			linkInfo.element.ID(),
		)
		deployCtx.Logger.Debug("context cancelled before link resource update")
		return nil, true, ctx.Err()
	default:
	}

	deployCtx.Logger.Info(
		"calling link plugin implementation to update the resources it links",
		core.IntegerLogField("attempt", int64(updateRetryInfo.Attempt)),
	)

	output, err := linkImplementation.UpdateLinkedResources(ctx, input)
	if err == nil {
		// Inside the locks, so the next link cannot reach a resource while it is still
		// applying what this one wrote.
		err = d.waitForWrittenResourcesToSettle(ctx, written, deployCtx)
	}
	// Regardless of whether or not the update was successful, all locks acquired for this
	// link must be released before retrying or moving on. This covers both the locks taken
	// above and any the link implementation took for itself.
	deployCtx.ResourceRegistry.ReleaseResourceLocksAcquiredBy(
		ctx,
		linkInfo.instanceID,
		linkInfo.element.ID(),
	)
	if err != nil {
		var retryErr *provider.RetryableError
		if provider.AsRetryableError(err, &retryErr) {
			deployCtx.Logger.Debug(
				"retryable error occurred during link resource update",
				core.IntegerLogField("attempt", int64(updateRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			return d.handleUpdateLinkedResourcesRetry(
				ctx,
				linkInfo,
				linkImplementation,
				provider.RetryContextWithStartTime(
					updateRetryInfo,
					updateStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: []string{retryErr.ChildError.Error()},
					input:          input,
				},
				deployCtx,
			)
		}

		var linkUpdateError *provider.LinkUpdateLinkedResourcesError
		if provider.AsLinkUpdateLinkedResourcesError(err, &linkUpdateError) {
			deployCtx.Logger.Debug(
				"terminal error occurred during link resource update",
				core.IntegerLogField("attempt", int64(updateRetryInfo.Attempt)),
				core.ErrorLogField("error", err),
			)
			stop, err := d.handleUpdateLinkedResourcesTerminalFailure(
				linkInfo,
				provider.RetryContextWithStartTime(
					updateRetryInfo,
					updateStartTime,
				),
				&linkUpdateResourceInfo{
					failureReasons: linkUpdateError.FailureReasons,
					input:          input,
				},
				deployCtx,
			)
			return nil, stop, err
		}

		deployCtx.Logger.Warn(
			unknownErrorWarningText("link resource update"),
			core.IntegerLogField("attempt", int64(updateRetryInfo.Attempt)),
			core.ErrorLogField("error", err),
		)
		// For errors that are not wrapped in a provider error, the error is assumed to be fatal
		// and the deployment process will be stopped without reporting a failure state.
		// It is really important that adequate guidance is provided for provider developers
		// to ensure that all errors are wrapped in the appropriate provider error.
		return nil, true, err
	}

	deployCtx.Channels.LinkUpdateChan <- d.createLinkedResourcesUpdatedMessage(
		linkInfo,
		deployCtx,
		provider.RetryContextWithStartTime(
			updateRetryInfo,
			updateStartTime,
		),
		input.LinkUpdateType,
	)

	return output, false, nil
}

// The resources of the link relationship this link declared it writes.
//
// A link that declared it does not write a side is only reading it, so locking it would
// serialise every link that reads the same resource against one another for no benefit,
// and waiting for it to settle would poll a resource this link did not change. A link
// that declared nothing is treated as writing both.
//
// Ordered by resource name rather than by side, so two links that write the same pair
// acquire in the same order and cannot hold what the other is waiting for.
func (d *defaultLinkDeployer) resourcesWrittenByLink(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	input *provider.LinkUpdateLinkedResourcesInput,
) []*provider.ResourceInfo {
	modifies := deployCtx.State.LinkModifies(linkInfo.element.LogicalName())

	written := []*provider.ResourceInfo{}
	if modifies.WritesResourceA() && input.ResourceAInfo != nil {
		written = append(written, input.ResourceAInfo)
	}
	if modifies.WritesResourceB() && input.ResourceBInfo != nil {
		written = append(written, input.ResourceBInfo)
	}

	slices.SortFunc(written, func(a, b *provider.ResourceInfo) int {
		return strings.Compare(a.ResourceName, b.ResourceName)
	})

	return written
}

func (d *defaultLinkDeployer) acquireLocksForWrittenResources(
	ctx context.Context,
	linkInfo *deploymentElementInfo,
	written []*provider.ResourceInfo,
	deployCtx *DeployContext,
) error {
	for _, resourceInfo := range written {
		deployCtx.Logger.Debug(
			"acquiring resource lock for link resource update",
			core.StringLogField("linkId", linkInfo.element.ID()),
			core.StringLogField("resourceName", resourceInfo.ResourceName),
		)
		err := deployCtx.ResourceRegistry.AcquireResourceLock(
			ctx,
			&provider.AcquireResourceLockInput{
				InstanceID:   linkInfo.instanceID,
				ResourceName: resourceInfo.ResourceName,
				AcquiredBy:   linkInfo.element.ID(),
			},
		)
		if err != nil {
			deployCtx.Logger.Error(
				"failed to acquire resource lock for link resource update",
				core.StringLogField("linkId", linkInfo.element.ID()),
				core.StringLogField("resourceName", resourceInfo.ResourceName),
				core.ErrorLogField("error", err),
			)
			// Locks taken before the one that failed are released here, since the caller
			// returns without reaching the release that follows a successful call.
			deployCtx.ResourceRegistry.ReleaseResourceLocksAcquiredBy(
				ctx,
				linkInfo.instanceID,
				linkInfo.element.ID(),
			)
			return err
		}
	}

	return nil
}

func (d *defaultLinkDeployer) waitForWrittenResourcesToSettle(
	ctx context.Context,
	written []*provider.ResourceInfo,
	deployCtx *DeployContext,
) error {
	for _, resourceInfo := range written {
		if err := d.waitForResourceToSettle(ctx, resourceInfo, deployCtx); err != nil {
			return err
		}
	}

	return nil
}

func (d *defaultLinkDeployer) handleUpdateLinkedResourcesRetry(
	ctx context.Context,
	linkInfo *deploymentElementInfo,
	linkImplementation provider.Link,
	updateRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (*provider.LinkUpdateLinkedResourcesOutput, bool, error) {
	currentAttemptDuration := d.clock.Since(updateRetryInfo.AttemptStartTime)
	nextRetryInfo := provider.RetryContextWithNextAttempt(updateRetryInfo, currentAttemptDuration)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkedResourcesUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons: updateInfo.failureReasons,
		// Attempt and retry information in the status update is specific to updating the
		// resources the link relates, each component of a link change will have its own
		// number of attempts and retry information.
		CurrentStageAttempt:  updateRetryInfo.Attempt,
		CanRetryCurrentStage: !nextRetryInfo.ExceededMaxRetries,
		UpdateTimestamp:      d.clock.Now().Unix(),
		// Attempt durations will be accumulated and sent in the status updates
		// for each subsequent retry.
		// Total duration will be calculated if retry limit is exceeded.
		Durations: determineLinkUpdateLinkedResourcesRetryFailureDurations(
			nextRetryInfo,
		),
	}

	if !nextRetryInfo.ExceededMaxRetries {
		waitTimeMS := provider.CalculateRetryWaitTimeMS(nextRetryInfo.Policy, nextRetryInfo.Attempt)
		if err := sleepWithContext(ctx, time.Duration(waitTimeMS)*time.Millisecond); err != nil {
			deployCtx.Logger.Debug("context cancelled during link resource update retry wait")
			return nil, true, err
		}
		return d.updateLinkedResources(
			ctx,
			linkImplementation,
			updateInfo.input,
			linkInfo,
			nextRetryInfo,
			deployCtx,
		)
	}

	deployCtx.Logger.Debug(
		"link resource update failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryInfo.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryInfo.Policy.MaxRetries)),
	)

	return nil, true, nil
}

func (d *defaultLinkDeployer) handleUpdateLinkedResourcesTerminalFailure(
	linkInfo *deploymentElementInfo,
	updateRetryInfo *provider.RetryContext,
	updateInfo *linkUpdateResourceInfo,
	deployCtx *DeployContext,
) (bool, error) {
	currentAttemptDuration := d.clock.Since(updateRetryInfo.AttemptStartTime)
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkInfo.element.LogicalName(),
		Status: determineLinkUpdateFailedStatus(
			deployCtx.Rollback,
			updateInfo.input.LinkUpdateType,
		),
		PreciseStatus: determinePreciseLinkedResourcesUpdateFailedStatus(
			deployCtx.Rollback,
		),
		FailureReasons:      updateInfo.failureReasons,
		CurrentStageAttempt: updateRetryInfo.Attempt,
		UpdateTimestamp:     d.clock.Now().Unix(),
		Durations: determineLinkUpdateLinkedResourcesFinishedDurations(
			updateRetryInfo,
			currentAttemptDuration,
		),
	}

	return true, nil
}

func (d *defaultLinkDeployer) createLinkUpdatingLinkedResourcesMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateRetryInfo *provider.RetryContext,
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
		PreciseStatus: determinePreciseLinkUpdatingLinkedResourcesStatus(
			deployCtx.Rollback,
		),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateRetryInfo.Attempt,
	}
}

func (d *defaultLinkDeployer) createLinkedResourcesUpdatedMessage(
	linkInfo *deploymentElementInfo,
	deployCtx *DeployContext,
	updateRetryInfo *provider.RetryContext,
	linkUpdateType provider.LinkUpdateType,
) LinkDeployUpdateMessage {
	durations := determineLinkUpdateLinkedResourcesFinishedDurations(
		updateRetryInfo,
		d.clock.Since(updateRetryInfo.AttemptStartTime),
	)
	linkName := linkInfo.element.LogicalName()
	deployCtx.State.SetLinkDurationInfo(linkName, durations)

	return LinkDeployUpdateMessage{
		InstanceID: linkInfo.instanceID,
		LinkID:     linkInfo.element.ID(),
		LinkName:   linkName,
		// We are still in the process of updating the link,
		// intermediary resources still need to be updated.
		Status: determineLinkUpdatingStatus(
			deployCtx.Rollback,
			linkUpdateType,
		),
		PreciseStatus:       determinePreciseLinkedResourcesUpdatedStatus(deployCtx.Rollback),
		UpdateTimestamp:     d.clock.Now().Unix(),
		CurrentStageAttempt: updateRetryInfo.Attempt,
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

	// Check for context cancellation before calling the plugin.
	select {
	case <-ctx.Done():
		deployCtx.Logger.Debug("context cancelled before link intermediary resources update")
		return ctx.Err()
	default:
	}

	deployCtx.Logger.Info(
		"calling link plugin implementation to update intermediary resources",
		core.IntegerLogField("attempt", int64(updateIntermediariesRetryInfo.Attempt)),
	)

	intermediaryResourcesOutput, err := linkImplementation.UpdateIntermediaryResources(ctx, input)
	// Regardless of whether or not the intermediary resources update was successful,
	// we need to ensure that all locks acquired by the link implementation's UpdateIntermediaryResources
	// method are released before retrying or moving on.
	deployCtx.ResourceRegistry.ReleaseResourceLocksAcquiredBy(
		ctx,
		linkInfo.instanceID,
		linkInfo.element.ID(),
	)
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
		resourceOutputs.linkedResourcesOutput,
		intermediaryResourcesOutput,
		resourceOutputs.contributionsOutput,
	)
	deployCtx.State.SetLinkDeployResult(linkInfo.element.LogicalName(), result)

	deployCtx.Channels.LinkUpdateChan <- d.createLinkIntermediariesUpdatedMessage(
		linkInfo,
		deployCtx,
		// The retry context has to carry the start time of this attempt, as it
		// does on the retry and terminal failure paths above. The duration for
		// this phase is measured from it, and a context created for a first
		// attempt has a zero start time, which measures from the zero instant
		// and saturates time.Duration instead of timing the update.
		provider.RetryContextWithStartTime(
			updateIntermediariesRetryInfo,
			updateIntermediariesStartTime,
		),
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
		if err := sleepWithContext(ctx, time.Duration(waitTimeMS)*time.Millisecond); err != nil {
			deployCtx.Logger.Debug("context cancelled during link intermediary resources retry wait")
			return err
		}
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
	resolvedResource *provider.ResolvedResource,
) *provider.ResourceInfo {
	resourceState := getResourceStateByName(instanceState, resourceName)
	if resourceState == nil {
		return nil
	}

	return &provider.ResourceInfo{
		ResourceID:               resourceState.ResourceID,
		ResourceName:             resourceName,
		InstanceID:               instanceState.InstanceID,
		CurrentResourceState:     resourceState,
		ResourceWithResolvedSubs: resolvedResource,
	}
}

func getResolvedResourceFromInputChanges(
	inputChanges *changes.BlueprintChanges,
	resourceName string,
) *provider.ResolvedResource {
	if inputChanges == nil {
		return nil
	}
	if resourceChanges, ok := inputChanges.NewResources[resourceName]; ok {
		return resourceChanges.AppliedResourceInfo.ResourceWithResolvedSubs
	}
	if resourceChanges, ok := inputChanges.ResourceChanges[resourceName]; ok {
		return resourceChanges.AppliedResourceInfo.ResourceWithResolvedSubs
	}
	return nil
}

func createLinkDeployResult(
	linkedResourcesOutput *provider.LinkUpdateLinkedResourcesOutput,
	intermediaryResourcesOutput *provider.LinkUpdateIntermediaryResourcesOutput,
	contributionsOutput *provider.LinkProduceResourceContributionsOutput,
) *LinkDeployResult {
	linkedResourcesLinkData := getLinkedResourcesOutputLinkData(linkedResourcesOutput)
	intermediaryResourcesOutputLinkData := getIntermediaryResourcesOutputLinkData(
		intermediaryResourcesOutput,
	)
	linkedResourcesDataMappings := getLinkedResourcesOutputDataMappings(linkedResourcesOutput)
	intermediaryResourcesDataMappings := getIntermediaryResourcesOutputDataMappings(
		intermediaryResourcesOutput,
	)
	intermediaryResourceStates := getIntermediaryResourcesOutputStates(
		intermediaryResourcesOutput,
	)

	contributions := getProducedContributions(contributionsOutput)
	contributed := ContributionsToLinkData(contributions)

	return &LinkDeployResult{
		IntermediaryResourceStates: append(
			intermediaryResourceStates,
			getProducedIntermediaryStates(contributionsOutput)...,
		),
		LinkData: core.MergeMaps(
			linkedResourcesLinkData,
			intermediaryResourcesOutputLinkData,
			getProducedContributionsLinkData(contributionsOutput),
			&core.MappingNode{Fields: contributed.Data},
		),
		ResourceDataMappings: core.MergeNativeMaps(
			linkedResourcesDataMappings,
			intermediaryResourcesDataMappings,
			contributed.ResourceDataMappings,
		),
		Contributions:       contributions,
		ContributionRecords: contributed.ContributionRecords,
	}
}

func getProducedContributions(
	output *provider.LinkProduceResourceContributionsOutput,
) []*provider.ResourceContribution {
	if output == nil {
		return nil
	}

	return output.Contributions
}

func getProducedContributionsLinkData(
	output *provider.LinkProduceResourceContributionsOutput,
) *core.MappingNode {
	if output == nil {
		return nil
	}

	return output.LinkData
}

func getProducedIntermediaryStates(
	output *provider.LinkProduceResourceContributionsOutput,
) []*state.LinkIntermediaryResourceState {
	if output == nil {
		return nil
	}

	return output.IntermediaryResourceStates
}

func getLinkedResourcesOutputLinkData(output *provider.LinkUpdateLinkedResourcesOutput) *core.MappingNode {
	if output == nil {
		return nil
	}

	return output.LinkData
}

func getLinkedResourcesOutputDataMappings(
	output *provider.LinkUpdateLinkedResourcesOutput,
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
	linkedResourcesOutput *provider.LinkUpdateLinkedResourcesOutput
	contributionsOutput   *provider.LinkProduceResourceContributionsOutput
}
