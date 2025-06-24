package deployengine

import (
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

func attachHeaders(
	req *http.Request,
	headers map[string]string,
) {
	for headerName, value := range headers {
		req.Header.Set(headerName, value)
	}
}

func attachQueryParams(
	req *http.Request,
	queryParams map[string]string,
) {
	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()
}

func createBlueprintValidationQueryToQueryParams(
	opts *types.CreateBlueprintValidationQuery,
) map[string]string {
	queryParams := make(map[string]string)
	if opts != nil {
		if opts.CheckBlueprintVars {
			queryParams["checkBlueprintVars"] = "true"
		}
		if opts.CheckPluginConfig {
			queryParams["checkPluginConfig"] = "true"
		}
	}
	return queryParams
}
