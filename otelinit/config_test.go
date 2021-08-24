package otelinit

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testServiceName = "unitTestService"

func TestNewConfig(t *testing.T) {
	tests := map[string]struct {
		envIn      map[string]string
		wantConfig Config
	}{
		"empty env gets empty config": {
			envIn: map[string]string{},
			wantConfig: Config{
				Servicename: testServiceName,
			},
		},
		"irrelevant envvar changes nothing": {
			envIn: map[string]string{
				"OTEL_SOMETHING_SOMETHING": "this should impact nothing",
			},
			wantConfig: Config{
				Servicename: testServiceName,
			},
		},
		"insecure false stays false": {
			envIn: map[string]string{
				"OTEL_EXPORTER_OTLP_INSECURE": "false",
			},
			wantConfig: Config{
				Servicename: testServiceName,
				Insecure:    false,
			},
		},
		"insecure true configs true": {
			envIn: map[string]string{
				"OTEL_EXPORTER_OTLP_INSECURE": "true",
			},
			wantConfig: Config{
				Servicename: testServiceName,
				Insecure:    true,
			},
		},
		// this is by far the most common configuration expected
		"otlp endpoint and insecure": {
			envIn: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317",
				"OTEL_EXPORTER_OTLP_INSECURE": "true",
			},
			wantConfig: Config{
				Servicename: testServiceName,
				Insecure:    true,
				Endpoint:    "localhost:4317",
			},
		},
		// TODO: maybe should NOT do this, and have newConfig() check
		// incoming values and ignore obviously bad ones
		"otlp endpoint allows arbitrary value": {
			envIn: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "asdf asdf asdf",
			},
			wantConfig: Config{
				Servicename: testServiceName,
				Insecure:    false,
				Endpoint:    "asdf asdf asdf",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// wipes the whole process's env every test run
			os.Clearenv()
			// set up the test envvars
			for k, v := range tc.envIn {
				err := os.Setenv(k, v)
				if err != nil {
					t.Fatalf("could not set test environment: %s", err)
				}
			}
			// generate a config
			c := newConfig(testServiceName)
			// see if it's any good
			if diff := cmp.Diff(c, tc.wantConfig); diff != "" {
				t.Errorf(diff)
			}
		})
	}

}
