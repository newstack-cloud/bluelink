package statestore

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/stretchr/testify/suite"
)

type ReencryptSuite struct {
	suite.Suite
	salt []byte
}

func (s *ReencryptSuite) SetupTest() {
	salt, err := crypto.GenerateSalt()
	s.Require().NoError(err)
	s.salt = salt
}

func (s *ReencryptSuite) providerFor(passphrase string) crypto.KeyProvider {
	provider, err := crypto.NewPassphraseKeyProvider(passphrase, s.salt)
	s.Require().NoError(err)

	return provider
}

func (s *ReencryptSuite) keySet(active crypto.KeyProvider, retired crypto.KeyProvider) crypto.KeyProvider {
	keySet, err := crypto.NewKeySet(active, retired)
	s.Require().NoError(err)

	return keySet
}

// A store with one object of each kind: state, and the salt that must survive a pass
// untouched.
func (s *ReencryptSuite) storeWith(provider crypto.KeyProvider, payloads map[string][]byte) *mapStorage {
	inner := newMapStorage()
	ctx := context.Background()

	_, err := LoadOrCreateSalt(ctx, inner, NewKeyBuilder(""))
	s.Require().NoError(err)

	for key, payload := range payloads {
		if provider == nil {
			s.Require().NoError(inner.Write(ctx, key, payload))
			continue
		}

		s.Require().NoError(NewEncryptedStorage(inner, provider).Write(ctx, key, payload))
	}

	return inner
}

func (s *ReencryptSuite) Test_encrypts_a_store_that_was_written_in_plaintext() {
	payload := []byte(`{"specData":{"token":"s3cr3t"}}`)
	inner := s.storeWith(nil, map[string][]byte{"instances_c0.json": payload})
	target := s.providerFor("new-passphrase")

	report, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Target: target,
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, report.Encrypted)

	stored, err := inner.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Require().True(crypto.IsEnvelope(stored))
	s.Assert().NotContains(string(stored), "s3cr3t")

	read, err := NewEncryptedStorage(inner, target).Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(payload, read)
}

// The reason the tool exists: after a pass, the old key is no longer needed to read anything.
func (s *ReencryptSuite) Test_moves_a_store_from_one_key_to_another() {
	payload := []byte(`{"specData":{"token":"s3cr3t"}}`)
	retired := s.providerFor("old-passphrase")
	active := s.providerFor("new-passphrase")
	inner := s.storeWith(retired, map[string][]byte{
		"instances_c0.json":  payload,
		"changesets_c0.json": []byte(`{"newValue":"s3cr3t"}`),
	})

	report, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: s.keySet(active, retired),
		Target: active,
	})
	s.Require().NoError(err)
	s.Assert().Equal(2, report.Reencrypted)

	read, err := NewEncryptedStorage(inner, active).Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(payload, read)

	// Nothing is left that the retired key is needed for.
	_, err = NewEncryptedStorage(inner, retired).Read(context.Background(), "instances_c0.json")
	s.Require().Error(err)
}

// An interrupted pass leaves a store holding both keys, so running it again has to pick up
// where it stopped rather than fail on what it already moved.
func (s *ReencryptSuite) Test_a_second_pass_leaves_objects_already_moved_alone() {
	retired := s.providerFor("old-passphrase")
	active := s.providerFor("new-passphrase")
	inner := s.storeWith(retired, map[string][]byte{"instances_c0.json": []byte("payload")})
	options := ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: s.keySet(active, retired),
		Target: active,
	}

	_, err := Reencrypt(context.Background(), inner, options)
	s.Require().NoError(err)

	report, err := Reencrypt(context.Background(), inner, options)
	s.Require().NoError(err)
	s.Assert().Equal(1, report.AlreadyCurrent)
	s.Assert().Equal(0, report.Reencrypted)
}

// The salt is an input to deriving the key, so a pass that encrypted or replaced it would
// make the store it just re-encrypted unreadable.
func (s *ReencryptSuite) Test_leaves_the_encryption_salt_untouched() {
	keys := NewKeyBuilder("")
	inner := s.storeWith(nil, map[string][]byte{"instances_c0.json": []byte("payload")})

	before, err := inner.Read(context.Background(), keys.EncryptionSalt())
	s.Require().NoError(err)

	report, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   keys,
		Target: s.providerFor("new-passphrase"),
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, report.Skipped)

	after, err := inner.Read(context.Background(), keys.EncryptionSalt())
	s.Require().NoError(err)
	s.Assert().Equal(before, after)
	s.Assert().False(crypto.IsEnvelope(after))
}

// Backing the feature out has to be possible, otherwise enabling it is a one-way door.
func (s *ReencryptSuite) Test_decrypts_a_store_when_given_no_target_key() {
	payload := []byte(`{"specData":{"token":"s3cr3t"}}`)
	source := s.providerFor("old-passphrase")
	inner := s.storeWith(source, map[string][]byte{"instances_c0.json": payload})

	report, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: source,
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, report.Decrypted)

	stored, err := inner.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(payload, stored)
}

func (s *ReencryptSuite) Test_a_dry_run_changes_nothing() {
	source := s.providerFor("old-passphrase")
	inner := s.storeWith(source, map[string][]byte{"instances_c0.json": []byte("payload")})

	before, err := inner.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)

	report, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: source,
		Target: s.providerFor("new-passphrase"),
		DryRun: true,
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, report.Reencrypted, "a dry run should still report what it would do")

	after, err := inner.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(before, after)
}

// Stopping on the first object it cannot open beats writing a store half-sealed under a key
// the operator has not noticed is missing.
func (s *ReencryptSuite) Test_stops_when_an_object_cannot_be_opened() {
	inner := s.storeWith(s.providerFor("a-third-passphrase"), map[string][]byte{
		"instances_c0.json": []byte("payload"),
	})

	_, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: s.providerFor("old-passphrase"),
		Target: s.providerFor("new-passphrase"),
	})
	s.Require().Error(err)
}

func (s *ReencryptSuite) Test_reports_an_object_sealed_with_no_source_key_configured() {
	inner := s.storeWith(s.providerFor("old-passphrase"), map[string][]byte{
		"instances_c0.json": []byte("payload"),
	})

	_, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Target: s.providerFor("new-passphrase"),
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no source key was configured")
}

func (s *ReencryptSuite) Test_requires_a_key_to_work_with() {
	_, err := Reencrypt(context.Background(), newMapStorage(), ReencryptOptions{
		Keys: NewKeyBuilder(""),
	})
	s.Require().Error(err)
}

func (s *ReencryptSuite) Test_reports_progress_for_each_object() {
	source := s.providerFor("old-passphrase")
	inner := s.storeWith(source, map[string][]byte{
		"instances_c0.json":  []byte("payload"),
		"changesets_c0.json": []byte("payload"),
	})

	outcomes := map[string]ReencryptOutcome{}
	_, err := Reencrypt(context.Background(), inner, ReencryptOptions{
		Keys:   NewKeyBuilder(""),
		Source: source,
		Target: s.providerFor("new-passphrase"),
		Observe: func(key string, outcome ReencryptOutcome) {
			outcomes[key] = outcome
		},
	})
	s.Require().NoError(err)

	s.Assert().Equal(map[string]ReencryptOutcome{
		"instances_c0.json":    ReencryptOutcomeReencrypted,
		"changesets_c0.json":   ReencryptOutcomeReencrypted,
		"encryption_salt.json": ReencryptOutcomeSkipped,
	}, outcomes)
}

func TestReencryptSuite(t *testing.T) {
	suite.Run(t, new(ReencryptSuite))
}
