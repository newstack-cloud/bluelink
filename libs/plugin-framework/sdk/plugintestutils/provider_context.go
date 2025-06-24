package plugintestutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// NewTestProviderContext creates a provider.Context for tests.
func NewTestProviderContext(
	providerName string,
	providerConfig map[string]*core.ScalarValue,
	contextVariables map[string]*core.ScalarValue,
) provider.Context {
	providerConfigMap := map[string]map[string]*core.ScalarValue{
		providerName: providerConfig,
	}
	params := core.NewDefaultParams(providerConfigMap, nil, contextVariables, nil)
	ctx := provider.NewProviderContextFromParams(providerName, params)
	return ctx
}
