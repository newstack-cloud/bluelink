package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/stretchr/testify/suite"
)

type APIKeyServiceSuite struct {
	suite.Suite
}

func (s *APIKeyServiceSuite) Test_check_verifies_a_valid_api_key() {
	service := NewAPIKeyService(
		&core.AuthConfig{
			APIKeys: []string{"valid-key-1", "valid-key-2"},
		},
	)

	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "valid-key-2")

	err := service.Check(context.Background(), headers)
	s.NoError(err)
}

func (s *APIKeyServiceSuite) Test_check_fails_for_invalid_api_key() {
	service := NewAPIKeyService(
		&core.AuthConfig{
			APIKeys: []string{"valid-key-3", "valid-key-4"},
		},
	)

	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "invalid-key")
	err := service.Check(context.Background(), headers)
	s.Error(err)
	authErr, ok := err.(*Error)
	s.True(ok)
	s.Equal("invalid API key", authErr.ChildErr.Error())
}

func (s *APIKeyServiceSuite) Test_check_fails_for_missing_api_key() {
	service := NewAPIKeyService(
		&core.AuthConfig{
			APIKeys: []string{"valid-key-5", "valid-key-6"},
		},
	)

	headers := make(http.Header)
	err := service.Check(context.Background(), headers)
	s.Error(err)
	authErr, ok := err.(*Error)
	s.True(ok)
	s.Equal("missing API key", authErr.ChildErr.Error())
}

func TestAPIKeyServiceSuite(t *testing.T) {
	suite.Run(t, new(APIKeyServiceSuite))
}
