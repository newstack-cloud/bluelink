package deployengine

import "time"

const (
	// BluelinkAPIKeyHeaderName is the name of the header
	// used to pass the API key for authentication.
	BluelinkAPIKeyHeaderName = "Bluelink-Api-Key"
	// AuthorisationHeaderName is the name of the header
	// used to send a bearer token issued by an OAuth2 or OIDC provider.
	AuthorisationHeaderName = "Authorization"
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
	sendToClientStreamTimeout = 5 * time.Second
)
