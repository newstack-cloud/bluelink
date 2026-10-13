package crypto

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CryptoSuite struct {
	suite.Suite
}

func (s *CryptoSuite) passphraseProvider() KeyProvider {
	salt, err := GenerateSalt()
	s.Require().NoError(err)

	provider, err := NewPassphraseKeyProvider("correct horse battery staple", salt)
	s.Require().NoError(err)

	return provider
}

func (s *CryptoSuite) Test_seals_and_opens_a_payload() {
	provider := s.passphraseProvider()
	plaintext := []byte(`{"secret":"a-connection-string"}`)

	sealed, err := Seal(context.Background(), provider, "instances/abc", plaintext, []byte("instances/abc"))
	s.Require().NoError(err)

	opened, err := Open(context.Background(), provider, sealed, []byte("instances/abc"))
	s.Require().NoError(err)
	s.Assert().Equal(plaintext, opened)
}

func (s *CryptoSuite) Test_sealed_payload_does_not_contain_the_plaintext() {
	provider := s.passphraseProvider()
	plaintext := []byte("a-connection-string-nobody-should-see")

	sealed, err := Seal(context.Background(), provider, "instances/abc", plaintext, nil)
	s.Require().NoError(err)

	s.Assert().NotContains(string(sealed), string(plaintext))
}

// Associated data binds ciphertext to where it belongs, so an envelope moved from one
// storage key to another must fail rather than decrypt.
func (s *CryptoSuite) Test_opening_with_different_associated_data_fails() {
	provider := s.passphraseProvider()

	sealed, err := Seal(context.Background(), provider, "instances/abc", []byte("payload"), []byte("instances/abc"))
	s.Require().NoError(err)

	_, err = Open(context.Background(), provider, sealed, []byte("instances/xyz"))
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "failed authentication")
}

func (s *CryptoSuite) Test_opening_with_a_different_key_fails() {
	sealed, err := Seal(context.Background(), s.passphraseProvider(), "instances/abc", []byte("payload"), nil)
	s.Require().NoError(err)

	_, err = Open(context.Background(), s.passphraseProvider(), sealed, nil)
	s.Require().Error(err)
}

// Each seal must use a data key of its own.
//
// This is the invariant AES-GCM's security rests on here: a (key, nonce) pair must never
// encrypt two different messages. A fresh DEK per seal guarantees that regardless of the
// nonce, which is why a cached or reused data key would make nonce handling essential
// rather than defence in depth.
func (s *CryptoSuite) Test_each_seal_uses_a_fresh_data_key() {
	recorder := &dekRecordingProvider{inner: s.passphraseProvider()}

	for range 3 {
		_, err := Seal(context.Background(), recorder, "instances/abc", []byte("payload"), nil)
		s.Require().NoError(err)
	}

	s.Require().Len(recorder.wrapped, 3)
	s.Assert().NotEqual(recorder.wrapped[0], recorder.wrapped[1])
	s.Assert().NotEqual(recorder.wrapped[1], recorder.wrapped[2])
	s.Assert().NotEqual(recorder.wrapped[0], recorder.wrapped[2])

	for i, dek := range recorder.wrapped {
		s.Assert().Len(dek, DEKSize, "data key %d is not the expected length", i)
	}
}

// The scope must reach the provider. A caching or per-scope-key implementation cannot honour
// its contract if the envelope layer drops it, and nothing else would notice. The local
// providers ignore it, so every other test here passes whether it arrives or not.
func (s *CryptoSuite) Test_the_scope_is_passed_through_to_the_provider() {
	recorder := &dekRecordingProvider{inner: s.passphraseProvider()}

	for _, scope := range []string{"instances/abc", "instances/xyz", ""} {
		_, err := Seal(context.Background(), recorder, scope, []byte("payload"), nil)
		s.Require().NoError(err)
	}

	s.Assert().Equal([]string{"instances/abc", "instances/xyz", ""}, recorder.scopes)
}

// Records the data keys a Seal asks it to wrap, so a test can assert on what the envelope
// layer generated rather than inferring it from ciphertext.
type dekRecordingProvider struct {
	inner   KeyProvider
	wrapped [][]byte
	scopes  []string
}

func (p *dekRecordingProvider) ID() string {
	return p.inner.ID()
}

func (p *dekRecordingProvider) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	p.scopes = append(p.scopes, scope)

	dek, wrappedDEK, err := p.inner.GenerateDataKey(ctx, scope)
	if err != nil {
		return nil, nil, err
	}

	recorded := make([]byte, len(dek))
	copy(recorded, dek)
	p.wrapped = append(p.wrapped, recorded)

	return dek, wrappedDEK, nil
}

func (p *dekRecordingProvider) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	return p.inner.UnwrapKey(ctx, kekID, wrapped)
}

// Two seals of the same plaintext must differ. Guaranteed by the fresh data key above rather
// than by the nonce, so this is a check on the observable outcome, not on nonce handling.
func (s *CryptoSuite) Test_sealing_the_same_payload_twice_produces_different_ciphertext() {
	provider := s.passphraseProvider()
	plaintext := []byte("payload")

	first, err := Seal(context.Background(), provider, "instances/abc", plaintext, nil)
	s.Require().NoError(err)
	second, err := Seal(context.Background(), provider, "instances/abc", plaintext, nil)
	s.Require().NoError(err)

	s.Assert().NotEqual(first, second)
}

func (s *CryptoSuite) Test_tampered_ciphertext_is_rejected() {
	provider := s.passphraseProvider()

	sealed, err := Seal(context.Background(), provider, "instances/abc", []byte("payload"), nil)
	s.Require().NoError(err)

	sealed[len(sealed)-1] ^= 0xff

	_, err = Open(context.Background(), provider, sealed, nil)
	s.Require().Error(err)
}

// A store written before encryption was enabled must be readable as plaintext rather than
// reported as corrupt, which is what the magic prefix is for.
func (s *CryptoSuite) Test_plaintext_is_reported_as_not_an_envelope() {
	_, err := Open(context.Background(), s.passphraseProvider(), []byte(`{"plain":"json"}`), nil)
	s.Require().ErrorIs(err, ErrNotEnvelope)

	s.Assert().False(IsEnvelope([]byte(`{"plain":"json"}`)))
	s.Assert().False(IsEnvelope(nil))
	s.Assert().False(IsEnvelope([]byte("BL")))
}

func (s *CryptoSuite) Test_truncated_envelopes_are_rejected_rather_than_panicking() {
	provider := s.passphraseProvider()

	sealed, err := Seal(context.Background(), provider, "instances/abc", []byte("payload"), nil)
	s.Require().NoError(err)

	for _, length := range []int{5, 7, 9, 20, len(sealed) - 1} {
		_, err := Open(context.Background(), provider, sealed[:length], nil)
		s.Assert().Error(err, "a %d-byte prefix of an envelope was accepted", length)
	}
}

// A passphrase provider is only reproducible with the same salt, which is why the salt has
// to be persisted alongside the data.
func (s *CryptoSuite) Test_the_same_passphrase_and_salt_reproduce_the_same_key() {
	salt, err := GenerateSalt()
	s.Require().NoError(err)

	first, err := NewPassphraseKeyProvider("passphrase", salt)
	s.Require().NoError(err)
	second, err := NewPassphraseKeyProvider("passphrase", salt)
	s.Require().NoError(err)

	s.Require().Equal(first.ID(), second.ID())

	sealed, err := Seal(context.Background(), first, "instances/abc", []byte("payload"), nil)
	s.Require().NoError(err)

	opened, err := Open(context.Background(), second, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)
}

func (s *CryptoSuite) Test_a_different_salt_derives_a_different_key() {
	firstSalt, err := GenerateSalt()
	s.Require().NoError(err)
	secondSalt, err := GenerateSalt()
	s.Require().NoError(err)

	first, err := NewPassphraseKeyProvider("passphrase", firstSalt)
	s.Require().NoError(err)
	second, err := NewPassphraseKeyProvider("passphrase", secondSalt)
	s.Require().NoError(err)

	s.Assert().NotEqual(first.ID(), second.ID())
}

func (s *CryptoSuite) Test_rejects_an_empty_passphrase() {
	salt, err := GenerateSalt()
	s.Require().NoError(err)

	_, err = NewPassphraseKeyProvider("", salt)
	s.Require().Error(err)
}

func (s *CryptoSuite) Test_rejects_a_short_salt() {
	_, err := NewPassphraseKeyProvider("passphrase", []byte("short"))
	s.Require().Error(err)
}

// The key id must not be derivable back to the secret, since it travels in envelope headers
// that sit beside the ciphertext.
func (s *CryptoSuite) Test_the_key_id_does_not_contain_the_passphrase() {
	salt, err := GenerateSalt()
	s.Require().NoError(err)

	provider, err := NewPassphraseKeyProvider("a-very-distinctive-passphrase", salt)
	s.Require().NoError(err)

	s.Assert().NotContains(provider.ID(), "a-very-distinctive-passphrase")
	s.Assert().Contains(provider.ID(), "passphrase:")
}

func (s *CryptoSuite) Test_local_key_file_round_trips() {
	path := s.writeKeyFile(0o600, nil)

	provider, err := NewLocalFileKeyProvider(path)
	s.Require().NoError(err)

	sealed, err := Seal(context.Background(), provider, "instances/abc", []byte("payload"), nil)
	s.Require().NoError(err)

	reopened, err := NewLocalFileKeyProvider(path)
	s.Require().NoError(err)

	opened, err := Open(context.Background(), reopened, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)
}

// A key other users can read is not a key. Failing at startup beats encrypting against it.
func (s *CryptoSuite) Test_rejects_a_local_key_file_readable_by_others() {
	if runtime.GOOS == "windows" {
		s.T().Skip("mode bits are not the access control on windows")
	}

	path := s.writeKeyFile(0o644, nil)

	_, err := NewLocalFileKeyProvider(path)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "accessible to users other than its owner")
}

func (s *CryptoSuite) Test_rejects_a_local_key_file_of_the_wrong_length() {
	path := s.writeKeyFile(0o600, []byte("too-short"))

	_, err := NewLocalFileKeyProvider(path)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "expected exactly 32")
}

func (s *CryptoSuite) Test_rejects_a_missing_local_key_file() {
	_, err := NewLocalFileKeyProvider(filepath.Join(s.T().TempDir(), "absent.key"))
	s.Require().Error(err)
}

func (s *CryptoSuite) writeKeyFile(mode os.FileMode, contents []byte) string {
	if contents == nil {
		generated, err := GenerateLocalKey()
		s.Require().NoError(err)
		contents = generated
	}

	path := filepath.Join(s.T().TempDir(), "kek.key")
	s.Require().NoError(os.WriteFile(path, contents, mode))
	// WriteFile applies the umask, so the mode is set explicitly to get a permissive one.
	s.Require().NoError(os.Chmod(path, mode))

	return path
}

func TestCryptoSuite(t *testing.T) {
	suite.Run(t, new(CryptoSuite))
}
