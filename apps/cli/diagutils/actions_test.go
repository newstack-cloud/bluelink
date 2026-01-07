package diagutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/stretchr/testify/suite"
)

type ActionsTestSuite struct {
	suite.Suite
}

func TestActionsTestSuite(t *testing.T) {
	suite.Run(t, new(ActionsTestSuite))
}

// --- GetConcreteAction tests ---

func (s *ActionsTestSuite) Test_GetConcreteAction_returns_nil_for_unsupported_action() {
	action := errors.SuggestedAction{Type: "unknown_action"}
	result := GetConcreteAction(action, nil)
	s.Nil(result)
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_install_provider() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeInstallProvider)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins install")
	s.Contains(result.Commands[0], "aws")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_update_provider() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeUpdateProvider)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins update")
	s.Contains(result.Commands[0], "aws")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_function_name() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckFunctionName)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_resource_type() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceType)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].URL, "aws")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_data_source_type() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckDataSourceType)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].URL, "aws")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_variable_type() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckVariableType)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_custom_variable_options() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckCustomVariableOptions)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_abstract_resource_type() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckAbstractResourceType)}
	metadata := map[string]any{"transformerNamespace": "celerity"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/transformers")
	s.Contains(result.Links[0].URL, "celerity")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_transformers() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckTransformers)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/transformers")
}

func (s *ActionsTestSuite) Test_GetConcreteAction_handles_check_resource_type_schema() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceTypeSchema)}
	metadata := map[string]any{"resourceType": "aws/s3/bucket"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].URL, "resources")
}

// --- installProviderAction tests ---

func (s *ActionsTestSuite) Test_installProviderAction_without_provider_namespace() {
	result := installProviderAction(nil)
	s.NotNil(result)
	s.Empty(result.Commands)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_installProviderAction_with_known_provider() {
	metadata := map[string]any{"providerNamespace": "aws"}
	result := installProviderAction(metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins install newstack-cloud/aws")
}

func (s *ActionsTestSuite) Test_installProviderAction_with_unknown_provider() {
	metadata := map[string]any{"providerNamespace": "custom-provider"}
	result := installProviderAction(metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins install <organisation>/custom-provider")
}

// --- updateProviderAction tests ---

func (s *ActionsTestSuite) Test_updateProviderAction_without_provider_namespace() {
	result := updateProviderAction(nil)
	s.NotNil(result)
	s.Empty(result.Commands)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Check for new versions")
}

func (s *ActionsTestSuite) Test_updateProviderAction_with_known_provider() {
	metadata := map[string]any{"providerNamespace": "azure"}
	result := updateProviderAction(metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins update newstack-cloud/azure")
}

func (s *ActionsTestSuite) Test_updateProviderAction_with_unknown_provider() {
	metadata := map[string]any{"providerNamespace": "my-custom-provider"}
	result := updateProviderAction(metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "<organisation>/my-custom-provider")
}

// --- checkFunctionNameAction tests ---

func (s *ActionsTestSuite) Test_checkFunctionNameAction_returns_links() {
	result := checkFunctionNameAction()
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore provider functions in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/providers", result.Links[0].URL)
}

// --- checkResourceTypeAction tests ---

func (s *ActionsTestSuite) Test_checkResourceTypeAction_without_provider_namespace() {
	result := checkResourceTypeAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_checkResourceTypeAction_with_provider_namespace() {
	metadata := map[string]any{"providerNamespace": "gcloud"}
	result := checkResourceTypeAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "resource types")
	s.Contains(result.Links[0].Title, "newstack-cloud/gcloud")
	s.Contains(result.Links[0].URL, "https://registry.bluelink.dev/providers/newstack-cloud/gcloud/latest")
}

// --- checkResourceTypeSchemaAction tests ---

func (s *ActionsTestSuite) Test_checkResourceTypeSchemaAction_without_resource_type() {
	result := checkResourceTypeSchemaAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_checkResourceTypeSchemaAction_with_resource_type() {
	metadata := map[string]any{"resourceType": "aws/s3/bucket"}
	result := checkResourceTypeSchemaAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "aws/s3/bucket")
	s.Contains(result.Links[0].URL, "resources/aws--s3--bucket")
}

func (s *ActionsTestSuite) Test_checkResourceTypeSchemaAction_encodes_colons() {
	metadata := map[string]any{"resourceType": "kubernetes::core/v1/pod"}
	result := checkResourceTypeSchemaAction(metadata)
	s.NotNil(result)
	s.Contains(result.Links[0].URL, "kubernetes--core--v1--pod")
}

// --- checkAbstractResourceTypeAction tests ---

func (s *ActionsTestSuite) Test_checkAbstractResourceTypeAction_without_transformer_namespace() {
	result := checkAbstractResourceTypeAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore transformers")
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/transformers")
}

func (s *ActionsTestSuite) Test_checkAbstractResourceTypeAction_with_transformer_namespace() {
	metadata := map[string]any{"transformerNamespace": "celerity"}
	result := checkAbstractResourceTypeAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "abstract resource types")
	s.Contains(result.Links[0].Title, "newstack-cloud/celerity")
	s.Contains(result.Links[0].URL, "transformers/newstack-cloud/celerity/latest")
}

// --- checkDataSourceTypeAction tests ---

func (s *ActionsTestSuite) Test_checkDataSourceTypeAction_without_provider_namespace() {
	result := checkDataSourceTypeAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_checkDataSourceTypeAction_with_provider_namespace() {
	metadata := map[string]any{"providerNamespace": "kubernetes"}
	result := checkDataSourceTypeAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "data source types")
	s.Contains(result.Links[0].Title, "newstack-cloud/kubernetes")
	s.Contains(result.Links[0].URL, "providers/newstack-cloud/kubernetes/latest")
}

// --- checkVariableTypeAction tests ---

func (s *ActionsTestSuite) Test_checkVariableTypeAction_without_provider_namespace() {
	result := checkVariableTypeAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_checkVariableTypeAction_with_provider_namespace() {
	metadata := map[string]any{"providerNamespace": "aws"}
	result := checkVariableTypeAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "custom variable types")
	s.Contains(result.Links[0].Title, "newstack-cloud/aws")
}

// --- checkCustomVariableOptionsAction tests ---

func (s *ActionsTestSuite) Test_checkCustomVariableOptionsAction_without_provider_namespace() {
	result := checkCustomVariableOptionsAction(nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_checkCustomVariableOptionsAction_with_provider_namespace() {
	metadata := map[string]any{"providerNamespace": "azure"}
	result := checkCustomVariableOptionsAction(metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "custom variable options")
	s.Contains(result.Links[0].Title, "newstack-cloud/azure")
	s.Contains(result.Links[0].URL, "providers/newstack-cloud/azure/latest")
}

// --- exploreTransformersAction tests ---

func (s *ActionsTestSuite) Test_exploreTransformersAction_returns_links() {
	result := exploreTransformersAction()
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore transformers in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/transformers", result.Links[0].URL)
}

// --- defaultProvidersAction tests ---

func (s *ActionsTestSuite) Test_defaultProvidersAction_returns_links() {
	result := defaultProvidersAction()
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore providers in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/providers", result.Links[0].URL)
}

// --- defaultTransformersAction tests ---

func (s *ActionsTestSuite) Test_defaultTransformersAction_returns_links() {
	result := defaultTransformersAction()
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore transformers in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/transformers", result.Links[0].URL)
}

// --- urlEncodeEntityType tests ---

func (s *ActionsTestSuite) Test_urlEncodeEntityType_encodes_slashes() {
	result := urlEncodeEntityType("aws/s3/bucket")
	s.Equal("aws--s3--bucket", result)
}

func (s *ActionsTestSuite) Test_urlEncodeEntityType_encodes_colons() {
	result := urlEncodeEntityType("kubernetes::core")
	s.Equal("kubernetes--core", result)
}

func (s *ActionsTestSuite) Test_urlEncodeEntityType_encodes_mixed() {
	result := urlEncodeEntityType("kubernetes::core/v1/pod")
	s.Equal("kubernetes--core--v1--pod", result)
}

func (s *ActionsTestSuite) Test_urlEncodeEntityType_handles_empty_string() {
	result := urlEncodeEntityType("")
	s.Equal("", result)
}

func (s *ActionsTestSuite) Test_urlEncodeEntityType_handles_no_special_chars() {
	result := urlEncodeEntityType("simple")
	s.Equal("simple", result)
}
