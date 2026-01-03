package deploymentsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

const (
	testChangesetID        = "bf8de86f-5762-4c59-a878-c17fc93b7651"
	testDestroyChangesetID = "a1b2c3d4-5762-4c59-a878-c17fc93b7651"
	nonExistentChangesetID = "13766f10-f82d-4441-a887-e1b6da8028ba"
)

func (s *ControllerTestSuite) Test_get_changeset_handler() {
	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes/{id}",
		s.ctrl.GetChangesetHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/changes/%s", testChangesetID)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	changeset := &manage.Changeset{}
	err = json.Unmarshal(respData, changeset)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(
		testChangesetID,
		changeset.ID,
	)
	s.Assert().Equal(
		manage.ChangesetStatusStarting,
		changeset.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		changeset.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		changeset.Created,
	)
	s.Assert().Equal(
		&changes.BlueprintChanges{
			RemovedResources: []string{"resource1", "resource2"},
		},
		changeset.Changes,
	)
}

func (s *ControllerTestSuite) Test_get_changeset_handler_returns_404_not_found() {
	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes/{id}",
		s.ctrl.GetChangesetHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/changes/%s", nonExistentChangesetID)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	responseError := map[string]string{}
	err = json.Unmarshal(respData, &responseError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusNotFound, result.StatusCode)
	s.Assert().Equal(
		fmt.Sprintf(
			"change set %q not found",
			nonExistentChangesetID,
		),
		responseError["message"],
	)
}

func (s *ControllerTestSuite) saveTestChangeset() error {
	changeset := &manage.Changeset{
		ID:                testChangesetID,
		Status:            manage.ChangesetStatusStarting,
		BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
		Created:           testTime.Unix(),
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"resource1", "resource2"},
		},
	}

	err := s.changesetStore.Save(
		context.Background(),
		changeset,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *ControllerTestSuite) saveDestroyChangeset() error {
	changeset := &manage.Changeset{
		ID:                testDestroyChangesetID,
		Status:            manage.ChangesetStatusStarting,
		BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
		Created:           testTime.Unix(),
		Destroy:           true,
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"resource1", "resource2"},
		},
	}

	return s.changesetStore.Save(
		context.Background(),
		changeset,
	)
}
