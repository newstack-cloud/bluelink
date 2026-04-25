package enginev1

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	azurestore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/azureblob"
	gcsstore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/gcs"
	s3store "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/s3"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

const (
	memfileStorageEngine     = "memfile"
	postgresStorageEngine    = "postgres"
	objectstoreStorageEngine = "objectstore"

	objectstoreProviderS3        = "s3"
	objectstoreProviderGCS       = "gcs"
	objectstoreProviderAzureBlob = "azureblob"
)

type stateServices struct {
	container              state.Container
	events                 manage.Events
	validation             manage.Validation
	changesets             manage.Changesets
	reconciliationResults  manage.ReconciliationResults
	cleanupOperations      manage.CleanupOperations
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

	if stateConfig.StorageEngine == objectstoreStorageEngine {
		return loadObjectstoreStateServices(
			ctx,
			stateConfig,
			logger,
		)
	}

	return nil, nil, fmt.Errorf(
		"unsupported %q storage engine provided, "+
			"only the \"memfile\", \"postgres\" and \"objectstore\""+
			" engines are supported for this version of the deploy engine",
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
	reconciliationResults := stateContainer.ReconciliationResults()
	cleanupOperations := stateContainer.CleanupOperations()

	return &stateServices{
		container:             stateContainer,
		validation:            validation,
		events:                events,
		changesets:            changesets,
		reconciliationResults: reconciliationResults,
		cleanupOperations:     cleanupOperations,
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
	reconciliationResults := stateContainer.ReconciliationResults()
	cleanupOperations := stateContainer.CleanupOperations()

	return &stateServices{
		container:             stateContainer,
		validation:            validation,
		events:                events,
		changesets:            changesets,
		reconciliationResults: reconciliationResults,
		cleanupOperations:     cleanupOperations,
	}, closePool, nil
}

func createPostgresConnPool(
	ctx context.Context,
	stateConfig *core.StateConfig,
) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, buildPostgresDatabaseURL(stateConfig))
}

func loadObjectstoreStateServices(
	ctx context.Context,
	stateConfig *core.StateConfig,
	logger bpcore.Logger,
) (*stateServices, func(), error) {
	svc, err := buildObjectstoreService(ctx, &stateConfig.ObjectStore)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to create objectstore service: %w",
			err,
		)
	}

	stateContainer, err := objectstore.LoadStateContainer(
		ctx,
		svc,
		stateConfig.ObjectStore.Prefix,
		logger,
		objectstore.WithRecentlyQueuedEventsThreshold(
			stateConfig.RecentlyQueuedEventsThreshold,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to create objectstore state container: %w",
			err,
		)
	}

	return &stateServices{
		container:             stateContainer,
		validation:            stateContainer.Validation(),
		events:                stateContainer.Events(),
		changesets:            stateContainer.Changesets(),
		reconciliationResults: stateContainer.ReconciliationResults(),
		cleanupOperations:     stateContainer.CleanupOperations(),
	}, objectstoreStubClose, nil
}

func objectstoreStubClose() {
	// No-op close function for objectstore state services — the
	// underlying provider SDK clients do not hold long-lived resources
	// that require explicit cleanup during normal engine shutdown.
}

func buildObjectstoreService(
	ctx context.Context,
	cfg *core.ObjectStoreConfig,
) (objectstore.Service, error) {
	switch cfg.Provider {
	case objectstoreProviderS3:
		return buildS3ObjectstoreService(ctx, &cfg.S3)
	case objectstoreProviderGCS:
		return buildGCSObjectstoreService(ctx, &cfg.GCS)
	case objectstoreProviderAzureBlob:
		return buildAzureObjectstoreService(&cfg.AzureBlob)
	default:
		return nil, fmt.Errorf(
			"unsupported objectstore provider %q, "+
				"supported providers are: \"s3\", \"gcs\", \"azureblob\"",
			cfg.Provider,
		)
	}
}

func buildS3ObjectstoreService(
	ctx context.Context,
	cfg *core.ObjectStoreS3Config,
) (objectstore.Service, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objectstore s3 bucket is required")
	}

	awsOpts := []func(*awsconfig.LoadOptions) error{}
	if cfg.Region != "" {
		awsOpts = append(awsOpts, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsOpts = append(awsOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	conf, err := awsconfig.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3store.NewClient(conf, s3store.ClientOptions{
		Endpoint:     cfg.Endpoint,
		UsePathStyle: cfg.UsePathStyle,
	})
	return s3store.NewService(client, cfg.Bucket), nil
}

func buildGCSObjectstoreService(
	ctx context.Context,
	cfg *core.ObjectStoreGCSConfig,
) (objectstore.Service, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objectstore gcs bucket is required")
	}

	client, err := gcsstore.NewClient(ctx, gcsstore.ClientOptions{
		Endpoint:              cfg.Endpoint,
		WithoutAuthentication: cfg.WithoutAuthentication,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gcs client: %w", err)
	}
	return gcsstore.NewService(client, cfg.Bucket), nil
}

func buildAzureObjectstoreService(
	cfg *core.ObjectStoreAzureBlobConfig,
) (objectstore.Service, error) {
	if cfg.Container == "" {
		return nil, fmt.Errorf("objectstore azureblob container is required")
	}
	if cfg.ServiceURL == "" || cfg.AccountName == "" || cfg.AccountKey == "" {
		return nil, fmt.Errorf(
			"objectstore azureblob service_url, account_name and account_key are required",
		)
	}

	client, err := azurestore.NewClient(azurestore.ClientOptions{
		ServiceURL:  cfg.ServiceURL,
		AccountName: cfg.AccountName,
		AccountKey:  cfg.AccountKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create azure blob client: %w", err)
	}
	return azurestore.NewService(client, cfg.Container), nil
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
