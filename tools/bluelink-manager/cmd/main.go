package main

import (
	"log"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/cmd/commands"
)

func main() {
	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
