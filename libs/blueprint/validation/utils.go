package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func deriveVarType(value *core.ScalarValue) schema.VariableType {
	if value != nil && value.IntValue != nil {
		return schema.VariableTypeInteger
	}

	if value != nil && value.FloatValue != nil {
		return schema.VariableTypeFloat
	}

	if value != nil && value.BoolValue != nil {
		return schema.VariableTypeBoolean
	}

	// This should only ever be used in a context where
	// the given scalar has a value, so string will always
	// be the default. BytesValue is also treated as a string
	// since it converts to UTF-8 string on serialization.
	return schema.VariableTypeString
}

func deriveScalarValueAsString(value *core.ScalarValue) string {
	if value != nil && value.IntValue != nil {
		return fmt.Sprintf("%d", *value.IntValue)
	}

	if value != nil && value.FloatValue != nil {
		return fmt.Sprintf("%.2f", *value.FloatValue)
	}

	if value != nil && value.BoolValue != nil {
		return fmt.Sprintf("%t", *value.BoolValue)
	}

	if value != nil && value.BytesValue != nil {
		return string(*value.BytesValue)
	}

	if value != nil && value.StringValue != nil {
		return *value.StringValue
	}

	return ""
}

func varTypeToUnit(varType schema.VariableType) string {
	switch varType {
	case schema.VariableTypeInteger:
		return "an integer"
	case schema.VariableTypeFloat:
		return "a float"
	case schema.VariableTypeBoolean:
		return "a boolean"
	case schema.VariableTypeString:
		return "a string"
	default:
		return "an unknown type"
	}
}

func isSubPrimitiveType(subType string) bool {
	switch substitutions.ResolvedSubExprType(subType) {
	case substitutions.ResolvedSubExprTypeString,
		substitutions.ResolvedSubExprTypeInteger,
		substitutions.ResolvedSubExprTypeFloat,
		substitutions.ResolvedSubExprTypeBoolean:
		return true
	default:
		return false
	}
}

func isEmptyStringWithSubstitutions(stringWithSubs *substitutions.StringOrSubstitutions) bool {
	if stringWithSubs == nil || stringWithSubs.Values == nil {
		return true
	}

	i := 0
	hasContent := false
	for !hasContent && i < len(stringWithSubs.Values) {
		if stringWithSubs.Values[i].SubstitutionValue != nil {
			hasContent = true
		} else {
			strVal := stringWithSubs.Values[i].StringValue
			hasContent = strVal != nil && strings.TrimSpace(*strVal) != ""
		}
		i += 1
	}

	return !hasContent
}

func validateDescription(
	ctx context.Context,
	usedIn string,
	usedInResourceDerivedFromTemplate bool,
	description *substitutions.StringOrSubstitutions,
	bpSchema *schema.Blueprint,
	params core.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if description == nil {
		return diagnostics, nil
	}

	errs := []error{}

	for _, stringOrSub := range description.Values {
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				bpSchema,
				usedInResourceDerivedFromTemplate,
				usedIn,
				"description",
				params,
				funcRegistry,
				refChainCollector,
				resourceRegistry,
				dataSourceRegistry,
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				if !isSubPrimitiveType(resolvedType) {
					errs = append(errs, errInvalidDescriptionSubType(
						usedIn,
						resolvedType,
						stringOrSub.SourceMeta,
					))
				}
			}
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func getSubNextLocation(i int, values []*substitutions.StringOrSubstitution) *source.Meta {
	if i+1 < len(values) {
		return values[i+1].SourceMeta
	}

	return nil
}

func getEndLocation(location *source.Meta) *source.Meta {
	if location == nil {
		return nil
	}

	if location.EndPosition != nil {
		return &source.Meta{Position: *location.EndPosition}
	}

	return &source.Meta{Position: source.Position{
		Line:   location.Line + 1,
		Column: location.Column,
	}}
}

func isMappingNodeEmpty(node *core.MappingNode) bool {
	return node == nil || (isEmptyScalar(node.Scalar) && node.Fields == nil &&
		node.Items == nil && node.StringWithSubstitutions == nil)
}

func isEmptyScalar(scalar *core.ScalarValue) bool {
	return scalar == nil || (scalar.StringValue == nil &&
		scalar.IntValue == nil &&
		scalar.BoolValue == nil &&
		scalar.FloatValue == nil &&
		scalar.BytesValue == nil)
}

func deriveMappingNodeResourceDefinitionsType(node *core.MappingNode) provider.ResourceDefinitionsSchemaType {
	if node.Scalar != nil && node.Scalar.BoolValue != nil {
		return provider.ResourceDefinitionsSchemaTypeBoolean
	}

	if node.Scalar != nil && node.Scalar.StringValue != nil {
		return provider.ResourceDefinitionsSchemaTypeString
	}

	if node.Scalar != nil && node.Scalar.IntValue != nil {
		return provider.ResourceDefinitionsSchemaTypeInteger
	}

	if node.Scalar != nil && node.Scalar.FloatValue != nil {
		return provider.ResourceDefinitionsSchemaTypeFloat
	}

	if node.Fields != nil {
		return provider.ResourceDefinitionsSchemaTypeObject
	}

	if node.Items != nil {
		return provider.ResourceDefinitionsSchemaTypeArray
	}

	if node.StringWithSubstitutions != nil {
		return provider.ResourceDefinitionsSchemaTypeString
	}

	return ""
}

func resourceDefinitionsUnionTypeToString(unionSchema []*provider.ResourceDefinitionsSchema) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i, schema := range unionSchema {
		sb.WriteString(string(schema.Type))
		if i < len(unionSchema)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

// CreateSubRefTag creates a reference chain node tag for a substitution reference.
func CreateSubRefTag(usedIn string) string {
	return fmt.Sprintf("subRef:%s", usedIn)
}

// CreateSubRefPropTag creates a reference chain node tag for a substitution reference
// including the property path within the resource that holds the reference.
func CreateSubRefPropTag(usedIn string, usedInPropPath string) string {
	return fmt.Sprintf("subRefProp:%s:%s", usedIn, usedInPropPath)
}

// CreateDependencyRefTag creates a reference chain node tag for a dependency reference
// defined in a blueprint resource with the "dependsOn" property.
func CreateDependencyRefTag(usedIn string) string {
	return fmt.Sprintf("dependencyOf:%s", usedIn)
}

// CreateLinkTag creates a reference chain node tag for a dependency resource
// that is linked to or from another resource.
// This should contain the name of the resource that depends on the resource being
// tagged.
func CreateLinkTag(linkDependencyOf string) string {
	return fmt.Sprintf("link:%s", linkDependencyOf)
}
