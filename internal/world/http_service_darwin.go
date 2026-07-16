//go:build darwin

package world

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type httpReadinessRead struct {
	bytes []byte
	eof   bool
	err   error
}

type httpExchangeResult struct {
	wire       []byte
	observed   int64
	written    int64
	complete   bool
	overflow   bool
	control    domain.ControlReason
	diagnostic string
}

func runPlatformLiveHTTPService(ctx context.Context, request httpServiceRequest) liveHTTPServiceResult {
	result := liveHTTPServiceResult{process: physicalProcessResult{
		exitCode: -1, markerBeforeSpawn: request.markerBeforeSpawn,
		preTermProbe: preTermProbeNotApplicable,
	}}
	if ctx == nil || !request.binding.Valid() || len(request.logicalArgv) == 0 ||
		request.onReadinessAccepted == nil || request.onResponseCaptured == nil ||
		request.responseLimit < 1 || request.stdoutLimit < 1 || request.stderrLimit < 1 ||
		request.readinessBudgetMS < 1 || request.probeBudgetMS < 1 || request.teardownBudgetMS < 1 {
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_SERVICE_REQUEST_INVALID_BEFORE_SPAWN"
		return result
	}
	if ctx.Err() != nil {
		result.process.primary = domain.ControlCancelled
		result.process.diagnosticCode = "CONTEXT_CANCELLED_BEFORE_HTTP_SPAWN"
		return result
	}
	if err := request.tool.revalidate(); err != nil {
		result.process.primary = domain.ControlStartError
		result.process.waitError = err.Error()
		result.process.diagnosticCode = "TOOL_CHANGED_BEFORE_HTTP_SPAWN"
		return result
	}

	portable := request.binding.StartSpec().Authority() == httpmodel.HTTPPortableStartAuthorityV1
	var listenerFile *os.File
	var endpoint string
	var port int
	if portable {
		result.readiness.protocol = httpmodel.PortableReadinessProtocolV1
		result.readiness.listenerFD = 0
		result.readiness.readinessFD = httpPortableReadinessChildFD
	} else {
		listener, inheritedFile, inheritedEndpoint, inheritedPort, err := allocateInheritedLoopbackListener()
		if err != nil {
			result.process.primary = domain.ControlStartError
			result.process.diagnosticCode = "HTTP_LOOPBACK_LISTENER_ALLOCATION_FAILED"
			return result
		}
		_ = listener.Close()
		listenerFile, endpoint, port = inheritedFile, inheritedEndpoint, inheritedPort
		defer listenerFile.Close()
		result.readiness.protocol = httpmodel.ReadinessProtocolV1
		result.readiness.listenerFD = httpListenerChildFD
		result.readiness.readinessFD = httpReadinessChildFD
		result.readiness.endpoint = endpoint
		result.readiness.port = port
	}
	readinessReader, readinessWriter, err := os.Pipe()
	if err != nil {
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_READINESS_PIPE_ALLOCATION_FAILED"
		return result
	}
	defer readinessReader.Close()
	defer readinessWriter.Close()

	var requestWire httpmodel.HTTPRequestWire
	if !portable {
		requestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)
		if err != nil || !requestWire.Valid() {
			result.process.primary = domain.ControlStartError
			result.process.diagnosticCode = "HTTP_REQUEST_WIRE_ENCODING_FAILED"
			return result
		}
		result.exchange.requestWire = requestWire.Bytes()
		result.exchange.requestSemanticDigest = requestWire.Digest()
		result.exchange.requestRawSHA256 = requestWire.RawSHA256()
	}
	request.environment, err = appendHTTPStartEnvironment(request.environment, request.binding.StartSpec().Authority(), port)
	if err != nil {
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_DESCRIPTOR_ENVIRONMENT_COLLISION"
		return result
	}
	result.environment = append([]string(nil), request.environment...)

	overflowC := make(chan struct{}, 2)
	captures := newPhysicalCapturePair(request.stdoutLimit, request.stderrLimit, overflowC)
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_STDOUT_PIPE_ALLOCATION_FAILED"
		return result
	}
	defer stdoutReader.Close()
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutWriter.Close()
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_STDERR_PIPE_ALLOCATION_FAILED"
		return result
	}
	defer stderrReader.Close()

	extraFiles := []*os.File{readinessWriter}
	if !portable {
		extraFiles = []*os.File{listenerFile, readinessWriter}
	}
	command := &exec.Cmd{
		Path:        request.tool.absolutePath,
		Args:        append([]string(nil), request.logicalArgv...),
		Env:         append([]string(nil), request.environment...),
		Dir:         request.cwd,
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
		// MUTATION_ANCHOR: http-readiness-must-use-inherited-pipe-not-http-probe
		ExtraFiles: extraFiles,
		Stdout:     stdoutWriter,
		Stderr:     stderrWriter,
	}
	result.process.physicalExecutionEntered = true
	result.process.spawnAttempted = true
	result.process.stdoutCaptureLimit = captures.stdoutReceiptLimit
	result.process.stderrCaptureLimit = captures.stderrReceiptLimit
	if err := command.Start(); err != nil {
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		result.process.primary = domain.ControlStartError
		result.process.diagnosticCode = "HTTP_SPAWN_FAILED"
		return result
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	if listenerFile != nil {
		_ = listenerFile.Close()
	}
	_ = readinessWriter.Close()
	result.process.started = true
	result.process.pid = command.Process.Pid
	processGroupID, groupErr := syscall.Getpgid(command.Process.Pid)
	result.process.processGroupID = processGroupID
	if groupErr == nil && processGroupID == command.Process.Pid {
		result.process.processGroupOwned = true
	} else {
		result.process.primary = domain.ControlStartError
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.finalProbeError = "new process group was not established"
		result.process.diagnosticCode = "HTTP_PROCESS_GROUP_NOT_OWNED"
		_ = command.Process.Kill()
	}

	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	go captures.stdout.drain(stdoutReader, stdoutDone)
	go captures.stderr.drain(stderrReader, stderrDone)
	waitC := make(chan waitResult, 1)
	go func() {
		waitErr := command.Wait()
		waitC <- waitResult{state: command.ProcessState, err: waitErr}
	}()
	var waited *waitResult

	if result.process.primary == "" {
		readiness, earlyWait, control, diagnostic := awaitExactHTTPReadiness(
			ctx, readinessReader, time.Duration(request.readinessBudgetMS)*time.Millisecond,
			captures.stdout, captures.stderr, overflowC, waitC, request.binding.Readiness(),
		)
		if portable {
			result.readiness.frameBytes = append([]byte(nil), readiness.bytes...)
		}
		result.readiness.bytesObserved = int64(len(readiness.bytes))
		if len(readiness.bytes) > 0 {
			result.readiness.observedByte = readiness.bytes[0]
		}
		result.readiness.eofObserved = readiness.eof
		result.readiness.accepted = control == ""
		result.readiness.diagnosticCode = diagnostic
		waited = earlyWait
		if portable && readiness.eof {
			frame, frameErr := httpmodel.ParseHTTPReadyPortFrame(readiness.bytes)
			if frameErr == nil && frame.Valid() {
				port = int(frame.Port())
				endpoint = "127.0.0.1:" + strconv.Itoa(port)
				result.readiness.port = port
				result.readiness.endpoint = endpoint
			}
		}
		if control != "" {
			result.process.primary = control
			result.process.diagnosticCode = diagnostic
		} else if portable {
			requestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)
			if err != nil || !requestWire.Valid() {
				result.process.primary = domain.ControlReadinessError
				result.process.diagnosticCode = "HTTP_PORTABLE_ENDPOINT_ENCODING_FAILED"
				result.readiness.accepted = false
				result.readiness.diagnosticCode = result.process.diagnosticCode
			} else {
				result.exchange.requestWire = requestWire.Bytes()
				result.exchange.requestSemanticDigest = requestWire.Digest()
				result.exchange.requestRawSHA256 = requestWire.RawSHA256()
			}
		}
		if result.process.primary == "" {
			if err := request.onReadinessAccepted(); err != nil {
				result.process.primary = domain.ControlReadinessError
				result.process.diagnosticCode = "HTTP_READINESS_STATE_TRANSITION_FAILED"
			}
		}
	}

	if result.process.primary == "" {
		// MUTATION_ANCHOR: http-exactly-one-direct-probe-no-redirect-or-retry
		result.exchange.connectionAttempts = 1
		exchange := performOneHTTPExchange(
			ctx, endpoint, requestWire.Bytes(), request.responseLimit,
			time.Duration(request.probeBudgetMS)*time.Millisecond,
		)
		result.exchange.requestWritten = exchange.written
		result.exchange.requestComplete = exchange.complete
		result.exchange.responseWire = append([]byte(nil), exchange.wire...)
		result.exchange.responseObserved = exchange.observed
		result.exchange.responseOverflow = exchange.overflow
		result.exchange.diagnosticCode = exchange.diagnostic
		// MUTATION_ANCHOR: http-service-control-precedence-output-cancel-timeout-transport
		if captures.stdout.overflowed() || captures.stderr.overflowed() {
			result.process.primary = domain.ControlOutputLimit
			result.process.diagnosticCode = "HTTP_PROCESS_OUTPUT_LIMIT"
		} else if ctx.Err() != nil {
			result.process.primary = domain.ControlCancelled
			result.process.diagnosticCode = "HTTP_EXCHANGE_CANCELLED"
		} else if exchange.control != "" {
			result.process.primary = exchange.control
			result.process.diagnosticCode = exchange.diagnostic
		} else {
			parsed, parseErr := httpmodel.ParseResponse(exchange.wire, request.binding.CapturePolicy())
			if parseErr != nil || !parsed.Valid() {
				result.process.primary, result.process.diagnosticCode = classifyHTTPResponseParseError(parseErr)
				result.exchange.diagnosticCode = result.process.diagnosticCode
			} else {
				result.exchange.responseSummary = httpResponseSummary{digest: parsed.Digest(), wireDigest: parsed.WireDigest(), status: parsed.Status()}
				result.exchange.responseParsed = true
				if err := request.onResponseCaptured(); err != nil {
					result.process.primary = domain.ControlProbeTransportError
					result.process.diagnosticCode = "HTTP_CAPTURE_STATE_TRANSITION_FAILED"
				}
			}
		}
	}

	teardownBudget := time.Duration(request.teardownBudgetMS) * time.Millisecond
	teardownDeadline := time.Now().Add(teardownBudget)
	if waited == nil {
		// A one-response service may close itself as soon as the response is
		// flushed. Reap that cooperative exit before probing its process group:
		// on Darwin an unreaped direct child can make a clean natural exit look
		// like an uncertain pre-TERM group observation. The wait is bounded by
		// the same declared teardown budget and leaves the majority of that
		// budget for group signaling, drains, and the final absence proof.
		naturalExitDeadline := time.Now().Add(teardownBudget / 4)
		if naturalExitDeadline.After(teardownDeadline) {
			naturalExitDeadline = teardownDeadline
		}
		waited = waitForDirectChild(waitC, naturalExitDeadline)
	}
	controller := systemDarwinGroups{}
	if result.process.processGroupOwned {
		teardownOwnedProcessGroup(
			&result.process, controller, teardownDeadline,
			teardownBudget,
		)
	}
	if waited == nil {
		waited = waitForDirectChild(waitC, teardownDeadline)
	}
	if waited != nil {
		classifyWait(&result.process, *waited)
		applyHTTPNaturalExitControl(&result.process)
	} else {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_DIRECT_CHILD_WAIT_DEADLINE")
		_ = command.Process.Kill()
	}

	stdoutDrain, stderrDrain := collectDrainCompletions(stdoutDone, stderrDone, teardownDeadline)
	result.process.stdoutDrained = stdoutDrain.beforeDeadline
	result.process.stderrDrained = stderrDrain.beforeDeadline
	if !stdoutDrain.observed {
		_ = stdoutReader.Close()
	}
	if !stderrDrain.observed {
		_ = stderrReader.Close()
	}
	if !stdoutDrain.observed && !awaitOwnerDrain(stdoutDone, ownerDrainCloseBudget) {
		captures.stdout.freeze()
	}
	if !stderrDrain.observed && !awaitOwnerDrain(stderrDone, ownerDrainCloseBudget) {
		captures.stderr.freeze()
	}
	if !result.process.stdoutDrained || !result.process.stderrDrained {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_PIPE_DRAIN_DEADLINE")
	}
	result.process.stdout, result.process.stdoutObserved, result.process.stdoutOverflow, err = captures.stdout.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_STDOUT_DRAIN_FAILED")
	}
	result.process.stderr, result.process.stderrObserved, result.process.stderrOverflow, err = captures.stderr.snapshot()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		result.process.teardownError = true
		result.process.orphanRisk = true
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "HTTP_STDERR_DRAIN_FAILED")
	}
	applyCompletedOutputControl(&result.process)

	if result.process.processGroupOwned {
		clean, probeErr := performFinalGroupProbe(controller, result.process.processGroupID, teardownDeadline)
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
	if err := request.tool.revalidate(); err != nil {
		result.process.teardownError = true
		result.process.waitError = err.Error()
		result.process.diagnosticCode = firstDiagnostic(result.process.diagnosticCode, "TOOL_CHANGED_AFTER_HTTP_EXECUTION")
	}
	if result.process.started && (!result.process.processGroupOwned || !result.process.directChildWaited ||
		!result.process.stdoutDrained || !result.process.stderrDrained || !result.process.finalProbeClean) {
		result.process.orphanRisk = true
	}
	return result
}

func applyHTTPNaturalExitControl(process *physicalProcessResult) {
	if process == nil || process.primary != "" || !process.directChildWaited || process.termSent || process.killSent {
		return
	}
	if process.exitCode == 0 && process.exitSignal == "" {
		return
	}
	// A response frame is not authority to ignore a service that independently
	// failed. Owner-initiated teardown signals are handled above; a natural
	// nonzero exit or signal remains a transport control and cannot project.
	process.primary = domain.ControlProbeTransportError
	process.diagnosticCode = firstDiagnostic(process.diagnosticCode, "HTTP_SERVICE_NONZERO_EXIT")
}

func classifyHTTPResponseParseError(err error) (domain.ControlReason, string) {
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

func allocateInheritedLoopbackListener() (*net.TCPListener, *os.File, string, int, error) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		return nil, nil, "", 0, err
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !address.IP.Equal(net.IPv4(127, 0, 0, 1)) || address.Port < 1 || address.Port > 65535 {
		_ = listener.Close()
		return nil, nil, "", 0, errors.New("listener did not bind literal IPv4 loopback")
	}
	file, err := listener.File()
	if err != nil {
		_ = listener.Close()
		return nil, nil, "", 0, err
	}
	return listener, file, "127.0.0.1:" + strconv.Itoa(address.Port), address.Port, nil
}

func appendHTTPDescriptorEnvironment(environment []string, port int) ([]string, error) {
	return appendHTTPStartEnvironment(environment, httpmodel.HTTPStartAuthorityV1, port)
}

func appendHTTPStartEnvironment(environment []string, authority string, port int) ([]string, error) {
	// MUTATION_ANCHOR: http-descriptor-environment-uses-only-plan-and-owned-facts
	legacy := authority == httpmodel.HTTPStartAuthorityV1
	portable := authority == httpmodel.HTTPPortableStartAuthorityV1
	if (!legacy && !portable) || (legacy && (port < 1 || port > 65535)) || (portable && port != 0) {
		return nil, errors.New("HTTP listener port is outside the valid range")
	}
	values := make(map[string]string, len(environment)+3)
	for _, entry := range environment {
		name, value, present := strings.Cut(entry, "=")
		if !present || name == "" {
			return nil, errors.New("HTTP service environment contains a malformed entry")
		}
		if _, duplicate := values[name]; duplicate {
			return nil, errors.New("HTTP service environment contains a duplicate name")
		}
		if isHTTPDescriptorEnvironmentName(name) {
			return nil, errors.New("HTTP service environment collides with runner-owned descriptor authority")
		}
		values[name] = value
	}
	if legacy {
		values[httpListenerFDEnvironment] = strconv.Itoa(httpListenerChildFD)
		values[httpReadinessFDEnvironment] = strconv.Itoa(httpReadinessChildFD)
		values[httpListenerPortEnvironment] = strconv.Itoa(port)
	} else {
		// The portable child owns one pipe only. It selects a literal-loopback
		// ephemeral port itself and reports that port in the exact EOF frame.
		values[httpReadinessFDEnvironment] = strconv.Itoa(httpPortableReadinessChildFD)
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]string, len(names))
	for index, name := range names {
		result[index] = name + "=" + values[name]
	}
	return result, nil
}

func awaitExactHTTPReadiness(
	ctx context.Context,
	reader *os.File,
	budget time.Duration,
	stdout, stderr *cappedCapture,
	overflowC <-chan struct{},
	waitC <-chan waitResult,
	contract httpmodel.HTTPReadinessContract,
) (httpReadinessRead, *waitResult, domain.ControlReason, string) {
	readLimit := 2
	if _, maxBytes, _, portable := contract.PortableFrameProfile(); portable {
		readLimit = maxBytes + 1
	}
	resultC := make(chan httpReadinessRead, 1)
	go readExactHTTPReadiness(reader, readLimit, resultC)
	timer := time.NewTimer(budget)
	defer timer.Stop()
	deadlineObserved := false
	var waited *waitResult
	var observed *httpReadinessRead
	for {
		deadlineObserved, waited, observed = latchHTTPReadinessOwnerFacts(
			timer.C, waitC, resultC, deadlineObserved, waited, observed,
		)
		if stdout.overflowed() || stderr.overflowed() {
			return stopHTTPReadinessRead(reader, resultC, observed), waited, domain.ControlOutputLimit, "HTTP_PROCESS_OUTPUT_LIMIT_BEFORE_READINESS"
		}
		if ctx.Err() != nil {
			return stopHTTPReadinessRead(reader, resultC, observed), waited, domain.ControlCancelled, "HTTP_READINESS_CANCELLED"
		}
		if deadlineObserved {
			return stopHTTPReadinessRead(reader, resultC, observed), waited, domain.ControlReadinessError, "HTTP_READINESS_TIMEOUT"
		}
		if waited != nil {
			return finishHTTPReadinessAfterChildExit(reader, resultC, observed), waited, domain.ControlReadinessError, "HTTP_PROCESS_EXITED_BEFORE_READINESS"
		}
		if observed != nil {
			// MUTATION_ANCHOR: http-readiness-requires-exact-profile-bytes-and-eof
			valid := observed.err == nil && observed.eof
			if contract.Protocol() == httpmodel.ReadinessProtocolV1 {
				valid = valid && len(observed.bytes) == 1 && observed.bytes[0] == httpmodel.ReadinessSuccessByte
			} else if contract.Protocol() == httpmodel.PortableReadinessProtocolV1 {
				frame, err := httpmodel.ParseHTTPReadyPortFrame(observed.bytes)
				valid = valid && err == nil && frame.Valid()
			} else {
				valid = false
			}
			if !valid {
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

// latchHTTPReadinessOwnerFacts converts select into a wake-up mechanism only.
// After every wake, the owner polls already-visible facts in fixed precedence:
// deadline, direct-child exit, then readiness bytes. The caller checks output
// and cancellation before these latched facts, so simultaneous observations
// cannot be classified by Go's randomized select choice.
func latchHTTPReadinessOwnerFacts(
	deadlineC <-chan time.Time,
	waitC <-chan waitResult,
	resultC <-chan httpReadinessRead,
	deadlineObserved bool,
	waited *waitResult,
	observed *httpReadinessRead,
) (bool, *waitResult, *httpReadinessRead) {
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

func finishHTTPReadinessAfterChildExit(
	reader *os.File,
	resultC <-chan httpReadinessRead,
	observed *httpReadinessRead,
) httpReadinessRead {
	if observed != nil {
		return *observed
	}
	// Direct-child exit has already won classification. Give the dedicated
	// reader one bounded owner interval to retain any readiness bytes and EOF
	// that the exiting child placed in the kernel pipe before closing its
	// profile-selected readiness descriptor.
	// A descendant may have inherited the writer, so this can never be an
	// unbounded wait; after the interval the owner closes its read end.
	timer := time.NewTimer(ownerDrainCloseBudget)
	defer timer.Stop()
	select {
	case result := <-resultC:
		return result
	case <-timer.C:
		return stopHTTPReadinessRead(reader, resultC, nil)
	}
}

func stopHTTPReadinessRead(
	reader *os.File,
	resultC <-chan httpReadinessRead,
	observed *httpReadinessRead,
) httpReadinessRead {
	if observed != nil {
		return *observed
	}
	_ = reader.Close()
	timer := time.NewTimer(ownerDrainCloseBudget)
	defer timer.Stop()
	select {
	case result := <-resultC:
		return result
	case <-timer.C:
		return httpReadinessRead{}
	}
}

func readExactHTTPReadiness(reader io.Reader, readLimit int, resultC chan<- httpReadinessRead) {
	if readLimit < 2 {
		readLimit = 2
	}
	result := httpReadinessRead{bytes: make([]byte, 0, readLimit)}
	buffer := make([]byte, 1)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			result.bytes = append(result.bytes, buffer[:count]...)
			if len(result.bytes) >= readLimit {
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

func performOneHTTPExchange(
	ctx context.Context,
	endpoint string,
	requestWire []byte,
	responseLimit int64,
	budget time.Duration,
) httpExchangeResult {
	result := httpExchangeResult{}
	probeContext, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(probeContext, "tcp4", endpoint)
	if err != nil {
		return classifyHTTPExchangeError(probeContext, result, err, "HTTP_CONNECT_FAILED")
	}
	defer connection.Close()
	deadline, present := probeContext.Deadline()
	if !present {
		result.control = domain.ControlProbeTransportError
		result.diagnostic = "HTTP_PROBE_DEADLINE_MISSING"
		return result
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return classifyHTTPExchangeError(probeContext, result, err, "HTTP_DEADLINE_INSTALL_FAILED")
	}
	stopCancellationWatch := make(chan struct{})
	cancellationWatchDone := make(chan struct{})
	go func() {
		defer close(cancellationWatchDone)
		select {
		case <-probeContext.Done():
			// net.Conn reads and writes are not context-aware after DialContext.
			// Advancing the connection deadline wakes an established exchange so
			// cancellation and a shorter parent deadline remain authoritative.
			_ = connection.SetDeadline(time.Now())
		case <-stopCancellationWatch:
		}
	}()
	defer func() {
		close(stopCancellationWatch)
		<-cancellationWatchDone
	}()
	for result.written < int64(len(requestWire)) {
		count, writeErr := connection.Write(requestWire[result.written:])
		result.written += int64(count)
		if writeErr != nil {
			return classifyHTTPExchangeError(probeContext, result, writeErr, "HTTP_REQUEST_WRITE_FAILED")
		}
		if count == 0 {
			result.control = domain.ControlProbeTransportError
			result.diagnostic = "HTTP_REQUEST_WRITE_NO_PROGRESS"
			return result
		}
	}
	result.complete = true
	if tcp, ok := connection.(*net.TCPConn); !ok {
		result.control = domain.ControlProbeTransportError
		result.diagnostic = "HTTP_NON_TCP_CONNECTION"
		return result
	} else if err := tcp.CloseWrite(); err != nil {
		return classifyHTTPExchangeError(probeContext, result, err, "HTTP_REQUEST_HALF_CLOSE_FAILED")
	}
	limited := &io.LimitedReader{R: connection, N: responseLimit}
	response, readErr := io.ReadAll(limited)
	result.wire = append([]byte(nil), response...)
	result.observed = int64(len(response))
	if result.observed >= responseLimit {
		result.overflow = true
		result.control = domain.ControlOutputLimit
		result.diagnostic = "HTTP_RESPONSE_WIRE_LIMIT"
		return result
	}
	if readErr != nil {
		return classifyHTTPExchangeError(probeContext, result, readErr, "HTTP_RESPONSE_READ_FAILED")
	}
	return result
}

func classifyHTTPExchangeError(
	ctx context.Context,
	result httpExchangeResult,
	err error,
	diagnostic string,
) httpExchangeResult {
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.control = domain.ControlTimeout
			result.diagnostic = "HTTP_PROBE_TIMEOUT"
			return result
		}
		result.control = domain.ControlCancelled
		result.diagnostic = "HTTP_PROBE_CANCELLED"
		return result
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		result.control = domain.ControlTimeout
		result.diagnostic = "HTTP_PROBE_TIMEOUT"
		return result
	}
	result.control = domain.ControlProbeTransportError
	result.diagnostic = diagnostic
	return result
}
