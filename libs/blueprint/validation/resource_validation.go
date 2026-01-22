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
)

// ValidateResourceName checks the validity of a resource name,
// primarily making sure that it does not contain any substitutions
// as per the spec.
func ValidateResourceName(mappingName string, resourceMap *schema.ResourceMap) error {
	if substitutions.ContainsSubstitution(mappingName) {
		return errMappingNameContainsSubstitution(
			mappingName,
			"resource",
			ErrorReasonCodeInvalidResource,
			getResourceSourceMeta(resourceMap, mappingName),
		)
	}
	return nil
}

// PreValidateResourceSpec pre-validates the resource specification against the blueprint
// specification. This primarily searches for invalid usage of substitutions in mapping keys.
// The main resource validation that invokes a user-provided resource implementation
// comes after this.
func PreValidateResourceSpec(
	ctx context.Context,
	resourceName string,
	resourceSchema *schema.Resource,
	resourceMap *schema.ResourceMap,
) error {
	if resourceSchema.Spec == nil {
		return nil
	}

	errors := preValidateMappingNode(ctx, resourceSchema.Spec, "resource", resourceName)
	if len(errors) > 0 {
		return errResourceSpecPreValidationFailed(
			errors,
			resourceName,
			getResourceSourceMeta(resourceMap, resourceName),
		)
	}

	return nil
}

// ValidateResource ensures that a given resource is valid as per the blueprint specification
// and the resource type specification definition exposed by the resource type provider.
func ValidateResource(
	ctx context.Context,
	name string,
	resource *schema.Resource,
	resourceMap *schema.ResourceMap,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	resourceDerivedFromTemplate bool,
	logger bpcore.Logger,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	var errs []error

	validateTypeDiagnostics, validateTypeErr := validateResourceType(
		ctx,
		name,
		resource.Type,
		resourceMap,
		resourceRegistry,
	)
	diagnostics = append(diagnostics, validateTypeDiagnostics...)
	if validateTypeErr != nil {
		errs = append(errs, validateTypeErr)
	}

	logger.Debug("Validating resource metadata")
	validateMetadataDiagnostics, validateMetadataErr := validateResourceMetadata(
		ctx,
		name,
		resourceDerivedFromTemplate,
		resource.Metadata,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, validateMetadataDiagnostics...)
	if validateMetadataErr != nil {
		errs = append(errs, validateMetadataErr)
	}

	logger.Debug("Validating resource dependencies")
	validateResDepsDiagnostics, validateResDepsErr := validateResourceDependencies(
		ctx,
		name,
		resource.DependsOn,
		bpSchema,
		refChainCollector,
	)
	diagnostics = append(diagnostics, validateResDepsDiagnostics...)
	if validateResDepsErr != nil {
		errs = append(errs, validateResDepsErr)
	}

	logger.Debug("Validating resource condition")
	validateResConditionDiagnostics, validateResConditionErr := validateResourceCondition(
		ctx,
		name,
		resourceDerivedFromTemplate,
		resource.Condition,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
		/* depth */ 0,
	)
	diagnostics = append(diagnostics, validateResConditionDiagnostics...)
	if validateResConditionErr != nil {
		errs = append(errs, validateResConditionErr)
	}

	logger.Debug("Validating resource each property (for resource templates)")
	validateEachDiagnostics, validateEachErr := validateResourceEach(
		ctx,
		name,
		resourceDerivedFromTemplate,
		resource.Each,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, validateEachDiagnostics...)
	if validateEachErr != nil {
		errs = append(errs, validateEachErr)
	}

	logger.Debug("Validating resource link selector")
	validateLSDiagnostics, validateLSErr := validateResourceLinkSelector(
		name,
		resource.LinkSelector,
	)
	diagnostics = append(diagnostics, validateLSDiagnostics...)
	if validateLSErr != nil {
		errs = append(errs, validateLSErr)
	}

	// Only validate spec if type validation passed - there's no point validating
	// the spec when we can't look up the spec definition for an unknown resource type.
	if resource.Type != nil && validateTypeErr == nil {
		logger.Debug("Validating resource spec")
		validationParams := ResourceValidationParams{
			ResourceName:                name,
			ResourceType:                resource.Type.Value,
			ResourceDerivedFromTemplate: resourceDerivedFromTemplate,
			BpSchema:                    bpSchema,
			Params:                      params,
			FuncRegistry:                funcRegistry,
			RefChainCollector:           refChainCollector,
			ResourceRegistry:            resourceRegistry,
			DataSourceRegistry:          dataSourceRegistry,
		}
		validateSpecDiagnostics, validateSpecErr := ValidateResourceSpec(
			ctx,
			validationParams,
			resource,
			resourceMap.SourceMeta[name],
		)
		diagnostics = append(diagnostics, validateSpecDiagnostics...)
		if validateSpecErr != nil {
			errs = append(errs, validateSpecErr)
		}
	}

	logger.Debug("Validating resource description")
	validateDescriptionDiagnostics, validateDescErr := validateDescription(
		ctx,
		bpcore.ResourceElementID(name),
		resourceDerivedFromTemplate,
		resource.Description,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, validateDescriptionDiagnostics...)
	if validateDescErr != nil {
		errs = append(errs, validateDescErr)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateResourceType(
	ctx context.Context,
	resourceName string,
	resourceType *schema.ResourceTypeWrapper,
	resourceMap *schema.ResourceMap,
	resourceRegistry resourcehelpers.Registry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if resourceType == nil {
		return diagnostics, errResourceMissingType(
			resourceName,
			getResourceSourceMeta(resourceMap, resourceName),
		)
	}

	// Use the type definition's location for more precise error reporting
	location := resourceType.SourceMeta
	hasType, err := resourceRegistry.HasResourceType(ctx, resourceType.Value)
	if err != nil {
		return diagnostics, wrapRegistryError(err, location)
	}

	if !hasType {
		return diagnostics, errResourceTypeNotSupported(
			resourceName,
			resourceType.Value,
			location,
		)
	}

	return diagnostics, nil
}

func validateResourceMetadata(
	ctx context.Context,
	resourceName string,
	resourceDerivedFromTemplate bool,
	metadataSchema *schema.Metadata,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if metadataSchema == nil {
		return diagnostics, nil
	}

	var errs []error

	displayNameDiagnostics, err := validateResourceMetadataDisplayName(
		ctx,
		resourceName,
		resourceDerivedFromTemplate,
		metadataSchema,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, displayNameDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	labelDiagnostics, err := validateResourceMetadataLabels(
		resourceName,
		metadataSchema,
	)
	diagnostics = append(diagnostics, labelDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	annotationsDiagnostics, err := validateResourceMetadataAnnotations(
		ctx,
		resourceName,
		metadataSchema,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, annotationsDiagnostics...)
	if err != nil {
		errs = append(errs, err)
	}

	customDiagnostics, err := ValidateMappingNode(
		ctx,
		bpcore.ResourceElementID(resourceName),
		"metadata.custom",
		resourceDerivedFromTemplate,
		metadataSchema.Custom,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
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

func validateResourceMetadataDisplayName(
	ctx context.Context,
	resourceName string,
	resourceDerivedFromTemplate bool,
	metadataSchema *schema.Metadata,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	if metadataSchema.DisplayName == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	resourceIdentifier := bpcore.ResourceElementID(resourceName)
	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for _, stringOrSub := range metadataSchema.DisplayName.Values {
		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				bpSchema,
				resourceDerivedFromTemplate,
				resourceIdentifier,
				"metadata.displayName",
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
					errs = append(errs, errInvalidDisplayNameSubType(
						resourceIdentifier,
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

func validateResourceMetadataLabels(
	resourceName string,
	metadataSchema *schema.Metadata,
) ([]*bpcore.Diagnostic, error) {
	if metadataSchema.Labels == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for key, value := range metadataSchema.Labels.Values {
		if substitutions.ContainsSubstitution(key) {
			errs = append(errs, errLabelKeyContainsSubstitution(
				resourceName,
				key,
				metadataSchema.Labels.SourceMeta[key],
			))
		}

		if substitutions.ContainsSubstitution(value) {
			errs = append(errs, errLabelValueContainsSubstitution(
				resourceName,
				key,
				value,
				metadataSchema.Labels.SourceMeta[key],
			))
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateResourceMetadataAnnotations(
	ctx context.Context,
	resourceName string,
	metadataSchema *schema.Metadata,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	if metadataSchema.Annotations == nil || metadataSchema.Annotations.Values == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	resourceIdentifier := bpcore.ResourceElementID(resourceName)
	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for key, annotation := range metadataSchema.Annotations.Values {
		if substitutions.ContainsSubstitution(key) {
			errs = append(errs, errAnnotationKeyContainsSubstitution(
				resourceName,
				key,
				annotation.SourceMeta,
			))
		}

		annotationDiagnostics, err := validateMetadataAnnotation(
			ctx,
			resourceIdentifier,
			annotation,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
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

func validateResourceDependencies(
	ctx context.Context,
	resourceName string,
	dependsOn *schema.DependsOnList,
	blueprint *schema.Blueprint,
	refChainCollector refgraph.RefChainCollector,
) ([]*bpcore.Diagnostic, error) {
	if dependsOn == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	for i, dependency := range dependsOn.Values {
		if substitutions.ContainsSubstitution(dependency) {
			errs = append(errs, errResourceDependencyContainsSubstitution(
				resourceName,
				dependency,
				dependsOn.SourceMeta[i],
			))
		}

		dependencyResource, hasResource := getResource(dependency, blueprint)
		if !hasResource {
			errs = append(errs, errResourceDependencyMissing(
				resourceName,
				dependency,
				dependsOn.SourceMeta[i],
			))
		}

		if resourceName == dependency {
			errs = append(errs, errSelfReferencingResourceDependency(
				resourceName,
				dependsOn.SourceMeta[i],
			))
		}

		// Collect reference in the ref chain collector for the dependency to cover
		// cycle detection across references, dependsOn and links.
		resourceID := bpcore.ResourceElementID(dependency)
		referencedByResourceID := bpcore.ResourceElementID(resourceName)
		dependencyTag := CreateDependencyRefTag(referencedByResourceID)
		err := refChainCollector.Collect(
			resourceID,
			dependencyResource,
			referencedByResourceID,
			[]string{dependencyTag},
		)
		if err != nil {
			return []*bpcore.Diagnostic{}, err
		}
	}

	if len(errs) > 0 {
		return []*bpcore.Diagnostic{}, ErrMultipleValidationErrors(errs)
	}

	return []*bpcore.Diagnostic{}, nil
}

func validateResourceCondition(
	ctx context.Context,
	resourceName string,
	resourceDerivedFromTemplate bool,
	conditionSchema *schema.Condition,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	depth int,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if conditionSchema == nil && depth == 0 {
		return diagnostics, nil
	}

	if (conditionSchema == nil || allConditionValuesNil(conditionSchema)) && depth > 0 {
		// Nested conditions should not be empty.
		return diagnostics, errNestedResourceConditionEmpty(
			resourceName,
			conditionSchema.SourceMeta,
		)
	}

	var errs []error
	if conditionSchema.And != nil {
		for _, andCondition := range conditionSchema.And {
			andDiagnostics, err := validateResourceCondition(
				ctx,
				resourceName,
				resourceDerivedFromTemplate,
				andCondition,
				bpSchema,
				params,
				funcRegistry,
				refChainCollector,
				resourceRegistry,
				dataSourceRegistry,
				depth+1,
			)
			diagnostics = append(diagnostics, andDiagnostics...)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if conditionSchema.Or != nil {
		for _, orCondition := range conditionSchema.Or {
			orDiagnostics, err := validateResourceCondition(
				ctx,
				resourceName,
				resourceDerivedFromTemplate,
				orCondition,
				bpSchema,
				params,
				funcRegistry,
				refChainCollector,
				resourceRegistry,
				dataSourceRegistry,
				depth+1,
			)
			diagnostics = append(diagnostics, orDiagnostics...)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	if conditionSchema.Not != nil {
		notDiagnostics, err := validateResourceCondition(
			ctx,
			resourceName,
			resourceDerivedFromTemplate,
			conditionSchema.Not,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
			depth+1,
		)
		diagnostics = append(diagnostics, notDiagnostics...)
		if err != nil {
			return diagnostics, err
		}
	}

	conditionValDiagnostics, err := validateConditionValue(
		ctx,
		resourceName,
		resourceDerivedFromTemplate,
		conditionSchema.StringValue,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
	diagnostics = append(diagnostics, conditionValDiagnostics...)
	if err != nil {
		return diagnostics, err
	}

	return diagnostics, nil
}

func validateConditionValue(
	ctx context.Context,
	resourceName string,
	resourceDerivedFromTemplate bool,
	conditionValue *substitutions.StringOrSubstitutions,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	if conditionValue == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}

	resourceIdentifier := bpcore.ResourceElementID(resourceName)

	if len(conditionValue.Values) > 1 {
		return diagnostics, errInvalidSubTypeNotBoolean(
			resourceIdentifier,
			"condition",
			// StringOrSubstitutions with multiple values is an
			// interpolated string.
			string(substitutions.ResolvedSubExprTypeString),
			conditionValue.SourceMeta,
		)
	}

	for i, stringOrSub := range conditionValue.Values {
		nextLocation := getSubNextLocation(i, conditionValue.Values)

		if stringOrSub.SubstitutionValue != nil {
			resolvedType, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nil,
				bpSchema,
				resourceDerivedFromTemplate,
				resourceIdentifier,
				"condition",
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
				handleResolvedTypeExpectingBoolean(
					resolvedType,
					resourceIdentifier,
					stringOrSub,
					conditionValue,
					"condition",
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

func handleResolvedTypeExpectingBoolean(
	resolvedType string,
	itemIdentifier string,
	stringOrSub *substitutions.StringOrSubstitution,
	value *substitutions.StringOrSubstitutions,
	valueContext string,
	nextLocation *source.Meta,
	diagnostics *[]*bpcore.Diagnostic,
	errs *[]error,
) {
	if resolvedType != string(substitutions.ResolvedSubExprTypeBoolean) &&
		resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
		*errs = append(*errs, errInvalidSubTypeNotBoolean(
			itemIdentifier,
			valueContext,
			resolvedType,
			stringOrSub.SourceMeta,
		))
	} else if resolvedType == string(substitutions.ResolvedSubExprTypeAny) {
		// Any type will produce a warning diagnostic as any could match a boolean
		// value in a context where the developer is confident a boolean value will
		// be resolved.
		*diagnostics = append(
			*diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Substitution returns \"any\" type, this may produce "+
						"unexpected output in the %s, %ss are expected to be boolean values",
					valueContext,
					valueContext,
				),
				Range: bpcore.DiagnosticRangeFromSourceMeta(value.SourceMeta, nextLocation),
			},
		)
	}
}

func validateResourceEach(
	ctx context.Context,
	resourceName string,
	resourceDerivedFromTemplate bool,
	each *substitutions.StringOrSubstitutions,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	// Only validate when a user has provided an empty array as a value
	// for the each property. A nil slice is a default empty value that
	// indicates no value was provided by the user.
	if each == nil || each.Values == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	diagnostics := []*bpcore.Diagnostic{}

	resourceIdentifier := bpcore.ResourceElementID(resourceName)

	if len(each.Values) == 0 {
		return diagnostics, errEmptyEachSubstitution(
			resourceIdentifier,
			"each",
			each.SourceMeta,
		)
	}

	if len(each.Values) > 1 {
		return diagnostics, errInvalidSubTypeNotArray(
			resourceIdentifier,
			"each",
			// StringOrSubstitutions with multiple values is an
			// interpolated string.
			string(substitutions.ResolvedSubExprTypeString),
			each.SourceMeta,
		)
	}

	stringOrSub := each.Values[0]
	nextLocation := getSubNextLocation(0, each.Values)

	if stringOrSub.SubstitutionValue != nil {
		resolvedType, subDiagnostics, err := ValidateSubstitution(
			ctx,
			stringOrSub.SubstitutionValue,
			nil,
			bpSchema,
			resourceDerivedFromTemplate,
			resourceIdentifier,
			"each",
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
		if err != nil {
			return diagnostics, err
		}

		var errs []error
		diagnostics = append(diagnostics, subDiagnostics...)
		handleResolvedTypeExpectingArray(
			resolvedType,
			resourceIdentifier,
			stringOrSub,
			each,
			"each",
			nextLocation,
			&diagnostics,
			&errs,
		)

		if len(errs) > 0 {
			return diagnostics, errs[0]
		}
	}

	return diagnostics, nil
}

func handleResolvedTypeExpectingArray(
	resolvedType string,
	itemIdentifier string,
	stringOrSub *substitutions.StringOrSubstitution,
	value *substitutions.StringOrSubstitutions,
	valueContext string,
	nextLocation *source.Meta,
	diagnostics *[]*bpcore.Diagnostic,
	errs *[]error,
) {
	if resolvedType != string(substitutions.ResolvedSubExprTypeArray) &&
		resolvedType != string(substitutions.ResolvedSubExprTypeAny) {
		*errs = append(*errs, errInvalidSubTypeNotArray(
			itemIdentifier,
			valueContext,
			resolvedType,
			stringOrSub.SourceMeta,
		))
	} else if resolvedType == string(substitutions.ResolvedSubExprTypeAny) {
		// Any type will produce a warning diagnostic as any could match an array,
		// an error will occur at runtime if the resolved value is not an array.
		*diagnostics = append(
			*diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Substitution returns \"any\" type, this may produce "+
						"unexpected output in %s, an array is expected",
					valueContext,
				),
				Range: bpcore.DiagnosticRangeFromSourceMeta(value.SourceMeta, nextLocation),
			},
		)
	}
}

// ValidateResourceEachDependencies validates the dependencies of the `each`
// property of a resource.
// This should be called after all validation of a blueprint has been carried out
// and the full set of references have been collected.
func ValidateResourceEachDependencies(
	blueprint *schema.Blueprint,
	refChainCollector refgraph.RefChainCollector,
) error {
	if blueprint.Resources == nil {
		return nil
	}

	var errs []error
	for resourceName, resource := range blueprint.Resources.Values {
		if !substitutions.IsNilStringSubs(resource.Each) {
			resourceIdentifier := bpcore.ResourceElementID(resourceName)
			eachTag := CreateSubRefPropTag(resourceIdentifier, "each")
			nodes := refChainCollector.FindByTag(eachTag)
			if len(nodes) > 0 {
				errsForCurrentResource := checkEachResourceOrChildDependencies(
					nodes,
					resourceIdentifier,
					resource.Each.SourceMeta,
					[]error{},
				)
				if len(errsForCurrentResource) > 0 {
					errs = append(errs, errsForCurrentResource...)
				}
			}
		}
	}

	if len(errs) > 0 {
		return ErrMultipleValidationErrors(errs)
	}

	return nil
}

func checkEachResourceOrChildDependencies(
	nodes []*refgraph.ReferenceChainNode,
	resourceIdentifier string,
	eachLocation *source.Meta,
	errs []error,
) []error {
	for _, node := range nodes {
		if _, isResource := node.Element.(*schema.Resource); isResource {
			errs = append(errs, errEachResourceDependencyDetected(
				resourceIdentifier,
				node.ElementName,
				eachLocation,
			))
		} else if _, isChild := node.Element.(*schema.Include); isChild {
			errs = append(errs, errEachChildDependencyDetected(
				resourceIdentifier,
				node.ElementName,
				eachLocation,
			))
		} else {
			errs = checkEachResourceOrChildDependencies(
				node.References,
				resourceIdentifier,
				eachLocation,
				errs,
			)
		}
	}

	return errs
}

func validateResourceLinkSelector(
	resourceName string,
	linkSelector *schema.LinkSelector,
) ([]*bpcore.Diagnostic, error) {
	if linkSelector == nil || linkSelector.ByLabel == nil {
		return []*bpcore.Diagnostic{}, nil
	}

	errs := []error{}
	diagnostics := []*bpcore.Diagnostic{}
	for key, value := range linkSelector.ByLabel.Values {
		if substitutions.ContainsSubstitution(key) {
			errs = append(errs, errLinkSelectorKeyContainsSubstitution(
				resourceName,
				key,
				linkSelector.ByLabel.SourceMeta[key],
			))
		}

		if substitutions.ContainsSubstitution(value) {
			errs = append(errs, errLinkSelectorValueContainsSubstitution(
				resourceName,
				key,
				value,
				linkSelector.ByLabel.SourceMeta[key],
			))
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func allConditionValuesNil(condition *schema.Condition) bool {
	return condition.And == nil && condition.Or == nil &&
		condition.Not == nil && condition.StringValue == nil
}

func getResourceSourceMeta(resourceMap *schema.ResourceMap, resourceName string) *source.Meta {
	if resourceMap == nil {
		return nil
	}

	return resourceMap.SourceMeta[resourceName]
}

func getResource(resourceName string, blueprint *schema.Blueprint) (*schema.Resource, bool) {
	if blueprint.Resources == nil {
		return nil, false
	}

	resource, hasResource := blueprint.Resources.Values[resourceName]
	return resource, hasResource
}
