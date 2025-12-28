package plugintestsuites

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

// Test_tagging_config_flows_through_grpc_deploy verifies that TaggingConfig
// set on the ProviderContext is correctly serialized through gRPC and
// accessible on the plugin side during resource deployment.
// The test provider echoes back the received tagging config in the deploy output,
// allowing us to verify the full round-trip through the gRPC layer.
func (s *ProviderPluginV1Suite) Test_tagging_config_flows_through_grpc_deploy() {
	resource, err := s.provider.Resource(context.Background(), lambdaFunctionResourceType)
	s.Require().NoError(err)

	taggingConfig := &provider.TaggingConfig{
		Prefix:                "bluelink:",
		DeployEngineVersion:   "1.0.0-grpc-test",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0-grpc-test",
		Enabled:               true,
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("aws", taggingConfig)

	output, err := resource.Deploy(
		context.Background(),
		&provider.ResourceDeployInput{
			ResourceID:      testResource1ID,
			InstanceID:      testInstance1ID,
			InstanceName:    "test-instance",
			Changes:         createDeployNewResourceChanges(),
			ProviderContext: providerCtx,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Verify the tagging config was received by the plugin through gRPC
	// by checking the echoed values in the deploy output.
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_enabled")
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_prefix")
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_deploy_engine_version")
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_provider_plugin_id")
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_provider_plugin_version")

	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_enabled"].Scalar.BoolValue)
	s.Equal(true, *output.ComputedFieldValues["__test_tagging_enabled"].Scalar.BoolValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Equal("bluelink:", *output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_deploy_engine_version"].Scalar.StringValue)
	s.Equal("1.0.0-grpc-test", *output.ComputedFieldValues["__test_tagging_deploy_engine_version"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_provider_plugin_id"].Scalar.StringValue)
	s.Equal("newstack-cloud/aws", *output.ComputedFieldValues["__test_tagging_provider_plugin_id"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
	s.Equal("2.0.0-grpc-test", *output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
}

// Test_tagging_config_with_custom_prefix_flows_through_grpc verifies that
// custom tag prefixes are preserved through the gRPC layer.
func (s *ProviderPluginV1Suite) Test_tagging_config_with_custom_prefix_flows_through_grpc() {
	resource, err := s.provider.Resource(context.Background(), lambdaFunctionResourceType)
	s.Require().NoError(err)

	taggingConfig := &provider.TaggingConfig{
		Prefix:                "myorg:bluelink:",
		DeployEngineVersion:   "3.0.0",
		ProviderPluginID:      "myorg/custom-provider",
		ProviderPluginVersion: "1.5.0",
		Enabled:               true,
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("custom", taggingConfig)

	output, err := resource.Deploy(
		context.Background(),
		&provider.ResourceDeployInput{
			ResourceID:      testResource1ID,
			InstanceID:      testInstance1ID,
			InstanceName:    "test-instance-custom",
			Changes:         createDeployNewResourceChanges(),
			ProviderContext: providerCtx,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Verify custom prefix and plugin IDs are preserved
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Equal("myorg:bluelink:", *output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_provider_plugin_id"].Scalar.StringValue)
	s.Equal("myorg/custom-provider", *output.ComputedFieldValues["__test_tagging_provider_plugin_id"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
	s.Equal("1.5.0", *output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
}

// Test_tagging_disabled_flows_through_grpc verifies that when tagging is disabled,
// the Enabled=false value is correctly passed through gRPC.
func (s *ProviderPluginV1Suite) Test_tagging_disabled_flows_through_grpc() {
	resource, err := s.provider.Resource(context.Background(), lambdaFunctionResourceType)
	s.Require().NoError(err)

	taggingConfig := &provider.TaggingConfig{
		Prefix:                "bluelink:",
		DeployEngineVersion:   "1.0.0",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0",
		Enabled:               false, // Tagging disabled
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("aws", taggingConfig)

	output, err := resource.Deploy(
		context.Background(),
		&provider.ResourceDeployInput{
			ResourceID:      testResource1ID,
			InstanceID:      testInstance1ID,
			InstanceName:    "test-instance",
			Changes:         createDeployNewResourceChanges(),
			ProviderContext: providerCtx,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Verify that Enabled=false is preserved through gRPC
	s.Require().Contains(output.ComputedFieldValues, "__test_tagging_enabled")
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_enabled"].Scalar.BoolValue)
	s.Equal(false, *output.ComputedFieldValues["__test_tagging_enabled"].Scalar.BoolValue)
}

// Test_nil_tagging_config_does_not_include_test_fields verifies that when
// no tagging config is set, the test provider does not include tagging fields
// in the output.
func (s *ProviderPluginV1Suite) Test_nil_tagging_config_does_not_include_test_fields() {
	resource, err := s.provider.Resource(context.Background(), lambdaFunctionResourceType)
	s.Require().NoError(err)

	// Create context without tagging config
	providerCtx := testutils.CreateTestProviderContext("aws")

	output, err := resource.Deploy(
		context.Background(),
		&provider.ResourceDeployInput{
			ResourceID:      testResource1ID,
			InstanceID:      testInstance1ID,
			InstanceName:    "test-instance",
			Changes:         createDeployNewResourceChanges(),
			ProviderContext: providerCtx,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Verify no tagging fields are present when config is nil
	s.NotContains(output.ComputedFieldValues, "__test_tagging_enabled")
	s.NotContains(output.ComputedFieldValues, "__test_tagging_prefix")
}

// Test_get_bluelink_tags_with_grpc_tagging_config verifies that GetBluelinkTags
// correctly generates tags when the tagging config flows through gRPC.
// This tests the complete integration: TaggingConfig → gRPC → GetBluelinkTags.
func (s *ProviderPluginV1Suite) Test_get_bluelink_tags_with_grpc_tagging_config() {
	taggingConfig := &provider.TaggingConfig{
		Prefix:                "bluelink:",
		DeployEngineVersion:   "1.0.0",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0",
		Enabled:               true,
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("aws", taggingConfig)

	input := &provider.ResourceDeployInput{
		ResourceID:      testResource1ID,
		InstanceID:      testInstance1ID,
		InstanceName:    "my-test-instance",
		Changes:         createDeployResourceChanges(false),
		ProviderContext: providerCtx,
	}

	// Test GetBluelinkTags helper on the client side
	tags := providerv1.GetBluelinkTags(input)
	s.Require().NotNil(tags)

	// Verify instance-level tags
	s.Equal(testInstance1ID, tags.InstanceID)
	s.Equal("my-test-instance", tags.InstanceName)

	// Verify resource-level tags
	s.Equal(testResource1Name, tags.ResourceName)
	s.Equal(lambdaFunctionResourceType, tags.ResourceType)

	// Verify provenance tags from tagging config
	s.Equal("bluelink", tags.ProvisionedBy)
	s.Equal("1.0.0", tags.DeployEngineVersion)
	s.Equal("newstack-cloud/aws", tags.ProviderPluginID)
	s.Equal("2.0.0", tags.ProviderPluginVersion)

	// Verify prefix
	s.Equal("bluelink:", tags.Prefix)
}

// Test_get_bluelink_tags_returns_nil_when_disabled verifies that GetBluelinkTags
// returns nil when tagging is disabled, even if config is present.
func (s *ProviderPluginV1Suite) Test_get_bluelink_tags_returns_nil_when_disabled() {
	taggingConfig := &provider.TaggingConfig{
		Prefix:                "bluelink:",
		DeployEngineVersion:   "1.0.0",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0",
		Enabled:               false, // Disabled
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("aws", taggingConfig)

	input := &provider.ResourceDeployInput{
		ResourceID:      testResource1ID,
		InstanceID:      testInstance1ID,
		InstanceName:    "test-instance",
		Changes:         createDeployNewResourceChanges(),
		ProviderContext: providerCtx,
	}

	tags := providerv1.GetBluelinkTags(input)
	s.Nil(tags)
}

// Test_get_bluelink_tags_returns_nil_when_config_is_nil verifies that
// GetBluelinkTags returns nil when no tagging config is set.
func (s *ProviderPluginV1Suite) Test_get_bluelink_tags_returns_nil_when_config_is_nil() {
	providerCtx := testutils.CreateTestProviderContext("aws")

	input := &provider.ResourceDeployInput{
		ResourceID:      testResource1ID,
		InstanceID:      testInstance1ID,
		InstanceName:    "test-instance",
		Changes:         createDeployNewResourceChanges(),
		ProviderContext: providerCtx,
	}

	tags := providerv1.GetBluelinkTags(input)
	s.Nil(tags)
}

// Test_tagging_config_flows_through_grpc_for_update verifies that TaggingConfig
// flows correctly through gRPC for resource update operations.
func (s *ProviderPluginV1Suite) Test_tagging_config_flows_through_grpc_for_update() {
	resource, err := s.provider.Resource(context.Background(), lambdaFunctionResourceType)
	s.Require().NoError(err)

	taggingConfig := &provider.TaggingConfig{
		Prefix:                "update-test:",
		DeployEngineVersion:   "2.5.0",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "3.0.0",
		Enabled:               true,
	}
	providerCtx := testutils.CreateTestProviderContextWithTagging("aws", taggingConfig)

	// Use update changes (existing resource)
	output, err := resource.Deploy(
		context.Background(),
		&provider.ResourceDeployInput{
			ResourceID:      testResource1ID,
			InstanceID:      testInstance1ID,
			InstanceName:    "test-instance",
			Changes:         createDeployResourceChanges(false),
			ProviderContext: providerCtx,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Verify the tagging config was received correctly for update operation
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Equal("update-test:", *output.ComputedFieldValues["__test_tagging_prefix"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_deploy_engine_version"].Scalar.StringValue)
	s.Equal("2.5.0", *output.ComputedFieldValues["__test_tagging_deploy_engine_version"].Scalar.StringValue)
	s.Require().NotNil(output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
	s.Equal("3.0.0", *output.ComputedFieldValues["__test_tagging_provider_plugin_version"].Scalar.StringValue)
}
