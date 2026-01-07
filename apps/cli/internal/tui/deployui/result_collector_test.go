package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ResultCollectorTestSuite struct {
	suite.Suite
}

func TestResultCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(ResultCollectorTestSuite))
}

// buildMapKey tests

func (s *ResultCollectorTestSuite) Test_buildMapKey_empty_prefix_returns_name() {
	result := buildMapKey("", "resourceName")
	s.Equal("resourceName", result)
}

func (s *ResultCollectorTestSuite) Test_buildMapKey_with_prefix_joins_with_slash() {
	result := buildMapKey("parent", "resourceName")
	s.Equal("parent/resourceName", result)
}

func (s *ResultCollectorTestSuite) Test_buildMapKey_nested_prefix() {
	result := buildMapKey("parent/child", "resourceName")
	s.Equal("parent/child/resourceName", result)
}

// buildElementPath tests

func (s *ResultCollectorTestSuite) Test_buildElementPath_empty_parent_returns_segment() {
	result := buildElementPath("", "resources", "myResource")
	s.Equal("resources.myResource", result)
}

func (s *ResultCollectorTestSuite) Test_buildElementPath_with_parent_joins_with_colons() {
	result := buildElementPath("children.parent", "resources", "myResource")
	s.Equal("children.parent::resources.myResource", result)
}

func (s *ResultCollectorTestSuite) Test_buildElementPath_nested_path() {
	result := buildElementPath("children.a::children.b", "resources", "myResource")
	s.Equal("children.a::children.b::resources.myResource", result)
}

// lookupResource tests

func (s *ResultCollectorTestSuite) Test_lookupResource_finds_by_path_key() {
	m := map[string]*ResourceDeployItem{
		"parent/resource1": {Name: "resource1"},
	}
	result := lookupResource(m, "parent/resource1", "resource1")
	s.NotNil(result)
	s.Equal("resource1", result.Name)
}

func (s *ResultCollectorTestSuite) Test_lookupResource_falls_back_to_name() {
	m := map[string]*ResourceDeployItem{
		"resource1": {Name: "resource1"},
	}
	result := lookupResource(m, "parent/resource1", "resource1")
	s.NotNil(result)
	s.Equal("resource1", result.Name)
}

func (s *ResultCollectorTestSuite) Test_lookupResource_returns_nil_when_not_found() {
	m := map[string]*ResourceDeployItem{}
	result := lookupResource(m, "parent/resource1", "resource1")
	s.Nil(result)
}

// lookupChild tests

func (s *ResultCollectorTestSuite) Test_lookupChild_finds_by_path_key() {
	m := map[string]*ChildDeployItem{
		"parent/child1": {Name: "child1"},
	}
	result := lookupChild(m, "parent/child1", "child1")
	s.NotNil(result)
	s.Equal("child1", result.Name)
}

func (s *ResultCollectorTestSuite) Test_lookupChild_falls_back_to_name() {
	m := map[string]*ChildDeployItem{
		"child1": {Name: "child1"},
	}
	result := lookupChild(m, "parent/child1", "child1")
	s.NotNil(result)
	s.Equal("child1", result.Name)
}

// ExtractResourceAFromLinkName tests

func (s *ResultCollectorTestSuite) Test_ExtractResourceAFromLinkName_extracts_first_part() {
	result := ExtractResourceAFromLinkName("resourceA::resourceB")
	s.Equal("resourceA", result)
}

func (s *ResultCollectorTestSuite) Test_ExtractResourceAFromLinkName_handles_no_separator() {
	result := ExtractResourceAFromLinkName("singleName")
	s.Equal("singleName", result)
}

func (s *ResultCollectorTestSuite) Test_ExtractResourceAFromLinkName_handles_empty_string() {
	result := ExtractResourceAFromLinkName("")
	s.Equal("", result)
}

// ExtractResourceBFromLinkName tests

func (s *ResultCollectorTestSuite) Test_ExtractResourceBFromLinkName_extracts_second_part() {
	result := ExtractResourceBFromLinkName("resourceA::resourceB")
	s.Equal("resourceB", result)
}

func (s *ResultCollectorTestSuite) Test_ExtractResourceBFromLinkName_handles_no_separator() {
	result := ExtractResourceBFromLinkName("singleName")
	s.Equal("", result)
}

// collectResourceResult tests

func (s *ResultCollectorTestSuite) Test_collectResourceResult_adds_failed_resource() {
	c := &resultCollector{}

	item := &ResourceDeployItem{
		Name:           "failedResource",
		Status:         core.ResourceStatusCreateFailed,
		FailureReasons: []string{"error 1", "error 2"},
	}

	c.collectResourceResult(item, "resources.failedResource")

	s.Len(c.failures, 1)
	s.Equal("failedResource", c.failures[0].ElementName)
	s.Equal("resources.failedResource", c.failures[0].ElementPath)
	s.Equal("resource", c.failures[0].ElementType)
	s.Len(c.failures[0].FailureReasons, 2)
	s.Empty(c.successful)
	s.Empty(c.interrupted)
}

func (s *ResultCollectorTestSuite) Test_collectResourceResult_adds_interrupted_resource() {
	c := &resultCollector{}

	item := &ResourceDeployItem{
		Name:   "interruptedResource",
		Status: core.ResourceStatusCreateInterrupted,
	}

	c.collectResourceResult(item, "resources.interruptedResource")

	s.Len(c.interrupted, 1)
	s.Equal("interruptedResource", c.interrupted[0].ElementName)
	s.Equal("resources.interruptedResource", c.interrupted[0].ElementPath)
	s.Equal("resource", c.interrupted[0].ElementType)
	s.Empty(c.successful)
	s.Empty(c.failures)
}

func (s *ResultCollectorTestSuite) Test_collectResourceResult_adds_successful_resource() {
	c := &resultCollector{}

	item := &ResourceDeployItem{
		Name:   "createdResource",
		Status: core.ResourceStatusCreated,
	}

	c.collectResourceResult(item, "resources.createdResource")

	s.Len(c.successful, 1)
	s.Equal("createdResource", c.successful[0].ElementName)
	s.Equal("resources.createdResource", c.successful[0].ElementPath)
	s.Equal("resource", c.successful[0].ElementType)
	s.Equal("created", c.successful[0].Action)
	s.Empty(c.failures)
	s.Empty(c.interrupted)
}

func (s *ResultCollectorTestSuite) Test_collectResourceResult_ignores_in_progress() {
	c := &resultCollector{}

	item := &ResourceDeployItem{
		Name:   "creatingResource",
		Status: core.ResourceStatusCreating,
	}

	c.collectResourceResult(item, "resources.creatingResource")

	s.Empty(c.successful)
	s.Empty(c.failures)
	s.Empty(c.interrupted)
}

// collectChildResult tests

func (s *ResultCollectorTestSuite) Test_collectChildResult_adds_failed_child() {
	c := &resultCollector{}

	item := &ChildDeployItem{
		Name:           "failedChild",
		Status:         core.InstanceStatusDeployFailed,
		FailureReasons: []string{"child error"},
	}

	c.collectChildResult(item, "children.failedChild")

	s.Len(c.failures, 1)
	s.Equal("failedChild", c.failures[0].ElementName)
	s.Equal("child", c.failures[0].ElementType)
}

func (s *ResultCollectorTestSuite) Test_collectChildResult_adds_successful_child() {
	c := &resultCollector{}

	item := &ChildDeployItem{
		Name:   "deployedChild",
		Status: core.InstanceStatusDeployed,
	}

	c.collectChildResult(item, "children.deployedChild")

	s.Len(c.successful, 1)
	s.Equal("deployedChild", c.successful[0].ElementName)
	s.Equal("deployed", c.successful[0].Action)
}

// collectLinkResult tests

func (s *ResultCollectorTestSuite) Test_collectLinkResult_adds_failed_link() {
	c := &resultCollector{}

	item := &LinkDeployItem{
		LinkName:       "resA::resB",
		Status:         core.LinkStatusCreateFailed,
		FailureReasons: []string{"link error"},
	}

	c.collectLinkResult(item, "links.resA::resB")

	s.Len(c.failures, 1)
	s.Equal("resA::resB", c.failures[0].ElementName)
	s.Equal("link", c.failures[0].ElementType)
}

func (s *ResultCollectorTestSuite) Test_collectLinkResult_adds_successful_link() {
	c := &resultCollector{}

	item := &LinkDeployItem{
		LinkName: "resA::resB",
		Status:   core.LinkStatusCreated,
	}

	c.collectLinkResult(item, "links.resA::resB")

	s.Len(c.successful, 1)
	s.Equal("resA::resB", c.successful[0].ElementName)
	s.Equal("created", c.successful[0].Action)
}

// collectFromItems tests

func (s *ResultCollectorTestSuite) Test_collectFromItems_collects_resources() {
	c := &resultCollector{}

	items := []DeployItem{
		{
			Type:     ItemTypeResource,
			Resource: &ResourceDeployItem{Name: "res1", Status: core.ResourceStatusCreated},
		},
		{
			Type:     ItemTypeResource,
			Resource: &ResourceDeployItem{Name: "res2", Status: core.ResourceStatusCreateFailed, FailureReasons: []string{"err"}},
		},
	}

	c.collectFromItems(items, "")

	s.Len(c.successful, 1)
	s.Len(c.failures, 1)
}

func (s *ResultCollectorTestSuite) Test_collectFromItems_collects_children() {
	c := &resultCollector{}

	items := []DeployItem{
		{
			Type:  ItemTypeChild,
			Child: &ChildDeployItem{Name: "child1", Status: core.InstanceStatusDeployed},
		},
	}

	c.collectFromItems(items, "")

	s.Len(c.successful, 1)
	s.Equal("children.child1", c.successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_collectFromItems_collects_links() {
	c := &resultCollector{}

	items := []DeployItem{
		{
			Type: ItemTypeLink,
			Link: &LinkDeployItem{LinkName: "a::b", Status: core.LinkStatusCreated},
		},
	}

	c.collectFromItems(items, "")

	s.Len(c.successful, 1)
	s.Equal("links.a::b", c.successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_collectFromItems_builds_nested_paths() {
	c := &resultCollector{}

	items := []DeployItem{
		{
			Type:     ItemTypeResource,
			Resource: &ResourceDeployItem{Name: "res1", Status: core.ResourceStatusCreated},
		},
	}

	c.collectFromItems(items, "children.parent")

	s.Len(c.successful, 1)
	s.Equal("children.parent::resources.res1", c.successful[0].ElementPath)
}

// collectFromChanges tests

func (s *ResultCollectorTestSuite) Test_collectFromChanges_nil_changes_is_noop() {
	c := &resultCollector{}

	c.collectFromChanges(nil, "", "")

	s.Empty(c.successful)
	s.Empty(c.failures)
	s.Empty(c.interrupted)
}

func (s *ResultCollectorTestSuite) Test_collectFromChanges_collects_new_resources() {
	c := &resultCollector{
		resourcesByName: map[string]*ResourceDeployItem{
			"child/newRes": {Name: "newRes", Status: core.ResourceStatusCreated},
		},
	}

	blueprintChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newRes": {},
		},
	}

	c.collectFromChanges(blueprintChanges, "children.child", "child")

	s.Len(c.successful, 1)
	s.Equal("children.child::resources.newRes", c.successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_collectFromChanges_collects_resource_changes() {
	c := &resultCollector{
		resourcesByName: map[string]*ResourceDeployItem{
			"child/changedRes": {Name: "changedRes", Status: core.ResourceStatusUpdated},
		},
	}

	blueprintChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"changedRes": {},
		},
	}

	c.collectFromChanges(blueprintChanges, "children.child", "child")

	s.Len(c.successful, 1)
	s.Equal("updated", c.successful[0].Action)
}
