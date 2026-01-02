package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func upsertResources(
	ctx context.Context,
	tx pgx.Tx,
	resources []*state.ResourceState,
) error {
	query := upsertResourcesQuery()
	batch := &pgx.Batch{}
	for _, resource := range resources {
		args := pgx.NamedArgs{
			"id":            resource.ResourceID,
			"type":          resource.Type,
			"templateName":  toNullableText(resource.TemplateName),
			"status":        resource.Status,
			"preciseStatus": resource.PreciseStatus,
			"lastStatusUpdateTimestamp": toNullableTimestamp(
				resource.LastStatusUpdateTimestamp,
			),
			"lastDeployedTimestamp": toUnixTimestamp(resource.LastDeployedTimestamp),
			"lastDeployAttemptTimestamp": toUnixTimestamp(
				resource.LastDeployAttemptTimestamp,
			),
			"specData":           mappingNodeOrNilFallback(resource.SpecData, emptyObjectMappingNode),
			"description":        toNullableText(resource.Description),
			"metadata":           resource.Metadata,
			"systemMetadata":     resource.SystemMetadata,
			"computedFields":     sliceOrEmpty(resource.ComputedFields),
			"dependsOnResources": resource.DependsOnResources,
			"dependsOnChildren":  resource.DependsOnChildren,
			"failureReasons":     sliceOrEmpty(resource.FailureReasons),
			"drifted":            resource.Drifted,
			"lastDriftDetectedTimestamp": ptrToNullableTimestamp(
				resource.LastDriftDetectedTimestamp,
			),
			"durations": resource.Durations,
		}
		batch.Queue(
			query,
			args,
		)
	}

	return tx.SendBatch(ctx, batch).Close()
}

func upsertBlueprintResourceRelations(
	ctx context.Context,
	tx pgx.Tx,
	instanceID string,
	resources []*state.ResourceState,
) error {
	query := upsertBlueprintResourceRelationsQuery()
	batch := &pgx.Batch{}
	for _, resource := range resources {
		args := pgx.NamedArgs{
			"instanceId":   instanceID,
			"resourceName": resource.Name,
			"resourceId":   resource.ResourceID,
		}
		batch.Queue(query, args)
	}

	return tx.SendBatch(ctx, batch).Close()
}

func upsertLinks(
	ctx context.Context,
	tx pgx.Tx,
	links []*state.LinkState,
) error {
	query := upsertLinksQuery()
	batch := &pgx.Batch{}
	for _, link := range links {
		args := pgx.NamedArgs{
			"id":            link.LinkID,
			"status":        link.Status,
			"preciseStatus": link.PreciseStatus,
			"lastStatusUpdateTimestamp": toNullableTimestamp(
				link.LastStatusUpdateTimestamp,
			),
			"lastDeployedTimestamp": toUnixTimestamp(link.LastDeployedTimestamp),
			"lastDeployAttemptTimestamp": toUnixTimestamp(
				link.LastDeployAttemptTimestamp,
			),
			"intermediaryResourcesState": sliceOrEmpty(link.IntermediaryResourceStates),
			"data":                       mapOrEmpty(link.Data),
			"resourceDataMappings":       link.ResourceDataMappings,
			"failureReasons":             sliceOrEmpty(link.FailureReasons),
			"durations":                  link.Durations,
		}
		batch.Queue(
			query,
			args,
		)
	}

	return tx.SendBatch(ctx, batch).Close()
}

func upsertBlueprintLinkRelations(
	ctx context.Context,
	tx pgx.Tx,
	instanceID string,
	links []*state.LinkState,
) error {
	query := upsertBlueprintLinkRelationsQuery()
	batch := &pgx.Batch{}
	for _, link := range links {
		args := pgx.NamedArgs{
			"instanceId": instanceID,
			"linkName":   link.Name,
			"linkId":     link.LinkID,
		}
		batch.Queue(query, args)
	}

	return tx.SendBatch(ctx, batch).Close()
}
