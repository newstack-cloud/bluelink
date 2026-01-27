package validation

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ValidateLinkAnnotations checks the validity of link annotations
// for one or more link chains represented as a graph-like data structure
// where resources are nodes and links are edges.
//
// Each annotation is checked against the link annotation definitions
// for the provider plugin link implementation that connects two resources.
// This is intended to be used at the end of the validation process
// once a graph of resources and links has been built
// after all other elements in a blueprint have been validated.
// This must only be called after the provided link chains have been checked
// for cycles.
//
// This validation supports dynamic annotation keys
// that can contain a single "<resourceName>" placeholder string.
// The value that "<resourceName>" represents must be the name of a resource
// that is linked to the resource type where the annotation is defined.
// Only a single "<resourceName>" placeholder is allowed for a dynamic annotation key.
// Dynamic keys are used to target specific resources when there are multiple resources
// of the same type linked to the resource where the annotation is defined.
// Default values are ignored for link annotation field definitions that have
// dynamic field names, the default value should be defined in an equivalent
// annotation that is not targeted at a specific resource name (e.g. "aws.lambda.dynamodb.accessType").
//
// When an annotation definition with a dynamic name is required, it means that
// at least one annotation value that matches the pattern must be present.
//
// Unknown annotation keys are ignored, allowing them to be used for other purposes.
//
// This returns an error for any unexpected errors and will return
// a list of diagnostics for any validation errors and warnings.
func ValidateLinkAnnotations(
	ctx context.Context,
	linkChains []*links.ChainLinkNode,
	params core.BlueprintParams,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}
	err := validateLinkAnnotations(
		ctx,
		linkChains,
		params,
		&diagnostics,
	)
	return diagnostics, err
}

func validateLinkAnnotations(
	ctx context.Context,
	linkChains []*links.ChainLinkNode,
	params core.BlueprintParams,
	diagnostics *[]*core.Diagnostic,
) error {
	for _, linkChainNode := range linkChains {
		resourceAnnotations := getAnnotations(linkChainNode.Resource)
		metadataBlockLocation := getMetadataBlockLocation(linkChainNode.Resource)

		for linksTo, linkImpl := range linkChainNode.LinkImplementations {
			linkAnnotationDefsOutput, err := linkImpl.GetAnnotationDefinitions(
				ctx,
				&provider.LinkGetAnnotationDefinitionsInput{
					LinkContext: provider.NewLinkContextFromParams(params),
				},
			)
			if err != nil {
				return err
			}

			err = validateLinkAnnotationsForResources(
				linkChainNode,
				linksTo,
				resourceAnnotations,
				metadataBlockLocation,
				linkAnnotationDefsOutput,
				diagnostics,
			)
			if err != nil {
				return err
			}

			err = validateLinkAnnotations(
				ctx,
				linkChainNode.LinksTo,
				params,
				diagnostics,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func validateLinkAnnotationsForResources(
	linkChainNode *links.ChainLinkNode,
	linksTo string,
	resourceAnnotations *schema.StringOrSubstitutionsMap,
	metadataBlockLocation *source.Meta,
	linkAnnotationDefsOutput *provider.LinkGetAnnotationDefinitionsOutput,
	diagnostics *[]*core.Diagnostic,
) error {
	annotationDefs := getLinkAnnotationDefinitions(
		linkAnnotationDefsOutput,
	)
	// Validate annotations for resource A (the current link chain node)
	err := validateResourceLinkAnnotations(
		linkChainNode.ResourceName,
		schema.GetResourceType(linkChainNode.Resource),
		linksTo,
		resourceAnnotations,
		metadataBlockLocation,
		annotationDefs,
		provider.LinkAnnotationResourceA,
		diagnostics,
	)
	if err != nil {
		return err
	}

	otherLinkChainNode := getChainNodeByResourceName(
		linkChainNode.LinksTo,
		linksTo,
	)
	otherResourceAnnotations := getAnnotations(
		otherLinkChainNode.Resource,
	)
	// Validate annotations for resource B (the linked resource)
	return validateResourceLinkAnnotations(
		otherLinkChainNode.ResourceName,
		schema.GetResourceType(otherLinkChainNode.Resource),
		linkChainNode.ResourceName,
		otherResourceAnnotations,
		getMetadataBlockLocation(otherLinkChainNode.Resource),
		annotationDefs,
		provider.LinkAnnotationResourceB,
		diagnostics,
	)
}

func getChainNodeByResourceName(
	linkChains []*links.ChainLinkNode,
	resourceName string,
) *links.ChainLinkNode {
	for _, linkChainNode := range linkChains {
		if linkChainNode.ResourceName == resourceName {
			return linkChainNode
		}
	}
	return nil
}

func validateResourceLinkAnnotations(
	resourceName string,
	resourceType string,
	linksTo string,
	resourceAnnotations *schema.StringOrSubstitutionsMap,
	// Used as the location for diagnostics and errors
	// when required annotations are missing.
	metadataBlockLocation *source.Meta,
	linkAnnotationDefinitions map[string]*provider.LinkAnnotationDefinition,
	// resourcePosition indicates whether this is resource A or B in the link relationship.
	// This is used to filter annotations based on the AppliesTo field.
	resourcePosition provider.LinkAnnotationResource,
	diagnostics *[]*core.Diagnostic,
) error {
	linkAnnotationDefinitionsForResourceType := extractAnnotationDefinitionsForResourceType(
		resourceType,
		linkAnnotationDefinitions,
		resourcePosition,
	)
	for definitionKey, definition := range linkAnnotationDefinitionsForResourceType {
		renderedDefAnnotationName := replacePlaceholderWithResourceName(
			definition.Name,
			linksTo,
		)

		// Get all annotations that match this definition.
		// For dynamic definitions (with placeholders), this uses pattern matching
		// to find all annotations that match the pattern, not just those
		// that exactly match the rendered definition name.
		matchingAnnotations, err := getAllMatchingAnnotations(
			definition.Name,
			resourceAnnotations,
		)
		if err != nil {
			return err
		}

		// Check if required annotation is missing
		if len(matchingAnnotations) == 0 && definition.Required {
			*diagnostics = append(*diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"The %q annotation is required for the %q resource in relation to the %q resource, but is missing or null.",
					renderedDefAnnotationName,
					resourceName,
					linksTo,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					metadataBlockLocation,
					nil,
				),
			})
			return nil
		}

		// Validate each matching annotation
		for _, resourceAnnotationInfo := range matchingAnnotations {
			if substitutions.IsNilStringSubs(resourceAnnotationInfo.annotation) {
				continue
			}

			parsedValue, isCorrectTypeAndValueKnown := validateAnnotationType(
				resourceAnnotationInfo,
				definition,
				resourceName,
				diagnostics,
			)

			if isCorrectTypeAndValueKnown && len(definition.AllowedValues) > 0 {
				validateAnnotationAllowedValues(
					parsedValue,
					resourceAnnotationInfo,
					definition,
					resourceName,
					definitionKey,
					diagnostics,
				)
			}

			if isCorrectTypeAndValueKnown && definition.ValidateFunc != nil {
				customValidateDiagnostics := definition.ValidateFunc(
					resourceAnnotationInfo.annotationKey,
					parsedValue,
				)
				*diagnostics = append(*diagnostics, customValidateDiagnostics...)
			}
		}
	}

	return nil
}

func validateAnnotationType(
	resourceAnnotationInfo *resourceAnnotationInfo,
	definition *provider.LinkAnnotationDefinition,
	resourceName string,
	diagnostics *[]*core.Diagnostic,
) (*core.ScalarValue, bool) {
	// An annotation can have an empty value if it is not a required
	// annotation.
	if len(resourceAnnotationInfo.annotation.Values) == 0 {
		return nil, false
	}

	// A StringOrSubstitutions struct with more than one value
	// represents a string interpolation for which the final resolved
	// value can not be known at the validation stage.
	if len(resourceAnnotationInfo.annotation.Values) > 1 ||
		// A StringOrSubstitutions struct with a single value
		// that contains a substitution value can not be known at
		// the validation stage.
		(len(resourceAnnotationInfo.annotation.Values) == 1 &&
			resourceAnnotationInfo.annotation.Values[0].SubstitutionValue != nil) {
		*diagnostics = append(
			*diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelWarning,
				Message: fmt.Sprintf(
					"The value of the %q annotation in the %q resource contains substitutions"+
						" and can not be validated against a type. "+
						"When substitutions are resolved, this value must be a valid %s.",
					resourceAnnotationInfo.annotationKey,
					resourceName,
					definition.Type,
				),
				Range: core.DiagnosticRangeFromSourceMeta(resourceAnnotationInfo.annotation.SourceMeta, nil),
			},
		)
		return nil, false
	}

	stringVal := resourceAnnotationInfo.annotation.Values[0].StringValue
	if stringVal == nil {
		return nil, false
	}

	scalarMappingNode := core.ParseScalarMappingNode(*stringVal)
	scalarValue := scalarMappingNode.Scalar
	// Ensure we carry over the original source meta information, if any,
	// so diagnostics from custom validation can point to the correct
	// location in the source blueprint.
	scalarValue.SourceMeta = resourceAnnotationInfo.annotation.SourceMeta

	if core.IsScalarBool(scalarValue) && definition.Type == core.ScalarTypeBool {
		return scalarValue, true
	}

	if core.IsScalarInt(scalarValue) && definition.Type == core.ScalarTypeInteger {
		return scalarValue, true
	}

	if core.IsScalarFloat(scalarValue) && definition.Type == core.ScalarTypeFloat {
		return scalarValue, true
	}

	if core.IsScalarString(scalarValue) && definition.Type == core.ScalarTypeString {
		return scalarValue, true
	}

	*diagnostics = append(
		*diagnostics,
		&core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"The value of the %q annotation in the %q resource is not a valid %s. "+
					"Expected a value of type %s, but got %s.",
				resourceAnnotationInfo.annotationKey,
				resourceName,
				definition.Type,
				definition.Type,
				core.TypeFromScalarValue(scalarValue),
			),
			Range: core.DiagnosticRangeFromSourceMeta(
				resourceAnnotationInfo.annotation.SourceMeta,
				nil,
			),
		},
	)

	return nil, false
}

func validateAnnotationAllowedValues(
	parsedValue *core.ScalarValue,
	resourceAnnotationInfo *resourceAnnotationInfo,
	definition *provider.LinkAnnotationDefinition,
	resourceName string,
	definitionKey string,
	diagnostics *[]*core.Diagnostic,
) {
	matchesAtLeastOne := slices.ContainsFunc(
		definition.AllowedValues,
		func(allowedValue *core.ScalarValue) bool {
			return allowedValue.Equal(parsedValue)
		},
	)

	if !matchesAtLeastOne {
		allowedValuesText := createAllowedValuesText(
			wrapWithMappingNodes(definition.AllowedValues),
			maxShowAllowedValues,
			fmt.Sprintf("%q annotation definition", definitionKey),
		)
		*diagnostics = append(
			*diagnostics,
			&core.Diagnostic{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"The value of the %q annotation in the %q resource is not one of the allowed values. "+
						"%s was provided but expected one of %s",
					resourceAnnotationInfo.annotationKey,
					resourceName,
					parsedValue.ToString(),
					allowedValuesText,
				),
				Range: core.DiagnosticRangeFromSourceMeta(
					resourceAnnotationInfo.annotation.SourceMeta,
					nil,
				),
			},
		)
	}
}

func extractAnnotationDefinitionsForResourceType(
	resourceType string,
	linkAnnotationDefinitions map[string]*provider.LinkAnnotationDefinition,
	resourcePosition provider.LinkAnnotationResource,
) map[string]*provider.LinkAnnotationDefinition {
	if resourceType == "" {
		return linkAnnotationDefinitions
	}

	resourceTypePrefix := fmt.Sprintf("%s::", resourceType)
	resourceTypeAnnotationDefs := make(map[string]*provider.LinkAnnotationDefinition)

	for key, definition := range linkAnnotationDefinitions {
		if strings.HasPrefix(key, resourceTypePrefix) &&
			annotationAppliesToResource(definition.AppliesTo, resourcePosition) {
			resourceTypeAnnotationDefs[key] = definition
		}
	}

	return resourceTypeAnnotationDefs
}

// annotationAppliesToResource checks if an annotation definition applies to the given
// resource position in the link relationship.
func annotationAppliesToResource(
	appliesTo provider.LinkAnnotationResource,
	resourcePosition provider.LinkAnnotationResource,
) bool {
	// LinkAnnotationResourceAny applies to either resource
	if appliesTo == provider.LinkAnnotationResourceAny {
		return true
	}
	// Otherwise, the annotation must match the specific resource position
	return appliesTo == resourcePosition
}

type resourceAnnotationInfo struct {
	annotation            *substitutions.StringOrSubstitutions
	annotationKey         string
	hasResourceAnnotation bool
}

// getAllMatchingAnnotations returns all annotations that match a definition.
// For static definitions (no placeholders), it returns at most one annotation
// with an exact key match.
// For dynamic definitions (with placeholders), it uses pattern matching to find
// ALL annotations that match the pattern, ensuring that annotations are validated
// even if the resource name in the annotation doesn't exactly match the linked resource.
func getAllMatchingAnnotations(
	definitionKey string,
	resourceAnnotations *schema.StringOrSubstitutionsMap,
) ([]*resourceAnnotationInfo, error) {
	if resourceAnnotations == nil || resourceAnnotations.Values == nil {
		return nil, nil
	}

	// Check for exact match with the definition key (literal placeholder in key)
	if annotation, exists := resourceAnnotations.Values[definitionKey]; exists {
		return []*resourceAnnotationInfo{
			{
				annotation:            annotation,
				annotationKey:         definitionKey,
				hasResourceAnnotation: true,
			},
		}, nil
	}

	// For non-dynamic definitions, only look for exact match
	if !core.IsDynamicFieldName(definitionKey) {
		return nil, nil
	}

	// For dynamic definitions, use pattern matching to find ALL matching annotations
	pattern, err := createPatternForAnnotationKey(definitionKey)
	if err != nil {
		return nil, err
	}

	var matchingAnnotations []*resourceAnnotationInfo
	for key, annotation := range resourceAnnotations.Values {
		if pattern.MatchString(key) {
			matchingAnnotations = append(matchingAnnotations, &resourceAnnotationInfo{
				annotation:            annotation,
				annotationKey:         key,
				hasResourceAnnotation: true,
			})
		}
	}

	return matchingAnnotations, nil
}

// createPatternForAnnotationKey creates a compiled regex pattern from an
// annotation definition name that may contain placeholders like "<resourceName>".
// The pattern will match any annotation key that fits the placeholder pattern.
func createPatternForAnnotationKey(definitionKey string) (*regexp.Regexp, error) {
	patternString := core.CreatePatternForDynamicFieldName(definitionKey, 1)
	// Anchor the pattern to match the entire string
	patternString = "^" + patternString + "$"
	return regexp.Compile(patternString)
}

func replacePlaceholderWithResourceName(
	definitionKey, linksToResource string,
) string {
	openAngleBracketIndex := strings.Index(definitionKey, "<")
	closeAngleBracketIndex := strings.Index(definitionKey, ">")

	if openAngleBracketIndex == -1 ||
		closeAngleBracketIndex == -1 ||
		closeAngleBracketIndex < openAngleBracketIndex {
		return definitionKey
	}

	return definitionKey[:openAngleBracketIndex] +
		linksToResource +
		definitionKey[closeAngleBracketIndex+1:]
}

func getAnnotations(resource *schema.Resource) *schema.StringOrSubstitutionsMap {
	if resource.Metadata == nil || resource.Metadata.Annotations == nil {
		return &schema.StringOrSubstitutionsMap{
			Values: map[string]*substitutions.StringOrSubstitutions{},
		}
	}

	return resource.Metadata.Annotations
}

func getMetadataBlockLocation(resource *schema.Resource) *source.Meta {
	if resource.Metadata == nil {
		return nil
	}

	return resource.Metadata.SourceMeta
}

func getLinkAnnotationDefinitions(
	output *provider.LinkGetAnnotationDefinitionsOutput,
) map[string]*provider.LinkAnnotationDefinition {
	if output == nil {
		return map[string]*provider.LinkAnnotationDefinition{}
	}

	return output.AnnotationDefinitions
}

func wrapWithMappingNodes(
	values []*core.ScalarValue,
) []*core.MappingNode {
	if values == nil {
		return nil
	}

	mappingNodes := make([]*core.MappingNode, len(values))
	for i, value := range values {
		mappingNodes[i] = &core.MappingNode{
			Scalar: value,
		}
	}

	return mappingNodes
}
