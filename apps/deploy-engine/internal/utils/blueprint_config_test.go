package utils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	resolverfs "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/assert"
)

func TestEnsureBlueprintDirContextVar_adds_directory_to_empty_config(t *testing.T) {
	config := &types.BlueprintOperationConfig{}
	directory := "/path/to/blueprint"

	result := EnsureBlueprintDirContextVar(config, directory)

	assert.NotNil(t, result.ContextVariables)
	assert.Equal(t, directory, *result.ContextVariables[resolverfs.BlueprintDirectoryContextVar].StringValue)
}

func TestEnsureBlueprintDirContextVar_adds_directory_to_existing_context_vars(t *testing.T) {
	existingValue := "existing-value"
	config := &types.BlueprintOperationConfig{
		ContextVariables: map[string]*core.ScalarValue{
			"existingVar": {
				StringValue: &existingValue,
			},
		},
	}
	directory := "/path/to/blueprint"

	result := EnsureBlueprintDirContextVar(config, directory)

	assert.Equal(t, directory, *result.ContextVariables[resolverfs.BlueprintDirectoryContextVar].StringValue)
	// Existing var should still be present
	assert.Equal(t, existingValue, *result.ContextVariables["existingVar"].StringValue)
}

func TestEnsureBlueprintDirContextVar_creates_config_for_nil_input(t *testing.T) {
	directory := "/path/to/blueprint"

	result := EnsureBlueprintDirContextVar(nil, directory)

	assert.NotNil(t, result)
	assert.NotNil(t, result.ContextVariables)
	assert.Equal(t, directory, *result.ContextVariables[resolverfs.BlueprintDirectoryContextVar].StringValue)
}

func TestEnsureBlueprintDirContextVar_returns_nil_for_empty_directory_and_nil_config(t *testing.T) {
	result := EnsureBlueprintDirContextVar(nil, "")

	// Should return nil since no directory was provided
	assert.Nil(t, result)
}

func TestEnsureBlueprintDirContextVar_returns_config_unchanged_for_empty_directory(t *testing.T) {
	config := &types.BlueprintOperationConfig{}

	result := EnsureBlueprintDirContextVar(config, "")

	// Should return same config unchanged
	assert.Same(t, config, result)
	assert.Nil(t, result.ContextVariables)
}

func TestEnsureBlueprintDirContextVar_overwrites_existing_blueprint_dir(t *testing.T) {
	oldDir := "/old/path"
	config := &types.BlueprintOperationConfig{
		ContextVariables: map[string]*core.ScalarValue{
			resolverfs.BlueprintDirectoryContextVar: {
				StringValue: &oldDir,
			},
		},
	}
	newDir := "/new/path"

	result := EnsureBlueprintDirContextVar(config, newDir)

	assert.Equal(t, newDir, *result.ContextVariables[resolverfs.BlueprintDirectoryContextVar].StringValue)
}
