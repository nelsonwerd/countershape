//go:build darwin

package http

import (
	"context"
	"errors"
	"io"
	"net"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type readinessRead struct {
	bytes []byte
	eof   bool
	err   error
}

type rawExchange struct {
	wire       []byte
	observed   int64
	written    int64
	complete   bool
	overflow   bool
	control    domain.ControlReason
	diagnostic string
}

func (running *runningService) Close() serviceResult {
	if running == nil {
		return serviceResult{}
	}
	running.once.Do(func() {
		running.result = running.close()
	})
	return cloneServiceResult(running.result)
}

func (running *runningService) close() serviceResult {
	result := running.initial
	var waited *waitObservation
	if result.process.primary == "" {
		readiness, earlyWait, control, diagnostic := awaitExactReadiness(
			running.processContext,
			running.readiness,
			time.Duration(running.config.budgets.ReadinessMS)*time.Millisecond,
			running.captures.stdout,
			running.captures.stderr,
			running.overflowC,
			running.waitC,
			running.config.readiness,
		)
		result.readiness.frameBytes = append([]byte(nil), readiness.bytes...)
		result.readiness.bytesObserved = int64(len(readiness.bytes))
		result.readiness.eofObserved = readiness.eof
		result.readiness.accepted = control == ""
		result.readiness.diagnosticCode = diagnostic
		waited = earlyWait
		if readiness.eof {
			frame, frameErr := httpmodel.ParseHTTPReadyPortFrame(readiness.bytes)
			if frameErr == nil && frame.Valid() {
				result.readiness.port = int(frame.Port())
				result.readiness.endpoint = "127.0.0.1:" + strconv.Itoa(result.readiness.port)
			}
		}
		if control != "" {
			result.process.primary = control
			result.process.diagnosticCode = diagnostic
		} else {
			wire, err := httpmodel.EncodeRequest(running.config.stimulus, result.readiness.port)
			if err != nil || !wire.Valid() {
				result.process.primary = domain.ControlReadinessError
				result.process.diagnosticCode = "HTTP_PORTABLE_ENDPOINT_ENCODING_FAILED"
				result.readiness.accepted = false
				result.readiness.diagnosticCode = result.process.diagnosticCode
			} else {
				result.exchange.requestWire = wire
			}
		}
	}
	if result.process.primary == "" {
		result.exchange.connectionAttempts = 1
		exchange := performOneExchange(
			running.processContext,
			result.readiness.endpoint,
			result.exchange.requestWire.Bytes(),
			running.config.capture.OwnerResponseReadLimit(),
			time.Duration(running.config.budgets.ProbeMS)*time.Millisecond,
		)
		result.exchange.requestWritten = exchange.written
		result.exchange.requestComplete = exchange.complete
		result.exchange.responseWire = append([]byte(nil), exchange.wire...)
		result.exchange.responseObserved = exchange.observed
		result.exchange.responseOverflow = exchange.overflow
		result.exchange.diagnosticCode = exchange.diagnostic
		switch {
		case running.captures.stdout.overflowed() || running.captures.stderr.overflowed():
			result.process.primary = domain.ControlOutputLimit
			result.process.diagnosticCode = "HTTP_PROCESS_OUTPUT_LIMIT"
		case running.processContext.Err() != nil:
			result.process.primary = domain.ControlCancelled
			result.process.diagnosticCode = "HTTP_EXCHANGE_CANCELLED"
		case exchange.control != "":
			result.process.primary = exchange.control
			result.process.diagnosticCode = exchange.diagnostic
		default:
			response, err := httpmodel.ParseResponse(exchange.wire, running.config.capture)
			if err != nil || !response.Valid() {
				result.process.primary, result.process.diagnosticCode = classifyResponseError(err)
				result.exchange.diagnosticCode = result.process.diagnosticCode
			} else {
				result.exchange.response = response
				result.exchange.responseParsed = true
			}
		}
	}
	running.closeProcess(&result, waited)
	return result
}

func (running *runningService) closeProcess(result *serviceResult, waited *waitObservation) {
	teardownBudget := time.Duration(running.config.budgets.TeardownMS) * time.Millisecond
	teardownDeadline := time.Now().Add(teardownBudget)
	if waited == nil {
		naturalDeadline := time.Now().Add(teardownBudget / 4)
		if naturalDeadline.After(teardownDeadline) {
			naturalDeadline = teardownDeadline
		}
		waited = waitForChild(running.waitC, naturalDeadline)
	}
	if result.process.processGroupOwned {
		teardownProcessGroup(&result.process, teardownDeadline, teardownBudget)
	}
	if waited == nil {
		waited = waitForChild(running.waitC, teardownDeadline)
	}
	if waited != nil {
		classifyWait(&result.process, *waited)
		applyNaturalExitControl(&result.process)
	} else {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_DIRECT_CHILD_WAIT_DEADLINE")
		_ = running.command.Process.Kill()
	}
	stdoutDrain, stderrDrain := collectDrains(running.stdoutDone, running.stderrDone, teardownDeadline)
	result.process.stdoutDrained = stdoutDrain.beforeDeadline
	result.process.stderrDrained = stderrDrain.beforeDeadline
	if !stdoutDrain.observed {
		_ = running.stdout.Close()
	}
	if !stderrDrain.observed {
		_ = running.stderr.Close()
	}
	if !stdoutDrain.observed && !awaitDrain(running.stdoutDone, ownerDrainBudget) {
		running.captures.stdout.freeze()
	}
	if !stderrDrain.observed && !awaitDrain(running.stderrDone, ownerDrainBudget) {
		running.captures.stderr.freeze()
	}
	if !result.process.stdoutDrained || !result.process.stderrDrained {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_PIPE_DRAIN_DEADLINE")
	}
	var err error
	result.process.stdout, result.process.stdoutObserved, result.process.stdoutOverflow, err = running.captures.stdout.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_STDOUT_DRAIN_FAILED")
	}
	result.process.stderr, result.process.stderrObserved, result.process.stderrOverflow, err = running.captures.stderr.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_STDERR_DRAIN_FAILED")
	}
	if result.process.primary == "" && (result.process.stdoutOverflow || result.process.stderrOverflow) {
		result.process.primary = domain.ControlOutputLimit
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_PROCESS_OUTPUT_LIMIT")
	}
	if result.process.processGroupOwned {
		clean, probeErr := waitForGroupAbsence(result.process.processGroupID, teardownDeadline)
		result.process.finalProbeClean = clean
		if probeErr != nil {
			result.process.finalProbeError = probeErr.Error()
			result.process.teardownError = true
			result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_FINAL_GROUP_PROBE_FAILED")
		}
		if !clean {
			result.process.orphanRisk = true
		}
	}
	recordRuntimeRevalidation(&result.process, validateExecutable(running.config.runtime))
	if result.process.started && (!result.process.processGroupOwned || !result.process.directChildWaited ||
		!result.process.stdoutDrained || !result.process.stderrDrained || !result.process.finalProbeClean) {
		result.process.orphanRisk = true
	}
	recordDescriptorCloseFailure(
		&result.process,
		closeFailureTerminalReaders,
		errors.Join(
			running.readiness.Close(),
			running.stdout.Close(),
			running.stderr.Close(),
		),
	)
}

func recordRuntimeRevalidation(result *processResult, err error) {
	if result == nil || err == nil {
		return
	}
	result.teardownError = true
	result.runtimeRevalidationError = err.Error()
	result.diagnosticCode = firstDiagnostic(
		result.diagnosticCode,
		"TOOL_CHANGED_AFTER_HTTP_EXECUTION",
	)
}

func awaitExactReadiness(
	ctx context.Context,
	reader io.ReadCloser,
	budget time.Duration,
	stdout, stderr *cappedCapture,
	overflowC <-chan struct{},
	waitC <-chan waitObservation,
	contract httpmodel.HTTPReadinessContract,
) (readinessRead, *waitObservation, domain.ControlReason, string) {
	_, readLimit, _, portable := contract.PortableFrameProfile()
	if !portable {
		return readinessRead{}, nil, domain.ControlReadinessError, "HTTP_READINESS_PROFILE_INVALID"
	}
	resultC := make(chan readinessRead, 1)
	go readExactReadiness(reader, readLimit+1, resultC)
	timer := time.NewTimer(budget)
	defer timer.Stop()
	deadlineObserved := false
	var waited *waitObservation
	var observed *readinessRead
	for {
		if stdout.overflowed() || stderr.overflowed() {
			return stopReadiness(reader, resultC, observed), waited, domain.ControlOutputLimit, "HTTP_PROCESS_OUTPUT_LIMIT_BEFORE_READINESS"
		}
		if ctx.Err() != nil {
			return stopReadiness(reader, resultC, observed), waited, domain.ControlCancelled, "HTTP_READINESS_CANCELLED"
		}
		deadlineObserved, waited, observed = latchReadinessFacts(
			timer.C, waitC, resultC, deadlineObserved, waited, observed,
		)
		if deadlineObserved {
			return stopReadiness(reader, resultC, observed), waited, domain.ControlReadinessError, "HTTP_READINESS_TIMEOUT"
		}
		if waited != nil {
			return finishReadinessAfterExit(reader, resultC, observed), waited, domain.ControlReadinessError, "HTTP_PROCESS_EXITED_BEFORE_READINESS"
		}
		if observed != nil {
			frame, err := httpmodel.ParseHTTPReadyPortFrame(observed.bytes)
			if observed.err != nil || !observed.eof || err != nil || !frame.Valid() {
				return *observed, waited, domain.ControlReadinessError, "HTTP_READINESS_PROTOCOL_REJECTED"
			}
			return *observed, waited, "", ""
		}
		select {
		case <-overflowC:
		case <-ctx.Done():
		case <-timer.C:
			deadlineObserved = true
		case completed := <-waitC:
			waited = &completed
		case completed := <-resultC:
			observed = &completed
		}
	}
}

func latchReadinessFacts(
	deadlineC <-chan time.Time,
	waitC <-chan waitObservation,
	resultC <-chan readinessRead,
	deadlineObserved bool,
	waited *waitObservation,
	observed *readinessRead,
) (bool, *waitObservation, *readinessRead) {
	if !deadlineObserved {
		select {
		case <-deadlineC:
			deadlineObserved = true
		default:
		}
	}
	if waited == nil {
		select {
		case completed := <-waitC:
			waited = &completed
		default:
		}
	}
	if observed == nil {
		select {
		case completed := <-resultC:
			observed = &completed
		default:
		}
	}
	return deadlineObserved, waited, observed
}

func finishReadinessAfterExit(
	reader io.ReadCloser,
	resultC <-chan readinessRead,
	observed *readinessRead,
) readinessRead {
	if observed != nil {
		return *observed
	}
	timer := time.NewTimer(ownerDrainBudget)
	defer timer.Stop()
	select {
	case result := <-resultC:
		return result
	case <-timer.C:
		return stopReadiness(reader, resultC, nil)
	}
}

func stopReadiness(reader io.ReadCloser, resultC <-chan readinessRead, observed *readinessRead) readinessRead {
	if observed != nil {
		return *observed
	}
	_ = reader.Close()
	timer := time.NewTimer(ownerDrainBudget)
	defer timer.Stop()
	select {
	case result := <-resultC:
		return result
	case <-timer.C:
		return readinessRead{}
	}
}

func readExactReadiness(reader io.Reader, limit int, resultC chan<- readinessRead) {
	if limit < 2 {
		limit = 2
	}
	result := readinessRead{bytes: make([]byte, 0, limit)}
	buffer := make([]byte, 1)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			result.bytes = append(result.bytes, buffer[:count]...)
			if len(result.bytes) >= limit {
				resultC <- result
				return
			}
		}
		if errors.Is(err, io.EOF) {
			result.eof = true
			resultC <- result
			return
		}
		if err != nil {
			result.err = err
			resultC <- result
			return
		}
	}
}

func performOneExchange(
	ctx context.Context,
	endpoint string,
	requestWire []byte,
	responseLimit int64,
	budget time.Duration,
) rawExchange {
	result := rawExchange{}
	probeContext, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(probeContext, "tcp4", endpoint)
	if err != nil {
		return classifyExchangeError(probeContext, result, err, "HTTP_CONNECT_FAILED")
	}
	defer connection.Close()
	deadline, present := probeContext.Deadline()
	if !present {
		result.control = domain.ControlProbeTransportError
		result.diagnostic = "HTTP_PROBE_DEADLINE_MISSING"
		return result
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return classifyExchangeError(probeContext, result, err, "HTTP_DEADLINE_INSTALL_FAILED")
	}
	stopWatch := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-probeContext.Done():
			_ = connection.SetDeadline(time.Now())
		case <-stopWatch:
		}
	}()
	defer func() {
		close(stopWatch)
		<-watchDone
	}()
	for result.written < int64(len(requestWire)) {
		count, writeErr := connection.Write(requestWire[result.written:])
		result.written += int64(count)
		if writeErr != nil {
			return classifyExchangeError(probeContext, result, writeErr, "HTTP_REQUEST_WRITE_FAILED")
		}
		if count == 0 {
			result.control = domain.ControlProbeTransportError
			result.diagnostic = "HTTP_REQUEST_WRITE_NO_PROGRESS"
			return result
		}
	}
	result.complete = true
	tcp, ok := connection.(*net.TCPConn)
	if !ok {
		result.control = domain.ControlProbeTransportError
		result.diagnostic = "HTTP_NON_TCP_CONNECTION"
		return result
	}
	if err := tcp.CloseWrite(); err != nil {
		return classifyExchangeError(probeContext, result, err, "HTTP_REQUEST_HALF_CLOSE_FAILED")
	}
	limited := &io.LimitedReader{R: connection, N: responseLimit}
	response, readErr := io.ReadAll(limited)
	result.wire = append([]byte(nil), response...)
	result.observed = int64(len(response))
	result.overflow = result.observed == responseLimit
	if result.overflow {
		result.control = domain.ControlOutputLimit
		result.diagnostic = "HTTP_RESPONSE_LIMIT"
		return result
	}
	if readErr != nil {
		return classifyExchangeError(probeContext, result, readErr, "HTTP_RESPONSE_READ_FAILED")
	}
	return result
}

func classifyExchangeError(
	ctx context.Context,
	result rawExchange,
	err error,
	diagnostic string,
) rawExchange {
	result.diagnostic = diagnostic
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.control = domain.ControlTimeout
		result.diagnostic = "HTTP_PROBE_TIMEOUT"
		return result
	}
	if ctx.Err() != nil {
		result.control = domain.ControlCancelled
		result.diagnostic = "HTTP_PROBE_CANCELLED"
		return result
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && networkErr.Timeout() {
		result.control = domain.ControlTimeout
		result.diagnostic = "HTTP_PROBE_TIMEOUT"
		return result
	}
	result.control = domain.ControlProbeTransportError
	return result
}

func classifyResponseError(err error) (domain.ControlReason, string) {
	code, ok := httpmodel.RefusalCodeOf(err)
	if ok {
		switch code {
		case httpmodel.CodeResponseStatusLimit, httpmodel.CodeResponseHeaderLimit, httpmodel.CodeResponseBodyLimit:
			return domain.ControlOutputLimit, string(code)
		default:
			return domain.ControlProbeTransportError, string(code)
		}
	}
	return domain.ControlProbeTransportError, "HTTP_RESPONSE_PROTOCOL_REJECTED"
}

func teardownProcessGroup(result *processResult, deadline time.Time, budget time.Duration) {
	present, err := resolvePreTermProbe(result.processGroupID, deadline, budget)
	if err != nil {
		result.preTermProbe = preTermUncertain
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "PRE_TERM_GROUP_PROBE_FAILED")
		return
	}
	if !present {
		result.preTermProbe = preTermAbsent
		return
	}
	result.preTermProbe = preTermPresent
	if err := syscall.Kill(-result.processGroupID, syscall.SIGTERM); err == nil {
		result.termSent = true
		graceDeadline := time.Now().Add(budget / 3)
		if graceDeadline.After(deadline) {
			graceDeadline = deadline
		}
		gone, probeErr := waitForGroupAbsence(result.processGroupID, graceDeadline)
		if probeErr != nil {
			result.teardownError = true
			result.orphanRisk = true
			result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TERM_GRACE_PROBE_FAILED")
		}
		if probeErr == nil && !gone {
			if killErr := syscall.Kill(-result.processGroupID, syscall.SIGKILL); killErr == nil {
				result.killSent = true
			} else if !errors.Is(killErr, syscall.ESRCH) {
				result.teardownError = true
				result.orphanRisk = true
				result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "KILL_SIGNAL_FAILED")
			}
		}
	} else if !errors.Is(err, syscall.ESRCH) {
		result.teardownError = true
		result.orphanRisk = true
		result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "TERM_SIGNAL_FAILED")
	}
}

func resolvePreTermProbe(processGroupID int, deadline time.Time, budget time.Duration) (bool, error) {
	present, err := probeGroup(processGroupID)
	if err == nil || !present || !errors.Is(err, syscall.EPERM) {
		return present, err
	}
	retryDeadline := time.Now().Add(budget / 4)
	if retryDeadline.After(deadline) {
		retryDeadline = deadline
	}
	for time.Now().Before(retryDeadline) {
		time.Sleep(time.Millisecond)
		nextPresent, nextErr := probeGroup(processGroupID)
		if nextErr == nil || !nextPresent || !errors.Is(nextErr, syscall.EPERM) {
			return nextPresent, nextErr
		}
		present, err = nextPresent, nextErr
	}
	return present, err
}

func probeGroup(processGroupID int) (bool, error) {
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

func waitForGroupAbsence(processGroupID int, deadline time.Time) (bool, error) {
	for {
		present, err := probeGroup(processGroupID)
		if err != nil && !present {
			return false, err
		}
		if !present {
			return true, nil
		}
		if !time.Now().Before(deadline) {
			return false, err
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForChild(waitC <-chan waitObservation, deadline time.Time) *waitObservation {
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

func classifyWait(result *processResult, waited waitObservation) {
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

func applyNaturalExitControl(result *processResult) {
	if result == nil || result.primary != "" || !result.directChildWaited || result.termSent || result.killSent {
		return
	}
	if result.exitCode == 0 && result.exitSignal == "" {
		return
	}
	result.primary = domain.ControlProbeTransportError
	result.diagnosticCode = firstDiagnostic(result.diagnosticCode, "HTTP_SERVICE_NONZERO_EXIT")
}

type drainCompletion struct {
	observed       bool
	beforeDeadline bool
}

func collectDrains(stdoutDone, stderrDone <-chan time.Time, deadline time.Time) (drainCompletion, drainCompletion) {
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
			stdout = drainCompletion{observed: true, beforeDeadline: !completed.After(deadline)}
		case completed := <-stderrDone:
			stderr = drainCompletion{observed: true, beforeDeadline: !completed.After(deadline)}
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
		*status = drainCompletion{observed: true, beforeDeadline: !completed.After(deadline)}
	default:
	}
}

func awaitDrain(done <-chan time.Time, budget time.Duration) bool {
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func firstDiagnostic(existing, next string) string {
	if existing != "" {
		return existing
	}
	return next
}

func cloneServiceResult(source serviceResult) serviceResult {
	source.process.stdout = append([]byte(nil), source.process.stdout...)
	source.process.stderr = append([]byte(nil), source.process.stderr...)
	source.readiness.frameBytes = append([]byte(nil), source.readiness.frameBytes...)
	source.exchange.responseWire = append([]byte(nil), source.exchange.responseWire...)
	source.environment = append([]string(nil), source.environment...)
	return source
}
