package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/stretchr/testify/suite"
)

type DependencyUtilsTestSuite struct {
	suite.Suite
}

func (s *DependencyUtilsTestSuite) Test_populates_dependency_for_linked_to_resource() {
	saveOrderFunctionNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "saveOrderFunction",
			LinksTo: []*links.ChainLinkNode{
				{
					ResourceName: "ordersTable",
				},
			},
			LinkedFrom: []*links.ChainLinkNode{},
			LinkImplementations: map[string]provider.Link{
				"ordersTable": &testLambdaDynamoDBTableLink{},
			},
		},
		DirectDependencies: []*DeploymentNode{},
	}
	ordersTableNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "ordersTable",
			LinkedFrom: []*links.ChainLinkNode{
				saveOrderFunctionNode.ChainLinkNode,
			},
			LinkImplementations: map[string]provider.Link{},
		},
		DirectDependencies: []*DeploymentNode{},
	}
	nodes := []*DeploymentNode{
		saveOrderFunctionNode,
		ordersTableNode,
	}
	err := PopulateDirectDependencies(
		context.Background(),
		nodes,
		refgraph.NewRefChainCollector(),
		s.createBlueprintParams(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(nodes[0].DirectDependencies, []*DeploymentNode{
		ordersTableNode,
	})
}

func (s *DependencyUtilsTestSuite) Test_does_not_populate_direct_deps_when_there_is_no_direct_dependency() {
	saveOrderFunctionNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "saveOrderFunction",
			LinksTo: []*links.ChainLinkNode{
				{
					ResourceName: "preprocessOrderFunction",
				},
			},
			LinkedFrom: []*links.ChainLinkNode{},
			LinkImplementations: map[string]provider.Link{
				// Lambda -> Lambda link does not have a priority
				// resource, therefore it should not be considered
				// as a dependency.
				"preprocessOrderFunction": &testLambdaLambdaLink{},
			},
		},
	}
	preprocessOrderFunctionNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "preprocessOrderFunction",
			LinkedFrom: []*links.ChainLinkNode{
				saveOrderFunctionNode.ChainLinkNode,
			},
			LinkImplementations: map[string]provider.Link{},
		},
	}
	nodes := []*DeploymentNode{
		saveOrderFunctionNode,
		preprocessOrderFunctionNode,
	}
	err := PopulateDirectDependencies(
		context.Background(),
		nodes,
		refgraph.NewRefChainCollector(),
		s.createBlueprintParams(),
	)
	s.Require().NoError(err)
	s.Assert().Empty(nodes[0].DirectDependencies)
}

func (s *DependencyUtilsTestSuite) createBlueprintParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func TestDependencyUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(DependencyUtilsTestSuite))
}
