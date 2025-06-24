package plugintestutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// NewTestLinkContext creates a provider.LinkContext
// for testing purposes.
func NewTestLinkContext(
	providerConfigMap map[string]map[string]*core.ScalarValue,
	contextVariables map[string]*core.ScalarValue,
) provider.LinkContext {
	params := core.NewDefaultParams(providerConfigMap, nil, contextVariables, nil)
	return provider.NewLinkContextFromParams(params)
}
