package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	. "gopkg.in/check.v1"
)

type ReferenceValidationTestSuite struct{}

var _ = Suite(&ReferenceValidationTestSuite{})

func (s *ReferenceValidationTestSuite) Test_succeeds_with_no_errors_for_a_set_of_valid_resource_references(c *C) {
	references := []string{
		"saveOrderFunction",
		"saveOrderFunction.spec.functionName",
		"saveOrderDatabase.spec.configuration[0].throughput",
		"ordersTopic.spec.arn",
		"resources.orders-topic.spec.topicName",
		"resources.saveOrderFunction",
		"resources.saveOrderFunction.spec.functionArn",
		"resources.saveOrderFunction.spec.endpoints[].host",
		"resources.saveOrderFunction.spec.endpoints[0].host",
		"resources.saveOrderFunction.spec.functionName",
		"resources.save-order-function.metadata.custom.apiEndpoint",
		"resources.save-order-function.metadata.displayName",
		"resources.saveOrderFunction.spec.confgiruations[0][1].concurrency",
		"resources.s3Buckets[].spec.bucketArn",
		"resources.s3Buckets[1].spec.bucketName",
		"resources[\"s3-buckets.v1\"][2][\"spec\"].bucketName",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.field",
			[]Referenceable{ReferenceableResource},
			nil,
		)
		c.Assert(err, IsNil)
	}
}

func (s *ReferenceValidationTestSuite) Test_succeeds_with_no_errors_for_a_set_of_valid_variable_references(c *C) {
	references := []string{
		"variables.environment",
		"variables.enableFeatureV2",
		"variables.enable_feature_v3",
		"variables.function-name",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.field",
			[]Referenceable{ReferenceableVariable},
			nil,
		)
		c.Assert(err, IsNil)
	}
}

func (s *ReferenceValidationTestSuite) Test_succeeds_with_no_errors_for_a_set_of_valid_value_references(c *C) {
	references := []string{
		"values.buckets[].name",
		"values.secretId",
		"values[\"common.config.v1.name\"]",
		"values.clusterConfig.endpoints[].host",
		"values.clusterConfig.nodes[1].endpoint",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.otherField",
			[]Referenceable{ReferenceableValue},
			nil,
		)
		c.Assert(err, IsNil)
	}
}

func (s *ReferenceValidationTestSuite) Test_succeeds_with_no_errors_for_a_set_of_valid_data_source_references(c *C) {
	references := []string{
		"datasources.network.vpc",
		"datasources.network.endpoints[]",
		"datasources.network.endpoints[0]",
		"datasources.core-infra.queueUrl",
		"datasources.coreInfra1.topics[1]",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.otherField",
			[]Referenceable{ReferenceableDataSource},
			nil,
		)
		c.Assert(err, IsNil)
	}
}

func (s *ReferenceValidationTestSuite) Test_succeeds_with_no_errors_for_a_set_of_valid_child_blueprint_references(c *C) {
	references := []string{
		"children.coreInfrastructure.ordersTopicId",
		"children.coreInfrastructure.cacheNodes[].host",
		"children.core-infrastructure.cacheNodes[0].host",
		"children.topics.orderTopicInfo.type",
		"children.topics.order_topic_info.arn",
		"children.topics.configurations[1]",
		"children.topics.configurations[1][2].messagesPerSecond",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.alternativeField",
			[]Referenceable{ReferenceableChild},
			nil,
		)
		c.Assert(err, IsNil)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_set_of_invalid_resource_references(c *C) {
	references := []string{
		// Resource names should not start with numbers.
		"resources.5430orders-topic.spec.topicName",
		// Metadata does not have a first class apiEndpoint property,
		// custom should be used for arbitrary metadata properties.
		"saveOrderFunction.metadata.apiEndpoint",
		// _innerState is not a valid property of a resource.
		"resources.saveOrderFunction._innerState",
		// displayName is not expected to have any child properties.
		"resources.save-order-function.metadata.displayName.chars[0]",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.field",
			[]Referenceable{ReferenceableResource},
			nil,
		)
		c.Assert(err, NotNil)
		loadErr, isLoadErr := err.(*errors.LoadError)
		c.Assert(isLoadErr, Equals, true)
		c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
		c.Assert(
			loadErr.Error(),
			Equals,
			fmt.Sprintf(
				"blueprint load error: validation failed due to an incorrectly formed reference to a resource (\"%s\") "+
					"in \"test.field\". See the spec documentation for examples and rules for references",
				reference,
			),
		)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_set_of_invalid_variable_references(c *C) {
	references := []string{
		// Variable values should not have child properties.
		"variables.orders-topic.topicName",
		// Variable values can not be arrays.
		"variables.orders-topic[0]",
		// Variable names should not start with numbers.
		"variables.42303some-orders",
		// Missing variable name.
		"variables.",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.field",
			[]Referenceable{ReferenceableVariable},
			nil,
		)
		c.Assert(err, NotNil)
		loadErr, isLoadErr := err.(*errors.LoadError)
		c.Assert(isLoadErr, Equals, true)
		c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
		c.Assert(
			loadErr.Error(),
			Equals,
			fmt.Sprintf(
				"blueprint load error: validation failed due to an incorrectly formed reference to a variable (\"%s\") "+
					"in \"test.field\". See the spec documentation for examples and rules for references",
				reference,
			),
		)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_set_of_invalid_value_references(c *C) {
	references := []string{
		// Values names should not start with numbers.
		"values.42301some-derived-config",
		// Missing value name.
		"values.",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.field",
			[]Referenceable{ReferenceableValue},
			nil,
		)
		c.Assert(err, NotNil)
		loadErr, isLoadErr := err.(*errors.LoadError)
		c.Assert(isLoadErr, Equals, true)
		c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
		c.Assert(
			loadErr.Error(),
			Equals,
			fmt.Sprintf(
				"blueprint load error: validation failed due to an incorrectly formed reference to a local value (\"%s\") "+
					"in \"test.field\". See the spec documentation for examples and rules for references",
				reference,
			),
		)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_set_of_invalid_data_source_references(c *C) {
	references := []string{
		// Data source fields should not be objects with child properties.
		"datasources.orders-topic.configuration.topicArn",
		// Data source arrays can only be one-dimensional primitive arrays.
		"datasources.orders-topic.field[0][1]",
		// Missing data source name.
		"datasources.",
		// Missing data source field.
		"datasources.orders-topic.",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.otherField",
			[]Referenceable{ReferenceableDataSource},
			nil,
		)
		c.Assert(err, NotNil)
		loadErr, isLoadErr := err.(*errors.LoadError)
		c.Assert(isLoadErr, Equals, true)
		c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
		c.Assert(
			loadErr.Error(),
			Equals,
			fmt.Sprintf(
				"blueprint load error: validation failed due to an incorrectly formed reference to a data source (\"%s\") "+
					"in \"test.otherField\". See the spec documentation for examples and rules for references",
				reference,
			),
		)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_set_of_invalid_child_blueprint_references(c *C) {
	references := []string{
		// Child blueprint names should not start with numbers.
		"children.32303-core-infra.ordersTopicId",
		// Missing child blueprint field.
		"children.core-infrastructure.",
		// Missing child blueprint name.
		"children.",
	}

	for _, reference := range references {
		err := ValidateReference(
			reference,
			"test.alternativeField",
			[]Referenceable{ReferenceableChild},
			nil,
		)
		c.Assert(err, NotNil)
		loadErr, isLoadErr := err.(*errors.LoadError)
		c.Assert(isLoadErr, Equals, true)
		c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
		c.Assert(
			loadErr.Error(),
			Equals,
			fmt.Sprintf(
				"blueprint load error: validation failed due to an incorrectly formed reference to a child blueprint (\"%s\") "+
					"in \"test.alternativeField\". See the spec documentation for examples and rules for references",
				reference,
			),
		)
	}
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_resource_reference_for_a_context_that_can_not_reference_resources(c *C) {
	err := ValidateReference(
		"resources.saveOrderFunction",
		"test.field",
		[]Referenceable{
			// ReferenceableResource is not included in the list of referenceable objects
			// for the given context.
			ReferenceableVariable,
			ReferenceableChild,
			ReferenceableDataSource,
		},
		nil,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a reference to a resource (\"resources.saveOrderFunction\") "+
			"being made from \"test.field\", which can not access values from a resource",
	)
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_variable_reference_for_a_context_that_can_not_reference_variables(c *C) {
	err := ValidateReference(
		"variables.ordersTopicName",
		"test.otherField",
		[]Referenceable{
			ReferenceableResource,
			// ReferenceableVariable is not included in the list of referenceable objects
			// for the given context.
			ReferenceableChild,
			ReferenceableDataSource,
		},
		nil,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a reference to a variable (\"variables.ordersTopicName\") "+
			"being made from \"test.otherField\", which can not access values from a variable",
	)
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_data_source_reference_for_a_context_that_can_not_reference_data_sources(c *C) {
	err := ValidateReference(
		"datasources.network.vpc",
		"test.alternativeField",
		[]Referenceable{
			ReferenceableResource,
			ReferenceableVariable,
			ReferenceableChild,
			// ReferenceableDataSource is not included in the list of referenceable objects
			// for the given context.
		},
		nil,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a reference to a data source (\"datasources.network.vpc\") "+
			"being made from \"test.alternativeField\", which can not access values from a data source",
	)
}

func (s *ReferenceValidationTestSuite) Test_reports_error_for_a_child_blueprint_reference_for_a_context_that_can_not_reference_child_blueprints(c *C) {
	err := ValidateReference(
		"children.coreInfra.ordersTopicId",
		"test.altField2",
		[]Referenceable{
			ReferenceableResource,
			ReferenceableVariable,
			ReferenceableDataSource,
			// ReferenceableChild is not included in the list of referenceable objects
			// for the given context.
		},
		nil,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a reference to a child blueprint (\"children.coreInfra.ordersTopicId\") "+
			"being made from \"test.altField2\", which can not access values from a child blueprint",
	)
}
