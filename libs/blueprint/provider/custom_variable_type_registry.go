package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

// CustomVariableTypeRegistry provides a way to get information about
// custom variable type plugins across multiple providers.
type CustomVariableTypeRegistry interface {
	// GetDescription returns the description of a custom variable type
	// in the registry.
	GetDescription(
		ctx context.Context,
		customVariableType string,
		input *CustomVariableTypeGetDescriptionInput,
	) (*CustomVariableTypeGetDescriptionOutput, error)

	// GetOptions returns the available options for a custom variable type.
	// Options are a fixed set of allowed values that behave like a runtime enum.
	GetOptions(
		ctx context.Context,
		customVariableType string,
		input *CustomVariableTypeOptionsInput,
	) (*CustomVariableTypeOptionsOutput, error)

	// HasCustomVariableType checks if a custom variable type is available in the registry.
	HasCustomVariableType(ctx context.Context, customVariableType string) (bool, error)

	// ListCustomVariableTypes retrieves a list of all the custom variable types avaiable
	// in the registry.
	ListCustomVariableTypes(ctx context.Context) ([]string, error)
}

type customVarTypeRegistryFromProviders struct {
	providers          map[string]Provider
	customVarTypeCache map[string]CustomVariableType
	customVarTypes     []string
}

// NewCustomVariableTypeRegistry creates a new CustomVariableTypeRegistry from a map of providers,
// matching against providers based on the data source type prefix.
func NewCustomVariableTypeRegistry(providers map[string]Provider) CustomVariableTypeRegistry {
	return &customVarTypeRegistryFromProviders{
		providers:          providers,
		customVarTypeCache: map[string]CustomVariableType{},
		customVarTypes:     []string{},
	}
}

func (r *customVarTypeRegistryFromProviders) GetDescription(
	ctx context.Context,
	customVariableType string,
	input *CustomVariableTypeGetDescriptionInput,
) (*CustomVariableTypeGetDescriptionOutput, error) {
	customVarTypeImpl, err := r.getCustomVariableType(ctx, customVariableType)
	if err != nil {
		return nil, err
	}

	return customVarTypeImpl.GetDescription(ctx, input)
}

func (r *customVarTypeRegistryFromProviders) GetOptions(
	ctx context.Context,
	customVariableType string,
	input *CustomVariableTypeOptionsInput,
) (*CustomVariableTypeOptionsOutput, error) {
	customVarTypeImpl, err := r.getCustomVariableType(ctx, customVariableType)
	if err != nil {
		return nil, err
	}

	return customVarTypeImpl.Options(ctx, input)
}

func (r *customVarTypeRegistryFromProviders) HasCustomVariableType(
	ctx context.Context,
	customVariableType string,
) (bool, error) {
	customVarTypeImpl, err := r.getCustomVariableType(ctx, customVariableType)
	if err != nil {
		if runErr, isRunErr := err.(*errors.RunError); isRunErr {
			if runErr.ReasonCode == ErrorReasonCodeProviderCustomVariableTypeNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return customVarTypeImpl != nil, nil
}

func (r *customVarTypeRegistryFromProviders) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	if len(r.customVarTypes) > 0 {
		return r.customVarTypes, nil
	}

	customVarTypes := []string{}
	for _, provider := range r.providers {
		types, err := provider.ListCustomVariableTypes(ctx)
		if err != nil {
			return nil, err
		}

		customVarTypes = append(customVarTypes, types...)
	}

	r.customVarTypes = customVarTypes

	return customVarTypes, nil
}

func (r *customVarTypeRegistryFromProviders) getCustomVariableType(
	ctx context.Context,
	customVariableType string,
) (CustomVariableType, error) {
	customVarType, cached := r.customVarTypeCache[customVariableType]
	if cached {
		return customVarType, nil
	}

	providerNamespace := ExtractProviderFromItemType(customVariableType)
	provider, ok := r.providers[providerNamespace]
	if !ok {
		return nil, errCustomVariableTypeProviderNotFound(providerNamespace, customVariableType)
	}
	customVarTypeImpl, err := provider.CustomVariableType(ctx, customVariableType)
	if err != nil {
		return nil, errProviderCustomVariableTypeNotFound(customVariableType, providerNamespace)
	}
	r.customVarTypeCache[customVariableType] = customVarTypeImpl

	return customVarTypeImpl, nil
}
