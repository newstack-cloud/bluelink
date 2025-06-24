package deployengine

import (
	"context"
	"net"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/httputils"
)

func finaliseTransport(client *Client, tcpOnly bool) http.RoundTripper {
	baseTransport := client.defaultHTTPTransport
	if !tcpOnly && client.protocol == ConnectProtocolUnixDomainSocket {
		baseTransport = createUnixDomainSocketTransport(
			client.defaultHTTPTransport,
			client.unixDomainSocket,
		)
	}

	if client.createRoundTripper != nil {
		return client.createRoundTripper(baseTransport)
	}

	// Default to a retryable transport with the default retry config
	// when a custom round tripper is not provided.
	return httputils.NewRetryableTransport(
		baseTransport,
	)
}

func createHTTPClientForSSE(
	client *Client,
) *http.Client {
	transport := client.defaultHTTPTransport
	if client.protocol == ConnectProtocolUnixDomainSocket {
		transport = createUnixDomainSocketTransport(
			client.defaultHTTPTransport,
			client.unixDomainSocket,
		)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   client.streamTimeout,
	}
}

func createUnixDomainSocketTransport(
	baseTransport *http.Transport,
	unixDomainSocket string,
) *http.Transport {
	unixTransport := baseTransport.Clone()
	defaultDialContext := unixTransport.DialContext
	unixTransport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return defaultDialContext(ctx, "unix", unixDomainSocket)
	}
	return unixTransport
}
