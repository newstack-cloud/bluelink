package enginev1

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/stretchr/testify/suite"
)

type SetupStateEncryptionTestSuite struct {
	suite.Suite
}

// Runs Setup with the given state configuration, which is as close as the
// engine gets to a seam for the config that selects a key provider.
func (s *SetupStateEncryptionTestSuite) setup(
	configure func(stateConfig *core.StateConfig),
) error {
	config, err := core.LoadConfig()
	s.Require().NoError(err)

	config.State.StorageEngine = memfileStorageEngine
	config.State.MemFileStateDir = s.T().TempDir()
	configure(&config.State)

	// Port 0 so the plugin service listener does not collide with another
	// test or a running engine.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer listener.Close()

	_, cleanup, setupErr := Setup(mux.NewRouter(), &config, listener)
	if cleanup != nil {
		s.T().Cleanup(cleanup)
	}

	return setupErr
}

// A salt written at startup is what makes the passphrase provider reproducible
// across restarts, so it has to be created before anything is persisted rather
// than on the first write.
func (s *SetupStateEncryptionTestSuite) Test_a_passphrase_provider_persists_its_salt_at_startup() {
	var stateDir string

	err := s.setup(func(stateConfig *core.StateConfig) {
		stateDir = stateConfig.MemFileStateDir
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = passphraseKeyProvider
		stateConfig.Encryption.Passphrase = "deployment-passphrase"
	})
	s.Require().NoError(err)

	saltKey := statestore.NewKeyBuilder(stateDir).EncryptionSalt()
	s.Assert().FileExists(saltKey)
}

func (s *SetupStateEncryptionTestSuite) Test_reads_a_passphrase_from_a_file() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = passphraseKeyProvider
		stateConfig.Encryption.PassphraseFile = s.writePassphraseFile("deployment-passphrase\n")
	})
	s.Assert().NoError(err)
}

func (s *SetupStateEncryptionTestSuite) Test_rejects_an_empty_passphrase_file() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = passphraseKeyProvider
		stateConfig.Encryption.PassphraseFile = s.writePassphraseFile("\n")
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "is empty")
}

func (s *SetupStateEncryptionTestSuite) Test_rejects_a_missing_passphrase_file() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = passphraseKeyProvider
		stateConfig.Encryption.PassphraseFile = filepath.Join(s.T().TempDir(), "absent")
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "passphrase_file")
}

// Starting with encryption on and no key at all must fail rather than fall
// back to writing plaintext, which would look like it had worked.
func (s *SetupStateEncryptionTestSuite) Test_rejects_encryption_with_no_passphrase_configured() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = passphraseKeyProvider
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "state.encryption.passphrase")
}

func (s *SetupStateEncryptionTestSuite) Test_rejects_an_unsupported_key_provider() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = "vault"
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "supported providers are")
	for _, provider := range supportedStateKeyProviders {
		s.Assert().Contains(err.Error(), provider)
	}
}

// Each provider needs the one thing that names its key. Without it the engine
// would otherwise reach a cloud API with an empty key id and report whatever
// that service says.
func (s *SetupStateEncryptionTestSuite) Test_each_key_provider_requires_the_key_it_wraps_with() {
	providerConfigKeys := map[string]string{
		localKeyFileProvider:     "state.encryption.local_key_file",
		awsKMSKeyProvider:        "state.encryption.awskms.key_id",
		gcpKMSKeyProvider:        "state.encryption.gcpkms.key_name",
		azureKeyVaultKeyProvider: "state.encryption.azurekeyvault.vault_url",
	}

	for provider, configKey := range providerConfigKeys {
		err := s.setup(func(stateConfig *core.StateConfig) {
			stateConfig.Encryption.Enabled = true
			stateConfig.Encryption.Provider = provider
		})
		s.Require().Error(err, "%q started with no key configured", provider)
		s.Assert().Contains(err.Error(), configKey)
	}
}

func (s *SetupStateEncryptionTestSuite) Test_requires_a_key_name_for_azure_key_vault() {
	err := s.setup(func(stateConfig *core.StateConfig) {
		stateConfig.Encryption.Enabled = true
		stateConfig.Encryption.Provider = azureKeyVaultKeyProvider
		stateConfig.Encryption.AzureKeyVault.VaultURL = "https://vault.vault.azure.net/"
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "state.encryption.azurekeyvault.key_name")
}

// The decorator only sits in front of the memfile storage, so asking for
// encryption with another engine has to say so rather than start and persist
// plaintext to a database or bucket.
func (s *SetupStateEncryptionTestSuite) Test_rejects_encryption_for_other_storage_engines() {
	for _, storageEngine := range []string{postgresStorageEngine, objectstoreStorageEngine} {
		err := s.setup(func(stateConfig *core.StateConfig) {
			stateConfig.StorageEngine = storageEngine
			stateConfig.Encryption.Enabled = true
			stateConfig.Encryption.Provider = passphraseKeyProvider
			stateConfig.Encryption.Passphrase = "deployment-passphrase"
		})
		s.Require().Error(err, "%q started with state encryption enabled", storageEngine)
		s.Assert().Contains(err.Error(), "only supported for the \"memfile\" storage engine")
	}
}

func (s *SetupStateEncryptionTestSuite) writePassphraseFile(contents string) string {
	path := filepath.Join(s.T().TempDir(), "passphrase")
	s.Require().NoError(os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestSetupStateEncryptionTestSuite(t *testing.T) {
	suite.Run(t, new(SetupStateEncryptionTestSuite))
}
