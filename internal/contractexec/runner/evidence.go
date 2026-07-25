//go:build darwin

package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	"github.com/nelsonwerd/countershape/internal/store"
)

const referenceStimulusDigestText = "sha256:ed949de9ef5e290b68164b1a1453aec8f95fe5ad8c80643fc528c81871a7e254"

const (
	cliInvocationFilename = "cli-invocation.json"
	maxCLIInvocationBytes = 64 << 10
	cliInvocationNonclaim = "CANDIDATE_WRITTEN_EVIDENCE_IS_NOT_MALICIOUS_PROCESS_ATTESTATION"
)

const (
	privateEvidenceFrameMagic              = "COUNTERSHAPE_C4_PRIVATE_EVIDENCE_FRAME_V1\n"
	privateEvidenceFrameVersion            = "private-evidence-frame/v1"
	privateEvidenceSummaryMaxBytes         = int64(1 << 20)
	privateEvidenceCaptureOverheadMaxBytes = int64(1 << 20)
	privateEvidenceProjectionFrameMaxBytes = int64(3 << 20)
	privateEvidenceCanonicalBodyCount      = int64(12)
	privateEvidenceStoreAggregateMaxBytes  = int64(64 << 20)
	privateEvidenceMaximumUniqueBlobCount  = 14
	privateEvidenceMaximumChannelBytes     = int64(16 << 20)
)

type cliEvidenceCapacity struct {
	maximumUniqueBytes int64
}

type preparedTargetIdentity struct {
	model         contractmodel.ContractExecutionTarget
	bundle        emitmodel.ContractBundle
	roots         store.ConformanceAttemptRoots
	candidateRoot string
	markerPath    string
}

func newPreparedTargetIdentity(
	model contractmodel.ContractExecutionTarget,
	bundle emitmodel.ContractBundle,
	roots store.ConformanceAttemptRoots,
) (preparedTargetIdentity, error) {
	identity := preparedTargetIdentity{
		model: model, bundle: bundle, roots: roots,
		candidateRoot: filepath.Join(roots.CandidateParent(), "candidate"), markerPath: roots.MarkerPath(),
	}
	if !identity.valid() {
		return preparedTargetIdentity{}, errors.New("prepared target identity is incomplete")
	}
	return identity, nil
}

func (identity preparedTargetIdentity) valid() bool {
	return identity.model.Valid() && identity.bundle.Digest().Valid() &&
		identity.bundle.Digest() == identity.model.ContractBundleDigest() &&
		identity.roots.AttemptRoot() != "" && identity.roots.CandidateParent() != "" &&
		filepath.IsAbs(identity.candidateRoot) && filepath.Clean(identity.candidateRoot) == identity.candidateRoot &&
		identity.candidateRoot == filepath.Join(identity.roots.CandidateParent(), "candidate") &&
		identity.markerPath != "" && identity.markerPath == identity.roots.MarkerPath()
}

func preflightPrivateEvidenceCapacity(
	target contractmodel.ContractExecutionTarget,
	pre inventorySnapshot,
	stdoutBytes, stderrBytes int64,
) (cliEvidenceCapacity, error) {
	if !target.Valid() || !pre.valid() || stdoutBytes <= 0 || stderrBytes <= 0 ||
		stdoutBytes > privateEvidenceMaximumChannelBytes || stderrBytes > privateEvidenceMaximumChannelBytes {
		return cliEvidenceCapacity{}, refuse(
			CodeUnsupportedProfile,
			"CLI private-evidence inputs exceed the closed per-channel profile",
			nil,
		)
	}
	maximum := int64(0)
	for _, next := range []int64{
		privateEvidenceCaptureOverheadMaxBytes,
		stdoutBytes,
		stderrBytes,
		privateEvidenceProjectionFrameMaxBytes,
		privateEvidenceCanonicalBodyCount * privateEvidenceSummaryMaxBytes,
	} {
		if next < 0 || maximum > math.MaxInt64-next {
			return cliEvidenceCapacity{}, refuse(CodeUnsupportedProfile, "CLI private-evidence capacity arithmetic overflowed", nil)
		}
		maximum += next
	}
	if maximum > privateEvidenceStoreAggregateMaxBytes {
		return cliEvidenceCapacity{}, refuse(CodeUnsupportedProfile, "CLI private-evidence capacity exceeds the store ceiling", nil)
	}
	return cliEvidenceCapacity{maximumUniqueBytes: maximum}, nil
}

func (capacity cliEvidenceCapacity) validate(bodies map[contractmodel.EvidenceKind][]byte) error {
	if capacity.maximumUniqueBytes <= 0 || capacity.maximumUniqueBytes > privateEvidenceStoreAggregateMaxBytes ||
		len(bodies) == 0 || len(bodies) > 15 {
		return errors.New("CLI private-evidence capacity or body roster is invalid")
	}
	drain, hasDrain := bodies[contractmodel.EvidenceDrainResult]
	captured, hasCaptured := bodies[contractmodel.EvidenceCapturedObservation]
	if hasDrain != hasCaptured || (hasDrain && !bytes.Equal(drain, captured)) {
		return errors.New("drain and captured-observation evidence do not share one raw body")
	}
	unique := make(map[[sha256.Size]byte][]byte, len(bodies))
	var aggregate int64
	for _, body := range bodies {
		if len(body) == 0 {
			return errors.New("CLI private-evidence body is empty")
		}
		digest := sha256.Sum256(body)
		if previous, present := unique[digest]; present {
			if !bytes.Equal(previous, body) {
				return errors.New("CLI private-evidence digest collision is ambiguous")
			}
			continue
		}
		count := int64(len(body))
		if aggregate > capacity.maximumUniqueBytes-count {
			return errors.New("CLI private-evidence bodies exceed the pre-admitted envelope")
		}
		aggregate += count
		unique[digest] = body
	}
	if len(unique) > privateEvidenceMaximumUniqueBlobCount {
		return errors.New("CLI private-evidence unique blob count exceeds the closed envelope")
	}
	return nil
}

type privateEvidenceSegment struct {
	name string
	body []byte
}

func framePrivateEvidence(
	purpose string,
	facts any,
	maximumBytes int64,
	segments ...privateEvidenceSegment,
) ([]byte, error) {
	if (purpose != "CAPTURE_AND_DRAIN" && purpose != "PROJECTION_RESULT") ||
		maximumBytes <= 0 || maximumBytes > privateEvidenceStoreAggregateMaxBytes ||
		len(segments) == 0 || len(segments) > 2 {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame inputs are invalid", nil)
	}
	segmentFacts := make([]map[string]any, len(segments))
	seen := make(map[string]struct{}, len(segments))
	var payloadBytes int64
	for index, segment := range segments {
		if segment.name == "" || strings.ContainsAny(segment.name, "\x00/\\") {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame segment name is invalid", nil)
		}
		if _, duplicate := seen[segment.name]; duplicate {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame segment repeats", nil)
		}
		seen[segment.name] = struct{}{}
		count := int64(len(segment.body))
		if payloadBytes > math.MaxInt64-count {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame payload arithmetic overflowed", nil)
		}
		payloadBytes += count
		digest := sha256.Sum256(segment.body)
		segmentFacts[index] = map[string]any{
			"name": segment.name, "bytes": count, "sha256": hex.EncodeToString(digest[:]),
		}
	}
	metadata, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           "C4PrivateEvidenceFrame",
		"frame_version":  privateEvidenceFrameVersion,
		"purpose":        purpose,
		"facts":          facts,
		"segments":       segmentFacts,
	})
	if err != nil || len(metadata) == 0 || int64(len(metadata)) > privateEvidenceSummaryMaxBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame metadata is outside the summary ceiling", err)
	}
	framingBytes := int64(len(privateEvidenceFrameMagic) + 8 + len(metadata) + 8*len(segments))
	if framingBytes > maximumBytes || payloadBytes > maximumBytes-framingBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame exceeds its closed byte ceiling", nil)
	}
	body := make([]byte, 0, int(framingBytes+payloadBytes))
	body = append(body, privateEvidenceFrameMagic...)
	body = binary.BigEndian.AppendUint64(body, uint64(len(metadata)))
	body = append(body, metadata...)
	for _, segment := range segments {
		body = binary.BigEndian.AppendUint64(body, uint64(len(segment.body)))
		body = append(body, segment.body...)
	}
	return body, nil
}

const (
	invocationEvidenceValidated       = "VALIDATED"
	invocationEvidenceAbsent          = "ABSENT"
	invocationEvidenceFileFacts       = "FILE_FACTS_MISMATCH"
	invocationEvidenceTooLarge        = "TOO_LARGE"
	invocationEvidenceReadFailed      = "READ_FAILED"
	invocationEvidenceMalformed       = "MALFORMED_STRICT_JSON"
	invocationEvidenceSchemaMismatch  = "SCHEMA_MISMATCH"
	invocationEvidenceAttemptMismatch = "ATTEMPT_ID_MISMATCH"
	invocationEvidenceArgvMismatch    = "LOGICAL_ARGV_MISMATCH"
)

type invocationEvidence struct {
	Status               string
	Presence             string
	FileSHA256           string
	Bytes                int64
	AttemptIDValidated   bool
	LogicalArgvValidated bool
}

type invocationEvidenceAuthority struct {
	root   os.FileInfo
	marker os.FileInfo
}

func (authority invocationEvidenceAuthority) valid() bool {
	return authority.root != nil && authority.root.IsDir() && authority.root.Mode().Perm() == 0o700 &&
		authority.marker != nil && authority.marker.Mode().IsRegular()
}

func (evidence invocationEvidence) validated() bool {
	return evidence.Status == invocationEvidenceValidated && evidence.Presence == "PRESENT" &&
		len(evidence.FileSHA256) == sha256.Size*2 && evidence.AttemptIDValidated && evidence.LogicalArgvValidated
}

func (evidence invocationEvidence) contradictsBinding() bool {
	return evidence.Status == invocationEvidenceAttemptMismatch || evidence.Status == invocationEvidenceArgvMismatch
}

func (evidence invocationEvidence) facts() map[string]any {
	return map[string]any{
		"status": evidence.Status, "presence": evidence.Presence,
		"file_sha256": evidence.FileSHA256, "bytes": evidence.Bytes,
		"attempt_id_validated":   evidence.AttemptIDValidated,
		"logical_argv_validated": evidence.LogicalArgvValidated,
		"nonclaim":               cliInvocationNonclaim,
	}
}

func validCLIRuntimeAttemptID(value string) bool {
	if len(value) != len("attempt:")+sha256.Size*2 || !strings.HasPrefix(value, "attempt:") {
		return false
	}
	hexText := strings.TrimPrefix(value, "attempt:")
	decoded, err := hex.DecodeString(hexText)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(hexText) == hexText
}

func preflightCLIInvocationEvidence(root, markerPath string) (invocationEvidenceAuthority, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || filepath.Dir(markerPath) != root {
		return invocationEvidenceAuthority{}, errors.New("CLI evidence root or marker path is not exact")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 ||
		rootInfo.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return invocationEvidenceAuthority{}, errors.Join(err, errors.New("CLI evidence root is not an exact private directory"))
	}
	if err := boundedDirectRoster(context.Background(), root, rootInfo, []string{filepath.Base(markerPath)}); err != nil {
		return invocationEvidenceAuthority{}, errors.Join(err, errors.New("CLI evidence root is not marker-only before execution"))
	}
	marker, err := os.Lstat(markerPath)
	if err != nil || !marker.Mode().IsRegular() || marker.Mode()&os.ModeSymlink != 0 {
		return invocationEvidenceAuthority{}, errors.Join(err, errors.New("CLI attempt marker is not an exact regular file"))
	}
	if _, err := os.Lstat(filepath.Join(root, cliInvocationFilename)); !errors.Is(err, os.ErrNotExist) {
		return invocationEvidenceAuthority{}, errors.Join(err, errors.New("CLI invocation evidence leaf already exists"))
	}
	return invocationEvidenceAuthority{root: rootInfo, marker: marker}, nil
}

func inspectAndRetireCLIInvocationEvidence(
	ctx context.Context,
	root, markerPath, attemptID string,
	logicalArgv []string,
	authority invocationEvidenceAuthority,
) (invocationEvidence, error) {
	if ctx == nil || !authority.valid() || !validCLIRuntimeAttemptID(attemptID) || len(logicalArgv) == 0 || logicalArgv[0] == "" {
		return invocationEvidence{}, errors.New("CLI invocation evidence expectation is incomplete")
	}
	path := filepath.Join(root, cliInvocationFilename)
	observed := invocationEvidence{Status: invocationEvidenceAbsent, Presence: "ABSENT"}
	info, statErr := os.Lstat(path)
	expectedRoster := []string{filepath.Base(markerPath)}
	if statErr == nil {
		expectedRoster = append(expectedRoster, filepath.Base(path))
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return invocationEvidence{}, statErr
	}
	if err := boundedDirectRoster(ctx, root, authority.root, expectedRoster); err != nil {
		return invocationEvidence{}, err
	}
	currentMarker, markerErr := os.Lstat(markerPath)
	if markerErr != nil || currentMarker.Mode() != authority.marker.Mode() || !os.SameFile(authority.marker, currentMarker) {
		return invocationEvidence{}, errors.Join(markerErr, errors.New("CLI attempt marker identity changed before evidence inspection"))
	}
	if statErr == nil {
		observed.Presence = "PRESENT"
		switch {
		case !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 ||
			info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0:
			observed.Status = invocationEvidenceFileFacts
			observed.Bytes = info.Size()
		case info.Size() > maxCLIInvocationBytes:
			observed.Status = invocationEvidenceTooLarge
			observed.Bytes = info.Size()
		default:
			observed = readCLIInvocationEvidence(path, info, attemptID, logicalArgv)
		}
	}
	if err := retireCLIInvocationEvidence(ctx, root, markerPath, path, authority, info); err != nil {
		return invocationEvidence{}, err
	}
	return observed, nil
}

func readCLIInvocationEvidence(
	path string,
	expected os.FileInfo,
	attemptID string,
	logicalArgv []string,
) invocationEvidence {
	failure := func(status string, count int64, digest string) invocationEvidence {
		return invocationEvidence{Status: status, Presence: "PRESENT", FileSHA256: digest, Bytes: count}
	}
	descriptor, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return failure(invocationEvidenceReadFailed, 0, "")
	}
	handle := os.NewFile(uintptr(descriptor), path)
	if handle == nil {
		_ = syscall.Close(descriptor)
		return failure(invocationEvidenceReadFailed, 0, "")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.Mode().IsRegular() || opened.Size() != expected.Size() ||
		opened.Size() > maxCLIInvocationBytes || !os.SameFile(expected, opened) {
		_ = handle.Close()
		return failure(invocationEvidenceReadFailed, 0, "")
	}
	body, readErr := io.ReadAll(io.LimitReader(handle, maxCLIInvocationBytes+1))
	after, afterErr := handle.Stat()
	closeErr := handle.Close()
	pathAfter, pathErr := os.Lstat(path)
	if readErr != nil || afterErr != nil || closeErr != nil || pathErr != nil || len(body) > maxCLIInvocationBytes ||
		int64(len(body)) != opened.Size() || !os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) {
		return failure(invocationEvidenceReadFailed, int64(len(body)), "")
	}
	digest := sha256.Sum256(body)
	digestText := hex.EncodeToString(digest[:])
	value, parseErr := canon.Parse(body)
	if parseErr != nil {
		return failure(invocationEvidenceMalformed, int64(len(body)), digestText)
	}
	canonical, canonicalErr := value.CanonicalChecked()
	members, object := value.Members()
	if canonicalErr != nil || !bytes.Equal(canonical, body) || !object || len(members) != 2 {
		return failure(invocationEvidenceSchemaMismatch, int64(len(body)), digestText)
	}
	attemptValue, hasAttempt := value.LookupMember("attempt_id")
	argvValue, hasArgv := value.LookupMember("logical_argv")
	actualAttempt, attemptIsString := attemptValue.Text()
	argvElements, argvIsArray := argvValue.Elements()
	if !hasAttempt || !hasArgv || !attemptIsString || !argvIsArray || len(argvElements) != len(logicalArgv) {
		return failure(invocationEvidenceSchemaMismatch, int64(len(body)), digestText)
	}
	attemptMatches := actualAttempt == attemptID
	argvMatches := true
	for index, element := range argvElements {
		text, textOK := element.Text()
		if !textOK || text != logicalArgv[index] {
			argvMatches = false
		}
	}
	if !attemptMatches {
		result := failure(invocationEvidenceAttemptMismatch, int64(len(body)), digestText)
		result.LogicalArgvValidated = argvMatches
		return result
	}
	if !argvMatches {
		result := failure(invocationEvidenceArgvMismatch, int64(len(body)), digestText)
		result.AttemptIDValidated = true
		return result
	}
	return invocationEvidence{
		Status: invocationEvidenceValidated, Presence: "PRESENT", FileSHA256: digestText, Bytes: int64(len(body)),
		AttemptIDValidated: true, LogicalArgvValidated: true,
	}
}

func retireCLIInvocationEvidence(
	ctx context.Context,
	root, markerPath, path string,
	authority invocationEvidenceAuthority,
	observed os.FileInfo,
) error {
	if observed != nil {
		if err := removeRetainedLeaf(ctx, path, observed); err != nil {
			return errors.Join(err, errors.New("CLI invocation evidence leaf could not be retired exactly"))
		}
	}
	if err := boundedDirectRoster(ctx, root, authority.root, []string{filepath.Base(markerPath)}); err != nil {
		return errors.Join(err, errors.New("CLI evidence root did not return to its marker-only roster"))
	}
	currentMarker, markerErr := os.Lstat(markerPath)
	if markerErr != nil || currentMarker.Mode() != authority.marker.Mode() || !os.SameFile(authority.marker, currentMarker) {
		return errors.Join(markerErr, errors.New("CLI attempt marker identity changed during evidence retirement"))
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(err, errors.New("CLI invocation evidence leaf still exists after retirement"))
	}
	return nil
}

type scopeDraft struct {
	domain    contractmodel.ScopeDomain
	state     contractmodel.ScopeCheckState
	kind      contractmodel.EvidenceKind
	violation contractmodel.ScopeViolation
}

type evidenceDraft struct {
	bodies     map[contractmodel.EvidenceKind][]byte
	spawn      contractmodel.SpawnObservation
	result     processmechanics.Result
	primary    domain.ControlReason
	projection emitmodel.ExactTuple
	projected  bool
	captured   bool
	scope      []scopeDraft
}

func buildChildEvidenceDraft(
	input cliExecutionInput,
	targetModel contractmodel.ContractExecutionTarget,
	spawn contractmodel.SpawnObservation,
	result processmechanics.Result,
	post inventorySnapshot,
	measurements scopeMeasurements,
) (evidenceDraft, error) {
	spawnPID, childObserved := spawn.PID()
	if !spawn.Valid() || !childObserved || !targetModel.Valid() || !post.valid() || !result.Binding.Valid() ||
		!result.PhysicalExecutionEntered || !result.SpawnAttempted || !result.Started ||
		spawnPID != int64(result.PID) {
		return evidenceDraft{}, refuse(CodeEvidenceClosureFailed, "terminal child evidence inputs are invalid", nil)
	}
	if result.Binding.String() != input.prepared.BindingDigest().String() {
		return evidenceDraft{}, refuse(CodeEvidenceClosureFailed, "terminal mechanics binding differs", nil)
	}
	bodies := make(map[contractmodel.EvidenceKind][]byte, 15)
	put := func(kind contractmodel.EvidenceKind, facts any) error {
		body, err := canonicalEvidence(kind, facts)
		if err != nil {
			return err
		}
		bodies[kind] = body
		return nil
	}
	enrolled := input.pre.referenceEnrolled() && input.pre.equal(post) &&
		input.source.StimulusDigest().String() == referenceStimulusDigestText
	runtime := targetModel.Input().Runtime
	if err := put(contractmodel.EvidenceMaterializationRevalidation, map[string]any{
		"target_digest":           targetModel.Digest().String(),
		"before_inventory_sha256": input.pre.identity, "after_inventory_sha256": post.identity,
		"before_entry_count": len(input.pre.entries), "after_entry_count": len(post.entries),
		"unchanged":                  input.pre.equal(post),
		"reference_fixture_enrolled": enrolled,
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceRuntimeRevalidation, map[string]any{
		"admitted_executable_path": runtime.AdmittedExecutablePath,
		"executable_bytes_digest":  runtime.ExecutableBytesDigest.String(),
		"executable_mode":          runtime.ExecutableMode, "executable_byte_count": runtime.ExecutableByteCount,
		"measured_exec_path": runtime.MeasuredProcessExecPath, "version": runtime.Version,
		"major": runtime.Major, "platform": runtime.Platform, "architecture": runtime.Architecture,
		"probe_program_digest": runtime.ProbeProgramDigest.String(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceProcessResult, map[string]any{
		"physical_execution_entered": result.PhysicalExecutionEntered, "spawn_attempted": result.SpawnAttempted,
		"started": result.Started, "pid": result.PID, "process_group_id": result.ProcessGroupID,
		"process_group_owned": result.ProcessGroupOwned, "binding_digest": result.Binding.String(),
		"primary": result.Primary, "diagnostic_code": result.DiagnosticCode,
		"stdout_observed": result.StdoutObserved, "stderr_observed": result.StderrObserved,
		"stdout_overflow": result.StdoutOverflow, "stderr_overflow": result.StderrOverflow,
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceWaitResult, map[string]any{
		"child_waited": result.ChildWaited, "exit_code": result.ExitCode,
		"exit_signal": result.ExitSignal, "wait_error": result.WaitError,
	}); err != nil {
		return evidenceDraft{}, err
	}
	captureFrame, err := framePrivateEvidence(
		"CAPTURE_AND_DRAIN",
		map[string]any{
			"completion":     map[string]any{"exit_code": result.ExitCode, "exit_signal": result.ExitSignal},
			"stdout_drained": result.StdoutDrained, "stderr_drained": result.StderrDrained,
			"stdout_observed": result.StdoutObserved, "stderr_observed": result.StderrObserved,
			"stdout_overflow": result.StdoutOverflow, "stderr_overflow": result.StderrOverflow,
			"stdin_present": result.StdinPresent, "stdin_declared": result.StdinDeclared,
			"stdin_written": result.StdinWritten, "stdin_complete": result.StdinComplete,
		},
		int64(len(result.Stdout))+int64(len(result.Stderr))+privateEvidenceCaptureOverheadMaxBytes,
		privateEvidenceSegment{name: "stdout", body: result.Stdout},
		privateEvidenceSegment{name: "stderr", body: result.Stderr},
	)
	if err != nil {
		return evidenceDraft{}, err
	}
	bodies[contractmodel.EvidenceDrainResult] = captureFrame
	bodies[contractmodel.EvidenceCapturedObservation] = captureFrame
	if err := put(contractmodel.EvidenceTeardownResult, map[string]any{
		"pre_term_probe": result.PreTermProbe, "term_sent": result.TermSent, "kill_sent": result.KillSent,
		"teardown_error": result.TeardownError,
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceOrphanCheck, map[string]any{
		"final_group_probe_clean": result.FinalProbeClean, "final_group_probe_error": result.FinalProbeError,
		"orphan_risk": result.OrphanRisk,
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceFinalizationMarker, map[string]any{
		"terminal_facts_closed": true, "candidate_inventory_reopened": post.valid(),
		"scope_canaries_closed": !measurements.Ambiguous,
		"cli_invocation":        input.invocationEvidence.facts(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	primary, err := mechanicsPrimary(result.Primary)
	if err != nil {
		return evidenceDraft{}, err
	}
	tuple, projectionBytes, projectionErr := projectCLIResult(input, result)
	projected := projectionErr == nil && primary == ""
	if projected {
		projectionFrame, frameErr := framePrivateEvidence(
			"PROJECTION_RESULT",
			map[string]any{
				"portable_profile_digest": input.targetIdentity.bundle.Predicate().PortableProfileDigest().String(),
			},
			privateEvidenceProjectionFrameMaxBytes,
			privateEvidenceSegment{name: "projection", body: projectionBytes},
			privateEvidenceSegment{name: "exact_tuple", body: tuple.CanonicalBytes()},
		)
		if frameErr != nil {
			return evidenceDraft{}, frameErr
		}
		bodies[contractmodel.EvidenceProjectionResult] = projectionFrame
	} else if projectionErr != nil && primary == "" {
		primary = domain.ControlProjectionRejected
	}

	scope, err := buildScopeDrafts(input, result, post, measurements, enrolled, put)
	if err != nil {
		return evidenceDraft{}, err
	}
	return evidenceDraft{
		bodies: bodies, spawn: spawn, result: result, primary: primary,
		projection: tuple, projected: projected, captured: true, scope: scope,
	}, nil
}

func buildStartErrorEvidenceDraft(
	input cliExecutionInput,
	targetModel contractmodel.ContractExecutionTarget,
	spawn contractmodel.SpawnObservation,
	result processmechanics.Result,
	post inventorySnapshot,
	measurements scopeMeasurements,
) (evidenceDraft, error) {
	if !spawn.Valid() || spawn.State() != contractmodel.SpawnStartError || !targetModel.Valid() || !post.valid() ||
		input.prepared == nil || !result.Binding.Valid() ||
		result.Binding.String() != input.prepared.BindingDigest().String() || result.Started || result.PID != 0 ||
		result.ProcessGroupID != 0 || result.ProcessGroupOwned ||
		(result.SpawnAttempted && !result.PhysicalExecutionEntered) {
		return evidenceDraft{}, refuse(CodeEvidenceClosureFailed, "terminal start-error evidence inputs are invalid", nil)
	}
	bodies := make(map[contractmodel.EvidenceKind][]byte, 5)
	put := func(kind contractmodel.EvidenceKind, facts any) error {
		body, err := canonicalEvidence(kind, facts)
		if err == nil {
			bodies[kind] = body
		}
		return err
	}
	runtime := targetModel.Input().Runtime
	for kind, facts := range map[contractmodel.EvidenceKind]any{
		contractmodel.EvidenceMaterializationRevalidation: map[string]any{
			"target_digest": targetModel.Digest().String(), "before_identity": input.pre.identity,
			"after_identity": post.identity, "unchanged": input.pre.equal(post),
		},
		contractmodel.EvidenceRuntimeRevalidation: map[string]any{
			"admitted_executable_path": runtime.AdmittedExecutablePath,
			"executable_bytes_digest":  runtime.ExecutableBytesDigest.String(),
			"probe_program_digest":     runtime.ProbeProgramDigest.String(),
		},
		contractmodel.EvidenceTeardownResult: map[string]any{
			"binding_digest": result.Binding.String(), "diagnostic_code": result.DiagnosticCode,
			"physical_execution_entered": result.PhysicalExecutionEntered,
			"spawn_attempted":            result.SpawnAttempted, "started": result.Started,
			"pre_term_probe": result.PreTermProbe, "teardown_error": result.TeardownError,
		},
		contractmodel.EvidenceOrphanCheck: map[string]any{
			"final_group_probe_clean": result.FinalProbeClean, "orphan_risk": result.OrphanRisk,
		},
		contractmodel.EvidenceFinalizationMarker: map[string]any{
			"terminal_start_error_closed": true, "scope_canaries_closed": !measurements.Ambiguous,
			"import_canary_accepts": measurements.ImportAccepts, "service_canary_accepts": measurements.ServiceAccepts,
			"cli_invocation": input.invocationEvidence.facts(),
		},
	} {
		if err := put(kind, facts); err != nil {
			return evidenceDraft{}, err
		}
	}
	missing := []scopeDraft{
		{domain: contractmodel.ScopeTargetInventory, state: contractmodel.ScopeCheckMissing},
		{domain: contractmodel.ScopeChildBindings, state: contractmodel.ScopeCheckMissing},
		{domain: contractmodel.ScopeImportResolution, state: contractmodel.ScopeCheckMissing},
		{domain: contractmodel.ScopeServiceBindings, state: contractmodel.ScopeCheckMissing},
		{domain: contractmodel.ScopeSentinelInheritance, state: contractmodel.ScopeCheckMissing},
	}
	return evidenceDraft{
		bodies: bodies, spawn: spawn, result: result, primary: domain.ControlStartError,
		projected: false, captured: false, scope: missing,
	}, nil
}

func buildScopeDrafts(
	input cliExecutionInput,
	result processmechanics.Result,
	post inventorySnapshot,
	measurements scopeMeasurements,
	enrolled bool,
	put func(contractmodel.EvidenceKind, any) error,
) ([]scopeDraft, error) {
	unchanged := input.pre.equal(post)
	bindingsConsistent := result.Started && result.ProcessGroupOwned && result.ProcessGroupID == result.PID &&
		result.Binding.String() == input.prepared.BindingDigest().String()
	drafts := make([]scopeDraft, 0, 5)
	add := func(scopeDomain contractmodel.ScopeDomain, kind contractmodel.EvidenceKind, clean bool, violated bool, violation contractmodel.ScopeViolation, facts any) error {
		state := contractmodel.ScopeCheckMissing
		if violated {
			state = contractmodel.ScopeCheckViolated
		} else if clean {
			state = contractmodel.ScopeCheckClean
		}
		if state != contractmodel.ScopeCheckMissing {
			if err := put(kind, facts); err != nil {
				return err
			}
		}
		drafts = append(drafts, scopeDraft{domain: scopeDomain, state: state, kind: kind, violation: violation})
		return nil
	}
	if err := add(
		contractmodel.ScopeTargetInventory, contractmodel.EvidenceTargetInventory,
		enrolled && unchanged, false, contractmodel.ViolationTargetSourcePresent,
		map[string]any{
			"before_inventory_sha256": input.pre.identity, "after_inventory_sha256": post.identity,
			"before_entry_count": len(input.pre.entries), "after_entry_count": len(post.entries),
			"unchanged": unchanged, "reference_enrolled": enrolled,
		},
	); err != nil {
		return nil, err
	}
	if err := add(
		contractmodel.ScopeChildBindings, contractmodel.EvidenceChildBindings,
		enrolled && bindingsConsistent && input.invocationEvidence.validated(),
		input.invocationEvidence.contradictsBinding(), contractmodel.ViolationChildBinding,
		map[string]any{"binding_digest": result.Binding.String(), "pid": result.PID, "process_group_id": result.ProcessGroupID,
			"process_group_owned": result.ProcessGroupOwned, "candidate_invocation": input.invocationEvidence.facts()},
	); err != nil {
		return nil, err
	}
	importViolated := measurements.ImportAccepts > 0 || input.nodePathDeclared
	if err := add(
		contractmodel.ScopeImportResolution, contractmodel.EvidenceImportResolution,
		enrolled && unchanged && !importViolated && !measurements.Ambiguous, importViolated, contractmodel.ViolationImportResolution,
		map[string]any{"accepted_connections": measurements.ImportAccepts, "node_path_declared": input.nodePathDeclared,
			"outside_module_sha256": measurements.ModuleSHA256, "reference_enrolled": enrolled,
			"measurement_ambiguous": measurements.Ambiguous},
	); err != nil {
		return nil, err
	}
	serviceViolated := measurements.ServiceAccepts > 0
	if err := add(
		contractmodel.ScopeServiceBindings, contractmodel.EvidenceServiceBindings,
		enrolled && unchanged && !serviceViolated && !measurements.Ambiguous, serviceViolated, contractmodel.ViolationServiceBinding,
		map[string]any{"accepted_connections": measurements.ServiceAccepts, "reference_enrolled": enrolled,
			"measurement_ambiguous": measurements.Ambiguous},
	); err != nil {
		return nil, err
	}
	if err := add(
		contractmodel.ScopeSentinelInheritance, contractmodel.EvidenceSentinelInheritance,
		enrolled && !input.sentinelInherited, input.sentinelInherited, contractmodel.ViolationSentinelInherited,
		map[string]any{"named_parent_roster": parentSentinelRoster[:], "child_environment_omits_roster": !input.sentinelInherited},
	); err != nil {
		return nil, err
	}
	return drafts, nil
}

func (draft evidenceDraft) persistAndAssemble(
	ctxOwner store.ContractRunOwner,
	manifest store.PrivateRunManifest,
) (contractmodel.ClosedRunWitness, error) {
	if !manifest.Valid() || !draft.spawn.Valid() || len(draft.scope) != 5 {
		return contractmodel.ClosedRunWitness{}, refuse(CodeEvidenceClosureFailed, "evidence draft or private manifest is invalid", nil)
	}
	processKinds := []contractmodel.EvidenceKind{
		contractmodel.EvidenceMaterializationRevalidation,
		contractmodel.EvidenceRuntimeRevalidation,
	}
	if draft.spawn.State() == contractmodel.SpawnChildPIDObserved {
		processKinds = append(processKinds,
			contractmodel.EvidenceProcessResult,
			contractmodel.EvidenceWaitResult,
			contractmodel.EvidenceDrainResult,
		)
	}
	processKinds = append(processKinds,
		contractmodel.EvidenceTeardownResult,
		contractmodel.EvidenceOrphanCheck,
		contractmodel.EvidenceFinalizationMarker,
	)
	processRefs := make([]contractmodel.EvidenceRef, len(processKinds))
	for index, kind := range processKinds {
		ref, err := manifest.EvidenceRef(kind)
		if err != nil {
			return contractmodel.ClosedRunWitness{}, err
		}
		processRefs[index] = ref
	}
	var process contractmodel.ProcessClosure
	var err error
	if draft.primary == "" && !draft.result.TeardownError && !draft.result.OrphanRisk {
		process, err = contractmodel.NewCleanProcessClosure(processRefs)
	} else {
		process, err = contractmodel.NewControlledProcessClosure(
			draft.primary, draft.result.TeardownError, draft.result.OrphanRisk, processRefs,
		)
	}
	if err != nil {
		return contractmodel.ClosedRunWitness{}, err
	}
	observation := contractmodel.NewNoCapture()
	if draft.captured {
		captured, err := manifest.EvidenceRef(contractmodel.EvidenceCapturedObservation)
		if err != nil {
			return contractmodel.ClosedRunWitness{}, err
		}
		if draft.projected {
			projection, err := manifest.EvidenceRef(contractmodel.EvidenceProjectionResult)
			if err != nil {
				return contractmodel.ClosedRunWitness{}, err
			}
			observation, err = contractmodel.NewProjectedObservation(captured, projection, draft.projection)
			if err != nil {
				return contractmodel.ClosedRunWitness{}, err
			}
		} else {
			observation, err = contractmodel.NewCapturedUnprojected(captured)
			if err != nil {
				return contractmodel.ClosedRunWitness{}, err
			}
		}
	}
	checks := make([]contractmodel.ScopeCheck, len(draft.scope))
	for index, next := range draft.scope {
		switch next.state {
		case contractmodel.ScopeCheckMissing:
			checks[index], err = contractmodel.NewMissingScopeCheck(next.domain)
		case contractmodel.ScopeCheckClean:
			var ref contractmodel.EvidenceRef
			ref, err = manifest.EvidenceRef(next.kind)
			if err == nil {
				checks[index], err = contractmodel.NewCleanScopeCheck(next.domain, ref)
			}
		case contractmodel.ScopeCheckViolated:
			var ref contractmodel.EvidenceRef
			ref, err = manifest.EvidenceRef(next.kind)
			if err == nil {
				checks[index], err = contractmodel.NewViolatedScopeCheck(next.domain, ref, next.violation)
			}
		default:
			err = errors.New("unknown scope draft state")
		}
		if err != nil {
			return contractmodel.ClosedRunWitness{}, err
		}
	}
	scope, err := contractmodel.NewStandaloneScope(checks)
	if err != nil {
		return contractmodel.ClosedRunWitness{}, err
	}
	summary, err := manifest.Summary()
	if err != nil {
		return contractmodel.ClosedRunWitness{}, err
	}
	_ = ctxOwner
	return contractmodel.NewClosedRunWitness(draft.spawn, process, observation, scope, summary)
}

func projectCLIResult(input cliExecutionInput, result processmechanics.Result) (emitmodel.ExactTuple, []byte, error) {
	authority := input.view.Projection()
	fieldIDs := authority.FieldIDs()
	fields := make([]cli.CLIFieldID, len(fieldIDs))
	for index, fieldID := range fieldIDs {
		fields[index] = cli.CLIFieldID(fieldID)
	}
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
	if err != nil || definition.Digest() != authority.Digest() ||
		definition.Binding().Digest() != authority.Binding().Digest() ||
		!bytes.Equal(definition.CanonicalBytes(), authority.CanonicalBytes()) {
		return emitmodel.ExactTuple{}, nil, errors.Join(err, errors.New("CLI projection authority did not reconstruct"))
	}
	var completion cli.CLIProjectionCompletion
	if result.ExitSignal != "" {
		completion, err = cli.NewSignaledProjectionCompletion(result.ExitSignal)
	} else {
		completion, err = cli.NewExitedProjectionCompletion(result.ExitCode)
	}
	if err != nil {
		return emitmodel.ExactTuple{}, nil, err
	}
	projectionBytes, err := definition.ProjectInput(cli.CLIProjectionInput{
		Completion: completion, Stdout: result.Stdout, Stderr: result.Stderr,
	})
	if err != nil {
		return emitmodel.ExactTuple{}, nil, err
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		return emitmodel.ExactTuple{}, nil, err
	}
	translated, err := resolved.Translate(projectionBytes)
	if err != nil {
		return emitmodel.ExactTuple{}, nil, err
	}
	predicate := input.targetIdentity.bundle.Predicate()
	if translated.ProfileDigest() != predicate.PortableProfileDigest() {
		return emitmodel.ExactTuple{}, nil, errors.New("translated projection profile differs from the predicate")
	}
	selected := predicate.SelectedFields()
	exactFields := make([]emitmodel.ExactField, len(selected))
	for index, fieldID := range selected {
		value, present := translated.Value(fieldID)
		if !present {
			return emitmodel.ExactTuple{}, nil, errors.New("translated tuple omitted a selected predicate field")
		}
		exact, err := emitmodel.NewExactValue(value)
		if err != nil {
			return emitmodel.ExactTuple{}, nil, err
		}
		exactFields[index], err = emitmodel.NewExactField(fieldID, exact)
		if err != nil {
			return emitmodel.ExactTuple{}, nil, err
		}
	}
	tuple, err := emitmodel.NewExactTuple(exactFields)
	return tuple, projectionBytes, err
}

func mechanicsPrimary(value string) (domain.ControlReason, error) {
	if value == "" {
		return "", nil
	}
	reason := domain.ControlReason(value)
	if !reason.Valid() || reason == domain.ControlTeardownError || reason == domain.ControlOrphanRisk {
		return "", refuse(CodeEvidenceClosureFailed, "mechanics returned an unknown primary control", nil)
	}
	return reason, nil
}

func canonicalEvidence(kind contractmodel.EvidenceKind, facts any) ([]byte, error) {
	body, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           string(kind),
		"profile":        "C4_CLI_CANDIDATE_REFERENCE_V1",
		"facts":          facts,
	})
	if err != nil || len(body) == 0 || int64(len(body)) > privateEvidenceSummaryMaxBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence could not be canonicalized", err)
	}
	return body, nil
}
