package sigv1

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SignatureV1Suite struct {
	suite.Suite
}

func (s *SignatureV1Suite) Test_creates_signature_header() {
	clock := &testClock{timestamp: testTimestamp}
	keyPair := &KeyPair{
		KeyID:     "test-key-id",
		SecretKey: "test-secret_key",
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value")
	customHeaderNames := []string{"X-Custom-Header"}

	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		customHeaderNames,
		clock,
	)
	s.Require().NoError(err)

	expectedSignatureHeader := "keyId=\"test-key-id\", headers=\"bluelink-date x-custom-header\", " +
		"signature=\"v0FTyFsVEXJwi6DC9NNNo8axZ7N-v1l1ngkZN2k72xQ\""
	s.Assert().Equal(expectedSignatureHeader, signatureHeader)
}

func (s *SignatureV1Suite) Test_returns_expected_error_when_custom_header_is_missing() {
	clock := &testClock{timestamp: testTimestamp}
	keyPair := &KeyPair{
		KeyID:     "test-key-id",
		SecretKey: "test-secret_key",
	}
	headers := make(http.Header)
	// Custom header not set in the headers.

	customHeaderNames := []string{"X-Custom-Header"}

	_, err := CreateSignatureHeader(
		keyPair,
		headers,
		customHeaderNames,
		clock,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeCustomHeaderMissing, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_verify_valid_signature_header() {
	clock := &testClock{timestamp: testTimestamp}
	keyPair := &KeyPair{
		KeyID:     "test-key-id",
		SecretKey: "test-secret_key",
	}
	keyPairs := map[string]*KeyPair{
		"test-key-id": keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value")

	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock,
		/* options */ nil,
	)
	s.Require().NoError(err)
	// Make sure that the date header set in the headers gets the current
	// timestamp from the clock.
	s.Assert().Equal(fmt.Sprintf("%d", testTimestamp), headers.Get(DateHeaderName))
}

func (s *SignatureV1Suite) Test_verify_valid_signature_header_With_time_difference_within_skew_1() {
	clock := &testClock{timestamp: testTimestamp}
	clock2 := &testClock{
		// -3 minutes from the original timestamp,
		// but within the default clock skew of 5 minutes.
		timestamp: testTimestamp + 180,
	}
	keyID := "test-key-id"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value")
	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock2,
		/* options */ nil,
	)
	s.Require().NoError(err)
}

func (s *SignatureV1Suite) Test_verify_valid_signature_header_With_time_difference_within_skew_2() {
	clock := &testClock{timestamp: testTimestamp}
	clock2 := &testClock{
		// +4 minutes from the original timestamp,
		// but within the default clock skew of 5 minutes.
		timestamp: testTimestamp + 240,
	}
	keyID := "test-key-id-2"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-2",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-2")
	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock2,
		/* options */ nil,
	)
	s.Require().NoError(err)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_that_has_expired() {
	clock := &testClock{timestamp: testTimestamp}
	clock2 := &testClock{
		// +6 minutes, beyond the default clock skew of 5 minutes.
		timestamp: testTimestamp + 360,
	}
	keyID := "test-key-id-3"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-3",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-3")
	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock2,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeSignatureExpired, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_for_invalid_key_id() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-4",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")
	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	invalidSignatureHeader := strings.Replace(signatureHeader, keyID, "invalid-key-id", 1)

	headers.Set(SignatureHeaderName, invalidSignatureHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeInvalidKeyID, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_for_invalid_signature_signed_with_a_different_secret_key() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	signKeyPairs := map[string]*KeyPair{
		keyID: {
			KeyID:     keyID,
			SecretKey: "test-other_secret_key",
		},
	}
	verifyKeyPairs := map[string]*KeyPair{
		keyID: {
			KeyID:     keyID,
			SecretKey: "test-secret_key-4",
		},
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")
	signatureHeader, err := CreateSignatureHeader(
		signKeyPairs[keyID],
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		verifyKeyPairs,
		headers,
		clock,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeInvalidSignature, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_for_invalid_signature_due_to_date_header_mismatch() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-4",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")
	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	otherHeaders := headers.Clone()
	otherHeaders.Set(
		DateHeaderName,
		// A different date header value is set to test expected error.
		fmt.Sprintf("%d", testTimestamp+60),
	)
	otherHeaders.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		otherHeaders,
		clock,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeInvalidSignature, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_for_invalid_signature_due_to_custom_header_mismatch() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-4",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")

	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	otherHeaders := headers.Clone()
	otherHeaders.Set(
		"X-Custom-Header",
		// A different custom header value is set to test expected error.
		"custom-value-5",
	)
	otherHeaders.Set(SignatureHeaderName, signatureHeader)

	err = VerifySignature(
		keyPairs,
		otherHeaders,
		clock,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeInvalidSignature, signatureErr.Code)
}

func (s *SignatureV1Suite) Test_fails_verifying_signature_for_invalid_signature_that_is_not_a_base64_encoded_string() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	keyPair := &KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-4",
	}
	keyPairs := map[string]*KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")

	signatureHeader, err := CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	finalSigHeader := strings.Replace(
		signatureHeader,
		"signature=\"",
		"signature=\"invalid",
		1,
	)
	headers.Set(SignatureHeaderName, finalSigHeader)

	err = VerifySignature(
		keyPairs,
		headers,
		clock,
		/* options */ nil,
	)
	s.Require().Error(err)
	signatureErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Equal(ErrCodeInvalidSignature, signatureErr.Code)
}

func TestSignatureV1Suite(t *testing.T) {
	suite.Run(t, new(SignatureV1Suite))
}
