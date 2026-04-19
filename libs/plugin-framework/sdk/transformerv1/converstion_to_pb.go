package transformerv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
)

func toTransformNameErrorResponse(err error) *transformerserverv1.TransformNameResponse {
	return &transformerserverv1.TransformNameResponse{
		Response: &transformerserverv1.TransformNameResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBTransformNameResponse(transformName string) *transformerserverv1.TransformNameResponse {
	return &transformerserverv1.TransformNameResponse{
		Response: &transformerserverv1.TransformNameResponse_NameInfo{
			NameInfo: &transformerserverv1.TransformNameInfo{
				TransformName: transformName,
			},
		},
	}
}

func toBlueprintTransformErrorResponse(err error) *transformerserverv1.BlueprintTransformResponse {
	return &transformerserverv1.BlueprintTransformResponse{
		Response: &transformerserverv1.BlueprintTransformResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toListAbstractResourceTypesErrorResponse(err error) *transformerserverv1.AbstractResourceTypesResponse {
	return &transformerserverv1.AbstractResourceTypesResponse{
		Response: &transformerserverv1.AbstractResourceTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractResourceTypesResponse(
	abstractResourceTypes []string,
) *transformerserverv1.AbstractResourceTypesResponse {
	return &transformerserverv1.AbstractResourceTypesResponse{
		Response: &transformerserverv1.AbstractResourceTypesResponse_AbstractResourceTypes{
			AbstractResourceTypes: &transformerserverv1.AbstractResourceTypes{
				ResourceTypes: convertv1.ToPBResourceTypes(abstractResourceTypes),
			},
		},
	}
}

func toCustomValidateAbstractResourceErrorResponse(
	err error,
) *transformerserverv1.CustomValidateAbstractResourceResponse {
	return &transformerserverv1.CustomValidateAbstractResourceResponse{
		Response: &transformerserverv1.CustomValidateAbstractResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBCustomValidateAbstractResourceResponse(
	output *transform.AbstractResourceValidateOutput,
) (*transformerserverv1.CustomValidateAbstractResourceResponse, error) {
	if output == nil {
		return &transformerserverv1.CustomValidateAbstractResourceResponse{
			Response: &transformerserverv1.CustomValidateAbstractResourceResponse_ErrorResponse{
				ErrorResponse: errorsv1.CreateResponseFromError(
					errorsv1.ErrUnexpectedResponseType(
						errorsv1.PluginActionTransformerCustomValidateAbstractResource,
					),
				),
			},
		}, nil
	}

	diagnostics, err := sharedtypesv1.ToPBDiagnostics(output.Diagnostics)
	if err != nil {
		return nil, err
	}

	return &transformerserverv1.CustomValidateAbstractResourceResponse{
		Response: &transformerserverv1.CustomValidateAbstractResourceResponse_CompleteResponse{
			CompleteResponse: &transformerserverv1.CustomValidateAbstractResourceCompleteResponse{
				Diagnostics: diagnostics,
			},
		},
	}, nil
}

func toAbstractResourceSpecDefinitionErrorResponse(
	err error,
) *transformerserverv1.AbstractResourceSpecDefinitionResponse {
	return &transformerserverv1.AbstractResourceSpecDefinitionResponse{
		Response: &transformerserverv1.AbstractResourceSpecDefinitionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractResourceSpecDefinitionResponse(
	output *transform.AbstractResourceGetSpecDefinitionOutput,
) (*transformerserverv1.AbstractResourceSpecDefinitionResponse, error) {
	if output == nil {
		return &transformerserverv1.AbstractResourceSpecDefinitionResponse{
			Response: &transformerserverv1.AbstractResourceSpecDefinitionResponse_ErrorResponse{
				ErrorResponse: errorsv1.CreateResponseFromError(
					errorsv1.ErrUnexpectedResponseType(
						errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
					),
				),
			},
		}, nil
	}

	schema, err := convertv1.ToPBResourceDefinitionsSchema(output.SpecDefinition.Schema)
	if err != nil {
		return nil, err
	}

	return &transformerserverv1.AbstractResourceSpecDefinitionResponse{
		Response: &transformerserverv1.AbstractResourceSpecDefinitionResponse_SpecDefinition{
			SpecDefinition: &sharedtypesv1.ResourceSpecDefinition{
				Schema:  schema,
				IdField: output.SpecDefinition.IDField,
			},
		},
	}, nil
}

func toCanAbstractResourceLinkToErrorResponse(
	err error,
) *transformerserverv1.CanAbstractResourceLinkToResponse {
	return &transformerserverv1.CanAbstractResourceLinkToResponse{
		Response: &transformerserverv1.CanAbstractResourceLinkToResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBCanAbstractResourceLinkToResponse(
	output *transform.AbstractResourceCanLinkToOutput,
) *transformerserverv1.CanAbstractResourceLinkToResponse {
	if output == nil {
		return &transformerserverv1.CanAbstractResourceLinkToResponse{
			Response: &transformerserverv1.CanAbstractResourceLinkToResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &transformerserverv1.CanAbstractResourceLinkToResponse{
		Response: &transformerserverv1.CanAbstractResourceLinkToResponse_ResourceTypes{
			ResourceTypes: &sharedtypesv1.CanLinkTo{
				ResourceTypes: convertv1.ToPBResourceTypes(output.CanLinkTo),
			},
		},
	}
}

func toIsAbstractResourceCommonTerminalErrorResponse(
	err error,
) *transformerserverv1.IsAbstractResourceCommonTerminalResponse {
	return &transformerserverv1.IsAbstractResourceCommonTerminalResponse{
		Response: &transformerserverv1.IsAbstractResourceCommonTerminalResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractResourceCommonTerminalResponse(
	output *transform.AbstractResourceIsCommonTerminalOutput,
) *transformerserverv1.IsAbstractResourceCommonTerminalResponse {
	if output == nil {
		return &transformerserverv1.IsAbstractResourceCommonTerminalResponse{
			Response: &transformerserverv1.IsAbstractResourceCommonTerminalResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &transformerserverv1.IsAbstractResourceCommonTerminalResponse{
		Response: &transformerserverv1.IsAbstractResourceCommonTerminalResponse_Data{
			Data: &sharedtypesv1.ResourceCommonTerminalInfo{
				IsCommonTerminal: output.IsCommonTerminal,
			},
		},
	}
}

func toPBAbstractResourceTypeResponse(
	typeInfo *transform.AbstractResourceGetTypeOutput,
) *sharedtypesv1.ResourceTypeResponse {
	return &sharedtypesv1.ResourceTypeResponse{
		Response: &sharedtypesv1.ResourceTypeResponse_ResourceTypeInfo{
			ResourceTypeInfo: &sharedtypesv1.ResourceTypeInfo{
				Type:  convertv1.StringToResourceType(typeInfo.Type),
				Label: typeInfo.Label,
			},
		},
	}
}

func toPBAbstractResourceTypeDescriptionResponse(
	typeInfo *transform.AbstractResourceGetTypeDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				MarkdownDescription:  typeInfo.MarkdownDescription,
				PlainTextDescription: typeInfo.PlainTextDescription,
				MarkdownSummary:      typeInfo.MarkdownSummary,
				PlainTextSummary:     typeInfo.PlainTextSummary,
			},
		},
	}
}

func toPBAbstractResourceExamplesResponse(
	examples *transform.AbstractResourceGetExamplesOutput,
) *sharedtypesv1.ExamplesResponse {
	return &sharedtypesv1.ExamplesResponse{
		Response: &sharedtypesv1.ExamplesResponse_Examples{
			Examples: &sharedtypesv1.Examples{
				FormattedExamples: examples.MarkdownExamples,
				Examples:          examples.PlainTextExamples,
			},
		},
	}
}

func toPBListAbstractLinkTypesErrorResponse(
	err error,
) *transformerserverv1.AbstractLinkTypesResponse {
	return &transformerserverv1.AbstractLinkTypesResponse{
		Response: &transformerserverv1.AbstractLinkTypesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractLinkTypesResponse(
	abstractLinkTypes []string,
) *transformerserverv1.AbstractLinkTypesResponse {
	return &transformerserverv1.AbstractLinkTypesResponse{
		Response: &transformerserverv1.AbstractLinkTypesResponse_LinkTypes{
			LinkTypes: &transformerserverv1.AbstractLinkTypes{
				LinkTypes: abstractLinkTypes,
			},
		},
	}
}

func toPBAbstractLinkTypeErrorResponse(
	err error,
) *transformerserverv1.GetAbstractLinkTypeResponse {
	return &transformerserverv1.GetAbstractLinkTypeResponse{
		Response: &transformerserverv1.GetAbstractLinkTypeResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractLinkTypeResponse(
	typeInfo *transform.AbstractLinkGetTypeOutput,
) *transformerserverv1.GetAbstractLinkTypeResponse {
	return &transformerserverv1.GetAbstractLinkTypeResponse{
		Response: &transformerserverv1.GetAbstractLinkTypeResponse_LinkType{
			LinkType: &transformerserverv1.AbstractLinkType{
				LinkType: typeInfo.Type,
			},
		},
	}
}

func toPBAbstractLinkTypeDescriptionResponse(
	output *transform.AbstractLinkGetTypeDescriptionOutput,
) *sharedtypesv1.TypeDescriptionResponse {
	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_Description{
			Description: &sharedtypesv1.TypeDescription{
				MarkdownDescription:  output.MarkdownDescription,
				PlainTextDescription: output.PlainTextDescription,
				MarkdownSummary:      output.MarkdownSummary,
				PlainTextSummary:     output.PlainTextSummary,
			},
		},
	}
}

func toPBAbstractLinkAnnotationDefinitionsErrorResponse(
	err error,
) *sharedtypesv1.LinkAnnotationDefinitionsResponse {
	return &sharedtypesv1.LinkAnnotationDefinitionsResponse{
		Response: &sharedtypesv1.LinkAnnotationDefinitionsResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractLinkAnnotationDefinitionsResponse(
	output *transform.AbstractLinkGetAnnotationDefinitionsOutput,
) (*sharedtypesv1.LinkAnnotationDefinitionsResponse, error) {
	if output == nil {
		return &sharedtypesv1.LinkAnnotationDefinitionsResponse{
			Response: &sharedtypesv1.LinkAnnotationDefinitionsResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	annotations := make(
		map[string]*sharedtypesv1.LinkAnnotationDefinition,
		len(output.AnnotationDefinitions),
	)
	for key, annotation := range output.AnnotationDefinitions {
		pbAnnotation, err := toPBLinkAnnotationDefinition(annotation)
		if err != nil {
			return nil, err
		}

		annotations[key] = pbAnnotation
	}

	return &sharedtypesv1.LinkAnnotationDefinitionsResponse{
		Response: &sharedtypesv1.LinkAnnotationDefinitionsResponse_AnnotationDefinitions{
			AnnotationDefinitions: &sharedtypesv1.LinkAnnotationDefinitions{
				Definitions: annotations,
			},
		},
	}, nil
}

func toPBAbstractLinkCardinalityErrorResponse(
	err error,
) *sharedtypesv1.LinkCardinalityResponse {
	return &sharedtypesv1.LinkCardinalityResponse{
		Response: &sharedtypesv1.LinkCardinalityResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBAbstractLinkCardinalityResponse(
	output *transform.AbstractLinkGetCardinalityOutput,
) *sharedtypesv1.LinkCardinalityResponse {
	if output == nil {
		return &sharedtypesv1.LinkCardinalityResponse{
			Response: &sharedtypesv1.LinkCardinalityResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.LinkCardinalityResponse{
		Response: &sharedtypesv1.LinkCardinalityResponse_CardinalityInfo{
			CardinalityInfo: &sharedtypesv1.LinkCardinalityInfo{
				CardinalityA: &sharedtypesv1.LinkItemCardinality{
					Min: int32(output.CardinalityA.Min),
					Max: int32(output.CardinalityA.Max),
				},
				CardinalityB: &sharedtypesv1.LinkItemCardinality{
					Min: int32(output.CardinalityB.Min),
					Max: int32(output.CardinalityB.Max),
				},
			},
		},
	}
}

func toValidateLinksErrorResponse(err error) *transformerserverv1.ValidateLinksResponse {
	return &transformerserverv1.ValidateLinksResponse{
		Response: &transformerserverv1.ValidateLinksResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

func toPBValidateLinksResponse(
	output *transform.SpecTransformerValidateLinksOutput,
) (*transformerserverv1.ValidateLinksResponse, error) {
	if output == nil {
		return &transformerserverv1.ValidateLinksResponse{
			Response: &transformerserverv1.ValidateLinksResponse_ErrorResponse{
				ErrorResponse: errorsv1.CreateResponseFromError(
					errorsv1.ErrUnexpectedResponseType(
						errorsv1.PluginActionTransformerValidateLinks,
					),
				),
			},
		}, nil
	}

	diagnostics, err := sharedtypesv1.ToPBDiagnostics(output.Diagnostics)
	if err != nil {
		return nil, err
	}

	return &transformerserverv1.ValidateLinksResponse{
		Response: &transformerserverv1.ValidateLinksResponse_CompleteResponse{
			CompleteResponse: &transformerserverv1.ValidateLinksCompleteResponse{
				Diagnostics: diagnostics,
			},
		},
	}, nil
}

func toPBLinkAnnotationDefinition(
	definition *provider.LinkAnnotationDefinition,
) (*sharedtypesv1.LinkAnnotationDefinition, error) {
	if definition == nil {
		return nil, nil
	}

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

	return &sharedtypesv1.LinkAnnotationDefinition{
		Name:          definition.Name,
		Label:         definition.Label,
		Type:          convertv1.ToPBScalarType(definition.Type),
		Description:   definition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Examples:      examples,
		Required:      definition.Required,
		AppliesTo:     int32(definition.AppliesTo),
	}, nil
}
