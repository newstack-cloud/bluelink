package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Provider is the interface for an implementation of a provider
// of a set of resource and data source types that can be used in a blueprint.
// An example of a provider could be a cloud provider such as AWS
// or Google Cloud.
//
// When we have links between resources of different providers, a new provider
// implementation should be implemented to act as a bridge between the two providers
// the linked resources belong to.
type Provider interface {
	// Namespace retrieves the resource type prefix that is used as the namespace
	// for all resource types in the provider.
	// (e.g. "aws" for AWS resources such as "aws/lambda/function")
	Namespace(ctx context.Context) (string, error)
	// ConfigDefinition retrieves a detailed definition of the
	// configuration that is required for the provider.
	ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error)
	// Resource retrieves a resource plugin to handle a resource in a blueprint for
	// a given resource type.
	Resource(ctx context.Context, resourceType string) (Resource, error)
	// DataSource retrieves a data source plugin to handle a data source in a blueprint
	// for a given data source type.
	DataSource(ctx context.Context, dataSourceType string) (DataSource, error)
	// Link retrieves a link plugin to handle a link between two resource types
	// in a blueprint.
	Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (Link, error)
	// CustomVariableType retrieves a custom variable type plugin to handle validating
	// convenience variable types with a (usually large) fixed set of possible values.
	// These custom variable types should not be used for dynamically sourced values
	// external to a blueprint, data sources exist for that purpose.
	CustomVariableType(ctx context.Context, customVariableType string) (CustomVariableType, error)
	// Function retrieves a function plugin that provides custom pure functions for blueprint
	// substitutions "${..}".
	// Functions are global and which providers are to be used for which functions
	// should be configured during initialisation of an application using the framework.
	// The core functions that are defined in the blueprint specification can not be overridden
	// by a provider.
	Function(ctx context.Context, functionName string) (Function, error)
	// ListResourceTypes retrieves a list of all the resource types that are provided by the
	// provider. This is primarily used in tools and documentation to provide a list of
	// available resource types.
	ListResourceTypes(ctx context.Context) ([]string, error)
	// ListLinkTypes retrieves a list of all the link types that are provided by the
	// provider. This is primarily used in tools and documentation to provide a list of
	// available link types.
	ListLinkTypes(ctx context.Context) ([]string, error)
	// ListDataSourceTypes retrieves a list of all the data source types that are provided by the
	// provider. This is primarily used in tools and documentation to provide a list of
	// available data source types.
	ListDataSourceTypes(ctx context.Context) ([]string, error)
	// ListCustomVariableTypes retrieves a list of all the custom variable types that are provided by the
	// provider. This is primarily used in tools and documentation to provide a list of
	// available custom variable types.
	ListCustomVariableTypes(ctx context.Context) ([]string, error)
	// ListFunctions retrieves a list of all the function names that are provided by the
	// provider. This is primarily used to assign the correct provider to a function
	// as functions are globally named. When multiple providers provide the same function,
	// an error should be reported during initialisation.
	ListFunctions(ctx context.Context) ([]string, error)
	// RetryPolicy retrieves the retry policy that should be used for the provider
	// for resource, link and data source operations.
	// The retry policy will be applied for resources when deploying, updating and removing
	// resources, for links when creating and removing links and for data sources when
	// querying the upstream data source.
	// The retry behaviour only kicks in when the provider resource, data source or link
	// implementation returns an error of type `provider.RetryableError`,
	// in which case the retry policy will be applied.
	// A retry policy is optional and if not provided, a default retry policy
	// provided by the host tool will be used.
	RetryPolicy(ctx context.Context) (*RetryPolicy, error)
}

// Context provides access to information about the current provider
// and environment that a provider plugin is running in.
// This is not to be confused with the conventional Go context.Context
// used for setting deadlines, cancelling requests and storing request-scoped
// values in a Go program.
type Context interface {
	// ProviderConfigVariable retrieves a configuration value that was loaded
	// for the current provider.
	ProviderConfigVariable(name string) (*core.ScalarValue, bool)
	// ProviderConfigVariables retrieves all the configuration values that were loaded
	// for the current provider.
	// This is useful to export all the configuration values to be sent to plugins
	// that are running in a different process.
	ProviderConfigVariables() map[string]*core.ScalarValue
	// ContextVariable retrieves a context-wide variable
	// for the current environment, this differs from values extracted
	// from context.Context, as these context variables are specific
	// to the components that implement the interfaces of the blueprint library
	// and can be shared between processes over a network or similar.
	ContextVariable(name string) (*core.ScalarValue, bool)
	// ContextVariables retrieves all the context-wide variables
	// for the current environment.
	// This is useful to export all the context-wide variables to be sent to plugins
	// that are running in a different process.
	ContextVariables() map[string]*core.ScalarValue
}

// RetryPolicy defines the retry policy that should be used for the provider
// for resource, link and data source operations.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retries that should be attempted
	// for a resource, link or data source operation.
	// If MaxRetries is 0, no retries should be attempted.
	MaxRetries int `json:"maxRetries"`
	// FirstRetryDelay is the delay in seconds that should be used before the first retry
	// attempt.
	// Fractional seconds are supported.
	FirstRetryDelay float64 `json:"firstRetryDelay"`
	// MaxDelay represents the maximum interval in seconds to wait between retries.
	// If -1 is provided, no maximum delay is enforced.
	// Fractional seconds are supported.
	MaxDelay float64 `json:"maxDelay"`
	// BackoffFactor is the factor that should be used to calculate the backoff
	// time between retries.
	// This AWS blog post from 2015 provides a good insight into how exponential backoff works:
	// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	BackoffFactor float64 `json:"backoffFactor"`
	// Jitter is a boolean value that determines whether to apply jitter to the retry interval.
	// This AWS blog post from 2015 provides a good insight into how jitter works:
	// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	Jitter bool `json:"jitter"`
}

// DefaultRetryPolicy is the default retry policy that can be used when a provider
// does not provide a custom retry policy.
var DefaultRetryPolicy = &RetryPolicy{
	MaxRetries: 5,
	// The first retry delay is 2 seconds
	FirstRetryDelay: 2,
	// The maximum delay between retries is 300 seconds (5 minutes)
	MaxDelay:      300,
	BackoffFactor: 2,
	Jitter:        true,
}
