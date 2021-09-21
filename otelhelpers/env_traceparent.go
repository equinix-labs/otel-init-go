package otelhelpers

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
)

// ContextWithEnvTraceparent is a helper that looks for the the TRACEPARENT
// environment variable and if it's set, it grabs the traceparent and
// adds it to the context it returns. When there is no envvar or it's
// empty, the original context is returned unmodified.
func ContextWithEnvTraceparent(ctx context.Context) context.Context {
	traceparent := os.Getenv("TRACEPARENT")
	if traceparent != "" {
		carrier := SimpleCarrier{}
		carrier.Set("traceparent", traceparent)
		prop := otel.GetTextMapPropagator()
		return prop.Extract(ctx, carrier)
	}

	return ctx
}
