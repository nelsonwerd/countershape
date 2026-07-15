//go:build darwin && cgo

package world

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	t.Helper()
	roots := processRoots(t)
	logicalArgv := []string{"fixture", "--mode", mode}
	logicalArgv = append(logicalArgv, extra...)
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

func TestStdoutAndStderrHaveIndependentExactCaps(t *testing.T) {
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
		result, _ := runFixtureProcess(t, context.Background(), executable, "emit",
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

func TestSimultaneousChannelOverflowRetainsIndependentFacts(t *testing.T) {
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
		name, mode, pidFile, identityFile, kind string
	}{
		{name: "setsid", mode: "setsid-escape", pidFile: "escaped.pid", identityFile: "escaped.identity.json", kind: "setsid"},
		{name: "setpgid", mode: "setpgid-escape", pidFile: "escaped-group.pid", identityFile: "escaped-group.identity.json", kind: "setpgid"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			result, roots := runFixtureProcess(t, context.Background(), executable, fixture.mode, nil, 1024, 1024, 100*time.Millisecond, 300*time.Millisecond)
			pidBytes, err := os.ReadFile(filepath.Join(roots.state, fixture.pidFile))
			if err != nil {
				t.Fatalf("escape fixture did not publish pid: %v", err)
			}
			escapedPID, err := strconv.Atoi(string(pidBytes))
			if err != nil || escapedPID <= 0 {
				t.Fatalf("invalid escaped pid %q", pidBytes)
			}
			identityBytes, err := os.ReadFile(filepath.Join(roots.state, fixture.identityFile))
			var identity fixtureEscapeIdentity
			if err != nil || json.Unmarshal(identityBytes, &identity) != nil || identity.Kind != fixture.kind ||
				identity.PID != escapedPID || identity.ProcessGroupID != escapedPID ||
				identity.OriginalGroupID != result.processGroupID || identity.OriginalGroupID == identity.ProcessGroupID {
				t.Fatalf("escape fixture identity is incomplete: bytes=%s identity=%+v err=%v result=%+v", identityBytes, identity, err, result)
			}
			t.Cleanup(func() {
				_ = syscall.Kill(escapedPID, syscall.SIGKILL)
			})
			if result.primary != domain.ControlTimeout || !result.finalProbeClean || result.processGroupID == escapedPID {
				t.Fatalf("original process-group result is malformed: %+v", result)
			}
			if !result.teardownError || !result.orphanRisk || result.stdoutDrained || result.stderrDrained {
				t.Fatalf("escaped inherited pipes should make bounded drains visibly incomplete and orphan-uncertain: %+v", result)
			}
			if processEscapeExclusion != "PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM" {
				t.Fatalf("cleanup boundary changed: %q", processEscapeExclusion)
			}
			if processGroupReuseExclusion != "PRE_TERM_PROBE_AND_SIGNAL_ARE_NON_ATOMIC_PGID_REUSE_EXCLUDED_FROM_CLEANUP_CLAIM" {
				t.Fatalf("process-group signal boundary changed: %q", processGroupReuseExclusion)
			}
			if err := syscall.Kill(escapedPID, 0); err != nil {
				t.Fatalf("escaped child was not demonstrably outside the original group: %v", err)
			}
			if err := syscall.Kill(escapedPID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Fatalf("out-of-band cleanup of explicit exclusion failed: %v", err)
			}
		})
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

	t.Run("probe-error-skips-signal-and-retains-uncertainty", func(t *testing.T) {
		controller := &scriptedGroupProbe{probeErrors: []error{syscall.EPERM}}
		result := physicalProcessResult{processGroupOwned: true, processGroupID: 4242}
		teardownOwnedProcessGroup(&result, controller, time.Now().Add(time.Second), time.Second)
		if result.preTermProbe != preTermProbeUncertain || result.termSent || len(controller.signals) != 0 ||
			!result.teardownError || !result.orphanRisk || result.diagnosticCode != "PRE_TERM_GROUP_PROBE_FAILED" {
			t.Fatalf("uncertain pre-TERM probe was not fail-closed: result=%+v events=%v", result, controller.events)
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

func TestDirectCommandContainsNoShellOrAmbientCommandResolution(t *testing.T) {
	request := processRequest{
		tool:        resolvedTool{absolutePath: "/private/fixture"},
		logicalArgv: []string{"fixture", "--mode", "report"},
		environment: []string{"LANG=C"},
		cwd:         "/private/candidate",
	}
	command := newDirectCommand(request)
	if command.Path != request.tool.absolutePath || strings.Join(command.Args, "\x00") != strings.Join(request.logicalArgv, "\x00") ||
		command.Dir != request.cwd || strings.Join(command.Env, "\x00") != "LANG=C" || command.SysProcAttr == nil || !command.SysProcAttr.Setpgid {
		t.Fatalf("direct command changed authority: %+v", command)
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
