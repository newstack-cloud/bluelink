package includes

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// ChildResolver is an interface for a service that resolves
// child blueprints referenced through the "include"
// property in a blueprint.
// An example of a ChildResolver implementation would be one
// that resolves includes from a local file system
// or a remote source such as Amazon S3.
type ChildResolver interface {
	// Resolve deals with resolving a child blueprint
	// referenced through the "include" property in a blueprint.
	// This should resolve to a file path on the local file system
	// or the source of the child blueprint loaded into memory.
	Resolve(
		ctx context.Context,
		includeName string,
		include *subengine.ResolvedInclude,
		params core.BlueprintParams,
	) (*ChildBlueprintInfo, error)
}

// ChildBlueprintInfo provides information about a child blueprint
// that has been resolved.
type ChildBlueprintInfo struct {
	// AbsolutePath is the absolute path to the child blueprint
	// on the local file system.
	AbsolutePath *string
	// BlueprintSource is the child blueprint loaded into memory.
	BlueprintSource *string
}
