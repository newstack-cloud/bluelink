package container

import (
	"context"
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/stretchr/testify/suite"
)

type GroupOrderedNodesTestSuite struct {
	groupFixture1 groupDeploymentNodeFixture
	groupFixture2 groupDeploymentNodeFixture
	groupFixture3 groupDeploymentNodeFixture
	suite.Suite
}

type groupDeploymentNodeFixture struct {
	orderedNodes      []*DeploymentNode
	refChainCollector refgraph.RefChainCollector
	// All the resource names or child blueprint names that are expected to be in each group.
	// The order of the groups matter but the order of the resources or child blueprints
	// in each group doesn't.
	expectedPresent [][]string
}

func (s *GroupOrderedNodesTestSuite) SetupSuite() {
	groupFixture1, err := groupFixture1()
	if err != nil {
		s.FailNow(err.Error())
	}
	s.groupFixture1 = groupFixture1

	groupFixture2, err := groupFixture2()
	if err != nil {
		s.FailNow(err.Error())
	}
	s.groupFixture2 = groupFixture2

	groupFixture3, err := groupFixture3()
	if err != nil {
		s.FailNow(err.Error())
	}
	s.groupFixture3 = groupFixture3
}

func (s *GroupOrderedNodesTestSuite) Test_group_links_for_deployment_with_circular_links() {
	groups, err := GroupOrderedNodes(
		s.groupFixture1.orderedNodes,
		s.groupFixture1.refChainCollector,
	)
	s.Assert().NoError(err)
	s.Assert().Len(groups, len(s.groupFixture1.expectedPresent))

	s.assertExpectedGroups(groups, s.groupFixture1.expectedPresent)
}

func (s *GroupOrderedNodesTestSuite) Test_group_links_for_deployment_without_circular_links() {
	groups, err := GroupOrderedNodes(
		s.groupFixture2.orderedNodes,
		s.groupFixture2.refChainCollector,
	)
	s.Assert().NoError(err)
	s.Assert().Len(groups, len(s.groupFixture2.expectedPresent))

	s.assertExpectedGroups(groups, s.groupFixture2.expectedPresent)
}

func (s *GroupOrderedNodesTestSuite) Test_group_links_for_deployment_based_on_references_and_dependencies() {
	groups, err := GroupOrderedNodes(
		s.groupFixture3.orderedNodes,
		s.groupFixture3.refChainCollector,
	)
	s.Assert().NoError(err)
	s.Assert().Len(groups, len(s.groupFixture3.expectedPresent))

	s.assertExpectedGroups(groups, s.groupFixture3.expectedPresent)
}

func (s *GroupOrderedNodesTestSuite) assertExpectedGroups(
	groups [][]*DeploymentNode,
	expectedPresent [][]string,
) {
	for i, group := range groups {
		expectedGroupNames := expectedPresent[i]
		expectedGroupNamesNormalised := []string{}
		copy(expectedGroupNamesNormalised, expectedGroupNames)
		groupNormalised := []string{}
		for _, node := range group {
			groupNormalised = append(groupNormalised, node.Name())
		}
		slices.Sort(groupNormalised)
		slices.Sort(expectedGroupNames)
		s.Assert().Equal(expectedGroupNames, groupNormalised)

	}
}

var testGroupProviderImpl = newTestAWSProvider(
	/* alwaysStabilise */ false,
	/* skipRetryFailuresForLinkNames */ []string{},
	/* stateContainer */ nil,
)

func groupFixture1() (groupDeploymentNodeFixture, error) {
	orderedNodes := groupFixture1Nodes()
	refChainCollector, err := groupFixture1RefChains(orderedNodes)
	if err != nil {
		return groupDeploymentNodeFixture{}, err
	}

	return groupDeploymentNodeFixture{
		orderedNodes:      orderedNodes,
		refChainCollector: refChainCollector,
		expectedPresent: [][]string{
			{
				"resources.orderApi",
				"resources.ordersTable",
			},
			{
				"resources.ordersStream",
				"resources.getOrdersFunction",
				"resources.createOrderFunction",
				"resources.updateOrderFunction",
			},
			{
				"resources.statsAccumulatorFunction",
			},
		},
	}, nil
}

func groupFixture1Nodes() []*DeploymentNode {
	apiGatewayLambdaLinkImpl, _ := testGroupProviderImpl.Link(context.TODO(), "aws/apigateway/api", "aws/lambda/function")
	orderApi := &links.ChainLinkNode{
		ResourceName: "orderApi",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/apigateway/api"},
		},
		Paths: []string{},
		LinkImplementations: map[string]provider.Link{
			"getOrdersFunction":   apiGatewayLambdaLinkImpl,
			"createOrderFunction": apiGatewayLambdaLinkImpl,
			"updateOrderFunction": apiGatewayLambdaLinkImpl,
		},
		LinkedFrom: []*links.ChainLinkNode{},
		LinksTo:    []*links.ChainLinkNode{},
	}

	lambdaDynamoDBTableLink, _ := testProviderImpl.Link(context.TODO(), "aws/lambda/function", "aws/dynamodb/table")
	getOrdersFunction := &links.ChainLinkNode{
		ResourceName: "getOrdersFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		Paths: []string{"/orderApi"},
		LinkImplementations: map[string]provider.Link{
			"ordersTable": lambdaDynamoDBTableLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}
	createOrderFunction := &links.ChainLinkNode{
		ResourceName: "createOrderFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		LinkImplementations: map[string]provider.Link{
			"ordersTable": lambdaDynamoDBTableLink,
		},
		Paths: []string{"/orderApi"},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}
	updateOrderFunction := &links.ChainLinkNode{
		ResourceName: "updateOrderFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		LinkImplementations: map[string]provider.Link{
			"ordersTable": lambdaDynamoDBTableLink,
		},
		Paths: []string{"/orderApi"},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}

	dynamoDBTableStreamLink, _ := testProviderImpl.Link(context.TODO(), "aws/dynamodb/table", "aws/dynamodb/stream")
	// The only hard link in this chain is between the orders table
	// and the orders stream.
	ordersTable := &links.ChainLinkNode{
		ResourceName: "ordersTable",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
		},
		LinkImplementations: map[string]provider.Link{
			"ordersStream": dynamoDBTableStreamLink,
		},
		Paths: []string{
			"/orderApi/getOrdersFunction",
			"/orderApi/createOrderFunction",
			"/orderApi/updateOrderFunction",
		},
		LinkedFrom: []*links.ChainLinkNode{
			getOrdersFunction,
		},
	}

	dynamoDBStreamLambdaLink, _ := testProviderImpl.Link(context.TODO(), "aws/dynamodb/stream", "aws/lambda/function")
	ordersStream := &links.ChainLinkNode{
		ResourceName: "ordersStream",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/stream"},
		},
		Paths: []string{
			"/orderApi/getOrdersFunction/ordersTable",
			"/orderApi/createOrderFunction/ordersTable",
			"/orderApi/updateOrderFunction/ordersTable",
		},
		LinkImplementations: map[string]provider.Link{
			"statsAccumulatorFunction": dynamoDBStreamLambdaLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			getOrdersFunction,
			createOrderFunction,
			updateOrderFunction,
			ordersTable,
		},
		LinksTo: []*links.ChainLinkNode{},
	}

	// Includes transitive soft circular link.
	statsAccumulatorFunction := &links.ChainLinkNode{
		ResourceName: "statsAccumulatorFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		Paths: []string{
			"/orderApi/getOrdersFunction/ordersTable/ordersStream",
			"/orderApi/createOrderFunction/ordersTable/ordersStream",
			"/orderApi/updateOrderFunction/ordersTable/ordersStream",
		},
		LinkImplementations: map[string]provider.Link{
			"ordersTable": lambdaDynamoDBTableLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			ordersStream,
		},
	}

	orderApi.LinksTo = []*links.ChainLinkNode{
		getOrdersFunction,
		createOrderFunction,
		updateOrderFunction,
	}
	getOrdersFunction.LinksTo = []*links.ChainLinkNode{
		ordersTable,
	}
	createOrderFunction.LinksTo = []*links.ChainLinkNode{
		ordersTable,
	}
	updateOrderFunction.LinksTo = []*links.ChainLinkNode{
		ordersTable,
	}
	ordersTable.LinksTo = []*links.ChainLinkNode{
		ordersStream,
	}
	ordersStream.LinksTo = []*links.ChainLinkNode{
		statsAccumulatorFunction,
	}
	statsAccumulatorFunction.LinksTo = []*links.ChainLinkNode{
		ordersTable,
	}

	return []*DeploymentNode{
		{ChainLinkNode: orderApi},
		{ChainLinkNode: ordersTable},
		{ChainLinkNode: ordersStream},
		{ChainLinkNode: getOrdersFunction},
		{ChainLinkNode: createOrderFunction},
		{ChainLinkNode: updateOrderFunction},
		{ChainLinkNode: statsAccumulatorFunction},
	}
}

func groupFixture1RefChains(
	nodes []*DeploymentNode,
) (refgraph.RefChainCollector, error) {
	collector := refgraph.NewRefChainCollector()
	collectNodesAsRefs(nodes, collector)

	collector.Collect(
		"resources.ordersTable",
		nil,
		"resources.getOrdersFunction",
		[]string{"subRef:resources.getOrdersFunction"},
	)

	return collector, nil
}

func groupFixture2() (groupDeploymentNodeFixture, error) {
	var orderedNodes = groupFixture2Nodes()
	refChainCollector, err := groupFixture2RefChains(orderedNodes)
	if err != nil {
		return groupDeploymentNodeFixture{}, err
	}

	return groupDeploymentNodeFixture{
		orderedNodes:      orderedNodes,
		refChainCollector: refChainCollector,
		expectedPresent: [][]string{
			{
				"resources.vpc1",
			},
			{
				"resources.routeTable1",
				"resources.igw1",
			},
			{
				"resources.route1",
				"resources.subnet1",
				"resources.sg1",
			},
		},
	}, nil
}

func groupFixture2Nodes() []*DeploymentNode {
	routeRouteTableLink, _ := testGroupProviderImpl.Link(context.TODO(), "aws/ec2/route", "aws/ec2/routeTable")
	routeIGWLink, _ := testProviderImpl.Link(context.TODO(), "aws/ec2/route", "aws/ec2/internetGateway")
	route := &links.ChainLinkNode{
		ResourceName: "route1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/route"},
		},
		Paths: []string{},
		LinkImplementations: map[string]provider.Link{
			"routeTable1": routeRouteTableLink,
			"igw1":        routeIGWLink,
		},
		LinkedFrom: []*links.ChainLinkNode{},
		LinksTo:    []*links.ChainLinkNode{},
	}

	routeTableVPCLink, _ := testProviderImpl.Link(context.TODO(), "aws/ec2/routeTable", "aws/ec2/vpc")
	routeTable := &links.ChainLinkNode{
		ResourceName: "routeTable1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/routeTable"},
		},
		Paths: []string{
			"/route1",
		},
		LinkImplementations: map[string]provider.Link{
			"vpc1": routeTableVPCLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			route,
		},
		LinksTo: []*links.ChainLinkNode{},
	}

	internetGateway := &links.ChainLinkNode{
		ResourceName: "igw1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/internetGateway"},
		},
		Paths: []string{
			"/route1",
		},
		LinkImplementations: map[string]provider.Link{},
		LinkedFrom: []*links.ChainLinkNode{
			route,
		},
		LinksTo: []*links.ChainLinkNode{},
	}

	subnetVPCLink, _ := testProviderImpl.Link(context.TODO(), "aws/ec2/subnet", "aws/ec2/vpc")
	subnet := &links.ChainLinkNode{
		ResourceName: "subnet1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/subnet"},
		},
		Paths: []string{},
		LinkImplementations: map[string]provider.Link{
			"vpc1": subnetVPCLink,
		},
		LinkedFrom: []*links.ChainLinkNode{},
		LinksTo:    []*links.ChainLinkNode{},
	}

	securityGroupLink, _ := testProviderImpl.Link(context.TODO(), "aws/ec2/securityGroup", "aws/ec2/vpc")
	securityGroup := &links.ChainLinkNode{
		ResourceName: "sg1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/securityGroup"},
		},
		Paths: []string{},
		LinkImplementations: map[string]provider.Link{
			"vpc1": securityGroupLink,
		},
		LinkedFrom: []*links.ChainLinkNode{},
		LinksTo:    []*links.ChainLinkNode{},
	}

	vpc := &links.ChainLinkNode{
		ResourceName: "vpc1",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/vpc"},
		},
		Paths: []string{
			"/route1/routeTable1",
			"/subnet1",
			"/sg1",
		},
		LinkImplementations: map[string]provider.Link{},
		LinkedFrom: []*links.ChainLinkNode{
			routeTable,
			subnet,
			securityGroup,
		},
		LinksTo: []*links.ChainLinkNode{},
	}

	route.LinksTo = []*links.ChainLinkNode{
		routeTable,
		internetGateway,
	}
	routeTable.LinksTo = []*links.ChainLinkNode{
		vpc,
	}
	subnet.LinksTo = []*links.ChainLinkNode{
		vpc,
	}
	securityGroup.LinksTo = []*links.ChainLinkNode{
		vpc,
	}

	return []*DeploymentNode{
		{ChainLinkNode: vpc},
		{ChainLinkNode: routeTable},
		{ChainLinkNode: internetGateway},
		{ChainLinkNode: route},
		{ChainLinkNode: subnet},
		{ChainLinkNode: securityGroup},
	}
}

func groupFixture2RefChains(
	nodes []*DeploymentNode,
) (refgraph.RefChainCollector, error) {
	collector := refgraph.NewRefChainCollector()
	collectNodesAsRefs(nodes, collector)

	collector.Collect("resources.vpc1", nil, "resources.sg1", []string{"subRef:resources.sg1"})

	return collector, nil
}

func groupFixture3() (groupDeploymentNodeFixture, error) {
	var orderedNodes = groupFixture3Nodes()
	refChainCollector, err := groupFixture3RefChains(orderedNodes)
	if err != nil {
		return groupDeploymentNodeFixture{}, err
	}

	return groupDeploymentNodeFixture{
		orderedNodes:      orderedNodes,
		refChainCollector: refChainCollector,
		expectedPresent: [][]string{
			{
				"resources.ordersTable",
				"resources.orderApi",
			},
			{
				"resources.createOrderFunction",
			},
			{
				"children.billingStack",
			},
			{
				"children.logisticsStack",
			},
			{
				"resources.ordersStream",
				"resources.getOrdersFunction",
				"resources.updateOrderFunction",
			},
			{
				"resources.statsAccumulatorFunction",
			},
		},
	}, nil
}

func groupFixture3Nodes() []*DeploymentNode {
	apiGatewayLambdaLinkImpl, _ := testGroupProviderImpl.Link(context.TODO(), "aws/apigateway/api", "aws/lambda/function")
	orderApi := &links.ChainLinkNode{
		ResourceName: "orderApi",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/apigateway/api"},
		},
		Paths: []string{},
		LinkImplementations: map[string]provider.Link{
			"getOrdersFunction":   apiGatewayLambdaLinkImpl,
			"createOrderFunction": apiGatewayLambdaLinkImpl,
			"updateOrderFunction": apiGatewayLambdaLinkImpl,
		},
		LinkedFrom: []*links.ChainLinkNode{},
		LinksTo:    []*links.ChainLinkNode{},
	}

	lambdaDynamoDBTableLink, _ := testProviderImpl.Link(context.TODO(), "aws/lambda/function", "aws/dynamodb/table")
	// For fixture 3, functions do not link to tables, references are being tested for this fixture
	// so each function will have a reference to the orders table defined in the ref chain fixtures.
	getOrdersFunction := &links.ChainLinkNode{
		ResourceName: "getOrdersFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		Paths:               []string{"/orderApi"},
		LinkImplementations: map[string]provider.Link{},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}
	createOrderFunction := &links.ChainLinkNode{
		ResourceName: "createOrderFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		LinkImplementations: map[string]provider.Link{},
		Paths:               []string{"/orderApi"},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}
	updateOrderFunction := &links.ChainLinkNode{
		ResourceName: "updateOrderFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		LinkImplementations: map[string]provider.Link{},
		Paths:               []string{"/orderApi"},
		LinkedFrom: []*links.ChainLinkNode{
			orderApi,
		},
	}

	dynamoDBTableStreamLink, _ := testProviderImpl.Link(context.TODO(), "aws/dynamodb/table", "aws/dynamodb/stream")
	// The only hard link in this chain is between the orders table
	// and the orders stream.
	ordersTable := &links.ChainLinkNode{
		ResourceName: "ordersTable",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
		},
		LinkImplementations: map[string]provider.Link{
			"ordersStream": dynamoDBTableStreamLink,
		},
		Paths:      []string{},
		LinkedFrom: []*links.ChainLinkNode{},
	}

	dynamoDBStreamLambdaLink, _ := testProviderImpl.Link(context.TODO(), "aws/dynamodb/stream", "aws/lambda/function")
	ordersStream := &links.ChainLinkNode{
		ResourceName: "ordersStream",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/stream"},
		},
		Paths: []string{
			"/ordersTable",
		},
		LinkImplementations: map[string]provider.Link{
			"statsAccumulatorFunction": dynamoDBStreamLambdaLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			ordersTable,
		},
		LinksTo: []*links.ChainLinkNode{},
	}

	// Includes transitive soft circular link.
	statsAccumulatorFunction := &links.ChainLinkNode{
		ResourceName: "statsAccumulatorFunction",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		},
		Paths: []string{
			"/ordersTable/ordersStream",
		},
		LinkImplementations: map[string]provider.Link{
			"ordersTable": lambdaDynamoDBTableLink,
		},
		LinkedFrom: []*links.ChainLinkNode{
			ordersStream,
		},
	}

	orderApi.LinksTo = []*links.ChainLinkNode{
		getOrdersFunction,
		createOrderFunction,
		updateOrderFunction,
	}
	ordersTable.LinksTo = []*links.ChainLinkNode{
		ordersStream,
	}
	ordersStream.LinksTo = []*links.ChainLinkNode{
		statsAccumulatorFunction,
	}
	statsAccumulatorFunction.LinksTo = []*links.ChainLinkNode{
		ordersTable,
	}

	billingStack := &refgraph.ReferenceChainNode{
		ElementName: "children.billingStack",
	}
	logisticsStack := &refgraph.ReferenceChainNode{
		ElementName: "children.logisticsStack",
	}

	// This is the equivalent to the result of ordering the link nodes with the
	// OrderLinksForDeployment function.
	return []*DeploymentNode{
		{ChainLinkNode: orderApi},
		{ChainLinkNode: ordersTable},
		{ChainLinkNode: createOrderFunction},
		{ChildNode: billingStack},
		{ChildNode: logisticsStack},
		{ChainLinkNode: getOrdersFunction},
		{ChainLinkNode: updateOrderFunction},
		{ChainLinkNode: ordersStream},
		{ChainLinkNode: statsAccumulatorFunction},
	}
}

func groupFixture3RefChains(
	nodes []*DeploymentNode,
) (refgraph.RefChainCollector, error) {
	collector := refgraph.NewRefChainCollector()
	collectNodesAsRefs(nodes, collector)

	collector.Collect(
		"resources.ordersTable",
		nil,
		"resources.getOrdersFunction",
		[]string{"subRef:resources.getOrdersFunction"},
	)
	collector.Collect(
		"resources.ordersTable",
		nil,
		"resources.createOrderFunction",
		[]string{"subRef:resources.createOrderFunction"},
	)
	collector.Collect(
		"resources.ordersTable",
		nil,
		"resources.updateOrderFunction",
		[]string{"subRef:resources.updateOrderFunction"},
	)

	collector.Collect(
		"resources.createOrderFunction",
		nil,
		"resources.getOrdersFunction",
		[]string{validation.CreateDependencyRefTag("resources.getOrdersFunction")},
	)

	collector.Collect(
		"children.billingStack",
		nil,
		"children.logisticsStack",
		[]string{"subRef:children.logisticsStack"},
	)
	collector.Collect(
		"children.logisticsStack",
		nil,
		"resources.getOrdersFunction",
		[]string{"subRef:resources.getOrdersFunction"},
	)
	collector.Collect(
		"resources.createOrderFunction",
		nil,
		"children.billingStack",
		[]string{"subRef:children.billingStack"},
	)

	return collector, nil
}

func collectNodesAsRefs(nodes []*DeploymentNode, collector refgraph.RefChainCollector) error {
	params := createParams()
	for _, node := range nodes {
		if node.Type() == DeploymentNodeTypeResource {
			err := collectLinksFromChain(context.TODO(), node.ChainLinkNode, collector, params)
			if err != nil {
				return err
			}
		} else if node.Type() == DeploymentNodeTypeChild {
			collector.Collect(node.Name(), node.ChildNode.Element, "", []string{})
		}
	}

	return nil
}

func TestGroupOrderedNodesTestSuite(t *testing.T) {
	suite.Run(t, new(GroupOrderedNodesTestSuite))
}
