package azurekeyvault

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/stretchr/testify/suite"
)

type AzureKeyVaultKeyProviderSuite struct {
	suite.Suite
}

func (s *AzureKeyVaultKeyProviderSuite) provider(client Client, opts Options) crypto.KeyProvider {
	provider, err := NewKeyProvider(client, "bluelink-state", opts)
	s.Require().NoError(err)

	return provider
}

func (s *AzureKeyVaultKeyProviderSuite) Test_seals_and_opens_state_through_the_vault() {
	provider := s.provider(newmockVault(), Options{})

	sealed, err := crypto.Seal(
		context.Background(),
		provider,
		"instances_c0.json",
		[]byte(`{"specData":{"token":"s3cr3t"}}`),
		[]byte("instances_c0.json"),
	)
	s.Require().NoError(err)
	s.Require().NotContains(string(sealed), "s3cr3t")

	opened, err := crypto.Open(
		context.Background(),
		provider,
		sealed,
		[]byte("instances_c0.json"),
	)
	s.Require().NoError(err)
	s.Assert().Equal(`{"specData":{"token":"s3cr3t"}}`, string(opened))
}

func (s *AzureKeyVaultKeyProviderSuite) Test_the_wrapped_key_is_not_the_data_key() {
	dek, wrapped, err := s.provider(newmockVault(), Options{}).
		GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Require().Len(dek, crypto.DEKSize)
	s.Assert().NotEqual(dek, wrapped)
}

func (s *AzureKeyVaultKeyProviderSuite) Test_wraps_with_rsa_oaep_256_by_default() {
	mock := newmockVault()

	_, _, err := s.provider(mock, Options{}).
		GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	s.Assert().Equal(
		[]azkeys.EncryptionAlgorithm{azkeys.EncryptionAlgorithmRSAOAEP256},
		mock.wrapAlgorithms,
	)
}

// A Managed HSM deployment holding a symmetric key needs a key-wrap algorithm
// instead, so the configured one has to reach the vault.
func (s *AzureKeyVaultKeyProviderSuite) Test_wraps_with_a_configured_algorithm() {
	mock := newmockVault()

	_, _, err := s.provider(mock, Options{Algorithm: azkeys.EncryptionAlgorithmA256KW}).
		GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	s.Assert().Equal(
		[]azkeys.EncryptionAlgorithm{azkeys.EncryptionAlgorithmA256KW},
		mock.wrapAlgorithms,
	)
}

// A wrapped key does not name the version that produced it, so the version
// travels in the envelope. Rotating the configured version must leave state
// sealed under the previous one readable, the vault keeping old versions
// available for unwrapping.
func (s *AzureKeyVaultKeyProviderSuite) Test_state_sealed_before_a_rotation_stays_readable() {
	mock := newmockVault()

	beforeRotation := s.provider(mock, Options{KeyVersion: "v1"})
	sealed, err := crypto.Seal(
		context.Background(),
		beforeRotation,
		"instances_c0.json",
		[]byte("payload"),
		nil,
	)
	s.Require().NoError(err)

	afterRotation := s.provider(mock, Options{KeyVersion: "v2"})
	opened, err := crypto.Open(context.Background(), afterRotation, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)

	s.Assert().Equal([]string{"v1"}, mock.wrapVersions)
	s.Assert().Equal([]string{"v1"}, mock.unwrapVersions, "unwrapped with the wrong version")
}

func (s *AzureKeyVaultKeyProviderSuite) Test_an_envelope_from_another_key_is_refused() {
	mock := newmockVault()
	provider := s.provider(mock, Options{KeyVersion: "v1"})

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	for _, kekID := range []string{
		"azurekeyvault:another-key/v1",
		"awskms:alias/bluelink-state",
		"bluelink-state/v1",
	} {
		_, err = provider.UnwrapKey(context.Background(), kekID, wrapped)
		s.Require().Error(err, "%q was accepted", kekID)
	}

	s.Assert().Empty(mock.unwrapVersions, "a call reached the vault anyway")
}

func (s *AzureKeyVaultKeyProviderSuite) Test_rejects_a_data_key_of_the_wrong_length() {
	mock := newmockVault()
	mock.unwrapLength = 16
	provider := s.provider(mock, Options{})

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	_, err = provider.UnwrapKey(context.Background(), provider.ID(), wrapped)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "expected 32")
}

func (s *AzureKeyVaultKeyProviderSuite) Test_reports_a_vault_failure() {
	mock := newmockVault()
	mock.err = errors.New("Forbidden: caller does not have keys/wrapKey permission")

	_, _, err := s.provider(mock, Options{}).
		GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "wrapKey permission")
}

func (s *AzureKeyVaultKeyProviderSuite) Test_requires_a_client_and_a_key_name() {
	_, err := NewKeyProvider(nil, "bluelink-state", Options{})
	s.Require().Error(err)

	_, err = NewKeyProvider(newmockVault(), "", Options{})
	s.Require().Error(err)
}

// Stands in for Key Vault, recording each data key against an opaque wrapped
// form so a round trip can be exercised without reaching a vault.
type mockVault struct {
	keys           map[string][]byte
	wrapAlgorithms []azkeys.EncryptionAlgorithm
	wrapVersions   []string
	unwrapVersions []string
	unwrapLength   int
	err            error
}

func newmockVault() *mockVault {
	return &mockVault{keys: map[string][]byte{}, unwrapLength: crypto.DEKSize}
}

func (f *mockVault) WrapKey(
	ctx context.Context,
	name string,
	version string,
	parameters azkeys.KeyOperationParameters,
	options *azkeys.WrapKeyOptions,
) (azkeys.WrapKeyResponse, error) {
	if f.err != nil {
		return azkeys.WrapKeyResponse{}, f.err
	}

	f.wrapAlgorithms = append(f.wrapAlgorithms, *parameters.Algorithm)
	f.wrapVersions = append(f.wrapVersions, version)

	handle := make([]byte, 12)
	if _, err := rand.Read(handle); err != nil {
		return azkeys.WrapKeyResponse{}, err
	}
	wrapped := []byte(base64.StdEncoding.EncodeToString(handle))
	f.keys[string(wrapped)] = parameters.Value

	return azkeys.WrapKeyResponse{
		KeyOperationResult: azkeys.KeyOperationResult{Result: wrapped},
	}, nil
}

func (f *mockVault) UnwrapKey(
	ctx context.Context,
	name string,
	version string,
	parameters azkeys.KeyOperationParameters,
	options *azkeys.UnwrapKeyOptions,
) (azkeys.UnwrapKeyResponse, error) {
	if f.err != nil {
		return azkeys.UnwrapKeyResponse{}, f.err
	}

	f.unwrapVersions = append(f.unwrapVersions, version)

	dek, ok := f.keys[string(parameters.Value)]
	if !ok {
		return azkeys.UnwrapKeyResponse{}, errors.New("BadParameter: the wrapped key is not valid")
	}

	return azkeys.UnwrapKeyResponse{
		KeyOperationResult: azkeys.KeyOperationResult{Result: dek[:f.unwrapLength]},
	}, nil
}

func TestAzureKeyVaultKeyProviderSuite(t *testing.T) {
	suite.Run(t, new(AzureKeyVaultKeyProviderSuite))
}
