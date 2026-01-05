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

// Helper to create a minimal DeployModel for testing
func (s *EventProcessorsTestSuite) newTestModel() *DeployModel {
	return &DeployModel{
		instanceID:              "root-instance-id",
		resourcesByName:         make(map[string]*ResourceDeployItem),
		childrenByName:          make(map[string]*ChildDeployItem),
		linksByName:             make(map[string]*LinkDeployItem),
		instanceIDToChildName:   make(map[string]string),
		instanceIDToParentID:    make(map[string]string),
		childNameToInstancePath: make(map[string]string),
		footerRenderer:          &DeployFooterRenderer{},
	}
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

// buildResourcePath tests

func (s *EventProcessorsTestSuite) Test_buildResourcePath_empty_instanceID_returns_name() {
	m := s.newTestModel()
	path := m.buildResourcePath("", "myResource")
	s.Equal("myResource", path)
}

func (s *EventProcessorsTestSuite) Test_buildResourcePath_root_instanceID_returns_name() {
	m := s.newTestModel()
	path := m.buildResourcePath("root-instance-id", "myResource")
	s.Equal("myResource", path)
}

func (s *EventProcessorsTestSuite) Test_buildResourcePath_nested_instance_builds_path() {
	m := s.newTestModel()
	m.instanceIDToChildName["child-instance-id"] = "childBlueprint"
	m.instanceIDToParentID["child-instance-id"] = "root-instance-id"

	path := m.buildResourcePath("child-instance-id", "nestedResource")
	s.Equal("childBlueprint/nestedResource", path)
}

func (s *EventProcessorsTestSuite) Test_buildResourcePath_deeply_nested_builds_full_path() {
	m := s.newTestModel()
	m.instanceIDToChildName["child-instance-id"] = "childBlueprint"
	m.instanceIDToParentID["child-instance-id"] = "root-instance-id"
	m.instanceIDToChildName["grandchild-instance-id"] = "grandchildBlueprint"
	m.instanceIDToParentID["grandchild-instance-id"] = "child-instance-id"

	path := m.buildResourcePath("grandchild-instance-id", "deepResource")
	s.Equal("childBlueprint/grandchildBlueprint/deepResource", path)
}

// buildInstancePath tests

func (s *EventProcessorsTestSuite) Test_buildInstancePath_empty_parent_returns_name() {
	m := s.newTestModel()
	path := m.buildInstancePath("", "childBlueprint")
	s.Equal("childBlueprint", path)
}

func (s *EventProcessorsTestSuite) Test_buildInstancePath_root_parent_returns_name() {
	m := s.newTestModel()
	path := m.buildInstancePath("root-instance-id", "childBlueprint")
	s.Equal("childBlueprint", path)
}

func (s *EventProcessorsTestSuite) Test_buildInstancePath_nested_parent_builds_path() {
	m := s.newTestModel()
	m.instanceIDToChildName["child-instance-id"] = "parentChild"
	m.instanceIDToParentID["child-instance-id"] = "root-instance-id"

	path := m.buildInstancePath("child-instance-id", "nestedChild")
	s.Equal("parentChild/nestedChild", path)
}

// lookupOrMigrateResource tests

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateResource_returns_existing_by_path() {
	m := s.newTestModel()
	existingItem := &ResourceDeployItem{Name: "resource1"}
	m.resourcesByName["child/resource1"] = existingItem

	result := m.lookupOrMigrateResource("child/resource1", "resource1")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateResource_migrates_from_name_to_path() {
	m := s.newTestModel()
	existingItem := &ResourceDeployItem{Name: "resource1"}
	m.resourcesByName["resource1"] = existingItem

	result := m.lookupOrMigrateResource("child/resource1", "resource1")

	s.Same(existingItem, result)
	s.Nil(m.resourcesByName["resource1"])
	s.Same(existingItem, m.resourcesByName["child/resource1"])
}

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateResource_returns_nil_when_not_found() {
	m := s.newTestModel()

	result := m.lookupOrMigrateResource("child/resource1", "resource1")
	s.Nil(result)
}

// lookupOrMigrateChild tests

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateChild_returns_existing_by_path() {
	m := s.newTestModel()
	existingItem := &ChildDeployItem{Name: "child1"}
	m.childrenByName["parent/child1"] = existingItem

	result := m.lookupOrMigrateChild("parent/child1", "child1")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateChild_migrates_from_name_to_path() {
	m := s.newTestModel()
	existingItem := &ChildDeployItem{Name: "child1"}
	m.childrenByName["child1"] = existingItem

	result := m.lookupOrMigrateChild("parent/child1", "child1")

	s.Same(existingItem, result)
	s.Nil(m.childrenByName["child1"])
	s.Same(existingItem, m.childrenByName["parent/child1"])
}

// lookupOrMigrateLink tests

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateLink_returns_existing_by_path() {
	m := s.newTestModel()
	existingItem := &LinkDeployItem{LinkName: "resA::resB"}
	m.linksByName["child/resA::resB"] = existingItem

	result := m.lookupOrMigrateLink("child/resA::resB", "resA::resB")
	s.Same(existingItem, result)
}

func (s *EventProcessorsTestSuite) Test_lookupOrMigrateLink_migrates_from_name_to_path() {
	m := s.newTestModel()
	existingItem := &LinkDeployItem{LinkName: "resA::resB"}
	m.linksByName["resA::resB"] = existingItem

	result := m.lookupOrMigrateLink("child/resA::resB", "resA::resB")

	s.Same(existingItem, result)
	s.Nil(m.linksByName["resA::resB"])
	s.Same(existingItem, m.linksByName["child/resA::resB"])
}

// processResourceUpdate tests

func (s *EventProcessorsTestSuite) Test_processResourceUpdate_creates_new_root_item() {
	m := s.newTestModel()
	data := &container.ResourceDeployUpdateMessage{
		ResourceName:    "newResource",
		ResourceID:      "res-123",
		InstanceID:      "",
		Status:          core.ResourceStatusCreating,
		PreciseStatus:   core.PreciseResourceStatusCreating,
		Group:           1,
		UpdateTimestamp: 12345,
	}

	m.processResourceUpdate(data)

	s.Len(m.items, 1)
	s.Equal(ItemTypeResource, m.items[0].Type)
	s.Equal("newResource", m.items[0].Resource.Name)
	s.Equal("res-123", m.items[0].Resource.ResourceID)
	s.Equal(core.ResourceStatusCreating, m.items[0].Resource.Status)
}

func (s *EventProcessorsTestSuite) Test_processResourceUpdate_updates_existing_item() {
	m := s.newTestModel()
	existingItem := &ResourceDeployItem{
		Name:   "existingResource",
		Status: core.ResourceStatusCreating,
	}
	m.resourcesByName["existingResource"] = existingItem
	m.items = []DeployItem{{Type: ItemTypeResource, Resource: existingItem}}

	data := &container.ResourceDeployUpdateMessage{
		ResourceName:    "existingResource",
		ResourceID:      "res-123",
		InstanceID:      "",
		Status:          core.ResourceStatusCreated,
		PreciseStatus:   core.PreciseResourceStatusCreated,
		UpdateTimestamp: 12345,
	}

	m.processResourceUpdate(data)

	s.Len(m.items, 1)
	s.Equal(core.ResourceStatusCreated, existingItem.Status)
	s.Equal(int64(12345), existingItem.Timestamp)
}

func (s *EventProcessorsTestSuite) Test_processResourceUpdate_does_not_add_nested_to_root_items() {
	m := s.newTestModel()
	m.instanceIDToChildName["child-instance-id"] = "childBlueprint"
	m.instanceIDToParentID["child-instance-id"] = "root-instance-id"

	data := &container.ResourceDeployUpdateMessage{
		ResourceName:  "nestedResource",
		ResourceID:    "res-456",
		InstanceID:    "child-instance-id",
		Status:        core.ResourceStatusCreating,
		PreciseStatus: core.PreciseResourceStatusCreating,
	}

	m.processResourceUpdate(data)

	s.Len(m.items, 0)
	s.NotNil(m.resourcesByName["childBlueprint/nestedResource"])
}

// processChildUpdate tests

func (s *EventProcessorsTestSuite) Test_processChildUpdate_creates_new_direct_child() {
	m := s.newTestModel()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "newChild",
		ChildInstanceID:  "child-inst-123",
		ParentInstanceID: "root-instance-id",
		Status:           core.InstanceStatusDeploying,
		Group:            1,
		UpdateTimestamp:  12345,
	}

	m.processChildUpdate(data)

	s.Len(m.items, 1)
	s.Equal(ItemTypeChild, m.items[0].Type)
	s.Equal("newChild", m.items[0].Child.Name)
	s.Equal(core.InstanceStatusDeploying, m.items[0].Child.Status)
}

func (s *EventProcessorsTestSuite) Test_processChildUpdate_tracks_instance_mapping() {
	m := s.newTestModel()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "newChild",
		ChildInstanceID:  "child-inst-123",
		ParentInstanceID: "root-instance-id",
		Status:           core.InstanceStatusDeploying,
	}

	m.processChildUpdate(data)

	s.Equal("newChild", m.instanceIDToChildName["child-inst-123"])
	s.Equal("root-instance-id", m.instanceIDToParentID["child-inst-123"])
}

func (s *EventProcessorsTestSuite) Test_processChildUpdate_does_not_add_nested_child_to_root_items() {
	m := s.newTestModel()
	m.instanceIDToChildName["parent-child-id"] = "parentChild"
	m.instanceIDToParentID["parent-child-id"] = "root-instance-id"

	data := &container.ChildDeployUpdateMessage{
		ChildName:        "nestedChild",
		ChildInstanceID:  "nested-child-id",
		ParentInstanceID: "parent-child-id",
		Status:           core.InstanceStatusDeploying,
	}

	m.processChildUpdate(data)

	s.Len(m.items, 0)
	s.NotNil(m.childrenByName["parentChild/nestedChild"])
}

// processLinkUpdate tests

func (s *EventProcessorsTestSuite) Test_processLinkUpdate_creates_new_root_link() {
	m := s.newTestModel()
	data := &container.LinkDeployUpdateMessage{
		LinkName:        "resourceA::resourceB",
		LinkID:          "link-123",
		InstanceID:      "",
		Status:          core.LinkStatusCreating,
		PreciseStatus:   core.PreciseLinkStatusUpdatingResourceA,
		UpdateTimestamp: 12345,
	}

	m.processLinkUpdate(data)

	s.Len(m.items, 1)
	s.Equal(ItemTypeLink, m.items[0].Type)
	s.Equal("resourceA::resourceB", m.items[0].Link.LinkName)
	s.Equal(core.LinkStatusCreating, m.items[0].Link.Status)
	s.Equal("resourceA", m.items[0].Link.ResourceAName)
	s.Equal("resourceB", m.items[0].Link.ResourceBName)
}

func (s *EventProcessorsTestSuite) Test_processLinkUpdate_updates_existing_link() {
	m := s.newTestModel()
	existingLink := &LinkDeployItem{
		LinkName: "resourceA::resourceB",
		Status:   core.LinkStatusCreating,
	}
	m.linksByName["resourceA::resourceB"] = existingLink
	m.items = []DeployItem{{Type: ItemTypeLink, Link: existingLink}}

	data := &container.LinkDeployUpdateMessage{
		LinkName:        "resourceA::resourceB",
		LinkID:          "link-123",
		InstanceID:      "",
		Status:          core.LinkStatusCreated,
		PreciseStatus:   core.PreciseLinkStatusResourceBUpdated,
		UpdateTimestamp: 12345,
	}

	m.processLinkUpdate(data)

	s.Len(m.items, 1)
	s.Equal(core.LinkStatusCreated, existingLink.Status)
}

// processInstanceUpdate tests

func (s *EventProcessorsTestSuite) Test_processInstanceUpdate_updates_footer_status() {
	m := s.newTestModel()
	data := &container.DeploymentUpdateMessage{
		Status: core.InstanceStatusDeploying,
	}

	m.processInstanceUpdate(data)

	s.Equal(core.InstanceStatusDeploying, m.footerRenderer.CurrentStatus)
}

// processPreRollbackState tests

func (s *EventProcessorsTestSuite) Test_processPreRollbackState_stores_data() {
	m := s.newTestModel()
	data := &container.PreRollbackStateMessage{
		InstanceID:   "test-instance",
		InstanceName: "test-name",
		Status:       core.InstanceStatusDeployFailed,
	}

	m.processPreRollbackState(data)

	s.Same(data, m.preRollbackState)
	s.True(m.footerRenderer.HasPreRollbackState)
}

// getChildChanges tests

func (s *EventProcessorsTestSuite) Test_getChildChanges_returns_nil_when_no_changeset() {
	m := s.newTestModel()
	result := m.getChildChanges("someChild")
	s.Nil(result)
}

func (s *EventProcessorsTestSuite) Test_getChildChanges_returns_new_child_changes() {
	m := s.newTestModel()
	newResources := map[string]provider.Changes{
		"res1": {},
	}
	m.changesetChanges = &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild": {
				NewResources: newResources,
			},
		},
	}

	result := m.getChildChanges("newChild")

	s.NotNil(result)
	s.Equal(newResources, result.NewResources)
}

func (s *EventProcessorsTestSuite) Test_getChildChanges_returns_existing_child_changes() {
	m := s.newTestModel()
	childChanges := changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"res1": {},
		},
	}
	m.changesetChanges = &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"existingChild": childChanges,
		},
	}

	result := m.getChildChanges("existingChild")

	s.NotNil(result)
	s.Equal(childChanges.ResourceChanges, result.ResourceChanges)
}

// trackChildInstanceMapping tests

func (s *EventProcessorsTestSuite) Test_trackChildInstanceMapping_stores_mapping() {
	m := s.newTestModel()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "myChild",
		ChildInstanceID:  "child-123",
		ParentInstanceID: "parent-456",
	}

	m.trackChildInstanceMapping(data)

	s.Equal("myChild", m.instanceIDToChildName["child-123"])
	s.Equal("parent-456", m.instanceIDToParentID["child-123"])
}

func (s *EventProcessorsTestSuite) Test_trackChildInstanceMapping_ignores_empty_ids() {
	m := s.newTestModel()
	data := &container.ChildDeployUpdateMessage{
		ChildName:        "myChild",
		ChildInstanceID:  "",
		ParentInstanceID: "parent-456",
	}

	m.trackChildInstanceMapping(data)

	s.Empty(m.instanceIDToChildName)
	s.Empty(m.instanceIDToParentID)
}
