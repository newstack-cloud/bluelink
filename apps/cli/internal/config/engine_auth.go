package config

import (
	"encoding/json"
	"os"
)

// EngineAuthConfig is the configuration for the deploy engine authentication.
type EngineAuthConfig struct {
	Method              string                     `json:"method"`
	APIKey              string                     `json:"apiKey"`
	OAuth2              *OAuth2Config              `json:"oauth2"`
	BluelinkSignatureV1 *BluelinkSignatureV1Config `json:"bluelinkSignatureV1"`
}

// OAuth2Config is the configuration for the OAuth2 authentication.
type OAuth2Config struct {
	// ClientID is used as a part of the client credentials grant type
	// to obtain an access token from the OAuth2 or OIDC provider.
	ClientID string `json:"clientId"`
	// ClientSecret is used as a part of the client credentials grant type
	// to obtain an access token from the OAuth2 or OIDC provider.
	ClientSecret string `json:"clientSecret"`
	// ProviderBaseURL is the base URL of the OAuth2 or OIDC provider.
	// This is the URL from which the client will use to obtain
	// the discovery document for the provider at either `/.well-known/openid-configuration`
	// or `/.well-known/oauth-authorization-server`.
	// When TokenEndpoint is set, this value is ignored.
	ProviderBaseURL string `json:"providerBaseURL"`
	// TokenEndpoint is the fully qualified URL of the token endpoint to use to obtain
	// an access token from the OAuth2 or OIDC provider.
	// When this value is left empty, the client will attempt to obtain the discovery document
	// from the ProviderBaseURL and use the token endpoint from that document.
	TokenEndpoint string `json:"tokenEndpoint"`
}

// BluelinkSignatureV1Config is the configuration for the Bluelink Signature v1 authentication.
type BluelinkSignatureV1Config struct {
	KeyPair       BluelinkSignatureV1KeyPair `json:"keyPair"`
	CustomHeaders []string                   `json:"customHeaders"`
}

// BluelinkSignatureV1KeyPair is the key pair for the Bluelink Signature v1 authentication.
type BluelinkSignatureV1KeyPair struct {
	KeyID     string `json:"keyId"`
	SecretKey string `json:"secretKey"`
}

// LoadEngineAuthConfig loads the engine authentication configuration from
// an auth config file.
func LoadEngineAuthConfig(confProvider *Provider) (*EngineAuthConfig, error) {
	engineAuthConfig := &EngineAuthConfig{}
	engineAuthConfigFile, _ := confProvider.GetString("engineAuthConfigFile")
	engineAuthConfigBytes, err := os.ReadFile(engineAuthConfigFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(engineAuthConfigBytes, engineAuthConfig)
	if err != nil {
		return nil, err
	}

	return engineAuthConfig, nil
}
