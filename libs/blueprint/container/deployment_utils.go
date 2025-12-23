package container

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// sleepWithContext sleeps for the specified duration but can be interrupted
// by context cancellation. Returns the context error if cancelled, nil otherwise.
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// CreateDeployChannels creates a new DeployChannels struct that contains
// all the channels required to process deploy/destroy events.
func CreateDeployChannels() *DeployChannels {
	resourceUpdateChan := make(chan ResourceDeployUpdateMessage)
	childUpdateChan := make(chan ChildDeployUpdateMessage)
	linkUpdateChan := make(chan LinkDeployUpdateMessage)
	deploymentUpdateChan := make(chan DeploymentUpdateMessage)
	finishChan := make(chan DeploymentFinishedMessage)
	errChan := make(chan error)

	return &DeployChannels{
		ResourceUpdateChan:   resourceUpdateChan,
		ChildUpdateChan:      childUpdateChan,
		LinkUpdateChan:       linkUpdateChan,
		DeploymentUpdateChan: deploymentUpdateChan,
		FinishChan:           finishChan,
		ErrChan:              errChan,
	}
}

func isInstanceInProgress(instance *state.InstanceState, rollingBack bool) bool {
	if instance == nil {
		return false
	}

	if rollingBack {
		return instance.Status == core.InstanceStatusDeployRollingBack ||
			instance.Status == core.InstanceStatusUpdateRollingBack ||
			instance.Status == core.InstanceStatusDestroyRollingBack
	}

	return instance.Status == core.InstanceStatusDeploying ||
		instance.Status == core.InstanceStatusUpdating ||
		instance.Status == core.InstanceStatusDestroying
}

func determineResourceDestroyingStatus(rollingBack bool) core.ResourceStatus {
	if rollingBack {
		return core.ResourceStatusRollingBack
	}

	return core.ResourceStatusDestroying
}

func determinePreciseResourceDestroyingStatus(rollingBack bool) core.PreciseResourceStatus {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return core.PreciseResourceStatusCreateRollingBack
	}

	return core.PreciseResourceStatusDestroying
}

func determineResourceDestroyFailedStatus(rollingBack bool) core.ResourceStatus {
	if rollingBack {
		return core.ResourceStatusRollbackFailed
	}

	return core.ResourceStatusDestroyFailed
}

func determinePreciseResourceDestroyFailedStatus(rollingBack bool) core.PreciseResourceStatus {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return core.PreciseResourceStatusCreateRollbackFailed
	}

	return core.PreciseResourceStatusDestroyFailed
}

func determineResourceDestroyedStatus(rollingBack bool) core.ResourceStatus {
	if rollingBack {
		return core.ResourceStatusRollbackComplete
	}

	return core.ResourceStatusDestroyed
}

func determinePreciseResourceDestroyedStatus(rollingBack bool) core.PreciseResourceStatus {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return core.PreciseResourceStatusCreateRollbackComplete
	}

	return core.PreciseResourceStatusDestroyed
}

func determineInstanceDestroyFailedStatus(rollingBack bool) core.InstanceStatus {
	if rollingBack {
		return core.InstanceStatusDeployRollbackFailed
	}

	return core.InstanceStatusDestroyFailed
}

func emptyChangesDeployFailedMessage(rollingBack bool) string {
	if rollingBack {
		return "an empty set of changes was provided for " +
			"re-deploying the blueprint when rolling back a destroy operation"
	}

	return "an empty set of changes was provided for deployment"
}

func emptyChangesDestroyFailedMessage(rollingBack bool) string {
	if rollingBack {
		return "an empty set of changes was provided for " +
			"destroying the blueprint when rolling back a deploy operation"
	}

	return "an empty set of changes was provided " +
		"for the blueprint instance to be destroyed"
}

func instanceInProgressDeployFailedMessage(instanceID string, rollingBack bool) string {
	if rollingBack {
		return fmt.Sprintf(
			"a rollback is already in progress for the blueprint instance (%s)",
			instanceID,
		)
	}

	return fmt.Sprintf(
		"a deployment or removal is already in progress for the blueprint instance (%s)",
		instanceID,
	)
}

func determineInstanceDestroyedStatus(rollingBack bool) core.InstanceStatus {
	if rollingBack {
		return core.InstanceStatusDeployRollbackComplete
	}

	return core.InstanceStatusDestroyed
}

func determineInstanceDeployFailedStatus(rollingBack bool, newInstance bool) core.InstanceStatus {
	if rollingBack && newInstance {
		return core.InstanceStatusDestroyRollbackFailed
	}

	if !newInstance {
		return core.InstanceStatusUpdateFailed
	}

	return core.InstanceStatusDeployFailed
}

func determineInstanceDestroyingStatus(rollingBack bool) core.InstanceStatus {
	if rollingBack {
		// If the context is destroying an instance as a part of the rollback
		// process, the status should be rolling back the original deployment.
		return core.InstanceStatusDeployRollingBack
	}

	return core.InstanceStatusDestroying
}

func determineInstanceDeployingStatus(rollingBack bool, newInstance bool) core.InstanceStatus {
	if rollingBack && newInstance {
		// If the context is deploying an instance as a part of the rollback
		// process, the status should be rolling back the destruction of the
		// instance.
		return core.InstanceStatusDestroyRollingBack
	}

	if rollingBack && !newInstance {
		// If the context is re-deploying an existing instance as a part of the rollback
		// process, the status should be rolling back the original deployment.
		return core.InstanceStatusUpdateRollingBack
	}

	if !newInstance {
		return core.InstanceStatusUpdating
	}

	return core.InstanceStatusDeploying
}

func determineInstanceDeployedStatus(rollingBack bool, newInstance bool) core.InstanceStatus {
	if rollingBack && newInstance {
		// If the context is deploying an instance as a part of the rollback
		// process, the status should be rolling back the destruction of the
		// instance.
		return core.InstanceStatusDestroyRollbackComplete
	}

	if rollingBack && !newInstance {
		// If the context is re-deploying an existing instance as a part of the rollback
		// process, the status should be rolling back the original deployment.
		return core.InstanceStatusUpdateRollbackComplete
	}

	if !newInstance {
		return core.InstanceStatusUpdated
	}

	return core.InstanceStatusDeployed
}

func determineResourceDeployingStatus(rollingBack bool, newResource bool) core.ResourceStatus {
	if rollingBack {
		// If the context is deploying a resource as a part of the rollback
		// process, the general status should be to roll back.
		return core.ResourceStatusRollingBack
	}

	if !newResource {
		return core.ResourceStatusUpdating
	}

	return core.ResourceStatusCreating
}

func determinePreciseResourceDeployingStatus(rollingBack bool, newResource bool) core.PreciseResourceStatus {
	if rollingBack && newResource {
		// If the context is deploying a new resource as a part of the rollback
		// process, the status should be rolling back the destruction of the
		// resource.
		return core.PreciseResourceStatusDestroyRollingBack
	}

	if rollingBack && !newResource {
		// If the context is re-deploying an existing resource as a part of the rollback
		// process, the status should be rolling back the original deployment.
		return core.PreciseResourceStatusUpdateRollingBack
	}

	if !newResource {
		return core.PreciseResourceStatusUpdating
	}

	return core.PreciseResourceStatusCreating
}

func determineResourceDeployFailedStatus(rollingBack bool, newResource bool) core.ResourceStatus {
	if rollingBack {
		// If the context is deploying a resource as a part of the rollback
		// process, the status should be a failure to rollback the original
		// resource deployment.
		return core.ResourceStatusRollbackFailed
	}

	if !newResource {
		return core.ResourceStatusUpdateFailed
	}

	return core.ResourceStatusCreateFailed
}

func determinePreciseResourceDeployFailedStatus(rollingBack bool, newResource bool) core.PreciseResourceStatus {
	if rollingBack && newResource {
		// If the context is deploying a new resource as a part of the rollback
		// process, the status should be a failure to rollback the original
		// resource deployment.
		return core.PreciseResourceStatusDestroyRollbackFailed
	}

	if rollingBack && !newResource {
		// If the context is re-deploying an existing resource as a part of the rollback
		// process, the status should be a failure to rollback the original resource
		// deployment.
		return core.PreciseResourceStatusUpdateRollbackFailed
	}

	if !newResource {
		return core.PreciseResourceStatusUpdateFailed
	}

	return core.PreciseResourceStatusCreateFailed
}

func determineResourceConfigCompleteStatus(rollingBack bool, newResource bool) core.ResourceStatus {
	if rollingBack {
		// If the context is deploying a resource as a part of the rollback
		// process, the status should be rolling back as config completion
		// is still in progress, waiting for the resource to be stabilised.
		return core.ResourceStatusRollingBack
	}

	if !newResource {
		return core.ResourceStatusUpdating
	}

	return core.ResourceStatusCreating
}

func determinePreciseResourceConfigCompleteStatus(rollingBack bool, newResource bool) core.PreciseResourceStatus {
	if rollingBack && newResource {
		// If the context is deploying a new resource as a part of the rollback
		// process, the status should be the config complete state for a destruction
		// rollback.
		return core.PreciseResourceStatusDestroyRollbackConfigComplete
	}

	if rollingBack && !newResource {
		// If the context is re-deploying an existing resource as a part of the rollback
		// process, the status should be the config complete state for an update rollback.
		return core.PreciseResourceStatusUpdateRollbackConfigComplete
	}

	return core.PreciseResourceStatusConfigComplete
}

func determineResourceDeployedStatus(rollingBack bool, newResource bool) core.ResourceStatus {
	if rollingBack {
		return core.ResourceStatusRollbackComplete
	}

	if !newResource {
		return core.ResourceStatusUpdated
	}

	return core.ResourceStatusCreated
}

func determinePreciseResourceDeployedStatus(rollingBack bool, newResource bool) core.PreciseResourceStatus {
	if rollingBack && newResource {
		// If the context is deploying a new resource as a part of the rollback
		// process, the status should be the rollback complete state for a destruction
		// rollback.
		return core.PreciseResourceStatusDestroyRollbackComplete
	}

	if rollingBack && !newResource {
		// If the context is re-deploying an existing resource as a part of the rollback
		// process, the status should be the rollback complete state for an update rollback.
		return core.PreciseResourceStatusUpdateRollbackComplete
	}

	if !newResource {
		return core.PreciseResourceStatusUpdated
	}

	return core.PreciseResourceStatusCreated
}

func determineLinkUpdatingStatus(
	rollingBack bool,
	linkUpdateType provider.LinkUpdateType,
) core.LinkStatus {
	if rollingBack {
		return determineLinkUpdatingRollbackStatus(linkUpdateType)
	}

	if linkUpdateType == provider.LinkUpdateTypeCreate {
		return core.LinkStatusCreating
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		return core.LinkStatusUpdating
	}

	return core.LinkStatusDestroying
}

func determineLinkOperationSuccessfullyFinishedStatus(
	rollingBack bool,
	linkUpdateType provider.LinkUpdateType,
) core.LinkStatus {
	if rollingBack {
		return determineLinkRollbackFinishedStatus(linkUpdateType)
	}

	if linkUpdateType == provider.LinkUpdateTypeCreate {
		return core.LinkStatusCreated
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		return core.LinkStatusUpdated
	}

	return core.LinkStatusDestroyed
}

func determineLinkRollbackFinishedStatus(linkUpdateType provider.LinkUpdateType) core.LinkStatus {
	if linkUpdateType == provider.LinkUpdateTypeCreate {
		// A create update type in the context of rolling back
		// is to reverse the removal of a link.
		return core.LinkStatusDestroyRollbackComplete
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		// An update type in the context of rolling back
		// is to reverse the changes made to a link.
		return core.LinkStatusUpdateRollbackComplete
	}

	// A destroy update type in the context of rolling back
	// is to reverse the creation of a link.
	return core.LinkStatusCreateRollbackComplete
}

func determineLinkUpdatingRollbackStatus(linkUpdateType provider.LinkUpdateType) core.LinkStatus {
	if linkUpdateType == provider.LinkUpdateTypeCreate {
		// A create update type in the context of rolling back
		// is to reverse the removal of a link.
		return core.LinkStatusDestroyRollingBack
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		// An update type in the context of rolling back
		// is to reverse the changes made to a link.
		return core.LinkStatusUpdateRollingBack
	}

	// A destroy update type in the context of rolling back
	// is to reverse the creation of a link.
	return core.LinkStatusCreateRollingBack
}

func determineLinkUpdateFailedStatus(
	rollingBack bool,
	linkUpdateType provider.LinkUpdateType,
) core.LinkStatus {
	if rollingBack {
		return determineLinkUpdateRollbackFailedStatus(linkUpdateType)
	}

	if linkUpdateType == provider.LinkUpdateTypeCreate {
		return core.LinkStatusCreateFailed
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		return core.LinkStatusUpdateFailed
	}

	return core.LinkStatusDestroyFailed
}

func determineLinkUpdateRollbackFailedStatus(
	linkUpdateType provider.LinkUpdateType,
) core.LinkStatus {
	if linkUpdateType == provider.LinkUpdateTypeCreate {
		// A create update type in the context of rolling back
		// is to reverse the removal of a link.
		return core.LinkStatusDestroyRollbackFailed
	}

	if linkUpdateType == provider.LinkUpdateTypeUpdate {
		// An update type in the context of rolling back
		// is to reverse the changes made to a link.
		return core.LinkStatusUpdateRollbackFailed
	}

	// A destroy update type in the context of rolling back
	// is to reverse the creation of a link.
	return core.LinkStatusCreateRollbackFailed
}

func determinePreciseLinkUpdatingResourceAStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource A in the context of rolling back is to roll back
		// the latest changes made to resource A specific to the current link.
		return core.PreciseLinkStatusResourceAUpdateRollingBack
	}

	return core.PreciseLinkStatusUpdatingResourceA
}

func determinePreciseLinkUpdatingResourceBStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource B in the context of rolling back is to roll back
		// the latest changes made to resource B specific to the current link.
		return core.PreciseLinkStatusResourceBUpdateRollingBack
	}

	return core.PreciseLinkStatusUpdatingResourceB
}

func determinePreciseLinkResourceAUpdateFailedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource A in the context of rolling back is to roll back
		// the latest changes made to resource A specific to the current link.
		return core.PreciseLinkStatusResourceAUpdateRollbackFailed
	}

	return core.PreciseLinkStatusResourceAUpdateFailed
}

func determinePreciseLinkResourceAUpdatedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource A in the context of rolling back is to roll back
		// the latest changes made to resource A specific to the current link.
		return core.PreciseLinkStatusResourceAUpdateRollbackComplete
	}

	return core.PreciseLinkStatusResourceAUpdated
}

func determinePreciseLinkResourceBUpdatedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource B in the context of rolling back is to roll back
		// the latest changes made to resource B specific to the current link.
		return core.PreciseLinkStatusResourceBUpdateRollbackComplete
	}

	return core.PreciseLinkStatusResourceBUpdated
}

func determinePreciseLinkResourceBUpdateFailedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating resource B in the context of rolling back is to roll back
		// the latest changes made to resource B specific to the current link.
		return core.PreciseLinkStatusResourceBUpdateRollbackFailed
	}

	return core.PreciseLinkStatusResourceBUpdateFailed
}

func determinePreciseLinkUpdatingIntermediariesStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating intermediary resources in the context of rolling back is to roll back
		// the latest changes made to intermediary resources specific to the current link.
		return core.PreciseLinkStatusIntermediaryResourceUpdateRollingBack
	}

	return core.PreciseLinkStatusUpdatingIntermediaryResources
}

func determinePreciseLinkIntermediariesUpdateFailedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating intermediary resources in the context of rolling back is to roll back
		// the latest changes made to intermediary resources specific to the current link.
		return core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed
	}

	return core.PreciseLinkStatusIntermediaryResourceUpdateFailed
}

func determinePreciseLinkIntermediariesUpdatedStatus(rollingBack bool) core.PreciseLinkStatus {
	if rollingBack {
		// Updating intermediary resources in the context of rolling back is to roll back
		// the latest changes made to intermediary resources specific to the current link.
		return core.PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete
	}

	return core.PreciseLinkStatusIntermediaryResourcesUpdated
}

func determineLinkUpdateIntermediariesRetryFailureDurations(
	currentRetryInfo *provider.RetryContext,
) *state.LinkCompletionDurations {
	if currentRetryInfo.ExceededMaxRetries {
		totalDuration := core.Sum(currentRetryInfo.AttemptDurations)
		return &state.LinkCompletionDurations{
			IntermediaryResources: &state.LinkComponentCompletionDurations{
				TotalDuration:    &totalDuration,
				AttemptDurations: currentRetryInfo.AttemptDurations,
			},
		}
	}

	return &state.LinkCompletionDurations{
		IntermediaryResources: &state.LinkComponentCompletionDurations{
			AttemptDurations: currentRetryInfo.AttemptDurations,
		},
	}
}

func determineFinishedFailureStatus(destroyingInstance bool, rollingBack bool) core.InstanceStatus {
	if destroyingInstance {
		if rollingBack {
			// If the context is destroying an instance as a part of the rollback
			// process, the status should be a failure to rollback the original
			// deployment.
			return core.InstanceStatusDeployRollbackFailed
		}

		return core.InstanceStatusDestroyFailed
	}

	if rollingBack {
		// If the context is deploying an instance as a part of the rollback
		// process, the status should be a failure to rollback destroying the
		// instance.
		return core.InstanceStatusDestroyRollbackFailed
	}

	return core.InstanceStatusDeployFailed
}

func finishedFailureMessages(deployCtx *DeployContext, failedElements []string) []string {
	operation := determineOperation(deployCtx)
	messages := []string{
		fmt.Sprintf(
			"failed to %s the blueprint instance due to %d %s",
			operation,
			len(failedElements),
			pluralise("failure", "failures", len(failedElements)),
		),
	}
	messageTemplate := "failed to %s %q"
	for _, elementName := range failedElements {
		messages = append(messages, fmt.Sprintf(messageTemplate, operation, elementName))
	}

	return messages
}

func determineOperation(deployCtx *DeployContext) string {
	if deployCtx.Destroying && !deployCtx.Rollback {
		return "destroy"
	}

	if deployCtx.Destroying && deployCtx.Rollback {
		return "roll back deployment of"
	}

	if !deployCtx.Destroying && deployCtx.Rollback {
		return "roll back destruction of"
	}

	return "deploy"
}

func checkDeploymentForNewInstance(input *DeployInput, newIDGenerated bool) (bool, error) {
	if input.Changes == nil {
		return newIDGenerated, nil
	}

	hasExistingResourceChanges := len(input.Changes.ResourceChanges) > 0 ||
		len(input.Changes.RemovedResources) > 0

	hasExistingChildChanges := len(input.Changes.ChildChanges) > 0 ||
		len(input.Changes.RemovedChildren) > 0 ||
		len(input.Changes.RecreateChildren) > 0

	if newIDGenerated && (hasExistingResourceChanges || hasExistingChildChanges) {
		return false, errInstanceIDRequiredForChanges()
	}

	isForNewInstance := newIDGenerated &&
		!hasExistingResourceChanges &&
		!hasExistingChildChanges

	return isForNewInstance, nil
}

func isResourceDestroyEvent(preciseStatus core.PreciseResourceStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return preciseStatus == core.PreciseResourceStatusCreateRollingBack ||
			preciseStatus == core.PreciseResourceStatusCreateRollbackComplete ||
			preciseStatus == core.PreciseResourceStatusCreateRollbackFailed
	}

	return preciseStatus == core.PreciseResourceStatusDestroying ||
		preciseStatus == core.PreciseResourceStatusDestroyed ||
		preciseStatus == core.PreciseResourceStatusDestroyFailed
}

func isChildDestroyEvent(instanceStatus core.InstanceStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, destroying a child instance is to roll back
		// the deployment of the child instance.
		return instanceStatus == core.InstanceStatusDeployRollingBack ||
			instanceStatus == core.InstanceStatusDeployRollbackComplete ||
			instanceStatus == core.InstanceStatusDeployRollbackFailed
	}

	return instanceStatus == core.InstanceStatusDestroying ||
		instanceStatus == core.InstanceStatusDestroyed ||
		instanceStatus == core.InstanceStatusDestroyFailed
}

func isLinkDestroyEvent(linkStatus core.LinkStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, destroying a link is to roll back
		// the creation of the link.
		return linkStatus == core.LinkStatusCreateRollingBack ||
			linkStatus == core.LinkStatusCreateRollbackComplete ||
			linkStatus == core.LinkStatusCreateRollbackFailed
	}

	return linkStatus == core.LinkStatusDestroying ||
		linkStatus == core.LinkStatusDestroyed ||
		linkStatus == core.LinkStatusDestroyFailed
}

func isResourceUpdateEvent(preciseStatus core.PreciseResourceStatus, rollingBack bool) bool {
	if rollingBack {
		return preciseStatus == core.PreciseResourceStatusUpdateRollingBack ||
			preciseStatus == core.PreciseResourceStatusUpdateRollbackComplete ||
			preciseStatus == core.PreciseResourceStatusUpdateRollbackFailed ||
			preciseStatus == core.PreciseResourceStatusUpdateRollbackConfigComplete
	}

	return preciseStatus == core.PreciseResourceStatusUpdating ||
		preciseStatus == core.PreciseResourceStatusUpdated ||
		preciseStatus == core.PreciseResourceStatusUpdateFailed ||
		preciseStatus == core.PreciseResourceStatusUpdateConfigComplete
}

func isResourceCreationEvent(preciseStatus core.PreciseResourceStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, creating a resource is to roll back
		// the destruction of the resource.
		return preciseStatus == core.PreciseResourceStatusDestroyRollingBack ||
			preciseStatus == core.PreciseResourceStatusDestroyRollbackComplete ||
			preciseStatus == core.PreciseResourceStatusDestroyRollbackFailed ||
			preciseStatus == core.PreciseResourceStatusDestroyRollbackConfigComplete
	}

	return preciseStatus == core.PreciseResourceStatusCreating ||
		preciseStatus == core.PreciseResourceStatusCreated ||
		preciseStatus == core.PreciseResourceStatusCreateFailed ||
		preciseStatus == core.PreciseResourceStatusConfigComplete
}

func isChildUpdateEvent(status core.InstanceStatus, rollingBack bool) bool {
	if rollingBack {
		return status == core.InstanceStatusUpdateRollingBack ||
			status == core.InstanceStatusUpdateRollbackComplete ||
			status == core.InstanceStatusUpdateRollbackFailed
	}

	return status == core.InstanceStatusUpdating ||
		status == core.InstanceStatusUpdated ||
		status == core.InstanceStatusUpdateFailed
}

func isChildDeployEvent(status core.InstanceStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, deploying a child instance is to roll back
		// the destruction of the child instance.
		return status == core.InstanceStatusDestroyRollingBack ||
			status == core.InstanceStatusDestroyRollbackComplete ||
			status == core.InstanceStatusDestroyRollbackFailed
	}

	return status == core.InstanceStatusDeploying ||
		status == core.InstanceStatusDeployed ||
		status == core.InstanceStatusDeployFailed
}

func isLinkUpdateEvent(status core.LinkStatus, rollingBack bool) bool {
	if rollingBack {
		return status == core.LinkStatusUpdateRollingBack ||
			status == core.LinkStatusUpdateRollbackComplete ||
			status == core.LinkStatusUpdateRollbackFailed
	}

	return status == core.LinkStatusUpdating ||
		status == core.LinkStatusUpdated ||
		status == core.LinkStatusUpdateFailed
}

func isLinkCreationEvent(status core.LinkStatus, rollingBack bool) bool {
	if rollingBack {
		// In the context of rolling back, creating a link is to roll back
		// the destruction of the link.
		return status == core.LinkStatusDestroyRollingBack ||
			status == core.LinkStatusDestroyRollbackComplete ||
			status == core.LinkStatusDestroyRollbackFailed
	}

	return status == core.LinkStatusCreating ||
		status == core.LinkStatusCreated ||
		status == core.LinkStatusCreateFailed
}

func startedDestroyingResource(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return preciseStatus == core.PreciseResourceStatusCreateRollingBack
	}

	return preciseStatus == core.PreciseResourceStatusDestroying
}

func startedUpdatingResource(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return preciseStatus == core.PreciseResourceStatusUpdateRollingBack
	}

	return preciseStatus == core.PreciseResourceStatusUpdating
}

func resourceUpdateConfigComplete(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return preciseStatus == core.PreciseResourceStatusUpdateRollbackConfigComplete
	}

	return preciseStatus == core.PreciseResourceStatusUpdateConfigComplete
}

// This function should be used when the caller needs to know if the resource has
// finished updating, regardless of success or failure.
// This will report false if the current message represents a failure that can be retried.
func finishedUpdatingResource(
	msg ResourceDeployUpdateMessage,
	rollingBack bool,
) bool {
	if rollingBack {
		finishedRollingBack := msg.PreciseStatus == core.PreciseResourceStatusUpdateRollbackComplete ||
			msg.PreciseStatus == core.PreciseResourceStatusUpdateRollbackFailed

		return finishedRollingBack && !msg.CanRetry
	}

	finishedUpdating := msg.PreciseStatus == core.PreciseResourceStatusUpdated ||
		msg.PreciseStatus == core.PreciseResourceStatusUpdateFailed

	return finishedUpdating && !msg.CanRetry
}

func startedCreatingResource(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return preciseStatus == core.PreciseResourceStatusCreateRollingBack
	}

	return preciseStatus == core.PreciseResourceStatusCreating
}

func resourceCreationConfigComplete(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return preciseStatus == core.PreciseResourceStatusDestroyRollbackConfigComplete
	}

	return preciseStatus == core.PreciseResourceStatusConfigComplete
}

// This function should be used when the caller needs to know if resource
// creation has finished, regardless of success or failure.
// This will report false if the current message represents a failure that can be retried.
func finishedCreatingResource(
	msg ResourceDeployUpdateMessage,
	rollingBack bool,
) bool {
	if rollingBack {
		finishedRollingBack := msg.PreciseStatus == core.PreciseResourceStatusDestroyRollbackComplete ||
			msg.PreciseStatus == core.PreciseResourceStatusDestroyRollbackFailed

		return finishedRollingBack && !msg.CanRetry
	}

	finishedCreation := msg.PreciseStatus == core.PreciseResourceStatusCreated ||
		msg.PreciseStatus == core.PreciseResourceStatusCreateFailed

	return finishedCreation && !msg.CanRetry
}

func finishedUpdatingChild(
	status core.InstanceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return status == core.InstanceStatusUpdateRollbackComplete ||
			status == core.InstanceStatusUpdateRollbackFailed
	}

	return status == core.InstanceStatusUpdated ||
		status == core.InstanceStatusUpdateFailed
}

func finishedDeployingChild(
	status core.InstanceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		return status == core.InstanceStatusDestroyRollbackComplete ||
			status == core.InstanceStatusDestroyRollbackFailed
	}

	return status == core.InstanceStatusDeployed ||
		status == core.InstanceStatusDeployFailed
}

func startedDestroyingChild(
	status core.InstanceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a child instance is to roll back
		// the deployment of the child instance.
		return status == core.InstanceStatusDeployRollingBack
	}

	return status == core.InstanceStatusDestroying
}

func startedUpdatingLink(
	status core.LinkStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, updating a link is to roll back
		// the latest changes made to the link.
		return status == core.LinkStatusUpdateRollingBack
	}

	return status == core.LinkStatusUpdating
}

// This function should be used when the caller needs to know if the link has
// finished updating, regardless of success or failure.
// This will report false if the current message represents a failure that can be retried.
func finishedUpdatingLink(
	msg LinkDeployUpdateMessage,
	rollingBack bool,
) bool {
	if rollingBack {
		rollbackFinished := msg.Status == core.LinkStatusUpdateRollbackComplete ||
			msg.Status == core.LinkStatusUpdateRollbackFailed

		return rollbackFinished && !msg.CanRetryCurrentStage
	}

	updateFinished := msg.Status == core.LinkStatusUpdated ||
		msg.Status == core.LinkStatusUpdateFailed

	return updateFinished && !msg.CanRetryCurrentStage
}

func startedCreatingLink(
	status core.LinkStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, creating a link is to roll back
		// the destruction of the link.
		return status == core.LinkStatusDestroyRollingBack
	}

	return status == core.LinkStatusCreating
}

// This function should be used when the caller needs to know if creation of the link
// has finished, regardless of success or failure.
// This will report false if the current message represents a failure that can be retried.
func finishedCreatingLink(
	msg LinkDeployUpdateMessage,
	rollingBack bool,
) bool {
	if rollingBack {
		rollbackFinished := msg.Status == core.LinkStatusDestroyRollbackComplete ||
			msg.Status == core.LinkStatusDestroyRollbackFailed

		return rollbackFinished && !msg.CanRetryCurrentStage
	}

	creationFinished := msg.Status == core.LinkStatusCreated ||
		msg.Status == core.LinkStatusCreateFailed

	return creationFinished && !msg.CanRetryCurrentStage
}

func startedDestroyingLink(
	status core.LinkStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a link is to roll back
		// the creation of the link.
		return status == core.LinkStatusCreateRollingBack
	}

	return status == core.LinkStatusDestroying
}

func finishedDestroyingResource(
	msg ResourceDeployUpdateMessage,
	rollingBack bool,
) bool {

	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		rollbackFinished := msg.PreciseStatus == core.PreciseResourceStatusCreateRollbackComplete ||
			msg.PreciseStatus == core.PreciseResourceStatusCreateRollbackFailed
		return rollbackFinished && !msg.CanRetry
	}

	destroyFinished := msg.PreciseStatus == core.PreciseResourceStatusDestroyed ||
		msg.PreciseStatus == core.PreciseResourceStatusDestroyFailed

	return destroyFinished && !msg.CanRetry
}

func finishedDestroyingChild(
	msg ChildDeployUpdateMessage,
	rollingBack bool,
) bool {

	if rollingBack {
		// In the context of rolling back, destroying a child is to roll back
		// the deployment of the child instance.
		return msg.Status == core.InstanceStatusDeployRollbackComplete ||
			msg.Status == core.InstanceStatusDeployRollbackFailed
	}

	return msg.Status == core.InstanceStatusDestroyed ||
		msg.Status == core.InstanceStatusDestroyFailed
}

func finishedDestroyingLink(
	msg LinkDeployUpdateMessage,
	rollingBack bool,
) bool {

	if rollingBack {
		// In the context of rolling back, destroying a link is to roll back
		// the creation of the link.
		rollbackFinished := msg.Status == core.LinkStatusCreateRollbackComplete ||
			msg.Status == core.LinkStatusCreateRollbackFailed
		return rollbackFinished && !msg.CanRetryCurrentStage
	}

	destroyFinshed := msg.Status == core.LinkStatusDestroyed ||
		msg.Status == core.LinkStatusDestroyFailed

	return destroyFinshed && !msg.CanRetryCurrentStage
}

func wasResourceDestroyedSuccessfully(
	preciseStatus core.PreciseResourceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a resource is to roll back
		// the creation of the resource.
		return preciseStatus == core.PreciseResourceStatusCreateRollbackComplete
	}

	return preciseStatus == core.PreciseResourceStatusDestroyed
}

func wasChildDestroyedSuccessfully(
	status core.InstanceStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a child is to roll back
		// the deployment of the child instance.
		return status == core.InstanceStatusDeployRollbackComplete
	}

	return status == core.InstanceStatusDestroyed
}

func wasLinkDestroyedSuccessfully(
	status core.LinkStatus,
	rollingBack bool,
) bool {
	if rollingBack {
		// In the context of rolling back, destroying a link is to roll back
		// the creation of the link.
		return status == core.LinkStatusCreateRollbackComplete
	}

	return status == core.LinkStatusDestroyed
}

func wasDeploymentSuccessful(
	msg *DeploymentFinishedMessage,
	rollingBack bool,
	isNew bool,
) bool {
	if rollingBack && isNew {
		// If the context is deploying a new blueprint instance as a part of the rollback
		// process, the status should be a successful rollback of the destruction
		// of the instance.
		return msg.Status == core.InstanceStatusDestroyRollbackComplete
	}

	if rollingBack && !isNew {
		// If the context is deploying updates for an existing blueprint instance
		// as a part of the rollback process, the status should be a successful
		// rollback of the latest changes made to the instance.
		return msg.Status == core.InstanceStatusUpdateRollbackComplete
	}

	if !isNew {
		return msg.Status == core.InstanceStatusUpdated
	}

	return msg.Status == core.InstanceStatusDeployed
}

func determineResourceRetryFailureDurations(
	currentRetryInfo *provider.RetryContext,
) *state.ResourceCompletionDurations {
	if currentRetryInfo.ExceededMaxRetries {
		totalDuration := core.Sum(currentRetryInfo.AttemptDurations)
		return &state.ResourceCompletionDurations{
			TotalDuration:    &totalDuration,
			AttemptDurations: currentRetryInfo.AttemptDurations,
		}
	}

	return &state.ResourceCompletionDurations{
		AttemptDurations: currentRetryInfo.AttemptDurations,
	}
}

func determineResourceDeployConfigCompleteDurations(
	currentRetryInfo *provider.RetryContext,
	currentAttemptDuration time.Duration,
) *state.ResourceCompletionDurations {
	updatedAttemptDurations := append(
		currentRetryInfo.AttemptDurations,
		core.FractionalMilliseconds(currentAttemptDuration),
	)

	configCompleteDurations := core.Sum(updatedAttemptDurations)

	configCompleteDurationPtr := &configCompleteDurations
	return &state.ResourceCompletionDurations{
		ConfigCompleteDuration: configCompleteDurationPtr,
		AttemptDurations:       updatedAttemptDurations,
	}
}

// This only applies to reaching the "config complete" stage for deployments
// and is used for resource destruction durations.
// This is not used to calculate the final durations once a resource has been
// confirmed to be stable.
func determineResourceDeployFinishedDurations(
	currentRetryInfo *provider.RetryContext,
	currentAttemptDuration time.Duration,
	configCompleteDuration *time.Duration,
) *state.ResourceCompletionDurations {
	updatedAttemptDurations := append(
		currentRetryInfo.AttemptDurations,
		core.FractionalMilliseconds(currentAttemptDuration),
	)

	configCompleteDurationPtr := (*float64)(nil)
	if configCompleteDuration != nil {
		configCompleteMS := core.FractionalMilliseconds(*configCompleteDuration)
		configCompleteDurationPtr = &configCompleteMS
	}

	totalDuration := core.Sum(updatedAttemptDurations)
	return &state.ResourceCompletionDurations{
		ConfigCompleteDuration: configCompleteDurationPtr,
		TotalDuration:          &totalDuration,
		AttemptDurations:       updatedAttemptDurations,
	}
}

func addTotalToResourceCompletionDurations(
	durations *state.ResourceCompletionDurations,
	stabilisePollingDuration time.Duration,
) *state.ResourceCompletionDurations {
	stabilisePollingMS := core.FractionalMilliseconds(stabilisePollingDuration)
	totalDuration := core.Sum(durations.AttemptDurations) + stabilisePollingMS
	totalDurationPtr := &totalDuration
	return &state.ResourceCompletionDurations{
		TotalDuration:          totalDurationPtr,
		ConfigCompleteDuration: durations.ConfigCompleteDuration,
		AttemptDurations:       durations.AttemptDurations,
	}
}

func determineLinkUpdateResourceARetryFailureDurations(
	currentRetryInfo *provider.RetryContext,
) *state.LinkCompletionDurations {
	if currentRetryInfo.ExceededMaxRetries {
		totalDuration := core.Sum(currentRetryInfo.AttemptDurations)
		return &state.LinkCompletionDurations{
			ResourceAUpdate: &state.LinkComponentCompletionDurations{
				TotalDuration:    &totalDuration,
				AttemptDurations: currentRetryInfo.AttemptDurations,
			},
		}
	}

	return &state.LinkCompletionDurations{
		ResourceAUpdate: &state.LinkComponentCompletionDurations{
			AttemptDurations: currentRetryInfo.AttemptDurations,
		},
	}
}

func determineLinkUpdateResourceAFinishedDurations(
	currentRetryInfo *provider.RetryContext,
	currentAttemptDuration time.Duration,
) *state.LinkCompletionDurations {
	updatedAttemptDurations := append(
		currentRetryInfo.AttemptDurations,
		core.FractionalMilliseconds(currentAttemptDuration),
	)
	totalDuration := core.Sum(updatedAttemptDurations)
	return &state.LinkCompletionDurations{
		ResourceAUpdate: &state.LinkComponentCompletionDurations{
			TotalDuration:    &totalDuration,
			AttemptDurations: updatedAttemptDurations,
		},
	}
}

func determineLinkUpdateResourceBRetryFailureDurations(
	currentRetryInfo *provider.RetryContext,
) *state.LinkCompletionDurations {
	if currentRetryInfo.ExceededMaxRetries {
		totalDuration := core.Sum(currentRetryInfo.AttemptDurations)
		return &state.LinkCompletionDurations{
			ResourceBUpdate: &state.LinkComponentCompletionDurations{
				TotalDuration:    &totalDuration,
				AttemptDurations: currentRetryInfo.AttemptDurations,
			},
		}
	}

	return &state.LinkCompletionDurations{
		ResourceBUpdate: &state.LinkComponentCompletionDurations{
			AttemptDurations: currentRetryInfo.AttemptDurations,
		},
	}
}

func determineLinkUpdateResourceBFinishedDurations(
	currentRetryInfo *provider.RetryContext,
	currentAttemptDuration time.Duration,
	accumDurationInfo *state.LinkCompletionDurations,
) *state.LinkCompletionDurations {
	updatedAttemptDurations := append(
		currentRetryInfo.AttemptDurations,
		core.FractionalMilliseconds(currentAttemptDuration),
	)
	totalDuration := core.Sum(updatedAttemptDurations)
	durationInfo := accumDurationInfo
	if durationInfo == nil {
		durationInfo = &state.LinkCompletionDurations{}
	}
	durationInfo.ResourceBUpdate = &state.LinkComponentCompletionDurations{
		TotalDuration:    &totalDuration,
		AttemptDurations: updatedAttemptDurations,
	}
	return durationInfo
}

func determineLinkUpdateIntermediariesFinishedDurations(
	currentRetryInfo *provider.RetryContext,
	currentAttemptDuration time.Duration,
	accumDurationInfo *state.LinkCompletionDurations,
) *state.LinkCompletionDurations {
	updatedAttemptDurations := append(
		currentRetryInfo.AttemptDurations,
		core.FractionalMilliseconds(currentAttemptDuration),
	)
	stageTotalDuration := core.Sum(updatedAttemptDurations)
	durationInfo := accumDurationInfo
	if durationInfo == nil {
		durationInfo = &state.LinkCompletionDurations{}
	}
	durationInfo.IntermediaryResources = &state.LinkComponentCompletionDurations{
		TotalDuration:    &stageTotalDuration,
		AttemptDurations: updatedAttemptDurations,
	}
	totalDuration := sumLinkComponentCompletionDurations(durationInfo)
	durationInfo.TotalDuration = &totalDuration
	return durationInfo
}

func sumLinkComponentCompletionDurations(
	durations *state.LinkCompletionDurations,
) float64 {
	return getLinkComponentTotalDuration(durations.ResourceAUpdate) +
		getLinkComponentTotalDuration(durations.ResourceBUpdate) +
		getLinkComponentTotalDuration(durations.IntermediaryResources)
}

func getLinkComponentTotalDuration(
	componentDurations *state.LinkComponentCompletionDurations,
) float64 {
	if componentDurations == nil {
		return 0
	}

	if componentDurations.TotalDuration == nil {
		return 0
	}

	return *componentDurations.TotalDuration
}

func createDestroyChangesFromChildState(
	childState *state.InstanceState,
) *changes.BlueprintChanges {
	changes := &changes.BlueprintChanges{
		RemovedResources: []string{},
		RemovedLinks:     []string{},
		RemovedChildren:  []string{},
		RemovedExports:   []string{},
	}

	for _, resource := range childState.Resources {
		changes.RemovedResources = append(changes.RemovedResources, resource.Name)
	}

	for _, link := range childState.Links {
		changes.RemovedLinks = append(changes.RemovedLinks, link.Name)
	}

	for childName := range childState.ChildBlueprints {
		changes.RemovedChildren = append(changes.RemovedChildren, childName)
	}

	for exportName := range childState.Exports {
		changes.RemovedExports = append(changes.RemovedExports, exportName)
	}

	return changes
}

func getFailedRemovalsAndUpdateState(
	finished map[string]*deployUpdateMessageWrapper,
	group []state.Element,
	state DeploymentState,
	rollback bool,
) []string {
	failed := []string{}

	for _, element := range group {
		elementName := getNamespacedLogicalName(element)
		if msgWrapper, ok := finished[elementName]; ok {
			if removalWasSuccessful(msgWrapper, rollback) {
				state.SetDestroyedElement(element)
			} else {
				failed = append(failed, elementName)
			}
		} else {
			failed = append(failed, elementName)
		}
	}

	return failed
}

func getFailedElementDeploymentsAndUpdateState(
	finished map[string]*deployUpdateMessageWrapper,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) []string {
	failed := []string{}

	failedResources := getFailedResourceDeploymentsAndUpdateState(
		finished,
		changes,
		deployCtx,
	)
	failed = append(failed, failedResources...)

	failedLinks := getFailedLinkDeploymentsAndUpdateState(
		finished,
		changes,
		deployCtx,
	)
	failed = append(failed, failedLinks...)

	failedChildren := getFailedChildDeploymentsAndUpdateState(
		finished,
		changes,
		deployCtx,
	)
	failed = append(failed, failedChildren...)

	return failed
}

func getFailedResourceDeploymentsAndUpdateState(
	finished map[string]*deployUpdateMessageWrapper,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) []string {
	failed := []string{}

	for updateResourceName := range changes.ResourceChanges {
		resourceElementName := core.ResourceElementID(updateResourceName)
		resourceUpdateFailed := checkResourceUpdateFailedAndUpdateState(
			resourceElementName,
			finished,
			deployCtx,
			updateWasSuccessful,
		)
		if resourceUpdateFailed {
			failed = append(failed, resourceElementName)
		}
	}

	for createdResourceName := range changes.NewResources {
		resourceElementName := core.ResourceElementID(createdResourceName)
		resourceCreationFailed := checkResourceUpdateFailedAndUpdateState(
			resourceElementName,
			finished,
			deployCtx,
			creationWasSuccessful,
		)
		if resourceCreationFailed {
			failed = append(failed, resourceElementName)
		}
	}

	return failed
}

func checkResourceUpdateFailedAndUpdateState(
	resourceElementName string,
	finished map[string]*deployUpdateMessageWrapper,
	deployCtx *DeployContext,
	wasSuccessful func(*deployUpdateMessageWrapper, bool) bool,
) bool {
	if msgWrapper, ok := finished[resourceElementName]; ok {
		if wasSuccessful(msgWrapper, deployCtx.Rollback) {
			deployCtx.State.SetUpdatedElement(&ResourceIDInfo{
				ResourceID:   msgWrapper.resourceUpdateMessage.ResourceID,
				ResourceName: msgWrapper.resourceUpdateMessage.ResourceName,
			})
			return false
		}
	}

	return true
}

func getFailedLinkDeploymentsAndUpdateState(
	finished map[string]*deployUpdateMessageWrapper,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) []string {
	failed := []string{}

	for resourceName, updateResourceChanges := range changes.ResourceChanges {
		linkCreationFailed := checkLinkDeploymentsFailedAndUpdateState(
			resourceName,
			updateResourceChanges.NewOutboundLinks,
			finished,
			deployCtx,
			creationWasSuccessful,
		)
		failed = append(failed, linkCreationFailed...)

		linkUpdateFailed := checkLinkDeploymentsFailedAndUpdateState(
			resourceName,
			updateResourceChanges.OutboundLinkChanges,
			finished,
			deployCtx,
			updateWasSuccessful,
		)
		failed = append(failed, linkUpdateFailed...)
	}

	return failed
}

func checkLinkDeploymentsFailedAndUpdateState(
	linkFromResourceName string,
	linkChanges map[string]provider.LinkChanges,
	finished map[string]*deployUpdateMessageWrapper,
	deployCtx *DeployContext,
	wasSuccessful func(*deployUpdateMessageWrapper, bool) bool,
) []string {
	failedLinks := []string{}

	for linkToResourceName := range linkChanges {
		linkName := core.LogicalLinkName(linkFromResourceName, linkToResourceName)
		linkElementName := linkElementID(
			linkName,
		)
		if msgWrapper, ok := finished[linkElementName]; ok {
			if wasSuccessful(msgWrapper, deployCtx.Rollback) {
				deployCtx.State.SetCreatedElement(&LinkIDInfo{
					LinkID:   msgWrapper.linkUpdateMessage.LinkID,
					LinkName: linkName,
				})
			} else {
				failedLinks = append(failedLinks, linkElementName)
			}
		} else {
			failedLinks = append(failedLinks, linkElementName)
		}
	}

	return failedLinks
}

func getFailedChildDeploymentsAndUpdateState(
	finished map[string]*deployUpdateMessageWrapper,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) []string {
	failed := []string{}

	for childName := range changes.ChildChanges {
		childDeployFailed := checkChildDeploymentFailedAndUpdateState(
			childName,
			finished,
			deployCtx,
			updateWasSuccessful,
		)
		if childDeployFailed {
			childElementName := core.ChildElementID(childName)
			failed = append(failed, childElementName)
		}
	}

	for childName := range changes.NewChildren {
		childDeployFailed := checkChildDeploymentFailedAndUpdateState(
			childName,
			finished,
			deployCtx,
			creationWasSuccessful,
		)
		if childDeployFailed {
			childElementName := core.ChildElementID(childName)
			failed = append(failed, childElementName)
		}
	}

	for _, childName := range changes.RecreateChildren {
		childDeployFailed := checkChildDeploymentFailedAndUpdateState(
			childName,
			finished,
			deployCtx,
			creationWasSuccessful,
		)
		if childDeployFailed {
			childElementName := core.ChildElementID(childName)
			failed = append(failed, childElementName)
		}
	}

	return failed
}

func checkChildDeploymentFailedAndUpdateState(
	childName string,
	finished map[string]*deployUpdateMessageWrapper,
	deployCtx *DeployContext,
	wasSuccessful func(*deployUpdateMessageWrapper, bool) bool,
) bool {
	childElementName := core.ChildElementID(childName)
	if msgWrapper, ok := finished[childElementName]; ok {
		if wasSuccessful(msgWrapper, deployCtx.Rollback) {
			deployCtx.State.SetUpdatedElement(&ChildBlueprintIDInfo{
				ChildInstanceID: msgWrapper.childUpdateMessage.ChildInstanceID,
				ChildName:       msgWrapper.childUpdateMessage.ChildName,
			})
			return false
		}
	}

	return true
}

func removalWasSuccessful(
	msgWrapper *deployUpdateMessageWrapper,
	rollback bool,
) bool {
	if msgWrapper == nil {
		return false
	}

	if msgWrapper.resourceUpdateMessage != nil {
		if rollback {
			return msgWrapper.resourceUpdateMessage.PreciseStatus == core.PreciseResourceStatusCreateRollbackComplete
		}

		return msgWrapper.resourceUpdateMessage.Status == core.ResourceStatusDestroyed
	}

	if msgWrapper.childUpdateMessage != nil {
		if rollback {
			return msgWrapper.childUpdateMessage.Status == core.InstanceStatusDeployRollbackComplete
		}

		return msgWrapper.childUpdateMessage.Status == core.InstanceStatusDestroyed
	}

	if msgWrapper.linkUpdateMessage != nil {
		if rollback {
			return msgWrapper.linkUpdateMessage.Status == core.LinkStatusCreateRollbackComplete
		}

		return msgWrapper.linkUpdateMessage.Status == core.LinkStatusDestroyed
	}

	return false
}

func updateWasSuccessful(
	msgWrapper *deployUpdateMessageWrapper,
	rollback bool,
) bool {
	if msgWrapper == nil {
		return false
	}

	if msgWrapper.resourceUpdateMessage != nil {
		if rollback {
			return msgWrapper.resourceUpdateMessage.PreciseStatus == core.PreciseResourceStatusUpdateRollbackComplete
		}

		return msgWrapper.resourceUpdateMessage.Status == core.ResourceStatusUpdated
	}

	if msgWrapper.childUpdateMessage != nil {
		if rollback {
			return msgWrapper.childUpdateMessage.Status == core.InstanceStatusUpdateRollbackComplete
		}

		return msgWrapper.childUpdateMessage.Status == core.InstanceStatusUpdated
	}

	if msgWrapper.linkUpdateMessage != nil {
		if rollback {
			return msgWrapper.linkUpdateMessage.Status == core.LinkStatusUpdateRollbackComplete
		}

		return msgWrapper.linkUpdateMessage.Status == core.LinkStatusUpdated
	}

	return false
}

func creationWasSuccessful(
	msgWrapper *deployUpdateMessageWrapper,
	rollback bool,
) bool {
	if msgWrapper == nil {
		return false
	}

	if msgWrapper.resourceUpdateMessage != nil {
		if rollback {
			return msgWrapper.resourceUpdateMessage.PreciseStatus == core.PreciseResourceStatusDestroyRollbackComplete
		}

		return msgWrapper.resourceUpdateMessage.Status == core.ResourceStatusCreated
	}

	if msgWrapper.childUpdateMessage != nil {
		if rollback {
			return msgWrapper.childUpdateMessage.Status == core.InstanceStatusDestroyRollbackComplete
		}

		return msgWrapper.childUpdateMessage.Status == core.InstanceStatusDeployed
	}

	if msgWrapper.linkUpdateMessage != nil {
		if rollback {
			return msgWrapper.linkUpdateMessage.Status == core.LinkStatusDestroyRollbackComplete
		}

		return msgWrapper.linkUpdateMessage.Status == core.LinkStatusCreated
	}

	return false
}

func updateToChildUpdateMessage(
	msg *DeploymentUpdateMessage,
	parentInstanceID string,
	element state.Element,
	groupIndex int,
) ChildDeployUpdateMessage {
	return ChildDeployUpdateMessage{
		ParentInstanceID: parentInstanceID,
		ChildInstanceID:  msg.InstanceID,
		ChildName:        element.LogicalName(),
		Group:            groupIndex,
		Status:           msg.Status,
		UpdateTimestamp:  msg.UpdateTimestamp,
	}
}

func finishedToChildUpdateMessage(
	msg *DeploymentFinishedMessage,
	parentInstanceID string,
	element state.Element,
	groupIndex int,
) ChildDeployUpdateMessage {
	return ChildDeployUpdateMessage{
		ParentInstanceID: parentInstanceID,
		ChildInstanceID:  msg.InstanceID,
		ChildName:        element.LogicalName(),
		Group:            groupIndex,
		Status:           msg.Status,
		FailureReasons:   msg.FailureReasons,
		UpdateTimestamp:  msg.UpdateTimestamp,
		Durations:        msg.Durations,
	}
}

func getResourceChangeInfo(
	resourceName string,
	changes *changes.BlueprintChanges,
) *resourceChangeDeployInfo {
	for changeResourceName, resourceChanges := range changes.ResourceChanges {
		if changeResourceName == resourceName {
			return &resourceChangeDeployInfo{
				isNew:   false,
				changes: &resourceChanges,
			}
		}
	}

	for newResourceName, resourceChanges := range changes.NewResources {
		if newResourceName == resourceName {
			return &resourceChangeDeployInfo{
				isNew:   true,
				changes: &resourceChanges,
			}
		}
	}

	return nil
}

func getResolvedResourceFromChanges(
	changes *provider.Changes,
) *provider.ResolvedResource {
	if changes == nil {
		return nil
	}

	return changes.AppliedResourceInfo.ResourceWithResolvedSubs
}

func getDeploymentNode(
	element state.Element,
	groups [][]*DeploymentNode,
	groupIndex int,
) *DeploymentNode {
	if groupIndex < 0 || groupIndex >= len(groups) {
		return nil
	}

	group := groups[groupIndex]

	node := (*DeploymentNode)(nil)
	i := 0
	for node == nil && i < len(group) {
		current := group[i]
		if current.Type() == DeploymentNodeTypeResource {
			if current.ChainLinkNode.ResourceName == element.LogicalName() {
				node = current
			}
		} else {
			if current.ChildNode.ElementName == element.LogicalName() {
				node = current
			}
		}

		i += 1
	}

	return node
}

func createDeployFinishedInstanceStatusInfo(
	msg *DeploymentFinishedMessage,
	rollingBack bool,
	isNew bool,
) state.InstanceStatusInfo {
	updateTimestamp := int(msg.UpdateTimestamp)
	instanceStatusInfo := state.InstanceStatusInfo{
		Status:                     msg.Status,
		Durations:                  msg.Durations,
		FailureReasons:             msg.FailureReasons,
		LastDeployAttemptTimestamp: &updateTimestamp,
		LastStatusUpdateTimestamp:  &updateTimestamp,
	}

	if wasDeploymentSuccessful(msg, rollingBack, isNew) {
		finishTimestamp := int(msg.FinishTimestamp)
		instanceStatusInfo.LastDeployedTimestamp = &finishTimestamp
	}

	return instanceStatusInfo
}

type resourceChangeDeployInfo struct {
	isNew   bool
	changes *provider.Changes
}

func resourceHasFieldsToResolve(
	resourceName string,
	resolvePaths []string,
) bool {
	resourceElementIDDotPrefix := fmt.Sprintf("%s.", core.ResourceElementID(resourceName))
	resourceElementIDBracketPrefix := fmt.Sprintf("%s[", core.ResourceElementID(resourceName))

	return slices.ContainsFunc(resolvePaths, func(path string) bool {
		return strings.HasPrefix(path, resourceElementIDDotPrefix) ||
			strings.HasPrefix(path, resourceElementIDBracketPrefix)
	})
}

func prepareResourceChangesForDeployment(
	changes *provider.Changes,
	resolvedResource *provider.ResolvedResource,
	resourceState *state.ResourceState,
	resourceID string,
	instanceID string,
) *provider.Changes {
	return &provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID:               resourceID,
			ResourceName:             changes.AppliedResourceInfo.ResourceName,
			InstanceID:               instanceID,
			CurrentResourceState:     resourceState,
			ResourceWithResolvedSubs: resolvedResource,
		},
		MustRecreate:              changes.MustRecreate,
		ModifiedFields:            changes.ModifiedFields,
		NewFields:                 changes.NewFields,
		RemovedFields:             changes.RemovedFields,
		UnchangedFields:           changes.UnchangedFields,
		ComputedFields:            changes.ComputedFields,
		FieldChangesKnownOnDeploy: changes.FieldChangesKnownOnDeploy,
		ConditionKnownOnDeploy:    changes.ConditionKnownOnDeploy,
		NewOutboundLinks:          changes.NewOutboundLinks,
		OutboundLinkChanges:       changes.OutboundLinkChanges,
		RemovedOutboundLinks:      changes.RemovedOutboundLinks,
	}
}

func countElementsToDeploy(changes *changes.BlueprintChanges) int {
	linksToDeployCount := 0
	for _, newResourceChanges := range changes.NewResources {
		linksToDeployCount += len(newResourceChanges.NewOutboundLinks)
	}

	for _, resourceChanges := range changes.ResourceChanges {
		linksToDeployCount += len(resourceChanges.NewOutboundLinks) +
			len(resourceChanges.OutboundLinkChanges)
	}

	return len(changes.NewResources) +
		len(changes.ResourceChanges) +
		len(changes.NewChildren) +
		len(changes.RecreateChildren) +
		len(changes.ChildChanges) +
		linksToDeployCount
}

func getRetryPolicy(
	ctx context.Context,
	resourceProviders map[string]provider.Provider,
	resourceName string,
	defaultRetryPolicy *provider.RetryPolicy,
) (*provider.RetryPolicy, error) {
	provider, ok := resourceProviders[resourceName]
	if !ok {
		return defaultRetryPolicy, nil
	}

	retryPolicy, err := provider.RetryPolicy(ctx)
	if err != nil {
		return nil, err
	}

	if retryPolicy == nil {
		return defaultRetryPolicy, nil
	}

	return retryPolicy, nil
}

func createElementFromDeploymentNode(
	node *DeploymentNode,
) state.Element {
	if node.Type() == DeploymentNodeTypeResource {
		return &ResourceIDInfo{
			ResourceID:   "",
			ResourceName: node.ChainLinkNode.ResourceName,
		}
	}

	return &ChildBlueprintIDInfo{
		ChildInstanceID: "",
		ChildName:       core.ToLogicalChildName(node.ChildNode.ElementName),
	}
}

func dependencyMustStabilise(
	dependant *DeploymentNode,
	dependency *DeploymentNode,
	resourceTypesMustStabilise []string,
) bool {
	if dependency.Type() == DeploymentNodeTypeResource &&
		dependant.Type() == DeploymentNodeTypeResource {
		dependencyResource := dependency.ChainLinkNode.Resource
		dependencyResourceType := schema.GetResourceType(dependencyResource)

		return slices.Contains(
			resourceTypesMustStabilise,
			dependencyResourceType,
		)
	}

	// Either or both the dependant and dependency are not resources,
	// so the dependency should be considered as in progress.
	// If the dependency is a child blueprint, it must be fully deployed before
	// the dependant can be deployed.
	// If the dependency is a resource and the dependant is a child blueprint,
	// the resource must be stable before the child blueprint can be deployed.
	return true
}

func readyToDeployAfterDependency(
	dependant *DeploymentNode,
	dependency *DeploymentNode,
	stabilisedDeps []string,
	configComplete bool,
) bool {
	if dependant == nil || dependency == nil {
		return false
	}

	if configComplete {
		// Child blueprints are not deployed until all resources they depend on are stable,
		// only resources can be deployed on config complete.
		return dependant.Type() == DeploymentNodeTypeResource &&
			// The dependant can only be deployed on config complete if the dependency
			// is not required to be stable before the dependant can be deployed.
			!dependencyMustStabilise(dependant, dependency, stabilisedDeps)
	}

	// If we are not waiting for config completion, the dependant can be deployed
	// given that the dependency has already been deployed.
	return true
}

func getChildChanges(parentChanges *changes.BlueprintChanges, childName string) *changes.BlueprintChanges {
	childChanges, hasChildChanges := parentChanges.ChildChanges[childName]
	if hasChildChanges {
		return &childChanges
	}

	newChildDefinition, hasNewChildDefinition := parentChanges.NewChildren[childName]
	if hasNewChildDefinition {
		return &changes.BlueprintChanges{
			NewResources: newChildDefinition.NewResources,
			NewChildren:  newChildDefinition.NewChildren,
			NewExports:   newChildDefinition.NewExports,
		}
	}

	return nil
}

func getLinkUpdateTypeFromState(linkState state.LinkState) provider.LinkUpdateType {
	if linkState.LinkID == "" {
		return provider.LinkUpdateTypeCreate
	}

	return provider.LinkUpdateTypeUpdate
}

func getLinkRetryPolicy(
	ctx context.Context,
	logicalLinkName string,
	instanceState *state.InstanceState,
	linkRegistry provider.LinkRegistry,
	defaultRetryPolicy *provider.RetryPolicy,
) (*provider.RetryPolicy, error) {
	resourceTypeA, resourceTypeB, err := getResourceTypesForLink(logicalLinkName, instanceState)
	if err != nil {
		return nil, err
	}

	provider, err := linkRegistry.Provider(resourceTypeA, resourceTypeB)
	if err != nil {
		return nil, err
	}

	retryPolicy, err := provider.RetryPolicy(ctx)
	if err != nil {
		return nil, err
	}

	if retryPolicy == nil {
		return defaultRetryPolicy, nil
	}

	return retryPolicy, nil
}

func resolvedMetadataToState(
	resolvedMetadata *provider.ResolvedResourceMetadata,
) *state.ResourceMetadataState {
	if resolvedMetadata == nil {
		return nil
	}

	annotations := map[string]*core.MappingNode{}
	if resolvedMetadata.Annotations != nil {
		annotations = resolvedMetadata.Annotations.Fields
	}

	labels := map[string]string{}
	if resolvedMetadata.Labels != nil {
		labels = resolvedMetadata.Labels.Values
	}

	return &state.ResourceMetadataState{
		DisplayName: core.StringValue(resolvedMetadata.DisplayName),
		Annotations: annotations,
		Labels:      labels,
		Custom:      resolvedMetadata.Custom,
	}
}

func extractResolvedMetadataFromResourceInfo(
	resourceInfo *resourceDeployInfo,
) *provider.ResolvedResourceMetadata {
	if resourceInfo.changes == nil {
		return nil
	}

	appliedResourceInfo := resourceInfo.changes.AppliedResourceInfo
	if appliedResourceInfo.ResourceWithResolvedSubs == nil {
		return nil
	}

	return appliedResourceInfo.ResourceWithResolvedSubs.Metadata
}

func extractNodeDependencyInfo(node *DeploymentNode) *state.DependencyInfo {
	dependencyInfo := &state.DependencyInfo{
		DependsOnResources: []string{},
		DependsOnChildren:  []string{},
	}

	for _, dependency := range node.DirectDependencies {
		if dependency.Type() == DeploymentNodeTypeResource {
			dependencyInfo.DependsOnResources = append(
				dependencyInfo.DependsOnResources,
				dependency.ChainLinkNode.ResourceName,
			)
		} else {
			dependencyInfo.DependsOnChildren = append(
				dependencyInfo.DependsOnChildren,
				core.ToLogicalChildName(dependency.ChildNode.ElementName),
			)
		}
	}

	return dependencyInfo
}

func addRemovedResourcesToProvidersMap(
	resourceProviderMap map[string]provider.Provider,
	currentInstanceState *state.InstanceState,
	providers map[string]provider.Provider,
) map[string]provider.Provider {
	if currentInstanceState == nil {
		return resourceProviderMap
	}

	updatedResourceProviderMap := map[string]provider.Provider{}
	for resourceName, provider := range resourceProviderMap {
		updatedResourceProviderMap[resourceName] = provider
	}

	for resourceName, resourceID := range currentInstanceState.ResourceIDs {
		_, hasResourceProvider := updatedResourceProviderMap[resourceName]
		if !hasResourceProvider {
			resource, stateHasResource := currentInstanceState.Resources[resourceID]
			if stateHasResource {
				providerNamespace := provider.ExtractProviderFromItemType(resource.Type)
				provider, hasProvider := providers[providerNamespace]
				if hasProvider {
					updatedResourceProviderMap[resourceName] = provider
				}
			}
		}
	}

	return updatedResourceProviderMap
}

// This is used during the deploy and destroy process to provide an equivalent
// namespaced logical identifier for links that distinguishes link names
// from resource and child blueprint names.
func linkElementID(linkName string) string {
	return fmt.Sprintf("link(%s)", linkName)
}

func pluralise(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}

	return plural
}

// isTerminalResourceFailure returns true if the resource update message
// represents a terminal (non-retryable) failure.
func isTerminalResourceFailure(msg ResourceDeployUpdateMessage, rollback bool) bool {
	if msg.CanRetry {
		return false
	}

	if rollback {
		return msg.PreciseStatus == core.PreciseResourceStatusDestroyRollbackFailed ||
			msg.PreciseStatus == core.PreciseResourceStatusUpdateRollbackFailed ||
			msg.PreciseStatus == core.PreciseResourceStatusCreateRollbackFailed
	}

	return msg.PreciseStatus == core.PreciseResourceStatusCreateFailed ||
		msg.PreciseStatus == core.PreciseResourceStatusUpdateFailed
}

// isTerminalChildFailure returns true if the child update message
// represents a terminal failure.
func isTerminalChildFailure(msg ChildDeployUpdateMessage, rollback bool) bool {
	if rollback {
		return msg.Status == core.InstanceStatusDestroyRollbackFailed ||
			msg.Status == core.InstanceStatusUpdateRollbackFailed ||
			msg.Status == core.InstanceStatusDeployRollbackFailed
	}

	return msg.Status == core.InstanceStatusDeployFailed ||
		msg.Status == core.InstanceStatusUpdateFailed
}

// isTerminalLinkFailure returns true if the link update message
// represents a terminal (non-retryable) failure.
func isTerminalLinkFailure(msg LinkDeployUpdateMessage, rollback bool) bool {
	if msg.CanRetryCurrentStage {
		return false
	}

	if rollback {
		return msg.PreciseStatus == core.PreciseLinkStatusResourceAUpdateRollbackFailed ||
			msg.PreciseStatus == core.PreciseLinkStatusResourceBUpdateRollbackFailed ||
			msg.PreciseStatus == core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed
	}

	return msg.PreciseStatus == core.PreciseLinkStatusResourceAUpdateFailed ||
		msg.PreciseStatus == core.PreciseLinkStatusResourceBUpdateFailed ||
		msg.PreciseStatus == core.PreciseLinkStatusIntermediaryResourceUpdateFailed
}

// getFailedElementsFromFinished extracts the names of failed elements
// from the finished map.
func getFailedElementsFromFinished(
	finished map[string]*deployUpdateMessageWrapper,
	rollback bool,
) []string {
	failed := []string{}

	for elementName, wrapper := range finished {
		if wrapper.resourceUpdateMessage != nil {
			if !creationWasSuccessful(wrapper, rollback) &&
				!updateWasSuccessful(wrapper, rollback) {
				failed = append(failed, elementName)
			}
		}

		if wrapper.childUpdateMessage != nil {
			if !creationWasSuccessful(wrapper, rollback) &&
				!updateWasSuccessful(wrapper, rollback) {
				failed = append(failed, elementName)
			}
		}

		if wrapper.linkUpdateMessage != nil {
			if !creationWasSuccessful(wrapper, rollback) &&
				!updateWasSuccessful(wrapper, rollback) {
				failed = append(failed, elementName)
			}
		}
	}

	return failed
}

// drainFailureMessages generates failure messages for a drained deployment,
// combining the drain error with individual element failures.
func drainFailureMessages(
	deployCtx *DeployContext,
	failed []string,
) []string {
	operation := determineOperation(deployCtx)
	messages := []string{
		fmt.Sprintf(
			"failed to %s the blueprint instance due to %d %s",
			operation,
			len(failed),
			pluralise("failure", "failures", len(failed)),
		),
	}
	messageTemplate := "failed to %s %q"
	for _, elementName := range failed {
		messages = append(messages, fmt.Sprintf(messageTemplate, operation, elementName))
	}
	return messages
}

// drainFailureMessagesWithInterrupted generates failure messages for a drained
// deployment that includes both failed and interrupted elements.
func drainFailureMessagesWithInterrupted(
	deployCtx *DeployContext,
	failed []string,
	interrupted []string,
) []string {
	operation := determineOperation(deployCtx)
	totalFailures := len(failed) + len(interrupted)
	messages := []string{
		fmt.Sprintf(
			"failed to %s the blueprint instance due to %d %s",
			operation,
			totalFailures,
			pluralise("failure", "failures", totalFailures),
		),
	}

	messageTemplate := "failed to %s %q"
	for _, elementName := range failed {
		messages = append(messages, fmt.Sprintf(messageTemplate, operation, elementName))
	}

	interruptedMessageTemplate := "%q was interrupted and its state is unknown"
	for _, elementName := range interrupted {
		messages = append(messages, fmt.Sprintf(interruptedMessageTemplate, elementName))
	}

	return messages
}

// getElementNames extracts the namespaced logical names from a slice of elements.
func getElementNames(elements []state.Element) []string {
	names := make([]string, 0, len(elements))
	for _, elem := range elements {
		names = append(names, getNamespacedLogicalName(elem))
	}
	return names
}

// determineResourceInterruptedStatus returns the appropriate interrupted status
// for a resource based on whether the operation was a destroy or deploy/update.
func determineResourceInterruptedStatus(
	destroying bool,
	rollback bool,
) (core.ResourceStatus, core.PreciseResourceStatus) {
	if destroying && !rollback {
		return core.ResourceStatusDestroyInterrupted, core.PreciseResourceStatusDestroyInterrupted
	}

	if destroying && rollback {
		// Destroying during rollback means we were rolling back a deployment
		// by destroying the resource that was created.
		return core.ResourceStatusCreateInterrupted, core.PreciseResourceStatusCreateInterrupted
	}

	if !destroying && rollback {
		// Deploying during rollback means we were rolling back a destruction
		// or rolling back an update.
		return core.ResourceStatusUpdateInterrupted, core.PreciseResourceStatusUpdateInterrupted
	}

	// Normal deploy/update operation
	// We use UpdateInterrupted as a general case since we can't distinguish
	// between create and update at this point without more context.
	// The reconciliation process will determine the actual state.
	return core.ResourceStatusUpdateInterrupted, core.PreciseResourceStatusUpdateInterrupted
}

// determineLinkInterruptedStatus returns the appropriate interrupted status
// for a link based on whether the operation was a destroy or deploy/update.
func determineLinkInterruptedStatus(
	destroying bool,
	rollback bool,
) (core.LinkStatus, core.PreciseLinkStatus) {
	if destroying && !rollback {
		return core.LinkStatusDestroyInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	if destroying && rollback {
		// Destroying during rollback means we were rolling back a link creation.
		return core.LinkStatusCreateInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	if !destroying && rollback {
		// Deploying during rollback means we were rolling back a link destruction
		// or rolling back an update.
		return core.LinkStatusUpdateInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	// Normal deploy/update operation
	return core.LinkStatusUpdateInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
}

// determineChildInterruptedStatus returns the appropriate interrupted status
// for a child blueprint based on whether the operation was a destroy or deploy/update.
func determineChildInterruptedStatus(
	destroying bool,
	rollback bool,
) core.InstanceStatus {
	if destroying && !rollback {
		return core.InstanceStatusDestroyInterrupted
	}

	if destroying && rollback {
		// Destroying during rollback means we were rolling back a deployment.
		return core.InstanceStatusDeployInterrupted
	}

	if !destroying && rollback {
		// Deploying during rollback means we were rolling back a destruction
		// or rolling back an update.
		return core.InstanceStatusUpdateInterrupted
	}

	// Normal deploy/update operation
	return core.InstanceStatusUpdateInterrupted
}
