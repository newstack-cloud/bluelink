package commands

import (
	"errors"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/destroyui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// errDestroyFailed is a sentinel error used to indicate destroy failed
// after detailed error output has already been printed.
var errDestroyFailed = errors.New("destroy failed")

func setupDestroyCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a blueprint instance",
		Long: `Destroys a deployed blueprint instance, removing all associated resources,
child blueprints, and links.

The destruction streams events in real-time, allowing you to monitor progress
as resources are being destroyed.

Examples:
  # Interactive mode - select instance to destroy
  bluelink destroy

  # Destroy with pre-selected instance using latest destroy change set
  bluelink destroy --instance-name my-app

  # Destroy using a specific change set
  bluelink destroy --instance-name my-app --change-set-id abc123

  # Stage destroy changes first, then execute with auto-approve
  bluelink destroy --instance-name my-app --stage --auto-approve

  # Force destroy, overriding state conflicts
  bluelink destroy --instance-name my-app --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, handle, err := utils.SetupLogger()
			if err != nil {
				return err
			}
			defer handle.Close()

			destroyEngine, err := engine.Create(confProvider, logger)
			if err != nil {
				return err
			}

			changesetID, changesetIDIsDefault := confProvider.GetString("destroyChangeSetID")
			instanceID, instanceIDIsDefault := confProvider.GetString("destroyInstanceID")
			instanceName, instanceNameIsDefault := confProvider.GetString("destroyInstanceName")
			blueprintFile, isDefaultBlueprintFile := confProvider.GetString("destroyBlueprintFile")
			stageFirst, _ := confProvider.GetBool("destroyStage")
			autoApprove, _ := confProvider.GetBool("destroyAutoApprove")
			skipPrompts, _ := confProvider.GetBool("destroySkipPrompts")
			force, _ := confProvider.GetBool("destroyForce")
			jsonMode, _ := confProvider.GetBool("destroyJson")

			// In JSON mode, silence all Cobra error output - the TUI handles JSON error output
			// Also imply --auto-approve since JSON mode is non-interactive
			if jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				autoApprove = true
			}

			// Validate flag combinations in headless mode
			err = headless.Validate(
				// Either instance-name or instance-id must be provided
				headless.OneOf(
					headless.Flag{
						Name:      "instance-name",
						Value:     instanceName,
						IsDefault: instanceNameIsDefault,
					},
					headless.Flag{
						Name:      "instance-id",
						Value:     instanceID,
						IsDefault: instanceIDIsDefault,
					},
				),
				// Either --stage or --change-set-id must be provided
				headless.OneOf(
					headless.Flag{
						Name:      "stage",
						Value:     boolToString(stageFirst),
						IsDefault: !stageFirst,
					},
					headless.Flag{
						Name:      "change-set-id",
						Value:     changesetID,
						IsDefault: changesetIDIsDefault,
					},
				),
				// When --stage is set, --auto-approve is required
				headless.RequiredIfBool(
					headless.BoolFlagTrue("stage", stageFirst),
					"auto-approve",
					autoApprove,
				),
			)
			if err != nil {
				if jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errDestroyFailed
				}
				return err
			}

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			// From this point onwards, errors will not be related to usage
			// so the usage should not be printed if destroy fails,
			// we still need to return an error to allow cobra to exit with a non-zero exit code.
			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headlessMode := !inTerminal || jsonMode
			app, err := destroyui.NewDestroyApp(
				destroyEngine,
				logger,
				changesetID,
				instanceID,
				instanceName,
				blueprintFile,
				isDefaultBlueprintFile,
				force,
				stageFirst,
				autoApprove,
				skipPrompts,
				styles,
				headlessMode,
				os.Stdout,
				jsonMode,
			)
			if err != nil {
				return err
			}

			options := []tea.ProgramOption{}
			if !headlessMode {
				options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
			} else {
				options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
			}

			finalModel, err := tea.NewProgram(app, options...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(destroyui.MainModel)

			if finalApp.Error != nil {
				// The TUI has already displayed the detailed error (or JSON output).
				// Silence Cobra's error printing and return a sentinel error
				// just to ensure non-zero exit code.
				cmd.SilenceErrors = true
				return errDestroyFailed
			}

			return nil
		},
	}

	destroyCmd.PersistentFlags().String(
		"change-set-id",
		"",
		"The ID of the change set to use for destruction. "+
			"If not provided, the latest destroy change set for the instance will be used.",
	)
	confProvider.BindPFlag("destroyChangeSetID", destroyCmd.PersistentFlags().Lookup("change-set-id"))
	confProvider.BindEnvVar("destroyChangeSetID", "BLUELINK_CLI_DESTROY_CHANGE_SET_ID")

	destroyCmd.PersistentFlags().String(
		"instance-id",
		"",
		"The system-generated ID of the blueprint instance to destroy. "+
			"Leave empty if using --instance-name.",
	)
	confProvider.BindPFlag("destroyInstanceID", destroyCmd.PersistentFlags().Lookup("instance-id"))
	confProvider.BindEnvVar("destroyInstanceID", "BLUELINK_CLI_DESTROY_INSTANCE_ID")

	destroyCmd.PersistentFlags().String(
		"instance-name",
		"",
		"The user-defined unique identifier for the blueprint instance to destroy. "+
			"Leave empty if using --instance-id.",
	)
	confProvider.BindPFlag("destroyInstanceName", destroyCmd.PersistentFlags().Lookup("instance-name"))
	confProvider.BindEnvVar("destroyInstanceName", "BLUELINK_CLI_DESTROY_INSTANCE_NAME")

	destroyCmd.PersistentFlags().String(
		"blueprint-file",
		"project.blueprint.yaml",
		"The blueprint file for staging destroy changes. "+
			"Only used when --stage is set. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("destroyBlueprintFile", destroyCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("destroyBlueprintFile", "BLUELINK_CLI_DESTROY_BLUEPRINT_FILE")

	destroyCmd.PersistentFlags().Bool(
		"force",
		false,
		"Override state conflicts and force destruction.",
	)
	confProvider.BindPFlag("destroyForce", destroyCmd.PersistentFlags().Lookup("force"))
	confProvider.BindEnvVar("destroyForce", "BLUELINK_CLI_DESTROY_FORCE")

	destroyCmd.PersistentFlags().Bool(
		"stage",
		false,
		"Stage destroy changes and review them before execution. "+
			"When set, the CLI will first run the change staging process to show "+
			"what changes will be applied, allowing you to review and confirm before destroying.",
	)
	confProvider.BindPFlag("destroyStage", destroyCmd.PersistentFlags().Lookup("stage"))
	confProvider.BindEnvVar("destroyStage", "BLUELINK_CLI_DESTROY_STAGE")

	destroyCmd.PersistentFlags().Bool(
		"auto-approve",
		false,
		"Automatically approve staged changes without prompting for confirmation. "+
			"This is intended for CI/CD pipelines where manual approval is not possible. "+
			"Only applicable when --stage is set.",
	)
	confProvider.BindPFlag("destroyAutoApprove", destroyCmd.PersistentFlags().Lookup("auto-approve"))
	confProvider.BindEnvVar("destroyAutoApprove", "BLUELINK_CLI_DESTROY_AUTO_APPROVE")

	destroyCmd.PersistentFlags().Bool(
		"skip-prompts",
		false,
		"Skip interactive prompts and use flag values directly. "+
			"Requires all necessary flags to be provided (--instance-name or --instance-id, "+
			"and either --stage or --change-set-id).",
	)
	confProvider.BindPFlag("destroySkipPrompts", destroyCmd.PersistentFlags().Lookup("skip-prompts"))
	confProvider.BindEnvVar("destroySkipPrompts", "BLUELINK_CLI_DESTROY_SKIP_PROMPTS")

	destroyCmd.PersistentFlags().Bool(
		"json",
		false,
		"Output result as a single JSON object when the operation completes. "+
			"Implies non-interactive mode (no TUI, no streaming text output).",
	)
	confProvider.BindPFlag("destroyJson", destroyCmd.PersistentFlags().Lookup("json"))

	rootCmd.AddCommand(destroyCmd)
}
