package memfile

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/spf13/afero"
)

// StateContainer provides the in-memory with file persistence (memfile)
// implementation of the blueprint `state.Container` interface
// along with methods to manage persistence for
// blueprint validation requests, events and change sets.
type StateContainer struct {
	instancesContainer             *statestore.InstancesContainer
	resourcesContainer             *statestore.ResourcesContainer
	linksContainer                 *statestore.LinksContainer
	childrenContainer              *statestore.ChildrenContainer
	metadataContainer              *statestore.MetadataContainer
	exportContainer                *statestore.ExportsContainer
	eventsContainer                *statestore.EventsContainer
	changesetsContainer            *statestore.ChangesetsContainer
	validationContainer            *statestore.ValidationsContainer
	reconciliationResultsContainer *statestore.ReconciliationResultsContainer
	cleanupOperationsContainer     *statestore.CleanupOperationsContainer
	storePersister                 *statestore.Persister
}

// Option is a type for options that can be passed to LoadStateContainer
// when creating an in-memory state container with file persistence.
type Option func(*StateContainer)

// WithMaxGuideFileSize sets a guide for the maximum size of a state chunk file in bytes.
// If a single record (instance or resource drift entry) exceeds this size,
// it will not be split into multiple files.
// This is only a guide, the actual size of the files are often likely to be larger.
//
// When not set, the default value is 1MB (1,048,576 bytes).
func WithMaxGuideFileSize(maxGuideFileSize int64) func(*StateContainer) {
	return func(c *StateContainer) {
		statestore.WithMaxGuideFileSize(maxGuideFileSize)(c.storePersister)
	}
}

// WithMaxEventPartitionSize sets a maximum size of an event partition file in bytes.
// If the addition of a new event causes the partition to exceeds this size,
// an error will be returned for the save event operation.
// This determines the maximum size of the data in the partition file,
// depending on the operating system and file system, the actual size of the file
// will in most cases be larger.
//
// When not set, the default value is 10MB (10,485,760 bytes).
func WithMaxEventPartitionSize(maxEventPartitionSize int64) func(*StateContainer) {
	return func(c *StateContainer) {
		statestore.WithMaxEventPartitionSize(maxEventPartitionSize)(c.storePersister)
	}
}

// WithRecentlyQueuedEventsThreshold sets the threshold in seconds
// for retrieving recently queued events for a stream when a starting event ID
// is not provided.
//
// When not set, the default value is 5 minutes (300 seconds).
func WithRecentlyQueuedEventsThreshold(thresholdSeconds int64) func(*StateContainer) {
	return func(c *StateContainer) {
		d := time.Duration(thresholdSeconds) * time.Second
		statestore.WithEventsRecentlyQueuedThreshold(d)(c.eventsContainer)
	}
}

// WithClock sets the clock to use for the state container.
// This is used in tasks like determining the current time when checking for
// recently queued events.
//
// When not set, the default value is the system clock.
func WithClock(clock commoncore.Clock) func(*StateContainer) {
	return func(c *StateContainer) {
		statestore.WithEventsClock(clock)(c.eventsContainer)
	}
}

// LoadStateContainer loads a new state container that uses in-process memory
// to store state with local files used for persistence. State is loaded from
// stateDir at construction time via statestore.Load; subsequent writes go
// back to the same directory. stateDir can be relative to the current
// working directory or an absolute path.
func LoadStateContainer(
	stateDir string,
	fs afero.Fs,
	logger core.Logger,
	opts ...Option,
) (*StateContainer, error) {
	storage := newFSStorage(fs)
	storeState := statestore.NewState()
	cfg := memfileStatestoreConfig()

	if err := statestore.Load(context.Background(), storeState, storage, cfg, stateDir); err != nil {
		return nil, err
	}

	storePersister := statestore.NewPersister(
		storeState,
		storage,
		cfg,
		stateDir,
		statestore.WithMaxGuideFileSize(DefaultMaxGuideFileSize),
		statestore.WithLogger(logger),
	)

	container := &StateContainer{
		storePersister: storePersister,
		instancesContainer: statestore.NewInstancesContainer(
			storeState,
			storePersister,
			statestore.SingleProcessClaimFunc(storeState, storePersister),
			statestore.SingleProcessInitialiseAndClaimFunc(storeState, storePersister),
			logger,
		),
		resourcesContainer:             statestore.NewResourcesContainer(storeState, storePersister, logger),
		linksContainer:                 statestore.NewLinksContainer(storeState, storePersister, logger),
		childrenContainer:              statestore.NewChildrenContainer(storeState, storePersister, logger),
		metadataContainer:              statestore.NewMetadataContainer(storeState, storePersister, logger),
		exportContainer:                statestore.NewExportsContainer(storeState, storePersister, logger),
		eventsContainer:                statestore.NewEventsContainer(storeState, storePersister, logger),
		changesetsContainer:            statestore.NewChangesetsContainer(storeState, storePersister, logger),
		validationContainer:            statestore.NewValidationsContainer(storeState, storePersister, logger),
		reconciliationResultsContainer: statestore.NewReconciliationResultsContainer(storeState, storePersister, logger),
		cleanupOperationsContainer:     statestore.NewCleanupOperationsContainer(storeState, storePersister, logger),
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

func (c *StateContainer) Events() manage.Events {
	return c.eventsContainer
}

func (c *StateContainer) Changesets() manage.Changesets {
	return c.changesetsContainer
}

func (c *StateContainer) Validation() manage.Validation {
	return c.validationContainer
}

func (c *StateContainer) ReconciliationResults() manage.ReconciliationResults {
	return c.reconciliationResultsContainer
}

func (c *StateContainer) CleanupOperations() manage.CleanupOperations {
	return c.cleanupOperationsContainer
}

// The statestore.Config used by memfile:
// ModeEager so state is bulk-loaded from disk at construction; every entity
// category keeps the chunked, globally-scoped layout memfile has used
// historically. Cleanup operations sit on LayoutPerEntity — one tiny JSON
// per operation — because that category never had a chunked file layout.
func memfileStatestoreConfig() statestore.Config {
	chunkedGlobal := statestore.CategoryConfig{
		Layout: statestore.LayoutChunked,
		Scope:  statestore.ScopeGlobal,
	}
	chunkedPerChannel := statestore.CategoryConfig{
		Layout: statestore.LayoutChunked,
		Scope:  statestore.ScopePerChannel,
	}
	perEntityGlobal := statestore.CategoryConfig{
		Layout: statestore.LayoutPerEntity,
		Scope:  statestore.ScopeGlobal,
	}
	return statestore.Config{
		Mode:                  statestore.ModeEager,
		Instances:             chunkedGlobal,
		Resources:             chunkedGlobal,
		Links:                 chunkedGlobal,
		ResourceDrift:         chunkedGlobal,
		LinkDrift:             chunkedGlobal,
		Events:                chunkedPerChannel,
		Changesets:            chunkedGlobal,
		Validations:           chunkedGlobal,
		ReconciliationResults: chunkedGlobal,
		CleanupOperations:     perEntityGlobal,
	}
}
