package statev1

import (
	"errors"
	"net/http"

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

// Controller handles HTTP requests for maintaining the state backend itself,
// rather than any one category of state.
type Controller struct {
	cleanupRunner          *cleanup.Runner
	housekeeping           manage.Housekeeping
	cleanupOperationsStore manage.CleanupOperations
	logger                 core.Logger
}

// NewController creates a new state maintenance Controller
// instance with the provided dependencies.
func NewController(deps *typesv1.Dependencies) *Controller {
	return &Controller{
		cleanupRunner:          deps.CleanupRunner,
		housekeeping:           deps.StateHousekeeping,
		cleanupOperationsStore: deps.CleanupOperationsStore,
		logger:                 deps.Logger,
	}
}

// HousekeepingHandler is the handler for the
// POST /state/housekeeping endpoint that carries out
// the upkeep the state backend needs on its own storage,
// such as reclaiming storage that interrupted writes left
// behind. What that involves is up to the backend.
//
// A backend that needs no housekeeping still starts an
// operation, which completes having deleted nothing, so
// callers need not know which storage engine is in use.
func (c *Controller) HousekeepingHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	helpersv1.StartCleanup(
		w,
		r,
		c.cleanupRunner,
		cleanup.StateHousekeepingTask(c.housekeeping),
		c.requestLogger(r),
	)
}

// GetCleanupStatusHandler is the handler for the
// GET /state/housekeeping/{id} endpoint that retrieves
// the status of a housekeeping operation.
func (c *Controller) GetCleanupStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	operationID := mux.Vars(r)["id"]

	operation, err := c.cleanupOperationsStore.Get(r.Context(), operationID)
	if err != nil {
		notFoundErr := &manage.CleanupOperationNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(w, http.StatusNotFound, notFoundErr.Error())
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
func (c *Controller) requestLogger(r *http.Request) core.Logger {
	return httpmw.LoggerFromContext(r.Context(), c.logger)
}
