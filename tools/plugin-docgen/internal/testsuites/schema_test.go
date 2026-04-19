package testsuites

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/host"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/test/bufconn"
	"gopkg.in/yaml.v3"
)

// SchemaValidationTestSuite validates that docgen output conforms to
// the published JSON schema at schema/plugin-docs.schema.yaml.
type SchemaValidationTestSuite struct {
	suite.Suite
	hostContainer *host.Container
	providers     map[string]provider.Provider
	transformers  map[string]transform.SpecTransformer
	envConfig     *env.Config
	schema        *jsonschema.Schema
}

func (s *SchemaValidationTestSuite) SetupTest() {
	compiledSchema, err := loadPluginDocsSchema()
	s.Require().NoError(err)
	s.schema = compiledSchema

	listener := bufconn.Listen(1024 * 1024)
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
	executor.manager = container.Manager
	s.hostContainer = container
}

func (s *SchemaValidationTestSuite) TearDownTest() {
	s.hostContainer.CloseHostServer()
}

func (s *SchemaValidationTestSuite) TestProviderDocsMatchSchema() {
	s.validateGeneratedDocs("newstack-cloud/test")
}

func (s *SchemaValidationTestSuite) TestTransformerDocsMatchSchema() {
	s.validateGeneratedDocs("newstack-cloud/testTransform")
}

func (s *SchemaValidationTestSuite) TestFixturesMatchSchema() {
	fixtures := []string{
		"__testdata/provider-docs.json",
		"__testdata/transformer-docs.json",
	}
	for _, fixturePath := range fixtures {
		s.Run(fixturePath, func() {
			raw, err := os.ReadFile(fixturePath)
			s.Require().NoError(err)
			s.assertValidAgainstSchema(raw)
		})
	}
}

func (s *SchemaValidationTestSuite) TestInvalidDocsRejectedBySchema() {
	// Cardinality min must be a non-negative integer; -1 should fail.
	invalidDocs := &docgen.PluginDocs{
		ID:               "newstack-cloud/test",
		DisplayName:      "Test",
		Version:          "1.0.0",
		ProtocolVersions: []string{"1.0"},
		Description:      "desc",
		Author:           "Two Hundred",
		Repository:       "https://example.invalid/repo",
		Config: &docgen.PluginDocsVersionConfig{
			Fields:                map[string]*docgen.PluginDocsVersionConfigField{},
			AllowAdditionalFields: false,
		},
		Links: []*docgen.PluginDocsLink{
			{
				Type:                  "a::b",
				Summary:               "s",
				Description:           "d",
				AnnotationDefinitions: map[string]*docgen.PluginDocsLinkAnnotationDefinition{},
				CardinalityA: &docgen.PluginDocsLinkCardinality{
					Min: -1,
					Max: 0,
				},
			},
		},
	}

	raw, err := json.Marshal(invalidDocs)
	s.Require().NoError(err)

	var decoded any
	s.Require().NoError(json.Unmarshal(raw, &decoded))
	err = s.schema.Validate(decoded)
	s.Require().Error(err, "expected schema validation to reject negative cardinality min")
}

func (s *SchemaValidationTestSuite) validateGeneratedDocs(pluginID string) {
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

	raw, err := json.Marshal(pluginDocs)
	s.Require().NoError(err)
	s.assertValidAgainstSchema(raw)
}

func (s *SchemaValidationTestSuite) assertValidAgainstSchema(docJSON []byte) {
	var decoded any
	s.Require().NoError(json.Unmarshal(docJSON, &decoded))

	err := s.schema.Validate(decoded)
	if err != nil {
		s.Failf(
			"schema validation failed",
			"document did not conform to plugin-docs schema: %v",
			err,
		)
	}
}

// loadPluginDocsSchema reads the YAML schema, converts it to JSON-
// compatible data and compiles it with draft 2020-12 semantics.
func loadPluginDocsSchema() (*jsonschema.Schema, error) {
	schemaPath, err := schemaFilePath()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var yamlNode any
	if err := yaml.Unmarshal(raw, &yamlNode); err != nil {
		return nil, err
	}

	// The compiler needs JSON-compatible types (map[string]any, not
	// map[any]any that older yaml libs produced). yaml.v3 emits
	// map[string]any already, but nested maps in sequences may need
	// coercion in edge cases — handle defensively.
	coerced := normaliseYAMLForJSON(yamlNode)

	compiler := jsonschema.NewCompiler()
	const schemaURL = "https://bluelink.dev/schemas/plugin-docs.json"
	if err := compiler.AddResource(schemaURL, coerced); err != nil {
		return nil, err
	}
	return compiler.Compile(schemaURL)
}

func schemaFilePath() (string, error) {
	// Tests run from internal/testsuites — schema lives at repo-local
	// ../../schema/plugin-docs.schema.yaml.
	candidates := []string{
		filepath.Join("..", "..", "schema", "plugin-docs.schema.yaml"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

func normaliseYAMLForJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, v := range typed {
			out[key] = normaliseYAMLForJSON(v)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, v := range typed {
			out[toStringKey(key)] = normaliseYAMLForJSON(v)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			out[i] = normaliseYAMLForJSON(v)
		}
		return out
	default:
		return value
	}
}

func toStringKey(key any) string {
	if s, ok := key.(string); ok {
		return s
	}
	return strings.TrimSpace(
		strings.ReplaceAll(
			// fall back to default formatting for non-string keys —
			// should not occur for this schema.
			jsonMustMarshalString(key), "\"", "",
		),
	)
}

func jsonMustMarshalString(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func TestSchemaValidationTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaValidationTestSuite))
}
