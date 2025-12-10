package resolverfs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/spf13/afero"
)

// BlueprintDirectoryContextVar is the name of the context variable
// that holds the directory of the current blueprint being processed.
// This is used to resolve relative paths in child blueprint includes.
const BlueprintDirectoryContextVar = "__blueprintDir"

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
	includePath := core.StringValue(include.Path)
	if includePath == "" {
		return nil, includes.ErrInvalidPath(includeName, "file system")
	}

	// Resolve relative paths against the parent blueprint's directory.
	resolvedPath := includePath
	if !filepath.IsAbs(includePath) {
		baseDir := getBaseDirectory(params)
		if baseDir != "" {
			resolvedPath = filepath.Join(baseDir, includePath)
		}
	}

	blueprintSource, err := afero.ReadFile(r.fs, resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, includes.ErrBlueprintNotFound(includeName, resolvedPath)
		}
		if os.IsPermission(err) {
			return nil, includes.ErrPermissions(includeName, resolvedPath, err)
		}
		return nil, err
	}

	blueprintSourceStr := string(blueprintSource)
	return &includes.ChildBlueprintInfo{
		BlueprintSource: &blueprintSourceStr,
	}, nil
}

// getBaseDirectory retrieves the base directory for resolving relative paths.
// It first checks for a blueprint directory context variable, then falls back
// to the current working directory.
func getBaseDirectory(params core.BlueprintParams) string {
	if params == nil {
		return ""
	}

	// Check for the blueprint directory context variable
	blueprintDir := params.ContextVariable(BlueprintDirectoryContextVar)
	if blueprintDir != nil && blueprintDir.StringValue != nil {
		return *blueprintDir.StringValue
	}

	return ""
}
