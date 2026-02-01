package validation

import (
	"context"
	"fmt"
	"slices"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateDataSourceName checks the validity of a data source name,
// primarily making sure that it does not contain any substitutions
// as per the spec.
func ValidateDataSourceName(mappingName string, dataSourceMap *schema.DataSourceMap) error {
	if substitutions.ContainsSubstitution(mappingName) {
		return errMappingNameContainsSubstitution(
			mappingName,
			"data source",
			ErrorReasonCodeInvalidDataSource,
			getDataSourceMeta(dataSourceMap, mappingName),
		)
	}
	return nil
}

// ValidateDataSource ensures that a given data source matches the specification.
func ValidateDataSource(
	ctx context.Context,
	name string,
	dataSource *schema.DataSource,
	dataSourceMap *schema.DataSourceMap,
	valCtx *ValidationContext,
	logger bpcore.Logger,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	var errs []error

	logger.Debug("Validating data source type")
	validateTypeDiagnostics, validateTypeErr := validateDataSourceType(
		ctx,
		name,
		dataSource.Type,
		dataSourceMap,
		valCtx.DataSourceRegistry,
	)
	diagnostics = append(diagnostics, validateTypeDiagnostics...)
	if validateTypeErr != nil {
		errs = append(errs, validateTypeErr)
	}

	logger.Debug("Validating data source metadata")
	validateMetadataDiagnostics, validateMetadataErr := validateDataSourceMetadata(
		ctx,
		name,
		dataSource.DataSourceMetadata,
		valCtx,
	)
	diagnostics = append(diagnostics, validateMetadataDiagnostics...)
	if validateMetadataErr != nil {
		errs = append(errs, validateMetadataErr)
	}

	logger.Debug("Validating data source description")
	validateDescriptionDiagnostics, validateDescErr := validateDescription(
		ctx,
		bpcore.DataSourceElementID(name),
		/* usedInResourceDerivedFromTemplate */ false,
		dataSource.Description,
		valCtx,
	)
	diagnostics = append(diagnostics, validateDescriptionDiagnostics...)
	if validateDescErr != nil {
		errs = append(errs, validateDescErr)
	}

	// All validation after this point requires a data source type,
	// if one isn't set, we'll return the errors and diagnostics
	// collected so far.
	if dataSource.Type == nil {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	logger.Debug("Validating data source filters")
	validateFilterDiagnostics, validateFilterErr := validateDataSourceFilters(
		ctx,
		name,
		dataSource.Type.Value,
		dataSource.Filter,
		dataSourceMap,
		valCtx,
	)
	diagnostics = append(diagnostics, validateFilterDiagnostics...)
	if validateFilterErr != nil {
		errs = append(errs, validateFilterErr)
	}

	specDefinition, specDefErr := loadDataSourceSpecDefinition(
		ctx,
		dataSource.Type.Value,
		name,
		dataSource.SourceMeta,
		valCtx.Params,
		valCtx.DataSourceRegistry,
	)
	if specDefErr != nil {
		errs = append(errs, specDefErr)
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	logger.Debug("Validating data source exports")
	validateExportsDiagnostics, validateExportsErr := validateDataSourceExports(
		ctx,
		name,
		dataSource.Type.Value,
		dataSource.Exports,
		dataSourceMap,
		valCtx,
		specDefinition,
	)
	diagnostics = append(diagnostics, validateExportsDiagnostics...)
	if validateExportsErr != nil {
		errs = append(errs, validateExportsErr)
	}

	logger.Debug("Running custom validation for data source")
	providerNamespace := provider.ExtractProviderFromItemType(dataSource.Type.Value)
	customValidateOutput, err := valCtx.DataSourceRegistry.CustomValidate(
		ctx,
		dataSource.Type.Value,
		&provider.DataSourceValidateInput{
			SchemaDataSource: dataSource,
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				valCtx.Params,
			),
		},
	)
	if err != nil {
		errs = append(errs, err)
	}
	if customValidateOutput != nil {
		diagnostics = append(diagnostics, customValidateOutput.Diagnostics...)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateDataSourceType(
	ctx context.Context,
	dataSourceName string,
	dataSourceType *schema.DataSourceTypeWrapper,
	dataSourceMap *schema.DataSourceMap,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if dataSourceType == nil {
		return diagnostics, errDataSourceMissingType(
			dataSourceName,
			getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	// Use the type definition's location for more precise error reporting
	location := dataSourceType.SourceMeta
	hasType, err := dataSourceRegistry.HasDataSourceType(ctx, dataSourceType.Value)
	if err != nil {
		return diagnostics, wrapRegistryError(err, location)
	}

	if !hasType {
		return diagnostics, errDataSourceTypeNotSupported(
			dataSourceName,
			dataSourceType.Value,
			location,
		)
	}

	return diagnostics, nil
}

func validateDataSourceMetadata(
	ctx context.Context,
	dataSourceName string,
	metadataSchema *schema.DataSourceMetadata,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if metadataSchema == nil {
		return diagnostics, nil
	}

	var errs []error

	displayNameDiagnostics, err := validateDataSourceMetadataDisplayName(
		ctx,
		dataSourceName,
		metadataSchema,
		valCtx,
	)
	diagnostics = append(diagnostics, displayNameDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	annotationsDiagnostics, err := validateDataSourceMetadataAnnotations(
		ctx,
		dataSourceName,
		metadataSchema,
		valCtx,
	)
	diagnostics = append(diagnostics, annotationsDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	customDiagnostics, err := ValidateMappingNode(
		ctx,
		bpcore.DataSourceElementID(dataSourceName),
		"metadata.custom",
		/* usedInResourceDerivedFromTemplate */ false,
		metadataSchema.Custom,
		valCtx,
	)
	diagnostics = append(diagnostics, customDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateDataSourceMetadataDisplayName(
	ctx context.Context,
	dataSourceName string,
	metadataSchema *schema.DataSourceMetadata,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	if metadataSchema.DisplayName == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	dataSourceIdentifier := bpcore.DataSourceElementID(dataSourceName)
	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for _, stringOrSub := range metadataSchema.DisplayName.Values {
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				valCtx,
				/* usedInResourceDerivedFromTemplate */ false,
				dataSourceIdentifier,
				"metadata.displayName",
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				if !isSubPrimitiveType(resolvedType) {
					errs = append(errs, errInvalidDisplayNameSubType(
						dataSourceIdentifier,
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

func validateDataSourceMetadataAnnotations(
	ctx context.Context,
	dataSourceName string,
	metadataSchema *schema.DataSourceMetadata,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	if metadataSchema.Annotations == nil || metadataSchema.Annotations.Values == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	dataSourceIdentifier := bpcore.DataSourceElementID(dataSourceName)
	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for key, annotation := range metadataSchema.Annotations.Values {
		if substitutions.ContainsSubstitution(key) {
			errs = append(errs, errDataSourceAnnotationKeyContainsSubstitution(
				dataSourceName,
				key,
				annotation.SourceMeta,
			))
		}

		annotationDiagnostics, err := validateMetadataAnnotation(
			ctx,
			dataSourceIdentifier,
			annotation,
			valCtx,
		)
		diagnostics = append(diagnostics, annotationDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateMetadataAnnotation(
	ctx context.Context,
	itemIdentifier string,
	annotation *substitutions.StringOrSubstitutions,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	if annotation == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for i, stringOrSub := range annotation.Values {
		nextLocation := getSubNextLocation(i, annotation.Values)

		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				valCtx,
				/* usedInResourceDerivedFromTemplate */ false,
				itemIdentifier,
				"metadata.annotations",
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				handleResolvedTypeExpectingPrimitive(
					resolvedType,
					itemIdentifier,
					stringOrSub,
					annotation,
					"annotation",
					nextLocation,
					&diagnostics,
					&errs,
				)
			}
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func loadDataSourceSpecDefinition(
	ctx context.Context,
	dataSourceType string,
	dataSourceName string,
	location *source.Meta,
	params bpcore.BlueprintParams,
	dataSourceRegistry provider.DataSourceRegistry,
) (*provider.DataSourceSpecDefinition, error) {
	providerNamespace := provider.ExtractProviderFromItemType(dataSourceType)
	specDefOutput, err := dataSourceRegistry.GetSpecDefinition(
		ctx,
		dataSourceType,
		&provider.DataSourceGetSpecDefinitionInput{
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				params,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	if specDefOutput.SpecDefinition == nil {
		return nil, errDataSourceTypeMissingSpecDefinition(
			dataSourceName,
			dataSourceType,
			/* inSubstitution */ false,
			location,
			"spec definition not found during data source validation",
		)
	}

	return specDefOutput.SpecDefinition, nil
}

func validateDataSourceFilters(
	ctx context.Context,
	name string,
	dataSourceType string,
	filters *schema.DataSourceFilters,
	dataSourceMap *schema.DataSourceMap,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if filters == nil || len(filters.Filters) == 0 {
		return diagnostics, errDataSourceMissingFilter(
			name, getDataSourceMeta(dataSourceMap, name),
		)
	}

	errs := []error{}
	for _, filter := range filters.Filters {
		otherFilterFields := getOtherFilterFields(
			filters.Filters,
			filter,
		)
		filterDiagnostics, err := validateDataSourceFilter(
			ctx,
			name,
			dataSourceType,
			filter,
			otherFilterFields,
			dataSourceMap,
			valCtx,
		)
		diagnostics = append(diagnostics, filterDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func getOtherFilterFields(
	filters []*schema.DataSourceFilter,
	filter *schema.DataSourceFilter,
) []string {
	otherFilterFields := []string{}
	for _, f := range filters {
		if f != filter &&
			bpcore.IsScalarString(f.Field) {
			otherFilterFields = append(
				otherFilterFields,
				bpcore.StringValueFromScalar(f.Field),
			)
		}
	}
	return otherFilterFields
}

func validateDataSourceFilter(
	ctx context.Context,
	dataSourceName string,
	dataSourceType string,
	filter *schema.DataSourceFilter,
	otherFilterFields []string,
	dataSourceMap *schema.DataSourceMap,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if filter == nil {
		return diagnostics, errDataSourceEmptyFilter(
			dataSourceName, getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	if filter.Field == nil || filter.Field.StringValue == nil || *filter.Field.StringValue == "" {
		return diagnostics, errDataSourceMissingFilterField(
			dataSourceName, getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	if filter.Operator == nil || filter.Operator.Value == "" {
		return diagnostics, errDataSourceMissingFilterOperator(
			dataSourceName, getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	if filter.Search == nil || len(filter.Search.Values) == 0 {
		return diagnostics, errDataSourceMissingFilterSearch(
			dataSourceName, getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	providerNamespace := provider.ExtractProviderFromItemType(dataSourceType)
	filterFieldsOutput, err := valCtx.DataSourceRegistry.GetFilterFields(
		ctx,
		dataSourceType,
		&provider.DataSourceGetFilterFieldsInput{
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				valCtx.Params,
			),
		},
	)
	if err != nil {
		return diagnostics, err
	}

	if len(filterFieldsOutput.FilterFields) == 0 {
		return diagnostics, errDataSourceTypeMissingFields(
			dataSourceName,
			dataSourceType,
			filter.SourceMeta,
		)
	}

	filterFieldSchema, hasFilterField := filterFieldsOutput.FilterFields[*filter.Field.StringValue]
	if !hasFilterField {
		return diagnostics, errDataSourceFilterFieldNotSupported(
			dataSourceName,
			bpcore.StringValueFromScalar(filter.Field),
			filter.SourceMeta,
		)
	}

	validateConflictErr := validateDataSourceFilterFieldConflict(
		dataSourceName,
		bpcore.StringValueFromScalar(filter.Field),
		otherFilterFields,
		filterFieldSchema,
		filter,
	)
	if validateConflictErr != nil {
		return diagnostics, validateConflictErr
	}

	validateFilterOpDiagnostics, validateFilterOpErr := validateDataSourceFilterOperator(
		dataSourceName,
		filter.Operator,
		*filter.Field.StringValue,
		filterFieldSchema,
	)
	diagnostics = append(diagnostics, validateFilterOpDiagnostics...)
	if validateFilterOpErr != nil {
		return diagnostics, validateFilterOpErr
	}

	searchValidationDiagnostics, searchValidationErr := validateDataSourceFilterSearch(
		ctx,
		dataSourceName,
		filter.Search,
		valCtx,
	)
	diagnostics = append(diagnostics, searchValidationDiagnostics...)
	if searchValidationErr != nil {
		return diagnostics, searchValidationErr
	}

	return diagnostics, nil
}

func validateDataSourceFilterFieldConflict(
	dataSourceName string,
	fieldName string,
	otherFilterFields []string,
	filterFieldSchema *provider.DataSourceFilterSchema,
	filter *schema.DataSourceFilter,
) error {
	if filterFieldSchema == nil {
		return nil
	}

	for _, otherField := range otherFilterFields {
		if slices.Contains(filterFieldSchema.ConflictsWith, otherField) {
			return errDataSourceFilterFieldConflict(
				dataSourceName,
				fieldName,
				otherField,
				filter.SourceMeta,
			)
		}
	}

	return nil
}

func validateDataSourceFilterOperator(
	dataSourceName string,
	operator *schema.DataSourceFilterOperatorWrapper,
	filterFieldName string,
	filterFieldSchema *provider.DataSourceFilterSchema,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if !core.SliceContains(schema.DataSourceFilterOperators, operator.Value) {
		return diagnostics, errInvalidDataSourceFilterOperator(dataSourceName, operator)
	}

	if !core.SliceContains(filterFieldSchema.SupportedOperators, operator.Value) {
		return diagnostics, errDataSourceFilterOperatorNotSupported(
			dataSourceName,
			operator.Value,
			filterFieldName,
			filterFieldSchema.SupportedOperators,
			operator.SourceMeta,
		)
	}

	return diagnostics, nil
}

func validateDataSourceFilterSearch(
	ctx context.Context,
	dataSourceName string,
	search *schema.DataSourceFilterSearch,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {

	dataSourceIdentifier := bpcore.DataSourceElementID(dataSourceName)
	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for _, searchValue := range search.Values {
		searchValueDiagnostics, err := validateDataSourceFilterSearchValue(
			ctx,
			dataSourceIdentifier,
			searchValue,
			valCtx,
		)
		diagnostics = append(diagnostics, searchValueDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateDataSourceFilterSearchValue(
	ctx context.Context,
	dataSourceIdentifier string,
	searchValue *substitutions.StringOrSubstitutions,
	valCtx *ValidationContext,
) ([]*bpcore.Diagnostic, error) {
	if searchValue == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for i, stringOrSub := range searchValue.Values {
		nextLocation := getSubNextLocation(i, searchValue.Values)

		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				valCtx,
				/* usedInResourceDerivedFromTemplate */ false,
				dataSourceIdentifier,
				"filter.search",
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
				handleResolvedTypeExpectingPrimitive(
					resolvedType,
					dataSourceIdentifier,
					stringOrSub,
					searchValue,
					"search value",
					nextLocation,
					&diagnostics,
					&errs,
				)
			}
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func handleResolvedTypeExpectingPrimitive(
	resolvedType string,
	dataSourceIdentifier string,
	stringOrSub *substitutions.StringOrSubstitution,
	value *substitutions.StringOrSubstitutions,
	valueContext string,
	nextLocation *source.Meta,
	diagnostics *[]*bpcore.Diagnostic,
	errs *[]error,
) {
	if !isSubPrimitiveType(resolvedType) && resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
		*errs = append(*errs, errInvalidSubType(
			dataSourceIdentifier,
			valueContext,
			resolvedType,
			stringOrSub.SourceMeta,
		))
	} else if resolvedType == string(substitutions.ResolvedSubExprTypeAny) {
		// Any type will produce a warning diagnostic as any is likely to match
		// and will be stringified in the final output, which is undesired
		// but not undefined behaviour.
		*diagnostics = append(
			*diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Substitution returns \"any\" type, this may produce "+
						"unexpected output in the %s, %ss are expected to be scalar values",
					valueContext,
					valueContext,
				),
				Range:   bpcore.DiagnosticRangeFromSourceMeta(value.SourceMeta, nextLocation),
				Context: &bperrors.ErrorContext{ReasonCode: bperrors.ErrorReasonCodeAnyTypeWarning},
			},
		)
	}
}

func validateDataSourceExports(
	ctx context.Context,
	dataSourceName string,
	dataSourceType string,
	exports *schema.DataSourceFieldExportMap,
	dataSourceMap *schema.DataSourceMap,
	valCtx *ValidationContext,
	specDefinition *provider.DataSourceSpecDefinition,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if exports == nil || len(exports.Values) == 0 {
		return diagnostics, errDataSourceMissingExports(
			dataSourceName,
			getDataSourceMeta(dataSourceMap, dataSourceName),
		)
	}

	errs := []error{}
	for exportName, export := range exports.Values {
		exportDiagnostics, err := validateDataSourceExport(
			ctx,
			dataSourceName,
			dataSourceType,
			export,
			exportName,
			/* wrapperLocation */ exports.SourceMeta[exportName],
			valCtx,
			specDefinition,
		)
		diagnostics = append(diagnostics, exportDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateDataSourceExport(
	ctx context.Context,
	dataSourceName string,
	dataSourceType string,
	export *schema.DataSourceFieldExport,
	exportName string,
	wrapperLocation *source.Meta,
	valCtx *ValidationContext,
	specDefinition *provider.DataSourceSpecDefinition,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if export == nil {
		return diagnostics, errDataSourceExportEmpty(
			dataSourceName,
			exportName,
			wrapperLocation,
		)
	}

	finalExportName := exportName
	if export.AliasFor != nil &&
		export.AliasFor.StringValue != nil &&
		*export.AliasFor.StringValue != "" {
		finalExportName = *export.AliasFor.StringValue
	}
	fieldSchema, hasField := specDefinition.Fields[finalExportName]
	// Field schema may incorrectly set to nil by the data source provider.
	if !hasField || fieldSchema == nil {
		return diagnostics, errDataSourceExportFieldNotSupported(
			dataSourceName,
			dataSourceType,
			exportName,
			finalExportName,
			wrapperLocation,
		)
	}

	if export.Type == nil {
		return diagnostics, errDataSourceExportTypeMissing(
			dataSourceName,
			exportName,
			wrapperLocation,
		)
	}

	if !core.SliceContains(schema.DataSourceFieldTypes, export.Type.Value) {
		return diagnostics, errInvalidDataSourceFieldType(
			dataSourceName,
			exportName,
			export.Type,
		)
	}

	if !schemaMatchesDataSourceFieldType(fieldSchema, export.Type) {
		return diagnostics, errDataSourceExportFieldTypeMismatch(
			dataSourceName,
			exportName,
			finalExportName,
			string(fieldSchema.Type),
			string(export.Type.Value),
			wrapperLocation,
		)
	}

	diagnostics, err := validateDescription(
		ctx,
		fmt.Sprintf(
			"%s.exports.%s",
			bpcore.DataSourceElementID(dataSourceName),
			exportName,
		),
		/* usedInResourceDerivedFromTemplate */ false,
		export.Description,
		valCtx,
	)
	if err != nil {
		return diagnostics, err
	}

	return diagnostics, nil
}

func schemaMatchesDataSourceFieldType(
	fieldSchema *provider.DataSourceSpecSchema,
	exportType *schema.DataSourceFieldTypeWrapper,
) bool {
	if fieldSchema == nil || exportType == nil {
		return false
	}

	if fieldSchema.Type == provider.DataSourceSpecTypeString &&
		exportType.Value == schema.DataSourceFieldTypeString {
		return true
	}

	if fieldSchema.Type == provider.DataSourceSpecTypeInteger &&
		exportType.Value == schema.DataSourceFieldTypeInteger {
		return true
	}

	if fieldSchema.Type == provider.DataSourceSpecTypeFloat &&
		exportType.Value == schema.DataSourceFieldTypeFloat {
		return true
	}

	if fieldSchema.Type == provider.DataSourceSpecTypeBoolean &&
		exportType.Value == schema.DataSourceFieldTypeBoolean {
		return true
	}

	if fieldSchema.Type == provider.DataSourceSpecTypeArray &&
		exportType.Value == schema.DataSourceFieldTypeArray {
		return true
	}

	return false
}

func getDataSourceMeta(varMap *schema.DataSourceMap, varName string) *source.Meta {
	if varMap == nil {
		return nil
	}

	return varMap.SourceMeta[varName]
}
