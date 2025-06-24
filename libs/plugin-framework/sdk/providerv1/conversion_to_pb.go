package providerv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

func toProviderNamespaceResponse(namespace string) *providerserverv1.NamespaceResponse {
	return &providerserverv1.NamespaceResponse{
		Response: &providerserverv1.NamespaceResponse_Namespace{
			Namespace: &providerserverv1.Namespace{
				Namespace: namespace,
			},
		},
	}
}

func toProviderNamespaceErrorResponse(err error) *providerserverv1.NamespaceResponse {
	return &providerserverv1.NamespaceResponse{
		Response: &providerserverv1.NamespaceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toResourceTypesResponse(resourceTypes []string) *providerserverv1.ResourceTypesResponse {
	return &providerserverv1.ResourceTypesResponse{
		Response: &providerserverv1.ResourceTypesResponse_ResourceTypes{
			ResourceTypes: &providerserverv1.ResourceTypes{
				ResourceTypes: convertv1.ToPBResourceTypes(resourceTypes),
			},
		},
	}
}

func toResourceTypesErrorResponse(err error) *providerserverv1.ResourceTypesResponse {
	return &providerserverv1.ResourceTypesResponse{
		Response: &providerserverv1.ResourceTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toLinkTypesResponse(linkTypes []string) *providerserverv1.LinkTypesResponse {
	return &providerserverv1.LinkTypesResponse{
		Response: &providerserverv1.LinkTypesResponse_LinkTypes{
			LinkTypes: &providerserverv1.LinkTypes{
				LinkTypes: commoncore.Map(
					linkTypes,
					func(linkType string, _ int) *providerserverv1.LinkType {
						return &providerserverv1.LinkType{
							Type: linkType,
						}
					},
				),
			},
		},
	}
}

func toLinkTypesErrorResponse(err error) *providerserverv1.LinkTypesResponse {
	return &providerserverv1.LinkTypesResponse{
		Response: &providerserverv1.LinkTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toDataSourceTypesResponse(dataSourceTypes []string) *providerserverv1.DataSourceTypesResponse {
	return &providerserverv1.DataSourceTypesResponse{
		Response: &providerserverv1.DataSourceTypesResponse_DataSourceTypes{
			DataSourceTypes: &providerserverv1.DataSourceTypes{
				DataSourceTypes: commoncore.Map(
					dataSourceTypes,
					func(dataSourceType string, _ int) *providerserverv1.DataSourceType {
						return &providerserverv1.DataSourceType{
							Type: dataSourceType,
						}
					},
				),
			},
		},
	}
}

func toDataSourceTypesErrorResponse(err error) *providerserverv1.DataSourceTypesResponse {
	return &providerserverv1.DataSourceTypesResponse{
		Response: &providerserverv1.DataSourceTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toCustomVariableTypesResponse(customVariableTypes []string) *providerserverv1.CustomVariableTypesResponse {
	return &providerserverv1.CustomVariableTypesResponse{
		Response: &providerserverv1.CustomVariableTypesResponse_CustomVariableTypes{
			CustomVariableTypes: &providerserverv1.CustomVariableTypes{
				CustomVariableTypes: commoncore.Map(
					customVariableTypes,
					func(customVariableType string, _ int) *providerserverv1.CustomVariableType {
						return &providerserverv1.CustomVariableType{
							Type: customVariableType,
						}
					},
				),
			},
		},
	}
}

func toCustomVariableTypesErrorResponse(err error) *providerserverv1.CustomVariableTypesResponse {
	return &providerserverv1.CustomVariableTypesResponse{
		Response: &providerserverv1.CustomVariableTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toFunctionsResponse(functions []string) *providerserverv1.FunctionListResponse {
	return &providerserverv1.FunctionListResponse{
		Response: &providerserverv1.FunctionListResponse_FunctionList{
			FunctionList: &providerserverv1.FunctionList{
				Functions: functions,
			},
		},
	}
}

func toFunctionsErrorResponse(err error) *providerserverv1.FunctionListResponse {
	return &providerserverv1.FunctionListResponse{
		Response: &providerserverv1.FunctionListResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toRetryPolicyResponse(policy *provider.RetryPolicy) *providerserverv1.RetryPolicyResponse {
	if policy == nil {
		return &providerserverv1.RetryPolicyResponse{
			Response: &providerserverv1.RetryPolicyResponse_RetryPolicy{
				RetryPolicy: nil,
			},
		}
	}

	return &providerserverv1.RetryPolicyResponse{
		Response: &providerserverv1.RetryPolicyResponse_RetryPolicy{
			RetryPolicy: &providerserverv1.RetryPolicy{
				MaxRetries:      int32(policy.MaxRetries),
				FirstRetryDelay: policy.FirstRetryDelay,
				MaxDelay:        policy.MaxDelay,
				BackoffFactor:   policy.BackoffFactor,
				Jitter:          policy.Jitter,
			},
		},
	}
}

func toRetryPolicyErrorResponse(err error) *providerserverv1.RetryPolicyResponse {
	return &providerserverv1.RetryPolicyResponse{
		Response: &providerserverv1.RetryPolicyResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toCustomValidateResourceResponse(
	output *provider.ResourceValidateOutput,
) *providerserverv1.CustomValidateResourceResponse {
	if output == nil {
		return &providerserverv1.CustomValidateResourceResponse{
			Response: &providerserverv1.CustomValidateResourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.CustomValidateResourceResponse{
		Response: &providerserverv1.CustomValidateResourceResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.CustomValidateResourceCompleteResponse{
				Diagnostics: sharedtypesv1.ToPBDiagnostics(output.Diagnostics),
			},
		},
	}
}

func toCustomValidateErrorResponse(
	err error,
) *providerserverv1.CustomValidateResourceResponse {
	return &providerserverv1.CustomValidateResourceResponse{
		Response: &providerserverv1.CustomValidateResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBResourceSpecDefinitionResponse(
	output *provider.ResourceGetSpecDefinitionOutput,
) (*providerserverv1.ResourceSpecDefinitionResponse, error) {
	if output == nil {
		return &providerserverv1.ResourceSpecDefinitionResponse{
			Response: &providerserverv1.ResourceSpecDefinitionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	schema, err := convertv1.ToPBResourceDefinitionsSchema(output.SpecDefinition.Schema)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.ResourceSpecDefinitionResponse{
		Response: &providerserverv1.ResourceSpecDefinitionResponse_SpecDefinition{
			SpecDefinition: &sharedtypesv1.ResourceSpecDefinition{
				Schema:  schema,
				IdField: output.SpecDefinition.IDField,
			},
		},
	}, nil
}

func toResourceSpecDefinitionErrorResponse(
	err error,
) *providerserverv1.ResourceSpecDefinitionResponse {
	return &providerserverv1.ResourceSpecDefinitionResponse{
		Response: &providerserverv1.ResourceSpecDefinitionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toCanResourceLinkToErrorResponse(
	err error,
) *providerserverv1.CanResourceLinkToResponse {
	return &providerserverv1.CanResourceLinkToResponse{
		Response: &providerserverv1.CanResourceLinkToResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toCanResourceLinkToResponse(
	output *provider.ResourceCanLinkToOutput,
) *providerserverv1.CanResourceLinkToResponse {
	if output == nil {
		return &providerserverv1.CanResourceLinkToResponse{
			Response: &providerserverv1.CanResourceLinkToResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.CanResourceLinkToResponse{
		Response: &providerserverv1.CanResourceLinkToResponse_ResourceTypes{
			ResourceTypes: &sharedtypesv1.CanLinkTo{
				ResourceTypes: convertv1.ToPBResourceTypes(output.CanLinkTo),
			},
		},
	}
}

func toResourceStabilisedDepsResponse(
	output *provider.ResourceStabilisedDependenciesOutput,
) *providerserverv1.ResourceStabilisedDepsResponse {
	if output == nil {
		return &providerserverv1.ResourceStabilisedDepsResponse{
			Response: &providerserverv1.ResourceStabilisedDepsResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.ResourceStabilisedDepsResponse{
		Response: &providerserverv1.ResourceStabilisedDepsResponse_StabilisedDependencies{
			StabilisedDependencies: &providerserverv1.StabilisedDependencies{
				ResourceTypes: convertv1.ToPBResourceTypes(output.StabilisedDependencies),
			},
		},
	}
}

func toResourceStabilisedDepsErrorResponse(
	err error,
) *providerserverv1.ResourceStabilisedDepsResponse {
	return &providerserverv1.ResourceStabilisedDepsResponse{
		Response: &providerserverv1.ResourceStabilisedDepsResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toIsResourceCommonTerminalResponse(
	output *provider.ResourceIsCommonTerminalOutput,
) *providerserverv1.IsResourceCommonTerminalResponse {
	if output == nil {
		return &providerserverv1.IsResourceCommonTerminalResponse{
			Response: &providerserverv1.IsResourceCommonTerminalResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.IsResourceCommonTerminalResponse{
		Response: &providerserverv1.IsResourceCommonTerminalResponse_Data{
			Data: &sharedtypesv1.ResourceCommonTerminalInfo{
				IsCommonTerminal: output.IsCommonTerminal,
			},
		},
	}
}

func toIsResourceCommonTerminalErrorResponse(
	err error,
) *providerserverv1.IsResourceCommonTerminalResponse {
	return &providerserverv1.IsResourceCommonTerminalResponse{
		Response: &providerserverv1.IsResourceCommonTerminalResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toResourceTypeDescriptionResponse(
	output *provider.ResourceGetTypeDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	if output == nil {
		return &sharedtypesv1.TypeDescriptionResponse{
			Response: &sharedtypesv1.TypeDescriptionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				PlainTextDescription: output.PlainTextDescription,
				MarkdownDescription:  output.MarkdownDescription,
				PlainTextSummary:     output.PlainTextSummary,
				MarkdownSummary:      output.MarkdownSummary,
			},
		},
	}
}

func toResourceExamplesResponse(
	output *provider.ResourceGetExamplesOutput,
) *sharedtypesv1.ExamplesResponse {
	if output == nil {
		return &sharedtypesv1.ExamplesResponse{
			Response: &sharedtypesv1.ExamplesResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.ExamplesResponse{
		Response: &sharedtypesv1.ExamplesResponse_Examples{
			Examples: &sharedtypesv1.Examples{
				Examples:          output.PlainTextExamples,
				FormattedExamples: output.MarkdownExamples,
			},
		},
	}
}

func toResourceExternalStateResponse(
	output *provider.ResourceGetExternalStateOutput,
) (*providerserverv1.GetResourceExternalStateResponse, error) {
	if output == nil {
		return &providerserverv1.GetResourceExternalStateResponse{
			Response: &providerserverv1.GetResourceExternalStateResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	externalState, err := serialisation.ToMappingNodePB(
		output.ResourceSpecState,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.GetResourceExternalStateResponse{
		Response: &providerserverv1.GetResourceExternalStateResponse_ResourceSpecState{
			ResourceSpecState: externalState,
		},
	}, nil
}

func toResourceExternalStateErrorResponse(
	err error,
) *providerserverv1.GetResourceExternalStateResponse {
	return &providerserverv1.GetResourceExternalStateResponse{
		Response: &providerserverv1.GetResourceExternalStateResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toStageLinkChangesErrorResponse(
	err error,
) *providerserverv1.StageLinkChangesResponse {
	return &providerserverv1.StageLinkChangesResponse{
		Response: &providerserverv1.StageLinkChangesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBStageLinkChangesResponse(
	output *provider.LinkStageChangesOutput,
) (*providerserverv1.StageLinkChangesResponse, error) {
	if output == nil {
		return &providerserverv1.StageLinkChangesResponse{
			Response: &providerserverv1.StageLinkChangesResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	changes, err := toPBLinkChanges(output.Changes)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.StageLinkChangesResponse{
		Response: &providerserverv1.StageLinkChangesResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.StageLinkChangesCompleteResponse{
				Changes: changes,
			},
		},
	}, nil
}

func toUpdateLinkResourceErrorResponse(
	err error,
) *providerserverv1.UpdateLinkResourceResponse {
	return &providerserverv1.UpdateLinkResourceResponse{
		Response: &providerserverv1.UpdateLinkResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBLinkChanges(
	changes *provider.LinkChanges,
) (*sharedtypesv1.LinkChanges, error) {
	if changes == nil {
		return nil, nil
	}

	return convertv1.ToPBLinkChanges(*changes)
}

func toPBUpdateLinkResourceResponse(
	output *provider.LinkUpdateResourceOutput,
) (*providerserverv1.UpdateLinkResourceResponse, error) {
	if output == nil {
		return &providerserverv1.UpdateLinkResourceResponse{
			Response: &providerserverv1.UpdateLinkResourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	linkData, err := serialisation.ToMappingNodePB(
		output.LinkData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.UpdateLinkResourceResponse{
		Response: &providerserverv1.UpdateLinkResourceResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.UpdateLinkResourceCompleteResponse{
				LinkData: linkData,
			},
		},
	}, nil
}

func toUpdateLinkIntermediaryResourcesErrorResponse(
	err error,
) *providerserverv1.UpdateLinkIntermediaryResourcesResponse {
	return &providerserverv1.UpdateLinkIntermediaryResourcesResponse{
		Response: &providerserverv1.UpdateLinkIntermediaryResourcesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBUpdateLinkIntermediaryResourcesResponse(
	output *provider.LinkUpdateIntermediaryResourcesOutput,
) (*providerserverv1.UpdateLinkIntermediaryResourcesResponse, error) {
	if output == nil {
		return &providerserverv1.UpdateLinkIntermediaryResourcesResponse{
			Response: &providerserverv1.UpdateLinkIntermediaryResourcesResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	intermediaryResourceStates, err := toPBLinkIntermediaryResourceStates(
		output.IntermediaryResourceStates,
	)
	if err != nil {
		return nil, err
	}

	linkData, err := serialisation.ToMappingNodePB(
		output.LinkData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.UpdateLinkIntermediaryResourcesResponse{
		Response: &providerserverv1.UpdateLinkIntermediaryResourcesResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.UpdateLinkIntermediaryResourcesCompleteResponse{
				IntermediaryResourceStates: intermediaryResourceStates,
				LinkData:                   linkData,
			},
		},
	}, nil
}

func toPBLinkIntermediaryResourceStates(
	intermediaryResourceStates []*state.LinkIntermediaryResourceState,
) ([]*providerserverv1.LinkIntermediaryResourceState, error) {
	pbIntermediaryResourceStates := make(
		[]*providerserverv1.LinkIntermediaryResourceState,
		0,
		len(intermediaryResourceStates),
	)
	for _, intermediaryResourceState := range intermediaryResourceStates {
		pbIntermediaryResourceState, err := toPBLinkIntermediaryResourceState(
			intermediaryResourceState,
		)
		if err != nil {
			return nil, err
		}

		pbIntermediaryResourceStates = append(
			pbIntermediaryResourceStates,
			pbIntermediaryResourceState,
		)
	}

	return pbIntermediaryResourceStates, nil
}

func toPBLinkIntermediaryResourceState(
	intermediaryResourceState *state.LinkIntermediaryResourceState,
) (*providerserverv1.LinkIntermediaryResourceState, error) {
	resourceSpecData, err := serialisation.ToMappingNodePB(
		intermediaryResourceState.ResourceSpecData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.LinkIntermediaryResourceState{
		ResourceId: intermediaryResourceState.ResourceID,
		InstanceId: intermediaryResourceState.InstanceID,
		Status: sharedtypesv1.ResourceStatus(
			intermediaryResourceState.Status,
		),
		PreciseStatus: sharedtypesv1.PreciseResourceStatus(
			intermediaryResourceState.PreciseStatus,
		),
		LastDeployedTimestamp: int64(
			intermediaryResourceState.LastDeployedTimestamp,
		),
		LastDeployAttemptTimestamp: int64(
			intermediaryResourceState.LastDeployAttemptTimestamp,
		),
		ResourceSpecData: resourceSpecData,
		FailureReasons:   intermediaryResourceState.FailureReasons,
	}, nil
}

func toGetLinkPriorityResourceErrorResponse(
	err error,
) *providerserverv1.LinkPriorityResourceResponse {
	return &providerserverv1.LinkPriorityResourceResponse{
		Response: &providerserverv1.LinkPriorityResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetLinkPriorityResourceResponse(
	output *provider.LinkGetPriorityResourceOutput,
) *providerserverv1.LinkPriorityResourceResponse {
	if output == nil {
		return &providerserverv1.LinkPriorityResourceResponse{
			Response: &providerserverv1.LinkPriorityResourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.LinkPriorityResourceResponse{
		Response: &providerserverv1.LinkPriorityResourceResponse_PriorityInfo{
			PriorityInfo: &providerserverv1.LinkPriorityResourceInfo{
				PriorityResource: providerserverv1.LinkPriorityResource(output.PriorityResource),
				PriorityResourceType: convertv1.StringToResourceType(
					output.PriorityResourceType,
				),
			},
		},
	}
}

func toPBGetLinkTypeDescriptionResponse(
	output *provider.LinkGetTypeDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	if output == nil {
		return &sharedtypesv1.TypeDescriptionResponse{
			Response: &sharedtypesv1.TypeDescriptionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				PlainTextDescription: output.PlainTextDescription,
				MarkdownDescription:  output.MarkdownDescription,
				PlainTextSummary:     output.PlainTextSummary,
				MarkdownSummary:      output.MarkdownSummary,
			},
		},
	}
}

func toGetLinkAnnotationsDefinitionsErrorResponse(
	err error,
) *providerserverv1.LinkAnnotationDefinitionsResponse {
	return &providerserverv1.LinkAnnotationDefinitionsResponse{
		Response: &providerserverv1.LinkAnnotationDefinitionsResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetLinkAnnotationDefinitionsResponse(
	output *provider.LinkGetAnnotationDefinitionsOutput,
) (*providerserverv1.LinkAnnotationDefinitionsResponse, error) {
	if output == nil {
		return &providerserverv1.LinkAnnotationDefinitionsResponse{
			Response: &providerserverv1.LinkAnnotationDefinitionsResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	annotations := make(
		map[string]*providerserverv1.LinkAnnotationDefinition,
		len(output.AnnotationDefinitions),
	)
	for key, annotation := range output.AnnotationDefinitions {
		pbAnnotation, err := toPBLinkAnnotationDefinition(annotation)
		if err != nil {
			return nil, err
		}

		annotations[key] = pbAnnotation
	}

	return &providerserverv1.LinkAnnotationDefinitionsResponse{
		Response: &providerserverv1.LinkAnnotationDefinitionsResponse_AnnotationDefinitions{
			AnnotationDefinitions: &providerserverv1.LinkAnnotationDefinitions{
				Definitions: annotations,
			},
		},
	}, nil
}

func toPBLinkAnnotationDefinition(
	definition *provider.LinkAnnotationDefinition,
) (*providerserverv1.LinkAnnotationDefinition, error) {
	defaultValue, err := serialisation.ToScalarValuePB(
		definition.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	allowedValues, err := convertv1.ToPBScalarSlice(definition.AllowedValues)
	if err != nil {
		return nil, err
	}

	examples, err := convertv1.ToPBScalarSlice(definition.Examples)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.LinkAnnotationDefinition{
		Name:          definition.Name,
		Label:         definition.Label,
		Type:          convertv1.ToPBScalarType(definition.Type),
		Description:   definition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Examples:      examples,
		Required:      definition.Required,
	}, nil
}

func toGetLinkKindErrorResponse(
	err error,
) *providerserverv1.LinkKindResponse {
	return &providerserverv1.LinkKindResponse{
		Response: &providerserverv1.LinkKindResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetLinkKindResponse(
	output *provider.LinkGetKindOutput,
) *providerserverv1.LinkKindResponse {
	if output == nil {
		return &providerserverv1.LinkKindResponse{
			Response: &providerserverv1.LinkKindResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.LinkKindResponse{
		Response: &providerserverv1.LinkKindResponse_LinkKindInfo{
			LinkKindInfo: &providerserverv1.LinkKindInfo{
				Kind: toPBLinkKind(output.Kind),
			},
		},
	}
}

func toCustomValidateDataSourceErrorResponse(
	err error,
) *providerserverv1.CustomValidateDataSourceResponse {
	return &providerserverv1.CustomValidateDataSourceResponse{
		Response: &providerserverv1.CustomValidateDataSourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBCustomValidateDataSourceResponse(
	output *provider.DataSourceValidateOutput,
) *providerserverv1.CustomValidateDataSourceResponse {
	if output == nil {
		return &providerserverv1.CustomValidateDataSourceResponse{
			Response: &providerserverv1.CustomValidateDataSourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.CustomValidateDataSourceResponse{
		Response: &providerserverv1.CustomValidateDataSourceResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.CustomValidateDataSourceCompleteResponse{
				Diagnostics: sharedtypesv1.ToPBDiagnostics(output.Diagnostics),
			},
		},
	}
}

func toGetDataSourceTypeErrorResponse(
	err error,
) *providerserverv1.DataSourceTypeResponse {
	return &providerserverv1.DataSourceTypeResponse{
		Response: &providerserverv1.DataSourceTypeResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetDataSourceTypeResponse(
	output *provider.DataSourceGetTypeOutput,
) *providerserverv1.DataSourceTypeResponse {
	if output == nil {
		return &providerserverv1.DataSourceTypeResponse{
			Response: &providerserverv1.DataSourceTypeResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.DataSourceTypeResponse{
		Response: &providerserverv1.DataSourceTypeResponse_DataSourceTypeInfo{
			DataSourceTypeInfo: &providerserverv1.DataSourceTypeInfo{
				Type: &providerserverv1.DataSourceType{
					Type: output.Type,
				},
				Label: output.Label,
			},
		},
	}
}

func toDataSourceTypeDescriptionResponse(
	output *provider.DataSourceGetTypeDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	if output == nil {
		return &sharedtypesv1.TypeDescriptionResponse{
			Response: &sharedtypesv1.TypeDescriptionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				PlainTextDescription: output.PlainTextDescription,
				MarkdownDescription:  output.MarkdownDescription,
				PlainTextSummary:     output.PlainTextSummary,
				MarkdownSummary:      output.MarkdownSummary,
			},
		},
	}
}

func toGetDataSourceSpecDefinitionErrorResponse(
	err error,
) *providerserverv1.DataSourceSpecDefinitionResponse {
	return &providerserverv1.DataSourceSpecDefinitionResponse{
		Response: &providerserverv1.DataSourceSpecDefinitionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetDataSourceSpecDefinitionResponse(
	output *provider.DataSourceGetSpecDefinitionOutput,
) (*providerserverv1.DataSourceSpecDefinitionResponse, error) {
	if output == nil || output.SpecDefinition == nil {
		return &providerserverv1.DataSourceSpecDefinitionResponse{
			Response: &providerserverv1.DataSourceSpecDefinitionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	fields := make(map[string]*providerserverv1.DataSourceSpecSchema)
	for fieldName, fieldSchema := range output.SpecDefinition.Fields {
		pbFieldSchema, err := toPBDataSourceSpecSchema(fieldSchema)
		if err != nil {
			return nil, err
		}

		fields[fieldName] = pbFieldSchema
	}

	return &providerserverv1.DataSourceSpecDefinitionResponse{
		Response: &providerserverv1.DataSourceSpecDefinitionResponse_SpecDefinition{
			SpecDefinition: &providerserverv1.DataSourceSpecDefinition{
				Fields: fields,
			},
		},
	}, nil
}

func toPBDataSourceSpecSchema(
	providerSchema *provider.DataSourceSpecSchema,
) (*providerserverv1.DataSourceSpecSchema, error) {
	if providerSchema == nil {
		return nil, nil
	}

	items, err := toPBDataSourceSpecSchema(providerSchema.Items)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.DataSourceSpecSchema{
		Type:                 toPBDataSourceSpecSchemaType(providerSchema.Type),
		Label:                providerSchema.Label,
		Description:          providerSchema.Description,
		FormattedDescription: providerSchema.FormattedDescription,
		Items:                items,
		Nullable:             providerSchema.Nullable,
	}, nil
}

func toGetDataSourceFilterFieldsErrorResponse(
	err error,
) *providerserverv1.DataSourceFilterFieldsResponse {
	return &providerserverv1.DataSourceFilterFieldsResponse{
		Response: &providerserverv1.DataSourceFilterFieldsResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetDataSourceFilterFieldsResponse(
	output *provider.DataSourceGetFilterFieldsOutput,
) *providerserverv1.DataSourceFilterFieldsResponse {
	if output == nil {
		return &providerserverv1.DataSourceFilterFieldsResponse{
			Response: &providerserverv1.DataSourceFilterFieldsResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	pbFields := map[string]*providerserverv1.DataSourceFilterFieldSchema{}
	for fieldName, fieldSchema := range output.FilterFields {
		pbFields[fieldName] = &providerserverv1.DataSourceFilterFieldSchema{
			Type:                 string(fieldSchema.Type),
			Description:          fieldSchema.Description,
			FormattedDescription: fieldSchema.FormattedDescription,
			SupportedOperators:   toPBDataSourceOperators(fieldSchema.SupportedOperators),
			ConflictsWith:        fieldSchema.ConflictsWith,
		}
	}

	return &providerserverv1.DataSourceFilterFieldsResponse{
		Response: &providerserverv1.DataSourceFilterFieldsResponse_FilterFields{
			FilterFields: &providerserverv1.DataSourceFilterFields{
				FilterFields: pbFields,
			},
		},
	}
}

func toPBDataSourceOperators(
	operators []schema.DataSourceFilterOperator,
) []string {
	pbOperators := make(
		[]string,
		len(operators),
	)
	for i, operator := range operators {
		pbOperators[i] = string(operator)
	}

	return pbOperators
}

func toPBGetDataSourceExamplesResponse(
	output *provider.DataSourceGetExamplesOutput,
) *sharedtypesv1.ExamplesResponse {
	if output == nil {
		return &sharedtypesv1.ExamplesResponse{
			Response: &sharedtypesv1.ExamplesResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.ExamplesResponse{
		Response: &sharedtypesv1.ExamplesResponse_Examples{
			Examples: &sharedtypesv1.Examples{
				Examples:          output.PlainTextExamples,
				FormattedExamples: output.MarkdownExamples,
			},
		},
	}
}

func toFetchDataSourceErrorResponse(
	err error,
) *providerserverv1.FetchDataSourceResponse {
	return &providerserverv1.FetchDataSourceResponse{
		Response: &providerserverv1.FetchDataSourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBFetchDataSourceResponse(
	output *provider.DataSourceFetchOutput,
) (*providerserverv1.FetchDataSourceResponse, error) {
	if output == nil {
		return &providerserverv1.FetchDataSourceResponse{
			Response: &providerserverv1.FetchDataSourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	data, err := convertv1.ToPBMappingNodeMap(
		output.Data,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.FetchDataSourceResponse{
		Response: &providerserverv1.FetchDataSourceResponse_CompleteResponse{
			CompleteResponse: &providerserverv1.FetchDataSourceCompleteResponse{
				Data: data,
			},
		},
	}, nil
}

func toGetCustomVarTypeErrorResponse(
	err error,
) *providerserverv1.CustomVariableTypeResponse {
	return &providerserverv1.CustomVariableTypeResponse{
		Response: &providerserverv1.CustomVariableTypeResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBGetCustomVarTypeResponse(
	output *provider.CustomVariableTypeGetTypeOutput,
) *providerserverv1.CustomVariableTypeResponse {
	if output == nil {
		return &providerserverv1.CustomVariableTypeResponse{
			Response: &providerserverv1.CustomVariableTypeResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &providerserverv1.CustomVariableTypeResponse{
		Response: &providerserverv1.CustomVariableTypeResponse_CustomVarTypeInfo{
			CustomVarTypeInfo: &providerserverv1.CustomVariableTypeInfo{
				Type: &providerserverv1.CustomVariableType{
					Type: output.Type,
				},
				Label: output.Label,
			},
		},
	}
}

func toPBCustomVarTypeDescriptionResponse(
	output *provider.CustomVariableTypeGetDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	if output == nil {
		return &sharedtypesv1.TypeDescriptionResponse{
			Response: &sharedtypesv1.TypeDescriptionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				PlainTextDescription: output.PlainTextDescription,
				MarkdownDescription:  output.MarkdownDescription,
				PlainTextSummary:     output.PlainTextSummary,
				MarkdownSummary:      output.MarkdownSummary,
			},
		},
	}
}

func toGetCustomVarTypeOptionsErrorResponse(
	err error,
) *providerserverv1.CustomVariableTypeOptionsResponse {
	return &providerserverv1.CustomVariableTypeOptionsResponse{
		Response: &providerserverv1.CustomVariableTypeOptionsResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBCustomVarTypeOptionsResponse(
	output *provider.CustomVariableTypeOptionsOutput,
) (*providerserverv1.CustomVariableTypeOptionsResponse, error) {
	if output == nil {
		return &providerserverv1.CustomVariableTypeOptionsResponse{
			Response: &providerserverv1.CustomVariableTypeOptionsResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	options, err := toPBCustomVarTypeOptions(output.Options)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.CustomVariableTypeOptionsResponse{
		Response: &providerserverv1.CustomVariableTypeOptionsResponse_Options{
			Options: options,
		},
	}, nil
}

func toPBCustomVarTypeOptions(
	options map[string]*provider.CustomVariableTypeOption,
) (*providerserverv1.CustomVariableTypeOptions, error) {
	if options == nil {
		return nil, nil
	}

	optionsPB := make(
		map[string]*providerserverv1.CustomVariableTypeOption,
		len(options),
	)
	for key, option := range options {
		pbOption, err := toPBCustomVarTypeOption(option)
		if err != nil {
			return nil, err
		}

		optionsPB[key] = pbOption
	}

	return &providerserverv1.CustomVariableTypeOptions{
		Options: optionsPB,
	}, nil
}

func toPBCustomVarTypeOption(
	option *provider.CustomVariableTypeOption,
) (*providerserverv1.CustomVariableTypeOption, error) {
	if option == nil {
		return nil, nil
	}

	valuePB, err := serialisation.ToScalarValuePB(
		option.Value,
		/* optional */ false,
	)
	if err != nil {
		return nil, err
	}

	return &providerserverv1.CustomVariableTypeOption{
		Label:                option.Label,
		Value:                valuePB,
		Description:          option.Description,
		FormattedDescription: option.MarkdownDescription,
	}, nil
}

func toPBGetCustomVarTypeExamplesResponse(
	output *provider.CustomVariableTypeGetExamplesOutput,
) *sharedtypesv1.ExamplesResponse {
	if output == nil {
		return &sharedtypesv1.ExamplesResponse{
			Response: &sharedtypesv1.ExamplesResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.ExamplesResponse{
		Response: &sharedtypesv1.ExamplesResponse_Examples{
			Examples: &sharedtypesv1.Examples{
				Examples:          output.PlainTextExamples,
				FormattedExamples: output.MarkdownExamples,
			},
		},
	}
}

func toPBDataSourceSpecSchemaType(
	fieldSchemaType provider.DataSourceSpecSchemaType,
) providerserverv1.DataSourceSpecSchemaType {
	switch fieldSchemaType {
	case provider.DataSourceSpecTypeInteger:
		return providerserverv1.DataSourceSpecSchemaType_DATA_SOURCE_SPEC_INTEGER
	case provider.DataSourceSpecTypeFloat:
		return providerserverv1.DataSourceSpecSchemaType_DATA_SOURCE_SPEC_FLOAT
	case provider.DataSourceSpecTypeBoolean:
		return providerserverv1.DataSourceSpecSchemaType_DATA_SOURCE_SPEC_BOOLEAN
	case provider.DataSourceSpecTypeArray:
		return providerserverv1.DataSourceSpecSchemaType_DATA_SOURCE_SPEC_ARRAY
	default:
		return providerserverv1.DataSourceSpecSchemaType_DATA_SOURCE_SPEC_STRING
	}
}

func toPBLinkKind(kind provider.LinkKind) providerserverv1.LinkKind {
	if kind == provider.LinkKindSoft {
		return providerserverv1.LinkKind_LINK_KIND_SOFT
	}

	return providerserverv1.LinkKind_LINK_KIND_HARD
}
