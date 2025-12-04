package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build info set at build time via ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func setupVersionCommand(rootCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of bluelink-manager",
		Long:  `All software has versions. This is bluelink-manager's`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("bluelink-manager " + Version)
			cmd.Println(fmt.Sprintf("  OS/Arch:    %s/%s", runtime.GOOS, runtime.GOARCH))
			cmd.Println(fmt.Sprintf("  Built:      %s", BuildTime))
		},
	}

	rootCmd.AddCommand(versionCmd)
}
