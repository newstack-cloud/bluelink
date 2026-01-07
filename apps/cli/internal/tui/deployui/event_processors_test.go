package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type EventProcessorsTestSuite struct {
	suite.Suite
}

func TestEventProcessorsTestSuite(t *testing.T) {
	suite.Run(t, new(EventProcessorsTestSuite))
}

// Helper to create a minimal EventProcessorState for testing
func (s *EventProcessorsTestSuite) newTestState() *EventProcessorState {
	return NewEventProcessorState("root-instance-id")
}

// shared.JoinPath tests (delegating to shared package)

func (s *EventProcessorsTestSuite) Test_JoinPath_empty_slice_returns_empty_string() {
	result := shared.JoinPath([]string{})
	s.Equal("", result)
}

func (s *EventProcessorsTestSuite) Test_JoinPath_single_element() {
	result := shared.JoinPath([]string{"resource1"})
	s.Equal("resource1", result)
}

func (s *EventProcessorsTestSuite) Test_JoinPath_multiple_elements() {
	result := shared.JoinPath([]string{"parent", "child", "resource"})
	s.Equal("parent/child/resource", result)
}

// BuildResourcePath tests

func (s *EventProcessorsTestSuite) Test_BuildResourcePath_empty_instanceID_returns_name() {
	state := s.newTestState()
	path := BuildResourcePath(state, "", "myResource")
	s.Equal("myResource", path)
}

func (s *EventProcessorsTestSuite) Test_BuildResourcePath_root_instanceID_returns_name() {
	state := s.newTestState()
	path := BuildResourcePath(state, "root-instance-id", "myResource")
	s.Equal("myResource", path)
}

func (s *EventProcessorsTestSuite) Test_BuildResourcePath_nested_instance_builds_path() {
	state := s.newTestState()
	state.InstanceIDToChildName["child-instance-id"] = "childBlueprint"
	state.InstanceIDToParentID["child-instance-id"] = "root-instance-id"

	path := BuildResourcePath(state, "child-instance-id", "nestedResource")
	s.Equal("childBlueprint/nestedResource", path)
}

func (s *EventProcessorsTestSuite) Test_BuildResourcePath_deeply_nested_builds_full_path() {
	state := s.newTestState()
	state.InstanceIDToChildName["child-instance-id"] = "childBlueprint"
	state.InstanceIDToParentID["child-instance-id"] = "root-instance-id"
	state.InstanceIDToChildName["grandchild-instance-id"] = "grandchildBlueprint"
	state.InstanceIDToParentID["grandchild-instance-id"] = "child-instance-id"

	path := BuildResourcePath(state, "grandchild-instance-id", "deepResource")
	s.Equal("childBlueprint/grandchildBlueprint/deepResource", path)
}

// BuildInstancePath tests

func (s *EventProcessorsTestSuite) Test_BuildInstancePath_empty_parent_returns_name() {
	state := s.newTestState()
	path := BuildInstancePath(state, "", "childBlueprint")
	s.Equal("childBlueprint", path)
}

func (s *EventProcessorsTestSuite) Test_BuildInstancePath_root_parent_returns_name() {
	state := s.newTestState()
	path := BuildInstancePath(state, "root-instance-id", "childBlueprint")
	s.Equal("childBlueprint", path)
}

func (s *EventProcessorsTestSuite) Test_BuildInstancePath_nested_parent_builds_path() {
	state := s.newTestState()
	state.InstanceIDToChildName["child-instance-id"] = "parentChild"
	state.InstanceIDToParentID["child-instance-id"] = "root-instance-id"

	path := BuildInstancePath(state, "child-instance-id", "nestedChild")
	s.Equal("parentChild/nestedChild", path)
}

// LookupOrMigrateResource tests

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateResource_returns_existing_by_path() {
	state := s.newTestState()
	existingItem := &ResourceDeployItem{Name: "resource1"}
	state.ResourcesByName["child/resource1"] = existingItem

	result := LookupOrMigrateResource(state, "child/resource1", "resource1")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateResource_migrates_from_name_to_path() {
	state := s.newTestState()
	existingItem := &ResourceDeployItem{Name: "resource1"}
	state.ResourcesByName["resource1"] = existingItem

	result := LookupOrMigrateResource(state, "child/resource1", "resource1")

	s.Same(existingItem, result)
	s.Nil(state.ResourcesByName["resource1"])
	s.Same(existingItem, state.ResourcesByName["child/resource1"])
}

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateResource_returns_nil_when_not_found() {
	state := s.newTestState()

	result := LookupOrMigrateResource(state, "child/resource1", "resource1")
	s.Nil(result)
}

// LookupOrMigrateChild tests

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateChild_returns_existing_by_path() {
	state := s.newTestState()
	existingItem := &ChildDeployItem{Name: "child1"}
	state.ChildrenByName["parent/child1"] = existingItem

	result := LookupOrMigrateChild(state, "parent/child1", "child1")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateChild_migrates_from_name_to_path() {
	state := s.newTestState()
	existingItem := &ChildDeployItem{Name: "child1"}
	state.ChildrenByName["child1"] = existingItem

	result := LookupOrMigrateChild(state, "parent/child1", "child1")

	s.Same(existingItem, result)
	s.Nil(state.ChildrenByName["child1"])
	s.Same(existingItem, state.ChildrenByName["parent/child1"])
}

// LookupOrMigrateLink tests

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateLink_returns_existing_by_path() {
	state := s.newTestState()
	existingItem := &LinkDeployItem{LinkName: "resA::resB"}
	state.LinksByName["child/resA::resB"] = existingItem

	result := LookupOrMigrateLink(state, "child/resA::resB", "resA::resB")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_LookupOrMigrateLink_migrates_from_name_to_path() {
	state := s.newTestState()
	existingItem := &LinkDeployItem{LinkName: "resA::resB"}
	state.LinksByName["resA::resB"] = existingItem

	result := LookupOrMigrateLink(state, "child/resA::resB", "resA::resB")

	s.Same(existingItem, result)
	s.Nil(state.LinksByName["resA::resB"])
	s.Same(existingItem, state.LinksByName["child/resA::resB"])
}

// ProcessResourceUpdate tests

func (s *EventProcessorsTestSuite) Test_ProcessResourceUpdate_creates_new_root_item() {
	state := s.newTestState()
	data := &container.ResourceDeployUpdateMessage{
		ResourceName:    "newResource",
		ResourceID:      "res-123",
		InstanceID:      "",
		Status:          core.ResourceStatusCreating,
		PreciseStatus:   core.PreciseResourceStatusCreating,
		Group:           1,
		UpdateTimestamp: 12345,
	}

	ProcessResourceUpdate(state, data)

	s.Len(state.Items, 1)
	s.Equal(ItemTypeResource, state.Items[0].Type)
	s.Equal("newResource", state.Items[0].Resource.Name)
	s.Equal("res-123", state.Items[0].Resource.ResourceID)
	s.Equal(core.ResourceStatusCreating, state.Items[0].Resource.Status)
}

func (s *EventProcessorsTestSuite) Test_ProcessResourceUpdate_updates_existing_item() {
	state := s.newTestState()
	existingItem := &ResourceDeployItem{
		Name:   "existingResource",
		Status: core.ResourceStatusCreating,
	}
	state.ResourcesByName["existingResource"] = existingItem
	state.Items = []DeployItem{{Type: ItemTypeResource, Resource: existingItem}}

	data := &container.ResourceDeployUpdateMessage{
		ResourceName:    "existingResource",
		ResourceID:      "res-123",
		InstanceID:      "",
		Status:          core.ResourceStatusCreated,
		PreciseStatus:   core.PreciseResourceStatusCreated,
		UpdateTimestamp: 12345,
	}

	ProcessResourceUpdate(state, data)

	s.Len(state.Items, 1)
	s.Equal(core.ResourceStatusCreated, existingItem.Status)
	s.Equal(int64(12345), existingItem.Timestamp)
}

func (s *EventProcessorsTestSuite) Test_ProcessResourceUpdate_does_not_add_nested_to_root_items() {
	state := s.newTestState()
	state.InstanceIDToChildName["child-instance-id"] = "childBlueprint"
	state.InstanceIDToParentID["child-instance-id"] = "root-instance-id"

	data := &container.ResourceDeployUpdateMessage{
		ResourceName:  "nestedResource",
		ResourceID:    "res-456",
		InstanceID:    "child-instance-id",
		Status:        core.ResourceStatusCreating,
		PreciseStatus: core.PreciseResourceStatusCreating,
	}

	ProcessResourceUpdate(state, data)

	s.Len(state.Items, 0)
	s.NotNil(state.ResourcesByName["childBlueprint/nestedResource"])
}

// ProcessChildUpdate tests

func (s *EventProcessorsTestSuite) Test_ProcessChildUpdate_creates_new_direct_child() {
	state := s.newTestState()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "newChild",
		ChildInstanceID:  "child-inst-123",
		ParentInstanceID: "root-instance-id",
		Status:           core.InstanceStatusDeploying,
		Group:            1,
		UpdateTimestamp:  12345,
	}

	ProcessChildUpdate(state, data)

	s.Len(state.Items, 1)
	s.Equal(ItemTypeChild, state.Items[0].Type)
	s.Equal("newChild", state.Items[0].Child.Name)
	s.Equal(core.InstanceStatusDeploying, state.Items[0].Child.Status)
}

func (s *EventProcessorsTestSuite) Test_ProcessChildUpdate_tracks_instance_mapping() {
	state := s.newTestState()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "newChild",
		ChildInstanceID:  "child-inst-123",
		ParentInstanceID: "root-instance-id",
		Status:           core.InstanceStatusDeploying,
	}

	ProcessChildUpdate(state, data)

	s.Equal("newChild", state.InstanceIDToChildName["child-inst-123"])
	s.Equal("root-instance-id", state.InstanceIDToParentID["child-inst-123"])
}

func (s *EventProcessorsTestSuite) Test_ProcessChildUpdate_does_not_add_nested_child_to_root_items() {
	state := s.newTestState()
	state.InstanceIDToChildName["parent-child-id"] = "parentChild"
	state.InstanceIDToParentID["parent-child-id"] = "root-instance-id"

	data := &container.ChildDeployUpdateMessage{
		ChildName:        "nestedChild",
		ChildInstanceID:  "nested-child-id",
		ParentInstanceID: "parent-child-id",
		Status:           core.InstanceStatusDeploying,
	}

	ProcessChildUpdate(state, data)

	s.Len(state.Items, 0)
	s.NotNil(state.ChildrenByName["parentChild/nestedChild"])
}

// ProcessLinkUpdate tests

func (s *EventProcessorsTestSuite) Test_ProcessLinkUpdate_creates_new_root_link() {
	state := s.newTestState()
	data := &container.LinkDeployUpdateMessage{
		LinkName:        "resourceA::resourceB",
		LinkID:          "link-123",
		InstanceID:      "",
		Status:          core.LinkStatusCreating,
		PreciseStatus:   core.PreciseLinkStatusUpdatingResourceA,
		UpdateTimestamp: 12345,
	}

	ProcessLinkUpdate(state, data)

	s.Len(state.Items, 1)
	s.Equal(ItemTypeLink, state.Items[0].Type)
	s.Equal("resourceA::resourceB", state.Items[0].Link.LinkName)
	s.Equal(core.LinkStatusCreating, state.Items[0].Link.Status)
	s.Equal("resourceA", state.Items[0].Link.ResourceAName)
	s.Equal("resourceB", state.Items[0].Link.ResourceBName)
}

func (s *EventProcessorsTestSuite) Test_ProcessLinkUpdate_updates_existing_link() {
	state := s.newTestState()
	existingLink := &LinkDeployItem{
		LinkName: "resourceA::resourceB",
		Status:   core.LinkStatusCreating,
	}
	state.LinksByName["resourceA::resourceB"] = existingLink
	state.Items = []DeployItem{{Type: ItemTypeLink, Link: existingLink}}

	data := &container.LinkDeployUpdateMessage{
		LinkName:        "resourceA::resourceB",
		LinkID:          "link-123",
		InstanceID:      "",
		Status:          core.LinkStatusCreated,
		PreciseStatus:   core.PreciseLinkStatusResourceBUpdated,
		UpdateTimestamp: 12345,
	}

	ProcessLinkUpdate(state, data)

	s.Len(state.Items, 1)
	s.Equal(core.LinkStatusCreated, existingLink.Status)
}

// ProcessInstanceUpdate tests

func (s *EventProcessorsTestSuite) Test_ProcessInstanceUpdate_updates_footer_status() {
	state := s.newTestState()
	data := &container.DeploymentUpdateMessage{
		Status: core.InstanceStatusDeploying,
	}

	ProcessInstanceUpdate(state, data)

	s.Equal(core.InstanceStatusDeploying, state.FooterRenderer.CurrentStatus)
}

// GetChildChanges tests

func (s *EventProcessorsTestSuite) Test_GetChildChanges_returns_nil_when_no_changeset() {
	state := s.newTestState()
	result := GetChildChanges(state, "someChild")
	s.Nil(result)
}

func (s *EventProcessorsTestSuite) Test_GetChildChanges_returns_new_child_changes() {
	state := s.newTestState()
	newResources := map[string]provider.Changes{
		"res1": {},
	}
	state.ChangesetChanges = &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild": {
				NewResources: newResources,
			},
		},
	}

	result := GetChildChanges(state, "newChild")

	s.NotNil(result)
	s.Equal(newResources, result.NewResources)
}

func (s *EventProcessorsTestSuite) Test_GetChildChanges_returns_existing_child_changes() {
	state := s.newTestState()
	childChanges := changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"res1": {},
		},
	}
	state.ChangesetChanges = &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"existingChild": childChanges,
		},
	}

	result := GetChildChanges(state, "existingChild")

	s.NotNil(result)
	s.Equal(childChanges.ResourceChanges, result.ResourceChanges)
}

// TrackChildInstanceMapping tests

func (s *EventProcessorsTestSuite) Test_TrackChildInstanceMapping_stores_mapping() {
	state := s.newTestState()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "myChild",
		ChildInstanceID:  "child-123",
		ParentInstanceID: "parent-456",
	}

	TrackChildInstanceMapping(state, data)

	s.Equal("myChild", state.InstanceIDToChildName["child-123"])
	s.Equal("parent-456", state.InstanceIDToParentID["child-123"])
}

func (s *EventProcessorsTestSuite) Test_TrackChildInstanceMapping_ignores_empty_ids() {
	state := s.newTestState()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "myChild",
		ChildInstanceID:  "",
		ParentInstanceID: "parent-456",
	}

	TrackChildInstanceMapping(state, data)

	s.Empty(state.InstanceIDToChildName)
	s.Empty(state.InstanceIDToParentID)
}
