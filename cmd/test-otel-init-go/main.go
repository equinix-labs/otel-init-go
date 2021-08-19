package main

import (
	"context"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx := context.Background()
	otelShutdown := otelinit.InitOpenTelemetry(ctx, "otel-init-go-test")
	defer otelShutdown(ctx)

	tracer := otel.Tracer("otel-init-go-test")
	_, span := tracer.Start(ctx, "test span 1")
	span.End()
}
