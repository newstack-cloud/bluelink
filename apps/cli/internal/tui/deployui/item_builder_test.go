package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ItemBuilderTestSuite struct {
	suite.Suite
}

func TestItemBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(ItemBuilderTestSuite))
}

// BuildItemsFromChangeset tests

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_returns_empty_for_nil_changeset() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	items := BuildItemsFromChangeset(nil, resourcesByName, childrenByName, linksByName, nil)

	s.Empty(items)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_returns_empty_for_empty_changeset() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	items := BuildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, nil)

	s.Empty(items)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_new_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("newResource", items[0].Resource.Name)
	s.Equal(ActionCreate, items[0].Resource.Action)
	// Should be added to the shared map
	s.NotNil(resourcesByName["newResource"])
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_changed_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"changedResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("changedResource", items[0].Resource.Name)
	s.Equal(ActionUpdate, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_removed_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"removedResource"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("removedResource", items[0].Resource.Name)
	s.Equal(ActionDelete, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_removed_resources_with_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"removedResource": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "removedResource",
				Type:       "aws/s3/bucket",
			},
		},
	}

	bpChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"removedResource"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ActionDelete, items[0].Resource.Action)
	s.NotNil(items[0].Resource.ResourceState)
	s.Equal("res-123", items[0].Resource.ResourceState.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_new_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild": {
				NewResources: map[string]provider.Changes{
					"nestedResource": {},
				},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("newChild", items[0].Child.Name)
	s.Equal(ActionCreate, items[0].Child.Action)
	s.NotNil(items[0].Changes)
	// Should be added to the shared map
	s.NotNil(childrenByName["newChild"])
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_changed_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"changedChild": {InstanceID: "child-instance-123"},
		},
	}

	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"changedChild": {
				ResourceChanges: map[string]provider.Changes{
					"nestedResource": {
						ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
					},
				},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("changedChild", items[0].Child.Name)
	s.Equal(ActionUpdate, items[0].Child.Action)
	s.NotNil(items[0].InstanceState)
	s.Equal("child-instance-123", items[0].InstanceState.InstanceID)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_recreate_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RecreateChildren: []string{"recreateChild"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("recreateChild", items[0].Child.Name)
	s.Equal(ActionRecreate, items[0].Child.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_removed_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedChildren: []string{"removedChild"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("removedChild", items[0].Child.Name)
	s.Equal(ActionDelete, items[0].Child.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_links_from_new_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"resourceA": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"resourceB": {},
				},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// Should have both the resource and the link
	s.Len(items, 2)

	var linkItem *DeployItem
	for idx := range items {
		if items[idx].Type == ItemTypeLink {
			linkItem = &items[idx]
			break
		}
	}
	s.NotNil(linkItem)
	s.Equal("resourceA::resourceB", linkItem.Link.LinkName)
	s.Equal(ActionCreate, linkItem.Link.Action)
	s.Equal("resourceA", linkItem.Link.ResourceAName)
	s.Equal("resourceB", linkItem.Link.ResourceBName)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_links_from_changed_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"resourceA": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"resourceB": {},
				},
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"resourceC": {},
				},
				RemovedOutboundLinks: []string{"resourceA::resourceD"},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// Should have the resource and 3 links
	s.Len(items, 4)

	var linkActions []ActionType
	for idx := range items {
		if items[idx].Type == ItemTypeLink {
			linkActions = append(linkActions, items[idx].Link.Action)
		}
	}
	s.Contains(linkActions, ActionCreate) // New link
	s.Contains(linkActions, ActionUpdate) // Changed link
	s.Contains(linkActions, ActionDelete) // Removed link
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_removed_links() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedLinks: []string{"resX::resY"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeLink, items[0].Type)
	s.Equal("resX::resY", items[0].Link.LinkName)
	s.Equal(ActionDelete, items[0].Link.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_skips_duplicate_removed_links() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	// Link is both in RemovedOutboundLinks and RemovedLinks
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"resourceA": {
				RemovedOutboundLinks: []string{"resourceA::resourceB"},
			},
		},
		RemovedLinks: []string{"resourceA::resourceB"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// Should only have resource and one link (not duplicated)
	linkCount := 0
	for idx := range items {
		if items[idx].Type == ItemTypeLink {
			linkCount += 1
		}
	}
	s.Equal(1, linkCount)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_no_change_resources_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "unchangedResource",
				Type:       "aws/s3/bucket",
			},
		},
	}

	items := BuildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("unchangedResource", items[0].Resource.Name)
	s.Equal(ActionNoChange, items[0].Resource.Action)
	s.Equal("res-123", items[0].Resource.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_no_change_children_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"unchangedChild": {InstanceID: "child-instance-456"},
		},
	}

	items := BuildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("unchangedChild", items[0].Child.Name)
	s.Equal(ActionNoChange, items[0].Child.Action)
	s.NotNil(items[0].InstanceState)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_adds_no_change_links_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"resA::resB": {
				LinkID: "link-789",
				Status: core.LinkStatusCreated,
			},
		},
	}

	items := BuildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeLink, items[0].Type)
	s.Equal("resA::resB", items[0].Link.LinkName)
	s.Equal(ActionNoChange, items[0].Link.Action)
	s.Equal("link-789", items[0].Link.LinkID)
	s.Equal(core.LinkStatusCreated, items[0].Link.Status)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_does_not_duplicate_changed_items_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "changedResource",
				Type:       "aws/s3/bucket",
			},
		},
	}

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"changedResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	// Should only have one resource (from changes, not duplicated from state)
	s.Len(items, 1)
	s.Equal("changedResource", items[0].Resource.Name)
	s.Equal(ActionUpdate, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_populates_nested_items_in_shared_maps() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	// Create a nested hierarchy:
	// - parentChild (NewChildren at root level)
	//   - grandChild (NewChildren inside parentChild)
	//     - deepNestedResource (NewResources in grandChild)
	//     - greatGrandChild (NewChildren inside grandChild)
	//       - deepestResource (NewResources in greatGrandChild)
	bpChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"parentChild": {
				NewChildren: map[string]changes.NewBlueprintDefinition{
					"grandChild": {
						NewResources: map[string]provider.Changes{
							"deepNestedResource": {},
						},
						NewChildren: map[string]changes.NewBlueprintDefinition{
							"greatGrandChild": {
								NewResources: map[string]provider.Changes{
									"deepestResource": {},
								},
							},
						},
					},
				},
			},
		},
	}

	_ = BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// parentChild should be added via appendNewChildren
	s.NotNil(childrenByName["parentChild"])
	// grandChild, deepNestedResource, greatGrandChild, and deepestResource should be added
	// via populateNestedItems -> populateNestedNewChildren recursively
	s.NotNil(childrenByName["grandChild"])
	s.NotNil(resourcesByName["deepNestedResource"])
	s.NotNil(childrenByName["greatGrandChild"])
	s.NotNil(resourcesByName["deepestResource"])
}

// Tests for action determination behavior (via BuildItemsFromChangeset)

func (s *ItemBuilderTestSuite) Test_changed_resource_returns_recreate_for_must_recreate() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {MustRecreate: true},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ActionRecreate, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_changed_resource_returns_no_change_for_no_field_changes() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ActionNoChange, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_changed_resource_returns_update_for_field_changes() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ActionUpdate, items[0].Resource.Action)
}

// Tests for resource info extraction behavior (via BuildItemsFromChangeset)

func (s *ItemBuilderTestSuite) Test_changed_resource_extracts_resource_id() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID: "res-info-123",
				},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal("res-info-123", items[0].Resource.ResourceID)
	s.Equal("", items[0].Resource.ResourceType)
}

func (s *ItemBuilderTestSuite) Test_changed_resource_extracts_resource_type_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID: "res-info-456",
					CurrentResourceState: &state.ResourceState{
						Type: "aws/lambda/function",
					},
				},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal("res-info-456", items[0].Resource.ResourceID)
	s.Equal("aws/lambda/function", items[0].Resource.ResourceType)
}

// Tests for resource state lookup behavior (via BuildItemsFromChangeset)

func (s *ItemBuilderTestSuite) Test_changed_resource_includes_resource_state_from_instance() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"testResource": "res-state-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-state-123": {
				ResourceID: "res-state-123",
				Name:       "testResource",
				Type:       "aws/sqs/queue",
			},
		},
	}

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.NotNil(items[0].Resource.ResourceState)
	s.Equal("res-state-123", items[0].Resource.ResourceState.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_changed_resource_has_nil_state_when_not_in_instance() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{},
		Resources:   map[string]*state.ResourceState{},
	}

	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"unknownResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Nil(items[0].Resource.ResourceState)
}

// Tests for nested changed resources (via BuildItemsFromChangeset shared maps)
// Note: Resources inside ChildChanges are populated via populateNestedChangedChildren,
// which recurses through nested ChildChanges. For direct ResourceChanges within a
// ChildChange, they're added when the child itself is processed by appendChangedChildren
// and then recursively via populateNestedItems.

func (s *ItemBuilderTestSuite) Test_nested_changed_resource_creates_item_in_shared_maps() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	// Use a structure with nested ChildChanges to test recursive resource population
	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"parentChild": {
				ChildChanges: map[string]changes.BlueprintChanges{
					"grandChild": {
						ResourceChanges: map[string]provider.Changes{
							"nestedResource": {
								ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
								AppliedResourceInfo: provider.ResourceInfo{
									ResourceID: "nested-res-123",
									CurrentResourceState: &state.ResourceState{
										ResourceID: "nested-res-123",
										Type:       "aws/sns/topic",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_ = BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// Verify the nested resource was added to the shared map
	s.NotNil(resourcesByName["nestedResource"])
	s.Equal("nestedResource", resourcesByName["nestedResource"].Name)
	s.Equal(ActionUpdate, resourcesByName["nestedResource"].Action)
	s.Equal("nested-res-123", resourcesByName["nestedResource"].ResourceID)
	s.Equal("aws/sns/topic", resourcesByName["nestedResource"].ResourceType)
	s.NotNil(resourcesByName["nestedResource"].ResourceState)
}

func (s *ItemBuilderTestSuite) Test_nested_changed_resource_with_recreate() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	// Use a structure with nested ChildChanges to test recursive resource population
	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"parentChild": {
				ChildChanges: map[string]changes.BlueprintChanges{
					"grandChild": {
						ResourceChanges: map[string]provider.Changes{
							"nestedResource": {
								MustRecreate: true,
							},
						},
					},
				},
			},
		},
	}

	_ = BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.NotNil(resourcesByName["nestedResource"])
	s.Equal(ActionRecreate, resourcesByName["nestedResource"].Action)
}

// Complex integration test

func (s *ItemBuilderTestSuite) Test_BuildItemsFromChangeset_complex_hierarchy() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-unchanged": {
				ResourceID: "res-unchanged",
				Name:       "unchangedResource",
				Type:       "aws/s3/bucket",
			},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"unchangedChild": {InstanceID: "unchanged-child-instance"},
		},
		Links: map[string]*state.LinkState{
			"resX::resY": {LinkID: "link-unchanged"},
		},
	}

	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
		ResourceChanges: map[string]provider.Changes{
			"changedResource": {
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
			},
		},
		RemovedResources: []string{"removedResource"},
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild": {},
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"changedChild": {},
		},
		RemovedChildren: []string{"removedChild"},
	}

	items := BuildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	// Count items by type and action
	resourceItems := 0
	childItems := 0
	linkItems := 0
	for idx := range items {
		switch items[idx].Type {
		case ItemTypeResource:
			resourceItems += 1
		case ItemTypeChild:
			childItems += 1
		case ItemTypeLink:
			linkItems += 1
		}
	}

	// 3 from changes (new, changed, removed) + 1 unchanged from state
	s.Equal(4, resourceItems)
	// 3 from changes (new, changed, removed) + 1 unchanged from state
	s.Equal(4, childItems)
	// 1 unchanged from state
	s.Equal(1, linkItems)
}
