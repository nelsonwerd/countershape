//go:build darwin && cgo

package processmechanics

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func resolvedTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func buildPhysicalFixture(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime did not disclose process-mechanics test source")
	}
	moduleRoot, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(source), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root := resolvedTestRoot(t)
	output := filepath.Join(root, "countershape-process-fixture")
	goExecutable := filepath.Join(runtime.GOROOT(), "bin", "go")
	for _, name := range []string{"home", "tmp", "gocache", "gomodcache", "gopath"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command(goExecutable, "build", "-trimpath", "-o", output, "./testkit/processfixture")
	command.Dir = moduleRoot
	command.Env = []string{
		"HOME=" + filepath.Join(root, "home"),
		"TMPDIR=" + filepath.Join(root, "tmp"),
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1",
		"PATH=" + filepath.Dir(goExecutable) + ":/usr/bin:/bin",
		"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=",
		"GOCACHE=" + filepath.Join(root, "gocache"),
		"GOMODCACHE=" + filepath.Join(root, "gomodcache"),
		"GOPATH=" + filepath.Join(root, "gopath"),
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

func killProcessGroup(processGroupID int) {
	_ = syscall.Kill(-processGroupID, syscall.SIGKILL)
}

type scriptedProbeClock struct {
	current   time.Time
	overshoot time.Duration
	waits     []time.Duration
}

func (clock *scriptedProbeClock) now() time.Time { return clock.current }

func (clock *scriptedProbeClock) wait(duration time.Duration) {
	clock.waits = append(clock.waits, duration)
	clock.current = clock.current.Add(duration + clock.overshoot)
}

type deadlineCrossingProbe struct {
	clock       *scriptedProbeClock
	nextPresent bool
	calls       int
}

type finalProbeSequence struct {
	results []bool
	calls   int
}

func (*finalProbeSequence) signal(int, syscall.Signal) error { return nil }

func (probe *finalProbeSequence) probe(int) (bool, error) {
	probe.calls++
	if len(probe.results) == 0 {
		return false, nil
	}
	result := probe.results[0]
	probe.results = probe.results[1:]
	return result, nil
}

func (*deadlineCrossingProbe) signal(int, syscall.Signal) error { return nil }

func (probe *deadlineCrossingProbe) probe(int) (bool, error) {
	probe.calls++
	if probe.calls == 1 {
		return true, syscall.EPERM
	}
	probe.clock.current = probe.clock.current.Add(time.Millisecond)
	return probe.nextPresent, nil
}

func TestPreTermRetryNeverUsesPostDeadlineProbeAsSignalAuthority(t *testing.T) {
	for _, cleanResult := range []bool{false, true} {
		start := time.Unix(100, 0)
		clock := &scriptedProbeClock{current: start}
		controller := &deadlineCrossingProbe{clock: clock, nextPresent: cleanResult}
		present, err := resolvePreTermGroupProbeWithClock(
			controller, 4242, start.Add(time.Second), 8*time.Millisecond, clock,
		)
		if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 2 ||
			len(clock.waits) != 1 || clock.waits[0] != time.Millisecond {
			t.Fatalf("post-deadline probe became signal authority: present=%t err=%v calls=%d waits=%v",
				present, err, controller.calls, clock.waits)
		}
	}
	finalProbe := &finalProbeSequence{results: []bool{true, false}}
	clean, err := performFinalGroupProbe(finalProbe, 4242, time.Now().Add(100*time.Millisecond))
	if err != nil || !clean || finalProbe.calls != 2 {
		t.Fatalf("final group probe did not require observed absence: clean=%t err=%v calls=%d", clean, err, finalProbe.calls)
	}
}

type countingEPERMProbe struct{ calls int }

func (*countingEPERMProbe) signal(int, syscall.Signal) error { return nil }
func (probe *countingEPERMProbe) probe(int) (bool, error) {
	probe.calls++
	return true, syscall.EPERM
}

func TestPreTermRetryWaitOvershootDoesNotConsumeAnotherProbe(t *testing.T) {
	start := time.Unix(100, 0)
	clock := &scriptedProbeClock{current: start, overshoot: time.Nanosecond}
	controller := &countingEPERMProbe{}
	present, err := resolvePreTermGroupProbeWithClock(
		controller, 4242, start.Add(time.Second), 4*time.Millisecond, clock,
	)
	if !present || !errors.Is(err, syscall.EPERM) || controller.calls != 1 || len(clock.waits) != 1 {
		t.Fatalf("overshot wait consumed another probe: present=%t err=%v calls=%d waits=%v",
			present, err, controller.calls, clock.waits)
	}
}

func TestPreparedProcessStartsOnceAndClosesWithParentObservedFacts(t *testing.T) {
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/echo",
		[]string{"echo", "neutral-process-mechanics"},
		[]string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		AbsentStdin(),
		cwd,
		Limits{StdoutBytes: 1024, StderrBytes: 1024, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	binding := prepared.BindingDigest()
	if !binding.Valid() {
		t.Fatal("prepared process lacks an invocation binding digest")
	}
	observation, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if observation.PID <= 0 || observation.ProcessGroupID != observation.PID || !observation.ProcessGroupOwned ||
		observation.Binding != binding {
		t.Fatalf("spawn observation is incomplete: %+v", observation)
	}
	if _, _, secondErr := prepared.Start(context.Background()); secondErr == nil {
		t.Fatal("one prepared operation started twice")
	}
	result := running.Close()
	if !result.PhysicalExecutionEntered || !result.SpawnAttempted || !result.Started ||
		result.PID != observation.PID || result.ProcessGroupID != observation.ProcessGroupID ||
		!result.ProcessGroupOwned || result.Binding != binding || result.Primary != "" || result.ExitCode != 0 ||
		string(result.Stdout) != "neutral-process-mechanics\n" || len(result.Stderr) != 0 ||
		!result.ChildWaited || !result.StdoutDrained || !result.StderrDrained ||
		!result.FinalProbeClean || result.TeardownError || result.OrphanRisk {
		t.Fatalf("clean staged process closure is incomplete: %+v", result)
	}
	again := running.Close()
	if string(again.Stdout) != string(result.Stdout) || again.PID != result.PID || again.ExitCode != result.ExitCode {
		t.Fatalf("idempotent close changed result: first=%+v second=%+v", result, again)
	}
}

func TestCopiedPreparedHandleCannotMultiplyStartAuthority(t *testing.T) {
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/echo", []string{"echo", "shared-start-authority"},
		[]string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}, AbsentStdin(), cwd,
		Limits{StdoutBytes: 128, StderrBytes: 128, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	const contenders = 16
	copies := make([]Prepared, contenders)
	for index := range copies {
		copies[index] = *prepared
	}
	type outcome struct {
		running *Running
		err     error
	}
	gate := make(chan struct{})
	outcomes := make(chan outcome, contenders)
	var group sync.WaitGroup
	for index := range copies {
		group.Add(1)
		go func(candidate *Prepared) {
			defer group.Done()
			<-gate
			_, running, startErr := candidate.Start(context.Background())
			outcomes <- outcome{running: running, err: startErr}
		}(&copies[index])
	}
	close(gate)
	group.Wait()
	close(outcomes)
	successes := 0
	for observed := range outcomes {
		if observed.err == nil {
			successes++
			if observed.running == nil {
				t.Fatal("successful copied handle returned no running process")
			}
			result := observed.running.Close()
			if result.Primary != "" || result.ExitCode != 0 || string(result.Stdout) != "shared-start-authority\n" {
				t.Fatalf("winning copied handle did not close cleanly: %+v", result)
			}
			continue
		}
		var failure *StartError
		if observed.running != nil || !errors.As(observed.err, &failure) ||
			failure.Code() != "PREPARED_PROCESS_ALREADY_CONSUMED" {
			t.Fatalf("losing copied handle was not an exact consumed refusal: running=%v err=%v", observed.running, observed.err)
		}
		refused := failure.Result()
		if refused.Binding != prepared.BindingDigest() || refused.PhysicalExecutionEntered || refused.SpawnAttempted ||
			refused.Started || refused.PID != 0 || refused.ProcessGroupID != 0 || refused.ProcessGroupOwned {
			t.Fatalf("pre-spawn refusal forged physical execution facts: %+v", refused)
		}
	}
	if successes != 1 {
		t.Fatalf("copied prepared handles produced %d physical starts, want exactly one", successes)
	}
}

func TestCopiedRunningHandleSharesOneTerminalClosure(t *testing.T) {
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/sleep", []string{"sleep", "1"},
		[]string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}, AbsentStdin(), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	running.AbortSpawnObservationPersistence()
	const contenders = 16
	copies := make([]Running, contenders)
	for index := range copies {
		copies[index] = *running
	}
	gate := make(chan struct{})
	results := make(chan Result, contenders)
	var group sync.WaitGroup
	for index := range copies {
		group.Add(1)
		go func(candidate *Running) {
			defer group.Done()
			<-gate
			candidate.AbortSpawnObservationPersistence()
			results <- candidate.Close()
		}(&copies[index])
	}
	close(gate)
	group.Wait()
	close(results)
	var first Result
	for result := range results {
		if first.Binding == (BindingDigest{}) {
			first = result
		} else if !reflect.DeepEqual(result, first) {
			t.Fatalf("copied running handles observed different terminal results:\nfirst=%+v\nnext=%+v", first, result)
		}
	}
	if first.Primary != ControlStartError || first.DiagnosticCode != "SPAWN_OBSERVATION_PERSISTENCE_FAILED" ||
		!first.TeardownError || !first.ChildWaited || !first.FinalProbeClean {
		t.Fatalf("shared copied closure lost terminal abort facts: %+v", first)
	}
	if final := running.Close(); !reflect.DeepEqual(final, first) {
		t.Fatalf("original running handle did not share copied closure: first=%+v final=%+v", first, final)
	}
}

func TestPresentEmptyStdinRemainsPhysicallyDistinctFromAbsentStdin(t *testing.T) {
	cwd := resolvedTestRoot(t)
	var err error
	invocation, err := NewInvocation(
		"/bin/cat", []string{"cat"}, []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		PresentStdin(nil), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result := running.Close()
	if !result.StdinPresent || result.StdinDeclared != 0 || !result.StdinPipeAllocated ||
		!result.StdinWriterStarted || !result.StdinHandoffAttempted || result.StdinWritten != 0 ||
		!result.StdinComplete || result.StdinErrorCode != "" || result.Primary != "" {
		t.Fatalf("present-empty stdin collapsed into absence: %+v", result)
	}
	const payload = "exact-stdin-bytes"
	invocation, err = NewInvocation(
		"/bin/cat", []string{"cat"}, []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		PresentStdin([]byte(payload)), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, running, err = prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result = running.Close()
	if string(result.Stdout) != payload || result.StdinWritten != int64(len(payload)) || !result.StdinComplete {
		t.Fatalf("present stdin bytes changed during physical delivery: %+v", result)
	}
}

func TestExecutionBudgetStartsAtPhysicalStartNotClose(t *testing.T) {
	deadlineC := make(chan time.Time, 1)
	deadlineC <- time.Now()
	waited := &waitResult{}
	decision, ready, observed := pollTerminalPriority(
		context.Background(), deadlineC, newCappedCapture(1, make(chan struct{}, 1)),
		newCappedCapture(1, make(chan struct{}, 1)), false, waited, nil,
	)
	if !ready || !observed || decision.primary != ControlTimeout {
		t.Fatalf("latched wait outranked an observed execution deadline: decision=%+v ready=%t observed=%t", decision, ready, observed)
	}
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/sleep", []string{"sleep", "1"}, []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		AbsentStdin(), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: 40 * time.Millisecond, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	result := running.Close()
	if result.Primary != ControlTimeout || !result.TermSent || !result.ChildWaited ||
		!result.StdoutDrained || !result.StderrDrained || !result.FinalProbeClean || result.OrphanRisk {
		t.Fatalf("time between Start and Close escaped the execution budget: %+v", result)
	}
}

func TestPostPermitCancellationStillProducesAChildObservation(t *testing.T) {
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/sleep", []string{"sleep", "1"}, []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		AbsentStdin(), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	observation, running, err := prepared.Start(ctx)
	if err != nil || running == nil || observation.PID <= 0 {
		t.Fatalf("post-admission cancellation became an unmappable no-child start error: observation=%+v running=%v err=%v", observation, running, err)
	}
	result := running.Close()
	if result.Primary != ControlCancelled || !result.ChildWaited || !result.FinalProbeClean {
		t.Fatalf("post-spawn cancellation did not close physically: %+v", result)
	}
}

func TestSpawnObservationPersistenceAbortIsClosedAndTerminal(t *testing.T) {
	cwd := resolvedTestRoot(t)
	invocation, err := NewInvocation(
		"/bin/sleep", []string{"sleep", "1"}, []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
		AbsentStdin(), cwd,
		Limits{StdoutBytes: 32, StderrBytes: 32, Execution: time.Second, Teardown: 500 * time.Millisecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, running, err := prepared.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	running.AbortSpawnObservationPersistence()
	result := running.Close()
	if result.Primary != ControlStartError || result.DiagnosticCode != "SPAWN_OBSERVATION_PERSISTENCE_FAILED" ||
		!result.TeardownError || !result.ChildWaited || !result.FinalProbeClean {
		t.Fatalf("closed persistence abort did not retain terminal facts: %+v", result)
	}
}
