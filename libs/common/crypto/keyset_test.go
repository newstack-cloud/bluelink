package crypto

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type KeySetSuite struct {
	suite.Suite
}

func (s *KeySetSuite) passphraseProviderFor(passphrase string, salt []byte) KeyProvider {
	provider, err := NewPassphraseKeyProvider(passphrase, salt)
	s.Require().NoError(err)

	return provider
}

func (s *KeySetSuite) salt() []byte {
	salt, err := GenerateSalt()
	s.Require().NoError(err)

	return salt
}

func (s *KeySetSuite) Test_seals_with_the_active_key() {
	salt := s.salt()
	active := s.passphraseProviderFor("new-passphrase", salt)
	retired := s.passphraseProviderFor("old-passphrase", salt)

	keySet, err := NewKeySet(active, retired)
	s.Require().NoError(err)
	s.Require().Equal(active.ID(), keySet.ID())

	sealed, err := Seal(context.Background(), keySet, "instances_c0.json", []byte("payload"), nil)
	s.Require().NoError(err)

	keyID, err := EnvelopeKeyID(sealed)
	s.Require().NoError(err)
	s.Assert().Equal(active.ID(), keyID)

	// Sealed with the active key alone, so the retired key must not open it.
	_, err = Open(context.Background(), retired, sealed, nil)
	s.Require().Error(err)
}

// The point of a key set where state written under the previous key stays readable after the key
// is changed, which would otherwise be unrecoverable.
func (s *KeySetSuite) Test_opens_state_sealed_with_a_retired_key() {
	salt := s.salt()
	retired := s.passphraseProviderFor("old-passphrase", salt)

	sealed, err := Seal(context.Background(), retired, "instances_c0.json", []byte("payload"), nil)
	s.Require().NoError(err)

	keySet, err := NewKeySet(s.passphraseProviderFor("new-passphrase", salt), retired)
	s.Require().NoError(err)

	opened, err := Open(context.Background(), keySet, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)
}

// A key set spanning provider types is what lets a deployment move from a local key to a key
// management service without re-encrypting its state first.
func (s *KeySetSuite) Test_opens_state_sealed_by_a_different_kind_of_provider() {
	passphrase := s.passphraseProviderFor("old-passphrase", s.salt())

	sealed, err := Seal(context.Background(), passphrase, "instances_c0.json", []byte("payload"), nil)
	s.Require().NoError(err)

	localKey, err := NewLocalFileKeyProvider(s.writeKeyFile())
	s.Require().NoError(err)

	keySet, err := NewKeySet(localKey, passphrase)
	s.Require().NoError(err)

	opened, err := Open(context.Background(), keySet, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)
}

func (s *KeySetSuite) Test_reports_when_no_key_in_the_set_can_open_an_envelope() {
	sealed, err := Seal(
		context.Background(),
		s.passphraseProviderFor("a-third-passphrase", s.salt()),
		"instances_c0.json",
		[]byte("payload"),
		nil,
	)
	s.Require().NoError(err)

	salt := s.salt()
	keySet, err := NewKeySet(
		s.passphraseProviderFor("new-passphrase", salt),
		s.passphraseProviderFor("old-passphrase", salt),
	)
	s.Require().NoError(err)

	_, err = Open(context.Background(), keySet, sealed, nil)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no configured key could unwrap")
}

func (s *KeySetSuite) Test_a_set_of_one_key_behaves_as_that_key() {
	active := s.passphraseProviderFor("only-passphrase", s.salt())

	keySet, err := NewKeySet(active)
	s.Require().NoError(err)

	sealed, err := Seal(context.Background(), keySet, "instances_c0.json", []byte("payload"), nil)
	s.Require().NoError(err)

	opened, err := Open(context.Background(), active, sealed, nil)
	s.Require().NoError(err)
	s.Assert().Equal([]byte("payload"), opened)
}

func (s *KeySetSuite) Test_requires_an_active_key() {
	_, err := NewKeySet(nil)
	s.Require().Error(err)

	_, err = NewKeySet(s.passphraseProviderFor("passphrase", s.salt()), nil)
	s.Require().Error(err)
}

func (s *KeySetSuite) writeKeyFile() string {
	key, err := GenerateLocalKey()
	s.Require().NoError(err)

	path := filepath.Join(s.T().TempDir(), "kek.key")
	s.Require().NoError(os.WriteFile(path, key, 0o600))

	return path
}

func TestKeySetSuite(t *testing.T) {
	suite.Run(t, new(KeySetSuite))
}
