package objectstore_test

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	eqParentID        = "eq-parent"
	eqParentName      = "EqParent"
	eqChildID         = "eq-child"
	eqChildName       = "EqParent-Networking"
	eqChildRef        = "networking"
	eqGrandchildID    = "eq-grandchild"
	eqGrandchildName  = "EqParent-Networking-Subnets"
	eqGrandchildRef   = "subnets"
	eqLoneID          = "eq-lone"
	eqLoneName        = "EqLone"
	eqOrdersTableID   = "eq-orders-table"
	eqOrdersTable     = "ordersTable"
	eqSaveOrderFnID   = "eq-save-order-fn"
	eqSaveOrderFn     = "saveOrderFunction"
	eqVPCID           = "eq-vpc"
	eqVPC             = "vpc"
	eqSubnetID        = "eq-subnet"
	eqSubnet          = "subnet"
	eqLinkID          = "eq-link"
	eqLinkName        = "saveOrderFunction::ordersTable"
	eqChangesetID     = "eq-changeset"
	eqValidationID    = "eq-validation"
	eqReconResultID   = "eq-recon-result"
	eqCleanupOpID     = "eq-cleanup-op"
	eqMissingID       = "eq-missing"
	eqNewResourceID   = "eq-new-resource"
	eqNewResource     = "newQueue"
	eqNewLinkID       = "eq-new-link"
	eqNewLinkName     = "saveOrderFunction::newQueue"
	eqNewChildID      = "eq-new-child"
	eqNewChildName    = "EqParent-Storage"
	eqNewChildRef     = "storage"
	eqFixedTimestamp  = 1760000000
	eqLaterTimestamp  = 1760000500
	eqCreatedUnixTime = int64(1760000000)
)

// Every ID the equivalence view reads, including the ones mutations add and remove, so a
// mutation's effect on any of them shows up as a difference between the backends.
var (
	eqInstanceIDs   = []string{eqParentID, eqChildID, eqGrandchildID, eqLoneID, eqNewChildID, eqMissingID}
	eqInstanceNames = []string{eqParentName, eqChildName, eqGrandchildName, eqLoneName, eqNewChildName}
	eqResourceIDs   = []string{eqOrdersTableID, eqSaveOrderFnID, eqVPCID, eqSubnetID, eqNewResourceID}
	eqLinkIDs       = []string{eqLinkID, eqNewLinkID}
)

func eqResource(instanceID, resourceID, name, resourceType string) *state.ResourceState {
	return &state.ResourceState{
		ResourceID:            resourceID,
		Name:                  name,
		Type:                  resourceType,
		InstanceID:            instanceID,
		Status:                core.ResourceStatusCreated,
		PreciseStatus:         core.PreciseResourceStatusCreated,
		LastDeployedTimestamp: eqFixedTimestamp,
		SpecData: core.MappingNodeFromStringMap(map[string]string{
			"name": name,
		}),
		FailureReasons: []string{},
	}
}

func eqInstance(
	instanceID string,
	instanceName string,
	resources []*state.ResourceState,
	children map[string]*state.InstanceState,
) *state.InstanceState {
	resourceIDs := map[string]string{}
	resourceMap := map[string]*state.ResourceState{}
	for _, resource := range resources {
		resourceIDs[resource.Name] = resource.ResourceID
		resourceMap[resource.ResourceID] = resource
	}

	return &state.InstanceState{
		InstanceID:            instanceID,
		InstanceName:          instanceName,
		Status:                core.InstanceStatusDeployed,
		LastDeployedTimestamp: eqFixedTimestamp,
		ResourceIDs:           resourceIDs,
		Resources:             resourceMap,
		Links:                 map[string]*state.LinkState{},
		Metadata:              map[string]*core.MappingNode{},
		Exports:               map[string]*state.ExportState{},
		ChildBlueprints:       children,
	}
}

// Three levels deep so that rebuilding children on a lazy load has to recurse, plus a
// standalone instance that no mutation touches.
func eqSeedInstances() []state.InstanceState {
	grandchild := eqInstance(
		eqGrandchildID, eqGrandchildName,
		[]*state.ResourceState{eqResource(eqGrandchildID, eqSubnetID, eqSubnet, "aws/ec2/subnet")},
		map[string]*state.InstanceState{},
	)
	child := eqInstance(
		eqChildID, eqChildName,
		[]*state.ResourceState{eqResource(eqChildID, eqVPCID, eqVPC, "aws/ec2/vpc")},
		map[string]*state.InstanceState{eqGrandchildRef: grandchild},
	)

	parent := eqInstance(
		eqParentID, eqParentName,
		[]*state.ResourceState{
			eqResource(eqParentID, eqOrdersTableID, eqOrdersTable, "aws/dynamodb/table"),
			eqResource(eqParentID, eqSaveOrderFnID, eqSaveOrderFn, "aws/lambda/function"),
		},
		map[string]*state.InstanceState{eqChildRef: child},
	)
	parent.Links[eqLinkName] = &state.LinkState{
		LinkID:                eqLinkID,
		Name:                  eqLinkName,
		InstanceID:            eqParentID,
		Status:                core.LinkStatusCreated,
		PreciseStatus:         core.PreciseLinkStatusLinkedResourcesUpdated,
		LastDeployedTimestamp: eqFixedTimestamp,
		Data: map[string]*core.MappingNode{
			"saveOrderFunction": core.MappingNodeFromStringMap(map[string]string{"tableName": "orders"}),
		},
		ResourceDataMappings: map[string]string{
			"saveOrderFunction::spec.environment.variables.TABLE_NAME": "saveOrderFunction.tableName",
		},
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{},
		FailureReasons:             []string{},
	}
	parent.Metadata["owner"] = core.MappingNodeFromString("platform")
	parent.Exports["tableName"] = &state.ExportState{
		Value: core.MappingNodeFromString("orders"),
		Type:  schema.ExportTypeString,
		Field: "resources.ordersTable.spec.name",
	}
	parent.ChildDependencies = map[string]*state.DependencyInfo{
		eqChildRef: {DependsOnResources: []string{eqOrdersTable}},
	}

	lone := eqInstance(
		eqLoneID, eqLoneName,
		[]*state.ResourceState{},
		map[string]*state.InstanceState{},
	)

	return []state.InstanceState{*parent, *lone}
}

// Seeds the way a deploy writes with instances first without their resources and links, then
// each resource and link through its own container.
func seedEquivalenceState(ctx context.Context, c equivalenceContainer) error {
	for _, instance := range eqSeedInstances() {
		if err := c.Instances().Save(ctx, withoutResourcesAndLinks(instance)); err != nil {
			return err
		}
		if err := seedResourcesAndLinks(ctx, c, &instance); err != nil {
			return err
		}
	}

	if err := seedDrift(ctx, c); err != nil {
		return err
	}

	return seedManagedEntities(ctx, c)
}

func withoutResourcesAndLinks(instance state.InstanceState) state.InstanceState {
	instance.ResourceIDs = map[string]string{}
	instance.Resources = map[string]*state.ResourceState{}
	instance.Links = map[string]*state.LinkState{}

	children := make(map[string]*state.InstanceState, len(instance.ChildBlueprints))
	for name, child := range instance.ChildBlueprints {
		shell := withoutResourcesAndLinks(*child)
		children[name] = &shell
	}
	instance.ChildBlueprints = children

	return instance
}

func seedResourcesAndLinks(ctx context.Context, c equivalenceContainer, instance *state.InstanceState) error {
	for _, child := range instance.ChildBlueprints {
		if err := seedResourcesAndLinks(ctx, c, child); err != nil {
			return err
		}
	}
	for _, resource := range instance.Resources {
		if err := c.Resources().Save(ctx, *resource); err != nil {
			return err
		}
	}
	for _, link := range instance.Links {
		if err := c.Links().Save(ctx, *link); err != nil {
			return err
		}
	}
	return nil
}

func seedDrift(ctx context.Context, c equivalenceContainer) error {
	timestamp := eqFixedTimestamp
	err := c.Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   eqOrdersTableID,
		ResourceName: eqOrdersTable,
		SpecData:     core.MappingNodeFromStringMap(map[string]string{"name": "orders-drifted"}),
		Difference:   &state.ResourceDriftChanges{},
		Timestamp:    &timestamp,
	})
	if err != nil {
		return err
	}

	return c.Links().SaveDrift(ctx, state.LinkDriftState{
		LinkID:    eqLinkID,
		LinkName:  eqLinkName,
		Timestamp: &timestamp,
	})
}

func seedManagedEntities(ctx context.Context, c equivalenceContainer) error {
	err := c.Changesets().Save(ctx, &manage.Changeset{
		ID:                eqChangesetID,
		InstanceID:        eqParentID,
		Status:            manage.ChangesetStatusChangesStaged,
		BlueprintLocation: "file:///project/app.blueprint.yml",
		Created:           eqCreatedUnixTime,
	})
	if err != nil {
		return err
	}

	err = c.Validation().Save(ctx, &manage.BlueprintValidation{
		ID:                eqValidationID,
		Status:            manage.BlueprintValidationStatusValidated,
		BlueprintLocation: "file:///project/app.blueprint.yml",
		Created:           eqCreatedUnixTime,
	})
	if err != nil {
		return err
	}

	err = c.ReconciliationResults().Save(ctx, &manage.ReconciliationResult{
		ID:          eqReconResultID,
		ChangesetID: eqChangesetID,
		InstanceID:  eqParentID,
		Result:      &container.ReconciliationCheckResult{InstanceID: eqParentID},
		Created:     eqCreatedUnixTime,
	})
	if err != nil {
		return err
	}

	return c.CleanupOperations().Save(ctx, &manage.CleanupOperation{
		ID:            eqCleanupOpID,
		CleanupType:   manage.CleanupTypeChangesets,
		Status:        manage.CleanupOperationStatusCompleted,
		StartedAt:     eqCreatedUnixTime,
		EndedAt:       eqCreatedUnixTime + 5,
		ThresholdDate: eqCreatedUnixTime - 3600,
	})
}
