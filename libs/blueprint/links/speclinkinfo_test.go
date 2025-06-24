package links

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	. "gopkg.in/check.v1"
)

type SpecLinkInfoTestSuite struct {
	resourceProviders map[string]provider.Provider
	linkRegistry      provider.LinkRegistry
}

var _ = Suite(&SpecLinkInfoTestSuite{})

func (s *SpecLinkInfoTestSuite) SetUpSuite(c *C) {
	awsProvider := newTestAWSProvider()
	s.resourceProviders = map[string]provider.Provider{
		"aws/apigateway/api":         awsProvider,
		"aws/sqs/queue":              awsProvider,
		"aws/lambda/function":        awsProvider,
		"stratosaws/lambda/function": awsProvider,
		"aws/dynamodb/table":         awsProvider,
		"aws/dynamodb/stream":        awsProvider,
		"aws/iam/role":               awsProvider,
		"stratosaws/iam/role":        awsProvider,
	}
	s.linkRegistry = provider.NewLinkRegistry(s.resourceProviders)
}

func (s *SpecLinkInfoTestSuite) Test_get_links_from_spec_1(c *C) {
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{
			schema: testSpecLinkInfoBlueprintSchema1,
		}, nil)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	chains, err := specLinkInfo.Links(context.Background())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	// Using snapshots for chain links as is more convenient than building custom test functions
	// that traverse through and compare the produced chain with a "hand-crafted" expected chain.
	// Not using snapshots would require value comparisions as you traverse through the pointers in the chain.
	// However, it's really important that you take care when reviewing failing snapshot tests
	// and not just re-building the snapshots without checking the changes are correct!
	err = testhelpers.Snapshot(normaliseForSnapshot(chains, []string{}))
	if err != nil {
		c.Error(err)
	}
}

func (s *SpecLinkInfoTestSuite) Test_get_links_from_spec_2(c *C) {
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{
			schema: testSpecLinkInfoBlueprintSchema2,
		}, nil)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	chains, err := specLinkInfo.Links(context.Background())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	// Using snapshots for chain links as is more convenient than building custom test functions
	// that traverse through and compare the produced chain with a "hand-crafted" expected chain.
	// Not using snapshots would require value comparisions as you traverse through the pointers in the chain.
	// However, it's really important that you take care when reviewing failing snapshot tests
	// and not just re-building the snapshots without checking the changes are correct!
	err = testhelpers.Snapshot(normaliseForSnapshot(chains, []string{}))
	if err != nil {
		c.Error(err)
	}
}

func (s *SpecLinkInfoTestSuite) Test_get_links_from_spec_for_a_blueprint_with_circular_soft_links(c *C) {
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{
			schema: testSpecLinkInfoBlueprintSchema5,
		}, nil)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	chains, err := specLinkInfo.Links(context.Background())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	// We can't use cupaloy.Snapshot to take snapshots for blueprints with circular soft links
	// as the link that appears at the start of the chain is not deterministic.
	err = circularLinksApproximateSnapshotSchema5(normaliseForSnapshot(chains, []string{}))
	if err != nil {
		c.Error(err)
	}
}

func (s *SpecLinkInfoTestSuite) Test_get_links_fails_when_circular_hard_links_are_discovered(c *C) {
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{
			schema: testSpecLinkInfoBlueprintSchema4,
		}, nil)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	_, err = specLinkInfo.Links(context.Background())
	if err == nil {
		c.Error("expected an error for circular hard links")
		c.FailNow()
	}
	linkError, isLinkError := err.(*LinkError)
	if !isLinkError {
		c.Error("expected error to be an instance of a LinkError")
		c.FailNow()
	}

	if linkError.ReasonCode != LinkErrorReasonCodeCircularLinks {
		c.Errorf(
			"expected link error reason code to be %s, found %s",
			LinkErrorReasonCodeCircularLinks,
			linkError.ReasonCode,
		)
		c.FailNow()
	}

	if len(linkError.ChildErrors) != 2 {
		c.Errorf("expected 2 circular hard link child errors, found %d", len(linkError.ChildErrors))
		c.FailNow()
	}

	for i := 0; i < 2; i += 1 {
		childLinkError, isChildErrorLinkError := linkError.ChildErrors[i].(*LinkError)
		if !isChildErrorLinkError {
			c.Errorf("expected child error %d to be an instance of a LinkError", i+1)
			c.FailNow()
		}

		if childLinkError.ReasonCode != LinkErrorReasonCodeCircularLink {
			c.Errorf(
				"expected child link error %d reason code to be %s, found %s",
				i+1,
				LinkErrorReasonCodeCircularLink,
				linkError.ReasonCode,
			)
			c.FailNow()
		}
	}
}

func (s *SpecLinkInfoTestSuite) Test_get_link_warnings_from_spec_for_a_blueprint_with_non_common_terminals(c *C) {
	// Re-use schema fixture 1 as it has the standalone IAM role
	// and the IAM role resource type is not a common terminal.
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{
			schema: testSpecLinkInfoBlueprintSchema1,
		}, nil)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	warnings, err := specLinkInfo.Warnings(context.Background())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}
	c.Assert(warnings, DeepEquals, []string{
		"resource \"statsAccumulatorFunction\" of type \"aws/lambda/function\" does not link out to " +
			"any other resources where in most use-cases a resource of type \"aws/lambda/function\" is expected to link to other resources",
		"resource \"standaloneRole\" of type \"aws/iam/role\" does not link out to any other resources " +
			"where in most use-cases a resource of type \"aws/iam/role\" is expected to link to other resources",
	})
}

// Acts as a normaliser as ordering does not matter in chain links but does matter when comparing
// snapshots!
// Also, to simplify the structure that is snapshotted we will convert linked from references to strings
// containing resource names. This will resolve false negatives in snapshot failure and make it easier to read
// the snapshots.
func normaliseForSnapshot(chains []*ChainLinkNode, ancestors []string) []*snapshotChainLinkNode {
	orderedChainLinkNodes := append([]*ChainLinkNode{}, chains...)
	sort.SliceStable(orderedChainLinkNodes, func(i, j int) bool {
		return orderedChainLinkNodes[i].ResourceName < orderedChainLinkNodes[j].ResourceName
	})
	ssChainLinkNodes := []*snapshotChainLinkNode{}
	for _, chainLinkNode := range orderedChainLinkNodes {

		ssChainLinkNode := &snapshotChainLinkNode{
			ResourceName:        chainLinkNode.ResourceName,
			Selectors:           chainLinkNode.Selectors,
			LinkImplementations: chainLinkNode.LinkImplementations,
			Resource:            chainLinkNode.Resource,
			Paths:               chainLinkNode.Paths,
		}
		sort.Strings(ssChainLinkNode.Paths)
		// Prevent infinite loops/stack overflows when normalising chains
		// with circular links.
		if !core.SliceContainsComparable(ancestors, chainLinkNode.ResourceName) {
			ssChainLinkNode.LinksTo = normaliseForSnapshot(chainLinkNode.LinksTo, append(ancestors, chainLinkNode.ResourceName))
		} else {
			// To avoid going in circles, we'll create copies of the linked to
			// items only containing resource names for the purpose of snapshots.
			ssChainLinkNode.LinksTo = createCycleStubsForSnapshot(chainLinkNode.LinksTo)
		}
		ssChainLinkNode.LinkedFrom = core.Map(
			chainLinkNode.LinkedFrom,
			func(linkedFrom *ChainLinkNode, _ int) string {
				return linkedFrom.ResourceName
			},
		)
		sort.Strings(ssChainLinkNode.LinkedFrom)
		for selectorKey, selectedResources := range chainLinkNode.Selectors {
			sort.Strings(selectedResources)
			chainLinkNode.Selectors[selectorKey] = selectedResources
		}
		ssChainLinkNodes = append(ssChainLinkNodes, ssChainLinkNode)
	}

	return ssChainLinkNodes
}

func createCycleStubsForSnapshot(links []*ChainLinkNode) []*snapshotChainLinkNode {
	cycleStubs := []*snapshotChainLinkNode{}
	for _, link := range links {
		cycleStubs = append(cycleStubs, &snapshotChainLinkNode{
			ResourceName: fmt.Sprintf("%s-cycleStub", link.ResourceName),
		})
	}
	return cycleStubs
}

func circularLinksApproximateSnapshotSchema5(ssChainLinkNodes []*snapshotChainLinkNode) error {
	possiblePaths := map[string]interface{}{
		"statsRetrieverFunction": map[string]interface{}{
			"lambdaExecutionRole": map[string]interface{}{
				"statsRetrieverFunction": "lambdaExecutionRole-cycleStub",
			},
		},
		"lambdaExecutionRole": map[string]interface{}{
			"statsRetrieverFunction": map[string]interface{}{
				"lambdaExecutionRole": "statsRetrieverFunction-cycleStub",
			},
		},
		"ordersTable": map[string]interface{}{
			"ordersStream": map[string]interface{}{
				"statsAccumulatorFunction": map[string]interface{}{
					"ordersTable": "ordersStream-cycleStub",
				},
			},
		},
		"ordersStream": map[string]interface{}{
			"statsAccumulatorFunction": map[string]interface{}{
				"ordersTable": map[string]interface{}{
					"ordersStream": "statsAccumulatorFunction-cycleStub",
				},
			},
		},
		"statsAccumulatorFunction": map[string]interface{}{
			"ordersTable": map[string]interface{}{
				"ordersStream": map[string]interface{}{
					"statsAccumulatorFunction": "ordersTable-cycleStub",
				},
			},
		},
	}

	if len(ssChainLinkNodes) != 2 {
		return fmt.Errorf("expected 2 top-level chains, found %d", len(ssChainLinkNodes))
	}

	for _, ssChainLinkNode := range ssChainLinkNodes {
		ChainLinkNodeError := followPaths(ssChainLinkNode, possiblePaths)
		if ChainLinkNodeError != nil {
			return ChainLinkNodeError
		}
	}

	return nil
}

func followPaths(ssChainLinkNode *snapshotChainLinkNode, possiblePaths interface{}) error {
	possiblePathsStr, isTerminal := possiblePaths.(string)
	if isTerminal {
		if ssChainLinkNode.ResourceName != possiblePathsStr {
			return fmt.Errorf(
				"%s is not the expected cycle stub \"%s\"",
				ssChainLinkNode.ResourceName,
				possiblePathsStr,
			)
		}
		return nil
	}

	mapping, isMap := possiblePaths.(map[string]interface{})
	if isMap {
		nextLevelPath, hasNextLevelPath := mapping[ssChainLinkNode.ResourceName]
		if !hasNextLevelPath {
			return fmt.Errorf("%s is not an expected resource in possible paths", ssChainLinkNode.ResourceName)
		}
		return followPaths(ssChainLinkNode.LinksTo[0], nextLevelPath)
	}

	return errors.New("unexpected possiblePaths type provided")
}

// Lots of links.
var testSpecLinkInfoBlueprintSchema1 = &schema.Blueprint{
	Resources: &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"orderApi": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/apigateway/api"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "orderApi",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"app": "orderApi",
						},
					},
				},
			},
			"orderQueue": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/sqs/queue"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "orderWorkflow",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"app": "orderWorkflow",
						},
					},
				},
			},
			"processOrdersFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "orderWorkflow",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"createOrderFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "orderApi",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"getOrdersFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "orderApi",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"ordersTable": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"ordersStream": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/stream"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
			},
			"statsAccumulatorFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
			},
			"standaloneRole": {
				Type:     &schema.ResourceTypeWrapper{Value: "aws/iam/role"},
				Metadata: &schema.Metadata{},
			},
		},
	},
}

// No links.
var testSpecLinkInfoBlueprintSchema2 = &schema.Blueprint{
	Resources: &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"exchangeRateFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "exchangeRates",
						},
					},
				},
			},
			"refreshExchangeRatesFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"app": "exchangeRates",
						},
					},
				},
			},
			"standaloneRole2": {
				Type:     &schema.ResourceTypeWrapper{Value: "aws/iam/role"},
				Metadata: &schema.Metadata{},
			},
		},
	},
}

// Circular hard links.
// As soon as the first circular hard link is found in a chain, no further links
// in that chain are discovered until the first one is fixed!
// The test cases to capture multiple circular hard links here are for independent chains.
// Soft links break hard link cycles as soft links represent dependencies where one resource
// does not need to exist in order to deploy/create the other.
// For this blueprint, an error should be returned.
var testSpecLinkInfoBlueprintSchema4 = &schema.Blueprint{
	Resources: &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"ordersTable": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"ordersStream": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/stream"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
			},
			"statsAccumulatorFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
				// Indirect hard circular link back to orders table.
				// (In reality the relationship between an lambda and a DynamoDB table is
				// not hard but for the sake of this test case it is)
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"statsRetrieverFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
			},
			"lambdaExecutionRole": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/iam/role"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
				// Direct hard circular link between statsRetrieverFunction
				// and lambdaExecutionRole.
				// (In reality the relationship between an IAM role and a lambda is
				// not hard but for the sake of this test case it is)
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
			},
		},
	},
}

// Circular links with soft links breaking hard link cycles.
// This should not cause an error!
var testSpecLinkInfoBlueprintSchema5 = &schema.Blueprint{
	Resources: &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"ordersTable": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"ordersStream": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/stream"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
			},
			"statsAccumulatorFunction": {
				// Represents a theoretical stratos abstraction
				// of an aws lambda function.
				Type: &schema.ResourceTypeWrapper{Value: "stratosaws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"workflow": "orderStats",
						},
					},
				},
				// Indirect circular link back to orders table.
				// The soft link between "stratosaws/lambda/function"
				// and "aws/dynamodb/table" breaks the hard link cycle.
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"system": "orders",
						},
					},
				},
			},
			"statsRetrieverFunction": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
			},
			"lambdaExecutionRole": {
				// Represents a theoretical stratos abstraction
				// of an aws iam role.
				Type: &schema.ResourceTypeWrapper{Value: "stratosaws/iam/role"},
				Metadata: &schema.Metadata{
					Labels: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
				// Direct soft circular link between statsRetrieverFunction
				// and lambdaExecutionRole.
				LinkSelector: &schema.LinkSelector{
					ByLabel: &schema.StringMap{
						Values: map[string]string{
							"purpose": "retrieveStats",
						},
					},
				},
			},
		},
	},
}
