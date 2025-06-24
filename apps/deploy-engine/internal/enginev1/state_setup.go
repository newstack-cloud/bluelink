package enginev1

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

const (
	memfileStorageEngine  = "memfile"
	postgresStorageEngine = "postgres"
)

type stateServices struct {
	container  state.Container
	events     manage.Events
	validation manage.Validation
	changesets manage.Changesets
}

func loadStateServices(
	ctx context.Context,
	fileSystem afero.Fs,
	logger bpcore.Logger,
	stateConfig *core.StateConfig,
) (*stateServices, func(), error) {
	if stateConfig.StorageEngine == memfileStorageEngine {
		return loadMemfileStateServices(
			stateConfig,
			fileSystem,
			logger,
		)
	}

	if stateConfig.StorageEngine == postgresStorageEngine {
		return loadPostgresStateServices(
			ctx,
			stateConfig,
			logger,
		)
	}

	return nil, nil, fmt.Errorf(
		"unsupported %q storage engine provided, "+
			"only the \"memfile\" and \"postgres\" engines are supported"+
			" for this version of the deploy engine",
		stateConfig.StorageEngine,
	)
}

func loadMemfileStateServices(
	stateConfig *core.StateConfig,
	fileSystem afero.Fs,
	logger bpcore.Logger,
) (*stateServices, func(), error) {
	err := prepareMemfileStateDir(
		stateConfig.MemFileStateDir,
		fileSystem,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to prepare memfile state directory: %w",
			err,
		)
	}

	stateContainer, err := memfile.LoadStateContainer(
		stateConfig.MemFileStateDir,
		fileSystem,
		logger,
		memfile.WithMaxGuideFileSize(
			stateConfig.MemFileMaxGuideFileSize,
		),
		memfile.WithMaxEventPartitionSize(
			stateConfig.MemFileMaxEventPartitionSize,
		),
		memfile.WithRecentlyQueuedEventsThreshold(
			stateConfig.RecentlyQueuedEventsThreshold,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to create memfile state container: %w",
			err,
		)
	}

	events := stateContainer.Events()
	validation := stateContainer.Validation()
	changesets := stateContainer.Changesets()

	return &stateServices{
		container:  stateContainer,
		validation: validation,
		events:     events,
		changesets: changesets,
	}, memfileStubClose, nil
}

func memfileStubClose() {
	// No-op close function for memfile state services
	// as it does not require any special cleanup.
}

func prepareMemfileStateDir(
	stateDirPath string,
	fileSystem afero.Fs,
) error {
	return fileSystem.MkdirAll(
		stateDirPath,
		0755,
	)
}

func loadPostgresStateServices(
	ctx context.Context,
	stateConfig *core.StateConfig,
	logger bpcore.Logger,
) (*stateServices, func(), error) {
	pool, err := createPostgresConnPool(ctx, stateConfig)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to create postgres connection pool: %w",
			err,
		)
	}

	closePool := func() {
		pool.Close()
	}

	stateContainer, err := postgres.LoadStateContainer(
		ctx,
		pool,
		logger,
		postgres.WithRecentlyQueuedEventsThreshold(
			stateConfig.RecentlyQueuedEventsThreshold,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to create postgres state container: %w",
			err,
		)
	}

	events := stateContainer.Events()
	validation := stateContainer.Validation()
	changesets := stateContainer.Changesets()

	return &stateServices{
		container:  stateContainer,
		validation: validation,
		events:     events,
		changesets: changesets,
	}, closePool, nil
}

func createPostgresConnPool(
	ctx context.Context,
	stateConfig *core.StateConfig,
) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, buildPostgresDatabaseURL(stateConfig))
}

func buildPostgresDatabaseURL(stateConfig *core.StateConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d&pool_max_conn_lifetime=%s",
		stateConfig.PostgresUser,
		stateConfig.PostgresPassword,
		stateConfig.PostgresHost,
		stateConfig.PostgresPort,
		stateConfig.PostgresDatabase,
		stateConfig.PostgresSSLMode,
		stateConfig.PostgresPoolMaxConns,
		stateConfig.PostgresPoolMaxConnLifetime,
	)
}
