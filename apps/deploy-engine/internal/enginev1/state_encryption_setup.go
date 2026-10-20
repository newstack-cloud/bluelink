package enginev1

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/awskms"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/azurekeyvault"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/gcpkms"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"google.golang.org/api/option"
)

const (
	passphraseKeyProvider    = "passphrase"
	localKeyFileProvider     = "localkey"
	awsKMSKeyProvider        = "awskms"
	gcpKMSKeyProvider        = "gcpkms"
	azureKeyVaultKeyProvider = "azurekeyvault"
)

var supportedStateKeyProviders = []string{
	passphraseKeyProvider,
	localKeyFileProvider,
	awsKMSKeyProvider,
	gcpKMSKeyProvider,
	azureKeyVaultKeyProvider,
}

// Builds the provider state is encrypted with, as a key set when the deployment still lists
// keys that earlier state was written under.
//
// The key set is what keeps a key change from being a cliff: new objects are sealed with the
// current key while objects sealed with a previous one stay readable, which is also the state
// a store sits in part-way through a re-encryption pass.
func createStateKeyProvider(
	ctx context.Context,
	encryptionConfig *core.StateEncryptionConfig,
	salt []byte,
) (crypto.KeyProvider, error) {
	active, err := createKeyProviderForKey(ctx, &encryptionConfig.StateEncryptionKeyConfig, salt)
	if err != nil {
		return nil, err
	}

	if len(encryptionConfig.PreviousKeys) == 0 {
		return active, nil
	}

	previous := make([]crypto.KeyProvider, 0, len(encryptionConfig.PreviousKeys))
	for index := range encryptionConfig.PreviousKeys {
		provider, err := createKeyProviderForKey(ctx, &encryptionConfig.PreviousKeys[index], salt)
		if err != nil {
			return nil, fmt.Errorf("state.encryption.previous_keys[%d]: %w", index, err)
		}

		previous = append(previous, provider)
	}

	return crypto.NewKeySet(active, previous...)
}

// The salt is only meaningful to the passphrase provider; the others hold or reach a key that
// already exists.
func createKeyProviderForKey(
	ctx context.Context,
	encryptionConfig *core.StateEncryptionKeyConfig,
	salt []byte,
) (crypto.KeyProvider, error) {
	switch encryptionConfig.Provider {
	case passphraseKeyProvider:
		passphrase, err := statePassphrase(encryptionConfig)
		if err != nil {
			return nil, err
		}

		return crypto.NewPassphraseKeyProvider(passphrase, salt)
	case localKeyFileProvider:
		if encryptionConfig.LocalKeyFile == "" {
			return nil, requiredForProviderError(
				"state.encryption.local_key_file",
				localKeyFileProvider,
			)
		}

		return crypto.NewLocalFileKeyProvider(encryptionConfig.LocalKeyFile)
	case awsKMSKeyProvider:
		return createAWSKMSKeyProvider(ctx, &encryptionConfig.AWSKMS)
	case gcpKMSKeyProvider:
		return createGCPKMSKeyProvider(ctx, &encryptionConfig.GCPKMS)
	case azureKeyVaultKeyProvider:
		return createAzureKeyVaultKeyProvider(&encryptionConfig.AzureKeyVault)
	default:
		return nil, fmt.Errorf(
			"unsupported state encryption key provider %q, supported providers are: %s",
			encryptionConfig.Provider,
			strings.Join(supportedStateKeyProviders, ", "),
		)
	}
}

func statePassphrase(
	encryptionConfig *core.StateEncryptionKeyConfig,
) (string, error) {
	if encryptionConfig.PassphraseFile != "" {
		contents, err := os.ReadFile(encryptionConfig.PassphraseFile)
		if err != nil {
			return "", fmt.Errorf(
				"failed to read state.encryption.passphrase_file: %w",
				err,
			)
		}

		// Trailing whitespace is stripped so that a passphrase written to a
		// file by an editor or a shell redirect derives the same key as the
		// same passphrase supplied directly.
		passphrase := strings.TrimRight(string(contents), " \t\r\n")
		if passphrase == "" {
			return "", fmt.Errorf(
				"state.encryption.passphrase_file %q is empty",
				encryptionConfig.PassphraseFile,
			)
		}

		return passphrase, nil
	}

	if encryptionConfig.Passphrase == "" {
		return "", fmt.Errorf(
			"one of state.encryption.passphrase or state.encryption.passphrase_file "+
				"is required for the %q key provider",
			passphraseKeyProvider,
		)
	}

	return encryptionConfig.Passphrase, nil
}

func createAWSKMSKeyProvider(
	ctx context.Context,
	kmsConfig *core.StateEncryptionAWSKMSConfig,
) (crypto.KeyProvider, error) {
	if kmsConfig.KeyID == "" {
		return nil, requiredForProviderError(
			"state.encryption.awskms.key_id",
			awsKMSKeyProvider,
		)
	}

	awsOpts := tracedAWSOptions()
	if kmsConfig.Region != "" {
		awsOpts = append(awsOpts, awsconfig.WithRegion(kmsConfig.Region))
	}
	if kmsConfig.AccessKeyID != "" && kmsConfig.SecretAccessKey != "" {
		awsOpts = append(awsOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				kmsConfig.AccessKeyID,
				kmsConfig.SecretAccessKey,
				"",
			),
		))
	}

	conf, err := awsconfig.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config for state encryption: %w", err)
	}

	client := awskms.NewClient(conf, awskms.ClientOptions{
		Endpoint: kmsConfig.Endpoint,
	})

	return awskms.NewKeyProvider(client, kmsConfig.KeyID)
}

func createGCPKMSKeyProvider(
	ctx context.Context,
	kmsConfig *core.StateEncryptionGCPKMSConfig,
) (crypto.KeyProvider, error) {
	if kmsConfig.KeyName == "" {
		return nil, requiredForProviderError(
			"state.encryption.gcpkms.key_name",
			gcpKMSKeyProvider,
		)
	}

	clientOpts := []option.ClientOption{}
	if kmsConfig.CredentialsFile != "" {
		// The credential type is named rather than inferred from the file, so a file holding
		// some other kind of credential configuration is refused instead of being loaded.
		// option.WithCredentialsFile, which infers it, is deprecated for that reason.
		clientOpts = append(clientOpts, option.WithAuthCredentialsFile(
			option.ServiceAccount,
			kmsConfig.CredentialsFile,
		))
	}

	client, err := gcpkms.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud kms client: %w", err)
	}

	return gcpkms.NewKeyProvider(client, kmsConfig.KeyName)
}

func createAzureKeyVaultKeyProvider(
	vaultConfig *core.StateEncryptionAzureKeyVaultConfig,
) (crypto.KeyProvider, error) {
	if vaultConfig.VaultURL == "" {
		return nil, requiredForProviderError(
			"state.encryption.azurekeyvault.vault_url",
			azureKeyVaultKeyProvider,
		)
	}
	if vaultConfig.KeyName == "" {
		return nil, requiredForProviderError(
			"state.encryption.azurekeyvault.key_name",
			azureKeyVaultKeyProvider,
		)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve azure credentials for state encryption: %w", err)
	}

	client, err := azurekeyvault.NewClient(vaultConfig.VaultURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure key vault client: %w", err)
	}

	return azurekeyvault.NewKeyProvider(client, vaultConfig.KeyName, azurekeyvault.Options{
		KeyVersion: vaultConfig.KeyVersion,
		Algorithm:  azkeys.EncryptionAlgorithm(vaultConfig.Algorithm),
	})
}

func requiredForProviderError(configKey string, keyProvider string) error {
	return fmt.Errorf(
		"%s is required for the %q state encryption key provider",
		configKey,
		keyProvider,
	)
}
