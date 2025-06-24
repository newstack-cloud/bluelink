package validation

import (
	"context"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateCustomVariable validates a custom variable in a blueprint.
// This validation spans all the fields of a variable in the parsed schema
// as well as the runtime variable value provided by the user.
func ValidateCustomVariable(
	ctx context.Context,
	varName string,
	varSchema *schema.Variable,
	varMap *schema.VariableMap,
	params bpcore.BlueprintParams,
	customVariableType provider.CustomVariableType,
	validateRuntimeParams bool,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	optionLabels, err := validateCustomVariableOptions(
		ctx, varName, varSchema, varMap, params, customVariableType,
	)
	if err != nil {
		return diagnostics, err
	}

	// Values for custom variables must be the string labels for the options
	// provided by the custom type.
	if varSchema.Default != nil && varSchema.Default.StringValue == nil {
		return diagnostics, errCustomVariableInvalidDefaultValueType(
			varSchema.Type.Value,
			varName,
			varSchema.Default,
			getVarSourceMeta(varMap, varName),
		)
	}

	if varSchema.Default != nil && strings.TrimSpace(*varSchema.Default.StringValue) == "" {
		return diagnostics, errVariableEmptyDefaultValue(
			varSchema.Type.Value,
			varName,
			getVarSourceMeta(varMap, varName),
		)
	}

	if varSchema.Default != nil && !core.SliceContainsComparable(optionLabels, *varSchema.Default.StringValue) {
		return diagnostics, errCustomVariableDefaultValueNotInOptions(
			varSchema.Type.Value,
			varName,
			*varSchema.Default.StringValue,
			getVarSourceMeta(varMap, varName),
		)
	}

	userProvidedValue := params.BlueprintVariable(varName)
	finalValue := fallbackToDefault(userProvidedValue, varSchema.Default)

	if validateRuntimeParams && finalValue == nil {
		return diagnostics, errRequiredVariableMissing(varName, getVarSourceMeta(varMap, varName))
	}

	if validateRuntimeParams && finalValue.StringValue == nil {
		return diagnostics, errVariableInvalidOrMissing(
			varSchema.Type.Value,
			varName,
			finalValue,
			getVarSourceMeta(varMap, varName),
		)
	}

	if validateRuntimeParams && strings.TrimSpace(*finalValue.StringValue) == "" {
		return diagnostics, errVariableEmptyValue(
			varSchema.Type.Value,
			varName,
			getVarSourceMeta(varMap, varName),
		)
	}

	if validateRuntimeParams && !core.SliceContainsComparable(optionLabels, *finalValue.StringValue) {
		usingDefault := userProvidedValue == nil
		return diagnostics, errCustomVariableValueNotInOptions(
			varSchema.Type.Value,
			varName,
			finalValue,
			getVarSourceMeta(varMap, varName),
			usingDefault,
		)
	}

	if validateRuntimeParams && len(varSchema.AllowedValues) > 0 &&
		!bpcore.IsInScalarList(finalValue, varSchema.AllowedValues) {
		usingDefault := userProvidedValue == nil
		return diagnostics, errVariableValueNotAllowed(
			varSchema.Type.Value,
			varName,
			finalValue,
			varSchema.AllowedValues,
			getVarSourceMeta(varMap, varName),
			usingDefault,
		)
	}

	checkVarDescription(varName, varMap, varSchema.Description, &diagnostics)

	return diagnostics, nil
}

func validateCustomVariableOptions(
	ctx context.Context,
	varName string,
	varSchema *schema.Variable,
	varMap *schema.VariableMap,
	params bpcore.BlueprintParams,
	customVariableType provider.CustomVariableType,
) ([]string, error) {
	providerNamespace := provider.ExtractProviderFromItemType(string(varSchema.Type.Value))
	optionsOutput, err := customVariableType.Options(ctx, &provider.CustomVariableTypeOptionsInput{
		ProviderContext: provider.NewProviderContextFromParams(
			providerNamespace,
			params,
		),
	})
	if err != nil {
		return nil, errCustomVariableOptions(
			varName,
			varSchema,
			getVarSourceMeta(varMap, varName),
			err,
		)
	}

	optionsSlice := optionsMapToSlice(optionsOutput.Options)
	if hasMixedTypes(optionsSlice) {
		return nil, errCustomVariableMixedTypes(
			varName,
			varSchema,
			getVarSourceMeta(varMap, varName),
		)
	}

	optionLabels := keysToSlice(optionsOutput.Options)
	if len(varSchema.AllowedValues) > 0 {
		err := validateCustomVariableAllowedValues(
			ctx, varName, varSchema, varMap, params, optionLabels,
		)

		if err != nil {
			return nil, err
		}
	}

	return optionLabels, nil
}

func validateCustomVariableAllowedValues(
	ctx context.Context,
	varName string,
	varSchema *schema.Variable,
	varMap *schema.VariableMap,
	params bpcore.BlueprintParams,
	optionLabels []string,
) error {
	// Collect all invalid allowed values in one go to help
	// speed up the debugging process.
	invalidAllowedValueErrors := []error{}
	for _, allowedValue := range varSchema.AllowedValues {
		var err error
		if allowedValue == nil || scalarAllNil(allowedValue) {
			err = errVariableNullAllowedValue(
				varSchema.Type.Value,
				allowedValue,
				getVarSourceMeta(varMap, varName),
			)
		} else if allowedValue.StringValue == nil {
			err = errVariableInvalidAllowedValue(
				varSchema.Type.Value,
				allowedValue,
				getVarSourceMeta(varMap, varName),
			)
		}

		if err != nil {
			invalidAllowedValueErrors = append(invalidAllowedValueErrors, err)
		}
	}

	if len(invalidAllowedValueErrors) > 0 {
		return errVariableInvalidAllowedValues(
			varName,
			invalidAllowedValueErrors,
		)
	}

	invalidOptions := getInvalidOptions(varSchema.AllowedValues, optionLabels)
	if len(invalidOptions) > 0 {
		return errCustomVariableAllowedValuesNotInOptions(
			varSchema.Type.Value,
			varName,
			invalidOptions,
			getVarSourceMeta(varMap, varName),
		)
	}

	return nil
}

func getInvalidOptions(values []*bpcore.ScalarValue, optionLabels []string) []string {
	// Is is more important to reveal all invalid options as soon as possible to users
	// than to be efficient here, hence why we don't short circuit.
	invalidOptions := []string{}
	for _, value := range values {
		// Values with invalid types should have been caught before this point.
		if value.StringValue != nil && !core.SliceContainsComparable(optionLabels, *value.StringValue) {
			invalidOptions = append(invalidOptions, *value.StringValue)
		}
	}
	return invalidOptions
}

func hasMixedTypes(options []*bpcore.ScalarValue) bool {
	if len(options) == 0 {
		return false
	}

	currentType := (*schema.VariableType)(nil)
	hasMoreThanOneType := false
	i := 0
	for !hasMoreThanOneType && i < len(options) {
		varType := deriveVarType(options[i])
		if currentType != nil {
			if varType != *currentType {
				hasMoreThanOneType = true
			}
		} else {
			currentType = &varType
		}
		i += 1
	}

	return hasMoreThanOneType
}

func optionsMapToSlice(options map[string]*provider.CustomVariableTypeOption) []*bpcore.ScalarValue {
	result := make([]*bpcore.ScalarValue, 0, len(options))
	for _, option := range options {
		result = append(result, option.Value)
	}
	return result
}

// This could be useful as a general utility,
// might be worth moving out into the common package at some point.
func keysToSlice[Value any](mapping map[string]Value) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	return keys
}
