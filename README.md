# otel-init-go

OpenTelemetry plumbing initializer for Go that only supports OTLP/gRPC
and aims for a small code footprint and gets its configuration from environment
variables exclusively. The intent is to be able to drop this into existing
codebases with minimal code churn.

There is also an otelhelpers package in `github.com/equinix-labs/otel-init-go/otelinit`
to help with traceparent propagation. The propagation helpers depend on OTel
`otel.SetTextMapPropagator()` having been called. `otelinit.InitOpenTelemetry`
does this for you.

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

To send traces to a localhost OTLP server without encryption, you will need to
set both OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_EXPORTER_OTLP_INSECURE.

```sh
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
export OTEL_EXPORTER_OTLP_INSECURE=true
```

TODO:
- [ ] add config for TLS auth

| environment variable          | default         | example value  |
| ----------------------------- | --------------- | -------------- |
| OTEL_EXPORTER_OTLP_ENDPOINT   | ""              | localhost:4317 |
| OTEL_EXPORTER_OTLP_INSECURE   | false           | true           |
| OTEL_EXPORTER_OTLP_HEADERS    | ""              | key=value,k=v  |

