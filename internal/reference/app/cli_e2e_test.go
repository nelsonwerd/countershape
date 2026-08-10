package app

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompiledCLIStableFoundationFlows(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "countershape")
	build := exec.Command("go", "build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", binary, "./cmd/countershape")
	build.Dir = repository
	build.Env = append(os.Environ(), "GOFLAGS=-p=1")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	help := invokeBinary(t, binary, repository, nil)
	if help.code != ExitOK || help.stderr != "" || !strings.Contains(help.stdout, "Countershape reference CLI") ||
		!strings.Contains(help.stdout, "does not run candidates") {
		t.Fatalf("help=%#v", help)
	}

	source := filepath.Join(repository, "spec", "examples", "v1", "source-spec.valid.json")
	validated := invokeBinary(t, binary, repository, []string{"validate", "--spec", source, "--json"})
	if validated.code != ExitOK || validated.stderr != "" {
		t.Fatalf("validate=%#v", validated)
	}
	var response ResponseEnvelope
	if err := json.Unmarshal([]byte(validated.stdout), &response); err != nil || response.Status != "SOURCE_VALID" || response.Source == nil {
		t.Fatalf("validate JSON err=%v response=%#v raw=%q", err, response, validated.stdout)
	}

	preflight := invokeBinary(t, binary, repository, []string{"preflight", "--spec", source})
	if preflight.code != ExitOK || preflight.stderr != "" || !strings.Contains(normalizedWords(preflight.stdout), TrustWarning) ||
		!strings.Contains(preflight.stdout, "Execution authorized: no") || strings.Contains(preflight.stdout, "\x1b") {
		t.Fatalf("preflight=%#v", preflight)
	}
	for _, line := range strings.Split(strings.TrimSuffix(preflight.stdout, "\n"), "\n") {
		if len(line) > 60 {
			t.Fatalf("60-column preflight overflowed (%d): %q", len(line), line)
		}
	}

	usage := invokeBinary(t, binary, repository, []string{"not-a-command"})
	if usage.code != ExitUsage || usage.stdout != "" || !strings.Contains(usage.stderr, "Code: COMMAND_UNKNOWN") ||
		!strings.Contains(usage.stderr, "Next action:") {
		t.Fatalf("usage=%#v", usage)
	}
}

func invokeBinary(t *testing.T, binary, workingDirectory string, args []string) testRun {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "NO_COLOR=1", "COLUMNS=60", "LANG=C", "LC_ALL=C", "TZ=UTC")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errorsAs(err, &exit) {
			t.Fatalf("run CLI: %v", err)
		}
		code = exit.ExitCode()
	}
	return testRun{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func errorsAs(err error, target **exec.ExitError) bool {
	exit, ok := err.(*exec.ExitError)
	if ok {
		*target = exit
	}
	return ok
}
