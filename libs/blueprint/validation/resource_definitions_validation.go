package validation

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ResourceValidationParams groups the non-context parameters used across validation functions
type ResourceValidationParams struct {
	ResourceName                string
	ResourceType                string
	ResourceDerivedFromTemplate bool
	*ValidationContext
}

func validateResourceDefinition(
	ctx context.Context,
	params ResourceValidationParams,
	spec *core.MappingNode,
	parentLocation *source.Meta,
	validateAgainstSchema *provider.ResourceDefinitionsSchema,
	path string,
	depth int,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}
	// Counting depth starts from 0.
	if depth >= core.MappingNodeMaxTraverseDepth {
		return diagnostics, nil
	}

	isEmpty := isMappingNodeEmpty(spec)
	if isEmpty && validateAgainstSchema.Nullable {
		return diagnostics, nil
	}

	if validateAgainstSchema.Computed {
		return diagnostics, errComputedFieldDefinedInBlueprint(
			path,
			params.ResourceName,
			selectMappingNodeLocation(spec, parentLocation),
		)
	}

	switch validateAgainstSchema.Type {
	case provider.ResourceDefinitionsSchemaTypeObject:
		return validateResourceDefinitionObject(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
			depth,
		)
	case provider.ResourceDefinitionsSchemaTypeMap:
		return validateResourceDefinitionMap(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
			depth,
		)
	case provider.ResourceDefinitionsSchemaTypeArray:
		return validateResourceDefinitionArray(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
			depth,
		)
	case provider.ResourceDefinitionsSchemaTypeString:
		return validateResourceDefinitionString(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
		)
	case provider.ResourceDefinitionsSchemaTypeInteger:
		return validateResourceDefinitionInteger(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
		)
	case provider.ResourceDefinitionsSchemaTypeFloat:
		return validateResourceDefinitionFloat(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
		)
	case provider.ResourceDefinitionsSchemaTypeBoolean:
		return validateResourceDefinitionBoolean(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
		)
	case provider.ResourceDefinitionsSchemaTypeUnion:
		return validateResourceDefinitionUnion(
			ctx,
			params,
			spec,
			parentLocation,
			validateAgainstSchema,
			path,
			depth,
		)
	default:
		return diagnostics, provider.ErrUnknownResourceDefSchemaType(
			validateAgainstSchema.Type,
			params.ResourceType,
		)
	}
}

func validateResourceDefinitionObject(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	validateAgainstSchema *provider.ResourceDefinitionsSchema,
	path string,
	depth int,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !validateAgainstSchema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeObject,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && validateAgainstSchema.Nullable {
		return diagnostics, nil
	}

	invalidType := node.Fields == nil
	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeObject,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	var errs []error

	for attrName, attrSchema := range validateAgainstSchema.Attributes {
		attrPath := fmt.Sprintf("%s.%s", path, attrName)
		attrNode, hasAttr := node.Fields[attrName]
		if !hasAttr {
			if slices.Contains(validateAgainstSchema.Required, attrName) {
				// For missing required fields, use parentLocation (the parent object's key)
				// rather than node.SourceMeta (which points to the first field in the object).
				// This provides better error positioning by highlighting the parent object
				// where the missing field should be added.
				errs = append(errs, errResourceDefMissingRequiredField(
					attrPath,
					params.ResourceType,
					attrName,
					attrSchema.Type,
					parentLocation,
				))
			}
		} else {
			// Use the field's key location as the parent for better error positioning
			attrParentLocation := selectFieldLocation(node.FieldsSourceMeta, attrName, parentLocation)
			attrDiagnostics, err := validateResourceDefinition(
				ctx,
				params,
				attrNode,
				attrParentLocation,
				attrSchema,
				attrPath,
				depth+1,
			)
			diagnostics = append(diagnostics, attrDiagnostics...)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	for fieldName := range node.Fields {
		fieldPath := fmt.Sprintf("%s.%s", path, fieldName)
		if _, hasAttr := validateAgainstSchema.Attributes[fieldName]; !hasAttr {
			availableFields := getAttributeNames(validateAgainstSchema.Attributes)
			// Use the field's key location for better error positioning
			fieldKeyLocation := selectFieldLocation(node.FieldsSourceMeta, fieldName, parentLocation)
			errs = append(errs, errResourceDefUnknownField(
				fieldPath,
				params.ResourceType,
				fieldName,
				availableFields,
				fieldKeyLocation,
			))
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateResourceDefinitionMap(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	validateAgainstSchema *provider.ResourceDefinitionsSchema,
	path string,
	depth int,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !validateAgainstSchema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeMap,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && validateAgainstSchema.Nullable {
		return diagnostics, nil
	}

	invalidType := node.Fields == nil
	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeMap,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if validateAgainstSchema.MinLength > 0 {
		minLengthDiagnostics, err := validateResourceDefinitionMapMinLength(
			node,
			validateAgainstSchema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, minLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if validateAgainstSchema.MaxLength > 0 {
		maxLengthDiagnostics, err := validateResourceDefinitionMapMaxLength(
			node,
			validateAgainstSchema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, maxLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	var errs []error

	for fieldName, fieldNode := range node.Fields {
		fieldPath := fmt.Sprintf("%s.%s", path, fieldName)
		// Use the field's key location as the parent for better error positioning
		fieldParentLocation := selectFieldLocation(node.FieldsSourceMeta, fieldName, parentLocation)
		fieldDiagnostics, err := validateResourceDefinition(
			ctx,
			params,
			fieldNode,
			fieldParentLocation,
			validateAgainstSchema.MapValues,
			fieldPath,
			depth+1,
		)
		diagnostics = append(diagnostics, fieldDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}

	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateResourceDefinitionArray(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	validateAgainstSchema *provider.ResourceDefinitionsSchema,
	path string,
	depth int,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !validateAgainstSchema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeArray,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && validateAgainstSchema.Nullable {
		return diagnostics, nil
	}

	invalidType := node.Items == nil
	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeArray,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if validateAgainstSchema.MinLength > 0 {
		minLengthDiagnostics, err := validateResourceDefinitionArrayMinLength(
			node,
			validateAgainstSchema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, minLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if validateAgainstSchema.MaxLength > 0 {
		maxLengthDiagnostics, err := validateResourceDefinitionArrayMaxLength(
			node,
			validateAgainstSchema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, maxLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	var errs []error

	for itemIndex, itemNode := range node.Items {
		itemPath := fmt.Sprintf("%s[%d]", path, itemIndex)
		// Use the item's own location as the parent for better error positioning
		itemParentLocation := selectMappingNodeLocation(itemNode, parentLocation)
		fieldDiagnostics, err := validateResourceDefinition(
			ctx,
			params,
			itemNode,
			itemParentLocation,
			validateAgainstSchema.Items,
			itemPath,
			depth+1,
		)
		diagnostics = append(diagnostics, fieldDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateResourceDefinitionString(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	schema *provider.ResourceDefinitionsSchema,
	path string,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !schema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeString,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && schema.Nullable {
		return diagnostics, nil
	}

	invalidType := (node.Scalar == nil ||
		(node.Scalar != nil && node.Scalar.StringValue == nil)) &&
		node.StringWithSubstitutions == nil

	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)
		if specType == "" {
			return diagnostics, errResourceDefItemEmpty(
				path,
				params.ResourceType,
				provider.ResourceDefinitionsSchemaTypeString,
				selectMappingNodeLocation(node, parentLocation),
			)
		}
		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeString,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if len(schema.AllowedValues) > 0 {
		allowedValueDiagnostics, err := validateResourceDefinitionAllowedValues(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, allowedValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.MinLength > 0 {
		minLengthDiagnostics, err := validateResourceDefinitionStringMinLength(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, minLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.MaxLength > 0 {
		maxLengthDiagnostics, err := validateResourceDefinitionStringMaxLength(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, maxLengthDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.Pattern != "" {
		patternDiagnostics, err := validateResourceDefinitionPattern(
			node,
			schema,
			params,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, patternDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.ValidateFunc != nil {
		customValidateDiagnostics, err := customValidateResourceDefinitionValue(
			node,
			path,
			schema,
			params.ResourceName,
			params.BpSchema,
		)
		diagnostics = append(diagnostics, customValidateDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if node.StringWithSubstitutions != nil {
		subDiagnostics, err := validateResourceDefinitionSubstitution(
			ctx,
			params,
			node.StringWithSubstitutions,
			substitutions.ResolvedSubExprTypeString,
			path,
		)
		diagnostics = append(diagnostics, subDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	return diagnostics, nil
}

func validateResourceDefinitionInteger(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	schema *provider.ResourceDefinitionsSchema,
	path string,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !schema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeInteger,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && schema.Nullable {
		return diagnostics, nil
	}

	invalidType := (node.Scalar == nil ||
		(node.Scalar != nil && node.Scalar.IntValue == nil)) &&
		node.StringWithSubstitutions == nil

	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)
		if specType == "" {
			return diagnostics, errResourceDefItemEmpty(
				path,
				params.ResourceType,
				provider.ResourceDefinitionsSchemaTypeInteger,
				selectMappingNodeLocation(node, parentLocation),
			)
		}

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeInteger,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if len(schema.AllowedValues) > 0 {
		allowedValueDiagnostics, err := validateResourceDefinitionAllowedValues(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, allowedValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if core.IsScalarInt(schema.Minimum) {
		minimumValueDiagnostics, err := validateResourceDefinitionMinIntValue(
			node,
			schema,
			params,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, minimumValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if core.IsScalarInt(schema.Maximum) {
		maximumValueDiagnostics, err := validateResourceDefinitionMaxIntValue(
			node,
			schema,
			params,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, maximumValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.ValidateFunc != nil {
		customValidateDiagnostics, err := customValidateResourceDefinitionValue(
			node,
			path,
			schema,
			params.ResourceName,
			params.BpSchema,
		)
		diagnostics = append(diagnostics, customValidateDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if node.StringWithSubstitutions != nil {
		subDiagnostics, err := validateResourceDefinitionSubstitution(
			ctx,
			params,
			node.StringWithSubstitutions,
			substitutions.ResolvedSubExprTypeInteger,
			path,
		)
		diagnostics = append(diagnostics, subDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	return diagnostics, nil
}

func validateResourceDefinitionFloat(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	schema *provider.ResourceDefinitionsSchema,
	path string,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !schema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeFloat,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && schema.Nullable {
		return diagnostics, nil
	}

	invalidType := (node.Scalar == nil ||
		(node.Scalar != nil && node.Scalar.FloatValue == nil)) &&
		node.StringWithSubstitutions == nil

	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)
		if specType == "" {
			return diagnostics, errResourceDefItemEmpty(
				path,
				params.ResourceType,
				provider.ResourceDefinitionsSchemaTypeFloat,
				selectMappingNodeLocation(node, parentLocation),
			)
		}

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeFloat,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if len(schema.AllowedValues) > 0 {
		allowedValueDiagnostics, err := validateResourceDefinitionAllowedValues(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, allowedValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if core.IsScalarFloat(schema.Minimum) {
		minimumValueDiagnostics, err := validateResourceDefinitionMinFloatValue(
			node,
			schema,
			params,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, minimumValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if core.IsScalarFloat(schema.Maximum) {
		maximumValueDiagnostics, err := validateResourceDefinitionMaxFloatValue(
			node,
			schema,
			params.ResourceType,
			path,
			selectMappingNodeLocation(node, parentLocation),
		)
		diagnostics = append(diagnostics, maximumValueDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if schema.ValidateFunc != nil {
		customValidateDiagnostics, err := customValidateResourceDefinitionValue(
			node,
			path,
			schema,
			params.ResourceName,
			params.BpSchema,
		)
		diagnostics = append(diagnostics, customValidateDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if node.StringWithSubstitutions != nil {
		subDiagnostics, err := validateResourceDefinitionSubstitution(
			ctx,
			params,
			node.StringWithSubstitutions,
			substitutions.ResolvedSubExprTypeFloat,
			path,
		)
		diagnostics = append(diagnostics, subDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	return diagnostics, nil
}

func validateResourceDefinitionBoolean(
	ctx context.Context,
	params ResourceValidationParams,
	node *core.MappingNode,
	parentLocation *source.Meta,
	schema *provider.ResourceDefinitionsSchema,
	path string,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	isEmpty := isMappingNodeEmpty(node)
	if isEmpty && !schema.Nullable {
		return diagnostics, errResourceDefItemEmpty(
			path,
			params.ResourceType,
			provider.ResourceDefinitionsSchemaTypeBoolean,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if isEmpty && schema.Nullable {
		return diagnostics, nil
	}

	invalidType := (node.Scalar == nil ||
		(node.Scalar != nil && node.Scalar.BoolValue == nil)) &&
		node.StringWithSubstitutions == nil

	if invalidType {
		specType := deriveMappingNodeResourceDefinitionsType(node)
		if specType == "" {
			return diagnostics, errResourceDefItemEmpty(
				path,
				params.ResourceType,
				provider.ResourceDefinitionsSchemaTypeBoolean,
				selectMappingNodeLocation(node, parentLocation),
			)
		}

		return diagnostics, errResourceDefInvalidType(
			path,
			params.ResourceType,
			specType,
			provider.ResourceDefinitionsSchemaTypeBoolean,
			selectMappingNodeLocation(node, parentLocation),
		)
	}

	if schema.ValidateFunc != nil {
		customValidateDiagnostics, err := customValidateResourceDefinitionValue(
			node,
			path,
			schema,
			params.ResourceName,
			params.BpSchema,
		)
		diagnostics = append(diagnostics, customValidateDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	if node.StringWithSubstitutions != nil {
		subDiagnostics, err := validateResourceDefinitionSubstitution(
			ctx,
			params,
			node.StringWithSubstitutions,
			substitutions.ResolvedSubExprTypeBoolean,
			path,
		)
		diagnostics = append(diagnostics, subDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	return diagnostics, nil
}

func validateResourceDefinitionUnion(
	ctx context.Context,
	params ResourceValidationParams,
	spec *core.MappingNode,
	parentLocation *source.Meta,
	validateAgainstSchema *provider.ResourceDefinitionsSchema,
	path string,
	depth int,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if isMappingNodeEmpty(spec) && !validateAgainstSchema.Nullable {
		return diagnostics, errResourceDefUnionItemEmpty(
			path,
			params.ResourceType,
			validateAgainstSchema.OneOf,
			selectMappingNodeLocation(spec, parentLocation),
		)
	}

	foundMatch := false
	i := 0
	for !foundMatch && i < len(validateAgainstSchema.OneOf) {
		unionSchema := validateAgainstSchema.OneOf[i]
		unionDiagnostics, err := validateResourceDefinition(
			ctx,
			params,
			spec,
			parentLocation,
			unionSchema,
			path,
			depth,
		)
		diagnostics = append(diagnostics, unionDiagnostics...)
		if err == nil {
			foundMatch = true
		}
		i += 1
	}

	if !foundMatch {
		return diagnostics, errResourceDefUnionInvalidType(
			path,
			params.ResourceType,
			validateAgainstSchema.OneOf,
			selectMappingNodeLocation(spec, parentLocation),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionSubstitution(
	ctx context.Context,
	params ResourceValidationParams,
	value *substitutions.StringOrSubstitutions,
	expectedResolvedType substitutions.ResolvedSubExprType,
	path string,
) ([]*core.Diagnostic, error) {
	if value == nil {
		return []*core.Diagnostic{}, nil
	}

	resourceIdentifier := core.ResourceElementID(params.ResourceName)
	errs := []error{}
	diagnostics := []*core.Diagnostic{}

	if len(value.Values) > 1 && expectedResolvedType != substitutions.ResolvedSubExprTypeString {
		return diagnostics, errInvalidResourceDefSubType(
			// StringOrSubstitutions with multiple values is an
			// interpolated string.
			string(substitutions.ResolvedSubExprTypeString),
			params.ResourceType,
			path,
			string(expectedResolvedType),
			value.SourceMeta,
		)
	}

	for _, stringOrSub := range value.Values {
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				&ValidationContext{
					BpSchema:           params.BpSchema,
					Params:             params.Params,
					FuncRegistry:       params.FuncRegistry,
					RefChainCollector:  params.RefChainCollector,
					ResourceRegistry:   params.ResourceRegistry,
					DataSourceRegistry: params.DataSourceRegistry,
				},
				params.ResourceDerivedFromTemplate,
				resourceIdentifier,
				path,
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				if resolvedType != string(expectedResolvedType) &&
					resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
					errs = append(errs, errInvalidResourceDefSubType(
						resolvedType,
						params.ResourceType,
						path,
						string(expectedResolvedType),
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

func validateResourceDefinitionPattern(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	params ResourceValidationParams,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if !core.IsScalarMappingNode(node) && node.StringWithSubstitutions != nil {
		// When a value is a string with substitutions,
		// we can not validate a value that is not yet resolved.
		// Warnings are useful to make practitioners aware of the possibility
		// of a failure during change staging or deployment for a field
		// that must match a specific pattern.
		diagnostics = append(
			diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of %q contains substitutions and can not be validated against a pattern. "+
						"When substitutions are resolved, this value must match the following pattern: %q.",
					path,
					schema.Pattern,
				),
				Range: core.DiagnosticRangeFromSourceMeta(location, nil),
			},
		)
		return diagnostics, nil
	}

	patternRegexp, err := regexp.Compile(schema.Pattern)
	if err != nil {
		return diagnostics, err
	}

	if !patternRegexp.MatchString(core.StringValue(node)) {
		return diagnostics, errResourceDefPatternConstraintFailure(
			path,
			params.ResourceType,
			schema.Pattern,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionMinIntValue(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	params ResourceValidationParams,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	return validateResourceDefinitionNumericConstraint(
		node,
		schema.Minimum,
		schema,
		params.ResourceType,
		path,
		selectMappingNodeLocation(node, location),
		func(value *core.MappingNode, constraint *core.ScalarValue) bool {
			return core.IntValue(value) < core.IntValueFromScalar(constraint)
		},
		"minimum",
		"greater than or equal to",
		errResourceDefMinConstraintFailure,
	)
}

func validateResourceDefinitionMaxIntValue(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	params ResourceValidationParams,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	return validateResourceDefinitionNumericConstraint(
		node,
		schema.Maximum,
		schema,
		params.ResourceType,
		path,
		selectMappingNodeLocation(node, location),
		func(value *core.MappingNode, constraint *core.ScalarValue) bool {
			return core.IntValue(value) > core.IntValueFromScalar(constraint)
		},
		"maximum",
		"less than or equal to",
		errResourceDefMaxConstraintFailure,
	)
}

func validateResourceDefinitionMinFloatValue(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	params ResourceValidationParams,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	return validateResourceDefinitionNumericConstraint(
		node,
		schema.Minimum,
		schema,
		params.ResourceType,
		path,
		selectMappingNodeLocation(node, location),
		func(value *core.MappingNode, constraint *core.ScalarValue) bool {
			return core.FloatValue(value) < core.FloatValueFromScalar(constraint)
		},
		"minimum",
		"greater than or equal to",
		errResourceDefMinConstraintFailure,
	)
}

func validateResourceDefinitionMaxFloatValue(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	return validateResourceDefinitionNumericConstraint(
		node,
		schema.Maximum,
		schema,
		resourceType,
		path,
		selectMappingNodeLocation(node, location),
		func(value *core.MappingNode, constraint *core.ScalarValue) bool {
			return core.FloatValue(value) > core.FloatValueFromScalar(constraint)
		},
		"maximum",
		"less than or equal to",
		errResourceDefMaxConstraintFailure,
	)
}

func validateResourceDefinitionNumericConstraint(
	node *core.MappingNode,
	constraint *core.ScalarValue,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
	failsConstraint func(value *core.MappingNode, constraint *core.ScalarValue) bool,
	constraintName string,
	constraintText string,
	errFunc func(
		path string,
		resourceType string,
		value *core.ScalarValue,
		constraint *core.ScalarValue,
		location *source.Meta,
	) error,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if !core.IsScalarMappingNode(node) && node.StringWithSubstitutions != nil {
		// Interpolated strings will be resolved as strings,
		// an interpolated string is one that contains a combination of
		// strings and substitutions or has more than one substitution.
		if isInterpolatedString(node.StringWithSubstitutions) {
			return diagnostics, errResourceDefInvalidType(
				path,
				resourceType,
				deriveMappingNodeResourceDefinitionsType(node),
				schema.Type,
				selectMappingNodeLocation(node, location),
			)
		}

		// When a value is a string with substitutions,
		// we can not validate a value that is not yet resolved.
		// Warnings are useful to make practitioners aware of the possibility
		// of a failure during change staging or deployment for a field
		// that must meet a specific numeric constraint.
		diagnostics = append(
			diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of %q contains substitutions and can not be validated against a %s value. "+
						"When substitutions are resolved, this value must be %s %s.",
					path,
					constraintName,
					constraintText,
					constraint.ToString(),
				),
				Range: core.DiagnosticRangeFromSourceMeta(location, nil),
			},
		)
		return diagnostics, nil
	}

	if failsConstraint(node, constraint) {
		return diagnostics, errFunc(
			path,
			resourceType,
			node.Scalar,
			constraint,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

// A maximum number of allowed values to show in error and warning messages.
const maxShowAllowedValues = 5

func validateResourceDefinitionAllowedValues(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	allowedValuesText := createAllowedValuesText(
		schema.AllowedValues,
		maxShowAllowedValues,
		"schema definition",
	)
	if !core.IsScalarMappingNode(node) && node.StringWithSubstitutions != nil {
		if schema.Type != provider.ResourceDefinitionsSchemaTypeString &&
			// Interpolated strings will be resolved as strings,
			// an interpolated string is one that contains a combination of
			// strings and substitutions or has more than one substitution.
			isInterpolatedString(node.StringWithSubstitutions) {
			return diagnostics, errResourceDefInvalidType(
				path,
				resourceType,
				deriveMappingNodeResourceDefinitionsType(node),
				schema.Type,
				selectMappingNodeLocation(node, location),
			)
		}

		// When a value is a string with substitutions and the field schema is a string,
		// we can not validate a value that is not yet resolved.
		// Warnings are useful to make practitioners aware of the possibility
		// of a failure during change staging or deployment for a field
		// that must be one of a fixed set of values.
		diagnostics = append(
			diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of %q contains substitutions and can not be validated against the allowed values. "+
						"When substitutions are resolved, this value must match one of the allowed values: %s",
					path,
					allowedValuesText,
				),
				Range: core.DiagnosticRangeFromSourceMeta(location, nil),
			},
		)
		return diagnostics, nil
	}

	inAllowedList := slices.ContainsFunc(
		schema.AllowedValues,
		func(allowedValue *core.MappingNode) bool {
			return core.IsScalarMappingNode(node) &&
				core.IsScalarMappingNode(allowedValue) &&
				node.Scalar.Equal(allowedValue.Scalar)
		},
	)

	if !inAllowedList {
		return diagnostics, errResourceDefNotAllowedValue(
			path,
			resourceType,
			allowedValuesText,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func createAllowedValuesText(allowedValues []*core.MappingNode, maxCount int, definitionLabel string) string {
	if len(allowedValues) <= maxCount {
		return mappingNodesToCommaSeparatedString(allowedValues)
	}

	// Show only the first `maxCount` allowed values.
	allowedValuesStr := mappingNodesToCommaSeparatedString(allowedValues[:maxCount])
	return fmt.Sprintf("%s, and %d more, see the %s for the full list",
		allowedValuesStr,
		len(allowedValues)-maxCount,
		definitionLabel,
	)
}

func mappingNodesToCommaSeparatedString(nodes []*core.MappingNode) string {
	values := make([]string, len(nodes))
	for i, node := range nodes {
		if core.IsScalarMappingNode(node) {
			values[i] = node.Scalar.ToString()
		} else {
			values[i] = "<unknown>"
		}
	}
	return strings.Join(values, ", ")
}

func isInterpolatedString(value *substitutions.StringOrSubstitutions) bool {
	return !substitutions.IsNilStringSubs(value) &&
		(len(value.Values) > 1 || value.Values[0].StringValue != nil)
}

func validateResourceDefinitionMapMinLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if len(node.Fields) < schema.MinLength {
		return diagnostics, errResourceDefComplexMinLengthConstraintFailure(
			path,
			resourceType,
			provider.ResourceDefinitionsSchemaTypeMap,
			len(node.Fields),
			schema.MinLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionMapMaxLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if len(node.Fields) > schema.MaxLength {
		return diagnostics, errResourceDefComplexMaxLengthConstraintFailure(
			path,
			resourceType,
			provider.ResourceDefinitionsSchemaTypeMap,
			len(node.Fields),
			schema.MaxLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionArrayMinLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if len(node.Items) < schema.MinLength {
		return diagnostics, errResourceDefComplexMinLengthConstraintFailure(
			path,
			resourceType,
			provider.ResourceDefinitionsSchemaTypeArray,
			len(node.Items),
			schema.MinLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionArrayMaxLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if len(node.Items) > schema.MaxLength {
		return diagnostics, errResourceDefComplexMaxLengthConstraintFailure(
			path,
			resourceType,
			provider.ResourceDefinitionsSchemaTypeArray,
			len(node.Items),
			schema.MaxLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionStringMinLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if !core.IsScalarMappingNode(node) && node.StringWithSubstitutions != nil {
		// When a value is a string with substitutions,
		// we can not validate a value that is not yet resolved.
		// Warnings are useful to make practitioners aware of the possibility
		// of a failure during change staging or deployment for a field
		// that must be greater than or equal to a specific length.
		diagnostics = append(
			diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of %q contains substitutions and can not be validated against a minimum length. "+
						"When substitutions are resolved, this value must have %d or more characters.",
					path,
					schema.MinLength,
				),
				Range: core.DiagnosticRangeFromSourceMeta(location, nil),
			},
		)
		return diagnostics, nil
	}

	numberOfChars := utf8.RuneCountInString(core.StringValue(node))
	if numberOfChars < schema.MinLength {
		return diagnostics, errResourceDefStringMinLengthConstraintFailure(
			path,
			resourceType,
			numberOfChars,
			schema.MinLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func validateResourceDefinitionStringMaxLength(
	node *core.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
	resourceType string,
	path string,
	location *source.Meta,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	if !core.IsScalarMappingNode(node) && node.StringWithSubstitutions != nil {
		// When a value is a string with substitutions,
		// we can not validate a value that is not yet resolved.
		// Warnings are useful to make practitioners aware of the possibility
		// of a failure during change staging or deployment for a field
		// that must be less than or equal to a specific length.
		diagnostics = append(
			diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of %q contains substitutions and can not be validated against a maximum length. "+
						"When substitutions are resolved, this value must have %d or less characters.",
					path,
					schema.MaxLength,
				),
				Range: core.DiagnosticRangeFromSourceMeta(location, nil),
			},
		)
		return diagnostics, nil
	}

	numberOfChars := utf8.RuneCountInString(core.StringValue(node))
	if numberOfChars > schema.MaxLength {
		return diagnostics, errResourceDefStringMaxLengthConstraintFailure(
			path,
			resourceType,
			numberOfChars,
			schema.MaxLength,
			selectMappingNodeLocation(node, location),
		)
	}

	return diagnostics, nil
}

func customValidateResourceDefinitionValue(
	node *core.MappingNode,
	path string,
	schema *provider.ResourceDefinitionsSchema,
	resourceName string,
	bpSchema *schema.Blueprint,
) ([]*core.Diagnostic, error) {
	if node.StringWithSubstitutions != nil {
		// Custom validation functions can not be applied to
		// strings with substitutions, as the values are not resolved yet.
		return []*core.Diagnostic{}, nil
	}

	blueprintResource, _ := getResource(resourceName, bpSchema)
	diagnostics := schema.ValidateFunc(
		path,
		node,
		blueprintResource,
	)
	// Custom validation functions return a slice of diagnostics
	// containing errors, warnings and info messages.
	// For this reason, we need to extract diagnostics and errors
	// to be consistent with the rest of the validation process
	// where diagnostics are separated from errors.
	return ExtractDiagnosticsAndErrors(
		diagnostics,
		ErrorReasonCodeInvalidResource,
	)
}

func selectMappingNodeLocation(node *core.MappingNode, parentLocation *source.Meta) *source.Meta {
	if node != nil && node.SourceMeta != nil {
		return node.SourceMeta
	}

	return parentLocation
}

// selectFieldLocation returns the source location for a field name from FieldsSourceMeta,
// or falls back to parentLocation if not available. This provides better error positioning
// by using the field's key location rather than its value location.
func selectFieldLocation(
	fieldsSourceMeta map[string]*source.Meta,
	fieldName string,
	parentLocation *source.Meta,
) *source.Meta {
	if fieldsSourceMeta != nil {
		if fieldMeta, ok := fieldsSourceMeta[fieldName]; ok && fieldMeta != nil {
			return fieldMeta
		}
	}
	return parentLocation
}

// getAttributeNames returns a sorted slice of attribute names from a schema attributes map.
func getAttributeNames(attributes map[string]*provider.ResourceDefinitionsSchema) []string {
	if len(attributes) == 0 {
		return nil
	}
	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	return names
}
