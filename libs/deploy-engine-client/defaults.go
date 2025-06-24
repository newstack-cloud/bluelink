package deployengine

import (
	"time"
)

const (
	// DefaultEndpoint specifies the default endpoint to use to connect
	// to the Bluelink Deploy Engine when no endpoint is specified for
	// the client.
	// There is no default production endpoint for the Deploy Engine
	// so it makes more sense for the end user to default to a version
	// of the Deploy Engine that is running locally as a part of a standard
	// Bluelink installation or otherwise.
	DefaultEndpoint = "http://localhost:8325"
	// DefaultProtocol specifies the default connection protocol to use
	// to connect to the Bluelink Deploy Engine when no protocol is specified
	// for the client.
	DefaultProtocol = ConnectProtocolTCP
	// DefaultUnixDomainSocket specifies the default Unix domain socket
	// file to use to connect to the Bluelink Deploy Engine when
	// the protocol is set to `ConnectProtocolUnixDomainSocket`.
	DefaultUnixDomainSocket = "/tmp/bluelink.sock"
	// DefaultAuthMethod specifies the default authentication method to use
	// to connect to the Bluelink Deploy Engine when no authentication method
	// is specified for the client.
	// This defaults to the Bluelink Signature v1 method which is the more secure
	// of the simpler authentication methods.
	DefaultAuthMethod = AuthMethodBluelinkSignatureV1
	// DefaultRequestTimeout specifies the default timeout to use for HTTP requests
	// to the Bluelink Deploy Engine when no timeout is specified for the client.
	// This only applies to standard HTTP requests and not streaming requests.
	DefaultRequestTimeout = 60 * time.Second
	// DefaultStreamTimeout specifies the default timeout to use for streaming
	// requests to the Bluelink Deploy Engine when no timeout is specified for
	// the client.
	// This default is high to allow for long-running processes like deployments
	// of infrastructure that can take a long time to complete.
	DefaultStreamTimeout = 3 * time.Hour
)
