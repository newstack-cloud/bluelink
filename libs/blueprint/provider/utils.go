package provider

import (
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// ExtractProviderFromItemType extracts the provider namespace from a resource type
// or data source type.
func ExtractProviderFromItemType(itemType string) string {
	parts := strings.Split(itemType, "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

// CalculateRetryWaitTimeMS calculates the wait time in milliseconds between retries
// based on a provided retry policy and current retry attempt.
func CalculateRetryWaitTimeMS(
	retryPolicy *RetryPolicy,
	currentRetryAttempt int,
) int {
	// Interval is configured in seconds, convert to milliseconds
	// to allow for millisecond precision for fractional backoff rates.
	intervalMS := float64(retryPolicy.FirstRetryDelay * 1000)
	computedWaitTimeMS := intervalMS * math.Pow(
		retryPolicy.BackoffFactor,
		float64(currentRetryAttempt-1),
	)

	if retryPolicy.MaxDelay != -1 {
		computedWaitTimeMS = math.Min(
			computedWaitTimeMS,
			float64(retryPolicy.MaxDelay*1000),
		)
	}

	if retryPolicy.Jitter {
		computedWaitTimeMS = rand.Float64() * computedWaitTimeMS
	}

	return int(math.Trunc(computedWaitTimeMS))
}

// CreateRetryContext creates a new retry context
// with state for the initial attempt with the provided retry policy.
func CreateRetryContext(policy *RetryPolicy) *RetryContext {
	return &RetryContext{
		Policy: policy,
		// Start at 0 for first attempt as retries are counted from 1.
		Attempt:            0,
		AttemptDurations:   []float64{},
		ExceededMaxRetries: false,
	}
}

// RetryContextWithStartTime creates a new retry context
// with the provided start time for the current attempt.
func RetryContextWithStartTime(retryCtx *RetryContext, startTime time.Time) *RetryContext {
	return &RetryContext{
		Attempt:            retryCtx.Attempt,
		ExceededMaxRetries: retryCtx.ExceededMaxRetries,
		Policy:             retryCtx.Policy,
		AttemptDurations:   retryCtx.AttemptDurations,
		AttemptStartTime:   startTime,
	}
}

// RetryContextWithNextAttempt creates a new retry context
// with state for the next retry attempt based on the provided retry context.
func RetryContextWithNextAttempt(
	retryCtx *RetryContext,
	currentAttemptDuration time.Duration,
) *RetryContext {
	nextAttempt := retryCtx.Attempt + 1
	return &RetryContext{
		Policy:  retryCtx.Policy,
		Attempt: nextAttempt,
		AttemptDurations: append(
			retryCtx.AttemptDurations,
			core.FractionalMilliseconds(currentAttemptDuration),
		),
		ExceededMaxRetries: nextAttempt > retryCtx.Policy.MaxRetries,
		AttemptStartTime:   retryCtx.AttemptStartTime,
	}
}

type providerCtxFromParams struct {
	providerNamespace string
	blueprintParams   core.BlueprintParams
}

// NewProviderContextFromParams creates a new provider context
// from a set of blueprint parameters for the current environment.
// The provider context will then be passed into provider plugins
// to allow them to access configuration values and context variables.
func NewProviderContextFromParams(
	providerNamespace string,
	blueprintParams core.BlueprintParams,
) Context {
	return &providerCtxFromParams{
		providerNamespace: providerNamespace,
		blueprintParams:   blueprintParams,
	}
}

func (p *providerCtxFromParams) ProviderConfigVariable(name string) (*core.ScalarValue, bool) {
	providerConfig := p.blueprintParams.ProviderConfig(p.providerNamespace)
	if providerConfig == nil {
		return nil, false
	}

	configValue, ok := providerConfig[name]
	return configValue, ok
}

func (p *providerCtxFromParams) ProviderConfigVariables() map[string]*core.ScalarValue {
	return p.blueprintParams.ProviderConfig(p.providerNamespace)
}

func (p *providerCtxFromParams) ContextVariable(name string) (*core.ScalarValue, bool) {
	contextVar := p.blueprintParams.ContextVariable(name)
	if contextVar == nil {
		return nil, false
	}
	return contextVar, true
}

func (p *providerCtxFromParams) ContextVariables() map[string]*core.ScalarValue {
	return p.blueprintParams.AllContextVariables()
}

type linkCtxFromParams struct {
	blueprintParams core.BlueprintParams
}

// NewLinkContextFromParams creates a new link context
// from a set of blueprint parameters for the current environment.
// The link context will then be passed into provider link plugins
// to allow them to access configuration values and context variables.
func NewLinkContextFromParams(
	blueprintParams core.BlueprintParams,
) LinkContext {
	return &linkCtxFromParams{
		blueprintParams: blueprintParams,
	}
}

func (p *linkCtxFromParams) ProviderConfigVariable(providerNamespace string, varName string) (*core.ScalarValue, bool) {
	providerConfig := p.blueprintParams.ProviderConfig(providerNamespace)
	if providerConfig == nil {
		return nil, false
	}

	configValue, ok := providerConfig[varName]
	return configValue, ok
}

func (p *linkCtxFromParams) ProviderConfigVariables(providerNamespace string) map[string]*core.ScalarValue {
	return p.blueprintParams.ProviderConfig(providerNamespace)
}

func (p *linkCtxFromParams) AllProviderConfigVariables() map[string]map[string]*core.ScalarValue {
	return p.blueprintParams.AllProvidersConfig()
}

func (p *linkCtxFromParams) ContextVariable(name string) (*core.ScalarValue, bool) {
	contextVar := p.blueprintParams.ContextVariable(name)
	if contextVar == nil {
		return nil, false
	}
	return contextVar, true
}

func (p *linkCtxFromParams) ContextVariables() map[string]*core.ScalarValue {
	return p.blueprintParams.AllContextVariables()
}

// NewProviderContextFromLinkContext creates a new provider context
// from a link context for the current environment.
// This is primarily useful for link plugin implementations that need to
// manage intermediary resource deployment by hooking into the `ResourceDeployService`
// to deploy resources registered with the blueprint framework host application.
func NewProviderContextFromLinkContext(
	linkCtx LinkContext,
	providerNamespace string,
) Context {
	return &providerCtxFromLinkCtx{
		linkCtx:           linkCtx,
		providerNamespace: providerNamespace,
	}
}

type providerCtxFromLinkCtx struct {
	linkCtx           LinkContext
	providerNamespace string
}

func (p *providerCtxFromLinkCtx) ProviderConfigVariable(name string) (*core.ScalarValue, bool) {
	return p.linkCtx.ProviderConfigVariable(p.providerNamespace, name)
}

func (p *providerCtxFromLinkCtx) ProviderConfigVariables() map[string]*core.ScalarValue {
	return p.linkCtx.ProviderConfigVariables(p.providerNamespace)
}

func (p *providerCtxFromLinkCtx) ContextVariable(name string) (*core.ScalarValue, bool) {
	return p.linkCtx.ContextVariable(name)
}

func (p *providerCtxFromLinkCtx) ContextVariables() map[string]*core.ScalarValue {
	return p.linkCtx.ContextVariables()
}
