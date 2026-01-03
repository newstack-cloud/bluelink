package changes

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type RollbackFilterTestSuite struct {
	suite.Suite
}

func (s *RollbackFilterTestSuite) Test_returns_original_when_changes_nil() {
	result := FilterReverseChangesetByCurrentState(nil, &state.InstanceState{})
	s.Nil(result.FilteredChanges)
	s.False(result.HasSkippedItems)
	s.Empty(result.SkippedItems)
}

func (s *RollbackFilterTestSuite) Test_returns_original_when_state_nil() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {},
		},
	}
	result := FilterReverseChangesetByCurrentState(original, nil)
	s.Equal(original, result.FilteredChanges)
	s.False(result.HasSkippedItems)
	s.Empty(result.SkippedItems)
}

func (s *RollbackFilterTestSuite) Test_includes_resource_changes_when_resource_updated() {
	reverseChanges := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"myResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "myResource",
				Status:     core.ResourceStatusUpdated,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.ResourceChanges, "myResource")
	s.False(result.HasSkippedItems)
	s.Empty(result.SkippedItems)
}

func (s *RollbackFilterTestSuite) Test_includes_resource_changes_when_config_complete() {
	reverseChanges := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"myResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID:    "res-id-1",
				Name:          "myResource",
				Status:        core.ResourceStatusUpdating,
				PreciseStatus: core.PreciseResourceStatusUpdateConfigComplete,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.ResourceChanges, "myResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_skips_resource_changes_when_update_failed() {
	reverseChanges := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"myResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "myResource",
				Status:     core.ResourceStatusUpdateFailed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.ResourceChanges, "myResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("myResource", result.SkippedItems[0].Name)
	s.Equal("resource", result.SkippedItems[0].Type)
	s.Equal("UPDATE FAILED", result.SkippedItems[0].Status)
}

func (s *RollbackFilterTestSuite) Test_includes_removed_resources_when_created() {
	reverseChanges := &BlueprintChanges{
		RemovedResources: []string{"newResource"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"newResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "newResource",
				Status:     core.ResourceStatusCreated,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.RemovedResources, "newResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_includes_removed_resources_when_config_complete() {
	reverseChanges := &BlueprintChanges{
		RemovedResources: []string{"newResource"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"newResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID:    "res-id-1",
				Name:          "newResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusConfigComplete,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.RemovedResources, "newResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_skips_removed_resources_when_create_failed() {
	reverseChanges := &BlueprintChanges{
		RemovedResources: []string{"newResource"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"newResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "newResource",
				Status:     core.ResourceStatusCreateFailed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.RemovedResources, "newResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("newResource", result.SkippedItems[0].Name)
	s.Equal("CREATE FAILED", result.SkippedItems[0].Status)
}

func (s *RollbackFilterTestSuite) Test_skips_removed_resources_when_creating() {
	reverseChanges := &BlueprintChanges{
		RemovedResources: []string{"newResource"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"newResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID:    "res-id-1",
				Name:          "newResource",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreating,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.RemovedResources, "newResource")
	s.True(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_includes_new_resources_when_destroyed() {
	reverseChanges := &BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"deletedResource": {},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"deletedResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "deletedResource",
				Status:     core.ResourceStatusDestroyed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.NewResources, "deletedResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_skips_new_resources_when_destroy_failed() {
	reverseChanges := &BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"deletedResource": {},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"deletedResource": "res-id-1",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "deletedResource",
				Status:     core.ResourceStatusDestroyFailed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.NewResources, "deletedResource")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("deletedResource", result.SkippedItems[0].Name)
	s.Equal("DESTROY FAILED", result.SkippedItems[0].Status)
}

func (s *RollbackFilterTestSuite) Test_includes_removed_links_when_created() {
	reverseChanges := &BlueprintChanges{
		RemovedLinks: []string{"newLink"},
	}
	currentState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"newLink": {
				LinkID: "link-id-1",
				Name:   "newLink",
				Status: core.LinkStatusCreated,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.RemovedLinks, "newLink")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_skips_removed_links_when_create_failed() {
	reverseChanges := &BlueprintChanges{
		RemovedLinks: []string{"newLink"},
	}
	currentState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"newLink": {
				LinkID: "link-id-1",
				Name:   "newLink",
				Status: core.LinkStatusCreateFailed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.RemovedLinks, "newLink")
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("newLink", result.SkippedItems[0].Name)
	s.Equal("link", result.SkippedItems[0].Type)
	s.Equal("CREATE FAILED", result.SkippedItems[0].Status)
}

func (s *RollbackFilterTestSuite) Test_filters_child_changes_recursively() {
	childChanges := BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"childResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}
	reverseChanges := &BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"childBlueprint": childChanges,
		},
	}
	currentState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID: "child-instance",
				ResourceIDs: map[string]string{
					"childResource": "child-res-id",
				},
				Resources: map[string]*state.ResourceState{
					"child-res-id": {
						ResourceID: "child-res-id",
						Name:       "childResource",
						Status:     core.ResourceStatusUpdateFailed,
					},
				},
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("childResource", result.SkippedItems[0].Name)
	s.Equal("childBlueprint", result.SkippedItems[0].ChildPath)
}

func (s *RollbackFilterTestSuite) Test_filters_mixed_success_and_failure() {
	reverseChanges := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"successResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
			"failedResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
		RemovedResources: []string{"newSuccessResource", "newFailedResource"},
		RemovedLinks:     []string{"successLink", "failedLink"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"successResource":    "res-id-1",
			"failedResource":     "res-id-2",
			"newSuccessResource": "res-id-3",
			"newFailedResource":  "res-id-4",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "successResource",
				Status:     core.ResourceStatusUpdated,
			},
			"res-id-2": {
				ResourceID: "res-id-2",
				Name:       "failedResource",
				Status:     core.ResourceStatusUpdateFailed,
			},
			"res-id-3": {
				ResourceID: "res-id-3",
				Name:       "newSuccessResource",
				Status:     core.ResourceStatusCreated,
			},
			"res-id-4": {
				ResourceID: "res-id-4",
				Name:       "newFailedResource",
				Status:     core.ResourceStatusCreateFailed,
			},
		},
		Links: map[string]*state.LinkState{
			"successLink": {
				LinkID: "link-id-1",
				Name:   "successLink",
				Status: core.LinkStatusCreated,
			},
			"failedLink": {
				LinkID: "link-id-2",
				Name:   "failedLink",
				Status: core.LinkStatusCreateFailed,
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)

	// Should include successful items
	s.Contains(result.FilteredChanges.ResourceChanges, "successResource")
	s.Contains(result.FilteredChanges.RemovedResources, "newSuccessResource")
	s.Contains(result.FilteredChanges.RemovedLinks, "successLink")

	// Should exclude failed items
	s.NotContains(result.FilteredChanges.ResourceChanges, "failedResource")
	s.NotContains(result.FilteredChanges.RemovedResources, "newFailedResource")
	s.NotContains(result.FilteredChanges.RemovedLinks, "failedLink")

	// Should report skipped items
	s.True(result.HasSkippedItems)
	s.Len(result.SkippedItems, 3)

	skippedNames := make([]string, len(result.SkippedItems))
	for i, item := range result.SkippedItems {
		skippedNames[i] = item.Name
	}
	s.ElementsMatch([]string{"failedResource", "newFailedResource", "failedLink"}, skippedNames)
}

func (s *RollbackFilterTestSuite) Test_preserves_exports_and_metadata() {
	reverseChanges := &BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {FieldPath: "resources.myResource.spec.output"},
		},
		ExportChanges: map[string]provider.FieldChange{
			"export2": {FieldPath: "resources.myResource.spec.value"},
		},
		RemovedExports:   []string{"export3"},
		UnchangedExports: []string{"export4"},
		MetadataChanges: MetadataChanges{
			ModifiedFields: []provider.FieldChange{
				{FieldPath: "metadata.labels.app"},
			},
		},
	}
	currentState := &state.InstanceState{}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.NewExports, "export1")
	s.Contains(result.FilteredChanges.ExportChanges, "export2")
	s.Contains(result.FilteredChanges.RemovedExports, "export3")
	s.ElementsMatch([]string{"export4"}, result.FilteredChanges.UnchangedExports)
	s.Len(result.FilteredChanges.MetadataChanges.ModifiedFields, 1)
}

func (s *RollbackFilterTestSuite) Test_includes_resource_when_not_in_state() {
	reverseChanges := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"unknownResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{},
		Resources:   map[string]*state.ResourceState{},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.ResourceChanges, "unknownResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_skips_removed_resource_not_in_state() {
	reverseChanges := &BlueprintChanges{
		RemovedResources: []string{"nonExistentResource"},
	}
	currentState := &state.InstanceState{
		ResourceIDs: map[string]string{},
		Resources:   map[string]*state.ResourceState{},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.NotContains(result.FilteredChanges.RemovedResources, "nonExistentResource")
	s.False(result.HasSkippedItems)
}

func (s *RollbackFilterTestSuite) Test_copies_new_and_removed_children() {
	reverseChanges := &BlueprintChanges{
		NewChildren: map[string]NewBlueprintDefinition{
			"newChild": {},
		},
		RemovedChildren: []string{"removedChild"},
	}
	currentState := &state.InstanceState{}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.Require().NotNil(result.FilteredChanges)
	s.Contains(result.FilteredChanges.NewChildren, "newChild")
	s.Contains(result.FilteredChanges.RemovedChildren, "removedChild")
}

func (s *RollbackFilterTestSuite) Test_builds_correct_child_path_for_nested_children() {
	nestedChildChanges := BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"nestedResource": {},
		},
	}
	childChanges := BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"nestedChild": nestedChildChanges,
		},
	}
	reverseChanges := &BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"parentChild": childChanges,
		},
	}
	currentState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"parentChild": {
				InstanceID: "parent-child-instance",
				ChildBlueprints: map[string]*state.InstanceState{
					"nestedChild": {
						InstanceID: "nested-child-instance",
						ResourceIDs: map[string]string{
							"nestedResource": "nested-res-id",
						},
						Resources: map[string]*state.ResourceState{
							"nested-res-id": {
								ResourceID: "nested-res-id",
								Name:       "nestedResource",
								Status:     core.ResourceStatusUpdateFailed,
							},
						},
					},
				},
			},
		},
	}

	result := FilterReverseChangesetByCurrentState(reverseChanges, currentState)

	s.True(result.HasSkippedItems)
	s.Require().Len(result.SkippedItems, 1)
	s.Equal("nestedResource", result.SkippedItems[0].Name)
	s.Equal("parentChild.nestedChild", result.SkippedItems[0].ChildPath)
}

func TestRollbackFilterTestSuite(t *testing.T) {
	suite.Run(t, new(RollbackFilterTestSuite))
}
