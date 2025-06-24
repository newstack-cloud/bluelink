package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// WrapForPluginConfig wraps a general validation function
// so it can be used with plugin config definition fields.
func WrapForPluginConfig(
	validationFunc func(string, *core.ScalarValue) []*core.Diagnostic,
) func(string, *core.ScalarValue, core.PluginConfig) []*core.Diagnostic {
	return func(
		fieldName string,
		value *core.ScalarValue,
		_ core.PluginConfig,
	) []*core.Diagnostic {
		return validationFunc(fieldName, value)
	}
}

// WrapForResourceDefinition wraps a general validation function
// so it can be used with blueprint resource definition schema fields.
// The validation helpers package provides helpers for scalar values only,
// this should not be used with complex resource definition
// schema types such as arrays, objects, or mappings.
func WrapForResourceDefinition(
	validationFunc func(string, *core.ScalarValue) []*core.Diagnostic,
) func(string, *core.MappingNode, *schema.Resource) []*core.Diagnostic {
	return func(
		fieldName string,
		value *core.MappingNode,
		// The full resource definition
		// in a blueprint, which is used for conditional validation based on
		// other parts of a resource.
		_ *schema.Resource,
	) []*core.Diagnostic {
		if core.IsScalarMappingNode(value) {
			return validationFunc(fieldName, value.Scalar)
		}

		return []*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"%s is not a valid type for the configured validator, expected a scalar "+
						"(string, integer, float or boolean), but got %s.",
					fieldName,
					typeFromMappingNode(value),
				),
			},
		}
	}
}
