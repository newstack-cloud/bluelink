package memfile

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/spf13/afero"
)

func (s *statePersister) saveEventPartition(
	partitionName string,
	partition []*manage.Event,
	eventToSave *manage.Event,
	eventPartitionIndex int,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Unlike other entities modelled in the state container,
	// events are stored in partition files, where each channel
	// has its own partition.
	// As it is efficient to look up events by their channel in memory
	// for streaming events, a partition is managed in memory.
	// This means that all we need to do is persist the current in-memory
	// partition to disk.
	partitionFilePath := eventPartitionFilePath(s.stateDir, partitionName)

	partitionData, err := json.Marshal(partition)
	if err != nil {
		return err
	}

	if len(partitionData) > int(s.maxEventPartitionSize) {
		return errMaxEventPartitionSizeExceeded(
			partitionName,
			s.maxEventPartitionSize,
		)
	}

	err = afero.WriteFile(s.fs, partitionFilePath, partitionData, 0644)
	if err != nil {
		return err
	}

	return s.addEventToIndex(
		eventToSave,
		partitionName,
		eventPartitionIndex,
	)
}

func (s *statePersister) updateEventPartitionsForRemovals(
	partitions map[string][]*manage.Event,
	removedPartitions []string,
	removedEvents []string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, partitionName := range removedPartitions {
		partitionFilePath := eventPartitionFilePath(s.stateDir, partitionName)
		partitionFileExists, err := afero.Exists(s.fs, partitionFilePath)
		if err != nil {
			return err
		}

		if partitionFileExists {
			err := s.fs.Remove(partitionFilePath)
			if err != nil {
				return err
			}
		}
	}

	for partitionName, partition := range partitions {
		partitionFilePath := eventPartitionFilePath(s.stateDir, partitionName)

		partitionData, err := json.Marshal(partition)
		if err != nil {
			return err
		}

		err = afero.WriteFile(s.fs, partitionFilePath, partitionData, 0644)
		if err != nil {
			return err
		}
	}

	return s.removeEventIndexEntries(removedEvents)
}

// A lock must be held when calling this method.
func (s *statePersister) removeEventIndexEntries(eventIDs []string) error {
	for _, eventID := range eventIDs {
		delete(s.eventIndex, eventID)
	}

	return s.persistEventIndexFile()
}

// A lock must be held when calling this method.
func (s *statePersister) addEventToIndex(
	event *manage.Event,
	partitionName string,
	indexInPartition int,
) error {
	s.eventIndex[event.ID] = &eventIndexLocation{
		Partition:        toPartitionFileBaseName(partitionName),
		IndexInPartition: indexInPartition,
	}

	return s.persistEventIndexFile()
}

// Reads from the event index must be done through this method to ensure
// that the mutex is held when reading from the index.
// This must only be called for reading in contexts that are not holding
// a lock on the state persister.
func (s *statePersister) getEventIndexEntry(eventID string) *eventIndexLocation {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, hasEntry := s.eventIndex[eventID]
	if !hasEntry {
		return nil
	}

	return entry
}

func (s *statePersister) persistEventIndexFile() error {
	indexData, err := json.Marshal(s.eventIndex)
	if err != nil {
		return err
	}

	indexFilePath := eventIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func eventIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "event_index.json")
}

func eventPartitionFilePath(stateDir string, partitionName string) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("%s.json", toPartitionFileBaseName(partitionName)),
	)
}

func toPartitionFileBaseName(partitionName string) string {
	return fmt.Sprintf("events__%s", partitionName)
}
