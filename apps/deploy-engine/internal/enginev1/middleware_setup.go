package enginev1

import (
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/audit"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// What the middleware chain is built from.
type middlewareParams struct {
	router *mux.Router
	config *core.Config
	// Where the audit trail is written, which is also what its endpoints read.
	auditStore manage.Audit
	// The routes served without authenticating.
	unauthenticated []*mux.Route
	clock           commoncore.Clock
	logger          bpcore.Logger
}

// Builds the middleware chain and registers it on the router.
//
// The order each is registered in is the order a request passes through them, and
// several of them depend on that order rather than only on being present. Returns the
// callback that closes the audit sink.
func setupMiddleware(params *middlewareParams) (func(), error) {
	authMiddleware, err := setupAuth(
		&params.config.Auth,
		params.clock,
		params.unauthenticated,
		params.logger,
	)
	if err != nil {
		return nil, err
	}

	auditSink, closeAuditSink, err := setupAuditSink(
		&params.config.Audit,
		params.auditStore,
		params.logger,
	)
	if err != nil {
		return nil, err
	}

	sourceIPMiddleware, err := setupSourceIPMiddleware(params.config, params.logger)
	if err != nil {
		return nil, err
	}

	// Both of the tracing middleware below are registered only where tracing was asked
	// for. Request instrumentation does real work per request whether or not anything is
	// recording: it reads the propagation headers, builds the attributes describing the
	// request, and wraps both the body and the response writer.
	tracingEnabled := params.config.Tracing.Enabled

	// Request instrumentation first, so that the span covers the whole of handling a
	// request, the middleware below included. A request refused by authentication is
	// one whose time is worth accounting for too.
	if tracingEnabled {
		params.router.Use(otelmux.Middleware(params.config.Tracing.ServiceName))
	}

	// Source IP next, so that everything after it attributes a request to the same
	// address rather than each deriving its own answer.
	params.router.Use(sourceIPMiddleware.Middleware)

	// Then the request's correlation ID and logger, before authentication, so that a
	// request refused for failing to authenticate is still logged with where it came
	// from. Such a request changed nothing, so it has no audit record, and this is the
	// only account of it.
	params.router.Use(
		httpmw.NewRequestContextMiddleware(params.logger).Middleware,
	)

	// Then the identity of the request's span onto that logger, now that both the span
	// and the logger exist. Registered the other way round, it adds nothing.
	if tracingEnabled {
		params.router.Use(
			tracing.NewLogCorrelationMiddleware(
				traceSettings(&params.config.Tracing, params.config.Version).IsAWS(),
			).Middleware,
		)
	}

	// Audit after authentication, so that a record names the principal the request
	// authenticated as. A request that failed to authenticate changed nothing and has no
	// principal, so it belongs in the operational logs rather than the trail of actions
	// taken.
	params.router.Use(authMiddleware.Middleware)
	params.router.Use(
		audit.NewMiddleware(auditSink, params.clock, params.logger).Middleware,
	)

	return closeAuditSink, nil
}
