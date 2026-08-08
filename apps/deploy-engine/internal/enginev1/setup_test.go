package enginev1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/stretchr/testify/assert"
)

// fakeTransformer embeds the SpecTransformer interface so it satisfies the type
// assertion in transformerConfigProvidersByPluginName without implementing every
// method (none are invoked by the helper).
type fakeTransformer struct {
	transform.SpecTransformer
}

// fakeManager embeds the Manager interface and only implements GetPlugins.
type fakeManager struct {
	pluginservicev1.Manager
	transformers []*pluginservicev1.PluginInstance
}

func (m *fakeManager) GetPlugins(
	pluginType pluginservicev1.PluginType,
) []*pluginservicev1.PluginInstance {
	if pluginType == pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER {
		return m.transformers
	}
	return nil
}

func TestTransformerConfigProvidersByPluginName_keysByPluginName(t *testing.T) {
	manager := &fakeManager{
		transformers: []*pluginservicev1.PluginInstance{
			{
				Info:   &pluginservicev1.PluginInstanceInfo{ID: "newstack-cloud/celerity"},
				Client: &fakeTransformer{},
			},
			{
				Info:   &pluginservicev1.PluginInstanceInfo{ID: "acme/other"},
				Client: &fakeTransformer{},
			},
		},
	}

	result := transformerConfigProvidersByPluginName(manager)

	// Transformer config is keyed by the simplified plugin name (namespace),
	// matching how transformer config is keyed in a blueprint operation config.
	assert.Contains(t, result, "celerity")
	assert.Contains(t, result, "other")
	// It must NOT be keyed by the transform name (the string in a blueprint's
	// `transform` section) as that keying belongs to the blueprint loader, and
	// conflating the two silently drops transformer config such as deployTarget.
	assert.NotContains(t, result, "celerity-2025-08-01")
	assert.NotContains(t, result, "newstack-cloud/celerity")
	assert.Len(t, result, 2)
}

func TestTransformerConfigProvidersByPluginName_skipsNonTransformers(t *testing.T) {
	manager := &fakeManager{
		transformers: []*pluginservicev1.PluginInstance{
			{
				Info:   &pluginservicev1.PluginInstanceInfo{ID: "newstack-cloud/celerity"},
				Client: &fakeTransformer{},
			},
			{
				// A registered instance whose client is not a SpecTransformer
				// must be skipped rather than panicking on the type assertion.
				Info:   &pluginservicev1.PluginInstanceInfo{ID: "bad/not-a-transformer"},
				Client: struct{}{},
			},
		},
	}

	result := transformerConfigProvidersByPluginName(manager)

	assert.Contains(t, result, "celerity")
	assert.NotContains(t, result, "not-a-transformer")
	assert.Len(t, result, 1)
}
