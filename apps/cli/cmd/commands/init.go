package commands

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/git"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/initui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func setupInitCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialises a new Bluelink project",
		Long: `Initialises a new Bluelink project, this will take you through an interactive set up
		process but you can also use flags to skip certain prompts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := ""
			if len(args) >= 1 {
				directory = args[0]
			}

			template, isDefaultTemplate := confProvider.GetString("initTemplate")
			projectName, isDefaultProjectName := confProvider.GetString("initProjectName")
			blueprintFormat, isDefaultBlueprintFormat := confProvider.GetString("initBlueprintFormat")
			noGit, isDefaultNoGit := confProvider.GetBool("initNoGit")
			noGitPtr := &noGit

			// Validate required flags in headless mode
			if err := headless.Validate(
				headless.Required(headless.Flag{
					Name:      "project-name",
					Value:     projectName,
					IsDefault: isDefaultProjectName,
				}),
			); err != nil {
				return err
			}

			if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
				log.Fatal(err)
			}

			gitService := git.NewDefaultGit()

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				stylespkg.NewBluelinkPalette(),
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			app, err := initui.NewInitApp(
				initui.InitialState{
					Template:                 template,
					IsDefaultTemplate:        isDefaultTemplate,
					ProjectName:              projectName,
					BlueprintFormat:          blueprintFormat,
					IsDefaultBlueprintFormat: isDefaultBlueprintFormat,
					NoGit:                    noGitPtr,
					IsDefaultNoGit:           isDefaultNoGit,
					Directory:                directory,
				},
				styles,
				gitService,
				project.NewDefaultPreparer(),
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
			finalApp := finalModel.(initui.InitModel)

			if finalApp.Error != nil {
				return finalApp.Error
			}

			return err
		},
	}

	initCmd.PersistentFlags().String(
		"template",
		"scaffold",
		"The template to use for the new blueprint project. "+
			"The default template is a scaffold project that generates "+
			"all the files needed for a project, populating the blueprint file with placeholder values.",
	)
	confProvider.BindPFlag("initTemplate", initCmd.PersistentFlags().Lookup("template"))
	confProvider.BindEnvVar("initTemplate", "BLUELINK_CLI_INIT_TEMPLATE")

	initCmd.PersistentFlags().String(
		"project-name",
		"",
		"The name of the new blueprint project.",
	)
	confProvider.BindPFlag("initProjectName", initCmd.PersistentFlags().Lookup("project-name"))
	confProvider.BindEnvVar("initProjectName", "BLUELINK_CLI_INIT_PROJECT_NAME")

	initCmd.PersistentFlags().String(
		"blueprint-format",
		"yaml",
		"The format of the blueprint file to use for the new project. "+
			"The default format is yaml, but can be set to json or toml.",
	)
	confProvider.BindPFlag("initBlueprintFormat", initCmd.PersistentFlags().Lookup("blueprint-format"))
	confProvider.BindEnvVar("initBlueprintFormat", "BLUELINK_CLI_INIT_BLUEPRINT_FORMAT")

	initCmd.PersistentFlags().Bool(
		"no-git",
		false,
		"Whether to initialise the project without git. "+
			"If this is set to true, the project will not be initialised with a git repository.",
	)
	confProvider.BindPFlag("initNoGit", initCmd.PersistentFlags().Lookup("no-git"))
	confProvider.BindEnvVar("initNoGit", "BLUELINK_CLI_INIT_NO_GIT")

	rootCmd.AddCommand(initCmd)
}
