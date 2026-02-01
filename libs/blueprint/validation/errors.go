package validation

import (
	"fmt"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// ErrorReasonCodeMissingType is provided when the reason
	// for a blueprint spec load error is due to the version property
	// not being provided for a blueprint.
	ErrorReasonCodeMissingVersion errors.ErrorReasonCode = "missing_version"
	// ErrorReasonCodeInvalidVersion is provided when the reason
	// for a blueprint spec load error is due to an invalid version
	// of the spec being provided.
	ErrorReasonCodeInvalidVersion errors.ErrorReasonCode = "invalid_version"
	// ErrorReasonCodeInvalidResource is provided when the reason
	// for a blueprint spec load error is due to one or more resources
	// being invalid.
	ErrorReasonCodeInvalidResource errors.ErrorReasonCode = "invalid_resource"
	// ErrorReasonCodeResourceDefItemEmpty is provided when the reason
	// for a blueprint spec load error is due to an empty resource definition item.
	ErrorReasonCodeResourceDefItemEmpty errors.ErrorReasonCode = "resource_def_item_empty"
	// ErrorReasonCodeResourceDefInvalidType is provided when the reason
	// for a blueprint spec load error is due to an invalid type in a resource definition.
	ErrorReasonCodeResourceDefInvalidType errors.ErrorReasonCode = "resource_def_invalid_type"
	// ErrorReasonCodeResourceDefMissingRequiredField is provided when the reason
	// for a blueprint spec load error is due to a missing required field in a resource definition.
	ErrorReasonCodeResourceDefMissingRequiredField errors.ErrorReasonCode = "resource_def_missing_required_field"
	// ErrorReasonCodeResourceDefUnknownField is provided when the reason
	// for a blueprint spec load error is due to an unknown field in a resource definition.
	ErrorReasonCodeResourceDefUnknownField errors.ErrorReasonCode = "resource_def_unknown_field"
	// ErrorReasonCodeResourceDefUnionItemEmpty is provided when the reason
	// for a blueprint spec load error is due to an empty union item in a resource definition.
	ErrorReasonCodeResourceDefUnionItemEmpty errors.ErrorReasonCode = "resource_def_union_item_empty"
	// ErrorReasonCodeResourceDefUnionInvalidType is provided when the reason
	// for a blueprint spec load error is due to an invalid type in a union item.
	ErrorReasonCodeResourceDefUnionInvalidType errors.ErrorReasonCode = "resource_def_union_invalid_type"
	// ErrorReasonCodeResourceDefNotAllowedValue is provided when the reason
	// for a blueprint spec load error is due to a value not being in the allowed values list.
	ErrorReasonCodeResourceDefNotAllowedValue errors.ErrorReasonCode = "resource_def_not_allowed_value"
	// ErrorReasonCodeResourceDefPatternConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a pattern constraint failure.
	ErrorReasonCodeResourceDefPatternConstraintFailure errors.ErrorReasonCode = "resource_def_pattern_constraint_failure"
	// ErrorReasonCodeResourceDefMinConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a minimum value constraint failure.
	ErrorReasonCodeResourceDefMinConstraintFailure errors.ErrorReasonCode = "resource_def_min_constraint_failure"
	// ErrorReasonCodeResourceDefMaxConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a maximum value constraint failure.
	ErrorReasonCodeResourceDefMaxConstraintFailure errors.ErrorReasonCode = "resource_def_max_constraint_failure"
	// ErrorReasonCodeResourceDefComplexMinLengthConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a minimum length constraint failure for complex types.
	ErrorReasonCodeResourceDefComplexMinLengthConstraintFailure errors.ErrorReasonCode = "resource_def_complex_min_length_constraint_failure"
	// ErrorReasonCodeResourceDefComplexMaxLengthConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a maximum length constraint failure for complex types.
	ErrorReasonCodeResourceDefComplexMaxLengthConstraintFailure errors.ErrorReasonCode = "resource_def_complex_max_length_constraint_failure"
	// ErrorReasonCodeResourceDefStringMinLengthConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a minimum length constraint failure for strings.
	ErrorReasonCodeResourceDefStringMinLengthConstraintFailure errors.ErrorReasonCode = "resource_def_string_min_length_constraint_failure"
	// ErrorReasonCodeResourceDefStringMaxLengthConstraintFailure is provided when the reason
	// for a blueprint spec load error is due to a maximum length constraint failure for strings.
	ErrorReasonCodeResourceDefStringMaxLengthConstraintFailure errors.ErrorReasonCode = "resource_def_string_max_length_constraint_failure"
	// ErrorReasonCodeResourceTypeSpecDefMissing is provided when the reason
	// for a blueprint spec load error is due to a missing spec definition for a resource.
	ErrorReasonCodeResourceTypeSpecDefMissing errors.ErrorReasonCode = "resource_type_spec_def_missing"
	// ErrorReasonCodeResourceTypeSpecDefMissingSchema is provided when the reason
	// for a blueprint spec load error is due to a missing spec definition schema for a resource.
	ErrorReasonCodeResourceTypeSpecDefMissingSchema errors.ErrorReasonCode = "resource_type_spec_def_missing_schema"
	// ErrorReasonCodeDataSourceSpecDefMissing is provided when the reason
	// for a blueprint spec load error is due to a missing spec definition for a data source.
	ErrorReasonCodeDataSourceSpecDefMissing errors.ErrorReasonCode = "data_source_spec_def_missing"
	// ErrorReasonCodeMissingResourcesOrIncludes is provided when the reason
	// for a blueprint spec load error is due no resources or includes
	// being defined in the blueprint.
	// An empty map or omitted property will result in this error.
	ErrorReasonCodeMissingResourcesOrIncludes errors.ErrorReasonCode = "missing_resources"
	// ErrorReasonCodeInvalidVariable is provided when the reason
	// for a blueprint spec load error is due to one or more variables
	// being invalid.
	// This could be due to a mismatch between the type and the value,
	// a missing required variable (one without a default value),
	// an invalid default value, invalid allowed values or an incorrect variable type.
	ErrorReasonCodeInvalidVariable errors.ErrorReasonCode = "invalid_variable"
	// ErrorReasonCodeInvalidValue is provided when the reason
	// for a blueprint spec load error is due to an invalid value
	// being provided.
	ErrorReasonCodeInvalidValue errors.ErrorReasonCode = "invalid_value"
	// ErrorReasonCodeInvalidValueType is provided
	// when the reason for a blueprint spec load error is due
	// to an invalid value type.
	ErrorReasonCodeInvalidValueType errors.ErrorReasonCode = "invalid_value_type"
	// ErrorReasonCodeInvalidExport is provided when the reason
	// for a blueprint spec load error is due to one or more exports
	// being invalid.
	ErrorReasonCodeInvalidExport errors.ErrorReasonCode = "invalid_export"
	// ErrorReasonCodeInvalidReference is provided when the reason
	// for a blueprint spec load error is due to one or more references
	// being invalid.
	ErrorReasonCodeInvalidReference errors.ErrorReasonCode = "invalid_reference"
	// ErrorReasonCodeInvalidSubstitution is provided when the reason
	// for a blueprint spec load error is due to one or more substitutions
	// being invalid.
	ErrorReasonCodeInvalidSubstitution errors.ErrorReasonCode = "invalid_substitution"
	// ErrorReasonCodeInvalidInclude is provided when the reason
	// for a blueprint spec load error is due to one or more includes
	// being invalid.
	ErrorReasonCodeInvalidInclude errors.ErrorReasonCode = "invalid_include"
	// ErrorReasonCodeInvalidResource is provided when the reason
	// for a blueprint spec load error is due to one or more data sources
	// being invalid.
	ErrorReasonCodeInvalidDataSource errors.ErrorReasonCode = "invalid_data_source"
	// ErrorReasonCodeInvalidDataSourceFilterOperator is provided
	// when the reason for a blueprint spec load error is due
	// to an invalid data source filter operator being provided.
	ErrorReasonCodeInvalidDataSourceFilterOperator errors.ErrorReasonCode = "invalid_data_source_filter_operator"
	// ErrorReasonCodeUnsupportedDataSourceFilterOperator is provided
	// when the reason for a blueprint spec load error is due
	// to an unsupported data source filter operator being provided.
	ErrorReasonCodeUnsupportedDataSourceFilterOperator errors.ErrorReasonCode = "unsupported_data_source_filter_operator"
	// ErrorReasonCodeInvalidDataSourceFieldType is provided
	// when the reason for a blueprint spec load error is due
	// to an invalid data source field type.
	ErrorReasonCodeInvalidDataSourceFieldType errors.ErrorReasonCode = "invalid_data_source_field_type"
	// ErrorReasonCodeInvalidDataSourceFilterConflict is provided
	// when the reason for a blueprint spec load error is due
	// to a conflict between two filter fields in a data source,
	// where both fields can not be used to filter the same data source.
	ErrorReasonCodeDataSourceFilterConflict errors.ErrorReasonCode = "data_source_filter_conflict"
	// ErrorReasonCodeDataSourceFilterFieldNotSupported is provided
	// when the reason for a blueprint spec load error is due
	// to a data source having a field set as a filter that can't be used for filtering.
	ErrorReasonCodeDataSourceFilterFieldNotSupported errors.ErrorReasonCode = "data_source_filter_field_not_supported"
	// ErrorReasonCodeDataSourceMissingType is provided
	// when the reason for a blueprint spec load error is due
	// to a missing type for a data source.
	ErrorReasonCodeDataSourceMissingType errors.ErrorReasonCode = "data_source_missing_type"
	// ErrorReasonCodeInvalidMapKey is provided when the reason
	// for a blueprint spec load error is due to an invalid map key.
	ErrorReasonCodeInvalidMapKey errors.ErrorReasonCode = "invalid_map_key"
	// ErrorReasonCodeMultipleValidationErrors is provided when the reason
	// for a blueprint spec load error is due to multiple validation errors.
	ErrorReasonCodeMultipleValidationErrors errors.ErrorReasonCode = "multiple_validation_errors"
	// ErrorReasonCodeReferenceCycle is provided when the reason
	// for a blueprint spec load error is due to a reference cycle being detected.
	// This error code is used to collect and surface reference cycle errors
	// for pure substitution reference cycles and link <-> substitution reference cycles.
	ErrorReasonCodeReferenceCycle errors.ErrorReasonCode = "reference_cycle"
	// ErrorReasonCodeInvalidMappingNode is provided when the reason
	// for a blueprint spec load error is due to an invalid mapping node.
	ErrorReasonCodeInvalidMappingNode errors.ErrorReasonCode = "invalid_mapping_node"
	// ErrorReasonCodeInvalidResourceDependency is provided when the reason
	// for a blueprint spec load error is due to a resource dependency in the "dependsOn"
	// property not being a valid resource.
	ErrorReasonCodeMissingResourceDependency errors.ErrorReasonCode = "missing_resource_dependency"
	// ErrorReasonCodeComputedFieldInBlueprint is provided when the reason
	// for a blueprint spec load error is due to a computed field being used in a blueprint.
	// Computed fields are not allowed to be defined in blueprints,
	// they are computed by providers when a resource has been created.
	ErrorReasonCodeComputedFieldInBlueprint errors.ErrorReasonCode = "computed_field_in_blueprint"
	// ErrorReasonCodeEachResourceDependency is provided when the reason
	// for a blueprint spec load error is due to the "each" property of a resource
	// having a dependency on another resource.
	ErrorReasonCodeEachResourceDependency errors.ErrorReasonCode = "each_resource_dependency"
	// ErrorReasonCodeEachChildDependency is provided when the reason
	// for a blueprint spec load error is due to the "each" property of a resource
	// having a dependency on a child blueprint.
	ErrorReasonCodeEachChildDependency errors.ErrorReasonCode = "each_child_dependency"
	// ErrorReasonCodeSubFuncLinkArgResourceNotFound is provided when the reason
	// for a blueprint spec load error is due to a resource not being found
	// in an argument to the "link" substitution function.
	ErrorReasonCodeSubFuncLinkArgResourceNotFound errors.ErrorReasonCode = "sub_func_link_arg_resource_not_found"
	// ErrorReasonCodeVariableEmptyDefaultValue is provided when the reason
	// for a blueprint spec load error is due to an empty default value for a variable.
	ErrorReasonCodeVariableEmptyDefaultValue errors.ErrorReasonCode = "variable_empty_default_value"
	// ErrorReasonCodeVariableInvalidOrMissing is provided when the reason
	// for a blueprint spec load error is due to an invalid or missing variable value.
	ErrorReasonCodeVariableInvalidOrMissing errors.ErrorReasonCode = "variable_invalid_or_missing"
	// ErrorReasonCodeVariableEmptyValue is provided when the reason
	// for a blueprint spec load error is due to an empty variable value.
	ErrorReasonCodeVariableEmptyValue errors.ErrorReasonCode = "variable_empty_value"
	// ErrorReasonCodeVariableInvalidAllowedValue is provided when the reason
	// for a blueprint spec load error is due to an invalid allowed value for a variable.
	ErrorReasonCodeVariableInvalidAllowedValue errors.ErrorReasonCode = "variable_invalid_allowed_value"
	// ErrorReasonCodeVariableNullAllowedValue is provided when the reason
	// for a blueprint spec load error is due to a null allowed value for a variable.
	ErrorReasonCodeVariableNullAllowedValue errors.ErrorReasonCode = "variable_null_allowed_value"
	// ErrorReasonCodeVariableInvalidAllowedValues is provided when the reason
	// for a blueprint spec load error is due to invalid allowed values for a variable.
	ErrorReasonCodeVariableInvalidAllowedValues errors.ErrorReasonCode = "variable_invalid_allowed_values"
	// ErrorReasonCodeVariableInvalidAllowedValuesNotSupported is provided when the reason
	// for a blueprint spec load error is due to allowed values not being supported for a variable type.
	ErrorReasonCodeVariableInvalidAllowedValuesNotSupported errors.ErrorReasonCode = "variable_invalid_allowed_values_not_supported"
	// ErrorReasonCodeVariableValueNotAllowed is provided when the reason
	// for a blueprint spec load error is due to a variable value not being in the allowed values.
	ErrorReasonCodeVariableValueNotAllowed errors.ErrorReasonCode = "variable_value_not_allowed"
	// ErrorReasonCodeVariableInvalidSecretValue is provided when the reason
	// for a blueprint spec load error is due to an invalid secret field value for a variable.
	ErrorReasonCodeVariableInvalidSecretValue errors.ErrorReasonCode = "variable_invalid_secret_value"
	// ErrorReasonCodeRequiredVariableMissing is provided when the reason
	// for a blueprint spec load error is due to a required variable being missing.
	ErrorReasonCodeRequiredVariableMissing errors.ErrorReasonCode = "required_variable_missing"
	// ErrorReasonCodeCustomVarValueNotInOptions is provided when the reason
	// for a blueprint spec load error is due to a custom variable value not being in the available options.
	ErrorReasonCodeCustomVarValueNotInOptions errors.ErrorReasonCode = "custom_variable_value_not_in_options"
	// ErrorReasonCodeMixedVariableTypes is provided when the reason
	// for a blueprint spec load error is due to mixed variable types
	// used in the options for a custom variable type.
	ErrorReasonCodeMixedVariableTypes errors.ErrorReasonCode = "mixed_variable_types"
	// ErrorReasonCodeCustomVarAllowedValuesNotInOptions is provided when the reason
	// for a blueprint spec load error is due to allowed values not being in the available options
	// for a custom variable type.
	ErrorReasonCodeCustomVarAllowedValuesNotInOptions errors.ErrorReasonCode = "custom_variable_allowed_values_not_in_options"
	// ErrorReasonCodeCustomVarDefaultValueNotInOptions is provided when the reason
	// for a blueprint spec load error is due to a default value not being in the available options
	// for a custom variable type.
	ErrorReasonCodeCustomVarDefaultValueNotInOptions errors.ErrorReasonCode = "custom_variable_default_value_not_in_options"
	// ErrorReasonCodeInvalidExportType is provided when the reason
	// for a blueprint spec load error is due to an invalid export type.
	ErrorReasonCodeInvalidExportType errors.ErrorReasonCode = "invalid_export_type"
	// ErrorReasonCodeMissingExportType is provided when the reason
	// for a blueprint spec load error is due to a missing export type.
	ErrorReasonCodeMissingExportType errors.ErrorReasonCode = "missing_export_type"
	// ErrorReasonCodeEmptyExportField is provided when the reason
	// for a blueprint spec load error is due to an empty export field.
	ErrorReasonCodeEmptyExportField errors.ErrorReasonCode = "empty_export_field"
	// ErrorReasonCodeInvalidReferencePattern is provided when the reason
	// for a blueprint spec load error is due to an invalid reference pattern.
	ErrorReasonCodeInvalidReferencePattern errors.ErrorReasonCode = "invalid_reference_pattern"
	// ErrorReasonCodeReferenceContextAccess is provided when the reason
	// for a blueprint spec load error is due to invalid reference context access.
	ErrorReasonCodeReferenceContextAccess errors.ErrorReasonCode = "reference_context_access"
	// ErrorReasonCodeIncludeEmptyPath is provided when the reason
	// for a blueprint spec load error is due to an empty include path.
	ErrorReasonCodeIncludeEmptyPath errors.ErrorReasonCode = "include_empty_path"
	// ErrorReasonCodeDataSourceMissingFilter is provided when the reason
	// for a blueprint spec load error is due to a missing data source filter.
	ErrorReasonCodeDataSourceMissingFilter errors.ErrorReasonCode = "data_source_missing_filter"
	// ErrorReasonCodeDataSourceEmptyFilter is provided when the reason
	// for a blueprint spec load error is due to an empty data source filter.
	ErrorReasonCodeDataSourceEmptyFilter errors.ErrorReasonCode = "data_source_empty_filter"
	// ErrorReasonCodeDataSourceMissingFilterField is provided when the reason
	// for a blueprint spec load error is due to a missing data source filter field.
	ErrorReasonCodeDataSourceMissingFilterField errors.ErrorReasonCode = "data_source_missing_filter_field"
	// ErrorReasonCodeDataSourceMissingFilterSearch is provided when the reason
	// for a blueprint spec load error is due to a missing data source filter search.
	ErrorReasonCodeDataSourceMissingFilterSearch errors.ErrorReasonCode = "data_source_missing_filter_search"
	// ErrorReasonCodeDataSourceMissingExports is provided when the reason
	// for a blueprint spec load error is due to missing data source exports.
	ErrorReasonCodeDataSourceMissingExports errors.ErrorReasonCode = "data_source_missing_exports"
	// ErrorReasonCodeDataSourceFilterFieldConflict is provided when the reason
	// for a blueprint spec load error is due to a data source filter field conflict.
	ErrorReasonCodeDataSourceFilterFieldConflict errors.ErrorReasonCode = "data_source_filter_field_conflict"
	// ErrorReasonCodeDataSourceFilterOperatorNotSupported is provided when the reason
	// for a blueprint spec load error is due to an unsupported data source filter operator.
	ErrorReasonCodeDataSourceFilterOperatorNotSupported errors.ErrorReasonCode = "data_source_filter_operator_not_supported"
	// ErrorReasonCodeDataSourceMissingFilterOperator is provided when the reason
	// for a blueprint spec load error is due to a missing data source filter operator.
	ErrorReasonCodeDataSourceMissingFilterOperator errors.ErrorReasonCode = "data_source_missing_filter_operator"
	// ErrorReasonCodeResourceSpecPreValidationFailed is provided when the reason
	// for a blueprint spec load error is due to resource spec pre-validation failure.
	ErrorReasonCodeResourceSpecPreValidationFailed errors.ErrorReasonCode = "resource_spec_pre_validation_failed"
	// ErrorReasonCodeMappingNodeKeyContainsSubstitution is provided when the reason
	// for a blueprint spec load error is due to a mapping node key containing substitution.
	ErrorReasonCodeMappingNodeKeyContainsSubstitution errors.ErrorReasonCode = "mapping_node_key_contains_substitution"
	// ErrorReasonCodeVariableInvalidDefaultValue is provided when the reason
	// for a blueprint spec load error is due to an invalid default value for a variable.
	ErrorReasonCodeVariableInvalidDefaultValue errors.ErrorReasonCode = "variable_invalid_default_value"
	// ErrorReasonCodeIncludePathNotFound is provided when the reason
	// for a blueprint spec load error is due to a child blueprint include path
	// pointing to a file that does not exist on the local filesystem.
	ErrorReasonCodeIncludePathNotFound errors.ErrorReasonCode = "include_path_not_found"
	// ErrorReasonCodeIncludeMissingRequiredVar is provided when a required variable
	// is not provided to a child blueprint include.
	ErrorReasonCodeIncludeMissingRequiredVar errors.ErrorReasonCode = "include_missing_required_variable"
	// ErrorReasonCodeIncludeVarTypeMismatch is provided when a variable provided
	// to a child blueprint include has a different type than expected.
	ErrorReasonCodeIncludeVarTypeMismatch errors.ErrorReasonCode = "include_variable_type_mismatch"
	// ErrorReasonCodeChildExportNotFound is provided when a substitution or export field
	// references a child blueprint export that does not exist in the resolved child blueprint.
	ErrorReasonCodeChildExportNotFound errors.ErrorReasonCode = "child_export_not_found"
	// ErrorReasonCodeChildExportScalarNavigation is provided when a substitution or export field
	// attempts to navigate into a child blueprint export that has a scalar type.
	ErrorReasonCodeChildExportScalarNavigation errors.ErrorReasonCode = "child_export_scalar_navigation"
	// ErrorReasonCodeSubFuncPathIndexOnNonArray is provided when a substitution
	// applies an array index to a function that returns a scalar type.
	ErrorReasonCodeSubFuncPathIndexOnNonArray errors.ErrorReasonCode = "sub_func_path_index_on_non_array"
	// ErrorReasonCodeSubFuncPathFieldOnNonObject is provided when a substitution
	// applies a field accessor to a function that returns a scalar type.
	ErrorReasonCodeSubFuncPathFieldOnNonObject errors.ErrorReasonCode = "sub_func_path_field_on_non_object"
)

func errBlueprintMissingVersion() error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeMissingVersion,
		Err:        fmt.Errorf("validation failed due to a version not being provided, version is a required property"),
	}
}

func errBlueprintMissingResourcesOrIncludes() error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeMissingResourcesOrIncludes,
		Err: fmt.Errorf(
			"validation failed as no resources or includes have been defined," +
				" at least one resource must be defined in a blueprint if there are no includes and" +
				" at least one include must be defined in a blueprint if there are no resources",
		),
	}
}

func errBlueprintUnsupportedVersion(version string) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidVersion,
		Err: fmt.Errorf(
			"validation failed due to an unsupported version \"%s\" being provided. "+
				"supported versions include: %s",
			version,
			strings.Join(SupportedVersions, ", "),
		),
	}
}

func errMappingNameContainsSubstitution(
	mappingName string,
	mappingType string,
	reasonCode errors.ErrorReasonCode,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: reasonCode,
		Err: fmt.Errorf(
			"${..} substitutions can not be used in %s names, found in %s \"%s\"",
			mappingType,
			mappingType,
			mappingName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errVariableInvalidDefaultValue(
	varType schema.VariableType,
	varName string,
	defaultValue *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	defaultVarType := deriveVarType(defaultValue)

	posRange := positionFromScalarValue(defaultValue, varSourceMeta)
	return &errors.LoadError{
		ReasonCode:     ErrorReasonCodeVariableInvalidDefaultValue,
		Err:            fmt.Errorf("variable %q: expected %s, got %s", varName, varType, defaultVarType),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidDefaultValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Variable Type",
					Description: fmt.Sprintf("Update the variable type or default value for %s", varName),
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"expectedType": string(varType),
				"actualType":   string(defaultVarType),
			},
		},
	}
}

func errVariableEmptyDefaultValue(varType schema.VariableType, varName string, varSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableEmptyDefaultValue,
		Err: fmt.Errorf(
			"validation failed due to an empty default %s value for variable \"%s\", you must provide a value when declaring a default in a blueprint",
			varType,
			varName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableEmptyDefaultValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Variable Default Value",
					Description: "Provide a valid default value for the variable or remove the default declaration.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
	}
}

func errVariableInvalidOrMissing(
	varType schema.VariableType,
	varName string,
	value *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	actualVarType := deriveOptionalVarType(value)
	if actualVarType == nil {
		posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
		return &errors.LoadError{
			ReasonCode: ErrorReasonCodeVariableInvalidOrMissing,
			Err: fmt.Errorf(
				"validation failed to a missing value for variable \"%s\", a value of type %s must be provided",
				varName,
				varType,
			),
			Line:           posRange.Line,
			EndLine:        posRange.EndLine,
			Column:         posRange.Column,
			EndColumn:      posRange.EndColumn,
			ColumnAccuracy: posRange.ColumnAccuracy,
			Context: &errors.ErrorContext{
				Category:   errors.ErrorCategoryVariableType,
				ReasonCode: ErrorReasonCodeVariableInvalidOrMissing,
				SuggestedActions: []errors.SuggestedAction{
					{
						Type:        string(errors.ActionTypeFixVariableType),
						Title:       "Provide Variable Value",
						Description: "Provide a valid value for the variable with the correct type.",
						Priority:    1,
					},
				},
				Metadata: map[string]any{
					"variableName": varName,
					"variableType": varType,
				},
			},
		}
	}

	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidOrMissing,
		Err: fmt.Errorf(
			"validation failed due to an incorrect type used for variable \"%s\", "+
				"expected a value of type %s but one of type %s was provided",
			varName,
			varType,
			*actualVarType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidOrMissing,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Variable Type",
					Description: "Provide a value with the correct type for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
				"actualType":   *actualVarType,
			},
		},
	}
}

func errVariableEmptyValue(
	varType schema.VariableType,
	varName string,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableEmptyValue,
		Err: fmt.Errorf(
			"validation failed due to an empty value being provided for variable \"%s\", "+
				"please provide a valid %s value that is not empty",
			varName,
			varType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableEmptyValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Provide Non-Empty Variable Value",
					Description: "Provide a valid non-empty value for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
	}
}

func errVariableInvalidSecretValue(
	varName string,
	secretValue *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	actualType := deriveVarType(secretValue)
	posRange := positionFromScalarValue(secretValue, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidSecretValue,
		Err: fmt.Errorf(
			"validation failed due to an invalid secret field value for variable %q, "+
				"expected a boolean but got %s",
			varName,
			actualType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidSecretValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Secret Field Value",
					Description: "The secret field must be a boolean value (true or false).",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"actualType":   string(actualType),
			},
		},
	}
}

func errVariableInvalidAllowedValue(
	varType schema.VariableType,
	allowedValue *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	allowedValueVarType := deriveVarType(allowedValue)
	scalarValueStr := deriveScalarValueAsString(allowedValue)

	posRange := positionFromScalarValue(allowedValue, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidAllowedValue,
		Err: fmt.Errorf(
			"an invalid allowed value was provided, %s with the value \"%s\" was provided when only %ss are allowed",
			varTypeToUnit(allowedValueVarType),
			scalarValueStr,
			varType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidAllowedValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Allowed Value Type",
					Description: "Provide an allowed value with the correct type for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableType":     varType,
				"allowedValueType": allowedValueVarType,
				"allowedValue":     scalarValueStr,
			},
		},
	}
}

func errVariableNullAllowedValue(
	varType schema.VariableType,
	allowedValue *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	posRange := positionFromScalarValue(allowedValue, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableNullAllowedValue,
		Err: fmt.Errorf(
			"null was provided for an allowed value, a valid %s must be provided",
			varType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableNullAllowedValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Allowed Value",
					Description: "Provide a valid non-null allowed value for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableType": varType,
			},
		},
	}
}

func errVariableInvalidAllowedValues(
	varName string,
	allowedValueErrors []error,
) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidAllowedValues,
		Err: fmt.Errorf(
			"validation failed due to one or more invalid allowed values being provided for variable \"%s\"",
			varName,
		),
		ChildErrors: allowedValueErrors,
	}
}

func errVariableInvalidAllowedValuesNotSupported(
	varType schema.VariableType,
	varName string,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidAllowedValuesNotSupported,
		Err: fmt.Errorf(
			"validation failed due to an allowed values list being provided for %s variable \"%s\","+
				" %s variables do not support allowed values enumeration",
			varType,
			varName,
			varType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidAllowedValuesNotSupported,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Remove Allowed Values",
					Description: "Remove the allowed values list as this variable type does not support enumeration.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
	}
}

func errVariableValueNotAllowed(
	varType schema.VariableType,
	varName string,
	value *bpcore.ScalarValue,
	allowedValues []*bpcore.ScalarValue,
	varSourceMeta *source.Meta,
	usingDefault bool,
) error {
	valueLabel := deriveValueLabel(usingDefault)
	posRange := positionFromScalarValue(value, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValueNotAllowed,
		Err: fmt.Errorf(
			"validation failed due to an invalid %s being provided for %s variable \"%s\","+
				" only the following values are supported: %s",
			valueLabel,
			varType,
			varName,
			scalarListToString(allowedValues),
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableValueNotAllowed,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Use Allowed Value",
					Description: "Provide a value from the allowed values list for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName":      varName,
				"variableType":      varType,
				"valueLabel":        valueLabel,
				"allowedValuesText": scalarListToString(allowedValues),
			},
		},
	}
}

func errCustomVariableValueNotInOptions(
	varType schema.VariableType,
	varName string,
	value *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
	usingDefault bool,
) error {
	valueLabel := deriveValueLabel(usingDefault)
	posRange := positionFromScalarValue(value, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeCustomVarValueNotInOptions,
		Err: fmt.Errorf(
			"validation failed due to an invalid %s \"%s\" being provided for variable \"%s\","+
				" which is not a valid %s option, see the custom type documentation for more details",
			valueLabel,
			deriveScalarValueAsString(value),
			varName,
			varType,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeCustomVarValueNotInOptions,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckCustomVariableOptions),
					Title:       "Check Custom Variable Options",
					Description: "Check the options for the custom variable type.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errRequiredVariableMissing(varName string, varSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode:     ErrorReasonCodeRequiredVariableMissing,
		Err:            fmt.Errorf("required variable %q has no value", varName),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeRequiredVariableMissing,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeProvideValue),
					Title:       "Provide Value",
					Description: fmt.Sprintf("Provide a value for the required variable %s", varName),
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeAddDefaultValue),
					Title:       "Add Default Value",
					Description: fmt.Sprintf("Add a default value to the variable definition for %s", varName),
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
			},
		},
	}
}

func errCustomVariableOptions(
	varName string,
	varSchema *schema.Variable,
	varSourceMeta *source.Meta,
	err error,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidVariable,
		Err: fmt.Errorf(
			"validation failed due to an error when loading options for variable \"%s\" of custom type \"%s\"",
			varName,
			varSchema.Type.Value,
		),
		ChildErrors:    []error{err},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errCustomVariableMixedTypes(
	varName string,
	varSchema *schema.Variable,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeMixedVariableTypes,
		Err: fmt.Errorf(
			"validation failed due to mixed types provided as options for variable type \"%s\" used in variable \"%s\", "+
				"all options must be of the same scalar type",
			varSchema.Type.Value,
			varName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeMixedVariableTypes,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeContactVariableTypeDeveloper),
					Title:       "Contact Variable Type Developer",
					Description: "Contact the developer of the variable type to fix the issue.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varSchema.Type.Value,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errCustomVariableInvalidDefaultValueType(
	varType schema.VariableType,
	varName string,
	defaultValue *bpcore.ScalarValue,
	varSourceMeta *source.Meta,
) error {
	defaultVarType := deriveVarType(defaultValue)
	posRange := positionFromScalarValue(defaultValue, varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableInvalidDefaultValue,
		Err: fmt.Errorf(
			"validation failed due to an invalid type for a default value for variable \"%s\", %s was provided "+
				"when a custom variable type option of %s was expected",
			varName,
			defaultVarType,
			varType,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableInvalidDefaultValue,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Fix Variable Default Value",
					Description: "Provide a valid default value for the variable.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"expectedType": varType,
				"actualType":   defaultVarType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errCustomVariableAllowedValuesNotInOptions(
	varType schema.VariableType,
	varName string,
	invalidOptions []string,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeCustomVarAllowedValuesNotInOptions,
		Err: fmt.Errorf(
			"validation failed due to invalid allowed values being provided for variable \"%s\" "+
				"of custom type \"%s\". See custom type documentation for possible values. Invalid values provided: %s",
			varName,
			varType,
			strings.Join(invalidOptions, ", "),
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeCustomVarAllowedValuesNotInOptions,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckCustomVariableOptions),
					Title:       "Check Custom Variable Options",
					Description: "Check the options for the custom variable type.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errCustomVariableDefaultValueNotInOptions(
	varType schema.VariableType,
	varName string,
	defaultValue string,
	varSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeCustomVarDefaultValueNotInOptions,
		Err: fmt.Errorf(
			"validation failed due to an invalid default value for variable \"%s\" "+
				"of custom type \"%s\". See custom type documentation for possible values. Invalid default value provided: %s",
			varName,
			varType,
			defaultValue,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeCustomVarDefaultValueNotInOptions,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckCustomVariableOptions),
					Title:       "Check Custom Variable Options",
					Description: "Check the options for the custom variable type.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName": varName,
				"variableType": varType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errMissingExportType(exportName string, exportSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(exportSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeMissingExportType,
		Err: fmt.Errorf(
			"validation failed due to a missing export type for export \"%s\"",
			exportName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryExport,
			ReasonCode: ErrorReasonCodeMissingExportType,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Add Export Type",
					Description: "Add a valid export type to the export definition.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"exportName": exportName,
			},
		},
	}
}

func errInvalidExportType(exportType schema.ExportType, exportName string, exportSourceMeta *source.Meta) error {
	validExportTypes := strings.Join(
		core.Map(
			schema.ExportTypes,
			func(exportType schema.ExportType, index int) string {
				return string(exportType)
			},
		),
		", ",
	)
	posRange := source.PositionRangeFromSourceMeta(exportSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidExportType,
		Err: fmt.Errorf(
			"validation failed due to an invalid export type of \"%s\" being provided for export \"%s\". "+
				"The following export types are supported: %s",
			exportType,
			exportName,
			validExportTypes,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryExport,
			ReasonCode: ErrorReasonCodeInvalidExportType,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Check Supported Export Types",
					Description: "Use a valid export type from the supported list.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"exportName":       exportName,
				"exportType":       exportType,
				"validExportTypes": validExportTypes,
			},
		},
	}
}

func errEmptyExportField(exportName string, exportSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(exportSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeEmptyExportField,
		Err: fmt.Errorf(
			"validation failed due to an empty field string being provided for export \"%s\"",
			exportName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryExport,
			ReasonCode: ErrorReasonCodeEmptyExportField,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeFixVariableType),
					Title:       "Provide Export Field",
					Description: "Provide a non-empty field value for the export.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"exportName": exportName,
			},
		},
	}
}

func errReferenceContextAccess(reference string, context string, referenceableType Referenceable, location *source.Meta) error {
	referencedObjectLabel := referenceableLabel(referenceableType)
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidReference,
		Err: fmt.Errorf(
			"validation failed due to a reference to a %s (\"%s\") being made from \"%s\", "+
				"which can not access values from a %s",
			referencedObjectLabel,
			reference,
			context,
			referencedObjectLabel,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidReferencePattern(
	reference string,
	context string,
	referenceableType Referenceable,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidReference,
		Err: fmt.Errorf(
			"validation failed due to an incorrectly formed reference to a %s (\"%s\") in \"%s\". "+
				"See the spec documentation for examples and rules for references",
			referenceableLabel(referenceableType),
			reference,
			context,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errIncludeEmptyPath(includeName string, varSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(varSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidInclude,
		Err: fmt.Errorf(
			"validation failed due to a missing or empty path for include \"%s\"",
			includeName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errIncludeMissingRequiredVar(
	includeName string,
	varName string,
	sourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(sourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeIncludeMissingRequiredVar,
		Err: fmt.Errorf(
			"validation failed due to required variable %q not being provided"+
				" to include %q",
			varName,
			includeName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errIncludeVarTypeMismatch(
	includeName string,
	varName string,
	actualType string,
	expectedType string,
	sourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(sourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeIncludeVarTypeMismatch,
		Err: fmt.Errorf(
			"validation failed due to variable %q provided to include %q"+
				" having type %q, but the child blueprint expects type %q",
			varName,
			includeName,
			actualType,
			expectedType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingFilter(dataSourceName string, dataSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceMissingFilter,
		Err: fmt.Errorf(
			"validation failed due to a missing filter in "+
				"data source \"%s\", every data source must have a filter",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceMissingFilter,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Add Filter",
					Description: "Add a filter to the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceEmptyFilter(dataSourceName string, dataSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceEmptyFilter,
		Err: fmt.Errorf(
			"validation failed due to an empty filter in "+
				"data source \"%s\", filters cannot be null or empty objects",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceEmptyFilter,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Provide Valid Filter",
					Description: "Provide a valid filter for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingFilterField(dataSourceName string, dataSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceMissingFilterField,
		Err: fmt.Errorf(
			"validation failed due to a missing field in filter for "+
				"data source \"%s\", field must be set for a data source filter",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceMissingFilterField,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Add Filter Field",
					Description: "Add a filter field to the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingFilterSearch(dataSourceName string, dataSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceMissingFilterSearch,
		Err: fmt.Errorf(
			"validation failed due to a missing search in filter for "+
				"data source \"%s\", at least one search value must be provided for a filter",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceMissingFilterSearch,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Add Filter Search Value",
					Description: "Add at least one search value to the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingExports(dataSourceName string, dataSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceMissingExports,
		Err: fmt.Errorf(
			"validation failed due to missing exports for "+
				"data source \"%s\", at least one field must be exported for a data source",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceMissingExports,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceExport),
					Title:       "Add Export",
					Description: "Add an exported field for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceFilterFieldConflict(
	dataSourceName string,
	fieldName string,
	otherField string,
	filterLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(filterLocation)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceFilterConflict,
		Err: fmt.Errorf(
			"validation failed due to a conflict between the filter fields %q and %q in data source %q, "+
				"you must use one or the other in the filter section of the data source",
			fieldName,
			otherField,
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceFilterConflict,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Fix Filter Field Conflict",
					Description: "Fix the conflict between the filter fields.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
				"fieldName":      fieldName,
				"otherFieldName": otherField,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidDataSourceFilterOperator(
	dataSourceName string,
	dataSourceFilterOperator *schema.DataSourceFilterOperatorWrapper,
) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceFilterOperator.SourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSourceFilterOperator,
		Err: fmt.Errorf(
			"invalid filter operator %q has been provided in data source %q, you can choose from %s",
			dataSourceFilterOperator.Value,
			dataSourceName,
			strings.Join(
				core.Map(schema.DataSourceFilterOperators, func(operator schema.DataSourceFilterOperator, index int) string {
					return fmt.Sprintf("\"%s\"", operator)
				}),
				", ",
			),
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeInvalidDataSourceFilterOperator,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Choose Valid Filter Operator",
					Description: "Choose a valid filter operator for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
				"filterOperator": dataSourceFilterOperator.Value,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceFilterOperatorNotSupported(
	dataSourceName string,
	operator schema.DataSourceFilterOperator,
	filterFieldName string,
	supportedOperators []schema.DataSourceFilterOperator,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeUnsupportedDataSourceFilterOperator,
		Err: fmt.Errorf(
			"data source %q does not support the filter operator %q for field %q, "+
				"supported operators are: %s",
			dataSourceName,
			operator,
			filterFieldName,
			strings.Join(
				core.Map(supportedOperators, func(op schema.DataSourceFilterOperator, index int) string {
					return fmt.Sprintf("\"%s\"", op)
				}),
				", ",
			),
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeUnsupportedDataSourceFilterOperator,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Choose Valid Filter Operator",
					Description: "Choose a valid filter operator for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName":  dataSourceName,
				"filterOperator":  operator,
				"filterFieldName": filterFieldName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingFilterOperator(dataSourceName string, location *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSourceFilterOperator,
		Err: fmt.Errorf(
			"data source %q has an empty filter operator, you can choose from %s",
			dataSourceName,
			strings.Join(
				core.Map(schema.DataSourceFilterOperators, func(operator schema.DataSourceFilterOperator, index int) string {
					return fmt.Sprintf("\"%s\"", operator)
				}),
				", ",
			),
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeInvalidDataSourceFilterOperator,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceFilter),
					Title:       "Choose Valid Filter Operator",
					Description: "Choose a valid filter operator for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidDataSourceFieldType(
	dataSourceName string,
	exportName string,
	dataSourceFieldType *schema.DataSourceFieldTypeWrapper,
) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceFieldType.SourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSourceFieldType,
		Err: fmt.Errorf(
			"unsupported field type %q has been provided for export %q in data source %q, "+
				"you can choose from: string, integer, float, boolean and array",
			dataSourceFieldType.Value,
			exportName,
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeInvalidDataSourceFieldType,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceExport),
					Title:       "Choose Valid Field Export Type",
					Description: "Choose a valid field export type for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
				"exportName":     exportName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceSpecPreValidationFailed(errs []error, resourceName string, resourceSourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(resourceSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to errors in the pre-validation of the resource spec for resource \"%s\"",
			resourceName,
		),
		ChildErrors:    errs,
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

// errMultipleValidationErrors is used to wrap multiple errors that occurred during validation.
// The idea is to collect and surface as many validation errors to the user as possible
// to provide them the full picture of issues in the blueprint instead of just the first error.
func ErrMultipleValidationErrors(errs []error) error {
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeMultipleValidationErrors,
		Err:         fmt.Errorf("validation failed due to multiple errors"),
		ChildErrors: errs,
	}
}

func errMappingNodeKeyContainsSubstitution(
	key string,
	nodeParentType string,
	nodeParentName string,
	nodeSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(nodeSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidMapKey,
		Err: fmt.Errorf(
			"${..} substitutions can not be used in map keys,"+
				" found \"%s\" in child mapping key of %s \"%s\"",
			key,
			nodeParentType,
			nodeParentName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncInvalidNumberOfArgs(
	expectedParamCount int,
	passedArgCount int,
	subFunc *substitutions.SubstitutionFunctionExpr,
) error {
	posRange := source.PositionRangeFromSourceMeta(subFunc.SourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid number of arguments "+
				"being provided for substitution function \"%s\", expected %d but got %d",
			subFunc.FunctionName,
			expectedParamCount,
			passedArgCount,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncArgTypeMismatch(
	argIndex int,
	expectedType string,
	actualType string,
	funcName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid argument type being provided for substitution function \"%s\", "+
				"expected argument %d to be of type %s but got %s",
			funcName,
			argIndex,
			expectedType,
			actualType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncArgInvalidStringChoice(
	argIndex int,
	expectedChoices []string,
	actualValue string,
	funcName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid argument value being provided for substitution function \"%s\", "+
				"expected argument %d to be one of the following choices: %s but got \"%s\"",
			funcName,
			argIndex,
			strings.Join(expectedChoices, ", "),
			actualValue,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncNamedArgsNotAllowed(
	argName string,
	funcName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to named arguments being provided for substitution function \"%s\", "+
				"found named argument \"%s\", named arguments are only supported in the \"%s\" function",
			funcName,
			argName,
			substitutions.SubstitutionFunctionObject,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFailedToLoadFunctionDefintion(
	funcName string,
	location *source.Meta,
	errInfo string,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to failure to load function definition for substitution function \"%s\": %s",
			funcName,
			errInfo,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubVarNotFound(
	varName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the variable \"%s\" not existing in the blueprint",
			varName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubValSelfReference(
	valName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the value \"%s\" referencing itself",
			valName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubValNotFound(
	valName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the value \"%s\" not existing in the blueprint",
			valName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubElemRefNotInResource(
	elemRefType string,
	location *source.Meta,
) error {
	elemRefTypeLabel := deriveElemRefTypeLabel(elemRefType)
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an %s reference being used outside of a resource",
			elemRefTypeLabel,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubElemRefResourceNotFound(
	elemRefType string,
	resourceName string,
	location *source.Meta,
) error {
	elemRefTypeLabel := deriveElemRefTypeLabel(elemRefType)
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the resource \"%s\" for %s reference not existing in the blueprint",
			resourceName,
			elemRefTypeLabel,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubElemRefResourceNotEach(
	elemRefType string,
	resourceName string,
	location *source.Meta,
) error {
	elemRefTypeLabel := deriveElemRefTypeLabel(elemRefType)
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the resource \"%s\" for %s reference not "+
				"being a resource template, a resource template must have the `each` property defined",
			resourceName,
			elemRefTypeLabel,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceNotFound(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the resource \"%s\" not existing in the blueprint",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceSelfReference(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the resource \"%s\" referencing itself",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceNotFound(
	dataSourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the data source \"%s\" not existing in the blueprint",
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceSelfReference(
	dataSourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the data source \"%s\" referencing itself",
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubChildBlueprintNotFound(
	childName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the child blueprint \"%s\" not existing in the blueprint",
			childName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubChildBlueprintSelfReference(
	childName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the child blueprint \"%s\" referencing itself",
			childName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceNotEach(
	resourceName string,
	indexAccessed *int64,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the index %d is accessed for resource \"%s\""+
				" which is not a resource template, "+
				"a resource template must have the `each` property defined",
			*indexAccessed,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceNoExportedFields(
	dataSourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to no fields being exported for data source \"%s\" "+
				"referenced in substitution",
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceFieldNotExported(
	dataSourceName string,
	field string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the field \"%s\" referenced in the substitution"+
				" not being an exported field for data source \"%s\"",
			field,
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceFieldMissingType(
	dataSourceName string,
	field string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to the field \"%s\" referenced in the substitution"+
				" not having a type defined for data source \"%s\"",
			field,
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubDataSourceFieldNotArray(
	dataSourceName string,
	field string,
	indexAccessed int64,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the field \"%s\" being referenced with index \"%d\" in the substitution"+
				" is not an array for data source \"%s\"",
			field,
			indexAccessed,
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceMissingType(resourceName string, location *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode:     ErrorReasonCodeInvalidResource,
		Err:            fmt.Errorf("resource %q missing type", resourceName),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeInvalidResource,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddResourceType),
					Title:       "Add Resource Type",
					Description: fmt.Sprintf("Add a type field to the resource %s", resourceName),
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckResourceType),
					Title:       "Check Available Resource Types",
					Description: "See available resource types from installed providers",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"resourceName": resourceName,
			},
		},
	}
}

type wrapRegistryErrorOption func(*wrapRegistryErrorOptions)

type wrapRegistryErrorOptions struct {
	inSubstitution bool
}

// WrapInSubstitution marks the error as occurring within a substitution context.
func WrapInSubstitution() wrapRegistryErrorOption {
	return func(opts *wrapRegistryErrorOptions) {
		opts.inSubstitution = true
	}
}

// wrapRegistryError wraps a RunError from the resource or data source registry into a LoadError
// while preserving the original error's context and suggested actions.
// This is needed because the language server only handles LoadError types for diagnostics
// and errors such as a missing provider or resource type should appear in diagnostics.
func wrapRegistryError(err error, location *source.Meta, opts ...wrapRegistryErrorOption) error {
	posRange := source.PositionRangeFromSourceMeta(location)

	options := &wrapRegistryErrorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	contextSuffix := ""
	if options.inSubstitution {
		contextSuffix = " (referenced in substitution)"
	}

	runErr, isRunErr := err.(*errors.RunError)
	if !isRunErr {
		return &errors.LoadError{
			Err:            fmt.Errorf("%s%s", err.Error(), contextSuffix),
			Line:           posRange.Line,
			EndLine:        posRange.EndLine,
			Column:         posRange.Column,
			EndColumn:      posRange.EndColumn,
			ColumnAccuracy: posRange.ColumnAccuracy,
		}
	}

	// If this is a multiple errors wrapper (e.g., provider not found AND transformer not found),
	// extract the first child error which typically has the most relevant context for the user.
	if len(runErr.ChildErrors) > 0 {
		if firstChild, ok := runErr.ChildErrors[0].(*errors.RunError); ok {
			return &errors.LoadError{
				ReasonCode:     firstChild.ReasonCode,
				Err:            fmt.Errorf("%s%s", firstChild.Err.Error(), contextSuffix),
				Context:        firstChild.Context,
				Line:           posRange.Line,
				EndLine:        posRange.EndLine,
				Column:         posRange.Column,
				EndColumn:      posRange.EndColumn,
				ColumnAccuracy: posRange.ColumnAccuracy,
			}
		}
	}

	return &errors.LoadError{
		ReasonCode:     runErr.ReasonCode,
		Err:            fmt.Errorf("%s%s", runErr.Err.Error(), contextSuffix),
		Context:        runErr.Context,
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceTypeMissingSpecDefinition(
	resourceName string,
	resourceType string,
	inSubstitution bool,
	resourceSourceMeta *source.Meta,
	extraDetails string,
) error {
	posRange := source.PositionRangeFromSourceMeta(resourceSourceMeta)
	contextInfo := ""
	if inSubstitution {
		contextInfo = " referenced in substitution"
	}
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceTypeSpecDefMissing,
		Err: fmt.Errorf(
			"validation failed due to a missing spec definition for resource \"%s\" "+
				"of type \"%s\"%s: %s",
			resourceName,
			resourceType,
			contextInfo,
			extraDetails,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeResourceTypeSpecDefMissing,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeContactResourceTypeDeveloper),
					Title:       "Contact Resource Type Developer",
					Description: fmt.Sprintf("Contact the developer of the resource type %s to fix the issue.", resourceType),
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"resourceName": resourceName,
				"resourceType": resourceType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceTypeSpecDefMissingSchema(
	resourceName string,
	resourceType string,
	inSubstitution bool,
	resourceSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(resourceSourceMeta)
	contextInfo := ""
	if inSubstitution {
		contextInfo = " referenced in substitution"
	}
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceTypeSpecDefMissingSchema,
		Err: fmt.Errorf(
			"validation failed due to a missing spec definition schema for resource \"%s\" "+
				"of type \"%s\"%s",
			resourceName,
			resourceType,
			contextInfo,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeResourceTypeSpecDefMissingSchema,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeContactResourceTypeDeveloper),
					Title:       "Contact Resource Type Developer",
					Description: fmt.Sprintf("Contact the developer of the resource type %s to fix the issue.", resourceType),
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"resourceName": resourceName,
				"resourceType": resourceType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceTypeMissingSpecDefinition(
	dataSourceName string,
	dataSourceType string,
	inSubstitution bool,
	dataSourceLocation *source.Meta,
	extraDetails string,
) error {
	posRange := source.PositionRangeFromSourceMeta(dataSourceLocation)
	contextInfo := ""
	if inSubstitution {
		contextInfo = " referenced in substitution"
	}

	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceSpecDefMissing,
		Err: fmt.Errorf(
			"validation failed due to a missing spec definition for data source \"%s\" "+
				"of type \"%s\"%s: %s",
			dataSourceName,
			dataSourceType,
			contextInfo,
			extraDetails,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceSpecDefMissing,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeContactDataSourceTypeDeveloper),
					Title:       "Contact Data Source Type Developer",
					Description: fmt.Sprintf("Contact the developer of the data source type %s to fix the issue.", dataSourceType),
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
				"dataSourceType": dataSourceType,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceTypeMissingFields(
	dataSourceName string,
	dataSourceType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceSpecDefMissing,
		Err: fmt.Errorf(
			"validation failed due to a missing fields definition for data source \"%s\" "+
				"of type \"%s\"",
			dataSourceName,
			dataSourceType,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceSpecDefMissing,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeContactDataSourceTypeDeveloper),
					Title:       "Contact Data Source Type Developer",
					Description: fmt.Sprintf("Contact the developer of the data source type %s to fix the issue.", dataSourceType),
					Priority:    1,
				},
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceFilterFieldNotSupported(
	dataSourceName string,
	field string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceFilterFieldNotSupported,
		Err: fmt.Errorf(
			"validation failed due to the field \"%s\" in the filter for data source \"%s\" "+
				"not being supported",
			field,
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceFilterFieldNotSupported,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckDataSourceFilterFields),
					Title:       "Choose a different field for filtering",
					Description: "Choose a different field for filtering from the list of supported fields.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
				"fieldName":      field,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceMissingType(dataSourceName string, location *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeDataSourceMissingType,
		Err: fmt.Errorf(
			"validation failed due to a missing type for data source \"%s\"",
			dataSourceName,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeDataSourceMissingType,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddDataSourceType),
					Title:       "Add Data Source Type",
					Description: "Add a type for the data source.",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"dataSourceName": dataSourceName,
			},
		},
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceSpecInvalidRef(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the spec reference for resource \"%s\" is not valid",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataInvalidRef(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata reference for resource \"%s\" is not valid",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataInvalidProperty(
	resourceName string,
	property string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata property \"%s\" provided for resource \"%s\" is not valid",
			property,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataInvalidDisplayNameRef(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata display name reference for "+
				"resource \"%s\" provided can not have children",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataInvalidAnnotationsRef(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata annotations reference for "+
				"resource \"%s\" was invalid, must be of the form "+
				"`metadata.annotations.<key>` or `metadata.annotations[\"<key>\"]`",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataMissingAnnotation(
	resourceName string,
	annotationKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata annotation \"%s\" for "+
				"resource \"%s\" was not found",
			annotationKey,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataInvalidLabelsRef(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata labels reference for "+
				"resource \"%s\" was invalid, must be of the form "+
				"`metadata.labels.<key>` or `metadata.labels[\"<key>\"]`",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataMissingLabel(
	resourceName string,
	labelKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the metadata label \"%s\" for "+
				"resource \"%s\" was not found",
			labelKey,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourceMetadataCustomEmpty(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as resource \"%s\" has no custom metadata defined",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubMappingNodePathNotFound(
	contextName string,
	path string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the path \"%s\" in \"%s\" could not be resolved",
			path,
			contextName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubMappingNodeFieldNotFound(
	contextName string,
	path string,
	fieldName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as the field \"%s\" in path \"%s\" was not found in \"%s\"",
			fieldName,
			path,
			contextName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubMappingNodeIndexOutOfBounds(
	contextName string,
	path string,
	index int,
	length int,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as index %d in path \"%s\" is out of bounds (length %d) in \"%s\"",
			index,
			path,
			length,
			contextName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubResourcePropertyNotFound(
	resourceName string,
	path []*substitutions.SubstitutionPathItem,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed as %s is not valid for resource \"%s\"",
			subPathToString(path),
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidDescriptionSubType(
	usedIn string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by descriptions, "+
				"only values that resolve as primitives are supported",
			usedIn,
			resolvedType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidIncludePathSubType(
	usedIn string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by include paths, "+
				"only values that resolve as primitives are supported",
			usedIn,
			resolvedType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidDisplayNameSubType(
	usedIn string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by display names, "+
				"only values that resolve as primitives are supported",
			usedIn,
			resolvedType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidSubType(
	usedIn string,
	valueContext string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by %ss, "+
				"only values that resolve as primitives are supported",
			usedIn,
			resolvedType,
			valueContext,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidSubTypeNotBoolean(
	usedIn string,
	valueContext string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by %ss, "+
				"only values that resolve as booleans are supported",
			usedIn,
			resolvedType,
			valueContext,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidSubTypeNotArray(
	usedIn string,
	valueContext string,
	resolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported in %s, "+
				"only values that resolve as arrays are supported",
			usedIn,
			resolvedType,
			valueContext,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errEmptyEachSubstitution(
	usedIn string,
	valueContext string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSubstitution,
		Err: fmt.Errorf(
			"%s substitution in the \"each\" property of %q is empty, "+
				"a single value must be provided that resolves to an array",
			valueContext,
			usedIn,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errMissingValueContent(
	valueID string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValue,
		Err: fmt.Errorf(
			"validation failed as an empty value was found in %q, "+
				"values must be populated with a value that resolves to the defined value type",
			valueID,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errValueIncorrectTypeInterpolatedString(
	usedIn string,
	valueType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValue,
		Err: fmt.Errorf(
			"validation failed due to an interpolated string being used in %q, "+
				"value type %q does not support interpolated strings",
			usedIn,
			valueType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidValueSubType(
	usedIn string,
	resolvedType string,
	expectedResolvedType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValue,
		Err: fmt.Errorf(
			"validation failed due to an invalid substitution found in %q, "+
				"resolved type %q is not supported by value of type %q",
			usedIn,
			resolvedType,
			expectedResolvedType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidValueContentType(
	valIdentifier string,
	resolvedSubType string,
	expectedResolveType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValue,
		Err: fmt.Errorf(
			"validation failed due to incorrect value content in %q, "+
				"the content provided is of type %q but the expected value type is %q",
			valIdentifier,
			resolvedSubType,
			expectedResolveType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errMissingValueType(
	valName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValue,
		Err: fmt.Errorf(
			"validation failed as the value %q is missing a type, "+
				"all values must have a type defined",
			valName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errInvalidValueType(
	valName string,
	valType *schema.ValueTypeWrapper,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidValueType,
		Err: fmt.Errorf(
			"validation failed as an unsupported type %q was provided for value %q, "+
				"you can choose from: string, integer, float, boolean, object and array",
			valType.Value,
			valName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

// ErrReferenceCycles is used to wrap errors that occurred during reference cycle validation.
// This error is used to collect and surface reference cycle errors for pure substitution reference
// cycles and link <-> substitution reference cycles.
func ErrReferenceCycles(rootRefChains []*refgraph.ReferenceChainNode) error {
	var errs []error
	for _, refChain := range rootRefChains {
		errs = append(errs, &errors.LoadError{
			ReasonCode: ErrorReasonCodeReferenceCycle,
			Err: fmt.Errorf(
				"validation failed due to a reference cycle in the blueprint, "+
					"the cycle started with element: %q, this could be due to explicit references between elements "+
					"or an implicit link conflicting with an explicit item reference",
				refChain.ElementName,
			),
		})
	}
	return ErrMultipleValidationErrors(errs)
}

func errDataSourceExportFieldNotSupported(
	dataSourceName string,
	dataSourceType string,
	exportAlias string,
	exportedSourceField string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to the exported field %q in data source %q not being supported, "+
				"the exported field %q is not present for data source type %q",
			exportAlias,
			dataSourceName,
			exportedSourceField,
			dataSourceType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceExportFieldTypeMismatch(
	dataSourceName string,
	exportAlias string,
	dataSourceField string,
	dataSourceFieldType string,
	exportedFieldType string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to the exported field %q in data source %q having an unexpected type, "+
				"the data source field %q has a type of %q, but the exported type is %q",
			exportAlias,
			dataSourceName,
			dataSourceField,
			dataSourceFieldType,
			exportedFieldType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceExportEmpty(
	dataSourceName string,
	exportName string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to the exported field %q in data source %q having an empty value",
			exportName,
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceExportTypeMissing(
	dataSourceName string,
	exportName string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to export %q in data source %q missing a type",
			exportName,
			dataSourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errDataSourceTypeNotSupported(
	dataSourceName string,
	dataSourceType string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	providerNamespace := provider.ExtractProviderFromItemType(dataSourceType)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to data source %q having an unsupported type %q,"+
				" this type is not made available by any of the loaded providers",
			dataSourceName,
			dataSourceType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryDataSourceType,
			ReasonCode: ErrorReasonCodeInvalidDataSource,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Provider",
					Description: fmt.Sprintf("Install the %s provider to support %s data sources", providerNamespace, dataSourceType),
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckDataSourceType),
					Title:       "Check Data Source Type",
					Description: "Verify the data source type name is correct",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"providerNamespace": providerNamespace,
				"dataSourceName":    dataSourceName,
				"dataSourceType":    dataSourceType,
			},
		},
	}
}

func errDataSourceAnnotationKeyContainsSubstitution(
	dataSourceName string,
	annotationKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidDataSource,
		Err: fmt.Errorf(
			"validation failed due to an annotation key containing a substitution in data source %q, "+
				"the annotation key %q can not contain substitutions",
			dataSourceName,
			annotationKey,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceTypeNotSupported(
	resourceName string,
	resourceType string,
	wrapperLocation *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(wrapperLocation)
	pluginNamespace := provider.ExtractProviderFromItemType(resourceType)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to resource %q having an unsupported type %q,"+
				" this type is not made available by any of the loaded plugins",
			resourceName,
			resourceType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeInvalidResource,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Plugin",
					Description: fmt.Sprintf("Install a provider or transformer plugin with namespace %q that supports the %s resource type", pluginNamespace, resourceType),
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckResourceType),
					Title:       "Check Resource Type",
					Description: "Verify the resource type name is correct",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"providerNamespace": pluginNamespace,
				"resourceName":      resourceName,
				"resourceType":      resourceType,
				"category":          "resource",
			},
		},
	}
}

func errLabelKeyContainsSubstitution(
	resourceName string,
	labelKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a label key containing a substitution in resource %q, "+
				"the label key %q can not contain substitutions",
			resourceName,
			labelKey,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errLabelValueContainsSubstitution(
	resourceName string,
	labelKey string,
	labelValue string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a label value containing a substitution in resource %q, "+
				"the label %q with value %q can not contain substitutions",
			resourceName,
			labelKey,
			labelValue,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errLinkSelectorKeyContainsSubstitution(
	resourceName string,
	linkSelectorKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a link selector \"byLabel\" key containing a "+
				"substitution in resource %q, "+
				"the link selector label key %q can not contain substitutions",
			resourceName,
			linkSelectorKey,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errLinkSelectorValueContainsSubstitution(
	resourceName string,
	linkSelectorKey string,
	linkSelectorValue string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a link selector \"byLabel\" value containing a "+
				"substitution in resource %q, "+
				"the link selector label %q with value %q can not contain substitutions",
			resourceName,
			linkSelectorKey,
			linkSelectorValue,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errLinkSelectorExcludeContainsSubstitution(
	resourceName string,
	excludeValue string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a link selector \"exclude\" value containing a "+
				"substitution in resource %q, "+
				"the exclude value %q can not contain substitutions",
			resourceName,
			excludeValue,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errLinkSelectorExcludeResourceNotFound(
	resourceName string,
	excludeValue string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a link selector \"exclude\" value referencing "+
				"a resource that does not exist in resource %q, "+
				"the exclude value %q is not a valid resource name in this blueprint",
			resourceName,
			excludeValue,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errAnnotationKeyContainsSubstitution(
	resourceName string,
	annotationKey string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an annotation key containing a substitution in resource %q, "+
				"the annotation key %q can not contain substitutions",
			resourceName,
			annotationKey,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errNestedResourceConditionEmpty(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a nested condition for resource %q being empty, "+
				"all nested conditions must have a value defined",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errExportTypeMismatch(
	exportType schema.ExportType,
	resolvedType string,
	exportName string,
	field string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidExport,
		Err: fmt.Errorf(
			"validation failed due to a type mismatch in export %q, "+
				"the expected export type %s does not match the resolved type %s for field %q",
			exportName,
			exportType,
			resolvedType,
			field,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceDependencyMissing(
	resourceName string,
	dependencyName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeMissingResourceDependency,
		Err: fmt.Errorf(
			"validation failed due to a missing dependency %q for resource %q",
			dependencyName,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errResourceDependencyContainsSubstitution(
	resourceName string,
	dependencyName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a dependency %q containing a substitution in resource %q, "+
				"the dependency name %q can not contain substitutions and must be a resource in the same blueprint",
			dependencyName,
			resourceName,
			dependencyName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSelfReferencingResourceDependency(
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a self-referencing dependency in resource %q",
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errComputedFieldDefinedInBlueprint(
	path string,
	resourceName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeComputedFieldInBlueprint,
		Err: fmt.Errorf(
			"validation failed due to %q being a computed field defined in the blueprint for resource %q, "+
				"this field is computed by the provider after the resource has been created",
			path,
			resourceName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errEachResourceDependencyDetected(
	resourceIDWithEachProp string,
	dependencyName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeEachResourceDependency,
		Err: fmt.Errorf(
			"validation failed due to a resource %q having a direct or transitive dependency %q in the each property, "+
				"the each property can not depend on resources",
			resourceIDWithEachProp,
			dependencyName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errEachChildDependencyDetected(
	resourceIDWithEachProp string,
	dependencyName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeEachChildDependency,
		Err: fmt.Errorf(
			"validation failed due to a resource %q having a direct or transitive dependency "+
				"on a child blueprint %q in the each property, "+
				"the each property can not depend on child blueprints",
			resourceIDWithEachProp,
			dependencyName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncLinkArgResourceNotFound(
	resourceName string,
	argIndex int,
	usedIn string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeSubFuncLinkArgResourceNotFound,
		Err: fmt.Errorf(
			"validation failed due to a missing resource %q being referenced in the link function"+
				" call argument at position %d in %q",
			resourceName,
			argIndex,
			usedIn,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func deriveElemRefTypeLabel(elemRefType string) string {
	switch elemRefType {
	case "index":
		return "element index"
	default:
		return "element"
	}
}

func deriveOptionalVarType(value *bpcore.ScalarValue) *schema.VariableType {
	if value.IntValue != nil {
		intVarType := schema.VariableTypeInteger
		return &intVarType
	}

	if value.FloatValue != nil {
		floatVarType := schema.VariableTypeFloat
		return &floatVarType
	}

	if value.BoolValue != nil {
		boolVarType := schema.VariableTypeBoolean
		return &boolVarType
	}

	if value.StringValue != nil {
		stringVarType := schema.VariableTypeString
		return &stringVarType
	}

	return nil
}

func scalarListToString(scalars []*bpcore.ScalarValue) string {
	scalarStrings := make([]string, len(scalars))
	for i, scalar := range scalars {
		scalarStrings[i] = deriveScalarValueAsString(scalar)
	}

	return strings.Join(scalarStrings, ", ")
}

func deriveValueLabel(usingDefault bool) string {
	if usingDefault {
		return "default value"
	}

	return "value"
}

func positionFromScalarValue(
	value *bpcore.ScalarValue,
	parentSourceMeta *source.Meta,
) *source.PositionRange {
	if value == nil {
		if parentSourceMeta != nil {
			return source.PositionRangeFromSourceMeta(parentSourceMeta)
		}
		return nil
	}

	return source.PositionRangeFromSourceMeta(value.SourceMeta)
}

func subPathToString(path []*substitutions.SubstitutionPathItem) string {
	sb := strings.Builder{}
	for _, item := range path {
		if item.FieldName != "" {
			fieldStr := fmt.Sprintf("[\"%s\"]", item.FieldName)
			sb.WriteString(fieldStr)
		} else {
			pathStr := fmt.Sprintf("[%d]", *item.ArrayIndex)
			sb.WriteString(pathStr)
		}
	}

	return sb.String()
}

func errIncludePathNotFound(
	includeName string,
	resolvedPath string,
	pathSourceMeta *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(pathSourceMeta)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeIncludePathNotFound,
		Err: fmt.Errorf(
			"validation failed due to the include path for %q resolving to %q"+
				" which does not exist on the local filesystem",
			includeName,
			resolvedPath,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

// ErrChildExportNotFound returns an error when a referenced export is not found
// in a resolved child blueprint.
func ErrChildExportNotFound(
	childName string,
	exportName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeChildExportNotFound,
		Err: fmt.Errorf(
			"validation failed due to export %q not being found"+
				" in child blueprint %q",
			exportName,
			childName,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errChildExportScalarNavigation(
	childName string,
	exportName string,
	exportType string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeChildExportScalarNavigation,
		Err: fmt.Errorf(
			"validation failed due to an attempt to access a nested property"+
				" of export %q in child blueprint %q which has scalar type %q",
			exportName,
			childName,
			exportType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func warnArrayIndexBoundsUnverifiable(
	contextName string,
	contextType string,
	index int64,
	location *source.Meta,
) *bpcore.Diagnostic {
	return &bpcore.Diagnostic{
		Level: bpcore.DiagnosticLevelWarning,
		Message: fmt.Sprintf(
			"Array index [%d] in %s %q cannot be validated at this stage;"+
				" the value at this index may not exist at deploy time",
			index,
			contextType,
			contextName,
		),
		Range: bpcore.DiagnosticRangeFromSourceMeta(location, nil),
		Context: &errors.ErrorContext{
			ReasonCode: errors.ErrorReasonCodeAnyTypeWarning,
		},
	}
}

func errSubFuncPathIndexOnNonArray(
	funcName string,
	returnType string,
	index int64,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeSubFuncPathIndexOnNonArray,
		Err: fmt.Errorf(
			"validation failed due to an array index [%d] being applied to the result"+
				" of function %q which returns type %q; array index access"+
				" requires the return type to be an array or object",
			index,
			funcName,
			returnType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func errSubFuncPathFieldOnNonObject(
	funcName string,
	returnType string,
	fieldName string,
	location *source.Meta,
) error {
	posRange := source.PositionRangeFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeSubFuncPathFieldOnNonObject,
		Err: fmt.Errorf(
			"validation failed due to a field accessor %q being applied to the result"+
				" of function %q which returns type %q; field access"+
				" requires the return type to be an object",
			fieldName,
			funcName,
			returnType,
		),
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}
