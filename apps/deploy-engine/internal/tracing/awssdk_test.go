package tracing_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	"github.com/stretchr/testify/suite"
)

// Time the engine spends waiting on AWS has to be accounted for as such. The engine
// reaches object storage and key management through the SDK, and a deployment waiting on
// one of them is otherwise indistinguishable from one waiting on the engine itself.
type AWSSDKSuite struct {
	suite.Suite

	recorder *telemetry.Recorder
}

func (s *AWSSDKSuite) SetupTest() {
	s.recorder = telemetry.NewRecorder()
	telemetry.SetTracer(s.recorder)
}

func (s *AWSSDKSuite) TearDownTest() {
	telemetry.SetTracer(nil)
}

// Named for the service and the operation, which is how a backend presents a call to a
// remote service rather than as a span the engine named.
func (s *AWSSDKSuite) Test_records_a_span_for_a_call_the_sdk_made() {
	_, err := s.listBuckets(s.respondWith(http.StatusOK, listBucketsResponse))
	s.Require().NoError(err)

	span, found := s.recorder.Span("S3.ListBuckets")
	s.Require().True(found, "recorded %v", s.recorder.Names())
	s.Assert().True(span.Ended)

	s.assertAttr(span, "rpc.system", "aws-api")
	s.assertAttr(span, "rpc.service", "S3")
	s.assertAttr(span, "rpc.method", "ListBuckets")
}

// The identifier AWS support asks for when a call has to be accounted for, which is only
// known once the call has been made.
func (s *AWSSDKSuite) Test_records_the_request_id_aws_replied_with() {
	_, err := s.listBuckets(s.respondWith(http.StatusOK, listBucketsResponse))
	s.Require().NoError(err)

	span, found := s.recorder.Span("S3.ListBuckets")
	s.Require().True(found)
	s.assertAttr(span, "aws.requestId", "a-request-id-from-aws")
}

func (s *AWSSDKSuite) Test_marks_a_call_that_failed_as_failed() {
	_, err := s.listBuckets(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("the collector of a broken network")
	})
	s.Require().Error(err)

	span, found := s.recorder.Span("S3.ListBuckets")
	s.Require().True(found)
	s.Assert().Error(span.Err)
	s.Assert().True(span.Ended)
}

// One span per call rather than one per attempt, so that a call the SDK retried reads as
// the single call the engine made.
func (s *AWSSDKSuite) Test_records_one_span_for_a_call_the_sdk_retried() {
	attempts := 0
	_, err := s.listBuckets(func(request *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 2 {
			return s.respondWith(http.StatusInternalServerError, "")(request)
		}
		return s.respondWith(http.StatusOK, listBucketsResponse)(request)
	})
	s.Require().NoError(err)
	s.Require().Equal(2, attempts, "the call should have been retried")

	_, found := s.recorder.Span("S3.ListBuckets")
	s.Assert().True(found, "recorded %v", s.recorder.Names())
}

func (s *AWSSDKSuite) listBuckets(
	respond func(*http.Request) (*http.Response, error),
) (*s3.ListBucketsOutput, error) {
	conf, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("eu-west-2"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("key-id", "secret", ""),
		),
		awsconfig.WithHTTPClient(stubHTTPClient{respond: respond}),
		awsconfig.WithAPIOptions([]func(*middleware.Stack) error{
			tracing.InstrumentAWSSDK,
		}),
	)
	s.Require().NoError(err)

	return s3.NewFromConfig(conf).ListBuckets(
		context.Background(),
		&s3.ListBucketsInput{},
	)
}

func (s *AWSSDKSuite) respondWith(
	status int,
	body string,
) func(*http.Request) (*http.Response, error) {
	return func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"X-Amz-Request-Id": []string{"a-request-id-from-aws"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	}
}

func (s *AWSSDKSuite) assertAttr(span *telemetry.RecordedSpan, key string, expected any) {
	value, found := span.Attr(key)
	s.Require().True(found, "%q should have been recorded", key)
	s.Assert().Equal(expected, value)
}

type stubHTTPClient struct {
	respond func(*http.Request) (*http.Response, error)
}

func (c stubHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return c.respond(request)
}

const listBucketsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult><Buckets></Buckets><Owner></Owner></ListAllMyBucketsResult>`

func TestAWSSDKSuite(t *testing.T) {
	suite.Run(t, new(AWSSDKSuite))
}
