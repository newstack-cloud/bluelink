package testutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// CreateTestProviderContext creates a provider context for testing
// with the given namespace.
func CreateTestProviderContext(namespace string) provider.Context {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	return provider.NewProviderContextFromParams(namespace, params)
}

// CreateTestTransformerContext creates a transformer context for testing
// with the given namespace.
func CreateTestTransformerContext(namespace string) transform.Context {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	return transform.NewTransformerContextFromParams(namespace, params)
}

// CreateTestLinkContext creates a link context for testing.
func CreateTestLinkContext() provider.LinkContext {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	return provider.NewLinkContextFromParams(params)
}

// CreateEmptyTestParams creates an empty set of parameters for testing,
// primarily used for testing plugin functions.
func CreateEmptyTestParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

// CreateEmptyConcreteParams creates an empty set of concrete parameters
// for testing, primarily used for testing plugin functions.
func CreateEmptyConcreteParams() *core.ParamsImpl {
	return &core.ParamsImpl{
		ProviderConf:       map[string]map[string]*core.ScalarValue{},
		TransformerConf:    map[string]map[string]*core.ScalarValue{},
		ContextVariables:   map[string]*core.ScalarValue{},
		BlueprintVariables: map[string]*core.ScalarValue{},
	}
}

// CreateTestProviderContextWithTagging creates a provider context for testing
// with the given namespace and tagging configuration.
func CreateTestProviderContextWithTagging(
	namespace string,
	taggingConfig *provider.TaggingConfig,
) provider.Context {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	return provider.NewProviderContextFromParamsWithOptions(
		namespace,
		params,
		&provider.ProviderContextOptions{
			TaggingConfig: taggingConfig,
		},
	)
}

// TestTaggingConfig returns a standard tagging configuration for testing.
func TestTaggingConfig() *provider.TaggingConfig {
	return &provider.TaggingConfig{
		Prefix:                "bluelink:",
		DeployEngineVersion:   "1.0.0-test",
		ProviderPluginID:      "newstack-cloud/aws",
		ProviderPluginVersion: "2.0.0-test",
		Enabled:               true,
	}
}
