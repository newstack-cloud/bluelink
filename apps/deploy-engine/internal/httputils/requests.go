package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// DecodeRequestBody decodes a JSON request body into the provided payload
// and returns true if an error occurred and a response has been sent to the client.
func DecodeRequestBody(w http.ResponseWriter, r *http.Request, payload any, logger core.Logger) bool {
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		logger.Debug(
			"failed to parse the request body",
			core.ErrorLogField("error", err),
		)
		HTTPError(w, http.StatusBadRequest, "failed to parse the request body")
		return true
	}

	return false
}
