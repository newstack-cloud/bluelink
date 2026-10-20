package tracing

import (
	"context"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
)

// InstrumentAWSSDK traces every call an AWS SDK client makes.
//
// Pass it to awsconfig.WithAPIOptions when loading a configuration, and every client
// built from that configuration is traced. The engine reaches S3 and KMS through the
// SDK, and a deployment waiting on one of them is indistinguishable from one waiting on
// the engine itself without this.
//
// The tracer is resolved per call, so a client built before a provider is installed is
// still traced once one is, and a client built where tracing is off costs a call that
// returns immediately.
func InstrumentAWSSDK(stack *middleware.Stack) error {
	// The initialize step, so the span covers the retries of a call rather than
	// producing one span per attempt, and added last within it so that the step which
	// records which service and operation this is has already run.
	return stack.Initialize.Add(awsSDKTracing{}, middleware.After)
}

type awsSDKTracing struct{}

func (awsSDKTracing) ID() string {
	return "BluelinkTracing"
}

func (m awsSDKTracing) HandleInitialize(
	ctx context.Context,
	in middleware.InitializeInput,
	next middleware.InitializeHandler,
) (middleware.InitializeOutput, middleware.Metadata, error) {
	service := awsmiddleware.GetServiceID(ctx)
	operation := awsmiddleware.GetOperationName(ctx)

	ctx, span := telemetry.CurrentTracer().Start(
		ctx,
		service+"."+operation,
		// The names a backend expects of a span describing a call to a remote service,
		// so that it is presented as one rather than as a span the engine named.
		telemetry.StringAttr("rpc.system", "aws-api"),
		telemetry.StringAttr("rpc.service", service),
		telemetry.StringAttr("rpc.method", operation),
	)
	defer span.End()

	out, metadata, err := next.HandleInitialize(ctx, in)
	if err != nil {
		span.RecordError(err)
	}

	// The identifier AWS support asks for when a call has to be accounted for, which is
	// only known once the call has been made.
	if requestID, found := awsmiddleware.GetRequestIDMetadata(metadata); found {
		span.SetAttributes(telemetry.StringAttr("aws.requestId", requestID))
	}

	return out, metadata, err
}
