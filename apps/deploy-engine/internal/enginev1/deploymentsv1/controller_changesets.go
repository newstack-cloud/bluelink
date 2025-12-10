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
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	resolverfs "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
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
	finalConfig = ensureBlueprintDirContextVar(finalConfig, payload.BlueprintDocumentInfo.Directory)
	params := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	go c.startChangeStaging(
		changeset,
		blueprintInfo,
		helpersv1.GetFormat(payload.BlueprintFile),
		params,
		c.logger.Named("changeStagingProcess").WithFields(
			core.StringLogField("changesetId", changesetID),
			core.StringLogField("blueprintLocation", blueprintLocation),
		),
	)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		changeset,
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
	// Carry out the cleanup process in a separate goroutine
	// to avoid blocking the request,
	// general clean up should be a task that a client can trigger
	// but not need to wait for.
	go c.cleanupChangesets()

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.MessageResponse{
			Message: "Cleanup started",
		},
	)
}

func (c *Controller) cleanupChangesets() {
	logger := c.logger.Named("changesetCleanup")

	cleanupBefore := c.clock.Now().Add(
		-c.changesetRetentionPeriod,
	)

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		changesetCleanupTimeout,
	)
	defer cancel()

	err := c.changesetStore.Cleanup(
		ctxWithTimeout,
		cleanupBefore,
	)
	if err != nil {
		logger.Error(
			"failed to clean up old change sets",
			core.ErrorLogField("error", err),
		)
		return
	}
}

func (c *Controller) startChangeStaging(
	changeset *manage.Changeset,
	blueprintInfo *includes.ChildBlueprintInfo,
	format schema.SpecFormat,
	params core.BlueprintParams,
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
		return c.instances.LookupIDByName(ctx, payload.InstanceName)
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

// ensureBlueprintDirContextVar adds the blueprint directory to the context variables
// so that the file system resolver can resolve relative child blueprint paths.
// If the config is nil, a new config is created with just the context variable.
// Returns the config (possibly newly created) with the blueprint directory set.
func ensureBlueprintDirContextVar(config *types.BlueprintOperationConfig, directory string) *types.BlueprintOperationConfig {
	if directory == "" {
		return config
	}

	if config == nil {
		config = &types.BlueprintOperationConfig{}
	}

	if config.ContextVariables == nil {
		config.ContextVariables = make(map[string]*core.ScalarValue)
	}

	config.ContextVariables[resolverfs.BlueprintDirectoryContextVar] = &core.ScalarValue{
		StringValue: &directory,
	}

	return config
}
