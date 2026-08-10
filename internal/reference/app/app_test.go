package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode"
)

func TestHelpIsMonochromeAndPhaseHonest(t *testing.T) {
	result := runForTest(t, nil, nil, "")
	if result.code != ExitOK || result.stderr != "" {
		t.Fatalf("help exit=%d stderr=%q", result.code, result.stderr)
	}
	for _, exact := range []string{
		"countershape validate --spec <path|-> [--json]",
		"countershape preflight --spec <path|-> [--json]",
		"It does not run candidates, resume processes, compare outcomes, or emit contracts.",
	} {
		if !strings.Contains(normalizedWords(result.stdout), exact) {
			t.Fatalf("help is missing %q:\n%s", exact, result.stdout)
		}
	}
	assertNoTerminalControl(t, result.stdout)
}

func TestArgumentGrammarRefusesAmbiguity(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		code    string
		machine bool
	}{
		{name: "unknown command", args: []string{"run"}, code: "COMMAND_UNKNOWN"},
		{name: "missing spec", args: []string{"validate"}, code: "ARGUMENT_MISSING"},
		{name: "missing value", args: []string{"validate", "--spec"}, code: "ARGUMENT_MISSING"},
		{name: "json consumed as value", args: []string{"validate", "--spec", "--json"}, code: "ARGUMENT_MISSING"},
		{name: "spec consumed as value", args: []string{"validate", "--spec", "--spec"}, code: "ARGUMENT_MISSING"},
		{name: "duplicate spec", args: []string{"validate", "--spec", "a", "--spec", "b"}, code: "ARGUMENT_DUPLICATE"},
		{name: "duplicate json", args: []string{"validate", "--json", "--json", "--spec", "-"}, code: "ARGUMENT_DUPLICATE", machine: true},
		{name: "unknown option", args: []string{"preflight", "--spec", "-", "--latest"}, code: "ARGUMENT_UNKNOWN"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result := runForTest(t, test.args, strings.NewReader("{}"), "")
			output := result.stderr
			if test.machine {
				output = result.stdout
			}
			if result.code != ExitUsage || !strings.Contains(output, test.code) {
				t.Fatalf("exit=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
			if !strings.Contains(output, "next_action") && !strings.Contains(output, "Next action:") {
				t.Fatal("usage refusal omitted its safe next action")
			}
		})
	}
}

func TestDashLeadingSpecPathRequiresExplicitRelativePrefix(t *testing.T) {
	parsed, err := parseInvocation([]string{"validate", "--spec", "./--json"})
	if err != nil || parsed.specPath != "./--json" || parsed.json {
		t.Fatalf("explicit dash-leading path was not preserved: parsed=%#v err=%v", parsed, err)
	}
}

func TestStrictSourceAuthorityAndJSONRefusals(t *testing.T) {
	valid := validSourceBytes(t)
	validRun := runForTest(t, []string{"validate", "--spec", "-", "--json"}, bytes.NewReader(valid), "")
	if validRun.code != ExitOK || validRun.stderr != "" {
		t.Fatalf("valid exit=%d stderr=%q", validRun.code, validRun.stderr)
	}
	var accepted ResponseEnvelope
	if err := json.Unmarshal([]byte(validRun.stdout), &accepted); err != nil {
		t.Fatalf("valid JSON: %v; %q", err, validRun.stdout)
	}
	if accepted.Status != "SOURCE_VALID" || accepted.Source == nil || accepted.Source.SourceDigest == "" ||
		accepted.Source.AdapterDomain != "CLI" || accepted.Source.ExecutionShape != "ONE_CLI_INVOCATION" ||
		accepted.Source.Budgets.ReadinessMS != 0 {
		t.Fatalf("unexpected accepted envelope: %#v", accepted)
	}

	base := string(valid)
	cases := []struct {
		name string
		raw  []byte
		code string
	}{
		{name: "duplicate", raw: []byte(strings.Replace(base, `"kind": "SourceSpec",`, `"kind": "SourceSpec", "kind": "SourceSpec",`, 1)), code: "CANON_DUPLICATE_KEY"},
		{name: "negative zero", raw: []byte(strings.Replace(base, `"required_tools":`, `"budgets":{"candidate_count":-0}, "required_tools":`, 1)), code: "CANON_NEGATIVE_ZERO"},
		{name: "unsafe integer", raw: []byte(strings.Replace(base, `"required_tools":`, `"budgets":{"candidate_count":9007199254740992}, "required_tools":`, 1)), code: "CANON_UNSAFE_INTEGER"},
		{name: "lone surrogate", raw: []byte(strings.Replace(base, `"kind": "SourceSpec"`, `"kind": "Source\uD800Spec"`, 1)), code: "CANON_LONE_SURROGATE"},
		{name: "trailing", raw: append(append([]byte(nil), valid...), []byte("\n{}")...), code: "CANON_TRAILING_DATA"},
		{name: "unknown member", raw: []byte(strings.Replace(base, `"kind": "SourceSpec",`, `"kind": "SourceSpec", "surprise": true,`, 1)), code: "UNKNOWN_SOURCE_KEY"},
		{name: "invalid utf8", raw: append([]byte{0xff}, valid...), code: "CANON_INVALID_UTF8"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result := runForTest(t, []string{"validate", "--spec", "-", "--json"}, bytes.NewReader(test.raw), "")
			if result.code != ExitRefused || result.stderr != "" {
				t.Fatalf("exit=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
			var refused ResponseEnvelope
			if err := json.Unmarshal([]byte(result.stdout), &refused); err != nil {
				t.Fatalf("refusal JSON: %v; %q", err, result.stdout)
			}
			if refused.Failure == nil || refused.Failure.Code != test.code || refused.NextAction == "" {
				t.Fatalf("refusal=%#v, want %s", refused, test.code)
			}
			if bytes.Contains([]byte(result.stdout), test.raw) {
				t.Fatal("refusal echoed the source bytes")
			}
		})
	}
}

func TestHumanSourceRefusalEscapesTerminalControls(t *testing.T) {
	base := string(validSourceBytes(t))
	hostileName := `\u001b]0;pwn\u0007\u009b31m`
	hostile := []byte(strings.Replace(base, `"kind": "SourceSpec",`, `"kind": "SourceSpec", "`+hostileName+`": true,`, 1))
	result := runForTest(t, []string{"validate", "--spec", "-"}, bytes.NewReader(hostile), "")
	if result.code != ExitRefused || result.stdout != "" || !strings.Contains(result.stderr, "UNKNOWN_SOURCE_KEY") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
	for _, escaped := range []string{`\u001b`, `\u0007`, `\u009b`} {
		if !strings.Contains(result.stderr, escaped) {
			t.Fatalf("human refusal omitted visible escape %q: %q", escaped, result.stderr)
		}
	}
	assertNoTerminalControl(t, result.stderr)
}

func TestPreflightIsUsefulWithoutClaimingExecutionAuthority(t *testing.T) {
	result := runForTest(t, []string{"preflight", "--spec", "-", "--json"}, bytes.NewReader(validSourceBytes(t)), "")
	if result.code != ExitOK || result.stderr != "" {
		t.Fatalf("exit=%d stderr=%q", result.code, result.stderr)
	}
	var response ResponseEnvelope
	if err := json.Unmarshal([]byte(result.stdout), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "SOURCE_VALID_DEPENDENCIES_UNRESOLVED" || response.Warning == nil || *response.Warning != TrustWarning ||
		response.Preflight == nil || response.Preflight.ExecutionStarted || response.Preflight.ExecutionAuthorized ||
		response.Preflight.ProjectionAuthority != "UNRESOLVED" || response.Preflight.CandidateAuthority != "UNRESOLVED" ||
		response.Preflight.RuntimeAuthority != "UNRESOLVED" || response.NextAction == "" {
		t.Fatalf("preflight overclaimed authority: %#v", response)
	}

	human := runForTest(t, []string{"preflight", "--spec", "-"}, bytes.NewReader(validSourceBytes(t)), "")
	if human.code != ExitOK || human.stderr != "" || !strings.Contains(normalizedWords(human.stdout), TrustWarning) ||
		!strings.Contains(human.stdout, "Execution started: no") || !strings.Contains(human.stdout, "Execution authorized: no") ||
		!strings.Contains(human.stdout, "Next action:") {
		t.Fatalf("human preflight=%#v", human)
	}
	assertNoTerminalControl(t, human.stdout)
}

func TestStableFileReadRejectsLinksSpecialFilesAndOversize(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "source.json")
	if err := os.WriteFile(validPath, validSourceBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked.json")
	if err := os.Symlink(validPath, linked); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"linked.json", "missing.json"} {
		result := runForTest(t, []string{"validate", "--spec", path, "--json"}, nil, root)
		if result.code != ExitInput {
			t.Fatalf("%s exit=%d stdout=%q", path, result.code, result.stdout)
		}
	}
	oversize := bytes.Repeat([]byte{'x'}, maximumSourceBytes+1)
	result := runForTest(t, []string{"validate", "--spec", "-", "--json"}, bytes.NewReader(oversize), root)
	if result.code != ExitInput || !strings.Contains(result.stdout, "INPUT_SIZE_INVALID") {
		t.Fatalf("oversize exit=%d stdout=%q", result.code, result.stdout)
	}
}

func TestRenderPreservesSuppliedTransportFacts(t *testing.T) {
	warning := "supplied warning"
	summary := SpecSummary{
		Input: "fixture", SourceDigest: "supplied-source-reference", AdapterDomain: "HTTP",
		ExecutionShape: "ONE_CLI_INVOCATION", Budgets: BudgetView{CandidateCount: 4, TotalCandidateTrials: 17},
	}
	response := baseEnvelope("test", "SUPPLIED", "supplied message", "supplied next", ExitOK)
	response.Warning = &warning
	response.Source = &summary
	response.Preflight = &PreflightView{
		ExecutionStarted: true, ExecutionAuthorized: true,
		ProjectionAuthority: "SUPPLIED", CandidateAuthority: "UNRESOLVED", RuntimeAuthority: "SUPPLIED",
	}
	var human bytes.Buffer
	if err := renderHuman(&human, response, 80); err != nil {
		t.Fatal(err)
	}
	for _, exact := range []string{"HTTP / ONE_CLI_INVOCATION", "supplied-source-reference", "Candidate ceiling: 4", "Trial ceiling: 17", "supplied warning", "supplied next", "Execution started: yes", "Execution authorized: yes", "Unresolved: candidates"} {
		if !strings.Contains(normalizedWords(human.String()), exact) {
			t.Fatalf("renderer recomputed or omitted %q:\n%s", exact, human.String())
		}
	}
	var machine bytes.Buffer
	if err := renderJSON(&machine, response); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(machine.String(), "\n") || strings.Count(machine.String(), "\n") != 1 {
		t.Fatalf("JSON framing = %q", machine.String())
	}
}

func TestValidateNextActionIsReplaySafeAndInputHonest(t *testing.T) {
	stdin := runForTest(t, []string{"validate", "--spec", "-"}, bytes.NewReader(validSourceBytes(t)), "")
	if stdin.code != ExitOK || !strings.Contains(stdin.stdout, "stdin bytes are not retained") || strings.Contains(stdin.stdout, "--spec stdin") {
		t.Fatalf("stdin next action is not honest: %#v", stdin)
	}

	root := t.TempDir()
	name := "source with '$HOME'.json"
	if err := os.WriteFile(filepath.Join(root, name), validSourceBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	file := runForTest(t, []string{"validate", "--spec", name}, nil, root)
	want := "--spec 'source with '\"'\"'$HOME'\"'\"'.json'"
	if file.code != ExitOK || !strings.Contains(normalizedWords(file.stdout), want) || !strings.Contains(file.stdout, "Source: "+name) {
		t.Fatalf("file next action is not safely quoted: want=%q run=%#v", want, file)
	}
	for _, hostile := range []string{"bidi-\u202ereversed.json", "line-\u2028separator.json"} {
		if validInputPath(hostile) {
			t.Fatalf("display-hostile path was admitted: %q", hostile)
		}
	}
}

func TestConcurrentRunsShareNoMutableState(t *testing.T) {
	const count = 16
	var wait sync.WaitGroup
	errorsFound := make(chan error, count)
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result := runForTest(t, []string{"validate", "--spec", "-", "--json"}, bytes.NewReader(validSourceBytes(t)), "")
			if result.code != ExitOK || result.stderr != "" || !strings.Contains(result.stdout, `"status":"SOURCE_VALID"`) {
				errorsFound <- fmt.Errorf("result=%#v", result)
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func TestWriterFailuresReturnInternalExit(t *testing.T) {
	runtime := Runtime{Stdin: strings.NewReader(""), Stdout: failingWriter{}, Stderr: failingWriter{}}
	if code := Run(nil, runtime); code != ExitInternal {
		t.Fatalf("writer failure exit=%d", code)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type testRun struct {
	code   int
	stdout string
	stderr string
}

func runForTest(t *testing.T, args []string, stdin io.Reader, workingDirectory string) testRun {
	t.Helper()
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	var stdout, stderr bytes.Buffer
	code := Run(args, Runtime{Stdin: stdin, Stdout: &stdout, Stderr: &stderr, WorkingDirectory: workingDirectory})
	return testRun{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func validSourceBytes(t *testing.T) []byte {
	t.Helper()
	exact, err := os.ReadFile(filepath.Join("..", "..", "..", "spec", "examples", "v1", "source-spec.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func assertNoTerminalControl(t *testing.T, value string) {
	t.Helper()
	for _, current := range value {
		if current == '\n' {
			continue
		}
		if unicode.IsControl(current) || unicode.Is(unicode.Cf, current) || current == '\u2028' || current == '\u2029' {
			t.Fatalf("terminal control rune %U present: %q", current, value)
		}
	}
}

func normalizedWords(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func TestHumanRenderingHonorsAdmittedTerminalWidths(t *testing.T) {
	response := preflightSourceCommand("-", Runtime{Stdin: bytes.NewReader(validSourceBytes(t))})
	for _, columns := range []int{60, 80, 120} {
		var output bytes.Buffer
		if err := renderHuman(&output, response, columns); err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
			if len(line) > columns {
				t.Fatalf("width=%d line=%d: %q", columns, len(line), line)
			}
		}
		if !strings.Contains(normalizedWords(output.String()), TrustWarning) || !strings.Contains(output.String(), "Next action:") {
			t.Fatalf("width=%d omitted authority facts:\n%s", columns, output.String())
		}
	}
	for _, raw := range []string{"", "39", "241", "not-a-number"} {
		if TerminalColumns(raw) != defaultTerminalColumns {
			t.Fatalf("TerminalColumns(%q) admitted an unsafe width", raw)
		}
	}
}
