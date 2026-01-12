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

func (s *statePersister) createReconciliationResult(result *manage.ReconciliationResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := reconciliationResultChunkFilePath(
		s.stateDir,
		s.lastReconciliationResultChunk,
	)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastReconciliationResultChunk,
		lastChunkFilePath,
		reconciliationResultChunkFilePath,
		func(incrementBy int) {
			s.lastReconciliationResultChunk += incrementBy
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

	chunkResults := []*manage.ReconciliationResult{}
	err = json.Unmarshal(existingData, &chunkResults)
	if err != nil {
		return err
	}

	chunkResults = append(chunkResults, result)

	slices.SortFunc(
		chunkResults,
		func(a, b *manage.ReconciliationResult) int {
			return int(a.Created - b.Created)
		},
	)

	updatedData, err := json.Marshal(chunkResults)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.updateReconciliationResultChunkIndexEntries(
		s.lastReconciliationResultChunk,
		chunkResults,
	)
}

func (s *statePersister) cleanupReconciliationResults(
	thresholdDate time.Time,
) (map[string]*manage.ReconciliationResult, error) {
	keepResults, err := s.loadReconciliationResultsToKeep(thresholdDate)
	if err != nil {
		return nil, err
	}

	// Reset state for reconciliation results to rebuild the index and chunk files.
	err = s.resetReconciliationResultState()
	if err != nil {
		return nil, err
	}

	// In order to know the file size to test against the guide size,
	// we need to gradually persist reconciliation results.
	for _, result := range keepResults {
		err = s.createReconciliationResult(result)
		if err != nil {
			return nil, err
		}
	}

	return createReconciliationResultLookup(keepResults), nil
}

func (s *statePersister) resetReconciliationResultState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.removeIndexFile(
		reconciliationResultIndexFilePath,
	)
	if err != nil {
		return err
	}

	err = s.removeChunkFiles(
		s.lastReconciliationResultChunk,
		reconciliationResultChunkFilePath,
	)
	if err != nil {
		return err
	}

	s.reconciliationResultIndex = map[string]*indexLocation{}
	s.lastReconciliationResultChunk = 0

	return nil
}

func (s *statePersister) loadReconciliationResultsToKeep(
	thresholdDate time.Time,
) ([]*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resultChunks, err := s.loadAllReconciliationResultChunks()
	if err != nil {
		return nil, err
	}

	keepResults := []*manage.ReconciliationResult{}

	for _, chunk := range resultChunks {
		entities := reconciliationResultsToEntities(chunk)
		deleteUpToIndex := findIndexBeforeThreshold(entities, thresholdDate)

		if deleteUpToIndex >= 0 && deleteUpToIndex < len(chunk)-1 {
			// Only include reconciliation results in the recreated state that are
			// newer than the threshold date.
			keepResults = append(
				keepResults,
				chunk[deleteUpToIndex+1:]...,
			)
		}
	}

	return keepResults, nil
}

func (s *statePersister) loadAllReconciliationResultChunks() (
	[][]*manage.ReconciliationResult,
	error,
) {
	// If there are no reconciliation results in the index,
	// there are no chunk files to load.
	if len(s.reconciliationResultIndex) == 0 {
		return nil, nil
	}

	resultChunks := [][]*manage.ReconciliationResult{}

	for i := 0; i <= s.lastReconciliationResultChunk; i++ {
		chunkFilePath := reconciliationResultChunkFilePath(s.stateDir, i)
		existingData, err := afero.ReadFile(s.fs, chunkFilePath)
		if err != nil {
			return nil, err
		}
		chunkResults := []*manage.ReconciliationResult{}
		err = json.Unmarshal(existingData, &chunkResults)
		if err != nil {
			return nil, err
		}
		resultChunks = append(resultChunks, chunkResults)
	}

	return resultChunks, nil
}

func (s *statePersister) updateReconciliationResultChunkIndexEntries(
	chunkNumber int,
	resultChunk []*manage.ReconciliationResult,
) error {
	for i, result := range resultChunk {
		s.reconciliationResultIndex[result.ID] = &indexLocation{
			ChunkNumber:  chunkNumber,
			IndexInChunk: i,
		}
	}

	return s.persistReconciliationResultIndexFile()
}

func (s *statePersister) persistReconciliationResultIndexFile() error {
	indexData, err := json.Marshal(s.reconciliationResultIndex)
	if err != nil {
		return err
	}

	indexFilePath := reconciliationResultIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func reconciliationResultChunkFilePath(
	stateDir string,
	chunkIndex int,
) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("reconciliation_results_c%d.json", chunkIndex),
	)
}

func reconciliationResultIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "reconciliation_result_index.json")
}

func reconciliationResultsToEntities(
	resultChunk []*manage.ReconciliationResult,
) []manage.Entity {
	entities := make([]manage.Entity, len(resultChunk))

	for i, result := range resultChunk {
		entities[i] = result
	}

	return entities
}

func createReconciliationResultLookup(
	results []*manage.ReconciliationResult,
) map[string]*manage.ReconciliationResult {
	lookup := map[string]*manage.ReconciliationResult{}

	for _, result := range results {
		lookup[result.ID] = result
	}

	return lookup
}
