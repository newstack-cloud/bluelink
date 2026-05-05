package transformutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
)

const (
	// AnnotationSourceAbstractName is the annotation key set by transformer plugins
	// to record the original abstract resource name that a concrete resource was
	// expanded from.
	AnnotationSourceAbstractName = "bluelink.transform.source.abstractName"

	// AnnotationSourceAbstractType is the annotation key set by transformer plugins
	// to record the original abstract resource type that a concrete resource was
	// expanded from.
	AnnotationSourceAbstractType = "bluelink.transform.source.abstractType"

	// AnnotationResourceCategory is the annotation key set by transformer plugins
	// to classify a concrete resource as either "code-hosting" or "infrastructure".
	// Used by the code-only auto-approval mechanism.
	AnnotationResourceCategory = "bluelink.transform.resourceCategory"

	// ResourceCategoryCodeHosting indicates a resource that hosts application code
	// (e.g. Lambda function, ECS task, API Gateway).
	ResourceCategoryCodeHosting = "code-hosting"

	// ResourceCategoryInfrastructure indicates an infrastructure dependency
	// (e.g. DynamoDB table, S3 bucket, IAM role, VPC).
	ResourceCategoryInfrastructure = "infrastructure"
)

type TransformerBaseAnnotationsInput struct {
	// AbstractResourceName is the name of the abstract resource in the blueprint.
	AbstractResourceName string
	// AbstractResourceType is the type of the abstract resource in the blueprint.
	AbstractResourceType string
	// ResourceCategory is the category of the resource, either "code-hosting" or "infrastructure".
	ResourceCategory string
}

// TransformerBaseAnnotations returns base annotations
// to be used for concrete resources generated from an abstract
// resource type to maintain correlation between the abstract resource
// in your blueprint and the concrete resources that will be deployed.
func TransformerBaseAnnotations(
	input *TransformerBaseAnnotationsInput,
) *schema.StringOrSubstitutionsMap {
	return &schema.StringOrSubstitutionsMap{
		Values: map[string]*substitutions.StringOrSubstitutions{
			AnnotationSourceAbstractName: pluginutils.StringToSubstitutions(input.AbstractResourceName),
			AnnotationSourceAbstractType: pluginutils.StringToSubstitutions(input.AbstractResourceType),
			AnnotationResourceCategory:   pluginutils.StringToSubstitutions(input.ResourceCategory),
		},
	}
}
