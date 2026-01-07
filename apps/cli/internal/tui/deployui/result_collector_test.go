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

// CollectResourceResult tests

func (s *ResultCollectorTestSuite) Test_CollectResourceResult_adds_failed_resource() {
	c := &ResultCollector{}

	item := &ResourceDeployItem{
		Name:           "failedResource",
		Status:         core.ResourceStatusCreateFailed,
		FailureReasons: []string{"error 1", "error 2"},
	}

	c.CollectResourceResult(item, "resources.failedResource")

	s.Len(c.Failures, 1)
	s.Equal("failedResource", c.Failures[0].ElementName)
	s.Equal("resources.failedResource", c.Failures[0].ElementPath)
	s.Equal("resource", c.Failures[0].ElementType)
	s.Len(c.Failures[0].FailureReasons, 2)
	s.Empty(c.Successful)
	s.Empty(c.Interrupted)
}

func (s *ResultCollectorTestSuite) Test_CollectResourceResult_adds_interrupted_resource() {
	c := &ResultCollector{}

	item := &ResourceDeployItem{
		Name:   "interruptedResource",
		Status: core.ResourceStatusCreateInterrupted,
	}

	c.CollectResourceResult(item, "resources.interruptedResource")

	s.Len(c.Interrupted, 1)
	s.Equal("interruptedResource", c.Interrupted[0].ElementName)
	s.Equal("resources.interruptedResource", c.Interrupted[0].ElementPath)
	s.Equal("resource", c.Interrupted[0].ElementType)
	s.Empty(c.Successful)
	s.Empty(c.Failures)
}

func (s *ResultCollectorTestSuite) Test_CollectResourceResult_adds_successful_resource() {
	c := &ResultCollector{}

	item := &ResourceDeployItem{
		Name:   "createdResource",
		Status: core.ResourceStatusCreated,
	}

	c.CollectResourceResult(item, "resources.createdResource")

	s.Len(c.Successful, 1)
	s.Equal("createdResource", c.Successful[0].ElementName)
	s.Equal("resources.createdResource", c.Successful[0].ElementPath)
	s.Equal("resource", c.Successful[0].ElementType)
	s.Equal("created", c.Successful[0].Action)
	s.Empty(c.Failures)
	s.Empty(c.Interrupted)
}

func (s *ResultCollectorTestSuite) Test_CollectResourceResult_ignores_in_progress() {
	c := &ResultCollector{}

	item := &ResourceDeployItem{
		Name:   "creatingResource",
		Status: core.ResourceStatusCreating,
	}

	c.CollectResourceResult(item, "resources.creatingResource")

	s.Empty(c.Successful)
	s.Empty(c.Failures)
	s.Empty(c.Interrupted)
}

// CollectChildResult tests

func (s *ResultCollectorTestSuite) Test_CollectChildResult_adds_failed_child() {
	c := &ResultCollector{}

	item := &ChildDeployItem{
		Name:           "failedChild",
		Status:         core.InstanceStatusDeployFailed,
		FailureReasons: []string{"child error"},
	}

	c.CollectChildResult(item, "children.failedChild")

	s.Len(c.Failures, 1)
	s.Equal("failedChild", c.Failures[0].ElementName)
	s.Equal("child", c.Failures[0].ElementType)
}

func (s *ResultCollectorTestSuite) Test_CollectChildResult_adds_successful_child() {
	c := &ResultCollector{}

	item := &ChildDeployItem{
		Name:   "deployedChild",
		Status: core.InstanceStatusDeployed,
	}

	c.CollectChildResult(item, "children.deployedChild")

	s.Len(c.Successful, 1)
	s.Equal("deployedChild", c.Successful[0].ElementName)
	s.Equal("deployed", c.Successful[0].Action)
}

// CollectLinkResult tests

func (s *ResultCollectorTestSuite) Test_CollectLinkResult_adds_failed_link() {
	c := &ResultCollector{}

	item := &LinkDeployItem{
		LinkName:       "resA::resB",
		Status:         core.LinkStatusCreateFailed,
		FailureReasons: []string{"link error"},
	}

	c.CollectLinkResult(item, "links.resA::resB")

	s.Len(c.Failures, 1)
	s.Equal("resA::resB", c.Failures[0].ElementName)
	s.Equal("link", c.Failures[0].ElementType)
}

func (s *ResultCollectorTestSuite) Test_CollectLinkResult_adds_successful_link() {
	c := &ResultCollector{}

	item := &LinkDeployItem{
		LinkName: "resA::resB",
		Status:   core.LinkStatusCreated,
	}

	c.CollectLinkResult(item, "links.resA::resB")

	s.Len(c.Successful, 1)
	s.Equal("resA::resB", c.Successful[0].ElementName)
	s.Equal("created", c.Successful[0].Action)
}

// CollectFromItems tests

func (s *ResultCollectorTestSuite) Test_CollectFromItems_collects_resources() {
	c := &ResultCollector{}

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

	c.CollectFromItems(items, "")

	s.Len(c.Successful, 1)
	s.Len(c.Failures, 1)
}

func (s *ResultCollectorTestSuite) Test_CollectFromItems_collects_children() {
	c := &ResultCollector{}

	items := []DeployItem{
		{
			Type:  ItemTypeChild,
			Child: &ChildDeployItem{Name: "child1", Status: core.InstanceStatusDeployed},
		},
	}

	c.CollectFromItems(items, "")

	s.Len(c.Successful, 1)
	s.Equal("children.child1", c.Successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_CollectFromItems_collects_links() {
	c := &ResultCollector{}

	items := []DeployItem{
		{
			Type: ItemTypeLink,
			Link: &LinkDeployItem{LinkName: "a::b", Status: core.LinkStatusCreated},
		},
	}

	c.CollectFromItems(items, "")

	s.Len(c.Successful, 1)
	s.Equal("links.a::b", c.Successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_CollectFromItems_builds_nested_paths() {
	c := &ResultCollector{}

	items := []DeployItem{
		{
			Type:     ItemTypeResource,
			Resource: &ResourceDeployItem{Name: "res1", Status: core.ResourceStatusCreated},
		},
	}

	c.CollectFromItems(items, "children.parent")

	s.Len(c.Successful, 1)
	s.Equal("children.parent::resources.res1", c.Successful[0].ElementPath)
}

// CollectFromChanges tests

func (s *ResultCollectorTestSuite) Test_CollectFromChanges_nil_changes_is_noop() {
	c := &ResultCollector{}

	c.CollectFromChanges(nil, "", "")

	s.Empty(c.Successful)
	s.Empty(c.Failures)
	s.Empty(c.Interrupted)
}

func (s *ResultCollectorTestSuite) Test_CollectFromChanges_collects_new_resources() {
	c := &ResultCollector{
		ResourcesByName: map[string]*ResourceDeployItem{
			"child/newRes": {Name: "newRes", Status: core.ResourceStatusCreated},
		},
	}

	blueprintChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newRes": {},
		},
	}

	c.CollectFromChanges(blueprintChanges, "children.child", "child")

	s.Len(c.Successful, 1)
	s.Equal("children.child::resources.newRes", c.Successful[0].ElementPath)
}

func (s *ResultCollectorTestSuite) Test_CollectFromChanges_collects_resource_changes() {
	c := &ResultCollector{
		ResourcesByName: map[string]*ResourceDeployItem{
			"child/changedRes": {Name: "changedRes", Status: core.ResourceStatusUpdated},
		},
	}

	blueprintChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"changedRes": {},
		},
	}

	c.CollectFromChanges(blueprintChanges, "children.child", "child")

	s.Len(c.Successful, 1)
	s.Equal("updated", c.Successful[0].Action)
}
