package convertv1

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// ToPBConfigDefinitionResponse converts a core.ConfigDefinition to a
// ConfigDefinitionResponse protobuf message that can be sent over gRPC.
func ToPBConfigDefinitionResponse(
	configDefinition *core.ConfigDefinition,
) (*sharedtypesv1.ConfigDefinitionResponse, error) {
	if configDefinition == nil {
		return &sharedtypesv1.ConfigDefinitionResponse{
			Response: &sharedtypesv1.ConfigDefinitionResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	pbConfigDefinition, err := toPBConfigDefinition(configDefinition)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ConfigDefinitionResponse{
		Response: &sharedtypesv1.ConfigDefinitionResponse_ConfigDefinition{
			ConfigDefinition: pbConfigDefinition,
		},
	}, nil
}

// ToPBConfigDefinitionErrorResponse converts an error from a config definition to a
// ConfigDefinitionResponse protobuf message that can be sent over gRPC.
func ToPBConfigDefinitionErrorResponse(
	err error,
) *sharedtypesv1.ConfigDefinitionResponse {
	return &sharedtypesv1.ConfigDefinitionResponse{
		Response: &sharedtypesv1.ConfigDefinitionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBFunctionCallResponse converts the output from a function call to a
// FunctionCallResponse protobuf message that can be sent over gRPC.
func ToPBFunctionCallResponse(output *provider.FunctionCallOutput) (*sharedtypesv1.FunctionCallResponse, error) {
	responseData, err := pbutils.ConvertInterfaceToProtobuf(output.ResponseData)
	if err != nil {
		return nil, err
	}

	functionInfo, err := toPBFunctionRuntimeInfo(output.FunctionInfo)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionCallResponse{
		Response: &sharedtypesv1.FunctionCallResponse_FunctionResult{
			FunctionResult: &sharedtypesv1.FunctionCallResult{
				ResponseData: responseData,
				FunctionInfo: functionInfo,
			},
		},
	}, nil
}

// ToPBFunctionCallErrorResponse converts an error from a function call to a
// FunctionCallResponse protobuf message that can be sent over gRPC.
func ToPBFunctionCallErrorResponse(
	err error,
) *sharedtypesv1.FunctionCallResponse {
	return &sharedtypesv1.FunctionCallResponse{
		Response: &sharedtypesv1.FunctionCallResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBFunctionDefinitionResponse converts a function definition to a
// FunctionDefinitionResponse protobuf message that can be sent over gRPC.
func ToPBFunctionDefinitionResponse(
	definition *function.Definition,
) (*sharedtypesv1.FunctionDefinitionResponse, error) {
	pbDefinition, err := toPBFunctionDefinition(definition)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionDefinitionResponse{
		Response: &sharedtypesv1.FunctionDefinitionResponse_FunctionDefinition{
			FunctionDefinition: pbDefinition,
		},
	}, nil
}

// ToPBFunctionDefinitionErrorResponse converts an error from a function definition to a
// FunctionDefinitionResponse protobuf message that can be sent over gRPC.
func ToPBFunctionDefinitionErrorResponse(
	err error,
) *sharedtypesv1.FunctionDefinitionResponse {
	return &sharedtypesv1.FunctionDefinitionResponse{
		Response: &sharedtypesv1.FunctionDefinitionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBDeployResourceRequest converts a blueprint framework ResourceDeployServiceInput to a
// DeployResourceRequest protobuf message that can be sent over gRPC.
func ToPBDeployResourceRequest(
	resourceType string,
	input *provider.ResourceDeployInput,
) (*sharedtypesv1.DeployResourceRequest, error) {
	providerContext, err := ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, err
	}

	changes, err := ToPBChanges(input.Changes)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.DeployResourceRequest{
		ResourceType: StringToResourceType(resourceType),
		InstanceId:   input.InstanceID,
		ResourceId:   input.ResourceID,
		Changes:      changes,
		Context:      providerContext,
	}, nil
}

// ToPBDeployResourceResponse converts the output from a resource deployment to a
// DeployResourceResponse protobuf message that can be sent over gRPC.
func ToPBDeployResourceResponse(
	output *provider.ResourceDeployOutput,
) (*sharedtypesv1.DeployResourceResponse, error) {
	if output == nil {
		return &sharedtypesv1.DeployResourceResponse{
			Response: &sharedtypesv1.DeployResourceResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}, nil
	}

	computedFieldValues, err := ToPBMappingNodeMap(output.ComputedFieldValues)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.DeployResourceResponse{
		Response: &sharedtypesv1.DeployResourceResponse_CompleteResponse{
			CompleteResponse: &sharedtypesv1.DeployResourceCompleteResponse{
				ComputedFieldValues: computedFieldValues,
			},
		},
	}, nil
}

// ToPBDeployResourceErrorResponse converts an error from a resource deployment to a
// DeployResourceResponse protobuf message that can be sent over gRPC.
func ToPBDeployResourceErrorResponse(
	err error,
) *sharedtypesv1.DeployResourceResponse {
	return &sharedtypesv1.DeployResourceResponse{
		Response: &sharedtypesv1.DeployResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBDestroyResourceRequest converts a blueprint framework ResourceDestroyInput to a
// DestroyResourceRequest protobuf message that can be sent over gRPC.
func ToPBDestroyResourceRequest(
	resourceType string,
	input *provider.ResourceDestroyInput,
) (*sharedtypesv1.DestroyResourceRequest, error) {
	providerContext, err := ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, err
	}

	resourceState, err := ToPBResourceState(input.ResourceState)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.DestroyResourceRequest{
		ResourceType:  StringToResourceType(resourceType),
		InstanceId:    input.InstanceID,
		ResourceId:    input.ResourceID,
		ResourceState: resourceState,
		Context:       providerContext,
	}, nil
}

// ToPBResourceHasStabilisedResponse converts the output from a resource stabilisation check to a
// ResourceHasStabilisedResponse protobuf message that can be sent over gRPC.
func ToPBResourceHasStabilisedResponse(
	output *provider.ResourceHasStabilisedOutput,
) *sharedtypesv1.ResourceHasStabilisedResponse {
	if output == nil {
		return &sharedtypesv1.ResourceHasStabilisedResponse{
			Response: &sharedtypesv1.ResourceHasStabilisedResponse_ErrorResponse{
				ErrorResponse: sharedtypesv1.NoResponsePBError(),
			},
		}
	}

	return &sharedtypesv1.ResourceHasStabilisedResponse{
		Response: &sharedtypesv1.ResourceHasStabilisedResponse_ResourceStabilisationInfo{
			ResourceStabilisationInfo: &sharedtypesv1.ResourceStabilisationInfo{
				Stabilised: output.Stabilised,
			},
		},
	}
}

// ToPBResourceHasStabilisedErrorResponse converts an error from a resource stabilisation check to a
// ResourceHasStabilisedResponse protobuf message that can be sent over gRPC.
func ToPBResourceHasStabilisedErrorResponse(
	err error,
) *sharedtypesv1.ResourceHasStabilisedResponse {
	return &sharedtypesv1.ResourceHasStabilisedResponse{
		Response: &sharedtypesv1.ResourceHasStabilisedResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBDestroyResourceErrorResponse converts an error from destroying
// a resource to a DestroyResourceResponse protobuf message
// that can be sent over gRPC.
func ToPBDestroyResourceErrorResponse(
	err error,
) *sharedtypesv1.DestroyResourceResponse {
	return &sharedtypesv1.DestroyResourceResponse{
		Response: &sharedtypesv1.DestroyResourceResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBTypeDescriptionErrorResponse converts an error from trying to
// retrieve an entity type description to a TypeDescriptionResponse protobuf message
// that can be sent over gRPC.
func ToPBTypeDescriptionErrorResponse(
	err error,
) *sharedtypesv1.TypeDescriptionResponse {
	return &sharedtypesv1.TypeDescriptionResponse{
		Response: &sharedtypesv1.TypeDescriptionResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBExamplesErrorResponse converts an error from trying to
// retrieve examples to an ExamplesResponse protobuf message
// that can be sent over gRPC.
func ToPBExamplesErrorResponse(
	err error,
) *sharedtypesv1.ExamplesResponse {
	return &sharedtypesv1.ExamplesResponse{
		Response: &sharedtypesv1.ExamplesResponse_ErrorResponse{
			ErrorResponse: errorsv1.CreateResponseFromError(err),
		},
	}
}

// ToPBFunctionCallRequest converts a blueprint framework FunctionCallInput to a
// FunctionCallRequest protobuf message that can be sent over gRPC.
func ToPBFunctionCallRequest(
	ctx context.Context,
	functionName string,
	input *provider.FunctionCallInput,
	hostID string,
) (*sharedtypesv1.FunctionCallRequest, error) {
	args, err := toPBFunctionCallArguments(ctx, input.Arguments)
	if err != nil {
		return nil, err
	}

	callCtx, err := toPBFunctionCallContext(input.CallContext)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionCallRequest{
		FunctionName: functionName,
		Args:         args,
		CallContext:  callCtx,
		HostId:       hostID,
	}, nil
}

// ToPBFunctionDefinitionRequest converts a blueprint framework FunctionGetDefinitionInput to a
// FunctionDefinitionRequest protobuf message that can be sent over gRPC.
func ToPBFunctionDefinitionRequest(
	functionName string,
	input *provider.FunctionGetDefinitionInput,
	hostID string,
) (*sharedtypesv1.FunctionDefinitionRequest, error) {
	params, err := toPBBlueprintParams(input.Params)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionDefinitionRequest{
		FunctionName: functionName,
		Params:       params,
		HostId:       hostID,
	}, nil
}

func toPBFunctionCallArguments(
	ctx context.Context,
	args provider.FunctionCallArguments,
) (*sharedtypesv1.FunctionCallArgs, error) {
	argsSlice, err := args.Export(ctx)
	if err != nil {
		return nil, err
	}

	argsPBAny, err := pbutils.ConvertInterfaceToProtobuf(argsSlice)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionCallArgs{
		Args: argsPBAny,
	}, nil
}

func toPBFunctionCallContext(
	callContext provider.FunctionCallContext,
) (*sharedtypesv1.FunctionCallContext, error) {
	params, err := toPBBlueprintParams(callContext.Params())
	if err != nil {
		return nil, err
	}

	callStack, err := toPBFunctionCallStack(callContext.CallStackSnapshot())
	if err != nil {
		return nil, err
	}

	currentLocation := toPBSourceMeta(callContext.CurrentLocation())

	return &sharedtypesv1.FunctionCallContext{
		Params:          params,
		CallStack:       callStack,
		CurrentLocation: currentLocation,
	}, nil
}

func toPBBlueprintParams(
	params core.BlueprintParams,
) (*sharedtypesv1.BlueprintParams, error) {
	providerConfigVariables := params.AllProvidersConfig()
	namespacedProviderConfigVars := toNamespacedConfig(providerConfigVariables)
	pbProviderConfigVars, err := ToPBScalarMap(namespacedProviderConfigVars)
	if err != nil {
		return nil, err
	}

	transformerConfigVariables := params.AllTransformersConfig()
	namespacedTransformerConfigVars := toNamespacedConfig(transformerConfigVariables)
	pbTransformerConfigVars, err := ToPBScalarMap(namespacedTransformerConfigVars)
	if err != nil {
		return nil, err
	}

	contextVariables := params.AllContextVariables()
	pbContextVars, err := ToPBScalarMap(contextVariables)
	if err != nil {
		return nil, err
	}

	blueprintVariables := params.AllBlueprintVariables()
	pbBlueprintVars, err := ToPBScalarMap(blueprintVariables)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.BlueprintParams{
		ProviderConfigVariables:    pbProviderConfigVars,
		TransformerConfigVariables: pbTransformerConfigVars,
		ContextVariables:           pbContextVars,
		BlueprintVariables:         pbBlueprintVars,
	}, nil
}

func toPBFunctionCallStack(
	callStack []*function.Call,
) ([]*sharedtypesv1.FunctionCall, error) {
	if callStack == nil {
		return nil, nil
	}

	pbCallStack := make([]*sharedtypesv1.FunctionCall, len(callStack))
	for i, call := range callStack {
		pbCall := toPBFunctionCall(call)
		pbCallStack[i] = pbCall
	}

	return pbCallStack, nil
}

func toPBFunctionCall(
	call *function.Call,
) *sharedtypesv1.FunctionCall {
	if call == nil {
		return nil
	}

	location := toPBSourceMeta(call.Location)

	return &sharedtypesv1.FunctionCall{
		FilePath:     call.FilePath,
		FunctionName: call.FunctionName,
		Location:     location,
	}
}

func toPBSourceMeta(
	location *source.Meta,
) *sharedtypesv1.SourceMeta {
	if location == nil {
		return nil
	}

	startPosition := toPBSourcePosition(&location.Position)
	endPosition := toPBSourcePosition(location.EndPosition)

	return &sharedtypesv1.SourceMeta{
		StartPosition: startPosition,
		EndPosition:   endPosition,
	}
}

func toPBSourcePosition(
	position *source.Position,
) *sharedtypesv1.SourcePosition {
	if position == nil {
		return nil
	}

	return &sharedtypesv1.SourcePosition{
		Line:   int64(position.Line),
		Column: int64(position.Column),
	}
}

// ToPBChanges converts a blueprint framework resource change set
// to a Changes protobuf message that can be sent over gRPC.
func ToPBChanges(
	changes *provider.Changes,
) (*sharedtypesv1.Changes, error) {
	if changes == nil {
		return nil, nil
	}

	resourceInfo, err := ToPBResourceInfo(&changes.AppliedResourceInfo)
	if err != nil {
		return nil, err
	}

	modifiedFields, err := toPBFieldChanges(changes.ModifiedFields)
	if err != nil {
		return nil, err
	}

	newFields, err := toPBFieldChanges(changes.NewFields)
	if err != nil {
		return nil, err
	}

	newOutboundLinks, err := tpPBLinkChangesMap(changes.NewOutboundLinks)
	if err != nil {
		return nil, err
	}

	outboundLinkChanges, err := tpPBLinkChangesMap(changes.OutboundLinkChanges)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.Changes{
		AppliedResourceInfo:       resourceInfo,
		MustRecreate:              changes.MustRecreate,
		ModifiedFields:            modifiedFields,
		NewFields:                 newFields,
		RemovedFields:             changes.RemovedFields,
		UnchangedFields:           changes.UnchangedFields,
		ComputedFields:            changes.ComputedFields,
		FieldChangesKnownOnDeploy: changes.FieldChangesKnownOnDeploy,
		ConditionKnownOnDeploy:    changes.ConditionKnownOnDeploy,
		NewOutboundLinks:          newOutboundLinks,
		OutboundLinkChanges:       outboundLinkChanges,
		RemovedOutboundLinks:      changes.RemovedOutboundLinks,
	}, nil
}

// ToPBResourceInfo converts a blueprint framework ResourceInfo
// to a ResourceInfo protobuf message that can be sent over gRPC.
func ToPBResourceInfo(
	resourceInfo *provider.ResourceInfo,
) (*sharedtypesv1.ResourceInfo, error) {
	if resourceInfo == nil {
		return nil, nil
	}

	currentResourceState, err := ToPBResourceState(resourceInfo.CurrentResourceState)
	if err != nil {
		return nil, err
	}

	resolvedResource, err := toPBResolvedResource(resourceInfo.ResourceWithResolvedSubs)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResourceInfo{
		ResourceId:               resourceInfo.ResourceID,
		ResourceName:             resourceInfo.ResourceName,
		InstanceId:               resourceInfo.InstanceID,
		CurrentResourceState:     currentResourceState,
		ResourceWithResolvedSubs: resolvedResource,
	}, nil
}

// ToPBResourceState converts a blueprint framework ResourceState
// to a ResourceState protobuf message that can be sent over gRPC.
func ToPBResourceState(
	resourceState *state.ResourceState,
) (*sharedtypesv1.ResourceState, error) {
	if resourceState == nil {
		return nil, nil
	}

	specData, err := serialisation.ToMappingNodePB(
		resourceState.SpecData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	metadata, err := ToPBResourceMetadataState(resourceState.Metadata)
	if err != nil {
		return nil, err
	}

	lastDriftDetectedTimestamp := pbutils.IntPtrToPBWrapper(
		resourceState.LastDriftDetectedTimestamp,
	)

	durations, err := toPBResourceCompletionDurations(resourceState.Durations)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResourceState{
		Id:                         resourceState.ResourceID,
		Name:                       resourceState.Name,
		Type:                       resourceState.Type,
		TemplateName:               resourceState.TemplateName,
		InstanceId:                 resourceState.InstanceID,
		Status:                     sharedtypesv1.ResourceStatus(resourceState.Status),
		PreciseStatus:              sharedtypesv1.PreciseResourceStatus(resourceState.PreciseStatus),
		LastStatusUpdateTimestamp:  int64(resourceState.LastStatusUpdateTimestamp),
		LastDeployedTimestamp:      int64(resourceState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int64(resourceState.LastDeployAttemptTimestamp),
		SpecData:                   specData,
		Description:                resourceState.Description,
		Metadata:                   metadata,
		DependsOnResources:         resourceState.DependsOnResources,
		DependsOnChildren:          resourceState.DependsOnChildren,
		FailureReasons:             resourceState.FailureReasons,
		Drifted:                    resourceState.Drifted,
		LastDriftDetectedTimestamp: lastDriftDetectedTimestamp,
		Durations:                  durations,
	}, nil
}

// ToPBResourceMetadataState converts a blueprint framework ResourceMetadataState
// to a ResourceMetadataState protobuf message that can be sent over gRPC.
func ToPBResourceMetadataState(
	metadataState *state.ResourceMetadataState,
) (*sharedtypesv1.ResourceMetadataState, error) {
	if metadataState == nil {
		return nil, nil
	}

	annotations, err := ToPBMappingNodeMap(metadataState.Annotations)
	if err != nil {
		return nil, err
	}

	custom, err := serialisation.ToMappingNodePB(
		metadataState.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResourceMetadataState{
		DisplayName: metadataState.DisplayName,
		Annotations: annotations,
		Labels:      metadataState.Labels,
		Custom:      custom,
	}, nil
}

func toPBResourceCompletionDurations(
	durations *state.ResourceCompletionDurations,
) (*sharedtypesv1.ResourceCompletionDurations, error) {
	if durations == nil {
		return nil, nil
	}

	return &sharedtypesv1.ResourceCompletionDurations{
		ConfigCompleteDuration: pbutils.DoublePtrToPBWrapper(
			durations.ConfigCompleteDuration,
		),
		TotalDuration: pbutils.DoublePtrToPBWrapper(
			durations.TotalDuration,
		),
		AttemptDurations: durations.AttemptDurations,
	}, nil
}

func toPBResolvedResource(
	resolvedResource *provider.ResolvedResource,
) (*sharedtypesv1.ResolvedResource, error) {
	if resolvedResource == nil {
		return nil, nil
	}

	description, err := serialisation.ToMappingNodePB(
		resolvedResource.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	metadata, err := toPBResolvedResourceMetadata(resolvedResource.Metadata)
	if err != nil {
		return nil, err
	}

	condition, err := toPBResolvedResourceCondition(resolvedResource.Condition)
	if err != nil {
		return nil, err
	}

	linkSelector := serialisation.ToLinkSelectorPB(resolvedResource.LinkSelector)

	spec, err := serialisation.ToMappingNodePB(
		resolvedResource.Spec,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResolvedResource{
		Type: StringToResourceType(
			getResourceType(resolvedResource.Type),
		),
		Description:  description,
		Metadata:     metadata,
		Condition:    condition,
		LinkSelector: linkSelector,
		Spec:         spec,
	}, nil
}

func toPBResolvedResourceMetadata(
	metadata *provider.ResolvedResourceMetadata,
) (*sharedtypesv1.ResolvedResourceMetadata, error) {
	if metadata == nil {
		return nil, nil
	}

	displayName, err := serialisation.ToMappingNodePB(
		metadata.DisplayName,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	annotations, err := serialisation.ToMappingNodePB(
		metadata.Annotations,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	custom, err := serialisation.ToMappingNodePB(
		metadata.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResolvedResourceMetadata{
		DisplayName: displayName,
		Annotations: annotations,
		Labels:      getResolvedResourceLabels(metadata.Labels),
		Custom:      custom,
	}, nil
}

func toPBResolvedResourceCondition(
	condition *provider.ResolvedResourceCondition,
) (*sharedtypesv1.ResolvedResourceCondition, error) {
	if condition == nil {
		return nil, nil
	}

	and, err := toPBResolvedResourceConditions(condition.And)
	if err != nil {
		return nil, err
	}

	or, err := toPBResolvedResourceConditions(condition.Or)
	if err != nil {
		return nil, err
	}

	not, err := toPBResolvedResourceCondition(condition.Not)
	if err != nil {
		return nil, err
	}

	stringValue, err := serialisation.ToMappingNodePB(
		condition.StringValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ResolvedResourceCondition{
		And:         and,
		Or:          or,
		Not:         not,
		StringValue: stringValue,
	}, nil
}

func toPBResolvedResourceConditions(
	conditions []*provider.ResolvedResourceCondition,
) ([]*sharedtypesv1.ResolvedResourceCondition, error) {
	if conditions == nil {
		return nil, nil
	}

	pbConditions := make([]*sharedtypesv1.ResolvedResourceCondition, len(conditions))
	for i, condition := range conditions {
		pbCondition, err := toPBResolvedResourceCondition(condition)
		if err != nil {
			return nil, err
		}
		pbConditions[i] = pbCondition
	}

	return pbConditions, nil
}

func toPBFieldChangesFromPtrs(
	fieldChanges []*provider.FieldChange,
) ([]*sharedtypesv1.FieldChange, error) {
	if fieldChanges == nil {
		return nil, nil
	}

	pbFieldChanges := make([]*sharedtypesv1.FieldChange, len(fieldChanges))
	for i, fieldChange := range fieldChanges {
		changeDeref := provider.FieldChange{}
		if fieldChange != nil {
			changeDeref = *fieldChange
		}
		pbFieldChange, err := toPBFieldChange(changeDeref)
		if err != nil {
			return nil, err
		}
		pbFieldChanges[i] = pbFieldChange
	}

	return pbFieldChanges, nil
}

func toPBFieldChanges(
	fieldChanges []provider.FieldChange,
) ([]*sharedtypesv1.FieldChange, error) {
	if fieldChanges == nil {
		return nil, nil
	}

	pbFieldChanges := make([]*sharedtypesv1.FieldChange, len(fieldChanges))
	for i, fieldChange := range fieldChanges {
		pbFieldChange, err := toPBFieldChange(fieldChange)
		if err != nil {
			return nil, err
		}
		pbFieldChanges[i] = pbFieldChange
	}

	return pbFieldChanges, nil
}

func toPBFieldChange(
	fieldChange provider.FieldChange,
) (*sharedtypesv1.FieldChange, error) {
	prevValue, err := serialisation.ToMappingNodePB(
		fieldChange.PrevValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	newValue, err := serialisation.ToMappingNodePB(
		fieldChange.NewValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FieldChange{
		FieldPath:    fieldChange.FieldPath,
		PrevValue:    prevValue,
		NewValue:     newValue,
		MustRecreate: fieldChange.MustRecreate,
		Sensitive:    fieldChange.Sensitive,
	}, nil
}

func tpPBLinkChangesMap(
	linkChangesMap map[string]provider.LinkChanges,
) (map[string]*sharedtypesv1.LinkChanges, error) {
	if linkChangesMap == nil {
		return nil, nil
	}

	pbLinkChangesMap := make(map[string]*sharedtypesv1.LinkChanges, len(linkChangesMap))
	for key, linkChanges := range linkChangesMap {
		pbLinkChanges, err := ToPBLinkChanges(linkChanges)
		if err != nil {
			return nil, err
		}
		pbLinkChangesMap[key] = pbLinkChanges
	}

	return pbLinkChangesMap, nil
}

// ToPBLinkChanges converts a blueprint framework LinkChanges to a
// LinkChanges protobuf message that can be sent over gRPC.
func ToPBLinkChanges(
	linkChanges provider.LinkChanges,
) (*sharedtypesv1.LinkChanges, error) {
	modifiedFields, err := toPBFieldChangesFromPtrs(linkChanges.ModifiedFields)
	if err != nil {
		return nil, err
	}

	newFields, err := toPBFieldChangesFromPtrs(linkChanges.NewFields)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.LinkChanges{
		ModifiedFields:            modifiedFields,
		NewFields:                 newFields,
		RemovedFields:             linkChanges.RemovedFields,
		UnchangedFields:           linkChanges.UnchangedFields,
		FieldChangesKnownOnDeploy: linkChanges.FieldChangesKnownOnDeploy,
	}, nil
}

// ToPBMappingNodeMap converts a map of core MappingNodes to a map of protobuf MappingNodes.
func ToPBMappingNodeMap(
	mappingNodeMap map[string]*core.MappingNode,
) (map[string]*schemapb.MappingNode, error) {
	if mappingNodeMap == nil {
		return nil, nil
	}

	pbMappingNodeMap := make(map[string]*schemapb.MappingNode, len(mappingNodeMap))
	for key, mappingNode := range mappingNodeMap {
		pbMappingNode, err := serialisation.ToMappingNodePB(mappingNode, true /* optional */)
		if err != nil {
			return nil, err
		}

		pbMappingNodeMap[key] = pbMappingNode
	}

	return pbMappingNodeMap, nil
}

// ToPBMappingNodeSlice converts a slice of core MappingNodes to a slice of protobuf MappingNodes.
func ToPBMappingNodeSlice(
	mappingNodeSlice []*core.MappingNode,
) ([]*schemapb.MappingNode, error) {
	if mappingNodeSlice == nil {
		return nil, nil
	}

	pbMappingNodeSlice := make([]*schemapb.MappingNode, len(mappingNodeSlice))
	for i, mappingNode := range mappingNodeSlice {
		pbMappingNode, err := serialisation.ToMappingNodePB(mappingNode, true /* optional */)
		if err != nil {
			return nil, err
		}

		pbMappingNodeSlice[i] = pbMappingNode
	}

	return pbMappingNodeSlice, nil
}

// ToPBProviderContext converts a provider.Context to a ProviderContext protobuf message
// that can be sent over gRPC.
func ToPBProviderContext(providerCtx provider.Context) (*sharedtypesv1.ProviderContext, error) {
	providerConfigVars, err := ToPBScalarMap(providerCtx.ProviderConfigVariables())
	if err != nil {
		return nil, err
	}

	contextVars, err := ToPBScalarMap(providerCtx.ContextVariables())
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ProviderContext{
		ProviderConfigVariables: providerConfigVars,
		ContextVariables:        contextVars,
	}, nil
}

// ToPBScalarMap converts a map of core ScalarValues to a map of protobuf ScalarValues
// that can be sent over gRPC.
func ToPBScalarMap(m map[string]*core.ScalarValue) (map[string]*schemapb.ScalarValue, error) {
	pbMap := make(map[string]*schemapb.ScalarValue)
	for k, scalar := range m {
		pbScalar, err := serialisation.ToScalarValuePB(scalar, true /* optional */)
		if err != nil {
			return nil, err
		}
		pbMap[k] = pbScalar
	}
	return pbMap, nil
}

// ToPBScalarSlice converts a slice of core ScalarValues to a slice of protobuf ScalarValues
// that can be sent over gRPC.
func ToPBScalarSlice(slice []*core.ScalarValue) ([]*schemapb.ScalarValue, error) {
	pbSlice := make([]*schemapb.ScalarValue, len(slice))
	for i, scalar := range slice {
		pbScalar, err := serialisation.ToScalarValuePB(scalar, true /* optional */)
		if err != nil {
			return nil, err
		}
		pbSlice[i] = pbScalar
	}
	return pbSlice, nil
}

func toPBFunctionDefinition(
	definition *function.Definition,
) (*sharedtypesv1.FunctionDefinition, error) {
	pbParams, err := toPBFunctionParams(definition.Parameters)
	if err != nil {
		return nil, err
	}

	pbReturn, err := toPBFunctionReturn(definition.Return)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionDefinition{
		Name:                 definition.Name,
		Summary:              definition.Summary,
		FormattedSummary:     definition.FormattedSummary,
		Description:          definition.Description,
		FormattedDescription: definition.FormattedDescription,
		Parameters:           pbParams,
		Return:               pbReturn,
		Internal:             definition.Internal,
	}, nil
}

func toPBFunctionReturn(
	returnType function.Return,
) (*sharedtypesv1.FunctionReturn, error) {
	switch concreteReturnType := returnType.(type) {
	case *function.ScalarReturn:
		return toPBFunctionScalarReturn(concreteReturnType)
	case *function.ListReturn:
		return toPBFunctionListReturn(concreteReturnType)
	case *function.MapReturn:
		return toPBFunctionMapReturn(concreteReturnType)
	case *function.ObjectReturn:
		return toPBFunctionObjectReturn(concreteReturnType)
	case *function.FunctionReturn:
		return toPBFunctionTypeReturn(concreteReturnType)
	case *function.AnyReturn:
		return toPBFunctionAnyReturn(concreteReturnType)
	}

	return nil, fmt.Errorf("unsupported function return type %T", returnType)
}

func toPBFunctionScalarReturn(
	returnType *function.ScalarReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	valueTypeDef, err := toPBFunctionValueTypeDefinition(returnType.Type)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_ScalarReturn{
			ScalarReturn: &sharedtypesv1.FunctionScalarReturn{
				Type:                 valueTypeDef,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionListReturn(
	returnType *function.ListReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	elementTypeDef, err := toPBFunctionValueTypeDefinition(returnType.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_ListReturn{
			ListReturn: &sharedtypesv1.FunctionListReturn{
				ElementType:          elementTypeDef,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionMapReturn(
	returnType *function.MapReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	elementTypeDef, err := toPBFunctionValueTypeDefinition(returnType.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_MapReturn{
			MapReturn: &sharedtypesv1.FunctionMapReturn{
				ElementType:          elementTypeDef,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionObjectReturn(
	returnType *function.ObjectReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	objectValueType, err := toPBFunctionValueTypeDefinition(returnType.ObjectValueType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_ObjectReturn{
			ObjectReturn: &sharedtypesv1.FunctionObjectReturn{
				ObjectValueType:      objectValueType,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionTypeReturn(
	returnType *function.FunctionReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	functionTypeDef, err := toPBFunctionValueTypeDefinition(returnType.FunctionType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_FunctionTypeReturn{
			FunctionTypeReturn: &sharedtypesv1.FunctionTypeReturn{
				FunctionType:         functionTypeDef,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionAnyReturn(
	returnType *function.AnyReturn,
) (*sharedtypesv1.FunctionReturn, error) {
	unionTypes, err := toPBUnionValueTypeDefinitions(returnType.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionReturn{
		Return: &sharedtypesv1.FunctionReturn_AnyReturn{
			AnyReturn: &sharedtypesv1.FunctionAnyReturn{
				Type:                 toPBFunctionValueType(returnType.GetType()),
				UnionTypes:           unionTypes,
				Description:          returnType.Description,
				FormattedDescription: returnType.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionParams(
	params []function.Parameter,
) ([]*sharedtypesv1.FunctionParameter, error) {
	pbParams := make([]*sharedtypesv1.FunctionParameter, len(params))
	for i, param := range params {
		pbParam, err := toPBFunctionParameter(param)
		if err != nil {
			return nil, err
		}
		pbParams[i] = pbParam
	}
	return pbParams, nil
}

func toPBFunctionParameter(
	param function.Parameter,
) (*sharedtypesv1.FunctionParameter, error) {
	switch concreteParam := param.(type) {
	case *function.ScalarParameter:
		return toPBFunctionScalarParam(concreteParam)
	case *function.ListParameter:
		return toPBFunctionListParam(concreteParam)
	case *function.MapParameter:
		return toPBFunctionMapParam(concreteParam)
	case *function.ObjectParameter:
		return toPBFunctionObjectParam(concreteParam)
	case *function.FunctionParameter:
		return toPBFunctionTypeParam(concreteParam)
	case *function.VariadicParameter:
		return toPBFunctionVariadicParam(concreteParam)
	case *function.AnyParameter:
		return toPBFunctionAnyParam(concreteParam)
	}

	return nil, fmt.Errorf("unsupported function parameter type %T", param)
}

func toPBFunctionScalarParam(
	param *function.ScalarParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	valueTypeDef, err := toPBFunctionValueTypeDefinition(param.Type)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_ScalarParameter{
			ScalarParameter: &sharedtypesv1.FunctionScalarParameter{
				Name:                 param.Name,
				Label:                param.Label,
				Type:                 valueTypeDef,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBFunctionListParam(
	param *function.ListParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	valueTypeDef, err := toPBFunctionValueTypeDefinition(param.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_ListParameter{
			ListParameter: &sharedtypesv1.FunctionListParameter{
				Name:                 param.Name,
				Label:                param.Label,
				ElementType:          valueTypeDef,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBFunctionMapParam(
	param *function.MapParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	elementTypeDef, err := toPBFunctionValueTypeDefinition(param.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_MapParameter{
			MapParameter: &sharedtypesv1.FunctionMapParameter{
				Name:                 param.Name,
				Label:                param.Label,
				ElementType:          elementTypeDef,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBFunctionObjectParam(
	param *function.ObjectParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	objectValueType, err := toPBFunctionValueTypeDefinition(param.ObjectValueType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_ObjectParameter{
			ObjectParameter: &sharedtypesv1.FunctionObjectParameter{
				Name:                 param.Name,
				Label:                param.Label,
				ObjectValueType:      objectValueType,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBFunctionObjectAttributeTypes(
	attrTypes map[string]function.AttributeType,
) (map[string]*sharedtypesv1.FunctionObjectAttributeType, error) {
	pbAttrTypes := make(map[string]*sharedtypesv1.FunctionObjectAttributeType, len(attrTypes))
	for name, attrType := range attrTypes {
		bpValueTypeDef, err := toPBFunctionValueTypeDefinition(attrType.Type)
		if err != nil {
			return nil, err
		}
		pbAttrTypes[name] = &sharedtypesv1.FunctionObjectAttributeType{
			Type:           bpValueTypeDef,
			AllowNullValue: attrType.AllowNullValue,
		}
	}

	return pbAttrTypes, nil
}

func toPBFunctionTypeParam(
	param *function.FunctionParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	functionTypeDef, err := toPBFunctionValueTypeDefinition(param.FunctionType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_FunctionTypeParameter{
			FunctionTypeParameter: &sharedtypesv1.FunctionTypeParameter{
				Name:                 param.Name,
				Label:                param.Label,
				FunctionType:         functionTypeDef,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBFunctionVariadicParam(
	param *function.VariadicParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	valueTypeDef, err := toPBFunctionValueTypeDefinition(param.Type)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_VariadicParameter{
			VariadicParameter: &sharedtypesv1.FunctionVariadicParameter{
				Label:                param.Label,
				Type:                 valueTypeDef,
				SingleType:           param.SingleType,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Named:                param.Named,
			},
		},
	}, nil
}

func toPBFunctionAnyParam(
	param *function.AnyParameter,
) (*sharedtypesv1.FunctionParameter, error) {
	unionTypes, err := toPBUnionValueTypeDefinitions(param.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionParameter{
		Parameter: &sharedtypesv1.FunctionParameter_AnyParameter{
			AnyParameter: &sharedtypesv1.FunctionAnyParameter{
				Name:                 param.Name,
				Label:                param.Label,
				UnionTypes:           unionTypes,
				Description:          param.Description,
				FormattedDescription: param.FormattedDescription,
				AllowNullValue:       param.AllowNullValue,
				Optional:             param.Optional,
			},
		},
	}, nil
}

func toPBUnionValueTypeDefinitions(
	unionTypes []function.ValueTypeDefinition,
) ([]*sharedtypesv1.FunctionValueTypeDefinition, error) {
	pbUnionTypes := make([]*sharedtypesv1.FunctionValueTypeDefinition, len(unionTypes))
	for i, typeInUnion := range unionTypes {
		pbTypeInUnion, err := toPBFunctionValueTypeDefinition(typeInUnion)
		if err != nil {
			return nil, err
		}
		pbUnionTypes[i] = pbTypeInUnion
	}
	return pbUnionTypes, nil
}

func toPBFunctionValueTypeDefinition(
	valueType function.ValueTypeDefinition,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	switch concreteValueType := valueType.(type) {
	case *function.ValueTypeDefinitionScalar:
		return toPBFunctionScalarValueTypeDefinition(concreteValueType)
	case *function.ValueTypeDefinitionList:
		return toPBFunctionListValueTypeDefinition(concreteValueType)
	case *function.ValueTypeDefinitionMap:
		return toPBFunctionMapValueTypeDefinition(concreteValueType)
	case *function.ValueTypeDefinitionObject:
		return toPBFunctionObjectValueTypeDefinition(concreteValueType)
	case *function.ValueTypeDefinitionFunction:
		return toPBFunctionTypeValueTypeDefinition(concreteValueType)
	case *function.ValueTypeDefinitionAny:
		return toPBFunctionAnyValueTypeDefinition(concreteValueType)
	}

	return nil, fmt.Errorf("unsupported function value type %T", valueType)
}

func toPBFunctionScalarValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionScalar,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_ScalarValueType{
			ScalarValueType: &sharedtypesv1.FunctionScalarValueTypeDefinition{
				Type:                 toPBFunctionValueTypeFromDef(valueTypeDef),
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.Description,
				StringChoices:        valueTypeDef.StringChoices,
			},
		},
	}, nil
}

func toPBFunctionListValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionList,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	elementTypeDef, err := toPBFunctionValueTypeDefinition(valueTypeDef.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_ListValueType{
			ListValueType: &sharedtypesv1.FunctionListValueTypeDefinition{
				ElementType:          elementTypeDef,
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.Description,
			},
		},
	}, nil
}

func toPBFunctionMapValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionMap,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	elementTypeDef, err := toPBFunctionValueTypeDefinition(valueTypeDef.ElementType)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_MapValueType{
			MapValueType: &sharedtypesv1.FunctionMapValueTypeDefinition{
				ElementType:          elementTypeDef,
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.Description,
			},
		},
	}, nil
}

func toPBFunctionObjectValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionObject,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	attributeTypes, err := toPBFunctionObjectAttributeTypes(valueTypeDef.AttributeTypes)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_ObjectValueType{
			ObjectValueType: &sharedtypesv1.FunctionObjectValueTypeDefinition{
				AttributeTypes:       attributeTypes,
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.Description,
			},
		},
	}, nil
}

func toPBFunctionTypeValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionFunction,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	functionDef, err := toPBFunctionDefinition(&valueTypeDef.Definition)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_FunctionValueType{
			FunctionValueType: &sharedtypesv1.FunctionTypeValueTypeDefinition{
				FunctionType:         functionDef,
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.Description,
			},
		},
	}, nil
}

func toPBFunctionAnyValueTypeDefinition(
	valueTypeDef *function.ValueTypeDefinitionAny,
) (*sharedtypesv1.FunctionValueTypeDefinition, error) {
	unionTypes, err := toPBUnionValueTypeDefinitions(valueTypeDef.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionValueTypeDefinition{
		ValueTypeDefinition: &sharedtypesv1.FunctionValueTypeDefinition_AnyValueType{
			AnyValueType: &sharedtypesv1.FunctionAnyValueTypeDefinition{
				Type:                 toPBFunctionValueTypeFromDef(valueTypeDef),
				UnionTypes:           unionTypes,
				Label:                valueTypeDef.Label,
				Description:          valueTypeDef.Description,
				FormattedDescription: valueTypeDef.FormattedDescription,
			},
		},
	}, nil
}

func toPBFunctionValueTypeFromDef(
	valueTypeDef function.ValueTypeDefinition,
) sharedtypesv1.FunctionValueType {
	valueType := valueTypeDef.GetType()
	return toPBFunctionValueType(valueType)
}

func toPBFunctionValueType(
	valueType function.ValueType,
) sharedtypesv1.FunctionValueType {
	switch valueType {
	case function.ValueTypeString:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_STRING
	case function.ValueTypeInt32:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_INT32
	case function.ValueTypeInt64:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_INT64
	case function.ValueTypeUint32:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_UINT32
	case function.ValueTypeUint64:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_UINT64
	case function.ValueTypeFloat32:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FLOAT32
	case function.ValueTypeFloat64:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FLOAT64
	case function.ValueTypeBool:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_BOOL
	case function.ValueTypeList:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_LIST
	case function.ValueTypeMap:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_MAP
	case function.ValueTypeObject:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_OBJECT
	case function.ValueTypeFunction:
		return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FUNCTION
	}

	return sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_ANY
}

func toPBFunctionRuntimeInfo(
	info provider.FunctionRuntimeInfo,
) (*sharedtypesv1.FunctionRuntimeInfo, error) {
	partialArgs, err := pbutils.ConvertInterfaceToProtobuf(info.PartialArgs)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.FunctionRuntimeInfo{
		FunctionName: info.FunctionName,
		PartialArgs:  partialArgs,
		ArgsOffset:   int32(info.ArgsOffset),
	}, nil
}

func toPBConfigDefinition(
	definition *core.ConfigDefinition,
) (*sharedtypesv1.ConfigDefinition, error) {
	pbConfigDef := &sharedtypesv1.ConfigDefinition{
		Fields:                map[string]*sharedtypesv1.ConfigFieldDefinition{},
		AllowAdditionalFields: definition.AllowAdditionalFields,
	}

	for fieldName, fieldDef := range definition.Fields {
		pbFieldDef, err := toPBConfigFieldDefinition(fieldDef)
		if err != nil {
			return nil, err
		}
		pbConfigDef.Fields[fieldName] = pbFieldDef
	}

	return pbConfigDef, nil
}

func toPBConfigFieldDefinition(
	fieldDefinition *core.ConfigFieldDefinition,
) (*sharedtypesv1.ConfigFieldDefinition, error) {
	defaultValue, err := serialisation.ToScalarValuePB(
		fieldDefinition.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	allowedValues, err := ToPBScalarSlice(fieldDefinition.AllowedValues)
	if err != nil {
		return nil, err
	}

	examples, err := ToPBScalarSlice(fieldDefinition.Examples)
	if err != nil {
		return nil, err
	}

	return &sharedtypesv1.ConfigFieldDefinition{
		Type:          ToPBScalarType(fieldDefinition.Type),
		Label:         fieldDefinition.Label,
		Description:   fieldDefinition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Secret:        fieldDefinition.Secret,
		Examples:      examples,
		Required:      fieldDefinition.Required,
	}, nil
}

// ToPBScalarType converts a blueprint framework ScalarType
// to a ScalarType protobuf message that can be sent over gRPC.
func ToPBScalarType(
	scalarType core.ScalarType,
) sharedtypesv1.ScalarType {
	switch scalarType {
	case core.ScalarTypeInteger:
		return sharedtypesv1.ScalarType_SCALAR_TYPE_INTEGER
	case core.ScalarTypeFloat:
		return sharedtypesv1.ScalarType_SCALAR_TYPE_FLOAT
	case core.ScalarTypeBool:
		return sharedtypesv1.ScalarType_SCALAR_TYPE_BOOLEAN
	}

	return sharedtypesv1.ScalarType_SCALAR_TYPE_STRING
}

// StringToResourceType converts a string to a ResourceType
// protobuf message that can be sent over gRPC.
func StringToResourceType(resourceTypeStr string) *sharedtypesv1.ResourceType {
	return &sharedtypesv1.ResourceType{
		Type: resourceTypeStr,
	}
}

func getResourceType(resourceTypeWrapper *schema.ResourceTypeWrapper) string {
	if resourceTypeWrapper == nil {
		return ""
	}
	return resourceTypeWrapper.Value
}

func getResolvedResourceLabels(labels *schema.StringMap) map[string]string {
	if labels == nil {
		return nil
	}
	return labels.Values
}
