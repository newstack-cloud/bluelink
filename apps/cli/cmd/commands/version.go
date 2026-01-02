package commands

import (
	"fmt"
	"runtime"

	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/spf13/cobra"
)

func setupVersionCommand(rootCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Bluelink CLI",
		Long:  `All software has versions. This is Bluelink CLI's`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Bluelink CLI " + utils.Version)
			cmd.Println(fmt.Sprintf("  OS/Arch:    %s/%s", runtime.GOOS, runtime.GOARCH))
			cmd.Println(fmt.Sprintf("  Built:      %s", utils.BuildTime))
		},
	}

	rootCmd.AddCommand(versionCmd)
}
