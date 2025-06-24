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
		checkerFunc(testChecker1),
		checkerFunc(testChecker2),
		checkerFunc(testChecker3),
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "checker-1-key")
	err := checker.Check(context.Background(), headers)
	s.Assert().NoError(err)
}

func (s *MultiAuthCheckerSuite) Test_check_succeeds_on_last_check() {
	checker := NewMultiAuthChecker(
		checkerFunc(testChecker1),
		checkerFunc(testChecker2),
		checkerFunc(testChecker3),
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "checker-3-key")
	err := checker.Check(context.Background(), headers)
	s.Assert().NoError(err)
}

func (s *MultiAuthCheckerSuite) Test_check_fails_all_checks_returning_error_for_last_check() {
	checker := NewMultiAuthChecker(
		checkerFunc(testChecker1),
		checkerFunc(testChecker2),
		checkerFunc(testChecker3),
	)
	headers := make(http.Header)
	headers.Set(BluelinkAPIKeyHeaderName, "invalid-key")
	err := checker.Check(context.Background(), headers)
	s.Assert().Error(err)
	authErr, ok := err.(*Error)
	s.Assert().True(ok)
	s.Assert().Equal("checker-3 auth error", authErr.ChildErr.Error())
}

type checkerFunc func(ctx context.Context, headers http.Header) error

func (f checkerFunc) Check(ctx context.Context, headers http.Header) error {
	return f(ctx, headers)
}

func testChecker1(ctx context.Context, headers http.Header) error {
	if headers.Get(BluelinkAPIKeyHeaderName) == "checker-1-key" {
		return nil
	}

	return &Error{
		ChildErr: errors.New("checker-1 auth error"),
	}
}

func testChecker2(ctx context.Context, headers http.Header) error {
	if headers.Get(BluelinkAPIKeyHeaderName) == "checker-2-key" {
		return nil
	}

	return &Error{
		ChildErr: errors.New("checker-2 auth error"),
	}
}

func testChecker3(ctx context.Context, headers http.Header) error {
	if headers.Get(BluelinkAPIKeyHeaderName) == "checker-3-key" {
		return nil
	}

	return &Error{
		ChildErr: errors.New("checker-3 auth error"),
	}
}

func TestMultiAuthCheckerSuite(t *testing.T) {
	suite.Run(t, new(MultiAuthCheckerSuite))
}
