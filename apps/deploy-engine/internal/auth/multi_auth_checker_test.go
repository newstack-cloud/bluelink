package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MultiAuthCheckerSuite struct {
	suite.Suite
}

func (s *MultiAuthCheckerSuite) Test_check_succeeds_on_first_check() {
	checker := NewMultiAuthChecker(
		testChecker1,
		testChecker2,
		testChecker3,
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "checker-1-key")
	_, err := checker.Check(context.Background(), headers)
	s.Assert().NoError(err)
}

func (s *MultiAuthCheckerSuite) Test_check_succeeds_on_last_check() {
	checker := NewMultiAuthChecker(
		testChecker1,
		testChecker2,
		testChecker3,
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "checker-3-key")
	_, err := checker.Check(context.Background(), headers)
	s.Assert().NoError(err)
}

func (s *MultiAuthCheckerSuite) Test_check_fails_all_checks_returning_error_for_last_check() {
	checker := NewMultiAuthChecker(
		testChecker1,
		testChecker2,
		testChecker3,
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "invalid-key")
	_, err := checker.Check(context.Background(), headers)
	s.Assert().Error(err)
	authErr, ok := err.(*Error)
	s.Assert().True(ok)
	s.Assert().Equal("checker-3 auth error", authErr.ChildErr.Error())
}

type checkerFunc func(ctx context.Context, headers http.Header) (*Principal, error)

func (f checkerFunc) Check(
	ctx context.Context,
	headers http.Header,
) (*Principal, error) {
	return f(ctx, headers)
}

// Each stub names itself in the principal it returns, so a test can tell which checker
// established the principal rather than only that one was.
func testCheckerFor(name string, key string) checkerFunc {
	return func(ctx context.Context, headers http.Header) (*Principal, error) {
		if headers.Get(BluelinkAPIKeyHeaderName) == key {
			return &Principal{Method: MethodAPIKey, Subject: name}, nil
		}

		return nil, &Error{
			ChildErr: errors.New(name + " auth error"),
		}
	}
}

var (
	testChecker1 = testCheckerFor("checker-1", "checker-1-key")
	testChecker2 = testCheckerFor("checker-2", "checker-2-key")
	testChecker3 = testCheckerFor("checker-3", "checker-3-key")
)

func TestMultiAuthCheckerSuite(t *testing.T) {
	suite.Run(t, new(MultiAuthCheckerSuite))
}
