package validationv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/httputils"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/params"
	"github.com/two-hundred/celerity/apps/deploy-engine/internal/resolve"
	"github.com/two-hundred/celerity/apps/deploy-engine/utils"
	"github.com/two-hundred/celerity/libs/blueprint-state/manage"
	"github.com/two-hundred/celerity/libs/blueprint/container"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/includes"
	"github.com/two-hundred/celerity/libs/blueprint/schema"
	commoncore "github.com/two-hundred/celerity/libs/common/core"
)

const (
	// An internal timeout used for the background goroutine
	// that performs validation.
	// 5 minutes allows for provider or transformer plugins
	// that may take a while to respond if network requests are involved.
	// Examples of this could include network requests involved in custom
	// validation or to retrieve options for a custom variable type.
	// It may be worth making this configurable in the future.
	validateTimeout = 5 * time.Minute
	// An internal timeout used for the cleanup process
	// that cleans up old blueprint validations.
	// 10 minutes is a reasonable time to wait for the cleanup process
	// to complete for instances of the deploy engine with a lot of use.
	cleanupTimeout = 10 * time.Minute
)

const (
	eventTypeDiagnostic = "diagnostic"
)

// Controller handles validation-related HTTP requests
// including streaming validation events over Server-Sent Events (SSE).
type Controller struct {
	validationRetentionPeriod time.Duration
	eventStore                manage.Events
	validationStore           manage.Validation
	idGenerator               core.IDGenerator
	eventIDGenerator          core.IDGenerator
	blueprintLoader           container.Loader
	// Behaviour used to resolve child blueprints in the blueprint container
	// package is reused to load the "root" blueprints from multiple sources.
	blueprintResolver includes.ChildResolver
	// A source of parameters that are passed into the blueprint loader
	// for validating a source blueprint document.
	// This is useful for providing plugin-specific configuration
	// when validating a blueprint.
	paramsProvider params.Provider
	clock          commoncore.Clock
	logger         core.Logger
}

// NewController creates a new validation Controller instance
// with the provided dependencies.
func NewController(
	validationRetentionPeriod time.Duration,
	deps *typesv1.Dependencies,
) *Controller {
	return &Controller{
		validationRetentionPeriod: validationRetentionPeriod,
		eventStore:                deps.EventStore,
		validationStore:           deps.ValidationStore,
		idGenerator:               deps.IDGenerator,
		eventIDGenerator:          deps.EventIDGenerator,
		blueprintLoader:           deps.ValidationLoader,
		blueprintResolver:         deps.BlueprintResolver,
		paramsProvider:            deps.ParamsProvider,
		clock:                     deps.Clock,
		logger:                    deps.Logger,
	}
}

// CreateBlueprintValidationHandler is the handler for the POST /validation endpoint
// that creates a new validation for a blueprint.
func (c *Controller) CreateBlueprintValidationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	payload := &CreateValidationRequestPayload{}
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

	blueprintValidationID, err := c.idGenerator.GenerateID()
	if err != nil {
		c.logger.Debug(
			"failed to generate a new blueprint validation ID",
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
	blueprintValidation := &manage.BlueprintValidation{
		ID:                blueprintValidationID,
		Status:            manage.BlueprintValidationStatusStarting,
		BlueprintLocation: blueprintLocation,
		Created:           c.clock.Now().Unix(),
	}

	params := c.paramsProvider.CreateFromRequestConfig(
		payload.Config,
	)

	go c.startValidationStream(
		blueprintValidation,
		blueprintInfo,
		helpersv1.GetFormat(payload.BlueprintFile),
		params,
		c.logger.Named("validationProcess").WithFields(
			core.StringLogField("blueprintValidationId", blueprintValidationID),
			core.StringLogField("blueprintLocation", blueprintLocation),
		),
	)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		blueprintValidation,
	)
}

// StreamEventsHandler is the handler for the GET /validation/{id}/stream endpoint
// that streams validation events to the client using Server-Sent Events (SSE).
func (c *Controller) StreamEventsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	validationID := params["id"]

	helpersv1.SSEStreamEvents(
		w,
		r,
		&helpersv1.StreamInfo{
			ChannelType: helpersv1.ChannelTypeValidation,
			ChannelID:   validationID,
		},
		c.eventStore,
		c.logger.Named("validationStream").WithFields(
			core.StringLogField("validationId", validationID),
			core.StringLogField("eventChannelType", helpersv1.ChannelTypeValidation),
		),
	)
}

// GetBlueprintValidationHandler is the handler for the GET /validation/{id} endpoint
// that retrieves metadata and status of a blueprint validation.
func (c *Controller) GetBlueprintValidationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	validationID := params["id"]

	blueprintValidation, err := c.validationStore.Get(
		r.Context(),
		validationID,
	)
	if err != nil {
		notFoundErr := &manage.BlueprintValidationNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("blueprint validation %q not found", validationID),
			)
			return
		}

		c.logger.Debug(
			"failed to get blueprint validation",
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
		blueprintValidation,
	)
}

// CleanupBlueprintValidationsHandler is the handler for the
// POST /validation/cleanup endpoint that cleans up
// blueprint validations that are older than the configured
// retention period.
func (c *Controller) CleanupBlueprintValidationsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Carry out the cleanup process in a separate goroutine
	// to avoid blocking the request,
	// general clean up should be a task that a client can trigger
	// but not need to wait for.
	go c.cleanup()

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.MessageResponse{
			Message: "Cleanup started",
		},
	)
}

func (c *Controller) cleanup() {
	logger := c.logger.Named("validationCleanup")

	cleanupBefore := c.clock.Now().Add(
		-c.validationRetentionPeriod,
	)

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		cleanupTimeout,
	)
	defer cancel()

	err := c.validationStore.Cleanup(
		ctxWithTimeout,
		cleanupBefore,
	)
	if err != nil {
		logger.Error(
			"failed to clean up old blueprint validations",
			core.ErrorLogField("error", err),
		)
		return
	}
}

func (c *Controller) startValidationStream(
	blueprintValidation *manage.BlueprintValidation,
	blueprintInfo *includes.ChildBlueprintInfo,
	format schema.SpecFormat,
	params core.BlueprintParams,
	logger core.Logger,
) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		validateTimeout,
	)
	defer cancel()

	earlyExitBefore := c.saveBlueprintValidation(
		ctxWithTimeout,
		blueprintValidation,
		manage.BlueprintValidationStatusRunning,
		logger,
	)
	if earlyExitBefore {
		return
	}

	// The error returned here will be converted into a diagnostic event
	// and streamed to the client.
	validationResult, err := c.blueprintLoader.ValidateString(
		ctxWithTimeout,
		helpersv1.GetBlueprintSource(blueprintInfo),
		format,
		params,
	)

	validationStatus := determineValidationStatus(validationResult, err)
	earlyExitAfter := c.saveBlueprintValidation(
		ctxWithTimeout,
		blueprintValidation,
		validationStatus,
		logger,
	)
	if earlyExitAfter {
		return
	}

	c.prepareAndSaveEvents(
		ctxWithTimeout,
		blueprintValidation,
		validationResult,
		err,
		logger,
	)
}

func (c *Controller) saveBlueprintValidation(
	ctx context.Context,
	blueprintValidation *manage.BlueprintValidation,
	status manage.BlueprintValidationStatus,
	logger core.Logger,
) (earlyExit bool) {
	err := c.validationStore.Save(
		ctx,
		blueprintValidationWithStatus(
			blueprintValidation,
			status,
		),
	)
	if err != nil {
		logger.Error(
			"failed to save blueprint validation",
			core.ErrorLogField("error", err),
		)
		return true
	}

	return false
}

func (c *Controller) prepareAndSaveEvents(
	ctx context.Context,
	blueprintValidation *manage.BlueprintValidation,
	validationResult *container.ValidationResult,
	err error,
	logger core.Logger,
) {
	// Validation errors are converted to diagnostics to provide a consistent
	// experience for the user, the only errors that should be returned are failures
	// outside of the validation process.
	errDiagnostics := utils.DiagnosticsFromBlueprintValidationError(
		err,
		c.logger,
		/* fallbackToGeneralDiagnostic */ true,
	)

	allDiagnostics := append(
		validationResult.Diagnostics,
		errDiagnostics...,
	)

	currentTimestamp := c.clock.Now().Unix()

	for i, diagnostic := range allDiagnostics {
		isEnd := i == len(allDiagnostics)-1
		diagWithTimestamp := diagnosticWithTimestamp{
			Diagnostic: *diagnostic,
			Timestamp:  currentTimestamp,
			End:        isEnd,
		}
		serialisedDiagnostic, err := json.Marshal(diagWithTimestamp)
		if err != nil {
			logger.Error(
				"failed to marshal diagnostic for saving event",
				core.ErrorLogField("error", err),
			)
			continue
		}

		eventID, err := c.eventIDGenerator.GenerateID()
		if err != nil {
			logger.Error(
				"failed to generate event ID for validation diagnostic",
				core.ErrorLogField("error", err),
			)
			continue
		}

		err = c.eventStore.Save(
			ctx,
			&manage.Event{
				ID:          eventID,
				Type:        eventTypeDiagnostic,
				ChannelType: helpersv1.ChannelTypeValidation,
				ChannelID:   blueprintValidation.ID,
				Data:        string(serialisedDiagnostic),
				Timestamp:   currentTimestamp,
				End:         isEnd,
			},
		)
		if err != nil {
			logger.Error(
				"failed to save event for validation diagnostic",
				core.ErrorLogField("error", err),
				core.StringLogField("eventId", eventID),
			)
		}
	}
}

func determineValidationStatus(
	validationResult *container.ValidationResult,
	err error,
) manage.BlueprintValidationStatus {
	if err != nil {
		return manage.BlueprintValidationStatusFailed
	}

	diagnostics := getDiagnostics(validationResult)

	hasErrorDiagnostic := slices.ContainsFunc(
		diagnostics,
		func(diagnostic *core.Diagnostic) bool {
			return diagnostic.Level == core.DiagnosticLevelError
		},
	)
	if hasErrorDiagnostic {
		return manage.BlueprintValidationStatusFailed
	}

	return manage.BlueprintValidationStatusValidated
}

func getDiagnostics(
	validationResult *container.ValidationResult,
) []*core.Diagnostic {
	if validationResult == nil {
		return []*core.Diagnostic{}
	}

	return validationResult.Diagnostics
}

func blueprintValidationWithStatus(
	blueprintValidation *manage.BlueprintValidation,
	status manage.BlueprintValidationStatus,
) *manage.BlueprintValidation {
	return &manage.BlueprintValidation{
		ID:                blueprintValidation.ID,
		Status:            status,
		BlueprintLocation: blueprintValidation.BlueprintLocation,
		Created:           blueprintValidation.Created,
	}
}
