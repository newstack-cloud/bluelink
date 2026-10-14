package awskms

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/stretchr/testify/suite"
)

type AWSKMSKeyProviderSuite struct {
	suite.Suite
}

func (s *AWSKMSKeyProviderSuite) provider(client Client) crypto.KeyProvider {
	provider, err := NewKeyProvider(client, "alias/bluelink-state")
	s.Require().NoError(err)

	return provider
}

func (s *AWSKMSKeyProviderSuite) Test_seals_and_opens_state_through_kms() {
	provider := s.provider(newmockKMS())

	sealed, err := crypto.Seal(
		context.Background(),
		provider,
		"instances_c0.json",
		[]byte(`{"specData":{"connectionString":"postgres://user:s3cr3t@host/db"}}`),
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
	s.Assert().Equal(
		`{"specData":{"connectionString":"postgres://user:s3cr3t@host/db"}}`,
		string(opened),
	)
}

// The plaintext data key must never be what lands in the envelope, which is
// the whole point of asking KMS to wrap it.
func (s *AWSKMSKeyProviderSuite) Test_the_wrapped_key_is_not_the_data_key() {
	mock := newmockKMS()
	provider := s.provider(mock)

	dek, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Require().Len(dek, crypto.DEKSize)
	s.Assert().NotEqual(dek, wrapped)
}

// The configured key id has to reach KMS on both legs: on decrypt it is what
// makes KMS refuse a blob sealed under another key rather than decrypting it.
func (s *AWSKMSKeyProviderSuite) Test_the_configured_key_is_named_on_every_call() {
	mock := newmockKMS()
	provider := s.provider(mock)

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	_, err = provider.UnwrapKey(context.Background(), provider.ID(), wrapped)
	s.Require().NoError(err)

	s.Assert().Equal([]string{"alias/bluelink-state"}, mock.generateKeyIDs)
	s.Assert().Equal([]string{"alias/bluelink-state"}, mock.decryptKeyIDs)
}

// An envelope sealed with a different KMS key must be refused here rather than
// sent to KMS, which would otherwise answer for a key the deployment is no
// longer configured with.
func (s *AWSKMSKeyProviderSuite) Test_an_envelope_from_another_key_is_refused() {
	mock := newmockKMS()
	provider := s.provider(mock)

	_, wrapped, err := provider.GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	_, err = provider.UnwrapKey(context.Background(), "awskms:alias/some-other-key", wrapped)
	s.Require().Error(err)
	s.Assert().Empty(mock.decryptKeyIDs, "the call reached kms anyway")
}

func (s *AWSKMSKeyProviderSuite) Test_rejects_a_data_key_of_the_wrong_length() {
	mock := newmockKMS()
	mock.plaintextLength = 16

	_, _, err := s.provider(mock).GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "expected 32")
}

func (s *AWSKMSKeyProviderSuite) Test_reports_a_kms_failure() {
	mock := newmockKMS()
	mock.err = errors.New("AccessDeniedException: not authorized to perform kms:GenerateDataKey")

	_, _, err := s.provider(mock).GenerateDataKey(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "kms:GenerateDataKey")
}

func (s *AWSKMSKeyProviderSuite) Test_requires_a_client_and_a_key_id() {
	_, err := NewKeyProvider(nil, "alias/bluelink-state")
	s.Require().Error(err)

	_, err = NewKeyProvider(newmockKMS(), "")
	s.Require().Error(err)
}

// Stands in for KMS, wrapping a data key by recording it against an opaque
// blob so that a round trip can be exercised without reaching AWS.
type mockKMS struct {
	keys            map[string][]byte
	generateKeyIDs  []string
	decryptKeyIDs   []string
	plaintextLength int
	err             error
}

func newmockKMS() *mockKMS {
	return &mockKMS{
		keys:            map[string][]byte{},
		plaintextLength: crypto.DEKSize,
	}
}

func (f *mockKMS) GenerateDataKey(
	ctx context.Context,
	params *kms.GenerateDataKeyInput,
	optFns ...func(*kms.Options),
) (*kms.GenerateDataKeyOutput, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.generateKeyIDs = append(f.generateKeyIDs, aws.ToString(params.KeyId))

	dek := make([]byte, f.plaintextLength)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}

	handle := make([]byte, 12)
	if _, err := rand.Read(handle); err != nil {
		return nil, err
	}
	blob := []byte(base64.StdEncoding.EncodeToString(handle))
	f.keys[string(blob)] = dek

	return &kms.GenerateDataKeyOutput{Plaintext: dek, CiphertextBlob: blob}, nil
}

func (f *mockKMS) Decrypt(
	ctx context.Context,
	params *kms.DecryptInput,
	optFns ...func(*kms.Options),
) (*kms.DecryptOutput, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.decryptKeyIDs = append(f.decryptKeyIDs, aws.ToString(params.KeyId))

	dek, ok := f.keys[string(params.CiphertextBlob)]
	if !ok {
		return nil, errors.New("InvalidCiphertextException")
	}

	return &kms.DecryptOutput{Plaintext: dek}, nil
}

func TestAWSKMSKeyProviderSuite(t *testing.T) {
	suite.Run(t, new(AWSKMSKeyProviderSuite))
}
