package inspectui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type InspectItemsTestSuite struct {
	suite.Suite
}

func TestInspectItemsTestSuite(t *testing.T) {
	suite.Run(t, new(InspectItemsTestSuite))
}

// --- buildItemsFromInstanceState tests ---

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_returns_nil_for_nil_state() {
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(nil, resourcesByName, childrenByName, linksByName)
	s.Nil(items)
}

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_returns_empty_for_empty_state() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(instanceState, resourcesByName, childrenByName, linksByName)
	s.Empty(items)
}

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_creates_resource_items() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(instanceState, resourcesByName, childrenByName, linksByName)

	s.Len(items, 1)
	s.Equal(deployui.ItemTypeResource, items[0].Type)
	s.NotNil(items[0].Resource)
	s.Equal("myBucket", items[0].Resource.Name)
	s.Equal("res-123", items[0].Resource.ResourceID)
	s.Equal("aws/s3/bucket", items[0].Resource.ResourceType)
	s.Equal(shared.ActionInspect, items[0].Resource.Action)
	s.Equal(core.ResourceStatusCreated, items[0].Resource.Status)

	// Verify the resource was added to resourcesByName
	s.Contains(resourcesByName, "myBucket")
	s.Equal("res-123", resourcesByName["myBucket"].ResourceID)
}

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_creates_child_items() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID:   "child-123",
				InstanceName: "childBlueprint",
				Status:       core.InstanceStatusDeployed,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(instanceState, resourcesByName, childrenByName, linksByName)

	s.Len(items, 1)
	s.Equal(deployui.ItemTypeChild, items[0].Type)
	s.NotNil(items[0].Child)
	s.Equal("childBlueprint", items[0].Child.Name)
	s.Equal("child-123", items[0].Child.ChildInstanceID)
	s.Equal(shared.ActionInspect, items[0].Child.Action)
	s.Equal(core.InstanceStatusDeployed, items[0].Child.Status)

	// Verify the child was added to childrenByName
	s.Contains(childrenByName, "childBlueprint")
}

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_creates_link_items() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Links: map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID: "link-123",
				Status: core.LinkStatusCreated,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(instanceState, resourcesByName, childrenByName, linksByName)

	s.Len(items, 1)
	s.Equal(deployui.ItemTypeLink, items[0].Type)
	s.NotNil(items[0].Link)
	s.Equal("resourceA::resourceB", items[0].Link.LinkName)
	s.Equal("link-123", items[0].Link.LinkID)
	s.Equal("resourceA", items[0].Link.ResourceAName)
	s.Equal("resourceB", items[0].Link.ResourceBName)
	s.Equal(shared.ActionInspect, items[0].Link.Action)
	s.Equal(core.LinkStatusCreated, items[0].Link.Status)

	// Verify the link was added to linksByName
	s.Contains(linksByName, "resourceA::resourceB")
}

func (s *InspectItemsTestSuite) Test_buildItemsFromInstanceState_creates_all_item_types() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "myResource",
				Type:       "aws/lambda/function",
				Status:     core.ResourceStatusCreated,
			},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"myChild": {
				InstanceID:   "child-456",
				InstanceName: "myChild",
				Status:       core.InstanceStatusDeployed,
			},
		},
		Links: map[string]*state.LinkState{
			"resA::resB": {
				LinkID: "link-789",
				Status: core.LinkStatusCreated,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	items := buildItemsFromInstanceState(instanceState, resourcesByName, childrenByName, linksByName)

	s.Len(items, 3)

	// Verify all maps are populated
	s.Len(resourcesByName, 1)
	s.Len(childrenByName, 1)
	s.Len(linksByName, 1)
}

// --- extractResourceAFromLinkName tests ---

func (s *InspectItemsTestSuite) Test_extractResourceAFromLinkName_extracts_first_part() {
	result := extractResourceAFromLinkName("resourceA::resourceB")
	s.Equal("resourceA", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceAFromLinkName_returns_full_name_if_no_separator() {
	result := extractResourceAFromLinkName("singleResource")
	s.Equal("singleResource", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceAFromLinkName_handles_empty_string() {
	result := extractResourceAFromLinkName("")
	s.Equal("", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceAFromLinkName_handles_multiple_separators() {
	result := extractResourceAFromLinkName("a::b::c")
	s.Equal("a", result)
}

// --- extractResourceBFromLinkName tests ---

func (s *InspectItemsTestSuite) Test_extractResourceBFromLinkName_extracts_second_part() {
	result := extractResourceBFromLinkName("resourceA::resourceB")
	s.Equal("resourceB", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceBFromLinkName_returns_empty_if_no_separator() {
	result := extractResourceBFromLinkName("singleResource")
	s.Equal("", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceBFromLinkName_handles_empty_string() {
	result := extractResourceBFromLinkName("")
	s.Equal("", result)
}

func (s *InspectItemsTestSuite) Test_extractResourceBFromLinkName_handles_multiple_separators() {
	result := extractResourceBFromLinkName("a::b::c")
	s.Equal("b::c", result)
}

// --- appendResourcesFromState tests ---

func (s *InspectItemsTestSuite) Test_appendResourcesFromState_appends_to_existing_items() {
	existingItem := deployui.DeployItem{
		Type: deployui.ItemTypeChild,
		Child: &deployui.ChildDeployItem{
			Name: "existingChild",
		},
	}
	items := []deployui.DeployItem{existingItem}

	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "newResource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)

	result := appendResourcesFromState(items, instanceState, resourcesByName)

	s.Len(result, 2)
	s.Equal(deployui.ItemTypeChild, result[0].Type)
	s.Equal(deployui.ItemTypeResource, result[1].Type)
}

func (s *InspectItemsTestSuite) Test_appendResourcesFromState_preserves_resource_state() {
	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "testResource",
				Type:       "aws/dynamodb/table",
				Status:     core.ResourceStatusUpdated,
			},
		},
	}
	resourcesByName := make(map[string]*deployui.ResourceDeployItem)

	result := appendResourcesFromState(nil, instanceState, resourcesByName)

	s.Len(result, 1)
	s.NotNil(result[0].Resource.ResourceState)
	s.Equal("res-1", result[0].Resource.ResourceState.ResourceID)
}

// --- appendLinksFromState tests ---

func (s *InspectItemsTestSuite) Test_appendLinksFromState_appends_to_existing_items() {
	existingItem := deployui.DeployItem{
		Type: deployui.ItemTypeResource,
		Resource: &deployui.ResourceDeployItem{
			Name: "existingResource",
		},
	}
	items := []deployui.DeployItem{existingItem}

	instanceState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"a::b": {
				LinkID: "link-1",
				Status: core.LinkStatusCreated,
			},
		},
	}
	linksByName := make(map[string]*deployui.LinkDeployItem)

	result := appendLinksFromState(items, instanceState, linksByName)

	s.Len(result, 2)
	s.Equal(deployui.ItemTypeResource, result[0].Type)
	s.Equal(deployui.ItemTypeLink, result[1].Type)
}

// --- ToSplitPaneItems tests ---

func (s *InspectItemsTestSuite) Test_ToSplitPaneItems_converts_items() {
	items := []deployui.DeployItem{
		{
			Type: deployui.ItemTypeResource,
			Resource: &deployui.ResourceDeployItem{
				Name: "res1",
			},
		},
		{
			Type: deployui.ItemTypeChild,
			Child: &deployui.ChildDeployItem{
				Name: "child1",
			},
		},
	}

	result := ToSplitPaneItems(items)

	s.Len(result, 2)
}

func (s *InspectItemsTestSuite) Test_ToSplitPaneItems_handles_empty_input() {
	result := ToSplitPaneItems([]deployui.DeployItem{})
	s.Empty(result)
	s.NotNil(result)
}
