package utils

import (
	"context"
	"sync"
)

// TypeSetCache holds the set of item types (e.g. resource types) that a plugin
// supports, loaded from the plugin on first use.
// Client wrappers use this to determine whether a plugin supports an item type
// before handing back a client for it, so that unsupported item types are reported
// as not found rather than failing later on the first call made to the plugin
// for that item.
type TypeSetCache struct {
	loadTypes func(ctx context.Context) ([]string, error)
	mu        sync.Mutex
	types     map[string]struct{}
}

// NewTypeSetCache creates a cache for the set of item types supported by a plugin.
// The provided function is called at most once to populate the cache.
func NewTypeSetCache(
	loadTypes func(ctx context.Context) ([]string, error),
) *TypeSetCache {
	return &TypeSetCache{
		loadTypes: loadTypes,
	}
}

// Has determines whether the given item type is in the set of item types supported
// by a plugin, loading the set from the plugin the first time it is called.
func (c *TypeSetCache) Has(ctx context.Context, itemType string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.types == nil {
		types, err := c.loadTypes(ctx)
		if err != nil {
			return false, err
		}

		c.types = make(map[string]struct{}, len(types))
		for _, supportedType := range types {
			c.types[supportedType] = struct{}{}
		}
	}

	_, has := c.types[itemType]
	return has, nil
}
