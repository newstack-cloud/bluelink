package commands

import (
	"errors"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/templatelistui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errListTemplatesFailed = errors.New("list templates failed")

func setupTemplatesCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	templatesCmd := &cobra.Command{
		Use:   "templates",
		Short: "Browse available blueprint project templates",
		Long: `Commands for browsing blueprint project templates maintained by the
Bluelink team. Use subcommands to list and explore available templates.`,
	}

	setupTemplatesListCommand(templatesCmd, confProvider)

	rootCmd.AddCommand(templatesCmd)
}

func setupTemplatesListCommand(
	templatesCmd *cobra.Command,
	confProvider *config.Provider,
) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available blueprint project templates",
		Long: `Lists all available blueprint project templates maintained by the Bluelink team.

In interactive mode, templates are displayed in a split pane with a navigable
list on the left and template details on the right. Use the built-in filter
to search templates interactively.

Examples:
  # Interactive mode - browse and filter templates
  bluelink templates list

  # Filter templates by name
  bluelink templates list --search "api"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runTemplatesList(confProvider)
		},
	}

	listCmd.PersistentFlags().String(
		"search",
		"",
		"Filter templates by name (case-insensitive substring match).",
	)
	confProvider.BindPFlag(
		"templatesListSearch",
		listCmd.PersistentFlags().Lookup("search"),
	)
	confProvider.BindEnvVar(
		"templatesListSearch",
		"BLUELINK_CLI_TEMPLATES_LIST_SEARCH",
	)

	templatesCmd.AddCommand(listCmd)
}

func runTemplatesList(confProvider *config.Provider) error {
	search, _ := confProvider.GetString("templatesListSearch")

	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	app, err := templatelistui.NewListApp(templatelistui.ListAppOptions{
		Search:         search,
		Styles:         styles,
		Headless:       headlessMode,
		HeadlessWriter: os.Stdout,
	})
	if err != nil {
		return err
	}

	var teaOpts []tea.ProgramOption
	if headlessMode {
		teaOpts = append(teaOpts, tea.WithoutRenderer(), tea.WithInput(nil))
	} else {
		teaOpts = append(teaOpts, tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

	finalModel, err := tea.NewProgram(app, teaOpts...).Run()
	if err != nil {
		return err
	}

	switch m := finalModel.(type) {
	case templatelistui.MainModel:
		if m.Error != nil {
			return errListTemplatesFailed
		}
	case *templatelistui.MainModel:
		if m.Error != nil {
			return errListTemplatesFailed
		}
	}

	return nil
}
