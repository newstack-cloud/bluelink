package objectstore

import (
	"fmt"
)

// Error is a custom error type that provides errors specific to
// the object store implementation of the object store state container.
type Error struct {
	ReasonCode ErrorReasonCode
	Err        error
}

func (e *Error) Error() string {
	return fmt.Sprintf("object store state error (%s): %s", e.ReasonCode, e.Err)
}

// ErrorReasonCode is an enum of possible error reasons that can be returned by the object store implementation.
type ErrorReasonCode string

const (
	// ErrorReasonCodeObjectNotFound is the error code that is used when
	// an object is not found in the object store.
	ErrorReasonCodeObjectNotFound ErrorReasonCode = "object_not_found"

	// ErrorReasonCodePreconditionFailed is the error code that is used when
	// a precondition for an operation is not met, such as an ETag mismatch.
	ErrorReasonCodePreconditionFailed ErrorReasonCode = "precondition_failed"

	// ErrorReasonCodeAuthFailed is the error code that is used when
	// authentication with the object store fails.
	ErrorReasonCodeAuthFailed ErrorReasonCode = "authentication_failed"

	// ErrorReasonCodeRateLimited is the error code that is used when
	// the object store rate limits the client.
	ErrorReasonCodeRateLimited ErrorReasonCode = "rate_limited"
)

// NewObjectNotFound returns an Error with ReasonCodeObjectNotFound. Exported
// so provider-specific Service implementations can construct the canonical
// error shape from their SDK-native not-found signals.
func NewObjectNotFound(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeObjectNotFound,
		Err:        fmt.Errorf("object not found: %s", message),
	}
}

// NewPreconditionFailed returns an Error with ReasonCodePreconditionFailed.
// Used by provider Services when IfMatch / IfNoneMatch conditional writes
// fail.
func NewPreconditionFailed(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodePreconditionFailed,
		Err:        fmt.Errorf("precondition failed: %s", message),
	}
}

// NewAuthFailed returns an Error with ReasonCodeAuthFailed.
func NewAuthFailed(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeAuthFailed,
		Err:        fmt.Errorf("authentication failed: %s", message),
	}
}

// NewRateLimited returns an Error with ReasonCodeRateLimited.
func NewRateLimited(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeRateLimited,
		Err:        fmt.Errorf("rate limited: %s", message),
	}
}
