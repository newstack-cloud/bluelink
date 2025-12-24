package memfile

import (
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
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
	instancesContainer  *instancesContainerImpl
	resourcesContainer  *resourcesContainerImpl
	linksContainer      *linksContainerImpl
	childrenContainer   *childrenContainerImpl
	metadataContainer   *metadataContainerImpl
	exportContainer     *exportContainerImpl
	eventsContainer     *eventsContainerImpl
	changesetsContainer *changesetsContainerImpl
	validationContainer *validationContainerImpl
	persister           *statePersister
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
		c.persister.maxGuideFileSize = maxGuideFileSize
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
		c.persister.maxEventPartitionSize = maxEventPartitionSize
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

// LoadStateContainer loads a new state container
// that uses in-process memory to store state
// with local files used for persistence.
//
// This will load the state into memory from the given directory
// as the initial state and will write state files to the same
// directory as they are updated.
// stateDir can be relative to the current working directory
// or an absolute path.
func LoadStateContainer(
	stateDir string,
	fs afero.Fs,
	logger core.Logger,
	opts ...Option,
) (*StateContainer, error) {
	mu := &sync.RWMutex{}

	state, err := loadStateFromDir(stateDir, fs)
	if err != nil {
		return nil, err
	}

	persister := &statePersister{
		stateDir:                     stateDir,
		fs:                           fs,
		instanceIndex:                state.instanceIndex,
		resourceDriftIndex:           state.resourceDriftIndex,
		linkDriftIndex:               state.linkDriftIndex,
		eventIndex:                   state.eventIndex,
		changesetIndex:               state.changesetIndex,
		blueprintValidationIndex:     state.blueprintValidationIndex,
		maxGuideFileSize:             DefaultMaxGuideFileSize,
		maxEventPartitionSize:        DefaultMaxEventParititionSize,
		lastInstanceChunk:            getLastChunkFromIndex(state.instanceIndex),
		lastResourceDriftChunk:       getLastChunkFromIndex(state.resourceDriftIndex),
		lastLinkDriftChunk:           getLastChunkFromIndex(state.linkDriftIndex),
		lastChangesetChunk:           getLastChunkFromIndex(state.changesetIndex),
		lastBlueprintValidationChunk: getLastChunkFromIndex(state.blueprintValidationIndex),
	}

	container := &StateContainer{
		persister: persister,
		instancesContainer: &instancesContainerImpl{
			instances: state.instances,
			// The instance ID lookup is not something that is persisted,
			// it is generated at load time as for the vast majority of use-cases
			// there will not be a significant cost to generating it on load.
			instanceIDLookup: createInstanceIDLookup(state.instances),
			resources:        state.resources,
			links:            state.links,
			fs:               fs,
			persister:        persister,
			logger:           logger,
			mu:               mu,
		},
		resourcesContainer: &resourcesContainerImpl{
			resources:            state.resources,
			resourceDriftEntries: state.resourceDrift,
			instances:            state.instances,
			fs:                   fs,
			persister:            persister,
			logger:               logger,
			mu:                   mu,
		},
		linksContainer: &linksContainerImpl{
			links:                  state.links,
			linkDriftEntries:       state.linkDrift,
			instances:              state.instances,
			fs:                     fs,
			persister:              persister,
			resourceDataMappingIDs: map[string]map[string][]string{},
			logger:                 logger,
			mu:                     mu,
		},
		childrenContainer: &childrenContainerImpl{
			instances: state.instances,
			fs:        fs,
			persister: persister,
			logger:    logger,
			mu:        mu,
		},
		metadataContainer: &metadataContainerImpl{
			instances: state.instances,
			fs:        fs,
			persister: persister,
			logger:    logger,
			mu:        mu,
		},
		exportContainer: &exportContainerImpl{
			instances: state.instances,
			fs:        fs,
			persister: persister,
			logger:    logger,
			mu:        mu,
		},
		eventsContainer: &eventsContainerImpl{
			events:                        state.events,
			partitionEvents:               state.partitionEvents,
			fs:                            fs,
			recentlyQueuedEventsThreshold: manage.DefaultRecentlyQueuedEventsThreshold,
			clock:                         &commoncore.SystemClock{},
			listeners:                     make(map[string][]chan manage.Event),
			persister:                     persister,
			logger:                        logger,
			mu:                            mu,
		},
		changesetsContainer: &changesetsContainerImpl{
			changesets: state.changesets,
			fs:         fs,
			persister:  persister,
			logger:     logger,
			mu:         mu,
		},
		validationContainer: &validationContainerImpl{
			validations: state.blueprintValidations,
			fs:          fs,
			persister:   persister,
			logger:      logger,
			mu:          mu,
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

func (c *StateContainer) Events() manage.Events {
	return c.eventsContainer
}

func (c *StateContainer) Changesets() manage.Changesets {
	return c.changesetsContainer
}

func (c *StateContainer) Validation() manage.Validation {
	return c.validationContainer
}

func getLastChunkFromIndex(index map[string]*indexLocation) int {
	lastChunk := 0
	for _, locationInfo := range index {
		if locationInfo.ChunkNumber > lastChunk {
			lastChunk = locationInfo.ChunkNumber
		}
	}
	return lastChunk
}

func createInstanceIDLookup(
	instances map[string]*state.InstanceState,
) map[string]string {
	instanceIDLookup := make(map[string]string)
	for instanceID, instance := range instances {
		if instance.InstanceName != "" {
			instanceIDLookup[instance.InstanceName] = instanceID
		}
	}
	return instanceIDLookup
}
