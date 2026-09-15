package drift

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

const (
	orderingInstanceID = "link-drift-ordering-instance"
	orderingLinkID     = "link-drift-ordering-link"
	orderingFunctionID = "ordering-function-id"
)

type LinkDriftOrderingTestSuite struct {
	suite.Suite
}

// A field whose order is not significant must not be reported as drifted when the
// provider returns the same values in a different order.
func (s *LinkDriftOrderingTestSuite) Test_ignores_the_order_of_a_field_declared_unordered() {
	driftState := s.checkDriftFor(
		// What the link recorded writing.
		core.MappingNodeFields(
			"subnetIds", stringItems("subnet-a", "subnet-b", "subnet-c"),
		),
		// What the resource reports now: the same subnets, reordered by the provider.
		core.MappingNodeFields(
			"subnetIds", stringItems("subnet-b", "subnet-a", "subnet-c"),
			"securityGroupIds", stringItems("sg-1"),
		),
		map[string]string{
			"orderingFunction::spec.subnetIds": "orderingFunction.subnetIds",
		},
	)

	s.Assert().Nil(
		driftState,
		"the same values in a different order were reported as drift, so this "+
			"resource can never be staged against again",
	)
}

// The order of a field that has not been declared unordered still matters, so the
// comparison is not simply weakened for every array.
func (s *LinkDriftOrderingTestSuite) Test_still_reports_a_reordered_field_that_is_ordered() {
	driftState := s.checkDriftFor(
		core.MappingNodeFields(
			"layerArns", stringItems("layer-a", "layer-b"),
		),
		core.MappingNodeFields(
			"layerArns", stringItems("layer-b", "layer-a"),
		),
		map[string]string{
			"orderingFunction::spec.layerArns": "orderingFunction.layerArns",
		},
	)

	s.Require().NotNil(
		driftState,
		"a reordered field whose order is significant was not reported as drift",
	)
	s.Require().NotNil(driftState.ResourceADrift)
	s.Require().Len(driftState.ResourceADrift.MappedFieldChanges, 1)
	s.Assert().Equal(
		"spec.layerArns",
		driftState.ResourceADrift.MappedFieldChanges[0].ResourceFieldPath,
	)
}

// A field declared unordered whose values genuinely differ is still drift, so ignoring
// order does not amount to ignoring the field.
func (s *LinkDriftOrderingTestSuite) Test_reports_an_unordered_field_whose_values_differ() {
	driftState := s.checkDriftFor(
		core.MappingNodeFields(
			"subnetIds", stringItems("subnet-a", "subnet-b"),
		),
		core.MappingNodeFields(
			"subnetIds", stringItems("subnet-b", "subnet-z"),
		),
		map[string]string{
			"orderingFunction::spec.subnetIds": "orderingFunction.subnetIds",
		},
	)

	s.Require().NotNil(
		driftState,
		"a field whose values changed was not reported because its order is ignored",
	)
}

// A field addressed through an array selector must be compared through its schema too.
func (s *LinkDriftOrderingTestSuite) Test_ignores_the_order_of_an_unordered_field_reached_through_a_selector() {
	driftState := s.checkDriftFor(
		core.MappingNodeFields(
			"permission", statementNode("s1", "ec2:CreateNetworkInterface", "ec2:DescribeSubnets"),
		),
		policiesExternalState(
			statementNode("s1", "ec2:DescribeSubnets", "ec2:CreateNetworkInterface"),
		),
		map[string]string{
			selectorFieldPath: "orderingFunction.permission",
		},
	)

	s.Assert().Nil(
		driftState,
		"a reordered action list inside a statement selected by sid was reported as drift",
	)
}

// The same path, with a value that genuinely differs, is still drift.
func (s *LinkDriftOrderingTestSuite) Test_reports_a_selector_reached_field_whose_values_differ() {
	driftState := s.checkDriftFor(
		core.MappingNodeFields(
			"permission", statementNode("s1", "ec2:CreateNetworkInterface", "ec2:DescribeSubnets"),
		),
		policiesExternalState(
			statementNode("s1", "ec2:DescribeSubnets", "ec2:DeleteNetworkInterface"),
		),
		map[string]string{
			selectorFieldPath: "orderingFunction.permission",
		},
	)

	s.Require().NotNil(
		driftState,
		"a changed action was not reported because the order of the list is ignored",
	)
}

func (s *LinkDriftOrderingTestSuite) checkDriftFor(
	linkData *core.MappingNode,
	externalState *core.MappingNode,
	mappings map[string]string,
) *state.LinkDriftState {
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:   orderingInstanceID,
			InstanceName: orderingInstanceID,
			Status:       core.InstanceStatusDeployed,
			ResourceIDs:  map[string]string{"orderingFunction": orderingFunctionID},
			Resources: map[string]*state.ResourceState{
				orderingFunctionID: {
					ResourceID:    orderingFunctionID,
					Name:          "orderingFunction",
					Type:          "aws/lambda/function",
					InstanceID:    orderingInstanceID,
					Status:        core.ResourceStatusCreated,
					PreciseStatus: core.PreciseResourceStatusCreated,
					SpecData:      core.MappingNodeFields(),
				},
			},
			Links: map[string]*state.LinkState{
				"orderingFunction::orderingTarget": {
					LinkID:               orderingLinkID,
					Name:                 "orderingFunction::orderingTarget",
					InstanceID:           orderingInstanceID,
					Status:               core.LinkStatusCreated,
					Data:                 map[string]*core.MappingNode{"orderingFunction": linkData},
					ResourceDataMappings: mappings,
				},
			},
		},
	))

	checker := NewDefaultChecker(
		stateContainer,
		map[string]provider.Provider{
			"aws": &internal.ProviderMock{
				NamespaceValue: "aws",
				Resources: map[string]provider.Resource{
					"aws/lambda/function": &orderedFieldsResource{
						LambdaFunctionResource: &internal.LambdaFunctionResource{
							CurrentDestroyAttempts:         map[string]int{},
							CurrentDeployAttemps:           map[string]int{},
							CurrentGetExternalStateAttemps: map[string]int{},
							StabiliseResourceIDs:           map[string]*internal.StubResourceStabilisationConfig{},
							CurrentStabiliseCalls:          map[string]int{},
							ExternalState:                  externalState,
						},
					},
				},
				Links:               map[string]provider.Link{},
				CustomVariableTypes: map[string]provider.CustomVariableType{},
				DataSources:         map[string]provider.DataSource{},
			},
		},
		changes.NewDefaultResourceChangeGenerator(),
		core.SystemClock{},
		core.NewNopLogger(),
	)

	driftState, err := checker.CheckLinkDrift(
		context.Background(),
		orderingInstanceID,
		orderingLinkID,
		createParams(),
		nil,
	)
	s.Require().NoError(err)

	return driftState
}

// A function with one field whose order carries no meaning and one where it does, so a
// single comparison cannot satisfy both by being weakened.
type orderedFieldsResource struct {
	*internal.LambdaFunctionResource
}

func (r *orderedFieldsResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"subnetIds": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						IgnoreItemOrder: true,
					},
					"securityGroupIds": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						IgnoreItemOrder: true,
					},
					"layerArns": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
					},
					// A shared execution role's policies, which links write one statement
					// into and address by the sid they chose.
					"policies": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeObject,
							Attributes: map[string]*provider.ResourceDefinitionsSchema{
								"policyName": {
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
								"policyDocument": {
									Type: provider.ResourceDefinitionsSchemaTypeObject,
									Attributes: map[string]*provider.ResourceDefinitionsSchema{
										"statement": {
											Type: provider.ResourceDefinitionsSchemaTypeArray,
											Items: &provider.ResourceDefinitionsSchema{
												Type: provider.ResourceDefinitionsSchemaTypeObject,
												Attributes: map[string]*provider.ResourceDefinitionsSchema{
													"sid": {
														Type: provider.ResourceDefinitionsSchemaTypeString,
													},
													"action": {
														Type: provider.ResourceDefinitionsSchemaTypeArray,
														Items: &provider.ResourceDefinitionsSchema{
															Type: provider.ResourceDefinitionsSchemaTypeString,
														},
														IgnoreItemOrder: true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

const selectorFieldPath = `orderingFunction::spec.policies[@.policyName="link-access"]` +
	`.policyDocument.statement[@.sid="s1"]`

func statementNode(sid string, actions ...string) *core.MappingNode {
	return core.MappingNodeFields(
		"sid", core.MappingNodeFromString(sid),
		"action", stringItems(actions...),
	)
}

func policiesExternalState(statements ...*core.MappingNode) *core.MappingNode {
	return core.MappingNodeFields(
		"policies", &core.MappingNode{
			Items: []*core.MappingNode{
				core.MappingNodeFields(
					"policyName", core.MappingNodeFromString("link-access"),
					"policyDocument", core.MappingNodeFields(
						"statement", &core.MappingNode{Items: statements},
					),
				),
			},
		},
	)
}

func stringItems(values ...string) *core.MappingNode {
	items := make([]*core.MappingNode, 0, len(values))
	for _, value := range values {
		items = append(items, core.MappingNodeFromString(value))
	}

	return &core.MappingNode{Items: items}
}

func TestLinkDriftOrderingTestSuite(t *testing.T) {
	suite.Run(t, new(LinkDriftOrderingTestSuite))
}
