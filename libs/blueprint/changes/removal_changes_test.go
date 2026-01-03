package changes

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type RemovalChangesTestSuite struct {
	suite.Suite
}

func TestRemovalChangesTestSuite(t *testing.T) {
	suite.Run(t, new(RemovalChangesTestSuite))
}

func (s *RemovalChangesTestSuite) Test_returns_nil_changes_when_state_nil() {
	result := CreateRemovalChangesFromInstanceState(nil)

	s.Nil(result.Changes)
	s.False(result.HasSkippedItems)
	s.Empty(result.SkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_created_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "createdResource",
				Status:     core.ResourceStatusCreated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedResources, "createdResource")
	s.False(result.HasSkippedItems)
	s.Empty(result.SkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_config_complete_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID:    "res-1",
				Name:          "configCompleteResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusConfigComplete,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedResources, "configCompleteResource")
	s.False(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_precise_created_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID:    "res-1",
				Name:          "preciseCreatedResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedResources, "preciseCreatedResource")
	s.False(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_skips_failed_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "failedResource",
				Status:     core.ResourceStatusCreateFailed,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.NotContains(result.Changes.RemovedResources, "failedResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("failedResource", result.SkippedItems[0].Name)
	s.Equal("resource", result.SkippedItems[0].Type)
	s.Equal("CREATE FAILED", result.SkippedItems[0].Status)
	s.Equal("resource creation was not completed successfully", result.SkippedItems[0].Reason)
}

func (s *RemovalChangesTestSuite) Test_skips_in_progress_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID:    "res-1",
				Name:          "creatingResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreating,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.NotContains(result.Changes.RemovedResources, "creatingResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("creatingResource", result.SkippedItems[0].Name)
	s.Equal("CREATING", result.SkippedItems[0].Status)
}

func (s *RemovalChangesTestSuite) Test_skips_interrupted_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "interruptedResource",
				Status:     core.ResourceStatusCreateInterrupted,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.NotContains(result.Changes.RemovedResources, "interruptedResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("interruptedResource", result.SkippedItems[0].Name)
	s.Equal("CREATE INTERRUPTED", result.SkippedItems[0].Status)
}

func (s *RemovalChangesTestSuite) Test_includes_created_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"createdLink": {
				LinkID: "link-1",
				Name:   "createdLink",
				Status: core.LinkStatusCreated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedLinks, "createdLink")
	s.False(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_updated_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"updatedLink": {
				LinkID: "link-1",
				Name:   "updatedLink",
				Status: core.LinkStatusUpdated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedLinks, "updatedLink")
	s.False(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_destroyed_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"destroyedLink": {
				LinkID: "link-1",
				Name:   "destroyedLink",
				Status: core.LinkStatusDestroyed,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedLinks, "destroyedLink")
	s.False(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_skips_failed_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"failedLink": {
				LinkID: "link-1",
				Name:   "failedLink",
				Status: core.LinkStatusCreateFailed,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.NotContains(result.Changes.RemovedLinks, "failedLink")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("failedLink", result.SkippedItems[0].Name)
	s.Equal("link", result.SkippedItems[0].Type)
	s.Equal("CREATE FAILED", result.SkippedItems[0].Status)
	s.Equal("link creation was not completed successfully", result.SkippedItems[0].Reason)
}

func (s *RemovalChangesTestSuite) Test_skips_in_progress_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"creatingLink": {
				LinkID: "link-1",
				Name:   "creatingLink",
				Status: core.LinkStatusCreating,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.NotContains(result.Changes.RemovedLinks, "creatingLink")
	s.True(result.HasSkippedItems)
}

func (s *RemovalChangesTestSuite) Test_includes_exports() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Exports: map[string]*state.ExportState{
			"export1": {Value: &core.MappingNode{}},
			"export2": {Value: &core.MappingNode{}},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedExports, "export1")
	s.Contains(result.Changes.RemovedExports, "export2")
}

func (s *RemovalChangesTestSuite) Test_includes_children() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		ChildBlueprints: map[string]*state.InstanceState{
			"child1": {InstanceID: "child-1"},
			"child2": {InstanceID: "child-2"},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedChildren, "child1")
	s.Contains(result.Changes.RemovedChildren, "child2")
}

func (s *RemovalChangesTestSuite) Test_handles_mixed_resource_states() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "successfulResource",
				Status:     core.ResourceStatusCreated,
			},
			"res-2": {
				ResourceID: "res-2",
				Name:       "failedResource",
				Status:     core.ResourceStatusCreateFailed,
			},
			"res-3": {
				ResourceID:    "res-3",
				Name:          "inProgressResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreating,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedResources, "successfulResource")
	s.NotContains(result.Changes.RemovedResources, "failedResource")
	s.NotContains(result.Changes.RemovedResources, "inProgressResource")

	s.True(result.HasSkippedItems)
	s.Len(result.SkippedItems, 2)

	skippedNames := make([]string, len(result.SkippedItems))
	for i, item := range result.SkippedItems {
		skippedNames[i] = item.Name
	}
	s.Contains(skippedNames, "failedResource")
	s.Contains(skippedNames, "inProgressResource")
}

func (s *RemovalChangesTestSuite) Test_handles_mixed_link_states() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"successfulLink": {
				LinkID: "link-1",
				Name:   "successfulLink",
				Status: core.LinkStatusCreated,
			},
			"failedLink": {
				LinkID: "link-2",
				Name:   "failedLink",
				Status: core.LinkStatusCreateFailed,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedLinks, "successfulLink")
	s.NotContains(result.Changes.RemovedLinks, "failedLink")

	s.True(result.HasSkippedItems)
	s.Len(result.SkippedItems, 1)
	s.Equal("failedLink", result.SkippedItems[0].Name)
}

func (s *RemovalChangesTestSuite) Test_includes_child_path_for_nested_children() {
	childState := &state.InstanceState{
		InstanceID: "child-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "failedChildResource",
				Status:     core.ResourceStatusCreateFailed,
			},
		},
	}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		ChildBlueprints: map[string]*state.InstanceState{
			"myChild": childState,
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedChildren, "myChild")

	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("failedChildResource", result.SkippedItems[0].Name)
	s.Equal("myChild", result.SkippedItems[0].ChildPath)
}

func (s *RemovalChangesTestSuite) Test_builds_correct_child_path_for_deeply_nested_children() {
	grandchildState := &state.InstanceState{
		InstanceID: "grandchild-instance",
		Resources: map[string]*state.ResourceState{
			"failedGrandchildResource": {
				ResourceID: "res-1",
				Name:       "failedGrandchildResource",
				Status:     core.ResourceStatusCreateFailed,
			},
		},
	}

	childState := &state.InstanceState{
		InstanceID: "child-instance",
		ChildBlueprints: map[string]*state.InstanceState{
			"grandchild": grandchildState,
		},
	}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		ChildBlueprints: map[string]*state.InstanceState{
			"child": childState,
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("failedGrandchildResource", result.SkippedItems[0].Name)
	s.Equal("child.grandchild", result.SkippedItems[0].ChildPath)

	// Verify the failed resource is NOT in the grandchild's RemovedResources
	childChanges, ok := result.Changes.ChildChanges["child"]
	s.Require().True(ok)
	grandchildChanges, ok := childChanges.ChildChanges["grandchild"]
	s.Require().True(ok)
	s.NotContains(grandchildChanges.RemovedResources, "failedGrandchildResource")
}

func (s *RemovalChangesTestSuite) Test_processes_child_resources_and_links() {
	childState := &state.InstanceState{
		InstanceID: "child-instance",
		Resources: map[string]*state.ResourceState{
			"childResource": {
				ResourceID: "res-1",
				Name:       "childResource",
				Status:     core.ResourceStatusCreated,
			},
		},
		Links: map[string]*state.LinkState{
			"childLink": {
				LinkID: "link-1",
				Name:   "childLink",
				Status: core.LinkStatusCreated,
			},
		},
	}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		ChildBlueprints: map[string]*state.InstanceState{
			"myChild": childState,
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Contains(result.Changes.RemovedChildren, "myChild")

	childChanges, ok := result.Changes.ChildChanges["myChild"]
	s.Require().True(ok)
	s.Contains(childChanges.RemovedResources, "childResource")
	s.Contains(childChanges.RemovedLinks, "childLink")
}

func (s *RemovalChangesTestSuite) Test_skips_nil_resources() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Resources: map[string]*state.ResourceState{
			"res-1": nil,
			"res-2": {
				ResourceID: "res-2",
				Name:       "validResource",
				Status:     core.ResourceStatusCreated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Len(result.Changes.RemovedResources, 1)
	s.Contains(result.Changes.RemovedResources, "validResource")
}

func (s *RemovalChangesTestSuite) Test_skips_nil_links() {
	instanceState := &state.InstanceState{
		InstanceID: "test-instance",
		Links: map[string]*state.LinkState{
			"nilLink":   nil,
			"validLink": {
				LinkID: "link-2",
				Name:   "validLink",
				Status: core.LinkStatusCreated,
			},
		},
	}

	result := CreateRemovalChangesFromInstanceState(instanceState)

	s.Require().NotNil(result.Changes)
	s.Len(result.Changes.RemovedLinks, 1)
	s.Contains(result.Changes.RemovedLinks, "validLink")
}

func (s *RemovalChangesTestSuite) Test_IsResourceSafeToDestroy_created_status() {
	resource := &state.ResourceState{
		Status: core.ResourceStatusCreated,
	}
	s.True(IsResourceSafeToDestroy(resource))
}

func (s *RemovalChangesTestSuite) Test_IsResourceSafeToDestroy_config_complete_precise_status() {
	resource := &state.ResourceState{
		Status:        core.ResourceStatusCreating,
		PreciseStatus: core.PreciseResourceStatusConfigComplete,
	}
	s.True(IsResourceSafeToDestroy(resource))
}

func (s *RemovalChangesTestSuite) Test_IsResourceSafeToDestroy_precise_created_status() {
	resource := &state.ResourceState{
		Status:        core.ResourceStatusCreating,
		PreciseStatus: core.PreciseResourceStatusCreated,
	}
	s.True(IsResourceSafeToDestroy(resource))
}

func (s *RemovalChangesTestSuite) Test_IsResourceSafeToDestroy_creating_status() {
	resource := &state.ResourceState{
		Status:        core.ResourceStatusCreating,
		PreciseStatus: core.PreciseResourceStatusCreating,
	}
	s.False(IsResourceSafeToDestroy(resource))
}

func (s *RemovalChangesTestSuite) Test_IsResourceSafeToDestroy_failed_status() {
	resource := &state.ResourceState{
		Status: core.ResourceStatusCreateFailed,
	}
	s.False(IsResourceSafeToDestroy(resource))
}
