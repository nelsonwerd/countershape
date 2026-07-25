//go:build darwin && cgo

package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
)

type c4BridgeFixture struct {
	store        *ObjectStore
	root         string
	attempt      ConformanceAttemptRecord
	target       contractmodel.ContractExecutionTarget
	targetRecord ContractTargetRecord
	epoch        hostepoch.Epoch
	owner        ContractRunOwner
	spawn        contractmodel.SpawnObservation
	binding      domain.Digest
	manifest     PrivateRunManifest
	run          contractmodel.FinalizedContractRun
	closure      TerminalClosure
}

func c4TargetAndOwner(t *testing.T) (*ObjectStore, string, ConformanceAttemptRecord, contractmodel.ContractExecutionTarget, ContractTargetRecord, hostepoch.Epoch, ContractRunOwner) {
	t.Helper()
	ctx := context.Background()
	store, root := newObjectStoreForTest(t)
	bundle := c2Bundle(t)
	template := c2Target(t, bundle, c2Digest('8'), '9')
	input := template.Input()
	epoch, err := hostepoch.Measure(ctx)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := store.AllocateConformanceAttempt(ctx, ConformanceAttemptInput{
		ContractBundleDigest:        input.ContractBundleDigest,
		ResidueHeadDigest:           input.TerminalResidue.HeadDigest,
		TreeIdentityDigest:          input.Tree.TreeIdentityDigest,
		MaterializationPolicyDigest: input.Tree.MaterializationPolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Attempt = contractmodel.AttemptBinding{
		ArtifactDigest: attempt.Digest(), InstanceNonce: attempt.InstanceNonce(),
	}
	input.BootSession = contractmodel.BootSessionBinding{IdentityDigest: epoch.Digest()}
	target, err := contractmodel.NewContractExecutionTarget(input)
	if err != nil {
		t.Fatal(err)
	}
	targetRecord, err := store.PersistContractTargetRecord(ctx, attempt, target)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := AcquireContractRunOwner(ctx, targetRecord, epoch)
	if err != nil {
		t.Fatal(err)
	}
	return store, root, attempt, target, targetRecord, epoch, owner
}

func c4AllEvidenceBodies() map[contractmodel.EvidenceKind][]byte {
	bodies := make(map[contractmodel.EvidenceKind][]byte, len(privateEvidenceKindOrder))
	for _, kind := range privateEvidenceKindOrder {
		bodies[contractmodel.EvidenceKind(kind)] = []byte("private-evidence-" + kind)
	}
	return bodies
}

func c4AllEvidenceReferences(t *testing.T, manifest PrivateRunManifest) map[string]contractmodel.EvidenceRef {
	t.Helper()
	byKind := make(map[string]contractmodel.EvidenceRef, len(privateEvidenceKindOrder))
	for _, kind := range privateEvidenceKindOrder {
		reference, err := manifest.EvidenceRef(contractmodel.EvidenceKind(kind))
		if err != nil {
			t.Fatal(err)
		}
		byKind[kind] = reference
	}
	return byKind
}

func c4ClosedRun(
	t *testing.T,
	bundleDigestTarget contractmodel.ContractExecutionTarget,
	claim domain.Digest,
	spawn contractmodel.SpawnObservation,
	manifest PrivateRunManifest,
	references map[string]contractmodel.EvidenceRef,
) contractmodel.FinalizedContractRun {
	t.Helper()
	processKinds := []string{
		"MATERIALIZATION_REVALIDATION", "RUNTIME_REVALIDATION", "PROCESS_RESULT", "WAIT_RESULT",
		"DRAIN_RESULT", "TEARDOWN_RESULT", "ORPHAN_CHECK", "FINALIZATION_MARKER",
	}
	processReferences := make([]contractmodel.EvidenceRef, len(processKinds))
	for index, kind := range processKinds {
		processReferences[index] = references[kind]
	}
	process, err := contractmodel.NewCleanProcessClosure(processReferences)
	if err != nil {
		t.Fatal(err)
	}
	bundle := c2Bundle(t)
	allowed := bundle.Predicate().AllowedTuples()
	observation, err := contractmodel.NewProjectedObservation(
		references["CAPTURED_OBSERVATION"], references["PROJECTION_RESULT"], allowed[0],
	)
	if err != nil {
		t.Fatal(err)
	}
	scopeInputs := []struct {
		domain contractmodel.ScopeDomain
		kind   string
	}{
		{contractmodel.ScopeTargetInventory, "TARGET_INVENTORY"},
		{contractmodel.ScopeChildBindings, "CHILD_BINDINGS"},
		{contractmodel.ScopeImportResolution, "IMPORT_RESOLUTION"},
		{contractmodel.ScopeServiceBindings, "SERVICE_BINDINGS"},
		{contractmodel.ScopeSentinelInheritance, "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE"},
	}
	checks := make([]contractmodel.ScopeCheck, len(scopeInputs))
	for index, input := range scopeInputs {
		checks[index], err = contractmodel.NewCleanScopeCheck(input.domain, references[input.kind])
		if err != nil {
			t.Fatal(err)
		}
	}
	scope, err := contractmodel.NewStandaloneScope(checks)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := manifest.Summary()
	if err != nil {
		t.Fatal(err)
	}
	witness, err := contractmodel.NewClosedRunWitness(spawn, process, observation, scope, summary)
	if err != nil {
		t.Fatal(err)
	}
	run, err := contractmodel.NewFinalizedContractRun(bundleDigestTarget, claim, witness)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func c4CompleteBridgeFixture(t *testing.T) c4BridgeFixture {
	t.Helper()
	ctx := context.Background()
	store, root, attempt, target, targetRecord, epoch, owner := c4TargetAndOwner(t)
	copyOfOwner := owner
	binding := c2Digest('a')
	if err := owner.ConsumeForStart(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if err := copyOfOwner.ConsumeForStart(ctx, binding); err == nil {
		t.Fatal("copied owner multiplied one-shot start authority")
	}
	spawn, err := contractmodel.NewChildPIDObservation(4242)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		t.Fatal(err)
	}
	if owner.state.spawn.bindingDigest != binding {
		t.Fatal("durable spawn observation did not retain the consume-time invocation binding")
	}
	manifest, err := owner.PersistPrivateRunManifest(ctx, c4AllEvidenceBodies())
	if err != nil {
		t.Fatal(err)
	}
	references := c4AllEvidenceReferences(t, manifest)
	run := c4ClosedRun(t, target, owner.StartClaimDigest(), spawn, manifest, references)
	closure, err := owner.PersistFinalizedRun(ctx, manifest, run)
	if err != nil {
		t.Fatal(err)
	}
	return c4BridgeFixture{
		store: store, root: root, attempt: attempt, target: target, targetRecord: targetRecord,
		epoch: epoch, owner: owner, spawn: spawn, binding: binding, manifest: manifest,
		run: run, closure: closure,
	}
}

func TestC4ContractRunBridgePersistsClosesClassifiesAndReopens(t *testing.T) {
	ctx := context.Background()
	fixture := c4CompleteBridgeFixture(t)
	bundle := c2Bundle(t)
	execution, err := contractmodel.DeriveContractExecution(bundle, fixture.target, fixture.run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PersistContractExecutionRecord(ctx, fixture.closure.FinalizedRun(), execution); err == nil {
		t.Fatal("classification persisted before exact finalized-run release")
	}
	if err := fixture.closure.Release(ctx); err != nil {
		t.Fatal(err)
	}
	record, err := PersistContractExecutionRecord(ctx, fixture.closure.FinalizedRun(), execution)
	if err != nil || !record.Valid() || record.Digest() != execution.Digest() || !record.Model().Equal(execution) {
		t.Fatalf("execution persistence failed: %v", err)
	}

	restarted, err := OpenObjectStore(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := restarted.OpenConformanceAttempt(ctx, fixture.attempt.Digest())
	if err != nil {
		t.Fatal(err)
	}
	targetRecord, err := restarted.OpenContractTargetRecord(ctx, attempt)
	if err != nil {
		t.Fatal(err)
	}
	reopenedRun, err := OpenFinalizedRunRecord(ctx, targetRecord, fixture.target)
	if err != nil || !reopenedRun.Valid() || !reopenedRun.Model().Equal(fixture.run) {
		t.Fatalf("classification-only run reopen failed: %v", err)
	}
	rederived, err := contractmodel.DeriveContractExecution(bundle, fixture.target, reopenedRun.Model())
	if err != nil {
		t.Fatal(err)
	}
	reopenedExecution, err := PersistContractExecutionRecord(ctx, reopenedRun, rederived)
	if err != nil || !reopenedExecution.Model().Equal(execution) {
		t.Fatalf("classification-only exact convergence failed: %v", err)
	}
	reopenedClosure, err := OpenTerminalClosure(ctx, targetRecord, fixture.target)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopenedClosure.Release(ctx); err != nil {
		t.Fatalf("CLEAR receipt recovery/idempotence failed: %v", err)
	}

	tampered := c4CompleteBridgeFixture(t)
	tamperedExecution, err := contractmodel.DeriveContractExecution(bundle, tampered.target, tampered.run)
	if err != nil {
		t.Fatal(err)
	}
	if err := tampered.closure.Release(ctx); err != nil {
		t.Fatal(err)
	}
	runHex, err := strictDigestHex(tampered.run.Digest())
	if err != nil {
		t.Fatal(err)
	}
	releasePath := filepath.Join(tampered.store.contractOps, finalizedReleaseDirectory, runHex)
	if err := os.Remove(releasePath); err != nil {
		t.Fatal(err)
	}
	if _, err := PersistContractExecutionRecord(ctx, tampered.closure.FinalizedRun(), tamperedExecution); err == nil {
		t.Fatal("in-memory release bit authorized classification after durable release evidence disappeared")
	}
	if _, err := openExecutionByRunProfile(ctx, tampered.store, tampered.closure.FinalizedRun().record); err == nil {
		t.Fatal("failed durable release revalidation still published an execution relationship")
	}
}

func TestC4ContractRunBridgeRefusesSkippedAndMismatchedEdges(t *testing.T) {
	ctx := context.Background()
	_, _, _, target, _, _, owner := c4TargetAndOwner(t)
	spawn, _ := contractmodel.NewChildPIDObservation(4242)
	if err := owner.PersistSpawnObservation(ctx, spawn); err == nil {
		t.Fatal("unconsumed owner persisted a spawn observation")
	}
	if _, err := owner.PersistPrivateRunManifest(ctx, map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidenceProcessResult: []byte("body"),
	}); err == nil {
		t.Fatal("owner persisted private evidence before spawn")
	}
	if err := owner.ConsumeForStart(ctx, c2Digest('a')); err != nil {
		t.Fatal(err)
	}
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		t.Fatal(err)
	}
	alternate, _ := contractmodel.NewChildPIDObservation(4243)
	if err := owner.PersistSpawnObservation(ctx, alternate); err == nil {
		t.Fatal("owner accepted a second spawn identity")
	}
	if _, err := OpenFinalizedRunRecord(ctx, ContractTargetRecord{}, target); err == nil {
		t.Fatal("zero target record reopened a finalized run")
	}
}

func TestC4SpawnObservationPersistsEveryClosedStartErrorExactly(t *testing.T) {
	ctx := context.Background()
	for _, code := range []string{
		"OS_START_ERROR",
		"PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
		"PRESPAWN_RUNTIME_REVALIDATION_FAILED",
	} {
		t.Run(code, func(t *testing.T) {
			_, _, _, _, _, _, owner := c4TargetAndOwner(t)
			binding := c2Digest('a')
			if err := owner.ConsumeForStart(ctx, binding); err != nil {
				t.Fatal(err)
			}
			observation, err := contractmodel.NewStartErrorObservation(code)
			if err != nil {
				t.Fatal(err)
			}
			if err := owner.PersistSpawnObservation(ctx, observation); err != nil {
				t.Fatal(err)
			}
			reopened, err := openSpawnObservation(ctx, owner.state.store, owner.state.target, owner.state.claim)
			if err != nil || reopened.bindingDigest != binding || !sameSpawnObservation(reopened.observation, observation) {
				t.Fatalf("start-error observation did not reopen exactly: %v", err)
			}
		})
	}
}

func TestC4PrivateManifestDerivesRefsGroupsBodiesAndCannotFork(t *testing.T) {
	ctx := context.Background()
	_, _, _, _, _, _, owner := c4TargetAndOwner(t)
	if err := owner.ConsumeForStart(ctx, c2Digest('a')); err != nil {
		t.Fatal(err)
	}
	spawn, _ := contractmodel.NewChildPIDObservation(4242)
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.PersistPrivateRunManifest(ctx, nil); err == nil {
		t.Fatal("empty private evidence roster was admitted")
	}
	if _, err := owner.PersistPrivateRunManifest(ctx, map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidencePrivateManifest: []byte("recursive"),
	}); err == nil {
		t.Fatal("recursive private-manifest evidence was admitted")
	}
	common := []byte("one-body-for-the-complete-logical-roster")
	evidence := make(map[contractmodel.EvidenceKind][]byte, len(privateEvidenceKindOrder))
	for _, kind := range privateEvidenceKindOrder {
		evidence[contractmodel.EvidenceKind(kind)] = common
	}
	manifest, err := owner.PersistPrivateRunManifest(ctx, evidence)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := manifest.Summary()
	if err != nil || summary.BlobCount() != 1 || summary.AggregateByteCount() != int64(len(common)) {
		t.Fatalf("identical private bodies were not grouped exactly: %#v %v", summary, err)
	}
	wantDigest := privateBytesDigest(common)
	for _, kind := range privateEvidenceKindOrder {
		reference, err := manifest.EvidenceRef(contractmodel.EvidenceKind(kind))
		if err != nil || reference.Digest() != wantDigest {
			t.Fatalf("%s reference was not derived from retained bytes: %v", kind, err)
		}
	}
	if retried, err := owner.PersistPrivateRunManifest(ctx, evidence); err != nil || retried.record.digest != manifest.record.digest {
		t.Fatalf("exact manifest retry did not reopen the same record: %v", err)
	}
	directory := filepath.Dir(manifest.record.manifestPath)
	before, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	alternate := c4AllEvidenceBodies()
	alternate[contractmodel.EvidenceProcessResult] = []byte("different-process-result")
	if _, err := owner.PersistPrivateRunManifest(ctx, alternate); err == nil {
		t.Fatal("owner forked a second private manifest after phase advance")
	}
	after, err := os.ReadDir(directory)
	if err != nil || len(after) != len(before) {
		t.Fatalf("refused manifest fork left an orphan: before=%d after=%d err=%v", len(before), len(after), err)
	}
}

func TestC4FinalizedRunRefusesMissingSpawnObservationBeforePublication(t *testing.T) {
	ctx := context.Background()
	_, _, _, target, _, _, owner := c4TargetAndOwner(t)
	binding := c2Digest('a')
	if err := owner.ConsumeForStart(ctx, binding); err != nil {
		t.Fatal(err)
	}
	spawn, _ := contractmodel.NewChildPIDObservation(4242)
	if err := owner.PersistSpawnObservation(ctx, spawn); err != nil {
		t.Fatal(err)
	}
	manifest, err := owner.PersistPrivateRunManifest(ctx, c4AllEvidenceBodies())
	if err != nil {
		t.Fatal(err)
	}
	run := c4ClosedRun(t, target, owner.StartClaimDigest(), spawn, manifest, c4AllEvidenceReferences(t, manifest))
	spawnPath := owner.state.spawn.path
	if err := os.Remove(spawnPath); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(spawnPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.PersistFinalizedRun(ctx, manifest, run); err == nil {
		t.Fatal("finalized run published without reopening durable spawn evidence")
	}
	if _, err := openFinalizedRunByTarget(ctx, owner.state.store, owner.state.target); err == nil {
		t.Fatal("refused finalized run left a public FCR relationship")
	}
}

func TestC4TerminalClosureRequiresDurableSpawnObservationButClassificationDoesNot(t *testing.T) {
	ctx := context.Background()
	fixture := c4CompleteBridgeFixture(t)
	if err := fixture.closure.Release(ctx); err != nil {
		t.Fatal(err)
	}
	spawnPath := fixture.closure.state.spawn.path
	if err := os.Remove(spawnPath); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(spawnPath)); err != nil {
		t.Fatal(err)
	}
	if err := fixture.closure.Release(ctx); err == nil {
		t.Fatal("live terminal closure released after durable spawn evidence disappeared")
	}
	restarted, err := OpenObjectStore(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := restarted.OpenConformanceAttempt(ctx, fixture.attempt.Digest())
	if err != nil {
		t.Fatal(err)
	}
	targetRecord, err := restarted.OpenContractTargetRecord(ctx, attempt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFinalizedRunRecord(ctx, targetRecord, fixture.target); err != nil {
		t.Fatalf("classification-only reopen incorrectly required private spawn evidence: %v", err)
	}
	if _, err := OpenTerminalClosure(ctx, targetRecord, fixture.target); err == nil {
		t.Fatal("terminal closure reopened without the durable spawn observation")
	}
}

func TestC4ClassificationRecoveryUsesHistoricalRunReleaseLink(t *testing.T) {
	ctx := context.Background()
	fixture := c4CompleteBridgeFixture(t)
	if err := fixture.closure.Release(ctx); err != nil {
		t.Fatal(err)
	}
	input := fixture.target.Input()
	secondAttempt, err := fixture.store.AllocateConformanceAttempt(ctx, ConformanceAttemptInput{
		ContractBundleDigest:        input.ContractBundleDigest,
		ResidueHeadDigest:           input.TerminalResidue.HeadDigest,
		TreeIdentityDigest:          input.Tree.TreeIdentityDigest,
		MaterializationPolicyDigest: input.Tree.MaterializationPolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Attempt = contractmodel.AttemptBinding{
		ArtifactDigest: secondAttempt.Digest(), InstanceNonce: secondAttempt.InstanceNonce(),
	}
	secondTarget, err := contractmodel.NewContractExecutionTarget(input)
	if err != nil {
		t.Fatal(err)
	}
	secondRecord, err := fixture.store.PersistContractTargetRecord(ctx, secondAttempt, secondTarget)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireContractRunOwner(ctx, secondRecord, fixture.epoch); err != nil {
		t.Fatal(err)
	}
	if reopened, err := OpenFinalizedRunRecord(ctx, fixture.targetRecord, fixture.target); err != nil || !reopened.Model().Equal(fixture.run) {
		t.Fatalf("later interlock generation hid an exact historical release: %v", err)
	}
}

func TestC4ClassificationRecoveryDoesNotRequireRetainedPrivatePack(t *testing.T) {
	ctx := context.Background()
	fixture := c4CompleteBridgeFixture(t)
	if err := fixture.closure.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(fixture.manifest.record.packPath); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(fixture.manifest.record.packPath)); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenObjectStore(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := restarted.OpenConformanceAttempt(ctx, fixture.attempt.Digest())
	if err != nil {
		t.Fatal(err)
	}
	targetRecord, err := restarted.OpenContractTargetRecord(ctx, attempt)
	if err != nil {
		t.Fatal(err)
	}
	run, err := OpenFinalizedRunRecord(ctx, targetRecord, fixture.target)
	if err != nil {
		t.Fatalf("classification-only recovery required retained private evidence: %v", err)
	}
	execution, err := contractmodel.DeriveContractExecution(c2Bundle(t), fixture.target, run.Model())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PersistContractExecutionRecord(ctx, run, execution); err != nil {
		t.Fatalf("classification publication required retained private evidence: %v", err)
	}
}
