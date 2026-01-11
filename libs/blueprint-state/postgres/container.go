package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// StateContainer provides the postgres implementation of
// the blueprint `state.Container` interface
// along with methods to manage persistence for
// blueprint validation requests, events and change sets.
type StateContainer struct {
	instancesContainer              *instancesContainerImpl
	resourcesContainer              *resourcesContainerImpl
	linksContainer                  *linksContainerImpl
	childrenContainer               *childrenContainerImpl
	metadataContainer               *metadataContainerImpl
	exportContainer                 *exportContainerImpl
	validationContainer             *validationContainerImpl
	changesetsContainer             *changesetsContainerImpl
	eventsContainer                 *eventsContainerImpl
	reconciliationResultsContainer  *reconciliationResultsContainerImpl
	cleanupOperationsContainer      *cleanupOperationsContainerImpl
}

// Option is a type for options that can be passed to LoadStateContainer
// when creating a postgres state container.
type Option func(*StateContainer)

// WithClock sets the clock to use for the state container.
// This is used in tasks like determining the current time when checking for
// recently queued events.
//
// When not set, the default value is the system clock.
func WithClock(clock commoncore.Clock) func(*StateContainer) {
	return func(c *StateContainer) {
		c.eventsContainer.clock = clock
	}
}

// WithRecentlyQueuedEventsThreshold sets the threshold in seconds
// for retrieving recently queued events for a stream when a starting event ID
// is not provided.
//
// When not set, the default value is 5 minutes (300 seconds).
func WithRecentlyQueuedEventsThreshold(thresholdSeconds int64) func(*StateContainer) {
	return func(c *StateContainer) {
		c.eventsContainer.recentlyQueuedEventsThreshold = time.Duration(thresholdSeconds) * time.Second
	}
}

// LoadStateContainer loads a new state container
// that uses postgres for persistence.
//
// The postgres connection pool must be configured appropriately
// in the calling application where the application will take care of making
// sure the connection pool is cleaned up when a command is finished or the application
// is shutting down.
func LoadStateContainer(
	ctx context.Context,
	connPool *pgxpool.Pool,
	logger core.Logger,
	opts ...Option,
) (*StateContainer, error) {
	instancesContainer := &instancesContainerImpl{
		connPool: connPool,
	}

	container := &StateContainer{
		instancesContainer: instancesContainer,
		resourcesContainer: &resourcesContainerImpl{
			connPool: connPool,
		},
		linksContainer: &linksContainerImpl{
			connPool: connPool,
		},
		childrenContainer: &childrenContainerImpl{
			connPool:  connPool,
			instances: instancesContainer,
		},
		metadataContainer: &metadataContainerImpl{
			connPool: connPool,
		},
		exportContainer: &exportContainerImpl{
			connPool: connPool,
		},
		eventsContainer: &eventsContainerImpl{
			connPool:                      connPool,
			logger:                        logger,
			clock:                         &commoncore.SystemClock{},
			recentlyQueuedEventsThreshold: manage.DefaultRecentlyQueuedEventsThreshold,
		},
		changesetsContainer: &changesetsContainerImpl{
			connPool: connPool,
			logger:   logger,
		},
		validationContainer: &validationContainerImpl{
			connPool: connPool,
			logger:   logger,
		},
		reconciliationResultsContainer: &reconciliationResultsContainerImpl{
			connPool: connPool,
			logger:   logger,
		},
		cleanupOperationsContainer: &cleanupOperationsContainerImpl{
			connPool: connPool,
			logger:   logger,
		},
	}

	for _, opt := range opts {
		opt(container)
	}

	return container, nil
}

func (c *StateContainer) Instances() state.InstancesContainer {
	return c.instancesContainer
}

func (c *StateContainer) Resources() state.ResourcesContainer {
	return c.resourcesContainer
}

func (c *StateContainer) Links() state.LinksContainer {
	return c.linksContainer
}

func (c *StateContainer) Children() state.ChildrenContainer {
	return c.childrenContainer
}

func (c *StateContainer) Metadata() state.MetadataContainer {
	return c.metadataContainer
}

func (c *StateContainer) Exports() state.ExportsContainer {
	return c.exportContainer
}

func (c *StateContainer) Validation() manage.Validation {
	return c.validationContainer
}

func (c *StateContainer) Changesets() manage.Changesets {
	return c.changesetsContainer
}

func (c *StateContainer) Events() manage.Events {
	return c.eventsContainer
}

func (c *StateContainer) ReconciliationResults() manage.ReconciliationResults {
	return c.reconciliationResultsContainer
}

func (c *StateContainer) CleanupOperations() manage.CleanupOperations {
	return c.cleanupOperationsContainer
}
