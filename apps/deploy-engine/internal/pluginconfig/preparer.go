package pluginconfig

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
)

// DefinitionProvider is an interface that defines any plugin
// type that provides a config definition schema.
type DefinitionProvider interface {
	ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error)
}

// Preparer provides an interface for a service that validates
// and prepares plugin-specific configuration against the plugin-provided
// config definition schema.
type Preparer interface {
	// Prepare validates and populates default values for the provided configuration
	// using the plugin-provided config definition schemas.
	// It returns diagnostics for validation errors,
	// and an error if something went wrong during validation and preparation.
	// If validate is set to false, this will skip validation
	// and only populate default values.
	Prepare(
		ctx context.Context,
		blueprintOpConfig *types.BlueprintOperationConfig,
		validate bool,
	) (*types.BlueprintOperationConfig, []*core.Diagnostic, error)
}

type preparerImpl struct {
	providers     map[string]DefinitionProvider
	transformers  map[string]DefinitionProvider
	pluginManager pluginservicev1.Manager
}

// NewDefaultPreparer creates a new default implementation of a service
// that validates and populates defaults for plugin-specific configuration using the
// plugin-provided config definition schemas.
func NewDefaultPreparer(
	providers map[string]DefinitionProvider,
	transformers map[string]DefinitionProvider,
	pluginManager pluginservicev1.Manager,
) Preparer {
	return &preparerImpl{
		providers:     providers,
		transformers:  transformers,
		pluginManager: pluginManager,
	}
}

func (p *preparerImpl) Prepare(
	ctx context.Context,
	blueprintOpConfig *types.BlueprintOperationConfig,
	validate bool,
) (*types.BlueprintOperationConfig, []*core.Diagnostic, error) {
	if blueprintOpConfig == nil {
		return nil, nil, nil
	}

	diagnostics := make([]*core.Diagnostic, 0)
	preparedBlueprintOpConfig := &types.BlueprintOperationConfig{
		Providers:          map[string]map[string]*core.ScalarValue{},
		Transformers:       map[string]map[string]*core.ScalarValue{},
		ContextVariables:   blueprintOpConfig.ContextVariables,
		BlueprintVariables: blueprintOpConfig.BlueprintVariables,
		Dependencies:       blueprintOpConfig.Dependencies,
	}

	for providerName, config := range blueprintOpConfig.Providers {
		preparedConfig, providerDiagnostics, err := p.validateAndPreparePluginConfig(
			ctx,
			providerName,
			"provider",
			config,
			validate,
		)
		if err != nil {
			return nil, nil, err
		}
		diagnostics = append(diagnostics, providerDiagnostics...)
		preparedBlueprintOpConfig.Providers[providerName] = preparedConfig
	}

	for transformerName, config := range blueprintOpConfig.Transformers {
		preparedConfig, transformerDiagnostics, err := p.validateAndPreparePluginConfig(
			ctx,
			transformerName,
			"transformer",
			config,
			validate,
		)
		if err != nil {
			return nil, nil, err
		}
		diagnostics = append(diagnostics, transformerDiagnostics...)
		preparedBlueprintOpConfig.Transformers[transformerName] = preparedConfig
	}

	depsDiagnonstics, err := p.validateDependencies(blueprintOpConfig.Dependencies)
	if err != nil {
		return nil, nil, err
	}
	diagnostics = append(diagnostics, depsDiagnonstics...)

	return preparedBlueprintOpConfig, diagnostics, nil
}

func (p *preparerImpl) validateAndPreparePluginConfig(
	ctx context.Context,
	pluginName string,
	pluginType string,
	config map[string]*core.ScalarValue,
	validate bool,
) (map[string]*core.ScalarValue, []*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}
	plugin, ok := p.getPluginDefinitionProvider(pluginName, pluginType)
	if !ok {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelWarning,
			Message: fmt.Sprintf(
				"%q is present in the configuration but the %q %s could not be found,"+
					" skipping %s config validation and preparation",
				pluginName,
				pluginName,
				pluginType,
				pluginType,
			),
			Range: defaultDiagnosticRange(),
		})
		return map[string]*core.ScalarValue{}, diagnostics, nil
	}

	configDef, err := plugin.ConfigDefinition(ctx)
	if err != nil {
		return nil, nil, err
	}

	if validate {
		diagnostics, err = core.ValidateConfigDefinition(
			pluginName,
			pluginType,
			config,
			configDef,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	if utils.HasAtLeastOneError(diagnostics) {
		return nil, diagnostics, nil
	}

	preparedConfig, err := core.PopulateDefaultConfigValues(
		config,
		configDef,
	)
	if err != nil {
		return nil, nil, err
	}

	return preparedConfig, diagnostics, nil
}

func (p *preparerImpl) validateDependencies(
	dependencies map[string]string,
) ([]*core.Diagnostic, error) {
	diagnostics := []*core.Diagnostic{}

	for pluginID, versionConstraint := range dependencies {
		pluginInstance := p.getPluginInstance(pluginID)
		if pluginInstance == nil {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level:   core.DiagnosticLevelError,
				Message: fmt.Sprintf("plugin %q is not installed", pluginID),
				Range:   defaultDiagnosticRange(),
			})
		}

		installedVersion := getPluginVersionFromInstance(pluginInstance)
		if installedVersion == "" {
			return nil, fmt.Errorf("failed to get installed version for plugin %q", pluginID)
		}

		if isCompatible, err := CheckPluginVersionCompatibility(
			installedVersion,
			versionConstraint,
		); err != nil {
			return nil, err
		} else if !isCompatible {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"plugin %q is installed with version %q which is "+
						"not compatible with the version constraint %q",
					pluginID,
					pluginInstance.Info.Metadata.PluginVersion,
					versionConstraint,
				),
				Range: defaultDiagnosticRange(),
			})
		}
	}

	return diagnostics, nil
}

func (p *preparerImpl) getPluginInstance(pluginID string) *pluginservicev1.PluginInstance {
	providerInstance := p.pluginManager.GetPlugin(
		pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
		pluginID,
	)

	if providerInstance == nil {
		return p.pluginManager.GetPlugin(
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER,
			pluginID,
		)
	}

	return providerInstance
}

func (p *preparerImpl) getPluginDefinitionProvider(
	pluginName string,
	pluginType string,
) (DefinitionProvider, bool) {
	switch pluginType {
	case "provider":
		provider, ok := p.providers[pluginName]
		return provider, ok
	case "transformer":
		transformer, ok := p.transformers[pluginName]
		return transformer, ok
	default:
		return nil, false
	}
}

func getPluginVersionFromInstance(pluginInstance *pluginservicev1.PluginInstance) string {
	if pluginInstance == nil ||
		pluginInstance.Info == nil ||
		pluginInstance.Info.Metadata == nil {
		return ""
	}

	return pluginInstance.Info.Metadata.PluginVersion
}

func defaultDiagnosticRange() *core.DiagnosticRange {
	return &core.DiagnosticRange{
		Start: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 1,
			},
		},
		End: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 1,
			},
		},
	}
}
