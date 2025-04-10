package providerserverv1

import (
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/serialisation"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	"github.com/two-hundred/celerity/libs/plugin-framework/convertv1"
	sharedtypesv1 "github.com/two-hundred/celerity/libs/plugin-framework/sharedtypesv1"
)

func fromPBLinkIntermediaryResourcesCompleteResponse(
	response *UpdateLinkIntermediaryResourcesCompleteResponse,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	intermediaryResourceStates, err := fromPBLinkIntermediaryResourceStates(
		response.IntermediaryResourceStates,
	)
	if err != nil {
		return nil, err
	}

	linkData, err := serialisation.FromMappingNodePB(
		response.LinkData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateIntermediaryResourcesOutput{
		IntermediaryResourceStates: intermediaryResourceStates,
		LinkData:                   linkData,
	}, nil
}

func fromPBLinkIntermediaryResourceStates(
	intermediaryResourceStates []*LinkIntermediaryResourceState,
) ([]*state.LinkIntermediaryResourceState, error) {
	var states []*state.LinkIntermediaryResourceState
	for _, state := range intermediaryResourceStates {
		intermediaryResourceState, err := fromPBLinkIntermediaryResourceState(state)
		if err != nil {
			return nil, err
		}
		states = append(states, intermediaryResourceState)
	}
	return states, nil
}

func fromPBLinkIntermediaryResourceState(
	pbState *LinkIntermediaryResourceState,
) (*state.LinkIntermediaryResourceState, error) {
	resourceSpecData, err := serialisation.FromMappingNodePB(
		pbState.ResourceSpecData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &state.LinkIntermediaryResourceState{
		ResourceID: pbState.ResourceId,
		InstanceID: pbState.InstanceId,
		Status:     core.ResourceStatus(pbState.Status),
		PreciseStatus: core.PreciseResourceStatus(
			pbState.PreciseStatus,
		),
		LastDeployedTimestamp:      int(pbState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int(pbState.LastDeployAttemptTimestamp),
		ResourceSpecData:           resourceSpecData,
		FailureReasons:             pbState.FailureReasons,
	}, nil
}

func fromPBLinkPriorityResourceInfo(
	pbPriorityInfo *LinkPriorityResourceInfo,
) *provider.LinkGetPriorityResourceOutput {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResource(
			pbPriorityInfo.PriorityResource,
		),
		PriorityResourceType: convertv1.ResourceTypeToString(
			pbPriorityInfo.PriorityResourceType,
		),
	}
}

func fromPBTypeDescriptionForLink(
	typeDescription *sharedtypesv1.TypeDescription,
) *provider.LinkGetTypeDescriptionOutput {
	if typeDescription == nil {
		return nil
	}

	return &provider.LinkGetTypeDescriptionOutput{
		PlainTextDescription: typeDescription.PlainTextDescription,
		MarkdownDescription:  typeDescription.MarkdownDescription,
		PlainTextSummary:     typeDescription.PlainTextSummary,
		MarkdownSummary:      typeDescription.MarkdownSummary,
	}
}

func fromPBLinkAnnotationDefinitions(
	pbDefinitions *LinkAnnotationDefinitions,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	if pbDefinitions == nil {
		return nil, nil
	}

	annotations := make(map[string]*provider.LinkAnnotationDefinition)
	for key, pbAnnotation := range pbDefinitions.Definitions {
		annotation, err := fromPBLinkAnnotationDefinition(pbAnnotation)
		if err != nil {
			return nil, err
		}
		annotations[key] = annotation
	}

	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: annotations,
	}, nil
}

func fromPBLinkAnnotationDefinition(
	pbDefinition *LinkAnnotationDefinition,
) (*provider.LinkAnnotationDefinition, error) {
	if pbDefinition == nil {
		return nil, nil
	}

	defaultValue, err := serialisation.FromScalarValuePB(
		pbDefinition.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	allowedValues, err := convertv1.FromPBScalarSlice(pbDefinition.AllowedValues)
	if err != nil {
		return nil, err
	}

	examples, err := convertv1.FromPBScalarSlice(pbDefinition.Examples)
	if err != nil {
		return nil, err
	}

	return &provider.LinkAnnotationDefinition{
		Name:          pbDefinition.Name,
		Label:         pbDefinition.Label,
		Type:          convertv1.FromPBScalarType(pbDefinition.Type),
		Description:   pbDefinition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Examples:      examples,
		Required:      pbDefinition.Required,
	}, nil
}

func fromPBTypeDescriptionForDataSource(
	typeDescription *sharedtypesv1.TypeDescription,
) *provider.DataSourceGetTypeDescriptionOutput {
	if typeDescription == nil {
		return nil
	}

	return &provider.DataSourceGetTypeDescriptionOutput{
		PlainTextDescription: typeDescription.PlainTextDescription,
		MarkdownDescription:  typeDescription.MarkdownDescription,
		PlainTextSummary:     typeDescription.PlainTextSummary,
		MarkdownSummary:      typeDescription.MarkdownSummary,
	}
}

func fromPBDataSourceSpecDefinition(
	specDefinition *DataSourceSpecDefinition,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	if specDefinition == nil {
		return nil, nil
	}

	fields := make(map[string]*provider.DataSourceSpecSchema)
	for fieldName, pbFieldSchema := range specDefinition.Fields {
		field, err := fromPBDataSourceSpecSchema(pbFieldSchema)
		if err != nil {
			return nil, err
		}

		fields[fieldName] = field
	}

	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{
			Fields: fields,
		},
	}, nil
}

func fromPBDataSourceSpecSchema(
	pbFieldSchema *DataSourceSpecSchema,
) (*provider.DataSourceSpecSchema, error) {
	if pbFieldSchema == nil {
		return nil, nil
	}

	items, err := fromPBDataSourceSpecSchema(pbFieldSchema.Items)
	if err != nil {
		return nil, err
	}

	return &provider.DataSourceSpecSchema{
		Type:                 fromPBDataSourceSpecSchemaType(pbFieldSchema.Type),
		Label:                pbFieldSchema.Label,
		Description:          pbFieldSchema.Description,
		FormattedDescription: pbFieldSchema.FormattedDescription,
		Items:                items,
		Nullable:             pbFieldSchema.Nullable,
	}, nil
}

func fromPBDataSourceSpecSchemaType(pbFieldSchemaType DataSourceSpecSchemaType) provider.DataSourceSpecSchemaType {
	switch pbFieldSchemaType {
	case DataSourceSpecSchemaType_DATA_SOURCE_SPEC_INTEGER:
		return provider.DataSourceSpecTypeInteger
	case DataSourceSpecSchemaType_DATA_SOURCE_SPEC_FLOAT:
		return provider.DataSourceSpecTypeFloat
	case DataSourceSpecSchemaType_DATA_SOURCE_SPEC_BOOLEAN:
		return provider.DataSourceSpecTypeBoolean
	case DataSourceSpecSchemaType_DATA_SOURCE_SPEC_ARRAY:
		return provider.DataSourceSpecTypeArray
	default:
		return provider.DataSourceSpecTypeString
	}
}

func fromPBLinkKind(pbKind LinkKind) provider.LinkKind {
	if pbKind == LinkKind_LINK_KIND_SOFT {
		return provider.LinkKindSoft
	}

	return provider.LinkKindHard
}

func fromPBDataSourceFilterFields(
	pbFilterFields *DataSourceFilterFields,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	if pbFilterFields == nil {
		return nil, nil
	}

	return &provider.DataSourceGetFilterFieldsOutput{
		Fields: pbFilterFields.Fields,
	}, nil
}
