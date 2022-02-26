// otelhelpers is a package of helper functions for dealing with otel
// traceparent propagation over files and environment variables.
package otelhelpers

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"

	"go.opentelemetry.io/otel"
)

// ContextWithEnvTraceparent is a helper that looks for the the TRACEPARENT
// environment variable and if it's set, it grabs the traceparent and
// adds it to the context it returns. When there is no envvar or it's
// empty, the original context is returned unmodified.
// Depends on global OTel TextMapPropagator.
func ContextWithEnvTraceparent(ctx context.Context) context.Context {
	traceparent := os.Getenv("TRACEPARENT")
	if traceparent != "" {
		return ContextWithTraceparentString(ctx, traceparent)
	}
	return ctx
}

// ContextWithLinuxCmdlineTraceparent looks in /proc/cmdline for a traceparent=
// command line option and returns the context with that value as traceparent
// if it's there. Does no validation. Returns the original context if there is
// no cmdline option or if there's an error doing the read.
// This is Linux-only but should be safe on other operating systems.
// Depends on global OTel TextMapPropagator.
func ContextWithCmdlineTraceparent(ctx context.Context) context.Context {
	tp, err := tpFromCmdline("/proc/cmdline")
	if err != nil {
		// what to do with error? is there a way to hit the otel error handler infra?
		return ctx
	}

	return ContextWithTraceparentString(ctx, tp)
}

// ContextWithCmdlineOrEnvTraceparent checks the environment variable first,
// then /proc/cmdline and returns a context with them set, if available. When
// both are present, the cmdline is prioritized. When neither is present,
// the original context is returned as-is.
// Depends on global OTel TextMapPropagator.
func ContextWithCmdlineOrEnvTraceparent(ctx context.Context) context.Context {
	ctx = ContextWithEnvTraceparent(ctx)
	return ContextWithCmdlineTraceparent(ctx)
}

// ContextWithTraceparentString takes a W3C traceparent string, uses the otel
// carrier code to get it into a context it returns ready to go.
// Depends on global OTel TextMapPropagator.
func ContextWithTraceparentString(ctx context.Context, traceparent string) context.Context {
	carrier := SimpleCarrier{}
	carrier.Set("traceparent", traceparent)
	prop := otel.GetTextMapPropagator()
	return prop.Extract(ctx, carrier)
}

// TraceparentStringFromContext gets the current trace from the context and
// returns a W3C traceparent string. Depends on global OTel TextMapPropagator.
func TraceparentStringFromContext(ctx context.Context) string {
	carrier := SimpleCarrier{}
	prop := otel.GetTextMapPropagator()
	prop.Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// tpFromCmdline reads a /proc/cmdline style file, parses it, and returns whatever
// value is present for "traceparent=".
func tpFromCmdline(file string) (string, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	if bytes.Contains(data, []byte("traceparent=")) {
		kvpairs := bytes.Split(data, []byte(" "))
		for _, kv := range kvpairs {
			parts := bytes.SplitN(kv, []byte("="), 2)
			if string(parts[0]) == "traceparent" {
				return string(parts[1]), nil
			}
		}
	}

	return "", nil
}
