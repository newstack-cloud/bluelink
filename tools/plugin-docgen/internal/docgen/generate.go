package docgen

import (
	"context"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	pfutils "github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/utils"
)

// GenerateProviderDocs generates documentation for a provider
// or transformer plugin.
func GeneratePluginDocs(
	pluginID string,
	pluginInstance any,
	manager pluginservicev1.Manager,
	envConfig *env.Config,
) (*PluginDocs, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(envConfig.GenerateTimeoutMS)*time.Millisecond,
	)
	defer cancel()

	params := utils.CreateBlueprintParams(envConfig)

	providerPlugin, isProvider := pluginInstance.(provider.Provider)
	if isProvider {
		return generateProviderDocs(ctx, pluginID, providerPlugin, manager, params)
	}

	transformerPlugin, isTransformer := pluginInstance.(transform.SpecTransformer)
	if isTransformer {
		return generateTransformerDocs(ctx, pluginID, transformerPlugin, manager, params)
	}

	return nil, ErrInvalidPluginType
}

func generateProviderDocs(
	ctx context.Context,
	pluginID string,
	providerPlugin provider.Provider,
	manager pluginservicev1.Manager,
	params core.BlueprintParams,
) (*PluginDocs, error) {
	namespace := pfutils.ExtractPluginNamespace(pluginID)

	metadata := manager.GetPluginMetadata(
		pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
		pluginID,
	)
	if metadata == nil {
		return nil, ErrPluginMetadataNotFound
	}

	configDocs, err := getProviderConfigDocs(
		ctx,
		providerPlugin,
	)
	if err != nil {
		return nil, err
	}

	docsForResources, err := getProviderResourcesDocs(
		ctx,
		namespace,
		providerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	docsForLinks, err := getProviderLinksDocs(
		ctx,
		providerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	docsForDataSources, err := getProviderDataSourcesDocs(
		ctx,
		namespace,
		providerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	docsForCustomVarTypes, err := getProviderCustomVarTypesDocs(
		ctx,
		namespace,
		providerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	docsForFunctions, err := getProviderFunctionsDocs(
		ctx,
		providerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocs{
		ID:               pluginID,
		DisplayName:      metadata.DisplayName,
		Version:          metadata.PluginVersion,
		ProtocolVersions: metadata.ProtocolVersions,
		Description:      getPluginMetadataDescription(metadata),
		Author:           metadata.Author,
		Repository:       metadata.RepositoryUrl,
		Config:           configDocs,
		Resources:        docsForResources,
		Links:            docsForLinks,
		DataSources:      docsForDataSources,
		CustomVarTypes:   docsForCustomVarTypes,
		Functions:        docsForFunctions,
	}, nil
}

func generateTransformerDocs(
	ctx context.Context,
	pluginID string,
	transformerPlugin transform.SpecTransformer,
	manager pluginservicev1.Manager,
	params core.BlueprintParams,
) (*PluginDocs, error) {
	namespace := pfutils.ExtractPluginNamespace(pluginID)

	metadata := manager.GetPluginMetadata(
		pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER,
		pluginID,
	)
	if metadata == nil {
		return nil, ErrPluginMetadataNotFound
	}

	transformName, err := transformerPlugin.GetTransformName(ctx)
	if err != nil {
		return nil, err
	}

	configDocs, err := getTransformerConfigDocs(
		ctx,
		transformerPlugin,
	)
	if err != nil {
		return nil, err
	}

	docsForAbstractResources, err := getTransformerAbstractResourcesDocs(
		ctx,
		namespace,
		transformerPlugin,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocs{
		ID:                pluginID,
		DisplayName:       metadata.DisplayName,
		Version:           metadata.PluginVersion,
		ProtocolVersions:  metadata.ProtocolVersions,
		Description:       getPluginMetadataDescription(metadata),
		Author:            metadata.Author,
		Repository:        metadata.RepositoryUrl,
		Config:            configDocs,
		TransformName:     transformName,
		AbstractResources: docsForAbstractResources,
	}, nil
}

func getPluginMetadataDescription(
	metadata *pluginservicev1.PluginExtendedMetadata,
) string {
	if strings.TrimSpace(metadata.FormattedDescription) != "" {
		return metadata.FormattedDescription
	}

	return metadata.PlainTextDescription
}

func getProviderConfigDocs(
	ctx context.Context,
	providerPlugin provider.Provider,
) (*PluginDocsVersionConfig, error) {
	configDefinition, err := providerPlugin.ConfigDefinition(ctx)
	if err != nil {
		return nil, err
	}

	return createConfigDocs(configDefinition)
}

func getTransformerConfigDocs(
	ctx context.Context,
	transformerPlugin transform.SpecTransformer,
) (*PluginDocsVersionConfig, error) {
	configDefinition, err := transformerPlugin.ConfigDefinition(ctx)
	if err != nil {
		return nil, err
	}

	return createConfigDocs(configDefinition)
}

func createConfigDocs(
	configDefinition *core.ConfigDefinition,
) (*PluginDocsVersionConfig, error) {
	fields := make(map[string]*PluginDocsVersionConfigField)
	for key, field := range configDefinition.Fields {
		fields[key] = &PluginDocsVersionConfigField{
			Type:          string(field.Type),
			Label:         field.Label,
			Description:   field.Description,
			Required:      field.Required,
			Default:       field.DefaultValue,
			AllowedValues: field.AllowedValues,
			Secret:        field.Secret,
			Examples:      field.Examples,
		}
	}

	return &PluginDocsVersionConfig{
		Fields:                fields,
		AllowAdditionalFields: configDefinition.AllowAdditionalFields,
	}, nil
}

func getProviderResourcesDocs(
	ctx context.Context,
	namespace string,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
) ([]*PluginDocsResource, error) {
	resourceTypes, err := providerPlugin.ListResourceTypes(ctx)
	if err != nil {
		return nil, err
	}

	docsForResources := make([]*PluginDocsResource, len(resourceTypes))
	for i, resourceType := range resourceTypes {
		resourceDocs, err := getProviderResourceDocs(
			ctx,
			namespace,
			providerPlugin,
			resourceType,
			params,
		)
		if err != nil {
			return nil, err
		}

		docsForResources[i] = resourceDocs
	}

	return docsForResources, nil
}

func getProviderLinksDocs(
	ctx context.Context,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
) ([]*PluginDocsLink, error) {
	linkTypes, err := providerPlugin.ListLinkTypes(ctx)
	if err != nil {
		return nil, err
	}

	docsForLinks := make([]*PluginDocsLink, len(linkTypes))
	for i, linkType := range linkTypes {
		linkDocs, err := getProviderLinkDocs(
			ctx,
			providerPlugin,
			linkType,
			params,
		)
		if err != nil {
			return nil, err
		}

		docsForLinks[i] = linkDocs
	}

	return docsForLinks, nil
}

func getProviderDataSourcesDocs(
	ctx context.Context,
	namespace string,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
) ([]*PluginDocsDataSource, error) {
	dataSourceTypes, err := providerPlugin.ListDataSourceTypes(ctx)
	if err != nil {
		return nil, err
	}

	docsForDataSources := make([]*PluginDocsDataSource, len(dataSourceTypes))
	for i, dataSourceType := range dataSourceTypes {
		dataSourceDocs, err := getProviderDataSourceDocs(
			ctx,
			namespace,
			providerPlugin,
			dataSourceType,
			params,
		)
		if err != nil {
			return nil, err
		}

		docsForDataSources[i] = dataSourceDocs
	}

	return docsForDataSources, nil
}

func getProviderCustomVarTypesDocs(
	ctx context.Context,
	namespace string,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
) ([]*PluginDocsCustomVarType, error) {
	customVarTypes, err := providerPlugin.ListCustomVariableTypes(ctx)
	if err != nil {
		return nil, err
	}

	docsForCustomVarTypes := []*PluginDocsCustomVarType{}
	for _, customVarType := range customVarTypes {
		customVarTypeDocs, err := getProviderCustomVarTypeDocs(
			ctx,
			namespace,
			providerPlugin,
			customVarType,
			params,
		)
		if err != nil {
			return nil, err
		}

		docsForCustomVarTypes = append(docsForCustomVarTypes, customVarTypeDocs)
	}

	return docsForCustomVarTypes, nil
}

func getProviderFunctionsDocs(
	ctx context.Context,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
) ([]*PluginDocsFunction, error) {
	functions, err := providerPlugin.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}

	docsForFunctions := make([]*PluginDocsFunction, len(functions))
	for i, function := range functions {
		functionDocs, err := getProviderFunctionDocs(
			ctx,
			providerPlugin,
			params,
			function,
		)
		if err != nil {
			return nil, err
		}

		docsForFunctions[i] = functionDocs
	}

	return docsForFunctions, nil
}

func getTransformerAbstractResourcesDocs(
	ctx context.Context,
	namespace string,
	transformerPlugin transform.SpecTransformer,
	params core.BlueprintParams,
) ([]*PluginDocsResource, error) {
	abstractResourceTypes, err := transformerPlugin.ListAbstractResourceTypes(ctx)
	if err != nil {
		return nil, err
	}

	docsForAbstractResources := make([]*PluginDocsResource, len(abstractResourceTypes))
	for i, abstractResourceType := range abstractResourceTypes {
		resourceDocs, err := getTransformerAbstractResourceDocs(
			ctx,
			namespace,
			transformerPlugin,
			abstractResourceType,
			params,
		)
		if err != nil {
			return nil, err
		}

		docsForAbstractResources[i] = resourceDocs
	}

	return docsForAbstractResources, nil
}
