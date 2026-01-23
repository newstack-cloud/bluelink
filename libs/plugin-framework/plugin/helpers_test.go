package plugin

import (
	"path/filepath"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/spf13/afero"
)

var (
	testPluginRootPaths = []string{
		"/root/.bluelink/deploy-engine/plugins/bin",
		"/usr/local/bluelink/deploy-engine/plugins/bin",
	}
)

func loadPluginsIntoFS(plugins []*PluginPathInfo, fs afero.Fs) error {
	for _, pluginPath := range plugins {
		fs.MkdirAll(filepath.Dir(pluginPath.AbsolutePath), 0755)
		err := afero.WriteFile(fs, pluginPath.AbsolutePath, []byte{1, 1, 1, 1}, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadExpectedPluginPaths() []*PluginPathInfo {
	return []*PluginPathInfo{
		{
			AbsolutePath: "/root/.bluelink/deploy-engine/plugins/bin/providers/bluelink/aws/1.0.0/plugin",
			PluginType:   "provider",
			ID:           "bluelink/aws",
			Version:      "1.0.0",
		},
		{
			AbsolutePath: "/root/.bluelink/deploy-engine/plugins/bin/transformers/celerity/celerity/2.0.1/plugin",
			PluginType:   "transformer",
			ID:           "celerity/celerity",
			Version:      "2.0.1",
		},
		{
			AbsolutePath: "/usr/local/bluelink/deploy-engine/plugins/bin/providers" +
				"/registry.customhost.com/bluelink/azure/3.2.0/plugin",
			PluginType: "provider",
			ID:         "registry.customhost.com/bluelink/azure",
			Version:    "3.2.0",
		},
	}
}

type mockPluginManager struct {
	pluginMap         map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginInstance
	pluginMetadata    map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginExtendedMetadata
	testTransformName string
}

func (m *mockPluginManager) GetPlugins(
	pluginType pluginservicev1.PluginType,
) []*pluginservicev1.PluginInstance {
	instances := []*pluginservicev1.PluginInstance{}
	for _, instance := range m.pluginMap[pluginType] {
		instances = append(instances, instance)
	}
	return instances
}

func (m *mockPluginManager) GetPlugin(
	pluginType pluginservicev1.PluginType,
	id string,
) *pluginservicev1.PluginInstance {
	instancesForType, hasPluginType := m.pluginMap[pluginType]
	if !hasPluginType {
		return nil
	}

	pluginInstance, hasPlugin := instancesForType[id]
	if !hasPlugin {
		return nil
	}

	return pluginInstance
}

func (m *mockPluginManager) GetPluginMetadata(
	pluginType pluginservicev1.PluginType,
	id string,
) *pluginservicev1.PluginExtendedMetadata {
	metadataForType, hasMetadataType := m.pluginMetadata[pluginType]
	if !hasMetadataType {
		return nil
	}

	metadata, hasMetadata := metadataForType[id]
	if !hasMetadata {
		return nil
	}

	return metadata
}

func (m *mockPluginManager) RegisterPlugin(
	pluginInstanceInfo *pluginservicev1.PluginInstanceInfo,
) error {
	client := createMockPluginClient(pluginInstanceInfo, m.testTransformName)
	instance := &pluginservicev1.PluginInstance{
		Info:   pluginInstanceInfo,
		Client: client,
		CloseConn: func() {
			// Do nothing as the plugin is a stub for launch testing.
		},
	}
	m.pluginMap[pluginInstanceInfo.PluginType][pluginInstanceInfo.ID] = instance
	if pluginInstanceInfo.Metadata != nil {
		m.pluginMetadata[pluginInstanceInfo.PluginType][pluginInstanceInfo.ID] = &pluginservicev1.PluginExtendedMetadata{
			PluginVersion:        pluginInstanceInfo.Metadata.PluginVersion,
			DisplayName:          pluginInstanceInfo.Metadata.DisplayName,
			PlainTextDescription: pluginInstanceInfo.Metadata.PlainTextDescription,
			FormattedDescription: pluginInstanceInfo.Metadata.FormattedDescription,
			RepositoryUrl:        pluginInstanceInfo.Metadata.RepositoryUrl,
			Author:               pluginInstanceInfo.Metadata.Author,
			ProtocolVersions:     pluginInstanceInfo.ProtocolVersions,
		}
	}
	return nil
}

func (m *mockPluginManager) DeregisterPlugin(
	pluginType pluginservicev1.PluginType,
	id string,
) error {
	delete(m.pluginMap[pluginType], id)
	delete(m.pluginMetadata[pluginType], id)
	return nil
}

func (m *mockPluginManager) SetPluginProcess(
	pluginType pluginservicev1.PluginType,
	id string,
	killProcess func() error,
) {
	instancesForType, hasPluginType := m.pluginMap[pluginType]
	if !hasPluginType {
		return
	}

	instance, hasInstance := instancesForType[id]
	if !hasInstance {
		return
	}

	instance.KillProcess = killProcess
}

func (m *mockPluginManager) Close() {
	for _, instancesForType := range m.pluginMap {
		for _, instance := range instancesForType {
			if instance.CloseConn != nil {
				instance.CloseConn()
			}
			if instance.KillProcess != nil {
				_ = instance.KillProcess()
			}
		}
	}
}

type mockPluginExecutor struct {
	pluginManager pluginservicev1.Manager
	// A mapping of plugin paths to the number of times they should be
	// attempted before they register with the plugin manager.
	registerOnAttempt map[string]int
	// A mapping of plugin paths to the number of times they have been
	// attempted to register with the plugin manager.
	registerAttempts map[string]int
	// A mapping of plugin paths to the plugin instance info used
	// to register the plugin with the plugin manager.
	pluginInstances map[string]*pluginservicev1.PluginInstanceInfo
}

func (e *mockPluginExecutor) Execute(pluginID string, pluginPath string) (PluginProcess, error) {
	attempts, hasAttempts := e.registerAttempts[pluginPath]
	if !hasAttempts {
		e.registerAttempts[pluginPath] = 0
	}

	if attempts < e.registerOnAttempt[pluginPath] {
		e.registerAttempts[pluginPath] = attempts + 1
		return &mockPluginProcess{}, nil
	}

	pluginInstanceInfo := e.pluginInstances[pluginPath]
	err := e.pluginManager.RegisterPlugin(pluginInstanceInfo)
	if err != nil {
		return nil, err
	}

	return &mockPluginProcess{}, nil
}

type mockPluginProcess struct{}

func (p *mockPluginProcess) Kill() error {
	return nil
}

func createMockPluginClient(
	pluginInfo *pluginservicev1.PluginInstanceInfo,
	transformName string,
) any {
	if pluginInfo.PluginType == pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER {
		return &testutils.MockProvider{
			ProviderNamespace: utils.ExtractPluginNamespace(pluginInfo.ID),
		}
	}

	return &testutils.MockTransformer{
		TransformName: transformName,
	}
}
