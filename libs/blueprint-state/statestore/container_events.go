package statestore

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

const eventBroadcastDelay = 5 * time.Millisecond

// EventsContainer implements manage.Events against a shared statestore.State
// and Persister. Listener channels for Stream live in the container; events
// are broadcast to them after persistence completes.
type EventsContainer struct {
	state                         *State
	persister                     *Persister
	listeners                     map[string][]chan manage.Event
	recentlyQueuedEventsThreshold time.Duration
	clock                         commoncore.Clock
	logger                        core.Logger
}

// EventsContainerOption configures an EventsContainer at construction.
type EventsContainerOption func(*EventsContainer)

// WithEventsRecentlyQueuedThreshold sets the window used when a Stream call
// doesn't provide a StartingEventID — events created within this duration
// of "now" are sent to the new listener.
func WithEventsRecentlyQueuedThreshold(d time.Duration) EventsContainerOption {
	return func(c *EventsContainer) { c.recentlyQueuedEventsThreshold = d }
}

// WithEventsClock injects a clock for deterministic testing of the recently-
// queued events window.
func WithEventsClock(clock commoncore.Clock) EventsContainerOption {
	return func(c *EventsContainer) { c.clock = clock }
}

func NewEventsContainer(
	st *State,
	persister *Persister,
	logger core.Logger,
	opts ...EventsContainerOption,
) *EventsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	c := &EventsContainer{
		state:                         st,
		persister:                     persister,
		listeners:                     map[string][]chan manage.Event{},
		recentlyQueuedEventsThreshold: manage.DefaultRecentlyQueuedEventsThreshold,
		clock:                         &commoncore.SystemClock{},
		logger:                        logger,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *EventsContainer) Get(ctx context.Context, id string) (manage.Event, error) {
	e, ok, err := c.state.LookupEvent(ctx, id)
	if err != nil {
		return manage.Event{}, err
	}

	if !ok {
		return manage.Event{}, manage.EventNotFoundError(id)
	}

	return copyEvent(e), nil
}

func (c *EventsContainer) Save(ctx context.Context, event *manage.Event) error {
	eventLogger := c.logger.WithFields(core.StringLogField("eventId", event.ID))

	// Don't defer releasing the lock to end-of-function: new Stream callers
	// may want to register listeners during the broadcast window below.
	c.state.Lock()
	if err := c.save(ctx, event, eventLogger); err != nil {
		c.state.Unlock()
		return err
	}
	c.state.Unlock()

	time.Sleep(eventBroadcastDelay)

	c.state.RLock()
	defer c.state.RUnlock()

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

func (c *EventsContainer) save(
	ctx context.Context,
	event *manage.Event,
	eventLogger core.Logger,
) error {
	eventCopy := copyEvent(event)
	c.state.events[event.ID] = &eventCopy

	partitionName := partitionNameForChannel(event.ChannelType, event.ChannelID)
	partition, hasPartition := c.state.partitionEvents[partitionName]
	if !hasPartition {
		partition = []*manage.Event{}
	}
	insertedIndex := insertEventIntoPartition(&partition, &eventCopy)
	c.state.partitionEvents[partitionName] = partition

	eventLogger.Debug("persisting event partition update/creation")
	return c.persister.SaveEventPartition(ctx, partitionName, partition, event, insertedIndex)
}

func (c *EventsContainer) Stream(
	ctx context.Context,
	params *manage.EventStreamParams,
	streamTo chan manage.Event,
	errChan chan error,
) (chan struct{}, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	endChan := make(chan struct{})
	partitionName := partitionNameForChannel(params.ChannelType, params.ChannelID)
	partition := c.state.partitionEvents[partitionName]

	eventsToStream, err := c.collectInitialEventsToStream(partition, params)
	if err != nil {
		return nil, err
	}

	var lastEvent *manage.Event
	if len(partition) > 0 {
		lastEvent = partition[len(partition)-1]
	}
	go c.streamEvents(ctx, params, lastEvent, eventsToStream, partitionName, streamTo, endChan)
	return endChan, nil
}

func (c *EventsContainer) streamEvents(
	ctx context.Context,
	params *manage.EventStreamParams,
	lastEvent *manage.Event,
	initialEvents []*manage.Event,
	partitionName string,
	streamTo chan manage.Event,
	endChan chan struct{},
) {
	internalEventChan := make(chan manage.Event)
	c.addEventListener(partitionName, internalEventChan)
	defer c.removeEventListener(partitionName, internalEventChan)

	// If the last event is a stream end marker, no initial events are queued
	// and no starting ID was given, send just the end event so the caller
	// can stop the stream cleanly.
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

func (c *EventsContainer) addEventListener(partitionName string, eventChan chan manage.Event) {
	c.state.Lock()
	defer c.state.Unlock()
	c.listeners[partitionName] = append(c.listeners[partitionName], eventChan)
}

func (c *EventsContainer) removeEventListener(partitionName string, eventChan chan manage.Event) {
	c.state.Lock()
	defer c.state.Unlock()
	listeners, ok := c.listeners[partitionName]
	if !ok {
		return
	}
	idx := slices.Index(listeners, eventChan)
	if idx >= 0 {
		c.listeners[partitionName] = slices.Delete(listeners, idx, idx+1)
	}
}

func (c *EventsContainer) collectInitialEventsToStream(
	partition []*manage.Event,
	params *manage.EventStreamParams,
) ([]*manage.Event, error) {
	if params.StartingEventID == "" {
		return c.extractRecentlyQueuedEvents(partition), nil
	}

	indexLocation := c.persister.GetEventIndexEntry(params.StartingEventID)
	if indexLocation == nil {
		return c.extractRecentlyQueuedEvents(partition), nil
	}
	startingEventIndex := indexLocation.IndexInPartition
	if startingEventIndex < 0 || startingEventIndex >= len(partition) {
		return nil, errMalformedState(
			"malformed event index entry, location in partition is out of bounds",
		)
	}
	exclusiveStartIndex := startingEventIndex + 1
	if exclusiveStartIndex >= len(partition) {
		return []*manage.Event{}, nil
	}
	return partition[exclusiveStartIndex:], nil
}

func (c *EventsContainer) extractRecentlyQueuedEvents(partition []*manage.Event) []*manage.Event {
	entities := eventsToEntities(partition)
	thresholdDate := c.clock.Now().Add(-c.recentlyQueuedEventsThreshold)
	excludeUpToIndex := findIndexBeforeThreshold(entities, thresholdDate)
	if excludeUpToIndex < 0 {
		return partition
	}
	return partition[excludeUpToIndex+1:]
}

func (c *EventsContainer) GetLastEventID(
	ctx context.Context,
	channelType, channelID string,
) (string, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	partitionName := partitionNameForChannel(channelType, channelID)
	partition := c.state.partitionEvents[partitionName]
	if len(partition) == 0 {
		return "", nil
	}
	return partition[len(partition)-1].ID, nil
}

func (c *EventsContainer) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.state.Lock()
	defer c.state.Unlock()

	var removedEvents []string
	var removedPartitions []string
	for partitionName, partition := range c.state.partitionEvents {
		entities := eventsToEntities(partition)
		deleteUpToIndex := findIndexBeforeThreshold(entities, thresholdDate)

		if deleteUpToIndex == len(partition)-1 {
			delete(c.state.partitionEvents, partitionName)
			removedPartitions = append(removedPartitions, partitionName)
			removedEvents = append(removedEvents, extractEventIDs(partition)...)
		} else if deleteUpToIndex >= 0 {
			before := make([]*manage.Event, len(partition))
			copy(before, partition)
			c.state.partitionEvents[partitionName] = slices.Delete(partition, 0, deleteUpToIndex+1)
			removedEvents = append(removedEvents, extractEventIDs(before[:deleteUpToIndex+1])...)
		}
	}

	for _, eventID := range removedEvents {
		delete(c.state.events, eventID)
	}

	err := c.persister.UpdateEventPartitionsForRemovals(
		ctx, c.state.partitionEvents, removedPartitions, removedEvents,
	)

	return int64(len(removedEvents)), err
}

func eventsToEntities(partition []*manage.Event) []manage.Entity {
	entities := make([]manage.Entity, len(partition))
	for i, e := range partition {
		entities[i] = e
	}
	return entities
}

func extractEventIDs(partition []*manage.Event) []string {
	ids := make([]string, len(partition))
	for i, e := range partition {
		ids[i] = e.ID
	}
	return ids
}

// Appends an event and re-sorts by raw ID bytes,
// assuming IDs are sequential time-based (e.g. UUIDv7). Returns the inserted
// event's post-sort index.
func insertEventIntoPartition(partition *[]*manage.Event, event *manage.Event) int {
	if len(*partition) == 0 {
		*partition = append(*partition, event)
		return 0
	}
	*partition = append(*partition, event)
	slices.SortFunc(*partition, func(a, b *manage.Event) int {
		return bytes.Compare([]byte(a.ID), []byte(b.ID))
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

func partitionNameForChannel(channelType, channelID string) string {
	return fmt.Sprintf("%s_%s", channelType, channelID)
}

// Scans entities from newest to oldest and returns
// the highest index whose Created time is before thresholdDate, or -1 if
// none are.
func findIndexBeforeThreshold(entities []manage.Entity, thresholdDate time.Time) int {
	for i := len(entities) - 1; i >= 0; i-- {
		if time.Unix(entities[i].GetCreated(), 0).Before(thresholdDate) {
			return i
		}
	}
	return -1
}
