package httpmw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/stretchr/testify/suite"
)

const (
	clientIP    = "203.0.113.42"
	proxyIP     = "198.51.100.7"
	connectIP   = "10.0.0.1"
	connectAddr = "10.0.0.1:1234"
)

// Which address gets recorded against a caller's actions.
type ExtractIPMiddlewareSuite struct {
	suite.Suite
}

func (s *ExtractIPMiddlewareSuite) Test_direct_takes_the_connection_address() {
	s.Assert().Equal(connectIP, s.resolve(httpmw.DirectExtractionSource, nil))
}

func (s *ExtractIPMiddlewareSuite) Test_direct_ignores_every_header() {
	s.Assert().Equal(connectIP, s.resolve(httpmw.DirectExtractionSource, map[string]string{
		"X-Forwarded-For":  clientIP,
		"CF-Connecting-IP": clientIP,
		"X-Azure-ClientIP": clientIP,
	}))
}

func (s *ExtractIPMiddlewareSuite) Test_defaults_to_the_connection_address() {
	resolved, found := s.serve(nil, map[string]string{"X-Forwarded-For": clientIP})
	s.Require().True(found)
	s.Assert().Equal(connectIP, resolved, "the default must not trust a header")
}

func (s *ExtractIPMiddlewareSuite) Test_aws_alb_takes_the_address_it_appended() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.AWSALBExtractionSource,
		map[string]string{"X-Forwarded-For": clientIP},
	))
}

// An ALB appends the client port when routing.http.xff_client_port.enabled is set, so
// the entry is not an address until the port is taken off.
func (s *ExtractIPMiddlewareSuite) Test_aws_alb_strips_an_appended_port() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.AWSALBExtractionSource,
		map[string]string{"X-Forwarded-For": clientIP + ":45093"},
	))
}

// The caller sent its own entry and the balancer appended after it, so reading from the
// left would record whatever the caller chose.
func (s *ExtractIPMiddlewareSuite) Test_aws_alb_ignores_an_entry_the_caller_sent() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.AWSALBExtractionSource,
		map[string]string{"X-Forwarded-For": "1.2.3.4, " + clientIP},
	))
}

// A Google Cloud load balancer appends the client address and then its own, so the
// client is second from the right.
func (s *ExtractIPMiddlewareSuite) Test_gcp_load_balancer_takes_the_second_entry_from_the_right() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.GCPLoadBalancerExtractionSource,
		map[string]string{"X-Forwarded-For": clientIP + ", " + proxyIP},
	))
}

func (s *ExtractIPMiddlewareSuite) Test_gcp_load_balancer_ignores_an_entry_the_caller_sent() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.GCPLoadBalancerExtractionSource,
		map[string]string{"X-Forwarded-For": "1.2.3.4, " + clientIP + ", " + proxyIP},
	))
}

// Too few entries means the request did not come through the balancer at all, so there
// is no entry of its word to read.
func (s *ExtractIPMiddlewareSuite) Test_gcp_load_balancer_resolves_nothing_from_a_single_entry() {
	resolved, found := s.serve(
		optionsFor(httpmw.GCPLoadBalancerExtractionSource),
		map[string]string{"X-Forwarded-For": clientIP},
	)
	s.Assert().False(found)
	s.Assert().Empty(resolved)
}

func (s *ExtractIPMiddlewareSuite) Test_azure_app_gateway_strips_the_port_it_appends() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.AzureAppGatewayExtractionSource,
		map[string]string{"X-Forwarded-For": "1.2.3.4:5678, " + clientIP + ":45093"},
	))
}

// Front Door and Cloudflare set their own header rather than appending to one, so the
// whole value is theirs and a caller's cannot survive.
func (s *ExtractIPMiddlewareSuite) Test_azure_front_door_reads_its_own_header() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.AzureFrontDoorExtractionSource,
		map[string]string{
			"X-Azure-ClientIP": clientIP,
			"X-Forwarded-For":  "1.2.3.4",
		},
	))
}

func (s *ExtractIPMiddlewareSuite) Test_cloudflare_reads_its_own_header() {
	s.Assert().Equal(clientIP, s.resolve(
		httpmw.CloudflareExtractionSource,
		map[string]string{
			"CF-Connecting-IP": clientIP,
			"X-Forwarded-For":  "1.2.3.4",
		},
	))
}

func (s *ExtractIPMiddlewareSuite) Test_reads_ipv6_addresses_with_and_without_a_port() {
	const address = "2001:DB8::21f:5bff:febf:ce22:8a2e"

	s.Assert().Equal(address, s.resolve(
		httpmw.AWSALBExtractionSource,
		map[string]string{"X-Forwarded-For": address},
	))
	s.Assert().Equal(address, s.resolve(
		httpmw.AWSALBExtractionSource,
		map[string]string{"X-Forwarded-For": "[" + address + "]:50392"},
	))
}

// A value that is not an address is recorded as no address rather than as itself.
func (s *ExtractIPMiddlewareSuite) Test_resolves_nothing_from_a_value_that_is_not_an_address() {
	for _, forwarded := range []string{
		"not-an-address",
		"192.0.2.1; DROP TABLE audit_records",
		"",
	} {
		resolved, found := s.serve(
			optionsFor(httpmw.AWSALBExtractionSource),
			map[string]string{"X-Forwarded-For": forwarded},
		)
		s.Assert().False(found, "%q should resolve to nothing", forwarded)
		s.Assert().Empty(resolved, forwarded)
	}
}

// A misconfiguration shows as an absent address rather than a plausible wrong one.
// Startup rejects an unknown source before it reaches here.
func (s *ExtractIPMiddlewareSuite) Test_an_unknown_source_resolves_nothing() {
	resolved, found := s.serve(
		optionsFor(httpmw.IPExtractionSource("some-other-proxy")),
		map[string]string{"X-Forwarded-For": clientIP},
	)
	s.Assert().False(found)
	s.Assert().Empty(resolved)
}

func (s *ExtractIPMiddlewareSuite) Test_known_sources_covers_every_configurable_source() {
	known := httpmw.KnownSources()

	for _, source := range []httpmw.IPExtractionSource{
		httpmw.DirectExtractionSource,
		httpmw.AWSALBExtractionSource,
		httpmw.GCPLoadBalancerExtractionSource,
		httpmw.AzureAppGatewayExtractionSource,
		httpmw.AzureFrontDoorExtractionSource,
		httpmw.CloudflareExtractionSource,
	} {
		s.Assert().True(httpmw.IsKnownSource(source), source)
		s.Assert().Contains(known, source)
	}

	s.Assert().False(httpmw.IsKnownSource("some-other-proxy"))
	s.Assert().Len(known, 6, "KnownSources must list every source and nothing else")
}

func (s *ExtractIPMiddlewareSuite) Test_reports_no_source_ip_when_the_middleware_has_not_run() {
	request := httptest.NewRequest("POST", "/validations", nil)
	resolved, found := httpmw.SourceIPFromContext(request.Context())
	s.Assert().False(found)
	s.Assert().Empty(resolved)
}

func (s *ExtractIPMiddlewareSuite) resolve(
	source httpmw.IPExtractionSource,
	headers map[string]string,
) string {
	resolved, found := s.serve(optionsFor(source), headers)
	s.Require().True(found, "a source IP should have been resolved")

	return resolved
}

func (s *ExtractIPMiddlewareSuite) serve(
	opts []httpmw.ExtractIPMiddlewareOption,
	headers map[string]string,
) (string, bool) {
	var resolved string
	var found bool

	handler := httpmw.NewExtractIPMiddleware(opts...).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resolved, found = httpmw.SourceIPFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest("POST", "/validations", nil)
	request.RemoteAddr = connectAddr
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	handler.ServeHTTP(httptest.NewRecorder(), request)

	return resolved, found
}

func optionsFor(source httpmw.IPExtractionSource) []httpmw.ExtractIPMiddlewareOption {
	return []httpmw.ExtractIPMiddlewareOption{
		httpmw.WithExtractIPMiddlewareSource(source),
	}
}

func TestExtractIPMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(ExtractIPMiddlewareSuite))
}
