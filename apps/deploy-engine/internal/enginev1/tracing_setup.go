package enginev1

import (
	"context"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/middleware"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// How long a shutdown waits for the spans still held to reach the collector. Short,
// because this runs while the process is stopping and an unreachable collector must not
// hold it open.
const traceShutdownTimeout = 5 * time.Second

// The AWS configuration options that trace every call a client makes, so that time the
// engine spends waiting on AWS is accounted for as such rather than as time spent in
// the engine.
//
// Shared by every site that builds an AWS client, so that a new one is traced by
// construction.
func tracedAWSOptions() []func(*awsconfig.LoadOptions) error {
	return []func(*awsconfig.LoadOptions) error{
		awsconfig.WithAPIOptions([]func(*middleware.Stack) error{
			tracing.InstrumentAWSSDK,
		}),
	}
}

// Starts exporting traces where a deployment asked for it, and returns the callback
// that stops it.
//
// Configuration that cannot be acted on is refused here rather than at the first span,
// whether or not tracing is on, so that an operator is told at startup.
func setupTracing(
	ctx context.Context,
	config *core.Config,
	clock commoncore.Clock,
	logger bpcore.Logger,
) (func(), error) {
	settings := traceSettings(&config.Tracing, config.Version)
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	if !config.Tracing.Enabled {
		logger.Debug(
			"not exporting traces, set tracing.enabled to true to export them to a " +
				"collector",
		)

		return func() {}, nil
	}

	shutdown, err := tracing.Setup(ctx, settings, clock)
	if err != nil {
		return nil, err
	}

	logger.Info(
		"exporting traces to a collector",
		bpcore.StringLogField("otlpEndpoint", settings.OTLPEndpoint),
		bpcore.StringLogField("serviceName", settings.ServiceName),
		bpcore.StringLogField("samplingRate", fmt.Sprintf("%g", settings.SamplingRate)),
		bpcore.StringLogField(
			"operationSamplingRate",
			fmt.Sprintf("%g", settings.OperationSamplingRate),
		),
	)

	return traceShutdownFunc(shutdown, logger), nil
}

func traceSettings(config *core.TracingConfig, version string) tracing.Settings {
	return tracing.Settings{
		OTLPEndpoint:          config.OTLPEndpoint,
		ServiceName:           config.ServiceName,
		ServiceVersion:        version,
		SamplingRate:          config.SamplingRate,
		OperationSamplingRate: config.OperationSamplingRate,
		Platform:              config.Platform,
	}
}

// A context of its own rather than one from the shutdown calling this, since the spans
// being flushed describe work that has already finished and the context it ran under is
// cancelled by this point.
func traceShutdownFunc(
	shutdown func(context.Context) error,
	logger bpcore.Logger,
) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), traceShutdownTimeout)
		defer cancel()

		if err := shutdown(ctx); err != nil {
			logger.Error(
				"failed to flush the traces still held, so the last of them will be "+
					"missing from the collector",
				bpcore.ErrorLogField("error", err),
			)
		}
	}
}
