package deploymentsv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
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

// CreateChangesetHandler is the handler for the POST /deployments/changes
// endpoint that creates a new change set and starts the change staging process.
func (c *Controller) CreateChangesetHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	payload := &CreateChangesetRequestPayload{}
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

	changesetID, err := c.idGenerator.GenerateID()
	if err != nil {
		c.logger.Debug(
			"failed to generate a new change set ID",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	finalInstanceID, err := c.deriveInstanceID(r.Context(), payload)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			// For destroy operations with a non-existent instance,
			// return a 404 with the instance identifier in the message.
			identifier := payload.InstanceID
			if identifier == "" {
				identifier = payload.InstanceName
			}
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("instance %q not found", identifier),
			)
			return
		}
		c.logger.Debug(
			"failed to derive instance ID",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	blueprintLocation := resolve.BlueprintLocationString(&payload.BlueprintDocumentInfo)
	changeset := &manage.Changeset{
		ID:                changesetID,
		InstanceID:        finalInstanceID,
		Destroy:           payload.Destroy,
		Status:            manage.ChangesetStatusStarting,
		BlueprintLocation: blueprintLocation,
		Changes:           &changes.BlueprintChanges{},
		Created:           c.clock.Now().Unix(),
	}

	// Add blueprint directory to context variables for resolving relative child blueprint paths.
	finalConfig = internalutils.EnsureBlueprintDirContextVar(finalConfig, payload.BlueprintDocumentInfo.Directory)
	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)
	taggingConfig := c.createTaggingConfig(finalConfig)

	// Get the last event ID for the changeset channel before starting the async operation.
	// This allows clients to use it as a starting offset when streaming events.
	lastEventID, err := c.eventStore.GetLastEventID(
		r.Context(),
		helpersv1.ChannelTypeChangeset,
		changesetID,
	)
	if err != nil {
		c.logger.Debug(
			"failed to get last event ID for changeset channel",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	go c.startChangeStaging(
		changeset,
		blueprintInfo,
		helpersv1.GetFormat(payload.BlueprintFile),
		params,
		taggingConfig,
		payload.SkipDriftCheck,
		c.logger.Named("changeStagingProcess").WithFields(
			core.StringLogField("changesetId", changesetID),
			core.StringLogField("blueprintLocation", blueprintLocation),
		),
	)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[*manage.Changeset]{
			LastEventID: lastEventID,
			Data:        changeset,
		},
	)
}

// StreamChangesetEventsHandler is the handler for the GET /deployments/changes/{id}/stream endpoint
// that streams change staging events to the client using Server-Sent Events (SSE).
func (c *Controller) StreamChangesetEventsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	changesetID := params["id"]

	helpersv1.SSEStreamEvents(
		w,
		r,
		&helpersv1.StreamInfo{
			ChannelType: helpersv1.ChannelTypeChangeset,
			ChannelID:   changesetID,
		},
		c.eventStore,
		c.logger.Named("changeStagingStream").WithFields(
			core.StringLogField("changesetId", changesetID),
			core.StringLogField("eventChannelType", helpersv1.ChannelTypeChangeset),
		),
	)
}

// GetChangesetHandler is the handler for the GET /deployments/changes/{id} endpoint
// that retrieves a change set including its status and changes if available.
func (c *Controller) GetChangesetHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	changesetID := params["id"]

	changeset, err := c.changesetStore.Get(
		r.Context(),
		changesetID,
	)
	if err != nil {
		notFoundErr := &manage.ChangesetNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("change set %q not found", changesetID),
			)
			return
		}

		c.logger.Debug(
			"failed to get change set",
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
		http.StatusOK,
		changeset,
	)
}

// CleanupChangesetsHandler is the handler for the
// POST /deployments/changes/cleanup endpoint that cleans up
// change sets that are older than the configured
// retention period.
func (c *Controller) CleanupChangesetsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	operationID, err := c.idGenerator.GenerateID()
	if err != nil {
		c.logger.Debug(
			"failed to generate cleanup operation ID",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	cleanupBefore := c.clock.Now().Add(-c.changesetRetentionPeriod)

	operation := &manage.CleanupOperation{
		ID:            operationID,
		CleanupType:   manage.CleanupTypeChangesets,
		Status:        manage.CleanupOperationStatusRunning,
		StartedAt:     c.clock.Now().Unix(),
		ThresholdDate: cleanupBefore.Unix(),
	}

	if err := c.cleanupOperationsStore.Save(r.Context(), operation); err != nil {
		c.logger.Debug(
			"failed to save cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	// Copy the operation for the response to avoid data race between
	// json.Marshal and the goroutine modifying the original.
	responseCopy := *operation

	go c.cleanupChangesets(operation)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{
			Data: &responseCopy,
		},
	)
}

func (c *Controller) cleanupChangesets(operation *manage.CleanupOperation) {
	logger := c.logger.Named("changesetCleanup")

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		changesetCleanupTimeout,
	)
	defer cancel()

	thresholdDate := time.Unix(operation.ThresholdDate, 0)
	itemsDeleted, err := c.changesetStore.Cleanup(ctxWithTimeout, thresholdDate)

	operation.EndedAt = c.clock.Now().Unix()
	operation.ItemsDeleted = itemsDeleted

	if err != nil {
		logger.Error(
			"failed to clean up old change sets",
			core.ErrorLogField("error", err),
		)
		operation.Status = manage.CleanupOperationStatusFailed
		operation.ErrorMessage = err.Error()
	} else {
		operation.Status = manage.CleanupOperationStatusCompleted
	}

	if updateErr := c.cleanupOperationsStore.Update(ctxWithTimeout, operation); updateErr != nil {
		logger.Error(
			"failed to update cleanup operation",
			core.ErrorLogField("error", updateErr),
		)
	}
}

// GetChangesetsCleanupStatusHandler is the handler for the
// GET /deployments/changes/cleanup/{id} endpoint that retrieves the
// status of a cleanup operation.
func (c *Controller) GetChangesetsCleanupStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	operationID := params["id"]

	operation, err := c.cleanupOperationsStore.Get(r.Context(), operationID)
	if err != nil {
		notFoundErr := &manage.CleanupOperationNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				notFoundErr.Error(),
			)
			return
		}
		c.logger.Debug(
			"failed to get cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, operation)
}

// CleanupReconciliationResultsHandler is the handler for the
// POST /deployments/reconciliation-results/cleanup endpoint that cleans up
// reconciliation results that are older than the configured
// retention period.
func (c *Controller) CleanupReconciliationResultsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	operationID, err := c.idGenerator.GenerateID()
	if err != nil {
		c.logger.Debug(
			"failed to generate cleanup operation ID",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	cleanupBefore := c.clock.Now().Add(-c.reconciliationResultsRetentionPeriod)

	operation := &manage.CleanupOperation{
		ID:            operationID,
		CleanupType:   manage.CleanupTypeReconciliationResults,
		Status:        manage.CleanupOperationStatusRunning,
		StartedAt:     c.clock.Now().Unix(),
		ThresholdDate: cleanupBefore.Unix(),
	}

	if err := c.cleanupOperationsStore.Save(r.Context(), operation); err != nil {
		c.logger.Debug(
			"failed to save cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	// Copy the operation for the response to avoid data race between
	// json.Marshal and the goroutine modifying the original.
	responseCopy := *operation

	go c.cleanupReconciliationResults(operation)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{
			Data: &responseCopy,
		},
	)
}

func (c *Controller) cleanupReconciliationResults(operation *manage.CleanupOperation) {
	logger := c.logger.Named("reconciliationResultsCleanup")

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		reconciliationResultsCleanupTimeout,
	)
	defer cancel()

	thresholdDate := time.Unix(operation.ThresholdDate, 0)
	itemsDeleted, err := c.reconciliationResultsStore.Cleanup(ctxWithTimeout, thresholdDate)

	operation.EndedAt = c.clock.Now().Unix()
	operation.ItemsDeleted = itemsDeleted

	if err != nil {
		logger.Error(
			"failed to clean up old reconciliation results",
			core.ErrorLogField("error", err),
		)
		operation.Status = manage.CleanupOperationStatusFailed
		operation.ErrorMessage = err.Error()
	} else {
		operation.Status = manage.CleanupOperationStatusCompleted
	}

	if updateErr := c.cleanupOperationsStore.Update(ctxWithTimeout, operation); updateErr != nil {
		logger.Error(
			"failed to update cleanup operation",
			core.ErrorLogField("error", updateErr),
		)
	}
}

// GetReconciliationResultsCleanupStatusHandler is the handler for the
// GET /deployments/reconciliation-results/cleanup/{id} endpoint that retrieves the
// status of a cleanup operation.
func (c *Controller) GetReconciliationResultsCleanupStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	operationID := params["id"]

	operation, err := c.cleanupOperationsStore.Get(r.Context(), operationID)
	if err != nil {
		notFoundErr := &manage.CleanupOperationNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				notFoundErr.Error(),
			)
			return
		}
		c.logger.Debug(
			"failed to get cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, operation)
}

func (c *Controller) startChangeStaging(
	changeset *manage.Changeset,
	blueprintInfo *includes.ChildBlueprintInfo,
	format schema.SpecFormat,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
	skipDriftCheck bool,
	logger core.Logger,
) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		changeStagingTimeout,
	)
	defer cancel()

	earlyExitBefore := c.saveChangeset(
		ctxWithTimeout,
		changeset,
		manage.ChangesetStatusStagingChanges,
		logger,
	)
	if earlyExitBefore {
		return
	}

	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		helpersv1.GetBlueprintSource(blueprintInfo),
		format,
		params,
	)
	if err != nil {
		c.handleChangesetErrorAsEvent(
			ctxWithTimeout,
			changeset,
			err,
			logger,
		)
		return
	}

	// Perform drift check before staging changes if:
	// - skipDriftCheck is not set
	// - There is an existing instance to check (not a new deployment)
	if !skipDriftCheck && changeset.InstanceID != "" {
		driftDetected := c.performDriftCheckForChangeStaging(
			ctxWithTimeout,
			blueprintContainer,
			changeset,
			params,
			taggingConfig,
			logger,
		)
		if driftDetected {
			// Exit early - the drift check already handled saving the changeset
			// and sending the drift detected event
			return
		}
	}

	channels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		ctxWithTimeout,
		&container.StageChangesInput{
			InstanceID: changeset.InstanceID,
			Destroy:    changeset.Destroy,
		},
		channels,
		params,
	)
	if err != nil {
		c.handleChangesetErrorAsEvent(
			ctxWithTimeout,
			changeset,
			err,
			logger,
		)
		return
	}

	c.handleChangesetMessages(ctxWithTimeout, changeset, channels, logger)
}

func (c *Controller) handleChangesetMessages(
	ctx context.Context,
	changeset *manage.Changeset,
	channels *container.ChangeStagingChannels,
	logger core.Logger,
) {
	fullChanges := (*changes.BlueprintChanges)(nil)
	var err error
	for err == nil && fullChanges == nil {
		select {
		case msg := <-channels.ResourceChangesChan:
			c.handleChangesetResourceChangesMessage(ctx, msg, changeset, logger)
		case msg := <-channels.ChildChangesChan:
			c.handleChangesetChildChangesMessage(ctx, msg, changeset, logger)
		case msg := <-channels.LinkChangesChan:
			c.handleChangesetLinkChangesMessage(ctx, msg, changeset, logger)
		case changes := <-channels.CompleteChan:
			c.handleChangesetCompleteMessage(ctx, &changes, changeset, logger)
			fullChanges = &changes
		case err = <-channels.ErrChan:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	if err != nil {
		c.handleChangesetErrorAsEvent(
			ctx,
			changesetWithStatus(
				changeset,
				manage.ChangesetStatusFailed,
			),
			err,
			logger,
		)
		return
	}

	c.saveChangeset(
		ctx,
		changesetWithChanges(
			changeset,
			fullChanges,
		),
		manage.ChangesetStatusChangesStaged,
		logger,
	)
}

func (c *Controller) handleChangesetResourceChangesMessage(
	ctx context.Context,
	msg container.ResourceChangesMessage,
	changeset *manage.Changeset,
	logger core.Logger,
) {
	eventData := &resourceChangesEventWithTimestamp{
		ResourceChangesMessage: msg,
		Timestamp:              c.clock.Now().Unix(),
	}
	c.saveChangeStagingEvent(
		ctx,
		eventTypeResourceChanges,
		eventData,
		eventData.Timestamp,
		/* endOfStream */ false,
		changeset,
		logger,
	)
}

func (c *Controller) handleChangesetChildChangesMessage(
	ctx context.Context,
	msg container.ChildChangesMessage,
	changeset *manage.Changeset,
	logger core.Logger,
) {
	eventData := &childChangesEventWithTimestamp{
		ChildChangesMessage: msg,
		Timestamp:           c.clock.Now().Unix(),
	}
	c.saveChangeStagingEvent(
		ctx,
		eventTypeChildChanges,
		eventData,
		eventData.Timestamp,
		/* endOfStream */ false,
		changeset,
		logger,
	)
}

func (c *Controller) handleChangesetLinkChangesMessage(
	ctx context.Context,
	msg container.LinkChangesMessage,
	changeset *manage.Changeset,
	logger core.Logger,
) {
	eventData := &linkChangesEventWithTimestamp{
		LinkChangesMessage: msg,
		Timestamp:          c.clock.Now().Unix(),
	}
	c.saveChangeStagingEvent(
		ctx,
		eventTypeLinkChanges,
		eventData,
		eventData.Timestamp,
		/* endOfStream */ false,
		changeset,
		logger,
	)
}

func (c *Controller) handleChangesetCompleteMessage(
	ctx context.Context,
	changes *changes.BlueprintChanges,
	changeset *manage.Changeset,
	logger core.Logger,
) {
	eventData := &changeStagingCompleteEvent{
		Changes:   changes,
		Timestamp: c.clock.Now().Unix(),
	}
	c.saveChangeStagingEvent(
		ctx,
		eventTypeChangeStagingComplete,
		eventData,
		eventData.Timestamp,
		/* endOfStream */ true,
		changeset,
		logger,
	)
}

func (c *Controller) handleChangesetErrorAsEvent(
	ctx context.Context,
	changeset *manage.Changeset,
	changeStagingError error,
	logger core.Logger,
) {
	// In the case that the error is a validation error when loading the blueprint,
	// make sure that the specific errors are included in the event data.
	errDiagnostics := utils.DiagnosticsFromBlueprintValidationError(
		changeStagingError,
		c.logger,
		/* fallbackToGeneralDiagnostic */ true,
	)

	errorMsgEvent := &errorMessageEvent{
		Message:     changeStagingError.Error(),
		Diagnostics: errDiagnostics,
		Timestamp:   c.clock.Now().Unix(),
	}
	c.saveChangeStagingEvent(
		ctx,
		eventTypeError,
		errorMsgEvent,
		errorMsgEvent.Timestamp,
		/* endOfStream */ true,
		changeset,
		logger,
	)

	c.saveChangeset(
		ctx,
		changeset,
		manage.ChangesetStatusFailed,
		logger,
	)
}

func (c *Controller) saveChangeStagingEvent(
	ctx context.Context,
	eventType string,
	data any,
	eventTimestamp int64,
	endOfStream bool,
	changeset *manage.Changeset,
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
			ChannelType: helpersv1.ChannelTypeChangeset,
			ChannelID:   changeset.ID,
			Data:        string(dataBytes),
			Timestamp:   eventTimestamp,
			End:         endOfStream,
		},
	)
	if err != nil {
		logger.Error(
			"failed to save event for change staging",
			core.ErrorLogField("error", err),
		)
		return
	}
}

func (c *Controller) saveChangeset(
	ctx context.Context,
	changeset *manage.Changeset,
	status manage.ChangesetStatus,
	logger core.Logger,
) (earlyExit bool) {
	err := c.changesetStore.Save(
		ctx,
		changesetWithStatus(
			changeset,
			status,
		),
	)
	if err != nil {
		logger.Error(
			"failed to save change set",
			core.ErrorLogField("error", err),
		)
		return true
	}

	return false
}

func (c *Controller) deriveInstanceID(
	ctx context.Context,
	payload *CreateChangesetRequestPayload,
) (string, error) {
	if payload.InstanceID != "" {
		return payload.InstanceID, nil
	}

	if payload.InstanceID == "" && payload.InstanceName != "" {
		instanceID, err := c.instances.LookupIDByName(ctx, payload.InstanceName)
		if err != nil {
			if state.IsInstanceNotFound(err) {
				// For destroy operations, the instance must exist.
				// Return the error with the instance name for a helpful message.
				if payload.Destroy {
					return "", state.InstanceNotFoundError(payload.InstanceName)
				}
				// For non-destroy operations, this is a new deployment.
				// Return empty string to indicate no existing instance.
				return "", nil
			}
			return "", err
		}
		return instanceID, nil
	}

	// If no instance ID or name is provided, then there is no
	// existing instance to generate the change set against.
	return "", nil
}

func changesetWithChanges(
	changeset *manage.Changeset,
	changes *changes.BlueprintChanges,
) *manage.Changeset {
	return &manage.Changeset{
		ID:                changeset.ID,
		InstanceID:        changeset.InstanceID,
		Destroy:           changeset.Destroy,
		Status:            changeset.Status,
		BlueprintLocation: changeset.BlueprintLocation,
		Changes:           changes,
		Created:           changeset.Created,
	}
}

func changesetWithStatus(
	changeset *manage.Changeset,
	status manage.ChangesetStatus,
) *manage.Changeset {
	return &manage.Changeset{
		ID:                changeset.ID,
		InstanceID:        changeset.InstanceID,
		Destroy:           changeset.Destroy,
		Status:            status,
		BlueprintLocation: changeset.BlueprintLocation,
		Changes:           changeset.Changes,
		Created:           changeset.Created,
	}
}

// performDriftCheckForChangeStaging performs a reconciliation check before
// staging changes to detect drift or interrupted state.
// Returns true if drift/interrupted state was detected and the caller should
// exit early (the method handles saving the changeset and sending events).
func (c *Controller) performDriftCheckForChangeStaging(
	ctx context.Context,
	blueprintContainer container.BlueprintContainer,
	changeset *manage.Changeset,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
	logger core.Logger,
) bool {
	logger.Debug("performing drift check before change staging")

	result, err := blueprintContainer.CheckReconciliation(
		ctx,
		&container.CheckReconciliationInput{
			InstanceID:    changeset.InstanceID,
			Scope:         container.ReconciliationScopeAll,
			TaggingConfig: taggingConfig,
		},
		params,
	)
	if err != nil {
		// If the drift check fails, treat it as an error and fail the changeset
		c.handleChangesetErrorAsEvent(
			ctx,
			changeset,
			fmt.Errorf("failed to check for drift: %w", err),
			logger,
		)
		return true
	}

	// Check if any drift or interrupted state was detected
	if !result.HasDrift && !result.HasInterrupted {
		logger.Debug("no drift or interrupted state detected, proceeding with change staging")
		return false
	}

	// Drift or interrupted state detected - block the change staging
	logger.Info(
		"drift or interrupted state detected, blocking change staging",
		core.BoolLogField("hasDrift", result.HasDrift),
		core.BoolLogField("hasInterrupted", result.HasInterrupted),
	)

	c.handleDriftDetectedForChangeset(ctx, changeset, result, logger)
	return true
}

// handleDriftDetectedForChangeset saves the changeset with DRIFT_DETECTED status,
// saves the reconciliation result to the separate store, and sends a driftDetected
// event to the stream.
func (c *Controller) handleDriftDetectedForChangeset(
	ctx context.Context,
	changeset *manage.Changeset,
	reconciliationResult *container.ReconciliationCheckResult,
	logger core.Logger,
) {
	message := "Drift or interrupted state detected. Reconciliation " +
		"required before staging changes."
	if reconciliationResult.HasDrift && !reconciliationResult.HasInterrupted {
		message = "Drift detected. External changes to resources require " +
			"reconciliation before staging changes."
	} else if !reconciliationResult.HasDrift && reconciliationResult.HasInterrupted {
		message = "Interrupted state detected. Resources in interrupted" +
			" states require reconciliation before staging changes."
	}

	eventData := &driftDetectedEvent{
		Message:              message,
		ReconciliationResult: reconciliationResult,
		Timestamp:            c.clock.Now().Unix(),
	}

	c.saveChangeStagingEvent(
		ctx,
		eventTypeDriftDetected,
		eventData,
		eventData.Timestamp,
		/* endOfStream */ true,
		changeset,
		logger,
	)

	// Save the reconciliation result to the separate store
	resultID, err := c.idGenerator.GenerateID()
	if err != nil {
		logger.Error(
			"failed to generate reconciliation result ID",
			core.ErrorLogField("error", err),
		)
	} else {
		err = c.reconciliationResultsStore.Save(ctx, &manage.ReconciliationResult{
			ID:          resultID,
			ChangesetID: changeset.ID,
			InstanceID:  changeset.InstanceID,
			Result:      reconciliationResult,
			Created:     c.clock.Now().Unix(),
		})
		if err != nil {
			logger.Error(
				"failed to save reconciliation result",
				core.ErrorLogField("error", err),
			)
		}
	}

	// Save the changeset with DRIFT_DETECTED status
	c.saveChangeset(
		ctx,
		changeset,
		manage.ChangesetStatusDriftDetected,
		logger,
	)
}
