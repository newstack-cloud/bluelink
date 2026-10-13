// Package crypto provides envelope encryption primitives shared by components that need to
// protect data at rest that includes a pluggable key-encryption-key provider, and the envelope format
// that wraps a data-encryption key alongside its ciphertext.
//
// The split exists so that a deployment configures key management once. The same provider
// that protects persisted state can protect any other recoverable secret a component holds.
package crypto

import "context"

// KeyProvider wraps and unwraps data-encryption keys with a key-encryption key.
//
// Implementations range from a passphrase-derived key held in the process to a cloud key
// management service that never releases the KEK at all. Callers generate a fresh DEK per
// object, encrypt with it, and persist only the wrapped form, so the KEK is the single thing
// a deployment has to protect.
type KeyProvider interface {
	// ID identifies the active KEK.
	//
	// Recorded in each envelope so a key can be rotated without rewriting everything already
	// encrypted, an envelope names the KEK it was sealed with, and a provider that recognises
	// more than one can still unwrap the older ones.
	ID() string

	// GenerateDataKey returns a fresh data-encryption key, in plaintext and wrapped form,
	// for data belonging to the given scope.
	//
	// The intention is one key per object, and the envelope layer calls this for every
	// object it seals. How much a caller puts in an object, and whether it needs keys
	// grouped more coarsely, varies by use case.
	//
	// The scope is that coarser grouping, naming what the data belongs to, whether that be
	// the instance for blueprint state or another category. This way, a provider can select
	// a per-scope key-encryption key, which is what lets one scope's data be made
	// unreadable by destroying a key.
	//
	// An implementation may reuse a key across objects within one scope to save round
	// trips, but must never share one across scopes, and must bound what a key covers by
	// object count, bytes and age: the envelope's nonce is then the only thing preventing
	// keystream reuse.
	GenerateDataKey(
		ctx context.Context,
		scope string,
	) (plaintext []byte, wrapped []byte, err error)

	// UnwrapKey decrypts a data-encryption key that was wrapped with the named KEK.
	//
	// The kekID comes from the envelope rather than from the provider's current state, so an
	// implementation must fail cleanly when asked for a key it does not have rather than
	// assume the active one.
	UnwrapKey(ctx context.Context, kekID string, wrapped []byte) ([]byte, error)
}

// DEKSize is the length in bytes of the data-encryption keys this package generates and
// expects, matching AES-256.
const DEKSize = 32
