// Package azurekeyvault provides a crypto.KeyProvider backed by Azure Key
// Vault or Managed HSM, the key-encryption key staying in the vault and never
// reaching this process.
//
// Key Vault has no generate-data-key call, so each data key is generated
// locally from the system random source and sent to the vault to be wrapped.
// The plaintext key therefore exists in this process either way, for as long
// as it takes to seal one object.
package azurekeyvault

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
)

// DefaultAlgorithm is the wrapping algorithm used when none is configured,
// suited to the RSA keys a standard Key Vault holds. A Managed HSM deployment
// using a symmetric key needs azkeys.EncryptionAlgorithmA256KW instead.
const DefaultAlgorithm = azkeys.EncryptionAlgorithmRSAOAEP256

// Client is the subset of the Key Vault keys API a key provider uses.
//
// Narrowed to these two calls so a caller owns the client and its credential,
// and so the provider is testable without reaching a vault.
type Client interface {
	WrapKey(
		ctx context.Context,
		name string,
		version string,
		parameters azkeys.KeyOperationParameters,
		options *azkeys.WrapKeyOptions,
	) (azkeys.WrapKeyResponse, error)
	UnwrapKey(
		ctx context.Context,
		name string,
		version string,
		parameters azkeys.KeyOperationParameters,
		options *azkeys.UnwrapKeyOptions,
	) (azkeys.UnwrapKeyResponse, error)
}

// Options configures a key provider beyond the key it wraps with.
type Options struct {
	// KeyVersion pins the key version used for wrapping.
	//
	// Pinning a version is worth doing. A wrapped key is opaque bytes that does not
	// name the version that produced it, so the version has to come from
	// somewhere: when pinned, it is recorded in the envelope and used again to
	// unwrap, and data sealed before a rotation stays readable. When empty
	// the vault's current version is used for both, and rotating the key
	// leaves everything sealed under the previous version unreadable.
	KeyVersion string
	// Algorithm overrides the wrapping algorithm. Empty means
	// DefaultAlgorithm.
	Algorithm azkeys.EncryptionAlgorithm
}

type keyProvider struct {
	client     Client
	keyName    string
	keyVersion string
	algorithm  azkeys.EncryptionAlgorithm
}

// NewKeyProvider returns a key provider that wraps data keys with the named
// vault key.
//
// The key name, and the version where one is pinned, is recorded in every
// envelope the provider seals, so it must stay stable for the life of the
// data.
func NewKeyProvider(client Client, keyName string, opts Options) (crypto.KeyProvider, error) {
	if client == nil {
		return nil, errors.New("azurekeyvault: a client is required")
	}
	if keyName == "" {
		return nil, errors.New("azurekeyvault: a key name is required")
	}

	algorithm := opts.Algorithm
	if algorithm == "" {
		algorithm = DefaultAlgorithm
	}

	return &keyProvider{
		client:     client,
		keyName:    keyName,
		keyVersion: opts.KeyVersion,
		algorithm:  algorithm,
	}, nil
}

// NewClient builds a Key Vault keys client for the vault at the given URL.
func NewClient(
	vaultURL string,
	credential azcore.TokenCredential,
	options *azkeys.ClientOptions,
) (*azkeys.Client, error) {
	return azkeys.NewClient(vaultURL, credential, options)
}

func (p *keyProvider) ID() string {
	if p.keyVersion == "" {
		return fmt.Sprintf("azurekeyvault:%s", p.keyName)
	}

	return fmt.Sprintf("azurekeyvault:%s/%s", p.keyName, p.keyVersion)
}

// The scope is unused as a round trip per object is the price of the key never
// leaving the vault, and a provider that cached keys per scope would be
// trading that for latency. Reuse belongs here rather than in callers if it is
// ever needed.
func (p *keyProvider) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	dek := make([]byte, crypto.DEKSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, nil, fmt.Errorf("azurekeyvault: generating a data key: %w", err)
	}

	out, err := p.client.WrapKey(
		ctx,
		p.keyName,
		p.keyVersion,
		azkeys.KeyOperationParameters{
			Algorithm: &p.algorithm,
			Value:     dek,
		},
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("azurekeyvault: wrapping a data key: %w", err)
	}

	if len(out.Result) == 0 {
		return nil, nil, errors.New("azurekeyvault: the vault returned an empty wrapped data key")
	}

	return dek, out.Result, nil
}

func (p *keyProvider) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	keyVersion, err := p.versionForUnwrap(kekID)
	if err != nil {
		return nil, err
	}

	out, err := p.client.UnwrapKey(
		ctx,
		p.keyName,
		keyVersion,
		azkeys.KeyOperationParameters{
			Algorithm: &p.algorithm,
			Value:     wrapped,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("azurekeyvault: unwrapping a data key: %w", err)
	}

	if len(out.Result) != crypto.DEKSize {
		return nil, fmt.Errorf(
			"azurekeyvault: the vault returned a %d byte data key, expected %d",
			len(out.Result),
			crypto.DEKSize,
		)
	}

	return out.Result, nil
}

// Unwrapping uses the version the envelope was sealed with rather than the
// configured one, so a rotation does not strand data sealed under the previous
// version: the vault keeps old versions available for unwrapping, and the
// envelope is the only record of which one applies.
func (p *keyProvider) versionForUnwrap(kekID string) (string, error) {
	name, version, _ := strings.Cut(
		strings.TrimPrefix(kekID, "azurekeyvault:"),
		"/",
	)

	if !strings.HasPrefix(kekID, "azurekeyvault:") || name != p.keyName {
		return "", fmt.Errorf(
			"azurekeyvault: envelope was sealed with key %q, this provider holds %q",
			kekID,
			p.ID(),
		)
	}

	return version, nil
}
