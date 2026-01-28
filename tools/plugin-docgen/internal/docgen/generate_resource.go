package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func getProviderResourceDocs(
	ctx context.Context,
	namespace string,
	providerPlugin provider.Provider,
	resourceType string,
	params core.BlueprintParams,
) (*PluginDocsResource, error) {
	resource, err := providerPlugin.Resource(ctx, resourceType)
	if err != nil {
		return nil, err
	}

	typeInfo, err := resource.GetType(
		ctx,
		&provider.ResourceGetTypeInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	typeDescriptionOutput, err := resource.GetTypeDescription(
		ctx,
		&provider.ResourceGetTypeDescriptionInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	examplesOutput, err := resource.GetExamples(
		ctx,
		&provider.ResourceGetExamplesInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	resourceSpec, err := getProviderResourceSpecDocs(
		ctx,
		namespace,
		resource,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocsResource{
		Type:    typeInfo.Type,
		Label:   typeInfo.Label,
		Summary: getProviderResourceSummary(typeDescriptionOutput),
		Description: getProviderResourceDescription(
			typeDescriptionOutput,
		),
		Specification: resourceSpec,
		Examples: getProviderResourceExamples(
			examplesOutput,
		),
	}, nil
}

func getProviderResourceSummary(
	output *provider.ResourceGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownSummary) != "" {
		return output.MarkdownSummary
	}

	if strings.TrimSpace(output.PlainTextSummary) != "" {
		return output.PlainTextSummary
	}

	return truncateDescription(getProviderResourceDescription(output), 120)
}

func getProviderResourceDescription(
	output *provider.ResourceGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownDescription) != "" {
		return output.MarkdownDescription
	}

	return output.PlainTextDescription
}

func getProviderResourceExamples(
	output *provider.ResourceGetExamplesOutput,
) []string {
	if len(output.MarkdownExamples) > 0 {
		return output.MarkdownExamples
	}

	return output.PlainTextExamples
}

// getSchemaDescription returns the formatted description if available,
// otherwise falls back to plain text description.
func getSchemaDescription(formatted, plain string) string {
	if strings.TrimSpace(formatted) != "" {
		return formatted
	}
	return plain
}

func getProviderResourceSpecDocs(
	ctx context.Context,
	namespace string,
	resource provider.Resource,
	params core.BlueprintParams,
) (*PluginDocResourceSpec, error) {
	specDefinitionOutput, err := resource.GetSpecDefinition(
		ctx,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: createProviderContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	spec := &PluginDocResourceSpec{
		Schema:              convertSpecSchema(specDefinitionOutput.SpecDefinition.Schema),
		IDField:             specDefinitionOutput.SpecDefinition.IDField,
		TaggingSupport:      taggingSupportToString(specDefinitionOutput.SpecDefinition.TaggingSupport),
		DestroyBeforeCreate: specDefinitionOutput.SpecDefinition.DestroyBeforeCreate,
	}

	return spec, nil
}

// taggingSupportToString converts TaggingSupport enum to string representation.
func taggingSupportToString(ts provider.TaggingSupport) string {
	switch ts {
	case provider.TaggingSupportFull:
		return "full"
	case provider.TaggingSupportLabels:
		return "labels"
	default:
		return ""
	}
}

func convertSpecSchema(
	schema *provider.ResourceDefinitionsSchema,
) *PluginDocResourceSpecSchema {
	if schema == nil {
		return nil
	}

	convertedSchema := &PluginDocResourceSpecSchema{
		Type:         string(schema.Type),
		Label:        schema.Label,
		Description:  getSchemaDescription(schema.FormattedDescription, schema.Description),
		Nullable:     schema.Nullable,
		Computed:     schema.Computed,
		MustRecreate: schema.MustRecreate,
		Default:      schema.Default,
		Examples:     schema.Examples,
		// Validation constraints
		Minimum:       schema.Minimum,
		Maximum:       schema.Maximum,
		MinLength:     schema.MinLength,
		MaxLength:     schema.MaxLength,
		Pattern:       schema.Pattern,
		AllowedValues: schema.AllowedValues,
		// Behavior flags
		Sensitive:        schema.Sensitive,
		IgnoreDrift:      schema.IgnoreDrift,
		TrackDrift:       schema.TrackDrift,
		SortArrayByField: schema.SortArrayByField,
	}

	if len(schema.Attributes) > 0 {
		convertedSchema.Attributes = make(map[string]*PluginDocResourceSpecSchema)
		for key, attr := range schema.Attributes {
			convertedSchema.Attributes[key] = convertSpecSchema(attr)
		}
		convertedSchema.Required = schema.Required
	}

	if schema.MapValues != nil {
		convertedSchema.MapValues = convertSpecSchema(schema.MapValues)
	}

	if schema.Items != nil {
		convertedSchema.Items = convertSpecSchema(schema.Items)
	}

	if len(schema.OneOf) > 0 {
		convertedSchema.OneOf = make([]*PluginDocResourceSpecSchema, len(schema.OneOf))
		for i, oneOfSchema := range schema.OneOf {
			convertedSchema.OneOf[i] = convertSpecSchema(oneOfSchema)
		}
	}

	return convertedSchema
}
