package deployengine

import "time"

const (
	// BluelinkAPIKeyHeaderName is the name of the header
	// used to pass the API key for authentication.
	BluelinkAPIKeyHeaderName = "Bluelink-Api-Key"
	// AuthorisationHeaderName is the name of the header
	// used to send a bearer token issued by an OAuth2 or OIDC provider.
	AuthorisationHeaderName = "Authorization"
	// LastEventIDHeaderName is the name of the header used to specify
	// the starting event ID for SSE streaming.
	LastEventIDHeaderName = "Last-Event-ID"
	// ChannelTypeValidation is the channel type identifier
	// for validation events.
	ChannelTypeValidation = "validation"
	// ChannelTypeChangeset is the channel type identifier
	// for change staging (change set) events.
	ChannelTypeChangeset = "changeset"
	// ChannelTypeDeployment is the channel type identifier
	// for deployment events.
	ChannelTypeDeployment = "deployment"
)

const (
	// An internal timeout to wait for the client "streamTo" channel to
	// receive a message before closing the connection to the server used
	// for an SSE stream.
	//
	// This guards against a consumer that has gone away, so the goroutine and
	// its connection are not held forever. It is deliberately generous as a
	// consumer is not necessarily reading the channel at any given moment.
	// For example, a TUI takes an event, updates its model and renders before asking for
	// the next one and closing the stream aborts the operation the server
	// is running, not just the client's view of it. A deployment killed
	// mid-flight leaves half-created infrastructure behind, so the cost of
	// being too eager here is far higher than the cost of holding an idle
	// goroutine for a few minutes.
	sendToClientStreamTimeout = 5 * time.Minute
)
