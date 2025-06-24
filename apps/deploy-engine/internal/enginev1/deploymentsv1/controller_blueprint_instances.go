package deploymentsv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
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
func (c *Controller) UpdateBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceID := params["id"]

	instance, err := c.instances.Get(
		r.Context(),
		instanceID,
	)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceID)
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
func (c *Controller) StreamDeploymentEventsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	instanceID := params["id"]

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
func (c *Controller) GetBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceID := params["id"]

	instance, err := c.instances.Get(
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
		instance,
	)
}

// GetBlueprintInstanceExportsHandler is the handler for the
// GET /deployments/instances/{id}/exports endpoint that retrieves the
// exports of a blueprint instance.
func (c *Controller) GetBlueprintInstanceExportsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceID := params["id"]

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
func (c *Controller) DestroyBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	pathParams := mux.Vars(r)
	instanceID := pathParams["id"]

	instance, err := c.instances.Get(
		r.Context(),
		instanceID,
	)
	if err != nil {
		c.handleGetInstanceError(w, err, instanceID)
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

	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	go c.startDestroy(
		changeset,
		instance.InstanceID,
		payload.Rollback,
		params,
	)

	// The instance status will be updated by the deployment process
	// but we need to give an indicator to the caller that something
	// is happening in the response.
	instance.Status = core.InstanceStatusDestroying

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		instance,
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

	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	instanceID, err := c.startDeployment(
		blueprintInfo,
		changeset,
		getInstanceID(existingInstance),
		payload.Rollback,
		helpersv1.GetFormat(payload.BlueprintFile),
		params,
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

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		instance,
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
	forRollback bool,
	format schema.SpecFormat,
	params core.BlueprintParams,
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
			InstanceID: deployInstanceID,
			Changes:    changeset.Changes,
			Rollback:   forRollback,
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
	params core.BlueprintParams,
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
	}

	channels := container.CreateDeployChannels()
	blueprintContainer.Destroy(
		ctxWithTimeout,
		&container.DestroyInput{
			InstanceID: destroyInstanceID,
			Changes:    changeset.Changes,
			Rollback:   forRollback,
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
		c.logger.Named("destroy").WithFields(
			core.StringLogField("instanceId", destroyInstanceID),
		),
	)
}

func (c *Controller) listenForDeploymentUpdates(
	ctx context.Context,
	cancelCtx func(),
	instanceID string,
	action string,
	channels *container.DeployChannels,
	logger core.Logger,
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
			c.handleDeploymentFinishUpdateMessage(ctx, msg, instanceID, action, logger)
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

func (c *Controller) handleDeploymentFinishUpdateMessage(
	ctx context.Context,
	msg container.DeploymentFinishedMessage,
	instanceID string,
	action string,
	logger core.Logger,
) {
	c.saveDeploymentEvent(
		ctx,
		eventTypeDeployFinished,
		&msg,
		msg.UpdateTimestamp,
		/* endOfStream */ true,
		instanceID,
		action,
		logger,
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
