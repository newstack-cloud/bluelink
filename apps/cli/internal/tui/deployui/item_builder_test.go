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

// buildItemsFromChangeset tests

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_returns_empty_for_nil_changeset() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	items := buildItemsFromChangeset(nil, resourcesByName, childrenByName, linksByName, nil)

	s.Empty(items)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_returns_empty_for_empty_changeset() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	items := buildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, nil)

	s.Empty(items)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_new_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
	}

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("newResource", items[0].Resource.Name)
	s.Equal(ActionCreate, items[0].Resource.Action)
	// Should be added to the shared map
	s.NotNil(resourcesByName["newResource"])
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_changed_resources() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("changedResource", items[0].Resource.Name)
	s.Equal(ActionUpdate, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_removed_resources() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"removedResource"},
	}

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("removedResource", items[0].Resource.Name)
	s.Equal(ActionDelete, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_removed_resources_with_state() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ActionDelete, items[0].Resource.Action)
	s.NotNil(items[0].Resource.ResourceState)
	s.Equal("res-123", items[0].Resource.ResourceState.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_new_children() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("newChild", items[0].Child.Name)
	s.Equal(ActionCreate, items[0].Child.Action)
	s.NotNil(items[0].Changes)
	// Should be added to the shared map
	s.NotNil(childrenByName["newChild"])
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_changed_children() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("changedChild", items[0].Child.Name)
	s.Equal(ActionUpdate, items[0].Child.Action)
	s.NotNil(items[0].InstanceState)
	s.Equal("child-instance-123", items[0].InstanceState.InstanceID)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_recreate_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RecreateChildren: []string{"recreateChild"},
	}

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("recreateChild", items[0].Child.Name)
	s.Equal(ActionRecreate, items[0].Child.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_removed_children() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedChildren: []string{"removedChild"},
	}

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("removedChild", items[0].Child.Name)
	s.Equal(ActionDelete, items[0].Child.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_links_from_new_resources() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

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

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_links_from_changed_resources() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

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

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_removed_links() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	bpChanges := &changes.BlueprintChanges{
		RemovedLinks: []string{"resX::resY"},
	}

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	s.Len(items, 1)
	s.Equal(ItemTypeLink, items[0].Type)
	s.Equal("resX::resY", items[0].Link.LinkName)
	s.Equal(ActionDelete, items[0].Link.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_skips_duplicate_removed_links() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// Should only have resource and one link (not duplicated)
	linkCount := 0
	for idx := range items {
		if items[idx].Type == ItemTypeLink {
			linkCount++
		}
	}
	s.Equal(1, linkCount)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_no_change_resources_from_state() {
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

	items := buildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeResource, items[0].Type)
	s.Equal("unchangedResource", items[0].Resource.Name)
	s.Equal(ActionNoChange, items[0].Resource.Action)
	s.Equal("res-123", items[0].Resource.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_no_change_children_from_state() {
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"unchangedChild": {InstanceID: "child-instance-456"},
		},
	}

	items := buildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeChild, items[0].Type)
	s.Equal("unchangedChild", items[0].Child.Name)
	s.Equal(ActionNoChange, items[0].Child.Action)
	s.NotNil(items[0].InstanceState)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_adds_no_change_links_from_state() {
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

	items := buildItemsFromChangeset(&changes.BlueprintChanges{}, resourcesByName, childrenByName, linksByName, instanceState)

	s.Len(items, 1)
	s.Equal(ItemTypeLink, items[0].Type)
	s.Equal("resA::resB", items[0].Link.LinkName)
	s.Equal(ActionNoChange, items[0].Link.Action)
	s.Equal("link-789", items[0].Link.LinkID)
	s.Equal(core.LinkStatusCreated, items[0].Link.Status)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_does_not_duplicate_changed_items_from_state() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	// Should only have one resource (from changes, not duplicated from state)
	s.Len(items, 1)
	s.Equal("changedResource", items[0].Resource.Name)
	s.Equal(ActionUpdate, items[0].Resource.Action)
}

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_populates_nested_items_in_shared_maps() {
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

	_ = buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, nil)

	// parentChild should be added via appendNewChildren
	s.NotNil(childrenByName["parentChild"])
	// grandChild, deepNestedResource, greatGrandChild, and deepestResource should be added
	// via populateNestedItems -> populateNestedNewChildren recursively
	s.NotNil(childrenByName["grandChild"])
	s.NotNil(resourcesByName["deepNestedResource"])
	s.NotNil(childrenByName["greatGrandChild"])
	s.NotNil(resourcesByName["deepestResource"])
}

// determineResourceAction tests

func (s *ItemBuilderTestSuite) Test_determineResourceAction_returns_recreate_for_must_recreate() {
	rc := &provider.Changes{MustRecreate: true}
	s.Equal(ActionRecreate, determineResourceAction(rc))
}

func (s *ItemBuilderTestSuite) Test_determineResourceAction_returns_no_change_for_no_field_changes() {
	rc := &provider.Changes{}
	s.Equal(ActionNoChange, determineResourceAction(rc))
}

func (s *ItemBuilderTestSuite) Test_determineResourceAction_returns_update_for_field_changes() {
	rc := &provider.Changes{
		ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
	}
	s.Equal(ActionUpdate, determineResourceAction(rc))
}

// extractResourceInfo tests

func (s *ItemBuilderTestSuite) Test_extractResourceInfo_extracts_resource_id() {
	rc := &provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID: "res-info-123",
		},
	}
	resourceID, resourceType := extractResourceInfo(rc)
	s.Equal("res-info-123", resourceID)
	s.Equal("", resourceType)
}

func (s *ItemBuilderTestSuite) Test_extractResourceInfo_extracts_resource_type_from_state() {
	rc := &provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID: "res-info-456",
			CurrentResourceState: &state.ResourceState{
				Type: "aws/lambda/function",
			},
		},
	}
	resourceID, resourceType := extractResourceInfo(rc)
	s.Equal("res-info-456", resourceID)
	s.Equal("aws/lambda/function", resourceType)
}

// findResourceState tests

func (s *ItemBuilderTestSuite) Test_findResourceState_returns_nil_for_nil_instance_state() {
	result := findResourceState(nil, "resource")
	s.Nil(result)
}

func (s *ItemBuilderTestSuite) Test_findResourceState_returns_nil_for_missing_resource_id() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{},
		Resources:   map[string]*state.ResourceState{},
	}
	result := findResourceState(instanceState, "unknownResource")
	s.Nil(result)
}

func (s *ItemBuilderTestSuite) Test_findResourceState_finds_resource_by_name() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"myResource": "res-lookup-789",
		},
		Resources: map[string]*state.ResourceState{
			"res-lookup-789": {
				ResourceID: "res-lookup-789",
				Name:       "myResource",
			},
		},
	}
	result := findResourceState(instanceState, "myResource")
	s.NotNil(result)
	s.Equal("res-lookup-789", result.ResourceID)
}

// buildChangedResourceItem tests

func (s *ItemBuilderTestSuite) Test_buildChangedResourceItem_sets_action_from_changes() {
	rc := &provider.Changes{
		ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
	}
	item := buildChangedResourceItem("testResource", rc, nil)
	s.Equal("testResource", item.Name)
	s.Equal(ActionUpdate, item.Action)
	s.Equal(rc, item.Changes)
}

func (s *ItemBuilderTestSuite) Test_buildChangedResourceItem_includes_resource_state() {
	rc := &provider.Changes{
		ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
	}
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
	item := buildChangedResourceItem("testResource", rc, instanceState)
	s.NotNil(item.ResourceState)
	s.Equal("res-state-123", item.ResourceState.ResourceID)
}

func (s *ItemBuilderTestSuite) Test_buildChangedResourceItem_extracts_resource_info_from_changes() {
	rc := &provider.Changes{
		ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID: "res-from-changes",
			CurrentResourceState: &state.ResourceState{
				Type: "aws/dynamodb/table",
			},
		},
	}
	item := buildChangedResourceItem("testResource", rc, nil)
	s.Equal("res-from-changes", item.ResourceID)
	s.Equal("aws/dynamodb/table", item.ResourceType)
}

// buildNestedChangedResourceItem tests

func (s *ItemBuilderTestSuite) Test_buildNestedChangedResourceItem_creates_item() {
	rc := &provider.Changes{
		ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID: "nested-res-123",
			CurrentResourceState: &state.ResourceState{
				ResourceID: "nested-res-123",
				Type:       "aws/sns/topic",
			},
		},
	}
	item := buildNestedChangedResourceItem("nestedResource", rc)
	s.Equal("nestedResource", item.Name)
	s.Equal(ActionUpdate, item.Action)
	s.Equal("nested-res-123", item.ResourceID)
	s.Equal("aws/sns/topic", item.ResourceType)
	s.NotNil(item.ResourceState)
}

func (s *ItemBuilderTestSuite) Test_buildNestedChangedResourceItem_with_recreate() {
	rc := &provider.Changes{
		MustRecreate: true,
	}
	item := buildNestedChangedResourceItem("nestedResource", rc)
	s.Equal(ActionRecreate, item.Action)
}

// Complex integration test

func (s *ItemBuilderTestSuite) Test_buildItemsFromChangeset_complex_hierarchy() {
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

	items := buildItemsFromChangeset(bpChanges, resourcesByName, childrenByName, linksByName, instanceState)

	// Count items by type and action
	resourceItems := 0
	childItems := 0
	linkItems := 0
	for idx := range items {
		switch items[idx].Type {
		case ItemTypeResource:
			resourceItems++
		case ItemTypeChild:
			childItems++
		case ItemTypeLink:
			linkItems++
		}
	}

	// 3 from changes (new, changed, removed) + 1 unchanged from state
	s.Equal(4, resourceItems)
	// 3 from changes (new, changed, removed) + 1 unchanged from state
	s.Equal(4, childItems)
	// 1 unchanged from state
	s.Equal(1, linkItems)
}
