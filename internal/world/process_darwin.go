//go:build darwin

package world

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	directExecOnly                 = true // MUTANT_U2_SPAWN_THROUGH_SHELL
	signalOwnedProcessGroup        = true // MUTANT_U2_SIGNAL_DIRECT_PID
	killEscalationRequired         = true // MUTANT_U2_REMOVE_KILL_ESCALATION
	finalGroupProbeRequired        = true // MUTANT_U2_REMOVE_FINAL_GROUP_PROBE
	outputOverflowIsPrimaryControl = true // MUTANT_U2_MAP_OUTPUT_LIMIT_TO_SUCCESS
	drainDeadlineIsFailure         = true // MUTANT_U2_REPORT_DRAIN_TIMEOUT_COMPLETE
	ownerPriorityOutputFirst       = true // MUTANT_U2_LATE_CONTROL_OVERRIDES_OUTPUT
	ownerDrainCloseBudget          = 100 * time.Millisecond
)

type waitResult struct {
	state *os.ProcessState
	err   error
}

type terminalDecision struct {
	primary domain.ControlReason
	waited  *waitResult
}

type darwinGroupController interface {
	signal(processGroupID int, signal syscall.Signal) error
	probe(processGroupID int) (bool, error)
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

func newDirectCommand(request processRequest) *exec.Cmd {
	args := append([]string(nil), request.logicalArgv...)
	if !directExecOnly {
		return &exec.Cmd{
			Path: "/bin/sh", Args: []string{"sh", "-c", strings.Join(args, " ")},
			Env: append([]string(nil), request.environment...), Dir: request.cwd,
			SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
		}
	}
	return &exec.Cmd{
		Path: request.tool.absolutePath, Args: args,
		Env: append([]string(nil), request.environment...), Dir: request.cwd,
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
	}
}

func runPlatformProcess(ctx context.Context, request processRequest) physicalProcessResult {
	result := physicalProcessResult{
		exitCode: -1, markerBeforeSpawn: request.markerBeforeSpawn,
		preTermProbe: preTermProbeNotApplicable,
	}
	if ctx.Err() != nil {
		result.primary = domain.ControlCancelled
		result.diagnosticCode = "CONTEXT_CANCELLED_BEFORE_SPAWN"
		return result
	}
	if err := request.tool.revalidate(); err != nil {
		result.primary = domain.ControlStartError
		result.waitError = err.Error()
		result.diagnosticCode = "TOOL_CHANGED_BEFORE_SPAWN"
		return result
	}
	command := newDirectCommand(request)
	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		result.primary = domain.ControlStartError
		result.diagnosticCode = "STDOUT_PIPE_ALLOCATION_FAILED"
		return result
	}
	defer stdoutPipe.Close()
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutWriter.Close()
		result.primary = domain.ControlStartError
		result.diagnosticCode = "STDERR_PIPE_ALLOCATION_FAILED"
		return result
	}
	defer stderrPipe.Close()
	command.Stdout = stdoutWriter
	command.Stderr = stderrWriter
	result.spawnAttempted = true
	if err := command.Start(); err != nil {
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		result.primary = domain.ControlStartError
		result.diagnosticCode = "SPAWN_FAILED"
		return result
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	result.started = true
	result.pid = command.Process.Pid
	processGroupID, groupErr := syscall.Getpgid(command.Process.Pid)
	result.processGroupID = processGroupID
	if groupErr == nil && processGroupID == command.Process.Pid {
		result.processGroupOwned = true
		if request.onGroupOwned != nil {
			if err := request.onGroupOwned(); err != nil {
				result.primary = domain.ControlStartError
				result.teardownError = true
				result.diagnosticCode = "READY_TRANSITION_FAILED"
				_ = command.Process.Kill()
			}
		}
	} else {
		// Setpgid is only a request. Without the exact PGID==PID measurement,
		// negative-PGID signaling is forbidden and orphan uncertainty survives.
		result.primary = domain.ControlStartError
		result.teardownError = true
		result.orphanRisk = true
		result.finalProbeError = "new process group was not established"
		result.diagnosticCode = "PROCESS_GROUP_NOT_OWNED"
		_ = command.Process.Kill()
	}

	overflowC := make(chan struct{}, 2)
	stdoutCapture := newCappedCapture(request.stdoutLimit, overflowC)
	stderrCapture := newCappedCapture(request.stderrLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP
	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	go stdoutCapture.drain(stdoutPipe, stdoutDone)
	go stderrCapture.drain(stderrPipe, stderrDone)
	waitC := make(chan waitResult, 1)
	go func() {
		waitErr := command.Wait()
		waitC <- waitResult{state: command.ProcessState, err: waitErr}
	}()

	var waited *waitResult
	if result.primary == "" {
		executionTimer := time.NewTimer(time.Duration(request.executionBudgetMS) * time.Millisecond)
		decision := arbitrateTerminal(ctx, executionTimer.C, overflowC, stdoutCapture, stderrCapture, waitC)
		if !executionTimer.Stop() {
			select {
			case <-executionTimer.C:
			default:
			}
		}
		result.primary = decision.primary
		waited = decision.waited
	}

	teardownDeadline := time.Now().Add(time.Duration(request.teardownBudgetMS) * time.Millisecond)
	controller := systemDarwinGroups{}
	if result.processGroupOwned {
		teardownOwnedProcessGroup(
			&result, controller, teardownDeadline,
			time.Duration(request.teardownBudgetMS)*time.Millisecond,
		)
	}

	if waited == nil {
		waited = waitForDirectChild(waitC, teardownDeadline)
	}
	if waited != nil {
		classifyWait(&result, *waited)
	} else {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "DIRECT_CHILD_WAIT_DEADLINE")
		_ = command.Process.Kill()
	}

	stdoutDrain, stderrDrain := collectDrainCompletions(stdoutDone, stderrDone, teardownDeadline)
	result.stdoutDrained = stdoutDrain.beforeDeadline
	result.stderrDrained = stderrDrain.beforeDeadline
	if !stdoutDrain.observed {
		_ = stdoutPipe.Close()
	}
	if !stderrDrain.observed {
		_ = stderrPipe.Close()
	}
	if !stdoutDrain.observed && !awaitOwnerDrain(stdoutDone, ownerDrainCloseBudget) {
		stdoutCapture.freeze()
	}
	if !stderrDrain.observed && !awaitOwnerDrain(stderrDone, ownerDrainCloseBudget) {
		stderrCapture.freeze()
	}
	if !result.stdoutDrained || !result.stderrDrained {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "PIPE_DRAIN_DEADLINE")
	}
	result.stdout, result.stdoutObserved, result.stdoutOverflow, err = stdoutCapture.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "STDOUT_DRAIN_FAILED")
	}
	result.stderr, result.stderrObserved, result.stderrOverflow, err = stderrCapture.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "STDERR_DRAIN_FAILED")
	}
	applyCompletedOutputControl(&result)

	if result.processGroupOwned {
		clean, probeErr := performFinalGroupProbe(controller, result.processGroupID, teardownDeadline)
		result.finalProbeClean = clean
		if probeErr != nil {
			result.finalProbeError = probeErr.Error()
			result.teardownError = true
			result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "FINAL_GROUP_PROBE_FAILED")
		}
		if !clean {
			result.orphanRisk = true
		}
	}
	if err := request.tool.revalidate(); err != nil {
		result.teardownError = true
		result.waitError = err.Error()
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TOOL_CHANGED_AFTER_EXECUTION")
	}
	if result.started && (!result.processGroupOwned || !result.directChildWaited ||
		!result.stdoutDrained || !result.stderrDrained || !result.finalProbeClean) {
		// A clean receipt requires every independently observed cleanup edge.
		// Missing pipe completion is especially important: an escaped descendant
		// can keep inherited descriptors open after the original PGID is absent.
		result.orphanRisk = true
	}
	return result
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
	present, probeErr := controller.probe(result.processGroupID)
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
	deadlineObserved := false
	var waited *waitResult
	for {
		decision, ready, observed := pollOwnerTerminalPriority(
			ctx, deadlineC, stdoutCapture, stderrCapture, deadlineObserved, waited,
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
	if !ownerPriorityOutputFirst && ctx.Err() != nil {
		return terminalDecision{primary: domain.ControlCancelled}, true, deadlineObserved
	}
	if outputOverflowIsPrimaryControl && (stdoutCapture.overflowed() || stderrCapture.overflowed()) {
		return terminalDecision{primary: domain.ControlOutputLimit}, true, deadlineObserved
	}
	if ctx.Err() != nil {
		return terminalDecision{primary: domain.ControlCancelled}, true, deadlineObserved
	}
	if !deadlineObserved {
		select {
		case <-deadlineC:
			deadlineObserved = true
		default:
		}
	}
	if deadlineObserved {
		return terminalDecision{primary: domain.ControlTimeout}, true, true
	}
	if waited != nil {
		return terminalDecision{waited: waited}, true, false
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
