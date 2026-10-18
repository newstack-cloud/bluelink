package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
)

const (
	// BluelinkAPIKeyHeaderName is the name of the header
	// used to pass the API key for authentication.
	BluelinkAPIKeyHeaderName = "Bluelink-Api-Key"
)

type apiKeyService struct {
	// Digests of the configured keys, derived once at construction rather than per
	// request. Comparing digests rather than the keys themselves equalises length, so the
	// comparison reveals nothing about how long a configured key is.
	apiKeyDigests [][sha256.Size]byte
}

// NewAPIKeyService creates a new service that checks
// if the provided API key is valid against a list of
// known API keys.
func NewAPIKeyService(config *core.AuthConfig) Checker {
	digests := make([][sha256.Size]byte, 0, len(config.APIKeys))
	for _, apiKey := range config.APIKeys {
		digests = append(digests, sha256.Sum256([]byte(apiKey)))
	}

	return &apiKeyService{
		apiKeyDigests: digests,
	}
}

// Check verifies if the provided API key is valid.
// It returns an error if the key is invalid,
// or nil if it is valid.
func (s *apiKeyService) Check(
	ctx context.Context,
	headers http.Header,
) (*Principal, error) {
	apiKey := headers.Get(BluelinkAPIKeyHeaderName)
	if apiKey == "" {
		return nil, &Error{
			ChildErr: errors.New("missing API key"),
		}
	}

	if !s.matchesConfiguredKey(apiKey) {
		return nil, &Error{
			ChildErr: errors.New("invalid API key"),
		}
	}

	return &Principal{
		Method:  MethodAPIKey,
		Subject: apiKeyFingerprint(apiKey),
	}, nil
}

// A prefix of the key's digest, long enough to tell configured keys apart in a record.
//
// Safe only to the extent the key has entropy. The digest is unsalted and fast, so a record
// holding one can be used to test guesses offline; truncating it changes nothing about that,
// since 64 bits is far more than enough to confirm a correct guess. The engine warns at
// startup about keys short enough for this to matter, which is the most it can check.
func apiKeyFingerprint(apiKey string) string {
	digest := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("sha256:%x", digest[:8])
}

// Compared in constant time, so that the time taken to reject a key does not reveal how much
// of it was correct.
//
// Every configured key is compared with no early exit on a match, since returning as soon as
// one matched would make the time taken depend on that key's position in the list.
func (s *apiKeyService) matchesConfiguredKey(apiKey string) bool {
	presented := sha256.Sum256([]byte(apiKey))

	matched := 0
	for _, digest := range s.apiKeyDigests {
		matched |= subtle.ConstantTimeCompare(presented[:], digest[:])
	}

	return matched == 1
}
