package validation

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	bpcore "github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/refgraph"
	"github.com/two-hundred/celerity/libs/blueprint/resourcehelpers"
	"github.com/two-hundred/celerity/libs/blueprint/schema"
	"github.com/two-hundred/celerity/libs/blueprint/source"
	"github.com/two-hundred/celerity/libs/blueprint/substitutions"
	"github.com/two-hundred/celerity/libs/common/core"
)

// ValidateSubstitution validates a substitution usage in a blueprint.
//
// usedIn is the path to the element in the blueprint where the substitution is used.
// This should be in the format of "{elementType}.{elementName}"
// For example, "values.myValue" or "resources.myResource"
//
// funcRegistry provides a registry of functions that can be used in the substitution.
// resourceRegistry provides a registry of resource types that are used to check accessed
// attributes against the resource spec.
//
// This returns a string containing the type of the resolved value for the substitution
// where it can be determined, an empty string otherwise.
// The caller is responsible for ensuring that the resolved value type is compatible with
// the context where the substitution is used.
// It also returns a list of diagnostics that were generated during the
// validation process and an error if the validation process failed.
func ValidateSubstitution(
	ctx context.Context,
	sub *substitutions.Substitution,
	nextLocation *source.Meta,
	bpSchema *schema.Blueprint,
	usedInResourceDerivedFromTemplate bool,
	usedIn string,
	// The path to the property where the substitution is used
	// relative to the "usedIn" element.
	usedInPropertyPath string,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
) (string, []*bpcore.Diagnostic, error) {

	diagnostics := []*bpcore.Diagnostic{}
	if sub == nil {
		return "", diagnostics, nil
	}

	if sub.Function != nil {
		return validateFunctionSubstitution(
			ctx,
			sub.Function,
			nextLocation,
			bpSchema,
			usedInResourceDerivedFromTemplate,
			usedIn,
			usedInPropertyPath,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
		)
	}

	if sub.BoolValue != nil {
		return string(substitutions.ResolvedSubExprTypeBoolean), diagnostics, nil
	}

	if sub.StringValue != nil {
		return string(substitutions.ResolvedSubExprTypeString), diagnostics, nil
	}

	if sub.IntValue != nil {
		return string(substitutions.ResolvedSubExprTypeInteger), diagnostics, nil
	}

	if sub.FloatValue != nil {
		return string(substitutions.ResolvedSubExprTypeFloat), diagnostics, nil
	}

	if sub.Variable != nil {
		return validateVariableSubstitution(sub.Variable, bpSchema)
	}

	if sub.ValueReference != nil {
		return validateValueSubstitution(
			sub.ValueReference,
			bpSchema,
			usedIn,
			usedInPropertyPath,
			refChainCollector,
		)
	}

	if sub.ElemReference != nil {
		return validateElemReferenceSubstitution(
			"element",
			sub.ElemReference.SourceMeta,
			bpSchema,
			usedInResourceDerivedFromTemplate,
			usedIn,
		)
	}

	if sub.ElemIndexReference != nil {
		return validateElemReferenceSubstitution(
			"index",
			sub.ElemIndexReference.SourceMeta,
			bpSchema,
			usedInResourceDerivedFromTemplate,
			usedIn,
		)
	}

	if sub.ResourceProperty != nil {
		return validateResourcePropertySubstitution(
			ctx,
			sub.ResourceProperty,
			bpSchema,
			usedIn,
			usedInPropertyPath,
			params,
			refChainCollector,
			resourceRegistry,
			nextLocation,
		)
	}

	if sub.DataSourceProperty != nil {
		return validateDataSourcePropertySubstitution(
			sub.DataSourceProperty,
			bpSchema,
			usedIn,
			usedInPropertyPath,
			refChainCollector,
		)
	}

	if sub.Child != nil {
		return validateChildSubstitution(
			sub.Child,
			bpSchema,
			usedIn,
			usedInPropertyPath,
			refChainCollector,
		)
	}

	return "", diagnostics, nil
}

func validateVariableSubstitution(
	subVar *substitutions.SubstitutionVariable,
	bpSchema *schema.Blueprint,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	varName := subVar.VariableName

	if bpSchema.Variables == nil || bpSchema.Variables.Values == nil {
		return "", diagnostics, errSubVarNotFound(varName, subVar.SourceMeta)
	}

	varSchema, hasVar := bpSchema.Variables.Values[varName]
	if !hasVar {
		return "", diagnostics, errSubVarNotFound(varName, subVar.SourceMeta)
	}

	// Variable references aren't collected with the reference cycle service
	// as cycles involving variable references are not possible;
	// this is because references are not
	// supported in variable definitions.

	return subVarType(varSchema.Type), diagnostics, nil
}

func validateValueSubstitution(
	subVal *substitutions.SubstitutionValueReference,
	bpSchema *schema.Blueprint,
	usedIn string,
	usedInPropertyPath string,
	refChainCollector refgraph.RefChainCollector,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	valName := subVal.ValueName

	if bpSchema.Values == nil || bpSchema.Values.Values == nil {
		return "", diagnostics, errSubValNotFound(valName, subVal.SourceMeta)
	}

	valSchema, hasVal := bpSchema.Values.Values[valName]
	if !hasVal {
		return "", diagnostics, errSubValNotFound(valName, subVal.SourceMeta)
	}

	if usedIn == bpcore.ValueElementID(valName) {
		return "", diagnostics, errSubValSelfReference(valName, subVal.SourceMeta)
	}

	subRefTag := CreateSubRefTag(usedIn)
	subRefPropTag := CreateSubRefPropTag(usedIn, usedInPropertyPath)
	refChainCollector.Collect(
		bpcore.ValueElementID(valName),
		valSchema,
		usedIn,
		[]string{subRefTag, subRefPropTag},
	)

	if len(subVal.Path) >= 1 {
		// At this point, we can't know the exact type of the value reference without
		// inspecting the contents of the value definition, this could be quite an expensive operation
		// traversing through multiple levels of references and definitions.
		// When a nested attribute or index is accessed from a value, at validation time
		// we return any to account for all possible types.
		return string(substitutions.ResolvedSubExprTypeAny), diagnostics, nil
	}

	return subValType(valSchema.Type), diagnostics, nil
}

func validateElemReferenceSubstitution(
	elemRefType string,
	location *source.Meta,
	bpSchema *schema.Blueprint,
	derivedFromTemplate bool,
	usedIn string,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if !strings.HasPrefix(usedIn, "resources.") {
		return "", diagnostics, errSubElemRefNotInResource(elemRefType, location)
	}

	if bpSchema.Resources == nil || bpSchema.Resources.Values == nil {
		return "", diagnostics, errSubElemRefNotInResource(elemRefType, location)
	}

	resourceName := usedIn[10:]
	resource, hasResource := bpSchema.Resources.Values[resourceName]
	if !hasResource {
		return "", diagnostics, errSubElemRefResourceNotFound(elemRefType, resourceName, location)
	}

	if !derivedFromTemplate && resource.Each == nil {
		return "", diagnostics, errSubElemRefResourceNotEach(elemRefType, resourceName, location)
	}

	// Element (and element index) references aren't collected with the reference cycle service
	// as cycles involving element references are not possible;
	// this is because elements are a reference to a local value that is only
	// scoped to the resource where the `each` property is defined.

	// The type of an element reference isn't known until runtime
	// as it dependent on the `each` property of the resource.
	resolvedType := string(substitutions.ResolvedSubExprTypeAny)
	if elemRefType == "index" {
		resolvedType = string(substitutions.ResolvedSubExprTypeInteger)
	}
	return resolvedType, diagnostics, nil
}

func validateResourcePropertySubstitution(
	ctx context.Context,
	subResourceProp *substitutions.SubstitutionResourceProperty,
	bpSchema *schema.Blueprint,
	usedIn string,
	usedInPropertyPath string,
	params bpcore.BlueprintParams,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	nextLocation *source.Meta,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	resourceName := subResourceProp.ResourceName

	if bpSchema.Resources == nil || bpSchema.Resources.Values == nil {
		return "", diagnostics, errSubResourceNotFound(resourceName, subResourceProp.SourceMeta)
	}

	resourceSchema, hasResource := bpSchema.Resources.Values[resourceName]
	if !hasResource {
		return "", diagnostics, errSubResourceNotFound(resourceName, subResourceProp.SourceMeta)
	}

	if resourceSchema.Type == nil {
		return "", diagnostics, errResourceMissingType(resourceName, subResourceProp.SourceMeta)
	}

	if usedIn == bpcore.ResourceElementID(resourceName) {
		return "", diagnostics, errSubResourceSelfReference(resourceName, subResourceProp.SourceMeta)
	}

	if subResourceProp.ResourceEachTemplateIndex != nil && resourceSchema.Each == nil {
		return "", diagnostics, errSubResourceNotEach(
			resourceName,
			subResourceProp.ResourceEachTemplateIndex,
			subResourceProp.SourceMeta,
		)
	}

	resourcePropType, extractDiagnostics, err := extractResourcePropertySubType(
		ctx,
		subResourceProp,
		resourceSchema,
		resourceRegistry,
		nextLocation,
		params,
	)
	diagnostics = append(diagnostics, extractDiagnostics...)
	if err != nil {
		return "", diagnostics, err
	}

	subRefTag := CreateSubRefTag(usedIn)
	subRefPropTag := CreateSubRefPropTag(usedIn, usedInPropertyPath)
	refChainCollector.Collect(
		bpcore.ResourceElementID(resourceName),
		resourceSchema,
		usedIn,
		[]string{subRefTag, subRefPropTag},
	)

	return resourcePropType, diagnostics, nil
}

func extractResourcePropertySubType(
	ctx context.Context,
	subResourceProp *substitutions.SubstitutionResourceProperty,
	resourceSchema *schema.Resource,
	resourceRegistry resourcehelpers.Registry,
	nextLocation *source.Meta,
	params bpcore.BlueprintParams,
) (string, []*bpcore.Diagnostic, error) {
	if len(subResourceProp.Path) > 0 && subResourceProp.Path[0].FieldName == "spec" {
		return validateResourcePropertySubSpec(
			ctx,
			subResourceProp,
			resourceSchema.Type.Value,
			resourceRegistry,
			nextLocation,
			params,
		)
	}

	if len(subResourceProp.Path) > 0 && subResourceProp.Path[0].FieldName == "metadata" {
		return validateResourcePropertySubMetadata(
			subResourceProp,
			resourceSchema,
		)
	}

	return string(substitutions.ResolvedSubExprTypeAny), []*bpcore.Diagnostic{}, nil
}

func validateResourcePropertySubSpec(
	ctx context.Context,
	subResourceProp *substitutions.SubstitutionResourceProperty,
	resourceType string,
	resourceRegistry resourcehelpers.Registry,
	nextLocation *source.Meta,
	params bpcore.BlueprintParams,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if len(subResourceProp.Path) < 2 {
		return "", diagnostics, errSubResourceSpecInvalidRef(
			subResourceProp.ResourceName,
			subResourceProp.SourceMeta,
		)
	}

	earlyResolveType, err := checkResourceType(
		ctx,
		resourceType,
		resourceRegistry,
		subResourceProp,
		nextLocation,
		&diagnostics,
	)
	if err != nil {
		return "", diagnostics, err
	}
	if earlyResolveType != "" {
		return earlyResolveType, diagnostics, nil
	}

	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	specDefOutput, err := resourceRegistry.GetSpecDefinition(
		ctx,
		resourceType,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				params,
			),
		},
	)
	if err != nil {
		return "", diagnostics, errResourceTypeMissingSpecDefinition(
			subResourceProp.ResourceName,
			resourceType,
			/* inSubstitution */ true,
			subResourceProp.SourceMeta,
			"failed to load spec definition",
		)
	}

	if specDefOutput.SpecDefinition == nil {
		return "", diagnostics, errResourceTypeMissingSpecDefinition(
			subResourceProp.ResourceName,
			resourceType,
			/* inSubstitution */ true,
			subResourceProp.SourceMeta,
			"spec definition is nil",
		)
	}

	return validateResourcePropertySubDefinitionsPath(
		subResourceProp,
		resourceType,
		specDefOutput.SpecDefinition.Schema,
	)
}

func validateResourcePropertySubDefinitionsPath(
	subResourceProp *substitutions.SubstitutionResourceProperty,
	resourceType string,
	definitionSchema *provider.ResourceDefinitionsSchema,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	resolvedType := ""
	propertyMatches := true
	currentSchema := definitionSchema
	if currentSchema == nil {
		return "", diagnostics, errResourceTypeSpecDefMissingSchema(
			subResourceProp.ResourceName,
			resourceType,
			/* inSubstitution */ true,
			subResourceProp.SourceMeta,
		)
	}

	// At this point we already know the first element in the path is "spec".
	i := 1
	for propertyMatches && i < len(subResourceProp.Path) {
		property := subResourceProp.Path[i]
		if property.FieldName != "" && len(currentSchema.Attributes) > 0 {
			var attrMatches bool
			var attrType string
			attrMatches, attrType, currentSchema = checkSubResourcePropertyAttr(
				subResourceProp,
				currentSchema,
				property,
				i,
			)
			if !attrMatches {
				propertyMatches = false
			}
			if i == len(subResourceProp.Path)-1 {
				resolvedType = attrType
			}
		} else if property.ArrayIndex != nil &&
			currentSchema.Type == provider.ResourceDefinitionsSchemaTypeArray {
			currentSchema = currentSchema.Items
			if i == len(subResourceProp.Path)-1 {
				resolvedType = string(substitutions.ResolvedSubExprTypeArray)
			}
		} else {
			propertyMatches = false
		}
		i += 1
	}

	if !propertyMatches {
		return "", diagnostics, errSubResourcePropertyNotFound(
			subResourceProp.ResourceName,
			subResourceProp.Path,
			subResourceProp.SourceMeta,
		)
	}

	return resolvedType, diagnostics, nil
}

func checkResourceType(
	ctx context.Context,
	resourceType string,
	resourceRegistry resourcehelpers.Registry,
	subResourceProp *substitutions.SubstitutionResourceProperty,
	nextLocation *source.Meta,
	diagnostics *[]*bpcore.Diagnostic,
) (string, error) {
	hasResType, err := resourceRegistry.HasResourceType(ctx, resourceType)
	if err != nil {
		return "", err
	}

	if !hasResType {
		// If the resource type has not been loaded,
		// we can't know whether or not the accessed property of the
		// resource state is valid.
		// We return any to account for all possible types and a warning
		// to indicate that the resource type has not been loaded
		// and it will need to be loaded for change staging and deployment.
		*diagnostics = append(
			*diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Resource type %q is not currently loaded, when staging changes and deploying,"+
						" you will need to make sure the provider for the resource type is loaded.",
					resourceType,
				),
				Range: toDiagnosticRange(subResourceProp.SourceMeta, nextLocation),
			},
		)

		return string(substitutions.ResolvedSubExprTypeAny), nil
	}

	return "", nil
}

func validateResourcePropertySubMetadata(
	subResourceProp *substitutions.SubstitutionResourceProperty,
	blueprintResource *schema.Resource,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if len(subResourceProp.Path) < 2 {
		return "", diagnostics, errSubResourceMetadataInvalidRef(
			subResourceProp.ResourceName,
			subResourceProp.SourceMeta,
		)
	}

	if !isResourceMetadataProperty(subResourceProp.Path[1].FieldName) {
		return "", diagnostics, errSubResourceMetadataInvalidProperty(
			subResourceProp.ResourceName,
			subResourceProp.Path[1].FieldName,
			subResourceProp.SourceMeta,
		)
	}

	if subResourceProp.Path[1].FieldName == "displayName" {
		if len(subResourceProp.Path) > 2 {
			return "", diagnostics, errSubResourceMetadataInvalidDisplayNameRef(
				subResourceProp.ResourceName,
				subResourceProp.SourceMeta,
			)
		}

		return string(substitutions.ResolvedSubExprTypeString), diagnostics, nil
	}

	if subResourceProp.Path[1].FieldName == "annotations" {
		err := validateResourcePropertySubMetadataAnnotations(
			subResourceProp,
			blueprintResource,
		)
		if err != nil {
			return "", diagnostics, err
		}

		// To get a precise type for each annotation, we would need to traverse
		// substitution trees, which was deemed more effort than it was worth
		// in the initial version.
		// Runtime checks for the types of annotations will have to suffice.
		return string(substitutions.ResolvedSubExprTypeAny), diagnostics, nil
	}

	if subResourceProp.Path[1].FieldName == "labels" {
		err := validateResourcePropertySubMetadataLabels(
			subResourceProp,
			blueprintResource,
		)
		if err != nil {
			return "", diagnostics, err
		}

		return string(substitutions.ResolvedSubExprTypeString), diagnostics, nil
	}

	// Custom metadata is a free-form object, at the time of implementation,
	// deep validation wasn't deemed necessary due to the likelihood of `metadata.custom`
	// being used in substitution references being low.
	// This judgement was made as custom metadata is primarily used for storing information
	// to be used by external systems.
	return string(substitutions.ResolvedSubExprTypeAny), diagnostics, nil
}

func validateResourcePropertySubMetadataAnnotations(
	subResourceProp *substitutions.SubstitutionResourceProperty,
	blueprintResource *schema.Resource,
) error {
	if len(subResourceProp.Path) != 3 {
		return errSubResourceMetadataInvalidAnnotationsRef(
			subResourceProp.ResourceName,
			subResourceProp.SourceMeta,
		)
	}

	hasAnnotation := checkResourceHasAnnotation(
		subResourceProp.Path[2].FieldName,
		blueprintResource,
	)

	if !hasAnnotation {
		return errSubResourceMetadataMissingAnnotation(
			subResourceProp.ResourceName,
			subResourceProp.Path[2].FieldName,
			subResourceProp.SourceMeta,
		)
	}

	return nil
}

func checkResourceHasAnnotation(fieldName string, blueprintResource *schema.Resource) bool {
	if blueprintResource.Metadata == nil {
		return false
	}

	if blueprintResource.Metadata.Annotations == nil {
		return false
	}

	if blueprintResource.Metadata.Annotations.Values == nil {
		return false
	}

	_, hasAnnotation := blueprintResource.Metadata.Annotations.Values[fieldName]
	return hasAnnotation
}

func validateResourcePropertySubMetadataLabels(
	subResourceProp *substitutions.SubstitutionResourceProperty,
	blueprintResource *schema.Resource,
) error {
	if len(subResourceProp.Path) != 3 {
		return errSubResourceMetadataInvalidLabelsRef(
			subResourceProp.ResourceName,
			subResourceProp.SourceMeta,
		)
	}

	hasLabel := checkResourceHasLabel(
		subResourceProp.Path[2].FieldName,
		blueprintResource,
	)

	if !hasLabel {
		return errSubResourceMetadataMissingLabel(
			subResourceProp.ResourceName,
			subResourceProp.Path[2].FieldName,
			subResourceProp.SourceMeta,
		)
	}

	return nil
}

func checkResourceHasLabel(fieldName string, blueprintResource *schema.Resource) bool {
	if blueprintResource.Metadata == nil {
		return false
	}

	if blueprintResource.Metadata.Labels == nil {
		return false
	}

	if blueprintResource.Metadata.Labels.Values == nil {
		return false
	}

	_, hasLabel := blueprintResource.Metadata.Labels.Values[fieldName]
	return hasLabel
}

func isResourceMetadataProperty(fieldName string) bool {
	return fieldName == "displayName" ||
		fieldName == "annotations" ||
		fieldName == "labels" ||
		fieldName == "custom"
}

func checkSubResourcePropertyAttr(
	subResourceProp *substitutions.SubstitutionResourceProperty,
	currentSchema *provider.ResourceDefinitionsSchema,
	property *substitutions.SubstitutionPathItem,
	index int,
) (bool, string, *provider.ResourceDefinitionsSchema) {
	attrSchema, hasAttr := currentSchema.Attributes[property.FieldName]
	if hasAttr {
		if index < len(subResourceProp.Path)-1 && !isComplexResourceDefinitionsSchemaType(attrSchema.Type) {
			// Path is trying to access properties of a primitive type.
			return false, "", nil
		}
		return true, subResourceDefinitionsSchemaType(attrSchema.Type), attrSchema
	}

	return false, "", nil
}

func isComplexResourceDefinitionsSchemaType(schemaType provider.ResourceDefinitionsSchemaType) bool {
	return schemaType == provider.ResourceDefinitionsSchemaTypeObject ||
		schemaType == provider.ResourceDefinitionsSchemaTypeArray ||
		schemaType == provider.ResourceDefinitionsSchemaTypeMap
}

func validateDataSourcePropertySubstitution(
	subDataSourceProp *substitutions.SubstitutionDataSourceProperty,
	bpSchema *schema.Blueprint,
	usedIn string,
	usedInPropertyPath string,
	refChainCollector refgraph.RefChainCollector,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	dataSourceName := subDataSourceProp.DataSourceName
	if bpSchema.DataSources == nil || bpSchema.DataSources.Values == nil {
		return "", diagnostics, errSubDataSourceNotFound(dataSourceName, subDataSourceProp.SourceMeta)
	}

	dataSourceSchema, hasDataSource := bpSchema.DataSources.Values[dataSourceName]
	if !hasDataSource {
		return "", diagnostics, errSubDataSourceNotFound(dataSourceName, subDataSourceProp.SourceMeta)
	}

	if usedIn == bpcore.DataSourceElementID(dataSourceName) {
		return "", diagnostics, errSubDataSourceSelfReference(dataSourceName, subDataSourceProp.SourceMeta)
	}

	resolveType, err := validateDataSourcePropertyField(subDataSourceProp, dataSourceSchema)
	if err != nil {
		return "", diagnostics, err
	}

	subRefTag := CreateSubRefTag(usedIn)
	subRefPropTag := CreateSubRefPropTag(usedIn, usedInPropertyPath)
	refChainCollector.Collect(
		bpcore.DataSourceElementID(dataSourceName),
		dataSourceSchema,
		usedIn,
		[]string{subRefTag, subRefPropTag},
	)

	return resolveType, diagnostics, nil
}

func validateDataSourcePropertyField(
	subDataSourceProp *substitutions.SubstitutionDataSourceProperty,
	dataSourceSchema *schema.DataSource,
) (string, error) {
	if dataSourceSchema.Exports == nil {
		return "", errSubDataSourceNoExportedFields(
			subDataSourceProp.DataSourceName,
			subDataSourceProp.SourceMeta,
		)
	}

	field, hasField := dataSourceSchema.Exports.Values[subDataSourceProp.FieldName]
	if !hasField {
		return "", errSubDataSourceFieldNotExported(
			subDataSourceProp.DataSourceName,
			subDataSourceProp.FieldName,
			subDataSourceProp.SourceMeta,
		)
	}

	if field.Type == nil {
		return "", errSubDataSourceFieldMissingType(
			subDataSourceProp.DataSourceName,
			subDataSourceProp.FieldName,
			subDataSourceProp.SourceMeta,
		)
	}

	if subDataSourceProp.PrimitiveArrIndex != nil &&
		field.Type.Value != schema.DataSourceFieldTypeArray {
		return "", errSubDataSourceFieldNotArray(
			subDataSourceProp.DataSourceName,
			subDataSourceProp.FieldName,
			*subDataSourceProp.PrimitiveArrIndex,
			subDataSourceProp.SourceMeta,
		)
	}

	return subDataSourceFieldType(field.Type.Value), nil
}

func validateChildSubstitution(
	subChild *substitutions.SubstitutionChild,
	bpSchema *schema.Blueprint,
	usedIn string,
	usedInPropertyPath string,
	refChainCollector refgraph.RefChainCollector,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	childName := subChild.ChildName
	childSchema, hasChild := bpSchema.Include.Values[childName]
	if !hasChild {
		return "", diagnostics, errSubChildBlueprintNotFound(childName, subChild.SourceMeta)
	}

	if usedIn == bpcore.ChildElementID(childName) {
		return "", diagnostics, errSubChildBlueprintSelfReference(childName, subChild.SourceMeta)
	}

	subRefTag := CreateSubRefTag(usedIn)
	subRefPropTag := CreateSubRefPropTag(usedIn, usedInPropertyPath)
	refChainCollector.Collect(
		bpcore.ChildElementID(childName),
		childSchema,
		usedIn,
		[]string{subRefTag, subRefPropTag},
	)

	// There is no way of knowing the exact type of the child blueprint exports
	// until runtime, so we return any to account for all possible types.
	return string(substitutions.ResolvedSubExprTypeAny), diagnostics, nil
}

func validateFunctionSubstitution(
	ctx context.Context,
	subFunc *substitutions.SubstitutionFunctionExpr,
	nextLocation *source.Meta,
	bpSchema *schema.Blueprint,
	usedInResourceDerivedFromTemplate bool,
	usedIn string,
	usedInPropertyPath string,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	funcName := string(subFunc.FunctionName)

	defOutput, validateFuncDiagnostics, err := validateFunction(
		ctx,
		funcName,
		subFunc,
		nextLocation,
		funcRegistry,
		params,
	)
	diagnostics = append(diagnostics, validateFuncDiagnostics...)
	if err != nil {
		return "", diagnostics, err
	}

	if len(subFunc.Arguments) != len(defOutput.Definition.Parameters) &&
		!subFuncTakesVariadicArgs(defOutput.Definition) {
		return "", diagnostics, errSubFuncInvalidNumberOfArgs(
			len(defOutput.Definition.Parameters),
			len(subFunc.Arguments),
			subFunc,
		)
	}

	var errs []error
	for i, arg := range subFunc.Arguments {
		nextLocation := (*source.Meta)(nil)
		if i+1 < len(subFunc.Arguments) {
			nextLocation = subFunc.Arguments[i+1].SourceMeta
		}

		resolveType, argDiagnostics, err := validateSubFuncArgument(
			ctx,
			arg,
			nextLocation,
			bpSchema,
			usedInResourceDerivedFromTemplate,
			usedIn,
			usedInPropertyPath,
			funcName,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
		)
		diagnostics = append(diagnostics, argDiagnostics...)
		if err != nil {
			errs = append(errs, err)
		}

		if err == nil {
			err = checkSubFuncArgType(defOutput.Definition, i, arg.Value, resolveType, funcName, arg.SourceMeta)
			if err != nil {
				errs = append(errs, err)
			}
		}

		// "link" function arguments are a special case, where string literals
		// that contain resource names should be treated as references to resources in the blueprint.
		err = validateLinkFuncArg(funcName, i, arg, usedIn, bpSchema, refChainCollector)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return "", diagnostics, ErrMultipleValidationErrors(errs)
	}

	return subFunctionReturnType(defOutput), diagnostics, nil
}

func validateFunction(
	ctx context.Context,
	funcName string,
	subFunc *substitutions.SubstitutionFunctionExpr,
	nextLocation *source.Meta,
	funcRegistry provider.FunctionRegistry,
	params bpcore.BlueprintParams,
) (*provider.FunctionGetDefinitionOutput, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	isCoreFunc := core.SliceContainsComparable(
		substitutions.CoreSubstitutionFunctions,
		subFunc.FunctionName,
	)

	hasFunc, err := funcRegistry.HasFunction(ctx, funcName)
	if err != nil {
		return nil, diagnostics, err
	}

	if !hasFunc && !isCoreFunc {
		diagnostics = append(
			diagnostics,
			&bpcore.Diagnostic{
				Level: bpcore.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"Function %q is not a core function, when staging changes and deploying,"+
						" you will need to make sure the provider is loaded.",
					funcName,
				),
				Range: toDiagnosticRange(subFunc.SourceMeta, nextLocation),
			},
		)
	}

	defOutput, err := funcRegistry.GetDefinition(
		ctx,
		funcName,
		&provider.FunctionGetDefinitionInput{
			Params: params,
		},
	)
	if err != nil {
		return nil, diagnostics, errSubFailedToLoadFunctionDefintion(
			funcName,
			subFunc.SourceMeta,
			"the function may not be configured with the loaded providers",
		)
	}

	return defOutput, diagnostics, nil
}

func validateLinkFuncArg(
	funcName string,
	index int,
	arg *substitutions.SubstitutionFunctionArg,
	usedIn string,
	bpSchema *schema.Blueprint,
	refChainCollector refgraph.RefChainCollector,
) error {
	if funcName == string(substitutions.SubstitutionFunctionLink) {
		if arg.Value != nil && arg.Value.StringValue != nil {
			resourceName := *arg.Value.StringValue
			resource, hasResource := getResource(resourceName, bpSchema)
			if hasResource {
				subRefTag := CreateSubRefTag(usedIn)
				refChainCollector.Collect(
					bpcore.ResourceElementID(resourceName),
					resource,
					usedIn,
					[]string{subRefTag},
				)
			} else {
				return errSubFuncLinkArgResourceNotFound(resourceName, index, usedIn, arg.SourceMeta)
			}
		}
	}

	return nil
}

func checkSubFuncArgType(
	definition *function.Definition,
	argIndex int,
	argVal *substitutions.Substitution,
	resolveType string,
	funcName string,
	location *source.Meta,
) error {
	paramIndex := int(math.Min(float64(argIndex), float64(len(definition.Parameters)-1)))
	param := definition.Parameters[paramIndex]

	anyParam, isAny := param.(*function.AnyParameter)
	if isAny {
		err := validateSubFuncArgAnyType(anyParam.UnionTypes, resolveType, argIndex, funcName, location)
		if err != nil {
			return err
		}
		return nil
	}

	variadicParam, isVariadic := param.(*function.VariadicParameter)
	if isVariadic {
		anyTypeDef, isAny := variadicParam.Type.(*function.ValueTypeDefinitionAny)
		if isAny {
			err := validateSubFuncArgAnyType(anyTypeDef.UnionTypes, resolveType, argIndex, funcName, location)
			if err != nil {
				return err
			}
			return nil
		}

		if string(variadicParam.GetType()) != resolveType {
			return errSubFuncArgTypeMismatch(
				argIndex,
				string(variadicParam.GetType()),
				resolveType,
				funcName,
				location,
			)
		}
		return nil
	}

	// In some cases we can't know exactly what the resolved type is during the validation
	// stage, to account for these situations and reduce noise, the any resolved type is acceptable
	// for all function arguments.
	if string(param.GetType()) != resolveType && resolveType != string(substitutions.ResolvedSubExprTypeAny) {
		return errSubFuncArgTypeMismatch(
			argIndex,
			string(param.GetType()),
			resolveType,
			funcName,
			location,
		)
	}

	if string(param.GetType()) == resolveType &&
		resolveType == string(substitutions.ResolvedSubExprTypeString) {
		err := checkStringChoices(param, argIndex, argVal, funcName, location)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSubFuncArgAnyType(
	unionTypes []function.ValueTypeDefinition,
	resolveType string,
	argIndex int,
	funcName string,
	location *source.Meta,
) error {
	if len(unionTypes) == 0 {
		// Any type without a strict union type should allow all possible types.
		return nil
	}

	matchesUnionType := false
	i := 0
	for !matchesUnionType && i < len(unionTypes) {
		if string(unionTypes[i].GetType()) != resolveType {
			matchesUnionType = true
		}
		i += 1
	}

	if !matchesUnionType {
		return errSubFuncArgTypeMismatch(
			argIndex,
			string(subFuncUnionTypeToString(unionTypes)),
			resolveType,
			funcName,
			location,
		)
	}
	return nil
}

func validateSubFuncArgument(
	ctx context.Context,
	arg *substitutions.SubstitutionFunctionArg,
	nextLocation *source.Meta,
	bpSchema *schema.Blueprint,
	usedInResourceDerivedFromTemplate bool,
	usedIn string,
	usedInPropertyPath string,
	funcName string,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
) (string, []*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if arg == nil {
		return "", diagnostics, nil
	}

	if arg.Value == nil {
		return "", diagnostics, nil
	}

	if arg.Name != "" && funcName != string(substitutions.SubstitutionFunctionObject) {
		return "", diagnostics, errSubFuncNamedArgsNotAllowed(arg.Name, funcName, arg.SourceMeta)
	}

	return ValidateSubstitution(
		ctx,
		arg.Value,
		nextLocation,
		bpSchema,
		usedInResourceDerivedFromTemplate,
		usedIn,
		usedInPropertyPath,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
	)
}

func checkStringChoices(
	param function.Parameter,
	argIndex int,
	argVal *substitutions.Substitution,
	funcName string,
	location *source.Meta,
) error {
	if argVal.StringValue == nil {
		// If the argument is not a string literal, skip checking choices.
		return nil
	}

	scalarParam, isScalar := param.(*function.ScalarParameter)
	if !isScalar {
		return nil
	}

	scalarTypeDef, isScalarTypeDef := scalarParam.Type.(*function.ValueTypeDefinitionScalar)
	if !isScalarTypeDef {
		return nil
	}

	if len(scalarTypeDef.StringChoices) == 0 {
		return nil
	}

	if !slices.Contains(scalarTypeDef.StringChoices, *argVal.StringValue) {
		return errSubFuncArgInvalidStringChoice(
			argIndex,
			scalarTypeDef.StringChoices,
			*argVal.StringValue,
			funcName,
			location,
		)
	}

	return nil
}

func subFuncTakesVariadicArgs(def *function.Definition) bool {
	if len(def.Parameters) == 0 {
		return false
	}

	lastParam := def.Parameters[len(def.Parameters)-1]
	_, isVariadic := lastParam.(*function.VariadicParameter)
	return isVariadic
}

func subFunctionReturnType(defOutput *provider.FunctionGetDefinitionOutput) string {
	return subFunctionValueType(defOutput.Definition.Return.GetType())
}

func subFunctionValueType(valueType function.ValueType) string {
	switch valueType {
	case function.ValueTypeString:
		return string(substitutions.ResolvedSubExprTypeString)
	case function.ValueTypeInt32:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case function.ValueTypeInt64:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case function.ValueTypeFloat32:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case function.ValueTypeFloat64:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case function.ValueTypeBool:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	case function.ValueTypeList:
		return string(substitutions.ResolvedSubExprTypeArray)
	case function.ValueTypeMap:
		return string(substitutions.ResolvedSubExprTypeObject)
	case function.ValueTypeObject:
		return string(substitutions.ResolvedSubExprTypeObject)
	case function.ValueTypeFunction:
		return string(substitutions.ResolvedSubExprTypeFunction)
	default:
		return string(substitutions.ResolvedSubExprTypeAny)
	}
}

func subVarType(varType *schema.VariableTypeWrapper) string {
	if varType == nil {
		return string(substitutions.ResolvedSubExprTypeAny)
	}

	switch varType.Value {
	case schema.VariableTypeInteger:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case schema.VariableTypeFloat:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case schema.VariableTypeBoolean:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	default:
		// Strings and custom variable types are treated as strings.
		return string(substitutions.ResolvedSubExprTypeString)
	}
}

func subValType(valType *schema.ValueTypeWrapper) string {
	if valType == nil {
		return string(substitutions.ResolvedSubExprTypeAny)
	}

	switch valType.Value {
	case schema.ValueTypeInteger:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case schema.ValueTypeFloat:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case schema.ValueTypeBoolean:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	case schema.ValueTypeArray:
		return string(substitutions.ResolvedSubExprTypeArray)
	case schema.ValueTypeObject:
		return string(substitutions.ResolvedSubExprTypeObject)
	default:
		return string(substitutions.ResolvedSubExprTypeString)
	}
}

func subDataSourceFieldType(fieldType schema.DataSourceFieldType) string {
	switch fieldType {
	case schema.DataSourceFieldTypeInteger:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case schema.DataSourceFieldTypeFloat:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case schema.DataSourceFieldTypeBoolean:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	case schema.DataSourceFieldTypeArray:
		return string(substitutions.ResolvedSubExprTypeArray)
	default:
		return string(substitutions.ResolvedSubExprTypeString)
	}
}

func subResourceDefinitionsSchemaType(schemaType provider.ResourceDefinitionsSchemaType) string {
	switch schemaType {
	case provider.ResourceDefinitionsSchemaTypeInteger:
		return string(substitutions.ResolvedSubExprTypeInteger)
	case provider.ResourceDefinitionsSchemaTypeFloat:
		return string(substitutions.ResolvedSubExprTypeFloat)
	case provider.ResourceDefinitionsSchemaTypeBoolean:
		return string(substitutions.ResolvedSubExprTypeBoolean)
	case provider.ResourceDefinitionsSchemaTypeArray:
		return string(substitutions.ResolvedSubExprTypeArray)
	case provider.ResourceDefinitionsSchemaTypeObject, provider.ResourceDefinitionsSchemaTypeMap:
		return string(substitutions.ResolvedSubExprTypeObject)
	default:
		return string(substitutions.ResolvedSubExprTypeString)
	}
}

func subFuncUnionTypeToString(unionTypes []function.ValueTypeDefinition) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i, t := range unionTypes {
		sb.WriteString(string(t.GetType()))
		if i < len(unionTypes)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}
