package crypto

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// A key set holds the key new data is sealed with alongside keys that only still exist to
// read data sealed earlier.
type keySet struct {
	active  KeyProvider
	retired []KeyProvider
}

// NewKeySet returns a provider that seals with the active key and unwraps with the active
// key or any retired one.
func NewKeySet(active KeyProvider, retired ...KeyProvider) (KeyProvider, error) {
	if active == nil {
		return nil, errors.New("crypto: a key set needs an active key")
	}

	for _, provider := range retired {
		if provider == nil {
			return nil, errors.New("crypto: a key set cannot hold a nil retired key")
		}
	}

	return &keySet{active: active, retired: retired}, nil
}

func (k *keySet) ID() string {
	return k.active.ID()
}

func (k *keySet) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	return k.active.GenerateDataKey(ctx, scope)
}

// Unwrapping tries the key the envelope names, then falls back to trying each key in turn.
//
// The fallback is for providers whose key id is not the whole story, for example, an Azure Key Vault
// envelope names the key version it was sealed with, which its provider resolves itself and
// which will not match the configured id after a rotation.
func (k *keySet) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	providers := append([]KeyProvider{k.active}, k.retired...)

	for _, provider := range providers {
		if provider.ID() == kekID {
			return provider.UnwrapKey(ctx, kekID, wrapped)
		}
	}

	var failures []string
	for _, provider := range providers {
		dek, err := provider.UnwrapKey(ctx, kekID, wrapped)
		if err == nil {
			return dek, nil
		}
		failures = append(failures, fmt.Sprintf("%s: %s", provider.ID(), err))
	}

	return nil, fmt.Errorf(
		"crypto: no configured key could unwrap a data key sealed with %q (%s)",
		kekID,
		strings.Join(failures, "; "),
	)
}
