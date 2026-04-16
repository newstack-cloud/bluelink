package speccore

import (
	"maps"

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

// Stores the full blueprint schema and direct access to the
// mapping of resource names to their schemas for convenience.
// This is structure of the spec encapsulated by the blueprint container
// and other to create adaptors that need to fulfill the BlueprintSpec interface.
type defaultBlueprintSpec struct {
	resourceSchemas map[string]*schema.Resource
	schema          *schema.Blueprint
}

func (s *defaultBlueprintSpec) ResourceSchema(resourceName string) *schema.Resource {
	resourceSchema, ok := s.resourceSchemas[resourceName]
	if !ok {
		return nil
	}
	return resourceSchema
}

func (s *defaultBlueprintSpec) Schema() *schema.Blueprint {
	return s.schema
}

// BlueprintSpecFromSchema creates a BlueprintSpec from a parsed blueprint schema.
func BlueprintSpecFromSchema(bp *schema.Blueprint) BlueprintSpec {
	if bp == nil {
		return &defaultBlueprintSpec{
			resourceSchemas: map[string]*schema.Resource{},
			schema:          nil,
		}
	}

	resourceSchemas := map[string]*schema.Resource{}
	if bp.Resources != nil {
		maps.Copy(resourceSchemas, bp.Resources.Values)
	}

	return &defaultBlueprintSpec{
		resourceSchemas: resourceSchemas,
		schema:          bp,
	}
}
