package statestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/common/crypto"
)

// Decorates a Storage so that everything written through it is encrypted
// and everything read back is decrypted.
//
// Sitting at the byte-level Storage seam means every durable artifact is covered by one
// decorator with no changes to the persister or the
// chunking logic above it. That matters most for changesets and event partitions, which hold
// resolved secrets and cannot be redacted, because deployment, rollback and stream resume
// all read the persisted copies.
type encryptedStorage struct {
	inner    Storage
	provider crypto.KeyProvider
}

// NewEncryptedStorage returns a Storage that encrypts on write and decrypts on read.
func NewEncryptedStorage(
	inner Storage,
	provider crypto.KeyProvider,
) Storage {
	return &encryptedStorage{
		inner:    inner,
		provider: provider,
	}
}

// The storage key is the associated data, so an object only decrypts at the key it was
// written to. Moving ciphertext between keys, by hand or by a faulty backend, produces a
// failure rather than a successful read of the wrong record.
func (s *encryptedStorage) Read(ctx context.Context, key string) ([]byte, error) {
	sealed, err := s.inner.Read(ctx, key)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.Open(ctx, s.provider, sealed, []byte(key))
	if err != nil {
		if errors.Is(err, crypto.ErrNotEnvelope) {
			return nil, fmt.Errorf(
				"statestore: %q is not encrypted, so this state was written before encryption "+
					"was enabled; re-encrypt the state directory or start from an empty one",
				key,
			)
		}

		return nil, fmt.Errorf("statestore: decrypting %q: %w", key, err)
	}

	return plaintext, nil
}

func (s *encryptedStorage) Write(ctx context.Context, key string, data []byte) error {
	// The key is both the scope and the associated data for different reasons.
	// The scope groups what a provider may share a key across, and the associated data binds the
	// ciphertext to this key so it cannot be read at another. They coincide here because the
	// object is the grouping; a backend wanting a coarser scope can use different values.
	sealed, err := crypto.Seal(ctx, s.provider, key, data, []byte(key))
	if err != nil {
		return fmt.Errorf("statestore: encrypting %q: %w", key, err)
	}

	return s.inner.Write(ctx, key, sealed)
}

func (s *encryptedStorage) Delete(ctx context.Context, key string) error {
	return s.inner.Delete(ctx, key)
}

// Keys are not encrypted, so listing passes straight through. A key names a category and a
// chunk number or entity id, none of which is sensitive, and encrypting them would break the
// index and pattern matching the loader depends on.
func (s *encryptedStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return s.inner.List(ctx, prefix)
}

func (s *encryptedStorage) Exists(ctx context.Context, key string) (bool, error) {
	return s.inner.Exists(ctx, key)
}

// The stored form of the salt, as JSON so that it looks like every other
// object in the store rather than a stray binary file.
type persistedSalt struct {
	Salt []byte `json:"salt"`
}

// LoadOrCreateSalt returns the salt for a passphrase-derived key, creating and persisting one
// on first use.
//
// Must be called with the undecorated Storage, before NewEncryptedStorage wraps it: the salt
// is an input to deriving the key that would decrypt it.
//
// The salt has to outlive the process or a passphrase provider is useless, the same
// passphrase with a fresh salt derives a different key and cannot read anything already
// written. Keeping it in the store rather than in engine configuration means it travels with
// the state it belongs to.
func LoadOrCreateSalt(ctx context.Context, storage Storage, keys KeyBuilder) ([]byte, error) {
	key := keys.EncryptionSalt()

	existing, err := storage.Read(ctx, key)
	if err == nil {
		stored := &persistedSalt{}
		if err := json.Unmarshal(existing, stored); err != nil {
			return nil, fmt.Errorf("statestore: reading encryption salt at %q: %w", key, err)
		}

		if len(stored.Salt) < crypto.SaltSize {
			return nil, fmt.Errorf(
				"statestore: encryption salt at %q is %d bytes, expected at least %d; "+
					"the file is corrupt and the state it protects cannot be read without it",
				key,
				len(stored.Salt),
				crypto.SaltSize,
			)
		}

		return stored.Salt, nil
	}

	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("statestore: reading encryption salt at %q: %w", key, err)
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(&persistedSalt{Salt: salt})
	if err != nil {
		return nil, fmt.Errorf("statestore: encoding encryption salt: %w", err)
	}

	if err := storage.Write(ctx, key, encoded); err != nil {
		return nil, fmt.Errorf("statestore: writing encryption salt to %q: %w", key, err)
	}

	return salt, nil
}
