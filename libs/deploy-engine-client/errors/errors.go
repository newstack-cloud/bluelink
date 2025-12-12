package errors

import (
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// AuthPrepError is an error type that indicates an issue with
// preparing authentication headers for a request.
type AuthPrepError struct {
	Message string
}

func (e *AuthPrepError) Error() string {
	return fmt.Sprintf("auth prep error: %s", e.Message)
}

// AuthInitError is an error type that indicates an issue with
// initialising auth configuration when creating a new
// deploy engine client.
type AuthInitError struct {
	Message string
}

func (e *AuthInitError) Error() string {
	return fmt.Sprintf("auth init error: %s", e.Message)
}

// SerialiseError is an error type that indicates an issue with
// serialising a request or response.
type SerialiseError struct {
	Message string
}

func (e *SerialiseError) Error() string {
	return fmt.Sprintf("serialise error: %s", e.Message)
}

// DeserialiseError is an error type that indicates an issue with
// deserialising a request or response.
type DeserialiseError struct {
	Message string
}

func (e *DeserialiseError) Error() string {
	return fmt.Sprintf("deserialise error: %s", e.Message)
}

// RequestPrepError is an error type that indicates an issue with
// preparing a request for the deploy engine client.
type RequestPrepError struct {
	Message string
}

func (e *RequestPrepError) Error() string {
	return fmt.Sprintf("request prep error: %s", e.Message)
}

// RequestError is an error type that indicates an issue with
// making a request to the deploy engine client.
// This will usually wrap a network or timeout error.
type RequestError struct {
	Err error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request error: %s", e.Err.Error())
}

// ClientError is an error type that indicates an unexpected
// response from the deploy engine client.
// This will usually wrap a non-2xx status code and
// an error message from the server.
type ClientError struct {
	StatusCode int
	Message    string
	// An optional list of validation errors that will usually
	// be populated for 422 responses for input validation errors.
	ValidationErrors []*ValidationError
	// An optional list of diagnostics that provide additional information
	// about validation errors related to failed attempts to load a blueprint
	// document.
	// This will usually be populated for 422 responses.
	ValidationDiagnostics []*core.Diagnostic
}

// ValidationError is a struct that represents a validation error
// that can be returned in responses to clients.
type ValidationError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Type     string `json:"type"`
}

// Response is a struct that represents a JSON error response
// from the Deploy Engine API.
type Response struct {
	Message     string             `json:"message"`
	Errors      []*ValidationError `json:"errors,omitempty"`
	Diagnostics []*core.Diagnostic `json:"validationDiagnostics,omitempty"`
}

func (e *ClientError) Error() string {
	return fmt.Sprintf(
		"client error: %s (status code: %d)",
		e.Message,
		e.StatusCode,
	)
}

// StreamError is an error type that indicates an unexpected error
// during an operation that can be streamed.
type StreamError struct {
	Event *types.StreamErrorMessageEvent
}

func (e *StreamError) Error() string {
	diagnosticsMessage := ""
	if len(e.Event.Diagnostics) > 0 {
		diagnosticWord := "diagnostics"
		if len(e.Event.Diagnostics) == 1 {
			diagnosticWord = "diagnostic"
		}
		diagnosticsMessage = fmt.Sprintf(
			" (%d %s)",
			len(e.Event.Diagnostics),
			diagnosticWord,
		)
	}

	return fmt.Sprintf(
		"stream error: %s%s",
		e.Event.Message,
		diagnosticsMessage,
	)
}

// IsNotFoundError checks if the error is a client error
// with a 404 status code, indicating that the requested resource
// was not found.
// This also returns the concrete client error that allows the caller
// access to more precise information about the error.
func IsNotFoundError(err error) (*ClientError, bool) {
	if clientErr, ok := err.(*ClientError); ok {
		return clientErr, clientErr.StatusCode == http.StatusNotFound
	}
	return nil, false
}

// IsValidationError checks if the error is a client error
// with a 422 or 400 status code, indicating that one or more elements
// of the request failed validation.
// This also returns the concrete client error that allows the caller
// access to more precise information about the error.
func IsValidationError(err error) (*ClientError, bool) {
	if clientErr, ok := err.(*ClientError); ok {
		return clientErr, clientErr.StatusCode == http.StatusUnprocessableEntity ||
			clientErr.StatusCode == http.StatusBadRequest
	}
	return nil, false
}
