package provider

import (
	"context"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	. "gopkg.in/check.v1"
)

type CustomVariableTypeRegistryTestSuite struct {
	customVarTypeRegistry  CustomVariableTypeRegistry
	testCustomVariableType *testEC2InstanceTypeCustomVariableType
}

var _ = Suite(&CustomVariableTypeRegistryTestSuite{})

func (s *CustomVariableTypeRegistryTestSuite) SetUpTest(c *C) {
	testCustomVariableType := newTestEC2InstanceCustomVarType()

	providers := map[string]Provider{
		"aws": &testProvider{
			customVarTypes: map[string]CustomVariableType{
				"aws/ec2/instanceType": testCustomVariableType,
			},
			namespace: "aws",
		},
	}

	s.testCustomVariableType = testCustomVariableType.(*testEC2InstanceTypeCustomVariableType)
	s.customVarTypeRegistry = NewCustomVariableTypeRegistry(providers)
}

func (s *CustomVariableTypeRegistryTestSuite) Test_get_description(c *C) {
	output, err := s.customVarTypeRegistry.GetDescription(
		context.TODO(),
		"aws/ec2/instanceType",
		&CustomVariableTypeGetDescriptionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.MarkdownDescription, Equals, s.testCustomVariableType.markdownDescription)
	c.Assert(output.PlainTextDescription, Equals, s.testCustomVariableType.plainTextDescription)

	// Second time should be cached and produce the same result.
	output, err = s.customVarTypeRegistry.GetDescription(
		context.TODO(),
		"aws/ec2/instanceType",
		&CustomVariableTypeGetDescriptionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.MarkdownDescription, Equals, s.testCustomVariableType.markdownDescription)
	c.Assert(output.PlainTextDescription, Equals, s.testCustomVariableType.plainTextDescription)
}

func (s *CustomVariableTypeRegistryTestSuite) Test_has_custom_variable_type(c *C) {
	hasVarType, err := s.customVarTypeRegistry.HasCustomVariableType(context.TODO(), "aws/ec2/instanceType")
	c.Assert(err, IsNil)
	c.Assert(hasVarType, Equals, true)

	hasVarType, err = s.customVarTypeRegistry.HasCustomVariableType(context.TODO(), "aws/otherVarType")
	c.Assert(err, IsNil)
	c.Assert(hasVarType, Equals, false)
}

func (s *CustomVariableTypeRegistryTestSuite) Test_list_custom_var_types(c *C) {
	customVarTypes, err := s.customVarTypeRegistry.ListCustomVariableTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsTestCustomVarType := slices.Contains(
		customVarTypes,
		"aws/ec2/instanceType",
	)
	c.Assert(containsTestCustomVarType, Equals, true)

	// Second time should be cached and produce the same result.
	customVarTypesCached, err := s.customVarTypeRegistry.ListCustomVariableTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsCachedTestCustomVarType := slices.Contains(
		customVarTypesCached,
		"aws/ec2/instanceType",
	)
	c.Assert(containsCachedTestCustomVarType, Equals, true)
}

func (s *CustomVariableTypeRegistryTestSuite) Test_produces_error_for_missing_provider(c *C) {
	_, err := s.customVarTypeRegistry.HasCustomVariableType(context.TODO(), "otherProvider/otherVarType")
	c.Assert(err, NotNil)
	runErr, isRunErr := err.(*errors.RunError)
	c.Assert(isRunErr, Equals, true)
	c.Assert(runErr.ReasonCode, Equals, ErrorReasonCodeItemTypeProviderNotFound)
	c.Assert(runErr.Error(), Equals, "run error: provider \"otherProvider\" not found for custom variable type \"otherProvider/otherVarType\"")
}
