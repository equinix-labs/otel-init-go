package main

// testing test-otel-init-go tests otel-init-go by using otel-cli
// to receive spans and validate things are working
//
// this is still very much a work in progress idea and might not fully
// pan out but the first part is looking good
//
// TODOs:
// [ ] write data tests
// [ ] replace that time.Sleep with proper synchronization
// [ ] use random ports for listener address?

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// CliEvent is mostly the same as otel-cli's internal event format, with
// the addition that it has a place to stuff events.
type CliEvent struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_span_id"`
	Library    string            `json:"library"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	Start      string            `json:"start"`
	End        string            `json:"end"`
	ElapsedMS  int               `json:"elapsed_ms"`
	Attributes map[string]string `json:"attributes"`
	Events     []CliEvent        // reader code will stuff kind=event in here
}

//                 tid        sid
type CliEvents map[string]map[string]CliEvent

// StubData is the structure of the data that the stub program
// prints out.
type StubData struct {
	Config map[string]string `json:"config"`
	Env    map[string]string `json:"env"`
	Otel   map[string]string `json:"otel"`
}

// Scenario represents the configuration of a test scenario. Scenarios
// are found in json files in this directory.
type Scenario struct {
	Name          string            `json:"name"`
	StubEnv       map[string]string `json:"stub_env"`  // given to stub
	StubData      StubData          `json:"stub_data"` // data from stub, exact match
	SpansExpected int               `json:"spans_expected"`
	Timeout       int               `json:"timeout"`
	ShouldTimeout bool              `json:"should_timeout"` // otel connection stub->cli should fail
	SkipOtelCli   bool              `json:"skip_otel_cli"`  // don't run otel-cli at all
}

func TestMain(m *testing.M) {
	// wipe out this process's envvars right away to avoid pollution & leakage
	os.Clearenv()
	os.Exit(m.Run())
}

// TestOtelInit loads all the json files in this directory and executes the
// tests they define.
func TestOtelInit(t *testing.T) {
	// get a list of all json files in this directory
	// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
	wd, _ := os.Getwd()
	files, err := ioutil.ReadDir(wd)
	if err != nil {
		t.Fatalf("Failed to list test directory %q to detect json files.", wd)
	}

	scenarios := []Scenario{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			scenario := Scenario{StubEnv: map[string]string{}}
			js, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatalf("Failed to read json test file %q: %s", file.Name(), err)
			}
			err = json.Unmarshal(js, &scenario)
			if err != nil {
				t.Fatalf("Failed to parse json test file %q: %s", file.Name(), err)
			}
			scenarios = append(scenarios, scenario)
		}
	}

	t.Logf("Loaded %d tests.", len(scenarios))

	// run all the scenarios
	for _, s := range scenarios {
		stubData, events := runPrograms(t, s)
		checkData(t, s, stubData, events)

		// temporary debug print
		for tid, trace := range events {
			for sid, span := range trace {
				t.Logf("trace id %s span id %s is %q", tid, sid, span)
			}
		}
	}
}

// checkData takes the data returned from the stub and compares it to the
// preset data in the scenario and fails the tests if anything doesn't match.
func checkData(t *testing.T, scenario Scenario, stubData StubData, events CliEvents) {
	// check the env
	if !reflect.DeepEqual(stubData.Env, scenario.StubData.Env) {
		t.Log("env in stub output did not match test fixture")
		t.Fail()
	}

	// check the otel-init-go config
	if !reflect.DeepEqual(stubData.Config, scenario.StubData.Config) {
		t.Log("config in stub output did not match test fixture")
		t.Fail()
	}

	// check the otel span values
	if !reflect.DeepEqual(stubData.Otel, scenario.StubData.Otel) {
		t.Log("span in stub output did not match test fixture")
		t.Fail()
	}
}

// runPrograms runs the stub program and otel-cli together and captures their
// output as data to return for further testing.
// all failures are fatal, no point in testing if this is broken
func runPrograms(t *testing.T, scenario Scenario) (StubData, CliEvents) {
	tmpdir, err := os.MkdirTemp(os.TempDir(), "otel-init-go-test")
	defer os.RemoveAll(tmpdir)
	if err != nil {
		t.Fatalf("MkdirTemp failed: %s", err)
	}

	cliArgs := []string{"server", "json", "--dir", tmpdir}

	if scenario.Timeout > 0 {
		cliArgs = append(cliArgs, "--timeout", strconv.Itoa(scenario.Timeout))
	}

	if scenario.SpansExpected > 0 {
		cliArgs = append(cliArgs, "--max-spans", strconv.Itoa(scenario.SpansExpected))
	}

	// MAYBE: server json --stdout is maybe better? and could add a graceful exit on closed fds
	// TODO: obviously this is horrible
	otelcli := exec.Command("/home/atobey/src/otel-cli/otel-cli", cliArgs...)

	if !scenario.SkipOtelCli {
		go func() {
			err = otelcli.Run()
			if err != nil {
				log.Fatalf("Executing command %q failed: %s", otelcli.String(), err)
			}
		}()
	}

	// yes yes I know this is horrible
	time.Sleep(time.Millisecond * 10)

	// TODO: obviously this is horrible
	stub := exec.Command("/home/atobey/src/otel-init-go/cmd/test-otel-init-go/test-otel-init-go")
	stub.Env = mkEnviron(scenario.StubEnv)
	stubOut, err := stub.Output()
	if err != nil {
		t.Fatalf("Executing stub command %q failed: %s", stub.String(), err)
	}

	fmt.Printf("\n\n%s\n\n", string(stubOut))

	stubData := StubData{
		Config: map[string]string{},
		Env:    map[string]string{},
		Otel:   map[string]string{},
	}
	err = json.Unmarshal(stubOut, &stubData)
	if err != nil {
		fmt.Printf("\n\n%s\n\n", string(stubOut))
		t.Fatalf("Unmarshaling stub output failed: %s", err)
	}

	if !scenario.SkipOtelCli {
		otelcli.Wait()
	}

	events := make(CliEvents)
	filepath.WalkDir(tmpdir, func(path string, d fs.DirEntry, err error) error {
		// TODO: make sure to read span.json before events.json
		// so maybe a directory walk would be better anyways
		if strings.HasSuffix(path, ".json") {
			pi := strings.Split(path, string(os.PathSeparator))
			if len(pi) >= 3 {
				js, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatalf("error while reading file %q: %s", path, err)
				}

				evt := CliEvent{
					Attributes: make(map[string]string),
					Events:     make([]CliEvent, 0),
				}
				err = json.Unmarshal(js, &evt)
				if err != nil {
					t.Fatalf("error while parsing json file %q: %s", path, err)
				}

				tid := pi[len(pi)-3]
				sid := pi[len(pi)-2]
				if trace, ok := events[tid]; ok {
					if _, ok := trace[sid]; ok {
						t.Fatal("unfinished code path")
					}
					trace[sid] = evt
				} else {
					events[tid] = make(map[string]CliEvent)
					events[tid][sid] = evt
				}
				// TODO: events
			}
		}
		return nil
	})

	return stubData, events
}

// mkEnviron converts a string map to a list of k=v strings.
func mkEnviron(env map[string]string) []string {
	mapped := make([]string, len(env))
	var i int
	for k, v := range env {
		mapped[i] = k + "=" + v
		i++
	}

	return mapped
}
