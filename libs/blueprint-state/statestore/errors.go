package statestore

import (
	"fmt"
)

// Error is the structured error type returned by statestore. Callers can
// distinguish cases by inspecting ReasonCode, typically via errors.As:
//
//	var sErr *statestore.Error
//	if errors.As(err, &sErr) && sErr.ReasonCode == statestore.ErrorReasonCodeMalformedStateFile {
//	    ...
//	}
//
// Error also implements Unwrap so errors.Is against wrapped sentinels works.
type Error struct {
	ReasonCode ErrorReasonCode
	Err        error
}

func (e *Error) Error() string {
	return fmt.Sprintf("statestore error (%s): %s", e.ReasonCode, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorReasonCode enumerates the distinct error conditions statestore can
// surface. Add new values when introducing a new failure mode rather than
// overloading an existing one.
type ErrorReasonCode string

const (
	// ErrorReasonCodeMalformedStateFile indicates a persisted chunk file or
	// index file is corrupted, truncated, or out of sync with the index —
	// e.g. the stored IndexInChunk points beyond the end of the chunk.
	ErrorReasonCodeMalformedStateFile ErrorReasonCode = "malformed_state_file"

	// ErrorReasonCodeMalformedState indicates the in-memory state is
	// inconsistent — for example, a resource's InstanceID references an
	// instance that no longer exists.
	ErrorReasonCodeMalformedState ErrorReasonCode = "malformed_state"

	// ErrorReasonCodeMaxEventPartitionSizeExceeded indicates a save-event
	// operation was refused because accepting the event would push the
	// partition's on-disk size past its configured maximum.
	ErrorReasonCodeMaxEventPartitionSizeExceeded ErrorReasonCode = "max_event_partition_size_exceeded"

	// ErrorReasonCodeUnknownLayout indicates a CategoryConfig carries a
	// Layout value the persister does not recognise. Likely a programmer
	// error at container-construction time.
	ErrorReasonCodeUnknownLayout ErrorReasonCode = "unknown_layout"

	// ErrorReasonCodeInstanceNameTaken indicates a CreateInstance call
	// tried to write a name-lookup record for a name already reserved by
	// another instance. Only surfaces when Config.WriteNameRecords is set
	// and a NameRecordReserver is installed (objectstore with conditional
	// writes). Single-process backends like memfile never produce this.
	ErrorReasonCodeInstanceNameTaken ErrorReasonCode = "instance_name_taken"
)

func errMalformedStateFile(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeMalformedStateFile,
		Err:        fmt.Errorf("malformed state file: %s", message),
	}
}

func errMalformedState(message string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeMalformedState,
		Err:        fmt.Errorf("malformed state: %s", message),
	}
}

func errMaxEventPartitionSizeExceeded(partition string, maxBytes int64) error {
	return &Error{
		ReasonCode: ErrorReasonCodeMaxEventPartitionSizeExceeded,
		Err: fmt.Errorf(
			"maximum event partition size (%d bytes) exceeded for %q",
			maxBytes, partition,
		),
	}
}

func errUnknownLayout(category string, layout Layout) error {
	return &Error{
		ReasonCode: ErrorReasonCodeUnknownLayout,
		Err:        fmt.Errorf("unknown layout %d for category %q", layout, category),
	}
}

// ErrInstanceNameTaken constructs the sentinel error backends return when
// an atomic name-stub reservation fails because another instance already
// claims the name.
func ErrInstanceNameTaken(name string) error {
	return &Error{
		ReasonCode: ErrorReasonCodeInstanceNameTaken,
		Err:        fmt.Errorf("instance name %q already taken", name),
	}
}
