//go:build darwin

package processmechanics

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const ownerDrainCloseBudget = 100 * time.Millisecond

const (
	signalOwnedProcessGroup        = true // MUTANT_U2_SIGNAL_DIRECT_PID
	killEscalationRequired         = true // MUTANT_U2_REMOVE_KILL_ESCALATION
	finalGroupProbeRequired        = true // MUTANT_U2_REMOVE_FINAL_GROUP_PROBE
	outputOverflowIsPrimaryControl = true // MUTANT_U2_MAP_OUTPUT_LIMIT_TO_SUCCESS
	drainDeadlineIsFailure         = true // MUTANT_U2_REPORT_DRAIN_TIMEOUT_COMPLETE
	ownerPriorityOutputFirst       = true // MUTANT_U2_LATE_CONTROL_OVERRIDES_OUTPUT
)

type waitResult struct {
	state *os.ProcessState
	err   error
}

type stdinWriteResult struct {
	written   int64
	complete  bool
	errorCode string
}

type terminalDecision struct {
	primary string
	waited  *waitResult
	stdin   *stdinWriteResult
}

type capturePair struct {
	stdout             *cappedCapture
	stderr             *cappedCapture
	stdoutReceiptLimit int64
	stderrReceiptLimit int64
}

func newCapturePair(stdoutLimit, stderrLimit int64, overflowC chan<- struct{}) capturePair {
	stdout := newCappedCapture(stdoutLimit, overflowC)
	stderr := newCappedCapture(stderrLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP
	return capturePair{
		stdout: stdout, stderr: stderr,
		stdoutReceiptLimit: stdout.configuredLimit(), stderrReceiptLimit: stderr.configuredLimit(),
	}
}

type darwinGroupController interface {
	signal(processGroupID int, signal syscall.Signal) error
	probe(processGroupID int) (bool, error)
}

type systemDarwinGroups struct{}

type preTermProbeClock interface {
	now() time.Time
	wait(time.Duration)
}

type systemPreTermProbeClock struct{}

func (systemPreTermProbeClock) now() time.Time { return time.Now() }

func (systemPreTermProbeClock) wait(duration time.Duration) {
	timer := time.NewTimer(duration)
	<-timer.C
}

func (systemDarwinGroups) signal(processGroupID int, signal syscall.Signal) error {
	target := -processGroupID
	if !signalOwnedProcessGroup {
		target = processGroupID
	}
	return syscall.Kill(target, signal)
}

func (systemDarwinGroups) probe(processGroupID int) (bool, error) {
	err := syscall.Kill(-processGroupID, 0)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, syscall.EPERM) {
		return true, err
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, err
}

type runningState struct {
	prepared *preparedState
	ctx      context.Context
	command  *exec.Cmd

	stdoutPipe     *os.File
	stderrPipe     *os.File
	stdinWriter    io.WriteCloser
	stdinResultC   <-chan stdinWriteResult
	waitC          <-chan waitResult
	stdoutDone     <-chan time.Time
	stderrDone     <-chan time.Time
	overflowC      <-chan struct{}
	captures       capturePair
	executionTimer *time.Timer

	mu        sync.Mutex
	result    Result
	closing   bool
	closeOnce sync.Once
	final     Result
}

// Running owns every handle created by one successful os/exec Start. Its
// private shared state keeps abort and Close one-shot even if the exported
// handle value is copied. Close is the only source of terminal facts.
type Running struct {
	state *runningState
}

func (prepared *Prepared) startPlatform(ctx context.Context) (SpawnObservation, *Running, error) {
	if prepared == nil || prepared.state == nil {
		return SpawnObservation{}, nil, newStartError("NIL_PREPARED_PROCESS", nil, Result{ExitCode: -1, PreTermProbe: PreTermProbeNotApplicable})
	}
	result := initialResult(prepared.state.invocation)
	if err := prepared.begin(); err != nil {
		return SpawnObservation{}, nil, newStartError("PREPARED_PROCESS_ALREADY_CONSUMED", err, result)
	}
	if ctx == nil {
		return SpawnObservation{}, nil, newStartError("NIL_PROCESS_CONTEXT", nil, result)
	}
	invocation := prepared.state.invocation
	overflowC := make(chan struct{}, 2)
	captures := newCapturePair(invocation.limits.StdoutBytes, invocation.limits.StderrBytes, overflowC)
	command := &exec.Cmd{
		Path:        invocation.executable, // MUTANT_U2_SPAWN_THROUGH_SHELL
		Args:        append([]string(nil), invocation.argv...),
		Env:         append([]string(nil), invocation.environment...),
		Dir:         invocation.cwd,
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
	}

	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		return SpawnObservation{}, nil, newStartError("STDOUT_PIPE_ALLOCATION_FAILED", err, result)
	}
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		return SpawnObservation{}, nil, newStartError("STDERR_PIPE_ALLOCATION_FAILED", err, result)
	}

	var stdinWriter io.WriteCloser
	if invocation.stdin.presence == stdinPresent {
		stdinWriter, err = command.StdinPipe()
		if err != nil {
			_ = stdoutPipe.Close()
			_ = stdoutWriter.Close()
			_ = stderrPipe.Close()
			_ = stderrWriter.Close()
			return SpawnObservation{}, nil, newStartError("STDIN_PIPE_ALLOCATION_FAILED", err, result)
		}
		result.StdinPipeAllocated = true
	}
	command.Stdout = stdoutWriter
	command.Stderr = stderrWriter
	result.PhysicalExecutionEntered = true
	result.SpawnAttempted = true
	result.StdoutCaptureLimit = captures.stdoutReceiptLimit
	result.StderrCaptureLimit = captures.stderrReceiptLimit
	if err := command.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		_ = stderrPipe.Close()
		_ = stderrWriter.Close()
		if stdinWriter != nil {
			_ = stdinWriter.Close()
		}
		return SpawnObservation{}, nil, newStartError("SPAWN_FAILED", err, result)
	}
	executionTimer := time.NewTimer(invocation.limits.Execution)
	result.Started = true
	result.PID = command.Process.Pid

	var stdinResultC chan stdinWriteResult
	if stdinWriter != nil {
		stdinResultC = make(chan stdinWriteResult, 1)
		// MUTATION_ANCHOR: stdin-absence-must-not-collapse-to-present-empty
		startExactStdinWriter(stdinWriter, invocation.stdin.bytes, stdinResultC)
		result.StdinWriterStarted = true
		result.StdinHandoffAttempted = true
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	processGroupID, groupErr := syscall.Getpgid(command.Process.Pid)
	result.ProcessGroupID = processGroupID
	if groupErr == nil && processGroupID == command.Process.Pid {
		result.ProcessGroupOwned = true
	} else {
		result.Primary = ControlStartError
		result.TeardownError = true
		result.OrphanRisk = true
		result.FinalProbeError = "new process group was not established"
		result.DiagnosticCode = "PROCESS_GROUP_NOT_OWNED"
		_ = command.Process.Kill()
	}

	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	go captures.stdout.drain(stdoutPipe, stdoutDone)
	go captures.stderr.drain(stderrPipe, stderrDone)
	waitC := make(chan waitResult, 1)
	go func() {
		waitErr := command.Wait()
		waitC <- waitResult{state: command.ProcessState, err: waitErr}
	}()

	running := &Running{state: &runningState{
		prepared: prepared.state, ctx: ctx, command: command,
		stdoutPipe: stdoutPipe, stderrPipe: stderrPipe, stdinWriter: stdinWriter,
		stdinResultC: stdinResultC, waitC: waitC, stdoutDone: stdoutDone, stderrDone: stderrDone,
		overflowC: overflowC, captures: captures, result: result, executionTimer: executionTimer,
	}}
	observation := SpawnObservation{
		PID: result.PID, ProcessGroupID: result.ProcessGroupID, ProcessGroupOwned: result.ProcessGroupOwned,
		Binding: result.Binding,
	}
	return observation, running, nil
}

// AbortReadinessTransition is the closed world-compatibility abort after a
// failed parent-owned readiness transition.
func (running *Running) AbortReadinessTransition() {
	running.abortBeforeClose("READY_TRANSITION_FAILED")
}

// AbortSpawnObservationPersistence is the closed contract-runner abort when a
// parent-observed PID could not be durably recorded. Such a run can be cleaned
// but can never enter a durable finalized-run record.
func (running *Running) AbortSpawnObservationPersistence() {
	running.abortBeforeClose("SPAWN_OBSERVATION_PERSISTENCE_FAILED")
}

func (running *Running) abortBeforeClose(code string) {
	if running == nil || running.state == nil {
		return
	}
	state := running.state
	state.mu.Lock()
	if state.closing {
		state.mu.Unlock()
		return
	}
	if state.result.Primary == "" {
		state.result.Primary = ControlStartError
	}
	state.result.TeardownError = true
	state.result.DiagnosticCode = firstDiagnostic(state.result.DiagnosticCode, code)
	command := state.command
	state.mu.Unlock()
	if command != nil && command.Process != nil {
		_ = command.Process.Kill()
	}
}

func (running *Running) Close() Result {
	if running == nil || running.state == nil {
		return Result{ExitCode: -1, Primary: ControlStartError, PreTermProbe: PreTermProbeNotApplicable, DiagnosticCode: "NIL_RUNNING_PROCESS"}
	}
	state := running.state
	state.closeOnce.Do(func() {
		state.final = state.closePhysical()
	})
	return cloneResult(state.final)
}

func (state *runningState) closePhysical() Result {
	defer state.stdoutPipe.Close()
	defer state.stderrPipe.Close()
	state.mu.Lock()
	state.closing = true
	result := cloneResult(state.result)
	state.mu.Unlock()

	var waited *waitResult
	var stdinObserved *stdinWriteResult
	if result.Primary == "" {
		decision := arbitrateTerminalWithStdin(
			state.ctx, state.executionTimer.C, state.overflowC,
			state.captures.stdout, state.captures.stderr, state.waitC, state.stdinResultC,
		)
		if !state.executionTimer.Stop() {
			select {
			case <-state.executionTimer.C:
			default:
			}
		}
		result.Primary = decision.primary
		waited = decision.waited
		stdinObserved = decision.stdin
	} else if !state.executionTimer.Stop() {
		select {
		case <-state.executionTimer.C:
		default:
		}
	}

	teardownBudget := state.prepared.invocation.limits.Teardown
	teardownDeadline := time.Now().Add(teardownBudget)
	controller := systemDarwinGroups{}
	if result.ProcessGroupOwned {
		teardownOwnedProcessGroup(&result, controller, teardownDeadline, teardownBudget)
	}
	if waited == nil {
		waited = waitForDirectChild(state.waitC, teardownDeadline)
	}
	if waited != nil {
		classifyWait(&result, *waited)
	} else {
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "DIRECT_CHILD_WAIT_DEADLINE")
		_ = state.command.Process.Kill()
	}

	stdinDelivery := collectStdinDelivery(stdinObserved, state.stdinResultC, state.stdinWriter, teardownDeadline)
	result.StdinWritten = stdinDelivery.written
	result.StdinComplete = stdinDelivery.complete
	result.StdinErrorCode = stdinDelivery.errorCode
	if result.StdinPresent && !stdinDelivery.complete {
		if result.Primary == "" {
			result.Primary = ControlProbeTransportError
		}
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "STDIN_DELIVERY_"+stdinDelivery.errorCode)
		if stdinDelivery.errorCode == "WRITER_DRAIN_DEADLINE" {
			result.TeardownError = true
			result.OrphanRisk = true
		}
	}

	stdoutDrain, stderrDrain := collectDrainCompletions(state.stdoutDone, state.stderrDone, teardownDeadline)
	result.StdoutDrained = stdoutDrain.beforeDeadline
	result.StderrDrained = stderrDrain.beforeDeadline
	if !stdoutDrain.observed {
		_ = state.stdoutPipe.Close()
	}
	if !stderrDrain.observed {
		_ = state.stderrPipe.Close()
	}
	if !stdoutDrain.observed && !awaitOwnerDrain(state.stdoutDone, ownerDrainCloseBudget) {
		state.captures.stdout.freeze()
	}
	if !stderrDrain.observed && !awaitOwnerDrain(state.stderrDone, ownerDrainCloseBudget) {
		state.captures.stderr.freeze()
	}
	if !result.StdoutDrained || !result.StderrDrained {
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "PIPE_DRAIN_DEADLINE")
	}
	var err error
	result.Stdout, result.StdoutObserved, result.StdoutOverflow, err = state.captures.stdout.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "STDOUT_DRAIN_FAILED")
	}
	result.Stderr, result.StderrObserved, result.StderrOverflow, err = state.captures.stderr.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "STDERR_DRAIN_FAILED")
	}
	if outputOverflowIsPrimaryControl && result.Primary == "" && (result.StdoutOverflow || result.StderrOverflow) {
		result.Primary = ControlOutputLimit
	}

	if result.ProcessGroupOwned {
		clean, probeErr := performFinalGroupProbe(controller, result.ProcessGroupID, teardownDeadline)
		result.FinalProbeClean = clean
		if probeErr != nil {
			result.FinalProbeError = probeErr.Error()
			result.TeardownError = true
			result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "FINAL_GROUP_PROBE_FAILED")
		}
		if !clean {
			result.OrphanRisk = true
		}
	}
	if result.Started && (!result.ProcessGroupOwned || !result.ChildWaited || !result.StdoutDrained || !result.StderrDrained || !result.FinalProbeClean) {
		result.OrphanRisk = true
	}
	return result
}

func startExactStdinWriter(writer io.WriteCloser, input []byte, resultC chan<- stdinWriteResult) {
	copyInput := append([]byte(nil), input...)
	go writeExactStdin(writer, copyInput, resultC)
}

func writeExactStdin(writer io.WriteCloser, input []byte, resultC chan<- stdinWriteResult) {
	result := stdinWriteResult{}
	for result.written < int64(len(input)) {
		count, err := writer.Write(input[result.written:])
		result.written += int64(count)
		if err != nil {
			result.errorCode = classifyStdinWriteError(err)
			_ = writer.Close()
			resultC <- result
			return
		}
		if count == 0 {
			result.errorCode = "NO_PROGRESS"
			_ = writer.Close()
			resultC <- result
			return
		}
	}
	if err := writer.Close(); err != nil {
		result.errorCode = classifyStdinWriteError(err)
		resultC <- result
		return
	}
	result.complete = true
	resultC <- result
}

func classifyStdinWriteError(err error) string {
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe) {
		return "EPIPE"
	}
	return "WRITE_ERROR"
}

func collectStdinDelivery(observed *stdinWriteResult, resultC <-chan stdinWriteResult, writer io.WriteCloser, deadline time.Time) stdinWriteResult {
	if writer == nil {
		return stdinWriteResult{complete: true}
	}
	if observed != nil {
		return *observed
	}
	remaining := time.Until(deadline)
	if remaining > 0 {
		timer := time.NewTimer(remaining)
		select {
		case result := <-resultC:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return result
		case <-timer.C:
		}
	} else {
		select {
		case result := <-resultC:
			return result
		default:
		}
	}
	_ = writer.Close()
	timer := time.NewTimer(ownerDrainCloseBudget)
	defer timer.Stop()
	select {
	case result := <-resultC:
		return result
	case <-timer.C:
		return stdinWriteResult{errorCode: "WRITER_DRAIN_DEADLINE"}
	}
}

func arbitrateTerminalWithStdin(
	ctx context.Context,
	deadlineC <-chan time.Time,
	overflowC <-chan struct{},
	stdoutCapture, stderrCapture *cappedCapture,
	waitC <-chan waitResult,
	stdinC <-chan stdinWriteResult,
) terminalDecision {
	deadlineObserved := false
	var waited *waitResult
	var stdinObserved *stdinWriteResult
	for {
		if stdinC != nil && stdinObserved == nil {
			select {
			case completed := <-stdinC:
				stdinObserved = &completed
			default:
			}
		}
		decision, ready, observed := pollTerminalPriority(ctx, deadlineC, stdoutCapture, stderrCapture, deadlineObserved, waited, stdinObserved)
		deadlineObserved = observed
		if ready {
			return decision
		}
		select {
		case <-overflowC:
			continue
		default:
		}
		if ctx.Err() != nil {
			continue
		}
		select {
		case <-deadlineC:
			deadlineObserved = true
			continue
		default:
		}
		select {
		case completed := <-waitC:
			waited = &completed
			continue
		default:
		}
		select {
		case <-overflowC:
		case <-ctx.Done():
		case <-deadlineC:
			deadlineObserved = true
		case completed := <-waitC:
			waited = &completed
		case completed := <-stdinC:
			stdinObserved = &completed
		}
	}
}

func pollTerminalPriority(
	ctx context.Context,
	deadlineC <-chan time.Time,
	stdoutCapture, stderrCapture *cappedCapture,
	deadlineObserved bool,
	waited *waitResult,
	stdin *stdinWriteResult,
) (terminalDecision, bool, bool) {
	if !ownerPriorityOutputFirst && ctx.Err() != nil {
		return terminalDecision{primary: ControlCancelled, stdin: stdin}, true, deadlineObserved
	}
	if outputOverflowIsPrimaryControl && (stdoutCapture.overflowed() || stderrCapture.overflowed()) {
		return terminalDecision{primary: ControlOutputLimit, stdin: stdin}, true, deadlineObserved
	}
	if stdin != nil && !stdin.complete {
		return terminalDecision{primary: ControlProbeTransportError, stdin: stdin}, true, deadlineObserved
	}
	if ctx.Err() != nil {
		return terminalDecision{primary: ControlCancelled, stdin: stdin}, true, deadlineObserved
	}
	if !deadlineObserved {
		select {
		case <-deadlineC:
			deadlineObserved = true
		default:
		}
	}
	if deadlineObserved {
		return terminalDecision{primary: ControlTimeout, stdin: stdin}, true, true
	}
	if waited != nil {
		return terminalDecision{waited: waited, stdin: stdin}, true, false
	}
	return terminalDecision{}, false, false
}

func teardownOwnedProcessGroup(result *Result, controller darwinGroupController, teardownDeadline time.Time, teardownBudget time.Duration) {
	present, probeErr := resolvePreTermGroupProbe(controller, result.ProcessGroupID, teardownDeadline, teardownBudget)
	if probeErr != nil {
		result.PreTermProbe = PreTermProbeUncertain
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "PRE_TERM_GROUP_PROBE_FAILED")
		return
	}
	if !present {
		result.PreTermProbe = PreTermProbeAbsent
		return
	}
	result.PreTermProbe = PreTermProbePresent
	termErr := controller.signal(result.ProcessGroupID, syscall.SIGTERM)
	if termErr == nil {
		result.TermSent = true
		graceDeadline := time.Now().Add(teardownBudget / 3)
		if graceDeadline.After(teardownDeadline) {
			graceDeadline = teardownDeadline
		}
		gone, graceProbeErr := waitForGroupAbsence(controller, result.ProcessGroupID, graceDeadline)
		if graceProbeErr != nil {
			result.TeardownError = true
			result.OrphanRisk = true
			result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "TERM_GRACE_PROBE_FAILED")
		}
		if graceProbeErr == nil && !gone && killEscalationRequired {
			killErr := controller.signal(result.ProcessGroupID, syscall.SIGKILL)
			if killErr == nil {
				result.KillSent = true
			} else if !errors.Is(killErr, syscall.ESRCH) {
				result.TeardownError = true
				result.OrphanRisk = true
				result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "KILL_SIGNAL_FAILED")
			}
		}
		return
	}
	if !errors.Is(termErr, syscall.ESRCH) {
		result.TeardownError = true
		result.OrphanRisk = true
		result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "TERM_SIGNAL_FAILED")
	}
}

func resolvePreTermGroupProbe(controller darwinGroupController, processGroupID int, teardownDeadline time.Time, teardownBudget time.Duration) (bool, error) {
	return resolvePreTermGroupProbeWithClock(
		controller, processGroupID, teardownDeadline, teardownBudget, systemPreTermProbeClock{},
	)
}

func resolvePreTermGroupProbeWithClock(
	controller darwinGroupController,
	processGroupID int,
	teardownDeadline time.Time,
	teardownBudget time.Duration,
	clock preTermProbeClock,
) (bool, error) {
	present, err := controller.probe(processGroupID)
	if err == nil || !present || !errors.Is(err, syscall.EPERM) {
		return present, err
	}
	retryDeadline := clock.now().Add(teardownBudget / 4)
	if retryDeadline.After(teardownDeadline) {
		retryDeadline = teardownDeadline
	}
	for {
		remaining := retryDeadline.Sub(clock.now())
		if remaining <= 0 {
			return present, err
		}
		pause := time.Millisecond
		if pause > remaining {
			pause = remaining
		}
		clock.wait(pause)
		if !clock.now().Before(retryDeadline) {
			return present, err
		}
		nextPresent, nextErr := controller.probe(processGroupID)
		if !clock.now().Before(retryDeadline) {
			return present, err
		}
		present, err = nextPresent, nextErr
		if err == nil || !present || !errors.Is(err, syscall.EPERM) {
			return present, err
		}
	}
}

func classifyWait(result *Result, waited waitResult) {
	if waited.state != nil {
		result.ExitCode = waited.state.ExitCode()
		if status, ok := waited.state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			result.ExitSignal = status.Signal().String()
		}
	}
	if waited.err == nil {
		result.ChildWaited = true
		return
	}
	var exitError *exec.ExitError
	if errors.As(waited.err, &exitError) {
		result.ChildWaited = true
		return
	}
	result.WaitError = waited.err.Error()
	result.TeardownError = true
	result.OrphanRisk = true
	result.DiagnosticCode = firstDiagnostic(result.DiagnosticCode, "WAIT_FAILED")
}

func waitForDirectChild(waitC <-chan waitResult, deadline time.Time) *waitResult {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		select {
		case completed := <-waitC:
			return &completed
		default:
			return nil
		}
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case completed := <-waitC:
		return &completed
	case <-timer.C:
		return nil
	}
}

type drainCompletion struct{ observed, beforeDeadline bool }

func collectDrainCompletions(stdoutDone, stderrDone <-chan time.Time, deadline time.Time) (drainCompletion, drainCompletion) {
	var stdout, stderr drainCompletion
	for !stdout.observed || !stderr.observed {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			collectQueuedDrain(&stdout, stdoutDone, deadline)
			collectQueuedDrain(&stderr, stderrDone, deadline)
			return stdout, stderr
		}
		timer := time.NewTimer(remaining)
		select {
		case completed := <-stdoutDone:
			stdout = drainCompletion{observed: true, beforeDeadline: !drainDeadlineIsFailure || !completed.After(deadline)}
		case completed := <-stderrDone:
			stderr = drainCompletion{observed: true, beforeDeadline: !drainDeadlineIsFailure || !completed.After(deadline)}
		case <-timer.C:
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	return stdout, stderr
}

func collectQueuedDrain(status *drainCompletion, done <-chan time.Time, deadline time.Time) {
	if status.observed {
		return
	}
	select {
	case completed := <-done:
		*status = drainCompletion{observed: true, beforeDeadline: !drainDeadlineIsFailure || !completed.After(deadline)}
	default:
	}
}

func awaitOwnerDrain(done <-chan time.Time, budget time.Duration) bool {
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func waitForGroupAbsence(controller darwinGroupController, processGroupID int, deadline time.Time) (bool, error) {
	for {
		exists, err := controller.probe(processGroupID)
		if err != nil && !exists {
			return false, err
		}
		if !exists {
			return true, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false, err
		}
		pause := time.Millisecond
		if pause > remaining {
			pause = remaining
		}
		timer := time.NewTimer(pause)
		<-timer.C
	}
}

func performFinalGroupProbe(controller darwinGroupController, processGroupID int, deadline time.Time) (bool, error) {
	if !finalGroupProbeRequired {
		return true, nil
	}
	return waitForGroupAbsence(controller, processGroupID, deadline)
}
