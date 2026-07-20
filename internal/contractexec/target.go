// Package contractexec owns the sole production composition edge from inert
// target facts to a live pre-spawn OfficialTarget capability.
package contractexec

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	node "github.com/nelsonwerd/countershape/internal/emit/node"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/noderuntime"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidOfficialTarget = "INVALID_OFFICIAL_TARGET"
	CodeOfficialTargetChanged = "OFFICIAL_TARGET_CHANGED"
	CodeOfficialTargetClosed  = "OFFICIAL_TARGET_CLOSED"
)

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

// PublishOfficialTargetRequest contains only live owners and explicit physical
// inputs. It has no target bytes, boot value, nonce, OID bag, or generic CAS
// authority.
type PublishOfficialTargetRequest struct {
	Store          *store.ObjectStore
	Residue        node.Residue
	Repository     gitobj.Repository
	DisplayRef     string
	NodeExecutable string
}

// OpenOfficialTargetRequest uses a digest only as an exact locator. Every live
// prerequisite is reconstructed and compared before authority is issued.
type OpenOfficialTargetRequest struct {
	Store        *store.ObjectStore
	Residue      node.Residue
	Repository   gitobj.Repository
	TargetDigest domain.Digest
}

type officialTargetSeal struct{ marker byte }

var issuedOfficialTargetSeal = &officialTargetSeal{marker: 1}

type officialTargetFaultPhase string
type officialTargetFault func(officialTargetFaultPhase) error

const (
	officialFaultAfterInputsValidated          officialTargetFaultPhase = "after-inputs-validated"
	officialFaultAfterInitialResidueReopen     officialTargetFaultPhase = "after-initial-residue-reopen"
	officialFaultAfterPolicyDerivation         officialTargetFaultPhase = "after-policy-derivation"
	officialFaultAfterTreePin                  officialTargetFaultPhase = "after-tree-pin"
	officialFaultAfterAttemptAllocation        officialTargetFaultPhase = "after-attempt-allocation"
	officialFaultAfterTreeInspection           officialTargetFaultPhase = "after-tree-inspection"
	officialFaultAfterEntrypointJoin           officialTargetFaultPhase = "after-entrypoint-join"
	officialFaultAfterMaterialization          officialTargetFaultPhase = "after-materialization"
	officialFaultAfterMaterializationReopen    officialTargetFaultPhase = "after-materialization-reopen"
	officialFaultAfterRuntimeAdmission         officialTargetFaultPhase = "after-runtime-admission"
	officialFaultAfterRuntimeRevalidation      officialTargetFaultPhase = "after-runtime-revalidation"
	officialFaultAfterEpochMeasurement         officialTargetFaultPhase = "after-epoch-measurement"
	officialFaultAfterTerminalResidueJoin      officialTargetFaultPhase = "after-terminal-residue-join"
	officialFaultAfterTargetBuild              officialTargetFaultPhase = "after-target-build"
	officialFaultAfterTargetRecordPersistence  officialTargetFaultPhase = "after-target-record-persistence"
	officialFaultAfterProvisionalValidation    officialTargetFaultPhase = "after-provisional-validation"
	officialFaultAfterFinalResidueJoin         officialTargetFaultPhase = "after-final-residue-join"
	officialFaultAfterFinalPolicyJoin          officialTargetFaultPhase = "after-final-policy-join"
	officialFaultAfterFinalMaterializationJoin officialTargetFaultPhase = "after-final-materialization-join"
	officialFaultAfterFinalRuntimeJoin         officialTargetFaultPhase = "after-final-runtime-join"
	officialFaultAfterFinalEpochJoin           officialTargetFaultPhase = "after-final-epoch-join"
	officialFaultAfterFinalAttemptJoin         officialTargetFaultPhase = "after-final-attempt-join"
	officialFaultAfterFinalTargetRecordJoin    officialTargetFaultPhase = "after-final-target-record-join"
	officialFaultBeforeAuthoritySeal           officialTargetFaultPhase = "before-authority-seal"
)

type officialTargetState struct {
	store        *store.ObjectStore
	residue      node.Residue
	head         store.HeadToken
	repository   gitobj.Repository
	policy       gitobj.Policy
	source       gitobj.InspectedTree
	receipt      gitobj.MaterializationReceipt
	attempt      store.ConformanceAttemptRecord
	record       store.ContractTargetRecord
	runtime      noderuntime.Runtime
	epoch        hostepoch.Epoch
	model        model.ContractExecutionTarget
	attemptFacts officialAttemptFacts
	recordFacts  officialRecordFacts
	gate         *sync.RWMutex
	closed       bool
	seal         *officialTargetSeal
}

type officialAttemptFacts struct {
	digest                      domain.Digest
	contractBundleDigest        domain.Digest
	residueHeadDigest           domain.Digest
	treeIdentityDigest          domain.Digest
	materializationPolicyDigest domain.Digest
	instanceNonce               string
	roots                       store.ConformanceAttemptRoots
}

type officialRecordFacts struct {
	digest        domain.Digest
	attemptDigest domain.Digest
}

// OfficialTarget is the only live pre-spawn target authority. Copies share a
// revocation cell. Valid proves construction integrity, never freshness. A
// physical runner must consume only a fresh ReopenOfficialTarget result at its
// immediately adjacent boundary.
type OfficialTarget struct{ state *officialTargetState }

// PublishOfficialTarget performs no subject spawn, interlock, claim, permit,
// run, classification, or study-head mutation.
func PublishOfficialTarget(ctx context.Context, request PublishOfficialTargetRequest) (OfficialTarget, error) {
	return publishOfficialTarget(ctx, request, nil)
}

func publishOfficialTarget(
	ctx context.Context,
	request PublishOfficialTargetRequest,
	fault officialTargetFault,
) (OfficialTarget, error) {
	if ctx == nil || request.Store == nil || !request.Residue.Valid() || !request.Repository.Valid() ||
		request.DisplayRef == "" || request.NodeExecutable == "" {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "complete live publication inputs are required", nil)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterInputsValidated); err != nil {
		return OfficialTarget{}, err
	}
	residue, head, err := reopenResidueHead(ctx, request.Store, request.Residue)
	if err != nil {
		return OfficialTarget{}, err
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterInitialResidueReopen); err != nil {
		return OfficialTarget{}, err
	}
	policy, err := policyForResidue(residue)
	if err != nil {
		return OfficialTarget{}, err
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterPolicyDerivation); err != nil {
		return OfficialTarget{}, err
	}
	pinned, err := request.Repository.Pin(ctx, request.DisplayRef)
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "display ref could not be pinned", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterTreePin); err != nil {
		return OfficialTarget{}, err
	}
	attempt, err := request.Store.AllocateConformanceAttempt(ctx, store.ConformanceAttemptInput{
		ContractBundleDigest: residue.BundleDigest(), ResidueHeadDigest: head.HeadDigest(),
		TreeIdentityDigest: pinned.IdentityDigest(), MaterializationPolicyDigest: policy.Digest(),
	})
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "fresh attempt allocation failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterAttemptAllocation); err != nil {
		return OfficialTarget{}, err
	}
	roots := attempt.Roots()
	source, err := gitobj.Inspect(ctx, pinned, policy, roots.CandidateParent())
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "pinned target inspection failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterTreeInspection); err != nil {
		return OfficialTarget{}, err
	}
	if !sourceProfileJoinsTree(residue.Bundle(), source) {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "declared subject entrypoint is absent from the inspected tree", nil)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterEntrypointJoin); err != nil {
		return OfficialTarget{}, err
	}
	receipt, err := gitobj.MaterializeSingleTarget(ctx, source, roots.CandidateParent())
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "single target materialization failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterMaterialization); err != nil {
		return OfficialTarget{}, err
	}
	receipt, err = gitobj.ReopenSingleTarget(ctx, source, roots.CandidateParent())
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "single target did not reopen", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterMaterializationReopen); err != nil {
		return OfficialTarget{}, err
	}
	runtimeAuthority, err := noderuntime.Admit(ctx, request.NodeExecutable, roots.TemporaryRoot())
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "Node runtime admission failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterRuntimeAdmission); err != nil {
		return OfficialTarget{}, err
	}
	runtimeAuthority, err = runtimeAuthority.Revalidate(ctx)
	if err != nil {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "Node runtime changed during publication", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterRuntimeRevalidation); err != nil {
		return OfficialTarget{}, err
	}
	epoch, err := hostepoch.Measure(ctx)
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "host epoch measurement failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterEpochMeasurement); err != nil {
		return OfficialTarget{}, err
	}
	residue, finalHead, err := reopenResidueHead(ctx, request.Store, residue)
	if err != nil || !sameHead(head, finalHead) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "terminal residue changed during publication", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterTerminalResidueJoin); err != nil {
		return OfficialTarget{}, err
	}
	target, err := buildTarget(residue, finalHead, request.Repository, source, receipt, attempt, runtimeAuthority, epoch)
	if err != nil {
		return OfficialTarget{}, err
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterTargetBuild); err != nil {
		return OfficialTarget{}, err
	}
	record, err := request.Store.PersistContractTargetRecord(ctx, attempt, target)
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "inert target record publication failed", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterTargetRecordPersistence); err != nil {
		return OfficialTarget{}, err
	}
	return rejoinOfficial(ctx, &officialTargetState{
		store: request.Store, residue: residue, head: finalHead, repository: request.Repository,
		policy: policy, source: source, receipt: receipt, attempt: attempt,
		record: record, runtime: runtimeAuthority, epoch: epoch, model: target,
	}, fault)
}

// OpenOfficialTarget reconstructs the complete live graph after a store or
// process restart. TargetDigest is a locator and cannot bypass any live edge.
func OpenOfficialTarget(ctx context.Context, request OpenOfficialTargetRequest) (OfficialTarget, error) {
	if ctx == nil || request.Store == nil || !request.Residue.Valid() || !request.Repository.Valid() ||
		!request.TargetDigest.Valid() {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "complete live reopen inputs are required", nil)
	}
	residue, head, err := reopenResidueHead(ctx, request.Store, request.Residue)
	if err != nil {
		return OfficialTarget{}, err
	}
	policy, err := policyForResidue(residue)
	if err != nil {
		return OfficialTarget{}, err
	}
	object, authority, err := request.Store.Read(ctx, model.TargetKind, request.TargetDigest)
	if err != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "target locator did not reopen exact object authority", err)
	}
	if validationErr := request.Store.Validate(ctx, object, authority); validationErr != nil {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "target locator did not reopen exact object authority", validationErr)
	}
	target, err := model.ParseContractExecutionTarget(object.CanonicalBytes(), request.TargetDigest)
	if err != nil || !targetJoinsResidue(target, residue, head) ||
		target.Input().Tree.MaterializationPolicyDigest != policy.Digest() {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "target object does not join live residue or policy", err)
	}
	attempt, err := request.Store.OpenConformanceAttempt(ctx, target.AttemptArtifactDigest())
	if err != nil || !attemptJoinsModel(attempt, target) {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "target attempt did not reopen exactly", err)
	}
	record, err := request.Store.OpenContractTargetRecord(ctx, attempt)
	if err != nil || record.Digest() != target.Digest() {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "typed target relation did not reopen", err)
	}
	targetInput := target.Input()
	pinned, err := request.Repository.Pin(ctx, targetInput.Tree.CommitOID)
	if err != nil || pinned.IdentityDigest() != targetInput.Tree.TreeIdentityDigest ||
		pinned.Provenance().TreeOID != targetInput.Tree.TreeOID ||
		string(request.Repository.ObjectFormat()) != targetInput.Tree.ObjectFormat {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "pinned target objects differ on reopen", err)
	}
	source, err := gitobj.Inspect(ctx, pinned, policy, attempt.Roots().CandidateParent())
	if err != nil {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "target source did not reinspect", err)
	}
	if !sourceProfileJoinsTree(residue.Bundle(), source) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "declared subject entrypoint differs on reopen", nil)
	}
	receipt, err := gitobj.ReopenSingleTarget(ctx, source, attempt.Roots().CandidateParent())
	if err != nil || !treeBindingMatches(
		targetInput.Tree,
		request.Repository,
		source,
		receipt,
		attempt.Roots().CandidateParent(),
	) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "published target tree differs on reopen", err)
	}
	runtimeAuthority, err := noderuntime.Admit(ctx, targetInput.Runtime.AdmittedExecutablePath, attempt.Roots().TemporaryRoot())
	if err != nil || !runtimeBindingMatches(targetInput.Runtime, runtimeAuthority) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "Node runtime differs on reopen", err)
	}
	epoch, err := hostepoch.Measure(ctx)
	if err != nil || epoch.Digest() != targetInput.BootSession.IdentityDigest {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "host epoch differs on reopen", err)
	}
	return rejoinOfficial(ctx, &officialTargetState{
		store: request.Store, residue: residue, head: head, repository: request.Repository,
		policy: policy, source: source, receipt: receipt, attempt: attempt,
		record: record, runtime: runtimeAuthority, epoch: epoch, model: target,
	}, nil)
}

// ReopenOfficialTarget consumes only a still-live capability and repeats the
// same full-live graph reconstruction as restart recovery.
func ReopenOfficialTarget(ctx context.Context, retained OfficialTarget) (OfficialTarget, error) {
	return reopenOfficialTarget(ctx, retained, OpenOfficialTarget)
}

type officialTargetOpener func(context.Context, OpenOfficialTargetRequest) (OfficialTarget, error)

func reopenOfficialTarget(ctx context.Context, retained OfficialTarget, opener officialTargetOpener) (OfficialTarget, error) {
	state := retained.state
	if state == nil || state.gate == nil || opener == nil {
		return OfficialTarget{}, refuse(CodeOfficialTargetClosed, "official target is closed, zero, or changed", nil)
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return OfficialTarget{}, refuse(CodeOfficialTargetClosed, "official target is closed, zero, or changed", nil)
	}
	return opener(ctx, OpenOfficialTargetRequest{
		Store: state.store, Residue: state.residue, Repository: state.repository,
		TargetDigest: state.model.Digest(),
	})
}

func rejoinOfficial(ctx context.Context, provisional *officialTargetState, fault officialTargetFault) (OfficialTarget, error) {
	if provisional == nil || provisional.store == nil || !provisional.model.Valid() ||
		!provisional.attempt.Valid() || !provisional.record.Valid() || !provisional.runtime.Valid() ||
		!provisional.epoch.Valid() || !provisional.source.Valid() || !provisional.receipt.Valid() {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "provisional official target is incomplete", nil)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterProvisionalValidation); err != nil {
		return OfficialTarget{}, err
	}
	residue, head, err := reopenResidueHead(ctx, provisional.store, provisional.residue)
	if err != nil || !sameHead(head, provisional.head) || !targetJoinsResidue(provisional.model, residue, head) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "residue changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalResidueJoin); err != nil {
		return OfficialTarget{}, err
	}
	policy, err := policyForResidue(residue)
	if err != nil || policy.Digest() != provisional.policy.Digest() || !sourceProfileJoinsTree(residue.Bundle(), provisional.source) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "bundle execution profile differs during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalPolicyJoin); err != nil {
		return OfficialTarget{}, err
	}
	receipt, err := gitobj.ReopenSingleTarget(ctx, provisional.source, provisional.attempt.Roots().CandidateParent())
	if err != nil || !sameReceipt(receipt, provisional.receipt) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "materialization changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalMaterializationJoin); err != nil {
		return OfficialTarget{}, err
	}
	runtimeAuthority, err := provisional.runtime.Revalidate(ctx)
	if err != nil || !runtimeBindingMatches(provisional.model.Input().Runtime, runtimeAuthority) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "runtime changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalRuntimeJoin); err != nil {
		return OfficialTarget{}, err
	}
	epoch, err := provisional.epoch.Revalidate(ctx)
	if err != nil || epoch.Digest() != provisional.model.Input().BootSession.IdentityDigest {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "host epoch changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalEpochJoin); err != nil {
		return OfficialTarget{}, err
	}
	attempt, err := provisional.store.OpenConformanceAttempt(ctx, provisional.attempt.Digest())
	if err != nil || !attemptJoinsModel(attempt, provisional.model) {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "attempt changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalAttemptJoin); err != nil {
		return OfficialTarget{}, err
	}
	record, err := provisional.store.OpenContractTargetRecord(ctx, attempt)
	if err != nil || record.Digest() != provisional.model.Digest() || record.AttemptDigest() != attempt.Digest() {
		return OfficialTarget{}, refuse(CodeOfficialTargetChanged, "target relation changed during final rejoin", err)
	}
	if err := injectOfficialTargetFault(fault, officialFaultAfterFinalTargetRecordJoin); err != nil {
		return OfficialTarget{}, err
	}
	attemptFacts, recordFacts, err := captureOfficialTargetFacts(attempt, record, provisional.model)
	if err != nil {
		return OfficialTarget{}, err
	}
	if err := injectOfficialTargetFault(fault, officialFaultBeforeAuthoritySeal); err != nil {
		return OfficialTarget{}, err
	}
	state := &officialTargetState{
		store: provisional.store, residue: residue, head: head, repository: provisional.repository,
		policy: provisional.policy, source: provisional.source, receipt: receipt, attempt: attempt,
		record: record, runtime: runtimeAuthority, epoch: epoch, model: provisional.model,
		attemptFacts: attemptFacts, recordFacts: recordFacts,
		gate: &sync.RWMutex{}, seal: issuedOfficialTargetSeal,
	}
	official := OfficialTarget{state: state}
	if !official.Valid() {
		return OfficialTarget{}, refuse(CodeInvalidOfficialTarget, "final official target failed sealed construction", nil)
	}
	return official, nil
}

func captureOfficialTargetFacts(
	attempt store.ConformanceAttemptRecord,
	record store.ContractTargetRecord,
	target model.ContractExecutionTarget,
) (officialAttemptFacts, officialRecordFacts, error) {
	if !attempt.Valid() || !record.Valid() || !target.Valid() {
		return officialAttemptFacts{}, officialRecordFacts{}, refuse(CodeInvalidOfficialTarget, "sealed target facts are incomplete", nil)
	}
	attemptFacts := officialAttemptFacts{
		digest: attempt.Digest(), contractBundleDigest: attempt.ContractBundleDigest(),
		residueHeadDigest: attempt.ResidueHeadDigest(), treeIdentityDigest: attempt.TreeIdentityDigest(),
		materializationPolicyDigest: attempt.MaterializationPolicyDigest(), instanceNonce: attempt.InstanceNonce(),
		roots: attempt.Roots(),
	}
	recordFacts := officialRecordFacts{digest: record.Digest(), attemptDigest: record.AttemptDigest()}
	if !attemptFacts.joins(target) || !recordFacts.joins(target, attemptFacts) {
		return officialAttemptFacts{}, officialRecordFacts{}, refuse(CodeInvalidOfficialTarget, "sealed target facts do not join the target", nil)
	}
	return attemptFacts, recordFacts, nil
}

func (facts officialAttemptFacts) joins(target model.ContractExecutionTarget) bool {
	if !target.Valid() {
		return false
	}
	input := target.Input()
	attemptRoot := facts.roots.AttemptRoot()
	digestHex := strings.TrimPrefix(facts.digest.String(), "sha256:")
	if !facts.digest.Valid() || digestHex == facts.digest.String() || !filepath.IsAbs(attemptRoot) ||
		filepath.Clean(attemptRoot) != attemptRoot || filepath.Base(attemptRoot) != digestHex ||
		facts.digest != input.Attempt.ArtifactDigest || facts.instanceNonce != input.Attempt.InstanceNonce ||
		facts.contractBundleDigest != input.ContractBundleDigest ||
		facts.residueHeadDigest != input.TerminalResidue.HeadDigest ||
		facts.treeIdentityDigest != input.Tree.TreeIdentityDigest ||
		facts.materializationPolicyDigest != input.Tree.MaterializationPolicyDigest {
		return false
	}
	return facts.roots.CandidateParent() == filepath.Join(attemptRoot, "candidate-parent") &&
		facts.roots.FixtureRoot() == filepath.Join(attemptRoot, "fixture") &&
		facts.roots.HomeRoot() == filepath.Join(attemptRoot, "home") &&
		facts.roots.TemporaryRoot() == filepath.Join(attemptRoot, "tmp") &&
		facts.roots.XDGConfigRoot() == filepath.Join(attemptRoot, "xdg-config") &&
		facts.roots.XDGCacheRoot() == filepath.Join(attemptRoot, "xdg-cache") &&
		facts.roots.XDGDataRoot() == filepath.Join(attemptRoot, "xdg-data") &&
		facts.roots.XDGStateRoot() == filepath.Join(attemptRoot, "xdg-state") &&
		facts.roots.StateRoot() == filepath.Join(attemptRoot, "state") &&
		facts.roots.EvidenceRoot() == filepath.Join(attemptRoot, "evidence") &&
		facts.roots.MarkerPath() == filepath.Join(attemptRoot, "evidence", "attempt.marker.json")
}

func (facts officialRecordFacts) joins(target model.ContractExecutionTarget, attempt officialAttemptFacts) bool {
	return target.Valid() && facts.digest == target.Digest() && facts.attemptDigest == attempt.digest
}

func injectOfficialTargetFault(fault officialTargetFault, phase officialTargetFaultPhase) error {
	if fault == nil {
		return nil
	}
	if err := fault(phase); err != nil {
		return refuse(CodeInvalidOfficialTarget, "publication interrupted at "+string(phase), err)
	}
	return nil
}

func buildTarget(
	residue node.Residue,
	head store.HeadToken,
	repository gitobj.Repository,
	source gitobj.InspectedTree,
	receipt gitobj.MaterializationReceipt,
	attempt store.ConformanceAttemptRecord,
	runtimeAuthority noderuntime.Runtime,
	epoch hostepoch.Epoch,
) (model.ContractExecutionTarget, error) {
	provenance := source.Provenance()
	target, err := model.NewContractExecutionTarget(model.TargetInput{
		ContractBundleDigest: residue.BundleDigest(),
		TerminalResidue: model.TerminalResidueBinding{
			StudyID: residue.StudyID().String(), HeadRevision: head.Revision(), HeadDigest: head.HeadDigest(),
			LineageRootDigest: head.LineageRootDigest(),
		},
		Tree: model.TreeBinding{
			ObjectFormat: string(repository.ObjectFormat()), CommitOID: provenance.CommitOID, TreeOID: provenance.TreeOID,
			TreeIdentityDigest: source.TreeIdentityDigest(), PortableTreeDigest: source.PortableTreeDigest(),
			MaterializationPolicyDigest: source.PolicyDigest(), MaterializationManifestDigest: receipt.ManifestDigest,
		},
		Attempt:     model.AttemptBinding{ArtifactDigest: attempt.Digest(), InstanceNonce: attempt.InstanceNonce()},
		BootSession: model.BootSessionBinding{IdentityDigest: epoch.Digest()},
		Runtime:     runtimeBinding(runtimeAuthority),
	})
	if err != nil {
		return model.ContractExecutionTarget{}, refuse(CodeInvalidOfficialTarget, "exact target model construction failed", err)
	}
	return target, nil
}

func runtimeBinding(runtimeAuthority noderuntime.Runtime) model.RuntimeBinding {
	return model.RuntimeBinding{
		AdmittedExecutablePath: runtimeAuthority.Path(), MeasuredProcessExecPath: runtimeAuthority.Path(),
		Version: runtimeAuthority.Version(), Major: runtimeAuthority.Major(), Platform: runtimeAuthority.Platform(),
		Architecture: runtimeAuthority.Architecture(), ExecutableBytesDigest: runtimeAuthority.ExecutableBytesDigest(),
		ExecutableMode: runtimeAuthority.ExecutableMode(), ExecutableByteCount: runtimeAuthority.ExecutableByteCount(),
		ProbeProgramDigest: runtimeAuthority.ProbeProgramDigest(),
	}
}

func runtimeBindingMatches(expected model.RuntimeBinding, actual noderuntime.Runtime) bool {
	return actual.Valid() && expected == runtimeBinding(actual)
}

func treeBindingMatches(
	expected model.TreeBinding,
	repository gitobj.Repository,
	source gitobj.InspectedTree,
	receipt gitobj.MaterializationReceipt,
	candidateParent string,
) bool {
	provenance := source.Provenance()
	sourceEntries := source.Entries()
	if !filepath.IsAbs(candidateParent) || filepath.Clean(candidateParent) != candidateParent ||
		receipt.PublishedRoot != filepath.Join(candidateParent, "candidate") ||
		receipt.ManifestPath != filepath.Join(candidateParent, "materialization.manifest.json") ||
		receipt.PortableTreeDigest != source.PortableTreeDigest() ||
		receipt.PolicyDigest != source.PolicyDigest() || receipt.ObjectFormat != repository.ObjectFormat() ||
		receipt.RepositoryFingerprint != repository.Fingerprint() || len(receipt.Entries) != len(sourceEntries) {
		return false
	}
	for index := range sourceEntries {
		if receipt.Entries[index] != sourceEntries[index] {
			return false
		}
	}
	return source.Valid() && receipt.Valid() && expected.ObjectFormat == string(repository.ObjectFormat()) &&
		expected.CommitOID == provenance.CommitOID && expected.TreeOID == provenance.TreeOID &&
		expected.TreeIdentityDigest == source.TreeIdentityDigest() && expected.PortableTreeDigest == source.PortableTreeDigest() &&
		expected.MaterializationPolicyDigest == source.PolicyDigest() &&
		expected.MaterializationManifestDigest == receipt.ManifestDigest
}

func policyForResidue(residue node.Residue) (gitobj.Policy, error) {
	if !residue.Valid() || !residue.Bundle().Valid() {
		return gitobj.Policy{}, refuse(CodeInvalidOfficialTarget, "terminal residue bundle is invalid", nil)
	}
	plan := residue.Bundle().PortableSource().Plan()
	policy, err := gitobj.PolicyFromBudgets(plan.Budgets())
	if err != nil || !policy.Valid() || policy.Digest() != plan.MaterializationPolicyDigest() {
		return gitobj.Policy{}, refuse(CodeInvalidOfficialTarget, "bundle plan does not derive one exact materialization policy", err)
	}
	return policy, nil
}

func sourceProfileJoinsTree(bundle emitmodel.ContractBundle, source gitobj.InspectedTree) bool {
	if !bundle.Valid() || !source.Valid() {
		return false
	}
	profile := bundle.SourceProfile()
	if !profile.Valid() {
		return false
	}
	matches := 0
	for _, entry := range source.Entries() {
		if entry.Path != profile.SubjectEntrypoint() {
			continue
		}
		if entry.Mode != "100644" && entry.Mode != "100755" {
			return false
		}
		matches++
	}
	return matches == 1
}

func attemptJoinsModel(attempt store.ConformanceAttemptRecord, target model.ContractExecutionTarget) bool {
	if !attempt.Valid() || !target.Valid() {
		return false
	}
	input := target.Input()
	return attempt.Digest() == input.Attempt.ArtifactDigest && attempt.InstanceNonce() == input.Attempt.InstanceNonce &&
		attempt.ContractBundleDigest() == input.ContractBundleDigest &&
		attempt.ResidueHeadDigest() == input.TerminalResidue.HeadDigest &&
		attempt.TreeIdentityDigest() == input.Tree.TreeIdentityDigest &&
		attempt.MaterializationPolicyDigest() == input.Tree.MaterializationPolicyDigest
}

func targetJoinsResidue(target model.ContractExecutionTarget, residue node.Residue, head store.HeadToken) bool {
	if !target.Valid() || !residue.Valid() {
		return false
	}
	input := target.Input()
	return input.ContractBundleDigest == residue.BundleDigest() && input.TerminalResidue.StudyID == residue.StudyID().String() &&
		input.TerminalResidue.HeadRevision == head.Revision() && input.TerminalResidue.HeadDigest == head.HeadDigest() &&
		input.TerminalResidue.LineageRootDigest == head.LineageRootDigest() && head.Stage() == store.StageResidue &&
		head.CurrentKind() == "ContractBundle" && head.CurrentDigest() == residue.BundleDigest()
}

func reopenResidueHead(ctx context.Context, objectStore *store.ObjectStore, retained node.Residue) (node.Residue, store.HeadToken, error) {
	fresh, err := node.ReopenResidue(ctx, objectStore, retained)
	if err != nil {
		return node.Residue{}, store.HeadToken{}, refuse(CodeOfficialTargetChanged, "terminal residue did not reopen", err)
	}
	head, err := objectStore.OpenHead(ctx, fresh.StudyID())
	if err != nil || head.HeadDigest() != fresh.HeadDigest() || head.Stage() != store.StageResidue ||
		head.CurrentKind() != "ContractBundle" || head.CurrentDigest() != fresh.BundleDigest() {
		return node.Residue{}, store.HeadToken{}, refuse(CodeOfficialTargetChanged, "terminal residue and head differ", err)
	}
	return fresh, head, nil
}

func sameHead(left, right store.HeadToken) bool {
	leftPriorHead, leftHasPriorHead := left.PreviousHeadDigest()
	rightPriorHead, rightHasPriorHead := right.PreviousHeadDigest()
	leftPriorObject, leftHasPriorObject := left.PreviousObjectDigest()
	rightPriorObject, rightHasPriorObject := right.PreviousObjectDigest()
	return left.StudyID().String() == right.StudyID().String() && left.Revision() == right.Revision() &&
		left.Stage() == right.Stage() && left.CurrentKind() == right.CurrentKind() &&
		left.CurrentDigest() == right.CurrentDigest() && left.LineageRootDigest() == right.LineageRootDigest() &&
		left.HeadDigest() == right.HeadDigest() && leftHasPriorHead == rightHasPriorHead && leftPriorHead == rightPriorHead &&
		leftHasPriorObject == rightHasPriorObject && leftPriorObject == rightPriorObject
}

func sameReceipt(left, right gitobj.MaterializationReceipt) bool {
	if !left.Valid() || !right.Valid() || left.ManifestDigest != right.ManifestDigest ||
		left.PortableTreeDigest != right.PortableTreeDigest || left.PolicyDigest != right.PolicyDigest ||
		left.TargetFilesystem != right.TargetFilesystem || left.PublishedRoot != right.PublishedRoot ||
		left.ManifestPath != right.ManifestPath || left.ObjectFormat != right.ObjectFormat ||
		left.RepositoryFingerprint != right.RepositoryFingerprint || len(left.Entries) != len(right.Entries) {
		return false
	}
	for index := range left.Entries {
		if left.Entries[index] != right.Entries[index] {
			return false
		}
	}
	return true
}

func (target OfficialTarget) Valid() bool {
	state := target.state
	if state == nil || state.gate == nil {
		return false
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	return state.validLocked()
}

func (state *officialTargetState) validLocked() bool {
	return state != nil && state.seal == issuedOfficialTargetSeal && state.gate != nil && !state.closed &&
		state.store != nil && state.residue.Valid() && state.repository.Valid() && state.policy.Valid() &&
		state.source.Valid() && state.receipt.Valid() &&
		state.runtime.Valid() && state.epoch.Valid() && state.model.Valid() &&
		targetJoinsResidue(state.model, state.residue, state.head) && state.attemptFacts.joins(state.model) &&
		state.recordFacts.joins(state.model, state.attemptFacts) &&
		policyMatchesResidue(state.policy, state.residue) && sourceProfileJoinsTree(state.residue.Bundle(), state.source) &&
		treeBindingMatches(
			state.model.Input().Tree,
			state.repository,
			state.source,
			state.receipt,
			state.attemptFacts.roots.CandidateParent(),
		) &&
		runtimeBindingMatches(state.model.Input().Runtime, state.runtime) &&
		state.model.Input().BootSession.IdentityDigest == state.epoch.Digest()
}

func policyMatchesResidue(policy gitobj.Policy, residue node.Residue) bool {
	derived, err := policyForResidue(residue)
	return err == nil && policy.Valid() && policy.Digest() == derived.Digest()
}

func (target OfficialTarget) Close() error {
	state := target.state
	if state == nil || state.gate == nil || state.seal != issuedOfficialTargetSeal {
		return refuse(CodeInvalidOfficialTarget, "official target is zero or forged", nil)
	}
	state.gate.Lock()
	defer state.gate.Unlock()
	if state.closed {
		return refuse(CodeOfficialTargetClosed, "official target is already closed", nil)
	}
	state.closed = true
	return nil
}

func (target OfficialTarget) Digest() domain.Digest {
	state := target.state
	if state == nil || state.gate == nil {
		return ""
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return ""
	}
	return state.model.Digest()
}
func (target OfficialTarget) Model() model.ContractExecutionTarget {
	state := target.state
	if state == nil || state.gate == nil {
		return model.ContractExecutionTarget{}
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return model.ContractExecutionTarget{}
	}
	fresh, err := model.ParseContractExecutionTarget(state.model.CanonicalBytes(), state.model.Digest())
	if err != nil {
		return model.ContractExecutionTarget{}
	}
	return fresh
}
func (target OfficialTarget) TargetRecord() store.ContractTargetRecord {
	state := target.state
	if state == nil || state.gate == nil {
		return store.ContractTargetRecord{}
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return store.ContractTargetRecord{}
	}
	return state.record
}
func (target OfficialTarget) Roots() store.ConformanceAttemptRoots {
	state := target.state
	if state == nil || state.gate == nil {
		return store.ConformanceAttemptRoots{}
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return store.ConformanceAttemptRoots{}
	}
	return state.attemptFacts.roots
}
func (target OfficialTarget) CandidateRoot() string {
	state := target.state
	if state == nil || state.gate == nil {
		return ""
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return ""
	}
	return state.receipt.PublishedRoot
}

// ContractBundle returns a defensive immutable copy of the exact generated
// bundle the C4 runner must execute. It exposes no residue or store authority.
func (target OfficialTarget) ContractBundle() emitmodel.ContractBundle {
	state := target.state
	if state == nil || state.gate == nil {
		return emitmodel.ContractBundle{}
	}
	state.gate.RLock()
	defer state.gate.RUnlock()
	if !state.validLocked() {
		return emitmodel.ContractBundle{}
	}
	return state.residue.Bundle()
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}
