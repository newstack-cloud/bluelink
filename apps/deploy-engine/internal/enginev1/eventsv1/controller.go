package eventsv1

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Controller handles HTTP requests
// for managing events.
type Controller struct {
	eventsRetentionPeriod  time.Duration
	cleanupRunner          *cleanup.Runner
	eventStore             manage.Events
	cleanupOperationsStore manage.CleanupOperations
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
		cleanupRunner:          deps.CleanupRunner,
		eventStore:             deps.EventStore,
		cleanupOperationsStore: deps.CleanupOperationsStore,
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
	helpersv1.StartCleanup(
		w,
		r,
		c.cleanupRunner,
		cleanup.EventsTask(c.eventStore, c.eventsRetentionPeriod),
		c.requestLogger(r),
	)
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
		c.requestLogger(r).Debug(
			"failed to get cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, operation)
}

// The logger for the request being answered, carrying its correlation ID, source
// address and who it was authenticated for.
//
// Falls back to the process logger when the request-context middleware has not run, so
// that a handler never has to decide whether logging is worth attempting.
func (c *Controller) requestLogger(r *http.Request) core.Logger {
	return httpmw.LoggerFromContext(r.Context(), c.logger)
}
