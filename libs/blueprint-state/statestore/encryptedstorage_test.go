package statestore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/stretchr/testify/suite"
)

type EncryptedStorageSuite struct {
	suite.Suite
}

func (s *EncryptedStorageSuite) provider() crypto.KeyProvider {
	salt, err := crypto.GenerateSalt()
	s.Require().NoError(err)

	provider, err := crypto.NewPassphraseKeyProvider("deployment-passphrase", salt)
	s.Require().NoError(err)

	return provider
}

func (s *EncryptedStorageSuite) Test_round_trips_a_payload() {
	inner := newMapStorage()
	encrypted := NewEncryptedStorage(inner, s.provider())
	payload := []byte(`{"specData":{"connectionString":"postgres://user:s3cr3t@host/db"}}`)

	s.Require().NoError(encrypted.Write(context.Background(), "instances_c0.json", payload))

	read, err := encrypted.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(payload, read)
}

// The point of the decorator: what reaches the backend must not be the plaintext.
func (s *EncryptedStorageSuite) Test_what_reaches_the_inner_storage_is_encrypted() {
	inner := newMapStorage()
	encrypted := NewEncryptedStorage(inner, s.provider())

	s.Require().NoError(encrypted.Write(
		context.Background(),
		"changesets_c0.json",
		[]byte(`{"newValue":"s3cr3t"}`),
	))

	stored, err := inner.Read(context.Background(), "changesets_c0.json")
	s.Require().NoError(err)
	s.Assert().NotContains(string(stored), "s3cr3t")
	s.Assert().True(crypto.IsEnvelope(stored), "the stored payload is not an envelope")
}

// The storage key is the associated data, so ciphertext moved to another key must fail rather
// than decrypt as the wrong record.
func (s *EncryptedStorageSuite) Test_an_object_moved_to_another_key_does_not_decrypt() {
	inner := newMapStorage()
	encrypted := NewEncryptedStorage(inner, s.provider())

	s.Require().NoError(encrypted.Write(context.Background(), "instances_c0.json", []byte("payload")))

	sealed, err := inner.Read(context.Background(), "instances_c0.json")
	s.Require().NoError(err)
	s.Require().NoError(inner.Write(context.Background(), "instances_c1.json", sealed))

	_, err = encrypted.Read(context.Background(), "instances_c1.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "decrypting")
}

// Enabling encryption over an existing plaintext store must say so, rather than report
// corruption or return the plaintext.
func (s *EncryptedStorageSuite) Test_reading_unencrypted_state_reports_why() {
	inner := newMapStorage()
	s.Require().NoError(inner.Write(context.Background(), "instances_c0.json", []byte(`{"plain":1}`)))

	encrypted := NewEncryptedStorage(inner, s.provider())

	_, err := encrypted.Read(context.Background(), "instances_c0.json")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "is not encrypted")
}

// A missing key must stay distinguishable from a decryption failure: the persister treats
// ErrNotFound as an empty chunk rather than an error.
func (s *EncryptedStorageSuite) Test_not_found_passes_through_unchanged() {
	encrypted := NewEncryptedStorage(newMapStorage(), s.provider())

	_, err := encrypted.Read(context.Background(), "instances_c0.json")
	s.Require().ErrorIs(err, ErrNotFound)
}

func (s *EncryptedStorageSuite) Test_delete_list_and_exists_pass_through() {
	inner := newMapStorage()
	encrypted := NewEncryptedStorage(inner, s.provider())
	ctx := context.Background()

	s.Require().NoError(encrypted.Write(ctx, "instances_c0.json", []byte("a")))
	s.Require().NoError(encrypted.Write(ctx, "changesets_c0.json", []byte("b")))

	exists, err := encrypted.Exists(ctx, "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().True(exists)

	// Keys are not encrypted: the loader matches on them and the chunk index refers to them.
	keys, err := encrypted.List(ctx, "")
	s.Require().NoError(err)
	s.Assert().ElementsMatch([]string{"instances_c0.json", "changesets_c0.json"}, keys)

	s.Require().NoError(encrypted.Delete(ctx, "instances_c0.json"))
	exists, err = encrypted.Exists(ctx, "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().False(exists)
}

// Each object's key is its scope, so a caching provider groups by object rather than across
// the whole store. Nothing else here would notice if it did not, since the local providers
// ignore the scope.
func (s *EncryptedStorageSuite) Test_each_objects_key_is_its_scope() {
	recorder := &scopeRecordingProvider{inner: s.provider()}
	encrypted := NewEncryptedStorage(newMapStorage(), recorder)
	ctx := context.Background()

	s.Require().NoError(encrypted.Write(ctx, "instances_c0.json", []byte("a")))
	s.Require().NoError(encrypted.Write(ctx, "changesets_c0.json", []byte("b")))

	s.Assert().Equal([]string{"instances_c0.json", "changesets_c0.json"}, recorder.scopes)
}

// Rewrites of one object share a scope, which is what lets a caching provider use a key
// across them. Distinct objects must not share one.
func (s *EncryptedStorageSuite) Test_rewrites_of_one_object_share_its_scope() {
	recorder := &scopeRecordingProvider{inner: s.provider()}
	encrypted := NewEncryptedStorage(newMapStorage(), recorder)
	ctx := context.Background()

	for range 3 {
		s.Require().NoError(encrypted.Write(ctx, "instances_c0.json", []byte("a")))
	}
	s.Require().NoError(encrypted.Write(ctx, "instances_c1.json", []byte("b")))

	s.Assert().Equal(
		[]string{"instances_c0.json", "instances_c0.json", "instances_c0.json", "instances_c1.json"},
		recorder.scopes,
	)
}

func (s *EncryptedStorageSuite) Test_creates_a_salt_on_first_use_and_reuses_it_after() {
	inner := newMapStorage()
	keys := NewKeyBuilder("")
	ctx := context.Background()

	first, err := LoadOrCreateSalt(ctx, inner, keys)
	s.Require().NoError(err)
	s.Require().Len(first, crypto.SaltSize)

	second, err := LoadOrCreateSalt(ctx, inner, keys)
	s.Require().NoError(err)
	s.Assert().Equal(first, second, "a second call generated a new salt")
}

// The salt has to survive a restart or a passphrase provider cannot read what it wrote. This
// stands in for a process restart: a fresh storage view over the same persisted bytes.
func (s *EncryptedStorageSuite) Test_state_written_in_one_run_is_readable_in_the_next() {
	inner := newMapStorage()
	keys := NewKeyBuilder("")
	ctx := context.Background()
	payload := []byte(`{"specData":{"token":"s3cr3t"}}`)

	firstSalt, err := LoadOrCreateSalt(ctx, inner, keys)
	s.Require().NoError(err)
	firstProvider, err := crypto.NewPassphraseKeyProvider("passphrase", firstSalt)
	s.Require().NoError(err)
	s.Require().NoError(
		NewEncryptedStorage(inner, firstProvider).
			Write(ctx, "instances_c0.json", payload),
	)

	// Second run: same persisted bytes, same passphrase, salt loaded rather than generated.
	secondSalt, err := LoadOrCreateSalt(ctx, inner, keys)
	s.Require().NoError(err)
	secondProvider, err := crypto.NewPassphraseKeyProvider("passphrase", secondSalt)
	s.Require().NoError(err)

	read, err := NewEncryptedStorage(inner, secondProvider).
		Read(ctx, "instances_c0.json")
	s.Require().NoError(err)
	s.Assert().Equal(payload, read)
}

// The salt is an input to deriving the key that would decrypt it, so it cannot itself be
// encrypted. Written through the undecorated storage, it stays readable.
func (s *EncryptedStorageSuite) Test_the_salt_is_stored_unencrypted() {
	inner := newMapStorage()
	keys := NewKeyBuilder("")

	_, err := LoadOrCreateSalt(context.Background(), inner, keys)
	s.Require().NoError(err)

	stored, err := inner.Read(context.Background(), keys.EncryptionSalt())
	s.Require().NoError(err)
	s.Assert().False(crypto.IsEnvelope(stored))

	decoded := &persistedSalt{}
	s.Require().NoError(json.Unmarshal(stored, decoded))
	s.Assert().Len(decoded.Salt, crypto.SaltSize)
}

func (s *EncryptedStorageSuite) Test_a_corrupt_salt_is_reported_rather_than_replaced() {
	inner := newMapStorage()
	keys := NewKeyBuilder("")
	ctx := context.Background()

	s.Require().NoError(inner.Write(ctx, keys.EncryptionSalt(), []byte(`{"salt":"c2hvcnQ="}`)))

	_, err := LoadOrCreateSalt(ctx, inner, keys)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "cannot be read without it")
}

// Records the scopes it is asked to generate keys for.
type scopeRecordingProvider struct {
	inner  crypto.KeyProvider
	scopes []string
}

func (p *scopeRecordingProvider) ID() string { return p.inner.ID() }

func (p *scopeRecordingProvider) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	p.scopes = append(p.scopes, scope)

	return p.inner.GenerateDataKey(ctx, scope)
}

func (p *scopeRecordingProvider) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	return p.inner.UnwrapKey(ctx, kekID, wrapped)
}

// An in-memory Storage, so these tests exercise the decorator rather than a backend.
type mapStorage struct {
	objects map[string][]byte
}

func newMapStorage() *mapStorage {
	return &mapStorage{objects: map[string][]byte{}}
}

func (m *mapStorage) Read(ctx context.Context, key string) ([]byte, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, ErrNotFound
	}

	return data, nil
}

func (m *mapStorage) Write(ctx context.Context, key string, data []byte) error {
	stored := make([]byte, len(data))
	copy(stored, data)
	m.objects[key] = stored

	return nil
}

func (m *mapStorage) Delete(ctx context.Context, key string) error {
	if _, ok := m.objects[key]; !ok {
		return ErrNotFound
	}
	delete(m.objects, key)

	return nil
}

func (m *mapStorage) List(ctx context.Context, prefix string) ([]string, error) {
	keys := make([]string, 0, len(m.objects))
	for key := range m.objects {
		keys = append(keys, key)
	}

	return keys, nil
}

func (m *mapStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.objects[key]

	return ok, nil
}

func TestEncryptedStorageSuite(t *testing.T) {
	suite.Run(t, new(EncryptedStorageSuite))
}
