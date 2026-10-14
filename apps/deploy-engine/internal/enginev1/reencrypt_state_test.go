package enginev1

import (
	"bytes"
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const reencryptStateDir = "state"

type ReencryptStateTestSuite struct {
	suite.Suite
	fs afero.Fs
}

func (s *ReencryptStateTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
}

// Config for a deployment whose current key is the given passphrase, with any number of
// previous passphrases still listed so earlier state stays readable.
func (s *ReencryptStateTestSuite) config(
	enabled bool,
	passphrase string,
	previousPassphrases ...string,
) *core.Config {
	previousKeys := make([]core.StateEncryptionKeyConfig, 0, len(previousPassphrases))
	for _, previous := range previousPassphrases {
		previousKeys = append(previousKeys, core.StateEncryptionKeyConfig{
			Provider:   passphraseKeyProvider,
			Passphrase: previous,
		})
	}

	return &core.Config{
		// Port 0 so the running-engine check finds nothing listening.
		Port: 0,
		State: core.StateConfig{
			StorageEngine:   memfileStorageEngine,
			MemFileStateDir: reencryptStateDir,
			Encryption: core.StateEncryptionConfig{
				Enabled: enabled,
				StateEncryptionKeyConfig: core.StateEncryptionKeyConfig{
					Provider:   passphraseKeyProvider,
					Passphrase: passphrase,
				},
				PreviousKeys: previousKeys,
			},
		},
	}
}

func (s *ReencryptStateTestSuite) instance() state.InstanceState {
	return state.InstanceState{
		InstanceID:   "0c1a1b2c-3d4e-5f60-7182-93a4b5c6d7e8",
		InstanceName: "orders-api",
		Status:       bpcore.InstanceStatusDeployed,
		Metadata: map[string]*bpcore.MappingNode{
			"connectionString": bpcore.MappingNodeFromString("postgres://user:s3cr3t@host/db"),
		},
	}
}

// Writes an instance through a container, encrypted with the given passphrase or plaintext
// when it is empty, which is what a deployment's state directory looks like before a pass.
func (s *ReencryptStateTestSuite) writeExistingState(passphrase string) {
	storage := memfile.NewFSStorage(s.fs)

	if passphrase != "" {
		salt, err := statestore.LoadOrCreateSalt(
			context.Background(),
			storage,
			statestore.NewKeyBuilder(reencryptStateDir),
		)
		s.Require().NoError(err)

		provider, err := crypto.NewPassphraseKeyProvider(passphrase, salt)
		s.Require().NoError(err)
		storage = statestore.NewEncryptedStorage(storage, provider)
	}

	container, err := memfile.LoadStateContainerWithStorage(
		reencryptStateDir,
		storage,
		bpcore.NewNopLogger(),
	)
	s.Require().NoError(err)
	s.Require().NoError(container.Instances().Save(context.Background(), s.instance()))
}

// Loads the state the way the engine would with the given config, which is the only check
// that matters after a pass: the engine can read what the tool left behind.
func (s *ReencryptStateTestSuite) readStateAsEngineWould(config *core.Config) (state.InstanceState, error) {
	services, _, err := loadStateServices(
		context.Background(),
		s.fs,
		bpcore.NewNopLogger(),
		&config.State,
	)
	if err != nil {
		return state.InstanceState{}, err
	}

	return services.container.Instances().Get(
		context.Background(),
		s.instance().InstanceID,
	)
}

func (s *ReencryptStateTestSuite) reencrypt(
	config *core.Config,
	opts ReencryptStateOptions,
) (*statestore.ReencryptReport, error) {
	return ReencryptState(context.Background(), config, s.fs, opts)
}

// Encrypting a store that was written before encryption was enabled, which is otherwise a
// startup failure with no way forward.
func (s *ReencryptStateTestSuite) Test_encrypts_a_state_directory_written_in_plaintext() {
	s.writeExistingState("")
	config := s.config(true, "deployment-passphrase")

	_, err := s.readStateAsEngineWould(config)
	s.Require().Error(err, "the engine read plaintext state while configured to encrypt")

	report, err := s.reencrypt(config, ReencryptStateOptions{})
	s.Require().NoError(err)
	s.Require().Positive(report.Encrypted)

	instanceState, err := s.readStateAsEngineWould(config)
	s.Require().NoError(err)
	s.Assert().Equal(
		"postgres://user:s3cr3t@host/db",
		bpcore.StringValue(instanceState.Metadata["connectionString"]),
	)
}

// A key change: the engine runs with the new key and the old one still listed, the pass moves
// the state, and afterwards the old key can be dropped from the configuration.
func (s *ReencryptStateTestSuite) Test_moves_state_to_a_new_key_so_the_old_one_can_be_dropped() {
	s.writeExistingState("old-passphrase")
	duringRotation := s.config(true, "new-passphrase", "old-passphrase")

	report, err := s.reencrypt(duringRotation, ReencryptStateOptions{})
	s.Require().NoError(err)
	s.Require().Positive(report.Reencrypted)

	afterRotation := s.config(true, "new-passphrase")
	instanceState, err := s.readStateAsEngineWould(afterRotation)
	s.Require().NoError(err)
	s.Assert().Equal(
		"postgres://user:s3cr3t@host/db",
		bpcore.StringValue(instanceState.Metadata["connectionString"]),
	)
}

// Without the old key listed there is nothing to open the existing state with, and saying so
// beats a pass that writes over it.
func (s *ReencryptStateTestSuite) Test_refuses_a_pass_that_cannot_open_the_existing_state() {
	s.writeExistingState("old-passphrase")

	_, err := s.reencrypt(s.config(true, "new-passphrase"), ReencryptStateOptions{})
	s.Require().Error(err)
}

func (s *ReencryptStateTestSuite) Test_decrypts_state_back_to_plaintext() {
	s.writeExistingState("deployment-passphrase")
	config := s.config(true, "deployment-passphrase")

	report, err := s.reencrypt(config, ReencryptStateOptions{Decrypt: true})
	s.Require().NoError(err)
	s.Require().Positive(report.Decrypted)

	// Readable again by a deployment that has turned encryption off.
	instanceState, err := s.readStateAsEngineWould(s.config(false, ""))
	s.Require().NoError(err)
	s.Assert().Equal(
		"postgres://user:s3cr3t@host/db",
		bpcore.StringValue(instanceState.Metadata["connectionString"]),
	)
}

// A pass rewrites objects in place, so the copy is the only way back from one that fails
// part-way through.
func (s *ReencryptStateTestSuite) Test_copies_the_state_before_rewriting_it() {
	s.writeExistingState("old-passphrase")
	backupDir := "state-backup"

	_, err := s.reencrypt(
		s.config(true, "new-passphrase", "old-passphrase"),
		ReencryptStateOptions{BackupDir: backupDir},
	)
	s.Require().NoError(err)

	// The copy holds what was there before the pass: objects under the old key.
	copied, err := afero.ReadFile(s.fs, path.Join(backupDir, "instances_c0.json"))
	s.Require().NoError(err)
	s.Require().True(crypto.IsEnvelope(copied))

	salt, err := statestore.LoadOrCreateSalt(
		context.Background(),
		memfile.NewFSStorage(s.fs),
		statestore.NewKeyBuilder(reencryptStateDir),
	)
	s.Require().NoError(err)
	oldKey, err := crypto.NewPassphraseKeyProvider("old-passphrase", salt)
	s.Require().NoError(err)

	keyID, err := crypto.EnvelopeKeyID(copied)
	s.Require().NoError(err)
	s.Assert().Equal(oldKey.ID(), keyID)
}

// Writing over an existing copy could destroy the only readable version of the state.
func (s *ReencryptStateTestSuite) Test_refuses_to_write_over_an_existing_backup_directory() {
	s.writeExistingState("old-passphrase")
	s.Require().NoError(s.fs.MkdirAll("state-backup", 0o700))

	_, err := s.reencrypt(
		s.config(true, "new-passphrase", "old-passphrase"),
		ReencryptStateOptions{BackupDir: "state-backup"},
	)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already exists")
}

func (s *ReencryptStateTestSuite) Test_a_dry_run_writes_neither_state_nor_backup() {
	s.writeExistingState("old-passphrase")
	config := s.config(true, "new-passphrase", "old-passphrase")

	before, err := afero.ReadFile(s.fs, path.Join(reencryptStateDir, "instances_c0.json"))
	s.Require().NoError(err)

	output := &bytes.Buffer{}
	report, err := s.reencrypt(config, ReencryptStateOptions{
		DryRun:    true,
		BackupDir: "state-backup",
		Out:       output,
	})
	s.Require().NoError(err)
	s.Assert().Positive(report.Reencrypted, "a dry run should report what it would do")

	after, err := afero.ReadFile(s.fs, path.Join(reencryptStateDir, "instances_c0.json"))
	s.Require().NoError(err)
	s.Assert().Equal(before, after)

	backupExists, err := afero.DirExists(s.fs, "state-backup")
	s.Require().NoError(err)
	s.Assert().False(backupExists)
}

// A dry run against a store that has never been encrypted must not leave a salt behind
// either, since that is a write to the store it was asked not to touch.
func (s *ReencryptStateTestSuite) Test_a_dry_run_over_plaintext_state_creates_no_salt() {
	s.writeExistingState("")
	saltKey := statestore.NewKeyBuilder(reencryptStateDir).EncryptionSalt()

	_, err := s.reencrypt(s.config(true, "deployment-passphrase"), ReencryptStateOptions{
		DryRun: true,
	})
	s.Require().NoError(err)

	saltExists, err := afero.Exists(s.fs, saltKey)
	s.Require().NoError(err)
	s.Assert().False(saltExists, "a dry run wrote an encryption salt")
}

func (s *ReencryptStateTestSuite) Test_reports_each_object_it_deals_with() {
	s.writeExistingState("old-passphrase")

	output := &bytes.Buffer{}
	_, err := s.reencrypt(
		s.config(true, "new-passphrase", "old-passphrase"),
		ReencryptStateOptions{Out: output, SkipBackup: true},
	)
	s.Require().NoError(err)

	s.Assert().Contains(output.String(), "reencrypted state/instances_c0.json")
	s.Assert().Contains(output.String(), "skipped state/encryption_salt.json")
}

func (s *ReencryptStateTestSuite) Test_requires_a_configured_key() {
	s.writeExistingState("")

	_, err := s.reencrypt(s.config(false, ""), ReencryptStateOptions{})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "state.encryption.enabled is false")
}

func (s *ReencryptStateTestSuite) Test_is_only_supported_for_the_memfile_storage_engine() {
	for _, storageEngine := range []string{postgresStorageEngine, objectstoreStorageEngine} {
		config := s.config(true, "deployment-passphrase")
		config.State.StorageEngine = storageEngine

		_, err := s.reencrypt(config, ReencryptStateOptions{})
		s.Require().Error(err, "%q was accepted", storageEngine)
		s.Assert().Contains(err.Error(), "belongs to the database or bucket")
	}
}

// An engine name that is not one of the supported ones must not be told that its encryption
// belongs to a database or bucket it does not have.
func (s *ReencryptStateTestSuite) Test_reports_an_unsupported_storage_engine_as_unsupported() {
	config := s.config(true, "deployment-passphrase")
	config.State.StorageEngine = "sqlite"

	_, err := s.reencrypt(config, ReencryptStateOptions{})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not a storage engine this version")
}

func TestReencryptStateTestSuite(t *testing.T) {
	suite.Run(t, new(ReencryptStateTestSuite))
}
