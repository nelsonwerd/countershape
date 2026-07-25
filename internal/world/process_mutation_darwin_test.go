//go:build darwin && cgo

package world

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestBuildEnvironmentDoesNotInheritAmbientVariables(t *testing.T) {
	t.Setenv("COUNTERSHAPE_AMBIENT_SENTINEL", "must-not-cross")
	roots := Roots{
		home:      "/countershape/home",
		temporary: "/countershape/tmp",
		xdgConfig: "/countershape/xdg-config",
		xdgCache:  "/countershape/xdg-cache",
		xdgData:   "/countershape/xdg-data",
		xdgState:  "/countershape/xdg-state",
		state:     "/countershape/state",
		evidence:  "/countershape/evidence",
	}
	want := []string{
		"COUNTERSHAPE_ATTEMPT_ID=attempt:mutation-guard",
		"COUNTERSHAPE_EVIDENCE_ROOT=/countershape/evidence",
		"COUNTERSHAPE_STATE_ROOT=/countershape/state",
		"HOME=/countershape/home",
		"LANG=C",
		"TMPDIR=/countershape/tmp",
		"XDG_CACHE_HOME=/countershape/xdg-cache",
		"XDG_CONFIG_HOME=/countershape/xdg-config",
		"XDG_DATA_HOME=/countershape/xdg-data",
		"XDG_STATE_HOME=/countershape/xdg-state",
	}
	got := buildEnvironment(
		[]domain.EnvironmentEntry{{Name: "LANG", Value: "C"}},
		roots,
		"attempt:mutation-guard",
	)
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("candidate environment = %q, want exact sparse environment %q", got, want)
	}
}

func TestWorldAdapterPreservesIndependentLimitsMutationGuard(t *testing.T) {
	executable := buildProcessFixture(t)
	result, _ := runFixtureProcess(
		t,
		context.Background(),
		executable,
		"emit",
		[]string{"--stdout-bytes", "4", "--stderr-bytes", "10"},
		4,
		9,
		time.Second,
		400*time.Millisecond,
	)
	if result.primary != domain.ControlOutputLimit || result.stdoutOverflow || !result.stderrOverflow ||
		len(result.stdout) != 4 || result.stdoutObserved != 4 || len(result.stderr) != 9 || result.stderrObserved != 10 {
		t.Fatalf("independent capture limits changed: %+v", result)
	}
}

func TestOutputOverflowCannotBecomeSuccessfulTruncation(t *testing.T) {
	result := physicalProcessResult{stdout: []byte("retained"), stdoutObserved: 9, stdoutOverflow: true}
	applyCompletedOutputControl(&result)
	if result.primary != domain.ControlOutputLimit {
		t.Fatalf("overflow was reported as successful truncation: %+v", result)
	}
}

func TestNegativePGIDSignalCleansDescendantsAfterParentExit(t *testing.T) {
	executable := buildProcessFixture(t)
	result, roots := runFixtureProcess(
		t,
		context.Background(),
		executable,
		"parent-exits",
		nil,
		1024,
		1024,
		time.Second,
		500*time.Millisecond,
	)
	registerEmergencyGroupCleanup(t, result)
	grandchildPID := registerEmergencyPIDFileCleanup(t, filepath.Join(roots.state, "grandchild.pid"))
	if result.primary != "" || !result.processGroupOwned || result.processGroupID != result.pid ||
		!result.termSent || !result.directChildWaited || !result.stdoutDrained || !result.stderrDrained ||
		!result.finalProbeClean || result.teardownError || result.orphanRisk {
		t.Fatalf("negative-PGID teardown did not clean the surviving descendant: %+v", result)
	}
	if err := syscall.Kill(grandchildPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("grandchild %d remains after a clean receipt: %v", grandchildPID, err)
	}
}

func TestTermIgnoringProcessRequiresKillEscalation(t *testing.T) {
	executable := buildProcessFixture(t)
	result, _ := runFixtureProcess(
		t,
		context.Background(),
		executable,
		"ignore-term",
		nil,
		1024,
		1024,
		100*time.Millisecond,
		500*time.Millisecond,
	)
	registerEmergencyGroupCleanup(t, result)
	if result.primary != domain.ControlTimeout || !result.processGroupOwned || !result.termSent || !result.killSent ||
		!result.directChildWaited || !result.stdoutDrained || !result.stderrDrained || !result.finalProbeClean ||
		result.teardownError || result.orphanRisk {
		t.Fatalf("TERM-resistant process did not require and complete KILL escalation: %+v", result)
	}
}

func TestDrainDeadlineReportsIncompleteDrain(t *testing.T) {
	deadline := time.Now().Add(-time.Second)
	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	stdoutDone <- deadline.Add(time.Millisecond)
	stderrDone <- deadline.Add(-time.Millisecond)
	stdout, stderr := collectDrainCompletions(stdoutDone, stderrDone, deadline)
	if !stdout.observed || stdout.beforeDeadline {
		t.Fatalf("post-deadline stdout drain was reported complete: %+v", stdout)
	}
	if !stderr.observed || !stderr.beforeDeadline {
		t.Fatalf("pre-deadline stderr drain was reported incomplete: %+v", stderr)
	}
}

func TestProcessEscapeBoundaryCannotClaimContainment(t *testing.T) {
	if processEscapeExclusion != "PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM" {
		t.Fatalf("process escape was promoted into a containment claim: %q", processEscapeExclusion)
	}
}

func registerEmergencyGroupCleanup(t *testing.T, result physicalProcessResult) {
	t.Helper()
	if !result.processGroupOwned || result.processGroupID <= 0 || result.processGroupID != result.pid {
		return
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-result.processGroupID, syscall.SIGKILL)
		_ = syscall.Kill(result.pid, syscall.SIGKILL)
	})
}

func registerEmergencyPIDFileCleanup(t *testing.T, path string) int {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture did not publish emergency-cleanup pid at %s: %v", path, err)
	}
	pid, err := strconv.Atoi(string(bytes))
	if err != nil || pid <= 0 {
		t.Fatalf("fixture published invalid emergency-cleanup pid %q", bytes)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	})
	return pid
}
