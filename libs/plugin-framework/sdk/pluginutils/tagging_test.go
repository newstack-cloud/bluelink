package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type TaggingTestSuite struct {
	suite.Suite
}

// AWS Tags Tests

func (s *TaggingTestSuite) TestToAWSTags_NilInput() {
	result := ToAWSTags(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestToAWSTags_ValidInput() {
	tags := &provider.BluelinkTags{
		InstanceID:            "instance-123",
		InstanceName:          "my-instance",
		ResourceName:          "my-function",
		ResourceType:          "aws/lambda/function",
		ProvisionedBy:         "bluelink",
		DeployEngineVersion:   "1.0.0",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0",
		Prefix:                "bluelink:",
	}

	result := ToAWSTags(tags)

	s.Len(result, 8)

	// Find and check instance-id tag
	var instanceIDTag *AWSTag
	for i := range result {
		if result[i].Key != nil && *result[i].Key == "bluelink:instance-id" {
			instanceIDTag = &result[i]
			break
		}
	}
	s.NotNil(instanceIDTag)
	s.Equal("instance-123", *instanceIDTag.Value)
}

func (s *TaggingTestSuite) TestMergeAWSTags_UserTagsOverride() {
	bluelinkTags := []AWSTag{
		{Key: strPtr("bluelink:instance-id"), Value: strPtr("instance-123")},
		{Key: strPtr("shared-key"), Value: strPtr("bluelink-value")},
	}

	userTags := []AWSTag{
		{Key: strPtr("shared-key"), Value: strPtr("user-value")},
		{Key: strPtr("custom-tag"), Value: strPtr("custom-value")},
	}

	result := MergeAWSTags(bluelinkTags, userTags)

	// Find the shared-key tag - user value should win
	var sharedKeyTag *AWSTag
	for i := range result {
		if result[i].Key != nil && *result[i].Key == "shared-key" {
			sharedKeyTag = &result[i]
			break
		}
	}
	s.NotNil(sharedKeyTag)
	s.Equal("user-value", *sharedKeyTag.Value)

	// All tags should be present
	s.Len(result, 3)
}

// GCP Labels Tests

func (s *TaggingTestSuite) TestToGCPLabels_NilInput() {
	result := ToGCPLabels(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestToGCPLabels_ValidInput() {
	tags := &provider.BluelinkTags{
		InstanceID:   "Instance-123",
		InstanceName: "My-Instance",
		Prefix:       "bluelink:",
	}

	result := ToGCPLabels(tags)

	// GCP labels should be lowercase
	s.Contains(result, "bluelink-instance-id")
	s.Equal("instance-123", result["bluelink-instance-id"])
}

func (s *TaggingTestSuite) TestToGCPLabels_NormalizesInvalidChars() {
	tags := &provider.BluelinkTags{
		InstanceID:   "Instance/123:456",
		ResourceType: "aws/lambda/function",
		Prefix:       "bluelink:",
	}

	result := ToGCPLabels(tags)

	// Colons and slashes should be replaced with hyphens
	s.Equal("instance-123-456", result["bluelink-instance-id"])
	s.Equal("aws-lambda-function", result["bluelink-resource-type"])
}

func (s *TaggingTestSuite) TestToGCPLabels_TruncatesLongValues() {
	longValue := "this-is-a-very-long-value-that-exceeds-the-63-character-limit-for-gcp-labels"
	tags := &provider.BluelinkTags{
		InstanceID: longValue,
		Prefix:     "bluelink:",
	}

	result := ToGCPLabels(tags)

	// Value should be truncated to 63 characters
	s.LessOrEqual(len(result["bluelink-instance-id"]), 63)
}

func (s *TaggingTestSuite) TestMergeGCPLabels_UserLabelsOverride() {
	bluelinkLabels := map[string]string{
		"bluelink-instance-id": "instance-123",
		"shared-key":           "bluelink-value",
	}

	userLabels := map[string]string{
		"shared-key": "user-value",
		"custom-key": "custom-value",
	}

	result := MergeGCPLabels(bluelinkLabels, userLabels)

	// User value should win
	s.Equal("user-value", result["shared-key"])
	// All labels should be present
	s.Len(result, 3)
}

// K8s Labels Tests

func (s *TaggingTestSuite) TestToK8sLabels_NilInput() {
	result := ToK8sLabels(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestToK8sLabels_ValidInput() {
	tags := &provider.BluelinkTags{
		InstanceID:   "instance-123",
		InstanceName: "my-instance",
	}

	result := ToK8sLabels(tags)

	// K8s labels should use the bluelink.dev/ prefix
	s.Contains(result, "bluelink.dev/instance-id")
	s.Equal("instance-123", result["bluelink.dev/instance-id"])
}

func (s *TaggingTestSuite) TestToK8sLabels_NormalizesInvalidChars() {
	tags := &provider.BluelinkTags{
		InstanceID:   "Instance:123/456",
		ResourceType: "aws/lambda/function",
	}

	result := ToK8sLabels(tags)

	// Values should have invalid chars replaced
	s.Equal("Instance-123-456", result["bluelink.dev/instance-id"])
	s.Equal("aws-lambda-function", result["bluelink.dev/resource-type"])
}

func (s *TaggingTestSuite) TestToK8sLabels_TruncatesLongValues() {
	longValue := "this-is-a-very-long-value-that-exceeds-the-63-character-limit-for-k8s-labels"
	tags := &provider.BluelinkTags{
		InstanceID: longValue,
	}

	result := ToK8sLabels(tags)

	// Value should be truncated to 63 characters
	s.LessOrEqual(len(result["bluelink.dev/instance-id"]), 63)
}

// Helper function tests

func (s *TaggingTestSuite) TestNormalizeGCPLabelKey_LowercasesAndReplacesInvalidChars() {
	s.Equal("bluelink-", normalizeGCPLabelKey("bluelink:"))
	s.Equal("my-prefix-", normalizeGCPLabelKey("My-Prefix:"))
	s.Equal("prefix-value-", normalizeGCPLabelKey("prefix/value:"))
}

func (s *TaggingTestSuite) TestNormalizeGCPLabelValue_LowercasesAndTruncates() {
	s.Equal("my-value", normalizeGCPLabelValue("My-Value"))
	s.Equal("value-with-special", normalizeGCPLabelValue("value:with/special"))
}

func (s *TaggingTestSuite) TestNormalizeK8sLabelValue_RemovesInvalidChars() {
	s.Equal("my-value", normalizeK8sLabelValue("my-value"))
	s.Equal("value-with-special", normalizeK8sLabelValue("value:with/special"))
}

func TestTaggingTestSuite(t *testing.T) {
	suite.Run(t, new(TaggingTestSuite))
}
