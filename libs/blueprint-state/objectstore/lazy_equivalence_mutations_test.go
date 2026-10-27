package objectstore_test

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func instanceMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "update status", mutate: updateParentStatus},
		{name: "claim then update status", mutate: claimThenUpdateParentStatus},
		{name: "save an existing instance", mutate: resaveParent},
		{name: "save a batch of existing instances", mutate: resaveBatch},
		{name: "remove an instance", mutate: removeLone},
		{name: "remove an instance with resources and links", mutate: removeParent},
	}
}

func updateParentStatus(ctx context.Context, c equivalenceContainer) error {
	timestamp := eqLaterTimestamp
	return c.Instances().UpdateStatus(ctx, eqParentID, state.InstanceStatusInfo{
		Status:                    core.InstanceStatusUpdated,
		LastStatusUpdateTimestamp: &timestamp,
	})
}

// The shape of a real deploy where the first write after a claim is a status update, which
// would persist the instance without its children if the claim dropped them from the cache.
func claimThenUpdateParentStatus(ctx context.Context, c equivalenceContainer) error {
	instance, err := c.Instances().Get(ctx, eqParentID)
	if err != nil {
		return err
	}
	_, err = c.Instances().ClaimForDeployment(
		ctx, eqParentID, instance.Version, core.InstanceStatusUpdating,
	)
	if err != nil {
		return err
	}
	return updateParentStatus(ctx, c)
}

func resaveParent(ctx context.Context, c equivalenceContainer) error {
	instance, err := c.Instances().Get(ctx, eqParentID)
	if err != nil {
		return err
	}
	instance.Status = core.InstanceStatusUpdated
	return c.Instances().Save(ctx, instance)
}

func resaveBatch(ctx context.Context, c equivalenceContainer) error {
	instances, err := c.Instances().GetBatch(ctx, []string{eqParentID, eqLoneID})
	if err != nil {
		return err
	}
	for i := range instances {
		instances[i].LastDeployAttemptTimestamp = eqLaterTimestamp
	}
	return c.Instances().SaveBatch(ctx, instances)
}

func removeLone(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Instances().Remove(ctx, eqLoneID)
	return err
}

func removeParent(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Instances().Remove(ctx, eqParentID)
	return err
}

func resourceMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "save a new resource", mutate: saveNewResource},
		{name: "save an existing resource", mutate: resaveOrdersTable},
		{name: "update status", mutate: updateOrdersTableStatus},
		{name: "update status in a nested child", mutate: updateSubnetStatus},
		{name: "remove a resource", mutate: removeOrdersTable},
		{name: "save drift", mutate: saveSubnetDrift},
		{name: "remove drift", mutate: removeOrdersTableDrift},
	}
}

func saveNewResource(ctx context.Context, c equivalenceContainer) error {
	return c.Resources().Save(ctx, *eqResource(eqParentID, eqNewResourceID, eqNewResource, "aws/sqs/queue"))
}

func resaveOrdersTable(ctx context.Context, c equivalenceContainer) error {
	resource, err := c.Resources().Get(ctx, eqOrdersTableID)
	if err != nil {
		return err
	}
	resource.Description = "Orders"
	return c.Resources().Save(ctx, resource)
}

func updateOrdersTableStatus(ctx context.Context, c equivalenceContainer) error {
	return updateResourceStatus(ctx, c, eqOrdersTableID)
}

func updateSubnetStatus(ctx context.Context, c equivalenceContainer) error {
	return updateResourceStatus(ctx, c, eqSubnetID)
}

func updateResourceStatus(ctx context.Context, c equivalenceContainer, resourceID string) error {
	timestamp := eqLaterTimestamp
	return c.Resources().UpdateStatus(ctx, resourceID, state.ResourceStatusInfo{
		Status:                    core.ResourceStatusUpdated,
		PreciseStatus:             core.PreciseResourceStatusUpdated,
		LastStatusUpdateTimestamp: &timestamp,
	})
}

func removeOrdersTable(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Resources().Remove(ctx, eqOrdersTableID)
	return err
}

func saveSubnetDrift(ctx context.Context, c equivalenceContainer) error {
	timestamp := eqLaterTimestamp
	return c.Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   eqSubnetID,
		ResourceName: eqSubnet,
		SpecData:     core.MappingNodeFromStringMap(map[string]string{"name": "subnet-drifted"}),
		Difference:   &state.ResourceDriftChanges{},
		Timestamp:    &timestamp,
	})
}

func removeOrdersTableDrift(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Resources().RemoveDrift(ctx, eqOrdersTableID)
	return err
}

func linkMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "save a new link", mutate: saveNewLink},
		{name: "replace a link under the same name", mutate: replaceLink},
		{name: "update status", mutate: updateLinkStatus},
		{name: "remove a link", mutate: removeLink},
		{name: "remove drift", mutate: removeLinkDrift},
	}
}

func saveNewLink(ctx context.Context, c equivalenceContainer) error {
	return c.Links().Save(ctx, state.LinkState{
		LinkID:                     eqNewLinkID,
		Name:                       eqNewLinkName,
		InstanceID:                 eqParentID,
		Status:                     core.LinkStatusCreated,
		PreciseStatus:              core.PreciseLinkStatusLinkedResourcesUpdated,
		LastDeployedTimestamp:      eqFixedTimestamp,
		Data:                       map[string]*core.MappingNode{},
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{},
		FailureReasons:             []string{},
	})
}

func replaceLink(ctx context.Context, c equivalenceContainer) error {
	return c.Links().Save(ctx, state.LinkState{
		LinkID:                     eqNewLinkID,
		Name:                       eqLinkName,
		InstanceID:                 eqParentID,
		Status:                     core.LinkStatusCreated,
		PreciseStatus:              core.PreciseLinkStatusLinkedResourcesUpdated,
		LastDeployedTimestamp:      eqFixedTimestamp,
		Data:                       map[string]*core.MappingNode{},
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{},
		FailureReasons:             []string{},
	})
}

func updateLinkStatus(ctx context.Context, c equivalenceContainer) error {
	timestamp := eqLaterTimestamp
	return c.Links().UpdateStatus(ctx, eqLinkID, state.LinkStatusInfo{
		Status:                    core.LinkStatusUpdated,
		PreciseStatus:             core.PreciseLinkStatusLinkedResourcesUpdated,
		LastStatusUpdateTimestamp: &timestamp,
	})
}

func removeLink(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Links().Remove(ctx, eqLinkID)
	return err
}

func removeLinkDrift(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Links().RemoveDrift(ctx, eqLinkID)
	return err
}

func childMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "attach a new child", mutate: attachNewChild},
		{name: "detach a child", mutate: detachChild},
		{name: "save child dependencies", mutate: saveChildDependencies},
		{name: "remove a child without detaching it", mutate: removeChild},
		{name: "remove a child, then detach it", mutate: removeThenDetachChild},
		{name: "save a parent whose tree renames an instance", mutate: saveParentRenamingChild},
	}
}

// The order a blueprint destroy takes: the child is removed in its own destroy, and the
// parent drops its reference afterwards.
func removeChild(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Instances().Remove(ctx, eqChildID)
	return err
}

// A destroy reads the parent before destroying its children.
func removeThenDetachChild(ctx context.Context, c equivalenceContainer) error {
	if _, err := c.Instances().Get(ctx, eqParentID); err != nil {
		return err
	}
	if err := removeChild(ctx, c); err != nil {
		return err
	}
	return c.Children().Detach(ctx, eqParentID, eqChildRef)
}

func saveParentRenamingChild(ctx context.Context, c equivalenceContainer) error {
	parent, err := c.Instances().Get(ctx, eqParentID)
	if err != nil {
		return err
	}
	lone := eqInstance(eqLoneID, "RenamedLone", nil, map[string]*state.InstanceState{})
	parent.ChildBlueprints["lone"] = lone
	return c.Instances().Save(ctx, parent)
}

func attachNewChild(ctx context.Context, c equivalenceContainer) error {
	child := eqInstance(eqNewChildID, eqNewChildName, nil, map[string]*state.InstanceState{})
	if err := c.Instances().Save(ctx, *child); err != nil {
		return err
	}
	return c.Children().Attach(ctx, eqParentID, eqNewChildID, eqNewChildRef)
}

func detachChild(ctx context.Context, c equivalenceContainer) error {
	return c.Children().Detach(ctx, eqParentID, eqChildRef)
}

func saveChildDependencies(ctx context.Context, c equivalenceContainer) error {
	return c.Children().SaveDependencies(ctx, eqParentID, eqChildRef, &state.DependencyInfo{
		DependsOnResources: []string{eqOrdersTable, eqSaveOrderFn},
	})
}

func metadataAndExportMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "save metadata", mutate: saveMetadata},
		{name: "remove metadata", mutate: removeMetadata},
		{name: "save an export", mutate: saveExport},
		{name: "save all exports", mutate: saveAllExports},
		{name: "remove an export", mutate: removeExport},
		{name: "remove all exports", mutate: removeAllExports},
	}
}

func saveMetadata(ctx context.Context, c equivalenceContainer) error {
	return c.Metadata().Save(ctx, eqParentID, map[string]*core.MappingNode{
		"owner": core.MappingNodeFromString("payments"),
	})
}

func removeMetadata(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Metadata().Remove(ctx, eqParentID)
	return err
}

func saveExport(ctx context.Context, c equivalenceContainer) error {
	return c.Exports().Save(ctx, eqParentID, "functionName", state.ExportState{
		Value: core.MappingNodeFromString("save-order"),
		Type:  schema.ExportTypeString,
		Field: "resources.saveOrderFunction.spec.name",
	})
}

func saveAllExports(ctx context.Context, c equivalenceContainer) error {
	return c.Exports().SaveAll(ctx, eqParentID, map[string]*state.ExportState{
		"tableName": {
			Value: core.MappingNodeFromString("orders-v2"),
			Type:  schema.ExportTypeString,
			Field: "resources.ordersTable.spec.name",
		},
	})
}

func removeExport(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Exports().Remove(ctx, eqParentID, "tableName")
	return err
}

func removeAllExports(ctx context.Context, c equivalenceContainer) error {
	_, err := c.Exports().RemoveAll(ctx, eqParentID)
	return err
}

func cleanupOperationMutations() []equivalenceMutation {
	return []equivalenceMutation{
		{name: "update an operation", mutate: updateCleanupOperation},
	}
}

func updateCleanupOperation(ctx context.Context, c equivalenceContainer) error {
	return c.CleanupOperations().Update(ctx, &manage.CleanupOperation{
		ID:            eqCleanupOpID,
		CleanupType:   manage.CleanupTypeChangesets,
		Status:        manage.CleanupOperationStatusFailed,
		StartedAt:     eqCreatedUnixTime,
		EndedAt:       eqCreatedUnixTime + 9,
		ErrorMessage:  "bucket unavailable",
		ThresholdDate: eqCreatedUnixTime - 3600,
	})
}
