// Package gcpkms provides a crypto.KeyProvider backed by Google Cloud KMS,
// the key-encryption key staying in KMS and never reaching this process.
//
// Cloud KMS has no generate-data-key call, so each data key is generated
// locally from the system random source and sent to KMS to be wrapped. The
// plaintext key therefore exists in this process either way, for as long as it
// takes to seal one object.
package gcpkms

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"hash/crc32"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Client is the subset of the Cloud KMS API a key provider uses.
//
// Narrowed to these two calls so a caller owns the client lifecycle, closing
// it being its responsibility, and so the provider is testable without
// reaching Google Cloud.
type Client interface {
	Encrypt(
		ctx context.Context,
		req *kmspb.EncryptRequest,
		opts ...gax.CallOption,
	) (*kmspb.EncryptResponse, error)
	Decrypt(
		ctx context.Context,
		req *kmspb.DecryptRequest,
		opts ...gax.CallOption,
	) (*kmspb.DecryptResponse, error)
}

type keyProvider struct {
	client  Client
	keyName string
}

// NewKeyProvider returns a key provider that wraps data keys with the Cloud
// KMS key the given resource name identifies, in the form
// "projects/{project}/locations/{location}/keyRings/{ring}/cryptoKeys/{key}".
//
// Naming the key rather than one of its versions lets KMS rotate versions
// underneath: new data keys are wrapped with the primary version, and old ones
// unwrap with whichever version sealed them. The name is recorded in every
// envelope, so it must stay stable for the life of the data.
func NewKeyProvider(client Client, keyName string) (crypto.KeyProvider, error) {
	if client == nil {
		return nil, errors.New("gcpkms: a client is required")
	}
	if keyName == "" {
		return nil, errors.New("gcpkms: a key resource name is required")
	}

	return &keyProvider{client: client, keyName: keyName}, nil
}

// NewClient builds a Cloud KMS client.
// The caller is responsible for closing it.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*kms.KeyManagementClient, error) {
	return kms.NewKeyManagementClient(ctx, opts...)
}

func (p *keyProvider) ID() string {
	return fmt.Sprintf("gcpkms:%s", p.keyName)
}

// The scope is unused: a round trip per object is the price of the key never
// leaving KMS, and a provider that cached keys per scope would be trading that
// for latency. Reuse belongs here rather than in callers if it is ever needed.
func (p *keyProvider) GenerateDataKey(
	ctx context.Context,
	scope string,
) (plaintext []byte, wrapped []byte, err error) {
	dek := make([]byte, crypto.DEKSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, nil, fmt.Errorf("gcpkms: generating a data key: %w", err)
	}

	// Checksums on both legs are an extra layer only Cloud KMS offers of the three key
	// services, beyond the integrity TLS and AES-GCM already provide. They catch a key
	// corrupted outside TLS's span, in process memory or at a terminating hop, at the point
	// of writing rather than on a later read, when the state it protects is past recovering.
	out, err := p.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:            p.keyName,
		Plaintext:       dek,
		PlaintextCrc32C: checksum(dek),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("gcpkms: wrapping a data key: %w", err)
	}

	if !out.VerifiedPlaintextCrc32C {
		return nil, nil, errors.New(
			"gcpkms: kms did not verify the data key checksum, the request was corrupted in transit",
		)
	}

	if out.CiphertextCrc32C.GetValue() != checksum(out.Ciphertext).GetValue() {
		return nil, nil, errors.New(
			"gcpkms: wrapped data key failed its checksum, the response was corrupted in transit",
		)
	}

	return dek, out.Ciphertext, nil
}

func (p *keyProvider) UnwrapKey(
	ctx context.Context,
	kekID string,
	wrapped []byte,
) ([]byte, error) {
	if kekID != p.ID() {
		return nil, fmt.Errorf(
			"gcpkms: envelope was sealed with key %q, this provider holds %q",
			kekID,
			p.ID(),
		)
	}

	out, err := p.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:             p.keyName,
		Ciphertext:       wrapped,
		CiphertextCrc32C: checksum(wrapped),
	})
	if err != nil {
		return nil, fmt.Errorf("gcpkms: unwrapping a data key: %w", err)
	}

	if out.PlaintextCrc32C.GetValue() != checksum(out.Plaintext).GetValue() {
		return nil, errors.New(
			"gcpkms: data key failed its checksum, the response was corrupted in transit",
		)
	}

	if len(out.Plaintext) != crypto.DEKSize {
		return nil, fmt.Errorf(
			"gcpkms: kms returned a %d byte data key, expected %d",
			len(out.Plaintext),
			crypto.DEKSize,
		)
	}

	return out.Plaintext, nil
}

func checksum(data []byte) *wrapperspb.Int64Value {
	table := crc32.MakeTable(crc32.Castagnoli)

	return wrapperspb.Int64(int64(crc32.Checksum(data, table)))
}
