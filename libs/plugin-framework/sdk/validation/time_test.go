package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type TimeValidationSuite struct {
	suite.Suite
}

func (s *TimeValidationSuite) Test_days_of_week() {
	validDays := []string{
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"FRIDAY",
		"Saturday",
		"SUNdAy",
	}

	for _, day := range validDays {
		diagnostics := IsDayOfTheWeek(true)("exampleField", core.ScalarFromString(day))
		s.Assert().Empty(diagnostics)
	}
}

func (s *TimeValidationSuite) Test_invalid_days_of_week() {
	invalidDays := []string{
		"Funday",  // Not a valid day
		"Mondayy", // Misspelled
		"Mon",     // Too short
		"12345",   // Numeric string
	}

	for _, day := range invalidDays {
		diagnostics := IsDayOfTheWeek( /* ignoreCase */ true)(
			"exampleField",
			core.ScalarFromString(day),
		)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *TimeValidationSuite) Test_months_of_year() {
	validMonths := []string{
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"AUGusT",
		"September",
		"October",
		"November",
		"December",
	}

	for _, month := range validMonths {
		diagnostics := IsMonth( /* ignoreCase */ true)(
			"exampleField",
			core.ScalarFromString(month),
		)
		s.Assert().Empty(diagnostics)
	}
}

func (s *TimeValidationSuite) Test_invalid_months_of_year() {
	invalidMonths := []string{
		"Januar", // Misspelled
		"Feb",    // Too short
		"12345",  // Numeric string
		"MonthX", // Not a valid month
	}

	for _, month := range invalidMonths {
		diagnostics := IsMonth( /* ignoreCase */ true)(
			"exampleField",
			core.ScalarFromString(month),
		)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *TimeValidationSuite) Test_short_months_of_year() {
	validShortMonths := []string{
		"Jan",
		"Feb",
		"Mar",
		"Apr",
		"May",
		"Jun",
		"Jul",
		"Aug",
		"Sep",
		"Oct",
		"Nov",
		"Dec",
	}

	for _, month := range validShortMonths {
		diagnostics := IsShortMonth( /* ignoreCase */ true)(
			"exampleField",
			core.ScalarFromString(month),
		)
		s.Assert().Empty(diagnostics)
	}
}

func (s *TimeValidationSuite) Test_invalid_short_months_of_year() {
	invalidShortMonths := []string{
		"Janu",   // Misspelled
		"Febr",   // Too long
		"123",    // Numeric string
		"MonthX", // Not a valid short month
	}

	for _, month := range invalidShortMonths {
		diagnostics := IsShortMonth( /* ignoreCase */ true)(
			"exampleField",
			core.ScalarFromString(month),
		)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *TimeValidationSuite) Test_rfc3339_date_time() {
	validDates := []string{
		"2023-10-01T12:00:00Z",
		"2023-10-01T12:00:00+02:00",
		"2023-10-01T12:00:00.123456789Z",
	}

	for _, date := range validDates {
		diagnostics := IsRFC3339()("exampleField", core.ScalarFromString(date))
		s.Assert().Empty(diagnostics)
	}
}

func (s *TimeValidationSuite) Test_invalid_rfc3339_date_time() {
	invalidDates := []string{
		"2023-10-01T12:00:00",            // Missing timezone
		"2023-10-01T12:00:00+25:00",      // Invalid timezone offset
		"2023-10-01T12:00:00.123Z+02:00", // Multiple timezones
		"not-a-date-time",                // Completely invalid format
	}

	for _, date := range invalidDates {
		diagnostics := IsRFC3339()("exampleField", core.ScalarFromString(date))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *TimeValidationSuite) Test_invalid_type_for_time() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsRFC3339()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func TestTimeValidationSuite(t *testing.T) {
	suite.Run(t, new(TimeValidationSuite))
}
