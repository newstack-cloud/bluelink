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

type ResourceConfigVariableMapTestSuite struct {
	suite.Suite
}

func (s *ResourceConfigVariableMapTestSuite) Test_returns_resource_scoped_map_when_both_scopes_are_set() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.config.primaryConfig.regionKMSKeys.us-east-1": core.ScalarFromString("arn:aws:kms:us-east-1:resource"),
			"aws.config.primaryConfig.regionKMSKeys.eu-west-1": core.ScalarFromString("arn:aws:kms:eu-west-1:resource"),
			"aws.config.regionKMSKeys.us-east-1":               core.ScalarFromString("arn:aws:kms:us-east-1:service"),
			"aws.config.regionKMSKeys.eu-west-1":               core.ScalarFromString("arn:aws:kms:eu-west-1:service"),
		},
	)

	subMap, err := ResourceConfigVariableMap(transformCtx, "aws.config", "primaryConfig", "regionKMSKeys")

	s.Require().NoError(err)
	s.Require().Len(subMap, 2)
	s.Assert().Equal("arn:aws:kms:us-east-1:resource", core.StringValueFromScalar(subMap["us-east-1"]))
	s.Assert().Equal("arn:aws:kms:eu-west-1:resource", core.StringValueFromScalar(subMap["eu-west-1"]))
}

func (s *ResourceConfigVariableMapTestSuite) Test_falls_back_to_service_scoped_map_when_resource_scope_is_absent() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.config.regionKMSKeys.us-east-1": core.ScalarFromString("arn:aws:kms:us-east-1:service"),
			"aws.config.regionKMSKeys.eu-west-1": core.ScalarFromString("arn:aws:kms:eu-west-1:service"),
		},
	)

	subMap, err := ResourceConfigVariableMap(transformCtx, "aws.config", "primaryConfig", "regionKMSKeys")

	s.Require().NoError(err)
	s.Require().Len(subMap, 2)
	s.Assert().Equal("arn:aws:kms:us-east-1:service", core.StringValueFromScalar(subMap["us-east-1"]))
	s.Assert().Equal("arn:aws:kms:eu-west-1:service", core.StringValueFromScalar(subMap["eu-west-1"]))
}

func (s *ResourceConfigVariableMapTestSuite) Test_returns_error_when_neither_scope_has_entries() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{},
	)

	subMap, err := ResourceConfigVariableMap(transformCtx, "aws.config", "primaryConfig", "regionKMSKeys")

	s.Require().Error(err)
	s.Assert().Nil(subMap)
	s.Assert().Contains(err.Error(), "config variable map not found")
	s.Assert().Contains(err.Error(), "aws.config.primaryConfig.regionKMSKeys.")
}

func (s *ResourceConfigVariableMapTestSuite) Test_ignores_deeper_nested_keys_under_the_same_prefix() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.config.primaryConfig.regionKMSKeys.us-east-1":          core.ScalarFromString("arn:aws:kms:us-east-1:resource"),
			"aws.config.primaryConfig.regionKMSKeys.us-east-1.unwanted": core.ScalarFromString("noise"),
		},
	)

	subMap, err := ResourceConfigVariableMap(transformCtx, "aws.config", "primaryConfig", "regionKMSKeys")

	s.Require().NoError(err)
	s.Require().Len(subMap, 1)
	s.Assert().Equal("arn:aws:kms:us-east-1:resource", core.StringValueFromScalar(subMap["us-east-1"]))
}

func (s *ResourceConfigVariableMapTestSuite) Test_does_not_match_a_different_resource_name_with_the_same_key() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.config.otherConfig.regionKMSKeys.us-east-1": core.ScalarFromString("arn:aws:kms:us-east-1:other"),
		},
	)

	subMap, err := ResourceConfigVariableMap(transformCtx, "aws.config", "primaryConfig", "regionKMSKeys")

	s.Require().Error(err)
	s.Assert().Nil(subMap)
}

func TestResourceConfigVariableMapTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceConfigVariableMapTestSuite))
}

type ResourceConfigVariableSeqTestSuite struct {
	suite.Suite
}

func (s *ResourceConfigVariableSeqTestSuite) Test_returns_resource_scoped_seq_when_both_scopes_are_set() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.sns.ordersTopic.statusLogging.0.protocol":                core.ScalarFromString("https"),
			"aws.sns.ordersTopic.statusLogging.0.failureFeedbackRoleArn":  core.ScalarFromString("arn:resource:0"),
			"aws.sns.ordersTopic.statusLogging.1.protocol":                core.ScalarFromString("lambda"),
			"aws.sns.ordersTopic.statusLogging.1.failureFeedbackRoleArn":  core.ScalarFromString("arn:resource:1"),
			"aws.sns.statusLogging.0.protocol":                            core.ScalarFromString("sqs"),
			"aws.sns.statusLogging.0.failureFeedbackRoleArn":              core.ScalarFromString("arn:service:0"),
		},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().NoError(err)
	s.Require().Len(seq, 2)
	s.Assert().Equal("https", core.StringValueFromScalar(seq[0]["protocol"]))
	s.Assert().Equal("arn:resource:0", core.StringValueFromScalar(seq[0]["failureFeedbackRoleArn"]))
	s.Assert().Equal("lambda", core.StringValueFromScalar(seq[1]["protocol"]))
	s.Assert().Equal("arn:resource:1", core.StringValueFromScalar(seq[1]["failureFeedbackRoleArn"]))
}

func (s *ResourceConfigVariableSeqTestSuite) Test_falls_back_to_service_scoped_seq_when_resource_scope_is_absent() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.sns.statusLogging.0.protocol":               core.ScalarFromString("sqs"),
			"aws.sns.statusLogging.0.failureFeedbackRoleArn": core.ScalarFromString("arn:service:0"),
		},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().NoError(err)
	s.Require().Len(seq, 1)
	s.Assert().Equal("sqs", core.StringValueFromScalar(seq[0]["protocol"]))
	s.Assert().Equal("arn:service:0", core.StringValueFromScalar(seq[0]["failureFeedbackRoleArn"]))
}

func (s *ResourceConfigVariableSeqTestSuite) Test_stops_at_the_first_index_gap() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.sns.ordersTopic.statusLogging.0.protocol": core.ScalarFromString("https"),
			"aws.sns.ordersTopic.statusLogging.2.protocol": core.ScalarFromString("lambda"),
		},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().NoError(err)
	s.Require().Len(seq, 1)
	s.Assert().Equal("https", core.StringValueFromScalar(seq[0]["protocol"]))
}

func (s *ResourceConfigVariableSeqTestSuite) Test_treats_sequence_not_starting_at_zero_as_absent() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.sns.ordersTopic.statusLogging.1.protocol": core.ScalarFromString("https"),
		},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().Error(err)
	s.Assert().Nil(seq)
}

func (s *ResourceConfigVariableSeqTestSuite) Test_ignores_non_numeric_index_segments() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{
			"aws.sns.ordersTopic.statusLogging.0.protocol":   core.ScalarFromString("https"),
			"aws.sns.ordersTopic.statusLogging.foo.protocol": core.ScalarFromString("noise"),
		},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().NoError(err)
	s.Require().Len(seq, 1)
	s.Assert().Equal("https", core.StringValueFromScalar(seq[0]["protocol"]))
}

func (s *ResourceConfigVariableSeqTestSuite) Test_returns_error_when_neither_scope_has_entries() {
	transformCtx := newTransformerContextWithConfig(
		testConfigTransformerID,
		map[string]*core.ScalarValue{},
	)

	seq, err := ResourceConfigVariableSeq(transformCtx, "aws.sns", "ordersTopic", "statusLogging")

	s.Require().Error(err)
	s.Assert().Nil(seq)
	s.Assert().Contains(err.Error(), "config variable sequence not found")
	s.Assert().Contains(err.Error(), "aws.sns.ordersTopic.statusLogging.")
}

func TestResourceConfigVariableSeqTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceConfigVariableSeqTestSuite))
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
