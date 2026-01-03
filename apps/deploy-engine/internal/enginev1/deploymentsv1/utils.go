package deploymentsv1

import (
	"context"
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
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
