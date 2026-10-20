package httpmw

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// RequestIDHeader is the header a caller can supply its own correlation ID on.
const RequestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}

type loggerContextKey struct{}

// RequestContextMiddleware resolves a request's correlation ID and builds the
// request-scoped logger that everything handling the request logs through.
//
// Both are resolved once, here, because both are answers that have to be the same
// wherever they are read. A correlation ID generated separately by two consumers
// correlates nothing, and a log line that disagrees with its own audit record about
// where a request came from is worse than one that says nothing.
//
// This does not log requests. It prepares what a log line is enriched with; emitting
// one is the job of whatever is handling the request.
//
// Runs after ExtractIPMiddleware, whose answer it reads. Registered the other way round,
// every line is logged without a source IP, which is the field that matters most on the
// lines nothing else accounts for.
type RequestContextMiddleware struct {
	logger bpcore.Logger
}

// NewRequestContextMiddleware creates the middleware that resolves the request ID and
// the request-scoped logger, deriving the latter from the given base logger.
func NewRequestContextMiddleware(logger bpcore.Logger) *RequestContextMiddleware {
	return &RequestContextMiddleware{logger: logger}
}

func (m *RequestContextMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := resolveRequestID(r)

		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		ctx = context.WithValue(ctx, loggerContextKey{}, m.logger.WithFields(
			requestFields(r, requestID)...,
		))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// The fields every log line for a request carries, whether or not it went on to
// authenticate. Who acted is added later, by whatever establishes it.
func requestFields(r *http.Request, requestID string) []bpcore.LogField {
	sourceIP, _ := SourceIPFromContext(r.Context())

	return []bpcore.LogField{
		bpcore.StringLogField("requestId", requestID),
		bpcore.StringLogField("method", r.Method),
		bpcore.StringLogField("path", r.URL.EscapedPath()),
		// r.Host rather than r.URL.Hostname(): a server-side request carries the host
		// it was addressed to here, and an empty URL host.
		bpcore.StringLogField("host", r.Host),
		bpcore.StringLogField("sourceIp", sourceIP),
		bpcore.StringLogField("userAgent", r.UserAgent()),
	}
}

// The caller's correlation ID where one was supplied, so a request can be followed
// across the components it passed through, and a fresh one otherwise so that every
// request has one.
func resolveRequestID(r *http.Request) string {
	if requestID := r.Header.Get(RequestIDHeader); requestID != "" {
		return requestID
	}

	return uuid.New().String()
}

// RequestIDFromContext returns the correlation ID resolved for the request, and whether
// one was resolved at all.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)

	return requestID, ok && requestID != ""
}

// LoggerFromContext returns the request-scoped logger, falling back to the given logger
// when the middleware has not run.
//
// A fallback rather than a nil logger, so that a caller never has to decide whether it
// is worth logging at all.
func LoggerFromContext(ctx context.Context, fallback bpcore.Logger) bpcore.Logger {
	logger, ok := ctx.Value(loggerContextKey{}).(bpcore.Logger)
	if !ok {
		return fallback
	}

	return logger
}

// WithLoggerFields returns a context whose request-scoped logger also carries the given
// fields.
//
// This is how later middleware adds what it alone knows, without this package having to
// know about it. Returns the context unchanged when the middleware has not run, so that
// a caller does not have to check.
func WithLoggerFields(ctx context.Context, fields ...bpcore.LogField) context.Context {
	logger, ok := ctx.Value(loggerContextKey{}).(bpcore.Logger)
	if !ok {
		return ctx
	}

	return context.WithValue(ctx, loggerContextKey{}, logger.WithFields(fields...))
}
