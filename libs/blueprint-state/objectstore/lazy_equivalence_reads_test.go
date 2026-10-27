package objectstore_test

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type equivalenceRead struct {
	key  string
	read func(ctx context.Context, c equivalenceContainer) (any, error)
}

func equivalenceReads() []equivalenceRead {
	reads := instanceReads()
	reads = append(reads, resourceReads()...)
	reads = append(reads, linkReads()...)
	reads = append(reads, childMetadataAndExportReads()...)
	return append(reads, managedEntityReads()...)
}

func instanceReads() []equivalenceRead {
	reads := []equivalenceRead{}
	for _, id := range eqInstanceIDs {
		reads = append(reads, equivalenceRead{
			key: "instances.Get " + id,
			read: func(ctx context.Context, c equivalenceContainer) (any, error) {
				instance, err := c.Instances().Get(ctx, id)
				return scrubClaimTimestamps(instance), err
			},
		})
	}

	for _, name := range eqInstanceNames {
		reads = append(reads, equivalenceRead{
			key: "instances.LookupIDByName " + name,
			read: func(ctx context.Context, c equivalenceContainer) (any, error) {
				return c.Instances().LookupIDByName(ctx, name)
			},
		})
	}

	return append(
		reads,
		equivalenceRead{"instances.List", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Instances().List(ctx, state.ListInstancesParams{})
		}},
		equivalenceRead{"instances.List search", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Instances().List(ctx, state.ListInstancesParams{Search: "networking"})
		}},
		equivalenceRead{"instances.List page", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Instances().List(ctx, state.ListInstancesParams{Offset: 1, Limit: 2})
		}},
		equivalenceRead{"instances.GetBatch", func(ctx context.Context, c equivalenceContainer) (any, error) {
			batch, err := c.Instances().GetBatch(ctx, []string{eqParentID, eqLoneName})
			for i := range batch {
				batch[i] = scrubClaimTimestamps(batch[i])
			}
			return batch, err
		}},
	)
}

func resourceReads() []equivalenceRead {
	reads := []equivalenceRead{}
	for _, id := range eqResourceIDs {
		reads = append(
			reads,
			equivalenceRead{"resources.Get " + id, func(ctx context.Context, c equivalenceContainer) (any, error) {
				return c.Resources().Get(ctx, id)
			}},
			equivalenceRead{"resources.GetDrift " + id, func(ctx context.Context, c equivalenceContainer) (any, error) {
				return c.Resources().GetDrift(ctx, id)
			}},
		)
	}

	return append(
		reads,
		equivalenceRead{"resources.GetByName parent", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Resources().GetByName(ctx, eqParentID, eqOrdersTable)
		}},
		equivalenceRead{"resources.GetByName grandchild", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Resources().GetByName(ctx, eqGrandchildID, eqSubnet)
		}},
	)
}

func linkReads() []equivalenceRead {
	reads := []equivalenceRead{}
	for _, id := range eqLinkIDs {
		reads = append(
			reads,
			equivalenceRead{"links.Get " + id, func(ctx context.Context, c equivalenceContainer) (any, error) {
				return c.Links().Get(ctx, id)
			}},
			equivalenceRead{"links.GetDrift " + id, func(ctx context.Context, c equivalenceContainer) (any, error) {
				return c.Links().GetDrift(ctx, id)
			}},
		)
	}

	return append(
		reads,
		equivalenceRead{"links.GetByName", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Links().GetByName(ctx, eqParentID, eqLinkName)
		}},
		equivalenceRead{"links.ListWithResourceDataMappings", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Links().ListWithResourceDataMappings(ctx, eqParentID, eqSaveOrderFn)
		}},
	)
}

func childMetadataAndExportReads() []equivalenceRead {
	return []equivalenceRead{
		{"children.Get parent networking", func(ctx context.Context, c equivalenceContainer) (any, error) {
			child, err := c.Children().Get(ctx, eqParentID, eqChildRef)
			return scrubClaimTimestamps(child), err
		}},
		{"children.Get child subnets", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Children().Get(ctx, eqChildID, eqGrandchildRef)
		}},
		{"children.Get parent storage", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Children().Get(ctx, eqParentID, eqNewChildRef)
		}},
		{"metadata.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Metadata().Get(ctx, eqParentID)
		}},
		{"exports.GetAll", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Exports().GetAll(ctx, eqParentID)
		}},
		{"exports.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Exports().Get(ctx, eqParentID, "tableName")
		}},
	}
}

func managedEntityReads() []equivalenceRead {
	return []equivalenceRead{
		{"changesets.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Changesets().Get(ctx, eqChangesetID)
		}},
		{"validation.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.Validation().Get(ctx, eqValidationID)
		}},
		{"reconciliationResults.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.ReconciliationResults().Get(ctx, eqReconResultID)
		}},
		{"reconciliationResults.GetLatestByChangesetID", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.ReconciliationResults().GetLatestByChangesetID(ctx, eqChangesetID)
		}},
		{"reconciliationResults.GetAllByChangesetID", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.ReconciliationResults().GetAllByChangesetID(ctx, eqChangesetID)
		}},
		{"reconciliationResults.GetLatestByInstanceID", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.ReconciliationResults().GetLatestByInstanceID(ctx, eqParentID)
		}},
		{"reconciliationResults.GetAllByInstanceID", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.ReconciliationResults().GetAllByInstanceID(ctx, eqParentID)
		}},
		{"cleanupOperations.Get", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.CleanupOperations().Get(ctx, eqCleanupOpID)
		}},
		{"cleanupOperations.GetLatestByType", func(ctx context.Context, c equivalenceContainer) (any, error) {
			return c.CleanupOperations().GetLatestByType(ctx, manage.CleanupTypeChangesets)
		}},
	}
}

// Claims stamp the wall clock on both backends, so the two runs can land either side of a
// second boundary. Every other timestamp in these tests is fixed.
func scrubClaimTimestamps(instance state.InstanceState) state.InstanceState {
	instance.LastStatusUpdateTimestamp = 0
	for name, child := range instance.ChildBlueprints {
		scrubbed := scrubClaimTimestamps(*child)
		instance.ChildBlueprints[name] = &scrubbed
	}
	return instance
}
