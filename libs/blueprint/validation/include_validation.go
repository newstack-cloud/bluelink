package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
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
	valCtx *ValidationContext,
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
		valCtx,
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
		valCtx,
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
		valCtx,
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
		valCtx,
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
	valCtx *ValidationContext,
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
				valCtx,
				/* usedInResourceDerivedFromTemplate */ false,
				includeIdentifier,
				"path",
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

// ValidateIncludePathExists validates that a child blueprint include path
// points to an existing file on the local filesystem.
// This is a best-effort validation: paths that cannot be statically resolved
// (e.g. containing non-cwd substitutions or remote URLs) are silently skipped.
func ValidateIncludePathExists(
	includeName string,
	includeSchema *schema.Include,
	resolveWorkingDir corefunctions.WorkingDirResolver,
	params core.BlueprintParams,
) ([]*core.Diagnostic, error) {
	if includeSchema == nil || includeSchema.Path == nil {
		return []*core.Diagnostic{}, nil
	}

	if IsRemoteInclude(includeSchema) {
		return []*core.Diagnostic{}, nil
	}

	resolvedPath, ok := TryResolveIncludePath(includeSchema.Path, resolveWorkingDir)
	if !ok {
		return []*core.Diagnostic{}, nil
	}

	if !filepath.IsAbs(resolvedPath) {
		baseDir := resolveBaseDir(params, resolveWorkingDir)
		if baseDir == "" {
			return []*core.Diagnostic{}, nil
		}
		resolvedPath = filepath.Join(baseDir, resolvedPath)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errIncludePathNotFound(
				includeName, resolvedPath, includeSchema.Path.SourceMeta,
			)
		}
		// For other errors (e.g. permissions), skip validation silently.
		return []*core.Diagnostic{}, nil
	}

	if info.IsDir() {
		return []*core.Diagnostic{
			{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Include path for %q resolves to a directory %q,"+
						" expected a file path",
					includeName,
					resolvedPath,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					includeSchema.Path.SourceMeta,
					nil,
				),
			},
		}, nil
	}

	return []*core.Diagnostic{}, nil
}

// TryResolveIncludePath attempts to resolve an include path to a concrete
// filesystem path string. Returns the resolved path and true if successful,
// or ("", false) if the path contains unresolvable substitutions.
func TryResolveIncludePath(
	path *substitutions.StringOrSubstitutions,
	resolveWorkingDir corefunctions.WorkingDirResolver,
) (string, bool) {
	if path == nil || len(path.Values) == 0 {
		return "", false
	}

	var sb strings.Builder
	for _, segment := range path.Values {
		if segment.StringValue != nil {
			sb.WriteString(*segment.StringValue)
			continue
		}

		if segment.SubstitutionValue == nil {
			continue
		}

		sub := segment.SubstitutionValue
		if sub.Function != nil &&
			sub.Function.FunctionName == substitutions.SubstitutionFunctionCWD &&
			len(sub.Function.Arguments) == 0 {
			if resolveWorkingDir == nil {
				return "", false
			}
			cwd, err := resolveWorkingDir()
			if err != nil {
				return "", false
			}
			sb.WriteString(cwd)
			continue
		}

		// Any other substitution makes the path unresolvable.
		return "", false
	}

	resolved := sb.String()
	if resolved == "" {
		return "", false
	}

	return resolved, true
}

// remotePathPrefixes contains URL scheme prefixes that indicate
// an include path points to a remote location.
var remotePathPrefixes = []string{
	"http://",
	"https://",
	"s3://",
	"gs://",
}

// remoteSourceMetadataFields contains metadata field names
// that indicate an include uses a remote resolver.
// The primary field is "sourceType", used by the built-in router resolver
// to route to the correct child blueprint resolver plugin (e.g. aws/s3, gcs, azure, https).
// Other fields are checked as alternative conventions that custom resolver
// plugins may use.
var remoteSourceMetadataFields = []string{
	"sourceType",
	"source",
	"type",
	"provider",
	"protocol",
}

// IsRemoteInclude returns true if the include references a remote child blueprint.
// This is determined by checking:
//  1. Metadata fields that indicate a remote resolver (e.g. sourceType, source, type)
//  2. The include path for URL scheme prefixes (e.g. https://, s3://, gs://)
func IsRemoteInclude(include *schema.Include) bool {
	if include == nil {
		return false
	}

	if hasRemoteSourceMetadata(include) {
		return true
	}

	return hasRemotePathPrefix(include.Path)
}

func hasRemoteSourceMetadata(include *schema.Include) bool {
	if include.Metadata == nil || include.Metadata.Fields == nil {
		return false
	}

	for _, field := range remoteSourceMetadataFields {
		if core.StringValue(include.Metadata.Fields[field]) != "" {
			return true
		}
	}

	return false
}

func hasRemotePathPrefix(path *substitutions.StringOrSubstitutions) bool {
	if path == nil || len(path.Values) == 0 {
		return false
	}

	first := path.Values[0]
	if first.StringValue == nil {
		return false
	}

	for _, prefix := range remotePathPrefixes {
		if strings.HasPrefix(*first.StringValue, prefix) {
			return true
		}
	}

	return false
}

// ValidateIncludeVariables validates that variables provided to a child
// blueprint include match the child blueprint's variable definitions.
// Checks: unknown variables (warning), missing required (error),
// and type mismatches using substitution-aware type resolution (error).
// This only runs when the child blueprint has been successfully loaded.
func ValidateIncludeVariables(
	ctx context.Context,
	includeName string,
	includeSchema *schema.Include,
	includeMap *schema.IncludeMap,
	childBpSchema *schema.Blueprint,
	valCtx *ValidationContext,
) ([]*core.Diagnostic, error) {
	if childBpSchema.Variables == nil {
		return nil, nil
	}

	diagnostics := []*core.Diagnostic{}
	var errs []error
	includeIdentifier := fmt.Sprintf("include.%s", includeName)
	providedVars := includeVarFields(includeSchema)

	unknownDiagnostics := checkUnknownIncludeVars(
		includeName, includeSchema, childBpSchema,
	)
	diagnostics = append(diagnostics, unknownDiagnostics...)

	missingErrs := checkMissingRequiredIncludeVars(
		includeName, includeMap, childBpSchema, providedVars,
	)
	errs = append(errs, missingErrs...)

	typeDiagnostics, typeErrs := checkIncludeVarTypes(
		ctx, includeName, includeIdentifier, includeSchema,
		childBpSchema, valCtx,
		providedVars,
	)
	diagnostics = append(diagnostics, typeDiagnostics...)
	errs = append(errs, typeErrs...)

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func includeVarFields(includeSchema *schema.Include) map[string]*core.MappingNode {
	if includeSchema.Variables == nil || includeSchema.Variables.Fields == nil {
		return nil
	}
	return includeSchema.Variables.Fields
}

func checkUnknownIncludeVars(
	includeName string,
	includeSchema *schema.Include,
	childBpSchema *schema.Blueprint,
) []*core.Diagnostic {
	if includeSchema.Variables == nil || includeSchema.Variables.Fields == nil {
		return nil
	}

	var diagnostics []*core.Diagnostic
	for varName := range includeSchema.Variables.Fields {
		if _, ok := childBpSchema.Variables.Values[varName]; !ok {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Variable %q provided to include %q is not defined"+
						" in the child blueprint",
					varName, includeName,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					includeSchema.Variables.FieldsSourceMeta[varName],
					nil,
				),
			})
		}
	}
	return diagnostics
}

func checkMissingRequiredIncludeVars(
	includeName string,
	includeMap *schema.IncludeMap,
	childBpSchema *schema.Blueprint,
	providedVars map[string]*core.MappingNode,
) []error {
	var errs []error
	includeKeySourceMeta := getIncludeSourceMeta(includeMap, includeName)
	for varName, varDef := range childBpSchema.Variables.Values {
		if varDef.Default != nil {
			continue
		}
		if _, ok := providedVars[varName]; ok {
			continue
		}
		errs = append(errs, errIncludeMissingRequiredVar(
			includeName, varName, includeKeySourceMeta,
		))
	}
	return errs
}

func checkIncludeVarTypes(
	ctx context.Context,
	includeName string,
	includeIdentifier string,
	includeSchema *schema.Include,
	childBpSchema *schema.Blueprint,
	valCtx *ValidationContext,
	providedVars map[string]*core.MappingNode,
) ([]*core.Diagnostic, []error) {
	if providedVars == nil {
		return nil, nil
	}

	var diagnostics []*core.Diagnostic
	var errs []error
	for varName, varNode := range providedVars {
		childVar, ok := childBpSchema.Variables.Values[varName]
		if !ok || childVar.Type == nil {
			continue
		}

		resolvedType, typeDiagnostics, err := resolveIncludeVarType(
			ctx, varNode, includeIdentifier, varName,
			valCtx,
		)
		diagnostics = append(diagnostics, typeDiagnostics...)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if resolvedType == "" {
			continue
		}

		if resolvedType != string(childVar.Type.Value) {
			errs = append(errs, errIncludeVarTypeMismatch(
				includeName, varName, resolvedType,
				string(childVar.Type.Value),
				includeSchema.Variables.FieldsSourceMeta[varName],
			))
		}
	}

	return diagnostics, errs
}

// resolveIncludeVarType determines the type of a MappingNode variable value
// for comparison with a child blueprint's variable type.
// Uses ValidateSubstitution for single ${..} substitution values
// and validates string interpolation substitutions resolve to primitive types.
// Returns empty string when the type cannot be statically determined.
func resolveIncludeVarType(
	ctx context.Context,
	node *core.MappingNode,
	usedIn string,
	varName string,
	valCtx *ValidationContext,
) (string, []*core.Diagnostic, error) {
	if node == nil {
		return "", nil, nil
	}

	if node.Scalar != nil {
		return resolveScalarType(node.Scalar), nil, nil
	}

	if node.StringWithSubstitutions != nil {
		return resolveStringWithSubsType(
			ctx, node.StringWithSubstitutions, usedIn, varName,
			valCtx,
		)
	}

	// Fields or Items represent complex types that can't be compared
	// against variable types at validation time.
	return "", nil, nil
}

func resolveScalarType(scalar *core.ScalarValue) string {
	if scalar.IntValue != nil {
		return string(substitutions.ResolvedSubExprTypeInteger)
	}
	if scalar.FloatValue != nil {
		return string(substitutions.ResolvedSubExprTypeFloat)
	}
	if scalar.BoolValue != nil {
		return string(substitutions.ResolvedSubExprTypeBoolean)
	}
	if scalar.StringValue != nil {
		return string(substitutions.ResolvedSubExprTypeString)
	}
	return ""
}

func resolveStringWithSubsType(
	ctx context.Context,
	strWithSubs *substitutions.StringOrSubstitutions,
	usedIn string,
	varName string,
	valCtx *ValidationContext,
) (string, []*core.Diagnostic, error) {
	if isSingleSubstitution(strWithSubs) {
		return resolveSingleSubType(
			ctx, strWithSubs.Values[0].SubstitutionValue, usedIn, varName,
			valCtx,
		)
	}

	return resolveInterpolatedStringType(
		ctx, strWithSubs, usedIn, varName,
		valCtx,
	)
}

func isSingleSubstitution(strWithSubs *substitutions.StringOrSubstitutions) bool {
	return len(strWithSubs.Values) == 1 &&
		strWithSubs.Values[0].SubstitutionValue != nil
}

func resolveSingleSubType(
	ctx context.Context,
	sub *substitutions.Substitution,
	usedIn string,
	varName string,
	valCtx *ValidationContext,
) (string, []*core.Diagnostic, error) {
	propertyPath := fmt.Sprintf("variables.%s", varName)
	resolvedType, diagnostics, err := ValidateSubstitution(
		ctx,
		sub,
		nil,
		valCtx,
		/* usedInResourceDerivedFromTemplate */ false,
		usedIn,
		propertyPath,
	)
	if err != nil {
		return "", diagnostics, err
	}
	return resolvedType, diagnostics, nil
}

func resolveInterpolatedStringType(
	ctx context.Context,
	strWithSubs *substitutions.StringOrSubstitutions,
	usedIn string,
	varName string,
	valCtx *ValidationContext,
) (string, []*core.Diagnostic, error) {
	var diagnostics []*core.Diagnostic
	propertyPath := fmt.Sprintf("variables.%s", varName)

	for _, stringOrSub := range strWithSubs.Values {
		if stringOrSub.SubstitutionValue == nil {
			continue
		}

		resolvedType, subDiagnostics, err := ValidateSubstitution(
			ctx,
			stringOrSub.SubstitutionValue,
			nil,
			valCtx,
			/* usedInResourceDerivedFromTemplate */ false,
			usedIn,
			propertyPath,
		)
		if err != nil {
			// Substitution validation errors are already reported by
			// ValidateMappingNode; skip type check for this substitution.
			continue
		}

		diagnostics = append(diagnostics, subDiagnostics...)
		if resolvedType != "" && !isSubPrimitiveType(resolvedType) {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Substitution in variable %q for %q resolves to"+
						" non-primitive type %q in a string interpolation context,"+
						" only primitive types can be interpolated into strings",
					varName, usedIn, resolvedType,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					stringOrSub.SourceMeta,
					nil,
				),
			})
		}
	}

	return string(substitutions.ResolvedSubExprTypeString), diagnostics, nil
}

func resolveBaseDir(
	params core.BlueprintParams,
	resolveWorkingDir corefunctions.WorkingDirResolver,
) string {
	if params != nil {
		bpDir := params.ContextVariable("__blueprintDir")
		if bpDir != nil && bpDir.StringValue != nil && *bpDir.StringValue != "" {
			return *bpDir.StringValue
		}
	}

	if resolveWorkingDir != nil {
		dir, err := resolveWorkingDir()
		if err == nil {
			return dir
		}
	}

	return ""
}

