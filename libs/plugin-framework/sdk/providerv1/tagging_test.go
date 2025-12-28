package providerv1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

type TaggingTestSuite struct {
	suite.Suite
}

func (s *TaggingTestSuite) TestGetBluelinkTags_NilInput() {
	result := GetBluelinkTags(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestGetBluelinkTags_NilProviderContext() {
	input := &provider.ResourceDeployInput{}
	result := GetBluelinkTags(input)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestGetBluelinkTags_NilTaggingConfig() {
	input := &provider.ResourceDeployInput{
		ProviderContext: &mockProviderContext{taggingConfig: nil},
	}
	result := GetBluelinkTags(input)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestGetBluelinkTags_TaggingDisabled() {
	input := &provider.ResourceDeployInput{
		ProviderContext: &mockProviderContext{
			taggingConfig: &provider.TaggingConfig{Enabled: false},
		},
	}
	result := GetBluelinkTags(input)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestGetBluelinkTags_ValidInput() {
	input := &provider.ResourceDeployInput{
		InstanceID:   "instance-123",
		InstanceName: "my-instance",
		ProviderContext: &mockProviderContext{
			taggingConfig: &provider.TaggingConfig{
				Prefix:                "bluelink:",
				DeployEngineVersion:   "1.0.0",
				ProviderPluginID:      "newstack-cloud/aws",
				ProviderPluginVersion: "2.0.0",
				Enabled:               true,
			},
		},
		Changes: &provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceName: "my-function",
				ResourceWithResolvedSubs: &provider.ResolvedResource{
					Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				},
			},
		},
	}

	result := GetBluelinkTags(input)

	s.NotNil(result)
	s.Equal("instance-123", result.InstanceID)
	s.Equal("my-instance", result.InstanceName)
	s.Equal("my-function", result.ResourceName)
	s.Equal("aws/lambda/function", result.ResourceType)
	s.Equal("bluelink", result.ProvisionedBy)
	s.Equal("1.0.0", result.DeployEngineVersion)
	s.Equal("newstack-cloud/aws", result.ProviderPluginID)
	s.Equal("2.0.0", result.ProviderPluginVersion)
	s.Equal("bluelink:", result.Prefix)
}

func (s *TaggingTestSuite) TestToKeyValuePairs_NilInput() {
	result := ToKeyValuePairs(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestToKeyValuePairs_ValidInput() {
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

	result := ToKeyValuePairs(tags)

	s.Len(result, 8)
	s.Contains(result, TagKeyValue{Key: "bluelink:instance-id", Value: "instance-123"})
	s.Contains(result, TagKeyValue{Key: "bluelink:provisioned-by", Value: "bluelink"})
}

func (s *TaggingTestSuite) TestToKeyValuePairs_DefaultPrefix() {
	tags := &provider.BluelinkTags{
		InstanceID: "instance-123",
		Prefix:     "", // Empty prefix should default to "bluelink:"
	}

	result := ToKeyValuePairs(tags)

	s.NotEmpty(result)
	s.Equal("bluelink:instance-id", result[0].Key)
}

func (s *TaggingTestSuite) TestToMap_NilInput() {
	result := ToMap(nil)
	s.Nil(result)
}

func (s *TaggingTestSuite) TestToMap_ValidInput() {
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

	result := ToMap(tags)

	s.Len(result, 8)
	s.Equal("instance-123", result["bluelink:instance-id"])
	s.Equal("my-instance", result["bluelink:instance-name"])
	s.Equal("my-function", result["bluelink:resource-name"])
	s.Equal("aws/lambda/function", result["bluelink:resource-type"])
	s.Equal("bluelink", result["bluelink:provisioned-by"])
	s.Equal("1.0.0", result["bluelink:deploy-engine-version"])
	s.Equal("newstack-cloud/aws", result["bluelink:provider-plugin-id"])
	s.Equal("2.0.0", result["bluelink:provider-plugin-version"])
}

func TestTaggingTestSuite(t *testing.T) {
	suite.Run(t, new(TaggingTestSuite))
}

// mockProviderContext is a mock implementation of provider.Context for testing
type mockProviderContext struct {
	taggingConfig *provider.TaggingConfig
}

func (m *mockProviderContext) ProviderConfigVariable(name string) (*core.ScalarValue, bool) {
	return nil, false
}

func (m *mockProviderContext) ProviderConfigVariables() map[string]*core.ScalarValue {
	return nil
}

func (m *mockProviderContext) ContextVariable(name string) (*core.ScalarValue, bool) {
	return nil, false
}

func (m *mockProviderContext) ContextVariables() map[string]*core.ScalarValue {
	return nil
}

func (m *mockProviderContext) TaggingConfig() *provider.TaggingConfig {
	return m.taggingConfig
}
