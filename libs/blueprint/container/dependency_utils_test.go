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

// Regression test for dependencies routed through derived values:
// a resource referencing `values.<v>` where <v> is defined with a reference
// to another resource's computed field must get a direct dependency edge
// on that resource.
func (s *DependencyUtilsTestSuite) Test_populates_dependency_for_resource_referenced_through_derived_value() {
	configFunctionNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName:        "configFunction",
			LinksTo:             []*links.ChainLinkNode{},
			LinkedFrom:          []*links.ChainLinkNode{},
			LinkImplementations: map[string]provider.Link{},
		},
		DirectDependencies: []*DeploymentNode{},
	}
	secretStoreNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName:        "secretStore",
			LinksTo:             []*links.ChainLinkNode{},
			LinkedFrom:          []*links.ChainLinkNode{},
			LinkImplementations: map[string]provider.Link{},
		},
		DirectDependencies: []*DeploymentNode{},
	}
	nodes := []*DeploymentNode{
		secretStoreNode,
		configFunctionNode,
	}

	collector := refgraph.NewRefChainCollector()
	err := collector.Collect("resources.secretStore", nil, "", []string{})
	s.Require().NoError(err)
	err = collector.Collect("resources.configFunction", nil, "", []string{})
	s.Require().NoError(err)
	// configFunction references values.secretId, which is derived from
	// the secretStore resource's computed field.
	err = collector.Collect(
		"values.secretId",
		nil,
		"resources.configFunction",
		[]string{"subRef:resources.configFunction"},
	)
	s.Require().NoError(err)
	err = collector.Collect(
		"resources.secretStore",
		nil,
		"values.secretId",
		[]string{"subRef:values.secretId"},
	)
	s.Require().NoError(err)

	err = PopulateDirectDependencies(
		context.Background(),
		nodes,
		collector,
		s.createBlueprintParams(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		[]*DeploymentNode{secretStoreNode},
		configFunctionNode.DirectDependencies,
	)
	s.Assert().Empty(secretStoreNode.DirectDependencies)
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

// A soft link does not make the resource it links to a dependency, whatever priority
// resource it names.
//
// A hard link is one where the priority resource must be created before the other, and a
// soft link is one where it does not matter which is created first, so the two are a
// single statement about ordering rather than separate properties. Only hard links are
// recorded as references, which is what orders resources into groups, so taking the
// priority alone would produce a dependency that ordering knows nothing about, the
// resource that depends could be grouped before the resource it depends on, and be left
// waiting for the rest of the deployment.
func (s *DependencyUtilsTestSuite) Test_soft_link_with_a_priority_resource_is_not_a_dependency() {
	cacheNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName: "replicationGroup",
			LinksTo: []*links.ChainLinkNode{
				{ResourceName: "authSecret"},
			},
			LinkedFrom: []*links.ChainLinkNode{},
			LinkImplementations: map[string]provider.Link{
				"authSecret": &softLinkWithPriority{},
			},
		},
		DirectDependencies: []*DeploymentNode{},
	}
	secretNode := &DeploymentNode{
		ChainLinkNode: &links.ChainLinkNode{
			ResourceName:        "authSecret",
			LinksTo:             []*links.ChainLinkNode{},
			LinkedFrom:          []*links.ChainLinkNode{cacheNode.ChainLinkNode},
			LinkImplementations: map[string]provider.Link{},
		},
		DirectDependencies: []*DeploymentNode{},
	}

	nodes := []*DeploymentNode{cacheNode, secretNode}
	err := PopulateDirectDependencies(
		context.Background(),
		nodes,
		refgraph.NewRefChainCollector(),
		s.createBlueprintParams(),
	)
	s.Require().NoError(err)

	s.Assert().Empty(
		cacheNode.DirectDependencies,
		"a soft link produced a dependency that nothing orders",
	)
	s.Assert().Empty(
		secretNode.DirectDependencies,
		"a soft link produced a dependency that nothing orders",
	)
}

// A link that names a priority resource while declaring itself soft, which is the
// contradiction the framework has to ignore rather than half-honour.
type softLinkWithPriority struct {
	*testLambdaDynamoDBTableLink
}

func (l *softLinkWithPriority) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindSoft,
	}, nil
}

func (l *softLinkWithPriority) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResourceB,
	}, nil
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
