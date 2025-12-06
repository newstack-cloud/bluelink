package plugin

import (
	context "context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	testTransformName = "celerity-transform-2025"
)

type LaunchSuite struct {
	fs                afero.Fs
	expected          []*PluginPathInfo
	launcher          *Launcher
	alternateLauncher *Launcher
	suite.Suite
}

func (s *LaunchSuite) SetupTest() {
	s.expected = loadExpectedPluginPaths()
	s.fs = afero.NewMemMapFs()
	err := loadPluginsIntoFS(s.expected, s.fs)
	s.Require().NoError(err)

	pluginPath := strings.Join(testPluginRootPaths, string(os.PathListSeparator))
	manager := &mockPluginManager{
		pluginMap: map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginInstance{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    {},
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: {},
		},
		testTransformName: testTransformName,
	}
	executor := &mockPluginExecutor{
		pluginManager: manager,
		registerOnAttempt: map[string]int{
			s.expected[0].AbsolutePath: 1,
			s.expected[1].AbsolutePath: 4,
			s.expected[2].AbsolutePath: 2,
		},
		registerAttempts: map[string]int{},
		pluginInstances:  s.instancesFromPluginPaths(),
	}
	s.launcher = NewLauncher(
		pluginPath,
		manager,
		executor,
		core.NewNopLogger(),
		WithLauncherFS(s.fs),
		WithLauncherAttemptLimit(5),
		WithLauncherWaitTimeout(5*time.Millisecond),
		WithLauncherCheckRegisteredInterval(1*time.Millisecond),
	)
	s.alternateLauncher = NewLauncher(
		pluginPath,
		manager,
		executor,
		core.NewNopLogger(),
		WithLauncherFS(s.fs),
		WithLauncherAttemptLimit(5),
		WithLauncherWaitTimeout(5*time.Millisecond),
		WithLauncherCheckRegisteredInterval(1*time.Millisecond),
		WithLauncherTransformerKeyType(TransformerKeyTypePluginName),
	)
}

func (s *LaunchSuite) instancesFromPluginPaths() map[string]*pluginservicev1.PluginInstanceInfo {
	instances := map[string]*pluginservicev1.PluginInstanceInfo{}
	for _, pluginPath := range s.expected {
		instances[pluginPath.AbsolutePath] = &pluginservicev1.PluginInstanceInfo{
			PluginType: pluginservicev1.PluginTypeFromString(
				pluginPath.PluginType,
			),
			ProtocolVersions: []string{"1.0"},
			ID:               pluginPath.ID,
		}
	}
	return instances
}

func (s *LaunchSuite) Test_launches_plugins() {
	pluginMaps, err := s.launcher.Launch(context.Background())
	s.Require().NoError(err)

	s.Assert().Len(pluginMaps.Providers, 2)

	s.assertHasProvider(pluginMaps, "aws")
	s.assertHasProvider(pluginMaps, "azure")

	s.Assert().Len(pluginMaps.Transformers, 1)

	s.assertHasTransformer(pluginMaps, testTransformName, TransformerKeyTypeTransformName)
}

func (s *LaunchSuite) Test_launches_plugins_with_transform_plugin_name_key() {
	pluginMaps, err := s.alternateLauncher.Launch(context.Background())
	s.Require().NoError(err)

	s.Assert().Len(pluginMaps.Providers, 2)

	s.assertHasProvider(pluginMaps, "aws")
	s.assertHasProvider(pluginMaps, "azure")

	s.Assert().Len(pluginMaps.Transformers, 1)

	s.assertHasTransformer(pluginMaps, "celerity", TransformerKeyTypePluginName)
}

func (s *LaunchSuite) assertHasProvider(
	pluginMaps *PluginMaps,
	namespace string,
) {
	provider, hasProvider := pluginMaps.Providers[namespace]
	s.Assert().True(hasProvider)
	result, err := provider.Namespace(context.Background())
	s.Require().NoError(err)
	s.Assert().Equal(namespace, result)
}

func (s *LaunchSuite) assertHasTransformer(
	pluginMaps *PluginMaps,
	transformNameOrNamespace string,
	keyType TransformerKeyType,
) {
	transformer, hasTransformer := pluginMaps.Transformers[transformNameOrNamespace]
	s.Assert().True(hasTransformer)
	if keyType == TransformerKeyTypeTransformName {
		result, err := transformer.GetTransformName(context.Background())
		s.Require().NoError(err)
		s.Assert().Equal(transformNameOrNamespace, result)
	}
}

func TestLaunchSuite(t *testing.T) {
	suite.Run(t, new(LaunchSuite))
}
