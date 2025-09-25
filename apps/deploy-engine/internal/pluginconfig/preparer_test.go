package pluginconfig

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/stretchr/testify/suite"
)

type PreparerSuite struct {
	preparer Preparer
	suite.Suite
}

func (s *PreparerSuite) SetupTest() {
	s.preparer = NewDefaultPreparer(
		testProviders(),
		testTransformers(),
		createTestPluginManager(),
	)
}

func (s *PreparerSuite) Test_validation_passes_and_populates_defaults() {
	preparedConfig, diagnostics, err := s.preparer.Prepare(
		context.Background(),
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":    core.ScalarFromInt(22),
					"floatField":  core.ScalarFromFloat(3.14),
					"boolField":   core.ScalarFromBool(true),
					"stringField": core.ScalarFromString("test"),
					// intFieldWithDefault should be populated with the default value.
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test"),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"test-transformer": {
					"intTransformerField": core.ScalarFromInt(22),
					// intFieldWithDefault should be populated with the default value.
				},
			},
			// Provide context vars and blueprint vars to ensure they are included
			// in the modified config.
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		/* validate */ true,
	)
	s.Require().NoError(err)
	s.Assert().Empty(diagnostics)
	s.Assert().Equal(
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":            core.ScalarFromInt(22),
					"floatField":          core.ScalarFromFloat(3.14),
					"boolField":           core.ScalarFromBool(true),
					"stringField":         core.ScalarFromString("test"),
					"intFieldWithDefault": core.ScalarFromInt(100), // Populated with default value
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test"),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"test-transformer": {
					"intTransformerField": core.ScalarFromInt(22),
					"intFieldWithDefault": core.ScalarFromInt(220), // Populated with default value
				},
			},
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		preparedConfig,
	)
}

func (s *PreparerSuite) Test_validation_returns_warning_diagnostics_for_missing_plugins() {
	// A warning should be returned if a plugin is missing but
	// there is a configuration map for it.
	preparedConfig, diagnostics, err := s.preparer.Prepare(
		context.Background(),
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":    core.ScalarFromInt(57),
					"floatField":  core.ScalarFromFloat(4.14),
					"boolField":   core.ScalarFromBool(false),
					"stringField": core.ScalarFromString("test2"),
					// intFieldWithDefault should be populated with the default value.
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test2"),
				},
				"missing-provider": {
					"intField": core.ScalarFromInt(22),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"missing-transformer": {
					"intTransformerField": core.ScalarFromInt(22),
				},
			},
			// Provide context vars and blueprint vars to ensure they are included
			// in the modified config.
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		/* validate */ true,
	)
	s.Require().NoError(err)
	s.Assert().Len(diagnostics, 2)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelWarning,
				Message: "\"missing-provider\" is present in the configuration but the " +
					"\"missing-provider\" provider could not be found, skipping provider config validation and preparation",
				Range: defaultDiagnosticRange(),
			},
			{
				Level: core.DiagnosticLevelWarning,
				Message: "\"missing-transformer\" is present in the configuration but the " +
					"\"missing-transformer\" transformer could not be found, skipping transformer config validation and preparation",
				Range: defaultDiagnosticRange(),
			},
		},
		diagnostics,
	)
	s.Assert().Equal(
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":            core.ScalarFromInt(57),
					"floatField":          core.ScalarFromFloat(4.14),
					"boolField":           core.ScalarFromBool(false),
					"stringField":         core.ScalarFromString("test2"),
					"intFieldWithDefault": core.ScalarFromInt(100), // Populated with default value
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test2"),
				},
				"missing-provider": {},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"missing-transformer": {},
			},
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		preparedConfig,
	)
}

func (s *PreparerSuite) Test_validation_returns_error_diagnostics_for_invalid_config() {
	_, diagnostics, err := s.preparer.Prepare(
		context.Background(),
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":    core.ScalarFromInt(79), // 79 is not an allowed value
					"floatField":  core.ScalarFromFloat(3.52),
					"boolField":   core.ScalarFromBool(false),
					"stringField": core.ScalarFromString("test3"),
					// intFieldWithDefault should be populated with the default value.
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test3"),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"test-transformer": {
					"intTransformerField": core.ScalarFromInt(103), // 103 is not an allowed value
				},
			},
			// Provide context vars and blueprint vars to ensure they are included
			// in the modified config.
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		/* validate */ true,
	)
	s.Require().NoError(err)
	s.Assert().Len(diagnostics, 2)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: "The \"test-provider\" provider configuration field " +
					"\"intField\" has an unexpected value 79.",
				Range: defaultDiagnosticRange(),
			},
			{
				Level: core.DiagnosticLevelError,
				Message: "The \"test-transformer\" transformer configuration field " +
					"\"intTransformerField\" has an unexpected value 103.",
				Range: defaultDiagnosticRange(),
			},
		},
		diagnostics,
	)
}

func (s *PreparerSuite) Test_skips_validation_and_populates_defaults() {
	preparedConfig, diagnostics, err := s.preparer.Prepare(
		context.Background(),
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":    core.ScalarFromInt(79), // 79 is not an allowed value
					"floatField":  core.ScalarFromFloat(3.52),
					"boolField":   core.ScalarFromBool(false),
					"stringField": core.ScalarFromString("test3"),
					// intFieldWithDefault should be populated with the default value.
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test3"),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"test-transformer": {
					"intTransformerField": core.ScalarFromInt(103), // 103 is not an allowed value
				},
			},
			// Provide context vars and blueprint vars to ensure they are included
			// in the modified config.
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		/* validate */ false,
	)
	s.Require().NoError(err)
	s.Assert().Empty(diagnostics)
	s.Assert().Equal(
		&types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"intField":            core.ScalarFromInt(79),
					"floatField":          core.ScalarFromFloat(3.52),
					"boolField":           core.ScalarFromBool(false),
					"stringField":         core.ScalarFromString("test3"),
					"intFieldWithDefault": core.ScalarFromInt(100), // Populated with default value
					"aws.config.regionKMSKeys.us-east-1.other.placeholder": core.ScalarFromString("test3"),
				},
			},
			Transformers: map[string]map[string]*core.ScalarValue{
				"test-transformer": {
					"intTransformerField": core.ScalarFromInt(103),
					"intFieldWithDefault": core.ScalarFromInt(220), // Populated with default value
				},
			},
			ContextVariables: map[string]*core.ScalarValue{
				"test-context-variable": core.ScalarFromString("test"),
			},
			BlueprintVariables: map[string]*core.ScalarValue{
				"test-blueprint-variable": core.ScalarFromString("test"),
			},
		},
		preparedConfig,
	)
}

func testProviders() map[string]DefinitionProvider {
	return map[string]DefinitionProvider{
		"test-provider": &testProvider{},
	}
}

func testTransformers() map[string]DefinitionProvider {
	return map[string]DefinitionProvider{
		"test-transformer": &testTransformer{},
	}
}

type testProvider struct{}

func (p *testProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		AllowAdditionalFields: false,
		Fields: map[string]*core.ConfigFieldDefinition{
			"intField": {
				Type:        core.ScalarTypeInteger,
				Label:       "Int Field",
				Description: "An integer field",
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromInt(10),
					core.ScalarFromInt(22),
					core.ScalarFromInt(45),
					core.ScalarFromInt(57),
				},
				Required: true,
			},
			"floatField": {
				Type:        core.ScalarTypeFloat,
				Label:       "Float Field",
				Description: "A float field",
				Required:    true,
			},
			"boolField": {
				Type:        core.ScalarTypeBool,
				Label:       "Bool Field",
				Description: "A boolean field",
				Required:    true,
			},
			"stringField": {
				Type:        core.ScalarTypeString,
				Label:       "String Field",
				Description: "A string field",
				Required:    true,
			},
			"aws.config.regionKMSKeys.<region>.other.<placeholder>": {
				Type:        core.ScalarTypeString,
				Label:       "AWS Region KMS Keys",
				Description: "AWS region KMS keys",
				Required:    true,
			},
			"intFieldWithDefault": {
				Type:         core.ScalarTypeInteger,
				Label:        "Int Field with Default",
				Description:  "An integer field with a default value",
				DefaultValue: core.ScalarFromInt(100),
			},
		},
	}, nil
}

type testTransformer struct{}

func (p *testTransformer) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		AllowAdditionalFields: true,
		Fields: map[string]*core.ConfigFieldDefinition{
			"intTransformerField": {
				Type:        core.ScalarTypeInteger,
				Label:       "Int Field",
				Description: "An integer field",
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromInt(10),
					core.ScalarFromInt(22),
					core.ScalarFromInt(45),
					core.ScalarFromInt(57),
				},
				Required: true,
			},
			"intFieldWithDefault": {
				Type:         core.ScalarTypeInteger,
				Label:        "Int Field with Default",
				Description:  "An integer field with a default value",
				DefaultValue: core.ScalarFromInt(220),
			},
		},
	}, nil
}

type testPluginManager struct {
	plugins map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginInstance
}

func createTestPluginManager() pluginservicev1.Manager {
	return &testPluginManager{
		plugins: map[pluginservicev1.PluginType]map[string]*pluginservicev1.PluginInstance{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER: {
				"test-provider": {
					Info: &pluginservicev1.PluginInstanceInfo{
						ID:         "newstack-cloud/test-provider",
						PluginType: pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
						Metadata: &pluginservicev1.PluginMetadata{
							PluginVersion:        "1.0.0",
							DisplayName:          "Test Provider",
							PlainTextDescription: "Test Provider",
							FormattedDescription: "Test Provider",
							RepositoryUrl:        "https://github.com/newstack-cloud/bluelink-test-provider",
							Author:               "Newstack Cloud",
						},
						InstanceID:       "1",
						ProtocolVersions: []string{"1.0"},
					},
				},
			},
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: {
				"test-transformer": {
					Info: &pluginservicev1.PluginInstanceInfo{
						ID:         "newstack-cloud/test-transformer",
						PluginType: pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER,
						Metadata: &pluginservicev1.PluginMetadata{
							PluginVersion:        "1.0.0",
							DisplayName:          "Test Transformer",
							PlainTextDescription: "Test Transformer",
							FormattedDescription: "Test Transformer",
							RepositoryUrl:        "https://github.com/newstack-cloud/bluelink-test-transformer",
							Author:               "Newstack Cloud",
						},
						InstanceID:       "2",
						ProtocolVersions: []string{"1.0"},
					},
				},
			},
		},
	}
}

func (m *testPluginManager) RegisterPlugin(
	info *pluginservicev1.PluginInstanceInfo,
) error {
	return nil
}

func (m *testPluginManager) DeregisterPlugin(
	pluginType pluginservicev1.PluginType,
	pluginID string,
) error {
	return nil
}

func (m *testPluginManager) GetPlugin(
	pluginType pluginservicev1.PluginType,
	pluginID string,
) *pluginservicev1.PluginInstance {
	return m.plugins[pluginType][pluginID]
}

func (m *testPluginManager) GetPluginMetadata(
	pluginType pluginservicev1.PluginType,
	pluginID string,
) *pluginservicev1.PluginExtendedMetadata {
	plugin := m.plugins[pluginType][pluginID]
	if plugin == nil {
		return nil
	}
	return &pluginservicev1.PluginExtendedMetadata{
		PluginVersion:        plugin.Info.Metadata.PluginVersion,
		DisplayName:          plugin.Info.Metadata.DisplayName,
		PlainTextDescription: plugin.Info.Metadata.PlainTextDescription,
		FormattedDescription: plugin.Info.Metadata.FormattedDescription,
		RepositoryUrl:        plugin.Info.Metadata.RepositoryUrl,
		Author:               plugin.Info.Metadata.Author,
		ProtocolVersions:     plugin.Info.ProtocolVersions,
	}
}

func (m *testPluginManager) GetPlugins(
	pluginType pluginservicev1.PluginType,
) []*pluginservicev1.PluginInstance {
	instances := []*pluginservicev1.PluginInstance{}
	for _, instance := range m.plugins[pluginType] {
		instances = append(instances, instance)
	}
	return instances
}

func TestPrepareSuite(t *testing.T) {
	suite.Run(t, new(PreparerSuite))
}
