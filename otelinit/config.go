package otelinit

import (
	"log"
	"os"
	"strconv"
)

// Config holds the typed values of configuration read from the environment.
// It is public mainly to make testing easier and most users should never
// use it directly.
type Config struct {
	Servicename string `json:"service_name"`
	Endpoint    string `json:"endpoint"`
	Insecure    bool   `json:"insecure"`
}

// newConfig reads all of the documented environment variables and returns a
// config struct.
func newConfig(serviceName string) Config {
	// Use stdlib to parse. If it's an invalid value and doesn't parse, log it
	// and keep going. It should already be false on error but we force it to
	// be extra clear that it's failing closed.
	isEnv := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")
	var insecure bool
	if isEnv != "" {
		var err error
		insecure, err = strconv.ParseBool(isEnv)
		if err != nil {
			insecure = false
			log.Println("Invalid boolean value in OTEL_EXPORTER_OTLP_INSECURE. Try true or false.")
		}
	} else {
		insecure = false
	}

	return Config{
		Servicename: serviceName,
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:    insecure,
	}
}
