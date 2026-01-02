package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/stretchr/testify/suite"
)

type DeploymentUtilsTestSuite struct {
	suite.Suite
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_true_for_new_resource() {
	node := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "testResource",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"testResource": {},
		},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().True(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_true_for_changed_resource() {
	node := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "testResource",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"testResource": {},
		},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().True(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_false_for_unchanged_resource() {
	node := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "unchangedResource",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"otherResource": {},
		},
		ResourceChanges: map[string]provider.Changes{
			"anotherResource": {},
		},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().False(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_true_for_new_child() {
	node := &DeploymentNode{
		ChildNode: &refgraph.ReferenceChainNode{
			ElementName: "children.testChild",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"testChild": {},
		},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().True(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_true_for_changed_child() {
	node := &DeploymentNode{
		ChildNode: &refgraph.ReferenceChainNode{
			ElementName: "children.testChild",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"testChild": {},
		},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().True(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_true_for_recreate_child() {
	node := &DeploymentNode{
		ChildNode: &refgraph.ReferenceChainNode{
			ElementName: "children.testChild",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		RecreateChildren: []string{"testChild"},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().True(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_false_for_unchanged_child() {
	node := &DeploymentNode{
		ChildNode: &refgraph.ReferenceChainNode{
			ElementName: "children.unchangedChild",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"otherChild": {},
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"anotherChild": {},
		},
		RecreateChildren: []string{"yetAnotherChild"},
	}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().False(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_false_for_nil_changes() {
	node := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "testResource",
		},
	}

	result := nodeHasChanges(node, nil)
	s.Assert().False(result)
}

func (s *DeploymentUtilsTestSuite) Test_nodeHasChanges_returns_false_for_empty_changes() {
	node := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "testResource",
		},
	}
	blueprintChanges := &changes.BlueprintChanges{}

	result := nodeHasChanges(node, blueprintChanges)
	s.Assert().False(result)
}

func TestDeploymentUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(DeploymentUtilsTestSuite))
}
