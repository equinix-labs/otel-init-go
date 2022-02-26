package otelhelpers

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	// Many of the tests here won't work at all if otel is in non-recording mode
	// so we set a default endpoint and let it fail in the background.  If there
	// happens to be a listener it will connnect but should not receive spans.
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	otelinit.InitOpenTelemetry(context.Background(), "otel-init-go-helpers-test")

	os.Exit(m.Run())
}

func TestContextWithEnvTraceparent(t *testing.T) {
	// make sure the environment variable isn't polluting test state
	os.Unsetenv("TRACEPARENT")

	// trace id should not change, because there's no envvar and no file
	ctx := ContextWithEnvTraceparent(context.Background())
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		t.Error("traceparent detected where there should be none")
	}

	os.Setenv("TRACEPARENT", "00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01")
	ctx = ContextWithEnvTraceparent(context.Background())
	sc = trace.SpanContextFromContext(ctx)
	if sc.TraceID().String() != "f61fc53f926e07a9c3893b1a722e1b65" {
		t.Errorf("no trace id where one is expected. got: %q", sc.TraceID().String())
	}
	if sc.SpanID().String() != "7a2d6a804f3de137" {
		t.Errorf("no span id where one is expected. got: %q", sc.SpanID().String())
	}
	if !sc.IsSampled() {
		t.Error("expected sampling to be enabled but it is not")
	}
}

func TestTpFromCmdline(t *testing.T) {
	testTp := "00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01"
	testCmdlines := []string{
		"traceparent=00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01",
		"foo=bar initrd=lol root=wheeeeeeeeeeee-fun traceparent=00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01",
		"traceparent=00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01 foo=bar initrd=lol root=wheeeeeeeeeeee-fun",
		"kimi=ga baka=desu traceparent=00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01 foo=bar initrd=lol root=wheeeeeeeeeeee-fun",
	}

	for _, cmdline := range testCmdlines {
		file, err := ioutil.TempFile(t.TempDir(), "go-test-otel-init-go")
		if err != nil {
			t.Fatalf("unable to create tempfile for testing: %s", err)
		}
		defer os.Remove(file.Name())

		// write out a cmdline file for test
		file.WriteString(cmdline)
		file.Close()

		got, err := tpFromCmdline(file.Name())
		if err != nil {
			t.Errorf("reading cmdline test file failed unexpectedly: %s", err)
		}
		if got != testTp {
			t.Errorf("tpFromCmdline comparison failed, expected '%s', got '%s'", testTp, got)
		}
	}
}

func TestContextWithTraceparentString(t *testing.T) {
	testTp := "00-f61fc53f926e07a9c3893b1a722e1b65-7a2d6a804f3de137-01"

	ctx := ContextWithTraceparentString(context.Background(), testTp)
	tp := TraceparentStringFromContext(ctx)

	if tp != testTp {
		t.Errorf("expected %q got %q", testTp, tp)
	}
}
