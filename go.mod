module github.com/tobert/otel-init-go

go 1.15

require (
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
	google.golang.org/grpc v1.39.0
)
