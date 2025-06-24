package memfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/spf13/afero"
)

const (
	malformedChangesetStateFileMessage = "change set state file is malformed"
)

func (s *statePersister) createChangeset(changeset *manage.Changeset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := changesetChunkFilePath(s.stateDir, s.lastChangesetChunk)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastChangesetChunk,
		lastChunkFilePath,
		changesetChunkFilePath,
		func(incrementBy int) {
			s.lastChangesetChunk += incrementBy
		},
	)
	if err != nil {
		return err
	}

	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		if !errors.Is(err, afero.ErrFileNotFound) {
			return err
		}
		existingData = []byte("[]")
	}

	chunkChangesets := []*manage.Changeset{}
	err = json.Unmarshal(existingData, &chunkChangesets)
	if err != nil {
		return err
	}

	chunkChangesets = append(chunkChangesets, changeset)

	slices.SortFunc(
		chunkChangesets,
		func(a, b *manage.Changeset) int {
			return int(a.Created - b.Created)
		},
	)

	updatedData, err := json.Marshal(chunkChangesets)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.updateChangesetChunkIndexEntries(
		s.lastChangesetChunk,
		chunkChangesets,
	)
}

func (s *statePersister) updateChangeset(changeset *manage.Changeset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateChangesetEntry(changeset)
	if err != nil {
		return err
	}

	info.chunkInstances = slices.Delete(
		info.chunkInstances,
		info.entry.IndexInChunk,
		info.entry.IndexInChunk+1,
	)
	info.chunkInstances = append(info.chunkInstances, changeset)

	// Every time we update a change set,
	// we need to re-sort the chunk by timestamp.
	slices.SortFunc(
		info.chunkInstances,
		func(a, b *manage.Changeset) int {
			return int(a.Created - b.Created)
		},
	)

	updatedData, err := json.Marshal(info.chunkInstances)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.updateChangesetChunkIndexEntries(
		info.entry.ChunkNumber,
		info.chunkInstances,
	)
}

type persistedChangesetInfo struct {
	chunkInstances []*manage.Changeset
	entry          *indexLocation
	chunkFilePath  string
}

// A lock must be held when calling this method.
func (s *statePersister) loadAndValidateChangesetEntry(
	changeset *manage.Changeset,
) (*persistedChangesetInfo, error) {
	entry, hasEntry := s.changesetIndex[changeset.ID]
	if !hasEntry {
		return nil, manage.ChangesetNotFoundError(changeset.ID)
	}

	chunkFilePath := changesetChunkFilePath(s.stateDir, entry.ChunkNumber)
	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return nil, err
	}

	chunkInstances := []*manage.Changeset{}
	err = json.Unmarshal(existingData, &chunkInstances)
	if err != nil {
		return nil, err
	}

	if entry.IndexInChunk == -1 ||
		entry.IndexInChunk >= len(chunkInstances) {
		return nil, errMalformedStateFile(malformedChangesetStateFileMessage)
	}

	return &persistedChangesetInfo{
		chunkInstances: chunkInstances,
		entry:          entry,
		chunkFilePath:  chunkFilePath,
	}, nil
}

func (s *statePersister) cleanupChangesets(
	thresholdDate time.Time,
) (map[string]*manage.Changeset, error) {
	keepChangesets, err := s.loadChangesetsToKeep(thresholdDate)
	if err != nil {
		return nil, err
	}

	// Reset state for change sets to rebuild the index and chunk files.
	err = s.resetChangesetState()
	if err != nil {
		return nil, err
	}

	// In order to know the file size to test against the guide size,
	// we need to gradually persist change sets.
	// We don't know the file size until the changesets are serialised
	// in a JSON array form.
	// This isn't a very performant approach, however, clean up should be something
	// that is done relatively infrequently. (e.g. once a day)
	for _, changeset := range keepChangesets {
		err = s.createChangeset(changeset)
		if err != nil {
			return nil, err
		}
	}

	return createChangesetLookup(keepChangesets), nil
}

func (s *statePersister) resetChangesetState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.removeIndexFile(
		changesetIndexFilePath,
	)
	if err != nil {
		return err
	}

	err = s.removeChunkFiles(
		s.lastChangesetChunk,
		changesetChunkFilePath,
	)
	if err != nil {
		return err
	}

	s.changesetIndex = map[string]*indexLocation{}
	s.lastChangesetChunk = 0

	return nil
}

func (s *statePersister) loadChangesetsToKeep(
	thresholdDate time.Time,
) ([]*manage.Changeset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	changesetChunks, err := s.loadAllChangesetChunks()
	if err != nil {
		return nil, err
	}

	keepChangesets := []*manage.Changeset{}

	for _, chunk := range changesetChunks {
		entities := changesetsToEntities(chunk)
		deleteUpToIndex := findIndexBeforeThreshold(entities, thresholdDate)

		if deleteUpToIndex >= 0 && deleteUpToIndex < len(chunk)-1 {
			// Only include changesets in the recreated state that are
			// newer than the threshold date.
			keepChangesets = append(
				keepChangesets,
				chunk[deleteUpToIndex+1:]...,
			)
		}
	}

	return keepChangesets, nil
}

func (s *statePersister) loadAllChangesetChunks() (
	[][]*manage.Changeset,
	error,
) {
	changesetChunks := [][]*manage.Changeset{}

	for i := 0; i <= s.lastChangesetChunk; i++ {
		chunkFilePath := changesetChunkFilePath(s.stateDir, i)
		existingData, err := afero.ReadFile(s.fs, chunkFilePath)
		if err != nil {
			return nil, err
		}
		chunkChangesets := []*manage.Changeset{}
		err = json.Unmarshal(existingData, &chunkChangesets)
		if err != nil {
			return nil, err
		}
		changesetChunks = append(changesetChunks, chunkChangesets)
	}

	return changesetChunks, nil
}

func (s *statePersister) updateChangesetChunkIndexEntries(
	chunkNumber int,
	changesetChunk []*manage.Changeset,
) error {
	for i, changeset := range changesetChunk {
		s.changesetIndex[changeset.ID] = &indexLocation{
			ChunkNumber:  chunkNumber,
			IndexInChunk: i,
		}
	}

	return s.persistChangesetIndexFile()
}

func (s *statePersister) persistChangesetIndexFile() error {
	indexData, err := json.Marshal(s.changesetIndex)
	if err != nil {
		return err
	}

	indexFilePath := changesetIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func changesetChunkFilePath(stateDir string, chunkIndex int) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("changesets_c%d.json", chunkIndex),
	)
}

func changesetIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "changeset_index.json")
}

func changesetsToEntities(
	changesetChunk []*manage.Changeset,
) []manage.Entity {
	entities := make([]manage.Entity, len(changesetChunk))

	for i, changeset := range changesetChunk {
		entities[i] = changeset
	}

	return entities
}

func createChangesetLookup(
	changesets []*manage.Changeset,
) map[string]*manage.Changeset {
	lookup := map[string]*manage.Changeset{}

	for _, changeset := range changesets {
		lookup[changeset.ID] = changeset
	}

	return lookup
}
