package corefunctions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// HTTPResourceFunction provides the implementation of
// a function that fetches data from public HTTP/HTTPS URLs.
type HTTPResourceFunction struct {
	definition *function.Definition
	httpClient *http.Client
}

// NewHTTPResourceFunction creates a new instance of the HTTPResourceFunction with
// a complete function definition.
func NewHTTPResourceFunction() provider.Function {
	return &HTTPResourceFunction{
		definition: &function.Definition{
			Description: "A function that fetches data from public HTTP/HTTPS URLs and returns it as raw bytes.",
			FormattedDescription: "A function that fetches data from public HTTP/HTTPS URLs and returns it as raw bytes.\n\n" +
				"**Examples:**\n\n" +
				"Fetching public configuration data:\n" +
				"```\n${http_resource(\"https://raw.githubusercontent.com/example/config/main/app.json\")}\n```\n\n" +
				"Fetching public release files:\n" +
				"```\n${http_resource(\"https://github.com/example/app/releases/download/v1.0.0/app.zip\")}\n```\n\n" +
				"Fetching public documentation:\n" +
				"```\n${http_resource(\"https://example.com/docs/api-schema.json\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "url",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The public HTTP/HTTPS URL to fetch data from.",
				},
			},
			Return: &function.ValueTypeDefinitionScalar{
				Type: function.ValueTypeBytes,
			},
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (f *HTTPResourceFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *HTTPResourceFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var url string
	if err := input.Arguments.GetVar(ctx, 0, &url); err != nil {
		return nil, err
	}

	// Validate URL scheme
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for %q: %w", url, err)
	}

	// Execute the request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resource from %q: %w", url, err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request to %q failed with status code %d", url, resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %q: %w", url, err)
	}

	return &provider.FunctionCallOutput{
		ResponseData: data,
	}, nil
}
