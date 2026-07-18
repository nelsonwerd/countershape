package model

import (
	"bytes"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type SpawnObservation struct {
	state     SpawnObservationState
	errorCode string
	pid       int64
}

func NewStartErrorObservation(code string) (SpawnObservation, error) {
	if !validStartErrorCode(code) {
		return SpawnObservation{}, refuse(CodeInvalidWitness, "start-error code is outside the closed machine profile", nil)
	}
	return SpawnObservation{state: SpawnStartError, errorCode: code}, nil
}

func NewChildPIDObservation(pid int64) (SpawnObservation, error) {
	if pid < 1 || pid > 2147483647 {
		return SpawnObservation{}, refuse(CodeInvalidWitness, "observed child PID is outside the exact integer profile", nil)
	}
	return SpawnObservation{state: SpawnChildPIDObserved, pid: pid}, nil
}

func (observation SpawnObservation) Valid() bool {
	return (observation.state == SpawnStartError && validStartErrorCode(observation.errorCode) && observation.pid == 0) ||
		(observation.state == SpawnChildPIDObserved && observation.errorCode == "" && observation.pid >= 1 && observation.pid <= 2147483647)
}

func validStartErrorCode(code string) bool {
	switch code {
	case "OS_START_ERROR", "PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED", "PRESPAWN_RUNTIME_REVALIDATION_FAILED":
		return true
	default:
		return false
	}
}

func (observation SpawnObservation) State() SpawnObservationState { return observation.state }
func (observation SpawnObservation) ErrorCode() (string, bool) {
	return observation.errorCode, observation.state == SpawnStartError
}
func (observation SpawnObservation) PID() (int64, bool) {
	return observation.pid, observation.state == SpawnChildPIDObserved
}

type ProcessClosure struct {
	state         ProcessClosureState
	primary       domain.ControlReason
	teardownError bool
	orphanRisk    bool
	evidence      []EvidenceRef
}

func NewCleanProcessClosure(evidence []EvidenceRef) (ProcessClosure, error) {
	normalized, err := normalizeEvidence(evidence)
	if err != nil {
		return ProcessClosure{}, err
	}
	return ProcessClosure{state: ProcessClean, evidence: normalized}, nil
}

func NewControlledProcessClosure(
	primary domain.ControlReason,
	teardownError bool,
	orphanRisk bool,
	evidence []EvidenceRef,
) (ProcessClosure, error) {
	if primary != "" && !validPrimaryControl(primary) {
		return ProcessClosure{}, refuse(CodeInvalidWitness, "primary process control is outside the closed non-cleanup roster", nil)
	}
	if primary == "" && !teardownError && !orphanRisk {
		return ProcessClosure{}, refuse(CodeInvalidWitness, "controlled process closure has no control", nil)
	}
	normalized, err := normalizeEvidence(evidence)
	if err != nil {
		return ProcessClosure{}, err
	}
	return ProcessClosure{
		state: ProcessControlled, primary: primary, teardownError: teardownError,
		orphanRisk: orphanRisk, evidence: normalized,
	}, nil
}

func validPrimaryControl(reason domain.ControlReason) bool {
	switch reason {
	case domain.ControlMaterializationError, domain.ControlStartError, domain.ControlReadinessError,
		domain.ControlProbeTransportError, domain.ControlTimeout, domain.ControlCancelled,
		domain.ControlOutputLimit, domain.ControlProjectionRejected:
		return true
	default:
		return false
	}
}

func normalizeEvidence(input []EvidenceRef) ([]EvidenceRef, error) {
	if len(input) == 0 || len(input) > MaxWitnessReferences {
		return nil, refuse(CodeInvalidWitness, "evidence reference count is outside the closed ceiling", nil)
	}
	seen := make(map[EvidenceKind]struct{}, len(input))
	result := append([]EvidenceRef(nil), input...)
	for _, ref := range result {
		if !ref.Valid() {
			return nil, refuse(CodeInvalidWitness, "evidence roster contains an invalid typed reference", nil)
		}
		if _, duplicate := seen[ref.kind]; duplicate {
			return nil, refuse(CodeInvalidWitness, "evidence roster repeats a typed kind", nil)
		}
		seen[ref.kind] = struct{}{}
	}
	sort.Slice(result, func(left, right int) bool { return evidenceRank(result[left].kind) < evidenceRank(result[right].kind) })
	return result, nil
}

func evidenceRank(kind EvidenceKind) int {
	for index, candidate := range evidenceOrder {
		if kind == candidate {
			return index
		}
	}
	return len(evidenceOrder)
}

func (closure ProcessClosure) Valid() bool {
	var rebuilt ProcessClosure
	var err error
	if closure.state == ProcessClean {
		rebuilt, err = NewCleanProcessClosure(closure.evidence)
	} else if closure.state == ProcessControlled {
		rebuilt, err = NewControlledProcessClosure(closure.primary, closure.teardownError, closure.orphanRisk, closure.evidence)
	} else {
		return false
	}
	return err == nil && rebuilt.state == closure.state && rebuilt.primary == closure.primary &&
		rebuilt.teardownError == closure.teardownError && rebuilt.orphanRisk == closure.orphanRisk &&
		equalEvidence(rebuilt.evidence, closure.evidence)
}

func equalEvidence(left, right []EvidenceRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (closure ProcessClosure) State() ProcessClosureState { return closure.state }
func (closure ProcessClosure) Primary() (domain.ControlReason, bool) {
	return closure.primary, closure.primary != ""
}
func (closure ProcessClosure) TeardownError() bool { return closure.teardownError }
func (closure ProcessClosure) OrphanRisk() bool    { return closure.orphanRisk }
func (closure ProcessClosure) Evidence() []EvidenceRef {
	return append([]EvidenceRef(nil), closure.evidence...)
}

type CaptureObservation struct {
	state      CaptureState
	captured   EvidenceRef
	projection EvidenceRef
	tuple      emitmodel.ExactTuple
}

func NewNoCapture() CaptureObservation { return CaptureObservation{state: CaptureNone} }

func NewCapturedUnprojected(captured EvidenceRef) (CaptureObservation, error) {
	if !captured.Valid() || captured.kind != EvidenceCapturedObservation {
		return CaptureObservation{}, refuse(CodeInvalidWitness, "captured observation reference has the wrong typed kind", nil)
	}
	return CaptureObservation{state: CaptureUnprojected, captured: captured}, nil
}

func NewProjectedObservation(
	captured EvidenceRef,
	projection EvidenceRef,
	tuple emitmodel.ExactTuple,
) (CaptureObservation, error) {
	if !captured.Valid() || captured.kind != EvidenceCapturedObservation || !projection.Valid() ||
		projection.kind != EvidenceProjectionResult || !tuple.Valid() {
		return CaptureObservation{}, refuse(CodeInvalidWitness, "projected observation inputs are invalid or mistyped", nil)
	}
	copyTuple, err := emitmodel.NewExactTuple(tuple.Fields())
	if err != nil {
		return CaptureObservation{}, refuse(CodeInvalidWitness, "projected tuple could not be defensively reconstructed", err)
	}
	return CaptureObservation{state: CaptureProjected, captured: captured, projection: projection, tuple: copyTuple}, nil
}

func (observation CaptureObservation) Valid() bool {
	switch observation.state {
	case CaptureNone:
		return !observation.captured.Valid() && !observation.projection.Valid() && !observation.tuple.Valid()
	case CaptureUnprojected:
		return observation.captured.Valid() && observation.captured.kind == EvidenceCapturedObservation &&
			!observation.projection.Valid() && !observation.tuple.Valid()
	case CaptureProjected:
		return observation.captured.Valid() && observation.captured.kind == EvidenceCapturedObservation &&
			observation.projection.Valid() && observation.projection.kind == EvidenceProjectionResult && observation.tuple.Valid()
	default:
		return false
	}
}

func (observation CaptureObservation) State() CaptureState { return observation.state }
func (observation CaptureObservation) CapturedRef() (EvidenceRef, bool) {
	return observation.captured, observation.state == CaptureUnprojected || observation.state == CaptureProjected
}
func (observation CaptureObservation) ProjectionRef() (EvidenceRef, bool) {
	return observation.projection, observation.state == CaptureProjected
}
func (observation CaptureObservation) Tuple() (emitmodel.ExactTuple, bool) {
	if observation.state != CaptureProjected || !observation.tuple.Valid() {
		return emitmodel.ExactTuple{}, false
	}
	copyTuple, _ := emitmodel.NewExactTuple(observation.tuple.Fields())
	return copyTuple, true
}

type ScopeCheck struct {
	domain    ScopeDomain
	state     ScopeCheckState
	evidence  EvidenceRef
	violation ScopeViolation
}

func NewCleanScopeCheck(scopeDomain ScopeDomain, evidence EvidenceRef) (ScopeCheck, error) {
	if !validScopeEvidence(scopeDomain, evidence) {
		return ScopeCheck{}, refuse(CodeInvalidWitness, "clean scope check has the wrong typed reference", nil)
	}
	return ScopeCheck{domain: scopeDomain, state: ScopeCheckClean, evidence: evidence}, nil
}

func NewMissingScopeCheck(scopeDomain ScopeDomain) (ScopeCheck, error) {
	if !validScopeDomain(scopeDomain) {
		return ScopeCheck{}, refuse(CodeInvalidWitness, "missing scope check has an unknown domain", nil)
	}
	return ScopeCheck{domain: scopeDomain, state: ScopeCheckMissing}, nil
}

func NewViolatedScopeCheck(scopeDomain ScopeDomain, evidence EvidenceRef, violation ScopeViolation) (ScopeCheck, error) {
	if !validScopeEvidence(scopeDomain, evidence) || !validViolation(scopeDomain, violation) {
		return ScopeCheck{}, refuse(CodeInvalidWitness, "violated scope check is mistyped", nil)
	}
	return ScopeCheck{domain: scopeDomain, state: ScopeCheckViolated, evidence: evidence, violation: violation}, nil
}

func validScopeDomain(scopeDomain ScopeDomain) bool {
	for _, candidate := range scopeDomainOrder {
		if scopeDomain == candidate {
			return true
		}
	}
	return false
}

func scopeEvidenceKind(scopeDomain ScopeDomain) EvidenceKind {
	switch scopeDomain {
	case ScopeTargetInventory:
		return EvidenceTargetInventory
	case ScopeChildBindings:
		return EvidenceChildBindings
	case ScopeImportResolution:
		return EvidenceImportResolution
	case ScopeServiceBindings:
		return EvidenceServiceBindings
	case ScopeSentinelInheritance:
		return EvidenceSentinelInheritance
	default:
		return ""
	}
}

func validScopeEvidence(scopeDomain ScopeDomain, evidence EvidenceRef) bool {
	return validScopeDomain(scopeDomain) && evidence.Valid() && evidence.kind == scopeEvidenceKind(scopeDomain)
}

func validViolation(scopeDomain ScopeDomain, violation ScopeViolation) bool {
	switch scopeDomain {
	case ScopeTargetInventory:
		return violation == ViolationTargetSourcePresent || violation == ViolationTargetDependencyPresent ||
			violation == ViolationTargetSourceAndDependency
	case ScopeChildBindings:
		return violation == ViolationChildBinding
	case ScopeImportResolution:
		return violation == ViolationImportResolution
	case ScopeServiceBindings:
		return violation == ViolationServiceBinding
	case ScopeSentinelInheritance:
		return violation == ViolationSentinelInherited
	default:
		return false
	}
}

func (check ScopeCheck) Valid() bool {
	switch check.state {
	case ScopeCheckClean:
		return validScopeEvidence(check.domain, check.evidence) && check.violation == ""
	case ScopeCheckMissing:
		return validScopeDomain(check.domain) && !check.evidence.Valid() && check.violation == ""
	case ScopeCheckViolated:
		return validScopeEvidence(check.domain, check.evidence) && validViolation(check.domain, check.violation)
	default:
		return false
	}
}

func (check ScopeCheck) Domain() ScopeDomain           { return check.domain }
func (check ScopeCheck) State() ScopeCheckState        { return check.state }
func (check ScopeCheck) Evidence() (EvidenceRef, bool) { return check.evidence, check.evidence.Valid() }
func (check ScopeCheck) Violation() (ScopeViolation, bool) {
	return check.violation, check.state == ScopeCheckViolated
}

type StandaloneScope struct {
	state  ScopeState
	checks []ScopeCheck
}

func NewStandaloneScope(checks []ScopeCheck) (StandaloneScope, error) {
	if len(checks) != len(scopeDomainOrder) {
		return StandaloneScope{}, refuse(CodeInvalidWitness, "standalone scope must contain exactly five checks", nil)
	}
	byDomain := make(map[ScopeDomain]ScopeCheck, len(checks))
	for _, check := range checks {
		if !check.Valid() {
			return StandaloneScope{}, refuse(CodeInvalidWitness, "standalone scope contains an invalid check", nil)
		}
		if _, duplicate := byDomain[check.domain]; duplicate {
			return StandaloneScope{}, refuse(CodeInvalidWitness, "standalone scope repeats a domain", nil)
		}
		byDomain[check.domain] = check
	}
	ordered := make([]ScopeCheck, len(scopeDomainOrder))
	state := ScopeComplete
	for index, scopeDomain := range scopeDomainOrder {
		check, ok := byDomain[scopeDomain]
		if !ok {
			return StandaloneScope{}, refuse(CodeInvalidWitness, "standalone scope omits a domain", nil)
		}
		ordered[index] = check
		if check.state == ScopeCheckViolated {
			state = ScopeViolated
		} else if check.state == ScopeCheckMissing && state != ScopeViolated {
			state = ScopePartial
		}
	}
	return StandaloneScope{state: state, checks: ordered}, nil
}

func (scope StandaloneScope) Valid() bool {
	rebuilt, err := NewStandaloneScope(scope.checks)
	return err == nil && rebuilt.state == scope.state && equalScopeChecks(rebuilt.checks, scope.checks)
}

func equalScopeChecks(left, right []ScopeCheck) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (scope StandaloneScope) State() ScopeState    { return scope.state }
func (scope StandaloneScope) Checks() []ScopeCheck { return append([]ScopeCheck(nil), scope.checks...) }

type PrivateManifestSummary struct {
	manifestRef        EvidenceRef
	blobCount          int64
	aggregateByteCount int64
}

func NewPrivateManifestSummary(manifest EvidenceRef, blobCount, aggregateByteCount int64) (PrivateManifestSummary, error) {
	if !manifest.Valid() || manifest.kind != EvidencePrivateManifest || blobCount < 0 || blobCount > MaxPrivateEvidenceBlobs || aggregateByteCount < 0 ||
		aggregateByteCount > MaxPrivateEvidenceBytes || (blobCount == 0 && aggregateByteCount != 0) ||
		(blobCount > 0 && aggregateByteCount < blobCount) {
		return PrivateManifestSummary{}, refuse(CodeInvalidWitness, "private evidence manifest summary is outside the closed ceilings", nil)
	}
	return PrivateManifestSummary{manifestRef: manifest, blobCount: blobCount, aggregateByteCount: aggregateByteCount}, nil
}

func (summary PrivateManifestSummary) Valid() bool {
	_, err := NewPrivateManifestSummary(summary.manifestRef, summary.blobCount, summary.aggregateByteCount)
	return err == nil
}
func (summary PrivateManifestSummary) ManifestRef() EvidenceRef  { return summary.manifestRef }
func (summary PrivateManifestSummary) BlobCount() int64          { return summary.blobCount }
func (summary PrivateManifestSummary) AggregateByteCount() int64 { return summary.aggregateByteCount }

type ClosedRunWitness struct {
	spawn           SpawnObservation
	process         ProcessClosure
	observation     CaptureObservation
	scope           StandaloneScope
	privateManifest PrivateManifestSummary
	disposition     RunDisposition
	canonical       []byte
}

func NewClosedRunWitness(
	spawn SpawnObservation,
	process ProcessClosure,
	observation CaptureObservation,
	scope StandaloneScope,
	privateManifest PrivateManifestSummary,
) (ClosedRunWitness, error) {
	if !spawn.Valid() || !process.Valid() || !observation.Valid() || !scope.Valid() || !privateManifest.Valid() {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "closed-run axes are invalid", nil)
	}
	if err := validateSpawnProcessRelation(spawn, process, observation); err != nil {
		return ClosedRunWitness{}, err
	}
	if err := validateReferenceClosure(process, observation, scope, privateManifest); err != nil {
		return ClosedRunWitness{}, err
	}
	if process.state == ProcessClean && observation.state != CaptureProjected {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "clean process closure requires an exact projected observation", nil)
	}
	if observation.state != CaptureProjected && process.primary == "" {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "nonprojected observation requires a primary process control", nil)
	}
	if spawn.state == SpawnStartError && scope.state == ScopeComplete {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "start-error observation cannot claim complete standalone scope", nil)
	}
	disposition := deriveDisposition(process.state, scope.state)
	witness := ClosedRunWitness{
		spawn: spawn, process: cloneProcessClosure(process), observation: cloneCaptureObservation(observation),
		scope: cloneStandaloneScope(scope), privateManifest: privateManifest, disposition: disposition,
	}
	wire, err := witnessWire(witness)
	if err != nil {
		return ClosedRunWitness{}, err
	}
	exact, _, err := canonicalObject("ClosedRunWitness", wire)
	if err != nil {
		return ClosedRunWitness{}, refuse(CodeInvalidWitness, "closed-run witness could not be canonicalized", err)
	}
	if len(exact) > MaxClosedRunWitnessBytes {
		return ClosedRunWitness{}, refuse(CodeLimitExceeded, "closed-run witness exceeds 256 KiB", nil)
	}
	witness.canonical = exact
	return witness, nil
}

func validateSpawnProcessRelation(spawn SpawnObservation, process ProcessClosure, observation CaptureObservation) error {
	if spawn.state == SpawnStartError {
		if process.state != ProcessControlled || process.primary != domain.ControlStartError || observation.state != CaptureNone {
			return refuse(CodeInvalidWitness, "start-error observation requires START_ERROR control and no capture", nil)
		}
	} else if process.primary == domain.ControlStartError {
		return refuse(CodeInvalidWitness, "observed child PID cannot retain START_ERROR as primary control", nil)
	}
	allowed := []EvidenceKind{
		EvidenceMaterializationRevalidation, EvidenceRuntimeRevalidation, EvidenceTeardownResult,
		EvidenceOrphanCheck, EvidenceFinalizationMarker,
	}
	if spawn.state == SpawnChildPIDObserved {
		allowed = []EvidenceKind{
			EvidenceMaterializationRevalidation, EvidenceRuntimeRevalidation, EvidenceProcessResult,
			EvidenceWaitResult, EvidenceDrainResult, EvidenceTeardownResult, EvidenceOrphanCheck,
			EvidenceFinalizationMarker,
		}
	}
	if len(process.evidence) != len(allowed) {
		return refuse(CodeInvalidWitness, "process evidence roster is incomplete for the spawn observation", nil)
	}
	for index, expected := range allowed {
		if process.evidence[index].kind != expected {
			return refuse(CodeInvalidWitness, "process evidence roster differs from the spawn-specific order", nil)
		}
	}
	return nil
}

func validateReferenceClosure(process ProcessClosure, observation CaptureObservation, scope StandaloneScope, privateManifest PrivateManifestSummary) error {
	seen := make(map[EvidenceKind]struct{}, MaxWitnessReferences)
	add := func(ref EvidenceRef) error {
		if !ref.Valid() {
			return refuse(CodeInvalidWitness, "closed-run witness contains an invalid typed reference", nil)
		}
		if _, duplicate := seen[ref.kind]; duplicate {
			return refuse(CodeInvalidWitness, "closed-run witness repeats a typed reference kind", nil)
		}
		seen[ref.kind] = struct{}{}
		return nil
	}
	for _, ref := range process.evidence {
		if err := add(ref); err != nil {
			return err
		}
	}
	if ref, ok := observation.CapturedRef(); ok {
		if err := add(ref); err != nil {
			return err
		}
	}
	if ref, ok := observation.ProjectionRef(); ok {
		if err := add(ref); err != nil {
			return err
		}
	}
	for _, check := range scope.checks {
		if ref, ok := check.Evidence(); ok {
			if err := add(ref); err != nil {
				return err
			}
		}
	}
	if err := add(privateManifest.manifestRef); err != nil {
		return err
	}
	if len(seen) > MaxWitnessReferences {
		return refuse(CodeLimitExceeded, "closed-run witness exceeds the typed-reference ceiling", nil)
	}
	return nil
}

func deriveDisposition(process ProcessClosureState, scope ScopeState) RunDisposition {
	if process == ProcessClean && scope == ScopeComplete {
		return DispositionEligibleClean
	}
	if process == ProcessControlled && scope == ScopeComplete {
		return DispositionIneligibleControl
	}
	if process == ProcessClean {
		return DispositionIneligibleStandalone
	}
	return DispositionIneligibleControlStandalone
}

func (witness ClosedRunWitness) Valid() bool {
	rebuilt, err := NewClosedRunWitness(witness.spawn, witness.process, witness.observation, witness.scope, witness.privateManifest)
	return err == nil && rebuilt.disposition == witness.disposition && bytes.Equal(rebuilt.canonical, witness.canonical)
}

func (witness ClosedRunWitness) SpawnObservation() SpawnObservation { return witness.spawn }
func (witness ClosedRunWitness) ProcessClosure() ProcessClosure {
	return cloneProcessClosure(witness.process)
}
func (witness ClosedRunWitness) Observation() CaptureObservation {
	return cloneCaptureObservation(witness.observation)
}
func (witness ClosedRunWitness) StandaloneScope() StandaloneScope {
	return cloneStandaloneScope(witness.scope)
}
func (witness ClosedRunWitness) PrivateManifest() PrivateManifestSummary {
	return witness.privateManifest
}
func (witness ClosedRunWitness) Disposition() RunDisposition { return witness.disposition }
func (witness ClosedRunWitness) CanonicalBytes() []byte      { return cloneBytes(witness.canonical) }

func cloneProcessClosure(input ProcessClosure) ProcessClosure {
	input.evidence = append([]EvidenceRef(nil), input.evidence...)
	return input
}

func cloneCaptureObservation(input CaptureObservation) CaptureObservation {
	if input.tuple.Valid() {
		input.tuple, _ = emitmodel.NewExactTuple(input.tuple.Fields())
	}
	return input
}

func cloneStandaloneScope(input StandaloneScope) StandaloneScope {
	input.checks = append([]ScopeCheck(nil), input.checks...)
	return input
}

func evidenceRefWire(ref EvidenceRef) map[string]any {
	return map[string]any{"kind": string(ref.kind), "digest": ref.digest.String()}
}

func witnessWire(witness ClosedRunWitness) (map[string]any, error) {
	spawnWire := map[string]any{"status": string(witness.spawn.state)}
	if witness.spawn.state == SpawnStartError {
		spawnWire["error_code"] = witness.spawn.errorCode
	} else {
		spawnWire["pid"] = witness.spawn.pid
	}
	processEvidence := make([]any, len(witness.process.evidence))
	for index, ref := range witness.process.evidence {
		processEvidence[index] = evidenceRefWire(ref)
	}
	cleanup := make([]any, 0, 2)
	if witness.process.teardownError {
		cleanup = append(cleanup, string(domain.ControlTeardownError))
	}
	if witness.process.orphanRisk {
		cleanup = append(cleanup, string(domain.ControlOrphanRisk))
	}
	primary := "NONE"
	if witness.process.primary != "" {
		primary = string(witness.process.primary)
	}
	processWire := map[string]any{
		"status": string(witness.process.state), "primary_reason": primary,
		"cleanup_controls": cleanup, "evidence_refs": processEvidence,
	}
	observationWire := map[string]any{"status": string(witness.observation.state)}
	if witness.observation.state == CaptureUnprojected || witness.observation.state == CaptureProjected {
		observationWire["captured_observation_ref"] = evidenceRefWire(witness.observation.captured)
	}
	if witness.observation.state == CaptureProjected {
		observationWire["projection_result_ref"] = evidenceRefWire(witness.observation.projection)
		tupleValue, err := parseCanonicalValue(witness.observation.tuple.CanonicalBytes())
		if err != nil {
			return nil, refuse(CodeInvalidWitness, "projected tuple is not canonical", err)
		}
		plainTuple, err := plainValue(tupleValue)
		if err != nil {
			return nil, err
		}
		observationWire["observed_tuple"] = plainTuple
	}
	checks := make([]any, len(witness.scope.checks))
	for index, check := range witness.scope.checks {
		wire := map[string]any{"domain": string(check.domain), "status": string(check.state)}
		if check.evidence.Valid() {
			wire["evidence_ref"] = evidenceRefWire(check.evidence)
		}
		if check.state == ScopeCheckViolated {
			wire["violation"] = string(check.violation)
		}
		checks[index] = wire
	}
	return map[string]any{
		"witness_version":   ClosedRunWitnessVersionV1,
		"spawn_observation": spawnWire,
		"process_closure":   processWire,
		"observation":       observationWire,
		"standalone_scope":  map[string]any{"status": string(witness.scope.state), "checks": checks},
		"private_evidence": map[string]any{
			"profile":                   PrivateManifestProfileV1,
			"manifest_ref":              evidenceRefWire(witness.privateManifest.manifestRef),
			"blob_count":                witness.privateManifest.blobCount,
			"aggregate_byte_count":      witness.privateManifest.aggregateByteCount,
			"retention_at_finalization": PrivateRetentionV1,
			"default_export":            PrivateDefaultExportV1,
		},
	}, nil
}

func parseCanonicalValue(exact []byte) (canon.Value, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return canon.Value{}, err
	}
	checked, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(checked, exact) {
		return canon.Value{}, refuse(CodeInvalidWitness, "value is not exact canonical JSON", err)
	}
	return value, nil
}

type FinalizedContractRun struct {
	targetDigest     domain.Digest
	attemptDigest    domain.Digest
	startClaimDigest domain.Digest
	witness          ClosedRunWitness
	digest           domain.Digest
	canonical        []byte
}

func NewFinalizedContractRun(
	target ContractExecutionTarget,
	startClaimDigest domain.Digest,
	witness ClosedRunWitness,
) (FinalizedContractRun, error) {
	if !target.Valid() || !startClaimDigest.Valid() || !witness.Valid() {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run inputs are invalid", nil)
	}
	wire, err := finalizedRunWire(target.Digest(), target.AttemptArtifactDigest(), startClaimDigest, witness)
	if err != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run body could not be assembled", err)
	}
	exact, digest, err := canonicalObject(RunKind, wire)
	if err != nil {
		return FinalizedContractRun{}, refuse(CodeInvalidRun, "run body could not be built", err)
	}
	return FinalizedContractRun{
		targetDigest: target.Digest(), attemptDigest: target.AttemptArtifactDigest(),
		startClaimDigest: startClaimDigest, witness: witness,
		digest: digest, canonical: exact,
	}, nil
}

func finalizedRunWire(targetDigest, attemptDigest, startClaimDigest domain.Digest, witness ClosedRunWitness) (map[string]any, error) {
	witnessMap, err := witnessWire(witness)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"schema_version":                   domain.SchemaVersion,
		"kind":                             RunKind,
		"run_version":                      RunVersionV1,
		"publication_scope":                RunPublicationScopeV1,
		"contract_execution_target_digest": targetDigest.String(),
		"attempt_artifact_digest":          attemptDigest.String(),
		"start_claim_ref":                  map[string]any{"kind": "StartClaim", "digest": startClaimDigest.String()},
		"closed_run_witness":               witnessMap,
	}, nil
}

func (run FinalizedContractRun) Valid() bool {
	if !run.targetDigest.Valid() || !run.attemptDigest.Valid() || !run.startClaimDigest.Valid() ||
		!run.witness.Valid() || !run.digest.Valid() || len(run.canonical) == 0 {
		return false
	}
	wire, err := finalizedRunWire(run.targetDigest, run.attemptDigest, run.startClaimDigest, run.witness)
	if err != nil {
		return false
	}
	exact, digest, err := canonicalObject(RunKind, wire)
	return err == nil && digest == run.digest && bytes.Equal(exact, run.canonical)
}

func (run FinalizedContractRun) MatchesTarget(target ContractExecutionTarget) bool {
	return run.Valid() && target.Valid() && run.targetDigest == target.Digest() && run.attemptDigest == target.AttemptArtifactDigest()
}

func (run FinalizedContractRun) Digest() domain.Digest                { return run.digest }
func (run FinalizedContractRun) CanonicalBytes() []byte               { return cloneBytes(run.canonical) }
func (run FinalizedContractRun) TargetDigest() domain.Digest          { return run.targetDigest }
func (run FinalizedContractRun) AttemptArtifactDigest() domain.Digest { return run.attemptDigest }
func (run FinalizedContractRun) StartClaimDigest() domain.Digest      { return run.startClaimDigest }
func (run FinalizedContractRun) Witness() ClosedRunWitness {
	copyWitness, _ := NewClosedRunWitness(run.witness.spawn, run.witness.process, run.witness.observation, run.witness.scope, run.witness.privateManifest)
	return copyWitness
}
func (run FinalizedContractRun) PrivateManifest() PrivateManifestSummary {
	return run.witness.privateManifest
}
func (run FinalizedContractRun) Disposition() RunDisposition { return run.witness.disposition }

func (run FinalizedContractRun) Equal(other FinalizedContractRun) bool {
	return run.Valid() && other.Valid() && run.digest == other.digest && bytes.Equal(run.canonical, other.canonical)
}
