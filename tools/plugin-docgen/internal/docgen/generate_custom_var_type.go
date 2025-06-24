package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func getProviderCustomVarTypeDocs(
	ctx context.Context,
	namespace string,
	providerPlugin provider.Provider,
	customVarType string,
	params core.BlueprintParams,
) (*PluginDocsCustomVarType, error) {
	customVariableType, err := providerPlugin.CustomVariableType(ctx, customVarType)
	if err != nil {
		return nil, err
	}

	typeInfo, err := customVariableType.GetType(
		ctx,
		&provider.CustomVariableTypeGetTypeInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	typeDescriptionOutput, err := customVariableType.GetDescription(
		ctx,
		&provider.CustomVariableTypeGetDescriptionInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	examplesOutput, err := customVariableType.GetExamples(
		ctx,
		&provider.CustomVariableTypeGetExamplesInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	options, err := getProviderOptionsDocs(
		ctx,
		namespace,
		customVariableType,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocsCustomVarType{
		Type:    typeInfo.Type,
		Label:   typeInfo.Label,
		Summary: getProviderCustomVarTypeSummary(typeDescriptionOutput),
		Description: getProviderCustomVarTypeDescription(
			typeDescriptionOutput,
		),
		Options: options,
		Examples: getProviderCustomVarTypeExamples(
			examplesOutput,
		),
	}, nil
}

func getProviderOptionsDocs(
	ctx context.Context,
	namespace string,
	customVariableType provider.CustomVariableType,
	params core.BlueprintParams,
) (map[string]*PluginDocsCustomVarTypeOption, error) {
	optionsOutput, err := customVariableType.Options(
		ctx,
		&provider.CustomVariableTypeOptionsInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	optionsDocs := map[string]*PluginDocsCustomVarTypeOption{}
	for value, option := range optionsOutput.Options {
		optionsDocs[value] = &PluginDocsCustomVarTypeOption{
			Label:       option.Label,
			Description: getProviderCustomVarTypeOptionDescription(option),
		}
	}

	return optionsDocs, nil
}

func getProviderCustomVarTypeOptionDescription(
	option *provider.CustomVariableTypeOption) string {
	if strings.TrimSpace(option.MarkdownDescription) != "" {
		return option.MarkdownDescription
	}

	return option.Description
}

func getProviderCustomVarTypeSummary(
	output *provider.CustomVariableTypeGetDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownSummary) != "" {
		return output.MarkdownSummary
	}

	if strings.TrimSpace(output.PlainTextSummary) != "" {
		return output.PlainTextSummary
	}

	return truncateDescription(getProviderCustomVarTypeDescription(output), 120)
}

func getProviderCustomVarTypeDescription(
	output *provider.CustomVariableTypeGetDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownDescription) != "" {
		return output.MarkdownDescription
	}

	return output.PlainTextDescription
}

func getProviderCustomVarTypeExamples(
	output *provider.CustomVariableTypeGetExamplesOutput,
) []string {
	if len(output.MarkdownExamples) > 0 {
		return output.MarkdownExamples
	}

	return output.PlainTextExamples
}
