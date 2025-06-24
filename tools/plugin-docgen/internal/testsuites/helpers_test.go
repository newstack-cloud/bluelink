package testsuites

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/providertest"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/transformertest"
	"github.com/spf13/afero"
)

type stubExecutor struct {
	manager pluginservicev1.Manager
}

func (e *stubExecutor) Execute(pluginID string, pluginPath string) (plugin.PluginProcess, error) {

	// Simulate the execution of a plugin process
	// by registering the plugin with the manager.
	if pluginID == "newstack-cloud/test" {
		err := e.manager.RegisterPlugin(testProviderPluginInstanceInfo)
		if err != nil {
			return nil, err
		}
	}

	if pluginID == "newstack-cloud/testTransform" {
		err := e.manager.RegisterPlugin(testTransformerPluginInstanceInfo)
		if err != nil {
			return nil, err
		}
	}

	return &stubPluginProcess{}, nil
}

type stubPluginProcess struct{}

func (p *stubPluginProcess) Kill() error {
	return nil
}

func createPluginInstance(info *pluginservicev1.PluginInstanceInfo, hostID string) (any, func(), error) {
	if info.PluginType == pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER &&
		slices.Contains(info.ProtocolVersions, "1.0") {
		return providertest.NewProvider(), func() {}, nil
	}

	if info.PluginType == pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER &&
		slices.Contains(info.ProtocolVersions, "1.0") {
		return transformertest.NewTransformer(), func() {}, nil
	}

	return nil, nil, errors.New("unsupported plugin type or protocol version")
}

func loadPluginsIntoFS(plugins []*plugin.PluginPathInfo, fs afero.Fs) error {
	for _, pluginPath := range plugins {
		fs.MkdirAll(filepath.Dir(pluginPath.AbsolutePath), 0755)
		err := afero.WriteFile(fs, pluginPath.AbsolutePath, []byte{1, 1, 1, 1}, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadExpectedPluginPaths() []*plugin.PluginPathInfo {
	return []*plugin.PluginPathInfo{
		{
			AbsolutePath: "/root/.bluelink/deploy-engine/plugins/bin/providers/newstack-cloud/test/1.0.0/plugin",
			PluginType:   "provider",
			ID:           "newstack-cloud/test",
			Version:      "1.0.0",
		},
		{
			AbsolutePath: "/root/.bluelink/deploy-engine/plugins/bin/transformers/newstack-cloud/testTransform/1.0.0/plugin",
			PluginType:   "transformer",
			ID:           "newstack-cloud/testTransform",
			Version:      "1.0.0",
		},
	}
}

var (
	testProviderPluginInstanceInfo = &pluginservicev1.PluginInstanceInfo{
		PluginType: pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
		ID:         "newstack-cloud/test",
		ProtocolVersions: []string{
			"1.0",
		},
		InstanceID: "1",
		Metadata: &pluginservicev1.PluginMetadata{
			PluginVersion: "1.0.0",
			DisplayName:   "AWS",
			FormattedDescription: "AWS provider for the Deploy Engine including `resources`, `data sources`," +
				" `links` and `custom variable types` for interacting with AWs services.",
			RepositoryUrl: "https://github.com/newstack-cloud/bluelink-provider-aws",
			Author:        "Two Hundred",
		},
	}

	testTransformerPluginInstanceInfo = &pluginservicev1.PluginInstanceInfo{
		PluginType: pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER,
		ID:         "newstack-cloud/testTransform",
		ProtocolVersions: []string{
			"1.0",
		},
		InstanceID: "2",
		Metadata: &pluginservicev1.PluginMetadata{
			PluginVersion:        "1.0.0",
			DisplayName:          "Celerity Transform",
			FormattedDescription: "Celerity application transformer for the Deploy Engine containing the abstract resources that power Celerity application primitives.",
			RepositoryUrl:        "https://github.com/newstack-cloud/celerity-trasformer-testTransform",
			Author:               "Two Hundred",
		},
	}
)

func loadExpectedDocsFromFile(filePath string) (*docgen.PluginDocs, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	docs := &docgen.PluginDocs{}
	err = json.Unmarshal(data, docs)
	return docs, err
}
