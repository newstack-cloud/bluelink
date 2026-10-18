package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log fields are the one way a value reaches a log, so what the field types admit bounds
// what can leak. The constructors take scalars and scalar slices only, which is why a
// resource spec or a link's data cannot be logged without being deliberately stringified
// first. The guards here are what keep that true as the type grows.
type LoggingZapTestSuite struct {
	suite.Suite
}

// A field type added without a case in the serialiser used to fall through to zap.Any,
// which writes whatever the field holds. This fails if a type is added without being
// listed as handled, which is the point at which somebody has to decide how to serialise
// it rather than leaving it to the fallback.
func (s *LoggingZapTestSuite) Test_every_declared_log_field_type_is_handled() {
	s.Require().Len(
		LogFieldTypes(),
		int(logFieldTypeCount),
		"a log field type was added without being listed in LogFieldTypes",
	)

	for _, fieldType := range LogFieldTypes() {
		entry := s.logged(LogField{Type: fieldType, Key: "field", String: "a value"})
		s.Assert().NotEqual(
			RedactedPlaceholder,
			entry["field"],
			"a listed field type was withheld, so the serialiser has no case for it",
		)
	}
}

// Whatever reaches the fallback is withheld rather than serialised, the serialiser does not
// know the shape of a type it has no case for, and the value could be a resource spec or a
// link's data.
func (s *LoggingZapTestSuite) Test_withholds_the_value_of_an_unhandled_field_type() {
	entry := s.logged(LogField{
		Type:      logFieldTypeCount,
		Key:       "spec",
		Interface: map[string]string{"dbPassword": "super-secret-password"},
	})

	s.Assert().Equal(RedactedPlaceholder, entry["spec"])
}

func (s *LoggingZapTestSuite) Test_a_scalar_field_carries_its_value() {
	entry := s.logged(
		StringLogField("instanceId", "instance-1"),
		IntegerLogField("resourceCount", 3),
		ErrorLogField("error", errors.New("failed")),
	)

	s.Assert().Equal("instance-1", entry["instanceId"])
	s.Assert().Equal(float64(3), entry["resourceCount"])
	s.Assert().Equal("failed", entry["error"])
}

func (s *LoggingZapTestSuite) logged(fields ...LogField) map[string]any {
	var buffer bytes.Buffer
	zapLogger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buffer),
		zapcore.DebugLevel,
	))

	NewLoggerFromZap(zapLogger).Info("logged", fields...)

	entry := map[string]any{}
	s.Require().NoError(json.Unmarshal(buffer.Bytes(), &entry))

	return entry
}

func TestLoggingZapTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingZapTestSuite))
}
