package objectstore

import (
	"context"
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
	auditContainer                 *auditContainer
	storePersister                 *statestore.Persister
	clock                          commoncore.Clock
	revalidateCache                bool
	cacheMaxAge                    time.Duration
}

// DefaultCacheMaxAge is how long a cached instance is used before it is checked against the
// bucket again, unless WithCacheRevalidation or WithoutCacheRevalidation says otherwise.
//
// Typically, engine instances that share the same object storage backend will be running in CI
// environments that get fresh state on every run, for the more unusual case where long-lived
// engine instances share an object storage backend, deployments for the same instances
// do not run in parallel, so subsequent runs are usually minutes apart and 30 seconds is enough
// to reduce subsequent reads throughout a single deployment process.
const DefaultCacheMaxAge = 30 * time.Second

// Option is a type for options that can be passed to LoadStateContainer.
type Option func(*StateContainer)

// WithCacheRevalidation sets how long a cached instance is used before it is checked against
// the bucket again (a HEAD request), so that a long-running engine sees changes made by other
// engines sharing the bucket. A changed instance is reloaded and a removed one dropped. Zero
// checks on every use. Defaults to DefaultCacheMaxAge.
//
// Writes do not depend on it as every instance write is conditional on the version last read, so
// a write based on a stale instance is rejected and applied again to the current one.
func WithCacheRevalidation(maxAge time.Duration) Option {
	return func(c *StateContainer) {
		c.revalidateCache = true
		c.cacheMaxAge = maxAge
	}
}

// WithoutCacheRevalidation turns off checking cached instances against the bucket, for an
// engine that has the bucket to itself or is short-lived, such as one started for a CI run.
// It loads instances fresh, its own writes keep its cache current, and it makes no HEAD
// requests to revalidate.
func WithoutCacheRevalidation() Option {
	return func(c *StateContainer) {
		c.revalidateCache = false
	}
}

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

// WithClock sets the clock used by the events container, and to timestamp instance name
// reservations and judge whether an old one was abandoned.
func WithClock(clock commoncore.Clock) Option {
	return func(c *StateContainer) {
		c.clock = clock
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

	// Options are applied after the containers are built,
	// so the clock is read when used.
	var container *StateContainer
	getClock := func() commoncore.Clock {
		return container.clock
	}

	storePersister := statestore.NewPersister(
		storeState,
		storage,
		cfg,
		prefix,
		statestore.WithLogger(logger),
		statestore.WithNameRecordReserver(newNameRecordReserver(svc, keys, getClock)),
	)

	container = &StateContainer{
		clock:           &commoncore.SystemClock{},
		revalidateCache: true,
		cacheMaxAge:     DefaultCacheMaxAge,
		storePersister:  storePersister,
		instancesContainer: statestore.NewInstancesContainer(
			storeState,
			storePersister,
			objectstoreClaimFunc(svc, keys, storeState, getClock),
			objectstoreInitialiseAndClaimFunc(svc, keys, storeState, getClock),
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
		auditContainer: &auditContainer{
			AuditContainer: statestore.NewAuditContainer(storeState, storePersister, logger),
			svc:            svc,
			keys:           keys,
		},
	}

	for _, opt := range opts {
		opt(container)
	}
	if container.revalidateCache {
		statestore.WithRevalidation(container.cacheMaxAge, clockFunc(getClock))(storeState)
	}
	return container, nil
}

// Reads the container's clock when the time is asked for, so WithClock applies whatever
// order the options are given in.
type clockFunc func() commoncore.Clock

func (c clockFunc) Now() time.Time {
	return c().Now()
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

func (c *StateContainer) Audit() manage.Audit {
	return c.auditContainer
}

// Every category sits on LayoutPerEntity so that each record has its own object key.
// Cross-run writes to unrelated entities are then collision-free and the lazy
// EntityLoader reads any single record with one GET. Resources and links are persisted
// inside their instance's object, so Resources and Links inherit Instances' layout, and
// are found by ID through a resources_by_id or links_by_id record naming the instance.
//
// Events are per-entity for a stronger reason than the rest. A chunked channel is one
// object rewritten whole on every save, and an engine that has not read the channel
// rewrites it with only its own event, so two engines sharing a bucket destroy each
// other's events. Their keys carry the channel and a zero-padded timestamp, so a channel
// is read, resumed and cleaned up from a listing alone.
func getConfig() statestore.Config {
	perEntityGlobal := statestore.CategoryConfig{
		Layout: statestore.LayoutPerEntity,
		Scope:  statestore.ScopeGlobal,
	}
	perEntityPerChannel := statestore.CategoryConfig{
		Layout: statestore.LayoutPerEntity,
		Scope:  statestore.ScopePerChannel,
	}
	return statestore.Config{
		Mode:                  statestore.ModeLazy,
		WriteNameRecords:      true,
		WriteEntityRecords:    true,
		Instances:             perEntityGlobal,
		Resources:             perEntityGlobal,
		Links:                 perEntityGlobal,
		ResourceDrift:         perEntityGlobal,
		LinkDrift:             perEntityGlobal,
		Events:                perEntityPerChannel,
		Changesets:            perEntityGlobal,
		Validations:           perEntityGlobal,
		ReconciliationResults: perEntityGlobal,
		CleanupOperations:     perEntityGlobal,
		AuditRecords:          perEntityGlobal,
	}
}

// Returns a NameRecordReserver backed by reserveName, so persister.CreateInstance
// surfaces state.ErrInstanceAlreadyExists for a name another live instance holds,
// the same sentinel the in-process backends return.
func newNameRecordReserver(
	svc Service,
	keys statestore.KeyBuilder,
	clock func() commoncore.Clock,
) statestore.NameRecordReserver {
	return func(ctx context.Context, name, instanceID string) error {
		// Runs before every create of a named instance, so it is also where an interrupted
		// removal of the same ID is finished before the new instance is written.
		if err := finishRemoval(ctx, svc, keys, instanceID); err != nil {
			return err
		}
		_, err := reserveName(ctx, svc, keys, clock(), name, instanceID)
		return err
	}
}

func objectstoreClaimFunc(
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
	clock func() commoncore.Clock,
) statestore.ClaimFunc {
	return func(
		ctx context.Context,
		instanceID string,
		expectedVersion int64,
		newStatus core.InstanceStatus,
	) (int64, error) {
		return ClaimForDeployment(ctx, svc, keys, st, clock(), instanceID, expectedVersion, newStatus)
	}
}

func objectstoreInitialiseAndClaimFunc(
	svc Service,
	keys statestore.KeyBuilder,
	st *statestore.State,
	clock func() commoncore.Clock,
) statestore.InitialiseAndClaimFunc {
	return func(
		ctx context.Context,
		instanceState state.InstanceState,
		newStatus core.InstanceStatus,
	) (int64, error) {
		return InitialiseAndClaim(ctx, svc, keys, st, clock(), instanceState, newStatus)
	}
}
