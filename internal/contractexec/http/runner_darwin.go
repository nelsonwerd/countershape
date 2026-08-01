//go:build darwin

package http

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	contractscope "github.com/nelsonwerd/countershape/internal/contractexec/scope"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/store"
)

func executeHTTP(
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
	preparation, err := prepareHTTPExecution(first)
	if err != nil {
		_ = first.Close()
		return store.ContractExecutionRecord{}, err
	}
	input := executionInput{target: first, executionPreparation: preparation}
	defer func() { _ = input.closePrepared() }()
	second, err := contractexec.ReopenOfficialTarget(ctx, first)
	if err != nil {
		_ = first.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target changed while HTTP execution was prepared", err)
	}
	if !samePreparedTarget(input.executionPreparation, second) {
		_ = second.Close()
		_ = first.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "successive fresh target authorities differ", nil)
	}
	_ = first.Close()
	input.target = second
	defer func() { _ = second.Close() }()
	if err := input.prepared.Revalidate(); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "admitted runtime changed before HTTP admission", err)
	}
	epoch, err := hostepoch.Measure(ctx)
	if err != nil || epoch.Digest() != input.targetIdentity.model.Input().BootSession.IdentityDigest {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "live host epoch differs immediately before admission", err)
	}
	owner, err := store.AcquireContractRunOwner(ctx, second.TargetRecord(), epoch)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeAdmissionRefused, "HTTP admission did not produce a fresh owner", err)
	}
	revalidationContext, cancelRevalidation := detachedHTTPPhaseContext(ctx, input)
	spawnTarget, reopenErr := contractexec.ReopenOfficialTarget(revalidationContext, input.target)
	closureContext, cancelClosure := detachedHTTPPhaseContext(ctx, input)
	defer cancelClosure()
	revalidationErr := revalidationContext.Err()
	cancelRevalidation()
	if reopenErr != nil || revalidationErr != nil {
		_ = spawnTarget.Close()
		return store.ContractExecutionRecord{}, refuse(
			CodeTargetChanged,
			"target changed or bounded revalidation expired after admission and immediately before start",
			errors.Join(reopenErr, revalidationErr),
		)
	}
	if !samePreparedTarget(input.executionPreparation, spawnTarget) {
		_ = spawnTarget.Close()
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "start-adjacent target differs from the admitted target", nil)
	}
	input.target = spawnTarget
	defer func() { _ = spawnTarget.Close() }()
	initial, running, startErr := consumeAndStartHTTP(
		ctx, closureContext, owner, input.prepared, input.binding,
	)
	if startErr != nil {
		if IsCode(startErr, CodeAdmissionRefused) {
			return store.ContractExecutionRecord{}, startErr
		}
		return closeHTTPStartError(closureContext, owner, input, initial, startErr)
	}
	if running == nil || initial.process.pid < 1 || !initial.process.started ||
		!initial.process.processGroupOwned || initial.process.processGroupID != initial.process.pid ||
		initial.binding != input.binding {
		if running != nil {
			_ = running.Close()
		}
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent start observation lacks an exact owned process-group binding", nil)
	}
	spawn, err := contractmodel.NewChildPIDObservation(int64(initial.process.pid))
	if err != nil {
		_ = running.Close()
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent child PID is outside the semantic profile", err)
	}
	if err := owner.PersistSpawnObservation(closureContext, spawn); err != nil {
		_ = running.Close()
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "parent start observation did not become durable", err)
	}
	result := running.Close()
	measurements, _ := input.finishProbe(closureContext)
	input.receiptFinding, err = inspectAndRetireInvocation(
		closureContext, input.evidenceRoot, input.targetIdentity.markerPath,
		input.attemptID, input.source.StimulusDigest(), result.exchange.requestWire,
		input.receipt,
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "HTTP invocation receipt did not close and retire exactly", err)
	}
	after, err := contractscope.Snapshot(input.targetIdentity.candidateRoot)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "candidate inventory could not be measured after terminal closure", err)
	}
	terminalTarget, err := contractexec.ReopenOfficialTarget(closureContext, input.target)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target or runtime failed terminal revalidation", err)
	}
	defer func() { _ = terminalTarget.Close() }()
	if !samePreparedTarget(input.executionPreparation, terminalTarget) {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "terminal target differs from the admitted target", nil)
	}
	input.target = terminalTarget
	draft, err := buildChildEvidenceDraft(input, input.targetIdentity.model, spawn, result, after, measurements)
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	return persistHTTPRunAndClassification(closureContext, owner, input, draft)
}

func consumeAndStartHTTP(
	processContext context.Context,
	closureContext context.Context,
	owner store.ContractRunOwner,
	prepared *preparedService,
	binding domain.Digest,
) (serviceResult, *runningService, error) {
	if err := owner.ConsumeForStart(closureContext, binding); err != nil {
		return serviceResult{}, nil, refuse(CodeAdmissionRefused, "one-shot HTTP start authority could not be consumed", err)
	}
	return prepared.Start(processContext, closureContext)
}

func detachedHTTPPhaseContext(parent context.Context, input executionInput) (context.Context, context.CancelFunc) {
	budgets := input.source.Plan().Budgets()
	budget := time.Duration(budgets.ReadinessMS+budgets.ProbeMS+budgets.TeardownMS)*time.Millisecond + 30*time.Second
	return context.WithTimeout(context.WithoutCancel(parent), budget)
}

func closeHTTPStartError(
	ctx context.Context,
	owner store.ContractRunOwner,
	input executionInput,
	result serviceResult,
	startErr error,
) (store.ContractExecutionRecord, error) {
	if !result.binding.Valid() || result.binding != input.binding ||
		result.process.started || result.process.pid != 0 ||
		!result.process.physicalExecutionEntered || !result.process.spawnAttempted {
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "start failure lacks the exact prepared attempt", startErr)
	}
	spawn, err := contractmodel.NewStartErrorObservation("OS_START_ERROR")
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeSpawnClosureFailed, "start-error observation did not become durable", err)
	}
	measurements, _ := input.finishProbe(ctx)
	input.receiptFinding, err = inspectAndRetireInvocation(
		ctx, input.evidenceRoot, input.targetIdentity.markerPath,
		input.attemptID, input.source.StimulusDigest(), result.exchange.requestWire,
		input.receipt,
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "start-error receipt closure failed", err)
	}
	after, err := contractscope.Snapshot(input.targetIdentity.candidateRoot)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "candidate inventory did not close after start error", err)
	}
	terminalTarget, err := contractexec.ReopenOfficialTarget(ctx, input.target)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "target failed terminal start-error revalidation", err)
	}
	defer func() { _ = terminalTarget.Close() }()
	if !samePreparedTarget(input.executionPreparation, terminalTarget) {
		return store.ContractExecutionRecord{}, refuse(CodeTargetChanged, "terminal start-error target differs", nil)
	}
	input.target = terminalTarget
	draft, err := buildStartErrorEvidenceDraft(
		input, input.targetIdentity.model, spawn, result, after, measurements,
	)
	if err != nil {
		return store.ContractExecutionRecord{}, err
	}
	return persistHTTPRunAndClassification(ctx, owner, input, draft)
}

func persistHTTPRunAndClassification(
	ctx context.Context,
	owner store.ContractRunOwner,
	input executionInput,
	draft evidenceDraft,
) (store.ContractExecutionRecord, error) {
	if err := input.capacity.validate(draft.bodies); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "private evidence escaped its pre-admitted envelope", err)
	}
	manifest, err := owner.PersistPrivateRunManifest(ctx, draft.bodies)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "private evidence did not persist exactly", err)
	}
	witness, err := draft.assemble(manifest)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "closed-run witness did not assemble", err)
	}
	target := input.targetIdentity.model
	run, err := contractmodel.NewFinalizedContractRun(target, owner.StartClaimDigest(), witness)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "finalized run did not derive", err)
	}
	closure, err := owner.PersistFinalizedRun(ctx, manifest, run)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "finalized run did not persist and reopen", err)
	}
	if err := closure.Release(ctx); err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "terminal closure did not converge release", err)
	}
	released := closure.FinalizedRun()
	execution, err := contractmodel.DeriveContractExecution(
		input.targetIdentity.bundle, target, released.Model(),
	)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "HTTP classification did not derive", err)
	}
	record, err := store.PersistContractExecutionRecord(ctx, released, execution)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeEvidenceClosureFailed, "HTTP classification did not persist and reopen", err)
	}
	return record, nil
}

func resumeHTTPClassification(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	if ctx == nil || !retained.Valid() {
		return store.ContractExecutionRecord{}, refuse(CodeInvalidRequest, "live context and official target are required", nil)
	}
	fresh, err := contractexec.ReopenOfficialTarget(ctx, retained)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "official target did not freshly reopen for recovery", err)
	}
	defer func() { _ = fresh.Close() }()
	bundle := fresh.ContractBundle()
	if !bundle.Valid() {
		return store.ContractExecutionRecord{}, refuse(
			CodeRecoveryRefused,
			"official HTTP bundle is invalid during classification recovery",
			nil,
		)
	}
	if _, profileErr := requireStandaloneHTTPProfile(
		bundle.PortableSource(),
		bundle.SourceProfile(),
	); profileErr != nil {
		return store.ContractExecutionRecord{}, profileErr
	}
	targetRecord := fresh.TargetRecord()
	target := fresh.Model()
	run, openErr := store.OpenFinalizedRunRecord(ctx, targetRecord, target)
	if openErr != nil {
		closure, closureErr := store.OpenTerminalClosure(ctx, targetRecord, target)
		if closureErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "no exact terminal closure can resume classification", errors.Join(openErr, closureErr))
		}
		if releaseErr := closure.Release(ctx); releaseErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "terminal release did not converge during recovery", releaseErr)
		}
		run, openErr = store.OpenFinalizedRunRecord(ctx, targetRecord, target)
		if openErr != nil {
			return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "released finalized run did not reopen", openErr)
		}
	}
	execution, err := contractmodel.DeriveContractExecution(bundle, target, run.Model())
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "durable HTTP classification did not derive", err)
	}
	record, err := store.PersistContractExecutionRecord(ctx, run, execution)
	if err != nil {
		return store.ContractExecutionRecord{}, refuse(CodeRecoveryRefused, "durable HTTP classification did not converge", err)
	}
	return record, nil
}

func samePreparedTarget(preparation executionPreparation, fresh contractexec.OfficialTarget) bool {
	if !preparation.targetIdentity.valid() {
		return false
	}
	model := fresh.Model()
	if !model.Valid() || !bytes.Equal(preparation.targetIdentity.model.CanonicalBytes(), model.CanonicalBytes()) {
		return false
	}
	roots := fresh.Roots()
	return preparation.targetIdentity.roots.AttemptRoot() == roots.AttemptRoot() &&
		preparation.targetIdentity.roots.CandidateParent() == roots.CandidateParent() &&
		preparation.targetIdentity.roots.FixtureRoot() == roots.FixtureRoot() &&
		preparation.targetIdentity.roots.HomeRoot() == roots.HomeRoot() &&
		preparation.targetIdentity.roots.TemporaryRoot() == roots.TemporaryRoot() &&
		preparation.targetIdentity.roots.StateRoot() == roots.StateRoot() &&
		preparation.targetIdentity.roots.EvidenceRoot() == roots.EvidenceRoot()
}
