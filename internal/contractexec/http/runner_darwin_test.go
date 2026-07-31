//go:build darwin && arm64 && cgo

package http

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodemodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/store"
	httpfixture "github.com/nelsonwerd/countershape/testkit/contractexec/http"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

func TestMain(m *testing.M) { os.Exit(httpfixture.RunMain(m)) }

func TestHTTPStartErrorClosesDurableRunAndClassificationWithoutChild(t *testing.T) {
	t.Run("seed-materialization-lease", testHTTPSeedMaterializationLease)
	t.Run("seed-residue-refusal", testHTTPSeedResidueRefusal)
	t.Run("profile-gate", testHTTPRecoveryProfileGate)
	t.Run("phase-context-independence", testHTTPPhaseContextIndependence)
	t.Run("clean", func(t *testing.T) {
		testHTTPStartErrorClosure(t, false)
	})
	t.Run("descriptor-close-failure", func(t *testing.T) {
		testHTTPStartErrorClosure(t, true)
	})
	t.Run("parent-writer-close-failure", testHTTPParentWriterCloseFailure)
}

func testHTTPPhaseContextIndependence(t *testing.T) {
	t.Helper()
	fixture := httpfixture.NewReferenceTarget(t)
	preparation, err := prepareHTTPExecution(fixture.Target)
	if err != nil {
		t.Fatalf("prepare phase-context fixture: %v", err)
	}
	input := executionInput{target: fixture.Target, executionPreparation: preparation}
	defer func() {
		if closeErr := input.closePrepared(); closeErr != nil {
			t.Errorf("close phase-context fixture: %v", closeErr)
		}
	}()

	parent, cancelParent := context.WithCancel(context.Background())
	revalidationContext, cancelRevalidation := detachedHTTPPhaseContext(parent, input)
	closureContext, cancelClosure := detachedHTTPPhaseContext(parent, input)
	if _, ok := revalidationContext.Deadline(); !ok {
		t.Fatal("revalidation phase has no deadline")
	}
	if _, ok := closureContext.Deadline(); !ok {
		t.Fatal("closure phase has no deadline")
	}
	cancelRevalidation()
	if !errors.Is(revalidationContext.Err(), context.Canceled) {
		t.Fatalf("revalidation phase did not cancel exactly: %v", revalidationContext.Err())
	}
	if closureContext.Err() != nil {
		t.Fatalf("revalidation cancellation crossed into closure phase: %v", closureContext.Err())
	}
	cancelParent()
	if closureContext.Err() != nil {
		t.Fatalf("caller cancellation crossed detached closure phase: %v", closureContext.Err())
	}
	cancelClosure()
	if !errors.Is(closureContext.Err(), context.Canceled) {
		t.Fatalf("closure phase did not cancel exactly: %v", closureContext.Err())
	}
}

func testHTTPSeedMaterializationLease(t *testing.T) {
	t.Helper()
	fixture := httpfixture.NewReferenceTarget(t)
	roots := fixture.Target.Roots()
	fixtureEntries, err := os.ReadDir(roots.FixtureRoot())
	if err != nil || len(fixtureEntries) != 0 {
		t.Fatalf("seed fixture was not initially empty: entries=%v err=%v", fixtureEntries, err)
	}
	temporaryInfo, err := os.Lstat(roots.TemporaryRoot())
	if err != nil {
		t.Fatal(err)
	}
	lease, err := acquireSeedMaterializationLease(roots.TemporaryRoot(), temporaryInfo)
	if err != nil {
		t.Fatalf("acquire seed materialization lease: %v", err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			_ = lease.release()
		}
	})
	preparation, prepareErr := prepareHTTPExecution(fixture.Target)
	if prepareErr == nil || !IsCode(prepareErr, CodeAdmissionRefused) ||
		preparation.prepared != nil || preparation.probe != nil ||
		preparation.binding.Valid() {
		t.Fatalf(
			"live seed-materialization contention crossed preparation: prepared=%t probe=%t binding=%s err=%v",
			preparation.prepared != nil, preparation.probe != nil, preparation.binding, prepareErr,
		)
	}
	afterEntries, fixtureErr := os.ReadDir(roots.FixtureRoot())
	_, stagingErr := os.Lstat(filepath.Join(roots.TemporaryRoot(), seedStagingDirectory))
	if fixtureErr != nil || len(afterEntries) != 0 ||
		!errors.Is(stagingErr, os.ErrNotExist) {
		t.Fatalf(
			"live seed-materialization refusal mutated roots: fixture=%v staging=%v err=%v",
			afterEntries,
			stagingErr,
			fixtureErr,
		)
	}
	assertInternalMarkerOnly(t, fixture)
	if err := lease.release(); err != nil {
		t.Fatalf("release seed materialization lease: %v", err)
	}
	released = true
	heldPreparation, prepareErr := prepareHTTPExecution(fixture.Target)
	if prepareErr != nil || heldPreparation.prepared == nil ||
		heldPreparation.probe == nil || heldPreparation.seedLease.root == nil ||
		!heldPreparation.binding.Valid() {
		t.Fatalf(
			"released seed-materialization lease did not restore preparation: prepared=%t probe=%t lease=%t binding=%s err=%v",
			heldPreparation.prepared != nil,
			heldPreparation.probe != nil,
			heldPreparation.seedLease.root != nil,
			heldPreparation.binding,
			prepareErr,
		)
	}
	heldInput := executionInput{target: fixture.Target, executionPreparation: heldPreparation}
	heldClosed := false
	t.Cleanup(func() {
		if !heldClosed {
			_ = heldInput.closePrepared()
		}
	})
	activeReceiptPath := filepath.Join(roots.EvidenceRoot(), invocationFilename)
	activeReceiptBody := []byte("{\"active\":\"cooperating-holder\"}\n")
	if err := os.WriteFile(activeReceiptPath, activeReceiptBody, 0o600); err != nil {
		t.Fatal(err)
	}
	activeReceiptInfo, err := os.Lstat(activeReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	contendedPreparation, contendedErr := prepareHTTPExecution(fixture.Target)
	if contendedErr == nil || !IsCode(contendedErr, CodeAdmissionRefused) ||
		contendedPreparation.prepared != nil || contendedPreparation.probe != nil ||
		contendedPreparation.seedLease.root != nil || contendedPreparation.binding.Valid() {
		t.Fatalf(
			"held execution-attempt lease did not dominate live receipt interpretation: prepared=%t probe=%t lease=%t binding=%s err=%v",
			contendedPreparation.prepared != nil,
			contendedPreparation.probe != nil,
			contendedPreparation.seedLease.root != nil,
			contendedPreparation.binding,
			contendedErr,
		)
	}
	afterReceiptInfo, statErr := os.Lstat(activeReceiptPath)
	afterReceiptBody, readErr := os.ReadFile(activeReceiptPath)
	if statErr != nil || readErr != nil ||
		!os.SameFile(activeReceiptInfo, afterReceiptInfo) ||
		string(afterReceiptBody) != string(activeReceiptBody) {
		t.Fatalf(
			"live-holder admission refusal changed active receipt: info=%v body=%q err=%v",
			afterReceiptInfo,
			afterReceiptBody,
			errors.Join(statErr, readErr),
		)
	}
	if err := heldInput.closePrepared(); err != nil {
		t.Fatalf("close held execution preparation: %v", err)
	}
	heldClosed = true
	residuePreparation, residueErr := prepareHTTPExecution(fixture.Target)
	if residueErr == nil || !IsCode(residueErr, CodeFixtureRejected) ||
		residuePreparation.prepared != nil || residuePreparation.probe != nil ||
		residuePreparation.seedLease.root != nil || residuePreparation.binding.Valid() {
		t.Fatalf(
			"unowned invocation residue did not retain fixture classification: prepared=%t probe=%t lease=%t binding=%s err=%v",
			residuePreparation.prepared != nil,
			residuePreparation.probe != nil,
			residuePreparation.seedLease.root != nil,
			residuePreparation.binding,
			residueErr,
		)
	}
	if err := os.Remove(activeReceiptPath); err != nil {
		t.Fatal(err)
	}
	assertInternalMarkerOnly(t, fixture)
	recoveredPreparation, prepareErr := prepareHTTPExecution(fixture.Target)
	if prepareErr != nil || recoveredPreparation.prepared == nil ||
		recoveredPreparation.probe == nil || recoveredPreparation.seedLease.root == nil ||
		!recoveredPreparation.binding.Valid() {
		t.Fatalf(
			"retired residue did not restore execution preparation: prepared=%t probe=%t lease=%t binding=%s err=%v",
			recoveredPreparation.prepared != nil,
			recoveredPreparation.probe != nil,
			recoveredPreparation.seedLease.root != nil,
			recoveredPreparation.binding,
			prepareErr,
		)
	}
	recoveredInput := executionInput{
		target: fixture.Target, executionPreparation: recoveredPreparation,
	}
	if err := recoveredInput.closePrepared(); err != nil {
		t.Fatalf("close recovered execution preparation: %v", err)
	}
}

func testHTTPSeedResidueRefusal(t *testing.T) {
	t.Helper()
	emptyAttempt := t.TempDir()
	emptyFixture := filepath.Join(emptyAttempt, "fixture")
	emptyTemporary := filepath.Join(emptyAttempt, "tmp")
	if err := os.Mkdir(emptyFixture, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(emptyTemporary, 0o700); err != nil {
		t.Fatal(err)
	}
	emptyStaging := filepath.Join(emptyTemporary, seedStagingDirectory)
	if err := os.Mkdir(emptyStaging, 0o700); err != nil {
		t.Fatal(err)
	}
	emptyLeaf := filepath.Join(emptyStaging, ".seed.pending-empty-crash")
	emptyBody := []byte("preserve empty-roster staging residue")
	if err := os.WriteFile(emptyLeaf, emptyBody, 0o600); err != nil {
		t.Fatal(err)
	}
	emptyStagingInfo, err := os.Lstat(emptyStaging)
	if err != nil {
		t.Fatal(err)
	}
	emptyLeafInfo, err := os.Lstat(emptyLeaf)
	if err != nil {
		t.Fatal(err)
	}
	if _, emptyErr := materializeHTTPSeeds(
		emptyFixture,
		emptyTemporary,
		nil,
	); emptyErr == nil {
		t.Fatal("empty seed roster ignored crash-left staging residue")
	}
	afterEmptyStaging, emptyStagingErr := os.Lstat(emptyStaging)
	afterEmptyLeaf, emptyLeafErr := os.Lstat(emptyLeaf)
	afterEmptyBody, emptyReadErr := os.ReadFile(emptyLeaf)
	emptyEntries, emptyRosterErr := os.ReadDir(emptyFixture)
	if emptyStagingErr != nil || emptyLeafErr != nil || emptyReadErr != nil ||
		emptyRosterErr != nil || len(emptyEntries) != 0 ||
		!os.SameFile(emptyStagingInfo, afterEmptyStaging) ||
		!os.SameFile(emptyLeafInfo, afterEmptyLeaf) ||
		string(afterEmptyBody) != string(emptyBody) {
		t.Fatalf(
			"empty-roster crash residue was changed: staging=%v leaf=%v body=%q entries=%v err=%v",
			afterEmptyStaging,
			afterEmptyLeaf,
			afterEmptyBody,
			emptyEntries,
			errors.Join(emptyStagingErr, emptyLeafErr, emptyReadErr, emptyRosterErr),
		)
	}
	if err := os.Remove(emptyLeaf); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(emptyStaging); err != nil {
		t.Fatal(err)
	}
	emptyLease, err := materializeHTTPSeeds(emptyFixture, emptyTemporary, nil)
	if err != nil {
		t.Fatalf("empty seed roster did not retire clean staging: %v", err)
	}
	if err := emptyLease.release(); err != nil {
		t.Fatalf("release empty seed materialization lease: %v", err)
	}
	if _, statErr := os.Lstat(emptyStaging); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("empty seed roster retained staging: %v", statErr)
	}
	emptyEntries, err = os.ReadDir(emptyFixture)
	if err != nil || len(emptyEntries) != 0 {
		t.Fatalf("empty seed roster changed fixture: entries=%v err=%v", emptyEntries, err)
	}

	fixture := httpfixture.NewReferenceTarget(t)
	roots := fixture.Target.Roots()

	stagingRoot := filepath.Join(roots.TemporaryRoot(), seedStagingDirectory)
	if err := os.Mkdir(stagingRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	stagingLeaf := filepath.Join(stagingRoot, ".seed.pending-crash")
	stagingBody := []byte("preserve staging crash residue")
	if err := os.WriteFile(stagingLeaf, stagingBody, 0o600); err != nil {
		t.Fatal(err)
	}
	stagingInfo, err := os.Lstat(stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	stagingLeafInfo, err := os.Lstat(stagingLeaf)
	if err != nil {
		t.Fatal(err)
	}
	preparation, prepareErr := prepareHTTPExecution(fixture.Target)
	if prepareErr == nil || !IsCode(prepareErr, CodeFixtureRejected) ||
		preparation.prepared != nil || preparation.probe != nil ||
		preparation.binding.Valid() {
		t.Fatalf(
			"crash-left staging residue crossed preparation: prepared=%t probe=%t binding=%s err=%v",
			preparation.prepared != nil, preparation.probe != nil, preparation.binding, prepareErr,
		)
	}
	afterStaging, stagingErr := os.Lstat(stagingRoot)
	afterLeaf, leafErr := os.Lstat(stagingLeaf)
	leafBody, readErr := os.ReadFile(stagingLeaf)
	if stagingErr != nil || leafErr != nil || readErr != nil ||
		!os.SameFile(stagingInfo, afterStaging) ||
		!os.SameFile(stagingLeafInfo, afterLeaf) ||
		string(leafBody) != string(stagingBody) {
		t.Fatalf(
			"crash-left staging residue was changed: staging=%v leaf=%v body=%q err=%v",
			afterStaging, afterLeaf, leafBody,
			errors.Join(stagingErr, leafErr, readErr),
		)
	}
	assertInternalMarkerOnly(t, fixture)
	if err := os.Remove(stagingLeaf); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(stagingRoot); err != nil {
		t.Fatal(err)
	}

	source := fixture.Target.ContractBundle().PortableSource()
	view, ok := source.HTTPView()
	if !ok || !view.Valid() || len(view.Stimulus().Seeds()) == 0 {
		t.Fatal("reference HTTP seed roster is absent")
	}
	firstSeed := view.Stimulus().Seeds()[0]
	legacyPath := filepath.Join(
		roots.FixtureRoot(),
		"."+filepath.Base(firstSeed.Path())+".pending-crash",
	)
	legacyBody := []byte("preserve legacy fixture residue")
	if err := os.WriteFile(legacyPath, legacyBody, 0o600); err != nil {
		t.Fatal(err)
	}
	legacyInfo, err := os.Lstat(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	preparation, prepareErr = prepareHTTPExecution(fixture.Target)
	if prepareErr == nil || !IsCode(prepareErr, CodeFixtureRejected) ||
		preparation.prepared != nil || preparation.probe != nil ||
		preparation.binding.Valid() {
		t.Fatalf(
			"legacy fixture residue crossed preparation: prepared=%t probe=%t binding=%s err=%v",
			preparation.prepared != nil, preparation.probe != nil, preparation.binding, prepareErr,
		)
	}
	afterLegacy, legacyErr := os.Lstat(legacyPath)
	legacyRead, legacyReadErr := os.ReadFile(legacyPath)
	if legacyErr != nil || legacyReadErr != nil ||
		!os.SameFile(legacyInfo, afterLegacy) ||
		string(legacyRead) != string(legacyBody) {
		t.Fatalf(
			"legacy fixture residue was changed: info=%v body=%q err=%v",
			afterLegacy, legacyRead, errors.Join(legacyErr, legacyReadErr),
		)
	}
	for _, seed := range view.Stimulus().Seeds() {
		path := filepath.Join(roots.FixtureRoot(), filepath.FromSlash(seed.Path()))
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("declared seed was written before foreign-roster refusal: %s: %v", path, statErr)
		}
	}
	assertInternalMarkerOnly(t, fixture)
	if err := os.Remove(legacyPath); err != nil {
		t.Fatal(err)
	}
}

func testHTTPRecoveryProfileGate(t *testing.T) {
	t.Helper()
	cliSource, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	cliProfile, err := nodemodel.NewSourceProfile(cliSource)
	if err != nil {
		t.Fatal(err)
	}
	if view, profileErr := requireStandaloneHTTPProfile(cliSource, cliProfile); profileErr == nil || !IsCode(profileErr, CodeUnsupportedProfile) || view.Valid() {
		t.Fatalf(
			"CLI source crossed HTTP recovery profile gate: view=%t err=%v",
			view.Valid(), profileErr,
		)
	}

	httpSource, err := contractfixtures.HTTPSource()
	if err != nil {
		t.Fatal(err)
	}
	httpProfile, err := nodemodel.NewSourceProfile(httpSource)
	if err != nil {
		t.Fatal(err)
	}
	if view, profileErr := requireStandaloneHTTPProfile(httpSource, httpProfile); profileErr != nil || !view.Valid() {
		t.Fatalf(
			"portable HTTP source failed recovery profile gate: view=%t err=%v",
			view.Valid(), profileErr,
		)
	}
}

func testHTTPParentWriterCloseFailure(t *testing.T) {
	t.Helper()
	fixture := httpfixture.NewReferenceTarget(t)
	preparation, err := prepareHTTPExecution(fixture.Target)
	if err != nil {
		t.Fatal(err)
	}
	input := executionInput{target: fixture.Target, executionPreparation: preparation}
	t.Cleanup(func() { _ = input.closePrepared() })
	closeErr := errors.New("injected parent writer close failure")
	closeFn := input.prepared.readinessWriter.closeFn
	input.prepared.readinessWriter.closeFn = func(file *os.File) error {
		return errors.Join(closeFn(file), closeErr)
	}
	initial, running, startErr := input.prepared.Start(
		context.Background(),
		context.Background(),
	)
	if startErr != nil || running == nil || !initial.process.started ||
		!initial.process.teardownError || initial.process.orphanRisk ||
		initial.process.primary != "" ||
		strings.Join(initial.process.descriptorCloseDiagnostics(), ",") !=
			"HTTP_PARENT_WRITER_CLOSE_FAILED" {
		t.Fatalf(
			"successful start erased writer-close failure: initial=%+v running=%t err=%v",
			initial,
			running != nil,
			startErr,
		)
	}
	terminalCloseErr := errors.New("injected terminal reader close failure")
	terminalCloseCalls := make([]int, 3)
	for index, owned := range []*ownedServiceDescriptor{
		running.readiness,
		running.stdout,
		running.stderr,
	} {
		index := index
		closeFn := owned.closeFn
		owned.closeFn = func(file *os.File) error {
			terminalCloseCalls[index]++
			return errors.Join(closeFn(file), terminalCloseErr)
		}
	}
	result := running.Close()
	if !result.process.teardownError || result.process.orphanRisk ||
		strings.Join(result.process.descriptorCloseDiagnostics(), ",") !=
			"HTTP_PARENT_WRITER_CLOSE_FAILED,HTTP_TERMINAL_READER_CLOSE_FAILED" ||
		terminalCloseCalls[0] != 1 || terminalCloseCalls[1] != 1 ||
		terminalCloseCalls[2] != 1 {
		t.Fatalf(
			"terminal closure lost writer/reader close failures: process=%+v calls=%v",
			result.process,
			terminalCloseCalls,
		)
	}
}

func testHTTPStartErrorClosure(t *testing.T, injectDescriptorCloseFailure bool) {
	t.Helper()
	fixture := httpfixture.NewReferenceTarget(t)
	preparation, err := prepareHTTPExecution(fixture.Target)
	if err != nil {
		t.Fatalf("prepare HTTP start-error execution: %v (cause: %v)", err, errors.Unwrap(err))
	}
	var injectedCloseErr error
	if injectDescriptorCloseFailure {
		injectedCloseErr = errors.New("injected readiness close failure")
		closeFn := preparation.prepared.readinessReader.closeFn
		preparation.prepared.readinessReader.closeFn = func(file *os.File) error {
			return errors.Join(closeFn(file), injectedCloseErr)
		}
	}
	input := executionInput{target: fixture.Target, executionPreparation: preparation}
	t.Cleanup(func() { _ = input.closePrepared() })

	ctx := context.Background()
	epoch, err := hostepoch.Measure(ctx)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.AcquireContractRunOwner(ctx, fixture.Target.TargetRecord(), epoch)
	if err != nil {
		t.Fatal(err)
	}

	candidateRoot := input.target.CandidateRoot()
	displacedRoot := candidateRoot + ".start-error-test"
	if err := os.Rename(candidateRoot, displacedRoot); err != nil {
		t.Fatal(err)
	}
	restored := false
	t.Cleanup(func() {
		if !restored {
			_ = os.Rename(displacedRoot, candidateRoot)
		}
	})
	initial, running, startErr := consumeAndStartHTTP(
		ctx, ctx, owner, input.prepared, input.binding,
	)
	if running != nil {
		t.Cleanup(func() { _ = running.Close() })
	}
	restoreErr := os.Rename(displacedRoot, candidateRoot)
	restored = restoreErr == nil
	if restoreErr != nil {
		t.Fatalf("restore exact candidate root after start refusal: %v", restoreErr)
	}
	if startErr == nil || running != nil || initial.binding != input.binding ||
		!initial.process.physicalExecutionEntered || !initial.process.spawnAttempted ||
		initial.process.started || initial.process.pid != 0 ||
		initial.process.processGroupID != 0 || initial.process.processGroupOwned ||
		initial.process.primary != domain.ControlStartError ||
		initial.process.diagnosticCode != "HTTP_SPAWN_FAILED" ||
		initial.process.waitError == "" {
		t.Fatalf("missing cwd did not retain the exact HTTP start refusal: result=%+v running=%v err=%v",
			initial, running, startErr)
	}
	if injectDescriptorCloseFailure &&
		(!errors.Is(startErr, injectedCloseErr) ||
			strings.Contains(initial.process.waitError, injectedCloseErr.Error())) {
		t.Fatalf(
			"start failure did not separate spawn and descriptor causes: wait=%q err=%v",
			initial.process.waitError,
			startErr,
		)
	}
	wantCloseDiagnostics := ""
	if injectDescriptorCloseFailure {
		wantCloseDiagnostics = "HTTP_START_ERROR_DESCRIPTOR_CLOSE_FAILED"
	}
	if initial.process.teardownError != injectDescriptorCloseFailure ||
		initial.process.orphanRisk ||
		strings.Join(initial.process.descriptorCloseDiagnostics(), ",") != wantCloseDiagnostics {
		t.Fatalf(
			"start-error descriptor closure teardown=%t orphan=%t diagnostics=%v want=%q",
			initial.process.teardownError, initial.process.orphanRisk,
			initial.process.descriptorCloseDiagnostics(), wantCloseDiagnostics,
		)
	}

	closureContext, cancelClosure := detachedHTTPPhaseContext(ctx, input)
	defer cancelClosure()
	record, err := closeHTTPStartError(
		closureContext, owner, input, initial, startErr,
	)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("HTTP start refusal did not close a durable classification: valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	run, err := store.OpenFinalizedRunRecord(
		ctx, fixture.Target.TargetRecord(), fixture.Target.Model(),
	)
	if err != nil || !run.Valid() {
		t.Fatalf("HTTP start-refusal finalized run did not reopen: valid=%t err=%v", run.Valid(), err)
	}
	witness := run.Model().Witness()
	errorCode, hasErrorCode := witness.SpawnObservation().ErrorCode()
	primary, hasPrimary := witness.ProcessClosure().Primary()
	if witness.SpawnObservation().State() != contractmodel.SpawnStartError || !hasErrorCode ||
		errorCode != "OS_START_ERROR" || witness.ProcessClosure().State() != contractmodel.ProcessControlled ||
		!hasPrimary || primary != domain.ControlStartError ||
		witness.ProcessClosure().TeardownError() != injectDescriptorCloseFailure ||
		witness.ProcessClosure().OrphanRisk() || witness.Observation().State() != contractmodel.CaptureNone ||
		witness.StandaloneScope().State() != contractmodel.ScopePartial ||
		witness.Disposition() != contractmodel.DispositionIneligibleControlStandalone {
		t.Fatalf("HTTP start-refusal axes spawn=%s code=%q process=%s primary=%s capture=%s scope=%s disposition=%s",
			witness.SpawnObservation().State(), errorCode, witness.ProcessClosure().State(), primary,
			witness.Observation().State(), witness.StandaloneScope().State(), witness.Disposition())
	}
	wantProcessKinds := []contractmodel.EvidenceKind{
		contractmodel.EvidenceMaterializationRevalidation,
		contractmodel.EvidenceRuntimeRevalidation,
		contractmodel.EvidenceTeardownResult,
		contractmodel.EvidenceOrphanCheck,
		contractmodel.EvidenceFinalizationMarker,
	}
	processEvidence := witness.ProcessClosure().Evidence()
	if len(processEvidence) != len(wantProcessKinds) {
		t.Fatalf("HTTP start-refusal process evidence count=%d want=%d",
			len(processEvidence), len(wantProcessKinds))
	}
	for index, reference := range processEvidence {
		if !reference.Valid() || reference.Kind() != wantProcessKinds[index] {
			t.Fatalf("HTTP start-refusal evidence %d kind=%s want=%s",
				index, reference.Kind(), wantProcessKinds[index])
		}
	}
	checks := witness.StandaloneScope().Checks()
	if len(checks) != 5 {
		t.Fatalf("HTTP start-refusal scope has %d checks, want five", len(checks))
	}
	for index, check := range checks {
		wantState := contractmodel.ScopeCheckMissing
		if index == 0 {
			wantState = contractmodel.ScopeCheckClean
		}
		if check.State() != wantState {
			t.Fatalf("HTTP start-refusal domain %s state=%s want=%s",
				check.Domain(), check.State(), wantState)
		}
	}
	recovered, err := ResumeClassification(ctx, fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("HTTP start-refusal recovery diverged: valid=%t err=%v", recovered.Valid(), err)
	}
	assertInternalMarkerOnly(t, fixture)
}

func assertInternalMarkerOnly(t testing.TB, fixture httpfixture.Fixture) {
	t.Helper()
	entries, err := os.ReadDir(fixture.Target.Roots().EvidenceRoot())
	want := filepath.Base(fixture.Target.Roots().MarkerPath())
	if err != nil || len(entries) != 1 || entries[0].Name() != want {
		t.Fatalf("HTTP evidence root did not retire to marker-only: entries=%v want=%q err=%v",
			entries, want, err)
	}
	info, err := os.Lstat(fixture.Target.Roots().EvidenceRoot())
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 ||
		info.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		t.Fatalf("HTTP evidence root facts changed: mode=%v err=%v", info, err)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Geteuid() {
		t.Fatal("HTTP evidence root ownership changed")
	}
}
