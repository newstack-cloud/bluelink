package utils

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	resolverfs "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// EnsureBlueprintDirContextVar adds the blueprint directory to the context variables
// so that the file system resolver can resolve relative child blueprint paths.
// If the config is nil, a new config is created with just the context variable.
// Returns the config (possibly newly created) with the blueprint directory set.
func EnsureBlueprintDirContextVar(config *types.BlueprintOperationConfig, directory string) *types.BlueprintOperationConfig {
	if directory == "" {
		return config
	}

	if config == nil {
		config = &types.BlueprintOperationConfig{}
	}

	if config.ContextVariables == nil {
		config.ContextVariables = make(map[string]*core.ScalarValue)
	}

	config.ContextVariables[resolverfs.BlueprintDirectoryContextVar] = &core.ScalarValue{
		StringValue: &directory,
	}

	return config
}
