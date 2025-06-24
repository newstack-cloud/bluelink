package changes

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

func CollectMetadataChanges(
	collectIn *MetadataChanges,
	resolveResult *subengine.ResolveInMappingNodeResult,
	currentMetadata map[string]*core.MappingNode,
) {

	if resolveResult.ResolvedMappingNode == nil ||
		!core.IsObjectMappingNode(resolveResult.ResolvedMappingNode) {
		// Blueprint-wide metadata must always be an object/map.
		return
	}

	changes := &provider.Changes{
		NewFields:                 []provider.FieldChange{},
		ModifiedFields:            []provider.FieldChange{},
		RemovedFields:             []string{},
		UnchangedFields:           []string{},
		FieldChangesKnownOnDeploy: resolveResult.ResolveOnDeploy,
	}
	// To reuse the same logic used to collect changes in the `metadata.custom`
	// field of a resource, collect changes in the resource provider.Changes structure
	// and then map the collected changes to the MetadataChanges structure.
	collectMappingNodeChanges(
		changes,
		resolveResult.ResolvedMappingNode,
		&core.MappingNode{
			Fields: currentMetadata,
		},
		&fieldChangeContext{
			fieldsToResolveOnDeploy: resolveResult.ResolveOnDeploy,
			currentPath:             "metadata",
			depth:                   0,
		},
	)

	collectIn.NewFields = changes.NewFields
	collectIn.ModifiedFields = changes.ModifiedFields
	collectIn.RemovedFields = changes.RemovedFields
	collectIn.UnchangedFields = changes.UnchangedFields
}

// AnyEmptyString returns true if any of the given strings are empty
// or contain only whitespace.
func AnyEmptyString(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}
