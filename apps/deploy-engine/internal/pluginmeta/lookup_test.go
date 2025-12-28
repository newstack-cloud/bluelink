package pluginmeta

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/stretchr/testify/suite"
)

// mockPluginManager is a test double for pluginservicev1.Manager
type mockPluginManager struct {
	metadata map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginExtendedMetadata
}

func newMockPluginManager() *mockPluginManager {
	return &mockPluginManager{
		metadata: make(map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginExtendedMetadata),
	}
}

func (m *mockPluginManager) RegisterPlugin(info *pluginservicev1.PluginInstanceInfo) error {
	return nil
}

func (m *mockPluginManager) DeregisterPlugin(pluginType pluginservicev1.PluginType, id string) error {
	return nil
}

func (m *mockPluginManager) GetPlugin(
	pluginType pluginservicev1.PluginType,
	id string,
) *pluginservicev1.PluginInstance {
	return nil
}

func (m *mockPluginManager) GetPluginMetadata(
	pluginType pluginservicev1.PluginType,
	id string,
) *pluginservicev1.PluginExtendedMetadata {
	typeMap, ok := m.metadata[pluginType]
	if !ok {
		return nil
	}
	return typeMap[id]
}

func (m *mockPluginManager) GetPlugins(
	pluginType pluginservicev1.PluginType,
) []*pluginservicev1.PluginInstance {
	return nil
}

func (m *mockPluginManager) addProviderMetadata(id string, metadata *pluginservicev1.PluginExtendedMetadata) {
	if m.metadata[pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER] == nil {
		m.metadata[pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER] = make(
			map[string]*pluginservicev1.PluginExtendedMetadata,
		)
	}
	m.metadata[pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER][id] = metadata
}

type LookupTestSuite struct {
	suite.Suite
	manager *mockPluginManager
	lookup  Lookup
}

func (s *LookupTestSuite) SetupTest() {
	s.manager = newMockPluginManager()
	s.lookup = NewLookup(s.manager)
}

func (s *LookupTestSuite) Test_GetProviderMetadata_with_registered_provider() {
	s.manager.addProviderMetadata("aws", &pluginservicev1.PluginExtendedMetadata{
		PluginVersion: "1.2.3",
		DisplayName:   "AWS Provider",
	})

	metadata := s.lookup.GetProviderMetadata("aws")

	s.NotNil(metadata)
	s.Equal("aws", metadata.PluginID)
	s.Equal("1.2.3", metadata.PluginVersion)
}

func (s *LookupTestSuite) Test_GetProviderMetadata_with_unknown_provider_returns_nil() {
	metadata := s.lookup.GetProviderMetadata("unknown")

	s.Nil(metadata)
}

func (s *LookupTestSuite) Test_GetProviderMetadata_with_nil_manager_returns_nil() {
	lookup := NewLookup(nil)

	metadata := lookup.GetProviderMetadata("aws")

	s.Nil(metadata)
}

func (s *LookupTestSuite) Test_ToLookupFunc_with_valid_lookup() {
	s.manager.addProviderMetadata("gcp", &pluginservicev1.PluginExtendedMetadata{
		PluginVersion: "2.0.0",
	})

	lookupFunc := ToLookupFunc(s.lookup)

	pluginID, pluginVersion := lookupFunc("gcp")
	s.Equal("gcp", pluginID)
	s.Equal("2.0.0", pluginVersion)
}

func (s *LookupTestSuite) Test_ToLookupFunc_with_unknown_provider() {
	lookupFunc := ToLookupFunc(s.lookup)

	pluginID, pluginVersion := lookupFunc("unknown")
	s.Empty(pluginID)
	s.Empty(pluginVersion)
}

func (s *LookupTestSuite) Test_ToLookupFunc_with_nil_lookup_returns_nil() {
	lookupFunc := ToLookupFunc(nil)

	s.Nil(lookupFunc)
}

func TestLookupTestSuite(t *testing.T) {
	suite.Run(t, new(LookupTestSuite))
}
