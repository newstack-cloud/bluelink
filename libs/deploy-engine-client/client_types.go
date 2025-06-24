package deployengine

import "github.com/newstack-cloud/bluelink/libs/common/sigv1"

// ConnectProtocol represents the protocol used to connect
// to an instance of the Bluelink Deploy Engine.
type ConnectProtocol int32

const (
	// ConnectProtocolTCP indicates that the client should connect
	// to the Bluelink Deploy Engine using HTTP over TCP.
	ConnectProtocolTCP ConnectProtocol = iota
	// ConnectProtocolUnixDomainSocket indicates that the client should connect
	// to the Bluelink Deploy Engine using a Unix domain socket.
	ConnectProtocolUnixDomainSocket
)

// AuthMethod represents the method of authentication that should be
// used to connect to an instance of the Bluelink Deploy Engine.
type AuthMethod int32

const (
	// AuthMethodAPIKey indicates that the client should
	// authenticate using an API key.
	AuthMethodAPIKey AuthMethod = iota
	// AuthMethodOAuth2 indicates that the client should
	// authenticate using OAuth where a token is obtained
	// from a third-party identity provider through the
	// client credentials grant type.
	// This version of the Deploy Engine supports OAuth2/OIDC
	// providers that produce JWTs for access tokens that are compatible
	// with the auth method documentation that can be found here:
	// https://www.bluelink.dev/docs/auth/jwts
	AuthMethodOAuth2
	// AuthMethodBluelinkSignatureV1 indicates that the client should
	// authenticate using the Bluelink Signature v1 method.
	// See: https://www.bluelink.dev/docs/auth/signature-v1
	AuthMethodBluelinkSignatureV1
)

type AuthConfig struct {
	// Method specifies the authentication method to use for requests
	// to the Bluelink Deploy Engine.
	Method AuthMethod
	// APIKey is the API key to use to authenticate requests
	// when Method is `AuthMethodAPIKey`.
	APIKey string
	// OAuth2Config is the OAuth configuration to use to authenticate
	// requests when Method is `AuthMethodOAuth2`.
	OAuth2Config *OAuth2Config
	// BluelinkSignatureKeyPair is the Bluelink Signature v1 key pair
	// to use to authenticate requests when Method is `AuthMethodBluelinkSignatureV1`.
	BluelinkSignatureKeyPair *sigv1.KeyPair
	// BluelinkSignatureCustomHeaders is a list of custom headers
	// to include in the signed message for the Bluelink Signature v1
	// authentication method.
	BluelinkSignatureCustomHeaders []string
}

// OAuth2Config contains the configuration for gaining access to the Deploy Engine
// using an OAuth2 or OIDC provider.
type OAuth2Config struct {
	// ProviderBaseURL is the base URL of the OAuth2 or OIDC provider.
	// This is the URL from which the client will use to obtain
	// the discovery document for the provider at either `/.well-known/openid-configuration`
	// or `/.well-known/oauth-authorization-server`.
	// When TokenEndpoint is set, this value is ignored.
	ProviderBaseURL string
	// TokenEndpoint is the fully qualified URL of the token endpoint to use to obtain
	// an access token from the OAuth2 or OIDC provider.
	// When this value is left empty, the client will attempt to obtain the discovery document
	// from the ProviderBaseURL and use the token endpoint from that document.
	TokenEndpoint string
	// ClientID is used as a part of the client credentials grant type
	// to obtain an access token from the OAuth2 or OIDC provider.
	ClientID string
	// ClientSecret is used as a part of the client credentials grant type
	// to obtain an access token from the OAuth2 or OIDC provider.
	ClientSecret string
}
