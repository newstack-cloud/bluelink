package statestore_test

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/stretchr/testify/assert"
)

// Pins KeyBuilder behaviour at the public API. The absolute-prefix case
// is the regression guard: an earlier revision unconditionally stripped
// the leading "/" from joined keys, which silently re-rooted memfile
// state directories like "$HOME/.bluelink/engine/state" underneath the
// process working directory and made every subsequent reload miss.

func TestKeyBuilder_preserves_absolute_filesystem_prefix(t *testing.T) {
	keys := statestore.NewKeyBuilder("/var/lib/bluelink/state")

	assert.Equal(t,
		"/var/lib/bluelink/state/instances/abc.json",
		keys.Instance("abc"),
	)
	assert.Equal(t,
		"/var/lib/bluelink/state/instance_index.json",
		keys.InstanceIndex(),
	)
}

func TestKeyBuilder_handles_relative_object_store_prefix(t *testing.T) {
	keys := statestore.NewKeyBuilder("bluelink-state/")

	assert.Equal(t,
		"bluelink-state/instances/abc.json",
		keys.Instance("abc"),
	)
	assert.Equal(t,
		"bluelink-state/changesets/cs-1.json",
		keys.Changeset("cs-1"),
	)
}

func TestKeyBuilder_handles_empty_prefix(t *testing.T) {
	keys := statestore.NewKeyBuilder("")

	assert.Equal(t, "instances/abc.json", keys.Instance("abc"))
	assert.Equal(t, "instance_index.json", keys.InstanceIndex())
}

func TestKeyBuilder_handles_root_prefix(t *testing.T) {
	keys := statestore.NewKeyBuilder("/")

	assert.Equal(t, "/instances/abc.json", keys.Instance("abc"))
}

func TestKeyBuilder_per_entity_category_prefixes_end_in_a_separator(t *testing.T) {
	keys := statestore.NewKeyBuilder("bluelink-state/")

	assert.Equal(t, "bluelink-state/validations/", keys.ValidationsPrefix())
	assert.Equal(t, "bluelink-state/changesets/", keys.ChangesetsPrefix())
	assert.Equal(t,
		"bluelink-state/reconciliation_results/",
		keys.ReconciliationResultsPrefix(),
	)
	assert.Equal(t, "bluelink-state/audit_records/", keys.AuditRecordsPrefix())
	assert.Equal(t, "bluelink-state/events/", keys.EventsPrefix())
}

func TestKeyBuilder_event_object_keys_sort_by_timestamp(t *testing.T) {
	keys := statestore.NewKeyBuilder("bluelink-state/")

	assert.Equal(t,
		"bluelink-state/events/changesets_ch1/",
		keys.EventChannelPrefix("changesets_ch1"),
	)
	assert.Equal(t,
		"bluelink-state/events/changesets_ch1/01746282442-evt-1.json",
		keys.EventObject("changesets_ch1", 1746282442, "evt-1"),
	)

	// Padded to a fixed width so lexicographic order is timestamp order. An unpadded
	// key would sort "999" after "1746282442".
	early := keys.EventObject("changesets_ch1", 999, "evt-early")
	late := keys.EventObject("changesets_ch1", 1746282442, "evt-late")
	assert.Equal(t,
		"bluelink-state/events/changesets_ch1/00000000999-evt-early.json",
		early,
	)
	assert.Less(t, early, late, "an earlier event's key must sort before a later one's")
}

// The absolute-prefix case is the one that silently re-rooted memfile state before, so
// the newer keys are pinned against it too.
func TestKeyBuilder_preserves_absolute_prefix_for_per_entity_keys(t *testing.T) {
	keys := statestore.NewKeyBuilder("/var/lib/bluelink/state")

	assert.Equal(t, "/var/lib/bluelink/state/audit_records/", keys.AuditRecordsPrefix())
	assert.Equal(t,
		"/var/lib/bluelink/state/events/changesets_ch1/00000001000-evt-1.json",
		keys.EventObject("changesets_ch1", 1000, "evt-1"),
	)
}

func TestKeyBuilder_per_entity_keys_with_an_empty_prefix(t *testing.T) {
	keys := statestore.NewKeyBuilder("")

	assert.Equal(t, "audit_records/", keys.AuditRecordsPrefix())
	assert.Equal(t, "events/", keys.EventsPrefix())
	assert.Equal(t,
		"events/changesets_ch1/00000001000-evt-1.json",
		keys.EventObject("changesets_ch1", 1000, "evt-1"),
	)
}
