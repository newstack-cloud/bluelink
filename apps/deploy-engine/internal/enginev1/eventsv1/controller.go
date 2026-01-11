package eventsv1

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// An internal timeout used for the cleanup process
	// that cleans up old events.
	// 10 minutes is a reasonable time to wait for the cleanup process
	// to complete for instances of the deploy engine with a lot of use.
	eventsCleanupTimeout = 10 * time.Minute
)

// Controller handles HTTP requests
// for managing events.
type Controller struct {
	eventsRetentionPeriod  time.Duration
	eventStore             manage.Events
	cleanupOperationsStore manage.CleanupOperations
	idGenerator            core.IDGenerator
	clock                  commoncore.Clock
	logger                 core.Logger
}

// NewController creates a new events Controller
// instance with the provided dependencies.
func NewController(
	eventsRetentionPeriod time.Duration,
	deps *typesv1.Dependencies,
) *Controller {
	return &Controller{
		eventsRetentionPeriod:  eventsRetentionPeriod,
		eventStore:             deps.EventStore,
		cleanupOperationsStore: deps.CleanupOperationsStore,
		idGenerator:            deps.IDGenerator,
		clock:                  deps.Clock,
		logger:                 deps.Logger,
	}
}

// CleanupEventsHandler is the handler for the
// POST /events/cleanup endpoint that cleans up
// events that are older than the configured
// retention period.
func (c *Controller) CleanupEventsHandler(
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

	cleanupBefore := c.clock.Now().Add(-c.eventsRetentionPeriod)

	operation := &manage.CleanupOperation{
		ID:            operationID,
		CleanupType:   manage.CleanupTypeEvents,
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

	go c.cleanupEvents(operation)

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{
			Data: &responseCopy,
		},
	)
}

func (c *Controller) cleanupEvents(operation *manage.CleanupOperation) {
	logger := c.logger.Named("eventsCleanup")

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		eventsCleanupTimeout,
	)
	defer cancel()

	thresholdDate := time.Unix(operation.ThresholdDate, 0)
	itemsDeleted, err := c.eventStore.Cleanup(ctxWithTimeout, thresholdDate)

	operation.EndedAt = c.clock.Now().Unix()
	operation.ItemsDeleted = itemsDeleted

	if err != nil {
		logger.Error(
			"failed to clean up old events",
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

// GetCleanupStatusHandler is the handler for the
// GET /events/cleanup/{id} endpoint that retrieves the
// status of a cleanup operation.
func (c *Controller) GetCleanupStatusHandler(
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
