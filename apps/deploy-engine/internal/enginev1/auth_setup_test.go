package enginev1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// An audit record identifies the API key that acted by an unsalted digest of it, which can
// be guessed against offline. The engine cannot measure entropy, so it checks length and
// says so, rather than leaving the assumption that keys are strong unstated.
type AuthSetupSuite struct {
	suite.Suite
}

func (s *AuthSetupSuite) Test_warns_about_api_keys_short_enough_to_guess() {
	logger := &recordingLogger{}

	warnAboutWeakAPIKeys(
		&core.AuthConfig{APIKeys: []string{"devkey", "ci-token-2024"}},
		logger,
	)

	s.Require().Len(logger.warnings, 1)
	s.Assert().Contains(logger.warnings[0], "shorter than the recommended length")
}

// The warning goes to the same logs the fingerprints do, so it reports how many keys are
// weak rather than which.
func (s *AuthSetupSuite) Test_does_not_name_the_weak_key_in_the_warning() {
	logger := &recordingLogger{}

	warnAboutWeakAPIKeys(
		&core.AuthConfig{APIKeys: []string{"devkey"}},
		logger,
	)

	s.Require().Len(logger.warnings, 1)
	s.Assert().NotContains(logger.warnings[0], "devkey")
	for _, field := range logger.fields[0] {
		s.Assert().NotEqual("devkey", field.String)
	}
}

func (s *AuthSetupSuite) Test_stays_quiet_for_keys_of_the_recommended_length() {
	logger := &recordingLogger{}

	warnAboutWeakAPIKeys(
		&core.AuthConfig{
			APIKeys: []string{
				"0123456789abcdef0123456789abcdef",
				"fedcba9876543210fedcba9876543210ff",
			},
		},
		logger,
	)

	s.Assert().Empty(logger.warnings)
}

type recordingLogger struct {
	bpcore.Logger
	warnings []string
	fields   [][]bpcore.LogField
}

func (l *recordingLogger) Warn(message string, fields ...bpcore.LogField) {
	l.warnings = append(l.warnings, message)
	l.fields = append(l.fields, fields)
}

func TestAuthSetupSuite(t *testing.T) {
	suite.Run(t, new(AuthSetupSuite))
}
