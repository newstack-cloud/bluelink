package audit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/newstack-cloud/bluelink/libs/telemetry-otel/tracing"
)

// The action each mutating route is recorded as.
//
// Keyed by method and the route's path template rather than by the request path, so that an
// action does not change name when an identifier changes.
var actionsByRoute = map[string]string{
	"POST /validations":                                     "validation.create",
	"POST /validations/cleanup":                             "validation.cleanup",
	"POST /deployments/changes":                             "changeset.create",
	"POST /deployments/changes/cleanup":                     "changeset.cleanup",
	"POST /deployments/reconciliation-results/cleanup":      "reconciliation.cleanup",
	"POST /deployments/instances":                           "instance.create",
	"PATCH /deployments/instances/{id}":                     "instance.update",
	"POST /deployments/instances/{id}/destroy":              "instance.destroy",
	"POST /deployments/instances/{id}/reconciliation/check": "instance.drift.check",
	"POST /deployments/instances/{id}/reconciliation/apply": "instance.drift.apply",
	"POST /events/cleanup":                                  "event.cleanup",
	"POST /state/housekeeping":                              "state.housekeeping",
	// Deleting records from the trail is itself an action taken through the engine, so it
	// is recorded in the trail like any other.
	"POST /audit/records/cleanup": "audit.cleanup",
}

// Middleware records every request that can change something.
//
// A middleware rather than a call in each handler, because the requirement is that every
// mutating action is recorded and a per-handler call would be more difficult to maintain.
type Middleware struct {
	sink   Sink
	clock  commoncore.Clock
	logger bpcore.Logger
}

// NewMiddleware creates the middleware that records mutating requests.
func NewMiddleware(sink Sink, clock commoncore.Clock, logger bpcore.Logger) *Middleware {
	return &Middleware{
		sink:   sink,
		clock:  clock,
		logger: logger,
	}
}

func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !changesSomething(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		requestID := requestIDFor(r)
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		// Deferred rather than called after, so that a handler panicking part-way
		// through a mutation is still recorded instead of unwinding past this
		// middleware. Nothing recovers the panic here; the record is made and the
		// panic continues to net/http, which closes the connection.
		defer func() {
			if panicValue := recover(); panicValue != nil {
				// The handler may not have written a status, and a mutation that
				// panicked part-way through did not succeed either way.
				m.record(r, http.StatusInternalServerError, requestID)
				panic(panicValue)
			}

			m.record(r, recorder.status, requestID)
		}()

		next.ServeHTTP(recorder, r)
	})
}

// How long a sink is given to accept a record once the request has been answered. A sink
// that has not taken it by then has the record logged against it instead, which is a
// bounded delay to a response that has already been sent rather than an unbounded one.
const sinkWriteTimeout = 10 * time.Second

// A request that cannot change anything is not recorded. The trail is of actions taken, and
// a record of every read would bury them.
func changesSomething(method string) bool {
	return method != http.MethodGet && method != http.MethodHead &&
		method != http.MethodOptions
}

func (m *Middleware) record(r *http.Request, status int, requestID string) {
	record := &Record{
		ID:         uuid.New().String(),
		Timestamp:  m.clock.Now().Unix(),
		Action:     actionFor(r),
		Target:     mux.Vars(r)["id"],
		Outcome:    outcomeFor(status),
		StatusCode: status,
		Method:     r.Method,
		Path:       r.URL.Path,
		SourceIP:   sourceIPFor(r),
		UserAgent:  r.UserAgent(),
		RequestID:  requestID,
	}

	// The trace the action was carried out under, so that an audited action leads to the
	// trace of the work it did. Absent where tracing is off, or where the trace is one no
	// backend recorded.
	if spanCtx := tracing.SpanContextFromContext(r.Context()); spanCtx.Recorded() {
		record.TraceID = spanCtx.TraceID
		record.SpanID = spanCtx.SpanID
	}

	if principal, found := auth.PrincipalFromContext(r.Context()); found {
		record.PrincipalMethod = principal.Method
		record.PrincipalSubject = principal.Subject
		record.PrincipalName = principal.Name
		record.OnBehalfOf = principal.OnBehalfOf
	}

	// The request's own context is cancelled once the handler returns, and the record is
	// written after it. Stripping the cancellation keeps the values a sink may read from
	// the context while letting the write outlive the request it describes.
	ctx, cancel := context.WithTimeout(
		context.WithoutCancel(r.Context()),
		sinkWriteTimeout,
	)
	defer cancel()

	if err := m.sink.Write(ctx, record); err != nil {
		// Logged rather than returned as the action has already happened, so failing the
		// request now would report it as not having been taken.
		httpmw.LoggerFromContext(r.Context(), m.logger).Error(
			"failed to write an audit record for a request that changed something",
			bpcore.StringLogField("action", record.Action),
			bpcore.ErrorLogField("error", err),
		)
	}
}

// The name for the route the request matched, falling back to the method and path.
//
// A fallback rather than no record, so adding a mutating route without naming it here
// produces an unnamed record instead of silence.
func actionFor(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	}

	template, err := route.GetPathTemplate()
	if err != nil {
		return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	}

	if action, named := actionsByRoute[fmt.Sprintf("%s %s", r.Method, template)]; named {
		return action
	}

	return fmt.Sprintf("%s %s", r.Method, template)
}

func outcomeFor(status int) string {
	if status >= 200 && status < 300 {
		return OutcomeSuccess
	}

	return OutcomeFailure
}

// The address the extract-IP middleware resolved for the request.
//
// Read rather than derived here, so that a record and anything else attributing the
// request agree on one address, and so that which header to trust stays a deployment
// setting rather than a decision taken twice.
//
// Empty when that middleware has not run, which is a record with no source IP rather
// than a guess at one.
func sourceIPFor(r *http.Request) string {
	sourceIP, _ := httpmw.SourceIPFromContext(r.Context())

	return sourceIP
}

// The correlation ID the request-context middleware resolved.
//
// Read rather than generated here as a record and the log lines for the same request have
// to carry the same ID, and two consumers each minting one will make it hard to correlate
// output logs and audit records.
func requestIDFor(r *http.Request) string {
	requestID, _ := httpmw.RequestIDFromContext(r.Context())

	return requestID
}

// Captures the status the handler wrote, which is the only thing that says whether the
// action the record names actually took effect.
//
// Does not forward Flush or Hijack, which is safe only because every streaming route is a
// GET and so passes the original ResponseWriter through untouched. A streaming route that
// changed something would need those forwarded.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
