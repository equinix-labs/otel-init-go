# otel-init-go

[ THIS IS A WORK-IN-PROGRESS AND NOT EVEN EXPERIMENTAL YET ]

OpenTelemetry plumbing initializer for Go that only supports OTLP/gRPC
and aims for a small code footprint and get configuration from environment
variables exclusively. The intent is to be able to drop this into existing
codebases without minimal code churn.

TODO:
- [ ] bootstrap egressing traces to a collector
- [ ] figure out how to test envvar mixes (maybe work on test server to help with this)
- [ ] metrics support?
- [ ] logs support?

## API

```go
import (
    "github.com/tobert/otel-init-go/otelinit"
)

func main() {
    otelShutdown := otelinit.InitOpenTelemetry()
    defer otelShutdown()
}
```

## Configuration

Wherever possible environment variable names will comply to OpenTelemetry
standards.

If `OTEL_EXPORTER_OTLP_ENDPOINT` is unset or empty, the init code will
do almost nothing, so it's as safe as possible to add this to a service,
deploy it, and configure it later.

TODO:
- [ ] add config for TLS auth

| environment variable          | default         | example value  |
| ----------------------------- | --------------- | -------------- |
| OTEL_EXPORTER_OTLP_ENDPOINT   | ""              | localhost:4317 |
| OTEL_EXPORTER_OTLP_INSECURE   | false           | true           |
| OTEL_EXPORTER_OTLP_HEADERS    | ""              | key=value,k=v  |
| OTEL_EXPORTER_OTLP_BLOCKING   | false           | true           |

