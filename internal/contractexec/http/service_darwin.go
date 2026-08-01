//go:build darwin

package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	readinessChildFD     = 3
	ownerDrainBudget     = 100 * time.Millisecond
	preTermNotApplicable = "NOT_APPLICABLE"
	preTermPresent       = "PRESENT"
	preTermAbsent        = "ABSENT"
	preTermUncertain     = "UNCERTAIN"
)

type servicePreparation struct {
	executable  string
	runtime     contractmodel.RuntimeBinding
	logicalArgv []string
	environment []string
	cwd         string
	stimulus    httpmodel.HTTPStimulus
	capture     httpmodel.HTTPCapturePolicy
	readiness   httpmodel.HTTPReadinessContract
	budgets     domain.Budgets
	sourceBind  domain.Digest
}

type processResult struct {
	physicalExecutionEntered bool
	spawnAttempted           bool
	started                  bool
	pid                      int
	processGroupID           int
	processGroupOwned        bool
	exitCode                 int
	exitSignal               string
	waitError                string
	runtimeRevalidationError string
	stdout                   []byte
	stderr                   []byte
	stdoutObserved           int64
	stderrObserved           int64
	stdoutOverflow           bool
	stderrOverflow           bool
	stdoutCaptureLimit       int64
	stderrCaptureLimit       int64
	primary                  domain.ControlReason
	preTermProbe             string
	termSent                 bool
	killSent                 bool
	directChildWaited        bool
	stdoutDrained            bool
	stderrDrained            bool
	finalProbeClean          bool
	finalProbeError          string
	teardownError            bool
	orphanRisk               bool
	diagnosticCode           string
	descriptorCloseFailures  descriptorCloseFailure
}

type descriptorCloseFailure uint8

const (
	closeFailureStartError descriptorCloseFailure = 1 << iota
	closeFailureParentWriters
	closeFailureTerminalReaders
)

type readinessResult struct {
	protocol       string
	readinessFD    int
	endpoint       string
	port           int
	frameBytes     []byte
	bytesObserved  int64
	eofObserved    bool
	accepted       bool
	diagnosticCode string
}

type exchangeResult struct {
	connectionAttempts int
	requestWire        httpmodel.HTTPRequestWire
	requestWritten     int64
	requestComplete    bool
	responseWire       []byte
	responseObserved   int64
	responseOverflow   bool
	response           httpmodel.HTTPResponse
	responseParsed     bool
	diagnosticCode     string
}

type serviceResult struct {
	binding     domain.Digest
	process     processResult
	readiness   readinessResult
	exchange    exchangeResult
	environment []string
}

type ownedServiceDescriptor struct {
	file     *os.File
	once     sync.Once
	closeFn  func(*os.File) error
	closeErr error
}

func newOwnedServiceDescriptor(file *os.File) *ownedServiceDescriptor {
	return &ownedServiceDescriptor{
		file: file,
		closeFn: func(file *os.File) error {
			return file.Close()
		},
	}
}

func (owned *ownedServiceDescriptor) Read(body []byte) (int, error) {
	if owned == nil || owned.file == nil {
		return 0, os.ErrInvalid
	}
	return owned.file.Read(body)
}

func (owned *ownedServiceDescriptor) Close() error {
	if owned == nil || owned.file == nil {
		return nil
	}
	owned.once.Do(func() {
		if owned.closeFn == nil {
			owned.closeErr = os.ErrInvalid
			return
		}
		owned.closeErr = owned.closeFn(owned.file)
	})
	return owned.closeErr
}

type preparedService struct {
	mu sync.Mutex

	config          servicePreparation
	binding         domain.Digest
	command         *exec.Cmd
	readinessReader *ownedServiceDescriptor
	readinessWriter *ownedServiceDescriptor
	stdoutReader    *ownedServiceDescriptor
	stdoutWriter    *ownedServiceDescriptor
	stderrReader    *ownedServiceDescriptor
	stderrWriter    *ownedServiceDescriptor
	used            bool
	closed          bool
}

type runningService struct {
	once           sync.Once
	result         serviceResult
	processContext context.Context
	closureContext context.Context
	config         servicePreparation
	binding        domain.Digest
	command        *exec.Cmd
	readiness      *ownedServiceDescriptor
	stdout         *ownedServiceDescriptor
	stderr         *ownedServiceDescriptor
	captures       capturePair
	overflowC      chan struct{}
	stdoutDone     chan time.Time
	stderrDone     chan time.Time
	waitC          chan waitObservation
	initial        serviceResult
}

type capturePair struct {
	stdout *cappedCapture
	stderr *cappedCapture
}

type waitObservation struct {
	state *os.ProcessState
	err   error
}

func prepareHTTPService(config servicePreparation) (*preparedService, domain.Digest, error) {
	budgets := config.budgets
	prefix, maximum, eofRequired, portable := config.readiness.PortableFrameProfile()
	if config.executable == "" || len(config.logicalArgv) != 2 || config.logicalArgv[0] != "node" ||
		config.logicalArgv[1] == "" || !config.stimulus.Valid() || !config.capture.Valid() ||
		!config.readiness.Valid() || !portable || prefix != httpmodel.PortableReadinessFramePrefix ||
		maximum != httpmodel.PortableReadinessFrameMax || !eofRequired ||
		budgets.StdoutBytes < 1 || budgets.StderrBytes < 1 || budgets.ReadinessMS < 1 ||
		budgets.ProbeMS < 1 || budgets.TeardownMS < 1 || !config.sourceBind.Valid() {
		return nil, "", errors.New("HTTP service preparation is outside the closed profile")
	}
	if err := validateServiceEnvironment(config.environment); err != nil {
		return nil, "", err
	}
	digest, _, err := canon.DigestTyped("StandaloneHTTPServiceBinding", map[string]any{
		"schema_version":            domain.SchemaVersion,
		"kind":                      "StandaloneHTTPServiceBinding",
		"executable":                config.executable,
		"executable_bytes_digest":   config.runtime.ExecutableBytesDigest.String(),
		"logical_argv":              config.logicalArgv,
		"environment":               config.environment,
		"cwd":                       config.cwd,
		"stimulus_digest":           config.stimulus.Digest().String(),
		"capture_policy_digest":     config.capture.Digest().String(),
		"readiness_contract_digest": config.readiness.Digest().String(),
		"source_binding_digest":     config.sourceBind.String(),
		"stdout_bytes":              budgets.StdoutBytes,
		"stderr_bytes":              budgets.StderrBytes,
		"readiness_ms":              budgets.ReadinessMS,
		"probe_ms":                  budgets.ProbeMS,
		"teardown_ms":               budgets.TeardownMS,
	})
	if err != nil {
		return nil, "", err
	}
	binding, err := domain.ParseDigest(digest.String())
	if err != nil {
		return nil, "", err
	}
	readinessReader, readinessWriter, err := os.Pipe()
	if err != nil {
		return nil, "", err
	}
	ownedReadinessReader := newOwnedServiceDescriptor(readinessReader)
	ownedReadinessWriter := newOwnedServiceDescriptor(readinessWriter)
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, "", errors.Join(
			err,
			ownedReadinessReader.Close(),
			ownedReadinessWriter.Close(),
		)
	}
	ownedStdoutReader := newOwnedServiceDescriptor(stdoutReader)
	ownedStdoutWriter := newOwnedServiceDescriptor(stdoutWriter)
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		return nil, "", errors.Join(
			err,
			ownedReadinessReader.Close(),
			ownedReadinessWriter.Close(),
			ownedStdoutReader.Close(),
			ownedStdoutWriter.Close(),
		)
	}
	ownedStderrReader := newOwnedServiceDescriptor(stderrReader)
	ownedStderrWriter := newOwnedServiceDescriptor(stderrWriter)
	command := &exec.Cmd{
		Path:        config.executable,
		Args:        append([]string(nil), config.logicalArgv...),
		Env:         append([]string(nil), config.environment...),
		Dir:         config.cwd,
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
		ExtraFiles:  []*os.File{readinessWriter},
		Stdout:      stdoutWriter,
		Stderr:      stderrWriter,
	}
	prepared := &preparedService{
		config: config, binding: binding, command: command,
		readinessReader: ownedReadinessReader, readinessWriter: ownedReadinessWriter,
		stdoutReader: ownedStdoutReader, stdoutWriter: ownedStdoutWriter,
		stderrReader: ownedStderrReader, stderrWriter: ownedStderrWriter,
	}
	return prepared, binding, nil
}

func validateServiceEnvironment(environment []string) error {
	if len(environment) == 0 {
		return errors.New("HTTP environment is empty")
	}
	seen := make(map[string]string, len(environment))
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name == "" || strings.ContainsAny(name, "=\x00") || strings.ContainsRune(value, '\x00') {
			return errors.New("HTTP environment contains invalid text")
		}
		if _, duplicate := seen[name]; duplicate {
			return errors.New("HTTP environment repeats a name")
		}
		seen[name] = value
	}
	if seen[readinessFDEnvironment] != "3" {
		return errors.New("HTTP readiness descriptor environment differs")
	}
	if _, present := seen[listenerFDEnvironment]; present {
		return errors.New("portable HTTP environment contains listener authority")
	}
	if _, present := seen[listenerPortEnvironment]; present {
		return errors.New("portable HTTP environment contains port authority")
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) != len(environment) {
		return errors.New("HTTP environment cardinality differs")
	}
	for index, name := range names {
		if environment[index] != name+"="+seen[name] {
			return errors.New("HTTP environment is not in canonical name order")
		}
	}
	return nil
}

func (prepared *preparedService) BindingDigest() domain.Digest {
	if prepared == nil {
		return ""
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	if prepared.closed || !prepared.binding.Valid() {
		return ""
	}
	return prepared.binding
}

func (prepared *preparedService) Revalidate() error {
	if prepared == nil {
		return errors.New("prepared HTTP service is absent")
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	if prepared.closed || prepared.used {
		return errors.New("prepared HTTP service is closed or consumed")
	}
	return validateExecutable(prepared.config.runtime)
}

func (prepared *preparedService) Start(
	processContext context.Context,
	closureContext context.Context,
) (serviceResult, *runningService, error) {
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	base := serviceResult{
		binding: prepared.binding,
		process: processResult{
			exitCode: -1, preTermProbe: preTermNotApplicable,
			stdoutCaptureLimit:       prepared.config.budgets.StdoutBytes,
			stderrCaptureLimit:       prepared.config.budgets.StderrBytes,
			physicalExecutionEntered: true, spawnAttempted: true,
		},
		readiness: readinessResult{
			protocol:    httpmodel.PortableReadinessProtocolV1,
			readinessFD: readinessChildFD,
		},
		environment: append([]string(nil), prepared.config.environment...),
	}
	if prepared.closed || prepared.used || processContext == nil || closureContext == nil {
		base.process.primary = domain.ControlStartError
		base.process.diagnosticCode = "HTTP_PREPARED_SERVICE_UNAVAILABLE"
		return base, nil, errors.New("prepared HTTP service is unavailable")
	}
	prepared.used = true
	if err := prepared.command.Start(); err != nil {
		closeErr := errors.Join(
			prepared.readinessWriter.Close(),
			prepared.stdoutWriter.Close(),
			prepared.stderrWriter.Close(),
			prepared.readinessReader.Close(),
			prepared.stdoutReader.Close(),
			prepared.stderrReader.Close(),
		)
		recordDescriptorCloseFailure(&base.process, closeFailureStartError, closeErr)
		prepared.closed = true
		base.process.primary = domain.ControlStartError
		base.process.diagnosticCode = "HTTP_SPAWN_FAILED"
		base.process.waitError = err.Error()
		return base, nil, errors.Join(err, closeErr)
	}
	recordDescriptorCloseFailure(
		&base.process,
		closeFailureParentWriters,
		errors.Join(
			prepared.readinessWriter.Close(),
			prepared.stdoutWriter.Close(),
			prepared.stderrWriter.Close(),
		),
	)
	base.process.started = true
	base.process.pid = prepared.command.Process.Pid
	group, groupErr := syscall.Getpgid(base.process.pid)
	base.process.processGroupID = group
	if groupErr == nil && group == base.process.pid {
		base.process.processGroupOwned = true
	} else {
		base.process.primary = domain.ControlStartError
		base.process.teardownError = true
		base.process.orphanRisk = true
		base.process.finalProbeError = "new process group was not established"
		base.process.diagnosticCode = "HTTP_PROCESS_GROUP_NOT_OWNED"
		_ = prepared.command.Process.Kill()
	}
	overflowC := make(chan struct{}, 2)
	captures := capturePair{
		stdout: newCappedCapture(prepared.config.budgets.StdoutBytes, overflowC),
		stderr: newCappedCapture(prepared.config.budgets.StderrBytes, overflowC),
	}
	stdoutDone := make(chan time.Time, 1)
	stderrDone := make(chan time.Time, 1)
	go captures.stdout.drain(prepared.stdoutReader, stdoutDone)
	go captures.stderr.drain(prepared.stderrReader, stderrDone)
	waitC := make(chan waitObservation, 1)
	go func(command *exec.Cmd) {
		waitErr := command.Wait()
		waitC <- waitObservation{state: command.ProcessState, err: waitErr}
	}(prepared.command)
	running := &runningService{
		processContext: processContext, closureContext: closureContext,
		config: prepared.config, binding: prepared.binding, command: prepared.command,
		readiness: prepared.readinessReader, stdout: prepared.stdoutReader, stderr: prepared.stderrReader,
		captures: captures, overflowC: overflowC, stdoutDone: stdoutDone, stderrDone: stderrDone,
		waitC: waitC, initial: base,
	}
	return base, running, nil
}

func (prepared *preparedService) Close() error {
	if prepared == nil {
		return nil
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	if prepared.closed {
		return nil
	}
	prepared.closed = true
	if prepared.used {
		return nil
	}
	return errors.Join(
		prepared.readinessReader.Close(), prepared.readinessWriter.Close(),
		prepared.stdoutReader.Close(), prepared.stdoutWriter.Close(),
		prepared.stderrReader.Close(), prepared.stderrWriter.Close(),
	)
}

func recordDescriptorCloseFailure(
	result *processResult,
	phase descriptorCloseFailure,
	err error,
) {
	if result == nil || err == nil || phase == 0 {
		return
	}
	result.teardownError = true
	result.descriptorCloseFailures |= phase
}

func (result processResult) descriptorCloseDiagnostics() []string {
	diagnostics := make([]string, 0, 3)
	for _, item := range []struct {
		phase      descriptorCloseFailure
		diagnostic string
	}{
		{
			phase:      closeFailureStartError,
			diagnostic: "HTTP_START_ERROR_DESCRIPTOR_CLOSE_FAILED",
		},
		{
			phase:      closeFailureParentWriters,
			diagnostic: "HTTP_PARENT_WRITER_CLOSE_FAILED",
		},
		{
			phase:      closeFailureTerminalReaders,
			diagnostic: "HTTP_TERMINAL_READER_CLOSE_FAILED",
		},
	} {
		if result.descriptorCloseFailures&item.phase != 0 {
			diagnostics = append(diagnostics, item.diagnostic)
		}
	}
	return diagnostics
}

func validateExecutable(runtime contractmodel.RuntimeBinding) error {
	before, err := os.Lstat(runtime.AdmittedExecutablePath)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 ||
		before.Size() != runtime.ExecutableByteCount ||
		fmt.Sprintf("100%03o", before.Mode().Perm()) != runtime.ExecutableMode {
		return errors.Join(err, errors.New("admitted executable facts differ"))
	}
	fd, err := syscall.Open(
		runtime.AdmittedExecutablePath,
		syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(fd), runtime.AdmittedExecutablePath)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("admitted executable descriptor could not be owned")
	}
	opened, statErr := handle.Stat()
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, "countershape/v1/NodeExecutableBytes")
	_, _ = hasher.Write([]byte{0})
	count, readErr := io.Copy(hasher, io.LimitReader(handle, runtime.ExecutableByteCount+1))
	after, afterErr := handle.Stat()
	closeErr := handle.Close()
	pathAfter, pathErr := os.Lstat(runtime.AdmittedExecutablePath)
	digest, digestErr := domain.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if statErr != nil || readErr != nil || afterErr != nil || closeErr != nil || pathErr != nil ||
		digestErr != nil || count != runtime.ExecutableByteCount || digest != runtime.ExecutableBytesDigest ||
		!os.SameFile(before, opened) || !os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) {
		return errors.Join(
			statErr, readErr, afterErr, closeErr, pathErr, digestErr,
			errors.New("admitted executable changed during revalidation"),
		)
	}
	return nil
}
