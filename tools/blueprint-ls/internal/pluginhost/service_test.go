package pluginhost

import (
	"context"
	"net"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite
}

func (s *ServiceSuite) TestLoadDefaultService_creates_service() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer listener.Close()

	config := &mockConfig{
		pluginPath:              "",
		launchWaitTimeoutMS:     5000,
		totalLaunchWaitTimeout:  60000,
		enabled:                 true,
	}

	service, err := LoadDefaultService(
		&LoadDependencies{
			Executor:         &mockExecutor{},
			InstanceFactory:  mockInstanceFactory,
			PluginHostConfig: config,
		},
		WithPluginServiceListener(listener),
		WithServiceFS(afero.NewMemMapFs()),
		WithServiceLogger(core.NewNopLogger()),
	)

	s.Require().NoError(err)
	s.Require().NotNil(service)
	s.Assert().NotNil(service.Manager())

	service.Close()
}

func (s *ServiceSuite) TestLoadDefaultService_with_initial_providers() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer listener.Close()

	config := &mockConfig{
		pluginPath:              "",
		launchWaitTimeoutMS:     5000,
		totalLaunchWaitTimeout:  60000,
		enabled:                 true,
	}

	initialProviders := map[string]provider.Provider{
		"core": &mockProvider{namespace: "core"},
	}

	service, err := LoadDefaultService(
		&LoadDependencies{
			Executor:         &mockExecutor{},
			InstanceFactory:  mockInstanceFactory,
			PluginHostConfig: config,
		},
		WithPluginServiceListener(listener),
		WithServiceFS(afero.NewMemMapFs()),
		WithInitialProviders(initialProviders),
	)

	s.Require().NoError(err)
	s.Require().NotNil(service)

	service.Close()
}

func (s *ServiceSuite) TestService_close_is_idempotent() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer listener.Close()

	config := &mockConfig{
		pluginPath:              "",
		launchWaitTimeoutMS:     5000,
		totalLaunchWaitTimeout:  60000,
		enabled:                 true,
	}

	service, err := LoadDefaultService(
		&LoadDependencies{
			Executor:         &mockExecutor{},
			InstanceFactory:  mockInstanceFactory,
			PluginHostConfig: config,
		},
		WithPluginServiceListener(listener),
		WithServiceFS(afero.NewMemMapFs()),
	)

	s.Require().NoError(err)

	// Close multiple times should not panic
	service.Close()
	service.Close()
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

// Mock implementations

type mockConfig struct {
	pluginPath             string
	logFileRootDir         string
	launchWaitTimeoutMS    int
	totalLaunchWaitTimeout int
	enabled                bool
}

func (c *mockConfig) GetPluginPath() string {
	return c.pluginPath
}

func (c *mockConfig) GetLogFileRootDir() string {
	return c.logFileRootDir
}

func (c *mockConfig) GetLaunchWaitTimeoutMS() int {
	return c.launchWaitTimeoutMS
}

func (c *mockConfig) GetTotalLaunchWaitTimeoutMS() int {
	return c.totalLaunchWaitTimeout
}

func (c *mockConfig) IsEnabled() bool {
	return c.enabled
}

type mockExecutor struct{}

func (e *mockExecutor) Execute(pluginID string, pluginPath string) (plugin.PluginProcess, error) {
	return &mockProcess{}, nil
}

type mockProcess struct{}

func (p *mockProcess) Kill() error {
	return nil
}

func mockInstanceFactory(
	info *pluginservicev1.PluginInstanceInfo,
	hostID string,
) (any, func(), error) {
	return &mockProvider{}, func() {}, nil
}

type mockProvider struct {
	namespace string
}

func (p *mockProvider) Namespace(_ context.Context) (string, error) {
	return p.namespace, nil
}

func (p *mockProvider) ConfigDefinition(_ context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *mockProvider) Resource(_ context.Context, _ string) (provider.Resource, error) {
	return nil, nil
}

func (p *mockProvider) DataSource(_ context.Context, _ string) (provider.DataSource, error) {
	return nil, nil
}

func (p *mockProvider) Link(_ context.Context, _, _ string) (provider.Link, error) {
	return nil, nil
}

func (p *mockProvider) CustomVariableType(_ context.Context, _ string) (provider.CustomVariableType, error) {
	return nil, nil
}

func (p *mockProvider) Function(_ context.Context, _ string) (provider.Function, error) {
	return nil, nil
}

func (p *mockProvider) ListResourceTypes(_ context.Context) ([]string, error) {
	return nil, nil
}

func (p *mockProvider) ListLinkTypes(_ context.Context) ([]string, error) {
	return nil, nil
}

func (p *mockProvider) ListDataSourceTypes(_ context.Context) ([]string, error) {
	return nil, nil
}

func (p *mockProvider) ListCustomVariableTypes(_ context.Context) ([]string, error) {
	return nil, nil
}

func (p *mockProvider) ListFunctions(_ context.Context) ([]string, error) {
	return nil, nil
}

func (p *mockProvider) RetryPolicy(_ context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}
