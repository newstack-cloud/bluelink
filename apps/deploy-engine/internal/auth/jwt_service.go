package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
)

type jwtService struct {
	validator JWTValidator
}

// LoadJWTService loads a new instance of an auth.Checker
// that validates JSON Web Tokens (JWTs).
// This will use configuration from the provided auth.Config to wire up
// an appropriate JWT validator.
// This will make requests to the issuer URL as a part of the load process
// to determine how the JWKS URI should be configured.
func LoadJWTService(config *core.AuthConfig) (Checker, error) {
	provider, err := setupJWKSProvider(config)
	if err != nil {
		return nil, err
	}

	validator, err := validator.New(
		provider.KeyFunc,
		validator.SignatureAlgorithm(config.JWTSignatureAlgorithm),
		config.JWTIssuer,
		[]string{config.JWTAudience},
	)
	if err != nil {
		return nil, err
	}

	return &jwtService{
		validator: validator,
	}, nil
}

// Check verifies a JWT token in the provided request headers.
// It returns an error if the token is invalid,
// or nil if it is valid.
func (v *jwtService) Check(ctx context.Context, headers http.Header) error {
	token := headers.Get("Authorization")
	if strings.TrimSpace(token) == "" {
		return &Error{
			ChildErr: errors.New("missing Authorization header"),
		}
	}

	if !strings.HasPrefix(token, "Bearer ") {
		return &Error{
			ChildErr: errors.New("invalid Authorization header format"),
		}
	}

	// Strip the "Bearer " prefix
	finalToken := token[7:]

	_, err := v.validator.ValidateToken(ctx, finalToken)
	if err != nil {
		// Any error returned from the go-jwt-middleware
		// should be considered an authentication error.
		return &Error{
			ChildErr: err,
		}
	}

	return nil
}

func setupJWKSProvider(config *core.AuthConfig) (JWKSProvider, error) {
	opts := []any{}
	// Handle the case where the issuer is an OAuth2 authorization server
	// but not an OIDC provider.
	// As per the authentication spec for Bluelink components,
	// OAuth2 servers with a well-known endpoint need to be supported
	// as well as OIDC providers.
	oauthMetadata, err := getOAuthAuthzServerMetadata(config)
	if err != nil {
		if !errors.Is(err, errOAuthMetadataNotFound) {
			return nil, err
		}
	}

	if oauthMetadata.JWKSURI != "" {
		jwksURL, err := url.Parse(oauthMetadata.JWKSURI)
		if err != nil {
			return nil, err
		}

		opts = append(opts, jwks.WithCustomJWKSURI(jwksURL))
	}

	domainURL, err := url.Parse(
		createDomainURLString(config.JWTIssuer, config.JWTIssuerSecure),
	)
	if err != nil {
		return nil, err
	}

	return jwks.NewCachingProvider(
		domainURL,
		// Use the default cache time of 1 minute.
		/* cacheTTL */
		0,
		opts...,
	), nil
}

var (
	oauthMetadataTimeout     = 5 * time.Second
	errOAuthMetadataNotFound = errors.New("OAuth metadata not found")
)

func getOAuthAuthzServerMetadata(config *core.AuthConfig) (oauthMetadata, error) {
	client := &http.Client{
		Timeout: oauthMetadataTimeout,
	}

	resp, err := client.Get(oauthConfigURL(config.JWTIssuer, config.JWTIssuerSecure))
	if err != nil {
		return oauthMetadata{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return oauthMetadata{}, errOAuthMetadataNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return oauthMetadata{}, errors.New("failed to fetch OAuth metadata")
	}

	var metadata oauthMetadata
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthMetadata{}, err
	}

	err = json.Unmarshal(respBytes, &metadata)
	if err != nil {
		return oauthMetadata{}, err
	}

	return metadata, nil
}

type oauthMetadata struct {
	JWKSURI string `json:"jwks_uri"`
}

func oauthConfigURL(issuer string, secure bool) string {
	domainIssuerURL := createDomainURLString(issuer, secure)
	return fmt.Sprintf(
		"%s/.well-known/oauth-authorization-server",
		domainIssuerURL,
	)
}

func createDomainURLString(issuer string, secure bool) string {
	if secure {
		return fmt.Sprintf("https://%s", issuer)
	}
	return fmt.Sprintf("http://%s", issuer)
}

// JWTValidator provides an interface for a token validator
// that is compatible with the auth0 JWT middleware.
type JWTValidator interface {
	ValidateToken(ctx context.Context, tokenString string) (interface{}, error)
}

// JWKSProvider provides an interface for a provider
// that can provide a JSON Web Key Set (JWKS) for validating
// JSON Web Tokens (JWTs).
type JWKSProvider interface {
	KeyFunc(ctx context.Context) (any, error)
}
