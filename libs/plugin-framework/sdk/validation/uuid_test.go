package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type UUIDValidationSuite struct {
	suite.Suite
}

func (s *UUIDValidationSuite) Test_valid_uuids() {
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"123e4567-e89b-12d3-a456-426614174000",
		"550e8400-e29b-41d4-a716-446655440001",
	}

	for _, uuidStr := range validUUIDs {
		diagnostics := IsUUID()("exampleField", core.ScalarFromString(uuidStr))
		s.Assert().Empty(diagnostics)
	}
}

func (s *UUIDValidationSuite) Test_fails_for_invalid_uuids() {
	invalidUUIDs := []string{
		"550e8400-e29b-41d4-a716-44665544000",   // Too short
		"550e8400-e29b-41d4-a716-4466554400000", // Too long
		"not-a-uuid",
		"123e4567-e89b-12d3-a456-42661417400z", // Invalid character
	}

	for _, uuidStr := range invalidUUIDs {
		diagnostics := IsUUID()("exampleField", core.ScalarFromString(uuidStr))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid UUID")
	}
}

func (s *UUIDValidationSuite) Test_invalid_type_for_uuid() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsUUID()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func TestUUIDValidationSuite(t *testing.T) {
	suite.Run(t, new(UUIDValidationSuite))
}
