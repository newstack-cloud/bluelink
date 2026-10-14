package statestore

import (
	"context"
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/common/crypto"
)

// ReencryptOptions describes a pass over a store that re-seals every object under one key.
//
// Covers the three transitions a deployment needs: encrypting a store that was written in
// plaintext, moving a store from one key to another, and decrypting one to store state in
// plaintext.
type ReencryptOptions struct {
	// Keys names where the store's objects live, built with the same prefix the state is
	// persisted under. The encryption salt is found through it and left alone, since the
	// active key may have been derived from it.
	Keys KeyBuilder
	// Source opens objects as they are now. Required unless every object is plaintext, and
	// it should be a crypto.KeySet holding both the old and new keys where a pass is being
	// resumed, since objects already moved are sealed with the new one.
	Source crypto.KeyProvider
	// Target seals objects as they will be. Nil writes plaintext, which backs out of encryption
	// and is worth a deliberate confirmation from whoever asked for it.
	Target crypto.KeyProvider
	// DryRun reports what a pass would do without writing anything.
	DryRun bool
	// Observe is called as each object is dealt with, for progress output on a store large
	// enough that a pass is not instant.
	Observe func(key string, outcome ReencryptOutcome)
}

// ReencryptOutcome is what a pass did with one object.
type ReencryptOutcome string

const (
	// ReencryptOutcomeReencrypted means an object sealed with one key was re-sealed with
	// another.
	ReencryptOutcomeReencrypted ReencryptOutcome = "reencrypted"
	// ReencryptOutcomeEncrypted means a plaintext object was sealed for the first time.
	ReencryptOutcomeEncrypted ReencryptOutcome = "encrypted"
	// ReencryptOutcomeDecrypted means an object was written back as plaintext.
	ReencryptOutcomeDecrypted ReencryptOutcome = "decrypted"
	// ReencryptOutcomeAlreadyCurrent means an object was already sealed with the target key,
	// which is what makes an interrupted pass safe to run again.
	ReencryptOutcomeAlreadyCurrent ReencryptOutcome = "already-current"
	// ReencryptOutcomeSkipped means an object was deliberately left alone, which today is
	// the encryption salt.
	ReencryptOutcomeSkipped ReencryptOutcome = "skipped"
)

// ReencryptReport counts what a pass did.
type ReencryptReport struct {
	Reencrypted    int
	Encrypted      int
	Decrypted      int
	AlreadyCurrent int
	Skipped        int
}

// Reencrypt re-seals every object in a store under the target key.
//
// Objects are dealt with one at a time and each is independent, so an interrupted pass can be
// run again: objects already under the target key are recognised and left alone. What it does
// not do is make the pass atomic. An object is written in place, so a failure part-way through
// leaves a store holding a mix of keys, which the Source key set can still read but which a
// deployment configured with only the new key cannot. Take a copy of the store first.
//
// The store must not be in use. The in-memory state a container holds would be written back
// over this pass, and nothing here can detect that.
func Reencrypt(
	ctx context.Context,
	storage Storage,
	opts ReencryptOptions,
) (*ReencryptReport, error) {
	if opts.Target == nil && opts.Source == nil {
		return nil, errors.New("statestore: re-encrypting needs a source key, a target key, or both")
	}

	keys, err := storage.List(ctx, opts.Keys.Prefix())
	if err != nil {
		return nil, fmt.Errorf("statestore: listing the store to re-encrypt: %w", err)
	}

	report := &ReencryptReport{}
	saltKey := opts.Keys.EncryptionSalt()

	for _, key := range keys {
		outcome, err := reencryptObject(ctx, storage, key, saltKey, &opts)
		if err != nil {
			return report, err
		}

		countOutcome(report, outcome)
		if opts.Observe != nil {
			opts.Observe(key, outcome)
		}
	}

	return report, nil
}

func reencryptObject(
	ctx context.Context,
	storage Storage,
	key string,
	saltKey string,
	opts *ReencryptOptions,
) (ReencryptOutcome, error) {
	// The salt is an input to deriving the key that protects everything else, so it is
	// neither encrypted nor replaced.
	if key == saltKey {
		return ReencryptOutcomeSkipped, nil
	}

	stored, err := storage.Read(ctx, key)
	if err != nil {
		return "", fmt.Errorf("statestore: reading %q to re-encrypt: %w", key, err)
	}

	plaintext, outcome, err := openForReencrypt(ctx, storage, key, stored, opts)
	if err != nil || outcome == ReencryptOutcomeAlreadyCurrent {
		return outcome, err
	}

	if opts.DryRun {
		return outcome, nil
	}

	if err := writeReencrypted(ctx, storage, key, plaintext, opts); err != nil {
		return "", err
	}

	return outcome, nil
}

func openForReencrypt(
	ctx context.Context,
	storage Storage,
	key string,
	stored []byte,
	opts *ReencryptOptions,
) ([]byte, ReencryptOutcome, error) {
	if !crypto.IsEnvelope(stored) {
		if opts.Target == nil {
			return nil, ReencryptOutcomeSkipped, nil
		}

		return stored, ReencryptOutcomeEncrypted, nil
	}

	keyID, err := crypto.EnvelopeKeyID(stored)
	if err != nil {
		return nil, "", fmt.Errorf("statestore: reading the key id of %q: %w", key, err)
	}

	if opts.Target != nil && keyID == opts.Target.ID() {
		return nil, ReencryptOutcomeAlreadyCurrent, nil
	}

	if opts.Source == nil {
		return nil, "", fmt.Errorf(
			"statestore: %q is sealed with key %q and no source key was configured to open it",
			key,
			keyID,
		)
	}

	plaintext, err := NewEncryptedStorage(storage, opts.Source).Read(ctx, key)
	if err != nil {
		return nil, "", err
	}

	if opts.Target == nil {
		return plaintext, ReencryptOutcomeDecrypted, nil
	}

	return plaintext, ReencryptOutcomeReencrypted, nil
}

// Written through the same decorator the engine persists with, so an object re-sealed here is
// indistinguishable from one the engine would have written.
func writeReencrypted(
	ctx context.Context,
	storage Storage,
	key string,
	plaintext []byte,
	opts *ReencryptOptions,
) error {
	if opts.Target == nil {
		if err := storage.Write(ctx, key, plaintext); err != nil {
			return fmt.Errorf("statestore: writing %q as plaintext: %w", key, err)
		}

		return nil
	}

	if err := NewEncryptedStorage(storage, opts.Target).Write(ctx, key, plaintext); err != nil {
		return err
	}

	return nil
}

func countOutcome(report *ReencryptReport, outcome ReencryptOutcome) {
	switch outcome {
	case ReencryptOutcomeReencrypted:
		report.Reencrypted++
	case ReencryptOutcomeEncrypted:
		report.Encrypted++
	case ReencryptOutcomeDecrypted:
		report.Decrypted++
	case ReencryptOutcomeAlreadyCurrent:
		report.AlreadyCurrent++
	case ReencryptOutcomeSkipped:
		report.Skipped++
	}
}
