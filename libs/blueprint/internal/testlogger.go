package internal

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestLogger is an implementation of the core.Logger interface
// that is used for automated testing.
type TestLogger struct {
	zapLogger *zap.Logger
}

// NewTestLogger creates a new instance of a logger to be used for testing.
func NewTestLogger() (core.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &TestLogger{
		zapLogger: logger,
	}, nil
}

func (l *TestLogger) Debug(message string, fields ...core.LogField) {
	l.zapLogger.Debug(message, convertLogFieldsToZap(fields)...)
}

func (l *TestLogger) Info(message string, fields ...core.LogField) {
	l.zapLogger.Info(message, convertLogFieldsToZap(fields)...)
}

func (l *TestLogger) Warn(message string, fields ...core.LogField) {
	l.zapLogger.Warn(message, convertLogFieldsToZap(fields)...)
}

func (l *TestLogger) Error(message string, fields ...core.LogField) {
	l.zapLogger.Error(message, convertLogFieldsToZap(fields)...)
}

func (l *TestLogger) Fatal(message string, fields ...core.LogField) {
	l.zapLogger.Fatal(message, convertLogFieldsToZap(fields)...)
}

func (l *TestLogger) WithFields(fields ...core.LogField) core.Logger {
	return &TestLogger{
		zapLogger: l.zapLogger.With(convertLogFieldsToZap(fields)...),
	}
}

func (l *TestLogger) Named(name string) core.Logger {
	return &TestLogger{
		zapLogger: l.zapLogger.Named(name),
	}
}

func convertLogFieldsToZap(fields []core.LogField) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		zapFields = append(zapFields, convertLogFieldToZap(field))
	}
	return zapFields
}

func convertLogFieldToZap(field core.LogField) zap.Field {
	switch field.Type {
	case core.StringLogFieldType:
		return zap.String(field.Key, field.String)
	case core.IntegerLogFieldType:
		return zap.Int64(field.Key, field.Integer)
	case core.FloatLogFieldType:
		return zap.Float64(field.Key, field.Float)
	case core.BoolLogFieldType:
		return zap.Bool(field.Key, field.Bool)
	case core.ErrorLogFieldType:
		return zap.Error(field.Err)
	case core.StringsLogFieldType:
		strings, ok := field.Interface.([]string)
		if !ok {
			strings = []string{}
		}
		return zap.Strings(field.Key, strings)
	case core.IntegersLogFieldType:
		integers, ok := field.Interface.([]int64)
		if !ok {
			integers = []int64{}
		}
		return zap.Int64s(field.Key, integers)
	case core.FloatsLogFieldType:
		floats, ok := field.Interface.([]float64)
		if !ok {
			floats = []float64{}
		}
		return zap.Float64s(field.Key, floats)
	case core.BoolsLogFieldType:
		bools, ok := field.Interface.([]bool)
		if !ok {
			bools = []bool{}
		}
		return zap.Bools(field.Key, bools)
	case core.ErrorsLogFieldType:
		errors, ok := field.Interface.([]error)
		if !ok {
			errors = []error{}
		}
		return zap.Errors(field.Key, errors)
	default:
		return zap.Any(field.Key, field.Interface)
	}
}
