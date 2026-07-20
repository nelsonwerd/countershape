// Package noderuntime owns admission and revalidation of the one explicit Node
// executable used by the contract runner. It never searches PATH and exports
// no constructor from serialized runtime facts.
package noderuntime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	CodeInvalidRuntime      = "INVALID_NODE_RUNTIME"
	CodeRuntimeChanged      = "NODE_RUNTIME_CHANGED"
	CodeProbeFailed         = "NODE_RUNTIME_PROBE_FAILED"
	CodeUnsupportedPlatform = "UNSUPPORTED_NODE_RUNTIME_PLATFORM"

	maxExecutableBytes = int64(512 << 20)
	maxExecutablePath  = 4096
	maxSafeInteger     = int64(9007199254740991)
)

var nodeVersionPattern = regexp.MustCompile(`^v?[1-9][0-9]*\.[0-9]+(?:\.[0-9]+)?(?:[-+][0-9A-Za-z.-]+)?$`)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

type runtimeSeal struct{ marker byte }

var admittedRuntimeSeal = &runtimeSeal{marker: 1}

type executableIdentity struct {
	device, inode, links, uid, gid uint64
	mode                           uint32
	size                           int64
	mtimeSec, mtimeNsec            int64
	ctimeSec, ctimeNsec            int64
	birthSec, birthNsec            int64
}

func (identity executableIdentity) valid() bool {
	return identity.device != 0 && identity.inode != 0 && identity.links > 0 && identity.size > 0
}

func (identity executableIdentity) equal(other executableIdentity) bool {
	return identity == other && identity.valid()
}

type runtimeSnapshot struct {
	identity    executableIdentity
	path        string
	bytesDigest domain.Digest
	mode        string
	byteCount   int64
}

type probeResult struct {
	path          string
	version       string
	major         int64
	platform      string
	architecture  string
	programDigest domain.Digest
}

type probeParentIdentity struct {
	device, inode, owner uint64
	permissions          uint32
}

func (identity probeParentIdentity) valid() bool {
	return identity.device != 0 && identity.inode != 0 && identity.permissions == 0o700
}

type runtimeState struct {
	gate        *sync.Mutex
	snapshot    runtimeSnapshot
	probe       probeResult
	probeParent string
	parentID    probeParentIdentity
	seal        *runtimeSeal
}

// Runtime is an opaque, copy-safe admission capability. Copies share the same
// revalidation gate; facts alone cannot reconstruct it.
type Runtime struct{ state *runtimeState }

type runtimeOperations struct {
	measure func(string) (runtimeSnapshot, error)
	probe   func(context.Context, string, string) (probeResult, error)
}

func productionOperations() runtimeOperations {
	return runtimeOperations{measure: measureExecutable, probe: runOwnedProbe}
}

// Admit resolves one explicit absolute path, measures it on both sides of one
// owner-controlled probe, and returns authority only for an exact stable tuple.
// The probe identifies the cooperating executable's reported Node profile; it
// is not vendor provenance or protection against an executable that emulates it.
func Admit(ctx context.Context, explicitPath, privateProbeParent string) (Runtime, error) {
	return admitWithOperations(ctx, explicitPath, privateProbeParent, productionOperations(), nil)
}

func admitWithOperations(
	ctx context.Context,
	explicitPath string,
	privateProbeParent string,
	operations runtimeOperations,
	sharedGate *sync.Mutex,
) (Runtime, error) {
	if ctx == nil || operations.measure == nil || operations.probe == nil {
		return Runtime{}, refuse(CodeInvalidRuntime, "context and closed runtime operations are required", nil)
	}
	if err := ctx.Err(); err != nil {
		return Runtime{}, refuse(CodeInvalidRuntime, "runtime admission context is closed", err)
	}
	resolved, err := resolveExplicitExecutable(explicitPath)
	if err != nil {
		return Runtime{}, err
	}
	probeParent, parentID, err := resolvePrivateProbeParent(privateProbeParent)
	if err != nil {
		return Runtime{}, err
	}
	gate := sharedGate
	if gate == nil {
		gate = &sync.Mutex{}
	}
	gate.Lock()
	defer gate.Unlock()
	return measureProbeMeasure(ctx, resolved, probeParent, parentID, operations, gate)
}

func measureProbeMeasure(
	ctx context.Context,
	resolved string,
	probeParent string,
	parentID probeParentIdentity,
	operations runtimeOperations,
	gate *sync.Mutex,
) (Runtime, error) {
	if err := ctx.Err(); err != nil {
		return Runtime{}, refuse(CodeInvalidRuntime, "runtime admission context is closed", err)
	}
	parentPath, beforeParent, err := resolvePrivateProbeParent(probeParent)
	if err != nil || parentPath != probeParent || !parentID.valid() || beforeParent != parentID {
		return Runtime{}, wrapPlatformError(CodeRuntimeChanged, "private probe parent changed before measurement", err)
	}
	before, err := operations.measure(resolved)
	if err != nil {
		return Runtime{}, wrapPlatformError(CodeInvalidRuntime, "executable pre-probe measurement failed", err)
	}
	measured, err := operations.probe(ctx, resolved, probeParent)
	if err != nil {
		return Runtime{}, wrapPlatformError(CodeProbeFailed, "owned Node probe failed", err)
	}
	after, err := operations.measure(resolved)
	if err != nil {
		return Runtime{}, wrapPlatformError(CodeRuntimeChanged, "executable post-probe measurement failed", err)
	}
	parentPath, afterParent, parentErr := resolvePrivateProbeParent(probeParent)
	if parentErr != nil || parentPath != probeParent || afterParent != parentID {
		return Runtime{}, wrapPlatformError(CodeRuntimeChanged, "private probe parent changed during measurement", parentErr)
	}
	if err := ctx.Err(); err != nil {
		return Runtime{}, refuse(CodeRuntimeChanged, "runtime admission context closed during measurement", err)
	}
	if !before.equal(after) || before.path != resolved || !measured.valid(resolved) ||
		measured.programDigest != nodeProbeDigest() {
		return Runtime{}, refuse(CodeRuntimeChanged, "runtime identity or exact probe tuple changed", nil)
	}
	authority := Runtime{state: &runtimeState{
		gate: gate, snapshot: before, probe: measured, probeParent: probeParent, parentID: parentID, seal: admittedRuntimeSeal,
	}}
	if !authority.Valid() {
		return Runtime{}, refuse(CodeRuntimeChanged, "runtime measurement did not close every authority invariant", nil)
	}
	return authority, nil
}

// Revalidate repeats the full measure-probe-measure sequence and returns a
// fresh capability only when every public and private identity fact rejoins.
func (r Runtime) Revalidate(ctx context.Context) (Runtime, error) {
	return revalidateWithOperations(ctx, r, productionOperations())
}

func revalidateWithOperations(ctx context.Context, r Runtime, operations runtimeOperations) (Runtime, error) {
	if !r.Valid() || ctx == nil {
		return Runtime{}, refuse(CodeInvalidRuntime, "runtime authority and context are required", nil)
	}
	r.state.gate.Lock()
	defer r.state.gate.Unlock()
	fresh, err := measureProbeMeasure(
		ctx, r.state.snapshot.path, r.state.probeParent, r.state.parentID, operations, r.state.gate,
	)
	if err != nil {
		return Runtime{}, err
	}
	if !r.state.snapshot.equal(fresh.state.snapshot) || !r.state.probe.equal(fresh.state.probe) {
		return Runtime{}, refuse(CodeRuntimeChanged, "fresh runtime authority differs from the admitted tuple", nil)
	}
	return fresh, nil
}

func (r Runtime) Valid() bool {
	return r.state != nil && r.state.gate != nil && r.state.seal == admittedRuntimeSeal &&
		r.state.snapshot.valid() && r.state.probe.valid(r.state.snapshot.path) &&
		r.state.probeParent != "" && r.state.parentID.valid()
}

func (r Runtime) Path() string {
	if !r.Valid() {
		return ""
	}
	return r.state.snapshot.path
}
func (r Runtime) Version() string {
	if !r.Valid() {
		return ""
	}
	return r.state.probe.version
}
func (r Runtime) Major() int64 {
	if !r.Valid() {
		return 0
	}
	return r.state.probe.major
}
func (r Runtime) Platform() string {
	if !r.Valid() {
		return ""
	}
	return r.state.probe.platform
}
func (r Runtime) Architecture() string {
	if !r.Valid() {
		return ""
	}
	return r.state.probe.architecture
}
func (r Runtime) ExecutableBytesDigest() domain.Digest {
	if !r.Valid() {
		return ""
	}
	return r.state.snapshot.bytesDigest
}
func (r Runtime) ExecutableMode() string {
	if !r.Valid() {
		return ""
	}
	return r.state.snapshot.mode
}
func (r Runtime) ExecutableByteCount() int64 {
	if !r.Valid() {
		return 0
	}
	return r.state.snapshot.byteCount
}
func (r Runtime) ProbeProgramDigest() domain.Digest {
	if !r.Valid() {
		return ""
	}
	return r.state.probe.programDigest
}

func (snapshot runtimeSnapshot) valid() bool {
	permissions := snapshot.identity.mode & 0o7777
	return snapshot.identity.valid() && validExecutablePathText(snapshot.path) &&
		(snapshot.identity.mode&0o170000) == 0o100000 && permissions&0o111 != 0 &&
		permissions&0o022 == 0 && permissions&0o7000 == 0 &&
		snapshot.identity.size == snapshot.byteCount && snapshot.bytesDigest.Valid() &&
		snapshot.mode == fmt.Sprintf("100%03o", permissions&0o777) &&
		snapshot.byteCount > 0 && snapshot.byteCount <= maxExecutableBytes
}

func (snapshot runtimeSnapshot) equal(other runtimeSnapshot) bool {
	return snapshot.valid() && other.valid() && snapshot.identity.equal(other.identity) &&
		snapshot.path == other.path && snapshot.bytesDigest == other.bytesDigest &&
		snapshot.mode == other.mode && snapshot.byteCount == other.byteCount
}

func (probe probeResult) valid(path string) bool {
	if probe.path != path || probe.platform != "darwin" || probe.architecture != "arm64" ||
		probe.major < 1 || probe.major > maxSafeInteger || probe.programDigest != nodeProbeDigest() {
		return false
	}
	major, ok := parseNodeVersion(probe.version)
	return ok && major == probe.major
}

func parseNodeVersion(version string) (int64, bool) {
	if len(version) == 0 || len(version) > 128 || !nodeVersionPattern.MatchString(version) {
		return 0, false
	}
	majorText := strings.TrimPrefix(strings.SplitN(version, ".", 2)[0], "v")
	major, err := strconv.ParseInt(majorText, 10, 64)
	return major, err == nil && major >= 1 && major <= maxSafeInteger
}

func (probe probeResult) equal(other probeResult) bool {
	return probe.path == other.path && probe.version == other.version && probe.major == other.major &&
		probe.platform == other.platform && probe.architecture == other.architecture &&
		probe.programDigest == other.programDigest
}

func resolveExplicitExecutable(raw string) (string, error) {
	if !validExecutablePathText(raw) {
		return "", refuse(CodeInvalidRuntime, "Node path must be explicit, clean, absolute, and bounded", nil)
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil || !validExecutablePathText(resolved) {
		return "", refuse(CodeInvalidRuntime, "Node path could not be resolved exactly", err)
	}
	return resolved, nil
}

func validExecutablePathText(path string) bool {
	if path == "" || len(path) > maxExecutablePath || !utf8.ValidString(path) ||
		strings.IndexByte(path, 0) >= 0 || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	for _, character := range path {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

func wrapPlatformError(code, detail string, err error) error {
	if IsCode(err, CodeUnsupportedPlatform) {
		return err
	}
	return refuse(code, detail, err)
}
