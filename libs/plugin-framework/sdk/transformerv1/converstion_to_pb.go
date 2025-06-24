package transformerv1

import (
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
) *transformerserverv1.CustomValidateAbstractResourceResponse {
	if output == nil {
		return &transformerserverv1.CustomValidateAbstractResourceResponse{
			Response: &transformerserverv1.CustomValidateAbstractResourceResponse_ErrorResponse{
				ErrorResponse: errorsv1.CreateResponseFromError(
					errorsv1.ErrUnexpectedResponseType(
						errorsv1.PluginActionTransformerCustomValidateAbstractResource,
					),
				),
			},
		}
	}

	return &transformerserverv1.CustomValidateAbstractResourceResponse{
		Response: &transformerserverv1.CustomValidateAbstractResourceResponse_CompleteResponse{
			CompleteResponse: &transformerserverv1.CustomValidateAbstractResourceCompleteResponse{
				Diagnostics: sharedtypesv1.ToPBDiagnostics(output.Diagnostics),
			},
		},
	}
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
