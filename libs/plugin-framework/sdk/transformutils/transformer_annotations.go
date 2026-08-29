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

	// AnnotationResourceRole is the annotation key set by transformer plugins to
	// describe the part an abstract resource plays in a deployment, as opposed
	// to what it is made of, which is what AnnotationResourceCategory records.
	//
	// Tools presenting a deployment use it to decide where a link belongs.
	// Defaults to "component" when absent.
	AnnotationResourceRole = "bluelink.transform.resourceRole"

	// ResourceRoleComponent indicates an abstract resource that is a part of the
	// application being deployed, such as an API, a handler or a datastore.
	// Links out of it describe the application's own wiring.
	ResourceRoleComponent = "component"

	// ResourceRoleAmbient indicates an abstract resource that most of a
	// deployment sits inside or depends on, such as a VPC. Nearly everything
	// links out of it, so those links describe the environment rather than the
	// application, and say more about the resource at the other end. Tools are
	// expected to attribute them to that counterpart rather than collecting
	// them all under the ambient resource.
	ResourceRoleAmbient = "ambient"
)

type TransformerBaseAnnotationsInput struct {
	// AbstractResourceName is the name of the abstract resource in the blueprint.
	AbstractResourceName string
	// AbstractResourceType is the type of the abstract resource in the blueprint.
	AbstractResourceType string
	// ResourceCategory is the category of the resource, either "code-hosting" or "infrastructure".
	ResourceCategory string
	// ResourceRole is the part the abstract resource plays in a deployment,
	// either "component" or "ambient". Leave empty for "component".
	ResourceRole string
}

// TransformerBaseAnnotations returns base annotations
// to be used for concrete resources generated from an abstract
// resource type to maintain correlation between the abstract resource
// in your blueprint and the concrete resources that will be deployed.
func TransformerBaseAnnotations(
	input *TransformerBaseAnnotationsInput,
) *schema.StringOrSubstitutionsMap {
	values := map[string]*substitutions.StringOrSubstitutions{
		AnnotationSourceAbstractName: pluginutils.StringToSubstitutions(input.AbstractResourceName),
		AnnotationSourceAbstractType: pluginutils.StringToSubstitutions(input.AbstractResourceType),
		AnnotationResourceCategory:   pluginutils.StringToSubstitutions(input.ResourceCategory),
	}

	// Only carried when it says something: the absence of the annotation and a
	// role of "component" mean the same thing, and every resource written
	// before this existed reads as a component.
	if input.ResourceRole != "" {
		values[AnnotationResourceRole] = pluginutils.StringToSubstitutions(input.ResourceRole)
	}

	return &schema.StringOrSubstitutionsMap{Values: values}
}
