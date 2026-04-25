package objectstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// StateContainer is the object storage backed implementation of the blueprint
// state.Container interface. It composes the shared statestore engine with a
// Service-backed Storage adapter; ETag-aware concurrency primitives
// (ClaimForDeployment, InitialiseAndClaim) go direct to the Service so
// IfMatch / IfNoneMatch conditional writes serialise concurrent deploys
// sharing a bucket.
type StateContainer struct {
	instancesContainer             *statestore.InstancesContainer
	resourcesContainer             *statestore.ResourcesContainer
	linksContainer                 *statestore.LinksContainer
	childrenContainer              *statestore.ChildrenContainer
	metadataContainer              *statestore.MetadataContainer
	exportContainer                *statestore.ExportsContainer
	eventsContainer                *statestore.EventsContainer
	changesetsContainer            *statestore.ChangesetsContainer
	validationContainer            *statestore.ValidationsContainer
	reconciliationResultsContainer *statestore.ReconciliationResultsContainer
	cleanupOperationsContainer     *statestore.CleanupOperationsContainer
	storePersister                 *statestore.Persister
}

// Option is a type for options that can be passed to LoadStateContainer.
type Option func(*StateContainer)

// WithMaxGuideFileSize sets a guide for the maximum size of a state chunk
// file in bytes. See statestore.WithMaxGuideFileSize.
func WithMaxGuideFileSize(maxGuideFileSize int64) Option {
	return func(c *StateContainer) {
		statestore.WithMaxGuideFileSize(maxGuideFileSize)(c.storePersister)
	}
}

// WithMaxEventPartitionSize sets the maximum size of an event partition file
// in bytes. See statestore.WithMaxEventPartitionSize.
func WithMaxEventPartitionSize(maxEventPartitionSize int64) Option {
	return func(c *StateContainer) {
		statestore.WithMaxEventPartitionSize(maxEventPartitionSize)(c.storePersister)
	}
}

// WithRecentlyQueuedEventsThreshold sets the threshold in seconds for
// retrieving recently queued events for a stream when a starting event ID
// is not provided.
func WithRecentlyQueuedEventsThreshold(thresholdSeconds int64) Option {
	return func(c *StateContainer) {
		d := time.Duration(thresholdSeconds) * time.Second
		statestore.WithEventsRecentlyQueuedThreshold(d)(c.eventsContainer)
	}
}

// WithClock sets the clock used by the events container.
func WithClock(clock commoncore.Clock) Option {
	return func(c *StateContainer) {
		statestore.WithEventsClock(clock)(c.eventsContainer)
	}
}

// LoadStateContainer constructs an object-storage-backed state container.
// svc provides the low-level object operations; prefix is prepended to every
// storage key (e.g. "bluelink-state/"). State runs under ModeLazy — entries
// materialise on demand from the Service so each process only touches the
// working subset of the shared bucket rather than eagerly walking it.
func LoadStateContainer(
	_ context.Context,
	svc Service,
	prefix string,
	logger core.Logger,
	opts ...Option,
) (*StateContainer, error) {
	storage := NewServiceStorage(svc)
	keys := statestore.NewKeyBuilder(prefix)
	storeState := statestore.NewState(
		statestore.WithEntityLoader(NewServiceLoader(svc, keys)),
	)
	cfg := getConfig()

	storePersister := statestore.NewPersister(
		storeState,
		storage,
		cfg,
		prefix,
		statestore.WithLogger(logger),
		statestore.WithNameRecordReserver(newNameRecordReserver(svc, keys)),
	)

	container := &StateContainer{
		storePersister: storePersister,
		instancesContainer: statestore.NewInstancesContainer(
			storeState,
			storePersister,
			objectstoreClaimFunc(svc, keys, storeState),
			objectstoreInitialiseAndClaimFunc(svc, keys, storeState),
			logger,
		),
		resourcesContainer:             statestore.NewResourcesContainer(storeState, storePersister, logger),
		linksContainer:                 statestore.NewLinksContainer(storeState, storePersister, logger),
		childrenContainer:              statestore.NewChildrenContainer(storeState, storePersister, logger),
		metadataContainer:              statestore.NewMetadataContainer(storeState, storePersister, logger),
		exportContainer:                statestore.NewExportsContainer(storeState, storePersister, logger),
		eventsContainer:                statestore.NewEventsContainer(storeState, storePersister, logger),
		changesetsContainer:            statestore.NewChangesetsContainer(storeState, storePersister, logger),
		validationContainer:            statestore.NewValidationsContainer(storeState, storePersister, logger),
		reconciliationResultsContainer: statestore.NewReconciliationResultsContainer(storeState, storePersister, logger),
		cleanupOperationsContainer:     statestore.NewCleanupOperationsContainer(storeState, storePersister, logger),
	}

	for _, opt := range opts {
		opt(container)
	}
	return container, nil
}

func (c *StateContainer) Instances() state.InstancesContainer { return c.instancesContainer }
func (c *StateContainer) Resources() state.ResourcesContainer { return c.resourcesContainer }
func (c *StateContainer) Links() state.LinksContainer         { return c.linksContainer }
func (c *StateContainer) Children() state.ChildrenContainer   { return c.childrenContainer }
func (c *StateContainer) Metadata() state.MetadataContainer   { return c.metadataContainer }
func (c *StateContainer) Exports() state.ExportsContainer     { return c.exportContainer }

func (c *StateContainer) Events() manage.Events         { return c.eventsContainer }
func (c *StateContainer) Changesets() manage.Changesets { return c.changesetsContainer }
func (c *StateContainer) Validation() manage.Validation { return c.validationContainer }
func (c *StateContainer) ReconciliationResults() manage.ReconciliationResults {
	return c.reconciliationResultsContainer
}
func (c *StateContainer) CleanupOperations() manage.CleanupOperations {
	return c.cleanupOperationsContainer
}

// Instances, drift, changesets, validations, reconciliation results and
// cleanup operations all sit on LayoutPerEntity so every record has its
// own object key — cross-run writes to unrelated entities are then
// collision-free and the lazy EntityLoader resolves any single record
// with one GET. Events stay chunked-per-channel because their partition
// key already provides isolation and their on-wire shape is a partition
// file. Resources and Links inherit Instances' layout — they are
// persisted inline with the parent instance record.
func getConfig() statestore.Config {
	perEntityGlobal := statestore.CategoryConfig{
		Layout: statestore.LayoutPerEntity,
		Scope:  statestore.ScopeGlobal,
	}
	chunkedPerChannel := statestore.CategoryConfig{
		Layout: statestore.LayoutChunked,
		Scope:  statestore.ScopePerChannel,
	}
	return statestore.Config{
		Mode:                  statestore.ModeLazy,
		WriteNameRecords:      true,
		Instances:             perEntityGlobal,
		Resources:             perEntityGlobal,
		Links:                 perEntityGlobal,
		ResourceDrift:         perEntityGlobal,
		LinkDrift:             perEntityGlobal,
		Events:                chunkedPerChannel,
		Changesets:            perEntityGlobal,
		Validations:           perEntityGlobal,
		ReconciliationResults: perEntityGlobal,
		CleanupOperations:     perEntityGlobal,
	}
}

// Returns a NameRecordReserver that writes the
// instance name-lookup record with IfNoneMatch: "*" so two concurrent
// first-deploys choosing the same name for different IDs are serialised
// at the bucket layer. Maps 412 to state.ErrInstanceAlreadyExists so
// persister.CreateInstance surfaces the same sentinel the in-process
// backends already return.
func newNameRecordReserver(svc Service, keys statestore.KeyBuilder) statestore.NameRecordReserver {
	return func(ctx context.Context, name, instanceID string) error {
		record := &statestore.InstanceNameRecord{ID: instanceID, Name: name}
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		_, err = svc.Put(ctx, keys.InstanceByName(name), data, &PutOptions{IfNoneMatch: "*"})
		if err != nil {
			if isPreconditionFailed(err) {
				return state.ErrInstanceAlreadyExists
			}
			return err
		}
		return nil
	}
}

func objectstoreClaimFunc(
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
) statestore.ClaimFunc {
	return func(
		ctx context.Context,
		instanceID string,
		expectedVersion int64,
		newStatus core.InstanceStatus,
	) (int64, error) {
		return ClaimForDeployment(ctx, svc, keys, st, instanceID, expectedVersion, newStatus)
	}
}

func objectstoreInitialiseAndClaimFunc(
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
) statestore.InitialiseAndClaimFunc {
	return func(
		ctx context.Context,
		instanceState state.InstanceState,
		newStatus core.InstanceStatus,
	) (int64, error) {
		return InitialiseAndClaim(ctx, svc, keys, st, instanceState, newStatus)
	}
}
