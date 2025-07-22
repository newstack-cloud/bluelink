package validation

import (
	"fmt"
	"regexp"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ConflictOptions defines additional options for field conflict validation.
type ConflictOptions struct {
	// ConflictsOnValue is the value in an another field that this field
	// conflicts with.
	ConflictsOnValue *core.ScalarValue
}

// ConflictsWithPluginConfig returns a validation function that checks if a
// given field conflicts with a specified plugin configuration key.
func ConflictsWithPluginConfig(
	conflictsWithKey string,
	conflictOptions *ConflictOptions,
) func(string, *core.ScalarValue, core.PluginConfig) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue, pluginConfig core.PluginConfig) []*core.Diagnostic {
		conflictOnValue := getConflictOnValue(conflictOptions)
		conflictingValue, hasConflictingKey := pluginConfig.Get(conflictsWithKey)
		if hasConflictingKey && core.IsScalarNil(conflictOnValue) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q cannot be set because it conflicts with the plugin configuration key %q.",
						fieldName,
						conflictsWithKey,
					),
				},
			}
		}

		if hasConflictingKey && conflictingValue.Equal(conflictOnValue) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q cannot be set when the %q plugin config key has a value of %q.",
						fieldName,
						conflictsWithKey,
						core.StringValueFromScalar(conflictingValue),
					),
				},
			}
		}

		return nil
	}
}

var (
	pathPrefixPattern = regexp.MustCompile(`^\$\.?`)
)

// ConflictsWithResourceDefinition returns a validation function that checks if a
// given field conflicts with a specified resource spec field path.
// The path notation used in the blueprint framework's `core.GetPathValue`
// package should be used to specify the path to the conflicting field, such as
// $[\"cluster.v1\"].config.endpoints[0], where "$" is the root of the resource
// `spec` field.
func ConflictsWithResourceDefinition(
	conflictsWithFieldPath string,
	conflictOptions *ConflictOptions,
) func(string, *core.MappingNode, *schema.Resource) []*core.Diagnostic {
	return func(fieldName string, value *core.MappingNode, resource *schema.Resource) []*core.Diagnostic {
		conflictingFieldValue, _ := core.GetPathValue(
			conflictsWithFieldPath,
			resource.Spec,
			core.MappingNodeMaxTraverseDepth,
		)
		conflictOnValue := getConflictOnValue(conflictOptions)

		if !core.IsNilMappingNode(conflictingFieldValue) && conflictOnValue == nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q cannot be set because it conflicts with the resource spec field %q.",
						fieldName,
						pathPrefixPattern.ReplaceAllString(
							conflictsWithFieldPath,
							"",
						),
					),
				},
			}
		}

		conflictOnValueMappingNode := &core.MappingNode{
			Scalar: conflictOnValue,
		}
		if !core.IsNilMappingNode(conflictingFieldValue) &&
			core.MappingNodeEqual(conflictOnValueMappingNode, conflictingFieldValue) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q cannot be set when the resource spec field %q has a value of %q.",
						fieldName,
						pathPrefixPattern.ReplaceAllString(
							conflictsWithFieldPath,
							"",
						),
						core.StringValue(conflictingFieldValue),
					),
				},
			}
		}

		return nil
	}
}

func getConflictOnValue(conflictOptions *ConflictOptions) *core.ScalarValue {
	if conflictOptions == nil {
		return nil
	}

	return conflictOptions.ConflictsOnValue
}
