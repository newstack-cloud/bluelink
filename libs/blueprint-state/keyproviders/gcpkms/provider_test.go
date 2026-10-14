package gcpkms

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"hash/crc32"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const testKeyName = "projects/p/locations/l/keyRings/r/cryptoKeys/bluelink-state"

type GCPKMSKeyProviderSuite struct {
	suite.Suite
}

func (s *GCPKMSKeyProviderSuite) provider(client Client) crypto.KeyProvider {
	provider, err := NewKeyProvider(client, testKeyName)
	s.Require().NoError(err)

	return provider
}

func (s *GCPKMSKeyProviderSuite) Test_seals_and_opens_state_through_kms() {
	provider := s.provider(newMockKMS())

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

func (s *GCPKMSKeyProviderSuite) Test_the_wrapped_key_is_not_the_data_key() {
	dek, wrapped, err := s.provider(newMockKMS()).
		GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Require().Len(dek, crypto.DEKSize)
	s.Assert().NotEqual(dek, wrapped)
}

func (s *GCPKMSKeyProviderSuite) Test_the_configured_key_is_named_on_every_call() {
	mock := newMockKMS()
	provider := s.provider(mock)

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	_, err = provider.UnwrapKey(context.Background(), provider.ID(), wrapped)
	s.Require().NoError(err)

	s.Assert().Equal([]string{testKeyName}, mock.encryptNames)
	s.Assert().Equal([]string{testKeyName}, mock.decryptNames)
}

func (s *GCPKMSKeyProviderSuite) Test_an_envelope_from_another_key_is_refused() {
	mock := newMockKMS()
	provider := s.provider(mock)

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	_, err = provider.UnwrapKey(
		context.Background(),
		"gcpkms:projects/p/locations/l/keyRings/r/cryptoKeys/another-key",
		wrapped,
	)
	s.Require().Error(err)
	s.Assert().Empty(mock.decryptNames, "the call reached kms anyway")
}

// A wrapped key corrupted in transit would otherwise be persisted and only
// surface later as state that cannot be decrypted, so the checksum KMS
// returns has to be checked rather than trusted.
func (s *GCPKMSKeyProviderSuite) Test_a_corrupted_wrapped_key_is_rejected() {
	mock := newMockKMS()
	mock.corruptCiphertextChecksum = true

	_, _, err := s.provider(mock).GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "corrupted in transit")
}

// A request corrupted on the way to KMS would wrap a different key than the
// one used to encrypt, which KMS reports by not verifying the checksum.
func (s *GCPKMSKeyProviderSuite) Test_an_unverified_request_checksum_is_rejected() {
	mock := newMockKMS()
	mock.withholdPlaintextVerification = true

	_, _, err := s.provider(mock).GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "corrupted in transit")
}

func (s *GCPKMSKeyProviderSuite) Test_a_corrupted_data_key_response_is_rejected() {
	mock := newMockKMS()
	provider := s.provider(mock)

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	mock.corruptPlaintextChecksum = true

	_, err = provider.UnwrapKey(context.Background(), provider.ID(), wrapped)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "corrupted in transit")
}

func (s *GCPKMSKeyProviderSuite) Test_reports_a_kms_failure() {
	mock := newMockKMS()
	mock.err = errors.New("PermissionDenied: cloudkms.cryptoKeyVersions.useToEncrypt denied")

	_, _, err := s.provider(mock).GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "useToEncrypt")
}

func (s *GCPKMSKeyProviderSuite) Test_requires_a_client_and_a_key_name() {
	_, err := NewKeyProvider(nil, testKeyName)
	s.Require().Error(err)

	_, err = NewKeyProvider(newMockKMS(), "")
	s.Require().Error(err)
}

// Stands in for Cloud KMS, recording each data key against an opaque
// ciphertext so a round trip can be exercised without reaching Google Cloud.
type mockKMS struct {
	keys         map[string][]byte
	encryptNames []string
	decryptNames []string

	corruptCiphertextChecksum     bool
	corruptPlaintextChecksum      bool
	withholdPlaintextVerification bool
	err                           error
}

func newMockKMS() *mockKMS {
	return &mockKMS{keys: map[string][]byte{}}
}

func (f *mockKMS) Encrypt(
	ctx context.Context,
	req *kmspb.EncryptRequest,
	opts ...gax.CallOption,
) (*kmspb.EncryptResponse, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.encryptNames = append(f.encryptNames, req.Name)

	handle := make([]byte, 12)
	if _, err := rand.Read(handle); err != nil {
		return nil, err
	}
	ciphertext := []byte(base64.StdEncoding.EncodeToString(handle))
	f.keys[string(ciphertext)] = req.Plaintext

	response := &kmspb.EncryptResponse{
		Ciphertext:              ciphertext,
		CiphertextCrc32C:        checksumOf(ciphertext),
		VerifiedPlaintextCrc32C: !f.withholdPlaintextVerification,
	}
	if f.corruptCiphertextChecksum {
		response.CiphertextCrc32C = wrapperspb.Int64(0)
	}

	return response, nil
}

func (f *mockKMS) Decrypt(
	ctx context.Context,
	req *kmspb.DecryptRequest,
	opts ...gax.CallOption,
) (*kmspb.DecryptResponse, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.decryptNames = append(f.decryptNames, req.Name)

	dek, ok := f.keys[string(req.Ciphertext)]
	if !ok {
		return nil, errors.New("InvalidArgument: decryption failed")
	}

	response := &kmspb.DecryptResponse{
		Plaintext:       dek,
		PlaintextCrc32C: checksumOf(dek),
	}
	if f.corruptPlaintextChecksum {
		response.PlaintextCrc32C = wrapperspb.Int64(0)
	}

	return response, nil
}

func checksumOf(data []byte) *wrapperspb.Int64Value {
	return wrapperspb.Int64(
		int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))),
	)
}

func TestGCPKMSKeyProviderSuite(t *testing.T) {
	suite.Run(t, new(GCPKMSKeyProviderSuite))
}
