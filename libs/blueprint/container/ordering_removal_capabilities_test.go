package container

import (
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/require"
)

func removalOrderOf(t *testing.T, ordered []*ElementWithAllDeps, linkName string) int {
	t.Helper()

	index := slices.IndexFunc(ordered, func(current *ElementWithAllDeps) bool {
		return current.Element.LogicalName() == linkName
	})
	require.GreaterOrEqualf(t, index, 0, "%q was not in the removal order", linkName)

	return index
}

// Teardown runs the deploy ordering backwards.
//
// The access link has to revoke its rules while the function is still attached to the
// network. If placement detaches first, the access link sees an unattached function,
// revokes nothing, and leaves rules holding the security group, which in turn holds the
// VPC, so the teardown fails partway through with resources that cannot be deleted.
func Test_a_link_requiring_a_capability_is_removed_before_the_link_providing_it(t *testing.T) {
	elementsToRemove := &CollectedElements{
		Links: []*LinkIDInfo{
			{LinkID: "placement-id", LinkName: "netVPC::netFunction"},
			{LinkID: "access-id", LinkName: "netFunction::netQueue"},
		},
		Total: 2,
	}

	ordered, err := OrderElementsForRemoval(
		elementsToRemove,
		&state.InstanceState{InstanceID: "test-instance"},
		map[string][]string{
			"netFunction::netQueue": {"netVPC::netFunction"},
		},
	)
	require.NoError(t, err)
	require.Len(t, ordered, 2)

	require.Less(
		t,
		removalOrderOf(t, ordered, "netFunction::netQueue"),
		removalOrderOf(t, ordered, "netVPC::netFunction"),
		"the access link must be removed while the function is still attached",
	)
}

// Links that declare nothing are the common case, and must not be given an ordering they
// did not ask for.
func Test_links_without_capabilities_are_not_ordered_for_removal(t *testing.T) {
	elementsToRemove := &CollectedElements{
		Links: []*LinkIDInfo{
			{LinkID: "first-id", LinkName: "appFunction::appTable"},
			{LinkID: "second-id", LinkName: "appTable::appFunction"},
		},
		Total: 2,
	}

	ordered, err := OrderElementsForRemoval(
		elementsToRemove,
		&state.InstanceState{InstanceID: "test-instance"},
		/* capabilityEdges */ nil,
	)
	require.NoError(t, err)
	require.Len(t, ordered, 2)

	for _, element := range ordered {
		require.Emptyf(
			t,
			element.DirectDependencies,
			"%s declared nothing and should depend on no other link",
			element.Element.LogicalName(),
		)
	}
}
