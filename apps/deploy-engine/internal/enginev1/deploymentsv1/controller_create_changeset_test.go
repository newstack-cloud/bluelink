package deploymentsv1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/r3labs/sse/v2"
)

func (s *ControllerTestSuite) Test_create_changeset_handler() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrl.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	changeset := &manage.Changeset{}
	err = json.Unmarshal(respData, changeset)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(changeset.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
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

	expectedEvents := changeStagingEventSequence()
	actualEvents, err := s.streamChangeStagingEvents(changeset.ID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Len(actualEvents, len(expectedEvents))
	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_changeset_handler_with_stream_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrlStreamErrors.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	changeset := &manage.Changeset{}
	err = json.Unmarshal(respData, changeset)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(changeset.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
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

	expectedEvents := []testutils.ChangeStagingEvent{
		{
			Error: errors.New("error: change staging error"),
		},
	}
	actualEvents, err := s.streamChangeStagingEvents(changeset.ID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Len(actualEvents, len(expectedEvents))
	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_changeset_handler_fails_for_invalid_input() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrl.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			// "files" is not a valid scheme.
			FileSourceScheme: "files",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	validationError := &inputvalidation.FormattedValidationError{}
	err = json.Unmarshal(respData, validationError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().Equal(
		"request body input validation failed",
		validationError.Message,
	)
	s.Assert().Len(validationError.Errors, 1)
	s.Assert().Equal(
		".fileSourceScheme",
		validationError.Errors[0].Location,
	)
	s.Assert().Equal(
		"the value must be one of the following: file s3 gcs azureblob https",
		validationError.Errors[0].Message,
	)
	s.Assert().Equal(
		"oneof",
		validationError.Errors[0].Type,
	)
}

func (s *ControllerTestSuite) Test_create_changeset_handler_fails_due_to_id_gen_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrlFailingIDGenerators.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	responseError := map[string]string{}
	err = json.Unmarshal(respData, &responseError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
	s.Assert().Equal(
		utils.UnexpectedErrorMessage,
		responseError["message"],
	)
}

func (s *ControllerTestSuite) streamChangeStagingEvents(
	changesetID string,
	expectedCount int,
) ([]testutils.ChangeStagingEvent, error) {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes/{id}/stream",
		s.ctrl.StreamChangesetEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/changes/%s/stream",
		streamServer.URL,
		changesetID,
	)

	client := sse.NewClient(url)

	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	collected := []*manage.Event{}
	for len(collected) < expectedCount {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			s.Require().NotNil(event)
		case <-time.After(5 * time.Second):
			s.Fail("Timed out waiting for events")
		}
	}

	return extractChangeStagingEvents(collected)
}

func extractChangeStagingEvents(
	collected []*manage.Event,
) ([]testutils.ChangeStagingEvent, error) {
	extractedEvents := []testutils.ChangeStagingEvent{}

	for _, event := range collected {
		if event.Type == eventTypeResourceChanges {
			resourceChangesMessage := &container.ResourceChangesMessage{}
			err := parseEventJSON(event, resourceChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				ResourceChangesEvent: resourceChangesMessage,
			})
		}

		if event.Type == eventTypeChildChanges {
			childChangesMessage := &container.ChildChangesMessage{}
			err := parseEventJSON(event, childChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				ChildChangesEvent: childChangesMessage,
			})
		}

		if event.Type == eventTypeLinkChanges {
			LinkChangesMessage := &container.LinkChangesMessage{}
			err := parseEventJSON(event, LinkChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				LinkChangesEvent: LinkChangesMessage,
			})
		}

		if event.Type == eventTypeChangeStagingComplete {
			finalChangesEvent := &changeStagingCompleteEvent{}
			err := parseEventJSON(event, finalChangesEvent)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				FinalBlueprintChanges: finalChangesEvent.Changes,
			})
		}

		if event.Type == eventTypeError {
			errorMessage := &errorMessageEvent{}
			err := parseEventJSON(event, errorMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				Error: fmt.Errorf("error: %s", errorMessage.Message),
			})
		}
	}

	return extractedEvents, nil
}
