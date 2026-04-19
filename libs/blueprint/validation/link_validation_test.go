package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type LinkCardinalityValidationTestSuite struct {
	suite.Suite
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_no_diagnostics_when_no_cardinality_rules_are_defined() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, map[string]provider.LinkGetCardinalityOutput{})
	s.Assert().Empty(diagnostics)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_no_diagnostics_when_cardinality_is_satisfied() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "usersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			CardinalityA: provider.LinkCardinality{Min: 1, Max: 3},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 2},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Empty(diagnostics)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_no_diagnostics_when_cardinality_is_unconstrained() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "usersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			// 0 means unconstrained on both sides.
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 0},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Empty(diagnostics)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_error_when_source_exceeds_max_outgoing_links() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "usersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 2},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(
		diagnostics[0].Message,
		"has 3 outgoing links",
	)
	s.Assert().Contains(
		diagnostics[0].Message,
		"exceeding the maximum of 2",
	)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_error_when_source_has_fewer_than_min_outgoing_links() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			CardinalityA: provider.LinkCardinality{Min: 2, Max: 0},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(
		diagnostics[0].Message,
		"has 1 outgoing links",
	)
	s.Assert().Contains(
		diagnostics[0].Message,
		"below the minimum of 2",
	)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_error_when_target_exceeds_max_incoming_links() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "inventoryFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "reportFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 0},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 2},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(
		diagnostics[0].Message,
		"has 3 incoming links",
	)
	s.Assert().Contains(
		diagnostics[0].Message,
		"exceeding the maximum of 2",
	)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_error_when_target_has_fewer_than_min_incoming_links() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 0},
			CardinalityB: provider.LinkCardinality{Min: 2, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(
		diagnostics[0].Message,
		"has 1 incoming links",
	)
	s.Assert().Contains(
		diagnostics[0].Message,
		"below the minimum of 2",
	)
}

func (s *LinkCardinalityValidationTestSuite) Test_reports_errors_for_both_sides_when_both_violate_constraints() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "usersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			// Source exceeds max (3 > 2).
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 2},
			// Each target has only 1 incoming but needs at least 2.
			CardinalityB: provider.LinkCardinality{Min: 2, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	// 1 source violation + 3 target violations (one per unique target resource).
	s.Assert().Len(diagnostics, 4)
}

func (s *LinkCardinalityValidationTestSuite) Test_validates_independently_for_different_link_types() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "ordersQueue",
				SourceType: "aws/lambda/function",
				TargetType: "aws/sqs/queue",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			// 1 outgoing link to dynamodb — within bounds.
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 2},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
		core.LinkType("aws/lambda/function", "aws/sqs/queue"): {
			// 1 outgoing link to sqs — within bounds.
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	s.Assert().Empty(diagnostics)
}

func (s *LinkCardinalityValidationTestSuite) Test_does_not_produce_duplicate_diagnostics_for_same_resource() {
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "usersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	rules := map[string]provider.LinkGetCardinalityOutput{
		core.LinkType("aws/lambda/function", "aws/dynamodb/table"): {
			// Source "orderFunction" has 3 outgoing, exceeding max of 1.
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	diagnostics := ValidateLinkCardinality(linkGraph, rules)
	// Only 1 diagnostic for orderFunction, not 3.
	s.Assert().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "orderFunction")
}

func TestLinkCardinalityValidationTestSuite(t *testing.T) {
	suite.Run(t, new(LinkCardinalityValidationTestSuite))
}

// testDeclaredLinkGraph is a minimal stub of linktypes.DeclaredLinkGraph
// for unit testing cardinality validation in isolation.
type testDeclaredLinkGraph struct {
	edges []*linktypes.ResolvedLink
}

func (g *testDeclaredLinkGraph) Edges() []*linktypes.ResolvedLink {
	return g.edges
}

func (g *testDeclaredLinkGraph) EdgesFrom(resourceName string) []*linktypes.ResolvedLink {
	var result []*linktypes.ResolvedLink
	for _, e := range g.edges {
		if e.Source == resourceName {
			result = append(result, e)
		}
	}
	return result
}

func (g *testDeclaredLinkGraph) EdgesTo(resourceName string) []*linktypes.ResolvedLink {
	var result []*linktypes.ResolvedLink
	for _, e := range g.edges {
		if e.Target == resourceName {
			result = append(result, e)
		}
	}
	return result
}

func (g *testDeclaredLinkGraph) Resource(name string) (*schema.Resource, linktypes.ResourceClass, bool) {
	return nil, "", false
}

// ----------------------------------------
// ValidateLinkConstraints test suite
// ----------------------------------------

type LinkConstraintsValidationTestSuite struct {
	suite.Suite
}

func TestLinkConstraintsValidationTestSuite(t *testing.T) {
	suite.Run(t, new(LinkConstraintsValidationTestSuite))
}

func (s *LinkConstraintsValidationTestSuite) Test_reports_no_diagnostics_when_all_constraints_are_satisfied() {
	linkImpl := &testConfigurableLink{
		linkType: "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 2},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		linkGraph,
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
}

func (s *LinkConstraintsValidationTestSuite) Test_reports_cardinality_violation_diagnostics() {
	linkImpl := &testConfigurableLink{
		linkType: "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable":   linkImpl,
			"productsTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeC := createTestChainLinkNode(
		"productsTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB, nodeC}

	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
		"productsTable": nodeC.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		linkGraph,
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "exceeding the maximum of 1")
}

func (s *LinkConstraintsValidationTestSuite) Test_reports_custom_validation_diagnostics() {
	linkImpl := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
		validateOutput: &provider.LinkValidateOutput{
			Diagnostics: []*core.Diagnostic{
				{
					Level:   core.DiagnosticLevelError,
					Message: "handler must be set when linking to a DynamoDB table",
				},
			},
		},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Len(diagnostics, 1)
	s.Assert().Equal(
		"handler must be set when linking to a DynamoDB table",
		diagnostics[0].Message,
	)
}

func (s *LinkConstraintsValidationTestSuite) Test_combines_cardinality_and_custom_validation_diagnostics() {
	linkImpl := &testConfigurableLink{
		linkType: "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
		validateOutput: &provider.LinkValidateOutput{
			Diagnostics: []*core.Diagnostic{
				{
					Level:   core.DiagnosticLevelWarning,
					Message: "custom warning",
				},
			},
		},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable":   linkImpl,
			"productsTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeC := createTestChainLinkNode(
		"productsTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB, nodeC}

	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "orderFunction",
				Target:     "productsTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
		},
	}
	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
		"productsTable": nodeC.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		linkGraph,
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	// 1 cardinality violation + 2 custom diagnostics (one per link instance).
	s.Assert().Len(diagnostics, 3)
}

func (s *LinkConstraintsValidationTestSuite) Test_passes_resource_specs_to_custom_validation() {
	specA := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"handler": core.MappingNodeFromString("src/orders.handler"),
		},
	}
	specB := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"tableName": core.MappingNodeFromString("Orders"),
		},
	}
	linkImpl := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		specA,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		specB,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
	})

	_, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Require().NotNil(linkImpl.capturedInput)
	s.Assert().Equal("orderFunction", linkImpl.capturedInput.ResourceAName)
	s.Assert().Equal("ordersTable", linkImpl.capturedInput.ResourceBName)
	s.Assert().Equal("aws/lambda/function", linkImpl.capturedInput.ResourceAType)
	s.Assert().Equal("aws/dynamodb/table", linkImpl.capturedInput.ResourceBType)
	s.Assert().Equal(specA, linkImpl.capturedInput.ResourceASpec)
	s.Assert().Equal(specB, linkImpl.capturedInput.ResourceBSpec)
}

func (s *LinkConstraintsValidationTestSuite) Test_passes_static_annotations_to_custom_validation() {
	linkImpl := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	accessType := "readOnly"
	nodeA.Resource.Metadata = &schema.Metadata{
		Annotations: &schema.StringOrSubstitutionsMap{
			Values: map[string]*substitutions.StringOrSubstitutions{
				"aws.lambda.dynamodb.accessType": {
					Values: []*substitutions.StringOrSubstitution{
						{StringValue: &accessType},
					},
				},
			},
		},
	}
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
	})

	_, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Require().NotNil(linkImpl.capturedInput)
	s.Require().Contains(linkImpl.capturedInput.Annotations, "aws.lambda.dynamodb.accessType")
	s.Assert().Equal(
		"readOnly",
		*linkImpl.capturedInput.Annotations["aws.lambda.dynamodb.accessType"].StringValue,
	)
}

func (s *LinkConstraintsValidationTestSuite) Test_traverses_chains_with_soft_link_cycle_without_error() {
	linkAB := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
	}
	linkBC := &testConfigurableLink{
		linkType:    "aws/dynamodb/table::aws/dynamodb/stream",
		cardinality: &provider.LinkGetCardinalityOutput{},
	}
	linkCA := &testConfigurableLink{
		linkType:    "aws/dynamodb/stream::aws/lambda/function",
		cardinality: &provider.LinkGetCardinalityOutput{},
	}

	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkAB,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		map[string]provider.Link{
			"ordersStream": linkBC,
		},
	)
	nodeC := createTestChainLinkNode(
		"ordersStream",
		"aws/dynamodb/stream",
		nil,
		map[string]provider.Link{
			"orderFunction": linkCA,
		},
	)
	// Create cycle: A→B→C→A
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}
	nodeB.LinksTo = []*links.ChainLinkNode{nodeC}
	nodeC.LinksTo = []*links.ChainLinkNode{nodeA}

	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
		"ordersStream":  nodeC.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
	// Each link implementation should have been called exactly once.
	s.Assert().NotNil(linkAB.capturedInput)
	s.Assert().NotNil(linkBC.capturedInput)
	s.Assert().NotNil(linkCA.capturedInput)
}

func (s *LinkConstraintsValidationTestSuite) Test_skips_custom_validation_when_resource_not_in_spec() {
	linkImpl := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
		validateOutput: &provider.LinkValidateOutput{
			Diagnostics: []*core.Diagnostic{
				{
					Level:   core.DiagnosticLevelError,
					Message: "should not appear",
				},
			},
		},
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	// Only include resource A in the spec, not B.
	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
	// ValidateLink should not have been called.
	s.Assert().Nil(linkImpl.capturedInput)
}

func (s *LinkConstraintsValidationTestSuite) Test_propagates_error_from_get_cardinality() {
	linkImpl := &testConfigurableLink{
		linkType:       "aws/lambda/function::aws/dynamodb/table",
		cardinalityErr: errors.New("cardinality provider error"),
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	_, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		createTestBlueprintSpec(nil),
		createParams(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "cardinality provider error")
}

func (s *LinkConstraintsValidationTestSuite) Test_propagates_error_from_validate_link() {
	linkImpl := &testConfigurableLink{
		linkType:    "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{},
		validateErr: errors.New("custom validation error"),
	}
	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": linkImpl,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}

	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
	})

	_, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		&testDeclaredLinkGraph{},
		spec,
		createParams(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "custom validation error")
}

func (s *LinkConstraintsValidationTestSuite) Test_extracts_cardinality_rules_from_nested_chain() {
	lambdaTableLink := &testConfigurableLink{
		linkType: "aws/lambda/function::aws/dynamodb/table",
		cardinality: &provider.LinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}
	tableStreamLink := &testConfigurableLink{
		linkType: "aws/dynamodb/table::aws/dynamodb/stream",
		cardinality: &provider.LinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		},
	}

	nodeA := createTestChainLinkNode(
		"orderFunction",
		"aws/lambda/function",
		nil,
		map[string]provider.Link{
			"ordersTable": lambdaTableLink,
		},
	)
	nodeB := createTestChainLinkNode(
		"ordersTable",
		"aws/dynamodb/table",
		nil,
		map[string]provider.Link{
			"ordersStream": tableStreamLink,
		},
	)
	nodeC := createTestChainLinkNode(
		"ordersStream",
		"aws/dynamodb/stream",
		nil,
		nil,
	)
	nodeA.LinksTo = []*links.ChainLinkNode{nodeB}
	nodeB.LinksTo = []*links.ChainLinkNode{nodeC}

	// Both link types present in the graph, both within bounds.
	linkGraph := &testDeclaredLinkGraph{
		edges: []*linktypes.ResolvedLink{
			{
				Source:     "orderFunction",
				Target:     "ordersTable",
				SourceType: "aws/lambda/function",
				TargetType: "aws/dynamodb/table",
			},
			{
				Source:     "ordersTable",
				Target:     "ordersStream",
				SourceType: "aws/dynamodb/table",
				TargetType: "aws/dynamodb/stream",
			},
		},
	}
	spec := createTestBlueprintSpec(map[string]*schema.Resource{
		"orderFunction": nodeA.Resource,
		"ordersTable":   nodeB.Resource,
		"ordersStream":  nodeC.Resource,
	})

	diagnostics, err := ValidateLinkConstraints(
		context.Background(),
		[]*links.ChainLinkNode{nodeA},
		linkGraph,
		spec,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
}

func createTestChainLinkNode(
	name string,
	resourceType string,
	spec *core.MappingNode,
	linkImpls map[string]provider.Link,
) *links.ChainLinkNode {
	if linkImpls == nil {
		linkImpls = map[string]provider.Link{}
	}
	return &links.ChainLinkNode{
		ResourceName: name,
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{Value: resourceType},
			Spec: spec,
		},
		LinkImplementations: linkImpls,
		Selectors:           map[string][]string{},
		LinksTo:             []*links.ChainLinkNode{},
		LinkedFrom:          []*links.ChainLinkNode{},
		Paths:               []string{},
	}
}

func createTestBlueprintSpec(resources map[string]*schema.Resource) speccore.BlueprintSpec {
	if resources == nil {
		resources = map[string]*schema.Resource{}
	}
	return speccore.BlueprintSpecFromSchema(&schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: resources,
		},
	})
}
