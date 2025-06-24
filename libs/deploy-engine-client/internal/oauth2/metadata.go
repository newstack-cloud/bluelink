package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// MetadataHelper is an interface for a service that can retrieve the token endpoint
// from an OAuth2 or OIDC provider's discovery document.
type MetadataHelper interface {
	// GetTokenEndpoint retrieves the token endpoint from
	// the provider's discovery document.
	GetTokenEndpoint() (string, error)
}

type metadataHelperImpl struct {
	providerBaseURL string
	client          *http.Client
	logger          core.Logger
}

// NewMetadataHelper creates a new instance of a service that can retrieve
// the token endpoint from an OAuth2 or OIDC provider's discovery document.
func NewMetadataHelper(
	providerBaseURL string,
	client *http.Client,
	logger core.Logger,
) MetadataHelper {
	return &metadataHelperImpl{
		providerBaseURL: providerBaseURL,
		client:          client,
		logger:          logger,
	}
}

func (h *metadataHelperImpl) GetTokenEndpoint() (string, error) {
	metadata, err := h.fetchMetadata()
	if err != nil {
		return "", err
	}

	return metadata["token_endpoint"].(string), nil
}

func (h *metadataHelperImpl) fetchMetadata() (map[string]any, error) {
	// Fetch the discovery document from the provider's base URL
	// and return the token endpoint.
	oidcURL := buildDiscoveryURL(h.providerBaseURL, "oidc")

	metadata, err := h.fetchDiscoveryDocument(oidcURL)
	if err != nil {
		h.logger.Debug(
			"Failed to fetch openid-configuratinon discovery document: %v",
			core.ErrorLogField("error", err),
		)
		oauth2URL := buildDiscoveryURL(h.providerBaseURL, "oauth2")
		return h.fetchDiscoveryDocument(oauth2URL)
	}

	return metadata, nil
}

func (h *metadataHelperImpl) fetchDiscoveryDocument(url string) (map[string]any, error) {
	h.logger.Debug("Fetching discovery document", core.StringLogField("url", url))
	resp, err := h.client.Get(url)
	if err != nil {
		h.logger.Debug(
			"Failed to fetch discovery document",
			core.StringLogField("url", url),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Debug(
			"Failed to fetch discovery document",
			core.StringLogField("url", url),
			core.IntegerLogField("statusCode", int64(resp.StatusCode)),
		)
		return nil, fmt.Errorf("failed to fetch discovery document: %s", resp.Status)
	}

	var metadata map[string]any

	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		h.logger.Debug(
			"Failed to decode discovery document",
			core.StringLogField("url", url),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}
	if _, ok := metadata["token_endpoint"]; !ok {
		h.logger.Debug(
			"Token endpoint not found in discovery document",
			core.StringLogField("url", url),
		)
		return nil, fmt.Errorf("token endpoint not found in discovery document")
	}
	h.logger.Debug(
		"Successfully fetched discovery document",
		core.StringLogField("url", url),
		core.StringLogField("token_endpoint", metadata["token_endpoint"].(string)),
	)
	return metadata, nil
}

func buildDiscoveryURL(baseURL, providerType string) string {
	if providerType == "oidc" {
		return fmt.Sprintf("%s/.well-known/openid-configuration", baseURL)
	}
	return fmt.Sprintf("%s/.well-known/oauth-authorization-server", baseURL)
}
