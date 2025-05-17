package deployengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/two-hundred/celerity/libs/blueprint-state/manage"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	"github.com/two-hundred/celerity/libs/common/sigv1"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/internal/oauth2"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/internal/sseconfig"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/types"
	"golang.org/x/oauth2/clientcredentials"
)

// Client provides a client implementation of the Celerity Deploy Engine API.
// This supports v1 of the API including the v1 streaming interface.
type Client struct {
	endpoint             string
	protocol             ConnectProtocol
	unixDomainSocket     string
	authConfig           *AuthConfig
	defaultHTTPTransport *http.Transport
	createRoundTripper   func(transport *http.Transport) http.RoundTripper
	requestTimeout       time.Duration
	streamTimeout        time.Duration
	httpClient           *http.Client
	streamHTTPClient     *http.Client
	oauthHTTPClient      *http.Client
	credentialsHelper    oauth2.CredentialsHelper
	clock                core.Clock
	logger               core.Logger
}

// ClientOption is a function that configures the client.
type ClientOption func(*Client)

// WithClientEndpoint configures the endpoint to use to connect
// to the Celerity Deploy Engine.
// When an endpoint is not provided, the client will use `http://localhost:8325`.
// When the protocol is set to `ConnectProtocolUnixDomainSocket`,
// the endpoint will be ignored and the client will use a placeholder
// endpoint of "http://unix" to make sure the underlying HTTP client does
// not try to resolve the endpoint via DNS.
func WithClientEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// WithClientConnectProtocol configures the protocol to use to connect
// to the Celerity Deploy Engine.
// This can be either `ConnectProtocolTCP` or `ConnectProtocolUnixDomainSocket`.
// When a protocol is not provided, the client will default to `ConnectProtocolTCP`.
func WithClientConnectProtocol(protocol ConnectProtocol) ClientOption {
	return func(c *Client) {
		c.protocol = protocol
	}
}

// WithClientUnixDomainSocket configures the Unix domain socket to use
// to connect to the Celerity Deploy Engine.
// This is only used when the protocol is set to `ConnectProtocolUnixDomainSocket`.
// When a Unix domain socket is not provided, the client will default to `/tmp/celerity.sock`.
func WithClientUnixDomainSocket(socket string) ClientOption {
	return func(c *Client) {
		c.unixDomainSocket = socket
	}
}

// WithClientAuthMethod configures the authentication method to use
// to connect to the Celerity Deploy Engine.
// This can be either `AuthMethodAPIKey`, `AuthMethodOAuth2` or
// `AuthMethodCeleritySignatureV1`.
// When an authentication method is not provided, the client will default to `AuthMethodCeleritySignatureV1`.
func WithClientAuthMethod(method AuthMethod) ClientOption {
	return func(c *Client) {
		c.authConfig.Method = method
	}
}

// WithClientAPIKey configures the API key to use to authenticate
// to the Celerity Deploy Engine.
// This is only used when the authentication method is set to `AuthMethodAPIKey`.
// When an API key is not provided, the client will not be able to authenticate
// to the Celerity Deploy Engine.
func WithClientAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.authConfig.APIKey = apiKey
	}
}

// WithClientOAuth2Config configures the OAuth2 configuration to use
// to authenticate to the Celerity Deploy Engine.
// This is only used when the authentication method is set to `AuthMethodOAuth2`.
// OAuth2 configuration must be provided when the authentication method
// is set to `AuthMethodOAuth2`.
func WithClientOAuth2Config(config *OAuth2Config) ClientOption {
	return func(c *Client) {
		c.authConfig.OAuth2Config = config
	}
}

// WithClientCeleritySigv1KeyPair configures the Celerity Signature v1
// configuration to use to authenticate to the Celerity Deploy Engine.
// This is only used when the authentication method is set to `AuthMethodCeleritySignatureV1`.
// Celerity Signature v1 configuration must be provided when the authentication method
// is set to `AuthMethodCeleritySignatureV1`.
func WithClientCeleritySigv1KeyPair(keyPair *sigv1.KeyPair) ClientOption {
	return func(c *Client) {
		c.authConfig.CeleritySignatureKeyPair = keyPair
	}
}

// WithClientCeleritySigv1CustomHeaders configures the custom headers to use
// to authenticate to the Celerity Deploy Engine using the Celerity Signature v1 method.
// This is only used when the authentication method is set to `AuthMethodCeleritySignatureV1`.
// Celerity Signature v1 configuration must be provided when the authentication method
// is set to `AuthMethodCeleritySignatureV1`.
// This is a list of headers that will be included in the signed message
// when creating the signature header.
func WithClientCeleritySigv1CustomHeaders(headers []string) ClientOption {
	return func(c *Client) {
		c.authConfig.CeleritySignatureCustomHeaders = headers
	}
}

// WithClientHTTPRoundTripper configures the HTTP round tripper to use
// to connect to the Celerity Deploy Engine.
// This is used to configure the HTTP client with a custom transport
// that supports retries and other features.
// This is a function that takes a transport and returns a round tripper
// as there is core configuration that needs to be applied to the underlying
// transport (e.g. Unix domain socket support) in all cases.
// This round tripper will only be used for standard HTTP requests and not streaming requests.
// When a round tripper is not provided, the client will default to a transport
// that supports retries with exponential backoff and jitter configured
// with reasonable defaults.
func WithClientHTTPRoundTripper(
	createRoundTripper func(transport *http.Transport) http.RoundTripper,
) ClientOption {
	return func(c *Client) {
		c.createRoundTripper = createRoundTripper
	}
}

// WithClientRequestTimeout configures the request timeout to use
// to connect to the Celerity Deploy Engine.
// This is used to configure the HTTP client with a custom timeout
// for requests.
// When a timeout is not provided, the client will default to 60 seconds.
// This only applies to standard HTTP requests and not streaming requests.
func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.requestTimeout = timeout
	}
}

// WithClientStreamTimeout configures the stream timeout to use
// to connect to the Celerity Deploy Engine.
// This is used to configure the HTTP client with a custom timeout
// for streaming requests.
// When a timeout is not provided, the client will default to 3 hours.
func WithClientStreamTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.streamTimeout = timeout
	}
}

// WithClientClock configures the clock to use
// to get the current time and measure elapsed time.
// When a clock is not provided, the client will default
// to the current system clock.
func WithClientClock(clock core.Clock) ClientOption {
	return func(c *Client) {
		c.clock = clock
	}
}

// WithClientLogger configures the logger to use
// to log messages from the Celerity Deploy Engine client.
// When a logger is not provided, the client will default
// to a no-op logger that does not log any messages.
func WithClientLogger(logger core.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a client for an instance of
// the Celerity Deploy Engine v1 API.
// If an OAuth2/OIDC provider is configured, the client
// will fetch the discovery document from the provider if
// a token endpoint is not provided during client creation.
func NewClient(
	opts ...ClientOption,
) (*Client, error) {

	client := &Client{
		endpoint:         DefaultEndpoint,
		protocol:         DefaultProtocol,
		unixDomainSocket: DefaultUnixDomainSocket,
		authConfig: &AuthConfig{
			Method:                         DefaultAuthMethod,
			CeleritySignatureCustomHeaders: []string{},
		},
		defaultHTTPTransport: http.DefaultTransport.(*http.Transport),
		requestTimeout:       DefaultRequestTimeout,
		streamTimeout:        DefaultStreamTimeout,
		logger:               core.NewNopLogger(),
		clock:                &core.SystemClock{},
	}

	for _, opt := range opts {
		opt(client)
	}

	client.httpClient = &http.Client{
		Timeout: client.requestTimeout,
		Transport: finaliseTransport(
			client,
			/* tcpOnly */ false,
		),
	}

	client.streamHTTPClient = createHTTPClientForSSE(client)

	client.oauthHTTPClient = &http.Client{
		Timeout: client.requestTimeout,
		Transport: finaliseTransport(
			client,
			// Only the deploy engine API supports connections over unix domain sockets,
			// OAuth2/OIDC providers are always expected to be accessible over TCP.
			/* tcpOnly */ true,
		),
	}

	if client.protocol == ConnectProtocolUnixDomainSocket {
		client.endpoint = "http://unix"
	}

	err := client.setupOAuth2CredentialsHelper()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateBlueprintValidation creates a new blueprint validation
// for the provided blueprint document and starts the validation process.
// This will return an ID that can be used to stream the validation events
// as they occur.
// This is the `POST {baseURL}/v1/validations` API endpoint.
func (c *Client) CreateBlueprintValidation(
	ctx context.Context,
	payload *types.CreateBlueprintValidationPayload,
	query *types.CreateBlueprintValidationQuery,
) (*manage.BlueprintValidation, error) {
	url := fmt.Sprintf(
		"%s/v1/validations",
		c.endpoint,
	)
	queryParams := createBlueprintValidationQueryToQueryParams(query)

	blueprintValidation := &manage.BlueprintValidation{}
	err := c.startMutatingAction(
		ctx,
		url,
		"POST",
		payload,
		blueprintValidation,
		queryParams,
	)
	if err != nil {
		return nil, err
	}

	return blueprintValidation, nil
}

// GetBlueprintValidation retrieves metadata and status information
// about a blueprint validation.
// To get validation events (diagnostics), use the `StreamBlueprintValidationEvents`
// method.
// This is the `GET {baseURL}/v1/validations/{id}` API endpoint.
func (c *Client) GetBlueprintValidation(
	ctx context.Context,
	validationID string,
) (*manage.BlueprintValidation, error) {
	url := fmt.Sprintf(
		"%s/v1/validations/%s",
		c.endpoint,
		validationID,
	)

	blueprintValidation := &manage.BlueprintValidation{}
	err := c.getResource(
		ctx,
		url,
		blueprintValidation,
	)
	if err != nil {
		return nil, err
	}

	return blueprintValidation, nil
}

// StreamBlueprintValidationEvents streams events from a blueprint
// validation process.
// This will produce a stream of events as they occur or that have
// recently occurred.
// Any HTTP errors that occur when estabilishing a connection will be sent
// to the provided error channel.
// This comes with built-in re-connect logic that makes use of the
// the `Last-Event-ID` header to resume the stream from the last
// event received.
// This is the `GET {baseURL}/v1/validations/{id}/stream` API SSE stream endpoint.
func (c *Client) StreamBlueprintValidationEvents(
	ctx context.Context,
	validationID string,
	streamTo chan<- types.BlueprintValidationEvent,
	errChan chan<- error,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/v1/validations/%s/stream",
		c.endpoint,
		validationID,
	)

	client := sse.NewClient(
		url,
		sseconfig.WithHeaders(headers),
		sseconfig.WithHTTPClient(c.streamHTTPClient),
		sseconfig.WithResponseValidator(
			c.createStreamResponseValidator(
				errChan,
			),
		),
	)

	internalEventChan := make(chan *sse.Event)
	// Subscribe with a context to give caller more control
	// over cancelling the stream.
	// The stream timeout configured with the stream HTTP client
	// will be used even if the provided context does not have a timeout.
	go client.SubscribeChanWithContext(ctx, "messages", internalEventChan)

	go c.handleValidationStreamEvents(
		client,
		internalEventChan,
		streamTo,
		errChan,
	)

	return nil
}

func (c *Client) handleValidationStreamEvents(
	client *sse.Client,
	internalEventChan chan *sse.Event,
	streamTo chan<- types.BlueprintValidationEvent,
	errChan chan<- error,
) {
	handleStreamEvents(
		"validation",
		client,
		internalEventChan,
		streamTo,
		errChan,
		sseToBlueprintValidationEvent,
		checkIsValidationStreamEnd,
		c.streamTimeout,
		c.logger,
	)
}

// CleanupBlueprintValidations cleans up blueprint validation that are
// older than the retention period configured for the Deploy Engine instance.
// This is the `POST {baseURL}/v1/validations/cleanup` API endpoint.
func (c *Client) CleanupBlueprintValidations(
	ctx context.Context,
) error {
	return c.cleanupResources(
		ctx,
		fmt.Sprintf(
			"%s/v1/validations/cleanup",
			c.endpoint,
		),
	)
}

// CreateChangeset creates a change set for a blueprint deployment.
// This will start a change staging process for the provided blueprint
// document and return an ID that can be used to retrieve the change set
// or stream change staging events.
//
// If a valid instance ID or name is provided, a change set will be created
// by comparing the provided blueprint document with the current state of the
// existing blueprint instance.
//
// Creating a change set should be carried out in preparation for deploying new blueprint
// instances or updating existing blueprint instances.
//
// This is the `POST {baseURL}/v1/deployments/changes` API endpoint.
func (c *Client) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*manage.Changeset, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/changes",
		c.endpoint,
	)

	changeset := &manage.Changeset{}
	err := c.startMutatingAction(
		ctx,
		url,
		"POST",
		payload,
		changeset,
		/* queryParams */ map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return changeset, nil
}

// GetChangeset retrieves a change set for a blueprint deployment.
// This will return the current status of the change staging process.
// If complete, the response will include a full set of changes that
// will be applied when deploying the change set.
// This is the `GET {baseURL}/v1/deployments/changes/{id}` API endpoint.
func (c *Client) GetChangeset(
	ctx context.Context,
	changesetID string,
) (*manage.Changeset, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/changes/%s",
		c.endpoint,
		changesetID,
	)

	changeset := &manage.Changeset{}
	err := c.getResource(
		ctx,
		url,
		changeset,
	)
	if err != nil {
		return nil, err
	}

	return changeset, nil
}

// StreamChangeStagingEvents streams events from the change staging process
// for the given change set ID.
// This will produce a stream of events as they occur or that have recently occurred.
// Any HTTP errors that occur when estabilishing a connection will be sent
// to the provided error channel.
// This comes with built-in re-connect logic that makes use of the
// the `Last-Event-ID` header to resume the stream from the last
// event received.
// This is the `GET {baseURL}/v1/deployments/changes/{id}/stream` API SSE stream endpoint.
func (c *Client) StreamChangeStagingEvents(
	ctx context.Context,
	changesetID string,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/v1/deployments/changes/%s/stream",
		c.endpoint,
		changesetID,
	)

	client := sse.NewClient(
		url,
		sseconfig.WithHeaders(headers),
		sseconfig.WithHTTPClient(c.streamHTTPClient),
		sseconfig.WithResponseValidator(
			c.createStreamResponseValidator(
				errChan,
			),
		),
	)

	internalEventChan := make(chan *sse.Event)
	// Subscribe with a context to give caller more control
	// over cancelling the stream.
	// The stream timeout configured with the stream HTTP client
	// will be used even if the provided context does not have a timeout.
	go client.SubscribeChanWithContext(ctx, "messages", internalEventChan)

	go c.handleChangeStagingStreamEvents(
		client,
		internalEventChan,
		streamTo,
		errChan,
	)

	return nil
}

func (c *Client) handleChangeStagingStreamEvents(
	client *sse.Client,
	internalEventChan chan *sse.Event,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) {
	handleStreamEvents(
		"change staging",
		client,
		internalEventChan,
		streamTo,
		errChan,
		sseToChangeStagingEvent,
		checkIsChangeStagingStreamEnd,
		c.streamTimeout,
		c.logger,
	)
}

// CleanupChangesets cleans up change sets that are older than the retention
// period configured for the Deploy Engine instance.
// This is the `POST {baseURL}/v1/deployments/changes/cleanup` API endpoint.
func (c *Client) CleanupChangesets(
	ctx context.Context,
) error {
	return c.cleanupResources(
		ctx,
		fmt.Sprintf(
			"%s/v1/deployments/changes/cleanup",
			c.endpoint,
		),
	)
}

// CreateBlueprintInstance (Deploy New) creates a new blueprint deployment instance.
// This will start the deployment process for the provided blueprint
// document and change set.
// It will return a blueprint instance resource containing an ID that
// can be used to stream deployment events as they occur.
// This is the `POST {baseURL}/v1/deployments/instances` API endpoint.
func (c *Client) CreateBlueprintInstance(
	ctx context.Context,
	payload *types.BlueprintInstancePayload,
) (*state.InstanceState, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/instances",
		c.endpoint,
	)

	instance := &state.InstanceState{}
	err := c.startMutatingAction(
		ctx,
		url,
		"POST",
		payload,
		instance,
		/* queryParams */ map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// UpdateBlueprintInstance (Deploy Existing) updates an existing blueprint
// deployment instance.
// This will start the deployment process for the provided blueprint
// document and change set.
// It will return the current state of the blueprint instance,
// the same ID provided should be used to stream deployment events as they occur.
// This is the `PATCH {baseURL}/v1/deployments/instances/{id}` API endpoint.
func (c *Client) UpdateBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.BlueprintInstancePayload,
) (*state.InstanceState, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/instances/%s",
		c.endpoint,
		instanceID,
	)

	instance := &state.InstanceState{}
	err := c.startMutatingAction(
		ctx,
		url,
		"PATCH",
		payload,
		instance,
		/* queryParams */ map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetBlueprintInstance retrieves a blueprint deployment instance.
// This will return the current status of the deployment along with
// the current state of the blueprint intance.
// This is the `GET {baseURL}/v1/deployments/instances/{id}` API endpoint.
func (c *Client) GetBlueprintInstance(
	ctx context.Context,
	instanceID string,
) (*state.InstanceState, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/instances/%s",
		c.endpoint,
		instanceID,
	)

	instance := &state.InstanceState{}
	err := c.getResource(
		ctx,
		url,
		instance,
	)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetBlueprintInstanceExports retrieves the exports from a blueprint
// deployment instance.
// This will return the exported fields from the blueprint instance.
// This is the `GET {baseURL}/v1/deployments/instances/{id}/exports` API endpoint.
func (c *Client) GetBlueprintInstanceExports(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/instances/%s/exports",
		c.endpoint,
		instanceID,
	)

	exports := map[string]*state.ExportState{}
	err := c.getResource(
		ctx,
		url,
		&exports,
	)
	if err != nil {
		return nil, err
	}

	return exports, nil
}

// DestroyBlueprintInstance destroys a blueprint deployment instance.
// This will start the destroy process for the provided change set.
// It will return the current state of the blueprint instance,
// the same ID provided should be used to stream destroy events as they occur.
// This is the `POST {baseURL}/v1/deployments/instances/{id}/destroy` API endpoint.
func (c *Client) DestroyBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.DestroyBlueprintInstancePayload,
) (*state.InstanceState, error) {
	url := fmt.Sprintf(
		"%s/v1/deployments/instances/%s/destroy",
		c.endpoint,
		instanceID,
	)

	instance := &state.InstanceState{}
	err := c.startMutatingAction(
		ctx,
		url,
		"POST",
		payload,
		instance,
		/* queryParams */ map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// StreamBlueprintInstanceEvents streams events from the current deployment
// process for the given blueprint instance ID.
//
// This will stream events for new deployments, updates and for destroying
// a blueprint instance.
//
// This will produce a stream of events as they occur or that have recently occurred.
//
// For a blueprint instance that has been destroyed, this stream will no longer be available
// to new connections once the destroy process has been successfully completed.
//
// Any HTTP errors that occur when estabilishing a connection or unexpected failures
// in the deployment process will be sent to the provided error channel.
//
// This comes with built-in re-connect logic that makes use of the
// the `Last-Event-ID` header to resume the stream from the last
// event received.
//
// This is the `GET {baseURL}/v1/deployments/instances/{id}/stream` API SSE stream endpoint.
func (c *Client) StreamBlueprintInstanceEvents(
	ctx context.Context,
	instanceID string,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/v1/deployments/instances/%s/stream",
		c.endpoint,
		instanceID,
	)

	client := sse.NewClient(
		url,
		sseconfig.WithHeaders(headers),
		sseconfig.WithHTTPClient(c.streamHTTPClient),
		sseconfig.WithResponseValidator(
			c.createStreamResponseValidator(
				errChan,
			),
		),
	)

	internalEventChan := make(chan *sse.Event)
	// Subscribe with a context to give caller more control
	// over cancelling the stream.
	// The stream timeout configured with the stream HTTP client
	// will be used even if the provided context does not have a timeout.
	go client.SubscribeChanWithContext(ctx, "messages", internalEventChan)

	go c.handleBlueprintInstanceStreamEvents(
		client,
		internalEventChan,
		streamTo,
		errChan,
	)

	return nil
}

func (c *Client) handleBlueprintInstanceStreamEvents(
	client *sse.Client,
	internalEventChan chan *sse.Event,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) {
	handleStreamEvents(
		"blueprint instance",
		client,
		internalEventChan,
		streamTo,
		errChan,
		sseToBlueprintInstanceEvent,
		checkIsBlueprintInstanceStreamEnd,
		c.streamTimeout,
		c.logger,
	)
}

// CleanupEvents cleans up events that are older than the retention
// period configured for the Deploy Engine instance.
//
// This will clean up events for all processes including blueprint validations,
// change staging and deployments. This will not clean up the resources themselves,
// only the events that are associated with them.
// You can clean up change sets and blueprint validations using the dedicated methods.
// This is the `POST {baseURL}/v1/events/cleanup` API endpoint.
func (c *Client) CleanupEvents(
	ctx context.Context,
) error {
	return c.cleanupResources(
		ctx,
		fmt.Sprintf(
			"%s/v1/events/cleanup",
			c.endpoint,
		),
	)
}

func (c *Client) cleanupResources(
	ctx context.Context,
	url string,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", url, nil,
	)
	if err != nil {
		return createRequestPrepError(
			fmt.Sprintf(
				"failed to prepare request: %s",
				err.Error(),
			),
		)
	}
	attachHeaders(req, headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return createRequestError(
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return createClientError(resp)
	}

	return nil
}

func (c *Client) startMutatingAction(
	ctx context.Context,
	url string,
	method string,
	payload any,
	respTarget any,
	queryParams map[string]string,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return createSerialiseError(
			fmt.Sprintf(
				"failed to serialise payload: %s",
				err.Error(),
			),
		)
	}

	req, err := http.NewRequestWithContext(
		ctx, method, url, bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return createRequestPrepError(
			fmt.Sprintf(
				"failed to prepare request: %s",
				err.Error(),
			),
		)
	}
	attachHeaders(req, headers)
	attachQueryParams(req, queryParams)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return createRequestError(
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return createClientError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(respTarget); err != nil {
		return createDeserialiseError(
			fmt.Sprintf(
				"failed to decode response: %s",
				err.Error(),
			),
		)
	}

	return nil
}

func (c *Client) getResource(
	ctx context.Context,
	url string,
	respTarget any,
) error {
	headers, err := c.prepareAuthHeaders()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "GET", url, nil,
	)
	if err != nil {
		return createRequestPrepError(
			fmt.Sprintf(
				"failed to prepare request: %s",
				err.Error(),
			),
		)
	}
	attachHeaders(req, headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return createRequestError(
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return createClientError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(respTarget); err != nil {
		return createDeserialiseError(
			fmt.Sprintf(
				"failed to decode response: %s",
				err.Error(),
			),
		)
	}

	return nil
}

func (c *Client) setupOAuth2CredentialsHelper() error {
	if c.authConfig.Method != AuthMethodOAuth2 {
		c.logger.Debug(
			"skipping OAuth2 credentials helper setup, not using OAuth2 auth method",
		)
		return nil
	}

	tokenEndpoint := getTokenEndpoint(c.authConfig.OAuth2Config)
	if tokenEndpoint == "" {
		metadataHelper := oauth2.NewMetadataHelper(
			getProviderBaseURL(c.authConfig.OAuth2Config),
			c.oauthHTTPClient,
			c.logger,
		)

		var err error
		tokenEndpoint, err = metadataHelper.GetTokenEndpoint()
		if err != nil {
			return createAuthInitError(
				fmt.Sprintf(
					"failed to get token endpoint from provider: %s",
					err.Error(),
				),
			)
		}
	}

	c.credentialsHelper = oauth2.NewCredentialsHelper(
		&clientcredentials.Config{
			ClientID:     getClientID(c.authConfig.OAuth2Config),
			ClientSecret: getClientSecret(c.authConfig.OAuth2Config),
			TokenURL:     tokenEndpoint,
		},
		c.oauthHTTPClient,
		context.Background(),
	)

	return nil
}

func (c *Client) prepareAuthHeaders() (map[string]string, error) {
	if c.authConfig.Method == AuthMethodAPIKey {
		return map[string]string{
			CelerityAPIKeyHeaderName: c.authConfig.APIKey,
		}, nil
	}

	if c.authConfig.Method == AuthMethodOAuth2 {
		return c.prepareOAuth2Headers()
	}

	if c.authConfig.Method == AuthMethodCeleritySignatureV1 {
		return c.prepareCeleritySignatureV1Headers()
	}

	return nil, createAuthPrepError(
		"no valid authentication method configured",
	)
}

func (c *Client) prepareOAuth2Headers() (map[string]string, error) {
	accessToken, err := c.credentialsHelper.GetAccessToken()
	if err != nil {
		return nil, createAuthPrepError(
			fmt.Sprintf("failed to get access token: %s", err.Error()),
		)
	}

	return map[string]string{
		AuthorisationHeaderName: fmt.Sprintf(
			"Bearer %s",
			accessToken,
		),
	}, nil
}

func (c *Client) prepareCeleritySignatureV1Headers() (map[string]string, error) {
	if c.authConfig.CeleritySignatureKeyPair == nil {
		return nil, createAuthPrepError(
			"no Celerity Signature v1 key pair configured",
		)
	}

	httpHeaders := make(http.Header)
	signatureHeader, err := sigv1.CreateSignatureHeader(
		&sigv1.KeyPair{
			KeyID:     c.authConfig.CeleritySignatureKeyPair.KeyID,
			SecretKey: c.authConfig.CeleritySignatureKeyPair.SecretKey,
		},
		httpHeaders,
		c.authConfig.CeleritySignatureCustomHeaders,
		c.clock,
	)
	if err != nil {
		return nil, createAuthPrepError(
			fmt.Sprintf(
				"failed to create Celerity Signature v1 header: %s",
				err.Error(),
			),
		)
	}

	httpHeaders.Set(
		sigv1.SignatureHeaderName,
		signatureHeader,
	)

	headers := make(map[string]string)
	for key, value := range httpHeaders {
		headers[key] = value[0]
	}

	return headers, nil
}

func (c *Client) createStreamResponseValidator(errChan chan<- error) sse.ResponseValidator {
	return func(sseClient *sse.Client, resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			clientErr := createClientError(resp)
			// Close the response body to avoid leaking resources.
			// This must be done after preparing the client error
			// as the body needs to be read when producing the error.
			resp.Body.Close()
			// Send client error to an error channel to get it out of the SSE client package,
			// r3labs/sse/v2 does not have an api for http error handling.
			// Using the response validator is the only way to get the error out of the sse.Client.
			errChan <- clientErr
			return clientErr
		}

		return nil
	}
}
