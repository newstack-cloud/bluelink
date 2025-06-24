package resolve

import (
	"errors"
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/blueprint"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
)

// ResolveBlueprintForRequest resolves a blueprint for a HTTP request.
// This will return the result containing resolved blueprint info
// and a boolean, that if true, indicates that an error occurred and a response
// has been sent to the client.
func ResolveBlueprintForRequest(
	r *http.Request,
	w http.ResponseWriter,
	documentInfo *BlueprintDocumentInfo,
	resolver includes.ChildResolver,
	logger core.Logger,
) (*includes.ChildBlueprintInfo, bool) {
	include, err := BlueprintDocumentInfoToInclude(documentInfo)
	if err != nil {
		var invalidLocationMetadataErr *InvalidLocationMetadataError
		if errors.As(err, &invalidLocationMetadataErr) {
			httputils.HTTPError(
				w,
				http.StatusBadRequest,
				invalidLocationMetadataErr.Reason,
			)
			return nil, true
		}
	}

	params := blueprint.CreateEmptyBlueprintParams()
	blueprintInfo, err := resolver.Resolve(
		r.Context(),
		// We are adapting to load top-level blueprints, so any include name
		// works but "root" appropriately distinguishes this use case in the logs.
		/* includeName */
		"root",
		include,
		params,
	)
	if err != nil {
		logger.Debug(
			"failed to resolve the blueprint",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(
			w,
			http.StatusBadRequest,
			"the provided blueprint could not be resolved",
		)
		return nil, true
	}

	return blueprintInfo, false
}
