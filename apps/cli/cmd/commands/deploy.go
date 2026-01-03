package commands

import (
	"errors"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// errDeploymentFailed is a sentinel error used to indicate deployment failed
// after detailed error output has already been printed.
var errDeploymentFailed = errors.New("deployment failed")

// boolToString converts a boolean to a string for flag validation.
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return ""
}

func setupDeployCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a blueprint instance",
		Long: `Executes a change set for a blueprint instance, supporting both new
deployments and updates to existing instances.

The deployment streams events in real-time, allowing you to monitor progress
of resources, child blueprints, and links as they are deployed.

Examples:
  # Interactive mode - select blueprint and instance
  bluelink deploy

  # Deploy with pre-selected instance using latest change set
  bluelink deploy --instance-name my-app

  # Deploy specific change set
  bluelink deploy --instance-name my-app --change-set-id abc123

  # Deploy from a specific blueprint file
  bluelink deploy --blueprint-file ./project.blueprint.yaml --instance-name my-app

  # Deploy with auto-rollback enabled
  bluelink deploy --instance-name my-app --auto-rollback`,
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

			changesetID, changesetIDIsDefault := confProvider.GetString("deployChangeSetID")
			instanceID, instanceIDIsDefault := confProvider.GetString("deployInstanceID")
			instanceName, instanceNameIsDefault := confProvider.GetString("deployInstanceName")
			blueprintFile, isDefault := confProvider.GetString("deployBlueprintFile")
			stageFirst, _ := confProvider.GetBool("deployStage")
			autoApprove, _ := confProvider.GetBool("deployAutoApprove")
			skipPrompts, _ := confProvider.GetBool("deploySkipPrompts")
			autoRollback, _ := confProvider.GetBool("deployAutoRollback")
			force, _ := confProvider.GetBool("deployForce")
			jsonMode, _ := confProvider.GetBool("deployJson")

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
					return errDeploymentFailed
				}
				return err
			}

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			// From this point onwards, errors will not be related to usage
			// so the usage should not be printed if deployment fails,
			// we still need to return an error to allow cobra to exit with a non-zero exit code.
			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headless := !inTerminal || jsonMode
			app, err := deployui.NewDeployApp(
				deployEngine,
				logger,
				changesetID,
				instanceID,
				instanceName,
				blueprintFile,
				isDefault,
				autoRollback,
				force,
				stageFirst,
				autoApprove,
				skipPrompts,
				styles,
				headless,
				os.Stdout,
				jsonMode,
			)
			if err != nil {
				return err
			}

			options := []tea.ProgramOption{}
			if !headless {
				options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
			} else {
				options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
			}

			finalModel, err := tea.NewProgram(app, options...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(deployui.MainModel)

			if finalApp.Error != nil {
				// The TUI has already displayed the detailed error (or JSON output).
				// Silence Cobra's error printing and return a sentinel error
				// just to ensure non-zero exit code.
				cmd.SilenceErrors = true
				return errDeploymentFailed
			}

			return nil
		},
	}

	deployCmd.PersistentFlags().String(
		"change-set-id",
		"",
		"The ID of the change set to deploy. "+
			"If not provided, the latest change set for the instance will be used.",
	)
	confProvider.BindPFlag("deployChangeSetID", deployCmd.PersistentFlags().Lookup("change-set-id"))
	confProvider.BindEnvVar("deployChangeSetID", "BLUELINK_CLI_DEPLOY_CHANGE_SET_ID")

	deployCmd.PersistentFlags().String(
		"instance-id",
		"",
		"The system-generated ID of the blueprint instance to deploy to. "+
			"Leave empty if using --instance-name or for new deployments.",
	)
	confProvider.BindPFlag("deployInstanceID", deployCmd.PersistentFlags().Lookup("instance-id"))
	confProvider.BindEnvVar("deployInstanceID", "BLUELINK_CLI_DEPLOY_INSTANCE_ID")

	deployCmd.PersistentFlags().String(
		"instance-name",
		"",
		"The user-defined unique identifier for the target blueprint instance. "+
			"Leave empty if using --instance-id or for new deployments.",
	)
	confProvider.BindPFlag("deployInstanceName", deployCmd.PersistentFlags().Lookup("instance-name"))
	confProvider.BindEnvVar("deployInstanceName", "BLUELINK_CLI_DEPLOY_INSTANCE_NAME")

	deployCmd.PersistentFlags().String(
		"blueprint-file",
		"project.blueprint.yaml",
		"The blueprint file for runtime substitution resolution. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("deployBlueprintFile", deployCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("deployBlueprintFile", "BLUELINK_CLI_DEPLOY_BLUEPRINT_FILE")

	deployCmd.PersistentFlags().Bool(
		"auto-rollback",
		false,
		"Automatically rollback on deployment failure.",
	)
	confProvider.BindPFlag("deployAutoRollback", deployCmd.PersistentFlags().Lookup("auto-rollback"))
	confProvider.BindEnvVar("deployAutoRollback", "BLUELINK_CLI_DEPLOY_AUTO_ROLLBACK")

	deployCmd.PersistentFlags().Bool(
		"force",
		false,
		"Override state conflicts and force deployment.",
	)
	confProvider.BindPFlag("deployForce", deployCmd.PersistentFlags().Lookup("force"))
	confProvider.BindEnvVar("deployForce", "BLUELINK_CLI_DEPLOY_FORCE")

	deployCmd.PersistentFlags().Bool(
		"stage",
		false,
		"Stage changes and review them before deployment. "+
			"When set, the CLI will first run the change staging process to show "+
			"what changes will be applied, allowing you to review and confirm before deploying.",
	)
	confProvider.BindPFlag("deployStage", deployCmd.PersistentFlags().Lookup("stage"))
	confProvider.BindEnvVar("deployStage", "BLUELINK_CLI_DEPLOY_STAGE")

	deployCmd.PersistentFlags().Bool(
		"auto-approve",
		false,
		"Automatically approve staged changes without prompting for confirmation. "+
			"This is intended for CI/CD pipelines where manual approval is not possible. "+
			"Only applicable when --stage is set.",
	)
	confProvider.BindPFlag("deployAutoApprove", deployCmd.PersistentFlags().Lookup("auto-approve"))
	confProvider.BindEnvVar("deployAutoApprove", "BLUELINK_CLI_DEPLOY_AUTO_APPROVE")

	deployCmd.PersistentFlags().Bool(
		"skip-prompts",
		false,
		"Skip interactive prompts and use flag values directly. "+
			"Requires all necessary flags to be provided (--instance-name or --instance-id, "+
			"and either --stage or --change-set-id).",
	)
	confProvider.BindPFlag("deploySkipPrompts", deployCmd.PersistentFlags().Lookup("skip-prompts"))
	confProvider.BindEnvVar("deploySkipPrompts", "BLUELINK_CLI_DEPLOY_SKIP_PROMPTS")

	deployCmd.PersistentFlags().Bool(
		"json",
		false,
		"Output result as a single JSON object when the operation completes. "+
			"Implies non-interactive mode (no TUI, no streaming text output).",
	)
	confProvider.BindPFlag("deployJson", deployCmd.PersistentFlags().Lookup("json"))

	rootCmd.AddCommand(deployCmd)
}
