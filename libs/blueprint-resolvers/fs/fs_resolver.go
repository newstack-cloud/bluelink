package resolverfs

import (
	"context"
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/spf13/afero"
)

type fsChildResolver struct {
	fs afero.Fs
}

// NewResolver creates a new instance of a ChildResolver
// that resolves child blueprints from the provided file system.
func NewResolver(fs afero.Fs) includes.ChildResolver {
	return &fsChildResolver{
		fs,
	}
}

func (r *fsChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	// Read the child blueprint from the file system,
	// the file system is expected to be relative to the absolute root
	// path on the current system.
	path := core.StringValue(include.Path)
	if path == "" {
		return nil, includes.ErrInvalidPath(includeName, "file system")
	}

	blueprintSource, err := afero.ReadFile(r.fs, path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, includes.ErrBlueprintNotFound(includeName, path)
		}
		if os.IsPermission(err) {
			return nil, includes.ErrPermissions(includeName, path, err)
		}
		return nil, err
	}

	blueprintSourceStr := string(blueprintSource)
	return &includes.ChildBlueprintInfo{
		BlueprintSource: &blueprintSourceStr,
	}, nil
}
