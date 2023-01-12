package main

// testing test-otel-init-go tests otel-init-go by using otel-cli
// to receive spans and validate things are working
//
// this is still very much a work in progress idea and might not fully
// pan out but the first part is looking good
//
// TODOs:
// [ ] replace that time.Sleep with proper synchronization
// [ ] use random ports for listener address?

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

// --------------- tid        sid
type CliEvents map[string]map[string]CliEvent

// StubData is the structure of the data that the stub program
// prints out.
type StubSpan map[string]string
type StubData struct {
	Config map[string]string `json:"config"`
	Env    map[string]string `json:"env"`
	Otel   StubSpan          `json:"otel"`
}

// Scenario represents the configuration of a test scenario. Scenarios
// are found in json files in this directory.
type Scenario struct {
	Name          string            `json:"name"`
	Filename      string            `json:"-"`
	StubEnv       map[string]string `json:"stub_env"`  // given to stub
	StubData      StubData          `json:"stub_data"` // data from stub, exact match
	SpansExpected int               `json:"spans_expected"`
	Timeout       int               `json:"timeout"`
	ShouldTimeout bool              `json:"should_timeout"` // otel connection stub->cli should fail
	SkipOtelCli   bool              `json:"skip_otel_cli"`  // don't run otel-cli at all
}

// needs to be discovered right at startup before env is cleared
var otelCliPath, testStubPath string

func TestMain(m *testing.M) {
	// find otel-cli in PATH before clearing the environment
	var err error
	otelCliPath, err = exec.LookPath("otel-cli")
	if err != nil {
		log.Fatalf("cannot run tests: otel-cli must be in PATH: %s", err)
	}

	// it is expected that the stub binary has already been built and CI does this
	wd, _ := os.Getwd()
	testStubPath = filepath.Join(wd, "test-otel-init-go")

	// wipe out this process's envvars right away to avoid pollution & leakage
	os.Clearenv()
	os.Exit(m.Run())
}

// TestOtelInit loads all the json files in this directory and executes the
// tests they define.
func TestOtelInit(t *testing.T) {
	// get a list of all json fixtures in the testdata directory
	// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
	wd, _ := os.Getwd()
	files, err := ioutil.ReadDir(filepath.Join(wd, "testdata"))
	if err != nil {
		t.Fatalf("Failed to list test directory %q to detect json files.", wd)
	}

	scenarios := []Scenario{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			scenario := Scenario{StubEnv: map[string]string{}}
			fp := filepath.Join("testdata", file.Name())
			js, err := os.ReadFile(fp)
			if err != nil {
				t.Fatalf("Failed to read json test file %q: %s", file.Name(), err)
			}
			err = json.Unmarshal(js, &scenario)
			if err != nil {
				t.Fatalf("Failed to parse json test file %q: %s", file.Name(), err)
			}
			scenario.Filename = filepath.Base(file.Name()) // for error reporting
			scenarios = append(scenarios, scenario)
		}
	}

	t.Logf("Loaded %d tests.", len(scenarios))
	if len(scenarios) == 0 {
		t.Fatal("no test fixtures loaded!")
	}

	// run all the scenarios, check the results
	for _, s := range scenarios {
		stubData, events := runPrograms(t, s)
		checkData(t, s, stubData, events)
	}
}

// checkData takes the data returned from the stub and compares it to the
// preset data in the scenario and fails the tests if anything doesn't match.
func checkData(t *testing.T, scenario Scenario, stubData StubData, events CliEvents) {
	// check the env
	if diff := cmp.Diff(scenario.StubData.Env, stubData.Env); diff != "" {
		t.Errorf("env data did not match fixture in %q (-want +got):\n%s", scenario.Filename, diff)
	}

	// check the otel-init-go config
	if diff := cmp.Diff(scenario.StubData.Config, stubData.Config); diff != "" {
		t.Errorf("config data did not match fixture in %q (-want +got):\n%s", scenario.Filename, diff)
	}

	// check the otel span values
	// find usages of *, do the check on the stub data manually, and set up cmpSpan
	scSpan := map[string]string{}  // to be passed to cmp.Diff
	cmpSpan := map[string]string{} // to be passed to cmp.Diff
	for what, re := range map[string]*regexp.Regexp{
		"trace_id":    regexp.MustCompile("^[0-9a-fA-F]{32}$"),
		"span_id":     regexp.MustCompile("^[0-9a-fA-F]{16}$"),
		"is_sampled":  regexp.MustCompile("^true|false$"),
		"trace_flags": regexp.MustCompile("^[0-9]{2}$"),
	} {
		if cv, ok := scenario.StubData.Otel[what]; ok {
			scSpan[what] = cv // make a straight copy to make cmp.Diff happy
			if sv, ok := stubData.Otel[what]; ok {
				cmpSpan[what] = sv // default to the existing value
				if cv == "*" {
					if re.MatchString(sv) {
						cmpSpan[what] = "*" // success!, make the Cmp test succeed
					} else {
						t.Errorf("stub span value %q for key %s is not valid", sv, what)
					}
				}
			}
		}
	}

	// do a diff on a generated map that sets values to * when the * check succeeded
	if diff := cmp.Diff(scSpan, cmpSpan); diff != "" {
		t.Errorf("otel data did not match fixture in %q (-want +got):\n%s", scenario.Filename, diff)
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
	otelcli := exec.Command(otelCliPath, cliArgs...)
	otelcli.Env = []string{"PATH=/bin"} // apparently PATH=/bin is required for 'getent', no idea why
	var b bytes.Buffer
	otelcli.Stdout = &b
	otelcli.Stderr = &b

	if !scenario.SkipOtelCli {
		// run otel-cli server in the background
		go func() {
			err := otelcli.Start()
			otelcli.Wait() // don't check error, it's fast and errors a lot for no reason
			if err != nil {
				log.Println(b.String())
				log.Fatalf("Executing command %q failed: %s", otelcli.String(), err)
			}
		}()
	}

	// TODO: replace this with something more reliable
	// the problem here is we need to wait for otel-cli to start and listen
	// so the solution is probably to do some kind of healthcheck on otel-cli's port
	// but this works ok for now
	time.Sleep(time.Millisecond * 10)

	// run the stub with the scenario's environment
	stub := exec.Command(testStubPath)
	stub.Env = mkEnviron(scenario.StubEnv)
	stubOut, err := stub.Output()
	if err != nil {
		t.Fatalf("Executing stub command %q failed: %s", stub.String(), err)
	}

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
