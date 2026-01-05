package shared

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PathBuildersSuite struct {
	suite.Suite
}

func TestPathBuildersSuite(t *testing.T) {
	suite.Run(t, new(PathBuildersSuite))
}

// --- JoinPath Tests ---

func (s *PathBuildersSuite) Test_JoinPath_returns_empty_for_empty_parts() {
	s.Equal("", JoinPath([]string{}))
}

func (s *PathBuildersSuite) Test_JoinPath_returns_single_part_unchanged() {
	s.Equal("resource", JoinPath([]string{"resource"}))
}

func (s *PathBuildersSuite) Test_JoinPath_joins_two_parts_with_slash() {
	s.Equal("parent/child", JoinPath([]string{"parent", "child"}))
}

func (s *PathBuildersSuite) Test_JoinPath_joins_multiple_parts() {
	s.Equal("a/b/c/d", JoinPath([]string{"a", "b", "c", "d"}))
}

// --- NewPathBuilder Tests ---

func (s *PathBuildersSuite) Test_NewPathBuilder_initializes_with_root_instance_id() {
	pb := NewPathBuilder("root-123")
	s.Equal("root-123", pb.RootInstanceID)
	s.NotNil(pb.InstanceIDToChildName)
	s.NotNil(pb.InstanceIDToParentID)
}

// --- BuildInstancePath Tests ---

func (s *PathBuildersSuite) Test_BuildInstancePath_returns_name_for_empty_parent() {
	pb := NewPathBuilder("root-123")
	s.Equal("child-name", pb.BuildInstancePath("", "child-name"))
}

func (s *PathBuildersSuite) Test_BuildInstancePath_returns_name_for_root_parent() {
	pb := NewPathBuilder("root-123")
	s.Equal("child-name", pb.BuildInstancePath("root-123", "child-name"))
}

func (s *PathBuildersSuite) Test_BuildInstancePath_builds_nested_path() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-instance-1", "parent-child", "root-123")

	result := pb.BuildInstancePath("child-instance-1", "nested-child")
	s.Equal("parent-child/nested-child", result)
}

func (s *PathBuildersSuite) Test_BuildInstancePath_builds_deeply_nested_path() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("level1-id", "level1", "root-123")
	pb.TrackChildInstance("level2-id", "level2", "level1-id")
	pb.TrackChildInstance("level3-id", "level3", "level2-id")

	result := pb.BuildInstancePath("level3-id", "deep-child")
	s.Equal("level1/level2/level3/deep-child", result)
}

// --- BuildItemPath Tests ---

func (s *PathBuildersSuite) Test_BuildItemPath_returns_name_for_empty_instance() {
	pb := NewPathBuilder("root-123")
	s.Equal("my-resource", pb.BuildItemPath("", "my-resource"))
}

func (s *PathBuildersSuite) Test_BuildItemPath_returns_name_for_root_instance() {
	pb := NewPathBuilder("root-123")
	s.Equal("my-resource", pb.BuildItemPath("root-123", "my-resource"))
}

func (s *PathBuildersSuite) Test_BuildItemPath_builds_path_for_nested_resource() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-instance-1", "child-blueprint", "root-123")

	result := pb.BuildItemPath("child-instance-1", "nested-resource")
	s.Equal("child-blueprint/nested-resource", result)
}

// --- BuildParentChain Tests ---

func (s *PathBuildersSuite) Test_BuildParentChain_returns_empty_for_root() {
	pb := NewPathBuilder("root-123")
	result := pb.BuildParentChain("root-123")
	s.Empty(result)
}

func (s *PathBuildersSuite) Test_BuildParentChain_returns_empty_for_empty_id() {
	pb := NewPathBuilder("root-123")
	result := pb.BuildParentChain("")
	s.Empty(result)
}

func (s *PathBuildersSuite) Test_BuildParentChain_returns_single_element_for_direct_child() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-1", "child-name", "root-123")

	result := pb.BuildParentChain("child-1")
	s.Equal([]string{"child-name"}, result)
}

func (s *PathBuildersSuite) Test_BuildParentChain_returns_full_chain_for_nested_child() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-1", "level1", "root-123")
	pb.TrackChildInstance("child-2", "level2", "child-1")

	result := pb.BuildParentChain("child-2")
	s.Equal([]string{"level1", "level2"}, result)
}

func (s *PathBuildersSuite) Test_BuildParentChain_handles_unknown_instance() {
	pb := NewPathBuilder("root-123")
	result := pb.BuildParentChain("unknown-id")
	s.Empty(result)
}

// --- TrackChildInstance Tests ---

func (s *PathBuildersSuite) Test_TrackChildInstance_stores_mapping() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-id", "child-name", "parent-id")

	s.Equal("child-name", pb.InstanceIDToChildName["child-id"])
	s.Equal("parent-id", pb.InstanceIDToParentID["child-id"])
}

func (s *PathBuildersSuite) Test_TrackChildInstance_ignores_empty_instance_id() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("", "child-name", "parent-id")

	s.Empty(pb.InstanceIDToChildName)
	s.Empty(pb.InstanceIDToParentID)
}

func (s *PathBuildersSuite) Test_TrackChildInstance_ignores_empty_child_name() {
	pb := NewPathBuilder("root-123")
	pb.TrackChildInstance("child-id", "", "parent-id")

	s.Empty(pb.InstanceIDToChildName)
	s.Empty(pb.InstanceIDToParentID)
}
