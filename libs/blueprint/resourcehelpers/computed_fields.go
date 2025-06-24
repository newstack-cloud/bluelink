package resourcehelpers

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// IsComputedField returns whether the given field path is a computed field
// in the given set of resource changes.
func IsComputedField(changes *provider.Changes, fieldPath string) bool {
	if changes == nil {
		return false
	}

	return IsComputedFieldInList(changes.ComputedFields, fieldPath)
}

// IsComputedFieldInList returns whether the given field path is a computed field
// in the given list of computed fields.
//
// This allows for matching "[0]" and "[\"<key>\"]" placeholders in the expectedComputedFields
// list to match any array item or map key-value pair in the computed field path.
func IsComputedFieldInList(expectedComputedFields []string, fieldPath string) bool {
	foundMatch := false
	i := 0
	for !foundMatch && i < len(expectedComputedFields) {
		computedField := expectedComputedFields[i]
		foundMatch = computedField == fieldPath
		if !foundMatch &&
			(strings.Contains(expectedComputedFields[i], "[0]") ||
				strings.Contains(expectedComputedFields[i], "[\"<key>\"]")) {
			// Compare each path item to match a placeholder for an array item
			// or map key-value pair with the given field path.
			// The computed field path will always be in a resource spec,
			// so we can safely use a resource property path representations
			// to reuse existing path parsing logic in the substitutions package.
			computedFieldPropPath := parseResourcePropertyPath(computedField)
			fieldPropPath := parseResourcePropertyPath(fieldPath)

			if len(computedFieldPropPath) == len(fieldPropPath) {
				foundMatch = parsedPathMatchesComputedField(computedFieldPropPath, fieldPropPath)
			}
		}
		i += 1
	}

	return foundMatch
}

func parsedPathMatchesComputedField(
	computedFieldPropPath []*substitutions.SubstitutionPathItem,
	fieldPropPath []*substitutions.SubstitutionPathItem,
) bool {

	pathsMatch := true
	i := 0
	for pathsMatch && i < len(computedFieldPropPath) {
		computedFieldPathItem := computedFieldPropPath[i]
		fieldPathItem := fieldPropPath[i]
		pathsMatch = pathsMatch && (pathItemMatchesForField(computedFieldPathItem, fieldPathItem) ||
			pathItemMatchesForArrayItem(computedFieldPathItem, fieldPathItem))
		i += 1
	}

	return pathsMatch
}

func pathItemMatchesForField(
	computedFieldPathItem *substitutions.SubstitutionPathItem,
	fieldPathItem *substitutions.SubstitutionPathItem,
) bool {
	return computedFieldPathItem.FieldName != "" &&
		fieldPathItem.FieldName != "" &&
		(computedFieldPathItem.FieldName == fieldPathItem.FieldName ||
			computedFieldPathItem.FieldName == "__key__")
}

func pathItemMatchesForArrayItem(
	computedFieldPathItem *substitutions.SubstitutionPathItem,
	fieldPathItem *substitutions.SubstitutionPathItem,
) bool {
	return computedFieldPathItem.ArrayIndex != nil &&
		fieldPathItem.ArrayIndex != nil &&
		(*computedFieldPathItem.ArrayIndex == *fieldPathItem.ArrayIndex ||
			// 0 is a placeholder for any array item in the computed field path.
			*computedFieldPathItem.ArrayIndex == 0)
}

func parseResourcePropertyPath(
	fieldPath string,
) []*substitutions.SubstitutionPathItem {
	path := fmt.Sprintf("resources.test.%s", fieldPath)
	// ["<key>"] is not valid in the substitutions language.
	// Underscores are allowed, so we'll use underscores instead of
	// angle brackets for the key placeholder.
	finalPath := strings.ReplaceAll(path, "[\"<key>\"]", "[\"__key__\"]")
	sub, err := substitutions.ParseSubstitution(
		"",
		finalPath,
		nil,
		/* outputLinkeInfo */ false,
		/* ignoreParentColumn */ true,
	)
	if err != nil {
		return []*substitutions.SubstitutionPathItem{}
	}

	return sub.ResourceProperty.Path[1:]
}
