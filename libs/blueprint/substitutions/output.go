package substitutions

import (
	"fmt"
	"strings"
)

// SubstitutionsToString converts a representation of a string as a sequence
// of string literals and interpolated substitutions to a string.
//
// An example output of this function would be:
//
//	"GetOrderFunction-${variables.env}-${variables.version}"
func SubstitutionsToString(substitutionContext string, substitutions *StringOrSubstitutions) (string, error) {
	var b strings.Builder
	// Validation of substitutions as per the spec is done at the time of serialisation/deserialisation
	// primarily to be as efficient as possible.
	subErrors := []error{}
	for _, value := range substitutions.Values {
		err := writeStringOrSubstitution(substitutionContext, &b, value)
		if err != nil {
			subErrors = append(subErrors, err)
		}
	}

	if len(subErrors) > 0 {
		return "", errSerialiseSubstitutions(substitutionContext, subErrors)
	}

	return b.String(), nil
}

func writeStringOrSubstitution(substitutionContext string, b *strings.Builder, value *StringOrSubstitution) error {
	if value.StringValue != nil {
		b.WriteString(*value.StringValue)
	} else {
		substitutionStr, err := SubstitutionToString(substitutionContext, value.SubstitutionValue)
		if err != nil {
			return err
		}
		b.WriteString("${")
		b.WriteString(substitutionStr)
		b.WriteString("}")

	}
	return nil
}

// SubstitutionToString converts a representation of a substitution with the ${..} syntax
// to a string.
func SubstitutionToString(substitutionContext string, substitution *Substitution) (string, error) {
	if substitution.Function != nil {
		return subFunctionToString(substitutionContext, substitution.Function)
	} else if substitution.Variable != nil {
		return subVariableToString(substitution.Variable)
	} else if substitution.ValueReference != nil {
		return subValueRefToString(substitution.ValueReference)
	} else if substitution.DataSourceProperty != nil {
		return subDataSourcePropertyToString(substitution.DataSourceProperty)
	} else if substitution.ResourceProperty != nil {
		return SubResourcePropertyToString(substitution.ResourceProperty)
	} else if substitution.Child != nil {
		return SubChildToString(substitution.Child)
	} else if substitution.ElemReference != nil {
		return SubElemToString(substitution.ElemReference)
	} else if substitution.ElemIndexReference != nil {
		return "i", nil
	}
	return "", nil
}

func subFunctionToString(substitutionContext string, function *SubstitutionFunctionExpr) (string, error) {
	var b strings.Builder

	b.WriteString(string(function.FunctionName))
	b.WriteString("(")

	subErrors := []error{}
	for i, arg := range function.Arguments {
		err := writeFunctionArgument(substitutionContext, &b, arg)
		if err != nil {
			subErrors = append(subErrors, err)
		}

		if i < len(function.Arguments)-1 {
			b.WriteString(",")
		}
	}

	if len(subErrors) > 0 {
		return "", errSerialiseSubstitutions(substitutionContext, subErrors)
	}

	b.WriteString(")")
	return b.String(), nil
}

func writeFunctionArgument(substitutionContext string, b *strings.Builder, arg *SubstitutionFunctionArg) error {
	if arg.Value == nil {
		return errSerialiseSubstitutionFunctionArgValueMissing()
	}

	if arg.Name != "" {
		b.WriteString(fmt.Sprintf("%s = ", arg.Name))
	}

	if arg.Value.StringValue != nil {
		// String literals in the context of a function call
		// are always wrapped in double quotes.
		b.WriteString(fmt.Sprintf("\"%s\"", *arg.Value.StringValue))
	} else if arg.Value.IntValue != nil {
		b.WriteString(fmt.Sprintf("%d", *arg.Value.IntValue))
	} else if arg.Value.FloatValue != nil {
		b.WriteString(fmt.Sprintf("%f", *arg.Value.FloatValue))
	} else if arg.Value.BoolValue != nil {
		b.WriteString(fmt.Sprintf("%t", *arg.Value.BoolValue))
	} else {
		substitutionStr, err := SubstitutionToString(substitutionContext, arg.Value)
		if err != nil {
			return err
		}
		b.WriteString(substitutionStr)
	}

	return nil
}

func subVariableToString(variable *SubstitutionVariable) (string, error) {
	if NamePattern.MatchString(variable.VariableName) {
		return fmt.Sprintf("variables.%s", variable.VariableName), nil
	}

	if NameStringLiteralPattern.MatchString(variable.VariableName) {
		return fmt.Sprintf("variables[\"%s\"]", variable.VariableName), nil
	}

	return "", errSerialiseSubstitutionInvalidVariableName(variable.VariableName)
}

func subValueRefToString(valueRef *SubstitutionValueReference) (string, error) {
	path := "values"
	if NamePattern.MatchString(valueRef.ValueName) {
		path += fmt.Sprintf(".%s", valueRef.ValueName)
	} else if NameStringLiteralPattern.MatchString(valueRef.ValueName) {
		path += fmt.Sprintf("[\"%s\"]", valueRef.ValueName)
	} else {
		return "", errSerialiseSubstitutionInvalidValueReferenceName(valueRef.ValueName)
	}

	return propertyPathToString(path, valueRef.Path, errSerialiseSubstitutionInvalidPath)
}

func subDataSourcePropertyToString(prop *SubstitutionDataSourceProperty) (string, error) {
	path := "datasources"
	if NamePattern.MatchString(prop.DataSourceName) {
		path += fmt.Sprintf(".%s", prop.DataSourceName)
	} else if NameStringLiteralPattern.MatchString(prop.DataSourceName) {
		path += fmt.Sprintf("[\"%s\"]", prop.DataSourceName)
	} else {
		return "", errSerialiseSubstitutionInvalidDataSourceName(prop.DataSourceName)
	}

	if NamePattern.MatchString(prop.FieldName) {
		path += fmt.Sprintf(".%s", prop.FieldName)
	} else if NameStringLiteralPattern.MatchString(prop.FieldName) {
		path += fmt.Sprintf("[\"%s\"]", prop.FieldName)
	} else {
		return "", errSerialiseSubstitutionInvalidDataSourcePath(prop.FieldName, prop.DataSourceName)
	}

	if prop.PrimitiveArrIndex != nil {
		path += fmt.Sprintf("[%d]", *prop.PrimitiveArrIndex)
	}

	return path, nil
}

// SubResourcePropertyToString produces a string representation of a substitution
// component that refers to a resource property.
func SubResourcePropertyToString(prop *SubstitutionResourceProperty) (string, error) {
	path := "resources"
	if NamePattern.MatchString(prop.ResourceName) {
		path += fmt.Sprintf(".%s", prop.ResourceName)
	} else if NameStringLiteralPattern.MatchString(prop.ResourceName) {
		path += fmt.Sprintf("[\"%s\"]", prop.ResourceName)
	} else {
		return "", errSerialiseSubstitutionInvalidResourceName(prop.ResourceName)
	}

	return propertyPathToString(path, prop.Path, errSerialiseSubstitutionInvalidPath)
}

func propertyPathToString(
	base string,
	propPath []*SubstitutionPathItem,
	errFunc func(string, string, []error) error,
) (string, error) {
	errors := []error{}
	var path strings.Builder
	path.WriteString(base)
	var rawPath strings.Builder
	for _, pathItem := range propPath {
		pathItemStr, err := propertyPathItemToString(pathItem)
		if err != nil {
			errors = append(errors, err)
		} else {
			path.WriteString(pathItemStr)
		}
		rawPath.WriteString(pathItemStr)
	}

	if len(errors) > 0 {
		return "", errFunc(rawPath.String(), base, errors)
	}

	return strings.TrimPrefix(path.String(), "."), nil
}

func propertyPathItemToString(pathItem *SubstitutionPathItem) (string, error) {
	if NamePattern.MatchString(pathItem.FieldName) {
		return fmt.Sprintf(".%s", pathItem.FieldName), nil
	} else if NameStringLiteralPattern.MatchString(pathItem.FieldName) {
		return fmt.Sprintf("[\"%s\"]", pathItem.FieldName), nil
	} else if pathItem.ArrayIndex != nil {
		return fmt.Sprintf("[%d]", *pathItem.ArrayIndex), nil
	}

	// Return the raw path item string so it can be used in higher level error messages.
	return fmt.Sprintf("[\"%s\"]", pathItem.FieldName), errSerialiseSubstitutionInvalidPathItem(pathItem)
}

// SubChildToString produces a string representation of a substitution
// component that refers to a child blueprint export.
func SubChildToString(child *SubstitutionChild) (string, error) {
	var path strings.Builder
	path.WriteString("children")
	if NamePattern.MatchString(child.ChildName) {
		fmt.Fprintf(&path, ".%s", child.ChildName)
	} else if NameStringLiteralPattern.MatchString(child.ChildName) {
		fmt.Fprintf(&path, "[\"%s\"]", child.ChildName)
	} else {
		return "", errSerialiseSubstitutionInvalidChildName(child.ChildName)
	}

	if len(child.Path) == 0 {
		return "", errSerialiseSubstitutionInvalidChildPath("", child.ChildName, []error{})
	}

	return propertyPathToString(
		path.String(),
		child.Path,
		errSerialiseSubstitutionInvalidChildPath,
	)
}

// SubElemToString produces a string representation of a substitution
// component that refers to the current element in an input array for
// a resource template.
func SubElemToString(elem *SubstitutionElemReference) (string, error) {
	var path strings.Builder
	path.WriteString("elem")

	return propertyPathToString(
		path.String(),
		elem.Path,
		errSerialiseSubstitutionInvalidCurrentElementPath,
	)
}

// PropertyPathToString converts a property path to a string.
func PropertyPathToString(path []*SubstitutionPathItem) (string, error) {
	return propertyPathToString("", path, errSerialiseSubstitutionInvalidPath)
}
