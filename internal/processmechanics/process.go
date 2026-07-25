// Package processmechanics owns the neutral physical process lifecycle used by
// Countershape's world adapters and contract-execution runner. It deliberately
// knows nothing about targets, stores, semantic models, run permits, or receipt
// identities.
package processmechanics

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"path/filepath"
	"sync/atomic"
	"time"
)

const (
	ControlStartError          = "START_ERROR"
	ControlProbeTransportError = "PROBE_TRANSPORT_ERROR"
	ControlTimeout             = "TIMEOUT"
	ControlCancelled           = "CANCELLED"
	ControlOutputLimit         = "OUTPUT_LIMIT"
)

const (
	PreTermProbeNotApplicable = "NOT_APPLICABLE"
	PreTermProbePresent       = "PRESENT"
	PreTermProbeAbsent        = "ABSENT"
	PreTermProbeUncertain     = "UNCERTAIN"
)

// ProcessGroupReuseExclusion names the Darwin residual: the zero-signal probe
// and following signal cannot atomically bind one process-group incarnation.
const ProcessGroupReuseExclusion = "PRE_TERM_PROBE_AND_SIGNAL_ARE_NON_ATOMIC_PGID_REUSE_EXCLUDED_FROM_CLEANUP_CLAIM"

// EscapeExclusion prevents process-group cleanup evidence from being promoted
// into a claim about descendants that deliberately escape the owned group.
const EscapeExclusion = "PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM" // MUTANT_U2_CLAIM_PROCESS_ESCAPE_CONTAINMENT

type stdinPresence uint8

const (
	stdinAbsent stdinPresence = iota
	stdinPresent
)

// Stdin is an immutable, typed input. A present zero-byte stream remains
// observably different from no stdin pipe.
type Stdin struct {
	presence stdinPresence
	bytes    []byte
}

func AbsentStdin() Stdin { return Stdin{presence: stdinAbsent} }

func PresentStdin(input []byte) Stdin {
	return Stdin{presence: stdinPresent, bytes: append([]byte(nil), input...)}
}

// Limits owns the bounded physical budgets. All values must be positive.
type Limits struct {
	StdoutBytes int64
	StderrBytes int64
	Execution   time.Duration
	Teardown    time.Duration
}

// Invocation is intentionally opaque. In particular, it has no argv,
// environment, executable, or stdin accessor that could turn a prepared
// physical operation into a reusable authority object.
type Invocation struct {
	executable  string
	argv        []string
	environment []string
	stdin       Stdin
	cwd         string
	limits      Limits
	binding     BindingDigest
}

// BindingDigest is a one-way identity for the exact opaque invocation stored
// by mechanics. It is evidence identity, not process or admission authority.
type BindingDigest [sha256.Size]byte

func (digest BindingDigest) String() string { return "sha256:" + hex.EncodeToString(digest[:]) }

func (digest BindingDigest) Valid() bool { return digest != BindingDigest{} }

func NewInvocation(
	executable string,
	argv []string,
	environment []string,
	stdin Stdin,
	cwd string,
	limits Limits,
) (Invocation, error) {
	if !filepath.IsAbs(executable) || filepath.Clean(executable) != executable {
		return Invocation{}, errors.New("process executable must be a clean absolute path")
	}
	if len(argv) == 0 || argv[0] == "" {
		return Invocation{}, errors.New("process argv must contain a non-empty argv[0]")
	}
	if !filepath.IsAbs(cwd) || filepath.Clean(cwd) != cwd {
		return Invocation{}, errors.New("process cwd must be a clean absolute path")
	}
	if limits.StdoutBytes <= 0 || limits.StderrBytes <= 0 || limits.Execution <= 0 || limits.Teardown <= 0 {
		return Invocation{}, errors.New("process limits and budgets must be positive")
	}
	if stdin.presence != stdinAbsent && stdin.presence != stdinPresent {
		return Invocation{}, errors.New("process stdin presence is invalid")
	}
	result := Invocation{
		executable:  executable,
		argv:        append([]string(nil), argv...),
		environment: append([]string(nil), environment...),
		stdin:       Stdin{presence: stdin.presence, bytes: append([]byte(nil), stdin.bytes...)},
		cwd:         cwd,
		limits:      limits,
	}
	result.binding = digestInvocation(result)
	return result, nil
}

type preparedState struct {
	invocation Invocation
	started    atomic.Bool
}

// Prepared freezes an immutable process request before admission. Its private
// shared state makes Start one-shot even if a caller copies the exported
// handle value. Executable authority stays with the world or contract runner.
type Prepared struct {
	state *preparedState
}

func Prepare(invocation Invocation) (*Prepared, error) {
	copyInvocation, err := NewInvocation(
		invocation.executable,
		invocation.argv,
		invocation.environment,
		invocation.stdin,
		invocation.cwd,
		invocation.limits,
	)
	if err != nil {
		return nil, err
	}
	if invocation.binding != copyInvocation.binding {
		return nil, errors.New("process invocation binding is invalid")
	}
	return &Prepared{state: &preparedState{invocation: copyInvocation}}, nil
}

func (prepared *Prepared) BindingDigest() BindingDigest {
	if prepared == nil || prepared.state == nil {
		return BindingDigest{}
	}
	return prepared.state.invocation.binding
}

func (prepared *Prepared) begin() error {
	if prepared == nil || prepared.state == nil {
		return errors.New("nil prepared process")
	}
	if !prepared.state.started.CompareAndSwap(false, true) {
		return errors.New("prepared process already consumed")
	}
	return nil
}

// SpawnObservation contains only parent-observed process identity facts.
type SpawnObservation struct {
	PID               int
	ProcessGroupID    int
	ProcessGroupOwned bool
	Binding           BindingDigest
}

// Result contains bounded physical observations and no invocation authority.
type Result struct {
	PhysicalExecutionEntered bool
	SpawnAttempted           bool
	Started                  bool
	PID                      int
	ProcessGroupID           int
	ProcessGroupOwned        bool
	Binding                  BindingDigest

	ExitCode   int
	ExitSignal string
	WaitError  string

	Stdout             []byte
	Stderr             []byte
	StdoutObserved     int64
	StderrObserved     int64
	StdoutOverflow     bool
	StderrOverflow     bool
	StdoutCaptureLimit int64
	StderrCaptureLimit int64

	Primary         string
	PreTermProbe    string
	TermSent        bool
	KillSent        bool
	ChildWaited     bool
	StdoutDrained   bool
	StderrDrained   bool
	FinalProbeClean bool
	FinalProbeError string
	TeardownError   bool
	OrphanRisk      bool
	DiagnosticCode  string

	StdinPresent          bool
	StdinDeclared         int64
	StdinPipeAllocated    bool
	StdinWriterStarted    bool
	StdinHandoffAttempted bool
	StdinWritten          int64
	StdinComplete         bool
	StdinErrorCode        string
}

func initialResult(invocation Invocation) Result {
	return Result{
		// MUTATION_ANCHOR: physical-entry-requires-spawn-attempt
		PhysicalExecutionEntered: false,
		ExitCode:                 -1,
		PreTermProbe:             PreTermProbeNotApplicable,
		StdinPresent:             invocation.stdin.presence == stdinPresent,
		StdinDeclared:            int64(len(invocation.stdin.bytes)),
		StdinComplete:            invocation.stdin.presence == stdinAbsent,
		Binding:                  invocation.binding,
	}
}

// StartError represents a conclusive parent-observed refusal or os/exec start
// failure. Result returns a defensive copy of any bounded capture bytes.
type StartError struct {
	code   string
	cause  error
	result Result
}

func (failure *StartError) Error() string {
	if failure == nil {
		return "process start failed"
	}
	if failure.cause == nil {
		return failure.code
	}
	return failure.code + ": " + failure.cause.Error()
}

func (failure *StartError) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.cause
}

func (failure *StartError) Code() string {
	if failure == nil {
		return ""
	}
	return failure.code
}

func (failure *StartError) Result() Result {
	if failure == nil {
		return Result{}
	}
	return cloneResult(failure.result)
}

func newStartError(code string, cause error, result Result) *StartError {
	result.Primary = ControlStartError
	result.DiagnosticCode = code
	if cause != nil {
		result.WaitError = cause.Error()
	}
	return &StartError{code: code, cause: cause, result: result}
}

func digestInvocation(invocation Invocation) BindingDigest {
	digest := sha256.New()
	writeDigestFrame(digest, "executable", []byte(invocation.executable))
	writeDigestStrings(digest, "argv", invocation.argv)
	writeDigestStrings(digest, "environment", invocation.environment)
	writeDigestFrame(digest, "stdin-presence", []byte{byte(invocation.stdin.presence)})
	writeDigestFrame(digest, "stdin", invocation.stdin.bytes)
	writeDigestFrame(digest, "cwd", []byte(invocation.cwd))
	var number [8]byte
	for _, value := range []int64{
		invocation.limits.StdoutBytes,
		invocation.limits.StderrBytes,
		invocation.limits.Execution.Nanoseconds(),
		invocation.limits.Teardown.Nanoseconds(),
	} {
		binary.BigEndian.PutUint64(number[:], uint64(value))
		writeDigestFrame(digest, "limit", number[:])
	}
	var result BindingDigest
	copy(result[:], digest.Sum(nil))
	return result
}

func writeDigestStrings(digest hash.Hash, tag string, values []string) {
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(values)))
	writeDigestFrame(digest, tag+"-count", count[:])
	for _, value := range values {
		writeDigestFrame(digest, tag+"-item", []byte(value))
	}
}

func writeDigestFrame(digest hash.Hash, tag string, body []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(tag)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(tag))
	binary.BigEndian.PutUint64(length[:], uint64(len(body)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write(body)
}

func cloneResult(result Result) Result {
	result.Stdout = append([]byte(nil), result.Stdout...)
	result.Stderr = append([]byte(nil), result.Stderr...)
	return result
}

func firstDiagnostic(existing, next string) string {
	if existing != "" {
		return existing
	}
	return next
}

// Start and the platform-owned Running lifecycle are implemented per host.
func (prepared *Prepared) Start(ctx context.Context) (SpawnObservation, *Running, error) {
	return prepared.startPlatform(ctx)
}
