package stateio

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

// ImportParams contains the parameters for an import operation.
type ImportParams struct {
	// FilePath is the path to the input file (local or remote URL).
	FilePath string
	// EngineConfig contains the deploy engine configuration.
	// Used to determine the storage backend (memfile or postgres).
	EngineConfig *EngineConfig
	// FileSystem is the filesystem to use for operations.
	FileSystem afero.Fs
	// Logger is the logger to use for logging.
	Logger core.Logger
	// FileData contains the raw file data to import.
	// If provided, FilePath is ignored for reading (but may be used for logging).
	FileData []byte
	// RemoteOptions contains options for downloading from remote storage.
	RemoteOptions *RemoteDownloadOptions
	// Importer is an optional StateImporter for import.
	// If not provided, a default importer will be created based on EngineConfig.
	Importer StateImporter
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Success        bool   `json:"success"`
	InstancesCount int    `json:"instancesCount,omitempty"`
	Message        string `json:"message"`
}

// Import performs a state import operation based on the provided parameters.
// The input file must be a JSON array of blueprint instances.
func Import(params ImportParams) (*ImportResult, error) {
	if params.FileSystem == nil {
		params.FileSystem = afero.NewOsFs()
	}

	data, err := getInputData(params)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	importer := params.Importer
	if importer == nil {
		importer, err = createDefaultImporter(params)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	result, err := ExecuteInstancesImport(ctx, importer, data)
	if err != nil {
		return nil, err
	}

	return &ImportResult{
		Success:        true,
		InstancesCount: result.InstancesCount,
		Message:        fmt.Sprintf("Successfully imported %d instances", result.InstancesCount),
	}, nil
}

func getInputData(params ImportParams) ([]byte, error) {
	if params.FileData != nil {
		return params.FileData, nil
	}

	if IsRemoteFile(params.FilePath) {
		return DownloadRemoteFile(context.Background(), params.FilePath, params.RemoteOptions)
	}

	return os.ReadFile(params.FilePath)
}

func createDefaultImporter(params ImportParams) (StateImporter, error) {
	logger := params.Logger
	if logger == nil {
		logger = core.NewNopLogger()
	}

	if params.EngineConfig == nil {
		return nil, fmt.Errorf("engine config is required for import")
	}

	return createImporterFromEngineConfig(params.EngineConfig, params.FileSystem, logger)
}

func createImporterFromEngineConfig(
	config *EngineConfig,
	fileSystem afero.Fs,
	logger core.Logger,
) (StateImporter, error) {
	switch config.State.StorageEngine {
	case StorageEnginePostgres:
		return createPostgresImporter(&config.State, logger)
	case StorageEngineMemfile, "":
		return createMemfileImporter(config.State.MemFileStateDir, fileSystem, logger)
	default:
		return nil, fmt.Errorf(
			"unsupported storage engine %q, only \"memfile\" and \"postgres\" are supported",
			config.State.StorageEngine,
		)
	}
}

func createMemfileImporter(
	stateDir string,
	fileSystem afero.Fs,
	logger core.Logger,
) (StateImporter, error) {
	container, err := memfile.LoadStateContainer(
		stateDir,
		fileSystem,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load memfile state container: %w", err)
	}

	return NewContainerStateImporter(container), nil
}

func createPostgresImporter(
	config *StateConfig,
	logger core.Logger,
) (StateImporter, error) {
	ctx := context.Background()
	connURL := BuildPostgresDatabaseURL(config)

	pool, err := pgxpool.New(ctx, connURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	container, err := postgres.LoadStateContainer(ctx, pool, logger)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to load postgres state container: %w", err)
	}

	return NewContainerStateImporter(container), nil
}
