package deploymentsv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/blueprint"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginmeta"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	internalutils "github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/utils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// CreateBlueprintInstanceHandler is the handler for the POST /deployments/instances
// endpoint that creates a new blueprint instance and begins the deployment
// process for the new blueprint instance.
func (c *Controller) CreateBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	c.handleDeployRequest(
		w,
		r,
		// There is no existing instance for a new deployment.
		/* existingInstance */
		nil,
	)
}

// UpdateBlueprintInstanceHandler is the handler for the PATCH /deployments/instances/{id}
// endpoint that updates an existing blueprint instance and begins the deployment
// process for the updates described in the specified change set.
// The {id} path parameter can be either an instance ID or an instance name.
func (c *Controller) UpdateBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceIDOrName := params["id"]

	instance, err := resolveInstance(r.Context(), instanceIDOrName, c.instances)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceIDOrName)
		return
	}

	c.handleDeployRequest(
		w,
		r,
		&instance,
	)
}

// StreamDeploymentEventsHandler is the handler for the GET /deployments/instances/{id}/stream endpoint
// that streams deployment events to the client using Server-Sent Events (SSE).
// The {id} path parameter can be either an instance ID or an instance name.
func (c *Controller) StreamDeploymentEventsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	instanceIDOrName := params["id"]

	instanceID, err := resolveInstanceID(r.Context(), instanceIDOrName, c.instances)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceIDOrName)
		return
	}

	helpersv1.SSEStreamEvents(
		w,
		r,
		&helpersv1.StreamInfo{
			ChannelType: helpersv1.ChannelTypeDeployment,
			ChannelID:   instanceID,
		},
		c.eventStore,
		c.logger.Named("deploymentStream").WithFields(
			core.StringLogField("instanceId", instanceID),
			core.StringLogField("eventChannelType", helpersv1.ChannelTypeDeployment),
		),
	)
}

// GetBlueprintInstanceHandler is the handler for the GET /deployments/instances/{id} endpoint
// that retrieves the full state of a blueprint instance.
// The {id} path parameter can be either an instance ID or an instance name.
func (c *Controller) GetBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceIDOrName := params["id"]

	instance, err := resolveInstance(r.Context(), instanceIDOrName, c.instances)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceIDOrName)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusOK,
		instance,
	)
}

// GetBlueprintInstanceExportsHandler is the handler for the
// GET /deployments/instances/{id}/exports endpoint that retrieves the
// exports of a blueprint instance.
// The {id} path parameter can be either an instance ID or an instance name.
func (c *Controller) GetBlueprintInstanceExportsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceIDOrName := params["id"]

	instanceID, err := resolveInstanceID(r.Context(), instanceIDOrName, c.instances)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceIDOrName)
		return
	}

	exports, err := c.exports.GetAll(
		r.Context(),
		instanceID,
	)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceID)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusOK,
		exports,
	)
}

// DestroyBlueprintInstanceHandler is the handler for the
// POST /deployments/instances/{id}/destroy endpoint
// that destroys a blueprint instance.
// This is a `POST` request as the destroy operation relies
// on inputs including configuration values that need to be
// provided in the request body.
// The {id} path parameter can be either an instance ID or an instance name.
func (c *Controller) DestroyBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	pathParams := mux.Vars(r)
	instanceIDOrName := pathParams["id"]

	instance, err := resolveInstance(r.Context(), instanceIDOrName, c.instances)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceIDOrName)
		return
	}

	payload := &BlueprintInstanceDestroyRequestPayload{}
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.logger)
	if responseWritten {
		return
	}

	if err := helpersv1.ValidateRequestBody.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		inputvalidation.HTTPValidationError(w, validationErrors)
		return
	}

	finalConfig, _, responseWritten := helpersv1.PrepareAndValidatePluginConfig(
		r,
		w,
		payload.Config,
		/* validate */ true,
		c.pluginConfigPreparer,
		c.logger,
	)
	if responseWritten {
		return
	}

	changeset, err := c.changesetStore.Get(r.Context(), payload.ChangeSetID)
	if err != nil {
		c.handleGetChangesetErrorForResponse(
			w,
			err,
			payload.ChangeSetID,
		)
		return
	}

	// Check if changeset has drift detected status and block the destroy unless force is set
	if !payload.Force {
		if changeset.Status == manage.ChangesetStatusDriftDetected {
			c.respondWithDriftBlocked(
				r.Context(),
				w,
				instance.InstanceID,
				changeset,
			)
			return
		}
	}

	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	// Create tagging config from the request payload, applying defaults as needed.
	taggingConfig := c.createTaggingConfig(payload.Config)

	// Get the last event ID for the deployment channel before starting the async operation.
	// This allows clients to use it as a starting offset when streaming events.
	lastEventID, err := c.eventStore.GetLastEventID(
		r.Context(),
		helpersv1.ChannelTypeDeployment,
		instance.InstanceID,
	)
	if err != nil {
		c.logger.Debug(
			"failed to get last event ID for deployment channel",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	go c.startDestroy(
		changeset,
		instance.InstanceID,
		payload.AsRollback,
		payload.Force,
		params,
		taggingConfig,
	)

	// The instance status will be updated by the deployment process
	// but we need to give an indicator to the caller that something
	// is happening in the response.
	instance.Status = core.InstanceStatusDestroying

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[state.InstanceState]{
			LastEventID: lastEventID,
			Data:        instance,
		},
	)
}

func (c *Controller) handleDeployRequest(
	w http.ResponseWriter,
	r *http.Request,
	existingInstance *state.InstanceState,
) {
	payload := &BlueprintInstanceRequestPayload{}
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.logger)
	if responseWritten {
		return
	}

	if err := helpersv1.ValidateRequestBody.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		inputvalidation.HTTPValidationError(w, validationErrors)
		return
	}

	helpersv1.PopulateBlueprintDocInfoDefaults(&payload.BlueprintDocumentInfo)

	finalConfig, _, responseWritten := helpersv1.PrepareAndValidatePluginConfig(
		r,
		w,
		payload.Config,
		/* validate */ true,
		c.pluginConfigPreparer,
		c.logger,
	)
	if responseWritten {
		return
	}

	blueprintInfo, responseWritten := resolve.ResolveBlueprintForRequest(
		r,
		w,
		&payload.BlueprintDocumentInfo,
		c.blueprintResolver,
		c.logger,
	)
	if responseWritten {
		return
	}

	changeset, err := c.changesetStore.Get(r.Context(), payload.ChangeSetID)
	if err != nil {
		c.handleGetChangesetErrorForResponse(
			w,
			err,
			payload.ChangeSetID,
		)
		return
	}

	if changeset.Destroy {
		httputils.HTTPErrorWithFields(
			w,
			http.StatusBadRequest,
			"cannot deploy using a destroy changeset",
			map[string]any{
				"code": "DESTROY_CHANGESET",
			},
		)
		return
	}

	// For updates (existing instances), check if changeset has drift detected status
	// and block the deployment unless force is set
	if existingInstance != nil && !payload.Force {
		if changeset.Status == manage.ChangesetStatusDriftDetected {
			c.respondWithDriftBlocked(
				r.Context(),
				w,
				existingInstance.InstanceID,
				changeset,
			)
			return
		}
	}

	// Add blueprint directory to context variables for resolving relative child blueprint paths.
	finalConfig = internalutils.EnsureBlueprintDirContextVar(finalConfig, payload.BlueprintDocumentInfo.Directory)
	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	// Create tagging config from the request payload, applying defaults as needed.
	taggingConfig := c.createTaggingConfig(payload.Config)

	instanceID, err := c.startDeployment(
		blueprintInfo,
		changeset,
		getInstanceID(existingInstance),
		payload.InstanceName,
		payload.AsRollback,
		payload.AutoRollback,
		payload.Force,
		existingInstance,
		helpersv1.GetFormat(payload.BlueprintFile),
		params,
		taggingConfig,
	)
	if err != nil {
		handleDeployErrorForResponse(w, err, c.logger)
		return
	}

	instance := existingInstance
	if existingInstance == nil {
		newInstance, err := c.instances.Get(r.Context(), instanceID)
		if err != nil {
			c.logger.Error(
				"Failed to get newly created instance",
				core.ErrorLogField("error", err),
				core.StringLogField("instanceId", instanceID),
			)
			httputils.HTTPError(
				w,
				http.StatusInternalServerError,
				utils.UnexpectedErrorMessage,
			)
			return
		}
		instance = &newInstance
	}

	// Get the last event ID for the deployment channel.
	// This allows clients to use it as a starting offset when streaming events.
	lastEventID, err := c.eventStore.GetLastEventID(
		r.Context(),
		helpersv1.ChannelTypeDeployment,
		instanceID,
	)
	if err != nil {
		c.logger.Debug(
			"failed to get last event ID for deployment channel",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[state.InstanceState]{
			LastEventID: lastEventID,
			Data:        *instance,
		},
	)
}

func (c *Controller) handleGetChangesetErrorForResponse(
	w http.ResponseWriter,
	err error,
	changesetID string,
) {
	changesetNotFoundErr := &manage.ChangesetNotFound{}
	if errors.As(err, &changesetNotFoundErr) {
		httputils.HTTPError(
			w,
			http.StatusBadRequest,
			"requested change set is missing",
		)
		return
	}

	c.logger.Error(
		"Failed to get changeset",
		core.ErrorLogField("error", err),
		core.StringLogField("changesetId", changesetID),
	)
	httputils.HTTPError(
		w,
		http.StatusInternalServerError,
		utils.UnexpectedErrorMessage,
	)
}

func (c *Controller) handleGetInstanceError(
	w http.ResponseWriter,
	err error,
	instanceID string,
) {
	if state.IsInstanceNotFound(err) {
		httputils.HTTPError(
			w,
			http.StatusNotFound,
			fmt.Sprintf("blueprint instance %q not found", instanceID),
		)
		return
	}

	c.logger.Debug(
		"failed to get blueprint instance",
		core.ErrorLogField("error", err),
		core.StringLogField("instanceId", instanceID),
	)
	httputils.HTTPError(
		w,
		http.StatusInternalServerError,
		utils.UnexpectedErrorMessage,
	)
}

func (c *Controller) startDeployment(
	blueprintInfo *includes.ChildBlueprintInfo,
	changeset *manage.Changeset,
	deployInstanceID string,
	instanceName string,
	forRollback bool,
	autoRollback bool,
	force bool,
	previousInstanceState *state.InstanceState,
	format schema.SpecFormat,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		c.deploymentTimeout,
	)

	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		helpersv1.GetBlueprintSource(blueprintInfo),
		format,
		params,
	)
	if err != nil {
		cancel()
		// As we don't have an ID for the blueprint instance at this stage,
		// we don't have a channel that we can associate events with.
		// For this reason, we'll return an error instead of writing to an event channel.
		return "", err
	}

	channels := container.CreateDeployChannels()
	err = blueprintContainer.Deploy(
		ctxWithTimeout,
		&container.DeployInput{
			InstanceID:             deployInstanceID,
			InstanceName:           instanceName,
			Changes:                changeset.Changes,
			Rollback:               forRollback,
			Force:                  force,
			TaggingConfig:          taggingConfig,
			ProviderMetadataLookup: pluginmeta.ToLookupFunc(c.providerMetadataLookup),
			DrainTimeout:           c.drainTimeout,
		},
		channels,
		params,
	)
	if err != nil {
		cancel()
		return "", err
	}

	finalInstanceID := deployInstanceID
	if finalInstanceID == "" {
		// Capture the instance ID from the "preparing" event
		// for a new deployment.
		finalInstanceID, err = c.captureInstanceIDFromEvent(
			ctxWithTimeout,
			channels,
		)
		if err != nil {
			cancel()
			return "", err
		}
	}

	go c.listenForDeploymentUpdates(
		ctxWithTimeout,
		cancel,
		finalInstanceID,
		"deploying blueprint instance",
		channels,
		changeset,
		autoRollback,
		previousInstanceState,
		c.logger.Named("deployment").WithFields(
			core.StringLogField("instanceId", finalInstanceID),
		),
	)

	return finalInstanceID, nil
}

func (c *Controller) captureInstanceIDFromEvent(
	ctx context.Context,
	channels *container.DeployChannels,
) (string, error) {
	var instanceID string
	var preparingMessage *container.DeploymentUpdateMessage
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case msg := <-channels.DeploymentUpdateChan:
		if msg.Status == core.InstanceStatusPreparing {
			instanceID = msg.InstanceID
			preparingMessage = &msg
		}
	case err := <-channels.ErrChan:
		return "", err
	}

	c.saveDeploymentEvent(
		ctx,
		eventTypeInstanceUpdate,
		preparingMessage,
		preparingMessage.UpdateTimestamp,
		/* endOfStream */ false,
		instanceID,
		"deploying blueprint instance",
		c.logger,
	)

	return instanceID, nil
}

func (c *Controller) startDestroy(
	changeset *manage.Changeset,
	destroyInstanceID string,
	forRollback bool,
	force bool,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		c.deploymentTimeout,
	)

	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		// The destroy operation does not use a source blueprint
		// document, however, in order to load the blueprint container,
		// we need to provide a source blueprint document.
		placeholderBlueprint,
		schema.YAMLSpecFormat,
		params,
	)
	if err != nil {
		cancel()
		c.handleDeploymentErrorAsEvent(
			ctxWithTimeout,
			destroyInstanceID,
			err,
			"destroying blueprint instance",
			c.logger,
		)
		return
	}

	channels := container.CreateDeployChannels()
	blueprintContainer.Destroy(
		ctxWithTimeout,
		&container.DestroyInput{
			InstanceID:             destroyInstanceID,
			Changes:                changeset.Changes,
			Rollback:               forRollback,
			Force:                  force,
			TaggingConfig:          taggingConfig,
			ProviderMetadataLookup: pluginmeta.ToLookupFunc(c.providerMetadataLookup),
			DrainTimeout:           c.drainTimeout,
		},
		channels,
		params,
	)

	c.listenForDeploymentUpdates(
		ctxWithTimeout,
		cancel,
		destroyInstanceID,
		"destroying blueprint instance",
		channels,
		changeset,
		false, // autoRollback is not applicable for destroy operations
		nil,   // no previous state needed for destroy operations
		c.logger.Named("destroy").WithFields(
			core.StringLogField("instanceId", destroyInstanceID),
		),
	)
}

// listenForDeploymentUpdatesParams holds optional parameters for listenForDeploymentUpdates.
type listenForDeploymentUpdatesParams struct {
	// SkippedRollbackItems contains items that were skipped during rollback filtering.
	// These will be attached to the finish message when the deployment completes.
	SkippedRollbackItems []changes.SkippedRollbackItem
}

func (c *Controller) listenForDeploymentUpdates(
	ctx context.Context,
	cancelCtx func(),
	instanceID string,
	action string,
	channels *container.DeployChannels,
	changeset *manage.Changeset,
	autoRollback bool,
	previousInstanceState *state.InstanceState,
	logger core.Logger,
) {
	c.listenForDeploymentUpdatesWithParams(
		ctx, cancelCtx, instanceID, action, channels, changeset,
		autoRollback, previousInstanceState, logger, nil,
	)
}

func (c *Controller) listenForDeploymentUpdatesWithParams(
	ctx context.Context,
	cancelCtx func(),
	instanceID string,
	action string,
	channels *container.DeployChannels,
	changeset *manage.Changeset,
	autoRollback bool,
	previousInstanceState *state.InstanceState,
	logger core.Logger,
	params *listenForDeploymentUpdatesParams,
) {
	defer cancelCtx()

	finishMsg := (*container.DeploymentFinishedMessage)(nil)
	var err error
	for err == nil && finishMsg == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			c.handleDeploymentResourceUpdateMessage(ctx, msg, instanceID, action, logger)
		case msg := <-channels.ChildUpdateChan:
			c.handleDeploymentChildUpdateMessage(ctx, msg, instanceID, action, logger)
		case msg := <-channels.LinkUpdateChan:
			c.handleDeploymentLinkUpdateMessage(ctx, msg, instanceID, action, logger)
		case msg := <-channels.DeploymentUpdateChan:
			c.handleDeploymentUpdateMessage(ctx, msg, instanceID, action, logger)
		case msg := <-channels.FinishChan:
			// Attach skipped rollback items if this is a rollback completion
			if params != nil && len(params.SkippedRollbackItems) > 0 {
				msg.SkippedRollbackItems = convertSkippedItemsToContainerType(params.SkippedRollbackItems)
			}
			shouldRollback, _ := shouldTriggerAutoRollback(msg.Status)
			willAutoRollback := autoRollback && shouldRollback
			// If auto-rollback will trigger, don't mark this as end of stream
			// as the rollback events will follow.
			c.handleDeploymentFinishUpdateMessageWithEndOfStream(
				ctx, msg, instanceID, action, !willAutoRollback, logger,
			)
			finishMsg = &msg
		case err = <-channels.ErrChan:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	if err != nil {
		c.handleDeploymentErrorAsEvent(
			ctx,
			instanceID,
			err,
			action,
			logger,
		)
		return
	}

	// Check if auto-rollback should be triggered after deployment failure
	if finishMsg != nil && autoRollback {
		shouldRollback, rollbackType := shouldTriggerAutoRollback(finishMsg.Status)
		if shouldRollback {
			logger.Info(
				"auto-rollback triggered due to deployment failure",
				core.IntegerLogField("status", int64(finishMsg.Status)),
				core.IntegerLogField("rollbackType", int64(rollbackType)),
			)
			switch rollbackType {
			case AutoRollbackTypeDestroy:
				c.executeNewDeploymentRollback(ctx, instanceID, changeset, finishMsg.FailureReasons, logger)
			case AutoRollbackTypeRevert:
				c.executeUpdateRollback(ctx, instanceID, changeset, previousInstanceState, finishMsg.FailureReasons, logger)
			}
		}
	}
}

func (c *Controller) handleDeploymentResourceUpdateMessage(
	ctx context.Context,
	msg container.ResourceDeployUpdateMessage,
	instanceID string,
	action string,
	logger core.Logger,
) {
	c.saveDeploymentEvent(
		ctx,
		eventTypeResourceUpdate,
		&msg,
		msg.UpdateTimestamp,
		/* endOfStream */ false,
		instanceID,
		action,
		logger,
	)
}

func (c *Controller) handleDeploymentChildUpdateMessage(
	ctx context.Context,
	msg container.ChildDeployUpdateMessage,
	instanceID string,
	action string,
	logger core.Logger,
) {
	c.saveDeploymentEvent(
		ctx,
		eventTypeChildUpdate,
		&msg,
		msg.UpdateTimestamp,
		/* endOfStream */ false,
		instanceID,
		action,
		logger,
	)
}

func (c *Controller) handleDeploymentLinkUpdateMessage(
	ctx context.Context,
	msg container.LinkDeployUpdateMessage,
	instanceID string,
	action string,
	logger core.Logger,
) {
	c.saveDeploymentEvent(
		ctx,
		eventTypeLinkUpdate,
		&msg,
		msg.UpdateTimestamp,
		/* endOfStream */ false,
		instanceID,
		action,
		logger,
	)
}

func (c *Controller) handleDeploymentUpdateMessage(
	ctx context.Context,
	msg container.DeploymentUpdateMessage,
	instanceID string,
	action string,
	logger core.Logger,
) {
	c.saveDeploymentEvent(
		ctx,
		eventTypeInstanceUpdate,
		&msg,
		msg.UpdateTimestamp,
		/* endOfStream */ false,
		instanceID,
		action,
		logger,
	)
}

func (c *Controller) handleDeploymentFinishUpdateMessageWithEndOfStream(
	ctx context.Context,
	msg container.DeploymentFinishedMessage,
	instanceID string,
	action string,
	endOfStream bool,
	logger core.Logger,
) {
	// Set EndOfStream in the message so clients can determine whether to keep listening
	msg.EndOfStream = endOfStream
	c.saveDeploymentEvent(
		ctx,
		eventTypeDeployFinished,
		&msg,
		msg.UpdateTimestamp,
		endOfStream,
		instanceID,
		action,
		logger,
	)
}

// AutoRollbackType specifies the type of automatic rollback to perform.
type AutoRollbackType int

const (
	// AutoRollbackTypeNone indicates no auto-rollback should be performed.
	AutoRollbackTypeNone AutoRollbackType = iota
	// AutoRollbackTypeDestroy indicates the failed deployment should be rolled back
	// by destroying partially-created resources (used for DeployFailed).
	AutoRollbackTypeDestroy
	// AutoRollbackTypeRevert indicates the failed update/destroy should be rolled back
	// by reverting to the previous state using a reverse changeset.
	AutoRollbackTypeRevert
)

// shouldTriggerAutoRollback determines if auto-rollback should be triggered
// based on the deployment finish status.
// Returns (true, AutoRollbackTypeDestroy) for failed new deployments that need destruction.
// Returns (true, AutoRollbackTypeRevert) for failed updates/destroys that need state reversal.
// Returns (false, AutoRollbackTypeNone) for all other statuses.
func shouldTriggerAutoRollback(status core.InstanceStatus) (bool, AutoRollbackType) {
	switch status {
	case core.InstanceStatusDeployFailed:
		// New deployment failed - destroy partially created resources
		return true, AutoRollbackTypeDestroy
	case core.InstanceStatusUpdateFailed:
		// Update failed - revert to previous state using reverse changeset
		return true, AutoRollbackTypeRevert
	case core.InstanceStatusDestroyFailed:
		// Destroy failed - recreate destroyed resources from previous state
		return true, AutoRollbackTypeRevert
	default:
		// Don't rollback for:
		// - Already rolling back statuses
		// - Successful statuses (Deployed, Updated, Destroyed)
		return false, AutoRollbackTypeNone
	}
}

// Initiates an automatic rollback after a deployment failure.
// For new instances, this destroys the partially created resources.
func (c *Controller) executeNewDeploymentRollback(
	ctx context.Context,
	instanceID string,
	changeset *manage.Changeset,
	failureReasons []string,
	logger core.Logger,
) {
	// Capture the pre-rollback state before destroying resources.
	// This allows users to see what state the deployment was in before rollback.
	c.capturePreRollbackState(ctx, instanceID, failureReasons, logger)

	// Create removal changes from the current instance state.
	// The original changeset has creation changes (NewResources, etc.) but
	// the destroy operation needs removal changes (RemovedResources, etc.).
	// Resources/links in failed or in-progress states are skipped.
	removalChanges, skippedItems, err := c.createRemovalChangesFromInstanceState(ctx, instanceID)
	if err != nil {
		logger.Error(
			"failed to create removal changes for auto-rollback",
			core.ErrorLogField("error", err),
		)
		return
	}

	if len(skippedItems) > 0 {
		c.logSkippedRollbackItems(skippedItems, instanceID, logger)
	}

	// Create a changeset with removal changes for the destroy operation.
	rollbackChangeset := &manage.Changeset{
		ID:      changeset.ID,
		Changes: removalChanges,
	}

	c.startDestroyRollback(rollbackChangeset, instanceID, skippedItems, logger)
}

// startDestroyRollback initiates a destroy operation for rollback with skipped items tracking.
func (c *Controller) startDestroyRollback(
	changeset *manage.Changeset,
	instanceID string,
	skippedItems []changes.SkippedRollbackItem,
	logger core.Logger,
) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		c.deploymentTimeout,
	)

	params := blueprint.CreateEmptyBlueprintParams()
	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		placeholderBlueprint,
		schema.YAMLSpecFormat,
		params,
	)
	if err != nil {
		cancel()
		c.handleDeploymentErrorAsEvent(
			ctxWithTimeout,
			instanceID,
			err,
			"rolling back deployment",
			logger,
		)
		return
	}

	channels := container.CreateDeployChannels()
	blueprintContainer.Destroy(
		ctxWithTimeout,
		&container.DestroyInput{
			InstanceID:             instanceID,
			Changes:                changeset.Changes,
			Rollback:               true,
			Force:                  false,
			TaggingConfig:          nil,
			ProviderMetadataLookup: pluginmeta.ToLookupFunc(c.providerMetadataLookup),
			DrainTimeout:           c.drainTimeout,
		},
		channels,
		params,
	)

	c.listenForDeploymentUpdatesWithParams(
		ctxWithTimeout,
		cancel,
		instanceID,
		"rolling back deployment",
		channels,
		changeset,
		false,
		nil,
		logger.Named("deployRollback"),
		&listenForDeploymentUpdatesParams{
			SkippedRollbackItems: skippedItems,
		},
	)
}

// Initiates an automatic rollback after an update or destroy failure.
// This generates a reverse changeset from the original changes and previous state,
// then deploys the reverse changes to restore the instance to its previous state.
// Resources and links that failed to complete their operations are skipped from
// rollback to avoid unpredictable behavior.
func (c *Controller) executeUpdateRollback(
	ctx context.Context,
	instanceID string,
	changeset *manage.Changeset,
	previousInstanceState *state.InstanceState,
	failureReasons []string,
	logger core.Logger,
) {
	if changeset == nil || changeset.Changes == nil {
		logger.Warn(
			"cannot execute update rollback: changeset or changes is nil",
			core.StringLogField("instanceId", instanceID),
		)
		return
	}

	if previousInstanceState == nil {
		logger.Warn(
			"cannot execute update rollback: previous instance state is nil",
			core.StringLogField("instanceId", instanceID),
		)
		return
	}

	// Capture the pre-rollback state before reverting changes.
	// This allows users to see what state the deployment was in before rollback.
	c.capturePreRollbackState(ctx, instanceID, failureReasons, logger)

	// Generate the reverse changeset to undo the original changes
	reverseChanges, err := changes.ReverseChangeset(changeset.Changes, previousInstanceState)
	if err != nil {
		logger.Error(
			"failed to generate reverse changeset for update rollback",
			core.ErrorLogField("error", err),
			core.StringLogField("instanceId", instanceID),
		)
		return
	}
	if reverseChanges == nil {
		logger.Warn(
			"cannot execute update rollback: reverse changeset generation returned nil",
			core.StringLogField("instanceId", instanceID),
		)
		return
	}

	// Fetch the current state (after failed deployment) to filter the reverse changeset.
	// Only resources/links in a completed state (Created, Updated, Destroyed, ConfigComplete)
	// will be included in the rollback to avoid unpredictable behavior.
	var skippedItems []changes.SkippedRollbackItem
	currentState, err := c.instances.Get(ctx, instanceID)
	if err != nil {
		logger.Warn(
			"failed to fetch current state for rollback filtering, proceeding with unfiltered rollback",
			core.ErrorLogField("error", err),
			core.StringLogField("instanceId", instanceID),
		)
	} else {
		filterResult := changes.FilterReverseChangesetByCurrentState(reverseChanges, &currentState)
		reverseChanges = filterResult.FilteredChanges
		skippedItems = filterResult.SkippedItems
		if filterResult.HasSkippedItems {
			c.logSkippedRollbackItems(skippedItems, instanceID, logger)
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		c.deploymentTimeout,
	)

	// Load the blueprint from the original changeset location
	blueprintContainer, err := c.blueprintLoader.Load(
		ctxWithTimeout,
		changeset.BlueprintLocation,
		blueprint.CreateEmptyBlueprintParams(),
	)
	if err != nil {
		cancel()
		logger.Error(
			"failed to load blueprint for update rollback",
			core.ErrorLogField("error", err),
			core.StringLogField("instanceId", instanceID),
			core.StringLogField("blueprintLocation", changeset.BlueprintLocation),
		)
		return
	}

	channels := container.CreateDeployChannels()
	err = blueprintContainer.Deploy(
		ctxWithTimeout,
		&container.DeployInput{
			InstanceID:   instanceID,
			InstanceName: previousInstanceState.InstanceName,
			Changes:      reverseChanges,
			Rollback:     true, // Mark as rollback operation
			// Tagging is not applied during rollback operations
			TaggingConfig:          nil,
			ProviderMetadataLookup: nil,
			DrainTimeout:           c.drainTimeout,
		},
		channels,
		blueprint.CreateEmptyBlueprintParams(),
	)
	if err != nil {
		cancel()
		logger.Error(
			"failed to start update rollback deployment",
			core.ErrorLogField("error", err),
			core.StringLogField("instanceId", instanceID),
		)
		return
	}

	// Listen for rollback deployment updates without triggering further auto-rollback
	c.listenForDeploymentUpdatesWithParams(
		ctxWithTimeout,
		cancel,
		instanceID,
		"rolling back update",
		channels,
		changeset,
		false, // autoRollback=false to prevent infinite rollback loops
		nil,   // no previous state needed for rollback of rollback
		logger.Named("updateRollback"),
		&listenForDeploymentUpdatesParams{
			SkippedRollbackItems: skippedItems,
		},
	)
}

func (c *Controller) handleDeploymentErrorAsEvent(
	ctx context.Context,
	instanceID string,
	deploymentError error,
	action string,
	logger core.Logger,
) {
	// In the case that the error is a validation error when loading the blueprint,
	// make sure that the specific errors are included in the event data.
	errDiagnostics := utils.DiagnosticsFromBlueprintValidationError(
		deploymentError,
		c.logger,
		/* fallbackToGeneralDiagnostic */ true,
	)

	errorMsgEvent := &errorMessageEvent{
		Message:     deploymentError.Error(),
		Diagnostics: errDiagnostics,
		Timestamp:   c.clock.Now().Unix(),
	}
	c.saveDeploymentEvent(
		ctx,
		eventTypeError,
		errorMsgEvent,
		errorMsgEvent.Timestamp,
		/* endOfStream */ true,
		instanceID,
		action,
		logger,
	)
}

func (c *Controller) saveDeploymentEvent(
	ctx context.Context,
	eventType string,
	data any,
	eventTimestamp int64,
	endOfStream bool,
	instanceID string,
	action string,
	logger core.Logger,
) {
	eventID, err := c.eventIDGenerator.GenerateID()
	if err != nil {
		logger.Error(
			"failed to generate a new event ID",
			core.ErrorLogField("error", err),
		)
		return
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		logger.Error(
			fmt.Sprintf("failed to marshal %q event", eventType),
			core.ErrorLogField("error", err),
		)
		return
	}

	err = c.eventStore.Save(
		ctx,
		&manage.Event{
			ID:          eventID,
			Type:        eventType,
			ChannelType: helpersv1.ChannelTypeDeployment,
			ChannelID:   instanceID,
			Data:        string(dataBytes),
			Timestamp:   eventTimestamp,
			End:         endOfStream,
		},
	)
	if err != nil {
		logger.Error(
			fmt.Sprintf(
				"failed to save event for %s",
				action,
			),
			core.ErrorLogField("error", err),
		)
		return
	}
}

// respondWithDriftBlocked sends a 409 Conflict response when an operation
// is blocked due to drift detection on the changeset.
func (c *Controller) respondWithDriftBlocked(
	ctx context.Context,
	w http.ResponseWriter,
	instanceID string,
	changeset *manage.Changeset,
) {
	// Lookup the reconciliation result from the separate store
	var result *container.ReconciliationCheckResult
	reconciliationResult, err := c.reconciliationResultsStore.GetLatestByChangesetID(ctx, changeset.ID)
	if err == nil && reconciliationResult != nil {
		result = reconciliationResult.Result
	}

	response := &DriftBlockedResponse{
		Message:              "Operation blocked due to drift detection. Reconciliation is required before proceeding.",
		InstanceID:           instanceID,
		ChangesetID:          changeset.ID,
		ReconciliationResult: result,
		Hint:                 "Use the reconciliation endpoints to review and resolve drift, or set force=true to bypass this check.",
	}
	httputils.HTTPJSONResponse(
		w,
		http.StatusConflict,
		response,
	)
}

// createTaggingConfig creates a provider.TaggingConfig from the request's
// BlueprintOperationConfig. Returns nil if tagging config provider is not configured.
func (c *Controller) createTaggingConfig(config *types.BlueprintOperationConfig) *provider.TaggingConfig {
	if c.taggingConfigProvider == nil {
		return nil
	}

	var taggingOpConfig *types.TaggingOperationConfig
	if config != nil {
		taggingOpConfig = config.Tagging
	}

	return c.taggingConfigProvider.CreateConfig(taggingOpConfig)
}
