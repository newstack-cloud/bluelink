package commands

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	bluelinkpreflight "github.com/newstack-cloud/bluelink/apps/cli/internal/preflight"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/validateui"
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
			transformSpecPtr := boolPtrIfSet(confProvider, "validateTransformSpec")
			validateAfterTransformPtr := boolPtrIfSet(confProvider, "validateValidateAfterTransform")

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			// From this point onwards, errors will not be related to usage
			// so the usage should not be printed if validation fails,
			// we still need to return an error to allow cobra to exit with a non-zero exit code.
			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headless := !inTerminal

			skipCheck, _ := confProvider.GetBool("skipPluginCheck")
			var preflightModel tea.Model
			if !skipCheck {
				factory := &bluelinkpreflight.BluelinkPreflightFactory{}
				preflightModel = factory.CreatePreflight(
					confProvider, "validate", styles, headless, os.Stdout, false,
				)
			}

			app, err := validateui.NewValidateApp(validateui.ValidateAppConfig{
				Engine:                 deployEngine,
				Logger:                 logger,
				BlueprintFile:          blueprintFile,
				IsDefaultBlueprintFile: isDefault,
				Styles:                 styles,
				Headless:               headless,
				HeadlessWriter:         os.Stdout,
				Preflight:              preflightModel,
				TransformSpec:          transformSpecPtr,
				ValidateAfterTransform: validateAfterTransformPtr,
			})
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
		project.DetectBlueprintFile("."),
		"The blueprint file to validate. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("validateBlueprintFile", validateCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("validateBlueprintFile", "BLUELINK_CLI_VALIDATE_BLUEPRINT_FILE")

	validateCmd.PersistentFlags().Bool(
		"transform-spec",
		true,
		"Run transformer plugins during validation so abstract resources are expanded into concrete resources for richer diagnostics. "+
			"Required for transformer-driven workflows to produce meaningful validation output. "+
			"When set to false, the blueprint will not be transformed during validation.",
	)
	confProvider.BindPFlag("validateTransformSpec", validateCmd.PersistentFlags().Lookup("transform-spec"))
	confProvider.BindEnvVar("validateTransformSpec", "BLUELINK_CLI_VALIDATE_TRANSFORM_SPEC")

	validateCmd.PersistentFlags().Bool(
		"validate-after-transform",
		false,
		"After transformation, validate resources against the transformed blueprint shape. "+
			"Catches resource-level issues that only surface once abstract resources have been expanded into their concrete forms, "+
			"which is typically useful when diagnosing deployment-time issues. "+
			"Has no effect unless --transform-spec is also true.",
	)
	confProvider.BindPFlag("validateValidateAfterTransform", validateCmd.PersistentFlags().Lookup("validate-after-transform"))
	confProvider.BindEnvVar("validateValidateAfterTransform", "BLUELINK_CLI_VALIDATE_AFTER_TRANSFORM")

	rootCmd.AddCommand(validateCmd)
}

// Returns a pointer to the resolved bool config value when the
// user has explicitly provided it (via flag, env var, or config file), and
// nil when the value comes from the cobra default. The deploy-cli-sdk's
// validateui uses this distinction to decide whether to show its interactive
// options form: nil means "ask the user", non-nil means "use this value".
func boolPtrIfSet(confProvider *config.Provider, key string) *bool {
	value, isDefault := confProvider.GetBool(key)
	if isDefault {
		return nil
	}
	return &value
}
