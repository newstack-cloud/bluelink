package enginev1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/spf13/afero"
)

// ReencryptStateOptions controls a re-encryption pass over a memfile state directory.
type ReencryptStateOptions struct {
	// DryRun reports what the pass would do and writes nothing, neither state nor backup.
	DryRun bool
	// Decrypt writes state back as plaintext instead of sealing it with the configured key,
	// which backs state encryption out. Needs the key that sealed the state to still be
	// configured.
	Decrypt bool
	// BackupDir is where a copy of the state is written before anything is rewritten. Empty
	// derives one alongside the state directory.
	BackupDir string
	// SkipBackup proceeds without taking a copy first. A pass rewrites objects in place, so
	// an interruption leaves a store holding a mix of keys; the copy is what makes that
	// recoverable.
	SkipBackup bool
	// Force proceeds even though something is listening on the engine's address. A running
	// engine holds state in memory and will write it back over this pass.
	Force bool
	// Out receives a line per object. Nil discards them.
	Out io.Writer
}

// ReencryptState re-seals a memfile state directory under the configured key.
//
// Covers encrypting a directory written in plaintext, moving one from a previous key to the
// current one, and decrypting one. Which of those happens follows from the configuration: the
// key under state.encryption is the target, state.encryption.previous_keys are the keys the
// existing objects are opened with, and Decrypt inverts the target.
//
// Run with the engine stopped. The engine holds state in memory and persists it on write, so
// a pass against a live engine is liable to be overwritten by it.
func ReencryptState(
	ctx context.Context,
	config *core.Config,
	fileSystem afero.Fs,
	opts ReencryptStateOptions,
) (*statestore.ReencryptReport, error) {
	if err := refuseUnlessMemfile(config.State.StorageEngine); err != nil {
		return nil, err
	}

	if err := refuseIfEngineIsRunning(config, opts.Force); err != nil {
		return nil, err
	}

	storage := memfile.NewFSStorage(fileSystem)
	keys := statestore.NewKeyBuilder(config.State.MemFileStateDir)

	salt, err := saltForPass(ctx, storage, keys, opts.DryRun)
	if err != nil {
		return nil, err
	}

	source, target, err := reencryptKeys(ctx, &config.State.Encryption, salt, opts.Decrypt)
	if err != nil {
		return nil, err
	}

	if !opts.DryRun && !opts.SkipBackup {
		backupDir, err := backupState(ctx, storage, fileSystem, keys, opts.BackupDir)
		if err != nil {
			return nil, err
		}

		writeLine(opts.Out, fmt.Sprintf("copied the state directory to %s", backupDir))
	}

	return statestore.Reencrypt(ctx, storage, statestore.ReencryptOptions{
		Keys:   keys,
		Source: source,
		Target: target,
		DryRun: opts.DryRun,
		Observe: func(key string, outcome statestore.ReencryptOutcome) {
			writeLine(opts.Out, fmt.Sprintf("%s %s", outcome, key))
		},
	})
}

// Only the memfile engine persists through a Storage this can decorate. For the other two the
// reason is worth saying, since a deployment that has configured encryption at rest in its
// database or bucket has not missed a step.
func refuseUnlessMemfile(storageEngine string) error {
	if storageEngine == memfileStorageEngine {
		return nil
	}

	if storageEngine == postgresStorageEngine || storageEngine == objectstoreStorageEngine {
		return fmt.Errorf(
			"re-encrypting state is only supported for the %q storage engine, this deployment "+
				"uses %q, where encryption at rest belongs to the database or bucket it "+
				"persists to",
			memfileStorageEngine,
			storageEngine,
		)
	}

	return fmt.Errorf(
		"re-encrypting state is only supported for the %q storage engine, and %q is not a "+
			"storage engine this version of the deploy engine supports",
		memfileStorageEngine,
		storageEngine,
	)
}

// A dry run must leave the store exactly as it found it, including not creating a salt that a
// passphrase provider would otherwise persist on first use.
//
// A store with no salt yet has never been encrypted, so a salt generated here only has to
// stand in for one: there is nothing sealed for it to have to match.
func saltForPass(
	ctx context.Context,
	storage statestore.Storage,
	keys statestore.KeyBuilder,
	dryRun bool,
) ([]byte, error) {
	if !dryRun {
		return statestore.LoadOrCreateSalt(ctx, storage, keys)
	}

	exists, err := storage.Exists(ctx, keys.EncryptionSalt())
	if err != nil {
		return nil, fmt.Errorf("failed to check for an encryption salt: %w", err)
	}

	if exists {
		return statestore.LoadOrCreateSalt(ctx, storage, keys)
	}

	return crypto.GenerateSalt()
}

// The configured key is the target of a normal pass and the source of a decrypting one. Either
// way the key set opens what is already there, since a store part-way through a pass holds
// objects under both keys.
func reencryptKeys(
	ctx context.Context,
	encryptionConfig *core.StateEncryptionConfig,
	salt []byte,
	decrypt bool,
) (source crypto.KeyProvider, target crypto.KeyProvider, err error) {
	if !encryptionConfig.Enabled {
		if decrypt {
			return nil, nil, errors.New(
				"state.encryption.enabled is false, so there is no configured key to decrypt " +
					"the state with; configure the key the state was sealed with and run this again",
			)
		}

		return nil, nil, errors.New(
			"state.encryption.enabled is false, so there is no key to encrypt the state with",
		)
	}

	configured, err := createStateKeyProvider(ctx, encryptionConfig, salt)
	if err != nil {
		return nil, nil, err
	}

	if decrypt {
		return configured, nil, nil
	}

	return configured, configured, nil
}

// Copies every object to a separate directory, so an interrupted pass can be recovered from.
//
// A pass rewrites objects in place and nothing here can make that atomic: a failure part-way
// through leaves a store holding a mix of keys, readable only by a deployment configured with
// both.
func backupState(
	ctx context.Context,
	storage statestore.Storage,
	fileSystem afero.Fs,
	keys statestore.KeyBuilder,
	backupDir string,
) (string, error) {
	if backupDir == "" {
		backupDir = fmt.Sprintf(
			"%s.backup-%s",
			strings.TrimSuffix(keys.Prefix(), "/"),
			time.Now().UTC().Format("20060102T150405Z"),
		)
	}

	exists, err := afero.DirExists(fileSystem, backupDir)
	if err != nil {
		return "", fmt.Errorf("failed to check the backup directory %q: %w", backupDir, err)
	}
	if exists {
		return "", fmt.Errorf(
			"the backup directory %q already exists, choose another one rather than writing "+
				"over a copy that may be the only readable one",
			backupDir,
		)
	}

	objectKeys, err := storage.List(ctx, keys.Prefix())
	if err != nil {
		return "", fmt.Errorf("failed to list the state to copy: %w", err)
	}

	for _, key := range objectKeys {
		stored, err := storage.Read(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to read %q to copy it: %w", key, err)
		}

		relativeKey := strings.TrimPrefix(strings.TrimPrefix(key, keys.Prefix()), "/")
		if err := storage.Write(ctx, path.Join(backupDir, relativeKey), stored); err != nil {
			return "", fmt.Errorf("failed to copy %q: %w", key, err)
		}
	}

	return backupDir, nil
}

// A running engine would write its in-memory state back over a pass, so something answering
// on its address is treated as a reason to stop. Connecting is the only reliable signal: a
// unix socket file can be left behind by a process that has gone.
func refuseIfEngineIsRunning(config *core.Config, force bool) error {
	network, address := "tcp", fmt.Sprintf("127.0.0.1:%d", config.Port)
	if config.UseUnixSocket {
		network, address = "unix", config.UnixSocketPath
	}

	conn, err := net.DialTimeout(network, address, time.Second)
	if err != nil {
		return nil
	}
	conn.Close()

	if force {
		return nil
	}

	return fmt.Errorf(
		"something is listening on %s, which suggests the deploy engine is running; it holds "+
			"state in memory and would write it back over this pass, so stop it first",
		address,
	)
}

func writeLine(out io.Writer, line string) {
	if out == nil {
		return
	}

	fmt.Fprintln(out, line)
}
