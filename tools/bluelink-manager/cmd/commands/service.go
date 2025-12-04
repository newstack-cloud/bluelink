package commands

import (
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/service"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

func setupServiceCommands(rootCmd *cobra.Command) {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Deploy Engine service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Start(); err != nil {
				return err
			}
			ui.Success("Deploy Engine started")
			return nil
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Deploy Engine service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Stop(); err != nil {
				return err
			}
			ui.Success("Deploy Engine stopped")
			return nil
		},
	}

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the Deploy Engine service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Restart(); err != nil {
				return err
			}
			ui.Success("Deploy Engine restarted")
			return nil
		},
	}

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
}
