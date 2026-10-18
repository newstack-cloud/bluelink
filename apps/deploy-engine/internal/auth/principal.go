package auth

import "context"

// Authentication methods a principal can be established by.
const (
	// MethodOIDC identifies a principal established from a JWT issued by an OIDC provider
	// or OAuth2 authorization server.
	MethodOIDC = "oidc"
	// MethodSigV1 identifies a principal established from a Bluelink Signature V1 header.
	MethodSigV1 = "sigv1"
	// MethodAPIKey identifies a principal established from a configured API key.
	MethodAPIKey = "apikey"
)

// Principal identifies who made a request.
//
// Attribution is the engine's to establish even though authorization is not, the engine is
// the only thing that sees the credential, and it is discarded once the check has passed,
// so nothing above the engine could work out who acted.
type Principal struct {
	// Method is the authentication method the principal was established by.
	Method string
	// Subject identifies the principal within that method.
	// This is a JWT's subject claim, a Signature V1 key ID, or an API key's fingerprint.
	//
	// Never the credential itself. An API key's subject is a digest prefix, so that a
	// record naming the key cannot be replayed as the key.
	Subject string
	// Name is a friendly name for the principal where the method carries one.
	Name string
	// Claims holds the raw claims of a JWT, for an audit sink to record.
	Claims map[string]any
	// OnBehalfOf is an end-user identity asserted by a trusted fronting service.
	OnBehalfOf string
}

type principalContextKey struct{}

// ContextWithPrincipal returns a context carrying the principal a request authenticated as.
func ContextWithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext returns the principal a request authenticated as, and whether one
// was established.
//
// Absent for a route that authentication is excluded from, so a caller has to handle a
// request that has no principal rather than assuming one.
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(*Principal)
	return principal, ok && principal != nil
}
