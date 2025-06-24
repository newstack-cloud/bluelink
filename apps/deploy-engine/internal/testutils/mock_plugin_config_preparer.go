package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

type MockPluginConfigPreparer struct {
	// Mapping of a config value to a list of diagnostics
	// that should be returned when the given value is used.
	Fixtures map[string][]*core.Diagnostic
}

func NewMockPluginConfigPreparer(
	fixtures map[string][]*core.Diagnostic,
) *MockPluginConfigPreparer {
	return &MockPluginConfigPreparer{
		Fixtures: fixtures,
	}
}

func (p *MockPluginConfigPreparer) Prepare(
	ctx context.Context,
	blueprintOpConfig *types.BlueprintOperationConfig,
	validate bool,
) (*types.BlueprintOperationConfig, []*core.Diagnostic, error) {
	if blueprintOpConfig == nil {
		return nil, nil, nil
	}

	diagnostics := make([]*core.Diagnostic, 0)

	if !validate {
		return blueprintOpConfig, diagnostics, nil
	}

	for _, providerConfig := range blueprintOpConfig.Providers {
		providerDiagnostics := p.getFixtureDiagnostics(
			providerConfig,
		)
		diagnostics = append(diagnostics, providerDiagnostics...)
	}

	for _, transformerConfig := range blueprintOpConfig.Transformers {
		transformerDiagnostics := p.getFixtureDiagnostics(
			transformerConfig,
		)
		diagnostics = append(diagnostics, transformerDiagnostics...)
	}

	return blueprintOpConfig, diagnostics, nil
}

func (p *MockPluginConfigPreparer) getFixtureDiagnostics(
	config map[string]*core.ScalarValue,
) []*core.Diagnostic {
	diagnostics := make([]*core.Diagnostic, 0)
	for _, value := range config {
		stringValue := core.StringValueFromScalar(value)
		if fixtureDiagnostics, ok := p.Fixtures[stringValue]; ok {
			diagnostics = append(diagnostics, fixtureDiagnostics...)
		}
	}
	return diagnostics
}
