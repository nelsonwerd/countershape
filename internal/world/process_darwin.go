//go:build darwin

package world

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
)

const (
	signalOwnedProcessGroup        = true
	killEscalationRequired         = true
	finalGroupProbeRequired        = true
	outputOverflowIsPrimaryControl = true
	drainDeadlineIsFailure         = true
	ownerPriorityOutputFirst       = true
	ownerDrainCloseBudget          = 100 * time.Millisecond
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

type stdinWriterStart struct {
	declared  int64
	digest    domain.Digest
	errorCode string
}

type terminalDecision struct {
	primary domain.ControlReason
	waited  *waitResult
	stdin   *stdinWriteResult
}

type physicalCapturePair struct {
	stdout             *cappedCapture
	stderr             *cappedCapture
	stdoutReceiptLimit int64
	stderrReceiptLimit int64
}

func newPhysicalCapturePair(stdoutLimit, stderrLimit int64, overflowC chan<- struct{}) physicalCapturePair {
	stdout := newCappedCapture(stdoutLimit, overflowC)
	stderr := newCappedCapture(stderrLimit, overflowC)
	return physicalCapturePair{
		stdout: stdout, stderr: stderr,
		stdoutReceiptLimit: stdout.configuredLimit(), stderrReceiptLimit: stderr.configuredLimit(),
	}
}

type darwinGroupController interface {
	signal(processGroupID int, signal syscall.Signal) error
	probe(processGroupID int) (bool, error)
}

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

type systemDarwinGroups struct{}

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

func runPlatformProcess(ctx context.Context, request processRequest) physicalProcessResult {
	stdinDigest, stdinDigestErr := digestProcessStdin(request.stdin)
	base := physicalProcessResult{
		exitCode: -1, markerBeforeSpawn: request.markerBeforeSpawn,
		preTermProbe:  preTermProbeNotApplicable,
		stdinPresence: request.stdin.presence, stdinDeclared: int64(len(request.stdin.bytes)),
		stdinDigest: stdinDigest, stdinComplete: request.stdin.presence != processStdinPresent,
	}
	if stdinDigestErr != nil {
		base.primary = domain.ControlStartError
		base.diagnosticCode = "STDIN_IDENTITY_FAILED_BEFORE_SPAWN"
		return base
	}
	if ctx.Err() != nil {
		base.primary = domain.ControlCancelled
		base.diagnosticCode = "CONTEXT_CANCELLED_BEFORE_SPAWN"
		return base
	}
	if !request.stdin.valid() {
		base.primary = domain.ControlStartError
		base.diagnosticCode = "INVALID_TYPED_STDIN_BEFORE_SPAWN"
		return base
	}
	if err := request.tool.revalidate(); err != nil {
		base.primary = domain.ControlStartError
		base.waitError = err.Error()
		base.diagnosticCode = "TOOL_CHANGED_BEFORE_SPAWN"
		return base
	}
	stdin := processmechanics.AbsentStdin()
	if request.stdin.presence == processStdinPresent {
		stdin = processmechanics.PresentStdin(request.stdin.bytes)
	}
	invocation, err := processmechanics.NewInvocation(
		request.tool.absolutePath,
		request.logicalArgv,
		request.environment,
		stdin,
		request.cwd,
		processmechanics.Limits{
			StdoutBytes: request.stdoutLimit,
			StderrBytes: request.stderrLimit,
			Execution:   time.Duration(request.executionBudgetMS) * time.Millisecond,
			Teardown:    time.Duration(request.teardownBudgetMS) * time.Millisecond,
		},
	)
	if err != nil {
		base.primary = domain.ControlStartError
		base.waitError = err.Error()
		base.diagnosticCode = "MECHANICS_INVOCATION_REJECTED"
		return base
	}
	prepared, err := processmechanics.Prepare(invocation)
	if err != nil {
		base.primary = domain.ControlStartError
		base.waitError = err.Error()
		base.diagnosticCode = "MECHANICS_PREPARE_FAILED"
		return base
	}
	observation, running, err := prepared.Start(ctx)
	if err != nil {
		var failure *processmechanics.StartError
		if errors.As(err, &failure) {
			return bindMechanicsResult(base, failure.Result())
		}
		base.primary = domain.ControlStartError
		base.waitError = err.Error()
		base.diagnosticCode = "MECHANICS_START_FAILED"
		return base
	}
	if observation.ProcessGroupOwned && observation.ProcessGroupID == observation.PID && request.onGroupOwned != nil {
		if err := request.onGroupOwned(); err != nil {
			running.AbortReadinessTransition()
		}
	}
	result := bindMechanicsResult(base, running.Close())
	if err := request.tool.revalidate(); err != nil {
		result.teardownError = true
		result.waitError = err.Error()
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TOOL_CHANGED_AFTER_EXECUTION")
	}
	return result
}

func bindMechanicsResult(base physicalProcessResult, result processmechanics.Result) physicalProcessResult {
	base.physicalExecutionEntered = result.PhysicalExecutionEntered
	base.spawnAttempted = result.SpawnAttempted
	base.started = result.Started
	base.pid = result.PID
	base.processGroupID = result.ProcessGroupID
	base.processGroupOwned = result.ProcessGroupOwned
	base.exitCode = result.ExitCode
	base.exitSignal = result.ExitSignal
	base.waitError = result.WaitError
	base.stdout = append([]byte(nil), result.Stdout...)
	base.stderr = append([]byte(nil), result.Stderr...)
	base.stdoutObserved = result.StdoutObserved
	base.stderrObserved = result.StderrObserved
	base.stdoutOverflow = result.StdoutOverflow
	base.stderrOverflow = result.StderrOverflow
	base.stdoutCaptureLimit = result.StdoutCaptureLimit
	base.stderrCaptureLimit = result.StderrCaptureLimit
	base.primary = domain.ControlReason(result.Primary)
	base.preTermProbe = result.PreTermProbe
	base.termSent = result.TermSent
	base.killSent = result.KillSent
	base.directChildWaited = result.ChildWaited
	base.stdoutDrained = result.StdoutDrained
	base.stderrDrained = result.StderrDrained
	base.finalProbeClean = result.FinalProbeClean
	base.finalProbeError = result.FinalProbeError
	base.teardownError = result.TeardownError
	base.orphanRisk = result.OrphanRisk
	base.diagnosticCode = result.DiagnosticCode
	if base.diagnosticCode == "EXECUTABLE_CHANGED_BEFORE_SPAWN" {
		base.diagnosticCode = "TOOL_CHANGED_BEFORE_SPAWN"
	}
	if base.diagnosticCode == "EXECUTABLE_CHANGED_AFTER_EXECUTION" {
		base.diagnosticCode = "TOOL_CHANGED_AFTER_EXECUTION"
	}
	base.stdinPipeAllocated = result.StdinPipeAllocated
	base.stdinWriterStarted = result.StdinWriterStarted
	base.stdinHandoffAttempted = result.StdinHandoffAttempted
	base.stdinWritten = result.StdinWritten
	base.stdinComplete = result.StdinComplete
	base.stdinErrorCode = result.StdinErrorCode
	return base
}

// The remaining world-owned helpers stay here for HTTP lifecycle parity in C5;
// C4 moves only the shared CLI process mechanics behind processmechanics.
func startExactStdinWriter(writer io.WriteCloser, input []byte, resultC chan<- stdinWriteResult) stdinWriterStart {
	started := make(chan stdinWriterStart, 1)
	go writeExactStdinObserved(writer, input, started, resultC)
	return <-started
}

func writeExactStdin(writer io.WriteCloser, input []byte, resultC chan<- stdinWriteResult) {
	writeExactStdinObserved(writer, input, nil, resultC)
}

func writeExactStdinObserved(
	writer io.WriteCloser,
	input []byte,
	started chan<- stdinWriterStart,
	resultC chan<- stdinWriteResult,
) {
	digest, err := digestProcessStdin(processStdin{presence: processStdinPresent, bytes: input})
	start := stdinWriterStart{declared: int64(len(input)), digest: digest}
	if err != nil {
		start.errorCode = "IDENTITY_ERROR"
	}
	if started != nil {
		started <- start
	}
	if start.errorCode != "" {
		_ = writer.Close()
		resultC <- stdinWriteResult{errorCode: start.errorCode}
		return
	}
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

func collectStdinDelivery(
	observed *stdinWriteResult,
	resultC <-chan stdinWriteResult,
	writer io.WriteCloser,
	deadline time.Time,
) stdinWriteResult {
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

func teardownOwnedProcessGroup(
	result *physicalProcessResult,
	controller darwinGroupController,
	teardownDeadline time.Time,
	teardownBudget time.Duration,
) {
	// The probe must be the immediately preceding kernel observation. If the
	// original group is already absent, signaling its numeric PGID would only
	// increase the chance of hitting an unrelated reused group.
	present, probeErr := resolvePreTermGroupProbe(
		controller,
		result.processGroupID,
		teardownDeadline,
		teardownBudget,
		systemPreTermProbeClock{},
	)
	if probeErr != nil {
		result.preTermProbe = preTermProbeUncertain
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "PRE_TERM_GROUP_PROBE_FAILED")
		return
	}
	if !present {
		result.preTermProbe = preTermProbeAbsent
		return
	}
	result.preTermProbe = preTermProbePresent

	// This probe/signal pair is necessarily non-atomic on Darwin. The residual
	// PGID-reuse race is carried by processGroupReuseExclusion in every receipt.
	termErr := controller.signal(result.processGroupID, syscall.SIGTERM)
	if termErr == nil {
		result.termSent = true
		graceDeadline := time.Now().Add(teardownBudget / 3)
		if graceDeadline.After(teardownDeadline) {
			graceDeadline = teardownDeadline
		}
		gone, graceProbeErr := waitForGroupAbsence(controller, result.processGroupID, graceDeadline)
		if graceProbeErr != nil {
			result.teardownError = true
			result.orphanRisk = true
			result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TERM_GRACE_PROBE_FAILED")
		}
		if graceProbeErr == nil && !gone && killEscalationRequired {
			killErr := controller.signal(result.processGroupID, syscall.SIGKILL)
			if killErr == nil {
				result.killSent = true
			} else if !errors.Is(killErr, syscall.ESRCH) {
				result.teardownError = true
				result.orphanRisk = true
				result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "KILL_SIGNAL_FAILED")
			}
		}
		return
	}
	if !errors.Is(termErr, syscall.ESRCH) {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TERM_SIGNAL_FAILED")
	}
}

func resolvePreTermGroupProbe(
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

	// Darwin can transiently return EPERM for an owned group containing only
	// zombies. It proves numeric presence but is not authority to signal. Spend
	// at most one quarter of the already-declared teardown budget resolving it,
	// while retaining the majority for TERM/KILL, drains, and final absence.
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

// arbitrateTerminal is deliberately owner-observed rather than a claim about
// unknowable kernel event time. Every wake returns to fixed priority polling,
// so Go select randomness cannot choose among facts already visible to owner.
func arbitrateTerminal(
	ctx context.Context,
	deadlineC <-chan time.Time,
	overflowC <-chan struct{},
	stdoutCapture, stderrCapture *cappedCapture,
	waitC <-chan waitResult,
) terminalDecision {
	return arbitrateTerminalWithStdin(ctx, deadlineC, overflowC, stdoutCapture, stderrCapture, waitC, nil)
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
		// Stdin transport is higher priority than cancellation, deadline, and
		// child exit. Latch an already-ready writer result before the owner
		// observes those lower-priority terminal facts.
		if stdinC != nil && stdinObserved == nil {
			select {
			case completed := <-stdinC:
				stdinObserved = &completed
			default:
			}
		}
		decision, ready, observed := pollOwnerTerminalPriorityWithStdin(
			ctx, deadlineC, stdoutCapture, stderrCapture, deadlineObserved, waited, stdinObserved,
		)
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

func pollOwnerTerminalPriority(
	ctx context.Context,
	deadlineC <-chan time.Time,
	stdoutCapture, stderrCapture *cappedCapture,
	deadlineObserved bool,
	waited *waitResult,
) (terminalDecision, bool, bool) {
	return pollOwnerTerminalPriorityWithStdin(
		ctx, deadlineC, stdoutCapture, stderrCapture, deadlineObserved, waited, nil,
	)
}

func pollOwnerTerminalPriorityWithStdin(
	ctx context.Context,
	deadlineC <-chan time.Time,
	stdoutCapture, stderrCapture *cappedCapture,
	deadlineObserved bool,
	waited *waitResult,
	stdin *stdinWriteResult,
) (terminalDecision, bool, bool) {
	if !ownerPriorityOutputFirst && ctx.Err() != nil {
		return terminalDecision{primary: domain.ControlCancelled, stdin: stdin}, true, deadlineObserved
	}
	if outputOverflowIsPrimaryControl && (stdoutCapture.overflowed() || stderrCapture.overflowed()) {
		return terminalDecision{primary: domain.ControlOutputLimit, stdin: stdin}, true, deadlineObserved
	}
	if stdin != nil && !stdin.complete {
		return terminalDecision{primary: domain.ControlProbeTransportError, stdin: stdin}, true, deadlineObserved
	}
	if ctx.Err() != nil {
		return terminalDecision{primary: domain.ControlCancelled, stdin: stdin}, true, deadlineObserved
	}
	if !deadlineObserved {
		select {
		case <-deadlineC:
			deadlineObserved = true
		default:
		}
	}
	if deadlineObserved {
		return terminalDecision{primary: domain.ControlTimeout, stdin: stdin}, true, true
	}
	if waited != nil {
		return terminalDecision{waited: waited, stdin: stdin}, true, false
	}
	return terminalDecision{}, false, false
}

func firstDiagnostic(existing, next string) string {
	if existing != "" {
		return existing
	}
	return next
}

func classifyWait(result *physicalProcessResult, waited waitResult) {
	if waited.state != nil {
		result.exitCode = waited.state.ExitCode()
		if status, ok := waited.state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			result.exitSignal = status.Signal().String()
		}
	}
	if waited.err == nil {
		result.directChildWaited = true
		return
	}
	var exitError *exec.ExitError
	if errors.As(waited.err, &exitError) {
		result.directChildWaited = true
		return
	}
	result.waitError = waited.err.Error()
	result.teardownError = true
	result.orphanRisk = true
	result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "WAIT_FAILED")
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

type drainCompletion struct {
	observed       bool
	beforeDeadline bool
}

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
			// Darwin can transiently report EPERM for a group containing only
			// zombies. That still proves numeric presence, so it is retryable; if
			// it remains the final observation, retain the uncertainty and never
			// authorize blind KILL escalation.
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
