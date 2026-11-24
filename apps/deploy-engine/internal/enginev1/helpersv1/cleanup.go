package helpersv1

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// StartCleanup answers a request to clean up one category of state.
//
// Shared by every cleanup endpoint, so that they cannot drift from each other over what
// a refused or a failed start looks like. A category already being cleaned up, whether
// by another request or by the scheduler, is answered with a conflict rather than by
// starting a second run over the same records. An engine that is shutting down is
// answered as unavailable.
func StartCleanup(
	w http.ResponseWriter,
	r *http.Request,
	runner *cleanup.Runner,
	task cleanup.Task,
	logger core.Logger,
) {
	operation, err := runner.Start(r.Context(), task)
	if err != nil {
		if errors.Is(err, cleanup.ErrCleanupInProgress) {
			httputils.HTTPError(w, http.StatusConflict, err.Error())
			return
		}

		if errors.Is(err, cleanup.ErrShuttingDown) {
			httputils.HTTPError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		logger.Debug(
			"failed to start cleanup operation",
			core.ErrorLogField("error", err),
		)

		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&AsyncOperationResponse[*manage.CleanupOperation]{
			Data: operation,
		},
	)
}

// GetCleanupStatus answers a request for the status of a cleanup operation in one category.
//
// An operation from another category is answered as not found, so that each status route
// only reports on operations started through its own cleanup endpoint.
func GetCleanupStatus(
	w http.ResponseWriter,
	r *http.Request,
	store manage.CleanupOperations,
	cleanupType manage.CleanupType,
	logger core.Logger,
) {
	operationID := mux.Vars(r)["id"]

	operation, err := store.Get(r.Context(), operationID)
	notFoundErr := &manage.CleanupOperationNotFound{}
	if errors.As(err, &notFoundErr) || (err == nil && operation.CleanupType != cleanupType) {
		httputils.HTTPError(
			w,
			http.StatusNotFound,
			manage.CleanupOperationNotFoundError(operationID).Error(),
		)
		return
	}

	if err != nil {
		logger.Debug(
			"failed to get cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, operation)
}
