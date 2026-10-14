package memfile

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const encryptedStateDir = "encrypted-state"

type MemFileEncryptedStateContainerTestSuite struct {
	suite.Suite
}

// Loads a container the way the deploy engine does when state encryption is
// configured: the salt read through the undecorated storage, then the
// container loaded over the decorated one.
func (s *MemFileEncryptedStateContainerTestSuite) loadEncryptedContainer(
	fs afero.Fs,
	passphrase string,
) (*StateContainer, error) {
	storage := NewFSStorage(fs)

	salt, err := statestore.LoadOrCreateSalt(
		context.Background(),
		storage,
		statestore.NewKeyBuilder(encryptedStateDir),
	)
	s.Require().NoError(err)

	provider, err := crypto.NewPassphraseKeyProvider(passphrase, salt)
	s.Require().NoError(err)

	return LoadStateContainerWithStorage(
		encryptedStateDir,
		statestore.NewEncryptedStorage(storage, provider),
		core.NewNopLogger(),
	)
}

func (s *MemFileEncryptedStateContainerTestSuite) instanceWithSecret() state.InstanceState {
	return state.InstanceState{
		InstanceID:   "0c1a1b2c-3d4e-5f60-7182-93a4b5c6d7e8",
		InstanceName: "orders-api",
		Status:       core.InstanceStatusDeployed,
		Metadata: map[string]*core.MappingNode{
			"connectionString": core.MappingNodeFromString(
				"postgres://user:s3cr3t@host/db",
			),
		},
	}
}

// State written by one process must be readable by the next, which is the
// whole point of persisting the salt alongside the state.
func (s *MemFileEncryptedStateContainerTestSuite) Test_state_written_in_one_run_is_readable_in_the_next() {
	fs := afero.NewMemMapFs()
	instance := s.instanceWithSecret()

	container, err := s.loadEncryptedContainer(fs, "deployment-passphrase")
	s.Require().NoError(err)
	s.Require().NoError(container.Instances().Save(context.Background(), instance))

	reloaded, err := s.loadEncryptedContainer(fs, "deployment-passphrase")
	s.Require().NoError(err)

	savedState, err := reloaded.Instances().Get(context.Background(), instance.InstanceID)
	s.Require().NoError(err)
	s.Assert().Equal(
		"postgres://user:s3cr3t@host/db",
		core.StringValue(savedState.Metadata["connectionString"]),
	)
}

// What lands on disk must not be the plaintext, for every file the container
// writes rather than only the instance chunk.
func (s *MemFileEncryptedStateContainerTestSuite) Test_nothing_on_disk_holds_the_plaintext() {
	fs := afero.NewMemMapFs()

	container, err := s.loadEncryptedContainer(fs, "deployment-passphrase")
	s.Require().NoError(err)
	s.Require().NoError(
		container.Instances().Save(context.Background(), s.instanceWithSecret()),
	)

	written := s.listFiles(fs, encryptedStateDir)
	s.Require().NotEmpty(written, "the container wrote nothing to inspect")

	saltKey := statestore.NewKeyBuilder(encryptedStateDir).EncryptionSalt()
	for _, file := range written {
		contents, err := afero.ReadFile(fs, file)
		s.Require().NoError(err)
		s.Assert().NotContains(string(contents), "s3cr3t", "%s holds the plaintext", file)

		if file == saltKey {
			continue
		}

		s.Assert().True(crypto.IsEnvelope(contents), "%s is not an envelope", file)
	}
}

// An existing plaintext store cannot be silently adopted: the engine has to
// say why rather than report the state as corrupt.
func (s *MemFileEncryptedStateContainerTestSuite) Test_enabling_encryption_over_plaintext_state_reports_why() {
	fs := afero.NewMemMapFs()

	plaintext, err := LoadStateContainer(encryptedStateDir, fs, core.NewNopLogger())
	s.Require().NoError(err)
	s.Require().NoError(
		plaintext.Instances().Save(context.Background(), s.instanceWithSecret()),
	)

	_, err = s.loadEncryptedContainer(fs, "deployment-passphrase")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "is not encrypted")
}

// A passphrase change is a key change, so the state must fail to open rather
// than come back empty, which would look like a fresh store and be deployed
// over.
func (s *MemFileEncryptedStateContainerTestSuite) Test_a_different_passphrase_cannot_open_the_state() {
	fs := afero.NewMemMapFs()

	container, err := s.loadEncryptedContainer(fs, "deployment-passphrase")
	s.Require().NoError(err)
	s.Require().NoError(
		container.Instances().Save(context.Background(), s.instanceWithSecret()),
	)

	_, err = s.loadEncryptedContainer(fs, "a-different-passphrase")
	s.Require().Error(err)
}

func (s *MemFileEncryptedStateContainerTestSuite) listFiles(fs afero.Fs, dir string) []string {
	entries, err := afero.ReadDir(fs, dir)
	s.Require().NoError(err)

	files := []string{}
	for _, entry := range entries {
		entryPath := path.Join(dir, entry.Name())
		if entry.IsDir() {
			files = append(files, s.listFiles(fs, entryPath)...)
			continue
		}

		if strings.HasSuffix(entry.Name(), ".json") {
			files = append(files, entryPath)
		}
	}

	return files
}

func TestMemFileEncryptedStateContainerTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileEncryptedStateContainerTestSuite))
}
