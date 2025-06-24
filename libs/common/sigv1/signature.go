package sigv1

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// SignatureHeaderName is the name of the HTTP header used for
	// Bluelink Signature v1.
	SignatureHeaderName = "Bluelink-Signature-V1"
	// DateHeaderName is the name of the HTTP header used to
	// store the date as a UNIX timestamp in seconds.
	DateHeaderName = "Bluelink-Date"
	// DefaultClockSkew is the default clock skew in seconds
	// used to determine if the request is within the
	// acceptable time window.
	DefaultClockSkew = 300
)

// VerifyOptions contains options for verifying a
// Bluelink Signature v1 signature.
type VerifyOptions struct {
	// The maximum clock skew in seconds.
	ClockSkew int
}

// VerifiySignature verifies a signature header with
// [Bluelink Signature v1](https://bluelink.dev/docs/auth/signature-v1).
// Returns nil if the signature is valid, or an error if
// the signature is invalid or cannot be verified.
//
// If options is nil, the default options are used.
func VerifySignature(
	keyPairs map[string]*KeyPair,
	headers http.Header,
	clock core.Clock,
	options *VerifyOptions,
) error {
	finalOptions := prepareOptions(options)
	signatureHeader := headers.Get(SignatureHeaderName)
	if signatureHeader == "" {
		return ErrSignatureHeaderMissing
	}

	parts, err := unpackSignature(signatureHeader)
	if err != nil {
		return err
	}

	keyPair, ok := keyPairs[parts.KeyID]
	if !ok {
		return ErrInvalidKeyID
	}

	message, err := createMessage(keyPair, headers, parts.Headers)
	if err != nil {
		return err
	}

	err = verifyMessage(keyPair, message, parts.Signature)
	if err != nil {
		return err
	}

	return checkTimeWindow(
		headers,
		clock,
		finalOptions,
	)
}

// CreateSignatureHeader creates a signature header
// value to be attached to a request for
// [Bluelink Signature v1](https://bluelink.dev/docs/auth/signature-v1).
// This functions will return the value of the signature header that should be set
// in the `Bluelink-Signature-V1` header of the request.
//
// The `Bluelink-Date` header does not need to be set in the provided headers,
// as it will be automatically added to the signature message using the provided
// clock and inserted into the provided headers map.
// The provided `http.Header` map will be modified to include the `Bluelink-Date` header
// with the current date in seconds since the epoch if it is not already present.
func CreateSignatureHeader(
	keyPair *KeyPair,
	headers http.Header,
	customHeaderNames []string,
	clock core.Clock,
) (string, error) {
	if headers.Get(DateHeaderName) == "" {
		populateDateHeader(headers, clock)
	}

	message, err := createMessage(keyPair, headers, customHeaderNames)
	if err != nil {
		return "", err
	}
	signature := signMessage(keyPair, message)
	signatureHeaderNames := prepareSignatureHeaderNames(customHeaderNames)

	return fmt.Sprintf(
		"keyId=\"%s\", headers=\"%s\", signature=\"%s\"",
		keyPair.KeyID,
		signatureHeaderNames,
		signature,
	), nil
}

func populateDateHeader(headers http.Header, clock core.Clock) {
	date := clock.Now().Unix()
	headers.Set(DateHeaderName, fmt.Sprintf("%d", date))
}

func prepareSignatureHeaderNames(customHeaderNames []string) string {
	headerNames := make([]string, len(customHeaderNames))
	for i, headerName := range customHeaderNames {
		headerNames[i] = strings.ToLower(headerName)
	}

	finalHeaderNames := append(
		[]string{strings.ToLower(DateHeaderName)},
		headerNames...,
	)

	return strings.Join(finalHeaderNames, " ")
}

func verifyMessage(
	keyPair *KeyPair,
	message []byte,
	signature string,
) error {
	decodedSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return ErrFailedToDecodeSignature
	}

	mac := hmac.New(sha256.New, []byte(keyPair.SecretKey))
	mac.Write(message)
	expectedMac := mac.Sum(nil)

	if !hmac.Equal(decodedSignature, expectedMac) {
		return ErrInvalidSignature
	}

	return nil
}

func signMessage(keyPair *KeyPair, message []byte) string {
	mac := hmac.New(sha256.New, []byte(keyPair.SecretKey))
	mac.Write(message)
	signature := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(signature)
}

func unpackSignature(signatureHeader string) (*SignatureParts, error) {
	parts := strings.Split(signatureHeader, ",")
	if len(parts) != 3 {
		return nil, ErrInvalidSignatureHeaderFormat
	}

	keyID, err := unpackKeyID(parts[0])
	if err != nil {
		return nil, err
	}

	headers, err := unpackHeaders(parts[1])
	if err != nil {
		return nil, err
	}

	signature, err := unpackSignatureValue(parts[2])
	if err != nil {
		return nil, err
	}

	return &SignatureParts{
		KeyID:     keyID,
		Headers:   headers,
		Signature: signature,
	}, nil
}

func unpackKeyID(keyIDHeaderPart string) (string, error) {
	parts := strings.Split(keyIDHeaderPart, "=")
	if len(parts) != 2 {
		return "", ErrInvalidSignatureHeaderFormat
	}

	if strings.TrimSpace(parts[0]) != "keyId" {
		return "", ErrInvalidSignatureHeaderFormat
	}

	if !strings.HasPrefix(parts[1], "\"") ||
		!strings.HasSuffix(parts[1], "\"") {
		return "", ErrInvalidSignatureHeaderFormat
	}

	// Remove the quotes around the key ID.
	return parts[1][1 : len(parts[1])-1], nil
}

func unpackHeaders(headersPart string) ([]string, error) {
	parts := strings.Split(headersPart, "=")
	if len(parts) != 2 {
		return nil, ErrInvalidSignatureHeaderFormat
	}

	if strings.TrimSpace(parts[0]) != "headers" {
		return nil, ErrInvalidSignatureHeaderFormat
	}

	if !strings.HasPrefix(parts[1], "\"") ||
		!strings.HasSuffix(parts[1], "\"") {
		return nil, ErrInvalidSignatureHeaderFormat
	}

	headers := strings.Split(
		// Remove the quotes around space-separated headers.
		strings.TrimSpace(parts[1][1:len(parts[1])-1]),
		" ",
	)
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	return headers, nil
}

func unpackSignatureValue(signaturePart string) (string, error) {
	parts := strings.Split(signaturePart, "=")
	if len(parts) != 2 {
		return "", ErrInvalidSignatureHeaderFormat
	}

	if strings.TrimSpace(parts[0]) != "signature" {
		return "", ErrInvalidSignatureHeaderFormat
	}

	if !strings.HasPrefix(parts[1], "\"") ||
		!strings.HasSuffix(parts[1], "\"") {
		return "", ErrInvalidSignatureHeaderFormat
	}

	// Remove the quotes around the signature.
	return parts[1][1 : len(parts[1])-1], nil
}

func createMessage(
	keyPair *KeyPair,
	headers http.Header,
	customHeaderNames []string,
) ([]byte, error) {
	date, err := extractDateFromHeader(headers)
	if err != nil {
		return nil, err
	}

	filteredCustomHeaderNames := core.Filter(
		customHeaderNames,
		func(headerName string, _ int) bool {
			return !strings.EqualFold(headerName, DateHeaderName)
		},
	)
	customHeaders, err := prepareCustomHeadersForMessage(
		filteredCustomHeaderNames,
		headers,
	)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf(
		"%s,%s=%d%s",
		keyPair.KeyID,
		strings.ToLower(DateHeaderName),
		date,
		customHeaders,
	)

	return []byte(message), nil
}

func extractDateFromHeader(headers http.Header) (int64, error) {
	dateHeader := headers.Get(DateHeaderName)
	if dateHeader == "" {
		return 0, ErrDateHeaderMissing
	}

	date, err := strconv.ParseInt(dateHeader, 10, 64)
	if err != nil {
		return 0, ErrInvalidDateHeader
	}

	return date, nil
}

func prepareCustomHeadersForMessage(
	customHeaderNames []string,
	requestHeaders http.Header,
) (string, error) {
	if len(customHeaderNames) == 0 {
		return "", nil
	}

	var customHeaders strings.Builder

	for _, headerName := range customHeaderNames {
		headerValue := requestHeaders.Get(headerName)
		if headerValue == "" {
			return "", errCustomHeaderMissing(headerName)
		}

		customHeaders.WriteString(
			fmt.Sprintf(",%s=%s", strings.ToLower(headerName), headerValue),
		)
	}

	return customHeaders.String(), nil
}

func checkTimeWindow(
	headers http.Header,
	clock core.Clock,
	options *VerifyOptions,
) error {
	currentTime := clock.Now().Unix()
	providedDate, err := extractDateFromHeader(headers)
	if err != nil {
		return err
	}

	if currentTime > providedDate+int64(options.ClockSkew) ||
		currentTime < providedDate-int64(options.ClockSkew) {
		return ErrSignatureExpired
	}

	return nil
}

func prepareOptions(options *VerifyOptions) *VerifyOptions {
	if options == nil {
		return &VerifyOptions{
			ClockSkew: DefaultClockSkew,
		}
	}

	return options
}
