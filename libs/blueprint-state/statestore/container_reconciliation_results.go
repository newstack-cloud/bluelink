package statestore

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// ReconciliationResultsContainer implements manage.ReconciliationResults
// against a shared statestore.State and Persister. Maintains in-container
// secondary indexes (changesetID → []resultID, instanceID → []resultID)
// sorted by Created desc. Indexes are rebuilt on Cleanup and kept current
// on Save; they assume eager-mode population for correctness and become
// best-effort (only covering materialised entries) under ModeLazy.
type ReconciliationResultsContainer struct {
	state          *State
	persister      *Persister
	changesetIndex map[string][]string
	instanceIndex  map[string][]string
	logger         core.Logger
}

// NewReconciliationResultsContainer builds secondary indexes from whatever
// reconciliation results are currently in state (typically the full set
// under ModeEager).
func NewReconciliationResultsContainer(st *State, persister *Persister, logger core.Logger) *ReconciliationResultsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ReconciliationResultsContainer{
		state:          st,
		persister:      persister,
		changesetIndex: buildReconciliationChangesetIndex(st.reconciliations),
		instanceIndex:  buildReconciliationInstanceIndex(st.reconciliations),
		logger:         logger,
	}
}

func (c *ReconciliationResultsContainer) Get(
	ctx context.Context,
	id string,
) (*manage.ReconciliationResult, error) {
	r, ok, err := c.state.LookupReconciliation(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, manage.ReconciliationResultNotFoundError(id)
	}
	return copyReconciliationResult(r)
}

func (c *ReconciliationResultsContainer) GetLatestByChangesetID(
	ctx context.Context,
	changesetID string,
) (*manage.ReconciliationResult, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	resultIDs, ok := c.changesetIndex[changesetID]
	if !ok || len(resultIDs) == 0 {
		return nil, manage.ReconciliationResultNotFoundForChangesetError(changesetID)
	}
	return copyReconciliationResult(c.state.reconciliations[resultIDs[0]])
}

func (c *ReconciliationResultsContainer) GetAllByChangesetID(
	ctx context.Context,
	changesetID string,
) ([]*manage.ReconciliationResult, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	resultIDs, ok := c.changesetIndex[changesetID]
	if !ok {
		return []*manage.ReconciliationResult{}, nil
	}
	results := make([]*manage.ReconciliationResult, 0, len(resultIDs))
	for _, id := range resultIDs {
		copied, err := copyReconciliationResult(c.state.reconciliations[id])
		if err != nil {
			return nil, err
		}
		results = append(results, copied)
	}
	return results, nil
}

func (c *ReconciliationResultsContainer) GetLatestByInstanceID(
	ctx context.Context,
	instanceID string,
) (*manage.ReconciliationResult, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	resultIDs, ok := c.instanceIndex[instanceID]
	if !ok || len(resultIDs) == 0 {
		return nil, manage.ReconciliationResultNotFoundForInstanceError(instanceID)
	}
	return copyReconciliationResult(c.state.reconciliations[resultIDs[0]])
}

func (c *ReconciliationResultsContainer) GetAllByInstanceID(
	ctx context.Context,
	instanceID string,
) ([]*manage.ReconciliationResult, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	resultIDs, ok := c.instanceIndex[instanceID]
	if !ok {
		return []*manage.ReconciliationResult{}, nil
	}
	results := make([]*manage.ReconciliationResult, 0, len(resultIDs))
	for _, id := range resultIDs {
		copied, err := copyReconciliationResult(c.state.reconciliations[id])
		if err != nil {
			return nil, err
		}
		results = append(results, copied)
	}
	return results, nil
}

func (c *ReconciliationResultsContainer) Save(
	ctx context.Context,
	result *manage.ReconciliationResult,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	// Reconciliation results are immutable — idempotent create.
	if _, exists := c.state.reconciliations[result.ID]; exists {
		return nil
	}
	c.state.reconciliations[result.ID] = result
	c.addToChangesetIndex(result)
	c.addToInstanceIndex(result)

	c.logger.Debug(
		"persisting new reconciliation result",
		core.StringLogField("reconciliationResultId", result.ID),
	)
	return c.persister.CreateReconciliationResult(ctx, result)
}

func (c *ReconciliationResultsContainer) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.state.Lock()
	defer c.state.Unlock()

	originalCount := len(c.state.reconciliations)
	newLookup, err := c.persister.CleanupReconciliationResults(ctx, thresholdDate)
	if err != nil {
		return 0, err
	}
	c.state.reconciliations = newLookup
	c.changesetIndex = buildReconciliationChangesetIndex(newLookup)
	c.instanceIndex = buildReconciliationInstanceIndex(newLookup)
	return int64(originalCount - len(newLookup)), nil
}

// addToChangesetIndex must be called with the state
// write lock held (Save acquires it). It mutates the container's own
// secondary indexes, not state's.
func (c *ReconciliationResultsContainer) addToChangesetIndex(result *manage.ReconciliationResult) {
	resultIDs := append(c.changesetIndex[result.ChangesetID], result.ID)
	sort.Slice(resultIDs, func(i, j int) bool {
		return c.state.reconciliations[resultIDs[i]].Created > c.state.reconciliations[resultIDs[j]].Created
	})
	c.changesetIndex[result.ChangesetID] = resultIDs
}

// addToInstanceIndex must be called with the state
// write lock held (Save acquires it). It mutates the container's own
// secondary indexes, not state's.
func (c *ReconciliationResultsContainer) addToInstanceIndex(result *manage.ReconciliationResult) {
	resultIDs := append(c.instanceIndex[result.InstanceID], result.ID)
	sort.Slice(resultIDs, func(i, j int) bool {
		return c.state.reconciliations[resultIDs[i]].Created > c.state.reconciliations[resultIDs[j]].Created
	})
	c.instanceIndex[result.InstanceID] = resultIDs
}

func buildReconciliationChangesetIndex(results map[string]*manage.ReconciliationResult) map[string][]string {
	index := map[string][]string{}
	for id, result := range results {
		index[result.ChangesetID] = append(index[result.ChangesetID], id)
	}
	for changesetID, resultIDs := range index {
		sort.Slice(resultIDs, func(i, j int) bool {
			return results[resultIDs[i]].Created > results[resultIDs[j]].Created
		})
		index[changesetID] = resultIDs
	}
	return index
}

func buildReconciliationInstanceIndex(results map[string]*manage.ReconciliationResult) map[string][]string {
	index := map[string][]string{}
	for id, result := range results {
		index[result.InstanceID] = append(index[result.InstanceID], id)
	}
	for instanceID, resultIDs := range index {
		sort.Slice(resultIDs, func(i, j int) bool {
			return results[resultIDs[i]].Created > results[resultIDs[j]].Created
		})
		index[instanceID] = resultIDs
	}
	return index
}

func copyReconciliationResult(result *manage.ReconciliationResult) (*manage.ReconciliationResult, error) {
	checkResultCopy, err := copyReconciliationCheckResult(result.Result)
	if err != nil {
		return nil, err
	}
	return &manage.ReconciliationResult{
		ID:          result.ID,
		ChangesetID: result.ChangesetID,
		InstanceID:  result.InstanceID,
		Result:      checkResultCopy,
		Created:     result.Created,
	}, nil
}

func copyReconciliationCheckResult(result *container.ReconciliationCheckResult) (*container.ReconciliationCheckResult, error) {
	if result == nil {
		return nil, nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	dst := &container.ReconciliationCheckResult{}
	if err := json.Unmarshal(data, dst); err != nil {
		return nil, err
	}
	return dst, nil
}
