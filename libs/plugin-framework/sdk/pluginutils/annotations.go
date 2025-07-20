package pluginutils

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// AnnotationQuery is a simple structure that holds query parameters
// for retrieving annotations from resource information passed into
// the methods of a link in a provider plugin.
type AnnotationQuery[AnnotationType any] struct {
	// Key is the primary key to retrieve the annotation.
	// This is usually a more targeted key that will include a targeted resource
	// name in the annotation key.
	// For example, "aws.lambda.function.myFunction.populateEnvVars" would be
	// a targeted key for the link annotation definition
	// "aws.lambda.function.<resourceName>.populateEnvVars".
	// For simple cases, this will be a static documented key when targeting
	// specific resources in a blueprint is not supported.
	Key string
	// FallbackKey is the fallback key to retrieve the annotation.
	// This is usually a more generic key that will include the resource type
	// in the annotation key.
	// For example, "aws.lambda.function.populateEnvVars" would be a general key
	// that applies to all outgoing or incoming links that involve the current resource type.
	FallbackKey string
	// The default value to return if the annotation could not be found.
	Default AnnotationType
}

// GetBoolAnnotation retrieves a boolean annotation from the resource info.
// It returns the value of the annotation if it exists, otherwise it returns the default value.
// The second return value indicates whether an annotation was found for the provided query,
// it will be false if the default value was returned.
func GetBoolAnnotation(
	resourceInfo *provider.ResourceInfo,
	query *AnnotationQuery[bool],
) (bool, bool) {
	if isMissingAnnotations(resourceInfo) {
		return query.Default, false
	}

	annotation, hasAnnotation := getAnnotation(resourceInfo, query)
	if hasAnnotation && annotation != nil {
		return core.BoolValue(annotation), true
	}

	return query.Default, false
}

// GetStringAnnotation retrieves a string annotation from the resource info.
// It returns the value of the annotation if it exists, otherwise it returns the default value.
func GetStringAnnotation(
	resourceInfo *provider.ResourceInfo,
	query *AnnotationQuery[string],
) (string, bool) {
	if isMissingAnnotations(resourceInfo) {
		return query.Default, false
	}

	annotation, hasAnnotation := getAnnotation(resourceInfo, query)
	if hasAnnotation && annotation != nil {
		return core.StringValue(annotation), true
	}

	return query.Default, false
}

// GetIntAnnotation retrieves an integer annotation from the resource info.
// It returns the value of the annotation if it exists, otherwise it returns the default value.
func GetIntAnnotation(
	resourceInfo *provider.ResourceInfo,
	query *AnnotationQuery[int],
) (int, bool) {
	if isMissingAnnotations(resourceInfo) {
		return query.Default, false
	}

	annotation, hasAnnotation := getAnnotation(resourceInfo, query)
	if hasAnnotation && annotation != nil {
		return core.IntValue(annotation), true
	}

	return query.Default, false
}

// GetFloatAnnotation retrieves a float annotation from the resource info.
// It returns the value of the annotation if it exists, otherwise it returns the default value.
func GetFloatAnnotation(
	resourceInfo *provider.ResourceInfo,
	query *AnnotationQuery[float64],
) (float64, bool) {
	if isMissingAnnotations(resourceInfo) {
		return query.Default, false
	}

	annotation, hasAnnotation := getAnnotation(resourceInfo, query)
	if hasAnnotation && annotation != nil {
		return core.FloatValue(annotation), true
	}

	return query.Default, false
}

func getAnnotation[AnnotationType any](
	resourceInfo *provider.ResourceInfo,
	query *AnnotationQuery[AnnotationType],
) (*core.MappingNode, bool) {
	annotations := resourceInfo.ResourceWithResolvedSubs.Metadata.Annotations
	path := fmt.Sprintf("$[%q]", query.Key)
	annotation, hasAnnotation := GetValueByPath(path, annotations)
	if hasAnnotation && annotation != nil {
		return annotation, true
	}

	if query.FallbackKey != "" {
		fallbackPath := fmt.Sprintf("$[%q]", query.FallbackKey)
		fallbackAnnotation, hasFallback := GetValueByPath(fallbackPath, annotations)
		if hasFallback && fallbackAnnotation != nil {
			return fallbackAnnotation, true
		}
	}

	return nil, false
}

func isMissingAnnotations(resourceInfo *provider.ResourceInfo) bool {
	return resourceInfo == nil ||
		resourceInfo.ResourceWithResolvedSubs == nil ||
		resourceInfo.ResourceWithResolvedSubs.Metadata == nil ||
		resourceInfo.ResourceWithResolvedSubs.Metadata.Annotations == nil
}
