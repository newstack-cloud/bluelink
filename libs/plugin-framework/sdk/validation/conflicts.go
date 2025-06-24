package validation

import (
	"fmt"
	"regexp"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ConflictsWithPluginConfig returns a validation function that checks if a
// given field conflicts with a specified plugin configuration key.
func ConflictsWithPluginConfig(
	conflictsWithKey string,
) func(string, *core.ScalarValue, core.PluginConfig) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue, pluginConfig core.PluginConfig) []*core.Diagnostic {
		if _, hasConflictingKey := pluginConfig.Get(conflictsWithKey); hasConflictingKey {
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
) func(string, *core.MappingNode, *schema.Resource) []*core.Diagnostic {
	return func(fieldName string, value *core.MappingNode, resource *schema.Resource) []*core.Diagnostic {
		conflictingFieldValue, _ := core.GetPathValue(
			conflictsWithFieldPath,
			resource.Spec,
			core.MappingNodeMaxTraverseDepth,
		)

		if !core.IsNilMappingNode(conflictingFieldValue) {
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

		return nil
	}
}
