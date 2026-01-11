package memfile

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/spf13/afero"
)

const (
	eventBroadcastDelay = 5 * time.Millisecond
)

type eventsContainerImpl struct {
	events                        map[string]*manage.Event
	partitionEvents               map[string][]*manage.Event
	listeners                     map[string][]chan manage.Event
	recentlyQueuedEventsThreshold time.Duration
	fs                            afero.Fs
	persister                     *statePersister
	clock                         commoncore.Clock
	logger                        core.Logger
	mu                            *sync.RWMutex
}

func (c *eventsContainerImpl) Get(ctx context.Context, id string) (manage.Event, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	event, hasEvent := c.events[id]
	if !hasEvent {
		return manage.Event{}, manage.EventNotFoundError(id)
	}

	return copyEvent(event), nil
}

func (c *eventsContainerImpl) Save(ctx context.Context, event *manage.Event) error {
	eventLogger := c.logger.WithFields(
		core.StringLogField("eventId", event.ID),
	)

	// Don't defer releasing the lock to the end of the function
	// as we need to allow new stream calls to register listeners
	// that happen to coincide with the save event.
	c.mu.Lock()

	err := c.save(event, eventLogger)
	if err != nil {
		c.mu.Unlock()
		return err
	}

	c.mu.Unlock()

	// Allow some time for any pending listeners to register before broadcasting.
	time.Sleep(eventBroadcastDelay)

	c.mu.RLock()
	defer c.mu.RUnlock()

	partitionName := partitionNameForChannel(event.ChannelType, event.ChannelID)
	listeners, hasListeners := c.listeners[partitionName]
	if !hasListeners {
		eventLogger.Debug("no listeners for event channel, skipping broadcast")
		return nil
	}

	eventLogger.Debug("broadcasting saved event to listeners")
	for _, listener := range listeners {
		select {
		case <-ctx.Done():
			eventLogger.Debug("context done, stopping event broadcast")
			return nil
		case listener <- *event:
			eventLogger.Debug(
				"event broadcasted to stream listener",
				core.StringLogField("listenerChannel", partitionName),
			)
		}
	}

	return nil
}

func (c *eventsContainerImpl) save(
	event *manage.Event,
	eventLogger core.Logger,
) error {

	eventCopy := copyEvent(event)
	c.events[event.ID] = &eventCopy

	partitionName := partitionNameForChannel(event.ChannelType, event.ChannelID)
	partition, hasPartition := c.partitionEvents[partitionName]
	if !hasPartition {
		partition = []*manage.Event{}
	}
	insertedIndex := insertEventIntoPartition(
		&partition,
		&eventCopy,
	)
	c.partitionEvents[partitionName] = partition

	eventLogger.Debug("persisting event partition update/creation")
	return c.persister.saveEventPartition(
		partitionName,
		partition,
		event,
		insertedIndex,
	)
}

func (c *eventsContainerImpl) Stream(
	ctx context.Context,
	params *manage.EventStreamParams,
	streamTo chan manage.Event,
	errChan chan error,
) (chan struct{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	endChan := make(chan struct{})

	partitionName := partitionNameForChannel(params.ChannelType, params.ChannelID)
	partition, hasPartition := c.partitionEvents[partitionName]
	if !hasPartition {
		partition = []*manage.Event{}
	}

	eventsToStream, err := c.collectInitialEventsToStream(
		partition,
		params,
	)
	if err != nil {
		return nil, err
	}

	lastEvent := (*manage.Event)(nil)
	if len(partition) > 0 {
		lastEvent = partition[len(partition)-1]
	}

	go c.streamEvents(
		ctx,
		params,
		lastEvent,
		eventsToStream,
		partitionName,
		streamTo,
		endChan,
	)

	return endChan, nil
}

func (c *eventsContainerImpl) streamEvents(
	ctx context.Context,
	params *manage.EventStreamParams,
	// The last event queued for the stream,
	// this may or may not be included in the initialEvents
	// but must be provided to be able to check if the last event
	// in the stream's partition is an end event, regardless if it is
	// not in the recently queued events time window.
	lastEvent *manage.Event,
	initialEvents []*manage.Event,
	partitionName string,
	streamTo chan manage.Event,
	endChan chan struct{},
) {
	internalEventChan := make(chan manage.Event)
	c.addEventListener(partitionName, internalEventChan)
	defer c.removeEventListener(partitionName, internalEventChan)

	// If the last event in the stream is an end event,
	// there are no recently queued events
	// and a starting event ID was not requested,
	// send just the end event to indicate to the caller
	// that there will be no more events in the stream.
	// The caller is then expected to send an empty struct to the endChan
	// to stop the stream.
	// Callers can bypass this by providing a specific starting event ID
	// if they want to stream historical events that have not yet been
	// cleaned up for the given channel.
	if len(initialEvents) == 0 &&
		lastEvent != nil &&
		lastEvent.End &&
		params.StartingEventID == "" {
		select {
		case <-ctx.Done():
			return
		case streamTo <- *lastEvent:
		}
	}

	for _, event := range initialEvents {
		select {
		case <-ctx.Done():
			return
		case <-endChan:
			return
		case streamTo <- *event:
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-endChan:
			return
		case event := <-internalEventChan:
			select {
			case <-ctx.Done():
				return
			case <-endChan:
				return
			case streamTo <- event:
			}
		}
	}
}

func (c *eventsContainerImpl) addEventListener(
	partitionName string,
	eventChan chan manage.Event,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	listeners, hasListeners := c.listeners[partitionName]
	if !hasListeners {
		listeners = []chan manage.Event{}
	}
	c.listeners[partitionName] = append(listeners, eventChan)
}

func (c *eventsContainerImpl) removeEventListener(
	partitionName string,
	eventChan chan manage.Event,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	listeners, hasListeners := c.listeners[partitionName]
	if !hasListeners {
		return
	}

	listenerIndex := slices.Index(listeners, eventChan)
	if listenerIndex >= 0 {
		c.listeners[partitionName] = slices.Delete(
			listeners,
			listenerIndex,
			listenerIndex+1,
		)
	}
}

func (c *eventsContainerImpl) collectInitialEventsToStream(
	partition []*manage.Event,
	params *manage.EventStreamParams,
) ([]*manage.Event, error) {
	if params.StartingEventID == "" {
		return c.extractRecentlyQueuedEvents(partition), nil
	}

	indexLocation := c.persister.getEventIndexEntry(params.StartingEventID)
	if indexLocation == nil {
		return c.extractRecentlyQueuedEvents(partition), nil
	}

	startingEventIndex := indexLocation.IndexInPartition
	if startingEventIndex < 0 || startingEventIndex >= len(partition) {
		return nil, errMalformedState(
			"malformed event index entry, location in partition is out of bounds",
		)
	}

	// Events are sorted in ascending order by the raw bytes of their IDs.
	// Any events after the starting event ID that have been saved need
	// to be streamed.
	// StartingEventID is exclusive, so we start from the next index.
	exclusiveStartIndex := startingEventIndex + 1
	if exclusiveStartIndex >= len(partition) {
		return []*manage.Event{}, nil
	}
	return partition[exclusiveStartIndex:], nil
}

func (c *eventsContainerImpl) extractRecentlyQueuedEvents(
	partition []*manage.Event,
) []*manage.Event {
	entities := eventsToEntities(partition)
	thresholdDate := c.clock.Now().Add(-c.recentlyQueuedEventsThreshold)
	excludeUpToIndex := findIndexBeforeThreshold(
		entities,
		thresholdDate,
	)
	if excludeUpToIndex < 0 {
		return partition
	}

	return partition[excludeUpToIndex+1:]
}

func (c *eventsContainerImpl) GetLastEventID(
	ctx context.Context,
	channelType, channelID string,
) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	partitionName := partitionNameForChannel(channelType, channelID)
	partition, hasPartition := c.partitionEvents[partitionName]
	if !hasPartition || len(partition) == 0 {
		return "", nil
	}

	return partition[len(partition)-1].ID, nil
}

func (c *eventsContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	removedEvents := []string{}
	removedPartitions := []string{}
	for partitionName, partition := range c.partitionEvents {
		entities := eventsToEntities(partition)
		deleteUpToIndex := findIndexBeforeThreshold(
			entities,
			thresholdDate,
		)

		if deleteUpToIndex == len(partition)-1 {
			// All events in the partition are older than the threshold date,
			// so we can delete the entire partition.
			delete(c.partitionEvents, partitionName)
			removedPartitions = append(removedPartitions, partitionName)
			removedEvents = append(
				removedEvents,
				extractEventIDs(partition)...,
			)
		} else if deleteUpToIndex >= 0 {
			beforeDeletion := make([]*manage.Event, len(partition))
			copy(beforeDeletion, partition)

			c.partitionEvents[partitionName] = slices.Delete(
				partition,
				0,
				deleteUpToIndex+1,
			)
			removedEvents = append(
				removedEvents,
				extractEventIDs(beforeDeletion[:deleteUpToIndex+1])...,
			)
		}
	}

	c.removeEventsFromMemoryLookup(removedEvents)

	err := c.persister.updateEventPartitionsForRemovals(
		c.partitionEvents,
		removedPartitions,
		removedEvents,
	)
	return int64(len(removedEvents)), err
}

func (c *eventsContainerImpl) removeEventsFromMemoryLookup(
	removedEvents []string,
) {
	for _, eventID := range removedEvents {
		delete(c.events, eventID)
	}
}

func eventsToEntities(partition []*manage.Event) []manage.Entity {
	entities := make([]manage.Entity, len(partition))
	for i, event := range partition {
		entities[i] = event
	}
	return entities
}

func extractEventIDs(partition []*manage.Event) []string {
	ids := make([]string, len(partition))
	for i, event := range partition {
		ids[i] = event.ID
	}
	return ids
}

func insertEventIntoPartition(
	partition *[]*manage.Event,
	event *manage.Event,
) int {
	if len(*partition) == 0 {
		*partition = append(*partition, event)
		return 0
	}

	// Events in each partition are sorted by the ID,
	// where the raw bytes of each ID are compared.
	// This assumes that the IDs are in a sequential time-based format
	// (e.g. UUIDv7).
	*partition = append(*partition, event)
	slices.SortFunc(*partition, func(a, b *manage.Event) int {
		return bytes.Compare(
			[]byte(a.ID),
			[]byte(b.ID),
		)
	})

	return slices.IndexFunc(*partition, func(current *manage.Event) bool {
		return current.ID == event.ID
	})
}

func copyEvent(event *manage.Event) manage.Event {
	return manage.Event{
		ID:          event.ID,
		Type:        event.Type,
		ChannelType: event.ChannelType,
		ChannelID:   event.ChannelID,
		Data:        event.Data,
		Timestamp:   event.Timestamp,
		End:         event.End,
	}
}

func partitionNameForChannel(
	channelType string,
	channelID string,
) string {
	return fmt.Sprintf(
		"%s_%s",
		channelType,
		channelID,
	)
}
