package reduce

import (
	"fmt"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

func digest(number int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", number))
}

func projectionBytes(number int) []byte {
	value, _ := canon.Integer(int64(number))
	return value.Canonical()
}

func testProjectionDerivation(
	t *testing.T,
	world domain.WorldInstance,
	observationDigest domain.Digest,
	projection []byte,
) observe.ProjectionDerivation {
	t.Helper()
	derivation, err := observe.NewProjectionDerivation(
		observationDigest,
		world.ProjectionDefinitionDigest(),
		projection,
		[]byte(`{"kind":"TEST_PROJECTION_DERIVATION","operations":["test"],"source_links":["test"]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	return derivation
}

type reductionFixture struct {
	envelope domain.ComparisonEnvelope
	plan     domain.WorldPlan
	bindings []domain.CandidateExecutionBinding
	roster   []domain.CandidateExecutionKey
}

func newReductionFixture(t *testing.T, version string) reductionFixture {
	t.Helper()
	envelope, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: version,
		Measured: []domain.MeasuredDimension{{
			Name: "os", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact,
		}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "os"}},
		Uncontrolled:  []string{"scheduler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          digest(8001),
		MaterializationPolicyDigest: digest(8002),
		ComparisonEnvelopeDigest:    envelope.Digest(),
		Adapter: domain.Adapter{
			Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: digest(8003),
		},
		ExecutionShape:      domain.OneCLIInvocation,
		StartArgv:           []string{"node", "fixture/cli.mjs", "--mode", "test"},
		SetupArgv:           []string{},
		Environment:         []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}},
		SecretSlots:         []domain.SecretSlot{},
		FixtureRecipeDigest: digest(8004),
		Readiness:           domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest: digest(8005),
		ProjectionDefinition: func() domain.ProjectionDefinitionBinding {
			binding, bindingErr := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
				AdapterDomain: domain.AdapterCLI, ImplementationDigest: digest(8006), ConfigurationDigest: digest(8007),
				AcceptedChannels: []string{"exit", "stderr", "stdout"},
				Operations:       []domain.ProjectionOperationBinding{{Name: "test-projection", RuleDigest: digest(8008)}},
				Comparator:       domain.ProjectionComparatorExact, FieldRegistryDigest: digest(8009),
			})
			if bindingErr != nil {
				t.Fatal(bindingErr)
			}
			return binding
		}(),
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 1, ConfirmationRepeats: 1,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 100, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ReadinessMS: 0, ProbeMS: 1000, TeardownMS: 1000,
			StdoutBytes: 1 << 16, StderrBytes: 1 << 16, HTTPBodyBytes: 1 << 16,
			ProposedShrinkStimuli: 10, TotalCandidateTrials: 100, ShrinkWallMS: 60_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture := reductionFixture{envelope: envelope, plan: plan}
	for index := 0; index < 2; index++ {
		binding, bindingErr := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
			TreeIdentityDigest: digest(8100 + index), MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
			WorldPlanDigest: plan.Digest(), AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
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

func cleanAttempt(t *testing.T, attemptDigest domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+attemptDigest.String(), attemptDigest, purpose)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady,
		domain.AttemptProbing, domain.AttemptCapturing, domain.AttemptTearingDown, domain.AttemptFinalized,
	} {
		attempt, err = attempt.Advance(state)
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func candidateMap(
	t *testing.T,
	fixture reductionFixture,
	stimulus domain.Digest,
	purpose domain.AttemptPurpose,
	leftProjection int,
	rightProjection int,
	offset int,
) compare.CandidateOutcomeMap {
	t.Helper()
	projections := []int{leftProjection, rightProjection}
	worlds := make([]domain.WorldInstance, len(fixture.bindings))
	attempts := make([]domain.FinalizedAttempt, len(fixture.bindings))
	measurements := make([]domain.InstanceMeasurements, len(fixture.bindings))
	value, err := canon.String("darwin")
	if err != nil {
		t.Fatal(err)
	}
	for index, binding := range fixture.bindings {
		attemptDigest := digest(10000 + offset*100 + index)
		attempts[index] = cleanAttempt(t, attemptDigest, purpose)
		world, worldErr := domain.NewWorldInstance(fixture.plan, binding, domain.WorldInstanceConfig{
			StimulusDigest: stimulus, AttemptArtifactDigest: attemptDigest, Purpose: purpose,
			InstanceNonce: fmt.Sprintf("world:%d:%d", index, offset), ScheduleOrdinal: offset*len(fixture.bindings) + index,
		})
		if worldErr != nil {
			t.Fatal(worldErr)
		}
		worlds[index] = world
		row, rowErr := domain.NewInstanceMeasurements(fixture.envelope, world, []domain.MeasurementValue{{
			Name: "os", Source: domain.MeasuredWorldInstance, Value: value,
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
		t.Fatalf("assessment = %T, want admitted", assessment)
	}
	batches := make([]observe.StableBatch, len(fixture.bindings))
	for index := range fixture.bindings {
		token, tokenErr := admitted.AdmissionFor(measurements[index])
		if tokenErr != nil {
			t.Fatal(tokenErr)
		}
		observationDigest := digest(20000 + offset*100 + index)
		projection := projectionBytes(projections[index])
		capture, captureErr := observe.NewStructuralCapture(
			worlds[index], attempts[index], observationDigest, projection,
			testProjectionDerivation(t, worlds[index], observationDigest, projection),
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

func divergentBaseline(t *testing.T, fixture reductionFixture, stimulus domain.Digest, offset int) compare.DivergentBaseline {
	t.Helper()
	mapValue := candidateMap(t, fixture, stimulus, domain.AttemptDiscovery, 10, 11, offset)
	baseline, err := compare.RequireDivergence(mapValue)
	if err != nil {
		t.Fatal(err)
	}
	return baseline
}

func neighborFor(t *testing.T, current, next domain.Digest, measure, nextMeasure int) Neighbor {
	t.Helper()
	definition := digest(690)
	before, err := NewMeasure(definition, []uint64{uint64(measure)})
	if err != nil {
		t.Fatal(err)
	}
	after, err := NewMeasure(definition, []uint64{uint64(nextMeasure)})
	if err != nil {
		t.Fatal(err)
	}
	rule, err := NewReducerRule("test-reducer", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewReducerSet("test", definition, digest(699), []ReducerRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	neighbor, err := NewNeighbor(NeighborInput{
		CurrentStimulus: current, CurrentMeasure: before, Stimulus: next, Measure: after,
		Rule: rule, Locus: "test.value", ReducerSet: set,
	})
	if err != nil {
		t.Fatal(err)
	}
	return neighbor
}

func TestNewNeighborRejectsNondecreasingMeasure(t *testing.T) {
	definition := digest(690)
	before, err := NewMeasure(definition, []uint64{2})
	if err != nil {
		t.Fatal(err)
	}
	equal, err := NewMeasure(definition, []uint64{2})
	if err != nil {
		t.Fatal(err)
	}
	rule, err := NewReducerRule("test-reducer", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewReducerSet("test", definition, digest(699), []ReducerRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewNeighbor(NeighborInput{
		CurrentStimulus: digest(1), CurrentMeasure: before, Stimulus: digest(2), Measure: equal,
		Rule: rule, Locus: "test.value", ReducerSet: set,
	}); err == nil {
		t.Fatal("equal-measure proposal was accepted as a direct reduction neighbor")
	}
}

func TestClassifyNeighborIsTriValuedAndUsesExactMap(t *testing.T) {
	fixture := newReductionFixture(t, "reduce/v1")
	current := digest(1)
	next := digest(2)
	baseline := divergentBaseline(t, fixture, current, 10)
	neighbor := neighborFor(t, current, next, 2, 1)

	preserved := candidateMap(t, fixture, next, domain.AttemptReduction, 10, 11, 20)
	preservedEvaluation, err := NewEvaluation(EvaluationInput{
		ID: "preserved", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &preserved,
	})
	if err != nil || preservedEvaluation.Decision() != Preserves {
		t.Fatalf("preserved = %#v, err = %v", preservedEvaluation, err)
	}
	shapeOnly := candidateMap(t, fixture, next, domain.AttemptReduction, 11, 10, 30)
	shapeEvaluation, err := NewEvaluation(EvaluationInput{
		ID: "shape", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &shapeOnly,
	})
	if err != nil || shapeEvaluation.Decision() != Changes {
		t.Fatalf("shape-only = %#v, err = %v", shapeEvaluation, err)
	}
	collapsed := candidateMap(t, fixture, next, domain.AttemptReduction, 10, 10, 40)
	collapsedEvaluation, err := NewEvaluation(EvaluationInput{
		ID: "collapsed", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &collapsed,
	})
	if err != nil || collapsedEvaluation.Decision() != Changes {
		t.Fatalf("collapsed = %#v, err = %v", collapsedEvaluation, err)
	}
}

func TestUnresolvedNeverCollapsesIntoChanges(t *testing.T) {
	fixture := newReductionFixture(t, "reduce/v1")
	baseline := divergentBaseline(t, fixture, digest(1), 50)
	evaluation, err := NewEvaluation(EvaluationInput{
		ID: "unresolved", Baseline: baseline, Neighbor: neighborFor(t, digest(1), digest(2), 2, 1),
		ObservedOutcomeMap: nil, UnresolvedReason: ReasonTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Decision() != Unresolved {
		t.Fatalf("decision = %s, want UNRESOLVED", evaluation.Decision())
	}
}

func TestTypedUnresolvedReasonsRetainFreshPartialEvidence(t *testing.T) {
	fixture := newReductionFixture(t, "typed-unresolved/v1")
	baseline := divergentBaseline(t, fixture, digest(1), 55)
	neighbor := neighborFor(t, digest(1), digest(2), 2, 1)
	reasons := []UnresolvedReason{
		ReasonUnstable, ReasonIncomplete, ReasonCancelled, ReasonTimeout, ReasonStale, ReasonTeardownError,
	}
	for index, reason := range reasons {
		t.Run(string(reason), func(t *testing.T) {
			base := 90_000 + index*10
			evidence, err := NewUnresolvedEvidence(
				[]domain.Digest{digest(base + 1)}, []domain.Digest{digest(base + 2)},
				[]domain.Digest{digest(base + 3)}, []domain.Digest{digest(base + 4)},
			)
			if err != nil {
				t.Fatal(err)
			}
			evaluation, err := NewEvaluation(EvaluationInput{
				ID: "typed-unresolved:" + string(reason), Baseline: baseline, Neighbor: neighbor,
				UnresolvedReason: reason, UnresolvedEvidence: evidence,
			})
			if err != nil || evaluation.Decision() != Unresolved || evaluation.ReasonCode() != string(reason) ||
				!evaluation.LogicalNonReuseWithBaseline() || len(evaluation.ObservedBatchDigests()) != 1 ||
				len(evaluation.ObservedAttemptDigests()) != 1 || len(evaluation.ObservedWorldDigests()) != 1 ||
				len(evaluation.ObservedObservationDigests()) != 1 {
				t.Fatalf("typed unresolved evidence was lost: evaluation=%#v err=%v", evaluation, err)
			}
		})
	}

	reused, err := NewUnresolvedEvidence(
		baseline.OutcomeMap().BatchDigests(), baseline.OutcomeMap().EvidenceAttemptDigests(),
		baseline.OutcomeMap().EvidenceWorldDigests(), baseline.OutcomeMap().EvidenceObservationDigests(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEvaluation(EvaluationInput{
		ID: "typed-unresolved:reused", Baseline: baseline, Neighbor: neighbor,
		UnresolvedReason: ReasonIncomplete, UnresolvedEvidence: reused,
	}); err == nil {
		t.Fatal("baseline evidence was relabeled as fresh unresolved evidence")
	}
}

func TestTerminalProtocolReasonsAreReservedFromPublicEvaluationConstruction(t *testing.T) {
	fixture := newReductionFixture(t, "reserved-protocol-reasons/v1")
	baseline := divergentBaseline(t, fixture, digest(1), 56)
	neighbor := neighborFor(t, digest(1), digest(2), 2, 1)
	for _, reason := range []UnresolvedReason{
		ReasonEvaluatorExceededBudget,
		ReasonObservedMapWithoutTrials,
		ReasonUnresolvedEvidenceMissing,
		ReasonInvalidEvaluatorResult,
		ReasonReusedEvaluationEvidence,
	} {
		t.Run(string(reason), func(t *testing.T) {
			if _, err := NewEvaluation(EvaluationInput{
				ID: "caller-authored:" + string(reason), Baseline: baseline, Neighbor: neighbor,
				UnresolvedReason: reason,
			}); err == nil {
				t.Fatal("public evaluation constructor accepted an internal protocol-refusal reason")
			}
		})
	}
}

func TestEvaluationBindsStimulusEnvelopeRosterAndDerivedBatches(t *testing.T) {
	fixture := newReductionFixture(t, "reduce/v1")
	otherFixture := newReductionFixture(t, "reduce/v2")
	baseline := divergentBaseline(t, fixture, digest(1), 60)
	neighbor := neighborFor(t, digest(1), digest(2), 2, 1)

	wrongStimulus := candidateMap(t, fixture, digest(3), domain.AttemptReduction, 10, 11, 70)
	if _, err := NewEvaluation(EvaluationInput{ID: "wrong-stimulus", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &wrongStimulus}); err == nil {
		t.Fatal("observed map from another stimulus entered evaluation")
	}
	wrongEnvelope := candidateMap(t, otherFixture, digest(2), domain.AttemptReduction, 10, 11, 80)
	incomparable, err := NewEvaluation(EvaluationInput{ID: "wrong-envelope", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &wrongEnvelope})
	if err != nil || incomparable.Decision() != Unresolved ||
		incomparable.ReasonCode() != "CANDIDATE_ELIGIBILITY_OR_ADMISSION_CHANGED" ||
		!incomparable.LogicalNonReuseWithBaseline() {
		t.Fatalf("incomparable map was not retained as typed unresolved evidence: %#v err=%v", incomparable, err)
	}
	zero := compare.CandidateOutcomeMap{}
	if _, err := NewEvaluation(EvaluationInput{ID: "zero-map", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &zero}); err == nil {
		t.Fatal("zero observed map entered evaluation")
	}
	valid := candidateMap(t, fixture, digest(2), domain.AttemptReduction, 10, 11, 90)
	evaluation, err := NewEvaluation(EvaluationInput{ID: "valid", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &valid})
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluation.ObservedBatchDigests()) != len(valid.CandidateRoster()) || !evaluation.LogicalNonReuseWithBaseline() {
		t.Fatal("evaluation did not derive exact batch set/non-overlap from observed map")
	}
}

func TestEvaluationRequiresPurposeBoundEvidence(t *testing.T) {
	fixture := newReductionFixture(t, "purpose/v1")
	current := digest(1)
	next := digest(2)
	baseline := divergentBaseline(t, fixture, current, 300)
	neighbor := neighborFor(t, current, next, 2, 1)

	for _, purpose := range []domain.AttemptPurpose{
		domain.AttemptDiscovery,
		domain.AttemptConfirmation,
		domain.AttemptConformance,
		domain.AttemptFinalSweep,
	} {
		observed := candidateMap(t, fixture, next, purpose, 10, 11, 310+int(purpose[0]))
		if _, err := NewEvaluation(EvaluationInput{
			ID: "wrong-purpose:" + string(purpose), Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &observed,
		}); err == nil {
			t.Fatalf("%s evidence entered ordinary reduction", purpose)
		}
	}

	reduction := candidateMap(t, fixture, next, domain.AttemptReduction, 10, 11, 350)
	if _, err := NewFinalSweepEvaluation(EvaluationInput{
		ID: "reduction-as-final", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &reduction,
	}); err == nil {
		t.Fatal("REDUCTION evidence entered a final-sweep evaluation")
	}
	final := candidateMap(t, fixture, next, domain.AttemptFinalSweep, 10, 11, 360)
	if _, err := NewFinalSweepEvaluation(EvaluationInput{
		ID: "final", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &final,
	}); err != nil {
		t.Fatalf("FINAL_SWEEP evidence was refused: %v", err)
	}
}

func TestEvaluationRejectsUnderlyingAttemptReuseAcrossStimuli(t *testing.T) {
	fixture := newReductionFixture(t, "reuse/v1")
	current := digest(1)
	next := digest(2)
	baseline := divergentBaseline(t, fixture, current, 400)
	neighbor := neighborFor(t, current, next, 2, 1)
	// The stimulus changes every WorldInstance and batch digest, but reusing the
	// offset deliberately reuses the underlying finalized attempt artifacts.
	observed := candidateMap(t, fixture, next, domain.AttemptReduction, 10, 11, 400)
	if _, err := NewEvaluation(EvaluationInput{
		ID: "reused-attempts", Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &observed,
	}); err == nil {
		t.Fatal("relabeling reused attempt evidence as a new stimulus entered reduction")
	}
}

func changedEvaluation(t *testing.T, id string, fixture reductionFixture, baseline compare.DivergentBaseline, neighbor Neighbor, offset int, finalSweep bool) Evaluation {
	t.Helper()
	purpose := domain.AttemptReduction
	constructor := NewEvaluation
	if finalSweep {
		purpose = domain.AttemptFinalSweep
		constructor = NewFinalSweepEvaluation
	}
	changed := candidateMap(t, fixture, neighbor.StimulusDigest(), purpose, 10, 10, offset)
	evaluation, err := constructor(EvaluationInput{ID: id, Baseline: baseline, Neighbor: neighbor, ObservedOutcomeMap: &changed})
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Decision() != Changes {
		t.Fatalf("derived decision = %s, want CHANGES", evaluation.Decision())
	}
	return evaluation
}

func logicalSweepInput(t *testing.T, state SweepState) LogicalSweepInput {
	t.Helper()
	fixture := newReductionFixture(t, "sweep/v1")
	current := digest(1)
	baseline := divergentBaseline(t, fixture, current, 100)
	first := neighborFor(t, current, digest(20), 2, 1)
	second := neighborFor(t, current, digest(21), 2, 1)
	return LogicalSweepInput{
		State: state, Baseline: baseline, CurrentStimulus: current, CurrentMeasure: first.CurrentMeasure(),
		ReducerSetDigest: first.ReducerSetDigest(), EnumeratedNeighbors: []Neighbor{second, first},
		Evaluations: []Evaluation{
			changedEvaluation(t, "eval:second", fixture, baseline, second, 120, true),
			changedEvaluation(t, "eval:first", fixture, baseline, first, 130, true),
		},
	}
}

func TestLogicalCompleteSweepRequiresExactBoundChangesSet(t *testing.T) {
	input := logicalSweepInput(t, SweepComplete)
	sweep, err := NewLogicalCompleteSweep(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(sweep.NeighborDigests()) != 2 || sweep.ReducerSetDigest() != input.ReducerSetDigest {
		t.Fatal("logical sweep lost exact reducer/neighbor binding")
	}
	mismatch := logicalSweepInput(t, SweepComplete)
	mismatch.EnumeratedNeighbors = mismatch.EnumeratedNeighbors[:1]
	if _, err := NewLogicalCompleteSweep(mismatch); err == nil {
		t.Fatal("incomplete enumerated neighbor set was accepted")
	}
	exhausted := logicalSweepInput(t, SweepComplete)
	exhausted.BudgetExhausted = true
	if _, err := NewLogicalCompleteSweep(exhausted); err == nil {
		t.Fatal("budget-exhausted sweep became logically complete")
	}
}

func TestLogicalCompleteSweepRequiresFinalSweepPurposeAndGlobalEvidenceUniqueness(t *testing.T) {
	wrongPurpose := logicalSweepInput(t, SweepComplete)
	wrongPurpose.Evaluations[0].purpose = domain.AttemptReduction
	if _, err := NewLogicalCompleteSweep(wrongPurpose); err == nil {
		t.Fatal("non-FINAL_SWEEP evaluation entered logical final completion")
	}

	fixture := newReductionFixture(t, "sweep-reuse/v1")
	current := digest(1)
	baseline := divergentBaseline(t, fixture, current, 500)
	first := neighborFor(t, current, digest(20), 2, 1)
	second := neighborFor(t, current, digest(21), 2, 1)
	// Same offset means distinct stimulus-bound worlds but the same attempt
	// artifacts. Each evaluation is individually nonoverlapping with baseline;
	// the complete sweep must still reject cross-neighbor reuse.
	firstEvaluation := changedEvaluation(t, "reuse:first", fixture, baseline, first, 510, true)
	secondEvaluation := changedEvaluation(t, "reuse:second", fixture, baseline, second, 510, true)
	if _, err := NewLogicalCompleteSweep(LogicalSweepInput{
		State: SweepComplete, Baseline: baseline, CurrentStimulus: current, CurrentMeasure: first.CurrentMeasure(),
		ReducerSetDigest: first.ReducerSetDigest(), EnumeratedNeighbors: []Neighbor{first, second},
		Evaluations: []Evaluation{firstEvaluation, secondEvaluation},
	}); err == nil {
		t.Fatal("cross-neighbor attempt reuse entered logical final completion")
	}
}

func TestIncompleteSweepCannotConstructOneMinimal(t *testing.T) {
	// U1 deliberately has no ONE_MINIMAL type or grade constructor. Even its
	// non-durable logical sweep authority refuses an incomplete state.
	input := logicalSweepInput(t, SweepIncomplete)
	if _, err := NewLogicalCompleteSweep(input); err == nil {
		t.Fatal("incomplete sweep constructed logical completion authority")
	}
}

func FuzzNeighborMeasureAndUnresolvedStaySafe(f *testing.F) {
	f.Add(uint8(2), uint8(1))
	f.Add(uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, currentByte, nextByte uint8) {
		currentMeasure := int(currentByte % 8)
		nextMeasure := int(nextByte % 8)
		if currentMeasure <= nextMeasure {
			return
		}
		neighbor := neighborFor(t, digest(1), digest(2), currentMeasure, nextMeasure)
		comparison, err := neighbor.Measure().Compare(neighbor.CurrentMeasure())
		if err != nil || comparison >= 0 {
			t.Fatal("constructed nondecreasing neighbor")
		}
		fixture := newReductionFixture(t, "fuzz/v1")
		baseline := divergentBaseline(t, fixture, digest(1), 200)
		evaluation, err := NewEvaluation(EvaluationInput{
			ID: "fuzz", Baseline: baseline, Neighbor: neighbor, UnresolvedReason: ReasonNoObservation,
		})
		if err != nil || evaluation.Decision() != Unresolved {
			t.Fatalf("nil observation = %#v, %v", evaluation, err)
		}
	})
}

func TestLogicalCompleteSweepAllowsDurablyMeaningfulEmptyEnumeration(t *testing.T) {
	fixture := newReductionFixture(t, "empty-sweep/v1")
	current := digest(1)
	baseline := divergentBaseline(t, fixture, current, 600)
	definition := digest(690)
	measure, err := NewMeasure(definition, []uint64{0, 0})
	if err != nil {
		t.Fatal(err)
	}
	rule, err := NewReducerRule("test-reducer", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewReducerSet("test", definition, digest(699), []ReducerRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	sweep, err := NewLogicalCompleteSweep(LogicalSweepInput{
		State: SweepComplete, Baseline: baseline, CurrentStimulus: current,
		CurrentMeasure: measure, ReducerSetDigest: set.Digest(),
		EnumeratedNeighbors: []Neighbor{}, Evaluations: []Evaluation{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sweep.NeighborDigests()) != 0 || sweep.CurrentMeasure().Digest() != measure.Digest() {
		t.Fatal("empty completed enumeration lost exact binding")
	}
}
