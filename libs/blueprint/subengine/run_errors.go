package subengine

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

const (
	// ErrorReasonCodeInvalidResolvedSubValue
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an substitution value that resolves to an invalid type.
	// For example, a substitution value that resolves to a
	// complex object or array when a type that can be cheaply converted
	// to a string is expected.
	ErrorReasonCodeInvalidResolvedSubValue errors.ErrorReasonCode = "invalid_resolved_sub_value"
	// ErrorReasonCodeInvalidSubstitutionValue
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an empty substitution value.
	ErrorReasonCodeEmptySubstitution errors.ErrorReasonCode = "empty_substitution"
	// ErrorReasonCodeMissingVariable
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing variable referenced in a substitution.
	ErrorReasonCodeMissingVariable errors.ErrorReasonCode = "missing_variable"
	// ErrorReasonCodeMissingValue
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing value referenced in a substitution.
	ErrorReasonCodeMissingValue errors.ErrorReasonCode = "missing_value"
	// ErrorReasonCodeMissingDataSource
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing data source referenced in a substitution.
	ErrorReasonCodeMissingDataSource errors.ErrorReasonCode = "missing_data_source"
	// ErrorReasonCodeEmptyDataSourceData
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// the result of fetching data from a data source being empty.
	ErrorReasonCodeEmptyDataSourceData errors.ErrorReasonCode = "empty_data_source_data"
	// ErrorReasonCodeMissingDataSourceProp
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing property in the data fetched for a
	// data source referenced in a substitution.
	ErrorReasonCodeMissingDataSourceProp errors.ErrorReasonCode = "missing_data_source_prop"
	// ErrorReasonCodeDataSourcePropNotArray
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a property in the data fetched for a data source
	// referenced in a substitution not being an array.
	ErrorReasonCodeDataSourcePropNotArray errors.ErrorReasonCode = "data_source_prop_not_array"
	// ErrorReasonCodeDataSourcePropArrayIndexOutOfBounds
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an index being out of bounds for an array property
	// in the data fetched for a data source referenced in a substitution.
	ErrorReasonCodeDataSourcePropArrayIndexOutOfBounds errors.ErrorReasonCode = "data_source_prop_array_index_out_of_bounds"
	// ErrorReasonCodeResourceNotResolved
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a resource not being resolved before use.
	ErrorReasonCodeResourceNotResolved errors.ErrorReasonCode = "resource_not_resolved"
	// ErrorReasonCodeResourceEachIndexOutOfBounds
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an index being out of bounds for a resource template
	// that is being used in a resource definition that is made a template
	// by the use of the `each` property.
	ErrorReasonCodeResourceEachIndexOutOfBounds errors.ErrorReasonCode = "resource_each_index_out_of_bounds"
	// ErrorReasonCodeResourceEachEmpty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// the `each` property of a resource template yielding an empty list.
	ErrorReasonCodeResourceEachEmpty errors.ErrorReasonCode = "resource_each_empty"
	// ErrorReasonCodeResourceEachInvalidType
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// the `each` property of a resource template yielding a value
	// that is not an array.
	ErrorReasonCodeResourceEachInvalidType errors.ErrorReasonCode = "resource_each_invalid_type"
	// ErrorReasonCodeMissingFunction
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing function in the registry.
	ErrorReasonCodeMissingFunction errors.ErrorReasonCode = "missing_function"
	// ErrorReasonCodeEmptyPositionalFunctionArgument
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an empty value being provided for a positional argument
	// in a function call.
	ErrorReasonCodeEmptyPositionalFunctionArgument errors.ErrorReasonCode = "empty_positional_function_argument"
	// ErrorReasonCodeEmptyNamedFunctionArgument
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an empty value being provided for a named argument
	// in a function call.
	ErrorReasonCodeEmptyNamedFunctionArgument errors.ErrorReasonCode = "empty_named_function_argument"
	// ErrorReasonCodeEmptyFunctionOutput
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a function call returning an empty output.
	ErrorReasonCodeEmptyFunctionOutput errors.ErrorReasonCode = "empty_function_output"
	// ErrorReasonCodeHigherOrderFunctionNotSupported
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a higher order function being used in a context where
	// it is not supported.
	ErrorReasonCodeHigherOrderFunctionNotSupported errors.ErrorReasonCode = "higher_order_function_not_supported"
	// ErrorReasonCodeResourceMissing
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a referenced resource not being present in the blueprint.
	ErrorReasonCodeResourceMissing errors.ErrorReasonCode = "resource_missing"
	// ErrorReasonCodeResourceSpecDefinitionMissing
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing spec definition for a resource.
	ErrorReasonCodeResourceSpecDefinitionMissing errors.ErrorReasonCode = "resource_spec_definition_missing"
	// ErrorReasonCodeInvalidResourceSpecDefinition
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an invalid spec definition for a resource.
	ErrorReasonCodeInvalidResourceSpecDefinition errors.ErrorReasonCode = "invalid_resource_spec_definition"
	// ErrorReasonCodeInvalidResourceSpecProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an unsupported property being referenced for a resource.
	ErrorReasonCodeInvalidResourceSpecProperty errors.ErrorReasonCode = "invalid_resource_spec_property"
	// ErrorReasonCodeMissingResourceSpecProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing property being referenced for a resource.
	ErrorReasonCodeMissingResourceSpecProperty errors.ErrorReasonCode = "missing_resource_spec_property"
	// ErrorReasonCodeInvalidResourceMetadataProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an unsupported property being referenced for the metadata
	// of a resource.
	ErrorReasonCodeInvalidResourceMetadataProperty errors.ErrorReasonCode = "invalid_resource_metadata_property"
	// ErrorReasonCodeMissingResourceMetadataProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing property being referenced for the metadata
	// of a resource.
	ErrorReasonCodeMissingResourceMetadataProperty errors.ErrorReasonCode = "missing_resource_metadata_property"
	// ErrorReasonCodeInvalidResourceMetadataNotSet
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a metadata property being referenced for a resource
	// that does not have any metadata set.
	ErrorReasonCodeInvalidResourceMetadataNotSet errors.ErrorReasonCode = "invalid_resource_metadata_not_set"
	// ErrorReasonCodeEmptyChildPath
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an empty child export path being provided for a child reference.
	ErrorReasonCodeEmptyChildPath errors.ErrorReasonCode = "empty_child_path"
	// ErrorReasonCodeMissingChildExport
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing export in a child reference.
	ErrorReasonCodeMissingChildExport errors.ErrorReasonCode = "missing_child_export"
	// ErrorReasonCodeMissingChildExportProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing property in the export data for a child reference.
	ErrorReasonCodeMissingChildExportProperty errors.ErrorReasonCode = "missing_child_export_property"
	// ErrorReasonCodeDisallowedElementType
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a referenced element type not beig a supported dependency for a given property.
	// For example, a resource being used as a dependency for the `each` property
	// of another resource.
	ErrorReasonCodeDisallowedElementType errors.ErrorReasonCode = "disallowed_element_type"
	// ErrorReasonCodeResourceNotTemplate
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a resource not being a template when a reference to one of
	// "elem" or "i" is used in a substitution.
	ErrorReasonCodeResourceNotTemplate errors.ErrorReasonCode = "resource_not_template"
	// ErrorReasonCodeMissingResourceTemplateInputElements
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// missing input elements for a resource template.
	// This error is expected to be returned when the stage changing process
	// is unable to resolve the list of input elements provided in the `each` property
	// for a resource template.
	ErrorReasonCodeMissingResourceTemplateInputElements errors.ErrorReasonCode = "missing_resource_template_input_elements"
	// ErrorReasonCodeMissingCurrentElementProperty
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing property in the current element reference.
	ErrorReasonCodeMissingCurrentElementProperty errors.ErrorReasonCode = "missing_current_element_property"
)

func errInvalidInterpolationSubType(elementName string, resolvedValue *core.MappingNode) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResolvedSubValue,
		Err: fmt.Errorf("[%s]: expected a string or primitive value that "+
			"can be converted to a string for an interpolation, got %v", elementName, determineValueType(resolvedValue)),
	}
}

func errEmptySubstitutionValue(elementName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptySubstitution,
		Err:        fmt.Errorf("[%s]: a string value or substitution value must be provided", elementName),
	}
}

func errMissingVariable(elementName string, variableName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingVariable,
		Err:        fmt.Errorf("[%s]: missing variable %q", elementName, variableName),
	}
}

func errMissingValue(elementName string, valueName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingValue,
		Err:        fmt.Errorf("[%s]: missing value %q", elementName, valueName),
	}
}

func errMissingDataSource(elementName string, dataSourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingDataSource,
		Err:        fmt.Errorf("[%s]: missing data source %q", elementName, dataSourceName),
	}
}

func errEmptyDataSourceData(elementName string, dataSourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyDataSourceData,
		Err:        fmt.Errorf("[%s]: data source %q returned no data", elementName, dataSourceName),
	}
}

func errMissingDataSourceProperty(elementName string, dataSourceName string, propertyName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingDataSourceProp,
		Err:        fmt.Errorf("[%s]: missing property %q in data source %q", elementName, propertyName, dataSourceName),
	}
}

func errDataSourcePropNotArray(elementName string, dataSourceName string, propertyName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDataSourcePropNotArray,
		Err:        fmt.Errorf("[%s]: property %q in data source %q is not an array", elementName, propertyName, dataSourceName),
	}
}

func errDataSourcePropArrayIndexOutOfBounds(elementName string, dataSourceName string, propertyName string, index int) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDataSourcePropArrayIndexOutOfBounds,
		Err:        fmt.Errorf("[%s]: index %d out of bounds for property %q in data source %q", elementName, index, propertyName, dataSourceName),
	}
}

func errResourceNotResolved(elementName string, resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceNotResolved,
		Err:        fmt.Errorf("[%s]: resource %q not resolved before use", elementName, resourceName),
	}
}

func errEmptyResourceEach(elementName string, resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceEachEmpty,
		Err: fmt.Errorf(
			"[%s]: resource template %q `each` property yields an empty list, it least one item must be in the list",
			elementName,
			resourceName,
		),
	}
}

func errResourceEachNotArray(elementName string, resourceName string, value *core.MappingNode) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceEachInvalidType,
		Err: fmt.Errorf(
			"[%s]: `each` property in "+
				"resource template %q must yield an array, %s found",
			elementName,
			resourceName,
			determineValueType(value),
		),
	}
}

func errDisallowedElementType(
	rootElementName string,
	rootElementProp string,
	referencedElementType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDisallowedElementType,
		Err: fmt.Errorf(
			"[%s]: element type %q can not be a dependency for the %q property, "+
				"a dependency can be either a direct or indirect reference to an element in a blueprint,"+
				" be sure to check the full trail of references",
			rootElementName,
			referencedElementType,
			rootElementProp,
		),
	}
}

func errMissingFunction(elementName string, functionName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingFunction,
		Err: fmt.Errorf(
			"[%s]: function not found, the function %q is not implemented by any of the loaded providers",
			elementName,
			functionName,
		),
	}
}

func errEmptyPositionalFunctionArgument(elementName string, functionName string, index int) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyPositionalFunctionArgument,
		Err: fmt.Errorf(
			"[%s]: a value must be provided for function argument at position %d in %q call",
			elementName,
			index,
			functionName,
		),
	}
}

func errEmptyNamedFunctionArgument(elementName string, functionName string, argumentName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyNamedFunctionArgument,
		Err: fmt.Errorf(
			"[%s]: a value must be provided for function argument %q in %q call",
			elementName,
			argumentName,
			functionName,
		),
	}
}

func errEmptyFunctionOutput(elementName string, functionName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyFunctionOutput,
		Err: fmt.Errorf(
			"[%s]: function %q returned an empty output",
			elementName,
			functionName,
		),
	}
}

func errHigherOrderFunctionNotSupported(elementName string, functionName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeHigherOrderFunctionNotSupported,
		Err: fmt.Errorf(
			"[%s]: higher order function %q can only be used as a function argument to functions like \"map\" or \"filter\", "+
				"this is because the function returns a partially applied function that needs to be executed "+
				"and not a value that can be resolved",
			elementName,
			functionName,
		),
	}
}

func errReferencedResourceMissing(elementName string, resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceMissing,
		Err: fmt.Errorf(
			"[%s]: referenced resource %q is missing, the resource must exist in the blueprint",
			elementName,
			resourceName,
		),
	}
}

func errMissingResourceSpecDefinition(elementName string, resourceName string, resourceType string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceSpecDefinitionMissing,
		Err: fmt.Errorf(
			"[%s]: missing or empty spec definition found for resource %q, "+
				"the provider plugin for the resource type %q should provide a spec definition, "+
				"you may need to update the plugin or contact the plugin developer",
			elementName,
			resourceName,
			resourceType,
		),
	}
}

func errResourceSpecMissingIDField(elementName string, resourceName string, resourceType string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResourceSpecDefinition,
		Err: fmt.Errorf(
			"[%s]: missing ID field in spec definition for resource %q, "+
				"the provider plugin for the resource should provide a spec definition with an ID field, "+
				"you may need to update the plugin or contact the plugin developer",
			elementName,
			resourceName,
		),
	}
}

func errInvalidResourcePropertyPath(
	elementName string,
	property *substitutions.SubstitutionResourceProperty,
) error {
	// Error is returned when a resource name is invalid, at this point,
	// the resource name is expected to be valid, given the blueprint has been successfully
	// loaded, it is safe to assume that the resource name is valid.
	path, _ := substitutions.SubResourcePropertyToString(property)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResolvedSubValue,
		Err:        fmt.Errorf("[%s]: invalid resource property path %q", elementName, path),
	}
}

func errInvalidResourceSpecProperty(
	elementName string,
	property *substitutions.SubstitutionResourceProperty,
	resourceType string,
) error {
	// Error is returned when a resource name is invalid, at this point,
	// the resource name is expected to be valid, given the blueprint has been successfully
	// loaded, it is safe to assume that the resource name is valid.
	path, _ := substitutions.SubResourcePropertyToString(property)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResourceSpecProperty,
		Err: fmt.Errorf(
			"[%s]: invalid property %q in spec definition for resource %q, "+
				"this property is not valid for resource type %q",
			elementName,
			path,
			property.ResourceName,
			resourceType,
		),
	}
}

func errMissingResourceSpecProperty(
	elementName string,
	property *substitutions.SubstitutionResourceProperty,
	// e.g. 1 for a mapping node defined in "spec"
	mappingNodeStartsAfter int,
	depth int,
	maxDepth int,
) error {
	depthWarning := ""
	if depth > maxDepth {
		depthWarning = fmt.Sprintf(
			", the depth of the property path is %d, resource spec properties "+
				" can not exceed a maximum depth of %d",
			depth+mappingNodeStartsAfter,
			maxDepth+mappingNodeStartsAfter,
		)
	}

	// Error is returned when a resource name is invalid, at this point,
	// the resource name is expected to be valid, given the blueprint has been successfully
	// loaded, it is safe to assume that the resource name is valid.
	path, _ := substitutions.SubResourcePropertyToString(property)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingResourceSpecProperty,
		Err: fmt.Errorf(
			"[%s]: missing property %q in spec definition for resource %q%s",
			elementName,
			path,
			property.ResourceName,
			depthWarning,
		),
	}
}

func errInvalidResourceMetadataProperty(
	elementName string,
	property *substitutions.SubstitutionResourceProperty,
) error {
	// Error is returned when a resource name is invalid, at this point,
	// the resource name is expected to be valid, given the blueprint has been successfully
	// loaded, it is safe to assume that the resource name is valid.
	path, _ := substitutions.SubResourcePropertyToString(property)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResourceMetadataProperty,
		Err: fmt.Errorf(
			"[%s]: invalid property %q in metadata for resource %q",
			elementName,
			property.ResourceName,
			path,
		),
	}
}

func errMissingResourceMetadataProperty(
	elementName string,
	property *substitutions.SubstitutionResourceProperty,
	// Offset to start counting the depth from.
	// For example, 2 for a mapping node defined in "metadata.custom".
	mappingNodeStartsAfter int,
	depth int,
	maxDepth int,
) error {
	// Error is returned when a resource name is invalid, at this point,
	// the resource name is expected to be valid, given the blueprint has been successfully
	// loaded, it is safe to assume that the resource name is valid.
	path, _ := substitutions.SubResourcePropertyToString(property)

	depthWarning := ""
	if depth > maxDepth {
		depthWarning = fmt.Sprintf(
			", the depth of the %q property path is %d, metadata "+
				"properties can not exceed a maximum depth of %d",
			path,
			depth+mappingNodeStartsAfter,
			maxDepth+mappingNodeStartsAfter,
		)
	}

	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingResourceMetadataProperty,
		Err: fmt.Errorf(
			"[%s]: missing property %q in metadata for resource %q%s",
			elementName,
			property.ResourceName,
			path,
			depthWarning,
		),
	}
}

func errMissingChildExportProperty(
	elementName string,
	property *substitutions.SubstitutionChild,
	// Offset to start counting the depth from.
	mappingNodeStartsAfter int,
	depth int,
	maxDepth int,
) error {
	fullPath, _ := substitutions.SubChildToString(property)
	propertyPath, _ := substitutions.PropertyPathToString(property.Path)

	depthWarning := ""
	if depth > maxDepth {
		depthWarning = fmt.Sprintf(
			", the depth of the %q property path is %d, child export "+
				"properties can not exceed a maximum depth of %d",
			fullPath,
			depth+mappingNodeStartsAfter,
			maxDepth+mappingNodeStartsAfter,
		)
	}

	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingChildExportProperty,
		Err: fmt.Errorf(
			"[%s]: missing property %q in export data for child %q%s",
			elementName,
			propertyPath,
			property.ChildName,
			depthWarning,
		),
	}
}

func errResourceMetadataNotSet(
	elementName string,
	resourceName string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidResourceMetadataNotSet,
		Err: fmt.Errorf(
			"[%s]: referenced resource metadata property does "+
				"not exist as metadata is not set for resource %q",
			elementName,
			resourceName,
		),
	}
}

func errEmptyChildPath(elementName string, childName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyChildPath,
		Err: fmt.Errorf(
			"[%s]: empty child path, a path to an exported value must be provided for the child %q",
			elementName,
			childName,
		),
	}
}

func errMissingChildExport(
	elementName string,
	childName string,
	childRefProp *substitutions.SubstitutionChild,
) error {
	exportPath, _ := substitutions.SubChildToString(childRefProp)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingChildExport,
		Err: fmt.Errorf(
			"[%s]: missing export in child %q referenced in path %q",
			elementName,
			childName,
			exportPath,
		),
	}
}

func errResourceNotTemplate(
	elementName string,
	resourceName string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceNotTemplate,
		Err: fmt.Errorf(
			"[%s]: the current resource %q is not a template, it must be made into a template using the each property "+
				"in order to reference the current element via \"elem\" or \"i\" in a substitution",
			elementName,
			resourceName,
		),
	}
}

func errMissingResourceTemplateInputElements(
	elementName string,
	resourceName string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingResourceTemplateInputElements,
		Err: fmt.Errorf(
			"[%s]: missing input elements for resource template %q, "+
				"the stage changing process seems to have not been able to resolve the "+
				"list of provided in the `each` property for the resource template",
			elementName,
			resourceName,
		),
	}
}

func errMissingCurrentElementProperty(
	elementName string,
	property *substitutions.SubstitutionElemReference,
	// Offset to start counting the depth from.
	mappingNodeStartsAfter int,
	depth int,
	maxDepth int,
) error {
	fullPath, _ := substitutions.SubElemToString(property)
	propertyPath, _ := substitutions.PropertyPathToString(property.Path)

	depthWarning := ""
	if depth > maxDepth {
		depthWarning = fmt.Sprintf(
			", the depth of the %q property path is %d, current element "+
				"properties can not exceed a maximum depth of %d",
			fullPath,
			depth+mappingNodeStartsAfter,
			maxDepth+mappingNodeStartsAfter,
		)
	}

	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingCurrentElementProperty,
		Err: fmt.Errorf(
			"[%s]: missing property %q in current element%s",
			elementName,
			propertyPath,
			depthWarning,
		),
	}
}

func determineValueType(resolvedValue *core.MappingNode) string {
	if resolvedValue == nil {
		return "null"
	}

	if resolvedValue.Scalar != nil {
		if resolvedValue.Scalar.StringValue != nil {
			return "string"
		}

		if resolvedValue.Scalar.IntValue != nil {
			return "int"
		}

		if resolvedValue.Scalar.FloatValue != nil {
			return "float"
		}

		if resolvedValue.Scalar.BoolValue != nil {
			return "bool"
		}
	}

	if resolvedValue.Fields != nil {
		return "object"
	}

	if resolvedValue.Items != nil {
		return "array"
	}

	// StringOrSubstitutions should not be set in a resolved value,
	// in the erroneous case where it is, we return the type as "null".

	return "null"
}
