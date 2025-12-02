package commands

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/config"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/engine"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/validateui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func setupValidateCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates a blueprint",
		Long: `Carries out validation on a blueprint.
	You can use this command to check for issues with a blueprint
	before deployment.

	It's worth noting that validation is carried out as a part of the deploy command as well.`,
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
			blueprintFile, isDefault := confProvider.GetString("validateBlueprintFile")

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			// From this point onwards, errors will not be related to usage
			// so the usage should not be printed if validation fails,
			// we still need to return an error to allow cobra to exit with a non-zero exit code.
			cmd.SilenceUsage = true

			styles := styles.NewDefaultBluelinkStyles()
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			app, err := validateui.NewValidateApp(
				deployEngine,
				logger,
				blueprintFile,
				isDefault,
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
			finalApp := finalModel.(validateui.MainModel)

			if finalApp.Error != nil {
				return finalApp.Error
			}

			return nil
		},
	}

	validateCmd.PersistentFlags().String(
		"blueprint-file",
		"project.blueprint.yaml",
		"The blueprint file to validate. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("validateBlueprintFile", validateCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("validateBlueprintFile", "BLUELINK_CLI_VALIDATE_BLUEPRINT_FILE")

	rootCmd.AddCommand(validateCmd)
}
