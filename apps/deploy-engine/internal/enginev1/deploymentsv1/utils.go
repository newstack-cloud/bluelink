package deploymentsv1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func createChangeStagingChannels() *container.ChangeStagingChannels {
	return &container.ChangeStagingChannels{
		ResourceChangesChan: make(chan container.ResourceChangesMessage),
		ChildChangesChan:    make(chan container.ChildChangesMessage),
		LinkChangesChan:     make(chan container.LinkChangesMessage),
		CompleteChan:        make(chan changes.BlueprintChanges),
		ErrChan:             make(chan error),
	}
}

func handleDeployErrorForResponse(
	w http.ResponseWriter,
	err error,
	logger core.Logger,
) {
	// If the error is a load error with validation errors,
	// make sure the validation errors are exposed to the client
	// to make it clear that the issue was with loading the source blueprint
	// provided in the request.
	diagnostics := utils.DiagnosticsFromBlueprintValidationError(
		err,
		logger,
		/* fallbackToGeneralDiagnostic */ false,
	)

	if len(diagnostics) == 0 {
		logger.Error(
			"failed to start blueprint instance deployment",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	validationErrors := &typesv1.ValidationDiagnosticErrors{
		Message:               "failed to load the blueprint document specified in the request",
		ValidationDiagnostics: diagnostics,
	}
	httputils.HTTPJSONResponse(
		w,
		http.StatusUnprocessableEntity,
		validationErrors,
	)
}

func getInstanceID(
	instance *state.InstanceState,
) string {
	if instance == nil {
		return ""
	}

	return instance.InstanceID
}

// Reports whether it wrote a response. An update may omit the name, in which case the
// instance keeps its own.
func rejectInstanceRename(
	w http.ResponseWriter,
	existingInstance *state.InstanceState,
	requestedName string,
) (responseWritten bool) {
	if existingInstance == nil ||
		requestedName == "" ||
		requestedName == existingInstance.InstanceName {
		return false
	}

	httputils.HTTPErrorWithFields(
		w,
		http.StatusBadRequest,
		"the name of an existing blueprint instance cannot be changed",
		map[string]any{
			"code":         "INSTANCE_NAME_IMMUTABLE",
			"instanceName": existingInstance.InstanceName,
		},
	)
	return true
}

// Reports whether it wrote a response. Given a name and no ID, the deploy resolves the name
// and deploys onto the instance it finds, so a create naming an existing instance would
// update that instance instead of failing.
func (c *Controller) rejectNewInstanceName(
	w http.ResponseWriter,
	r *http.Request,
	existingInstance *state.InstanceState,
	requestedName string,
) (responseWritten bool) {
	if existingInstance != nil {
		return false
	}

	if requestedName == "" {
		inputvalidation.HTTPMissingValueError(w, ".instanceName")
		return true
	}

	_, err := c.instances.LookupIDByName(r.Context(), requestedName)
	if state.IsInstanceNotFound(err) {
		return false
	}

	c.respondWithInstanceNameInUse(w, r, requestedName)
	return true
}

// The holder's ID is left out when it can no longer be found, as it can be after the deploy
// found the name taken and the holder was removed before it could be looked up.
func (c *Controller) respondWithInstanceNameInUse(
	w http.ResponseWriter,
	r *http.Request,
	name string,
) {
	fields := map[string]any{"code": "INSTANCE_ALREADY_EXISTS"}
	instanceID, err := c.instances.LookupIDByName(r.Context(), name)
	if err != nil && !state.IsInstanceNotFound(err) {
		c.requestLogger(r).Error(
			"failed to look up the blueprint instance that holds the requested name",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}
	if err == nil {
		fields["instanceId"] = instanceID
	}

	httputils.HTTPErrorWithFields(
		w,
		http.StatusConflict,
		fmt.Sprintf("a blueprint instance named %q already exists", name),
		fields,
	)
}

// A change set's changes were computed against the instance it was staged for, or against
// nothing for a new instance, so applying it to any other target would start from the wrong
// state. An empty targetInstanceID means a new instance.
func rejectChangesetForOtherInstance(
	w http.ResponseWriter,
	changeset *manage.Changeset,
	targetInstanceID string,
) (responseWritten bool) {
	if changeset.InstanceID == targetInstanceID {
		return false
	}

	fields := map[string]any{"code": "CHANGESET_INSTANCE_MISMATCH"}
	if changeset.InstanceID != "" {
		fields["changesetInstanceId"] = changeset.InstanceID
	}
	if targetInstanceID != "" {
		fields["instanceId"] = targetInstanceID
	}

	httputils.HTTPErrorWithFields(
		w,
		http.StatusConflict,
		changesetInstanceMismatchMessage(changeset, targetInstanceID),
		fields,
	)
	return true
}

func changesetInstanceMismatchMessage(changeset *manage.Changeset, targetInstanceID string) string {
	if changeset.InstanceID == "" {
		return fmt.Sprintf(
			"change set %q was staged for a new blueprint instance, not instance %q; "+
				"stage changes against the instance and try again",
			changeset.ID,
			targetInstanceID,
		)
	}

	if targetInstanceID == "" {
		return fmt.Sprintf(
			"change set %q was staged for instance %q, not a new blueprint instance; "+
				"stage changes for a new instance and try again",
			changeset.ID,
			changeset.InstanceID,
		)
	}

	return fmt.Sprintf(
		"change set %q was staged for instance %q, not instance %q; "+
			"stage changes against the instance and try again",
		changeset.ID,
		changeset.InstanceID,
		targetInstanceID,
	)
}

// resolveInstance resolves an instance from the provided identifier,
// which can be either an instance ID or an instance name.
// It first tries to get the instance by ID, and if not found,
// falls back to looking up the ID by name and then fetching the instance.
func resolveInstance(
	ctx context.Context,
	identifier string,
	instances state.InstancesContainer,
) (state.InstanceState, error) {
	// First, try to get the instance directly by ID.
	instance, err := instances.Get(ctx, identifier)
	if err == nil {
		return instance, nil
	}

	// If not found by ID, try to look up by name.
	if state.IsInstanceNotFound(err) {
		instanceID, lookupErr := instances.LookupIDByName(ctx, identifier)
		if lookupErr == nil {
			return instances.Get(ctx, instanceID)
		}
		// If lookup by name also fails, return the original "not found" error
		// with the original identifier for a clearer error message.
		if state.IsInstanceNotFound(lookupErr) {
			return state.InstanceState{}, err
		}
		return state.InstanceState{}, lookupErr
	}

	return state.InstanceState{}, err
}

// resolveInstanceID resolves an instance ID from the provided identifier,
// which can be either an instance ID or an instance name.
// Use this when you only need the ID and not the full instance state.
func resolveInstanceID(
	ctx context.Context,
	identifier string,
	instances state.InstancesContainer,
) (string, error) {
	// First, try to get the instance directly by ID.
	_, err := instances.Get(ctx, identifier)
	if err == nil {
		return identifier, nil
	}

	// If not found by ID, try to look up by name.
	if state.IsInstanceNotFound(err) {
		instanceID, lookupErr := instances.LookupIDByName(ctx, identifier)
		if lookupErr == nil {
			return instanceID, nil
		}
		// If lookup by name also fails, return the original "not found" error
		// with the original identifier for a clearer error message.
		if state.IsInstanceNotFound(lookupErr) {
			return "", err
		}
		return "", lookupErr
	}

	return "", err
}

// A placeholder template used to be able to make use of the blueprint loader
// to load a blueprint container for destroying a blueprint instance.
// Requests to destroy a blueprint instance are not expected to provide
// a source blueprint document as the blueprint container doesn't utilise
// the source blueprint document in the destroy process.
const placeholderBlueprint = `
version: 2025-11-02
resources:
  stubResource:
    type: core/stub
    description: "A stub resource that does nothing"
    metadata:
      displayName: A stub resource
      labels:
        app: stubService
    linkSelector:
      byLabel:
        app: stubService
    spec:
      value: "stubValue"
`
