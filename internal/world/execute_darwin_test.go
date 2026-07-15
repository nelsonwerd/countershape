//go:build darwin && cgo

package world

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

type executionHarness struct {
	plan           domain.WorldPlan
	candidate      gitobj.BoundCandidate
	registry       ToolRegistry
	allocationRoot string
	stimulus       domain.Digest
}

func worldTestDigest(character string) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(character, 64))
}

func worldTestProjection(t *testing.T) domain.ProjectionDefinitionBinding {
	t.Helper()
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain:        domain.AdapterCLI,
		ImplementationDigest: worldTestDigest("7"),
		ConfigurationDigest:  worldTestDigest("8"),
		AcceptedChannels:     []string{"exit", "stderr", "stdout"},
		Operations: []domain.ProjectionOperationBinding{
			{Name: "decode", RuleDigest: worldTestDigest("9")},
			{Name: "select", RuleDigest: worldTestDigest("a")},
		},
		Comparator:          domain.ProjectionComparatorExact,
		FieldRegistryDigest: worldTestDigest("b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func newExecutionHarness(t *testing.T, executable string, argv []string, probeMS, teardownMS, stdoutBytes, stderrBytes int64) executionHarness {
	t.Helper()
	ctx := context.Background()
	fixture, err := gitrepo.Init(ctx, "/usr/bin/git", resolvedPrivateTempDir(t), gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	firstCommit, err := fixture.CommitFiles(ctx, []gitrepo.File{{Path: "candidate.txt", Mode: "100644", Content: []byte("first\n")}}, "", "first")
	if err != nil {
		t.Fatal(err)
	}
	secondCommit, err := fixture.CommitFiles(ctx, []gitrepo.File{{Path: "candidate.txt", Mode: "100644", Content: []byte("second\n")}}, "", "second")
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(ctx, "refs/heads/first", firstCommit); err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(ctx, "refs/heads/second", secondCommit); err != nil {
		t.Fatal(err)
	}
	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: "/usr/bin/git", Repository: fixture.Root, ScratchRoot: resolvedPrivateTempDir(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	})
	first, err := repository.Pin(ctx, "refs/heads/first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Pin(ctx, "refs/heads/second")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := gitobj.SelectTrees(first, second)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := gitobj.NewPolicy(16, 1<<20, 1<<18)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, policy, resolvedPrivateTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: selected.Digest(), MaterializationPolicyDigest: policy.Digest(),
		ComparisonEnvelopeDigest: worldTestDigest("3"),
		Adapter:                  domain.Adapter{Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: worldTestDigest("4")},
		ExecutionShape:           domain.OneCLIInvocation, StartArgv: append([]string(nil), argv...), SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}, {Name: "NO_COLOR", Value: "1"}},
		SecretSlots: []domain.SecretSlot{}, FixtureRecipeDigest: worldTestDigest("5"),
		Readiness: domain.Readiness{Kind: domain.ReadinessNone}, CapturePolicyDigest: worldTestDigest("6"),
		ProjectionDefinition: worldTestProjection(t),
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools: []domain.RequiredTool{{Name: "fixture", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 16, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ReadinessMS: 0, ProbeMS: probeMS, TeardownMS: teardownMS,
			StdoutBytes: stdoutBytes, StderrBytes: stderrBytes, HTTPBodyBytes: 1 << 16,
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 8, ShrinkWallMS: 10_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind(plan)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewToolRegistry(ctx, resolvedPrivateTempDir(t), ToolSpec{
		Name: "fixture", AbsolutePath: executable, VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return executionHarness{
		plan: plan, candidate: bound[0], registry: registry, allocationRoot: resolvedPrivateTempDir(t),
		stimulus: worldTestDigest("c"),
	}
}

func (h executionHarness) request(nonce string) Request {
	return Request{
		Plan: h.plan, Candidate: h.candidate, Tools: h.registry,
		AllocationRoot: h.allocationRoot, StimulusDigest: h.stimulus, Purpose: domain.AttemptDiscovery,
		InstanceNonce: nonce, ScheduleOrdinal: 0,
	}
}

func TestInstanceNonceIsClosedTextBeforeAnyAllocation(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	for _, nonce := range []string{"nonce-plain", "試行-α", strings.Repeat("n", maxInstanceNonceBytes)} {
		if !validInstanceNonce(nonce) {
			t.Fatalf("valid closed nonce was rejected: %q", nonce)
		}
	}
	invalid := []string{
		"", " leading-space", "trailing-space ", "line\nbreak", "delete\x7f",
		string([]byte{0xff}), strings.Repeat("n", maxInstanceNonceBytes+1),
	}
	for index, nonce := range invalid {
		if validInstanceNonce(nonce) {
			t.Fatalf("invalid nonce %d was admitted: %q", index, nonce)
		}
		_, err := Execute(context.Background(), harness.request(nonce))
		if code, ok := RefusalCodeOf(err); !ok || code != CodeInvalidRequest {
			t.Fatalf("invalid nonce %d refusal = %v", index, err)
		}
		entries, readErr := os.ReadDir(harness.allocationRoot)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("invalid nonce %d crossed the allocation edge: entries=%v err=%v", index, entries, readErr)
		}
	}
}

func TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	first, err := Execute(context.Background(), harness.request("nonce-one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Execute(context.Background(), harness.request("nonce-two"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Roots().Attempt() == second.Roots().Attempt() || first.Process().AttemptID() == second.Process().AttemptID() ||
		first.World().Digest() == second.World().Digest() {
		t.Fatal("two executions reused physical or logical world identity")
	}
	for index, result := range []Result{first, second} {
		process := result.Process()
		if result.FinalizedAttempt().HasControls() || controlPresent(process) || !process.Digest().Valid() ||
			!process.MarkerExistedBeforeSpawn() || process.PID() <= 0 || process.PID() != process.ProcessGroupID() ||
			!process.DirectChildWaited() || !process.DrainsComplete() || !process.FinalGroupProbeClean() ||
			process.TeardownError() || process.OrphanRisk() {
			t.Fatalf("result %d lacks a clean receipted lifecycle: %+v", index, process)
		}
		if process.ToolPath() != executable || process.ToolMajor() != 1 || !process.ToolExecutableDigest().Valid() ||
			process.WorldDigest() != result.World().Digest() || process.AttemptArtifactDigest() != result.FinalizedAttempt().ArtifactDigest() {
			t.Fatalf("result %d receipt bindings are incomplete", index)
		}
		materialized := result.Materialization()
		if !materialized.Valid() || materialized.PublishedRoot != filepath.Join(result.Roots().CandidateParent(), "candidate") ||
			len(materialized.Entries) != 1 {
			t.Fatalf("result %d materialization receipt is incomplete: %+v", index, materialized)
		}
		if _, err := os.Lstat(filepath.Join(materialized.PublishedRoot, ".git")); !os.IsNotExist(err) {
			t.Fatalf("result %d published .git metadata: %v", index, err)
		}
		marker, err := os.Lstat(result.Roots().Marker())
		if err != nil || !marker.Mode().IsRegular() || marker.Mode().Perm() != 0o600 {
			t.Fatalf("result %d marker facts changed: %v %+v", index, err, marker)
		}
		invocations, err := filepath.Glob(filepath.Join(result.Roots().State(), "invocation.*.json"))
		if err != nil || len(invocations) != 1 {
			t.Fatalf("result %d invocation evidence = %q, err %v", index, invocations, err)
		}
		var invocation struct {
			AttemptID            string `json:"attempt_id"`
			MarkerPresentAtStart bool   `json:"marker_present_at_start"`
		}
		bytes, err := os.ReadFile(invocations[0])
		if err != nil || json.Unmarshal(bytes, &invocation) != nil || !invocation.MarkerPresentAtStart || invocation.AttemptID != process.AttemptID() {
			t.Fatalf("result %d fixture invocation does not bind the durable marker: %s, %v", index, bytes, err)
		}
		wantStates := []domain.AttemptState{
			domain.AttemptAllocated, domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady,
			domain.AttemptProbing, domain.AttemptCapturing, domain.AttemptTearingDown, domain.AttemptFinalized,
		}
		if strings.Join(stateStrings(result.StateHistory()), "\x00") != strings.Join(stateStrings(wantStates), "\x00") {
			t.Fatalf("result %d state history = %v, want %v", index, result.StateHistory(), wantStates)
		}
	}
}

func controlPresent(receipt ProcessReceipt) bool {
	_, present := receipt.PrimaryControl()
	return present
}

type markerCorruptingMaterializer struct{ delegate gitobj.DefaultMaterializer }

func (m markerCorruptingMaterializer) Materialize(ctx context.Context, candidate gitobj.BoundCandidate, parent string) (gitobj.MaterializationReceipt, error) {
	receipt, err := m.delegate.Materialize(ctx, candidate, parent)
	if err != nil {
		return receipt, err
	}
	marker := filepath.Join(filepath.Dir(parent), "evidence", markerFilename)
	if err := os.WriteFile(marker, []byte("changed-before-spawn"), 0o600); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func TestExecuteRefusesSpawnWhenDurableMarkerChanges(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	result, err := executeWithMaterializer(context.Background(), harness.request("corrupt-marker"), markerCorruptingMaterializer{})
	if err != nil {
		t.Fatal(err)
	}
	primary, present := result.Process().PrimaryControl()
	if !present || primary != domain.ControlStartError || result.Process().PID() != 0 || result.Process().MarkerExistedBeforeSpawn() ||
		result.Process().DiagnosticCode() != diagnosticMarkerContentInvalid || !result.Process().Digest().Valid() {
		t.Fatalf("changed marker did not stop spawn: primary=%s present=%v process=%+v", primary, present, result.Process())
	}
	invocations, err := filepath.Glob(filepath.Join(result.Roots().State(), "invocation.*.json"))
	if err != nil || len(invocations) != 0 {
		t.Fatalf("fixture ran despite changed marker: %q, %v", invocations, err)
	}
}

type changedCandidateMaterializer struct{ delegate gitobj.DefaultMaterializer }

func (m changedCandidateMaterializer) Materialize(ctx context.Context, candidate gitobj.BoundCandidate, parent string) (gitobj.MaterializationReceipt, error) {
	receipt, err := m.delegate.Materialize(ctx, candidate, parent)
	if err != nil {
		return receipt, err
	}
	path := filepath.Join(receipt.PublishedRoot, "candidate.txt")
	if err := os.WriteFile(path, []byte("evil!\n"), 0o644); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func TestPublishedCandidateIsRevalidatedBeforeSpawn(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	result, err := executeWithMaterializer(context.Background(), harness.request("changed-candidate"), changedCandidateMaterializer{})
	if err != nil {
		t.Fatal(err)
	}
	primary, present := result.Process().PrimaryControl()
	if !present || primary != domain.ControlMaterializationError || result.Process().SpawnAttempted() || result.Process().PID() != 0 ||
		result.Process().DiagnosticCode() != diagnosticMaterializationRevalidationFailed || !result.Process().Digest().Valid() {
		t.Fatalf("changed candidate crossed the spawn edge: primary=%s present=%v process=%+v", primary, present, result.Process())
	}
}

type refusingMaterializer struct{ code gitobj.RefusalCode }

func (m refusingMaterializer) Materialize(context.Context, gitobj.BoundCandidate, string) (gitobj.MaterializationReceipt, error) {
	return gitobj.MaterializationReceipt{}, &gitobj.Refusal{Code: m.code, Detail: "fixture materialization refusal"}
}

func TestMaterializationRefusalCodeSurvivesInProcessReceipt(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	for _, fixture := range []struct {
		name, nonce string
		code        gitobj.RefusalCode
		primary     domain.ControlReason
	}{
		{name: "missing-object", nonce: "missing-object", code: gitobj.CodeMissingObject, primary: domain.ControlMissingObject},
		{name: "publication-ambiguous", nonce: "publication-ambiguous", code: gitobj.CodePublicationAmbiguous, primary: domain.ControlMaterializationError},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			result, err := executeWithMaterializer(
				context.Background(), harness.request(fixture.nonce), refusingMaterializer{code: fixture.code},
			)
			if err != nil {
				t.Fatal(err)
			}
			primary, present := result.Process().PrimaryControl()
			if !present || primary != fixture.primary || result.Process().SpawnAttempted() ||
				result.Process().DiagnosticCode() != string(fixture.code) || !result.Process().Digest().Valid() {
				t.Fatalf("materialization refusal lost stable receipt data: primary=%s present=%v process=%+v", primary, present, result.Process())
			}
		})
	}
}

type contextIgnoringMaterializer struct{ delegate gitobj.DefaultMaterializer }

func (m contextIgnoringMaterializer) Materialize(_ context.Context, candidate gitobj.BoundCandidate, parent string) (gitobj.MaterializationReceipt, error) {
	return m.delegate.Materialize(context.Background(), candidate, parent)
}

type cancellingMaterializer struct{ cancel context.CancelFunc }

func (m cancellingMaterializer) Materialize(context.Context, gitobj.BoundCandidate, string) (gitobj.MaterializationReceipt, error) {
	m.cancel()
	return gitobj.MaterializationReceipt{}, context.Canceled
}

type cancellingTypedMaterializer struct {
	cancel context.CancelFunc
	code   gitobj.RefusalCode
}

func (m cancellingTypedMaterializer) Materialize(context.Context, gitobj.BoundCandidate, string) (gitobj.MaterializationReceipt, error) {
	m.cancel()
	return gitobj.MaterializationReceipt{}, &gitobj.Refusal{Code: m.code, Detail: "typed failure concurrent with cancellation"}
}

func TestCancellationDuringMaterializationHasPhaseStableDiagnostic(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	ctx, cancel := context.WithCancel(context.Background())
	result, err := executeWithMaterializer(ctx, harness.request("cancel-during-materialization"), cancellingMaterializer{cancel: cancel})
	if err != nil {
		t.Fatal(err)
	}
	primary, present := result.Process().PrimaryControl()
	if !present || primary != domain.ControlCancelled || result.Process().SpawnAttempted() ||
		result.Process().DiagnosticCode() != diagnosticCancelledDuringMaterialization || !result.Process().Digest().Valid() {
		t.Fatalf("materialization cancellation lost phase-stable diagnostics: primary=%s present=%v process=%+v", primary, present, result.Process())
	}
}

func TestTypedPublicationAmbiguityOutranksCoincidentCancellation(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	ctx, cancel := context.WithCancel(context.Background())
	result, err := executeWithMaterializer(ctx, harness.request("cancel-and-publication-ambiguous"), cancellingTypedMaterializer{
		cancel: cancel,
		code:   gitobj.CodePublicationAmbiguous,
	})
	if err != nil {
		t.Fatal(err)
	}
	primary, present := result.Process().PrimaryControl()
	if !present || primary != domain.ControlMaterializationError || result.Process().SpawnAttempted() ||
		result.Process().DiagnosticCode() != string(gitobj.CodePublicationAmbiguous) || !result.Process().Digest().Valid() {
		t.Fatalf("coincident cancellation masked publication ambiguity: primary=%s present=%v process=%+v", primary, present, result.Process())
	}
}

func TestPreCancelledContextCannotSpawnEvenWhenMaterializerIgnoresIt(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := executeWithMaterializer(ctx, harness.request("pre-cancelled"), contextIgnoringMaterializer{})
	if err != nil {
		t.Fatal(err)
	}
	primary, present := result.Process().PrimaryControl()
	if !present || primary != domain.ControlCancelled || result.Process().SpawnAttempted() || result.Process().PID() != 0 ||
		result.Process().DiagnosticCode() != diagnosticCancelledAfterMaterialization || !result.Process().Digest().Valid() {
		t.Fatalf("pre-cancelled request crossed the spawn edge: primary=%s present=%v process=%+v", primary, present, result.Process())
	}
}

func TestStableDiagnosticCodeParticipatesInProcessReceiptDigest(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16)
	allocated, err := allocateAttempt(harness.request("diagnostic-digest"))
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := allocated.domainAttempt.Advance(domain.AttemptMaterializing)
	if err != nil {
		t.Fatal(err)
	}
	first, err := finalizeWithoutProcess(
		allocated, attempt, domain.ControlMaterializationError,
		withReceiptDiagnostic(diagnosticMaterializationFailed, errors.New("same private detail")),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := finalizeWithoutProcess(
		allocated, attempt, domain.ControlMaterializationError,
		withReceiptDiagnostic(diagnosticMaterializationReceiptInvalid, errors.New("same private detail")),
	)
	if err != nil {
		t.Fatal(err)
	}
	third, err := finalizeWithoutProcess(
		allocated, attempt, domain.ControlMaterializationError,
		withReceiptDiagnostic(diagnosticMaterializationFailed, errors.New("different private detail")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Process().DiagnosticCode() != diagnosticMaterializationFailed ||
		second.Process().DiagnosticCode() != diagnosticMaterializationReceiptInvalid ||
		first.Process().Digest() == second.Process().Digest() || first.Process().Digest() != third.Process().Digest() {
		t.Fatalf("receipt digest did not bind only stable diagnostic code: first=%+v second=%+v third=%+v", first.Process(), second.Process(), third.Process())
	}
}

func TestEscapedPipeHolderRecordsOrphanRiskAsSecondaryControl(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(
		t, executable, []string{"fixture", "--mode", "setsid-escape"},
		100, 300, 1024, 1024,
	)
	result, err := Execute(context.Background(), harness.request("setsid-orphan-risk"))
	if err != nil {
		t.Fatal(err)
	}
	process := result.Process()
	registerEmergencyGroupCleanup(t, physicalProcessResult{
		processGroupOwned: process.ProcessGroupOwned(), processGroupID: process.ProcessGroupID(), pid: process.PID(),
	})
	registerEmergencyPIDFileCleanup(t, filepath.Join(result.Roots().State(), "escaped.pid"))
	primary, present := process.PrimaryControl()
	teardown := result.FinalizedAttempt().TeardownControls()
	if !present || primary != domain.ControlTimeout || len(teardown) != 2 ||
		teardown[0] != domain.ControlTeardownError || teardown[1] != domain.ControlOrphanRisk ||
		!process.TeardownError() || !process.OrphanRisk() || process.DrainsComplete() || !process.FinalGroupProbeClean() ||
		process.PreTermGroupProbe() != preTermProbePresent || process.DiagnosticCode() != "PIPE_DRAIN_DEADLINE" ||
		process.CleanupBoundary() != processEscapeExclusion || process.ProcessGroupSignalBoundary() != processGroupReuseExclusion {
		t.Fatalf("escaped pipe holder lost secondary cleanup evidence: primary=%s present=%v teardown=%v process=%+v", primary, present, teardown, process)
	}
}
