package commands

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateimportui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// errStateImportFailed is a sentinel error used to indicate state import failed
// The actual error details have already been displayed by the TUI.
var errStateImportFailed = errors.New("state import failed")

func setupStateCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "Manage deploy engine state",
		Long:  `Commands for managing deploy engine state, including import and export operations.`,
	}

	// Add persistent flag for engine config file (shared by all subcommands)
	stateCmd.PersistentFlags().String(
		"engine-config-file",
		"",
		"Path to deploy engine config file. Used to determine storage backend.",
	)
	confProvider.BindPFlag("stateEngineConfigFile", stateCmd.PersistentFlags().Lookup("engine-config-file"))
	confProvider.BindEnvVar("stateEngineConfigFile", "BLUELINK_CLI_STATE_ENGINE_CONFIG_FILE")

	setupStateImportCommand(stateCmd, confProvider)

	rootCmd.AddCommand(stateCmd)
}

func setupStateImportCommand(stateCmd *cobra.Command, confProvider *config.Provider) {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import state from a file",
		Long: `Import deploy engine state from a local file or remote object storage.

The input file must be a JSON array of blueprint instances. This format is
backend-agnostic and works with any storage backend (memfile, PostgreSQL, etc.).

Examples:
  # Import state from a local file
  bluelink state import --file ./backup/state.json

  # Import from S3
  bluelink state import --file s3://my-bucket/state.json

  # Import from GCS
  bluelink state import --file gcs://my-bucket/state.json

  # Import from Azure Blob Storage
  bluelink state import --file azureblob://my-container/state.json

  # Use deploy engine config to determine storage backend (flag inherited from state command)
  bluelink state --engine-config-file ~/.bluelink/engine/config.json import --file ./state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			filePath, filePathIsDefault := confProvider.GetString("stateImportFile")
			engineConfigFile, _ := confProvider.GetString("stateEngineConfigFile")
			jsonMode, _ := confProvider.GetBool("stateImportJson")

			// In JSON mode, silence all Cobra error output - the TUI handles JSON error output
			if jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true

				// In JSON mode, --file is required
				if filePathIsDefault || filePath == "" {
					err := fmt.Errorf("--file is required when --json is set")
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errStateImportFailed
				}
			}

			// Validate flag combinations in headless mode (non-terminal)
			err := headless.Validate(
				headless.Required(headless.Flag{
					Name:      "file",
					Value:     filePath,
					IsDefault: filePathIsDefault,
				}),
			)
			if err != nil {
				return err
			}

			// Load engine config to determine storage backend
			// If not provided, load from default location
			var engineConfig *stateio.EngineConfig
			if engineConfigFile != "" {
				engineConfig, err = stateio.LoadEngineConfig(engineConfigFile)
			} else {
				engineConfig, err = stateio.LoadEngineConfig(stateio.GetDefaultEngineConfigPath())
			}
			if err != nil {
				return fmt.Errorf("failed to load engine config: %w", err)
			}

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)

			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headlessMode := !inTerminal || jsonMode
			app, err := stateimportui.NewStateImportApp(stateimportui.StateImportAppConfig{
				FilePath:       filePath,
				EngineConfig:   engineConfig,
				Styles:         styles,
				Headless:       headlessMode,
				HeadlessWriter: os.Stdout,
				JSONMode:       jsonMode,
			})
			if err != nil {
				return err
			}

			options := []tea.ProgramOption{}
			if inTerminal && !jsonMode {
				options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
			} else {
				options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
			}

			finalModel, err := tea.NewProgram(app, options...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(stateimportui.MainModel)

			if finalApp.Error != nil {
				// The TUI has already displayed the detailed error (or JSON output).
				// Silence Cobra's error printing and return a sentinel error
				// just to ensure non-zero exit code.
				cmd.SilenceErrors = true
				return errStateImportFailed
			}

			return nil
		},
	}

	importCmd.Flags().String(
		"file",
		"",
		"Path to input file. Can be local or remote (s3://, gcs://, azureblob://).",
	)
	confProvider.BindPFlag("stateImportFile", importCmd.Flags().Lookup("file"))
	confProvider.BindEnvVar("stateImportFile", "BLUELINK_CLI_STATE_IMPORT_FILE")

	importCmd.Flags().Bool(
		"json",
		false,
		"Output result as JSON (for headless/CI mode).",
	)
	confProvider.BindPFlag("stateImportJson", importCmd.Flags().Lookup("json"))
	confProvider.BindEnvVar("stateImportJson", "BLUELINK_CLI_STATE_IMPORT_JSON")

	stateCmd.AddCommand(importCmd)
}
