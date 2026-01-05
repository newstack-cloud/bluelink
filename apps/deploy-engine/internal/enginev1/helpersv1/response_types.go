package helpersv1

// MessageResponse is a data type for a JSON response
// that contains a message string.
type MessageResponse struct {
	Message string `json:"message"`
}

// AsyncOperationResponse wraps responses for async operations that have SSE streams.
// The LastEventID can be used as the starting offset for SSE streaming.
type AsyncOperationResponse[T any] struct {
	// LastEventID is the ID of the last event for the channel before this operation started.
	// Clients should pass this to streaming methods to avoid missing events.
	// Omitted from JSON if no events have been generated yet for the channel.
	LastEventID string `json:"lastEventId,omitempty"`
	// Data contains the core response data.
	Data T `json:"data"`
}
