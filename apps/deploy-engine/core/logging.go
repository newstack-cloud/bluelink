package core

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CreateLogger creates a logger for deploy engine logs
// (debug, info, error etc.) that writes to stdout.
// This will determine the log level and format based on the
// provided configuration.
// A purely JSON format is used in production,
// while a more human-readable format is used in development.
func CreateLogger(config *Config) (core.Logger, error) {
	zapLevel, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, err
	}

	createZapEncoderConfig := zap.NewProductionEncoderConfig
	createZapEncoder := zapcore.NewJSONEncoder
	if config.Environment == "development" {
		createZapEncoderConfig = zap.NewDevelopmentEncoderConfig
		createZapEncoder = zapcore.NewConsoleEncoder
	}
	zapConf := createZapEncoderConfig()
	stdoutSyncer := zapcore.Lock(os.Stdout)
	encoder := createZapEncoder(zapConf)
	zapCore := zapcore.NewCore(encoder, stdoutSyncer, zapLevel)
	zapLogger := zap.New(zapCore)

	return core.NewLoggerFromZap(zapLogger), nil
}
