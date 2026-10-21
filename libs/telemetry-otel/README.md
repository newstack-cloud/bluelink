# Bluelink OpenTelemetry Library

The OpenTelemetry implementation of the tracing seam in
[`libs/common/telemetry`](../common/telemetry).

None of the libraries in Bluelink starts a span through this module. Call sites depend on the seam, and
a process that wants its traces exported installs this implementation behind the seam at
startup. The Deploy Engine is an example of an application that installs this implementation behind the seam.

## Packages

| Package | What it is for |
| --- | --- |
| `tracing` | Builds a tracer provider exporting over OTLP, installs it behind the seam, and returns the shutdown handle that flushes it. This also holds the X-Ray identifier generator and propagator, for traces that reach AWS X-Ray. |
| `awstrace` | Instruments the AWS SDK, so that calls a process makes through it become spans in whatever trace they were made under. |

The cloud SDK helpers are separate packages so that importing the provider setup does not
compile an SDK the caller never calls.

## Using it

```go
shutdown, err := tracing.Setup(ctx, tracing.Settings{
    OTLPEndpoint: "http://localhost:4317",
    ServiceName:  "my-plugin",
    SamplingRate: 0.1,
}, clock)
```

`Settings.Validate` reports what cannot be acted on, and is worth calling when
configuration is read rather than when tracing is first switched on. An endpoint the
exporter cannot parse is the case that matters as the exporter keeps its own default rather
than failing, so an engine given a typo would report success and export nothing.
