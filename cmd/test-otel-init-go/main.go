package main

// All this program does is load up otel-init-go, create one trace, dump state
// in json for all these things, then exit. This data is intended to be consumed
// in main_test.go, which is really about testing otel-cli-init itself.

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx := context.Background()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "otel-init-go-test")
	defer otelShutdown(ctx)

	tracer := otel.Tracer("otel-init-go-test")
	ctx, span := tracer.Start(ctx, "dump state")

	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		} else {
			log.Fatalf("BUG this shouldn't happen")
		}
	}

	// public.go stuffs the config in the context just so we can do this
	conf, ok := otelinit.ConfigFromContext(ctx)
	if !ok {
		log.Println("failed to retrieve otelinit.Config pointer from context, test results may be invalid")
		conf = &otelinit.Config{}
	}
	sc := span.SpanContext()
	outData := map[string]map[string]string{
		"config": {
			"endpoint":     conf.Endpoint,
			"service_name": conf.Servicename,
			"insecure":     strconv.FormatBool(conf.Insecure),
		},
		"otel": {
			"trace_id":    sc.TraceID().String(),
			"span_id":     sc.SpanID().String(),
			"trace_flags": sc.TraceFlags().String(),
			"is_sampled":  strconv.FormatBool(sc.IsSampled()),
		},
		"env": env,
	}

	js, err := json.MarshalIndent(outData, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(js)
	os.Stdout.WriteString("\n")

	span.End()
}
