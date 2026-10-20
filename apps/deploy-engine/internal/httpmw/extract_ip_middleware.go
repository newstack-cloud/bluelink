// Package httpmw holds HTTP middleware that runs ahead of the API's own handling.
package httpmw

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"
)

// IPExtractionSource names what sits in front of the engine, which is what decides how a
// request's source IP is determined.
//
// Which header to read, and whether to trust one at all, is a property of the deployment
// rather than of the engine. A header is the caller's own claim unless something in the
// request path is known to rewrite or append to it.
type IPExtractionSource string

const (
	// DirectExtractionSource takes the address the connection came from and ignores
	// every header.
	//
	// The default, because an engine reached directly is the case where a forwarded
	// header is purely caller-supplied and trusting one would let any caller choose the
	// address recorded against its actions.
	DirectExtractionSource IPExtractionSource = "direct"

	// AWSALBExtractionSource reads the address an AWS application load balancer
	// appended to X-Forwarded-For.
	AWSALBExtractionSource IPExtractionSource = "aws-alb"

	// GCPLoadBalancerExtractionSource reads the address a Google Cloud external
	// HTTP(S) load balancer appended to X-Forwarded-For.
	//
	// That balancer appends two entries, the client address followed by its own, so the
	// client is the second from the right rather than the last.
	GCPLoadBalancerExtractionSource IPExtractionSource = "gcp-lb"

	// AzureAppGatewayExtractionSource reads the address an Azure Application Gateway
	// appended to X-Forwarded-For, which carries a port.
	AzureAppGatewayExtractionSource IPExtractionSource = "azure-app-gateway"

	// AzureFrontDoorExtractionSource reads the address Azure Front Door set on
	// X-Azure-ClientIP.
	AzureFrontDoorExtractionSource IPExtractionSource = "azure-front-door"

	// CloudflareExtractionSource reads the address Cloudflare set on CF-Connecting-IP.
	CloudflareExtractionSource IPExtractionSource = "cloudflare"
)

const forwardedForHeader = "X-Forwarded-For"

// How one source's address is found.
type ipSourceRule struct {
	header string
	// fromRight is the 1-based position of the trustworthy entry counted from the right
	// of a comma-separated header, for a proxy that appends to it.
	//
	// Counted from the right because that is the only end a caller cannot reach, it can
	// send a header of its own, and the proxy's entries go after whatever it sent.
	// Counting from the left would read the caller's claim instead.
	//
	// Zero means the proxy sets the header outright rather than appending, so the whole
	// value is its answer and a caller's cannot survive.
	fromRight int
}

var ipSourceRules = map[IPExtractionSource]ipSourceRule{
	AWSALBExtractionSource:          {header: forwardedForHeader, fromRight: 1},
	GCPLoadBalancerExtractionSource: {header: forwardedForHeader, fromRight: 2},
	AzureAppGatewayExtractionSource: {header: forwardedForHeader, fromRight: 1},
	AzureFrontDoorExtractionSource:  {header: "X-Azure-ClientIP"},
	CloudflareExtractionSource:      {header: "CF-Connecting-IP"},
}

// KnownSources returns every source that can be configured, for validating what an
// operator asked for before the engine starts serving.
func KnownSources() []IPExtractionSource {
	sources := make([]IPExtractionSource, 0, len(ipSourceRules)+1)
	sources = append(sources, DirectExtractionSource)
	for source := range ipSourceRules {
		sources = append(sources, source)
	}
	slices.Sort(sources)

	return sources
}

// IsKnownSource reports whether the given source is one this middleware implements.
func IsKnownSource(source IPExtractionSource) bool {
	_, known := ipSourceRules[source]

	return known || source == DirectExtractionSource
}

// contextKey is unexported so that nothing outside this package can set the source IP on
// a context. A value in the context is this middleware's word on where the request came
// from, and a handler that could overwrite it would make the audit trail arguable.
type contextKey struct{}

// ExtractIPMiddleware determines the source IP of a request and puts it on the request
// context, so that everything downstream agrees on one answer rather than each deriving
// its own.
type ExtractIPMiddleware struct {
	source IPExtractionSource
}

// ExtractIPMiddlewareOption configures an ExtractIPMiddleware.
type ExtractIPMiddlewareOption func(*ExtractIPMiddleware)

// WithExtractIPMiddlewareSource sets what sits in front of the engine, in place of
// DirectExtractionSource.
func WithExtractIPMiddlewareSource(source IPExtractionSource) ExtractIPMiddlewareOption {
	return func(middleware *ExtractIPMiddleware) {
		middleware.source = source
	}
}

// NewExtractIPMiddleware creates the middleware that resolves a request's source IP.
func NewExtractIPMiddleware(opts ...ExtractIPMiddlewareOption) *ExtractIPMiddleware {
	middleware := &ExtractIPMiddleware{
		source: DirectExtractionSource,
	}
	for _, opt := range opts {
		opt(middleware)
	}

	return middleware
}

func (m *ExtractIPMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(
			context.WithValue(r.Context(), contextKey{}, deriveSourceIP(r, m.source)),
		))
	})
}

// SourceIPFromContext returns the source IP the middleware resolved for the request, and
// whether one was resolved at all.
//
// An absent value means no address could be trusted rather than that the request had
// none, so a caller records nothing rather than guessing.
func SourceIPFromContext(ctx context.Context) (string, bool) {
	sourceIP, ok := ctx.Value(contextKey{}).(string)

	return sourceIP, ok && sourceIP != ""
}

// Validated after the port is taken off, not before.
//
// A gateway that appends a port sends "192.168.0.3:45093", which is not an address until
// the port is removed. Rejecting it at that point would record the proxy rather than the
// client, and would do so silently.
//
// An unknown source resolves nothing rather than falling back to the connection address,
// so a misconfiguration shows as an absent address rather than as a plausible wrong one.
// Startup rejects one before it gets this far.
func deriveSourceIP(r *http.Request, source IPExtractionSource) string {
	candidate := ""
	switch source {
	case DirectExtractionSource:
		candidate = remoteAddrIP(r)
	default:
		rule, known := ipSourceRules[source]
		if !known {
			return ""
		}
		candidate = headerIP(r, rule)
	}

	if net.ParseIP(candidate) == nil {
		return ""
	}

	return candidate
}

func headerIP(r *http.Request, rule ipSourceRule) string {
	value := strings.TrimSpace(r.Header.Get(rule.header))
	if rule.fromRight == 0 {
		return stripPort(value)
	}

	entries := strings.Split(value, ",")
	index := len(entries) - rule.fromRight
	// Fewer entries than the proxy is expected to have appended means the request did
	// not come through it, so there is no address of its word to read.
	if index < 0 {
		return ""
	}

	return stripPort(strings.TrimSpace(entries[index]))
}

// Removes a port from an address, leaving one that has none alone.
func stripPort(address string) string {
	// An IPv6 address with a port, as "[2001:db8::1]:50392". The opening bracket goes
	// with the port.
	if bracketed, _, found := strings.Cut(address, "]:"); found {
		return strings.TrimPrefix(bracketed, "[")
	}

	// An IPv4 address with a port, as "192.168.0.3:45093". Guarded on the colon count so
	// that a bare IPv6 address, which has several, is left alone.
	if strings.Count(address, ":") == 1 {
		host, _, _ := strings.Cut(address, ":")
		return host
	}

	return address
}

func remoteAddrIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
