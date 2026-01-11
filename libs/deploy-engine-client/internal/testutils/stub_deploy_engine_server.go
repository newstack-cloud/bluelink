// Provides a stub implementation of a Deploy Engine server.
// This allows for thorough testing of the client including
// client streaming behaviour and the various authentication
// methods supported by the Deploy Engine v1 API.
package testutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/sigv1"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
)

const (
	testFailingStreamEventID = "test-failing-stream-event-id"
)

type TestServerConfig struct {
	AllowedAPIKeys                   []string
	AllowedBearerTokens              []string
	AllowedBluelinkSignatureKeyPairs map[string]*sigv1.KeyPair
	// A string that will be present in some part of the request
	// to the server that will trigger behaviour to simulate
	// an internal server error.
	InternalServerErrorTrigger   string
	NetworkErrorTrigger          string
	DeserialiseErrorTrigger      string
	InternalServerErrorTriggerID string
	NetworkErrorTriggerID        string
	DeserialiseErrorTriggerID    string
	FailingStreamTriggerID       string
	DriftDetectedChangesetID     string
	UseUnixDomainSocket          bool
	UnixDomainSocketPath         string
}

func CreateDeployEngineServer(
	serverConfig *TestServerConfig,
	stubValidationEvents []*manage.Event,
	stubChangeStagingEvents []*manage.Event,
	stubDriftDetectedEvents []*manage.Event,
	stubDeploymentEvents []*manage.Event,
	clock core.Clock,
) *httptest.Server {
	router := mux.NewRouter()
	router.Use(authMiddleware(serverConfig, clock))
	ctrl := &stubDeployEngineController{
		serverConfig:            serverConfig,
		clock:                   clock,
		stubValidationEvents:    stubValidationEvents,
		stubChangeStagingEvents: stubChangeStagingEvents,
		stubDriftDetectedEvents: stubDriftDetectedEvents,
		stubDeploymentEvents:    stubDeploymentEvents,
	}

	router.HandleFunc(
		"/v1/validations",
		ctrl.createBlueprintValidationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/validations/{id}",
		ctrl.getBlueprintValidationHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/validations/{id}/stream",
		ctrl.streamBlueprintValidationEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/validations/cleanup",
		ctrl.cleanupBlueprintValidationsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/validations/cleanup/{id}",
		ctrl.getCleanupStatusHandler(manage.CleanupTypeValidations),
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/changes",
		ctrl.createChangesetHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/changes/{id}",
		ctrl.getChangesetHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/changes/{id}/stream",
		ctrl.streamChangeStagingEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/changes/cleanup",
		ctrl.cleanupChangesetsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/changes/cleanup/{id}",
		ctrl.getCleanupStatusHandler(manage.CleanupTypeChangesets),
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/instances",
		ctrl.createBlueprintInstanceHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/instances/{id}",
		ctrl.updateBlueprintInstanceHandler,
	).Methods("PATCH")

	router.HandleFunc(
		"/v1/deployments/instances/{id}",
		ctrl.getBlueprintInstanceHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/instances/{id}/exports",
		ctrl.getBlueprintInstanceExportsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/instances/{id}/destroy",
		ctrl.destroyBlueprintInstanceHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/instances/{id}/stream",
		ctrl.streamDeploymentEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/v1/events/cleanup",
		ctrl.cleanupEventsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/events/cleanup/{id}",
		ctrl.getCleanupStatusHandler(manage.CleanupTypeEvents),
	).Methods("GET")

	router.HandleFunc(
		"/v1/deployments/instances/{id}/reconciliation/check",
		ctrl.checkReconciliationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/instances/{id}/reconciliation/apply",
		ctrl.applyReconciliationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/reconciliation-results/cleanup",
		ctrl.cleanupReconciliationResultsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/v1/deployments/reconciliation-results/cleanup/{id}",
		ctrl.getCleanupStatusHandler(manage.CleanupTypeReconciliationResults),
	).Methods("GET")

	if serverConfig.UseUnixDomainSocket {
		return NewUnixDomainSocketServer(
			serverConfig.UnixDomainSocketPath,
			router,
		)
	}

	server := httptest.NewServer(router)
	ctrl.server = server

	return server
}

type stubDeployEngineController struct {
	serverConfig                     *TestServerConfig
	clock                            core.Clock
	server                           *httptest.Server
	stubValidationEvents             []*manage.Event
	stubChangeStagingEvents          []*manage.Event
	stubDriftDetectedEvents          []*manage.Event
	stubDeploymentEvents             []*manage.Event
}

type postRequestPayload struct {
	// Will be checked to produce 422 error responses.
	FileSourceScheme string `json:"fileSourceScheme"`
	// Will be checked to produce 500 error responses.
	BlueprintFile string `json:"blueprintFile"`
	// InstanceName is the user-defined name for the blueprint instance.
	// This is used when creating a new blueprint instance.
	InstanceName string `json:"instanceName"`
}

func (c *stubDeployEngineController) createBlueprintValidationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For POST requests, the server error trigger will be in
	// the "blueprintFile" field of the request body.
	payload := &postRequestPayload{}
	exitEarly := decodeRequestBody(w, r, payload)
	if exitEarly {
		return
	}

	exitEarly = c.handlePostErrorTriggers(w, payload)
	if exitEarly {
		return
	}

	blueprintValidation := &manage.BlueprintValidation{
		ID:                "test-validation-id",
		Status:            manage.BlueprintValidationStatusStarting,
		BlueprintLocation: "test-blueprint-location",
		Created:           c.clock.Now().Unix(),
	}

	// Wrap response with lastEventId
	response := map[string]any{
		"lastEventId": "test-last-event-id",
		"data":        blueprintValidation,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) getBlueprintValidationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	blueprintValidation := &manage.BlueprintValidation{
		ID:                "test-validation-id",
		Status:            manage.BlueprintValidationStatusValidated,
		BlueprintLocation: "test-blueprint-location",
		Created:           c.clock.Now().Unix(),
	}

	respBytes, _ := json.Marshal(blueprintValidation)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) streamBlueprintValidationEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	failStream := id == c.serverConfig.FailingStreamTriggerID
	streamEvents(w, c.stubValidationEvents, failStream)
}

func (c *stubDeployEngineController) cleanupBlueprintValidationsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	c.writeCleanupOperationResponse(w, manage.CleanupTypeValidations)
}

func (c *stubDeployEngineController) createChangesetHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For POST requests, the server error trigger will be in
	// the "blueprintFile" field of the request body.
	payload := &postRequestPayload{}
	exitEarly := decodeRequestBody(w, r, payload)
	if exitEarly {
		return
	}

	exitEarly = c.handlePostErrorTriggers(w, payload)
	if exitEarly {
		return
	}

	changeset := &manage.Changeset{
		ID:                "test-changeset-id",
		InstanceID:        "test-instance-id",
		Status:            manage.ChangesetStatusStarting,
		Destroy:           false,
		Changes:           stubChanges,
		BlueprintLocation: "test-blueprint-location",
		Created:           c.clock.Now().Unix(),
	}

	// Wrap response with lastEventId
	response := map[string]any{
		"lastEventId": "test-last-event-id",
		"data":        changeset,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) getChangesetHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	changeset := &manage.Changeset{
		ID:                "test-changeset-id",
		InstanceID:        "test-instance-id",
		Destroy:           false,
		Status:            manage.ChangesetStatusChangesStaged,
		Changes:           stubChanges,
		BlueprintLocation: "test-blueprint-location",
		Created:           c.clock.Now().Unix(),
	}

	respBytes, _ := json.Marshal(changeset)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) streamChangeStagingEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	failStream := id == c.serverConfig.FailingStreamTriggerID

	// Serve drift detected events if the changeset ID matches.
	if id == c.serverConfig.DriftDetectedChangesetID {
		streamEvents(w, c.stubDriftDetectedEvents, failStream)
		return
	}

	streamEvents(w, c.stubChangeStagingEvents, failStream)
}

func (c *stubDeployEngineController) createBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For POST requests, the server error trigger will be in
	// the "blueprintFile" field of the request body.
	payload := &postRequestPayload{}
	exitEarly := decodeRequestBody(w, r, payload)
	if exitEarly {
		return
	}

	exitEarly = c.handlePostErrorTriggers(w, payload)
	if exitEarly {
		return
	}

	// Use the provided instance name from the payload if available,
	// otherwise fall back to a default test name.
	instanceName := payload.InstanceName
	if instanceName == "" {
		instanceName = "test-instance-name"
	}

	blueprintInstance := &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: instanceName,
		Status:       core.InstanceStatusDeploying,
	}

	// Wrap response with lastEventId
	response := map[string]any{
		"lastEventId": "test-last-event-id",
		"data":        blueprintInstance,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) updateBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For PATCH requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusAccepted)
	if exitEarly {
		return
	}

	blueprintInstance := &state.InstanceState{
		InstanceID:   id,
		InstanceName: "test-instance-name",
		Status:       core.InstanceStatusDeploying,
	}

	// Wrap response with lastEventId
	response := map[string]any{
		"lastEventId": "test-last-event-id",
		"data":        blueprintInstance,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) getBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	blueprintInstance := &state.InstanceState{
		InstanceID:   id,
		InstanceName: "test-instance-name",
		Status:       core.InstanceStatusDeploying,
	}

	respBytes, _ := json.Marshal(blueprintInstance)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) getBlueprintInstanceExportsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	exports := map[string]*state.ExportState{
		"exportedField1": {
			Value: core.MappingNodeFromString("exportedValue1"),
			Type:  schema.ExportTypeString,
			Field: "resources[\"resource-1\"].spec.name",
		},
	}

	respBytes, _ := json.Marshal(exports)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) destroyBlueprintInstanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For the requests to POST /deployments/instances/{id}/destroy,
	// the error trigger will be in the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusAccepted)
	if exitEarly {
		return
	}

	blueprintInstance := &state.InstanceState{
		InstanceID:   id,
		InstanceName: "test-instance-name",
		Status:       core.InstanceStatusDestroying,
	}

	// Wrap response with lastEventId
	response := map[string]any{
		"lastEventId": "test-last-event-id",
		"data":        blueprintInstance,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) streamDeploymentEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For GET requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	failStream := id == c.serverConfig.FailingStreamTriggerID
	streamEvents(w, c.stubDeploymentEvents, failStream)
}

func (c *stubDeployEngineController) handlePostErrorTriggers(
	w http.ResponseWriter,
	payload *postRequestPayload,
) bool {
	if payload.FileSourceScheme != "file" {
		writeUnprocessableEntityError(w)
		return true
	}

	if payload.BlueprintFile == c.serverConfig.InternalServerErrorTrigger {
		writeInternalServerError(w)
		return true
	}

	if payload.BlueprintFile == c.serverConfig.NetworkErrorTrigger {
		// Simulate a network error by closing all connections to the server.
		c.server.CloseClientConnections()
		return true
	}

	if payload.BlueprintFile == c.serverConfig.DeserialiseErrorTrigger {
		// Return a 202 response with invalid JSON.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"invalid": "json"`))
		return true
	}

	return false
}

func (c *stubDeployEngineController) cleanupChangesetsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	c.writeCleanupOperationResponse(w, manage.CleanupTypeChangesets)
}

func (c *stubDeployEngineController) cleanupEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	c.writeCleanupOperationResponse(w, manage.CleanupTypeEvents)
}

func (c *stubDeployEngineController) cleanupReconciliationResultsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	c.writeCleanupOperationResponse(w, manage.CleanupTypeReconciliationResults)
}

func (c *stubDeployEngineController) writeCleanupOperationResponse(
	w http.ResponseWriter,
	cleanupType manage.CleanupType,
) {
	operation := &manage.CleanupOperation{
		ID:            "test-cleanup-operation-id",
		CleanupType:   cleanupType,
		Status:        manage.CleanupOperationStatusRunning,
		StartedAt:     c.clock.Now().Unix(),
		ThresholdDate: c.clock.Now().Add(-7 * 24 * time.Hour).Unix(),
	}

	response := map[string]any{
		"data": operation,
	}

	respBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) getCleanupStatusHandler(
	cleanupType manage.CleanupType,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
		if exitEarly {
			return
		}

		operation := &manage.CleanupOperation{
			ID:            id,
			CleanupType:   cleanupType,
			Status:        manage.CleanupOperationStatusCompleted,
			StartedAt:     c.clock.Now().Add(-1 * time.Minute).Unix(),
			EndedAt:       c.clock.Now().Unix(),
			ItemsDeleted:  42,
			ThresholdDate: c.clock.Now().Add(-7 * 24 * time.Hour).Unix(),
		}

		respBytes, _ := json.Marshal(operation)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}
}

func (c *stubDeployEngineController) checkReconciliationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For POST requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	result := stubReconciliationCheckResult(id)
	respBytes, _ := json.Marshal(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) applyReconciliationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// For POST requests, the error trigger will be in
	// the id path parameter.
	vars := mux.Vars(r)
	id := vars["id"]
	exitEarly := c.handleIDErrorTriggers(w, id, http.StatusOK)
	if exitEarly {
		return
	}

	result := stubApplyReconciliationResult(id)
	respBytes, _ := json.Marshal(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (c *stubDeployEngineController) handleIDErrorTriggers(
	w http.ResponseWriter,
	id string,
	statusCode int,
) bool {
	if id == c.serverConfig.InternalServerErrorTriggerID {
		writeInternalServerError(w)
		return true
	}

	if id == c.serverConfig.NetworkErrorTriggerID {
		// Simulate a network error by closing all connections to the server.
		c.server.CloseClientConnections()
		return true
	}

	if id == c.serverConfig.DeserialiseErrorTriggerID {
		// Return a 200 response with invalid JSON.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"invalid": "json"`))
		return true
	}

	return false
}

func authMiddleware(
	serverConfig *TestServerConfig,
	clock core.Clock,
) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for API key
			apiKey := r.Header.Get("Bluelink-Api-Key")
			if apiKey != "" && slices.Contains(serverConfig.AllowedAPIKeys, apiKey) {
				next.ServeHTTP(w, r)
				return
			}

			// Check for Bearer token
			bearerTokenHeader := r.Header.Get("Authorization")
			bearerToken := strings.TrimPrefix(bearerTokenHeader, "Bearer ")
			if bearerToken != "" && slices.Contains(serverConfig.AllowedBearerTokens, bearerToken) {
				next.ServeHTTP(w, r)
				return
			}

			// Check for Bluelink signature key pair
			bluelinkSignature := r.Header.Get(sigv1.SignatureHeaderName)
			if bluelinkSignature != "" {
				err := sigv1.VerifySignature(
					serverConfig.AllowedBluelinkSignatureKeyPairs,
					r.Header,
					clock,
					&sigv1.VerifyOptions{},
				)
				if err == nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Unauthorized"}`))
		})
	}
}

func writeUnprocessableEntityError(w http.ResponseWriter) {
	response := errors.Response{
		Message: "fileSourceScheme must be \"file\"",
		Errors: []*errors.ValidationError{
			{
				Location: "fileSourceScheme",
				Message:  "fileSourceScheme must be \"file\"",
				Type:     "invalid",
			},
		},
	}
	bodyBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	w.Write(bodyBytes)
}

func writeInternalServerError(w http.ResponseWriter) {
	response := errors.Response{
		Message: "an unexpected error occurred",
	}
	bodyBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(bodyBytes)
}

// Decodes a JSON request body into the provided payload
// and returns true if an error occurred and a response has been sent to the client.
func decodeRequestBody(w http.ResponseWriter, r *http.Request, payload any) bool {
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"failed to parse the request body"}`))
		return true
	}

	return false
}

func streamEvents(
	w http.ResponseWriter,
	events []*manage.Event,
	failStream bool,
) {
	// Check if the ResponseWriter supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if failStream {
		writeErrorEvent(
			w,
			testFailingStreamEventID,
			"An unexpected error occurred in stream process",
			flusher,
		)
		return
	}

	for _, evt := range events {
		writeEvent(w, evt, flusher)
	}
}

func writeErrorEvent(
	w http.ResponseWriter,
	id string,
	message string,
	flusher http.Flusher,
) {
	errorEvt := &manage.Event{
		ID:   id,
		Type: "error",
		Data: fmt.Sprintf(
			`{"message":"%s"}`,
			message,
		),
	}
	writeEvent(w, errorEvt, flusher)
}

func writeEvent(
	w http.ResponseWriter,
	evt *manage.Event,
	flusher http.Flusher,
) {
	fmt.Fprintf(w, "event: %s\n", evt.Type)
	fmt.Fprintf(w, "id: %s\n", evt.ID)
	fmt.Fprintf(w, "data: %s\n\n", evt.Data)

	// Flush the data immediatly instead of buffering it for later.
	flusher.Flush()
}

var stubChanges = &changes.BlueprintChanges{
	ResourceChanges: map[string]provider.Changes{
		"resource-1": {
			NewFields: []provider.FieldChange{
				{
					FieldPath: "spec.name",
					PrevValue: core.MappingNodeFromString("old-name"),
					NewValue:  core.MappingNodeFromString("new-name"),
				},
			},
		},
	},
	RemovedResources: []string{"resource-2", "resource-3"},
}

func stubReconciliationCheckResult(instanceID string) *container.ReconciliationCheckResult {
	return &container.ReconciliationCheckResult{
		InstanceID: instanceID,
		Resources: []container.ResourceReconcileResult{
			{
				ResourceID:        "resource-1-id",
				ResourceName:      "resource-1",
				ResourceType:      "test/resource",
				Type:              container.ReconciliationTypeDrift,
				OldStatus:         core.PreciseResourceStatusCreated,
				NewStatus:         core.PreciseResourceStatusCreated,
				ExternalState:     core.MappingNodeFromString("external-state-value"),
				PersistedState:    core.MappingNodeFromString("persisted-state-value"),
				ResourceExists:    true,
				RecommendedAction: container.ReconciliationActionAcceptExternal,
			},
		},
		Links: []container.LinkReconcileResult{
			{
				LinkID:            "link-1-id",
				LinkName:          "resource-1::resource-2",
				Type:              container.ReconciliationTypeInterrupted,
				OldStatus:         core.PreciseLinkStatusResourceBUpdateRollingBack,
				NewStatus:         core.PreciseLinkStatusResourceBUpdated,
				RecommendedAction: container.ReconciliationActionUpdateStatus,
			},
		},
		HasInterrupted: true,
		HasDrift:       true,
	}
}

func stubApplyReconciliationResult(instanceID string) *container.ApplyReconciliationResult {
	return &container.ApplyReconciliationResult{
		InstanceID:       instanceID,
		ResourcesUpdated: 1,
		LinksUpdated:     1,
		Errors:           []container.ReconciliationError{},
	}
}
