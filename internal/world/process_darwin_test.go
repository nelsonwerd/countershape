//go:build darwin && cgo

package world

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
)

type fixtureReport struct {
	EnvironmentKeys      []string          `json:"environment_keys"`
	EnvironmentValues    map[string]string `json:"environment_values"`
	MarkerPresentAtStart bool              `json:"marker_present_at_start"`
	PID                  int               `json:"pid"`
	ProcessGroupID       int               `json:"process_group_id"`
}

type fixtureEscapeIdentity struct {
	Kind            string `json:"kind"`
	PID             int    `json:"pid"`
	ProcessGroupID  int    `json:"process_group_id"`
	OriginalGroupID int    `json:"original_process_group_id"`
}

const fixtureEscapeProtocol = "countershape/process-escape/v1"

var errFixtureEscapeRecordPublishing = errors.New("coordinated escape record publication is incomplete")

type fixtureEscapePreparedRecord struct {
	Protocol        string `json:"protocol"`
	Phase           string `json:"phase"`
	AttemptID       string `json:"attempt_id"`
	Kind            string `json:"kind"`
	PID             int    `json:"pid"`
	OriginalGroupID int    `json:"original_process_group_id"`
}

type fixtureEscapeAuthorizationRecord struct {
	Protocol        string `json:"protocol"`
	Phase           string `json:"phase"`
	AttemptID       string `json:"attempt_id"`
	Kind            string `json:"kind"`
	PID             int    `json:"pid"`
	OriginalGroupID int    `json:"original_process_group_id"`
	PreparedSHA256  string `json:"prepared_sha256"`
}

type fixtureEscapeReadyRecord struct {
	Protocol            string `json:"protocol"`
	Phase               string `json:"phase"`
	AttemptID           string `json:"attempt_id"`
	Kind                string `json:"kind"`
	PID                 int    `json:"pid"`
	OriginalGroupID     int    `json:"original_process_group_id"`
	PreparedSHA256      string `json:"prepared_sha256"`
	AuthorizationSHA256 string `json:"authorization_sha256"`
	ProcessGroupID      int    `json:"process_group_id"`
	SessionID           int    `json:"session_id"`
}

type fixtureEscapeReleaseRecord struct {
	Protocol            string `json:"protocol"`
	Phase               string `json:"phase"`
	AttemptID           string `json:"attempt_id"`
	Kind                string `json:"kind"`
	PID                 int    `json:"pid"`
	OriginalGroupID     int    `json:"original_process_group_id"`
	PreparedSHA256      string `json:"prepared_sha256"`
	AuthorizationSHA256 string `json:"authorization_sha256"`
	ReadySHA256         string `json:"ready_sha256"`
}

type fixtureEscapeReleasedRecord struct {
	Protocol            string `json:"protocol"`
	Phase               string `json:"phase"`
	AttemptID           string `json:"attempt_id"`
	Kind                string `json:"kind"`
	PID                 int    `json:"pid"`
	OriginalGroupID     int    `json:"original_process_group_id"`
	PreparedSHA256      string `json:"prepared_sha256"`
	AuthorizationSHA256 string `json:"authorization_sha256"`
	ReadySHA256         string `json:"ready_sha256"`
	ReleaseSHA256       string `json:"release_sha256"`
}

type fixtureEscapeCleanup struct {
	once                sync.Once
	err                 error
	stateRoot           string
	stem                string
	prepared            fixtureEscapePreparedRecord
	preparedSHA256      string
	authorizationSHA256 string
	readySHA256         string
	ready               bool
	leaseReapDeadline   time.Time
}

func resolvedPrivateTempDir(t *testing.T) string {
	t.Helper()
	raw := t.TempDir()
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(resolved, 0o700); err != nil {
		t.Fatal(err)
	}
	return resolved
}

func moduleRootForTest(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime did not disclose the test source path")
	}
	root, err := filepath.EvalSymlinks(filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

var fixtureBuildMu sync.Mutex

func buildProcessFixture(t *testing.T) string {
	t.Helper()
	// Serializing fixture builds keeps focused mutation tests from racing over
	// Go's shared cache while still producing a fresh executable capability.
	fixtureBuildMu.Lock()
	defer fixtureBuildMu.Unlock()
	root := moduleRootForTest(t)
	outputRoot := resolvedPrivateTempDir(t)
	output := filepath.Join(outputRoot, "countershape-process-fixture")
	goExecutable := filepath.Join(runtime.GOROOT(), "bin", "go")
	cacheRoot := resolvedPrivateTempDir(t)
	for _, name := range []string{"home", "tmp", "gocache", "gomodcache", "gopath"} {
		if err := os.Mkdir(filepath.Join(cacheRoot, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command(goExecutable, "build", "-trimpath", "-o", output, "./testkit/processfixture")
	command.Dir = root
	command.Env = []string{
		"HOME=" + filepath.Join(cacheRoot, "home"),
		"TMPDIR=" + filepath.Join(cacheRoot, "tmp"),
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
		"PATH=" + filepath.Dir(goExecutable) + ":/usr/bin:/bin",
		"GOENV=off",
		"GOWORK=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOFLAGS=",
		"GOCACHE=" + filepath.Join(cacheRoot, "gocache"),
		"GOMODCACHE=" + filepath.Join(cacheRoot, "gomodcache"),
		"GOPATH=" + filepath.Join(cacheRoot, "gopath"),
		"CGO_ENABLED=1",
	}
	combined, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build process fixture: %v\n%s", err, combined)
	}
	resolved, err := filepath.EvalSymlinks(output)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func admittedFixtureTool(t *testing.T, executable string) resolvedTool {
	t.Helper()
	registry, err := NewToolRegistry(context.Background(), resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: executable, VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry.byName["fixture"]
}

func processRoots(t *testing.T) Roots {
	t.Helper()
	root := resolvedPrivateTempDir(t)
	result := Roots{attempt: root}
	for name, target := range map[string]*string{
		"candidate":  &result.candidateParent,
		"home":       &result.home,
		"tmp":        &result.temporary,
		"xdg-config": &result.xdgConfig,
		"xdg-cache":  &result.xdgCache,
		"xdg-data":   &result.xdgData,
		"xdg-state":  &result.xdgState,
		"state":      &result.state,
		"evidence":   &result.evidence,
	} {
		path := filepath.Join(root, name)
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		*target = path
	}
	result.marker = filepath.Join(result.evidence, markerFilename)
	if err := os.WriteFile(result.marker, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return result
}

func runFixtureProcess(
	t *testing.T,
	ctx context.Context,
	executable string,
	mode string,
	extra []string,
	stdoutLimit int64,
	stderrLimit int64,
	probe time.Duration,
	teardown time.Duration,
) (physicalProcessResult, Roots) {
	return runFixtureProcessWithStdin(
		t, ctx, executable, mode, extra, processStdin{}, stdoutLimit, stderrLimit, probe, teardown,
	)
}

func runFixtureProcessWithStdin(
	t *testing.T,
	ctx context.Context,
	executable string,
	mode string,
	extra []string,
	stdin processStdin,
	stdoutLimit int64,
	stderrLimit int64,
	probe time.Duration,
	teardown time.Duration,
) (physicalProcessResult, Roots) {
	return runFixtureProcessWithStdinAndGroupOwnedBarrier(
		t, ctx, executable, mode, extra, stdin, stdoutLimit, stderrLimit, probe, teardown, nil,
	)
}

func runFixtureProcessWithGroupOwnedBarrier(
	t *testing.T,
	ctx context.Context,
	executable string,
	mode string,
	extra []string,
	stdoutLimit int64,
	stderrLimit int64,
	probe time.Duration,
	teardown time.Duration,
	onGroupOwned func(Roots) error,
) (physicalProcessResult, Roots) {
	return runFixtureProcessWithStdinAndGroupOwnedBarrier(
		t, ctx, executable, mode, extra, processStdin{}, stdoutLimit, stderrLimit,
		probe, teardown, onGroupOwned,
	)
}

func runFixtureProcessWithStdinAndGroupOwnedBarrier(
	t *testing.T,
	ctx context.Context,
	executable string,
	mode string,
	extra []string,
	stdin processStdin,
	stdoutLimit int64,
	stderrLimit int64,
	probe time.Duration,
	teardown time.Duration,
	onGroupOwned func(Roots) error,
) (physicalProcessResult, Roots) {
	t.Helper()
	roots := processRoots(t)
	logicalArgv := []string{"fixture", "--mode", mode}
	logicalArgv = append(logicalArgv, extra...)
	var barrier func() error
	if onGroupOwned != nil {
		barrier = func() error {
			return onGroupOwned(roots)
		}
	}
	result := runPlatformProcess(ctx, processRequest{
		tool:              admittedFixtureTool(t, executable),
		logicalArgv:       logicalArgv,
		environment:       buildEnvironment([]domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, roots, "attempt:test"),
		stdin:             stdin,
		cwd:               roots.candidateParent,
		stdoutLimit:       stdoutLimit,
		stderrLimit:       stderrLimit,
		executionBudgetMS: probe.Milliseconds(),
		teardownBudgetMS:  teardown.Milliseconds(),
		onGroupOwned:      barrier,
	})
	return result, roots
}

func TestPreSpawnRefusalDoesNotEnterPhysicalExecutionMutationGuard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runPlatformProcess(ctx, processRequest{
		stdin:       processStdin{presence: processStdinAbsent},
		stdoutLimit: 17, stderrLimit: 29, executionBudgetMS: 100, teardownBudgetMS: 100,
	})
	if result.physicalExecutionEntered || result.spawnAttempted || result.started ||
		result.stdoutCaptureLimit != 0 || result.stderrCaptureLimit != 0 ||
		result.stdinPipeAllocated || result.stdinWriterStarted || result.stdinHandoffAttempted ||
		result.stdinWritten != 0 || result.primary != domain.ControlCancelled ||
		result.diagnosticCode != "CONTEXT_CANCELLED_BEFORE_SPAWN" {
		t.Fatalf("pre-spawn cancellation forged physical execution facts: %+v", result)
	}
}

func TestPhysicalStartFailureRecordsAttemptedButUnstartedExecutionMutationGuard(t *testing.T) {
	executable := buildProcessFixture(t)
	roots := processRoots(t)
	stdin := processStdin{presence: processStdinPresent}
	result := runPlatformProcess(context.Background(), processRequest{
		tool: admittedFixtureTool(t, executable), logicalArgv: []string{"fixture", "--mode", "report"},
		environment: buildEnvironment([]domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, roots, "attempt:start-failure"),
		stdin:       stdin, cwd: filepath.Join(roots.candidateParent, "deliberately-missing-cwd"),
		stdoutLimit: 17, stderrLimit: 29, executionBudgetMS: 100, teardownBudgetMS: 100,
		markerBeforeSpawn: true,
	})
	wantDigest, err := digestProcessStdin(stdin)
	if err != nil {
		t.Fatal(err)
	}
	if !result.physicalExecutionEntered || !result.spawnAttempted || result.started ||
		result.primary != domain.ControlStartError || result.diagnosticCode != "SPAWN_FAILED" ||
		result.pid != 0 || result.processGroupID != 0 || result.processGroupOwned ||
		result.stdoutCaptureLimit != 17 || result.stderrCaptureLimit != 29 ||
		result.stdinPresence != processStdinPresent || result.stdinDeclared != 0 || result.stdinDigest != wantDigest ||
		!result.stdinPipeAllocated || result.stdinWriterStarted || result.stdinHandoffAttempted ||
		result.stdinWritten != 0 || result.stdinComplete || result.stdinErrorCode != "" ||
		result.directChildWaited || result.stdoutDrained || result.stderrDrained || result.finalProbeClean {
		t.Fatalf("real os/exec Start failure acquired false process or stdin facts: %+v", result)
	}
}

func TestPhysicalProcessReceiptsExactStdinAndCaptureAuthorities(t *testing.T) {
	executable := buildProcessFixture(t)
	tests := []struct {
		name  string
		stdin processStdin
		want  []byte
	}{
		{name: "absent", stdin: processStdin{presence: processStdinAbsent}},
		{name: "present-empty", stdin: processStdin{presence: processStdinPresent}, want: []byte{}},
		{name: "present-bytes", stdin: processStdin{presence: processStdinPresent, bytes: []byte("opaque stdin")}, want: []byte("opaque stdin")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := runFixtureProcessWithStdin(
				t, context.Background(), executable, "echo-stdin", nil, test.stdin,
				17, 29, time.Second, 400*time.Millisecond,
			)
			wantDigest, err := digestProcessStdin(test.stdin)
			if err != nil {
				t.Fatal(err)
			}
			if result.primary != "" || result.teardownError || result.orphanRisk ||
				!result.physicalExecutionEntered || !result.started {
				t.Fatalf("physical stdin fixture did not complete cleanly: %+v", result)
			}
			if result.stdoutCaptureLimit != 17 || result.stderrCaptureLimit != 29 {
				t.Fatalf("physical captures report limits %d/%d, want 17/29", result.stdoutCaptureLimit, result.stderrCaptureLimit)
			}
			if result.stdinPresence != test.stdin.presence || result.stdinDeclared != int64(len(test.stdin.bytes)) ||
				result.stdinDigest != wantDigest {
				t.Fatalf("physical stdin identity differs: %+v", result)
			}
			if !bytes.Equal(result.stdout, test.want) || result.stdoutObserved != int64(len(test.want)) ||
				result.stdoutOverflow || len(result.stderr) != 0 || result.stderrObserved != 0 || result.stderrOverflow {
				t.Fatalf("physical stdin was not delivered exactly: stdout=%q result=%+v", result.stdout, result)
			}
			if test.stdin.presence == processStdinAbsent {
				if result.stdinPipeAllocated || result.stdinWriterStarted || result.stdinHandoffAttempted ||
					result.stdinWritten != 0 || !result.stdinComplete || result.stdinErrorCode != "" {
					t.Fatalf("absent stdin acquired a physical pipe edge: %+v", result)
				}
				return
			}
			if !result.stdinPipeAllocated || !result.stdinWriterStarted || !result.stdinHandoffAttempted ||
				result.stdinWritten != int64(len(test.stdin.bytes)) || !result.stdinComplete || result.stdinErrorCode != "" {
				t.Fatalf("present stdin lacks exact physical delivery evidence: %+v", result)
			}
		})
	}
}

// These two top-level guards are intentionally subtest-free. The U3 mutation
// runner executes exact named tests and treats any unexpected descendant event
// as an infrastructure failure, so each physical stdin authority has one
// independently classifiable kill test.
func TestPhysicalPresentEmptyStdinAuthorityMutationGuard(t *testing.T) {
	assertPhysicalPresentStdinAuthority(t, []byte{})
}

func TestPhysicalPresentBytesStdinAuthorityMutationGuard(t *testing.T) {
	assertPhysicalPresentStdinAuthority(t, []byte("opaque stdin"))
}

func assertPhysicalPresentStdinAuthority(t *testing.T, input []byte) {
	t.Helper()
	executable := buildProcessFixture(t)
	stdin := processStdin{presence: processStdinPresent, bytes: append([]byte(nil), input...)}
	result, _ := runFixtureProcessWithStdin(
		t, context.Background(), executable, "echo-stdin", nil, stdin,
		17, 29, time.Second, 400*time.Millisecond,
	)
	wantDigest, err := digestProcessStdin(stdin)
	if err != nil {
		t.Fatal(err)
	}
	if result.primary != "" || result.teardownError || result.orphanRisk ||
		!result.physicalExecutionEntered || !result.started ||
		result.stdoutCaptureLimit != 17 || result.stderrCaptureLimit != 29 ||
		result.stdinPresence != processStdinPresent || result.stdinDeclared != int64(len(input)) ||
		result.stdinDigest != wantDigest || !result.stdinPipeAllocated || !result.stdinWriterStarted ||
		!result.stdinHandoffAttempted || result.stdinWritten != int64(len(input)) ||
		!result.stdinComplete || result.stdinErrorCode != "" || !bytes.Equal(result.stdout, input) ||
		result.stdoutObserved != int64(len(input)) || result.stdoutOverflow ||
		len(result.stderr) != 0 || result.stderrObserved != 0 || result.stderrOverflow {
		t.Fatalf("physical present stdin authority changed for %d bytes: %+v", len(input), result)
	}
}

func TestDirectProcessUsesSparseEnvironmentAndMeasuredProcessGroup(t *testing.T) {
	executable := buildProcessFixture(t)
	t.Setenv("COUNTERSHAPE_AMBIENT_SENTINEL", "must-not-cross")
	result, roots := runFixtureProcess(
		t, context.Background(), executable, "report", nil, 1<<16, 1<<16, time.Second, 400*time.Millisecond,
	)
	if result.primary != "" || result.teardownError || result.orphanRisk {
		t.Fatalf("unexpected process controls: primary=%s teardown=%v orphan=%v error=%s", result.primary, result.teardownError, result.orphanRisk, result.waitError)
	}
	if !result.started || !result.processGroupOwned || result.pid <= 0 || result.processGroupID != result.pid {
		t.Fatalf("process-group measurement is incomplete: %+v", result)
	}
	if !result.directChildWaited || !result.stdoutDrained || !result.stderrDrained || !result.finalProbeClean {
		t.Fatalf("terminal lifecycle is incomplete: %+v", result)
	}
	var report fixtureReport
	if err := json.Unmarshal(result.stdout, &report); err != nil {
		t.Fatalf("decode fixture report: %v: %q", err, result.stdout)
	}
	if !report.MarkerPresentAtStart || report.PID != result.pid || report.ProcessGroupID != result.pid {
		t.Fatalf("fixture did not observe the admitted marker/group: %+v", report)
	}
	wantKeys := []string{
		"COUNTERSHAPE_ATTEMPT_ID", "COUNTERSHAPE_EVIDENCE_ROOT", "COUNTERSHAPE_STATE_ROOT",
		"HOME", "LANG", "TMPDIR", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
	}
	if strings.Join(report.EnvironmentKeys, "\x00") != strings.Join(wantKeys, "\x00") {
		t.Fatalf("candidate environment keys = %q, want %q", report.EnvironmentKeys, wantKeys)
	}
	if _, leaked := report.EnvironmentValues["COUNTERSHAPE_AMBIENT_SENTINEL"]; leaked {
		t.Fatal("ambient test-process environment crossed the world boundary")
	}
	if report.EnvironmentValues["HOME"] != roots.home || report.EnvironmentValues["TMPDIR"] != roots.temporary ||
		report.EnvironmentValues["XDG_STATE_HOME"] != roots.xdgState {
		t.Fatalf("fresh roots were not projected exactly: %+v", report.EnvironmentValues)
	}
}

func TestWorldAdapterPreservesIndependentExactCaptureCaps(t *testing.T) {
	executable := buildProcessFixture(t)
	t.Run("exact-boundary", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "emit",
			[]string{"--stdout-bytes", "17", "--stderr-bytes", "9"}, 17, 9, time.Second, 400*time.Millisecond)
		if result.primary != "" || result.stdoutOverflow || result.stderrOverflow ||
			len(result.stdout) != 17 || len(result.stderr) != 9 || result.stdoutObserved != 17 || result.stderrObserved != 9 {
			t.Fatalf("exact channel boundaries were not preserved: %+v", result)
		}
	})
	t.Run("stdout-overflow", func(t *testing.T) {
		// Complete the sibling channel before triggering the overflow. The
		// OUTPUT_LIMIT contract tears the process group down immediately, so a
		// sequential stdout-first fixture cannot promise that its later stderr
		// write will be scheduled before SIGTERM.
		result, _ := runFixtureProcess(t, context.Background(), executable, "emit-stderr-first",
			[]string{"--stdout-bytes", "18", "--stderr-bytes", "9"}, 17, 9, time.Second, 400*time.Millisecond)
		if result.primary != domain.ControlOutputLimit || !result.stdoutOverflow || result.stderrOverflow ||
			len(result.stdout) != 17 || result.stdoutObserved != 18 || result.stderrObserved != 9 {
			t.Fatalf("stdout overflow was not independently controlled: %+v", result)
		}
	})
	t.Run("stderr-overflow", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "emit",
			[]string{"--stdout-bytes", "4", "--stderr-bytes", "10"}, 4, 9, time.Second, 400*time.Millisecond)
		if result.primary != domain.ControlOutputLimit || result.stdoutOverflow || !result.stderrOverflow ||
			result.stdoutObserved != 4 || len(result.stderr) != 9 || result.stderrObserved != 10 {
			t.Fatalf("stderr overflow was not independently controlled: %+v", result)
		}
	})
}

func TestWorldAdapterPreservesSimultaneousChannelOverflowFacts(t *testing.T) {
	executable := buildProcessFixture(t)
	const limit = int64(4096)
	result, _ := runFixtureProcess(
		t, context.Background(), executable, "emit-both-ignore-term",
		[]string{"--stdout-bytes", "65536", "--stderr-bytes", "65536"},
		limit, limit, time.Second, 500*time.Millisecond,
	)
	registerEmergencyGroupCleanup(t, result)
	if result.primary != domain.ControlOutputLimit || !result.stdoutOverflow || !result.stderrOverflow ||
		len(result.stdout) != int(limit) || len(result.stderr) != int(limit) ||
		result.stdoutObserved <= limit || result.stderrObserved <= limit ||
		!result.termSent || !result.killSent || !result.directChildWaited || !result.finalProbeClean {
		t.Fatalf("simultaneous channel overflow lost independent facts: %+v", result)
	}
}

func TestProcessLifecycleControlsAndCleansDescendants(t *testing.T) {
	executable := buildProcessFixture(t)
	t.Run("nonzero-exit-is-behavior", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "exit", []string{"--exit-code", "23"}, 1024, 1024, time.Second, 400*time.Millisecond)
		if result.primary != "" || result.exitCode != 23 || result.exitSignal != "" || !result.finalProbeClean {
			t.Fatalf("nonzero exit was misclassified: %+v", result)
		}
	})
	t.Run("signal-is-behavior", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "signal-self", nil, 1024, 1024, time.Second, 400*time.Millisecond)
		if result.primary != "" || result.exitSignal == "" || !result.finalProbeClean {
			t.Fatalf("signal exit was misclassified: %+v", result)
		}
	})
	t.Run("timeout-terminates-cooperative-child", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "hang", nil, 1024, 1024, 80*time.Millisecond, 300*time.Millisecond)
		if result.primary != domain.ControlTimeout || !result.termSent || !result.directChildWaited || !result.finalProbeClean || result.orphanRisk {
			t.Fatalf("cooperative timeout lifecycle is incomplete: %+v", result)
		}
	})
	t.Run("timeout-escalates-resistant-child", func(t *testing.T) {
		result, _ := runFixtureProcess(t, context.Background(), executable, "ignore-term", nil, 1024, 1024, 80*time.Millisecond, 300*time.Millisecond)
		if result.primary != domain.ControlTimeout || !result.termSent || !result.killSent || !result.directChildWaited || !result.finalProbeClean {
			t.Fatalf("resistant timeout lifecycle is incomplete: %+v", result)
		}
	})
	t.Run("descendant-group", func(t *testing.T) {
		result, roots := runFixtureProcess(t, context.Background(), executable, "descendant", nil, 1024, 1024, 100*time.Millisecond, 400*time.Millisecond)
		if result.primary != domain.ControlTimeout || !result.termSent || !result.finalProbeClean || result.orphanRisk || !result.directChildWaited {
			t.Fatalf("descendant group was not proven gone: %+v", result)
		}
		if _, err := os.Stat(filepath.Join(roots.state, "grandchild.pid")); err != nil {
			t.Fatalf("descendant fixture never reached the grandchild: %v", err)
		}
	})
	t.Run("parent-exit-still-cleans-group", func(t *testing.T) {
		result, roots := runFixtureProcess(t, context.Background(), executable, "parent-exits", nil, 1024, 1024, time.Second, 400*time.Millisecond)
		if result.primary != "" || !result.directChildWaited || !result.finalProbeClean || result.orphanRisk {
			t.Fatalf("direct-child exit hid a surviving process group: %+v", result)
		}
		if _, err := os.Stat(filepath.Join(roots.state, "grandchild.pid")); err != nil {
			t.Fatalf("parent-exit fixture never spawned the grandchild: %v", err)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		go func() {
			time.Sleep(30 * time.Millisecond)
			cancel()
		}()
		result, _ := runFixtureProcess(t, ctx, executable, "hang", nil, 1024, 1024, time.Second, 300*time.Millisecond)
		if result.primary != domain.ControlCancelled || !result.termSent || !result.finalProbeClean {
			t.Fatalf("cancellation lifecycle is incomplete: %+v", result)
		}
	})
}

func TestProcessGroupAndSessionEscapesRemainExplicitExclusions(t *testing.T) {
	executable := buildProcessFixture(t)
	for _, fixture := range []struct {
		name, mode, stem, kind string
	}{
		{name: "setsid", mode: "setsid-escape-coordinated", stem: "escaped-coordinated", kind: "setsid"},
		{name: "setpgid", mode: "setpgid-escape-coordinated", stem: "escaped-group-coordinated", kind: "setpgid"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var (
				setupErr error
				prepared fixtureEscapePreparedRecord
				ready    fixtureEscapeReadyRecord
				cleanup  *fixtureEscapeCleanup
			)
			result, _ := runFixtureProcessWithGroupOwnedBarrier(
				t, context.Background(), executable, fixture.mode, nil,
				1024, 1024, 100*time.Millisecond, 300*time.Millisecond,
				func(roots Roots) error {
					setupDeadline := time.Now().Add(10 * time.Second)
					preparedDeadline := time.Now().Add(5 * time.Second)
					preparedPath := filepath.Join(roots.state, fixture.stem+".prepared.json")
					preparedBytes, err := waitForFixtureEscapeRecord(preparedPath, preparedDeadline, &prepared)
					if err != nil {
						setupErr = err
						return err
					}
					if prepared.Protocol != fixtureEscapeProtocol || prepared.Phase != "prepared" ||
						prepared.AttemptID != "attempt:test" || prepared.Kind != fixture.kind ||
						prepared.PID <= 0 || prepared.OriginalGroupID <= 0 ||
						prepared.PID == prepared.OriginalGroupID {
						setupErr = fmt.Errorf("prepared record is incomplete: %+v", prepared)
						return setupErr
					}
					cleanup = &fixtureEscapeCleanup{
						stateRoot: roots.state, stem: fixture.stem, prepared: prepared,
						preparedSHA256:    fixtureEscapeRecordDigest(preparedBytes),
						leaseReapDeadline: time.Now().Add(17 * time.Second),
					}
					t.Cleanup(func() {
						if err := cleanup.close(); err != nil {
							t.Errorf("coordinated escape cleanup failed: %v", err)
						}
					})
					preAuthorizationGroup, err := syscall.Getpgid(prepared.PID)
					if err != nil || preAuthorizationGroup != prepared.OriginalGroupID {
						setupErr = fmt.Errorf(
							"prepared process left its original group before authorization: group=%d original=%d err=%v",
							preAuthorizationGroup, prepared.OriginalGroupID, err,
						)
						return setupErr
					}
					authorization := fixtureEscapeAuthorizationRecord{
						Protocol: fixtureEscapeProtocol, Phase: "authorized",
						AttemptID: prepared.AttemptID, Kind: prepared.Kind,
						PID: prepared.PID, OriginalGroupID: prepared.OriginalGroupID,
						PreparedSHA256: cleanup.preparedSHA256,
					}
					authorizationBytes, err := publishFixtureEscapeRecord(
						roots.state, fixture.stem+".authorization.json", authorization,
					)
					if err != nil {
						setupErr = err
						return err
					}
					cleanup.authorizationSHA256 = fixtureEscapeRecordDigest(authorizationBytes)
					readyDeadline := time.Now().Add(5 * time.Second)
					if readyDeadline.After(setupDeadline) {
						readyDeadline = setupDeadline
					}
					readyPath := filepath.Join(roots.state, fixture.stem+".ready.json")
					readyBytes, err := waitForFixtureEscapeRecord(readyPath, readyDeadline, &ready)
					if err != nil {
						setupErr = err
						return err
					}
					if ready.Protocol != fixtureEscapeProtocol || ready.Phase != "ready" ||
						ready.AttemptID != prepared.AttemptID || ready.Kind != prepared.Kind ||
						ready.PID != prepared.PID || ready.OriginalGroupID != prepared.OriginalGroupID ||
						ready.PreparedSHA256 != cleanup.preparedSHA256 ||
						ready.AuthorizationSHA256 != cleanup.authorizationSHA256 ||
						ready.ProcessGroupID != prepared.PID ||
						(fixture.kind == "setsid" && ready.SessionID != prepared.PID) ||
						(fixture.kind == "setpgid" && ready.SessionID != 0) {
						setupErr = fmt.Errorf("ready record is incomplete: prepared=%+v ready=%+v", prepared, ready)
						return setupErr
					}
					currentGroup, err := syscall.Getpgid(prepared.PID)
					if err != nil || currentGroup != prepared.PID {
						setupErr = fmt.Errorf(
							"ready process was not independently observed in its escaped group: group=%d err=%v",
							currentGroup, err,
						)
						return setupErr
					}
					if fixture.kind == "setsid" {
						sessionID, err := syscall.Getsid(prepared.PID)
						if err != nil || sessionID != prepared.PID {
							setupErr = fmt.Errorf(
								"ready process was not independently observed in its escaped session: session=%d err=%v",
								sessionID, err,
							)
							return setupErr
						}
					}
					cleanup.readySHA256 = fixtureEscapeRecordDigest(readyBytes)
					cleanup.ready = true
					return nil
				},
			)
			if setupErr != nil {
				t.Fatalf("coordinated escape setup failed: %v result=%+v", setupErr, result)
			}
			if cleanup == nil || !cleanup.ready {
				t.Fatalf("coordinated escape setup returned without cleanup authority: %+v", result)
			}
			stdout := []byte("countershape-coordinated-escape-" + fixture.kind + "-stdout\n")
			stderr := []byte("countershape-coordinated-escape-" + fixture.kind + "-stderr\n")
			if !result.physicalExecutionEntered || !result.spawnAttempted || !result.started ||
				result.pid <= 0 || !result.processGroupOwned || result.processGroupID != result.pid ||
				result.primary != domain.ControlTimeout || result.preTermProbe != preTermProbePresent ||
				!result.termSent || !result.directChildWaited || !result.finalProbeClean ||
				result.finalProbeError != "" || result.processGroupID != prepared.OriginalGroupID ||
				result.processGroupID == prepared.PID {
				t.Fatalf("original process-group result is malformed: %+v", result)
			}
			if !bytes.Equal(result.stdout, stdout) || !bytes.Equal(result.stderr, stderr) ||
				result.stdoutObserved != int64(len(stdout)) || result.stderrObserved != int64(len(stderr)) ||
				result.stdoutOverflow || result.stderrOverflow ||
				!result.teardownError || !result.orphanRisk || result.stdoutDrained || result.stderrDrained ||
				result.diagnosticCode != "PIPE_DRAIN_DEADLINE" {
				t.Fatalf("escaped inherited pipes should make bounded drains visibly incomplete and orphan-uncertain: %+v", result)
			}
			if processEscapeExclusion != "PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM" {
				t.Fatalf("cleanup boundary changed: %q", processEscapeExclusion)
			}
			if processGroupReuseExclusion != "PRE_TERM_PROBE_AND_SIGNAL_ARE_NON_ATOMIC_PGID_REUSE_EXCLUDED_FROM_CLEANUP_CLAIM" {
				t.Fatalf("process-group signal boundary changed: %q", processGroupReuseExclusion)
			}
			currentGroup, err := syscall.Getpgid(prepared.PID)
			if err != nil || currentGroup != prepared.PID || ready.ProcessGroupID != currentGroup {
				t.Fatalf(
					"escaped child was not demonstrably outside the original group after teardown: group=%d ready=%+v err=%v",
					currentGroup, ready, err,
				)
			}
			if fixture.kind == "setsid" {
				sessionID, err := syscall.Getsid(prepared.PID)
				if err != nil || sessionID != prepared.PID || ready.SessionID != sessionID {
					t.Fatalf(
						"escaped child was not demonstrably outside the original session after teardown: session=%d ready=%+v err=%v",
						sessionID, ready, err,
					)
				}
			}
			if err := cleanup.close(); err != nil {
				t.Fatalf("coordinated escape release failed: %v", err)
			}
		})
	}
}

func publishFixtureEscapeRecord(root, name string, value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	finalPath := filepath.Join(root, name)
	temporaryPath := filepath.Join(root, "."+name+"."+strconv.Itoa(os.Getpid())+".tmp")
	handle, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	removeTemporary := func() {
		_ = os.Remove(temporaryPath)
	}
	if _, err := handle.Write(raw); err != nil {
		_ = handle.Close()
		removeTemporary()
		return nil, err
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		removeTemporary()
		return nil, err
	}
	if err := handle.Close(); err != nil {
		removeTemporary()
		return nil, err
	}
	if err := os.Link(temporaryPath, finalPath); err != nil {
		removeTemporary()
		return nil, err
	}
	if err := os.Remove(temporaryPath); err != nil {
		return nil, err
	}
	directory, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	if err := errors.Join(directory.Sync(), directory.Close()); err != nil {
		return nil, err
	}
	info, err := os.Lstat(finalPath)
	if err != nil {
		return nil, err
	}
	stat, statOK := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 ||
		!statOK || stat.Nlink != 1 {
		return nil, fmt.Errorf("published coordinated escape record has invalid authority at %s", name)
	}
	published, err := os.ReadFile(finalPath)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(raw, published) {
		return nil, fmt.Errorf("published coordinated escape record drifted at %s", name)
	}
	return raw, nil
}

func waitForFixtureEscapeRecord(path string, deadline time.Time, value any) ([]byte, error) {
	var lastErr error
	for {
		bytes, err := readFixtureEscapeRecord(path, value)
		if err == nil {
			return bytes, nil
		}
		lastErr = err
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errFixtureEscapeRecordPublishing) {
			return nil, err
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf(
				"timed out waiting for coordinated escape record %s: %w",
				filepath.Base(path), lastErr,
			)
		}
		time.Sleep(time.Millisecond)
	}
}

func readFixtureEscapeRecord(path string, value any) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	stat, statOK := info.Sys().(*syscall.Stat_t)
	if info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o600 &&
		statOK && stat.Nlink == 2 {
		return nil, errFixtureEscapeRecordPublishing
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 ||
		!statOK || stat.Nlink != 1 {
		return nil, fmt.Errorf("coordinated escape record has invalid authority at %s", filepath.Base(path))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("coordinated escape record size is invalid at %s", filepath.Base(path))
	}
	if err := json.Unmarshal(raw, value); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(raw, canonical) {
		return nil, fmt.Errorf("coordinated escape record is not canonical at %s", filepath.Base(path))
	}
	return raw, nil
}

func fixtureEscapeRecordDigest(bytes []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(bytes))
}

func (cleanup *fixtureEscapeCleanup) close() error {
	if cleanup == nil {
		return errors.New("coordinated escape cleanup authority is nil")
	}
	cleanup.once.Do(func() {
		cleanup.err = cleanup.closeOnce()
	})
	return cleanup.err
}

func (cleanup *fixtureEscapeCleanup) closeOnce() error {
	var releaseErr, acknowledgementErr error
	if cleanup.ready {
		release := fixtureEscapeReleaseRecord{
			Protocol: fixtureEscapeProtocol, Phase: "release",
			AttemptID: cleanup.prepared.AttemptID, Kind: cleanup.prepared.Kind,
			PID: cleanup.prepared.PID, OriginalGroupID: cleanup.prepared.OriginalGroupID,
			PreparedSHA256:      cleanup.preparedSHA256,
			AuthorizationSHA256: cleanup.authorizationSHA256,
			ReadySHA256:         cleanup.readySHA256,
		}
		releaseBytes, err := publishFixtureEscapeRecord(
			cleanup.stateRoot, cleanup.stem+".release.json", release,
		)
		if err != nil {
			releaseErr = fmt.Errorf("publish release: %w", err)
		} else {
			var released fixtureEscapeReleasedRecord
			_, err := waitForFixtureEscapeRecord(
				filepath.Join(cleanup.stateRoot, cleanup.stem+".released.json"),
				cleanup.leaseReapDeadline,
				&released,
			)
			if err != nil {
				acknowledgementErr = fmt.Errorf("await release acknowledgement: %w", err)
			} else {
				expected := fixtureEscapeReleasedRecord{
					Protocol: fixtureEscapeProtocol, Phase: "released",
					AttemptID: cleanup.prepared.AttemptID, Kind: cleanup.prepared.Kind,
					PID: cleanup.prepared.PID, OriginalGroupID: cleanup.prepared.OriginalGroupID,
					PreparedSHA256:      cleanup.preparedSHA256,
					AuthorizationSHA256: cleanup.authorizationSHA256,
					ReadySHA256:         cleanup.readySHA256,
					ReleaseSHA256:       fixtureEscapeRecordDigest(releaseBytes),
				}
				if released != expected {
					acknowledgementErr = fmt.Errorf(
						"release acknowledgement did not bind the release record: got=%+v want=%+v",
						released, expected,
					)
				}
			}
		}
	}
	var absenceErr error
	if err := waitForFixturePIDAbsence(cleanup.prepared.PID, cleanup.leaseReapDeadline); err != nil {
		absenceErr = fmt.Errorf(
			"coordinated escape remained beyond its self-lease and reap grace without identity-safe cleanup: %w",
			err,
		)
	}
	return errors.Join(releaseErr, acknowledgementErr, absenceErr)
}

func waitForFixturePIDAbsence(pid int, deadline time.Time) error {
	var lastErr error
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil {
			return err
		}
		lastErr = fmt.Errorf("pid %d is still present", pid)
		if !time.Now().Before(deadline) {
			return lastErr
		}
		time.Sleep(time.Millisecond)
	}
}

type scriptedGroupProbe struct {
	results     []bool
	probeErrors []error
	steps       []scriptedProbeStep
	signals     []syscall.Signal
	events      []string
	calls       int
}

type scriptedProbeStep struct {
	present bool
	err     error
}

type scriptedPreTermProbeClock struct {
	current   time.Time
	overshoot time.Duration
	waits     []time.Duration
}

func (s *scriptedPreTermProbeClock) now() time.Time { return s.current }

func (s *scriptedPreTermProbeClock) wait(duration time.Duration) {
	s.waits = append(s.waits, duration)
	s.current = s.current.Add(duration + s.overshoot)
}

type deadlineCrossingGroupProbe struct {
	clock       *scriptedPreTermProbeClock
	nextPresent bool
	calls       int
}

func (*deadlineCrossingGroupProbe) signal(int, syscall.Signal) error { return nil }

func (s *deadlineCrossingGroupProbe) probe(int) (bool, error) {
	s.calls++
	if s.calls == 1 {
		return true, syscall.EPERM
	}
	s.clock.current = s.clock.current.Add(time.Millisecond)
	return s.nextPresent, nil
}

func (s *scriptedGroupProbe) signal(_ int, signal syscall.Signal) error {
	s.signals = append(s.signals, signal)
	s.events = append(s.events, "signal")
	return nil
}

func (s *scriptedGroupProbe) probe(int) (bool, error) {
	s.calls++
	s.events = append(s.events, "probe")
	if len(s.steps) > 0 {
		step := s.steps[0]
		s.steps = s.steps[1:]
		return step.present, step.err
	}
	if len(s.probeErrors) > 0 {
		err := s.probeErrors[0]
		s.probeErrors = s.probeErrors[1:]
		if err != nil {
			return false, err
		}
	}
	if len(s.results) == 0 {
		return false, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

func TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled(t *testing.T) {
	t.Run("absent-skips-term", func(t *testing.T) {
		controller := &scriptedGroupProbe{results: []bool{false}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbeAbsent || result.termSent || len(controller.signals) != 0 ||
			result.teardownError || result.orphanRisk || strings.Join(controller.events, ",") != "probe" {
			t.Fatalf("observed-absent group was signaled or misclassified: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("non-eperm-probe-error-skips-signal-and-retains-uncertainty", func(t *testing.T) {
		controller := &scriptedGroupProbe{probeErrors: []error{syscall.EIO}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbeUncertain || result.termSent || len(controller.signals) != 0 ||
			!result.teardownError || !result.orphanRisk || result.diagnosticCode != "PRE_TERM_GROUP_PROBE_FAILED" {
			t.Fatalf("uncertain pre-TERM probe was not fail-closed: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("transient-initial-eperm-then-absent-skips-term", func(t *testing.T) {
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true, err: syscall.EPERM},
			{present: false},
		}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbeAbsent || result.termSent || len(controller.signals) != 0 ||
			result.teardownError || result.orphanRisk || strings.Join(controller.events, ",") != "probe,probe" {
			t.Fatalf("transient initial EPERM did not resolve to clean absence: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("transient-initial-eperm-then-clean-presence-allows-term", func(t *testing.T) {
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true, err: syscall.EPERM},
			{present: true},
			{present: false},
		}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbePresent || !result.termSent || result.killSent ||
			len(controller.signals) != 1 || controller.signals[0] != syscall.SIGTERM ||
			result.teardownError || result.orphanRisk ||
			strings.Join(controller.events, ",") != "probe,probe,signal,probe" {
			t.Fatalf("transient initial EPERM did not require clean presence before TERM: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("persistent-initial-eperm-remains-uncertain", func(t *testing.T) {
		steps := make([]scriptedProbeStep, 64)
		for index := range steps {
			steps[index] = scriptedProbeStep{present: true, err: syscall.EPERM}
		}
		controller := &scriptedGroupProbe{steps: steps}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(3*time.Millisecond), 20*time.Millisecond)
		if result.preTermProbe != preTermProbeUncertain || result.termSent || len(controller.signals) != 0 ||
			!result.teardownError || !result.orphanRisk || result.diagnosticCode != "PRE_TERM_GROUP_PROBE_FAILED" {
			t.Fatalf("persistent initial EPERM did not remain fail-closed: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("exact-retry-deadline-does-not-consume-a-later-clean-probe", func(t *testing.T) {
		start := time.Unix(100, 0)
		clock := &scriptedPreTermProbeClock{current: start}
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true, err: syscall.EPERM},
			{present: false},
		}}
		present, err := resolvePreTermGroupProbe(controller, 4242, start.Add(time.Second), 4*time.Millisecond, clock)
		if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 1 ||
			len(clock.waits) != 1 || clock.waits[0] != time.Millisecond {
			t.Fatalf("exact retry deadline consumed post-deadline authority: present=%t err=%v calls=%d waits=%v",
				present, err, controller.calls, clock.waits)
		}
	})

	t.Run("retry-wait-overshoot-does-not-consume-a-later-clean-probe", func(t *testing.T) {
		start := time.Unix(100, 0)
		clock := &scriptedPreTermProbeClock{current: start, overshoot: time.Nanosecond}
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true, err: syscall.EPERM},
			{present: true},
		}}
		present, err := resolvePreTermGroupProbe(controller, 4242, start.Add(time.Second), 4*time.Millisecond, clock)
		if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 1 || len(clock.waits) != 1 {
			t.Fatalf("overshot retry deadline consumed post-deadline authority: present=%t err=%v calls=%d waits=%v",
				present, err, controller.calls, clock.waits)
		}
	})

	for _, test := range []struct {
		name        string
		cleanResult bool
	}{
		{name: "absence", cleanResult: false},
		{name: "presence", cleanResult: true},
	} {
		t.Run("probe-crossing-retry-deadline-discarded-"+test.name, func(t *testing.T) {
			start := time.Unix(100, 0)
			clock := &scriptedPreTermProbeClock{current: start}
			controller := &deadlineCrossingGroupProbe{clock: clock, nextPresent: test.cleanResult}
			present, err := resolvePreTermGroupProbe(controller, 4242, start.Add(time.Second), 8*time.Millisecond, clock)
			if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 2 ||
				len(clock.waits) != 1 || clock.waits[0] != time.Millisecond {
				t.Fatalf("probe crossing retry deadline became signal authority: present=%t err=%v calls=%d waits=%v",
					present, err, controller.calls, clock.waits)
			}
		})
	}

	t.Run("outer-teardown-deadline-bounds-retry-probes", func(t *testing.T) {
		start := time.Unix(100, 0)
		clock := &scriptedPreTermProbeClock{current: start}
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true, err: syscall.EPERM},
			{present: true, err: syscall.EPERM},
			{present: false},
		}}
		present, err := resolvePreTermGroupProbe(controller, 4242, start.Add(2*time.Millisecond), time.Second, clock)
		if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 2 ||
			len(clock.waits) != 2 || clock.waits[0] != time.Millisecond || clock.waits[1] != time.Millisecond {
			t.Fatalf("outer teardown deadline failed to bound probes: present=%t err=%v calls=%d waits=%v",
				present, err, controller.calls, clock.waits)
		}
	})

	t.Run("present-probes-immediately-before-term", func(t *testing.T) {
		controller := &scriptedGroupProbe{results: []bool{true, false}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbePresent || !result.termSent || result.killSent ||
			len(controller.signals) != 1 || controller.signals[0] != syscall.SIGTERM ||
			strings.Join(controller.events, ",") != "probe,signal,probe" {
			t.Fatalf("present group teardown order changed: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("uncertain-grace-probe-skips-blind-kill", func(t *testing.T) {
		controller := &scriptedGroupProbe{
			results: []bool{true}, probeErrors: []error{nil, syscall.EPERM},
		}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if !result.termSent || result.killSent || len(controller.signals) != 1 ||
			controller.signals[0] != syscall.SIGTERM || !result.teardownError || !result.orphanRisk ||
			result.diagnosticCode != "TERM_GRACE_PROBE_FAILED" {
			t.Fatalf("uncertain grace probe triggered a blind escalation: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("transient-zombie-group-eperm-is-retried", func(t *testing.T) {
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true},
			{present: true, err: syscall.EPERM},
			{present: false},
		}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if !result.termSent || result.killSent || len(controller.signals) != 1 ||
			result.teardownError || result.orphanRisk || result.diagnosticCode != "" {
			t.Fatalf("transient zombie-only group state was made sticky: result=%+v events=%v", result, controller.events)
		}
	})

	t.Run("persistent-zombie-group-eperm-remains-uncertain", func(t *testing.T) {
		controller := &scriptedGroupProbe{steps: []scriptedProbeStep{
			{present: true},
			{present: true, err: syscall.EPERM},
		}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(-time.Millisecond), time.Second)
		if !result.termSent || result.killSent || len(controller.signals) != 1 ||
			!result.teardownError || !result.orphanRisk || result.diagnosticCode != "TERM_GRACE_PROBE_FAILED" {
			t.Fatalf("persistent zombie-only group uncertainty was erased: result=%+v events=%v", result, controller.events)
		}
	})
}

func TestFinalGroupProbeRequiresObservedAbsence(t *testing.T) {
	controller := &scriptedGroupProbe{results: []bool{true, true, false}}
	clean, err := performFinalGroupProbe(controller, 4242, time.Now().Add(100*time.Millisecond))
	if err != nil || !clean || controller.calls != 3 {
		t.Fatalf("final probe = clean %v, err %v, calls %d", clean, err, controller.calls)
	}
	stillPresent := &scriptedGroupProbe{results: []bool{true}}
	clean, err = performFinalGroupProbe(stillPresent, 4242, time.Now().Add(-time.Millisecond))
	if err != nil || clean || stillPresent.calls != 1 {
		t.Fatalf("deadline probe = clean %v, err %v, calls %d", clean, err, stillPresent.calls)
	}
}

func TestTerminalArbiterUsesFixedOwnerObservedPriority(t *testing.T) {
	if terminalArbitrationContract != "OWNER_OBSERVED_PRIORITY_OUTPUT_PROBE_TRANSPORT_CANCEL_DEADLINE_WAIT" {
		t.Fatalf("receipt arbitration contract = %q", terminalArbitrationContract)
	}
	overflowC := make(chan struct{}, 2)
	stdout := newCappedCapture(1, overflowC)
	stderr := newCappedCapture(1, overflowC)
	stdout.retain([]byte("xx"))
	deadlineC := make(chan time.Time, 1)
	deadlineC <- time.Now()
	waitC := make(chan waitResult, 1)
	waitC <- waitResult{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	decision := arbitrateTerminal(ctx, deadlineC, overflowC, stdout, stderr, waitC)
	if decision.primary != domain.ControlOutputLimit {
		t.Fatalf("all-ready priority selected %s", decision.primary)
	}

	stdout = newCappedCapture(1, make(chan struct{}, 1))
	stderr = newCappedCapture(1, make(chan struct{}, 1))
	deadlineC = make(chan time.Time, 1)
	deadlineC <- time.Now()
	waitC = make(chan waitResult, 1)
	waitC <- waitResult{}
	stdinC := make(chan stdinWriteResult, 1)
	stdinC <- stdinWriteResult{complete: false, errorCode: "WRITE_FAILED"}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	decision = arbitrateTerminalWithStdin(ctx, deadlineC, make(chan struct{}), stdout, stderr, waitC, stdinC)
	if decision.primary != domain.ControlProbeTransportError {
		t.Fatalf("stdin transport/cancel/deadline/wait priority selected %+v", decision)
	}

	deadlineC = make(chan time.Time, 1)
	waitC = make(chan waitResult, 1)
	waitC <- waitResult{}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	decision = arbitrateTerminal(ctx, deadlineC, make(chan struct{}), stdout, stderr, waitC)
	if decision.primary != domain.ControlCancelled {
		t.Fatalf("cancel/wait priority selected %s", decision.primary)
	}

	deadlineC = make(chan time.Time, 1)
	deadlineC <- time.Now()
	waitC = make(chan waitResult, 1)
	waitC <- waitResult{}
	decision = arbitrateTerminal(context.Background(), deadlineC, make(chan struct{}), stdout, stderr, waitC)
	if decision.primary != domain.ControlTimeout {
		t.Fatalf("deadline/wait priority selected %s", decision.primary)
	}

	result := physicalProcessResult{primary: domain.ControlStartError, stdoutOverflow: true}
	applyCompletedOutputControl(&result)
	if result.primary != domain.ControlStartError {
		t.Fatalf("overflow overwrote structural start failure: %s", result.primary)
	}
}

func TestOwnerRepollsDeadlineBeforeHonoringLatchedWait(t *testing.T) {
	deadlineC := make(chan time.Time, 1)
	deadlineC <- time.Now()
	stdout := newCappedCapture(1, make(chan struct{}, 1))
	stderr := newCappedCapture(1, make(chan struct{}, 1))
	waited := &waitResult{}
	decision, ready, observed := pollOwnerTerminalPriority(
		context.Background(), deadlineC, stdout, stderr, false, waited,
	)
	if !ready || !observed || decision.primary != domain.ControlTimeout || decision.waited != nil {
		t.Fatalf("latched wait bypassed next-turn deadline poll: decision=%+v ready=%v observed=%v", decision, ready, observed)
	}
}

func TestUnexpectedWaitFailureIsNotACompletedCleanupEdge(t *testing.T) {
	result := physicalProcessResult{}
	classifyWait(&result, waitResult{err: errors.New("injected wait ownership failure")})
	if result.directChildWaited || !result.teardownError || !result.orphanRisk ||
		result.diagnosticCode != "WAIT_FAILED" || result.waitError == "" {
		t.Fatalf("unexpected wait failure was receipted as complete cleanup: %+v", result)
	}
}

func TestToolRegistryNeverConsultsPATHAndPinsMeasuredBytes(t *testing.T) {
	executable := buildProcessFixture(t)
	poison := resolvedPrivateTempDir(t)
	poisonExecutable := filepath.Join(poison, "fixture")
	fixtureBytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(poisonExecutable, fixtureBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	linkRoot := resolvedPrivateTempDir(t)
	link := filepath.Join(linkRoot, "fixture-link")
	if err := os.Symlink(executable, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", poison)
	_, err = NewToolRegistry(context.Background(), resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: "fixture", VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if code, ok := RefusalCodeOf(err); !ok || code != CodeToolRejected {
		t.Fatalf("logical PATH name was not rejected: %v", err)
	}
	_, err = NewToolRegistry(context.Background(), resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: link, VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if code, ok := RefusalCodeOf(err); !ok || code != CodeToolRejected {
		t.Fatalf("symlinked executable was not rejected: %v", err)
	}
	registry, err := NewToolRegistry(context.Background(), resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: executable, VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tool := registry.byName["fixture"]
	if tool.major != 1 || tool.version != "countershape-process-fixture v1.0.0" || !tool.executableDigest.Valid() {
		t.Fatalf("tool measurement is incomplete: %+v", tool)
	}
	bytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	bytes[len(bytes)/2] ^= 0x01
	if err := os.WriteFile(executable, bytes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := tool.revalidate(); err == nil {
		t.Fatal("changed executable bytes retained admitted authority")
	}
}

func TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound(t *testing.T) {
	executable := buildProcessFixture(t)
	started := time.Now()
	registry, err := NewToolRegistry(context.Background(), resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: executable, VersionConstraint: "executed-major-only", VersionArgs: []string{"--version-fork"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 3*time.Second || registry.byName["fixture"].major != 1 {
		t.Fatalf("version probe was not bounded and measured: elapsed=%s", time.Since(started))
	}
}

func TestCappedCaptureCountsAllBytesWhileRetainingOnlyLimit(t *testing.T) {
	overflow := make(chan struct{}, 1)
	capture := newCappedCapture(7, overflow)
	capture.retain([]byte("12345"))
	capture.retain([]byte("678901234567890"))
	bytes, observed, over, err := capture.snapshot()
	if err != nil || string(bytes) != "1234567" || observed != 20 || !over {
		t.Fatalf("capture = %q observed=%d overflow=%v err=%v", bytes, observed, over, err)
	}
}
