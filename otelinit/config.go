package otelinit

import "os"

// config holds the typed values of configuration read from environment variables
type config struct {
	servicename string
	endpoint    string
	insecure    bool
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
