package stateio

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

// ExportParams contains the parameters for an export operation.
type ExportParams struct {
	// FilePath is the path to the output file (local or remote URL).
	FilePath string
	// InstanceFilters is a list of instance IDs or names to export.
	// If empty, all instances are exported.
	InstanceFilters []string
	// EngineConfig contains the deploy engine configuration.
	// Used to determine the storage backend (memfile or postgres).
	EngineConfig *EngineConfig
	// FileSystem is the filesystem to use for local file operations.
	FileSystem afero.Fs
	// Logger is the logger to use for logging.
	Logger core.Logger
	// RemoteOptions contains options for uploading to remote storage.
	RemoteOptions *RemoteUploadOptions
	// Exporter is an optional StateExporter for export.
	// If not provided, a default exporter will be created based on EngineConfig.
	Exporter StateExporter
}

// ExportResult contains the result of an export operation.
type ExportResult struct {
	Success        bool   `json:"success"`
	InstancesCount int    `json:"instancesCount,omitempty"`
	FilePath       string `json:"filePath,omitempty"`
	Message        string `json:"message"`
}

// Export performs a state export operation based on the provided parameters.
// The output is a JSON array of blueprint instances.
func Export(params ExportParams) (*ExportResult, error) {
	if params.FileSystem == nil {
		params.FileSystem = afero.NewOsFs()
	}

	exporter := params.Exporter
	var err error
	if exporter == nil {
		exporter, err = createDefaultExporter(params)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	result, err := ExecuteInstancesExport(ctx, exporter, params.InstanceFilters)
	if err != nil {
		return nil, err
	}

	err = writeOutputData(params, result.Data)
	if err != nil {
		return nil, err
	}

	return &ExportResult{
		Success:        true,
		InstancesCount: result.InstancesCount,
		FilePath:       params.FilePath,
		Message:        fmt.Sprintf("Successfully exported %d instances to %s", result.InstancesCount, params.FilePath),
	}, nil
}

func writeOutputData(params ExportParams, data []byte) error {
	if IsRemoteFile(params.FilePath) {
		return UploadRemoteFile(context.Background(), params.FilePath, data, params.RemoteOptions)
	}

	return afero.WriteFile(params.FileSystem, params.FilePath, data, 0644)
}

func createDefaultExporter(params ExportParams) (StateExporter, error) {
	logger := params.Logger
	if logger == nil {
		logger = core.NewNopLogger()
	}

	if params.EngineConfig == nil {
		return nil, fmt.Errorf("engine config is required for export")
	}

	return createExporterFromEngineConfig(params.EngineConfig, params.FileSystem, logger)
}

func createExporterFromEngineConfig(
	config *EngineConfig,
	fileSystem afero.Fs,
	logger core.Logger,
) (StateExporter, error) {
	switch config.State.StorageEngine {
	case StorageEnginePostgres:
		return createPostgresExporter(&config.State, logger)
	case StorageEngineMemfile, "":
		return createMemfileExporter(config.State.MemFileStateDir, fileSystem, logger)
	default:
		return nil, fmt.Errorf(
			"unsupported storage engine %q, only \"memfile\" and \"postgres\" are supported",
			config.State.StorageEngine,
		)
	}
}

func createMemfileExporter(
	stateDir string,
	fileSystem afero.Fs,
	logger core.Logger,
) (StateExporter, error) {
	container, err := memfile.LoadStateContainer(
		stateDir,
		fileSystem,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load memfile state container: %w", err)
	}

	return NewContainerStateExporter(container), nil
}

func createPostgresExporter(
	config *StateConfig,
	logger core.Logger,
) (StateExporter, error) {
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

	return NewContainerStateExporter(container), nil
}
