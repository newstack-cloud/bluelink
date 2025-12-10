package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type ChildBlueprintUtilsSuite struct {
	suite.Suite
}

func (s *ChildBlueprintUtilsSuite) Test_createContextVarsForChildBlueprint_includes_all_vars() {
	parentInstanceID := "parent-123"
	instanceTreePath := "root/child1"
	includeTreePath := "include.child1"
	blueprintDir := "/path/to/blueprint"

	contextVars := createContextVarsForChildBlueprint(
		parentInstanceID,
		instanceTreePath,
		includeTreePath,
		blueprintDir,
	)

	s.Assert().Equal(parentInstanceID, *contextVars["parentInstanceID"].StringValue)
	s.Assert().Equal(instanceTreePath, *contextVars["instanceTreePath"].StringValue)
	s.Assert().Equal(includeTreePath, *contextVars["includeTreePath"].StringValue)
	s.Assert().Equal(blueprintDir, *contextVars[BlueprintDirectoryContextVar].StringValue)
}

func (s *ChildBlueprintUtilsSuite) Test_createContextVarsForChildBlueprint_omits_empty_blueprint_dir() {
	parentInstanceID := "parent-123"
	instanceTreePath := "root/child1"
	includeTreePath := "include.child1"
	blueprintDir := "" // Empty

	contextVars := createContextVarsForChildBlueprint(
		parentInstanceID,
		instanceTreePath,
		includeTreePath,
		blueprintDir,
	)

	s.Assert().Equal(parentInstanceID, *contextVars["parentInstanceID"].StringValue)
	s.Assert().Equal(instanceTreePath, *contextVars["instanceTreePath"].StringValue)
	s.Assert().Equal(includeTreePath, *contextVars["includeTreePath"].StringValue)
	// Blueprint dir should not be present when empty
	_, exists := contextVars[BlueprintDirectoryContextVar]
	s.Assert().False(exists)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_from_absolute_path() {
	absPath := "/path/to/child/blueprint.yaml"
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		AbsolutePath: &absPath,
	}
	resolvedInclude := &subengine.ResolvedInclude{}

	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, nil)

	s.Assert().Equal("/path/to/child", result)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_from_absolute_include_path() {
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		BlueprintSource: strPtr("blueprint content"),
	}
	includePath := "/absolute/path/to/child.blueprint.yaml"
	resolvedInclude := &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(includePath),
	}

	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, nil)

	s.Assert().Equal("/absolute/path/to", result)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_from_relative_path_with_parent_dir() {
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		BlueprintSource: strPtr("blueprint content"),
	}
	includePath := "subdir/child.blueprint.yaml"
	resolvedInclude := &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(includePath),
	}

	parentDir := "/parent/blueprint/dir"
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			BlueprintDirectoryContextVar: {
				StringValue: &parentDir,
			},
		},
		map[string]*core.ScalarValue{},
	)

	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, params)

	s.Assert().Equal("/parent/blueprint/dir/subdir", result)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_from_relative_path_same_dir() {
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		BlueprintSource: strPtr("blueprint content"),
	}
	includePath := "child.blueprint.yaml" // No subdirectory
	resolvedInclude := &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(includePath),
	}

	parentDir := "/parent/blueprint/dir"
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			BlueprintDirectoryContextVar: {
				StringValue: &parentDir,
			},
		},
		map[string]*core.ScalarValue{},
	)

	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, params)

	s.Assert().Equal("/parent/blueprint/dir", result)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_returns_empty_for_relative_path_without_parent() {
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		BlueprintSource: strPtr("blueprint content"),
	}
	includePath := "child.blueprint.yaml"
	resolvedInclude := &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(includePath),
	}

	// No params, so no parent directory
	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, nil)

	s.Assert().Equal("", result)
}

func (s *ChildBlueprintUtilsSuite) Test_deriveChildBlueprintDir_returns_empty_for_empty_path() {
	childBlueprintInfo := &includes.ChildBlueprintInfo{
		BlueprintSource: strPtr("blueprint content"),
	}
	resolvedInclude := &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(""),
	}

	result := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, nil)

	s.Assert().Equal("", result)
}

func strPtr(s string) *string {
	return &s
}

func TestChildBlueprintUtilsSuite(t *testing.T) {
	suite.Run(t, new(ChildBlueprintUtilsSuite))
}
