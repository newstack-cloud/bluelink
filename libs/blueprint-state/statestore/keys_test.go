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
