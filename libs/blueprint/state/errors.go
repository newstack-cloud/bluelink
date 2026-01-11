package state

import "fmt"

// Error is an error type that MUST be used by implementations
// of the state.Container interface to allow the engine to be able to distinguish
// between failures and expected errors such as resource not found.
type Error struct {
	Code   ErrorCode
	ItemID string
}

type ErrorCode string

const (
	// ErrResourceNotFound is used when a resource could not be found
	// in a blueprint instance.
	ErrResourceNotFound ErrorCode = "resource_not_found"
	// ErrLinkNotFound is used when a link could not be found
	// in a blueprint instance.
	ErrLinkNotFound ErrorCode = "link_not_found"
	// ErrInstanceNotFound is used when a blueprint instance could not be found.
	ErrInstanceNotFound ErrorCode = "instance_not_found"
	// ErrExportNotFound is the error code that is used when
	// an export is not found in the state.
	ErrExportNotFound ErrorCode = "export_not_found"
)

func (e *Error) Error() string {
	switch e.Code {
	case ErrResourceNotFound:
		return fmt.Sprintf("StateError: resource %q not found", e.ItemID)
	case ErrLinkNotFound:
		return fmt.Sprintf("StateError: link %q not found", e.ItemID)
	case ErrInstanceNotFound:
		return fmt.Sprintf("StateError: instance %q not found", e.ItemID)
	case ErrExportNotFound:
		return fmt.Sprintf("StateError: export %q not found", e.ItemID)
	default:
		return "StateError: unknown error"
	}
}

// ResourceNotFoundError is a helper function that creates a new error
// for when a resource can not be found in the persistence that backs
// a state container.
func ResourceNotFoundError(resourceID string) *Error {
	return &Error{
		Code:   ErrResourceNotFound,
		ItemID: resourceID,
	}
}

// LinkNotFoundError is a helper function that creates a new error
// for when a link can not be found in the persistence that backs
// a state container.
func LinkNotFoundError(linkID string) *Error {
	return &Error{
		Code:   ErrLinkNotFound,
		ItemID: linkID,
	}
}

// InstanceNotFoundError is a helper function that creates a new error
// for when a blueprint instance can not be found in the persistence that backs
// a state container.
func InstanceNotFoundError(instanceID string) *Error {
	return &Error{
		Code:   ErrInstanceNotFound,
		ItemID: instanceID,
	}
}

// ExportNotFoundError is a helper function that creates a new error
// for when an export can not be found for the given blueprint instance.
func ExportNotFoundError(instanceID string, exportName string) *Error {
	exportItemID := fmt.Sprintf("instance:%s:export:%s", instanceID, exportName)
	return &Error{
		Code:   ErrExportNotFound,
		ItemID: exportItemID,
	}
}

// IsResourceNotFound is a helper function that checks if the provided error
// is a resource not found error.
func IsResourceNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrResourceNotFound
	}
	return false
}

// IsLinkNotFound is a helper function that checks if the provided error
// is a link not found error.
func IsLinkNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrLinkNotFound
	}
	return false
}

// IsInstanceNotFound is a helper function that checks if the provided error
// is an instance not found error.
func IsInstanceNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInstanceNotFound
	}
	return false
}

// IsExportNotFound is a helper function that checks if the provided error
// is an export not found error.
func IsExportNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrExportNotFound
	}
	return false
}

// InstancesNotFoundError represents an error when multiple instances
// could not be found by their IDs or names.
type InstancesNotFoundError struct {
	MissingIDsOrNames []string
}

func (e *InstancesNotFoundError) Error() string {
	if len(e.MissingIDsOrNames) == 1 {
		return fmt.Sprintf("StateError: instance %q not found", e.MissingIDsOrNames[0])
	}
	return fmt.Sprintf("StateError: instances not found: %v", e.MissingIDsOrNames)
}

// NewInstancesNotFoundError creates a new error for when multiple
// instances could not be found.
func NewInstancesNotFoundError(missingIDsOrNames []string) *InstancesNotFoundError {
	return &InstancesNotFoundError{
		MissingIDsOrNames: missingIDsOrNames,
	}
}

// IsInstancesNotFound checks if the provided error is an instances not found error.
func IsInstancesNotFound(err error) bool {
	_, ok := err.(*InstancesNotFoundError)
	return ok
}
