package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/spf13/afero"
)

const reencryptStateCommand = "reencrypt-state"

// Re-seals a memfile state directory under the configured key, which is how a deployment
// encrypts state that was written in plaintext, moves state to a new key, or decrypts it.
//
// Lives in the engine binary rather than a separate tool because it has to read the same
// configuration the engine does, and because whoever rotates a key has the engine to hand.
func runReencryptState(args []string) int {
	flags := flag.NewFlagSet(reencryptStateCommand, flag.ExitOnError)
	dryRun := flags.Bool(
		"dry-run",
		false,
		"report what would change without writing anything",
	)
	decrypt := flags.Bool(
		"decrypt",
		false,
		"write state back as plaintext, which backs state encryption out",
	)
	backupDir := flags.String(
		"backup-dir",
		"",
		"where to copy the state before rewriting it (default: alongside the state directory)",
	)
	skipBackup := flags.Bool(
		"no-backup",
		false,
		"rewrite state without copying it first",
	)
	force := flags.Bool(
		"force",
		false,
		"proceed even though something is listening on the engine's address",
	)
	if err := flags.Parse(args); err != nil {
		return 1
	}

	config, err := core.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		return 1
	}

	report, err := enginev1.ReencryptState(
		context.Background(),
		&config,
		afero.NewOsFs(),
		enginev1.ReencryptStateOptions{
			DryRun:     *dryRun,
			Decrypt:    *decrypt,
			BackupDir:  *backupDir,
			SkipBackup: *skipBackup,
			Force:      *force,
			Out:        os.Stdout,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error re-encrypting state: %s\n", err)
		return 1
	}

	printReencryptReport(report, *dryRun)

	return 0
}

func printReencryptReport(report *statestore.ReencryptReport, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "would have "
	}

	fmt.Printf(
		"%sre-encrypted %d, encrypted %d, decrypted %d, left %d already under the current key, "+
			"skipped %d\n",
		prefix,
		report.Reencrypted,
		report.Encrypted,
		report.Decrypted,
		report.AlreadyCurrent,
		report.Skipped,
	)

	if dryRun {
		fmt.Println("nothing was written, run again without --dry-run to apply it")
	}
}
