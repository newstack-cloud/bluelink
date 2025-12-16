package commands

import (
	"errors"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stageui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// errStagingFailed is a sentinel error used to indicate staging failed
// after detailed error output has already been printed.
var errStagingFailed = errors.New("staging failed")

func setupStageCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	stageCmd := &cobra.Command{
		Use:   "stage",
		Short: "Stage changes for a blueprint deployment",
		Long: `Creates a changeset by computing the differences between a blueprint
and the current state of an existing instance (or empty state for new deployments).

The changeset can then be applied using the deploy command.

Examples:
  # Stage changes for a new deployment
  bluelink stage

  # Stage changes for an existing instance by name
  bluelink stage --instance-name my-app

  # Stage changes for an existing instance by ID
  bluelink stage --instance-id abc123

  # Stage changes for destroying an instance
  bluelink stage --instance-name my-app --destroy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, handle, err := utils.SetupLogger()
			if err != nil {
				return err
			}
			defer handle.Close()

			deployEngine, err := engine.Create(confProvider, logger)
			if err != nil {
				return err
			}

			blueprintFile, isDefault := confProvider.GetString("stageBlueprintFile")
			instanceID, _ := confProvider.GetString("stageInstanceID")
			instanceName, _ := confProvider.GetString("stageInstanceName")
			destroy, _ := confProvider.GetBool("stageDestroy")
			skipDriftCheck, _ := confProvider.GetBool("stageSkipDriftCheck")

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			// From this point onwards, errors will not be related to usage
			// so the usage should not be printed if staging fails,
			// we still need to return an error to allow cobra to exit with a non-zero exit code.
			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			app, err := stageui.NewStageApp(
				deployEngine,
				logger,
				blueprintFile,
				isDefault,
				instanceID,
				instanceName,
				destroy,
				skipDriftCheck,
				styles,
				!inTerminal,
				os.Stdout,
			)
			if err != nil {
				return err
			}

			options := []tea.ProgramOption{}
			if inTerminal {
				options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
			} else {
				options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
			}

			finalModel, err := tea.NewProgram(app, options...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(stageui.MainModel)

			if finalApp.Error != nil {
				// The TUI has already displayed the detailed error.
				// Silence Cobra's error printing and return a sentinel error
				// just to ensure non-zero exit code.
				cmd.SilenceErrors = true
				return errStagingFailed
			}

			return nil
		},
	}

	stageCmd.PersistentFlags().String(
		"blueprint-file",
		"project.blueprint.yaml",
		"The blueprint file to stage. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("stageBlueprintFile", stageCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("stageBlueprintFile", "BLUELINK_CLI_STAGE_BLUEPRINT_FILE")

	stageCmd.PersistentFlags().String(
		"instance-id",
		"",
		"The ID of an existing blueprint instance to stage changes for. "+
			"If not provided and --instance-name is not provided, changes will be staged for a new deployment.",
	)
	confProvider.BindPFlag("stageInstanceID", stageCmd.PersistentFlags().Lookup("instance-id"))
	confProvider.BindEnvVar("stageInstanceID", "BLUELINK_CLI_STAGE_INSTANCE_ID")

	stageCmd.PersistentFlags().String(
		"instance-name",
		"",
		"The user-defined name of an existing blueprint instance to stage changes for. "+
			"If not provided and --instance-id is not provided, changes will be staged for a new deployment.",
	)
	confProvider.BindPFlag("stageInstanceName", stageCmd.PersistentFlags().Lookup("instance-name"))
	confProvider.BindEnvVar("stageInstanceName", "BLUELINK_CLI_STAGE_INSTANCE_NAME")

	stageCmd.PersistentFlags().Bool(
		"destroy",
		false,
		"Stage changes for destroying an existing instance. "+
			"Requires --instance-id or --instance-name to be provided.",
	)
	confProvider.BindPFlag("stageDestroy", stageCmd.PersistentFlags().Lookup("destroy"))
	confProvider.BindEnvVar("stageDestroy", "BLUELINK_CLI_STAGE_DESTROY")

	stageCmd.PersistentFlags().Bool(
		"skip-drift-check",
		false,
		"Skip detection of external resource changes during staging.",
	)
	confProvider.BindPFlag("stageSkipDriftCheck", stageCmd.PersistentFlags().Lookup("skip-drift-check"))
	confProvider.BindEnvVar("stageSkipDriftCheck", "BLUELINK_CLI_STAGE_SKIP_DRIFT_CHECK")

	rootCmd.AddCommand(stageCmd)
}
