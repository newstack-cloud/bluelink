package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// MockProvider is a mock implementation of the `provider.Provider` interface
// for plugins in launch testing.
type MockProvider struct {
	ProviderNamespace string
}

func (m *MockProvider) Namespace(ctx context.Context) (string, error) {
	return m.ProviderNamespace, nil
}

func (m *MockProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{}, nil
}

func (m *MockProvider) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	return nil, nil
}

func (m *MockProvider) DataSource(
	ctx context.Context,
	dataSourceType string,
) (provider.DataSource, error) {
	return nil, nil
}

func (m *MockProvider) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (provider.Link, error) {
	return nil, nil
}

func (m *MockProvider) CustomVariableType(
	ctx context.Context,
	customVariableType string,
) (provider.CustomVariableType, error) {
	return nil, nil
}

func (m *MockProvider) Function(
	ctx context.Context,
	functionName string,
) (provider.Function, error) {
	return nil, nil
}

func (m *MockProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockProvider) ListFunctions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockProvider) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}
