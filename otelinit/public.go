package otelinit

import "context"

type OtelShutdown func(context.Context)

// InitOpenTelemetry sets up the OpenTelemetry plumbing so it's ready to use.
// It requires a context.Context and service name string that is the name of
// your service or application.
// TODO: should even this be overrideable via envvars?
// Returns a func() that encapuslates clean shutdown.
func InitOpenTelemetry(ctx context.Context, serviceName string) OtelShutdown {
	c := newConfig(serviceName)

	if c.endpoint != "" {
		tracingShutdown := c.initTracing(ctx)
		// TODO: initMetrics()
		// TODO: initLogs()

		return func(ctx context.Context) {
			tracingShutdown(ctx)
		}
	}

	// no configuration, nothing to do, the calling code is inert
	return func(context.Context) {}
}
