package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
)

const (
	// BluelinkAPIKeyHeaderName is the name of the header
	// used to pass the API key for authentication.
	BluelinkAPIKeyHeaderName = "Bluelink-Api-Key"
)

type apiKeyService struct {
	apiKeys []string
}

// NewAPIKeyService creates a new service that checks
// if the provided API key is valid against a list of
// known API keys.
func NewAPIKeyService(config *core.AuthConfig) Checker {
	return &apiKeyService{
		apiKeys: config.APIKeys,
	}
}

// Check verifies if the provided API key is valid.
// It returns an error if the key is invalid,
// or nil if it is valid.
func (s *apiKeyService) Check(ctx context.Context, headers http.Header) error {
	apiKey := headers.Get(BluelinkAPIKeyHeaderName)
	if apiKey == "" {
		return &Error{
			ChildErr: errors.New("missing API key"),
		}
	}

	matches := slices.Contains(s.apiKeys, apiKey)
	if !matches {
		return &Error{
			ChildErr: errors.New("invalid API key"),
		}
	}

	return nil
}
