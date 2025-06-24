package internal

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// NormaliseResourceChanges normalises the order of fields in a provider.Changes struct
// so that it can be compared deterministically in tests.
func NormaliseResourceChanges(changes *provider.Changes, excludeResourceInfo bool) *provider.Changes {
	appliedResourceInfo := changes.AppliedResourceInfo
	if excludeResourceInfo {
		appliedResourceInfo = provider.ResourceInfo{}
	}

	return &provider.Changes{
		AppliedResourceInfo:       appliedResourceInfo,
		MustRecreate:              changes.MustRecreate,
		ModifiedFields:            OrderFieldChanges(changes.ModifiedFields),
		NewFields:                 OrderFieldChanges(changes.NewFields),
		RemovedFields:             OrderStringSlice(changes.RemovedFields),
		UnchangedFields:           OrderStringSlice(changes.UnchangedFields),
		FieldChangesKnownOnDeploy: OrderStringSlice(changes.FieldChangesKnownOnDeploy),
		ComputedFields:            OrderStringSlice(changes.ComputedFields),
		NewOutboundLinks:          changes.NewOutboundLinks,
		OutboundLinkChanges:       changes.OutboundLinkChanges,
		RemovedOutboundLinks:      changes.RemovedOutboundLinks,
	}
}

func OrderFieldChanges(fieldChanges []provider.FieldChange) []provider.FieldChange {
	orderedFieldChanges := make([]provider.FieldChange, len(fieldChanges))
	copy(orderedFieldChanges, fieldChanges)
	slices.SortFunc(orderedFieldChanges, func(a, b provider.FieldChange) int {
		if a.FieldPath < b.FieldPath {
			return -1
		}

		if a.FieldPath > b.FieldPath {
			return 1
		}

		return 0
	})
	return orderedFieldChanges
}
