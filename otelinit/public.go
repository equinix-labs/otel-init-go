package otelinit

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
