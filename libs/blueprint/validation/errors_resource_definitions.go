package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/common/strsim"
)

// createResourceDefErrorContext creates a standardized error context for resource definition validation errors
func createResourceDefErrorContext(reasonCode errors.ErrorReasonCode, resourceType string, metadata map[string]any) *errors.ErrorContext {
	// Add resourceType to metadata
	metadata["resourceType"] = resourceType

	return &errors.ErrorContext{
		Category:   errors.ErrorCategoryResourceType,
		ReasonCode: reasonCode,
		SuggestedActions: []errors.SuggestedAction{
			{
				Type:        string(errors.ActionTypeCheckResourceTypeSchema),
				Title:       "Check Resource Type Schema",
				Description: "Check the schema for the resource type for the expected fields and their types.",
				Priority:    1,
			},
		},
		Metadata: metadata,
	}
}

func errResourceDefItemEmpty(
	path string,
	resourceType string,
	resourceSpecType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefItemEmpty,
		Err: fmt.Errorf(
			"validation failed due to an empty resource item "+
				"at path %q where the %s type was expected",
			path,
			resourceSpecType,
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefItemEmpty,
			resourceType,
			map[string]any{
				"path":             path,
				"resourceSpecType": resourceSpecType,
			},
		),
	}
}

func errResourceDefInvalidType(
	path string,
	resourceType string,
	foundType provider.ResourceDefinitionsSchemaType,
	expectedType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefInvalidType,
		Err: fmt.Errorf(
			"validation failed due to an invalid resource item "+
				"at path %q where the %s type was expected, but %s was found",
			path,
			expectedType,
			foundType,
		),
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefInvalidType,
			resourceType,
			map[string]any{
				"path":         path,
				"expectedType": expectedType,
				"actualType":   foundType,
			},
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefMissingRequiredField(
	path string,
	resourceType string,
	field string,
	fieldType provider.ResourceDefinitionsSchemaType,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefMissingRequiredField,
		Err: fmt.Errorf(
			"validation failed due to a missing required field %q of type %s "+
				"at path %q",
			field,
			fieldType,
			path,
		),
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefMissingRequiredField,
			resourceType,
			map[string]any{
				"path":         path,
				"missingField": field,
			},
		),
		Line:   line,
		Column: col,
	}
}

func errResourceDefUnknownField(
	path string,
	resourceType string,
	field string,
	availableFields []string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)

	// Find similar field names for typo suggestions (max 3 results, default threshold)
	suggestions := strsim.FindSimilar(field, availableFields, 3, 0)

	metadata := map[string]any{
		"path":         path,
		"unknownField": field,
	}

	// Include available fields and suggestions in metadata for LSP to use
	if len(availableFields) > 0 {
		metadata["availableFields"] = availableFields
	}
	if len(suggestions) > 0 {
		metadata["suggestions"] = suggestions
	}

	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefUnknownField,
		Err: fmt.Errorf(
			"validation failed due to an unknown field %q "+
				"at path %q, only fields that match the resource definition schema are allowed",
			field,
			path,
		),
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefUnknownField,
			resourceType,
			metadata,
		),
		Line:   line,
		Column: col,
	}
}

func errInvalidResourceDefSubType(
	resolvedType string,
	path string,
	resourceType string,
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
		Context: createResourceDefErrorContext(
			ErrorReasonCodeInvalidResource,
			resourceType,
			map[string]any{
				"path":                 path,
				"expectedResolvedType": expectedResolvedType,
				"actualResolvedType":   resolvedType,
			},
		),
	}
}

func errResourceDefUnionItemEmpty(
	path string,
	resourceType string,
	unionSchema []*provider.ResourceDefinitionsSchema,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	unionType := resourceDefinitionsUnionTypeToString(unionSchema)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefUnionItemEmpty,
		Err: fmt.Errorf(
			"validation failed due to an empty resource item "+
				"at path %s where one of the types %s was expected",
			path,
			unionType,
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefUnionItemEmpty,
			resourceType,
			map[string]any{
				"path":      path,
				"unionType": unionType,
			},
		),
	}
}

func errResourceDefUnionInvalidType(
	path string,
	resourceType string,
	unionSchema []*provider.ResourceDefinitionsSchema,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	unionType := resourceDefinitionsUnionTypeToString(unionSchema)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefUnionInvalidType,
		Err: fmt.Errorf(
			"validation failed due to an invalid resource item found "+
				"at path %q where one of the types %s was expected",
			path,
			unionType,
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefUnionInvalidType,
			resourceType,
			map[string]any{
				"path":      path,
				"unionType": unionType,
			},
		),
	}
}

func errResourceDefNotAllowedValue(
	path string,
	resourceType string,
	allowedValuesText string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefNotAllowedValue,
		Err: fmt.Errorf(
			"validation failed due to a value that is not allowed "+
				"being provided at path %q, the value must be one of: %s",
			path,
			allowedValuesText,
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefNotAllowedValue,
			resourceType,
			map[string]any{
				"path":              path,
				"allowedValuesText": allowedValuesText,
			},
		),
	}
}

func errResourceDefPatternConstraintFailure(
	path string,
	resourceType string,
	pattern string,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefPatternConstraintFailure,
		Err: fmt.Errorf(
			"validation failed due to a value that does not match the pattern "+

				"constraint at path %q, the value must match the pattern: %s",
			path,
			pattern,
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefPatternConstraintFailure,
			resourceType,
			map[string]any{
				"path":    path,
				"pattern": pattern,
			},
		),
	}
}

func errResourceDefMinConstraintFailure(
	path string,
	resourceType string,
	value *core.ScalarValue,
	minimum *core.ScalarValue,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefMinConstraintFailure,
		Err: fmt.Errorf(
			"validation failed due to a value that is less than the minimum "+
				"constraint at path %q, %s provided but the value must be greater than or equal to %s",
			path,
			value.ToString(),
			minimum.ToString(),
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefMinConstraintFailure,
			resourceType,
			map[string]any{
				"path":    path,
				"value":   value.ToString(),
				"minimum": minimum.ToString(),
			},
		),
	}
}

func errResourceDefMaxConstraintFailure(
	path string,
	resourceType string,
	value *core.ScalarValue,
	minimum *core.ScalarValue,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefMaxConstraintFailure,
		Err: fmt.Errorf(
			"validation failed due to a value that is greater than the maximum "+
				"constraint at path %q, %s provided but the value must be less than or equal to %s",
			path,
			value.ToString(),
			minimum.ToString(),
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefMaxConstraintFailure,
			resourceType,
			map[string]any{
				"path":    path,
				"value":   value.ToString(),
				"maximum": minimum.ToString(),
			},
		),
	}
}

func errResourceDefComplexMinLengthConstraintFailure(
	path string,
	resourceType string,
	schemaType provider.ResourceDefinitionsSchemaType,
	valueLength int,
	minimumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefComplexMinLengthConstraintFailure,
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
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefComplexMinLengthConstraintFailure,
			resourceType,
			map[string]any{
				"path":          path,
				"schemaType":    schemaType,
				"valueLength":   valueLength,
				"minimumLength": minimumLength,
			},
		),
	}
}

func errResourceDefComplexMaxLengthConstraintFailure(
	path string,
	resourceType string,
	schemaType provider.ResourceDefinitionsSchemaType,
	valueLength int,
	maximumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefComplexMaxLengthConstraintFailure,
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
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefComplexMaxLengthConstraintFailure,
			resourceType,
			map[string]any{
				"path":          path,
				"schemaType":    schemaType,
				"valueLength":   valueLength,
				"maximumLength": maximumLength,
			},
		),
	}
}

func errResourceDefStringMinLengthConstraintFailure(
	path string,
	resourceType string,
	numberOfChars int,
	minimumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefStringMinLengthConstraintFailure,
		Err: fmt.Errorf(
			"validation failed due to a string value that is shorter than the minimum "+
				"length constraint at path %q, %s provided when there must be at least %s",
			path,
			formatNumberOfItems(numberOfChars, "character"),
			formatNumberOfItems(minimumLength, "character"),
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefStringMinLengthConstraintFailure,
			resourceType,
			map[string]any{
				"path":          path,
				"numberOfChars": numberOfChars,
				"minimumLength": minimumLength,
			},
		),
	}
}

func errResourceDefStringMaxLengthConstraintFailure(
	path string,
	resourceType string,
	numberOfChars int,
	maximumLength int,
	location *source.Meta,
) error {
	line, col := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeResourceDefStringMaxLengthConstraintFailure,
		Err: fmt.Errorf(
			"validation failed due to a string value that is longer than the maximum "+
				"length constraint at path %q, %s provided when there must be at most %s",
			path,
			formatNumberOfItems(numberOfChars, "character"),
			formatNumberOfItems(maximumLength, "character"),
		),
		Line:   line,
		Column: col,
		Context: createResourceDefErrorContext(
			ErrorReasonCodeResourceDefStringMaxLengthConstraintFailure,
			resourceType,
			map[string]any{
				"path":          path,
				"numberOfChars": numberOfChars,
				"maximumLength": maximumLength,
			},
		),
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
