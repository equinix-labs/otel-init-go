package otelinit

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"
)

// config holds the typed values of configuration read from environment variables
type config struct {
	servicename string
	endpoint    string
	insecure    bool
}

// InitOpenTelemetry sets up the OpenTelemetry plumbing so it's ready to use.
// It requires a service name string that is the name of your service or application.
// TODO: should even this be overrideable via envvars?
// Returns a func() that encapuslates clean shutdown.
func InitOpenTelemetry(serviceName string) func() {
	c := newConfig(serviceName)

	if c.endpoint != "" {
		tracingShutdown := c.initTracing()
		// TODO: initMetrics()
		// TODO: initLogs()

		return func() {
			tracingShutdown()
		}
	}

	// no configuration, nothing to do, the calling code is inert
	return func() {}
}

// newConfig reads all of the documented environment variables and returns a
// config struct.
func newConfig(serviceName string) config {
	// TODO: actually read the envvars & definitely do not hard-code insecure=true
	return config{
		servicename: serviceName,
		endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		insecure:    true, // BAD, will replace this very soon
	}
}

func (c config) initTracing() func() {
	ctx := context.Background()

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
	return func() {
		err = tracerProvider.Shutdown(ctx)
		if err != nil {
			log.Fatalf("shutdown of OpenTelemetry tracerProvider failed: %s", err)
		}

		err = exporter.Shutdown(ctx)
		if err != nil {
			log.Fatalf("shutdown of OpenTelemetry OTLP exporter failed: %s", err)
		}
	}
}
