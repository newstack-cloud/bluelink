package plugintestutils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MockCallsSuite struct {
	suite.Suite
}

type serviceMock struct {
	MockCalls
}

func (m *serviceMock) SaveResource(arg1 string, arg2 int) {
	m.RegisterCall(arg1, arg2)
}

func (s *MockCallsSuite) Test_assertions_for_derived_method_name_call() {
	service := &serviceMock{}
	service.SaveResource("testArg1", 242)

	service.AssertCalled(&s.Suite, "SaveResource")
	service.AssertCalledWith(
		&s.Suite,
		"SaveResource",
		/* callIndex */ 0,
		"testArg1",
		242,
	)

	service.AssertNotCalled(&s.Suite, "DeleteResource")
}

func (s *MockCallsSuite) Test_assertions_for_named_call() {
	mockCalls := &MockCalls{}
	mockCalls.RegisterNamedCall("UpdateResource", "testArg1", 504)

	mockCalls.AssertCalled(&s.Suite, "UpdateResource")
	mockCalls.AssertCalledWith(
		&s.Suite,
		"UpdateResource",
		/* callIndex */ 0,
		"testArg1",
		504,
	)
	mockCalls.AssertNotCalled(&s.Suite, "CreateResource")
}

func (s *MockCallsSuite) Test_assertions_with_bool_matcher() {
	mockCalls := &MockCalls{}
	mockCalls.RegisterNamedCall("UpdateResource", "testArg1", 504)

	mockCalls.AssertCalledWith(
		&s.Suite,
		"UpdateResource",
		/* callIndex */ 0,
		func(arg any) bool {
			return arg == "testArg1"
		},
		func(arg any) bool {
			return arg == 504
		},
	)
}

func (s *MockCallsSuite) Test_assertions_with_equality_matcher() {
	mockCalls := &MockCalls{}
	mockCalls.RegisterNamedCall("UpdateResource", "{\"key\":\"value\"}", 504)

	mockCalls.AssertCalledWith(
		&s.Suite,
		"UpdateResource",
		/* callIndex */ 0,
		func(arg any) (EqualityCheckValues, error) {
			jsonStr, ok := arg.(string)
			if !ok {
				return EqualityCheckValues{}, fmt.Errorf("expected string, got %T", arg)
			}

			actualMap := map[string]string{}
			err := json.Unmarshal([]byte(jsonStr), &actualMap)
			if err != nil {
				return EqualityCheckValues{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
			}

			return EqualityCheckValues{
				Expected: map[string]string{"key": "value"},
				Actual:   actualMap,
			}, nil
		},
		504,
	)
}

func TestMockCallsSuite(t *testing.T) {
	suite.Run(t, new(MockCallsSuite))
}
