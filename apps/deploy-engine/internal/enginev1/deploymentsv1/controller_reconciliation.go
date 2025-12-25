package deploymentsv1

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	internalutils "github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/utils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	// reconciliationTimeout is the timeout for reconciliation operations.
	reconciliationTimeout = 10 * time.Minute
)

// CheckReconciliationHandler is the handler for the
// POST /deployments/instances/{id}/reconciliation/check endpoint
// that checks for drift and interrupted state in a blueprint instance.
func (c *Controller) CheckReconciliationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceID := params["id"]

	payload := &CheckReconciliationRequestPayload{}
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.logger)
	if responseWritten {
		return
	}

	if err := helpersv1.ValidateRequestBody.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		inputvalidation.HTTPValidationError(w, validationErrors)
		return
	}

	helpersv1.PopulateBlueprintDocInfoDefaults(&payload.BlueprintDocumentInfo)

	finalConfig, _, responseWritten := helpersv1.PrepareAndValidatePluginConfig(
		r,
		w,
		payload.Config,
		/* validate */ true,
		c.pluginConfigPreparer,
		c.logger,
	)
	if responseWritten {
		return
	}

	blueprintInfo, responseWritten := resolve.ResolveBlueprintForRequest(
		r,
		w,
		&payload.BlueprintDocumentInfo,
		c.blueprintResolver,
		c.logger,
	)
	if responseWritten {
		return
	}

	// Resolve the instance ID (may be name or ID)
	resolvedInstance, err := c.resolveInstance(r.Context(), instanceID)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("instance %q not found", instanceID),
			)
			return
		}
		c.logger.Debug(
			"failed to resolve instance",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	// Add blueprint directory to context variables for resolving relative child blueprint paths.
	finalConfig = internalutils.EnsureBlueprintDirContextVar(finalConfig, payload.BlueprintDocumentInfo.Directory)
	blueprintParams := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	result, err := c.performReconciliationCheck(
		r.Context(),
		resolvedInstance.InstanceID,
		payload,
		blueprintInfo,
		helpersv1.GetFormat(payload.BlueprintFile),
		blueprintParams,
	)
	if err != nil {
		c.logger.Debug(
			"failed to perform reconciliation check",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusOK,
		result,
	)
}

// ApplyReconciliationHandler is the handler for the
// POST /deployments/instances/{id}/reconciliation/apply endpoint
// that applies reconciliation actions to resolve drift or interrupted state.
func (c *Controller) ApplyReconciliationHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	instanceID := params["id"]

	payload := &ApplyReconciliationRequestPayload{}
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.logger)
	if responseWritten {
		return
	}

	if err := helpersv1.ValidateRequestBody.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		inputvalidation.HTTPValidationError(w, validationErrors)
		return
	}

	helpersv1.PopulateBlueprintDocInfoDefaults(&payload.BlueprintDocumentInfo)

	finalConfig, _, responseWritten := helpersv1.PrepareAndValidatePluginConfig(
		r,
		w,
		payload.Config,
		/* validate */ true,
		c.pluginConfigPreparer,
		c.logger,
	)
	if responseWritten {
		return
	}

	blueprintInfo, responseWritten := resolve.ResolveBlueprintForRequest(
		r,
		w,
		&payload.BlueprintDocumentInfo,
		c.blueprintResolver,
		c.logger,
	)
	if responseWritten {
		return
	}

	// Resolve the instance ID (may be name or ID)
	resolvedInstance, err := c.resolveInstance(r.Context(), instanceID)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			httputils.HTTPError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("instance %q not found", instanceID),
			)
			return
		}
		c.logger.Debug(
			"failed to resolve instance",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	// Add blueprint directory to context variables for resolving relative child blueprint paths.
	finalConfig = internalutils.EnsureBlueprintDirContextVar(finalConfig, payload.BlueprintDocumentInfo.Directory)
	blueprintParams := c.paramsProvider.CreateFromRequestConfig(finalConfig)

	result, err := c.applyReconciliation(
		r.Context(),
		resolvedInstance.InstanceID,
		payload,
		blueprintInfo,
		helpersv1.GetFormat(payload.BlueprintFile),
		blueprintParams,
	)
	if err != nil {
		c.logger.Debug(
			"failed to apply reconciliation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return
	}

	httputils.HTTPJSONResponse(
		w,
		http.StatusOK,
		result,
	)
}

func (c *Controller) performReconciliationCheck(
	ctx context.Context,
	instanceID string,
	payload *CheckReconciliationRequestPayload,
	blueprintInfo *includes.ChildBlueprintInfo,
	format schema.SpecFormat,
	params core.BlueprintParams,
) (*container.ReconciliationCheckResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, reconciliationTimeout)
	defer cancel()

	// Load the blueprint container
	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		helpersv1.GetBlueprintSource(blueprintInfo),
		format,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load blueprint container: %w", err)
	}

	scope := parseReconciliationScope(payload.Scope)
	input := &container.CheckReconciliationInput{
		InstanceID:    instanceID,
		Scope:         scope,
		ResourceNames: payload.ResourceNames,
		LinkNames:     payload.LinkNames,
	}

	return blueprintContainer.CheckReconciliation(ctxWithTimeout, input, params)
}

func (c *Controller) applyReconciliation(
	ctx context.Context,
	instanceID string,
	payload *ApplyReconciliationRequestPayload,
	blueprintInfo *includes.ChildBlueprintInfo,
	format schema.SpecFormat,
	params core.BlueprintParams,
) (*container.ApplyReconciliationResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, reconciliationTimeout)
	defer cancel()

	// Load the blueprint container
	blueprintContainer, err := c.blueprintLoader.LoadString(
		ctxWithTimeout,
		helpersv1.GetBlueprintSource(blueprintInfo),
		format,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load blueprint container: %w", err)
	}

	input := &container.ApplyReconciliationInput{
		InstanceID:      instanceID,
		ResourceActions: convertResourceActions(payload.ResourceActions),
		LinkActions:     convertLinkActions(payload.LinkActions),
	}

	return blueprintContainer.ApplyReconciliation(ctxWithTimeout, input, params)
}

func (c *Controller) resolveInstance(
	ctx context.Context,
	instanceIDOrName string,
) (*state.InstanceState, error) {
	// Try to get by ID first
	instance, err := c.instances.Get(ctx, instanceIDOrName)
	if err == nil {
		return &instance, nil
	}

	// If not found by ID, try by name
	if state.IsInstanceNotFound(err) {
		instanceID, lookupErr := c.instances.LookupIDByName(ctx, instanceIDOrName)
		if lookupErr != nil {
			return nil, lookupErr
		}
		instance, err = c.instances.Get(ctx, instanceID)
		if err != nil {
			return nil, err
		}
		return &instance, nil
	}

	return nil, err
}

func parseReconciliationScope(scope string) container.ReconciliationScope {
	switch scope {
	case "interrupted":
		return container.ReconciliationScopeInterrupted
	case "specific":
		return container.ReconciliationScopeSpecific
	case "all", "":
		return container.ReconciliationScopeAll
	default:
		// Default to "all" for unknown scope values
		return container.ReconciliationScopeAll
	}
}

func parseReconciliationAction(action string) container.ReconciliationAction {
	switch action {
	case "accept_external":
		return container.ReconciliationActionAcceptExternal
	case "update_status":
		return container.ReconciliationActionUpdateStatus
	case "mark_failed":
		return container.ReconciliationActionMarkFailed
	default:
		return container.ReconciliationActionUpdateStatus
	}
}

// parsePreciseResourceStatus parses a string to a PreciseResourceStatus.
// Uses a reverse lookup of the status string values.
func parsePreciseResourceStatus(status string) core.PreciseResourceStatus {
	// Map of string representations to PreciseResourceStatus values
	statusMap := map[string]core.PreciseResourceStatus{
		"UNKNOWN":                          core.PreciseResourceStatusUnknown,
		"CREATING":                         core.PreciseResourceStatusCreating,
		"CONFIG COMPLETE":                  core.PreciseResourceStatusConfigComplete,
		"CREATED":                          core.PreciseResourceStatusCreated,
		"CREATE FAILED":                    core.PreciseResourceStatusCreateFailed,
		"CREATE ROLLING BACK":              core.PreciseResourceStatusCreateRollingBack,
		"CREATE ROLLBACK FAILED":           core.PreciseResourceStatusCreateRollbackFailed,
		"CREATE ROLLBACK COMPLETE":         core.PreciseResourceStatusCreateRollbackComplete,
		"DESTROYING":                       core.PreciseResourceStatusDestroying,
		"DESTROYED":                        core.PreciseResourceStatusDestroyed,
		"DESTROY FAILED":                   core.PreciseResourceStatusDestroyFailed,
		"DESTROY ROLLING BACK":             core.PreciseResourceStatusDestroyRollingBack,
		"DESTROY ROLLBACK FAILED":          core.PreciseResourceStatusDestroyRollbackFailed,
		"DESTROY ROLLBACK CONFIG COMPLETE": core.PreciseResourceStatusDestroyRollbackConfigComplete,
		"DESTROY ROLLBACK COMPLETE":        core.PreciseResourceStatusDestroyRollbackComplete,
		"UPDATING":                         core.PreciseResourceStatusUpdating,
		"UPDATE CONFIG COMPLETE":           core.PreciseResourceStatusUpdateConfigComplete,
		"UPDATED":                          core.PreciseResourceStatusUpdated,
		"UPDATE FAILED":                    core.PreciseResourceStatusUpdateFailed,
		"UPDATE ROLLING BACK":              core.PreciseResourceStatusUpdateRollingBack,
		"UPDATE ROLLBACK FAILED":           core.PreciseResourceStatusUpdateRollbackFailed,
		"UPDATE ROLLBACK CONFIG COMPLETE":  core.PreciseResourceStatusUpdateRollbackConfigComplete,
		"UPDATE ROLLBACK COMPLETE":         core.PreciseResourceStatusUpdateRollbackComplete,
		"CREATE INTERRUPTED":               core.PreciseResourceStatusCreateInterrupted,
		"UPDATE INTERRUPTED":               core.PreciseResourceStatusUpdateInterrupted,
		"DESTROY INTERRUPTED":              core.PreciseResourceStatusDestroyInterrupted,
	}
	if s, ok := statusMap[status]; ok {
		return s
	}
	return core.PreciseResourceStatusUnknown
}

// parsePreciseLinkStatus parses a string to a PreciseLinkStatus.
// Uses a reverse lookup of the status string values.
func parsePreciseLinkStatus(status string) core.PreciseLinkStatus {
	// Map of string representations to PreciseLinkStatus values
	statusMap := map[string]core.PreciseLinkStatus{
		"UNKNOWN":                                         core.PreciseLinkStatusUnknown,
		"UPDATING RESOURCE A":                             core.PreciseLinkStatusUpdatingResourceA,
		"RESOURCE A UPDATED":                              core.PreciseLinkStatusResourceAUpdated,
		"RESOURCE A UPDATE FAILED":                        core.PreciseLinkStatusResourceAUpdateFailed,
		"RESOURCE A UPDATE ROLLING BACK":                  core.PreciseLinkStatusResourceAUpdateRollingBack,
		"RESOURCE A UPDATE ROLLBACK FAILED":               core.PreciseLinkStatusResourceAUpdateRollbackFailed,
		"RESOURCE A UPDATE ROLLBACK COMPLETE":             core.PreciseLinkStatusResourceAUpdateRollbackComplete,
		"UPDATING RESOURCE B":                             core.PreciseLinkStatusUpdatingResourceB,
		"RESOURCE B UPDATED":                              core.PreciseLinkStatusResourceBUpdated,
		"RESOURCE B UPDATE FAILED":                        core.PreciseLinkStatusResourceBUpdateFailed,
		"RESOURCE B UPDATE ROLLING BACK":                  core.PreciseLinkStatusResourceBUpdateRollingBack,
		"RESOURCE B UPDATE ROLLBACK FAILED":               core.PreciseLinkStatusResourceBUpdateRollbackFailed,
		"RESOURCE B UPDATE ROLLBACK COMPLETE":             core.PreciseLinkStatusResourceBUpdateRollbackComplete,
		"UPDATING INTERMEDIARY RESOURCES":                 core.PreciseLinkStatusUpdatingIntermediaryResources,
		"INTERMEDIARY RESOURCES UPDATED":                  core.PreciseLinkStatusIntermediaryResourcesUpdated,
		"INTERMEDIARY RESOURCES UPDATE FAILED":            core.PreciseLinkStatusIntermediaryResourceUpdateFailed,
		"INTERMEDIARY RESOURCES UPDATE ROLLING BACK":      core.PreciseLinkStatusIntermediaryResourceUpdateRollingBack,
		"INTERMEDIARY RESOURCES UPDATE ROLLBACK FAILED":   core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed,
		"INTERMEDIARY RESOURCES UPDATE ROLLBACK COMPLETE": core.PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete,
		"RESOURCE A UPDATE INTERRUPTED":                   core.PreciseLinkStatusResourceAUpdateInterrupted,
		"RESOURCE B UPDATE INTERRUPTED":                   core.PreciseLinkStatusResourceBUpdateInterrupted,
		"INTERMEDIARY RESOURCES UPDATE INTERRUPTED":       core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted,
	}
	if s, ok := statusMap[status]; ok {
		return s
	}
	return core.PreciseLinkStatusUnknown
}

func convertResourceActions(payloadActions []ResourceReconcileActionPayload) []container.ResourceReconcileAction {
	actions := make([]container.ResourceReconcileAction, len(payloadActions))
	for i, pa := range payloadActions {
		actions[i] = container.ResourceReconcileAction{
			ResourceID:    pa.ResourceID,
			Action:        parseReconciliationAction(pa.Action),
			ExternalState: pa.ExternalState,
			NewStatus:     parsePreciseResourceStatus(pa.NewStatus),
		}
	}
	return actions
}

func convertLinkActions(payloadActions []LinkReconcileActionPayload) []container.LinkReconcileAction {
	actions := make([]container.LinkReconcileAction, len(payloadActions))
	for i, pa := range payloadActions {
		actions[i] = container.LinkReconcileAction{
			LinkID:              pa.LinkID,
			Action:              parseReconciliationAction(pa.Action),
			NewStatus:           parsePreciseLinkStatus(pa.NewStatus),
			LinkDataUpdates:     pa.LinkDataUpdates,
			IntermediaryActions: convertIntermediaryActions(pa.IntermediaryActions),
		}
	}
	return actions
}

func convertIntermediaryActions(
	payloadActions map[string]*IntermediaryReconcileActionPayload,
) map[string]*container.IntermediaryReconcileAction {
	if payloadActions == nil {
		return nil
	}
	actions := make(map[string]*container.IntermediaryReconcileAction, len(payloadActions))
	for id, pa := range payloadActions {
		if pa == nil {
			continue
		}
		actions[id] = &container.IntermediaryReconcileAction{
			IntermediaryID: id,
			Action:         parseReconciliationAction(pa.Action),
			ExternalState:  pa.ExternalState,
			NewStatus:      parsePreciseResourceStatus(pa.NewStatus),
		}
	}
	return actions
}
