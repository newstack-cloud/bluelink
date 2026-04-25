package statestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// catSpec bundles the per-category state and key builders that the generic
// chunked-store helpers need. Grouping these keeps helper signatures under
// the 8-parameter limit and centralises the category wiring at call sites.
type catSpec struct {
	index        map[string]*IndexLocation
	lastChunkPtr *int
	chunkKey     func(int) string
	indexKey     string
}

func chunkSize(ctx context.Context, storage Storage, key string) (int64, error) {
	data, err := storage.Read(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return int64(len(data)), nil
}

// readChunk reads a JSON array chunk into the provided destination slice.
// A missing chunk is treated as an empty array (the persister creates chunks
// lazily on first write).
func readChunk(ctx context.Context, storage Storage, key string, dst any) error {
	data, err := storage.Read(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dst)
}

func writeChunk(ctx context.Context, storage Storage, key string, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return storage.Write(ctx, key, data)
}

// prepareChunk returns the key and chunk number to write a new entry into,
// rolling to a fresh chunk if the current one is at or above maxGuideFileSize.
// A newly-opened chunk is initialised to "[]" on storage so subsequent reads
// succeed deterministically.
func prepareChunk(
	ctx context.Context,
	storage Storage,
	currentChunk int,
	maxGuideFileSize int64,
	chunkKey func(int) string,
) (key string, chunkNumber int, err error) {
	currentKey := chunkKey(currentChunk)
	size, err := chunkSize(ctx, storage, currentKey)
	if err != nil {
		return "", 0, err
	}

	if size >= maxGuideFileSize {
		newChunk := currentChunk + 1
		newKey := chunkKey(newChunk)
		if err := storage.Write(ctx, newKey, []byte("[]")); err != nil {
			return "", 0, err
		}
		return newKey, newChunk, nil
	}
	return currentKey, currentChunk, nil
}

func maxChunkNumber(index map[string]*IndexLocation) int {
	max := 0
	for _, loc := range index {
		if loc.ChunkNumber > max {
			max = loc.ChunkNumber
		}
	}
	return max
}

func writeIndex(ctx context.Context, storage Storage, key string, index any) error {
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	return storage.Write(ctx, key, data)
}

func createInChunk[T any](
	ctx context.Context,
	storage Storage,
	maxGuideFileSize int64,
	spec catSpec,
	entity T,
	entityID string,
) error {
	key, chunkNumber, err := prepareChunk(ctx, storage, *spec.lastChunkPtr, maxGuideFileSize, spec.chunkKey)
	if err != nil {
		return err
	}

	var chunk []T
	if err := readChunk(ctx, storage, key, &chunk); err != nil {
		return err
	}

	chunk = append(chunk, entity)
	if err := writeChunk(ctx, storage, key, chunk); err != nil {
		return err
	}

	*spec.lastChunkPtr = chunkNumber
	spec.index[entityID] = &IndexLocation{
		ChunkNumber:  chunkNumber,
		IndexInChunk: len(chunk) - 1,
	}
	return writeIndex(ctx, storage, spec.indexKey, spec.index)
}

func updateInChunk[T any](
	ctx context.Context,
	storage Storage,
	spec catSpec,
	entity T,
	entityID string,
	notFoundErr error,
) error {
	entry, ok := spec.index[entityID]
	if !ok {
		return notFoundErr
	}
	key := spec.chunkKey(entry.ChunkNumber)

	var chunk []T
	if err := readChunk(ctx, storage, key, &chunk); err != nil {
		return err
	}

	if entry.IndexInChunk < 0 || entry.IndexInChunk >= len(chunk) {
		return errMalformedStateFile(fmt.Sprintf(
			"entity %q: position %d out of range for chunk of length %d",
			entityID, entry.IndexInChunk, len(chunk),
		))
	}

	chunk[entry.IndexInChunk] = entity
	return writeChunk(ctx, storage, key, chunk)
}

// loadAndFilterChunked reads every chunk in a LayoutChunked category in
// chunk order and returns entries for which keep returns true. Used by
// the bulk Cleanup flows.
func loadAndFilterChunked[T any](
	ctx context.Context,
	storage Storage,
	lastChunk int,
	chunkKey func(int) string,
	keep func(T) bool,
) ([]T, error) {
	var kept []T
	for i := 0; i <= lastChunk; i++ {
		var chunk []T
		if err := readChunk(ctx, storage, chunkKey(i), &chunk); err != nil {
			return nil, err
		}
		for _, item := range chunk {
			if keep(item) {
				kept = append(kept, item)
			}
		}
	}
	return kept, nil
}

// resetChunkedCategory removes every chunk file and the index file for a
// category so it can be rewritten from scratch. Missing files are ignored.
func resetChunkedCategory(
	ctx context.Context,
	storage Storage,
	lastChunk int,
	chunkKey func(int) string,
	indexKey string,
) error {
	for i := 0; i <= lastChunk; i++ {
		if err := deleteIgnoreNotFound(ctx, storage, chunkKey(i)); err != nil {
			return err
		}
	}
	return deleteIgnoreNotFound(ctx, storage, indexKey)
}

func removeFromChunk[T any](
	ctx context.Context,
	storage Storage,
	spec catSpec,
	entityID string,
) error {
	entry, ok := spec.index[entityID]
	if !ok {
		return nil
	}
	key := spec.chunkKey(entry.ChunkNumber)

	var chunk []T
	if err := readChunk(ctx, storage, key, &chunk); err != nil {
		return err
	}

	if entry.IndexInChunk < 0 || entry.IndexInChunk >= len(chunk) {
		return errMalformedStateFile(fmt.Sprintf(
			"entity %q: position %d out of range for chunk of length %d",
			entityID, entry.IndexInChunk, len(chunk),
		))
	}

	chunk = append(chunk[:entry.IndexInChunk], chunk[entry.IndexInChunk+1:]...)
	if err := writeChunk(ctx, storage, key, chunk); err != nil {
		return err
	}

	delete(spec.index, entityID)
	for id, loc := range spec.index {
		if loc.ChunkNumber == entry.ChunkNumber && loc.IndexInChunk > entry.IndexInChunk {
			spec.index[id] = &IndexLocation{
				ChunkNumber:  loc.ChunkNumber,
				IndexInChunk: loc.IndexInChunk - 1,
			}
		}
	}

	return writeIndex(ctx, storage, spec.indexKey, spec.index)
}
