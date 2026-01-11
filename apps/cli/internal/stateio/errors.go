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

// ExportErrorCode represents the type of export error.
type ExportErrorCode string

const (
	// ErrCodeExportFailed indicates a general export failure.
	ErrCodeExportFailed ExportErrorCode = "export_failed"
	// ErrCodeInstanceNotFound indicates one or more instances were not found.
	ErrCodeInstanceNotFound ExportErrorCode = "instance_not_found"
	// ErrCodeRemoteUploadFailed indicates a remote upload failed.
	ErrCodeRemoteUploadFailed ExportErrorCode = "remote_upload_failed"
)

// ExportError represents an error that occurred during export.
type ExportError struct {
	Code    ExportErrorCode
	Message string
	Err     error
}

func (e *ExportError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ExportError) Unwrap() error {
	return e.Err
}
