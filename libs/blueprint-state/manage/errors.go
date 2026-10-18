package manage

import "fmt"

// EventNotFound is an error type
// that indicates an event with the specified ID was not found.
type EventNotFound struct {
	ID string
}

func (e EventNotFound) Error() string {
	return fmt.Sprintf("event with ID %s not found", e.ID)
}

// EventNotFoundError creates a new EventNotFound error with the specified ID.
// An error can be checked against this type using:
//
//	var eventNotFoundErr *manage.EventNotFound
//	if errors.As(err, &eventNotFoundErr) {
//		// Handle the error
//	}
func EventNotFoundError(id string) error {
	return &EventNotFound{ID: id}
}

// ChangesetNotFound is an error type
// that indicates a changeset with the specified ID was not found.
type ChangesetNotFound struct {
	ID string
}

func (c ChangesetNotFound) Error() string {
	return fmt.Sprintf("change set with ID %s not found", c.ID)
}

// ChangesetNotFoundError creates a new ChangesetNotFound error with the specified ID.
// An error can be checked against this type using:
//
//	var changesetNotFoundErr *manage.ChangesetNotFound
//	if errors.As(err, &changesetNotFoundErr) {
//		// Handle the error
//	}
func ChangesetNotFoundError(id string) error {
	return &ChangesetNotFound{ID: id}
}

// BlueprintValidationNotFound is an error type
// that indicates a blueprint validation request with
// the specified ID was not found.
type BlueprintValidationNotFound struct {
	ID string
}

func (b BlueprintValidationNotFound) Error() string {
	return fmt.Sprintf("blueprint validation request with ID %s not found", b.ID)
}

// BlueprintValidationNotFoundError creates a new BlueprintValidationNotFound
// error with the specified ID.
// An error can be checked against this type using:
//
//	var blueprintValidationNotFoundErr *manage.BlueprintValidationNotFound
//	if errors.As(err, &blueprintValidationNotFoundErr) {
//		// Handle the error
//	}
func BlueprintValidationNotFoundError(id string) error {
	return &BlueprintValidationNotFound{ID: id}
}

// AuditRecordNotFound is an error type
// that indicates an audit record with
// the specified ID was not found.
type AuditRecordNotFound struct {
	ID string
}

func (a AuditRecordNotFound) Error() string {
	return fmt.Sprintf("audit record with ID %s not found", a.ID)
}

// AuditRecordNotFoundError creates a new AuditRecordNotFound
// error with the specified ID.
// An error can be checked against this type using:
//
//	var auditRecordNotFoundErr *manage.AuditRecordNotFound
//	if errors.As(err, &auditRecordNotFoundErr) {
//		// Handle the error
//	}
func AuditRecordNotFoundError(id string) error {
	return &AuditRecordNotFound{ID: id}
}
