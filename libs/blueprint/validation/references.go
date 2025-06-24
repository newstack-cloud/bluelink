package validation

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateReference validates a reference in a blueprint,
// a reference can be to a variable, resource, child blueprint or data source.
// This validation does not validate that the reference can be resolved,
// as this validation will normally be carried out at an early stage before information
// is available about what resources, variables, data sources or child blueprints are available.
func ValidateReference(
	reference string,
	context string,
	hasAccessTo []Referenceable,
	location *source.Meta,
) error {
	if strings.HasPrefix(reference, "variables.") || strings.HasPrefix(reference, "variables[") {
		return validateVariableReference(reference, context, hasAccessTo, location)
	}

	if strings.HasPrefix(reference, "datasources.") || strings.HasPrefix(reference, "datasources[") {
		return validateDataSourceReference(reference, context, hasAccessTo, location)
	}

	if strings.HasPrefix(reference, "children.") || strings.HasPrefix(reference, "children[") {
		return validateChildBlueprintReference(reference, context, hasAccessTo, location)
	}

	if strings.HasPrefix(reference, "values.") || strings.HasPrefix(reference, "values[") {
		return validateValueReference(reference, context, hasAccessTo, location)
	}

	// Resource references are used for all other cases as they can be made
	// with or without the "resources." prefix.
	return validateResourceReference(reference, context, hasAccessTo, location)
}

func validateVariableReference(reference string, context string, hasAccessTo []Referenceable, location *source.Meta) error {
	if !core.SliceContainsComparable(hasAccessTo, ReferenceableVariable) {
		return errReferenceContextAccess(reference, context, ReferenceableVariable, location)
	}

	if !substitutions.VariableReferencePattern.MatchString(reference) {
		return errInvalidReferencePattern(reference, context, ReferenceableVariable, location)
	}

	return nil
}

func validateValueReference(reference string, context string, hasAccessTo []Referenceable, location *source.Meta) error {
	if !core.SliceContainsComparable(hasAccessTo, ReferenceableValue) {
		return errReferenceContextAccess(reference, context, ReferenceableValue, location)
	}

	if !substitutions.ValueReferencePattern.MatchString(reference) {
		return errInvalidReferencePattern(reference, context, ReferenceableValue, location)
	}

	return nil
}

func validateDataSourceReference(reference string, context string, hasAccessTo []Referenceable, location *source.Meta) error {
	if !core.SliceContainsComparable(hasAccessTo, ReferenceableDataSource) {
		return errReferenceContextAccess(reference, context, ReferenceableDataSource, location)
	}

	if !substitutions.DataSourceReferencePattern.MatchString(reference) {
		return errInvalidReferencePattern(reference, context, ReferenceableDataSource, location)
	}

	return nil
}

func validateChildBlueprintReference(reference string, context string, hasAccessTo []Referenceable, location *source.Meta) error {
	if !core.SliceContainsComparable(hasAccessTo, ReferenceableChild) {
		return errReferenceContextAccess(reference, context, ReferenceableChild, location)
	}

	if !substitutions.ChildReferencePattern.MatchString(reference) {
		return errInvalidReferencePattern(reference, context, ReferenceableChild, location)
	}

	return nil
}

func validateResourceReference(reference string, context string, hasAccessTo []Referenceable, location *source.Meta) error {
	if !core.SliceContainsComparable(hasAccessTo, ReferenceableResource) {
		return errReferenceContextAccess(reference, context, ReferenceableResource, location)
	}

	if !substitutions.ResourceReferencePattern.MatchString(reference) {
		return errInvalidReferencePattern(reference, context, ReferenceableResource, location)
	}

	return nil
}

// Referencable is a type that can be referenced in a blueprint.
type Referenceable string

const (
	// ReferenceableResource signifies that a resource
	// can be referenced for a given context in a blueprint.
	ReferenceableResource Referenceable = "resource"
	// ReferenceableVariable signifies that a variable
	// can be referenced for a given context in a blueprint.
	ReferenceableVariable Referenceable = "variable"
	// ReferenceableValue signifies that a value
	// can be referenced for a given context in a blueprint.
	ReferenceableValue Referenceable = "value"
	// ReferenceableDataSource signifies that a data source
	// can be referenced for a given context in a blueprint.
	ReferenceableDataSource Referenceable = "datasource"
	// ReferenceableChild signifies that a child blueprint
	// can be referenced for a given context in a blueprint.
	ReferenceableChild Referenceable = "child"
)

func referenceableLabel(referenceable Referenceable) string {
	switch referenceable {
	case ReferenceableResource:
		return "resource"
	case ReferenceableVariable:
		return "variable"
	case ReferenceableValue:
		return "local value"
	case ReferenceableDataSource:
		return "data source"
	case ReferenceableChild:
		return "child blueprint"
	default:
		return "unknown"
	}
}
