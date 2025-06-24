package auth

import (
	"context"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/newstack-cloud/bluelink/libs/common/sigv1"
)

type sigV1Service struct {
	keyPairs map[string]*sigv1.KeyPair
	clock    core.Clock
	options  *sigv1.VerifyOptions
}

// NewSigV1Service creates a new instance of the Bluelink Signature V1 service.
// It takes a map of key pairs (key ID -> secret key), a clock for time verification,
// and options for customising configuration such as clock skew.
// Options can be nil, in which case default options will be used.
func NewSigV1Service(
	keyPairs map[string]string,
	clock core.Clock,
	options *sigv1.VerifyOptions,
) Checker {
	return &sigV1Service{
		keyPairs: toSigV1KeyPairs(keyPairs),
		clock:    clock,
		options:  options,
	}
}

// Check verifies a Bluelink Signature V1 header,
// where the date and other custom headers
// are expected to be in the provided request headers.
// This method will return an error if verification fails,
// or nil if it succeeds.
func (s *sigV1Service) Check(ctx context.Context, headers http.Header) error {
	err := sigv1.VerifySignature(
		s.keyPairs,
		headers,
		s.clock,
		s.options,
	)
	return handleVerifySigV1Error(err)
}

func handleVerifySigV1Error(err error) error {
	if err != nil {
		// All errors returned from the sigv1 package are
		// of the type *sigv1.Error, so we can safely
		// say that all errors can be treated as auth/bad request errors.
		// A malformed signature header will be treated the same as
		// a missing or invalid signature header, resulting in a 401
		// response.
		return &Error{
			ChildErr: err,
		}
	}

	return nil
}

func toSigV1KeyPairs(keyPairs map[string]string) map[string]*sigv1.KeyPair {
	sigV1KeyPairs := make(map[string]*sigv1.KeyPair)
	for keyID, secretKey := range keyPairs {
		sigV1KeyPairs[keyID] = &sigv1.KeyPair{
			KeyID:     keyID,
			SecretKey: secretKey,
		}
	}
	return sigV1KeyPairs
}
