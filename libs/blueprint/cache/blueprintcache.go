package cache

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// BlueprintCache provides an interface for caching blueprints.
// The blueprint framework does not provide a default implementation,
// this is to allow applications to build their own caching
// implementations.
type BlueprintCache interface {
	// Get a blueprint from the cache by key.
	Get(ctx context.Context, key string) (*schema.Blueprint, error)
	// Set a blueprint in the cache with the given key.
	Set(
		ctx context.Context,
		key string,
		blueprint *schema.Blueprint,
	) error
	// Set a blueprint in the cache with the given key for
	// the given duration.
	SetExpires(
		ctx context.Context,
		key string,
		blueprint *schema.Blueprint,
		expiresAfter time.Duration,
	) error
	// Delete a blueprint from the cache by key.
	Delete(
		ctx context.Context,
		key string,
	) error
}
