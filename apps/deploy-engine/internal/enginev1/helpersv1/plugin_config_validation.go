package helpersv1

import (
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// PrepareAndValidatePluginConfig prepares and validates the plugin configuration
// for a blueprint operation. This will write an error response to the provided
// http.ResponseWriter if there are any errors during preparation or validation.
func PrepareAndValidatePluginConfig(
	r *http.Request,
	w http.ResponseWriter,
	inputConfig *types.BlueprintOperationConfig,
	validate bool,
	pluginConfigPreparer pluginconfig.Preparer,
	logger core.Logger,
) (*types.BlueprintOperationConfig, []*core.Diagnostic, bool) {
	preparedConfig, diagnostics, err := pluginConfigPreparer.Prepare(
		r.Context(),
		inputConfig,
		validate,
	)
	if err != nil {
		logger.Debug(
			"failed to prepare plugin config",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return nil, nil, true
	}

	if utils.HasAtLeastOneError(diagnostics) {
		validationErrors := &typesv1.ValidationDiagnosticErrors{
			Message:               "plugin configuration validation failed",
			ValidationDiagnostics: diagnostics,
		}
		httputils.HTTPJSONResponse(
			w,
			http.StatusUnprocessableEntity,
			validationErrors,
		)
		return nil, nil, true
	}

	// Surface warning and info diagnostics to the user
	// by returning them to be passed in as the initial events
	// in the current process stream.
	return preparedConfig, diagnostics, false
}
