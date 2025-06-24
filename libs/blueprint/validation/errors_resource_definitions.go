package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

func errResourceDefItemEmpty(
	path string,
	resourceSpecType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an empty resource item "+
				"at path %q where the %s type was expected",
			path,
			resourceSpecType,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefInvalidType(
	path string,
	foundType provider.ResourceDefinitionsSchemaType,
	expectedType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an invalid resource item "+
				"at path %q where the %s type was expected, but %s was found",
			path,
			expectedType,
			foundType,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefMissingRequiredField(
	path string,
	field string,
	fieldType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a missing required field %q of type %s "+
				"at path %q",
			field,
			fieldType,
			path,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefUnknownField(
	path string,
	field string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an unknown field %q "+
				"at path %q, only fields that match the resource definition schema are allowed",
			field,
			path,
		),
		Line:   line,
		Column: col,
	}
}

func errInvalidResourceDefSubType(
	resolvedType string,
	path string,
	expectedResolvedType string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an invalid resource item "+
				"at path %q where a value of type %s was expected, but type %s was found",
			path,
			expectedResolvedType,
			resolvedType,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefUnionItemEmpty(
	path string,
	unionSchema []*provider.ResourceDefinitionsSchema,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	unionType := resourceDefinitionsUnionTypeToString(unionSchema)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an empty resource item "+
				"at path %s where one of the types %s was expected",
			path,
			unionType,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefUnionInvalidType(
	path string,
	unionSchema []*provider.ResourceDefinitionsSchema,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	unionType := resourceDefinitionsUnionTypeToString(unionSchema)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to an invalid resource item found "+
				"at path %q where one of the types %s was expected",
			path,
			unionType,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefNotAllowedValue(
	path string,
	allowedValuesText string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a value that is not allowed "+
				"being provided at path %q, the value must be one of: %s",
			path,
			allowedValuesText,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefPatternConstraintFailure(
	path string,
	pattern string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a value that does not match the pattern "+

				"constraint at path %q, the value must match the pattern: %s",
			path,
			pattern,
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefMinConstraintFailure(
	path string,
	value *core.ScalarValue,
	minimum *core.ScalarValue,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a value that is less than the minimum "+
				"constraint at path %q, %s provided but the value must be greater than or equal to %s",
			path,
			value.ToString(),
			minimum.ToString(),
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefMaxConstraintFailure(
	path string,
	value *core.ScalarValue,
	minimum *core.ScalarValue,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a value that is greater than the maximum "+
				"constraint at path %q, %s provided but the value must be less than or equal to %s",
			path,
			value.ToString(),
			minimum.ToString(),
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefComplexMinLengthConstraintFailure(
	path string,
	schemaType provider.ResourceDefinitionsSchemaType,
	valueLength int,
	minimumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to %s that has less items than the minimum "+
				"length constraint at path %q, %s provided when there must be at least %s",
			formatSchemaTypeForConstraintError(schemaType),
			path,
			formatNumberOfItems(valueLength, "item"),
			formatNumberOfItems(minimumLength, "item"),
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefComplexMaxLengthConstraintFailure(
	path string,
	schemaType provider.ResourceDefinitionsSchemaType,
	valueLength int,
	maximumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to %s that has more items than the maximum "+
				"length constraint at path %q, %s provided when there must be at most %s",
			formatSchemaTypeForConstraintError(schemaType),
			path,
			formatNumberOfItems(valueLength, "item"),
			formatNumberOfItems(maximumLength, "item"),
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefStringMinLengthConstraintFailure(
	path string,
	numberOfChars int,
	minimumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a string value that is shorter than the minimum "+
				"length constraint at path %q, %s provided when there must be at least %s",
			path,
			formatNumberOfItems(numberOfChars, "character"),
			formatNumberOfItems(minimumLength, "character"),
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefStringMaxLengthConstraintFailure(
	path string,
	numberOfChars int,
	maximumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResource,
		Err: fmt.Errorf(
			"validation failed due to a string value that is longer than the maximum "+
				"length constraint at path %q, %s provided when there must be at most %s",
			path,
			formatNumberOfItems(numberOfChars, "character"),
			formatNumberOfItems(maximumLength, "character"),
		),
		Line:   line,
		Column: col,
	}
}

func formatNumberOfItems(
	numberOfItems int,
	singularItemName string,
) string {
	if numberOfItems == 1 {
		return fmt.Sprintf("%d %s", numberOfItems, singularItemName)
	}
	return fmt.Sprintf("%d %ss", numberOfItems, singularItemName)
}

func formatSchemaTypeForConstraintError(
	schemaType provider.ResourceDefinitionsSchemaType,
) string {
	switch schemaType {
	case provider.ResourceDefinitionsSchemaTypeArray:
		return "an array"
	case provider.ResourceDefinitionsSchemaTypeMap:
		return "a map"
	case provider.ResourceDefinitionsSchemaTypeObject:
		return "an object"
	default:
		return fmt.Sprintf("a value of type %s", schemaType)
	}
}
