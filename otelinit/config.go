package otelinit

import (
	"log"
	"os"
	"strconv"
)

// config holds the typed values of configuration read from environment variables
type config struct {
	servicename string
	endpoint    string
	insecure    bool
}

// newConfig reads all of the documented environment variables and returns a
// config struct.
func newConfig(serviceName string) config {
	// Use stdlib to parse. If it's an invalid value and doesn't parse, log it
	// and keep going. It should already be false on error but we force it to
	// be extra clear that it's failing closed.
	insecure, err := strconv.ParseBool(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"))
	if err != nil {
		insecure = false
		log.Println("Invalid boolean value in OTEL_EXPORTER_OTLP_INSECURE. Try true or false.")
	}

	return config{
		servicename: serviceName,
		endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		insecure:    insecure,
	}
}
