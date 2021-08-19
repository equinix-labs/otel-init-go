# otel-init-go

OpenTelemetry plumbing initializer for Go that only supports OTLP/gRPC
and aims for a small code footprint and gets its configuration from environment
variables exclusively. The intent is to be able to drop this into existing
codebases with minimal code churn.

TODO:
- [x] bootstrap egressing traces to a collector
- [ ] figure out how to test envvar mixes (otel-cli server json)
- [ ] metrics support?
- [ ] logs support?

## API

```go
package main

import (
    "github.com/equinix-labs/otel-init-go/otelinit"
)

func main() {
    ctx := context.Background()
    ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "my-amazing-application")
    defer otelShutdown(ctx)
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

