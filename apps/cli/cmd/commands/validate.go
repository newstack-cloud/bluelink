package commands

import (
	"context"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/config"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/engine"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/handlers"
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

			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			if !inTerminal {
				handler := handlers.NewValidateHandler(
					deployEngine,
					blueprintFile,
					// When not in a terminal, print output
					// that is intended primarily for a human to read
					// should always go to stdout for the process.
					os.Stdout,
					// Logger is used to for more verbose, technical output
					// that is intended primarily for debugging.
					logger,
				)
				return handler.Handle(context.TODO())
			}

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			styles := styles.NewDefaultBluelinkStyles()
			app, err := validateui.NewValidateApp(deployEngine, logger, blueprintFile, isDefault, styles)
			if err != nil {
				return err
			}
			finalModel, err := tea.NewProgram(app).Run()
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

	validateCmd.PersistentFlags().StringP(
		"blueprint-file",
		"b",
		"app.blueprint.yaml",
		"The blueprint file to use in the validation process.",
	)
	confProvider.BindPFlag("validateBlueprintFile", validateCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("validateBlueprintFile", "BLUELINK_CLI_VALIDATE_BLUEPRINT_FILE")

	rootCmd.AddCommand(validateCmd)
}
