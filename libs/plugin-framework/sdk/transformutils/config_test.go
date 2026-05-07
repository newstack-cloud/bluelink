package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

const testConfigTransformerID = "celerity-test-2026"

type ResourceConfigVariableTestSuite struct {
	suite.Suite
}

func (s *ResourceConfigVariableTestSuite) Test_returns_resource_scoped_value_when_both_scopes_are_set() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.dynamodb.ordersTable.billingMode": core.ScalarFromString("PROVISIONED"),
			"aws.dynamodb.billingMode":             core.ScalarFromString("PAY_PER_REQUEST"),
		},
	)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().NoError(err)
	s.Require().NotNil(value)
	s.Assert().Equal("PROVISIONED", core.StringValueFromScalar(value))
}

func (s *ResourceConfigVariableTestSuite) Test_returns_resource_scoped_value_when_only_resource_scope_is_set() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.dynamodb.ordersTable.billingMode": core.ScalarFromString("PROVISIONED"),
		},
	)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().NoError(err)
	s.Require().NotNil(value)
	s.Assert().Equal("PROVISIONED", core.StringValueFromScalar(value))
}

func (s *ResourceConfigVariableTestSuite) Test_falls_back_to_service_scoped_value_when_resource_scope_is_absent() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.dynamodb.billingMode": core.ScalarFromString("PAY_PER_REQUEST"),
		},
	)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().NoError(err)
	s.Require().NotNil(value)
	s.Assert().Equal("PAY_PER_REQUEST", core.StringValueFromScalar(value))
}

func (s *ResourceConfigVariableTestSuite) Test_returns_error_when_neither_scope_is_set() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{},
	)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().Error(err)
	s.Assert().Nil(value)
	s.Assert().Contains(err.Error(), "config variable not found")
	s.Assert().Contains(err.Error(), "aws.dynamodb.ordersTable.billingMode")
}

func (s *ResourceConfigVariableTestSuite) Test_does_not_match_a_different_resource_name_with_the_same_key() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.dynamodb.otherTable.billingMode": core.ScalarFromString("PROVISIONED"),
		},
	)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().Error(err)
	s.Assert().Nil(value)
}

func (s *ResourceConfigVariableTestSuite) Test_only_reads_from_the_current_transformer_namespace() {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{
			"other-transformer": {
				"aws.dynamodb.ordersTable.billingMode": core.ScalarFromString("PROVISIONED"),
			},
		},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	transformCtx := transform.NewTransformerContextFromParams(testConfigTransformerID, params)

	value, err := ResourceConfigVariable(transformCtx, "aws.dynamodb", "ordersTable", "billingMode")

	s.Require().Error(err)
	s.Assert().Nil(value)
}

func TestResourceConfigVariableTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceConfigVariableTestSuite))
}

func newTransformerContextWithConfig(
	namespace string,
	config map[string]*core.ScalarValue,
) transform.Context {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{
			namespace: config,
		},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	return transform.NewTransformerContextFromParams(namespace, params)
}
