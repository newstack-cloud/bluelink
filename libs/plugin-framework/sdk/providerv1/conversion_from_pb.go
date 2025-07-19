package providerv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
)

func fromPBCustomValidateResourceRequest(
	req *providerserverv1.CustomValidateResourceRequest,
) (*provider.ResourceValidateInput, error) {
	resource, err := serialisation.FromResourcePB(req.SchemaResource)
	if err != nil {
		return nil, err
	}

	providerCtx, err := convertv1.FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceValidateInput{
		SchemaResource:  resource,
		ProviderContext: providerCtx,
	}, nil
}

func fromPBGetResourceExternalStateRequest(
	req *providerserverv1.GetResourceExternalStateRequest,
) (*provider.ResourceGetExternalStateInput, error) {
	providerCtx, err := convertv1.FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	currentResourceSpec, err := serialisation.FromMappingNodePB(
		req.CurrentResourceSpec,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	currentResourceMetadata, err := convertv1.FromPBResourceMetadataState(req.CurrentResourceMetadata)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceGetExternalStateInput{
		InstanceID:              req.InstanceId,
		InstanceName:            req.InstanceName,
		ResourceID:              req.ResourceId,
		CurrentResourceSpec:     currentResourceSpec,
		CurrentResourceMetadata: currentResourceMetadata,
		ProviderContext:         providerCtx,
	}, nil
}

func fromPBStageLinkChangesRequest(
	req *providerserverv1.StageLinkChangesRequest,
) (*provider.LinkStageChangesInput, error) {

	resourceAChanges, err := convertv1.FromPBResourceChanges(req.ResourceAChanges)
	if err != nil {
		return nil, err
	}

	resourceBChanges, err := convertv1.FromPBResourceChanges(req.ResourceBChanges)
	if err != nil {
		return nil, err
	}

	currentLinkState, err := fromPBLinkState(req.CurrentLinkState)
	if err != nil {
		return nil, err
	}

	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkStageChangesInput{
		ResourceAChanges: resourceAChanges,
		ResourceBChanges: resourceBChanges,
		CurrentLinkState: currentLinkState,
		LinkContext:      linkContext,
	}, nil
}

func fromPBLinkState(
	pbLinkState *providerserverv1.LinkState,
) (*state.LinkState, error) {
	intermediaryResourceStates, err := fromPBLinkIntermediaryResourceStates(
		pbLinkState.IntermediaryResourceStates,
	)
	if err != nil {
		return nil, err
	}

	data, err := convertv1.FromPBMappingNodeMap(pbLinkState.Data)
	if err != nil {
		return nil, err
	}

	return &state.LinkState{
		LinkID:     pbLinkState.Id,
		Name:       pbLinkState.Name,
		InstanceID: pbLinkState.InstanceId,
		Status:     core.LinkStatus(pbLinkState.Status),
		PreciseStatus: core.PreciseLinkStatus(
			pbLinkState.PreciseStatus,
		),
		LastStatusUpdateTimestamp:  int(pbLinkState.LastStatusUpdateTimestamp),
		LastDeployedTimestamp:      int(pbLinkState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int(pbLinkState.LastDeployAttemptTimestamp),
		IntermediaryResourceStates: intermediaryResourceStates,
		Data:                       data,
		FailureReasons:             pbLinkState.FailureReasons,
		Durations:                  fromPBLinkCompletionDurations(pbLinkState.Durations),
	}, nil
}

func fromPBLinkIntermediaryResourceStates(
	pbIntermediaryResourceStates []*providerserverv1.LinkIntermediaryResourceState,
) ([]*state.LinkIntermediaryResourceState, error) {
	intermediaryResourceStates := make(
		[]*state.LinkIntermediaryResourceState,
		0,
		len(pbIntermediaryResourceStates),
	)
	for _, pbIntermediaryResourceState := range pbIntermediaryResourceStates {
		intermediaryResourceState, err := fromPBLinkIntermediaryResourceState(
			pbIntermediaryResourceState,
		)
		if err != nil {
			return nil, err
		}
		intermediaryResourceStates = append(
			intermediaryResourceStates,
			intermediaryResourceState,
		)
	}
	return intermediaryResourceStates, nil
}

func fromPBLinkIntermediaryResourceState(
	pbIntermediaryResourceState *providerserverv1.LinkIntermediaryResourceState,
) (*state.LinkIntermediaryResourceState, error) {
	resourceSpecData, err := serialisation.FromMappingNodePB(
		pbIntermediaryResourceState.ResourceSpecData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}
	return &state.LinkIntermediaryResourceState{
		ResourceID:   pbIntermediaryResourceState.ResourceId,
		ResourceType: pbIntermediaryResourceState.ResourceType,
		InstanceID:   pbIntermediaryResourceState.InstanceId,
		Status:       core.ResourceStatus(pbIntermediaryResourceState.Status),
		PreciseStatus: core.PreciseResourceStatus(
			pbIntermediaryResourceState.PreciseStatus,
		),
		LastDeployedTimestamp:      int(pbIntermediaryResourceState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int(pbIntermediaryResourceState.LastDeployAttemptTimestamp),
		ResourceSpecData:           resourceSpecData,
		FailureReasons:             pbIntermediaryResourceState.FailureReasons,
	}, nil
}

func fromPBLinkCompletionDurations(
	pbDurations *providerserverv1.LinkCompletionDurations,
) *state.LinkCompletionDurations {
	if pbDurations == nil {
		return nil
	}

	return &state.LinkCompletionDurations{
		ResourceAUpdate: fromPBLinkComponentCompletionDurations(pbDurations.ResourceAUpdate),
		ResourceBUpdate: fromPBLinkComponentCompletionDurations(pbDurations.ResourceBUpdate),
		IntermediaryResources: fromPBLinkComponentCompletionDurations(
			pbDurations.IntermediaryResources,
		),
		TotalDuration: pbutils.DoublePtrFromPBWrapper(pbDurations.TotalDuration),
	}
}

func fromPBLinkComponentCompletionDurations(
	pbComponentDurations *providerserverv1.LinkComponentCompletionDurations,
) *state.LinkComponentCompletionDurations {
	if pbComponentDurations == nil {
		return nil
	}

	return &state.LinkComponentCompletionDurations{
		TotalDuration: pbutils.DoublePtrFromPBWrapper(
			pbComponentDurations.TotalDuration,
		),
		AttemptDurations: pbComponentDurations.AttemptDurations,
	}
}

func fromPBUpdateLinkResourceRequest(
	req *providerserverv1.UpdateLinkResourceRequest,
) (*provider.LinkUpdateResourceInput, error) {
	changes, err := convertv1.FromPBLinkChanges(req.Changes)
	if err != nil {
		return nil, err
	}

	resourceInfo, err := convertv1.FromPBResourceInfo(req.ResourceInfo)
	if err != nil {
		return nil, err
	}

	otherResourceInfo, err := convertv1.FromPBResourceInfo(req.OtherResourceInfo)
	if err != nil {
		return nil, err
	}

	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateResourceInput{
		Changes:           &changes,
		ResourceInfo:      &resourceInfo,
		OtherResourceInfo: &otherResourceInfo,
		LinkUpdateType:    provider.LinkUpdateType(req.UpdateType),
		InstanceName:      req.InstanceName,
		LinkContext:       linkContext,
	}, nil
}

func fromPBLinkIntermediaryResourceRequest(
	req *providerserverv1.UpdateLinkIntermediaryResourcesRequest,
	resourceService provider.ResourceService,
) (*provider.LinkUpdateIntermediaryResourcesInput, error) {
	resourceAInfo, err := convertv1.FromPBResourceInfo(req.ResourceAInfo)
	if err != nil {
		return nil, err
	}

	resourceBInfo, err := convertv1.FromPBResourceInfo(req.ResourceBInfo)
	if err != nil {
		return nil, err
	}

	changes, err := convertv1.FromPBLinkChanges(req.Changes)
	if err != nil {
		return nil, err
	}

	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateIntermediaryResourcesInput{
		ResourceAInfo:   &resourceAInfo,
		ResourceBInfo:   &resourceBInfo,
		Changes:         &changes,
		LinkID:          req.LinkId,
		InstanceName:    req.InstanceName,
		LinkUpdateType:  provider.LinkUpdateType(req.UpdateType),
		LinkContext:     linkContext,
		ResourceService: resourceService,
	}, nil
}

func fromPBLinkRequestForPriorityResource(
	req *providerserverv1.LinkRequest,
) (*provider.LinkGetPriorityResourceInput, error) {
	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkGetPriorityResourceInput{
		LinkContext: linkContext,
	}, nil
}

func fromPBLinkRequestForTypeDescription(
	req *providerserverv1.LinkRequest,
) (*provider.LinkGetTypeDescriptionInput, error) {
	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkGetTypeDescriptionInput{
		LinkContext: linkContext,
	}, nil
}

func fromPBLinkRequestForAnnotationDefinitions(
	req *providerserverv1.LinkRequest,
) (*provider.LinkGetAnnotationDefinitionsInput, error) {
	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkGetAnnotationDefinitionsInput{
		LinkContext: linkContext,
	}, nil
}

func fromPBLinkRequestForKind(
	req *providerserverv1.LinkRequest,
) (*provider.LinkGetKindInput, error) {
	linkContext, err := fromPBLinkContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.LinkGetKindInput{
		LinkContext: linkContext,
	}, nil
}

func fromPBCustomValidateDataSourceRequest(
	req *providerserverv1.CustomValidateDataSourceRequest,
) (*provider.DataSourceValidateInput, error) {
	dataSource, err := serialisation.FromDataSourcePB(req.SchemaDataSource)
	if err != nil {
		return nil, err
	}

	providerCtx, err := convertv1.FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.DataSourceValidateInput{
		SchemaDataSource: dataSource,
		ProviderContext:  providerCtx,
	}, nil
}

func fromPBResolvedDataSource(
	pbResolvedDataSource *providerserverv1.ResolvedDataSource,
) (*provider.ResolvedDataSource, error) {
	if pbResolvedDataSource == nil {
		return nil, nil
	}

	dataSourceMetadata, err := fromPBResolvedDataSourceMetadata(
		pbResolvedDataSource.DataSourceMetadata,
	)
	if err != nil {
		return nil, err
	}

	filter, err := fromPBResolvedDataSourceFilters(
		pbResolvedDataSource.Filter,
	)
	if err != nil {
		return nil, err
	}

	exports, err := fromPBResolvedDataSourceFieldExports(
		pbResolvedDataSource.Exports,
	)
	if err != nil {
		return nil, err
	}

	description, err := serialisation.FromMappingNodePB(
		pbResolvedDataSource.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedDataSource{
		Type: &schema.DataSourceTypeWrapper{
			Value: pbResolvedDataSource.Type.Type,
		},
		DataSourceMetadata: dataSourceMetadata,
		Filter:             filter,
		Exports:            exports,
		Description:        description,
	}, nil
}

func fromPBResolvedDataSourceMetadata(
	pbResolvedDataSourceMetadata *providerserverv1.ResolvedDataSourceMetadata,
) (*provider.ResolvedDataSourceMetadata, error) {
	if pbResolvedDataSourceMetadata == nil {
		return nil, nil
	}

	displayName, err := serialisation.FromMappingNodePB(
		pbResolvedDataSourceMetadata.DisplayName,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	annotations, err := serialisation.FromMappingNodePB(
		pbResolvedDataSourceMetadata.Annotations,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	custom, err := serialisation.FromMappingNodePB(
		pbResolvedDataSourceMetadata.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedDataSourceMetadata{
		DisplayName: displayName,
		Annotations: annotations,
		Custom:      custom,
	}, nil
}

func fromPBResolvedDataSourceFilters(
	pbResolvedDataSourceFilters []*providerserverv1.ResolvedDataSourceFilter,
) (*provider.ResolvedDataSourceFilters, error) {
	if pbResolvedDataSourceFilters == nil {
		return nil, nil
	}

	filters := []*provider.ResolvedDataSourceFilter{}
	for _, pbFilter := range pbResolvedDataSourceFilters {
		filter, err := fromPBResolvedDataSourceFilter(pbFilter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}

	return &provider.ResolvedDataSourceFilters{
		Filters: filters,
	}, nil
}

func fromPBResolvedDataSourceFilter(
	pbResolvedDataSourceFilter *providerserverv1.ResolvedDataSourceFilter,
) (*provider.ResolvedDataSourceFilter, error) {
	if pbResolvedDataSourceFilter == nil {
		return nil, nil
	}

	field, err := serialisation.FromScalarValuePB(
		pbResolvedDataSourceFilter.Field,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	search, err := fromPBResolvedDataSourceFilterSearch(
		pbResolvedDataSourceFilter.Search,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedDataSourceFilter{
		Field: field,
		Operator: &schema.DataSourceFilterOperatorWrapper{
			Value: schema.DataSourceFilterOperator(pbResolvedDataSourceFilter.Operator),
		},
		Search: search,
	}, nil
}

func fromPBResolvedDataSourceFilterSearch(
	pbResolvedDataSourceFilterSearch *providerserverv1.ResolvedDataSourceFilterSearch,
) (*provider.ResolvedDataSourceFilterSearch, error) {
	if pbResolvedDataSourceFilterSearch == nil {
		return nil, nil
	}

	values, err := convertv1.FromPBMappingNodeSlice(
		pbResolvedDataSourceFilterSearch.Values,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedDataSourceFilterSearch{
		Values: values,
	}, nil
}

func fromPBResolvedDataSourceFieldExports(
	pbResolvedDataSourceFieldExports map[string]*providerserverv1.ResolvedDataSourceFieldExport,
) (map[string]*provider.ResolvedDataSourceFieldExport, error) {
	if pbResolvedDataSourceFieldExports == nil {
		return nil, nil
	}

	exports := make(
		map[string]*provider.ResolvedDataSourceFieldExport,
		len(pbResolvedDataSourceFieldExports),
	)
	for key, pbExport := range pbResolvedDataSourceFieldExports {
		export, err := fromPBResolvedDataSourceFieldExport(pbExport)
		if err != nil {
			return nil, err
		}
		exports[key] = export
	}

	return exports, nil
}

func fromPBResolvedDataSourceFieldExport(
	pbResolvedDataSourceFieldExport *providerserverv1.ResolvedDataSourceFieldExport,
) (*provider.ResolvedDataSourceFieldExport, error) {
	if pbResolvedDataSourceFieldExport == nil {
		return nil, nil
	}

	aliasFor, err := serialisation.FromScalarValuePB(
		pbResolvedDataSourceFieldExport.AliasFor,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	description, err := serialisation.FromMappingNodePB(
		pbResolvedDataSourceFieldExport.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedDataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{
			Value: schema.DataSourceFieldType(pbResolvedDataSourceFieldExport.Type),
		},
		AliasFor:    aliasFor,
		Description: description,
	}, nil
}

func fromPBLinkContext(
	pbLinkContext *providerserverv1.LinkContext,
) (provider.LinkContext, error) {
	providerConfigVars, err := convertv1.FromPBScalarMap(pbLinkContext.ProviderConfigVariables)
	if err != nil {
		return nil, err
	}

	contextVars, err := convertv1.FromPBScalarMap(pbLinkContext.ContextVariables)
	if err != nil {
		return nil, err
	}

	return createLinkContextFromVarMaps(providerConfigVars, contextVars)
}

func dataSourceTypeToString(dataSourceType *providerserverv1.DataSourceType) string {
	return dataSourceType.Type
}

func customVariableTypeToString(
	customVariableType *providerserverv1.CustomVariableType,
) string {
	return customVariableType.Type
}
