package testsuites

import (
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/host"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/test/bufconn"
)

type ProviderDocGenTestSuite struct {
	suite.Suite
	hostContainer           *host.Container
	providers               map[string]provider.Provider
	transformers            map[string]transform.SpecTransformer
	envConfig               *env.Config
	expectedProviderDocs    *docgen.PluginDocs
	expectedTransformerDocs *docgen.PluginDocs
}

func (s *ProviderDocGenTestSuite) SetupTest() {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)

	expectedProviderDocs, err := loadExpectedDocsFromFile(
		"__testdata/provider-docs.json",
	)
	s.Require().NoError(err)
	s.expectedProviderDocs = expectedProviderDocs

	expectedTransformerDocs, err := loadExpectedDocsFromFile(
		"__testdata/transformer-docs.json",
	)
	s.Require().NoError(err)
	s.expectedTransformerDocs = expectedTransformerDocs

	envConfig := &env.Config{
		PluginPath:          "/root/.bluelink/deploy-engine/plugins/bin",
		LogLevel:            "debug",
		LaunchWaitTimeoutMS: 10,
		GenerateTimeoutMS:   10,
	}
	s.envConfig = envConfig
	s.providers = make(map[string]provider.Provider)
	s.transformers = make(map[string]transform.SpecTransformer)
	executor := &stubExecutor{}
	memFS := afero.NewMemMapFs()
	loadPluginsIntoFS(loadExpectedPluginPaths(), memFS)
	container, err := host.Setup(
		s.providers,
		s.transformers,
		executor,
		createPluginInstance,
		envConfig,
		memFS,
		listener,
	)
	s.Require().NoError(err)
	// The manager must be set on the executor so that when we call launch,
	// the test plugins can be registered.
	executor.manager = container.Manager
	s.Require().NoError(err)
	s.hostContainer = container
}

func (s *ProviderDocGenTestSuite) TestGenerateProviderDocs() {
	s.runGenerateDocsTest(
		"newstack-cloud/test",
		s.expectedProviderDocs,
	)
}

func (s *ProviderDocGenTestSuite) TestGenerateTransformerDocs() {
	s.runGenerateDocsTest(
		"newstack-cloud/testTransform",
		s.expectedTransformerDocs,
	)
}

func (s *ProviderDocGenTestSuite) runGenerateDocsTest(
	pluginID string,
	expectedDocs *docgen.PluginDocs,
) {
	pluginInstance, err := host.LaunchAndResolvePlugin(
		pluginID,
		s.hostContainer.Launcher,
		s.providers,
		s.transformers,
		s.envConfig,
	)
	s.Require().NoError(err)

	pluginDocs, err := docgen.GeneratePluginDocs(
		pluginID,
		pluginInstance,
		s.hostContainer.Manager,
		s.envConfig,
	)
	s.Require().NoError(err)

	// Serialise and deserialise to ensure all the JSON tags are correct
	// and ultimately, that valid JSON is produced.
	serialisedJSON, err := json.Marshal(pluginDocs)
	s.Require().NoError(err)

	deserialised := &docgen.PluginDocs{}
	err = json.Unmarshal(serialisedJSON, deserialised)
	s.Require().NoError(err)

	assertPluginDocsEqual(
		expectedDocs,
		deserialised,
		&s.Suite,
	)
}

func (s *ProviderDocGenTestSuite) TearDownTest() {
	s.hostContainer.CloseHostServer()
}

func TestProviderDocGenTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderDocGenTestSuite))
}
