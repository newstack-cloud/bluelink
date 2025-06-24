package serialisation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

const (
	// ErrorReasonCodeMissingScalar is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing scalar value.
	ErrorReasonCodeMissingScalar errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_scalar"
	// ErrorReasonCodeMissingStringOrSubValue is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing string or substitution value.
	ErrorReasonCodeMissingStringOrSubValue errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_string_or_sub_value"
	// ErrorReasonCodeMissingSubstitutionValue is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing substitution value.
	ErrorReasonCodeMissingSubstitutionValue errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_substitution_value"
	// ErrorReasonCodeMissingSubPathItemValue is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing substitution path item value. (e.g. [0], .key or ["key"])
	ErrorReasonCodeMissingSubPathItemValue errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_sub_path_item_value"
	// ErrorReasonCodeMissingMappingNodeValue is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing mapping node value.
	ErrorReasonCodeMissingMappingNodeValue errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_mapping_node_value"
	// ErrorReasonCodeMappingNodeIsNil is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a required mapping node being nil.
	ErrorReasonCodeMappingNodeIsNil errors.ErrorReasonCode = "expanded_blueprint_serialise_mapping_node_is_nil"
	// ErrorReasonCodeStringOrSubsIsNil is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// the string or substitutions being nil.
	ErrorReasonCodeStringOrSubsIsNil errors.ErrorReasonCode = "expanded_blueprint_serialise_string_or_subs_is_nil"
	// ErrorReasonCodeMissingSubFuncArgValue is provided when the reason
	// for an expanded blueprint spec serialisation error is due to
	// a missing substitution function argument value.
	ErrorReasonCodeMissingSubFuncArgValue errors.ErrorReasonCode = "expanded_blueprint_serialise_missing_sub_func_arg_value"
)

func errMissingScalarValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingScalar,
		Err:        fmt.Errorf("missing scalar value"),
	}
}

func errMissingStringOrSubstitutionValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingStringOrSubValue,
		Err:        fmt.Errorf("missing string or substitution value"),
	}
}

func errMissingSubstitutionValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingSubstitutionValue,
		Err:        fmt.Errorf("missing substitution value"),
	}
}

func errMissingSubstitutionPathItemValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingSubPathItemValue,
		Err:        fmt.Errorf("missing substitution path item value"),
	}
}

func errMissingMappingNodeValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingMappingNodeValue,
		Err:        fmt.Errorf("missing mapping node value"),
	}
}

func errMappingNodeIsNil() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMappingNodeIsNil,
		Err:        fmt.Errorf("required mapping node is set to nil"),
	}
}

func errStringOrSubstitutionsIsNil() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeStringOrSubsIsNil,
		Err:        fmt.Errorf("string or substitutions is nil"),
	}
}

func errMissingSubstitutionFunctionArgValue() error {
	return &errors.ExpandedSerialiseError{
		ReasonCode: ErrorReasonCodeMissingSubFuncArgValue,
		Err:        fmt.Errorf("missing substitution function argument value"),
	}
}
