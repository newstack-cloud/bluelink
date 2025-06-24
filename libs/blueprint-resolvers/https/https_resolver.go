package resolverhttps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

type httpsChildResolver struct {
	client *http.Client
}

// NewResolver creates a new instance of a ChildResolver
// that resolves child blueprints from public HTTPS URLs.
func NewResolver(client *http.Client) includes.ChildResolver {
	return &httpsChildResolver{
		client,
	}
}

func (r *httpsChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	path := core.StringValue(include.Path)
	if path == "" {
		return nil, includes.ErrInvalidPath(includeName, "https")
	}

	err := utils.ValidateInclude(include, includeName, []string{"host"}, "HTTPS", "https")
	if err != nil {
		return nil, err
	}

	host := core.StringValue(include.Metadata.Fields["host"])
	url := buildURL(host, path)
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, includes.ErrBlueprintNotFound(includeName, url)
	}

	if isPermErrorStatusCode(resp.StatusCode) {
		return nil, includes.ErrPermissions(
			includeName,
			url,
			fmt.Errorf("HTTP status code: %d", resp.StatusCode),
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	blueprintSource, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	blueprintSourceStr := string(blueprintSource)
	return &includes.ChildBlueprintInfo{
		BlueprintSource: &blueprintSourceStr,
	}, nil
}

func buildURL(host, path string) string {
	pathWithLoadingSlash := path
	if !strings.HasPrefix(path, "/") {
		pathWithLoadingSlash = fmt.Sprintf("/%s", path)
	}
	return fmt.Sprintf("https://%s%s", host, pathWithLoadingSlash)
}

func isPermErrorStatusCode(statusCode int) bool {
	return statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized
}
