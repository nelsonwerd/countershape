package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func ParseFinalizedContractRun(
	exact []byte,
	expected domain.Digest,
	target ContractExecutionTarget,
) (FinalizedContractRun, error) {
	if !target.Valid() {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "parent target is invalid", nil)
	}
	root, err := parseExactObject(exact, RunKind, expected, CodeInvalidRun, CodeRunDigest)
	if err != nil {
		return FinalizedContractRun{}, err
	}
	if err := requireRoster(root, "schema_version", "kind", "run_version", "publication_scope",
		"contract_execution_target_digest", "attempt_artifact_digest", "start_claim_ref", "closed_run_witness"); err != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run root roster differs", err)
	}
	schema, _ := textMember(root, "schema_version")
	kind, _ := textMember(root, "kind")
	version, _ := textMember(root, "run_version")
	scope, _ := textMember(root, "publication_scope")
	if schema != domain.SchemaVersion || kind != RunKind || version != RunVersionV1 || scope != RunPublicationScopeV1 {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run constants differ", nil)
	}
	targetDigest, targetErr := digestMember(root, "contract_execution_target_digest")
	attemptDigest, attemptErr := digestMember(root, "attempt_artifact_digest")
	if targetErr != nil || attemptErr != nil || targetDigest != target.Digest() || attemptDigest != target.AttemptArtifactDigest() {
		return FinalizedContractRun{}, refuse(CodeTargetRunMismatch, "run does not name the exact target and attempt", nil)
	}
	claimValue, err := member(root, "start_claim_ref")
	if err != nil || requireRoster(claimValue, "kind", "digest") != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "start-claim reference roster differs", err)
	}
	claimKind, _ := textMember(claimValue, "kind")
	claimDigest, claimErr := digestMember(claimValue, "digest")
	if claimKind != "StartClaim" || claimErr != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "start-claim reference is mistyped", claimErr)
	}
	witnessValue, err := member(root, "closed_run_witness")
	if err != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "closed-run witness is missing", err)
	}
	witness, err := parseClosedRunWitness(witnessValue)
	if err != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "closed-run witness is invalid", err)
	}
	rebuilt, err := NewFinalizedContractRun(target, claimDigest, witness)
	if err != nil || rebuilt.digest != expected || !bytes.Equal(rebuilt.canonical, exact) {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run does not reconstruct exactly", err)
	}
	return rebuilt, nil
}

func parseClosedRunWitness(value canon.Value) (ClosedRunWitness, error) {
	if err := requireRoster(value, "witness_version", "spawn_observation", "process_closure", "observation", "standalone_scope", "private_evidence"); err != nil {
		return ClosedRunWitness{}, err
	}
	version, _ := textMember(value, "witness_version")
	if version != ClosedRunWitnessVersionV1 {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "witness version differs", nil)
	}
	spawnValue, _ := member(value, "spawn_observation")
	spawn, err := parseSpawnObservation(spawnValue)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	processValue, _ := member(value, "process_closure")
	process, err := parseProcessClosure(processValue)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	observationValue, _ := member(value, "observation")
	observation, err := parseCaptureObservation(observationValue)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	scopeValue, _ := member(value, "standalone_scope")
	standalone, encodedScopeState, err := parseStandaloneScope(scopeValue)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	if standalone.state != encodedScopeState {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "encoded standalone state differs from its five checks", nil)
	}
	privateValue, _ := member(value, "private_evidence")
	privateManifest, err := parsePrivateManifest(privateValue)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	witness, err := NewClosedRunWitness(spawn, process, observation, standalone, privateManifest)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, witness.canonical) {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "witness does not reconstruct exactly", err)
	}
	return witness, nil
}

func parseSpawnObservation(value canon.Value) (SpawnObservation, error) {
	status, err := textMember(value, "status")
	if err != nil {
		return SpawnObservation{}, err
	}
	switch SpawnObservationState(status) {
	case SpawnStartError:
		if err := requireRoster(value, "status", "error_code"); err != nil {
			return SpawnObservation{}, err
		}
		code, err := textMember(value, "error_code")
		if err != nil {
			return SpawnObservation{}, err
		}
		return NewStartErrorObservation(code)
	case SpawnChildPIDObserved:
		if err := requireRoster(value, "status", "pid"); err != nil {
			return SpawnObservation{}, err
		}
		pid, err := integerMember(value, "pid")
		if err != nil {
			return SpawnObservation{}, err
		}
		return NewChildPIDObservation(pid)
	default:
		return SpawnObservation{}, refuse(CodeInvalidWitness, "spawn observation status is unknown", nil)
	}
}

func parseProcessClosure(value canon.Value) (ProcessClosure, error) {
	if err := requireRoster(value, "status", "primary_reason", "cleanup_controls", "evidence_refs"); err != nil {
		return ProcessClosure{}, err
	}
	status, _ := textMember(value, "status")
	primaryText, _ := textMember(value, "primary_reason")
	primary := domain.ControlReason("")
	if primaryText != "NONE" {
		primary = domain.ControlReason(primaryText)
	}
	cleanupValues, err := arrayMember(value, "cleanup_controls")
	if err != nil || len(cleanupValues) > 2 {
		return ProcessClosure{}, refuse(CodeInvalidWitness, "cleanup-control roster is invalid", err)
	}
	teardown := false
	orphan := false
	for index, entry := range cleanupValues {
		text, ok := entry.Text()
		if !ok || (index == 0 && text != string(domain.ControlTeardownError) && text != string(domain.ControlOrphanRisk)) ||
			(index == 1 && text != string(domain.ControlOrphanRisk)) {
			return ProcessClosure{}, refuse(CodeInvalidWitness, "cleanup controls differ from teardown-then-orphan order", nil)
		}
		if text == string(domain.ControlTeardownError) {
			teardown = true
		} else {
			orphan = true
		}
	}
	evidenceValues, err := arrayMember(value, "evidence_refs")
	if err != nil {
		return ProcessClosure{}, err
	}
	evidence := make([]EvidenceRef, len(evidenceValues))
	for index, entry := range evidenceValues {
		evidence[index], err = parseEvidenceRef(entry)
		if err != nil {
			return ProcessClosure{}, err
		}
	}
	switch ProcessClosureState(status) {
	case ProcessClean:
		if primaryText != "NONE" || teardown || orphan {
			return ProcessClosure{}, refuse(CodeInvalidWitness, "clean process closure contains control facts", nil)
		}
		return NewCleanProcessClosure(evidence)
	case ProcessControlled:
		return NewControlledProcessClosure(primary, teardown, orphan, evidence)
	default:
		return ProcessClosure{}, refuse(CodeInvalidWitness, "process closure status is unknown", nil)
	}
}

func parseCaptureObservation(value canon.Value) (CaptureObservation, error) {
	status, err := textMember(value, "status")
	if err != nil {
		return CaptureObservation{}, err
	}
	switch CaptureState(status) {
	case CaptureNone:
		if err := requireRoster(value, "status"); err != nil {
			return CaptureObservation{}, err
		}
		return NewNoCapture(), nil
	case CaptureUnprojected:
		if err := requireRoster(value, "status", "captured_observation_ref"); err != nil {
			return CaptureObservation{}, err
		}
		refValue, _ := member(value, "captured_observation_ref")
		ref, err := parseEvidenceRef(refValue)
		if err != nil {
			return CaptureObservation{}, err
		}
		return NewCapturedUnprojected(ref)
	case CaptureProjected:
		if err := requireRoster(value, "status", "captured_observation_ref", "projection_result_ref", "observed_tuple"); err != nil {
			return CaptureObservation{}, err
		}
		capturedValue, _ := member(value, "captured_observation_ref")
		captured, err := parseEvidenceRef(capturedValue)
		if err != nil {
			return CaptureObservation{}, err
		}
		projectionValue, _ := member(value, "projection_result_ref")
		projection, err := parseEvidenceRef(projectionValue)
		if err != nil {
			return CaptureObservation{}, err
		}
		tupleValue, _ := member(value, "observed_tuple")
		tuple, err := parseExactTupleValue(tupleValue)
		if err != nil {
			return CaptureObservation{}, err
		}
		return NewProjectedObservation(captured, projection, tuple)
	default:
		return CaptureObservation{}, refuse(CodeInvalidWitness, "capture status is unknown", nil)
	}
}

func parseStandaloneScope(value canon.Value) (StandaloneScope, ScopeState, error) {
	if err := requireRoster(value, "status", "checks"); err != nil {
		return StandaloneScope{}, "", err
	}
	stateText, _ := textMember(value, "status")
	state := ScopeState(stateText)
	if state != ScopeComplete && state != ScopePartial && state != ScopeViolated {
		return StandaloneScope{}, "", refuse(CodeInvalidWitness, "standalone scope status is unknown", nil)
	}
	entries, err := arrayMember(value, "checks")
	if err != nil {
		return StandaloneScope{}, "", err
	}
	checks := make([]ScopeCheck, len(entries))
	for index, entry := range entries {
		checks[index], err = parseScopeCheck(entry)
		if err != nil {
			return StandaloneScope{}, "", err
		}
	}
	scope, err := NewStandaloneScope(checks)
	return scope, state, err
}

func parseScopeCheck(value canon.Value) (ScopeCheck, error) {
	domainText, err := textMember(value, "domain")
	if err != nil {
		return ScopeCheck{}, err
	}
	stateText, err := textMember(value, "status")
	if err != nil {
		return ScopeCheck{}, err
	}
	scopeDomain := ScopeDomain(domainText)
	switch ScopeCheckState(stateText) {
	case ScopeCheckMissing:
		if err := requireRoster(value, "domain", "status"); err != nil {
			return ScopeCheck{}, err
		}
		return NewMissingScopeCheck(scopeDomain)
	case ScopeCheckClean:
		if err := requireRoster(value, "domain", "status", "evidence_ref"); err != nil {
			return ScopeCheck{}, err
		}
		refValue, _ := member(value, "evidence_ref")
		ref, err := parseEvidenceRef(refValue)
		if err != nil {
			return ScopeCheck{}, err
		}
		return NewCleanScopeCheck(scopeDomain, ref)
	case ScopeCheckViolated:
		if err := requireRoster(value, "domain", "status", "evidence_ref", "violation"); err != nil {
			return ScopeCheck{}, err
		}
		refValue, _ := member(value, "evidence_ref")
		ref, err := parseEvidenceRef(refValue)
		if err != nil {
			return ScopeCheck{}, err
		}
		violation, err := textMember(value, "violation")
		if err != nil {
			return ScopeCheck{}, err
		}
		return NewViolatedScopeCheck(scopeDomain, ref, ScopeViolation(violation))
	default:
		return ScopeCheck{}, refuse(CodeInvalidWitness, "scope check status is unknown", nil)
	}
}

func parsePrivateManifest(value canon.Value) (PrivateManifestSummary, error) {
	if err := requireRoster(value, "profile", "manifest_ref", "blob_count", "aggregate_byte_count", "retention_at_finalization", "default_export"); err != nil {
		return PrivateManifestSummary{}, err
	}
	profile, _ := textMember(value, "profile")
	retention, _ := textMember(value, "retention_at_finalization")
	defaultExport, _ := textMember(value, "default_export")
	if profile != PrivateManifestProfileV1 || retention != PrivateRetentionV1 || defaultExport != PrivateDefaultExportV1 {
		return PrivateManifestSummary{}, refuse(CodeInvalidWitness, "private manifest constants differ", nil)
	}
	refValue, _ := member(value, "manifest_ref")
	ref, err := parseEvidenceRef(refValue)
	if err != nil {
		return PrivateManifestSummary{}, err
	}
	blobCount, err := integerMember(value, "blob_count")
	if err != nil {
		return PrivateManifestSummary{}, err
	}
	byteCount, err := integerMember(value, "aggregate_byte_count")
	if err != nil {
		return PrivateManifestSummary{}, err
	}
	return NewPrivateManifestSummary(ref, blobCount, byteCount)
}

func parseEvidenceRef(value canon.Value) (EvidenceRef, error) {
	if err := requireRoster(value, "kind", "digest"); err != nil {
		return EvidenceRef{}, err
	}
	kind, _ := textMember(value, "kind")
	digest, err := digestMember(value, "digest")
	if err != nil {
		return EvidenceRef{}, err
	}
	return newEvidenceRef(EvidenceKind(kind), digest)
}
