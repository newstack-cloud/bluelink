package stateio

import "fmt"

// ImportErrorCode represents the type of import error.
type ImportErrorCode string

const (
	// ErrCodeInvalidJSON indicates the JSON input is malformed.
	ErrCodeInvalidJSON ImportErrorCode = "invalid_json"
	// ErrCodeFileNotFound indicates the input file was not found.
	ErrCodeFileNotFound ImportErrorCode = "file_not_found"
	// ErrCodeRemoteAccessFail indicates a remote file could not be accessed.
	ErrCodeRemoteAccessFail ImportErrorCode = "remote_access_failed"
)

// ImportError represents an error that occurred during import.
type ImportError struct {
	Code    ImportErrorCode
	Message string
	Err     error
}

func (e *ImportError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ImportError) Unwrap() error {
	return e.Err
}
