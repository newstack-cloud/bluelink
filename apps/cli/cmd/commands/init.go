package commands

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/config"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/initui"
	"github.com/spf13/cobra"
)

func setupInitCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialises a new Bluelink project",
		Long: `Initialises a new Bluelink project, this will take you through an interactive set up
		process but you can also use flags to skip certain prompts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := tea.NewProgram(initui.NewInitApp("")).Run()
			return err
		},
	}

	rootCmd.AddCommand(initCmd)
}
