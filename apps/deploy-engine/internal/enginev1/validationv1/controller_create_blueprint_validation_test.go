package validationv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/r3labs/sse/v2"
)

func (s *ControllerTestSuite) Test_create_blueprint_validation_handler() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations",
		s.ctrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	reqPayload := &CreateValidationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/validations", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.BlueprintValidation]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	blueprintValidation := wrappedResponse.Data
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(blueprintValidation.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		manage.BlueprintValidationStatusStarting,
		blueprintValidation.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		blueprintValidation.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		blueprintValidation.Created,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_validation_handler_fails_for_invalid_input() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations",
		s.ctrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	reqPayload := &CreateValidationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			// "files" is not a valid scheme.
			FileSourceScheme: "files",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/validations", bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) Test_create_blueprint_validation_handler_fails_for_plugin_config_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations",
		s.ctrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	reqPayload := &CreateValidationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"aws": {
					"field1": core.ScalarFromString("invalid value"),
				},
			},
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/validations?checkPluginConfig=true", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	validationError := &typesv1.ValidationDiagnosticErrors{}
	err = json.Unmarshal(respData, validationError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().Equal(
		"plugin configuration validation failed",
		validationError.Message,
	)
	s.Assert().Equal(
		pluginConfigPreparerFixtures["invalid value"],
		validationError.ValidationDiagnostics,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_validation_handler_streams_plugin_config_validation_warnings() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations",
		s.ctrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	reqPayload := &CreateValidationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"aws": {
					"field1": core.ScalarFromString("warnings value"),
				},
			},
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/validations?checkPluginConfig=true", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.BlueprintValidation]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	blueprintValidation := wrappedResponse.Data
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(blueprintValidation.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		manage.BlueprintValidationStatusStarting,
		blueprintValidation.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		blueprintValidation.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		blueprintValidation.Created,
	)

	expectedEvents := expectedValidationEventsWithWarnings()
	actualEvents, err := s.streamValidationEvents(blueprintValidation.ID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_validation_handler_fails_due_to_id_gen_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations",
		s.ctrlFailingIDGenerators.CreateBlueprintValidationHandler,
	).Methods("POST")

	reqPayload := &CreateValidationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/validations", bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) streamValidationEvents(
	instanceID string,
	expectedCount int,
) ([]*core.Diagnostic, error) {
	router := mux.NewRouter()
	router.HandleFunc(
		"/validations/{id}/stream",
		s.ctrl.StreamEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/validations/%s/stream",
		streamServer.URL,
		instanceID,
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

	return extractValidationEvents(collected)
}

func extractValidationEvents(
	collected []*manage.Event,
) ([]*core.Diagnostic, error) {
	extractedEvents := []*core.Diagnostic{}

	for _, event := range collected {
		if event.Type == eventTypeDiagnostic {
			diagnostic := &core.Diagnostic{}
			err := parseEventJSON(event, diagnostic)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, diagnostic)
		}
	}

	return extractedEvents, nil
}

func parseEventJSON(
	event *manage.Event,
	target any,
) error {
	return json.Unmarshal(
		[]byte(event.Data),
		target,
	)
}

func expectedValidationEventsWithWarnings() []*core.Diagnostic {
	// Combine the plugin config preparer fixture warnings with the blueprint loader
	// stub diagnostics.
	diagnostics := []*core.Diagnostic{}
	diagnostics = append(diagnostics, pluginConfigPreparerFixtures["warnings value"]...)
	diagnostics = append(diagnostics, stubDiagnostics...)

	return diagnostics
}
