package validation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ValidationContext bundles the common dependencies used across
// validation functions in the blueprint validation pipeline.
type ValidationContext struct {
	BpSchema           *schema.Blueprint
	Params             core.BlueprintParams
	FuncRegistry       provider.FunctionRegistry
	RefChainCollector  refgraph.RefChainCollector
	ResourceRegistry   resourcehelpers.Registry
	DataSourceRegistry provider.DataSourceRegistry
	ChildExportLookup  ChildExportTypeLookup
}
