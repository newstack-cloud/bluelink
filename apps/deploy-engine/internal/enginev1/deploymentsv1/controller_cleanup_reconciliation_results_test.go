package deploymentsv1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

func (s *ControllerTestSuite) Test_cleanup_reconciliation_results_handler() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/reconciliation-results/cleanup",
		s.ctrl.CleanupReconciliationResultsHandler,
	).Methods("POST")

	req := httptest.NewRequest("POST", "/deployments/reconciliation-results/cleanup", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	response := helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{}
	err = json.Unmarshal(respData, &response)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	s.Assert().Contains(
		[]manage.CleanupOperationStatus{
			manage.CleanupOperationStatusRunning,
			manage.CleanupOperationStatusCompleted,
		},
		response.Data.Status,
		"cleanup operation status should be running or completed",
	)
	s.Assert().Equal(manage.CleanupTypeReconciliationResults, response.Data.CleanupType)
	s.Assert().NotEmpty(response.Data.ID)
}
