package container

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A transformer that contributes no abstract resource types, leaving nothing to
// derive a namespace from.
type noAbstractResourcesTransformer struct {
	transform.SpecTransformer
}

func (t *noAbstractResourcesTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{}, nil
}

type failingListTransformer struct {
	transform.SpecTransformer
}

func (t *failingListTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return nil, errors.New("plugin unavailable")
}

// Configuration for a transformer is keyed by plugin namespace, the namespace
// its abstract resource types carry, not by the versioned name a blueprint
// selects it with. Reading it back under the transform name finds nothing, and
// a transformer that requires a config value is the only one that notices.
func Test_transformer_config_namespace_comes_from_the_abstract_resource_types(t *testing.T) {
	loader := &defaultLoader{}

	namespace := loader.transformerConfigNamespace(
		context.Background(),
		internal.CelerityTransformName,
		&internal.CelerityTransformer{},
	)

	require.NotEqual(
		t,
		internal.CelerityTransformName,
		namespace,
		"the transform name is versioned and is not what config is keyed by",
	)
	assert.Equal(t, "celerity", namespace)
}

// Nothing to derive a namespace from, and nothing that could read config under
// one either, so the transform name stands in.
func Test_transformer_config_namespace_falls_back_without_abstract_resources(t *testing.T) {
	loader := &defaultLoader{}

	namespace := loader.transformerConfigNamespace(
		context.Background(),
		"some-transform-2026",
		&noAbstractResourcesTransformer{},
	)

	assert.Equal(t, "some-transform-2026", namespace)
}

func Test_transformer_config_namespace_falls_back_when_the_plugin_errors(t *testing.T) {
	loader := &defaultLoader{}

	namespace := loader.transformerConfigNamespace(
		context.Background(),
		"some-transform-2026",
		&failingListTransformer{},
	)

	assert.Equal(t, "some-transform-2026", namespace)
}
