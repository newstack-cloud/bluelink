package validation

import (
	"context"
	"fmt"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateExport validates an export in a blueprint.
// This ensures that the export type is valid and that the referenced
// field is in the valid format.
// This does not validate that the field of the export can not be resolved,
// as this export validation should be carried out
// before staging changes or deploying a blueprint.
func ValidateExport(
	ctx context.Context,
	exportName string,
	exportSchema *schema.Export,
	exportMap *schema.ExportMap,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	err := validateExportType(exportSchema.Type, exportName, exportMap)
	if err != nil {
		return diagnostics, err
	}

	err = validateExportFieldFormat(exportSchema.Field, exportName, exportMap)
	if err != nil {
		return diagnostics, err
	}

	// Ensure the export field is present in the blueprint and that the resolved
	// type matches the export type.

	exportFieldAsSub, err := substitutions.ParseSubstitution(
		"exports",
		*exportSchema.Field.StringValue,
		/* parentSourceStart */ &source.Meta{Position: source.Position{}},
		/* outputLineInfo */ false,
		/* ignoreParentColumn */ true,
	)
	// As the substitution is derived and not user provided, we'll use the position
	// of the export field in the exports map as the source location to provide
	// a close enough location in errors and diagnostics.
	populateSubSourceMeta(exportFieldAsSub, exportMap.SourceMeta[exportName])
	if err != nil {
		return diagnostics, err
	}

	exportIdentifier := fmt.Sprintf("exports.%s", exportName)
	resolvedType, subDiagnostics, err := ValidateSubstitution(
		ctx,
		exportFieldAsSub,
		nil,
		bpSchema,
		/* usedInResourceDerivedFromTemplate */ false,
		exportIdentifier,
		"field",
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, subDiagnostics...)
	if err != nil {
		return diagnostics, err
	}

	var errs []error
	if resolvedType != subTypeFromExportType(exportSchema.Type.Value) &&
		resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
		errs = append(errs, errExportTypeMismatch(
			exportSchema.Type.Value,
			resolvedType,
			exportName,
			*exportSchema.Field.StringValue,
			getExportSourceMeta(exportMap, exportName),
		))
	} else if resolvedType == string(substitutions.ResolvedSubExprTypeAny) {
		// Any type will produce a warning diagnostic as any could match an array,
		// an error will occur at runtime if the resolved value is not an array.
		diagnostics = append(
			diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Export referenced field returns \"any\" type, this may produce "+
						"unexpected output in %s, a value with the %s type is expected",
					exportIdentifier,
					exportSchema.Type.Value,
				),
				Range: bpcore.DiagnosticRangeFromSourceMeta(
					getExportSourceMeta(exportMap, exportName),
					nil,
				),
			},
		)
	}

	descriptionDiagnostics, err := validateDescription(
		ctx,
		exportIdentifier,
		/* usedInResourceDerivedFromTemplate */ false,
		exportSchema.Description,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, descriptionDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateExportType(
	exportType *schema.ExportTypeWrapper,
	exportName string,
	exportMap *schema.ExportMap,
) error {
	if exportType == nil {
		return errMissingExportType(
			exportName,
			getExportSourceMeta(exportMap, exportName),
		)
	}

	if !core.SliceContainsComparable(schema.ExportTypes, exportType.Value) {
		return errInvalidExportType(
			exportType.Value,
			exportName,
			getExportSourceMeta(exportMap, exportName),
		)
	}
	return nil
}

func validateExportFieldFormat(exportField *bpcore.ScalarValue, exportName string, exportMap *schema.ExportMap) error {
	if exportField == nil || exportField.StringValue == nil || *exportField.StringValue == "" {
		return errEmptyExportField(
			exportName,
			getExportSourceMeta(exportMap, exportName),
		)
	}

	context := fmt.Sprintf("exports.%s", exportName)
	return ValidateReference(
		*exportField.StringValue,
		context,
		ExportCanReference,
		getExportSourceMeta(exportMap, exportName),
	)
}

var (
	// ExportCanReference is a list of objects that can be referenced
	// by an export.
	// In the current version of the specification, resources, data sources,
	// variables, values and child blueprints can be referenced by an export.
	ExportCanReference = []Referenceable{
		ReferenceableResource,
		ReferenceableDataSource,
		ReferenceableVariable,
		ReferenceableValue,
		ReferenceableChild,
	}
)

func getExportSourceMeta(exportMap *schema.ExportMap, varName string) *source.Meta {
	if exportMap == nil {
		return nil
	}

	return exportMap.SourceMeta[varName]
}

func subTypeFromExportType(exportType schema.ExportType) string {
	switch exportType {
	case schema.ExportTypeInteger:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case schema.ExportTypeFloat:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case schema.ExportTypeBoolean:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	case schema.ExportTypeArray:
		return string(substitutions.ResolvedSubExprTypeArray)
	case schema.ExportTypeObject:
		return string(substitutions.ResolvedSubExprTypeObject)
	default:
		return string(substitutions.ResolvedSubExprTypeString)
	}
}

func populateSubSourceMeta(sub *substitutions.Substitution, sourceMeta *source.Meta) {
	if sourceMeta == nil {
		return
	}

	sub.SourceMeta = &source.Meta{Position: source.Position{
		Line:   sourceMeta.Line,
		Column: sourceMeta.Column,
	}}

	if sub.Function != nil {
		sub.Function.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
		for _, arg := range sub.Function.Arguments {
			arg.SourceMeta = &source.Meta{Position: source.Position{
				Line:   sourceMeta.Line,
				Column: sourceMeta.Column,
			}}
			populateSubSourceMeta(arg.Value, sourceMeta)
		}
	}

	if sub.ElemIndexReference != nil {
		sub.ElemIndexReference.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.ElemReference != nil {
		sub.ElemReference.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.Child != nil {
		sub.Child.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.DataSourceProperty != nil {
		sub.DataSourceProperty.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.ResourceProperty != nil {
		sub.ResourceProperty.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.ValueReference != nil {
		sub.ValueReference.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}

	if sub.Variable != nil {
		sub.Variable.SourceMeta = &source.Meta{Position: source.Position{
			Line:   sourceMeta.Line,
			Column: sourceMeta.Column,
		}}
	}
}
