package otelhelpers

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestSimpleCarrier(t *testing.T) {
	carrier := SimpleCarrier{}
	carrier.Clear() // clean up after other tests

	// traceparent is the only key supported by SimpleCarrier
	got := carrier.Get("traceparent")
	if got != "" {
		t.Errorf("got a non-empty traceparent value '%s' where empty string was expected", got)
	}

	carrier.Set("foobar", "baz")
	if carrier.Get("foobar") != "baz" {
		t.Error("did not get the expected value back in Set/Get test")
	}

	// traceparent is supported so this should work fine
	tp := "00-b122b620341449410b9cd900c96d459d-aa21cda35388b694-01"
	carrier.Set("traceparent", tp)

	// we've set 2 keys so far, and both should get returned
	keys := carrier.Keys()
	if len(keys) != 2 {
		t.Errorf("expected exactly 2 keys from Keys() but instead got %q", keys)
	}

	// make sure the value round-trips in one piece
	got = carrier.Get("traceparent")
	if got != tp {
		t.Errorf("expected traceparent value '%s' but got '%s'", tp, got)
	}

	// it's impractical to test the internal state of otel-go, so the next best
	// thing is to round-trip our traceparent through it and make sure it comes
	// back as expected
	prop := otel.GetTextMapPropagator()
	ctx := prop.Extract(context.Background(), carrier)
	if ctx == nil {
		t.Errorf("expected a context but got nil, likely a problem in otel? this shouldn't happen...")
	}

	// try to round trip the traceparent back out of that context ^^
	rtCarrier := SimpleCarrier{}
	prop.Inject(ctx, rtCarrier)
	got = carrier.Get("traceparent")
	if got != tp {
		t.Errorf("round-tripping traceparent through a context failed, expected '%s', got '%s'", tp, got)
	}

	carrier.Clear() // clean up for other tests
}
