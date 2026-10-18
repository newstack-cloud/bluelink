package core

import (
	"bytes"
	"encoding/json"
	"testing"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// A plugin config value can be logged from anywhere, including code with no idea the value
// came from a plugin's configuration, so the guarantee is made by the core rather than by
// each call site.
type LoggingRedactionTestSuite struct {
	suite.Suite
}

func (s *LoggingRedactionTestSuite) Test_masks_a_field_whose_key_is_a_secret() {
	entry := s.logged(func(logger *zap.Logger) {
		logger.Info(
			"prepared plugin config",
			zap.String("secretField", "super-secret-password"),
			zap.String("stringField", "a value"),
		)
	})

	s.Assert().Equal(bpcore.RedactedPlaceholder, entry["secretField"])
	s.Assert().Equal("a value", entry["stringField"])
}

// Fields attached to a derived logger are written for every entry it produces, so masking
// only what is passed to the call would leave them in the clear.
func (s *LoggingRedactionTestSuite) Test_masks_a_field_attached_to_a_derived_logger() {
	entry := s.logged(func(logger *zap.Logger) {
		logger.With(zap.String("secretField", "super-secret-password")).
			Info("prepared plugin config")
	})

	s.Assert().Equal(bpcore.RedactedPlaceholder, entry["secretField"])
}

func (s *LoggingRedactionTestSuite) Test_leaves_every_field_alone_without_a_matcher() {
	var buffer bytes.Buffer
	core := &redactingCore{
		Core: zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buffer),
			zapcore.DebugLevel,
		),
	}
	zap.New(core).Info("no matcher", zap.String("secretField", "super-secret-password"))

	s.Assert().Contains(buffer.String(), "super-secret-password")
}

func (s *LoggingRedactionTestSuite) logged(write func(logger *zap.Logger)) map[string]any {
	var buffer bytes.Buffer
	core := &redactingCore{
		Core: zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buffer),
			zapcore.DebugLevel,
		),
		matcher: &stubSecretFields{secret: map[string]bool{"secretField": true}},
	}

	write(zap.New(core))

	entry := map[string]any{}
	s.Require().NoError(json.Unmarshal(buffer.Bytes(), &entry))

	return entry
}

type stubSecretFields struct {
	secret map[string]bool
}

func (s *stubSecretFields) IsSecret(key string) bool {
	return s.secret[key]
}

func TestLoggingRedactionTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingRedactionTestSuite))
}
