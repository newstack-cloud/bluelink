package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// Marks a payload as an envelope produced by this package, so that a
	// reader can tell ciphertext from plaintext written before encryption was enabled.
	// BLSE stands for Bluelink State Encryption.
	envelopeMagic = "BLSE1"

	// The format version recorded in each header. A reader refuses a
	// version it does not know rather than guessing at the layout.
	envelopeVersion = 1

	headerLengthBytes = 4

	// Bounds the header a reader will allocate for, so a corrupt or hostile
	// length prefix cannot ask for an arbitrary amount of memory.
	maxHeaderLength = 64 * 1024
)

// ErrNotEnvelope is returned when a payload does not carry the envelope magic, which means
// it was written before encryption was enabled.
var ErrNotEnvelope = errors.New("crypto: payload is not an envelope")

// Describes the serialised layout:
//
//	magic "BLSE1" | uint32 header length | JSON header | AES-256-GCM ciphertext
type envelopeHeader struct {
	Version    int    `json:"v"`
	KEKID      string `json:"kekId"`
	WrappedDEK []byte `json:"wrappedDek"`
	Nonce      []byte `json:"nonce"`
}

// Seal encrypts plaintext under a freshly generated data-encryption key, wrapped by the
// provider's key-encryption key, and returns the serialised envelope.
//
// scope is passed to the provider unchanged; see KeyProvider.GenerateDataKey.
//
// associatedData is authenticated but not encrypted, so a reader passing different
// associated data gets a failure rather than plaintext. Callers pass something that names
// where the ciphertext belongs, so an envelope moved elsewhere no longer decrypts. Open
// needs no scope, since the envelope records which key-encryption key sealed it.
func Seal(
	ctx context.Context,
	provider KeyProvider,
	scope string,
	plaintext []byte,
	associatedData []byte,
) ([]byte, error) {
	dek, wrappedDEK, err := provider.GenerateDataKey(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("crypto: generating data key: %w", err)
	}

	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}

	// Load-bearing whenever the provider caches data keys, which is a supported case: a key
	// covering many objects makes this nonce the only thing preventing keystream reuse, and
	// with it recovery of the GCM authentication subkey. It must stay random, and a caching
	// provider must bound how many objects one key covers to stay clear of the birthday
	// bound on a 96-bit nonce.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: generating nonce: %w", err)
	}

	header, err := json.Marshal(&envelopeHeader{
		Version:    envelopeVersion,
		KEKID:      provider.ID(),
		WrappedDEK: wrappedDEK,
		Nonce:      nonce,
	})
	if err != nil {
		return nil, fmt.Errorf("crypto: encoding envelope header: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, associatedData)

	sealed := make([]byte, 0, len(envelopeMagic)+headerLengthBytes+len(header)+len(ciphertext))
	sealed = append(sealed, envelopeMagic...)
	sealed = binary.BigEndian.AppendUint32(sealed, uint32(len(header)))
	sealed = append(sealed, header...)
	sealed = append(sealed, ciphertext...)

	return sealed, nil
}

// Open decrypts an envelope produced by Seal.
//
// Returns ErrNotEnvelope for a payload without the magic prefix, which a caller reading a
// store written before encryption was enabled treats as plaintext rather than an error.
func Open(
	ctx context.Context,
	provider KeyProvider,
	sealed []byte,
	associatedData []byte,
) ([]byte, error) {
	if !IsEnvelope(sealed) {
		return nil, ErrNotEnvelope
	}

	rest := sealed[len(envelopeMagic):]
	if len(rest) < headerLengthBytes {
		return nil, errors.New("crypto: envelope truncated before header length")
	}

	headerLength := binary.BigEndian.Uint32(rest[:headerLengthBytes])
	if headerLength > maxHeaderLength {
		return nil, fmt.Errorf("crypto: envelope header length %d exceeds maximum", headerLength)
	}

	rest = rest[headerLengthBytes:]
	if uint32(len(rest)) < headerLength {
		return nil, errors.New("crypto: envelope truncated before end of header")
	}

	header := &envelopeHeader{}
	if err := json.Unmarshal(rest[:headerLength], header); err != nil {
		return nil, fmt.Errorf("crypto: decoding envelope header: %w", err)
	}

	if header.Version != envelopeVersion {
		return nil, fmt.Errorf(
			"crypto: unsupported envelope version %d, this build understands %d",
			header.Version,
			envelopeVersion,
		)
	}

	dek, err := provider.UnwrapKey(ctx, header.KEKID, header.WrappedDEK)
	if err != nil {
		return nil, fmt.Errorf("crypto: unwrapping data key: %w", err)
	}

	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}

	if len(header.Nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf(
			"crypto: envelope nonce is %d bytes, expected %d",
			len(header.Nonce),
			gcm.NonceSize(),
		)
	}

	plaintext, err := gcm.Open(nil, header.Nonce, rest[headerLength:], associatedData)
	if err != nil {
		// Deliberately not wrapped with the underlying message, which says only "message
		// authentication failed" and is indistinguishable between a corrupt payload, the
		// wrong key and mismatched associated data.
		return nil, errors.New("crypto: envelope failed authentication, it may be corrupt, " +
			"sealed with a different key, or read from a different location than it was written")
	}

	return plaintext, nil
}

// IsEnvelope reports whether a payload carries the envelope magic.
//
// Lets a caller tell data written under encryption from data written before it was enabled,
// without attempting to decrypt.
func IsEnvelope(payload []byte) bool {
	return len(payload) >= len(envelopeMagic) && string(payload[:len(envelopeMagic)]) == envelopeMagic
}

func newGCM(dek []byte) (cipher.AEAD, error) {
	if len(dek) != DEKSize {
		return nil, fmt.Errorf("crypto: data key is %d bytes, expected %d", len(dek), DEKSize)
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: creating AEAD: %w", err)
	}

	return gcm, nil
}
