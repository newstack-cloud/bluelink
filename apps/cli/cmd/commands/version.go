package commands

import (
	"github.com/spf13/cobra"
)

func setupVersionCommand(rootCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Bluelink CLI",
		Long:  `All software has versions. This is Bluelink CLI's`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Use goreleaser to inject version
			cmd.Println("Bluelink CLI v0.1.0")
		},
	}

	rootCmd.AddCommand(versionCmd)
}
