package core

import (
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// Error represents an error due to an issue
// with a core data type (mapping or scalar) in a blueprint.
type Error struct {
	ReasonCode ErrorCoreReasonCode
	Err        error
	// The line in the source blueprint file
	// where the error occurred.
	// This will be nil if the error is not related
	// to a specific line in the blueprint file
	// or the source format is JSON.
	SourceLine *int
	// The column on a line in the source blueprint file
	// where the error occurred.
	// This will be nil if the error is not related
	// to a specific line/column in the blueprint file
	// or the source format is JSON.
	SourceColumn *int
}

func (e *Error) Error() string {
	return e.Err.Error()
}

type ErrorCoreReasonCode string

const (
	// ErrorCoreReasonCodeInvalidMappingNode is provided
	// when a mapping node can not be parsed from a serialised blueprint.
	ErrorCoreReasonCodeInvalidMappingNode ErrorCoreReasonCode = "invalid_mapping_node"
	// ErrorCoreReasonCodeMissingMappingNode is provided
	// when a mapping node is missing a value in a serialised blueprint.
	ErrorCoreReasonCodeMissingMappingNode ErrorCoreReasonCode = "missing_mapping_node"
	// ErrorCoreReasonCodeMustBeScalar is an error that is returned
	// when a value that is expected to be a blueprint scalar is not a scalar value.
	ErrorCoreReasonCodeMustBeScalar ErrorCoreReasonCode = "must_be_scalar"
)

func errInvalidMappingNode(posInfo source.PositionInfo) error {
	innerError := errors.New("a blueprint mapping node must be a valid scalar, mapping or sequence")
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorCoreReasonCodeInvalidMappingNode,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorCoreReasonCodeInvalidMappingNode,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}

func errMissingMappingNode(posInfo source.PositionInfo) error {
	innerError := errors.New("a blueprint mapping node must have a valid value set")
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorCoreReasonCodeMissingMappingNode,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorCoreReasonCodeMissingMappingNode,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}

func errMustBeScalar(posInfo source.PositionInfo) error {
	innerError := errors.New("a blueprint scalar value must be a scalar (string, int, bool or float)")
	return createScalarError(posInfo, innerError)
}

func errMustBeScalarWithParentPath(
	posInfo source.PositionInfo,
	parentPath string,
) error {
	innerError := fmt.Errorf(
		"blueprint scalar value in %q must be a scalar (string, int, bool or float)",
		parentPath,
	)
	return createScalarError(posInfo, innerError)
}

func createScalarError(
	posInfo source.PositionInfo,
	innerError error,
) error {
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorCoreReasonCodeMustBeScalar,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorCoreReasonCodeMustBeScalar,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}
