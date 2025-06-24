package speccore

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// BlueprintSpec provides an interface for a service that holds
// a parsed blueprint schema and direct access to resource schemas.
// This interface is provided to decouple containers and loaders
// to make every component of the blueprint mechanism composable.
type BlueprintSpec interface {
	// ResourceSchema provides a convenient way to get the
	// schema for a resource without having to first get
	// the blueprint spec.
	ResourceSchema(resourceName string) *schema.Resource
	// Schema retrieves the schema for a loaded
	// blueprint.
	Schema() *schema.Blueprint
}
