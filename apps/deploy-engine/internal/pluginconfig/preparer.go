package pluginconfig

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
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
	providers    map[string]DefinitionProvider
	transformers map[string]DefinitionProvider
}

// NewDefaultPreparer creates a new default implementation of a service
// that validates and populates defaults for plugin-specific configuration using the
// plugin-provided config definition schemas.
func NewDefaultPreparer(
	providers map[string]DefinitionProvider,
	transformers map[string]DefinitionProvider,
) Preparer {
	return &preparerImpl{
		providers:    providers,
		transformers: transformers,
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
	plugin, ok := p.getPlugin(pluginName, pluginType)
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

func (p *preparerImpl) getPlugin(
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
