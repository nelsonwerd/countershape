package model

import (
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	TargetKind                  = "ContractExecutionTarget"
	TargetVersionV1             = "contract-execution-target/v1"
	TargetPublicationScopeV1    = "IMMUTABLE_NONHEAD_PRESPAWN_AUTHORITY_V1"
	RunKind                     = "FinalizedContractRun"
	RunVersionV1                = "finalized-contract-run/v1"
	RunPublicationScopeV1       = "IMMUTABLE_NONHEAD_FINALIZED_RUN_V1"
	ExecutionKind               = "ContractExecution"
	ExecutionVersionV1          = "contract-execution/v1"
	ExecutionPublicationScopeV1 = "IMMUTABLE_NONHEAD_CLASSIFICATION_V1"
	ClassifierProfileV1         = "CONTRACT_EXECUTION_EXACT_TUPLE_V1"
	ClosedRunWitnessVersionV1   = "closed-run-witness/v1"
	PrivateManifestProfileV1    = "PRIVATE_EVIDENCE_MANIFEST_V1"
	PrivateRetentionV1          = "RETAINED_AT_FINALIZATION"
	PrivateDefaultExportV1      = "OMITTED"
	MaxClosedRunWitnessBytes    = 256 * 1024
	MaxWitnessReferences        = 16
	MaxPrivateEvidenceBlobs     = 16
	MaxPrivateEvidenceBytes     = 64 * 1024 * 1024
)

type EvidenceKind string

const (
	EvidenceMaterializationRevalidation EvidenceKind = "MATERIALIZATION_REVALIDATION"
	EvidenceRuntimeRevalidation         EvidenceKind = "RUNTIME_REVALIDATION"
	EvidenceProcessResult               EvidenceKind = "PROCESS_RESULT"
	EvidenceWaitResult                  EvidenceKind = "WAIT_RESULT"
	EvidenceDrainResult                 EvidenceKind = "DRAIN_RESULT"
	EvidenceTeardownResult              EvidenceKind = "TEARDOWN_RESULT"
	EvidenceOrphanCheck                 EvidenceKind = "ORPHAN_CHECK"
	EvidenceFinalizationMarker          EvidenceKind = "FINALIZATION_MARKER"
	EvidenceCapturedObservation         EvidenceKind = "CAPTURED_OBSERVATION"
	EvidenceProjectionResult            EvidenceKind = "PROJECTION_RESULT"
	EvidenceTargetInventory             EvidenceKind = "TARGET_INVENTORY"
	EvidenceChildBindings               EvidenceKind = "CHILD_BINDINGS"
	EvidenceImportResolution            EvidenceKind = "IMPORT_RESOLUTION"
	EvidenceServiceBindings             EvidenceKind = "SERVICE_BINDINGS"
	EvidenceSentinelInheritance         EvidenceKind = "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE"
	EvidencePrivateManifest             EvidenceKind = "PRIVATE_EVIDENCE_MANIFEST"
)

var evidenceOrder = [...]EvidenceKind{
	EvidenceMaterializationRevalidation,
	EvidenceRuntimeRevalidation,
	EvidenceProcessResult,
	EvidenceWaitResult,
	EvidenceDrainResult,
	EvidenceTeardownResult,
	EvidenceOrphanCheck,
	EvidenceFinalizationMarker,
	EvidenceCapturedObservation,
	EvidenceProjectionResult,
	EvidenceTargetInventory,
	EvidenceChildBindings,
	EvidenceImportResolution,
	EvidenceServiceBindings,
	EvidenceSentinelInheritance,
	EvidencePrivateManifest,
}

func (kind EvidenceKind) valid() bool {
	for _, candidate := range evidenceOrder {
		if kind == candidate {
			return true
		}
	}
	return false
}

type EvidenceRef struct {
	kind   EvidenceKind
	digest domain.Digest
}

func newEvidenceRef(kind EvidenceKind, digest domain.Digest) (EvidenceRef, error) {
	if !kind.valid() || !digest.Valid() {
		return EvidenceRef{}, refuse(CodeInvalidWitness, "typed evidence reference is invalid", nil)
	}
	return EvidenceRef{kind: kind, digest: digest}, nil
}

func NewMaterializationRevalidationRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceMaterializationRevalidation, digest)
}
func NewRuntimeRevalidationRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceRuntimeRevalidation, digest)
}
func NewProcessResultRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceProcessResult, digest)
}
func NewWaitResultRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceWaitResult, digest)
}
func NewDrainResultRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceDrainResult, digest)
}
func NewTeardownResultRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceTeardownResult, digest)
}
func NewOrphanCheckRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceOrphanCheck, digest)
}
func NewFinalizationMarkerRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceFinalizationMarker, digest)
}
func NewCapturedObservationRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceCapturedObservation, digest)
}
func NewProjectionResultRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceProjectionResult, digest)
}
func NewTargetInventoryRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceTargetInventory, digest)
}
func NewChildBindingsRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceChildBindings, digest)
}
func NewImportResolutionRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceImportResolution, digest)
}
func NewServiceBindingsRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceServiceBindings, digest)
}
func NewSentinelInheritanceRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidenceSentinelInheritance, digest)
}
func NewPrivateEvidenceManifestRef(digest domain.Digest) (EvidenceRef, error) {
	return newEvidenceRef(EvidencePrivateManifest, digest)
}

func (ref EvidenceRef) Valid() bool           { return ref.kind.valid() && ref.digest.Valid() }
func (ref EvidenceRef) Kind() EvidenceKind    { return ref.kind }
func (ref EvidenceRef) Digest() domain.Digest { return ref.digest }

type SpawnObservationState string

const (
	SpawnStartError       SpawnObservationState = "START_ERROR"
	SpawnChildPIDObserved SpawnObservationState = "CHILD_PID_OBSERVED"
)

type ProcessClosureState string

const (
	ProcessClean      ProcessClosureState = "PROCESS_CLEAN"
	ProcessControlled ProcessClosureState = "PROCESS_CONTROLLED"
)

type CaptureState string

const (
	CaptureNone        CaptureState = "NO_CAPTURE"
	CaptureUnprojected CaptureState = "CAPTURED_UNPROJECTED"
	CaptureProjected   CaptureState = "PROJECTED"
)

type ScopeDomain string

const (
	ScopeTargetInventory     ScopeDomain = "TARGET_INVENTORY"
	ScopeChildBindings       ScopeDomain = "CHILD_BINDINGS"
	ScopeImportResolution    ScopeDomain = "IMPORT_RESOLUTION"
	ScopeServiceBindings     ScopeDomain = "SERVICE_BINDINGS"
	ScopeSentinelInheritance ScopeDomain = "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE"
)

var scopeDomainOrder = [...]ScopeDomain{
	ScopeTargetInventory,
	ScopeChildBindings,
	ScopeImportResolution,
	ScopeServiceBindings,
	ScopeSentinelInheritance,
}

type ScopeCheckState string

const (
	ScopeCheckClean    ScopeCheckState = "PRESENT_CLEAN"
	ScopeCheckMissing  ScopeCheckState = "MISSING"
	ScopeCheckViolated ScopeCheckState = "PRESENT_VIOLATION"
)

type ScopeState string

const (
	ScopeComplete ScopeState = "COMPLETE"
	ScopePartial  ScopeState = "PARTIAL"
	ScopeViolated ScopeState = "VIOLATED"
)

type ScopeViolation string

const (
	ViolationTargetSourcePresent       ScopeViolation = "COUNTERSHAPE_SOURCE_PRESENT"
	ViolationTargetDependencyPresent   ScopeViolation = "COUNTERSHAPE_DEPENDENCY_PRESENT"
	ViolationTargetSourceAndDependency ScopeViolation = "COUNTERSHAPE_SOURCE_AND_DEPENDENCY_PRESENT"
	ViolationChildBinding              ScopeViolation = "CHILD_BINDING_REACHES_COUNTERSHAPE"
	ViolationImportResolution          ScopeViolation = "IMPORT_RESOLUTION_REACHES_COUNTERSHAPE"
	ViolationServiceBinding            ScopeViolation = "COUNTERSHAPE_SERVICE_BINDING_PRESENT"
	ViolationSentinelInherited         ScopeViolation = "NAMED_PARENT_SECRET_SENTINEL_INHERITED"
)

type RunDisposition string

const (
	DispositionEligibleClean               RunDisposition = "ELIGIBLE_CLEAN"
	DispositionIneligibleControl           RunDisposition = "INELIGIBLE_CONTROL"
	DispositionIneligibleStandalone        RunDisposition = "INELIGIBLE_STANDALONE"
	DispositionIneligibleControlStandalone RunDisposition = "INELIGIBLE_CONTROL_AND_STANDALONE"
)

type ExecutionResult string

const (
	ResultConforms    ExecutionResult = "CONFORMS"
	ResultContradicts ExecutionResult = "CONTRADICTS"
	ResultIneligible  ExecutionResult = "INELIGIBLE_EXECUTION"
)
