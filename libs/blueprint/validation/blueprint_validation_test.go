package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	. "gopkg.in/check.v1"
)

type BlueprintValidationTestSuite struct{}

var _ = Suite(&BlueprintValidationTestSuite{})

func (s *BlueprintValidationTestSuite) Test_succeeds_without_any_issues_for_a_valid_blueprint(c *C) {

	instanceType := "t2.micro"
	version := Version2025_05_12
	blueprint := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"resource1": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/instance"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"instanceType": {
								Scalar: &core.ScalarValue{
									StringValue: &instanceType,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := ValidateBlueprint(context.Background(), blueprint)
	c.Assert(err, IsNil)
}

func (s *BlueprintValidationTestSuite) Test_reports_errors_when_the_version_is_not_set(c *C) {
	instanceType := "t2.micro"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"resource1": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/instance"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"instanceType": {
								Scalar: &core.ScalarValue{
									StringValue: &instanceType,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ValidateBlueprint(context.Background(), blueprint)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)

	childLoadErr, isChildLoadErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildLoadErr, Equals, true)
	c.Assert(childLoadErr.ReasonCode, Equals, ErrorReasonCodeMissingVersion)
	c.Assert(
		childLoadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a version not being provided, version is a required property",
	)
}

func (s *BlueprintValidationTestSuite) Test_reports_errors_when_the_version_is_incorrect(c *C) {
	// In the intial version of blueprint framework, only version
	// 2025-05-12-draft of the spec is supported.
	instanceType := "t2.micro"
	version := "2023-09-15"
	blueprint := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"resource1": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ec2/instance"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"instanceType": {
								Scalar: &core.ScalarValue{
									StringValue: &instanceType,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := ValidateBlueprint(context.Background(), blueprint)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)

	childLoadErr, isChildLoadErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildLoadErr, Equals, true)
	c.Assert(childLoadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVersion)
	c.Assert(
		childLoadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an unsupported version \"2023-09-15\" being provided. "+
			"supported versions include: 2025-05-12-draft",
	)
}

func (s *BlueprintValidationTestSuite) Test_reports_errors_when_the_resources_and_include_properties_are_missing(c *C) {
	version := Version2025_05_12
	blueprint := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
	}
	_, err := ValidateBlueprint(context.Background(), blueprint)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)

	childLoadErr, isChildLoadErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildLoadErr, Equals, true)
	c.Assert(childLoadErr.ReasonCode, Equals, ErrorReasonCodeMissingResourcesOrIncludes)
	c.Assert(
		childLoadErr.Error(),
		Equals,
		"blueprint load error: validation failed as no resources or includes have been defined,"+
			" at least one resource must be defined in a blueprint if there are no includes and"+
			" at least one include must be defined in a blueprint if there are no resources",
	)
}

func (s *BlueprintValidationTestSuite) Test_reports_errors_when_no_resources_or_includes_are_provided(c *C) {
	version := Version2025_05_12
	blueprint := &schema.Blueprint{
		Version:   &core.ScalarValue{StringValue: &version},
		Resources: &schema.ResourceMap{},
		Include:   &schema.IncludeMap{},
	}
	_, err := ValidateBlueprint(context.Background(), blueprint)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)

	childLoadErr, isChildLoadErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildLoadErr, Equals, true)
	c.Assert(childLoadErr.ReasonCode, Equals, ErrorReasonCodeMissingResourcesOrIncludes)
	c.Assert(
		childLoadErr.Error(),
		Equals,
		"blueprint load error: validation failed as no resources or includes have been defined,"+
			" at least one resource must be defined in a blueprint if there are no includes and"+
			" at least one include must be defined in a blueprint if there are no resources",
	)
}
