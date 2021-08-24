package otelinit

import "context"

// OtelShutdown is a function that should be called with context
// when you want to shut down OpenTelemetry, usually as a defer
// in main.
type OtelShutdown func(context.Context)

// InitOpenTelemetry sets up the OpenTelemetry plumbing so it's ready to use.
// It requires a context.Context and service name string that is the name of
// your service or application.
// TODO: should even this be overrideable via envvars?
// Returns context and a func() that encapuslates clean shutdown.
func InitOpenTelemetry(ctx context.Context, serviceName string) (context.Context, OtelShutdown) {
	c := newConfig(serviceName)

	// no idea if this is gonna work...
	// or even if this is a good idea but it would be well out of most folks'
	// way here and I can snag it from test code without burdening anyone else
	// and it's a teensy amount of memory
	ctx = context.WithValue(ctx, "otel-init-config", &c)

	if c.Endpoint != "" {
		ctx, tracingShutdown := c.initTracing(ctx)
		// TODO: initMetrics()
		// TODO: initLogs()

		return ctx, func(ctx context.Context) {
			tracingShutdown(ctx)
		}
	}

	// no configuration, nothing to do, the calling code is inert
	// config is available in the returned context (for test/debug)
	return ctx, func(context.Context) {}
}

// ConfigFromContext extracts the Config struct from the provided context.
// Returns the Config and true if it was retried successfully, false otherwise.
func ConfigFromContext(ctx context.Context) (*Config, bool) {
	raw := ctx.Value("otel-init-config")
	if raw != nil {
		if conf, ok := raw.(*Config); ok {
			return conf, true
		}
	}

	return &Config{}, false
}
