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
	malformedValidationStateFileMessage = "blueprint validation state file is malformed"
)

func (s *statePersister) createBlueprintValidation(validation *manage.BlueprintValidation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := blueprintValidationChunkFilePath(
		s.stateDir,
		s.lastBlueprintValidationChunk,
	)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastBlueprintValidationChunk,
		lastChunkFilePath,
		blueprintValidationChunkFilePath,
		func(incrementBy int) {
			s.lastBlueprintValidationChunk += incrementBy
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

	chunkValidations := []*manage.BlueprintValidation{}
	err = json.Unmarshal(existingData, &chunkValidations)
	if err != nil {
		return err
	}

	chunkValidations = append(chunkValidations, validation)

	slices.SortFunc(
		chunkValidations,
		func(a, b *manage.BlueprintValidation) int {
			return int(a.Created - b.Created)
		},
	)

	updatedData, err := json.Marshal(chunkValidations)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.updateBlueprintValidationChunkIndexEntries(
		s.lastBlueprintValidationChunk,
		chunkValidations,
	)
}

func (s *statePersister) updateBlueprintValidation(validation *manage.BlueprintValidation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateBlueprintValidationEntry(validation)
	if err != nil {
		return err
	}

	info.chunkInstances = slices.Delete(
		info.chunkInstances,
		info.entry.IndexInChunk,
		info.entry.IndexInChunk+1,
	)
	info.chunkInstances = append(info.chunkInstances, validation)

	// Every time we update a change set,
	// we need to re-sort the chunk by timestamp.
	slices.SortFunc(
		info.chunkInstances,
		func(a, b *manage.BlueprintValidation) int {
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

	return s.updateBlueprintValidationChunkIndexEntries(
		info.entry.ChunkNumber,
		info.chunkInstances,
	)
}

type persistedBlueprintValidationInfo struct {
	chunkInstances []*manage.BlueprintValidation
	entry          *indexLocation
	chunkFilePath  string
}

// A lock must be held when calling this method.
func (s *statePersister) loadAndValidateBlueprintValidationEntry(
	validation *manage.BlueprintValidation,
) (*persistedBlueprintValidationInfo, error) {
	entry, hasEntry := s.blueprintValidationIndex[validation.ID]
	if !hasEntry {
		return nil, manage.BlueprintValidationNotFoundError(validation.ID)
	}

	chunkFilePath := blueprintValidationChunkFilePath(s.stateDir, entry.ChunkNumber)
	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return nil, err
	}

	chunkInstances := []*manage.BlueprintValidation{}
	err = json.Unmarshal(existingData, &chunkInstances)
	if err != nil {
		return nil, err
	}

	if entry.IndexInChunk == -1 ||
		entry.IndexInChunk >= len(chunkInstances) {
		return nil, errMalformedStateFile(malformedValidationStateFileMessage)
	}

	return &persistedBlueprintValidationInfo{
		chunkInstances: chunkInstances,
		entry:          entry,
		chunkFilePath:  chunkFilePath,
	}, nil
}

func (s *statePersister) cleanupBlueprintValidations(
	thresholdDate time.Time,
) (map[string]*manage.BlueprintValidation, error) {
	keepValidations, err := s.loadBlueprintValidationsToKeep(thresholdDate)
	if err != nil {
		return nil, err
	}

	// Reset state for blueprint validations to rebuild the index and chunk files.
	err = s.resetBlueprintValidationState()
	if err != nil {
		return nil, err
	}

	// In order to know the file size to test against the guide size,
	// we need to gradually persist blueprint validations.
	// We don't know the file size until the validations are serialised
	// in a JSON array form.
	// This isn't a very performant approach, however, clean up should be something
	// that is done relatively infrequently. (e.g. once a day)
	for _, validation := range keepValidations {
		err = s.createBlueprintValidation(validation)
		if err != nil {
			return nil, err
		}
	}

	return createBlueprintValidationLookup(keepValidations), nil
}

func (s *statePersister) resetBlueprintValidationState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.removeIndexFile(
		blueprintValidationIndexFilePath,
	)
	if err != nil {
		return err
	}

	err = s.removeChunkFiles(
		s.lastBlueprintValidationChunk,
		blueprintValidationChunkFilePath,
	)
	if err != nil {
		return err
	}

	s.blueprintValidationIndex = map[string]*indexLocation{}
	s.lastBlueprintValidationChunk = 0

	return nil
}

func (s *statePersister) loadBlueprintValidationsToKeep(
	thresholdDate time.Time,
) ([]*manage.BlueprintValidation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	validationChunks, err := s.loadAllBlueprintValidationChunks()
	if err != nil {
		return nil, err
	}

	keepValidations := []*manage.BlueprintValidation{}

	for _, chunk := range validationChunks {
		entities := blueprintValidationsToEntities(chunk)
		deleteUpToIndex := findIndexBeforeThreshold(entities, thresholdDate)

		if deleteUpToIndex >= 0 && deleteUpToIndex < len(chunk)-1 {
			// Only include blueprint validations in the recreated state that are
			// newer than the threshold date.
			keepValidations = append(
				keepValidations,
				chunk[deleteUpToIndex+1:]...,
			)
		}
	}

	return keepValidations, nil
}

func (s *statePersister) loadAllBlueprintValidationChunks() (
	[][]*manage.BlueprintValidation,
	error,
) {
	// If there are no blueprint validations in the index,
	// there are no chunk files to load.
	if len(s.blueprintValidationIndex) == 0 {
		return nil, nil
	}

	validationChunks := [][]*manage.BlueprintValidation{}

	for i := 0; i <= s.lastBlueprintValidationChunk; i++ {
		chunkFilePath := blueprintValidationChunkFilePath(s.stateDir, i)
		existingData, err := afero.ReadFile(s.fs, chunkFilePath)
		if err != nil {
			return nil, err
		}
		chunkValidations := []*manage.BlueprintValidation{}
		err = json.Unmarshal(existingData, &chunkValidations)
		if err != nil {
			return nil, err
		}
		validationChunks = append(validationChunks, chunkValidations)
	}

	return validationChunks, nil
}

func (s *statePersister) updateBlueprintValidationChunkIndexEntries(
	chunkNumber int,
	validationChunk []*manage.BlueprintValidation,
) error {
	for i, validation := range validationChunk {
		s.blueprintValidationIndex[validation.ID] = &indexLocation{
			ChunkNumber:  chunkNumber,
			IndexInChunk: i,
		}
	}

	return s.persistBlueprintValidationIndexFile()
}

func (s *statePersister) persistBlueprintValidationIndexFile() error {
	indexData, err := json.Marshal(s.blueprintValidationIndex)
	if err != nil {
		return err
	}

	indexFilePath := blueprintValidationIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func blueprintValidationChunkFilePath(
	stateDir string,
	chunkIndex int,
) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("blueprint_validations_c%d.json", chunkIndex),
	)
}

func blueprintValidationIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "blueprint_validation_index.json")
}

func blueprintValidationsToEntities(
	validationChunk []*manage.BlueprintValidation,
) []manage.Entity {
	entities := make([]manage.Entity, len(validationChunk))

	for i, validation := range validationChunk {
		entities[i] = validation
	}

	return entities
}

func createBlueprintValidationLookup(
	blueprintValidations []*manage.BlueprintValidation,
) map[string]*manage.BlueprintValidation {
	lookup := map[string]*manage.BlueprintValidation{}

	for _, validation := range blueprintValidations {
		lookup[validation.ID] = validation
	}

	return lookup
}
