//go:build darwin && cgo

package processmechanics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStdoutAndStderrHaveIndependentExactCaps(t *testing.T) {
	executable := buildPhysicalFixture(t)
	result := runPhysicalFixture(
		t, executable, "emit", []string{"--stdout-bytes", "17", "--stderr-bytes", "9"},
		17, 9, time.Second, 400*time.Millisecond,
	)
	if result.Primary != "" || result.StdoutOverflow || result.StderrOverflow ||
		len(result.Stdout) != 17 || len(result.Stderr) != 9 ||
		result.StdoutObserved != 17 || result.StderrObserved != 9 ||
		!result.ChildWaited || !result.StdoutDrained || !result.StderrDrained || !result.FinalProbeClean {
		t.Fatalf("physical channel boundaries were not preserved: %+v", result)
	}
	deadline := time.Now().Add(-time.Second)
	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	stdoutDone <- deadline.Add(time.Millisecond)
	stderrDone <- deadline.Add(-time.Millisecond)
	stdoutDrain, stderrDrain := collectDrainCompletions(stdoutDone, stderrDone, deadline)
	if !stdoutDrain.observed || stdoutDrain.beforeDeadline || !stderrDrain.observed || !stderrDrain.beforeDeadline {
		t.Fatalf("drain deadline facts changed: stdout=%+v stderr=%+v", stdoutDrain, stderrDrain)
	}
}

func TestStdoutAndStderrLimitsAreIndependentMutationGuard(t *testing.T) {
	executable := buildPhysicalFixture(t)
	result := runPhysicalFixture(
		t, executable, "emit", []string{"--stdout-bytes", "4", "--stderr-bytes", "10"},
		4, 9, time.Second, 400*time.Millisecond,
	)
	if result.Primary != ControlOutputLimit || result.StdoutOverflow || !result.StderrOverflow ||
		len(result.Stdout) != 4 || result.StdoutObserved != 4 ||
		len(result.Stderr) != 9 || result.StderrObserved != 10 ||
		!result.ChildWaited || !result.FinalProbeClean {
		t.Fatalf("physical stdout/stderr limits were coupled: %+v", result)
	}
}

func TestSimultaneousChannelOverflowRetainsIndependentFacts(t *testing.T) {
	executable := buildPhysicalFixture(t)
	const limit = int64(4096)
	result := runPhysicalFixture(
		t, executable, "emit-both-ignore-term",
		[]string{"--stdout-bytes", "65536", "--stderr-bytes", "65536"},
		limit, limit, time.Second, 500*time.Millisecond,
	)
	if result.Primary != ControlOutputLimit || !result.StdoutOverflow || !result.StderrOverflow ||
		len(result.Stdout) != int(limit) || len(result.Stderr) != int(limit) ||
		result.StdoutObserved <= limit || result.StderrObserved <= limit ||
		!result.TermSent || !result.KillSent || !result.ChildWaited || !result.FinalProbeClean {
		t.Fatalf("simultaneous physical overflow lost independent facts: %+v", result)
	}
	overflow := make(chan struct{}, 1)
	stdout := newCappedCapture(1, overflow)
	stderr := newCappedCapture(1, overflow)
	stdout.retain([]byte("xx"))
	deadlineC := make(chan time.Time, 1)
	deadlineC <- time.Now()
	waited := &waitResult{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	decision, ready, _ := pollTerminalPriority(ctx, deadlineC, stdout, stderr, false, waited, nil)
	if !ready || decision.primary != ControlOutputLimit {
		t.Fatalf("all-ready owner priority selected %+v", decision)
	}
}

func runPhysicalFixture(
	t *testing.T,
	executable string,
	mode string,
	extra []string,
	stdoutLimit int64,
	stderrLimit int64,
	executionBudget time.Duration,
	teardownBudget time.Duration,
) Result {
	t.Helper()
	cwd := resolvedTestRoot(t)
	state := filepath.Join(cwd, "state")
	evidence := filepath.Join(cwd, "evidence")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "attempt.marker.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	argv := []string{"fixture", "--mode", mode}
	argv = append(argv, extra...)
	invocation, err := NewInvocation(
		executable,
		argv,
		[]string{
			"COUNTERSHAPE_ATTEMPT_ID=attempt:processmechanics-test",
			"COUNTERSHAPE_EVIDENCE_ROOT=" + evidence,
			"COUNTERSHAPE_STATE_ROOT=" + state,
			"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin",
		},
		AbsentStdin(), cwd,
		Limits{StdoutBytes: stdoutLimit, StderrBytes: stderrLimit, Execution: executionBudget, Teardown: teardownBudget},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	observation, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if observation.ProcessGroupOwned && observation.ProcessGroupID > 0 {
			killProcessGroup(observation.ProcessGroupID)
		}
	})
	return running.Close()
}
