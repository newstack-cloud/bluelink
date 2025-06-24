package validation

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ValidateIncludeName checks the validity of a include name,
// primarily making sure that it does not contain any substitutions
// as per the spec.
func ValidateIncludeName(mappingName string, includeMap *schema.IncludeMap) error {
	if substitutions.ContainsSubstitution(mappingName) {
		return errMappingNameContainsSubstitution(
			mappingName,
			"include",
			ErrorReasonCodeInvalidResource,
			getIncludeSourceMeta(includeMap, mappingName),
		)
	}
	return nil
}

// ValidateInclude deals with early stage validation of a child blueprint
// include. This validation is primarily responsible for ensuring the
// path of an include is not empty and that any substitutions used
// are valid.
// As we don't have enough extra information at the early stage at which this should run,
// it does not include validation of the path format or variables.
// Variable validation requires information about the variables that are available
// in the child blueprint, which is not available at this stage.
func ValidateInclude(
	ctx context.Context,
	includeName string,
	includeSchema *schema.Include,
	includeMap *schema.IncludeMap,
	bpSchema *schema.Blueprint,
	params core.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}
	var errs []error

	if isEmptyStringWithSubstitutions(includeSchema.Path) {
		return diagnostics, errIncludeEmptyPath(
			includeName,
			getIncludeSourceMeta(includeMap, includeName),
		)
	}

	includePathDiagnostics, err := validateIncludePath(
		ctx,
		includeName,
		includeSchema,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, includePathDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	includeIdentifier := fmt.Sprintf("include.%s", includeName)

	variablesDiagnostics, err := ValidateMappingNode(
		ctx,
		includeIdentifier,
		"variables",
		/* usedInResourceDerivedFromTemplate */ false,
		includeSchema.Variables,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, variablesDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	includeDescriptionDiagnostics, err := validateDescription(
		ctx,
		includeIdentifier,
		/* usedInResourceDerivedFromTemplate */ false,
		includeSchema.Description,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, includeDescriptionDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	metadataDiagnostics, err := ValidateMappingNode(
		ctx,
		includeIdentifier,
		"metadata",
		/* usedInResourceDerivedFromTemplate */ false,
		includeSchema.Metadata,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, metadataDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateIncludePath(
	ctx context.Context,
	includeName string,
	includeSchema *schema.Include,
	bpSchema *schema.Blueprint,
	params core.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*core.Diagnostic, error) {
	if includeSchema.Path == nil {
		return []*core.Diagnostic{}, nil
	}

	includeIdentifier := fmt.Sprintf("include.%s", includeName)
	errs := []error{}
	diagnostics := []*core.Diagnostic{}
	for _, stringOrSub := range includeSchema.Path.Values {
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				bpSchema,
				/* usedInResourceDerivedFromTemplate */ false,
				includeIdentifier,
				"path",
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
					errs = append(errs, errInvalidIncludePathSubType(
						includeIdentifier,
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

func getIncludeSourceMeta(includeMap *schema.IncludeMap, varName string) *source.Meta {
	if includeMap == nil {
		return nil
	}

	return includeMap.SourceMeta[varName]
}
