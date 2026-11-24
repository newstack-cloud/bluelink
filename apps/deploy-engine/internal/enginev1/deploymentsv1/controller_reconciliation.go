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
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.requestLogger(r))
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
		c.requestLogger(r),
	)
	if responseWritten {
		return
	}

	blueprintInfo, responseWritten := resolve.ResolveBlueprintForRequest(
		r,
		w,
		&payload.BlueprintDocumentInfo,
		c.blueprintResolver,
		c.requestLogger(r),
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
		c.requestLogger(r).Debug(
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
		c.requestLogger(r).Debug(
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
	responseWritten := httputils.DecodeRequestBody(w, r, payload, c.requestLogger(r))
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
		c.requestLogger(r),
	)
	if responseWritten {
		return
	}

	blueprintInfo, responseWritten := resolve.ResolveBlueprintForRequest(
		r,
		w,
		&payload.BlueprintDocumentInfo,
		c.blueprintResolver,
		c.requestLogger(r),
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
		c.requestLogger(r).Debug(
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
		c.requestLogger(r).Debug(
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
	taggingConfig := c.createTaggingConfig(payload.Config)
	input := &container.CheckReconciliationInput{
		InstanceID:      instanceID,
		Scope:           scope,
		ResourceNames:   payload.ResourceNames,
		LinkNames:       payload.LinkNames,
		IncludeChildren: payload.IncludeChildren,
		ChildPath:       payload.ChildPath,
		TaggingConfig:   taggingConfig,
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

// The scope, actions and statuses are validated with the request body, so the casts and
// parse results below are not checked again.
func parseReconciliationScope(scope string) container.ReconciliationScope {
	if scope == "" {
		return container.ReconciliationScopeAll
	}
	return container.ReconciliationScope(scope)
}

func convertResourceActions(payloadActions []ResourceReconcileActionPayload) []container.ResourceReconcileAction {
	actions := make([]container.ResourceReconcileAction, len(payloadActions))
	for i, pa := range payloadActions {
		newStatus, _ := helpersv1.ParsePreciseResourceStatus(pa.NewStatus)
		actions[i] = container.ResourceReconcileAction{
			ResourceID:    pa.ResourceID,
			ChildPath:     pa.ChildPath,
			Action:        container.ReconciliationAction(pa.Action),
			ExternalState: pa.ExternalState,
			NewStatus:     newStatus,
		}
	}
	return actions
}

func convertLinkActions(payloadActions []LinkReconcileActionPayload) []container.LinkReconcileAction {
	actions := make([]container.LinkReconcileAction, len(payloadActions))
	for i, pa := range payloadActions {
		newStatus, _ := helpersv1.ParsePreciseLinkStatus(pa.NewStatus)
		actions[i] = container.LinkReconcileAction{
			LinkID:              pa.LinkID,
			ChildPath:           pa.ChildPath,
			Action:              container.ReconciliationAction(pa.Action),
			NewStatus:           newStatus,
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
		newStatus, _ := helpersv1.ParsePreciseResourceStatus(pa.NewStatus)
		actions[id] = &container.IntermediaryReconcileAction{
			IntermediaryID: id,
			Action:         container.ReconciliationAction(pa.Action),
			ExternalState:  pa.ExternalState,
			NewStatus:      newStatus,
		}
	}
	return actions
}
