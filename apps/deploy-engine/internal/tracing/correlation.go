package tracing

import (
	"net/http"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// LogCorrelationMiddleware puts the identity of a request's span onto everything
// logged for that request.
type LogCorrelationMiddleware struct {
	// Whether the X-Ray form of the trace ID is added alongside the OpenTelemetry one,
	// which is the only form the X-Ray console will find a trace by.
	xray bool
}

// NewLogCorrelationMiddleware creates the middleware, including the X-Ray form of the
// trace ID where the engine's traces reach X-Ray.
func NewLogCorrelationMiddleware(xray bool) *LogCorrelationMiddleware {
	return &LogCorrelationMiddleware{xray: xray}
}

func (m *LogCorrelationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields := m.correlationFields(r)
		if len(fields) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(
			httpmw.WithLoggerFields(r.Context(), fields...),
		))
	})
}

// Nothing is added for a request no backend will have a trace for. See
// telemetry.SpanContext.Recorded.
func (m *LogCorrelationMiddleware) correlationFields(r *http.Request) []bpcore.LogField {
	spanCtx := SpanContextFromContext(r.Context())
	if !spanCtx.Recorded() {
		return nil
	}

	fields := []bpcore.LogField{
		bpcore.StringLogField(telemetry.TraceIDField, spanCtx.TraceID),
		bpcore.StringLogField(telemetry.SpanIDField, spanCtx.SpanID),
	}
	if m.xray {
		fields = append(fields, bpcore.StringLogField(
			XRayTraceIDField,
			ToXRayTraceID(spanCtx.TraceID),
		))
	}

	return fields
}
