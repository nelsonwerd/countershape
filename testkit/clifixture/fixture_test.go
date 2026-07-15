package clifixture

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

type fixtureRun struct {
	stdout  []byte
	stderr  []byte
	receipt invocationReceipt
	err     error
}

type invocationReceipt struct {
	AttemptID   string   `json:"attempt_id"`
	LogicalArgv []string `json:"logical_argv"`
}

func TestCandidateFilesAreClosedAndDistinct(t *testing.T) {
	t.Parallel()
	seen := map[string]struct{}{}
	for _, role := range Roles() {
		files, err := CandidateFiles(role)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 || files[0].Path != "candidate-role.json" || files[1].Path != Entrypoint {
			t.Fatalf("unexpected files for %s: %#v", role, files)
		}
		if !bytes.Equal(files[1].Content, Program()) {
			t.Fatalf("program bytes changed for %s", role)
		}
		identity := string(files[0].Content)
		if _, duplicate := seen[identity]; duplicate {
			t.Fatalf("duplicate role identity %q", identity)
		}
		seen[identity] = struct{}{}
	}
}

func TestPrecedenceCandidatesProduceThreeExactValues(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	want := map[CandidateRole]map[string]string{
		ConfigFirst:      {"mode": "config", "source": "config"},
		EnvironmentFirst: {"mode": "env", "source": "env"},
		ArgvFirst:        {"mode": "argv", "source": "argv"},
	}
	for _, role := range Roles() {
		role := role
		t.Run(string(role), func(t *testing.T) {
			run := runFixture(t, node, role, []string{"--mode", "argv"}, map[string]string{"APP_MODE": "env"})
			if run.err != nil {
				t.Fatalf("fixture failed: %v; stderr=%q", run.err, run.stderr)
			}
			var got map[string]string
			if err := json.Unmarshal(run.stdout, &got); err != nil {
				t.Fatalf("invalid stdout JSON %q: %v", run.stdout, err)
			}
			if !reflect.DeepEqual(got, want[role]) {
				t.Fatalf("got %#v, want %#v", got, want[role])
			}
			wantArgv := []string{"node", Entrypoint, "--mode", "argv"}
			if !reflect.DeepEqual(run.receipt.LogicalArgv, wantArgv) {
				t.Fatalf("receipt argv = %#v, want %#v", run.receipt.LogicalArgv, wantArgv)
			}
		})
	}
}

func TestEnvironmentAbsentDiffersFromPresentEmpty(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	absent := runFixture(t, node, EnvironmentFirst, nil, nil)
	presentEmpty := runFixture(t, node, EnvironmentFirst, nil, map[string]string{"APP_MODE": ""})
	if absent.err != nil || presentEmpty.err != nil {
		t.Fatalf("fixture failed: absent=%v empty=%v", absent.err, presentEmpty.err)
	}
	if string(absent.stdout) != `{"mode":"config","source":"config"}` {
		t.Fatalf("absent env result = %q", absent.stdout)
	}
	if string(presentEmpty.stdout) != `{"mode":"","source":"env"}` {
		t.Fatalf("present-empty env result = %q", presentEmpty.stdout)
	}
}

func TestAlternatingUsesRunnerRepetitionOutsideProjectionInput(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	for repetition, want := range []string{
		`{"mode":"alternating-a","source":"schedule-repetition"}`,
		`{"mode":"alternating-b","source":"schedule-repetition"}`,
	} {
		run := runFixture(t, node, ArgvFirst, []string{"--behavior", "alternating"}, map[string]string{
			"COUNTERSHAPE_SCHEDULE_REPETITION": strconv.Itoa(repetition),
		})
		if run.err != nil || string(run.stdout) != want {
			t.Fatalf("repetition %d: stdout=%q stderr=%q err=%v", repetition, run.stdout, run.stderr, run.err)
		}
	}
}

func TestNonzeroExitRetainsBehaviorBytesAndReceipt(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	run := runFixture(t, node, ArgvFirst, []string{"--mode", "argv", "--behavior", "nonzero-exit"}, nil)
	if run.err == nil {
		t.Fatal("nonzero fixture unexpectedly exited zero")
	}
	if exit, ok := run.err.(*exec.ExitError); !ok || exit.ExitCode() != 7 {
		t.Fatalf("exit = %v, want code 7", run.err)
	}
	if string(run.stdout) != `{"mode":"argv","source":"argv"}` || run.receipt.AttemptID == "" {
		t.Fatalf("stdout=%q receipt=%#v", run.stdout, run.receipt)
	}
}

func runFixture(t *testing.T, node string, role CandidateRole, arguments []string, extraEnvironment map[string]string) fixtureRun {
	t.Helper()
	root := t.TempDir()
	candidateRoot := filepath.Join(root, "candidate")
	fixtureRoot := filepath.Join(root, "fixture")
	evidenceRoot := filepath.Join(root, "evidence")
	homeRoot := filepath.Join(root, "home")
	for _, path := range []string{candidateRoot, fixtureRoot, evidenceRoot, homeRoot} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	files, err := CandidateFiles(role)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		mode := os.FileMode(0o600)
		if file.Mode == "100755" {
			mode = 0o700
		}
		if err := os.WriteFile(filepath.Join(candidateRoot, file.Path), file.Content, mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(fixtureRoot, "config.json"), ConfigJSON("config"), 0o600); err != nil {
		t.Fatal(err)
	}
	attemptID := "attempt:test-" + string(role)
	environment := []string{
		"HOME=" + homeRoot,
		"COUNTERSHAPE_ATTEMPT_ID=" + attemptID,
		"COUNTERSHAPE_EVIDENCE_ROOT=" + evidenceRoot,
		"COUNTERSHAPE_FIXTURE_ROOT=" + fixtureRoot,
	}
	for name, value := range extraEnvironment {
		environment = append(environment, name+"="+value)
	}
	command := exec.Command(node, append([]string{Entrypoint}, arguments...)...)
	command.Dir = candidateRoot
	command.Env = environment
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	receiptBytes, readErr := os.ReadFile(filepath.Join(evidenceRoot, InvocationReceiptFilename))
	if readErr != nil {
		t.Fatalf("read invocation receipt: %v; stderr=%q", readErr, stderr.Bytes())
	}
	var receipt invocationReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatalf("decode invocation receipt: %v", err)
	}
	if receipt.AttemptID != attemptID {
		t.Fatalf("receipt attempt = %q, want %q", receipt.AttemptID, attemptID)
	}
	return fixtureRun{stdout: stdout.Bytes(), stderr: stderr.Bytes(), receipt: receipt, err: runErr}
}
