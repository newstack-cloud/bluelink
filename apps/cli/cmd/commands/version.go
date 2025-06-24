package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func setupVersionCommand(rootCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Bluelink CLI",
		Long:  `All software has versions. This is Bluelink CLI's`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Bluelink CLI v0.1")
		},
	}

	rootCmd.AddCommand(versionCmd)
}
