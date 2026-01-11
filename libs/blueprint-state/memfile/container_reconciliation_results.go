package memfile

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

type reconciliationResultsContainerImpl struct {
	reconciliationResults map[string]*manage.ReconciliationResult
	// Secondary index: changesetID -> list of result IDs (sorted by created desc)
	changesetIndex map[string][]string
	// Secondary index: instanceID -> list of result IDs (sorted by created desc)
	instanceIndex map[string][]string
	fs            afero.Fs
	persister     *statePersister
	logger        core.Logger
	mu            *sync.RWMutex
}

func (c *reconciliationResultsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.ReconciliationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, ok := c.reconciliationResults[id]
	if !ok {
		return nil, manage.ReconciliationResultNotFoundError(id)
	}

	resultCopy, err := copyReconciliationResult(result)
	if err != nil {
		return nil, err
	}

	return resultCopy, nil
}

func (c *reconciliationResultsContainerImpl) GetLatestByChangesetID(
	ctx context.Context,
	changesetID string,
) (*manage.ReconciliationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resultIDs, ok := c.changesetIndex[changesetID]
	if !ok || len(resultIDs) == 0 {
		return nil, manage.ReconciliationResultNotFoundForChangesetError(changesetID)
	}

	// Index is sorted by created desc, so first entry is the latest
	result := c.reconciliationResults[resultIDs[0]]
	resultCopy, err := copyReconciliationResult(result)
	if err != nil {
		return nil, err
	}

	return resultCopy, nil
}

func (c *reconciliationResultsContainerImpl) GetAllByChangesetID(
	ctx context.Context,
	changesetID string,
) ([]*manage.ReconciliationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resultIDs, ok := c.changesetIndex[changesetID]
	if !ok {
		return []*manage.ReconciliationResult{}, nil
	}

	results := make([]*manage.ReconciliationResult, 0, len(resultIDs))
	for _, id := range resultIDs {
		result := c.reconciliationResults[id]
		resultCopy, err := copyReconciliationResult(result)
		if err != nil {
			return nil, err
		}
		results = append(results, resultCopy)
	}

	return results, nil
}

func (c *reconciliationResultsContainerImpl) GetLatestByInstanceID(
	ctx context.Context,
	instanceID string,
) (*manage.ReconciliationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resultIDs, ok := c.instanceIndex[instanceID]
	if !ok || len(resultIDs) == 0 {
		return nil, manage.ReconciliationResultNotFoundForInstanceError(instanceID)
	}

	// Index is sorted by created desc, so first entry is the latest
	result := c.reconciliationResults[resultIDs[0]]
	resultCopy, err := copyReconciliationResult(result)
	if err != nil {
		return nil, err
	}

	return resultCopy, nil
}

func (c *reconciliationResultsContainerImpl) GetAllByInstanceID(
	ctx context.Context,
	instanceID string,
) ([]*manage.ReconciliationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resultIDs, ok := c.instanceIndex[instanceID]
	if !ok {
		return []*manage.ReconciliationResult{}, nil
	}

	results := make([]*manage.ReconciliationResult, 0, len(resultIDs))
	for _, id := range resultIDs {
		result := c.reconciliationResults[id]
		resultCopy, err := copyReconciliationResult(result)
		if err != nil {
			return nil, err
		}
		results = append(results, resultCopy)
	}

	return results, nil
}

func (c *reconciliationResultsContainerImpl) Save(
	ctx context.Context,
	result *manage.ReconciliationResult,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.save(result)
}

func (c *reconciliationResultsContainerImpl) save(
	result *manage.ReconciliationResult,
) error {
	resultLogger := c.logger.WithFields(
		core.StringLogField("reconciliationResultId", result.ID),
	)

	// Reconciliation results are immutable - no updates allowed
	_, alreadyExists := c.reconciliationResults[result.ID]
	if alreadyExists {
		// If it already exists, just return (idempotent)
		return nil
	}

	c.reconciliationResults[result.ID] = result

	// Update secondary indexes
	c.addToChangesetIndex(result)
	c.addToInstanceIndex(result)

	resultLogger.Debug("persisting new reconciliation result")
	return c.persister.createReconciliationResult(result)
}

func (c *reconciliationResultsContainerImpl) addToChangesetIndex(result *manage.ReconciliationResult) {
	resultIDs := c.changesetIndex[result.ChangesetID]
	resultIDs = append(resultIDs, result.ID)
	// Sort by created desc
	sort.Slice(resultIDs, func(i, j int) bool {
		return c.reconciliationResults[resultIDs[i]].Created > c.reconciliationResults[resultIDs[j]].Created
	})
	c.changesetIndex[result.ChangesetID] = resultIDs
}

func (c *reconciliationResultsContainerImpl) addToInstanceIndex(result *manage.ReconciliationResult) {
	resultIDs := c.instanceIndex[result.InstanceID]
	resultIDs = append(resultIDs, result.ID)
	// Sort by created desc
	sort.Slice(resultIDs, func(i, j int) bool {
		return c.reconciliationResults[resultIDs[i]].Created > c.reconciliationResults[resultIDs[j]].Created
	})
	c.instanceIndex[result.InstanceID] = resultIDs
}

func (c *reconciliationResultsContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	originalCount := len(c.reconciliationResults)

	newLookup, err := c.persister.cleanupReconciliationResults(thresholdDate)
	if err != nil {
		return 0, err
	}
	c.reconciliationResults = newLookup

	// Rebuild secondary indexes
	c.changesetIndex = buildChangesetIndex(newLookup)
	c.instanceIndex = buildInstanceIndex(newLookup)

	return int64(originalCount - len(newLookup)), nil
}

func buildChangesetIndex(results map[string]*manage.ReconciliationResult) map[string][]string {
	index := map[string][]string{}
	for id, result := range results {
		index[result.ChangesetID] = append(index[result.ChangesetID], id)
	}
	// Sort each list by created desc
	for changesetID, resultIDs := range index {
		sort.Slice(resultIDs, func(i, j int) bool {
			return results[resultIDs[i]].Created > results[resultIDs[j]].Created
		})
		index[changesetID] = resultIDs
	}
	return index
}

func buildInstanceIndex(results map[string]*manage.ReconciliationResult) map[string][]string {
	index := map[string][]string{}
	for id, result := range results {
		index[result.InstanceID] = append(index[result.InstanceID], id)
	}
	// Sort each list by created desc
	for instanceID, resultIDs := range index {
		sort.Slice(resultIDs, func(i, j int) bool {
			return results[resultIDs[i]].Created > results[resultIDs[j]].Created
		})
		index[instanceID] = resultIDs
	}
	return index
}

func copyReconciliationResult(
	result *manage.ReconciliationResult,
) (*manage.ReconciliationResult, error) {
	resultCopy, err := copyReconciliationCheckResult(result.Result)
	if err != nil {
		return nil, err
	}

	return &manage.ReconciliationResult{
		ID:          result.ID,
		ChangesetID: result.ChangesetID,
		InstanceID:  result.InstanceID,
		Result:      resultCopy,
		Created:     result.Created,
	}, nil
}

func copyReconciliationCheckResult(
	result *container.ReconciliationCheckResult,
) (*container.ReconciliationCheckResult, error) {
	if result == nil {
		return nil, nil
	}

	// Use JSON marshal/unmarshal to deep copy
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	resultCopy := &container.ReconciliationCheckResult{}
	err = json.Unmarshal(resultBytes, resultCopy)
	if err != nil {
		return nil, err
	}

	return resultCopy, nil
}
