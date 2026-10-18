package auth

import (
	"context"
	"net/http"
)

// Checker provides an interface for carrying out
// an authentication check given a set of request
// headers.
type Checker interface {
	// Check carries out an authentication check
	// using the provided request headers.
	// It returns an error of the `auth.Error` type
	// if the check fails, or the principal the request
	// authenticated as if it succeeds.
	// Other error types should be considered
	// as internal server errors.
	//
	// The principal is the only record of who acted as the credential it was
	// established from is discarded once the check has passed.
	Check(ctx context.Context, headers http.Header) (*Principal, error)
}
