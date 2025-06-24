package plugin

import (
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type DiscoverSuite struct {
	fs       afero.Fs
	expected []*PluginPathInfo
	suite.Suite
}

func (s *DiscoverSuite) SetupTest() {
	s.expected = loadExpectedPluginPaths()
	s.fs = afero.NewMemMapFs()
	err := loadPluginsIntoFS(s.expected, s.fs)
	s.Require().NoError(err)
}

func (s *DiscoverSuite) Test_discovers_plugins() {
	pluginPath := strings.Join(testPluginRootPaths, ":")
	discoveredPlugins, err := DiscoverPlugins(pluginPath, s.fs, core.NewNopLogger())
	s.Require().NoError(err)
	s.Require().Len(discoveredPlugins, len(s.expected))
	for i, plugin := range discoveredPlugins {
		s.Equal(s.expected[i].AbsolutePath, plugin.AbsolutePath)
		s.Equal(s.expected[i].PluginType, plugin.PluginType)
		s.Equal(s.expected[i].ID, plugin.ID)
		s.Equal(s.expected[i].Version, plugin.Version)
	}
}

func TestDiscoverSuite(t *testing.T) {
	suite.Run(t, new(DiscoverSuite))
}
