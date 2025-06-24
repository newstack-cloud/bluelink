package auth

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/sigv1"
	"github.com/stretchr/testify/suite"
)

type SigV1ServiceSuite struct {
	suite.Suite
}

func (s *SigV1ServiceSuite) Test_check_verifies_a_valid_signature() {
	clock := &testClock{timestamp: testTimestamp}
	keyPair := &sigv1.KeyPair{
		KeyID:     "test-key-id",
		SecretKey: "test-secret_key",
	}
	keyPairs := map[string]*sigv1.KeyPair{
		"test-key-id": keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value")

	signatureHeader, err := sigv1.CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	headers.Set(sigv1.SignatureHeaderName, signatureHeader)

	service := NewSigV1Service(
		keyPairsToSimpleMap(keyPairs),
		clock,
		/* options */ nil,
	)

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().NoError(err)
}

func (s *SigV1ServiceSuite) Test_check_fails_for_invalid_signature_returning_expected_error_type() {
	clock := &testClock{timestamp: testTimestamp}
	keyID := "test-key-id-4"
	keyPair := &sigv1.KeyPair{
		KeyID:     keyID,
		SecretKey: "test-secret_key-4",
	}
	keyPairs := map[string]*sigv1.KeyPair{
		keyID: keyPair,
	}

	headers := make(http.Header)
	headers.Set("X-Custom-Header", "custom-value-4")
	signatureHeader, err := sigv1.CreateSignatureHeader(
		keyPair,
		headers,
		[]string{"X-Custom-Header"},
		clock,
	)
	s.Require().NoError(err)

	invalidSignatureHeader := strings.Replace(signatureHeader, keyID, "invalid-key-id", 1)

	headers.Set(sigv1.SignatureHeaderName, invalidSignatureHeader)

	service := NewSigV1Service(
		keyPairsToSimpleMap(keyPairs),
		clock,
		/* options */ nil,
	)

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().Error(err)
	authErr, ok := err.(*Error)
	s.Require().True(ok)
	signatureErr, ok := authErr.ChildErr.(*sigv1.Error)
	s.Require().True(ok)
	s.Assert().Equal(sigv1.ErrCodeInvalidKeyID, signatureErr.Code)
}

func keyPairsToSimpleMap(keyPairs map[string]*sigv1.KeyPair) map[string]string {
	simpleMap := make(map[string]string)
	for keyID, keyPair := range keyPairs {
		simpleMap[keyID] = keyPair.SecretKey
	}
	return simpleMap
}

func TestSigV1ServiceSuite(t *testing.T) {
	suite.Run(t, new(SigV1ServiceSuite))
}
