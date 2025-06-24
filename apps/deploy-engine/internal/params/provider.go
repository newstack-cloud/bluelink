package params

import (
	"maps"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Provider is an interface for a service that produces
// `core.BlueprintParams` to be passed into the blueprint loader
// for validation, change staging, and deployment.
// This merges deploy engine context with caller-provided configuration.
type Provider interface {
	// GetDefaultParams returns the default set of parameters
	// containing context for the current version of the deploy engine.
	GetDefaultParams() core.BlueprintParams
	// CreateFromRequestConfig creates a set of parameters
	// that merges the provided configuration derived from a request to
	// the deploy engine with the default parameters containing context
	// for the current version of the deploy engine.
	CreateFromRequestConfig(
		reqConfig *types.BlueprintOperationConfig,
	) core.BlueprintParams
}

type providerImpl struct {
	defaultContext map[string]*core.ScalarValue
}

// NewDefaultProvider creates a new instance of the default params provider.
// This will include the provided default context values in the context variables
// that can be accessed through the `params.ContextVariable("key")` method
// in each set of blueprint parameters produced by the provider.
func NewDefaultProvider(defaultContext map[string]*core.ScalarValue) Provider {
	return &providerImpl{
		defaultContext: defaultContext,
	}
}

func (p *providerImpl) GetDefaultParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		p.defaultContext,
		map[string]*core.ScalarValue{},
	)
}

func (p *providerImpl) CreateFromRequestConfig(
	reqConfig *types.BlueprintOperationConfig,
) core.BlueprintParams {
	if reqConfig == nil {
		return p.GetDefaultParams()
	}

	return core.NewDefaultParams(
		reqConfig.Providers,
		reqConfig.Transformers,
		mergeVars(p.defaultContext, reqConfig.ContextVariables),
		reqConfig.BlueprintVariables,
	)
}

func mergeVars(
	defaultVars map[string]*core.ScalarValue,
	reqVars map[string]*core.ScalarValue,
) map[string]*core.ScalarValue {
	merged := make(map[string]*core.ScalarValue)
	maps.Copy(merged, defaultVars)
	maps.Copy(merged, reqVars)
	return merged
}
