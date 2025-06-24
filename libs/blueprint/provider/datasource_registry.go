package provider

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

// DataSourceRegistry provides a way to retrieve data source plugins
// across multiple providers for tasks such as data source exports
// validation.
type DataSourceRegistry interface {
	// GetSpecDefinition returns the definition of a data source spec
	// in the registry that includes allowed parameters and return types.
	GetSpecDefinition(
		ctx context.Context,
		dataSourceType string,
		input *DataSourceGetSpecDefinitionInput,
	) (*DataSourceGetSpecDefinitionOutput, error)

	// GetFilterFields returns the fields that can be used in a filter for a data source.
	GetFilterFields(
		ctx context.Context,
		dataSourceType string,
		input *DataSourceGetFilterFieldsInput,
	) (*DataSourceGetFilterFieldsOutput, error)

	// GetTypeDescription returns the description of a data source type
	// in the registry.
	GetTypeDescription(
		ctx context.Context,
		dataSourceType string,
		input *DataSourceGetTypeDescriptionInput,
	) (*DataSourceGetTypeDescriptionOutput, error)

	// HasDataSourceType checks if a data source type is available in the registry.
	HasDataSourceType(ctx context.Context, dataSourceType string) (bool, error)

	// ListDataSourceTypes retrieves a list of all the data source types avaiable
	// in the registry.
	ListDataSourceTypes(ctx context.Context) ([]string, error)

	// CustomValidate allows for custom validation of a data source of a given type.
	CustomValidate(
		ctx context.Context,
		dataSourceType string,
		input *DataSourceValidateInput,
	) (*DataSourceValidateOutput, error)

	// Fetch retrieves the data from a data source using the provider
	// of the given type.
	Fetch(
		ctx context.Context,
		dataSourceType string,
		input *DataSourceFetchInput,
	) (*DataSourceFetchOutput, error)
}

type dataSourceRegistryFromProviders struct {
	providers       map[string]Provider
	dataSourceCache *core.Cache[DataSource]
	dataSourceTypes []string
	logger          core.Logger
	clock           core.Clock
	mu              sync.Mutex
}

// NewDataSourceRegistry creates a new DataSourceRegistry from a map of providers,
// matching against providers based on the data source type prefix.
func NewDataSourceRegistry(
	providers map[string]Provider,
	clock core.Clock,
	logger core.Logger,
) DataSourceRegistry {
	return &dataSourceRegistryFromProviders{
		providers:       providers,
		dataSourceCache: core.NewCache[DataSource](),
		dataSourceTypes: []string{},
		clock:           clock,
		logger:          logger,
	}
}

func (r *dataSourceRegistryFromProviders) GetSpecDefinition(
	ctx context.Context,
	dataSourceType string,
	input *DataSourceGetSpecDefinitionInput,
) (*DataSourceGetSpecDefinitionOutput, error) {
	dataSourceImpl, _, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		return nil, err
	}

	return dataSourceImpl.GetSpecDefinition(ctx, input)
}

func (r *dataSourceRegistryFromProviders) GetTypeDescription(
	ctx context.Context,
	dataSourceType string,
	input *DataSourceGetTypeDescriptionInput,
) (*DataSourceGetTypeDescriptionOutput, error) {
	dataSourceImpl, _, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		return nil, err
	}

	return dataSourceImpl.GetTypeDescription(ctx, input)
}

func (r *dataSourceRegistryFromProviders) GetFilterFields(
	ctx context.Context,
	dataSourceType string,
	input *DataSourceGetFilterFieldsInput,
) (*DataSourceGetFilterFieldsOutput, error) {
	dataSourceImpl, _, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		return nil, err
	}

	return dataSourceImpl.GetFilterFields(ctx, input)
}

func (r *dataSourceRegistryFromProviders) HasDataSourceType(ctx context.Context, dataSourceType string) (bool, error) {
	dataSourceImpl, _, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		if runErr, isRunErr := err.(*errors.RunError); isRunErr {
			if runErr.ReasonCode == ErrorReasonCodeProviderDataSourceTypeNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return dataSourceImpl != nil, nil
}

func (r *dataSourceRegistryFromProviders) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.dataSourceTypes) > 0 {
		return r.dataSourceTypes, nil
	}

	dataSourceTypes := []string{}
	for _, provider := range r.providers {
		types, err := provider.ListDataSourceTypes(ctx)
		if err != nil {
			return nil, err
		}

		dataSourceTypes = append(dataSourceTypes, types...)
	}

	r.dataSourceTypes = dataSourceTypes

	return dataSourceTypes, nil
}

func (r *dataSourceRegistryFromProviders) CustomValidate(
	ctx context.Context,
	dataSourceType string,
	input *DataSourceValidateInput,
) (*DataSourceValidateOutput, error) {
	dataSourceImpl, _, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		return nil, err
	}

	return dataSourceImpl.CustomValidate(ctx, input)
}

func (r *dataSourceRegistryFromProviders) Fetch(
	ctx context.Context,
	dataSourceType string,
	input *DataSourceFetchInput,
) (*DataSourceFetchOutput, error) {
	dataSourceImpl, dataSourceProvider, err := r.getDataSourceType(ctx, dataSourceType)
	if err != nil {
		return nil, err
	}

	fetchLogger := r.logger.Named("fetch").WithFields(
		core.StringLogField("dataSourceType", dataSourceType),
	)

	fetchLogger.Debug(
		"Loading retry policy for data source provider",
	)
	policy, err := r.getRetryPolicy(
		ctx,
		dataSourceProvider,
		DefaultRetryPolicy,
	)
	if err != nil {
		fetchLogger.Debug(
			"Failed to load retry policy for data source provider",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	fetchLogger.Info(
		"Fetching data from data source",
	)
	retryCtx := CreateRetryContext(policy)
	return r.fetch(
		ctx,
		dataSourceImpl,
		input,
		retryCtx,
		fetchLogger,
	)
}

func (r *dataSourceRegistryFromProviders) fetch(
	ctx context.Context,
	dataSource DataSource,
	input *DataSourceFetchInput,
	retryCtx *RetryContext,
	fetchLogger core.Logger,
) (*DataSourceFetchOutput, error) {
	fetchStartTime := r.clock.Now()
	fetchOutput, err := dataSource.Fetch(ctx, input)
	if err != nil {
		if IsRetryableError(err) {
			fetchLogger.Debug(
				"retryable error occurred while attempting to fetch data from data source",
				core.IntegerLogField("attempt", int64(retryCtx.Attempt)),
				core.ErrorLogField("error", err),
			)

			return r.handleFetchRetry(
				ctx,
				dataSource,
				input,
				RetryContextWithStartTime(
					retryCtx,
					fetchStartTime,
				),
				fetchLogger,
			)
		}

		return nil, err
	}

	return fetchOutput, nil
}

func (r *dataSourceRegistryFromProviders) handleFetchRetry(
	ctx context.Context,
	dataSource DataSource,
	input *DataSourceFetchInput,
	retryCtx *RetryContext,
	fetchLogger core.Logger,
) (*DataSourceFetchOutput, error) {
	currentAttemptDuration := r.clock.Since(
		retryCtx.AttemptStartTime,
	)
	nextRetryCtx := RetryContextWithNextAttempt(retryCtx, currentAttemptDuration)

	if !nextRetryCtx.ExceededMaxRetries {
		waitTimeMs := CalculateRetryWaitTimeMS(nextRetryCtx.Policy, nextRetryCtx.Attempt)
		time.Sleep(time.Duration(waitTimeMs) * time.Millisecond)
		return r.fetch(
			ctx,
			dataSource,
			input,
			nextRetryCtx,
			fetchLogger,
		)
	}

	fetchLogger.Debug(
		"fetching data from data source failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryCtx.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryCtx.Policy.MaxRetries)),
	)

	return nil, nil
}

func (r *dataSourceRegistryFromProviders) getRetryPolicy(
	ctx context.Context,
	dataSourceProvider Provider,
	defaultRetryPolicy *RetryPolicy,
) (*RetryPolicy, error) {
	retryPolicy, err := dataSourceProvider.RetryPolicy(ctx)
	if err != nil {
		return nil, err
	}

	if retryPolicy == nil {
		return defaultRetryPolicy, nil
	}

	return retryPolicy, nil
}

func (r *dataSourceRegistryFromProviders) getDataSourceType(
	ctx context.Context,
	dataSourceType string,
) (DataSource, Provider, error) {
	dataSource, cached := r.dataSourceCache.Get(dataSourceType)
	if cached {
		return dataSource, nil, nil
	}

	providerNamespace := ExtractProviderFromItemType(dataSourceType)
	provider, ok := r.providers[providerNamespace]
	if !ok {
		return nil, nil, errDataSourceTypeProviderNotFound(providerNamespace, dataSourceType)
	}
	dataSourceImpl, err := provider.DataSource(ctx, dataSourceType)
	if err != nil {
		return nil, nil, errProviderDataSourceTypeNotFound(dataSourceType, providerNamespace)
	}
	r.dataSourceCache.Set(dataSourceType, dataSourceImpl)

	return dataSourceImpl, provider, nil
}
