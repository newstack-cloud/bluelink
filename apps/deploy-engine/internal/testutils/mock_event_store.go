package testutils

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

type MockEventStore struct {
	Events          map[string]*manage.Event
	EventPartitions map[string][]*manage.Event
	mu              sync.Mutex
}

func NewMockEventStore(
	events map[string]*manage.Event,
) manage.Events {
	return &MockEventStore{
		Events:          events,
		EventPartitions: partitionEvents(events),
	}
}

func (s *MockEventStore) Get(
	ctx context.Context,
	id string,
) (manage.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event, ok := s.Events[id]; ok {
		return *event, nil
	}

	return manage.Event{}, manage.EventNotFoundError(id)
}

func (s *MockEventStore) Save(
	ctx context.Context,
	evt *manage.Event,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Events[evt.ID] = evt
	// Partitions must retain order, so avoid rebuilding
	// the partition map.
	partitionKey := createPartitionKey(evt.ChannelType, evt.ChannelID)
	partition, partitionExists := s.EventPartitions[partitionKey]
	if partitionExists {
		insertedInExistingPosition := false
		for i, event := range partition {
			if event.ID == evt.ID {
				partition[i] = evt
				insertedInExistingPosition = true
				break
			}
		}

		if !insertedInExistingPosition {
			s.EventPartitions[partitionKey] = append(partition, evt)
		}
	} else {
		s.EventPartitions[partitionKey] = []*manage.Event{evt}
	}

	return nil
}

func (s *MockEventStore) Stream(
	ctx context.Context,
	params *manage.EventStreamParams,
	streamTo chan manage.Event,
	errChan chan error,
) (chan struct{}, error) {
	// Keep the implementation simple for the purpose
	// of tests.
	// This will not make use of the starting event ID
	// param.
	endChan := make(chan struct{})
	go func() {
		sentEvents := []string{}
		for {
			s.mu.Lock()
			partitionKey := createPartitionKey(params.ChannelType, params.ChannelID)
			partition, exists := s.EventPartitions[partitionKey]
			s.mu.Unlock()
			if exists && len(partition) > 0 {
				reachedEndOfStream := false
				i := 0
				for !reachedEndOfStream && i < len(partition) {
					event := partition[i]
					if !slices.Contains(sentEvents, event.ID) {
						select {
						case streamTo <- *event:
							sentEvents = append(sentEvents, event.ID)
							reachedEndOfStream = event.End
						case <-endChan:
							// Stream has been closed by the caller,
							// exit the stream.
							return
						case <-ctx.Done():
							// Context is done, exit the stream.
							return
						}
					}
					i += 1
				}
			}
			// Wait to allow for events to be added to the partition before
			// checking again, a lock must not be held while waiting.
			time.Sleep(50 * time.Millisecond)
		}
	}()

	return endChan, nil
}

func (s *MockEventStore) Cleanup(ctx context.Context, thresholdDate time.Time) (int64, error) {
	// This is a no-op for the mock event store.
	return 0, nil
}

func (s *MockEventStore) GetLastEventID(
	ctx context.Context,
	channelType, channelID string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	partitionKey := createPartitionKey(channelType, channelID)
	partition, exists := s.EventPartitions[partitionKey]
	if !exists || len(partition) == 0 {
		return "", nil
	}

	return partition[len(partition)-1].ID, nil
}

func partitionEvents(
	events map[string]*manage.Event,
) map[string][]*manage.Event {
	partitioned := make(map[string][]*manage.Event)
	for _, event := range events {
		partition := createPartitionKey(
			event.ChannelType,
			event.ChannelID,
		)
		if _, ok := partitioned[partition]; !ok {
			partitioned[partition] = []*manage.Event{}
		}
		partitioned[partition] = append(partitioned[partition], event)
	}
	return partitioned
}

func createPartitionKey(
	channelType string,
	channelID string,
) string {
	return fmt.Sprintf("%s::%s", channelType, channelID)
}
