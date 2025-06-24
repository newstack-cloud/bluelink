package schema

import (
	"fmt"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

// Error represents an error due to an issue
// with the schema of a blueprint.
type Error struct {
	ReasonCode ErrorSchemaReasonCode
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

type ErrorSchemaReasonCode string

const (
	// ErrorSchemaReasonCodeInvalidDataSourceFieldType is provided
	// when the reason for a blueprint schema load error is due
	// to an invalid data source exported field type.
	ErrorSchemaReasonCodeInvalidDataSourceFieldType ErrorSchemaReasonCode = "invalid_data_source_field_type"
	// ErrorSchemaReasonCodeInvalidTransformType is provided
	// when the reason for a blueprint schema load error is due to
	// an invalid transform field value being provided.
	ErrorSchemaReasonCodeInvalidTransformType ErrorSchemaReasonCode = "invalid_transform_type"
	// ErrorSchemaReasonCodeInvalidDependencyType is provided
	// when the reason for a blueprint schema load error is due
	// to an invalid dependsOn field value being provided for a resource.
	ErrorSchemaReasonCodeInvalidDependencyType ErrorSchemaReasonCode = "invalid_dependency_type"
	// ErrorSchemaReasonCodeInvalidMap is provided when the reason
	// for a blueprint schema load error is due to an invalid map
	// being provided.
	ErrorSchemaReasonCodeInvalidMap ErrorSchemaReasonCode = "invalid_map"
	// ErrorSchemaReasonCodeInvalidMapKey is provided when the reason
	// for a blueprint schema load error is due to an invalid array (sequence)
	// being provided.
	ErrorSchemaReasonCodeInvalidArray ErrorSchemaReasonCode = "invalid_array"
	// ErrorSchemaReasonCodeInvalidArrayOrString is provided when the reason
	// for a blueprint schema load error is due to an invalid value being provided
	// for a field that can be either a string or an array of strings.
	ErrorSchemaReasonCodeInvalidArrayOrString ErrorSchemaReasonCode = "invalid_array_or_string"
	// ErrorSchemaReasonCodeInvalidResourceCondition is provided
	// when the reason for a blueprint schema load error is due
	// to an invalid resource condition being provided.
	ErrorSchemaReasonCodeInvalidResourceCondition ErrorSchemaReasonCode = "invalid_resource_condition"
	// ErrorSchemaReasonCodeGeneral is provided when the reason
	// for a blueprint schema load error is not specific,
	// primarily used for errors wrapped with parent scope line information.
	ErrorSchemaReasonCodeGeneral ErrorSchemaReasonCode = "general"
)

func errInvalidTransformType(underlyingError error, line *int, column *int) error {
	return &Error{
		ReasonCode: ErrorSchemaReasonCodeInvalidTransformType,
		Err: fmt.Errorf(
			"unsupported type provided for spec transform, must be string or a list of strings: %s",
			underlyingError.Error(),
		),
		SourceLine:   line,
		SourceColumn: column,
	}
}

func errInvalidDependencyType(underlyingError error, line *int, column *int) error {
	return &Error{
		ReasonCode: ErrorSchemaReasonCodeInvalidDependencyType,
		Err: fmt.Errorf(
			"unsupported type provided for resource dependency, must be string or a list of strings: %s",
			underlyingError.Error(),
		),
		SourceLine:   line,
		SourceColumn: column,
	}
}

func errInvalidMap(posInfo source.PositionInfo, field string) error {
	innerError := fmt.Errorf("an invalid value has been provided for %s, expected a mapping", field)
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorSchemaReasonCodeInvalidMap,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeInvalidMap,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}

func errInvalidArray(posInfo source.PositionInfo, field string) error {
	innerError := fmt.Errorf("an invalid value has been provided for %s, expected a sequence", field)
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorSchemaReasonCodeInvalidArray,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeInvalidArray,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}

func errInvalidArrayOrString(posInfo source.PositionInfo, field string) error {
	innerError := fmt.Errorf("an invalid value has been provided for %s, expected a sequence or string", field)
	if posInfo == nil {
		return &Error{
			ReasonCode: ErrorSchemaReasonCodeInvalidArrayOrString,
			Err:        innerError,
		}
	}

	line := posInfo.GetLine()
	col := posInfo.GetColumn()
	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeInvalidArrayOrString,
		Err:          innerError,
		SourceLine:   &line,
		SourceColumn: &col,
	}
}

func errInvalidGeneralMap(value *yaml.Node) error {
	innerError := fmt.Errorf("an invalid value has been provided, expected a mapping")
	if value == nil {
		return &Error{
			ReasonCode: ErrorSchemaReasonCodeInvalidMap,
			Err:        innerError,
		}
	}

	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeInvalidMap,
		Err:          innerError,
		SourceLine:   &value.Line,
		SourceColumn: &value.Column,
	}
}

func errInvalidResourceCondition(value *yaml.Node) error {
	innerError := fmt.Errorf(
		"an invalid resource condition has been provided, only one of \"and\", \"or\" or \"not\" can be set",
	)
	if value == nil {
		return &Error{
			ReasonCode: ErrorSchemaReasonCodeInvalidResourceCondition,
			Err:        innerError,
		}
	}

	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeInvalidResourceCondition,
		Err:          innerError,
		SourceLine:   &value.Line,
		SourceColumn: &value.Column,
	}
}

func wrapErrorWithLineInfo(underlyingError error, parent *yaml.Node) error {
	if _, isSchemaError := underlyingError.(*Error); isSchemaError {
		return underlyingError
	}

	if _, isCoreError := underlyingError.(*bpcore.Error); isCoreError {
		return underlyingError
	}

	if _, isLoadError := underlyingError.(*errors.LoadError); isLoadError {
		return underlyingError
	}

	return &Error{
		ReasonCode:   ErrorSchemaReasonCodeGeneral,
		Err:          underlyingError,
		SourceLine:   &parent.Line,
		SourceColumn: &parent.Column,
	}
}
