package sigv1

import (
	"fmt"
)

// ErrorCode is an integer type that represents
// different error codes for the sigv1 package.
type ErrorCode int

const (
	// ErrCodeSignatureHeaderMissing is for when the signature header
	// is not present in the request headers.
	ErrCodeSignatureHeaderMissing ErrorCode = iota
	// ErrCodeInvalidSignatureHeaderFormat is for when the signature
	// header is not in the correct format.
	ErrCodeInvalidSignatureHeaderFormat
	// ErrCodeInvalidKeyID is for when
	// the key ID in the signature header does not match any of
	// the key IDs in the known key pairs.
	ErrCodeInvalidKeyID
	// ErrCodeDateHeaderMissing is for when the date header is not
	// present in the request headers.
	ErrCodeDateHeaderMissing
	// ErrCodeInvalidDateHeader is for when the date header is not
	// a valid unix timestamp that can be parsed as an integer.
	ErrCodeInvalidDateHeader
	// ErrCodeCustomHeaderMissing is for when a custom header
	// included in the signature header string is not present
	// in the request headers.
	ErrCodeCustomHeaderMissing
	// ErrCodeFailedToDecodeSignature is for when the signature
	// cannot be decoded as a base64 url safe string.
	ErrCodeFailedToDecodeSignature
	// ErrCodeInvalidSignature is for when signature
	// verification fails due to a mismatch between the
	// calculated signature and the signature in the header.
	ErrCodeInvalidSignature
	// ErrCodeSignatureExpired is for when the date used in creating
	// the signature is outside of the allowed time window.
	ErrCodeSignatureExpired
)

var (
	// ErrSignatureHeaderMissing is returned when the signature header
	// is not present in the request headers.
	ErrSignatureHeaderMissing = &Error{
		Code: ErrCodeSignatureHeaderMissing,
		Message: "signature verification failed due to the signature header " +
			"not being in the provided request headers",
	}

	// ErrInvalidSignatureHeaderFormat is returned when the signature
	// header is not in the correct format.
	ErrInvalidSignatureHeaderFormat = &Error{
		Code: ErrCodeInvalidSignatureHeaderFormat,
		Message: "signature verification failed due to the signature header " +
			"not being in the correct format, expected " +
			"'keyId=\"{keyId}\", headers=\"{header1} {header2} ...\", signature=\"{signature}\"'",
	}

	// ErrInvalidKeyID is returned when the key ID in the signature
	// header does not match any of the key IDs in the known key pairs.
	ErrInvalidKeyID = &Error{
		Code:    ErrCodeInvalidKeyID,
		Message: "signature verification failed due to an invalid key ID",
	}

	// ErrDateHeaderMissing is returned when the date header is not present
	// in the request headers.
	ErrDateHeaderMissing = &Error{
		Code: ErrCodeDateHeaderMissing,
		Message: "signature verification failed due to the date header " +
			"not being in the provided request headers",
	}

	// ErrInvalidDateHeader is returned when the date header is not
	// a valid unix timestamp that can be parsed as an integer.
	ErrInvalidDateHeader = &Error{
		Code: ErrCodeInvalidDateHeader,
		Message: "signature verification failed due to the date header " +
			"not being a valid unix timestamp",
	}

	// ErrFailedToDecodeSignature is returned when the signature
	// cannot be decoded a basee64 url safe string.
	ErrFailedToDecodeSignature = &Error{
		Code: ErrCodeFailedToDecodeSignature,
		Message: "signature verification failed due to the signature " +
			"being a valid url safe base64 string",
	}

	// ErrInvalidSignature is returned when signature
	// verification fails due to a mismatch between the
	// calculated signature and the signature in the header.
	ErrInvalidSignature = &Error{
		Code: ErrCodeInvalidSignature,
		Message: "signature verification failed due to the signature " +
			"not matching the expected signature",
	}

	// ErrSignatureExpired is returned when the date used in creating
	// the signature is outside of the allowed time window.
	ErrSignatureExpired = &Error{
		Code:    ErrCodeSignatureExpired,
		Message: "signature verification failed due to an expired signature",
	}
)

func errCustomHeaderMissing(header string) error {
	return &Error{
		Code: ErrCodeCustomHeaderMissing,
		Message: fmt.Sprintf(
			"signature verification failed due to the %s header "+
				"not being in the provided request headers",
			header,
		),
	}
}

// Error is a custom error type for verification
// and signing errors in the sigv1 package.
type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("bluelink sigv1 error: %s", e.Message)
}
