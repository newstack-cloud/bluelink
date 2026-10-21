package deploymentsv1

import (
	"context"
	"errors"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Span names for the operations this controller carries on with after the request that
// asked for one has been answered.
//
// Named for the work rather than for what prompted it, so a rollback is the deploy or
// the destroy that it carries out. A rollback that recreates what a failed destroy
// removed is a deploy, and one that removes what a failed create made is a destroy, so
// naming them for the failure they answer would put them under the wrong verb. Whether
// an operation is a rollback, and who started it, are attributes: see rollbackAttrs.
const (
	deploySpanName        = "instance.deploy"
	destroySpanName       = "instance.destroy"
	changeStagingSpanName = "changeset.stage"
)

// How an operation came to be started.
const (
	triggerRequest      = "request"
	triggerAutoRollback = "autoRollback"
)

// The attributes marking an operation the engine started itself to undo a failure.
//
// Returned together so that neither can be set without the other: a rollback with no
// trigger reads as one a client asked for, and a trigger with no rollback flag leaves
// "every rollback" unanswerable.
func rollbackAttrs() []telemetry.Attr {
	return []telemetry.Attr{
		telemetry.BoolAttr("rollback", true),
		telemetry.StringAttr("trigger", triggerAutoRollback),
	}
}

// The context a detached operation runs under, bounded by the timeout it is given and
// carrying the root span covering the whole of it.
//
// The span is left in the context rather than handed back, because a deployment's span
// is ended by the goroutine that sees its terminal status rather than by the caller
// here. See endOperationSpan and failOperation.
func operationContext(
	linkedTo telemetry.SpanContext,
	name string,
	timeout time.Duration,
	attrs ...telemetry.Attr,
) (context.Context, context.CancelFunc) {
	ctx, _ := telemetry.CurrentTracer().StartRoot(
		context.Background(),
		name,
		linkedTo,
		attrs...,
	)

	return context.WithTimeout(ctx, timeout)
}

// A failure is recorded as an error as well as an attribute, since a backend filters
// failed spans by their status rather than by reading attributes. An operation that
// produced neither a terminal status nor an error ended without reporting, so the span
// is left without a status rather than reading as a success.
func endOperationSpan(
	ctx context.Context,
	finishMsg *container.DeploymentFinishedMessage,
	err error,
) {
	span := telemetry.CurrentTracer().SpanFromContext(ctx)
	defer span.End()

	if err != nil {
		span.RecordError(err)
		return
	}

	if finishMsg == nil {
		return
	}

	span.SetAttributes(telemetry.StringAttr("instanceStatus", finishMsg.Status.String()))
	if isFailedInstanceStatus(finishMsg.Status) {
		span.RecordError(errors.New("the operation ended as " + finishMsg.Status.String()))
	}
}

// Ensures an error is recorded in a span before cancelling due to a failure
// that either doesn't map to a failed status or where the status has already
// been updated somewhere else.
func failOperation(ctx context.Context, cancel context.CancelFunc, err error) {
	span := telemetry.CurrentTracer().SpanFromContext(ctx)
	span.RecordError(err)
	span.End()
	cancel()
}

func isFailedInstanceStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusDeployFailed,
		core.InstanceStatusDeployRollbackFailed,
		core.InstanceStatusDestroyFailed,
		core.InstanceStatusDestroyRollbackFailed,
		core.InstanceStatusUpdateFailed,
		core.InstanceStatusUpdateRollbackFailed,
		core.InstanceStatusDeployInterrupted,
		core.InstanceStatusUpdateInterrupted,
		core.InstanceStatusDestroyInterrupted:
		return true
	default:
		return false
	}
}
