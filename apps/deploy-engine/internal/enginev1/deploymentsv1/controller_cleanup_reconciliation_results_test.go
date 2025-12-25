package deploymentsv1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
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

	response := map[string]string{}
	err = json.Unmarshal(respData, &response)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	s.Assert().Equal(
		"Cleanup started",
		response["message"],
	)
}
