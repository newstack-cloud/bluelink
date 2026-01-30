package validation

import (
	"context"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateValueName checks the validity of a value name,
// primarily making sure that it does not contain any substitutions
// as per the spec.
func ValidateValueName(mappingName string, valMap *schema.ValueMap) error {
	if substitutions.ContainsSubstitution(mappingName) {
		return errMappingNameContainsSubstitution(
			mappingName,
			"value",
			ErrorReasonCodeInvalidValue,
			getValSourceMeta(valMap, mappingName),
		)
	}
	return nil
}

// ValidateValue deals with validating a blueprint value
// against the supported value types in the blueprint
// specification.
func ValidateValue(
	ctx context.Context,
	valName string,
	valSchema *schema.Value,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	valueTypeDiagnostics, err := validateValueType(
		valName,
		valSchema,
	)
	diagnostics = append(diagnostics, valueTypeDiagnostics...)
	if err != nil {
		return diagnostics, err
	}

	expectedResolveType := subValType(valSchema.Type)

	return validateValue(
		ctx,
		valName,
		valSchema,
		expectedResolveType,
		valCtx,
	)
}

func validateValue(
	ctx context.Context,
	valName string,
	valSchema *schema.Value,
	expectedResolveType string,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	descriptionDiagnostics, err := validateValueDescription(
		ctx,
		valName,
		valSchema,
		valCtx,
	)
	diagnostics = append(diagnostics, descriptionDiagnostics...)
	if err != nil {
		return diagnostics, err
	}

	valueDiagnostics, err := validateValueContent(
		ctx,
		expectedResolveType,
		valName,
		valSchema,
		valCtx,
	)
	diagnostics = append(diagnostics, valueDiagnostics...)
	if err != nil {
		return valueDiagnostics, err
	}

	return diagnostics, nil
}

func validateValueDescription(
	ctx context.Context,
	valName string,
	valSchema *schema.Value,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	if valSchema.Description == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	valIdentifier := bpcore.ValueElementID(valName)
	return validateDescription(
		ctx,
		valIdentifier,
		/* usedInResourceDerivedFromTemplate */ false,
		valSchema.Description,
		valCtx,
	)
}

func validateValueContent(
	ctx context.Context,
	expectedResolveType string,
	valName string,
	valSchema *schema.Value,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if valSchema.Value == nil {
		return diagnostics, errMissingValueContent(valName, valSchema.SourceMeta)
	}

	valIdentifier := bpcore.ValueElementID(valName)

	// For string values with substitutions, we care about the resolved type
	// of substitutions, so we need to validate the string with substitutions
	// content type directly instead of through mapping node validation.
	if bpcore.IsStringWithSubsMappingNode(valSchema.Value) {
		return validateValueContentForStringWithSubs(
			ctx,
			valIdentifier,
			valSchema,
			valCtx,
			expectedResolveType,
		)
	}

	contentTypeDiags, err := validateValueContentType(
		valIdentifier,
		valSchema,
		expectedResolveType,
	)
	diagnostics = append(diagnostics, contentTypeDiags...)
	if err != nil {
		return diagnostics, err
	}

	mappingNodeDiags, err := ValidateMappingNode(
		ctx,
		valIdentifier,
		"value",
		/* usedInResourceDerivedFromTemplate */ false,
		valSchema.Value,
		valCtx,
	)
	diagnostics = append(diagnostics, mappingNodeDiags...)

	return diagnostics, err
}

func validateValueContentType(
	valIdentifier string,
	valSchema *schema.Value,
	expectedResolveType string,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	resolvedSubType := resolvedSubTypeFromMappingNode(valSchema.Value)
	if resolvedSubType != expectedResolveType {
		return diagnostics, errInvalidValueContentType(
			valIdentifier,
			resolvedSubType,
			expectedResolveType,
			valSchema.SourceMeta,
		)
	}

	return diagnostics, nil
}

func validateValueContentForStringWithSubs(
	ctx context.Context,
	valIdentifier string,
	valSchema *schema.Value,
	valCtx *ValidationContext,
	expectedResolveType string,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if len(valSchema.Value.StringWithSubstitutions.Values) > 1 &&
		// More than one value in a stringOrSubstitutions slice represents a string interpolation
		// which is only allowed for string values.
		expectedResolveType != string(substitutions.ResolvedSubExprTypeString) {
		return diagnostics, errValueIncorrectTypeInterpolatedString(
			valIdentifier,
			expectedResolveType,
			valSchema.SourceMeta,
		)
	}

	errs := []error{}

	for _, stringOrSub := range valSchema.Value.StringWithSubstitutions.Values {
		if stringOrSub.StringValue != nil {
			if expectedResolveType != string(substitutions.ResolvedSubExprTypeString) {
				errs = append(errs, errValueIncorrectTypeInterpolatedString(
					valIdentifier,
					expectedResolveType,
					stringOrSub.SourceMeta,
				))
			}
		}
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				valCtx,
				/* usedInResourceDerivedFromTemplate */ false,
				valIdentifier,
				"value",
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				if resolvedType != expectedResolveType &&
					// Allow any type to account for functions like jsondecode() that can return any type.
					// This means the user is responsible for ensuring the type of the value is correct.
					resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
					errs = append(errs, errInvalidValueSubType(
						valIdentifier,
						resolvedType,
						expectedResolveType,
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

func validateValueType(
	valName string,
	valSchema *schema.Value,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if valSchema.Type == nil || valSchema.Type.Value == "" {
		return diagnostics, errMissingValueType(valName, valSchema.SourceMeta)
	}

	if !core.SliceContains(schema.ValueTypes, valSchema.Type.Value) {
		return diagnostics, errInvalidValueType(
			valName,
			valSchema.Type,
			valSchema.SourceMeta,
		)
	}

	return diagnostics, nil
}

func getValSourceMeta(valMap *schema.ValueMap, varName string) *source.Meta {
	if valMap == nil {
		return nil
	}

	return valMap.SourceMeta[varName]
}

func resolvedSubTypeFromMappingNode(mappingNode *bpcore.MappingNode) string {
	if bpcore.IsScalarMappingNode(mappingNode) &&
		bpcore.IsScalarString(mappingNode.Scalar) {
		return string(substitutions.ResolvedSubExprTypeString)
	}

	if bpcore.IsScalarMappingNode(mappingNode) &&
		bpcore.IsScalarInt(mappingNode.Scalar) {
		return string(substitutions.ResolvedSubExprTypeInteger)
	}

	if bpcore.IsScalarMappingNode(mappingNode) &&
		bpcore.IsScalarFloat(mappingNode.Scalar) {
		return string(substitutions.ResolvedSubExprTypeFloat)
	}

	if bpcore.IsScalarMappingNode(mappingNode) &&
		bpcore.IsScalarBool(mappingNode.Scalar) {
		return string(substitutions.ResolvedSubExprTypeBoolean)
	}

	if bpcore.IsArrayMappingNode(mappingNode) {
		return string(substitutions.ResolvedSubExprTypeArray)
	}

	if bpcore.IsObjectMappingNode(mappingNode) {
		return string(substitutions.ResolvedSubExprTypeObject)
	}

	return string(substitutions.ResolvedSubExprTypeAny)
}
