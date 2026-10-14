package enginev1

import (
	"context"
	"fmt"
	"runtime"
	"strings"

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
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

// State holds resolved secrets whether or not encryption is enabled, so the directory it lives
// in is private to its owner.
const stateDirMode = 0o700

const (
	postgresSSLModeDisable = "disable"
	postgresSSLModePrefer  = "prefer"
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
	container             state.Container
	events                manage.Events
	validation            manage.Validation
	changesets            manage.Changesets
	reconciliationResults manage.ReconciliationResults
	cleanupOperations     manage.CleanupOperations
}

func loadStateServices(
	ctx context.Context,
	fileSystem afero.Fs,
	logger bpcore.Logger,
	stateConfig *core.StateConfig,
) (*stateServices, func(), error) {
	if stateConfig.StorageEngine == memfileStorageEngine {
		return loadMemfileStateServices(
			ctx,
			stateConfig,
			fileSystem,
			logger,
		)
	}

	if stateConfig.Encryption.Enabled {
		return nil, nil, fmt.Errorf(
			"state encryption is only supported for the %q storage engine, "+
				"the %q engine relies on encryption at rest in the database or bucket "+
				"it persists to",
			memfileStorageEngine,
			stateConfig.StorageEngine,
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
	ctx context.Context,
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

	storage, err := memfileStorage(ctx, stateConfig, fileSystem, logger)
	if err != nil {
		return nil, nil, err
	}

	stateContainer, err := memfile.LoadStateContainerWithStorage(
		stateConfig.MemFileStateDir,
		storage,
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

// Returns the storage the memfile container persists through, encrypted when
// the deployment has configured a key.
//
// The salt is read through the undecorated storage, since it is an input to
// deriving the key that would decrypt it, and before the container is loaded,
// since loading reads the whole store.
func memfileStorage(
	ctx context.Context,
	stateConfig *core.StateConfig,
	fileSystem afero.Fs,
	logger bpcore.Logger,
) (statestore.Storage, error) {
	storage := memfile.NewFSStorage(fileSystem)

	if !stateConfig.Encryption.Enabled {
		logger.Warn(
			"state is being persisted unencrypted, resolved secrets in resource specs, " +
				"changesets and events will be readable on disk and in any backup taken " +
				"from it; set state.encryption.enabled to encrypt them, or ensure the " +
				"host's volume and backups are encrypted",
		)

		return storage, nil
	}

	salt, err := statestore.LoadOrCreateSalt(
		ctx,
		storage,
		statestore.NewKeyBuilder(stateConfig.MemFileStateDir),
	)
	if err != nil {
		return nil, err
	}

	keyProvider, err := createStateKeyProvider(ctx, &stateConfig.Encryption, salt)
	if err != nil {
		return nil, err
	}

	logger.Info(
		"state encryption is enabled",
		bpcore.StringLogField("keyProvider", stateConfig.Encryption.Provider),
		bpcore.StringLogField("keyID", keyProvider.ID()),
	)

	return statestore.NewEncryptedStorage(storage, keyProvider), nil
}

func memfileStubClose() {
	// No-op close function for memfile state services
	// as it does not require any special cleanup.
}

// Creates the state directory private to the user the engine runs as.
func prepareMemfileStateDir(
	stateDirPath string,
	fileSystem afero.Fs,
) error {
	if err := fileSystem.MkdirAll(stateDirPath, stateDirMode); err != nil {
		return err
	}

	// Mode bits are not the access control on Windows, where the directory's ACL governs
	// access, so failing to apply them there is not a reason to refuse to start.
	if err := fileSystem.Chmod(stateDirPath, stateDirMode); err != nil &&
		runtime.GOOS != "windows" {
		return err
	}

	return nil
}

func loadPostgresStateServices(
	ctx context.Context,
	stateConfig *core.StateConfig,
	logger bpcore.Logger,
) (*stateServices, func(), error) {
	warnAboutUnprotectedPostgresConnection(stateConfig, logger)

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

// Warns when state and the credentials to reach it cross a network without TLS.
//
// Only raised for a host that is not the local machine as an unencrypted connection to a local
// database is how local development and CI normally run, and warning about it on every start
// would be noise rather than a signal.
//
// "prefer" is included because it is not a guarantee. It falls back to an unencrypted
// connection when the server will not negotiate TLS, and does so silently, so it protects a
// deployment only as far as its database happens to be configured.
func warnAboutUnprotectedPostgresConnection(
	stateConfig *core.StateConfig,
	logger bpcore.Logger,
) {
	unprotected := stateConfig.PostgresSSLMode == postgresSSLModeDisable ||
		stateConfig.PostgresSSLMode == postgresSSLModePrefer
	if !unprotected || isLocalPostgresHost(stateConfig.PostgresHost) {
		return
	}

	logger.Warn(
		"state is being persisted to a postgres host over a connection that may not be "+
			"encrypted, so credentials and resolved secrets in state can be read in transit; "+
			"set state.postgres_ssl_mode to require, or verify-full where the deployment has "+
			"to verify the server it is talking to",
		bpcore.StringLogField("sslMode", stateConfig.PostgresSSLMode),
		bpcore.StringLogField("host", stateConfig.PostgresHost),
	)
}

// A path rather than a hostname is a unix socket directory, which does not leave the machine.
func isLocalPostgresHost(host string) bool {
	if host == "" || strings.HasPrefix(host, "/") {
		return true
	}

	return host == "localhost" || host == "127.0.0.1" || host == "::1"
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
