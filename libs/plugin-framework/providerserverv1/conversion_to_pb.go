package providerserverv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	schemapb "github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

func toPBLinkState(linkState *state.LinkState) (*LinkState, error) {
	intermediaryResourceStates, err := toPBLinkIntermediaryResourceStates(
		linkState.IntermediaryResourceStates,
	)
	if err != nil {
		return nil, err
	}

	pbData, err := convertv1.ToPBMappingNodeMap(linkState.Data)
	if err != nil {
		return nil, err
	}

	return &LinkState{
		Id:         linkState.LinkID,
		Name:       linkState.Name,
		InstanceId: linkState.InstanceID,
		Status:     LinkStatus(linkState.Status),
		PreciseStatus: PreciseLinkStatus(
			linkState.PreciseStatus,
		),
		LastStatusUpdateTimestamp:  int64(linkState.LastStatusUpdateTimestamp),
		LastDeployedTimestamp:      int64(linkState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int64(linkState.LastDeployAttemptTimestamp),
		IntermediaryResourceStates: intermediaryResourceStates,
		Data:                       pbData,
		FailureReasons:             linkState.FailureReasons,
		Durations:                  toPBLinkCompletionDurations(linkState.Durations),
	}, nil
}

func toPBLinkCompletionDurations(
	durations *state.LinkCompletionDurations,
) *LinkCompletionDurations {
	if durations == nil {
		return nil
	}

	return &LinkCompletionDurations{
		ResourceAUpdate: toPBLinkComponentCompletionDurations(
			durations.ResourceAUpdate,
		),
		ResourceBUpdate: toPBLinkComponentCompletionDurations(
			durations.ResourceBUpdate,
		),
		IntermediaryResources: toPBLinkComponentCompletionDurations(
			durations.IntermediaryResources,
		),
		TotalDuration: pbutils.DoublePtrToPBWrapper(
			durations.TotalDuration,
		),
	}
}

func toPBLinkComponentCompletionDurations(
	componentDurations *state.LinkComponentCompletionDurations,
) *LinkComponentCompletionDurations {
	if componentDurations == nil {
		return nil
	}

	return &LinkComponentCompletionDurations{
		TotalDuration: pbutils.DoublePtrToPBWrapper(
			componentDurations.TotalDuration,
		),
		AttemptDurations: componentDurations.AttemptDurations,
	}
}

func toPBLinkIntermediaryResourceStates(
	intermediaryResourceStates []*state.LinkIntermediaryResourceState,
) ([]*LinkIntermediaryResourceState, error) {
	pbIntermediaryResourceStates := make([]*LinkIntermediaryResourceState, 0, len(intermediaryResourceStates))
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
) (*LinkIntermediaryResourceState, error) {
	pbResourceSpecData, err := serialisation.ToMappingNodePB(
		intermediaryResourceState.ResourceSpecData,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &LinkIntermediaryResourceState{
		ResourceId: intermediaryResourceState.ResourceID,
		InstanceId: intermediaryResourceState.InstanceID,
		Status:     sharedtypesv1.ResourceStatus(intermediaryResourceState.Status),
		PreciseStatus: sharedtypesv1.PreciseResourceStatus(
			intermediaryResourceState.PreciseStatus,
		),
		LastDeployedTimestamp: int64(intermediaryResourceState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int64(
			intermediaryResourceState.LastDeployAttemptTimestamp,
		),
		ResourceSpecData: pbResourceSpecData,
	}, nil
}

func toPBLinkChanges(
	changes *provider.LinkChanges,
) (*sharedtypesv1.LinkChanges, error) {
	if changes == nil {
		return nil, nil
	}

	changesPB, err := convertv1.ToPBLinkChanges(*changes)
	if err != nil {
		return nil, err
	}

	return changesPB, nil
}

func toPBLinkContext(linkCtx provider.LinkContext) (*LinkContext, error) {
	providerConfigVars, err := toPBLinkContextProviderConfigVars(
		linkCtx.AllProviderConfigVariables(),
	)
	if err != nil {
		return nil, err
	}

	contextVars, err := convertv1.ToPBScalarMap(linkCtx.ContextVariables())
	if err != nil {
		return nil, err
	}

	return &LinkContext{
		ProviderConfigVariables: providerConfigVars,
		ContextVariables:        contextVars,
	}, nil
}

func toPBLinkContextProviderConfigVars(
	allProviderConfigVars map[string]map[string]*core.ScalarValue,
) (map[string]*schemapb.ScalarValue, error) {
	providerConfigVars := make(map[string]*schemapb.ScalarValue)

	for providerName, configVars := range allProviderConfigVars {
		for key, value := range configVars {
			pbValue, err := serialisation.ToScalarValuePB(
				value,
				/* optional */ true,
			)
			if err != nil {
				return nil, err
			}

			namespacedKey := fmt.Sprintf("%s::%s", providerName, key)
			providerConfigVars[namespacedKey] = pbValue
		}
	}

	return providerConfigVars, nil
}

func toPBResolvedDataSource(
	resolvedDataSource *provider.ResolvedDataSource,
) (*ResolvedDataSource, error) {
	if resolvedDataSource == nil {
		return nil, nil
	}

	resolvedDataSourceMetadataPB, err := toPBResolvedDataSourceMetadata(
		resolvedDataSource.DataSourceMetadata,
	)
	if err != nil {
		return nil, err
	}

	resolvedDataSourceFilterPB, err := toPBResolvedDataSourceFilters(
		resolvedDataSource.Filter,
	)
	if err != nil {
		return nil, err
	}

	resolvedDataSourceExportsPB, err := toPBResolvedDataSourceExports(
		resolvedDataSource.Exports,
	)
	if err != nil {
		return nil, err
	}

	descriptionPB, err := serialisation.ToMappingNodePB(
		resolvedDataSource.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &ResolvedDataSource{
		Type:               toPBDataSourceType(resolvedDataSource.Type),
		DataSourceMetadata: resolvedDataSourceMetadataPB,
		Filter:             resolvedDataSourceFilterPB,
		Exports:            resolvedDataSourceExportsPB,
		Description:        descriptionPB,
	}, nil
}

func toPBResolvedDataSourceMetadata(
	dataSourceMetadata *provider.ResolvedDataSourceMetadata,
) (*ResolvedDataSourceMetadata, error) {

	displayNamePB, err := serialisation.ToMappingNodePB(
		dataSourceMetadata.DisplayName,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	annotationsPB, err := serialisation.ToMappingNodePB(
		dataSourceMetadata.Annotations,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	customPB, err := serialisation.ToMappingNodePB(
		dataSourceMetadata.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &ResolvedDataSourceMetadata{
		DisplayName: displayNamePB,
		Annotations: annotationsPB,
		Custom:      customPB,
	}, nil
}

func toPBResolvedDataSourceFilters(
	filters *provider.ResolvedDataSourceFilters,
) ([]*ResolvedDataSourceFilter, error) {
	if filters == nil {
		return nil, nil
	}

	pbFilters := make([]*ResolvedDataSourceFilter, 0, len(filters.Filters))
	for _, filter := range filters.Filters {
		pbFilter, err := toPBResolvedDataSourceFilter(filter)
		if err != nil {
			return nil, err
		}

		pbFilters = append(pbFilters, pbFilter)
	}

	return pbFilters, nil
}

func toPBResolvedDataSourceFilter(
	dataSourceFilter *provider.ResolvedDataSourceFilter,
) (*ResolvedDataSourceFilter, error) {
	if dataSourceFilter == nil {
		return nil, nil
	}

	fieldPB, err := serialisation.ToScalarValuePB(
		dataSourceFilter.Field,
		/* optional */ false,
	)
	if err != nil {
		return nil, err
	}

	searchPB, err := toPBResolvedDataSourceFilterSearch(
		dataSourceFilter.Search,
	)
	if err != nil {
		return nil, err
	}

	return &ResolvedDataSourceFilter{
		Field:    fieldPB,
		Operator: toPBDataSourceFilterOperator(dataSourceFilter.Operator),
		Search:   searchPB,
	}, nil
}

func toPBResolvedDataSourceFilterSearch(
	search *provider.ResolvedDataSourceFilterSearch,
) (*ResolvedDataSourceFilterSearch, error) {
	if search == nil {
		return nil, nil
	}

	valuesPB, err := convertv1.ToPBMappingNodeSlice(
		search.Values,
	)
	if err != nil {
		return nil, err
	}

	return &ResolvedDataSourceFilterSearch{
		Values: valuesPB,
	}, nil
}

func toPBResolvedDataSourceExports(
	exports map[string]*provider.ResolvedDataSourceFieldExport,
) (map[string]*ResolvedDataSourceFieldExport, error) {
	if exports == nil {
		return nil, nil
	}

	pbExports := make(map[string]*ResolvedDataSourceFieldExport)
	for key, export := range exports {
		pbExport, err := toPBResolvedDataSourceFieldExport(export)
		if err != nil {
			return nil, err
		}

		pbExports[key] = pbExport
	}

	return pbExports, nil
}

func toPBResolvedDataSourceFieldExport(
	export *provider.ResolvedDataSourceFieldExport,
) (*ResolvedDataSourceFieldExport, error) {
	if export == nil {
		return nil, nil
	}

	aliasForPB, err := serialisation.ToScalarValuePB(
		export.AliasFor,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	descriptionPB, err := serialisation.ToMappingNodePB(
		export.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &ResolvedDataSourceFieldExport{
		Type:        toPBDataSourceExportFieldType(export.Type),
		AliasFor:    aliasForPB,
		Description: descriptionPB,
	}, nil
}

func toPBDataSourceFilterOperator(
	operator *schema.DataSourceFilterOperatorWrapper,
) string {
	if operator == nil {
		return ""
	}

	return string(operator.Value)
}

func toPBDataSourceExportFieldType(
	fieldType *schema.DataSourceFieldTypeWrapper,
) string {
	if fieldType == nil {
		return ""
	}

	return string(fieldType.Value)
}

func toPBDataSourceType(
	schemaDataSourceType *schema.DataSourceTypeWrapper,
) *DataSourceType {
	if schemaDataSourceType == nil {
		return nil
	}

	return &DataSourceType{
		Type: schemaDataSourceType.Value,
	}
}
