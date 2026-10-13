package crypto

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters for passphrase-derived keys, at the OWASP-recommended minimum of 19MiB
// memory with two iterations. Deriving happens once at startup rather than per object, so the
// cost is paid once and does not bear on write throughput.
const (
	argon2Time      = 2
	argon2MemoryKiB = 19 * 1024
	argon2Threads   = 1

	// SaltSize is the length of the salt a passphrase provider needs. A deployment persists
	// it alongside the data it protects. A salt is not secret, and losing it loses the data.
	SaltSize = 16
)

// localKeyProvider wraps data keys with a key-encryption key held in process.
//
// Shared by the passphrase and local-key-file providers, which differ only in where the KEK
// comes from. The wrapping itself is AES-256-GCM with a random nonce prefixed to each
// wrapped key, so two data keys wrapped with the same KEK produce different ciphertext.
type localKeyProvider struct {
	id  string
	kek []byte
}

// NewPassphraseKeyProvider derives a key-encryption key from a passphrase using Argon2id.
//
// The salt must be stable for the life of the data, the same passphrase with a different salt
// derives a different key and cannot unwrap anything previously wrapped.
func NewPassphraseKeyProvider(passphrase string, salt []byte) (KeyProvider, error) {
	if passphrase == "" {
		return nil, errors.New("crypto: passphrase must not be empty")
	}
	if len(salt) < SaltSize {
		return nil, fmt.Errorf("crypto: salt is %d bytes, expected at least %d", len(salt), SaltSize)
	}

	kek := argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2MemoryKiB,
		argon2Threads,
		DEKSize,
	)

	// Identified by a digest of the derived key rather than by anything naming the
	// passphrase, so the id can go in an envelope header without hinting at the secret, and
	// still changes when the passphrase or salt does.
	return &localKeyProvider{id: keyID("passphrase", kek), kek: kek}, nil
}

// NewLocalFileKeyProvider reads a raw 32-byte key-encryption key from a file.
//
// The file's permissions are checked rather than assumed, a key readable by other users is
// not deemed a secure key, and failing at startup is better than encrypting against it.
func NewLocalFileKeyProvider(path string) (KeyProvider, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("crypto: reading local key file: %w", err)
	}

	// Skipped on Windows, where the mode bits os.Stat reports are not the access control and
	// the file's ACL is what governs access.
	if isPermissionCheckSupported() && info.Mode()&0o077 != 0 {
		return nil, fmt.Errorf(
			"crypto: local key file %q is accessible to users other than its owner (mode %04o), "+
				"restrict it to 0600",
			path,
			info.Mode().Perm(),
		)
	}

	kek, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("crypto: reading local key file: %w", err)
	}

	if len(kek) != DEKSize {
		return nil, fmt.Errorf(
			"crypto: local key file %q holds %d bytes, expected exactly %d",
			path,
			len(kek),
			DEKSize,
		)
	}

	return &localKeyProvider{id: keyID("localkey", kek), kek: kek}, nil
}

// GenerateLocalKey returns a random key-encryption key suitable for a local key file.
func GenerateLocalKey() ([]byte, error) {
	kek := make([]byte, DEKSize)
	if _, err := rand.Read(kek); err != nil {
		return nil, fmt.Errorf("crypto: generating local key: %w", err)
	}

	return kek, nil
}

// GenerateSalt returns a random salt suitable for a passphrase provider.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("crypto: generating salt: %w", err)
	}

	return salt, nil
}

func (p *localKeyProvider) ID() string {
	return p.id
}

// A key per object, generated locally and wrapped immediately. The scope is ignored, a local
// KEK has no round trip to save, so reusing a key would take on the risks of sharing one for
// no gain.
func (p *localKeyProvider) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	dek := make([]byte, DEKSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, nil, fmt.Errorf("crypto: generating data key: %w", err)
	}

	wrappedDEK, err := p.wrapKey(dek)
	if err != nil {
		return nil, nil, err
	}

	return dek, wrappedDEK, nil
}

func (p *localKeyProvider) wrapKey(dek []byte) ([]byte, error) {
	gcm, err := newGCM(p.kek)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: generating wrap nonce: %w", err)
	}

	// The nonce is prefixed to the ciphertext so that a wrapped key carries everything
	// UnwrapKey needs to read it back. Unlike an envelope, which has a header to record its
	// nonce in, a wrapped key is opaque bytes with nowhere else to put it.
	//
	// Seal appends to the destination it is given, so starting from a buffer already holding
	// the nonce produces nonce ‖ ciphertext ‖ tag in one allocation.
	wrapped := make([]byte, 0, gcm.NonceSize()+len(dek)+gcm.Overhead())
	wrapped = append(wrapped, nonce...)

	return gcm.Seal(wrapped, nonce, dek, nil), nil
}

func (p *localKeyProvider) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	if kekID != p.id {
		return nil, fmt.Errorf(
			"crypto: envelope was sealed with key %q, this provider holds %q",
			kekID,
			p.id,
		)
	}

	gcm, err := newGCM(p.kek)
	if err != nil {
		return nil, err
	}

	if len(wrapped) < gcm.NonceSize() {
		return nil, errors.New("crypto: wrapped data key is truncated")
	}

	nonce := wrapped[:gcm.NonceSize()]
	dek, err := gcm.Open(nil, nonce, wrapped[gcm.NonceSize():], nil)
	if err != nil {
		return nil, errors.New("crypto: wrapped data key failed authentication")
	}

	return dek, nil
}

// Permission bits are the access control on unix. On Windows os.Stat reports bits that are
// not what governs access, so a check on them would reject valid key files and pass invalid
// ones.
func isPermissionCheckSupported() bool {
	return runtime.GOOS != "windows"
}

// A short, stable identifier for a key, derived from the key itself so that it changes
// whenever the key does and reveals nothing about it.
func keyID(prefix string, kek []byte) string {
	digest := sha256.Sum256(kek)

	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(digest[:8]))
}
