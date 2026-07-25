//go:build darwin

package runner

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
	"github.com/nelsonwerd/countershape/internal/store"
)

func executeCLI(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	if ctx == nil || !retained.Valid() {
		return store.ContractExecutionRecord{}, refuse(CodeInvalidRequest, "live context and official target are required", nil)
	}
	first, err := contractexec.ReopenOfficialTarget(ctx, retained)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "official target did not freshly reopen", err)
	}
	preparation, err := prepareCLIExecution(first)
	if err != nil {
		_ = first.Close()
		return store.ContractExecutionRecord{}, err
	}
	input := cliExecutionInput{target: first, cliExecutionPreparation: preparation}
	defer func() { _ = input.closeProbe() }()
	second, err := contractexec.ReopenOfficialTarget(ctx, first)
	if err != nil {
		_ = first.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target changed while the physical operation was prepared", err)
	}
	if !samePreparedTarget(input.cliExecutionPreparation, second) {
		_ = second.Close()
		_ = first.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "successive fresh target authorities differ", nil)
	}
	_ = first.Close()
	input.target = second
	defer func() { _ = second.Close() }()
	epoch, err := hostepoch.Measure(ctx)
	if err != nil || epoch.Digest() != input.targetIdentity.model.Input().BootSession.IdentityDigest {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "live host epoch differs immediately before admission", err)
	}
	owner, err := store.AcquireContractRunOwner(ctx, second.TargetRecord(), epoch)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeAdmissionRefused, "contract-run admission did not produce a fresh owner", err)
	}
	closureContext, cancelClosure := terminalClosureContext(ctx, input)
	defer cancelClosure()
	spawnTarget, err := contractexec.ReopenOfficialTarget(closureContext, input.target)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target changed after admission and immediately before spawn", err)
	}
	if !samePreparedTarget(input.cliExecutionPreparation, spawnTarget) {
		_ = spawnTarget.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "spawn-adjacent target authority differs from the admitted target", nil)
	}
	input.target = spawnTarget
	defer func() { _ = spawnTarget.Close() }()
	physicalObservation, running, startErr := consumeAndStart(
		ctx, closureContext, owner, input.prepared, input.binding,
	)
	if startErr != nil {
		if IsCode(startErr, CodeAdmissionRefused) {
			return store.ContractExecutionRecord{}, startErr
		}
		return closeStartError(closureContext, owner, input, startErr)
	}
	if running == nil || physicalObservation.PID < 1 ||
		physicalObservation.Binding.String() != input.prepared.BindingDigest().String() {
		if running != nil {
			running.AbortSpawnObservationPersistence()
			_ = running.Close()
		}
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent spawn observation lacks an exact owned process-group binding", nil)
	}
	spawn, err := contractmodel.NewChildPIDObservation(int64(physicalObservation.PID))
	if err != nil {
		running.AbortSpawnObservationPersistence()
		_ = running.Close()
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent child PID is outside the semantic profile", err)
	}
	if err := owner.PersistSpawnObservation(closureContext, spawn); err != nil {
		running.AbortSpawnObservationPersistence()
		_ = running.Close()
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent spawn observation did not become durable", err)
	}
	result := running.Close()
	measurements, _ := input.scope.finish(closureContext)
	input.invocationEvidence, err = inspectAndRetireCLIInvocationEvidence(
		closureContext, input.evidenceRoot, input.targetIdentity.markerPath, input.attemptID, input.logicalArgv,
		input.evidenceAuthority,
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "child invocation evidence did not close and retire exactly", err)
	}
	post, err := snapshotCandidate(input.targetIdentity.candidateRoot)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "candidate inventory could not be measured after terminal closure", err)
	}
	terminalTarget, err := contractexec.ReopenOfficialTarget(closureContext, input.target)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target or runtime failed terminal revalidation", err)
	}
	defer func() { _ = terminalTarget.Close() }()
	if !samePreparedTarget(input.cliExecutionPreparation, terminalTarget) {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "terminal target authority differs from the admitted target", nil)
	}
	input.target = terminalTarget
	draft, err := buildChildEvidenceDraft(input, input.targetIdentity.model, spawn, result, post, measurements)
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	return persistRunAndClassification(closureContext, owner, input, draft)
}

func consumeAndStart(
	processContext context.Context,
	closureContext context.Context,
	owner store.ContractRunOwner,
	prepared *processmechanics.Prepared,
	binding domain.Digest,
) (processmechanics.SpawnObservation, *processmechanics.Running, error) {
	if err := owner.ConsumeForStart(closureContext, binding); err != nil {
		return processmechanics.SpawnObservation{}, nil,
			refuse(CodeAdmissionRefused, "one-shot start authority could not be consumed", err)
	}
	return prepared.Start(processContext)
}

func terminalClosureContext(parent context.Context, input cliExecutionInput) (context.Context, context.CancelFunc) {
	budgets := input.source.Plan().Budgets()
	budget := time.Duration(budgets.ProbeMS+budgets.TeardownMS)*time.Millisecond + 30*time.Second
	return context.WithTimeout(context.WithoutCancel(parent), budget)
}

func closeStartError(
	ctx context.Context,
	owner store.ContractRunOwner,
	input cliExecutionInput,
	startErr error,
) (store.ContractExecutionRecord, error) {
	var failure *processmechanics.StartError
	if !errors.As(startErr, &failure) {
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "process start failed without a conclusive mechanics observation", startErr)
	}
	result := failure.Result()
	if result.Binding.String() != input.prepared.BindingDigest().String() {
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "start error lacks the exact prepared spawn attempt", startErr)
	}
	spawn, err := contractmodel.NewStartErrorObservation("OS_START_ERROR")
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "start-error observation did not become durable", err)
	}
	measurements, _ := input.scope.finish(ctx)
	input.invocationEvidence, err = inspectAndRetireCLIInvocationEvidence(
		ctx, input.evidenceRoot, input.targetIdentity.markerPath, input.attemptID, input.logicalArgv,
		input.evidenceAuthority,
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "start-error invocation evidence did not close and retire exactly", err)
	}
	post, err := snapshotCandidate(input.targetIdentity.candidateRoot)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "candidate inventory could not be closed after start error", err)
	}
	terminalTarget, err := contractexec.ReopenOfficialTarget(ctx, input.target)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target failed terminal start-error revalidation", err)
	}
	defer func() { _ = terminalTarget.Close() }()
	if !samePreparedTarget(input.cliExecutionPreparation, terminalTarget) {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "terminal start-error target differs", nil)
	}
	input.target = terminalTarget
	draft, err := buildStartErrorEvidenceDraft(input, input.targetIdentity.model, spawn, result, post, measurements)
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	return persistRunAndClassification(ctx, owner, input, draft)
}

func persistRunAndClassification(
	ctx context.Context,
	owner store.ContractRunOwner,
	input cliExecutionInput,
	draft evidenceDraft,
) (store.ContractExecutionRecord, error) {
	if err := input.evidenceCapacity.validate(draft.bodies); err != nil {
		return store.ContractExecutionRecord{}, refuse(
			CodeEvidenceClosureFailed,
			"private run evidence escaped its pre-admitted capacity envelope",
			err,
		)
	}
	manifest, err := owner.PersistPrivateRunManifest(ctx, draft.bodies)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "private run evidence did not persist exactly", err)
	}
	witness, err := draft.persistAndAssemble(owner, manifest)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "closed-run witness did not assemble", err)
	}
	targetModel := input.targetIdentity.model
	run, err := contractmodel.NewFinalizedContractRun(targetModel, owner.StartClaimDigest(), witness)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "finalized run did not derive", err)
	}
	closure, err := owner.PersistFinalizedRun(ctx, manifest, run)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "finalized run did not persist and reopen", err)
	}
	if err := closure.Release(ctx); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "terminal closure did not converge interlock release", err)
	}
	released := closure.FinalizedRun()
	execution, err := contractmodel.DeriveContractExecution(
		input.targetIdentity.bundle, targetModel, released.Model(),
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "contract classification did not derive", err)
	}
	record, err := store.PersistContractExecutionRecord(ctx, released, execution)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "contract classification did not persist and reopen", err)
	}
	return record, nil
}

func resumeCLIClassification(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	if ctx == nil || !retained.Valid() {
		return store.ContractExecutionRecord{}, refuse(CodeInvalidRequest, "live context and official target are required", nil)
	}
	fresh, err := contractexec.ReopenOfficialTarget(ctx, retained)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "official target did not freshly reopen for classification recovery", err)
	}
	defer func() { _ = fresh.Close() }()
	targetRecord := fresh.TargetRecord()
	targetModel := fresh.Model()
	run, openErr := store.OpenFinalizedRunRecord(ctx, targetRecord, targetModel)
	if openErr != nil {
		closure, closureErr := store.OpenTerminalClosure(ctx, targetRecord, targetModel)
		if closureErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "no exact terminal closure can resume classification", errors.Join(openErr, closureErr))
		}
		if releaseErr := closure.Release(ctx); releaseErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "terminal release did not converge during recovery", releaseErr)
		}
		run, openErr = store.OpenFinalizedRunRecord(ctx, targetRecord, targetModel)
		if openErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "released finalized run did not reopen", openErr)
		}
	}
	execution, err := contractmodel.DeriveContractExecution(fresh.ContractBundle(), targetModel, run.Model())
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "durable run classification did not derive", err)
	}
	record, err := store.PersistContractExecutionRecord(ctx, run, execution)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "durable classification did not converge", err)
	}
	return record, nil
}

// samePreparedTarget compares only the newly reopened live capability with the
// inert identity captured from the first fresh target. ReopenOfficialTarget
// already validates its retained predecessor under that predecessor's read
// lock before reconstructing the returned capability, so validating both
// getters again here adds no freshness edge and multiplies full bundle parses.
func samePreparedTarget(preparation cliExecutionPreparation, fresh contractexec.OfficialTarget) bool {
	if !preparation.targetIdentity.valid() {
		return false
	}
	freshModel := fresh.Model()
	if !freshModel.Valid() || !bytes.Equal(preparation.targetIdentity.model.CanonicalBytes(), freshModel.CanonicalBytes()) {
		return false
	}
	freshRoots := fresh.Roots()
	return preparation.targetIdentity.roots.AttemptRoot() == freshRoots.AttemptRoot() &&
		preparation.targetIdentity.roots.CandidateParent() == freshRoots.CandidateParent() &&
		preparation.targetIdentity.roots.FixtureRoot() == freshRoots.FixtureRoot() &&
		preparation.targetIdentity.roots.HomeRoot() == freshRoots.HomeRoot() &&
		preparation.targetIdentity.roots.TemporaryRoot() == freshRoots.TemporaryRoot() &&
		preparation.targetIdentity.roots.StateRoot() == freshRoots.StateRoot() &&
		preparation.targetIdentity.roots.EvidenceRoot() == freshRoots.EvidenceRoot()
}
