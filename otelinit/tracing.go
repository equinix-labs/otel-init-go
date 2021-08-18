package otelinit

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"
)

func (c config) initTracing(ctx context.Context) OtelShutdown {
	// set the service name that will show up in tracing UIs
	resAttrs := resource.WithAttributes(semconv.ServiceNameKey.String(c.servicename))
	res, err := resource.New(ctx, resAttrs)
	if err != nil {
		log.Fatalf("failed to create OpenTelemetry service name resource: %s", err)
	}

	grpcOpts := []otlpgrpc.Option{otlpgrpc.WithEndpoint(c.endpoint)}
	if c.insecure {
		grpcOpts = append(grpcOpts, otlpgrpc.WithInsecure())
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		grpcOpts = append(grpcOpts, otlpgrpc.WithTLSCredentials(creds))
	}
	// TODO: add TLS client cert auth

	exporter, err := otlpgrpc.New(ctx, grpcOpts...)
	if err != nil {
		log.Fatalf("failed to configure OTLP exporter: %s", err)
	}

	// TODO: more configuration opportunities here
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)

	// inject the tracer into the otel globals, start background goroutines
	otel.SetTracerProvider(tracerProvider)

	// the public function will wrap this in its own shutdown function
	return func(ctx context.Context) {
		err = tracerProvider.Shutdown(ctx)
		if err != nil {
			log.Printf("shutdown of OpenTelemetry tracerProvider failed: %s", err)
		}

		err = exporter.Shutdown(ctx)
		if err != nil {
			log.Printf("shutdown of OpenTelemetry OTLP exporter failed: %s", err)
		}
	}
}
