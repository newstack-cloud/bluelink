package main

import (
	"log"

	"github.com/newstack-cloud/bluelink/apps/cli/cmd/commands"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(commands.OnInitialise)
}

func main() {
	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
