//go:build darwin

package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"path/filepath"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/contractexec/scope"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	evidenceFrameMagic          = "COUNTERSHAPE_C5_PRIVATE_EVIDENCE_FRAME_V1\n"
	evidenceFrameVersion        = "private-evidence-frame/v1"
	evidenceSummaryMaxBytes     = int64(1 << 20)
	evidenceFrameOverhead       = int64(1 << 20)
	evidenceProjectionMaxBytes  = int64(3 << 20)
	evidenceStoreMaxBytes       = int64(64 << 20)
	evidenceMaximumBodyCount    = 15
	evidenceMaximumChannelBytes = int64(16 << 20)
	evidenceFramedBodyCount     = 3
	evidenceCanonicalBodyCount  = evidenceMaximumBodyCount - evidenceFramedBodyCount
)

type preparedTargetIdentity struct {
	model         targetModel
	bundle        emitmodel.ContractBundle
	roots         store.ConformanceAttemptRoots
	candidateRoot string
	markerPath    string
}

func newPreparedTargetIdentity(
	model targetModel,
	bundle emitmodel.ContractBundle,
	roots store.ConformanceAttemptRoots,
) (preparedTargetIdentity, error) {
	identity := preparedTargetIdentity{
		model: model, bundle: bundle, roots: roots,
		candidateRoot: filepath.Join(roots.CandidateParent(), "candidate"),
		markerPath:    roots.MarkerPath(),
	}
	if !identity.valid() {
		return preparedTargetIdentity{}, errors.New("prepared target identity is incomplete")
	}
	return identity, nil
}

func (identity preparedTargetIdentity) valid() bool {
	return identity.model.Valid() && identity.bundle.Valid() &&
		identity.bundle.Digest() == identity.model.ContractBundleDigest() &&
		filepath.IsAbs(identity.candidateRoot) &&
		filepath.Clean(identity.candidateRoot) == identity.candidateRoot &&
		identity.candidateRoot == filepath.Join(identity.roots.CandidateParent(), "candidate") &&
		identity.roots.AttemptRoot() != "" && identity.roots.FixtureRoot() != "" &&
		identity.roots.HomeRoot() != "" && identity.roots.TemporaryRoot() != "" &&
		identity.roots.StateRoot() != "" && identity.roots.EvidenceRoot() != "" &&
		identity.markerPath == identity.roots.MarkerPath() && identity.markerPath != ""
}

type evidenceCapacity struct {
	maximumUniqueBytes    int64
	maximumRequestBytes   int64
	maximumReadinessBytes int64
}

func preflightEvidenceCapacity(
	source contractsource.PortableSource,
	before scope.Inventory,
) (evidenceCapacity, error) {
	if !source.Valid() || !before.Valid() {
		return evidenceCapacity{}, refuse(CodeUnsupportedProfile, "private-evidence inputs are invalid", nil)
	}
	view, ok := source.HTTPView()
	budgets := source.Plan().Budgets()
	if !ok || budgets.StdoutBytes < 1 || budgets.StderrBytes < 1 ||
		budgets.StdoutBytes > evidenceMaximumChannelBytes ||
		budgets.StderrBytes > evidenceMaximumChannelBytes {
		return evidenceCapacity{}, refuse(CodeUnsupportedProfile, "private-evidence channels exceed the closed profile", nil)
	}
	maximumRequestBytes, err := maximumHTTPRequestBytes(view.Stimulus())
	if err != nil {
		return evidenceCapacity{}, refuse(
			CodeUnsupportedProfile,
			"maximum-width HTTP request did not encode",
			err,
		)
	}
	maximumReadinessBytes, err := maximumReadinessObservationBytes(view.Readiness())
	if err != nil {
		return evidenceCapacity{}, refuse(
			CodeUnsupportedProfile,
			"portable readiness ceiling is unavailable",
			err,
		)
	}
	maximum := int64(0)
	for _, next := range []int64{
		budgets.StdoutBytes,
		budgets.StderrBytes,
		view.Capture().OwnerResponseReadLimit(),
		maximumRequestBytes,
		maximumReadinessBytes,
		evidenceProjectionMaxBytes,
		evidenceCanonicalBodyCount * evidenceSummaryMaxBytes,
		2 * evidenceFrameOverhead,
	} {
		if next < 0 || maximum > math.MaxInt64-next {
			return evidenceCapacity{}, refuse(CodeUnsupportedProfile, "private-evidence capacity arithmetic overflowed", nil)
		}
		maximum += next
	}
	if maximum > evidenceStoreMaxBytes {
		return evidenceCapacity{}, refuse(CodeUnsupportedProfile, "private-evidence capacity exceeds the store ceiling", nil)
	}
	return evidenceCapacity{
		maximumUniqueBytes:    maximum,
		maximumRequestBytes:   maximumRequestBytes,
		maximumReadinessBytes: maximumReadinessBytes,
	}, nil
}

func maximumHTTPRequestBytes(stimulus counterhttp.HTTPStimulus) (int64, error) {
	request, err := counterhttp.EncodeRequest(stimulus, 65535)
	if err != nil || !request.Valid() {
		return 0, errors.Join(err, errors.New("maximum-width HTTP request is invalid"))
	}
	return int64(request.ByteLength()), nil
}

func maximumReadinessObservationBytes(
	readiness counterhttp.HTTPReadinessContract,
) (int64, error) {
	_, maximum, _, portable := readiness.PortableFrameProfile()
	converted := int64(maximum)
	if !portable || converted < 1 || converted == math.MaxInt64 {
		return 0, errors.New("portable readiness observation ceiling is invalid")
	}
	return converted + 1, nil
}

func (capacity evidenceCapacity) validate(bodies map[contractmodel.EvidenceKind][]byte) error {
	if capacity.maximumUniqueBytes < 1 || capacity.maximumUniqueBytes > evidenceStoreMaxBytes ||
		capacity.maximumRequestBytes < 1 || capacity.maximumReadinessBytes < 1 ||
		len(bodies) < 1 || len(bodies) > evidenceMaximumBodyCount {
		return errors.New("private-evidence capacity or body roster is invalid")
	}
	unique := make(map[[sha256.Size]byte][]byte, len(bodies))
	var aggregate int64
	for _, body := range bodies {
		if len(body) == 0 {
			return errors.New("private-evidence body is empty")
		}
		digest := sha256.Sum256(body)
		if previous, present := unique[digest]; present {
			if !bytes.Equal(previous, body) {
				return errors.New("private-evidence digest collision is ambiguous")
			}
			continue
		}
		count := int64(len(body))
		if aggregate > capacity.maximumUniqueBytes-count {
			return errors.New("private-evidence bodies exceed the pre-admitted envelope")
		}
		aggregate += count
		unique[digest] = body
	}
	return nil
}

type evidenceSegment struct {
	name string
	body []byte
}

func frameEvidence(
	purpose string,
	facts any,
	maximumBytes int64,
	segments ...evidenceSegment,
) ([]byte, error) {
	if purpose == "" || maximumBytes < 1 || maximumBytes > evidenceStoreMaxBytes ||
		len(segments) < 1 || len(segments) > 3 {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame inputs are invalid", nil)
	}
	seen := make(map[string]struct{}, len(segments))
	segmentFacts := make([]map[string]any, len(segments))
	var payloadBytes int64
	for index, segment := range segments {
		if segment.name == "" {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence segment name is absent", nil)
		}
		if _, duplicate := seen[segment.name]; duplicate {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence segment name repeats", nil)
		}
		seen[segment.name] = struct{}{}
		count := int64(len(segment.body))
		if payloadBytes > math.MaxInt64-count {
			return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame arithmetic overflowed", nil)
		}
		payloadBytes += count
		digest := sha256.Sum256(segment.body)
		segmentFacts[index] = map[string]any{
			"name": segment.name, "bytes": count, "sha256": hex.EncodeToString(digest[:]),
		}
	}
	metadata, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           "C5PrivateEvidenceFrame",
		"frame_version":  evidenceFrameVersion,
		"purpose":        purpose,
		"facts":          facts,
		"segments":       segmentFacts,
	})
	if err != nil || len(metadata) == 0 || int64(len(metadata)) > evidenceSummaryMaxBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence metadata exceeds its ceiling", err)
	}
	framingBytes := int64(len(evidenceFrameMagic) + 8 + len(metadata) + 8*len(segments))
	if framingBytes > maximumBytes || payloadBytes > maximumBytes-framingBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence frame exceeds its ceiling", nil)
	}
	body := make([]byte, 0, int(framingBytes+payloadBytes))
	body = append(body, evidenceFrameMagic...)
	body = binary.BigEndian.AppendUint64(body, uint64(len(metadata)))
	body = append(body, metadata...)
	for _, segment := range segments {
		body = binary.BigEndian.AppendUint64(body, uint64(len(segment.body)))
		body = append(body, segment.body...)
	}
	return body, nil
}

type scopeDraft struct {
	domain     contractmodel.ScopeDomain
	state      contractmodel.ScopeCheckState
	kind       contractmodel.EvidenceKind
	violation  contractmodel.ScopeViolation
	diagnostic scope.Diagnostic
}

type evidenceDraft struct {
	bodies     map[contractmodel.EvidenceKind][]byte
	spawn      contractmodel.SpawnObservation
	result     serviceResult
	primary    domain.ControlReason
	projection emitmodel.ExactTuple
	projected  bool
	captured   bool
	scope      []scopeDraft
}

func buildChildEvidenceDraft(
	input executionInput,
	target targetModel,
	spawn contractmodel.SpawnObservation,
	result serviceResult,
	after scope.Inventory,
	measurements scope.Measurements,
) (evidenceDraft, error) {
	pid, child := spawn.PID()
	if !spawn.Valid() || !child || !target.Valid() || !after.Valid() || !result.binding.Valid() ||
		!result.process.physicalExecutionEntered || !result.process.spawnAttempted ||
		!result.process.started || pid != int64(result.process.pid) ||
		result.binding != input.binding {
		return evidenceDraft{}, refuse(CodeEvidenceClosureFailed, "terminal HTTP child evidence inputs are invalid", nil)
	}
	bodies := make(map[contractmodel.EvidenceKind][]byte, evidenceMaximumBodyCount)
	put := func(kind contractmodel.EvidenceKind, facts any) error {
		body, err := canonicalEvidence(kind, facts)
		if err == nil {
			bodies[kind] = body
		}
		return err
	}
	runtime := target.Input().Runtime
	if err := put(contractmodel.EvidenceMaterializationRevalidation, map[string]any{
		"target_digest":              target.Digest().String(),
		"before_inventory_sha256":    input.before.Digest(),
		"after_inventory_sha256":     after.Digest(),
		"before_entry_count":         input.before.EntryCount(),
		"after_entry_count":          after.EntryCount(),
		"unchanged":                  input.before.Equal(after),
		"reference_fixture_enrolled": input.before.ReferenceHTTPFixture(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceRuntimeRevalidation, map[string]any{
		"admitted_executable_path":    runtime.AdmittedExecutablePath,
		"executable_bytes_digest":     runtime.ExecutableBytesDigest.String(),
		"executable_mode":             runtime.ExecutableMode,
		"executable_byte_count":       runtime.ExecutableByteCount,
		"measured_exec_path":          runtime.MeasuredProcessExecPath,
		"version":                     runtime.Version,
		"major":                       runtime.Major,
		"platform":                    runtime.Platform,
		"architecture":                runtime.Architecture,
		"probe_program_digest":        runtime.ProbeProgramDigest.String(),
		"terminal_revalidation_error": result.process.runtimeRevalidationError,
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceWaitResult, map[string]any{
		"child_waited": result.process.directChildWaited,
		"exit_code":    result.process.exitCode,
		"exit_signal":  result.process.exitSignal,
		"wait_error":   result.process.waitError,
	}); err != nil {
		return evidenceDraft{}, err
	}
	drainFrame, err := buildProcessDrainEvidence(result)
	if err != nil {
		return evidenceDraft{}, err
	}
	bodies[contractmodel.EvidenceDrainResult] = drainFrame
	if err := put(contractmodel.EvidenceTeardownResult, map[string]any{
		"pre_term_probe":               result.process.preTermProbe,
		"term_sent":                    result.process.termSent,
		"kill_sent":                    result.process.killSent,
		"teardown_error":               result.process.teardownError,
		"descriptor_close_diagnostics": result.process.descriptorCloseDiagnostics(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceOrphanCheck, map[string]any{
		"final_group_probe_clean": result.process.finalProbeClean,
		"final_group_probe_error": result.process.finalProbeError,
		"orphan_risk":             result.process.orphanRisk,
	}); err != nil {
		return evidenceDraft{}, err
	}
	captured := len(result.exchange.responseWire) > 0
	if captured {
		captureFrame, frameErr := frameEvidence(
			"RAW_HTTP_EXCHANGE",
			map[string]any{
				"endpoint":           result.readiness.endpoint,
				"request_written":    result.exchange.requestWritten,
				"request_complete":   result.exchange.requestComplete,
				"response_observed":  result.exchange.responseObserved,
				"response_overflow":  result.exchange.responseOverflow,
				"response_parsed":    result.exchange.responseParsed,
				"invocation_receipt": input.receiptFinding.facts(),
			},
			int64(len(result.exchange.requestWire.Bytes()))+
				int64(len(result.exchange.responseWire))+evidenceFrameOverhead,
			evidenceSegment{name: "request_wire", body: result.exchange.requestWire.Bytes()},
			evidenceSegment{name: "response_wire", body: result.exchange.responseWire},
		)
		if frameErr != nil {
			return evidenceDraft{}, frameErr
		}
		bodies[contractmodel.EvidenceCapturedObservation] = captureFrame
	}
	tuple, projectionFrame, projected, err := resolveHTTPProjection(input, &result)
	if err != nil {
		return evidenceDraft{}, err
	}
	if projected {
		bodies[contractmodel.EvidenceProjectionResult] = projectionFrame
	}
	if err := put(
		contractmodel.EvidenceProcessResult,
		processResultEvidenceFacts(result),
	); err != nil {
		return evidenceDraft{}, err
	}
	findings := scope.Evaluate(scope.AssessmentInput{
		Before: input.before, After: after,
		ChildStarted:        result.process.started,
		ChildPID:            result.process.pid,
		ProcessGroupID:      result.process.processGroupID,
		ProcessGroupOwned:   result.process.processGroupOwned,
		PreparedBinding:     input.binding.String(),
		ObservedBinding:     result.binding.String(),
		ReceiptValidated:    input.receiptFinding.validated(),
		ReceiptContradicted: input.receiptFinding.contradictsBinding(),
		ReadinessAccepted:   result.readiness.accepted,
		ReadinessEndpoint:   result.readiness.endpoint,
		ConnectionAttempts:  result.exchange.connectionAttempts,
		SentinelInherited:   input.sentinelInherited,
		NodePathDeclared:    input.nodePathDeclared,
		Measurements:        measurements,
	})
	drafts, diagnostics, err := materializeScopeFindings(findings, put)
	if err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceFinalizationMarker, map[string]any{
		"terminal_facts_closed":        true,
		"candidate_inventory_reopened": after.Valid(),
		"scope_probe_ambiguous":        measurements.Ambiguous,
		"scope_diagnostics":            diagnostics,
		"invocation_receipt":           input.receiptFinding.facts(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	return evidenceDraft{
		bodies: bodies, spawn: spawn, result: result, primary: result.process.primary,
		projection: tuple, projected: projected, captured: captured, scope: drafts,
	}, nil
}

func buildProcessDrainEvidence(result serviceResult) ([]byte, error) {
	return frameEvidence(
		"PROCESS_DRAINS",
		map[string]any{
			"stdout_drained":       result.process.stdoutDrained,
			"stderr_drained":       result.process.stderrDrained,
			"stdout_observed":      result.process.stdoutObserved,
			"stderr_observed":      result.process.stderrObserved,
			"stdout_overflow":      result.process.stdoutOverflow,
			"stderr_overflow":      result.process.stderrOverflow,
			"readiness_bytes":      result.readiness.bytesObserved,
			"readiness_eof":        result.readiness.eofObserved,
			"readiness_accepted":   result.readiness.accepted,
			"readiness_diagnostic": result.readiness.diagnosticCode,
		},
		int64(len(result.process.stdout))+int64(len(result.process.stderr))+
			int64(len(result.readiness.frameBytes))+evidenceFrameOverhead,
		evidenceSegment{name: "stdout", body: result.process.stdout},
		evidenceSegment{name: "stderr", body: result.process.stderr},
		evidenceSegment{name: "readiness_frame", body: result.readiness.frameBytes},
	)
}

func resolveHTTPProjection(
	input executionInput,
	result *serviceResult,
) (emitmodel.ExactTuple, []byte, bool, error) {
	if result == nil {
		return emitmodel.ExactTuple{}, nil, false, refuse(
			CodeEvidenceClosureFailed,
			"HTTP projection result is absent",
			nil,
		)
	}
	if result.process.primary != "" || !result.exchange.responseParsed {
		return emitmodel.ExactTuple{}, nil, false, nil
	}
	tuple, projectionBytes, err := projectHTTPResult(input, result.exchange.response)
	if err != nil {
		result.process.primary = domain.ControlProjectionRejected
		result.process.diagnosticCode = firstDiagnostic(
			result.process.diagnosticCode,
			"HTTP_PROJECTION_REJECTED",
		)
		return emitmodel.ExactTuple{}, nil, false, nil
	}
	projectionFrame, err := frameEvidence(
		"HTTP_PROJECTION",
		map[string]any{
			"portable_profile_digest": input.targetIdentity.bundle.Predicate().PortableProfileDigest().String(),
		},
		evidenceProjectionMaxBytes,
		evidenceSegment{name: "projection", body: projectionBytes},
		evidenceSegment{name: "exact_tuple", body: tuple.CanonicalBytes()},
	)
	if err != nil {
		return emitmodel.ExactTuple{}, nil, false, err
	}
	return tuple, projectionFrame, true, nil
}

func processResultEvidenceFacts(result serviceResult) map[string]any {
	return map[string]any{
		"physical_execution_entered": result.process.physicalExecutionEntered,
		"spawn_attempted":            result.process.spawnAttempted,
		"started":                    result.process.started,
		"pid":                        result.process.pid,
		"process_group_id":           result.process.processGroupID,
		"process_group_owned":        result.process.processGroupOwned,
		"binding_digest":             result.binding.String(),
		"primary":                    result.process.primary,
		"diagnostic_code":            result.process.diagnosticCode,
		"readiness_protocol":         result.readiness.protocol,
		"readiness_bytes_observed":   result.readiness.bytesObserved,
		"readiness_eof_observed":     result.readiness.eofObserved,
		"readiness_accepted":         result.readiness.accepted,
		"readiness_diagnostic_code":  result.readiness.diagnosticCode,
		"readiness_endpoint":         result.readiness.endpoint,
		"connection_attempts":        result.exchange.connectionAttempts,
	}
}

func materializeScopeFindings(
	findings []scope.Finding,
	put func(contractmodel.EvidenceKind, any) error,
) ([]scopeDraft, []map[string]any, error) {
	if len(findings) != 5 || put == nil {
		return nil, nil, refuse(
			CodeEvidenceClosureFailed,
			"scope finding roster is incomplete",
			nil,
		)
	}
	drafts := make([]scopeDraft, len(findings))
	diagnostics := make([]map[string]any, len(findings))
	for index, finding := range findings {
		if !finding.Valid() {
			return nil, nil, refuse(
				CodeEvidenceClosureFailed,
				"scope assessment produced an invalid finding",
				nil,
			)
		}
		drafts[index] = scopeDraft{
			domain: finding.Domain, state: finding.State, kind: finding.Kind,
			violation: finding.Violation, diagnostic: finding.Diagnostic,
		}
		diagnostics[index] = map[string]any{
			"domain":     finding.Domain,
			"state":      finding.State,
			"diagnostic": finding.Diagnostic.Code(),
		}
		if finding.State != contractmodel.ScopeCheckMissing {
			if err := put(finding.Kind, finding.Facts); err != nil {
				return nil, nil, err
			}
		}
	}
	return drafts, diagnostics, nil
}

func buildStartErrorEvidenceDraft(
	input executionInput,
	target targetModel,
	spawn contractmodel.SpawnObservation,
	result serviceResult,
	after scope.Inventory,
	measurements scope.Measurements,
) (evidenceDraft, error) {
	if !spawn.Valid() || spawn.State() != contractmodel.SpawnStartError ||
		!target.Valid() || !after.Valid() || !result.binding.Valid() ||
		result.binding != input.binding || result.process.started || result.process.pid != 0 {
		return evidenceDraft{}, refuse(CodeEvidenceClosureFailed, "terminal start-error evidence inputs are invalid", nil)
	}
	bodies := make(map[contractmodel.EvidenceKind][]byte, evidenceMaximumBodyCount)
	put := func(kind contractmodel.EvidenceKind, facts any) error {
		body, err := canonicalEvidence(kind, facts)
		if err == nil {
			bodies[kind] = body
		}
		return err
	}
	runtime := target.Input().Runtime
	for kind, facts := range map[contractmodel.EvidenceKind]any{
		contractmodel.EvidenceMaterializationRevalidation: map[string]any{
			"target_digest":           target.Digest().String(),
			"before_inventory_sha256": input.before.Digest(),
			"after_inventory_sha256":  after.Digest(),
			"unchanged":               input.before.Equal(after),
		},
		contractmodel.EvidenceRuntimeRevalidation: map[string]any{
			"admitted_executable_path":    runtime.AdmittedExecutablePath,
			"executable_bytes_digest":     runtime.ExecutableBytesDigest.String(),
			"probe_program_digest":        runtime.ProbeProgramDigest.String(),
			"terminal_revalidation_error": result.process.runtimeRevalidationError,
		},
		contractmodel.EvidenceTeardownResult: map[string]any{
			"binding_digest":               result.binding.String(),
			"diagnostic_code":              result.process.diagnosticCode,
			"physical_execution_entered":   result.process.physicalExecutionEntered,
			"spawn_attempted":              result.process.spawnAttempted,
			"started":                      result.process.started,
			"start_error":                  result.process.waitError,
			"teardown_error":               result.process.teardownError,
			"descriptor_close_diagnostics": result.process.descriptorCloseDiagnostics(),
		},
		contractmodel.EvidenceOrphanCheck: map[string]any{
			"final_group_probe_clean": result.process.finalProbeClean,
			"orphan_risk":             result.process.orphanRisk,
		},
	} {
		if err := put(kind, facts); err != nil {
			return evidenceDraft{}, err
		}
	}
	findings := scope.Evaluate(scope.AssessmentInput{
		Before: input.before, After: after,
		ChildStarted:        false,
		PreparedBinding:     input.binding.String(),
		ObservedBinding:     result.binding.String(),
		ReceiptValidated:    input.receiptFinding.validated(),
		ReceiptContradicted: input.receiptFinding.contradictsBinding(),
		ReadinessAccepted:   false,
		SentinelInherited:   input.sentinelInherited,
		NodePathDeclared:    input.nodePathDeclared,
		Measurements:        measurements,
	})
	findings = conservativeStartErrorFindings(findings)
	drafts, diagnostics, err := materializeScopeFindings(findings, put)
	if err != nil {
		return evidenceDraft{}, err
	}
	if err := put(contractmodel.EvidenceFinalizationMarker, map[string]any{
		"terminal_start_error_closed": true,
		"scope_probe_ambiguous":       measurements.Ambiguous,
		"scope_diagnostics":           diagnostics,
		"invocation_receipt":          input.receiptFinding.facts(),
	}); err != nil {
		return evidenceDraft{}, err
	}
	return evidenceDraft{
		bodies: bodies, spawn: spawn, result: result, primary: domain.ControlStartError,
		scope: drafts,
	}, nil
}

func conservativeStartErrorFindings(findings []scope.Finding) []scope.Finding {
	result := append([]scope.Finding(nil), findings...)
	for index := 1; index < len(result); index++ {
		if result[index].State == contractmodel.ScopeCheckViolated {
			continue
		}
		result[index] = scope.Finding{
			Domain:     result[index].Domain,
			State:      contractmodel.ScopeCheckMissing,
			Kind:       result[index].Kind,
			Diagnostic: scope.NewDiagnostic(scope.CodeProbeAbsent),
		}
	}
	return result
}

func (draft evidenceDraft) assemble(manifest store.PrivateRunManifest) (contractmodel.ClosedRunWitness, error) {
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
	if draft.primary == "" && !draft.result.process.teardownError && !draft.result.process.orphanRisk {
		process, err = contractmodel.NewCleanProcessClosure(processRefs)
	} else {
		process, err = contractmodel.NewControlledProcessClosure(
			draft.primary, draft.result.process.teardownError, draft.result.process.orphanRisk, processRefs,
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
	standalone, err := contractmodel.NewStandaloneScope(checks)
	if err != nil {
		return contractmodel.ClosedRunWitness{}, err
	}
	summary, err := manifest.Summary()
	if err != nil {
		return contractmodel.ClosedRunWitness{}, err
	}
	return contractmodel.NewClosedRunWitness(draft.spawn, process, observation, standalone, summary)
}

func projectHTTPResult(
	input executionInput,
	response counterhttp.HTTPResponse,
) (emitmodel.ExactTuple, []byte, error) {
	authority := input.view.Projection()
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil || !definition.Valid() || definition.Digest() != authority.Digest() ||
		definition.Binding().Digest() != authority.Binding().Digest() ||
		!bytes.Equal(definition.CanonicalBytes(), authority.CanonicalBytes()) ||
		!bytes.Equal(definition.Binding().CanonicalBytes(), authority.Binding().CanonicalBytes()) {
		return emitmodel.ExactTuple{}, nil, errors.Join(err, errors.New("HTTP projection authority did not reconstruct"))
	}
	projectionBytes, err := counterhttp.ProjectParsedResponse(response, input.targetIdentity.roots.StateRoot())
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
		return emitmodel.ExactTuple{}, nil, errors.New("translated HTTP profile differs from the predicate")
	}
	selected := predicate.SelectedFields()
	exactFields := make([]emitmodel.ExactField, len(selected))
	for index, fieldID := range selected {
		value, present := translated.Value(fieldID)
		if !present {
			return emitmodel.ExactTuple{}, nil, errors.New("translated HTTP tuple omitted a selected field")
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

func canonicalEvidence(kind contractmodel.EvidenceKind, facts any) ([]byte, error) {
	body, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           string(kind),
		"profile":        "C5_HTTP_CHILD_BIND_REFERENCE_V1",
		"facts":          facts,
	})
	if err != nil || len(body) == 0 || int64(len(body)) > evidenceSummaryMaxBytes {
		return nil, refuse(CodeEvidenceClosureFailed, "private evidence could not be canonicalized", err)
	}
	return body, nil
}
