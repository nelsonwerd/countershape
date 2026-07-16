package reduction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/store"
)

type finalizationStimulus struct {
	digest  domain.Digest
	measure reduce.Measure
}

type finalizationFixture struct {
	envelope domain.ComparisonEnvelope
	plan     domain.WorldPlan
	bindings []domain.CandidateExecutionBinding
	roster   []domain.CandidateExecutionKey
}

func finalizationDigest(number int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", number))
}

func newFinalizationFixture(t *testing.T, seed int) finalizationFixture {
	t.Helper()
	envelope, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: fmt.Sprintf("finalization-%d/v1", seed),
		Measured: []domain.MeasuredDimension{{
			Name: "os", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact,
		}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "os"}},
		Uncontrolled:  []string{"scheduler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := seed * 100_000
	projection, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: finalizationDigest(base + 6),
		ConfigurationDigest: finalizationDigest(base + 7), AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations: []domain.ProjectionOperationBinding{{Name: "test-projection", RuleDigest: finalizationDigest(base + 8)}},
		Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: finalizationDigest(base + 9),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: finalizationDigest(base + 1), MaterializationPolicyDigest: finalizationDigest(base + 2),
		ComparisonEnvelopeDigest: envelope.Digest(),
		Adapter:                  domain.Adapter{Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: finalizationDigest(base + 3)},
		ExecutionShape:           domain.OneCLIInvocation, StartArgv: []string{"node", "fixture.mjs"}, SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, SecretSlots: []domain.SecretSlot{},
		FixtureRecipeDigest: finalizationDigest(base + 4), Readiness: domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest: finalizationDigest(base + 5), ProjectionDefinition: projection,
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: 1, ConfirmationRepeats: 1, Concurrency: domain.ScheduleSequential,
			Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 10, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ProbeMS: 1000, TeardownMS: 1000, StdoutBytes: 1 << 16,
			StderrBytes: 1 << 16, HTTPBodyBytes: 1 << 16, ProposedShrinkStimuli: 10,
			TotalCandidateTrials: 100, ShrinkWallMS: 60_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture := finalizationFixture{envelope: envelope, plan: plan}
	for index := 0; index < 2; index++ {
		binding, bindingErr := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
			TreeIdentityDigest:          finalizationDigest(base + 100 + index),
			MaterializationPolicyDigest: plan.MaterializationPolicyDigest(), WorldPlanDigest: plan.Digest(),
			AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
			ProjectionDefinitionDigest: plan.ProjectionDefinitionDigest(),
		})
		if bindingErr != nil {
			t.Fatal(bindingErr)
		}
		fixture.bindings = append(fixture.bindings, binding)
		fixture.roster = append(fixture.roster, binding.Key())
	}
	return fixture
}

func finalizationAttempt(t *testing.T, digest domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+digest.String(), digest, purpose)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady, domain.AttemptProbing,
		domain.AttemptCapturing, domain.AttemptTearingDown, domain.AttemptFinalized,
	} {
		attempt, err = attempt.Advance(state)
		if err != nil {
			t.Fatal(err)
		}
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	return finalized
}

func finalizationProjection(t *testing.T, world domain.WorldInstance, observation domain.Digest, value int) observe.ProjectionDerivation {
	t.Helper()
	projection, err := canon.Integer(int64(value))
	if err != nil {
		t.Fatal(err)
	}
	derivation, err := observe.NewProjectionDerivation(
		observation, world.ProjectionDefinitionDigest(), projection.Canonical(),
		[]byte(`{"kind":"TEST_PROJECTION_DERIVATION","operations":["test"],"source_links":["test"]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	return derivation
}

func finalizationMap(
	t *testing.T,
	fixture finalizationFixture,
	stimulus domain.Digest,
	purpose domain.AttemptPurpose,
	offset int,
) compare.CandidateOutcomeMap {
	t.Helper()
	worlds := make([]domain.WorldInstance, len(fixture.bindings))
	attempts := make([]domain.FinalizedAttempt, len(fixture.bindings))
	measurements := make([]domain.InstanceMeasurements, len(fixture.bindings))
	osValue, err := canon.String("darwin")
	if err != nil {
		t.Fatal(err)
	}
	for index, binding := range fixture.bindings {
		attemptDigest := finalizationDigest(offset*100 + index + 1)
		attempts[index] = finalizationAttempt(t, attemptDigest, purpose)
		world, worldErr := domain.NewWorldInstance(fixture.plan, binding, domain.WorldInstanceConfig{
			StimulusDigest: stimulus, AttemptArtifactDigest: attemptDigest, Purpose: purpose,
			InstanceNonce: fmt.Sprintf("finalization:%d:%d", offset, index), ScheduleOrdinal: offset*2 + index,
		})
		if worldErr != nil {
			t.Fatal(worldErr)
		}
		worlds[index] = world
		row, rowErr := domain.NewInstanceMeasurements(fixture.envelope, world, []domain.MeasurementValue{{
			Name: "os", Source: domain.MeasuredWorldInstance, Value: osValue,
		}})
		if rowErr != nil {
			t.Fatal(rowErr)
		}
		measurements[index] = row
	}
	assessment, err := domain.AssessComparison(fixture.envelope, measurements)
	if err != nil {
		t.Fatal(err)
	}
	admitted, ok := assessment.(domain.AdmittedComparison)
	if !ok {
		t.Fatalf("comparison assessment = %T, want admitted", assessment)
	}
	batches := make([]observe.StableBatch, len(fixture.bindings))
	for index := range fixture.bindings {
		token, tokenErr := admitted.AdmissionFor(measurements[index])
		if tokenErr != nil {
			t.Fatal(tokenErr)
		}
		observationDigest := finalizationDigest(offset*100 + 50 + index)
		projectionValue := 10 + index
		projection, projectionErr := canon.Integer(int64(projectionValue))
		if projectionErr != nil {
			t.Fatal(projectionErr)
		}
		capture, captureErr := observe.NewStructuralCapture(
			worlds[index], attempts[index], observationDigest, projection.Canonical(),
			finalizationProjection(t, worlds[index], observationDigest, projectionValue),
		)
		if captureErr != nil {
			t.Fatal(captureErr)
		}
		trial, trialErr := observe.NewCapturedTrial(worlds[index], attempts[index], token, capture)
		if trialErr != nil {
			t.Fatal(trialErr)
		}
		batch, batchErr := observe.Classify(observe.BatchInput{Trials: []observe.TrialFact{trial}})
		if batchErr != nil {
			t.Fatal(batchErr)
		}
		batches[index] = batch
	}
	result, err := compare.NewCandidateOutcomeMap(stimulus, fixture.envelope.Digest(), fixture.roster, batches)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func finalizationRun(t *testing.T, seed int, accepted bool) (reduce.ReductionRun, reduce.CompletedSweepDraft) {
	return finalizationRunWithBudgetBinding(t, seed, accepted, true)
}

func finalizationRunWithBudgetBinding(t *testing.T, seed int, accepted, planBound bool) (reduce.ReductionRun, reduce.CompletedSweepDraft) {
	t.Helper()
	fixture := newFinalizationFixture(t, seed)
	base := seed * 100_000
	definition := finalizationDigest(base + 500)
	rule, err := reduce.NewReducerRule("drop-noise", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := reduce.NewReducerSet("test", definition, finalizationDigest(base+501), []reduce.ReducerRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	originalMeasure, err := reduce.NewMeasure(definition, []uint64{2, 2})
	if err != nil {
		t.Fatal(err)
	}
	smallerMeasure, err := reduce.NewMeasure(definition, []uint64{1, 1})
	if err != nil {
		t.Fatal(err)
	}
	original := finalizationStimulus{digest: finalizationDigest(base + 600), measure: originalMeasure}
	smaller := finalizationStimulus{digest: finalizationDigest(base + 601), measure: smallerMeasure}
	graph := map[domain.Digest][]reduce.TypedProposal[finalizationStimulus]{}
	if accepted {
		neighbor, neighborErr := reduce.NewNeighbor(reduce.NeighborInput{
			CurrentStimulus: original.digest, CurrentMeasure: original.measure, Stimulus: smaller.digest,
			Measure: smaller.measure, Rule: rule, Locus: "fixture.noise", ReducerSet: set,
		})
		if neighborErr != nil {
			t.Fatal(neighborErr)
		}
		graph[original.digest] = []reduce.TypedProposal[finalizationStimulus]{{Stimulus: smaller, Neighbor: neighbor}}
	}
	baselineMap := finalizationMap(t, fixture, original.digest, domain.AttemptDiscovery, base+700)
	baseline, err := compare.RequireDivergence(baselineMap)
	if err != nil {
		t.Fatal(err)
	}
	var budget reduce.Budget
	if planBound {
		budget, err = reduce.NewBudgetFromWorldPlan(fixture.plan)
	} else {
		planBudgets := fixture.plan.Budgets()
		reserved := planBudgets.CandidateCount *
			(fixture.plan.RepeatSchedule().DiscoveryRepeats + fixture.plan.RepeatSchedule().ConfirmationRepeats)
		budget, err = reduce.NewBudget(
			uint64(planBudgets.ProposedShrinkStimuli),
			uint64(planBudgets.TotalCandidateTrials-reserved),
			time.Duration(planBudgets.ShrinkWallMS)*time.Millisecond,
		)
	}
	if err != nil {
		t.Fatal(err)
	}
	evaluationOffset := base + 800
	run, err := reduce.Run(context.Background(), reduce.RunInput[finalizationStimulus]{
		Original: original,
		Reference: func(value finalizationStimulus) (domain.Digest, reduce.Measure, bool) {
			return value.digest, value.measure, value.digest.Valid() && value.measure.Valid()
		},
		Enumerate: func(_ context.Context, current finalizationStimulus) ([]reduce.TypedProposal[finalizationStimulus], error) {
			return append([]reduce.TypedProposal[finalizationStimulus](nil), graph[current.digest]...), nil
		},
		Evaluate: func(_ context.Context, value finalizationStimulus, _ reduce.Neighbor, purpose domain.AttemptPurpose, _ reduce.EvaluationAllowance) (reduce.EvaluationObservation, error) {
			observed := finalizationMap(t, fixture, value.digest, purpose, evaluationOffset)
			evaluationOffset++
			return reduce.EvaluationObservation{OutcomeMap: &observed, CandidateTrials: 2}, nil
		},
		Baseline: baseline, ReducerSet: set, Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 {
		t.Fatalf("empty-neighbor completion: present=%t valid=%t neighbors=%d err=%v", present, draft.Valid(), len(draft.NeighborDigests()), err)
	}
	return run, draft
}

func newFinalizationStore(t *testing.T) (*store.ReductionSweepStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "reduction-store")
	value, err := store.OpenReductionSweepStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return value, root
}

func publishCompletion(t *testing.T, value *store.ReductionSweepStore, draft reduce.CompletedSweepDraft) SweepCompletion {
	t.Helper()
	authority, err := value.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	return SweepCompletion{Store: value, Draft: draft, Authority: authority}
}

func TestFinalizeEmptyNeighborDurableSweepConstructsExactStrongGrade(t *testing.T) {
	run, draft := finalizationRun(t, 1, false)
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)

	result, err := Finalize(context.Background(), run, &completion)
	if err != nil {
		t.Fatal(err)
	}
	grade := result.Grade()
	setDigest, setPresent := grade.QuantifiedReducerSetDigest()
	sweepDigest, sweepPresent := grade.CompletedSweepDigest()
	if !result.Valid() || !grade.Valid() || grade.Status() != StatusOneMinimalUnder ||
		!setPresent || setDigest != run.ReducerSet().Digest() || !sweepPresent || sweepDigest != draft.Digest() ||
		result.RunDigest() != run.Digest() || result.TranscriptDigest() != run.Transcript().Digest() ||
		len(grade.Limitations()) != 0 || !strings.Contains(grade.String(), run.ReducerSet().Digest().String()) {
		t.Fatalf("unexpected strong result: valid=%t grade=%s limitations=%v", result.Valid(), grade.String(), grade.Limitations())
	}
}

func TestFinalizeDurableSweepOutranksAcceptedBestKnown(t *testing.T) {
	run, draft := finalizationRun(t, 2, true)
	if !run.HasAcceptedReduction() || run.DraftGrade() != reduce.GradeBestKnown {
		t.Fatal("fixture did not contain an accepted reduction")
	}
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)
	result, err := Finalize(context.Background(), run, &completion)
	if err != nil || result.Grade().Status() != StatusOneMinimalUnder {
		t.Fatalf("durable completion did not outrank weak draft grade: result=%#v err=%v", result, err)
	}
}

func TestFinalizeRefusesStrongGradeFromUnboundAdHocBudget(t *testing.T) {
	run, draft := finalizationRunWithBudgetBinding(t, 24, false, false)
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)
	if _, err := Finalize(context.Background(), run, &completion); err == nil {
		t.Fatal("ad hoc numeric budget was promoted without compiled WorldPlan authority")
	}
}

func TestStrongGradeRejectsEveryRecordedLimitation(t *testing.T) {
	run, draft := finalizationRun(t, 22, false)
	if _, err := newGrade(
		StatusOneMinimalUnder,
		run.Digest(),
		run.ReducerSet().Digest(),
		draft.Digest(),
		[]string{"UNRESOLVED_TRANSIENT_SEARCH_FAILURE"},
	); err == nil {
		t.Fatal("strong grade accepted a run with an unresolved limitation")
	}
}

func TestStrongGradeCanonicalIdentityBindsReducerSetDigest(t *testing.T) {
	run, draft := finalizationRun(t, 23, false)
	grade, err := newGrade(
		StatusOneMinimalUnder,
		run.Digest(),
		run.ReducerSet().Digest(),
		draft.Digest(),
		[]string{},
	)
	if err != nil {
		t.Fatal(err)
	}
	var identity gradeIdentity
	if err := json.Unmarshal(grade.CanonicalBytes(), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.ReducerSetDigest != run.ReducerSet().Digest().String() {
		t.Fatalf("canonical grade reducer set = %q, want %q", identity.ReducerSetDigest, run.ReducerSet().Digest())
	}
	other, err := newGrade(
		StatusOneMinimalUnder,
		run.Digest(),
		finalizationDigest(23*100_000+999),
		draft.Digest(),
		[]string{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if grade.Digest() == other.Digest() || bytes.Equal(grade.CanonicalBytes(), other.CanonicalBytes()) {
		t.Fatal("changing the quantified reducer set did not change strong-grade identity")
	}
}

func TestFinalizeRefusesWrongDraftRunStoreAndStaleAuthority(t *testing.T) {
	run, draft := finalizationRun(t, 3, false)
	otherRun, otherDraft := finalizationRun(t, 4, false)
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)

	t.Run("wrong-draft", func(t *testing.T) {
		wrong := completion
		wrong.Draft = otherDraft
		if _, err := Finalize(context.Background(), run, &wrong); err == nil {
			t.Fatal("authority crossed to a different draft")
		}
	})

	t.Run("wrong-live-run", func(t *testing.T) {
		if _, err := Finalize(context.Background(), otherRun, &completion); err == nil {
			t.Fatal("validated draft crossed to a different live run")
		}
	})

	t.Run("other-store", func(t *testing.T) {
		otherStore, _ := newFinalizationStore(t)
		wrong := completion
		wrong.Store = otherStore
		if _, err := Finalize(context.Background(), run, &wrong); err == nil {
			t.Fatal("authority crossed to another store instance")
		}
	})

	t.Run("restart-stales-process-authority", func(t *testing.T) {
		_, root := newFinalizationStore(t)
		first, err := store.OpenReductionSweepStore(root)
		if err != nil {
			t.Fatal(err)
		}
		issued := publishCompletion(t, first, draft)
		restarted, err := store.OpenReductionSweepStore(root)
		if err != nil {
			t.Fatal(err)
		}
		issued.Store = restarted
		if _, err := Finalize(context.Background(), run, &issued); err == nil {
			t.Fatal("pre-restart authority survived a new store instance")
		}
	})
}

func TestFinalizeRefusesZeroAndJSONCopiedAuthority(t *testing.T) {
	run, draft := finalizationRun(t, 5, false)
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)

	zero := completion
	zero.Authority = store.SweepCompletionAuthority{}
	if _, err := Finalize(context.Background(), run, &zero); err == nil {
		t.Fatal("zero authority constructed a grade")
	}

	wire, err := json.Marshal(completion.Authority)
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != "{}" {
		t.Fatalf("authority unexpectedly exposed wire fields: %s", wire)
	}
	var copied store.SweepCompletionAuthority
	if err := json.Unmarshal(wire, &copied); err != nil {
		t.Fatal(err)
	}
	fromJSON := completion
	fromJSON.Authority = copied
	if _, err := Finalize(context.Background(), run, &fromJSON); err == nil {
		t.Fatal("JSON copy reconstructed authority")
	}
}

func TestFinalizeReopensAndRefusesCorruptionAfterAuthorityIssuance(t *testing.T) {
	run, draft := finalizationRun(t, 6, false)
	value, root := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)
	objects := filepath.Join(root, "objects", "sha256")
	objectPath := ""
	err := filepath.WalkDir(objects, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			if objectPath != "" {
				return fmt.Errorf("multiple immutable objects found")
			}
			objectPath = path
		}
		return nil
	})
	if err != nil || objectPath == "" {
		t.Fatalf("published object path = %q, err=%v", objectPath, err)
	}
	body := draft.CanonicalBytes()
	body[0] = '['
	if err := os.WriteFile(objectPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(context.Background(), run, &completion); err == nil {
		t.Fatal("post-authority corruption constructed a grade")
	}
}

func TestFinalizeFallsBackHonestlyWithoutAuthority(t *testing.T) {
	t.Run("accepted-is-best-known", func(t *testing.T) {
		run, _ := finalizationRun(t, 7, true)
		result, err := Finalize(context.Background(), run, nil)
		if err != nil {
			t.Fatal(err)
		}
		grade := result.Grade()
		if !result.Valid() || grade.Status() != StatusBestKnown ||
			!slices.Contains(grade.Limitations(), limitationDurableSweepAuthorityAbsent) {
			t.Fatalf("accepted fallback = %s, limitations=%v", grade.Status(), grade.Limitations())
		}
		if _, present := grade.QuantifiedReducerSetDigest(); present {
			t.Fatal("BEST_KNOWN exposed a local-minimality quantifier")
		}
	})

	t.Run("no-accepted-reduction-is-unchanged", func(t *testing.T) {
		run, _ := finalizationRun(t, 8, false)
		result, err := Finalize(context.Background(), run, nil)
		if err != nil {
			t.Fatal(err)
		}
		grade := result.Grade()
		if !result.Valid() || grade.Status() != StatusUnchanged || run.HasAcceptedReduction() ||
			!slices.Contains(grade.Limitations(), limitationDurableSweepAuthorityAbsent) {
			t.Fatalf("unchanged fallback = %s, limitations=%v", grade.Status(), grade.Limitations())
		}
	})
}

func TestFinalizeNeverConstructsStrongGradeFromSerializedData(t *testing.T) {
	run, draft := finalizationRun(t, 9, false)
	value, _ := newFinalizationStore(t)
	completion := publishCompletion(t, value, draft)
	result, err := Finalize(context.Background(), run, &completion)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(result.Grade())
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != "{}" {
		t.Fatalf("grade leaked a wire constructor: %s", wire)
	}
	var copied Grade
	if err := json.Unmarshal(wire, &copied); err != nil {
		t.Fatal(err)
	}
	if copied.Valid() || copied.Status() == StatusOneMinimalUnder {
		t.Fatal("serialized grade reconstructed strong authority")
	}
}
