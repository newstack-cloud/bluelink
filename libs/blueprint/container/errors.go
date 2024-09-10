package container

import (
	"fmt"
	"strings"

	"github.com/two-hundred/celerity/libs/blueprint/errors"
	"github.com/two-hundred/celerity/libs/blueprint/schema"
	"github.com/two-hundred/celerity/libs/blueprint/source"
	"github.com/two-hundred/celerity/libs/common/core"
)

const (
	// ErrorReasonCodeInvalidSpecExtension is provided
	// when the reason for a blueprint spec load error
	// is due to an invalid specification file extension.
	ErrorReasonCodeInvalidSpecExtension errors.ErrorReasonCode = "invalid_spec_ext"
	// ErrorReasonCodeInvalidResourceType is provided
	// when the reason for a blueprint spec load error
	// is due to an invalid resource type provided in one
	// of the resources in the spec.
	ErrorReasonCodeInvalidResourceType errors.ErrorReasonCode = "invalid_resource_type"
	// ErrorReasonCodeMissingProvider is provided when the
	// reason for a blueprint spec load error is due to
	// a missing provider for one of the resources in
	// the spec.
	ErrorReasonCodeMissingProvider errors.ErrorReasonCode = "missing_provider"
	// ErrorReasonCodeMissingResource is provided when the
	// reason for a blueprint spec load error is due to
	// the resource provider missing an implementation for the
	// resource type for one of the resources in the spec.
	ErrorReasonCodeMissingResource errors.ErrorReasonCode = "missing_resource"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more resources in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeResourceValidationErrors errors.ErrorReasonCode = "resource_validation_errors"
	// ErrorReasonCodeDataSourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more data sources in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeDataSourceValidationErrors errors.ErrorReasonCode = "data_source_validation_errors"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more variables in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeVariableValidationErrors errors.ErrorReasonCode = "variable_validation_errors"
	// ErrorReasonCodeIncludeValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more includes in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeIncludeValidationErrors errors.ErrorReasonCode = "include_validation_errors"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more variables in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeExportValidationErrors errors.ErrorReasonCode = "export_validation_errors"
	// ErrorReasonMissingTransformers is provided when the
	// reason for a blueprint spec load error is due to a spec referencing
	// transformers that aren't supported by the blueprint loader
	// used to parse the schema.
	ErrorReasonMissingTransformers errors.ErrorReasonCode = "missing_transformers"
)

func errUnsupportedSpecFileExtension(filePath string) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSpecExtension,
		Err:        fmt.Errorf("unsupported spec file extension in %s, only json and yaml are supported", filePath),
	}
}

// func errInvalidResourceType(resourceType string) error {
// 	return &errors.LoadError{
// 		ReasonCode: ErrorReasonCodeInvalidResourceType,
// 		Err:        fmt.Errorf("resource type format is invalid for %s, resource type must be of the form {provider}/*", resourceType),
// 	}
// }

// func errMissingProvider(providerKey string, resourceType string) error {
// 	return &errors.LoadError{
// 		ReasonCode: ErrorReasonCodeMissingProvider,
// 		Err:        fmt.Errorf("missing provider %s for the resource type %s", providerKey, resourceType),
// 	}
// }

// func errMissingResource(providerKey string, resourceType string) error {
// 	return &errors.LoadError{
// 		ReasonCode: ErrorReasonCodeMissingResource,
// 		Err:        fmt.Errorf("missing resource in provider %s for the resource type %s", providerKey, resourceType),
// 	}
// }

// func errResourceValidationError(errorMap map[string]error) error {
// 	errCount := len(errorMap)
// 	return &errors.LoadError{
// 		ReasonCode:  ErrorReasonCodeResourceValidationErrors,
// 		Err:         fmt.Errorf("validation failed due to issues with %d resources in the spec", errCount),
// 		ChildErrors: core.MapToSlice(errorMap),
// 	}
// }

func errVariableValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeVariableValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d variables in the spec", errCount),
		ChildErrors: errs,
	}
}

func errResourceValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeResourceValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d resources in the spec", errCount),
		ChildErrors: errs,
	}
}

func errDataSourceValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeDataSourceValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d data sources in the spec", errCount),
		ChildErrors: errs,
	}
}

func errIncludeValidationError(errorMap map[string]error) error {
	errCount := len(errorMap)
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeIncludeValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d includes in the spec", errCount),
		ChildErrors: core.MapToSlice(errorMap),
	}
}

func errExportValidationError(errorMap map[string]error) error {
	errCount := len(errorMap)
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeExportValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d exports in the spec", errCount),
		ChildErrors: core.MapToSlice(errorMap),
	}
}

func errTransformersMissing(missingTransformers []string, childErrors []error, line *int, column *int) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonMissingTransformers,
		Err: fmt.Errorf(
			"the following transformers are missing in the blueprint loader: %s", strings.Join(missingTransformers, ", "),
		),
		ChildErrors: childErrors,
		Line:        line,
		Column:      column,
	}
}

func errTransformerMissing(transformer string, line *int, column *int) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonMissingTransformers,
		Err: fmt.Errorf(
			"the following transformer is missing from the blueprint loader: %s", transformer,
		),
		Line:   line,
		Column: column,
	}
}

func errMissingVariableType(variableName string, location *source.Meta) error {
	line, column := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"variable type missing for variable %s", variableName,
		),
		Line:   line,
		Column: column,
	}
}

func errInvalidCustomVariableType(variableName string, variableType schema.VariableType, line *int, column *int) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"invalid custom variable type %s for variable %s", variableType, variableName,
		),
		Line:   line,
		Column: column,
	}
}

func errMissingProviderForCustomVarType(
	providerKey string,
	variableName string,
	variableType schema.VariableType,
	line *int,
	column *int,
) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"missing provider %s for custom variable type %s in variable %s", providerKey, variableType, variableName,
		),
		Line:   line,
		Column: column,
	}
}

func flattenErrorMap(errorMap map[string][]error) []error {
	errs := []error{}
	for _, errSlice := range errorMap {
		errs = append(errs, errSlice...)
	}
	return errs
}
