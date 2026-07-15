package observe

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func digest(number int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", number))
}

func fingerprint(number int) domain.ProjectionFingerprint {
	value, err := domain.NewProjectionFingerprint(projectionBytes(number))
	if err != nil {
		panic(err)
	}
	return value
}

func projectionBinding(number int, adapter domain.AdapterDomain) domain.ProjectionDefinitionBinding {
	channels := []string{"exit", "stderr", "stdout"}
	if adapter == domain.AdapterHTTP {
		channels = []string{"http.body", "http.headers", "http.status"}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: adapter, ImplementationDigest: digest(number), ConfigurationDigest: digest(number + 1),
		AcceptedChannels: channels,
		Operations:       []domain.ProjectionOperationBinding{{Name: "test-projection", RuleDigest: digest(number + 2)}},
		Comparator:       domain.ProjectionComparatorExact, FieldRegistryDigest: digest(number + 3),
	})
	if err != nil {
		panic(err)
	}
	return binding
}

func projectionBytes(number int) []byte {
	value, _ := canon.Integer(int64(number))
	return value.Canonical()
}

func envelope(t *testing.T) domain.ComparisonEnvelope {
	t.Helper()
	value, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: "test/v1",
		Measured: []domain.MeasuredDimension{{
			Name: "operating system", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact,
		}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "operating system"}},
		Uncontrolled:  []string{"scheduler timing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type executionFixture struct {
	envelope domain.ComparisonEnvelope
	plan     domain.WorldPlan
	stimulus domain.Digest
}

func newExecutionFixture(t *testing.T, discoveryRepeats, confirmationRepeats int) executionFixture {
	return newExecutionFixtureWithMarker(t, discoveryRepeats, confirmationRepeats, 8001)
}

func newExecutionFixtureWithMarker(
	t *testing.T,
	discoveryRepeats, confirmationRepeats, planMarker int,
) executionFixture {
	t.Helper()
	envelopeValue := envelope(t)
	config := domain.WorldPlanConfig{
		CandidateSetDigest:          digest(planMarker),
		MaterializationPolicyDigest: digest(8002),
		ComparisonEnvelopeDigest:    envelopeValue.Digest(),
		Adapter: domain.Adapter{
			Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: digest(8004),
		},
		ExecutionShape:       domain.OneCLIInvocation,
		StartArgv:            []string{"fixture"},
		FixtureRecipeDigest:  digest(8005),
		Readiness:            domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest:  digest(8006),
		ProjectionDefinition: projectionBinding(8007, domain.AdapterCLI),
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: discoveryRepeats, ConfirmationRepeats: confirmationRepeats,
		},
		RequiredTools: []domain.RequiredTool{{Name: "fixture", VersionConstraint: "test-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 100, MaterializedBytesPerWorld: 4096,
			SingleBlobBytes: 2048, ProbeMS: 1000, TeardownMS: 1000, StdoutBytes: 4096,
			StderrBytes: 4096, HTTPBodyBytes: 4096, ProposedShrinkStimuli: 10,
			TotalCandidateTrials: 100, ShrinkWallMS: 10000,
		},
	}
	plan, err := domain.NewWorldPlan(config)
	if err != nil {
		t.Fatal(err)
	}
	return executionFixture{envelope: envelopeValue, plan: plan, stimulus: digest(1)}
}

func (f executionFixture) binding(t *testing.T, number int) domain.CandidateExecutionBinding {
	t.Helper()
	binding, err := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
		TreeIdentityDigest:          digest(100000 + number),
		MaterializationPolicyDigest: f.plan.MaterializationPolicyDigest(),
		WorldPlanDigest:             f.plan.Digest(),
		AdapterDigest:               f.plan.AdapterDigest(),
		RunnerDigest:                f.plan.Adapter().RunnerDigest,
		ProjectionDefinitionDigest:  f.plan.ProjectionDefinitionDigest(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func cleanFinalizedAttempt(t *testing.T, artifact domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+artifact.String(), artifact, purpose)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing,
		domain.AttemptStarting,
		domain.AttemptReady,
		domain.AttemptProbing,
		domain.AttemptCapturing,
		domain.AttemptTearingDown,
		domain.AttemptFinalized,
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

func controlledFinalizedAttempt(
	t *testing.T,
	artifact domain.Digest,
	purpose domain.AttemptPurpose,
	primary domain.ControlReason,
	teardown ...domain.ControlReason,
) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+artifact.String(), artifact, purpose)
	if err != nil {
		t.Fatal(err)
	}
	if len(teardown) > 0 {
		for _, state := range []domain.AttemptState{
			domain.AttemptMaterializing,
			domain.AttemptStarting,
			domain.AttemptReady,
			domain.AttemptProbing,
			domain.AttemptCapturing,
		} {
			attempt, err = attempt.Advance(state)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	attempt, err = attempt.Fail(primary)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State() == domain.AttemptTearingDown {
		for _, reason := range teardown {
			attempt, err = attempt.RecordTeardownControl(reason)
			if err != nil {
				t.Fatal(err)
			}
		}
		attempt, err = attempt.Advance(domain.AttemptFinalized)
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

type admittedWorld struct {
	world domain.WorldInstance
	token domain.AdmissionToken
}

func measurementValues(t *testing.T, envelopeValue domain.ComparisonEnvelope) []domain.MeasurementValue {
	t.Helper()
	values := make([]domain.MeasurementValue, 0, len(envelopeValue.Config().Measured))
	for _, dimension := range envelopeValue.Config().Measured {
		value, err := canon.String("fixed:" + dimension.Name)
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, domain.MeasurementValue{Name: dimension.Name, Source: dimension.Source, Value: value})
	}
	return values
}

func worldFor(
	t *testing.T,
	index int,
	binding domain.CandidateExecutionBinding,
	purpose domain.AttemptPurpose,
	ordinal int,
	attemptDigest domain.Digest,
	fixture executionFixture,
) admittedWorld {
	t.Helper()
	world, err := domain.NewWorldInstance(fixture.plan, binding, domain.WorldInstanceConfig{
		StimulusDigest:        fixture.stimulus,
		AttemptArtifactDigest: attemptDigest,
		Purpose:               purpose,
		InstanceNonce:         fmt.Sprintf("nonce:%d", index),
		ScheduleOrdinal:       ordinal,
	})
	if err != nil {
		t.Fatal(err)
	}
	peer := fixture.binding(t, 1)
	if binding.Key() == peer.Key() {
		peer = fixture.binding(t, 2)
	}
	dummy, err := domain.NewWorldInstance(fixture.plan, peer, domain.WorldInstanceConfig{
		StimulusDigest: fixture.stimulus, AttemptArtifactDigest: digest(7000 + index),
		Purpose: purpose, InstanceNonce: fmt.Sprintf("dummy:%d", index), ScheduleOrdinal: 10000 + ordinal,
	})
	if err != nil {
		t.Fatal(err)
	}
	left, err := domain.NewInstanceMeasurements(fixture.envelope, world, measurementValues(t, fixture.envelope))
	if err != nil {
		t.Fatal(err)
	}
	rightValues := measurementValues(t, fixture.envelope)
	right, err := domain.NewInstanceMeasurements(fixture.envelope, dummy, rightValues)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := domain.AssessComparison(fixture.envelope, []domain.InstanceMeasurements{left, right})
	if err != nil {
		t.Fatal(err)
	}
	comparison, ok := assessment.(domain.AdmittedComparison)
	if !ok {
		t.Fatalf("fixture comparison = %T, want admitted", assessment)
	}
	token, err := comparison.AdmissionFor(left)
	if err != nil {
		t.Fatal(err)
	}
	return admittedWorld{world: world, token: token}
}

func eligibleTrial(
	t *testing.T,
	index int,
	value int,
	binding domain.CandidateExecutionBinding,
	purpose domain.AttemptPurpose,
	ordinal int,
	fixture executionFixture,
) TrialFact {
	t.Helper()
	base := 100 + index*10
	attemptDigest := digest(base + 2)
	attempt := cleanFinalizedAttempt(t, attemptDigest, purpose)
	admitted := worldFor(t, index, binding, purpose, ordinal, attemptDigest, fixture)
	capture, err := NewStructuralCapture(admitted.world, attempt, digest(base+3), projectionBytes(value))
	if err != nil {
		t.Fatal(err)
	}
	trial, err := NewCapturedTrial(admitted.world, attempt, admitted.token, capture)
	if err != nil {
		t.Fatal(err)
	}
	return trial
}

func controlledTrial(
	t *testing.T,
	index int,
	binding domain.CandidateExecutionBinding,
	purpose domain.AttemptPurpose,
	ordinal int,
	fixture executionFixture,
	primary domain.ControlReason,
	teardown ...domain.ControlReason,
) TrialFact {
	t.Helper()
	attemptDigest := digest(5000 + index*10)
	attempt := controlledFinalizedAttempt(t, attemptDigest, purpose, primary, teardown...)
	admitted := worldFor(t, 500+index, binding, purpose, ordinal, attemptDigest, fixture)
	trial, err := NewControlledTrial(admitted.world, attempt, admitted.token)
	if err != nil {
		t.Fatal(err)
	}
	return trial
}

// capturedTrialPair creates one concrete two-candidate comparison matrix. The
// returned trials therefore share the exact per-matrix admission digest rather
// than independently recreating lookalike admissions.
func capturedTrialPair(
	t *testing.T,
	index int,
	leftValue, rightValue int,
	purpose domain.AttemptPurpose,
	fixture executionFixture,
) (TrialFact, TrialFact) {
	t.Helper()
	bindings := []domain.CandidateExecutionBinding{fixture.binding(t, 1), fixture.binding(t, 2)}
	attemptDigests := []domain.Digest{digest(20000 + index*20), digest(20001 + index*20)}
	ordinals := []int{index * 2, index*2 + 1}
	worlds := make([]domain.WorldInstance, 2)
	measurements := make([]domain.InstanceMeasurements, 2)
	for candidateIndex := range bindings {
		world, err := domain.NewWorldInstance(fixture.plan, bindings[candidateIndex], domain.WorldInstanceConfig{
			StimulusDigest: fixture.stimulus, AttemptArtifactDigest: attemptDigests[candidateIndex],
			Purpose: purpose, InstanceNonce: fmt.Sprintf("matrix:%d:%d", index, candidateIndex),
			ScheduleOrdinal: ordinals[candidateIndex],
		})
		if err != nil {
			t.Fatal(err)
		}
		worlds[candidateIndex] = world
		row, err := domain.NewInstanceMeasurements(fixture.envelope, world, measurementValues(t, fixture.envelope))
		if err != nil {
			t.Fatal(err)
		}
		measurements[candidateIndex] = row
	}
	assessment, err := domain.AssessComparison(fixture.envelope, measurements)
	if err != nil {
		t.Fatal(err)
	}
	comparison, ok := assessment.(domain.AdmittedComparison)
	if !ok {
		t.Fatalf("matrix comparison = %T, want admitted", assessment)
	}
	values := []int{leftValue, rightValue}
	trials := make([]TrialFact, 2)
	for candidateIndex := range bindings {
		token, tokenErr := comparison.AdmissionFor(measurements[candidateIndex])
		if tokenErr != nil {
			t.Fatal(tokenErr)
		}
		attempt := cleanFinalizedAttempt(t, attemptDigests[candidateIndex], purpose)
		capture, captureErr := NewStructuralCapture(
			worlds[candidateIndex], attempt,
			digest(21000+index*20+candidateIndex),
			projectionBytes(values[candidateIndex]),
		)
		if captureErr != nil {
			t.Fatal(captureErr)
		}
		trial, trialErr := NewCapturedTrial(worlds[candidateIndex], attempt, token, capture)
		if trialErr != nil {
			t.Fatal(trialErr)
		}
		trials[candidateIndex] = trial
	}
	return trials[0], trials[1]
}

func inputFor(
	t *testing.T,
	trials ...TrialFact,
) BatchInput {
	t.Helper()
	return BatchInput{Trials: trials}
}

func TestControlFailureNeverBecomesEligible(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	fact := controlledTrial(t, 0, fixture.binding(t, 1), domain.AttemptDiscovery, 0, fixture, domain.ControlTimeout)
	eligibility := Eligible(fact)
	if eligibility.IsEligible() {
		t.Fatal("detected control failure became eligible behavior")
	}
	reason, ok := eligibility.Reason()
	if !ok || reason != domain.ControlTimeout {
		t.Fatalf("reason = %q, want TIMEOUT", reason)
	}
}

func TestCapturedAndControlAreMutuallyExclusive(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	attemptDigest := digest(777)
	controlled := controlledFinalizedAttempt(t, attemptDigest, domain.AttemptDiscovery, domain.ControlTimeout)
	world := worldFor(t, 777, binding, domain.AttemptDiscovery, 0, attemptDigest, fixture)
	capture, err := NewStructuralCapture(world.world, controlled, digest(778), projectionBytes(10))
	if err != nil {
		t.Fatal(err)
	}
	if capture.ProjectionResultDigest().String() == capture.ProjectionFingerprint().String() {
		t.Fatal("lineage-bound projection result collapsed into the byte-equality fingerprint")
	}
	if _, err := NewCapturedTrial(world.world, controlled, world.token, capture); err == nil {
		t.Fatal("constructed contradictory captured-plus-control trial")
	}
	clean := cleanFinalizedAttempt(t, digest(780), domain.AttemptDiscovery)
	cleanWorld := worldFor(t, 780, binding, domain.AttemptDiscovery, 0, digest(780), fixture)
	if _, err := NewControlledTrial(cleanWorld.world, clean, cleanWorld.token); err == nil {
		t.Fatal("constructed control trial without a control")
	}
	if _, err := NewStructuralCapture(cleanWorld.world, clean, digest(781), []byte(" 10")); err == nil {
		t.Fatal("noncanonical projection entered structural capture")
	}
	oversizedProjection := []byte(`"` + strings.Repeat("x", maxProjectionResultCanonicalBytes) + `"`)
	if _, err := NewStructuralCapture(cleanWorld.world, clean, digest(782), oversizedProjection); err == nil {
		t.Fatal("projection result consumed the metadata headroom reserved by the canonical profile")
	}
	otherAttempt := cleanFinalizedAttempt(t, digest(783), domain.AttemptDiscovery)
	otherWorld := worldFor(t, 783, binding, domain.AttemptDiscovery, 1, digest(783), fixture)
	if _, err := NewCapturedTrial(otherWorld.world, otherAttempt, otherWorld.token, capture); err == nil {
		t.Fatal("capture bound to another world/attempt was replayed as fresh evidence")
	}
	otherCapture, err := NewStructuralCapture(otherWorld.world, otherAttempt, digest(784), projectionBytes(10))
	if err != nil {
		t.Fatal(err)
	}
	if otherCapture.ProjectionFingerprint() != capture.ProjectionFingerprint() {
		t.Fatal("byte-identical canonical projections did not retain one equality fingerprint")
	}
	if otherCapture.ProjectionResultDigest() == capture.ProjectionResultDigest() {
		t.Fatal("distinct world/attempt/observation lineage retained one projection-result identity")
	}
	projection := capture.CanonicalProjection()
	projection[0] = '9'
	if capture.ProjectionFingerprint() != fingerprint(10) || string(capture.CanonicalProjection()) != "10" {
		t.Fatal("structural capture exposed mutable canonical projection")
	}
}

func TestClassifyObservedStableKeepsExactCountsAndLabel(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	value := 10
	batch, err := Classify(inputFor(t,
		eligibleTrial(t, 0, value, binding, domain.AttemptDiscovery, 0, fixture),
		eligibleTrial(t, 1, value, binding, domain.AttemptDiscovery, 1, fixture),
		eligibleTrial(t, 2, value, binding, domain.AttemptDiscovery, 2, fixture),
	))
	if err != nil {
		t.Fatal(err)
	}
	classification := batch.Classification()
	if classification.Status() != ObservedStable || classification.EligibleTrials() != 3 || classification.RequiredTrials() != 3 {
		t.Fatalf("classification = %#v", classification)
	}
	wantLabel := "OBSERVED_STABLE(3/3," + fingerprint(value).String() + ")"
	if classification.BoundedLabel() != wantLabel {
		t.Fatalf("label = %q, want %q", classification.BoundedLabel(), wantLabel)
	}
	bytes := batch.CanonicalBytes()
	bytes[0] = 'x'
	if len(batch.CanonicalBytes()) == 0 || batch.CanonicalBytes()[0] == 'x' {
		t.Fatal("canonical byte accessor exposed mutable storage")
	}
}

func TestClassifyUnstableKeepsExactHistogram(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	left := 10
	right := 11
	input := inputFor(t,
		eligibleTrial(t, 0, right, binding, domain.AttemptDiscovery, 0, fixture),
		eligibleTrial(t, 1, left, binding, domain.AttemptDiscovery, 1, fixture),
	)
	batch, err := Classify(input)
	if err != nil {
		t.Fatal(err)
	}
	classification := batch.Classification()
	if classification.Status() != Unstable || classification.EligibleTrials() != 2 {
		t.Fatalf("classification = %#v", classification)
	}
	histogram := classification.Histogram()
	if len(histogram) != 2 || histogram[0].Count != 1 || histogram[1].Count != 1 ||
		histogram[0].Fingerprint.String() > histogram[1].Fingerprint.String() {
		t.Fatalf("histogram = %#v", histogram)
	}
}

func TestClassifyControlTakesPrecedenceAndRetainsCleanupControls(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	control := controlledTrial(
		t, 2, binding, domain.AttemptDiscovery, 2, fixture,
		domain.ControlTimeout, domain.ControlTeardownError, domain.ControlOrphanRisk,
	)
	batch, err := Classify(inputFor(t,
		eligibleTrial(t, 0, 10, binding, domain.AttemptDiscovery, 0, fixture),
		eligibleTrial(t, 1, 11, binding, domain.AttemptDiscovery, 1, fixture),
		control,
	))
	if err != nil {
		t.Fatal(err)
	}
	classification := batch.Classification()
	if classification.Status() != Uncomparable || len(classification.Reasons()) != 3 {
		t.Fatalf("classification = %#v", classification)
	}
}

func TestClassifyRefusesZeroExcessReuseAndLineageMismatch(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	if _, err := Classify(inputFor(t)); err == nil {
		t.Fatal("zero-evidence batch was accepted")
	}

	trials := make([]TrialFact, 4)
	for index := range trials {
		trials[index] = eligibleTrial(t, index, 10+index%2, binding, domain.AttemptDiscovery, index, fixture)
	}
	excess := inputFor(t, trials...)
	if _, err := Classify(excess); err == nil {
		t.Fatal("excess trials were classified instead of requiring a new batch")
	}

	reusedTrial := eligibleTrial(t, 10, 10, binding, domain.AttemptDiscovery, 0, fixture)
	reused := inputFor(t, reusedTrial, reusedTrial)
	if _, err := Classify(reused); err == nil {
		t.Fatal("reused attempt/world evidence was accepted")
	}

	wrongCandidateTrial := eligibleTrial(t, 20, 10, fixture.binding(t, 2), domain.AttemptDiscovery, 1, fixture)
	if _, err := Classify(inputFor(t, reusedTrial, wrongCandidateTrial)); err == nil {
		t.Fatal("trial from another candidate was admitted")
	}
	wrongPhaseTrial := eligibleTrial(t, 21, 10, binding, domain.AttemptReduction, 1, fixture)
	if _, err := Classify(inputFor(t, reusedTrial, wrongPhaseTrial)); err == nil {
		t.Fatal("trial from another phase was admitted")
	}

	firstObservation := eligibleTrial(t, 30, 10, binding, domain.AttemptDiscovery, 0, fixture)
	replayedObservation := eligibleTrial(t, 31, 10, binding, domain.AttemptDiscovery, 1, fixture)
	replayedObservation.capture.observationDigest = firstObservation.capture.observationDigest
	if _, err := Classify(inputFor(t, firstObservation, replayedObservation)); err == nil {
		t.Fatal("one captured observation was counted as two fresh trials")
	}
}

func TestSharedComparisonMatricesProduceMatchingAdmissionSets(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	leftTrials := make([]TrialFact, 0, 3)
	rightTrials := make([]TrialFact, 0, 3)
	for repetition := 0; repetition < 3; repetition++ {
		left, right := capturedTrialPair(t, repetition, 10, 11, domain.AttemptDiscovery, fixture)
		leftTrials = append(leftTrials, left)
		rightTrials = append(rightTrials, right)
	}
	leftBatch, err := Classify(inputFor(t, leftTrials...))
	if err != nil {
		t.Fatal(err)
	}
	rightBatch, err := Classify(inputFor(t, rightTrials...))
	if err != nil {
		t.Fatal(err)
	}
	leftAdmissions := leftBatch.AdmissionDigests()
	rightAdmissions := rightBatch.AdmissionDigests()
	if len(leftAdmissions) != 3 || len(rightAdmissions) != 3 {
		t.Fatalf("admission set lengths = %d / %d, want 3 / 3", len(leftAdmissions), len(rightAdmissions))
	}
	for index := range leftAdmissions {
		if leftAdmissions[index] != rightAdmissions[index] {
			t.Fatalf("matrix %d admission mismatch: %s / %s", index, leftAdmissions[index], rightAdmissions[index])
		}
	}
}

func TestBatchIdentityCannotRelabelIdenticalTrialsToAnotherStimulus(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	trials := []TrialFact{
		eligibleTrial(t, 40, 10, binding, domain.AttemptDiscovery, 0, fixture),
		eligibleTrial(t, 41, 10, binding, domain.AttemptDiscovery, 1, fixture),
		eligibleTrial(t, 42, 10, binding, domain.AttemptDiscovery, 2, fixture),
	}
	first, err := Classify(inputFor(t, trials...))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Classify(inputFor(t, trials...))
	if err != nil {
		t.Fatal(err)
	}
	if first.StimulusDigest() != fixture.stimulus || second.StimulusDigest() != fixture.stimulus ||
		first.Digest() != second.Digest() {
		t.Fatal("identical trial facts did not retain one deterministic world-bound stimulus identity")
	}
	otherStimulus := fixture
	otherStimulus.stimulus = digest(2)
	foreign := eligibleTrial(t, 43, 10, binding, domain.AttemptDiscovery, 1, otherStimulus)
	if _, err := Classify(inputFor(t, trials[0], foreign)); err == nil {
		t.Fatal("mixed-stimulus trials were accepted as one batch")
	}
}

func TestParsedBareCandidateKeyCannotAllocateWorldEvidence(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	parsed, err := domain.ParseCandidateExecutionKey(binding.Key().String())
	if err != nil || parsed != binding.Key() {
		t.Fatalf("parsed display reference = %v, %v", parsed, err)
	}
	// NewWorldInstance accepts CandidateExecutionBinding, not the parsed key.
	// A zero binding stands in for the missing retained identity and must refuse.
	if _, err := domain.NewWorldInstance(fixture.plan, domain.CandidateExecutionBinding{}, domain.WorldInstanceConfig{
		StimulusDigest: fixture.stimulus, AttemptArtifactDigest: digest(30001),
		Purpose: domain.AttemptDiscovery, InstanceNonce: "bare-key", ScheduleOrdinal: 0,
	}); err == nil {
		t.Fatal("world evidence was allocated without retained candidate execution identity")
	}
}

func TestPlanEnvelopeMismatchRefusesMeasurements(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	otherEnvelope, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: "other/v1",
		Measured: []domain.MeasuredDimension{{
			Name: "operating system", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact,
		}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "operating system"}},
		Uncontrolled:  []string{"scheduler timing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	world, err := domain.NewWorldInstance(fixture.plan, fixture.binding(t, 1), domain.WorldInstanceConfig{
		StimulusDigest: fixture.stimulus, AttemptArtifactDigest: digest(30002),
		Purpose: domain.AttemptDiscovery, InstanceNonce: "envelope-mismatch", ScheduleOrdinal: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewInstanceMeasurements(otherEnvelope, world, measurementValues(t, otherEnvelope)); err == nil {
		t.Fatal("measurements used an envelope other than the one bound by the world plan")
	}
}

func TestConfirmationRepeatRequirementCannotBeDowngraded(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 5)
	binding := fixture.binding(t, 1)
	one := eligibleTrial(t, 50, 10, binding, domain.AttemptConfirmation, 0, fixture)
	partial, err := Classify(inputFor(t, one))
	if err != nil {
		t.Fatal(err)
	}
	if partial.RequiredFreshTrials() != 5 || partial.Classification().RequiredTrials() != 5 ||
		partial.Classification().Status() != Incomplete {
		t.Fatalf("1-of-5 confirmation = %#v", partial.Classification())
	}
	trials := []TrialFact{one}
	for index := 1; index < 5; index++ {
		trials = append(trials, eligibleTrial(t, 50+index, 10, binding, domain.AttemptConfirmation, index, fixture))
	}
	complete, err := Classify(inputFor(t, trials...))
	if err != nil {
		t.Fatal(err)
	}
	if complete.Classification().Status() != ObservedStable || complete.Classification().RequiredTrials() != 5 {
		t.Fatalf("5-of-5 confirmation = %#v", complete.Classification())
	}
}

func TestClassifyRefusesCrossPlanPurposeStimulusAndCandidateBindings(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	base := eligibleTrial(t, 60, 10, binding, domain.AttemptDiscovery, 0, fixture)

	otherPlan := newExecutionFixtureWithMarker(t, 3, 3, 8101)
	foreignPlan := eligibleTrial(t, 61, 10, otherPlan.binding(t, 1), domain.AttemptDiscovery, 1, otherPlan)
	if _, err := Classify(inputFor(t, base, foreignPlan)); err == nil {
		t.Fatal("cross-plan trials were accepted as one batch")
	}
	if _, err := domain.NewWorldInstance(fixture.plan, otherPlan.binding(t, 1), domain.WorldInstanceConfig{
		StimulusDigest: fixture.stimulus, AttemptArtifactDigest: digest(30003),
		Purpose: domain.AttemptDiscovery, InstanceNonce: "plan-binding-mismatch", ScheduleOrdinal: 1,
	}); err == nil {
		t.Fatal("candidate identity bound to another plan allocated world evidence")
	}

	foreignPurpose := eligibleTrial(t, 62, 10, binding, domain.AttemptReduction, 1, fixture)
	if _, err := Classify(inputFor(t, base, foreignPurpose)); err == nil {
		t.Fatal("cross-purpose trials were accepted as one batch")
	}

	otherStimulus := fixture
	otherStimulus.stimulus = digest(2)
	foreignStimulus := eligibleTrial(t, 63, 10, binding, domain.AttemptDiscovery, 1, otherStimulus)
	if _, err := Classify(inputFor(t, base, foreignStimulus)); err == nil {
		t.Fatal("cross-stimulus trials were accepted as one batch")
	}

	left, right := capturedTrialPair(t, 32, 10, 10, domain.AttemptDiscovery, fixture)
	if _, err := Classify(inputFor(t, left, right)); err == nil {
		t.Fatal("different candidates from one shared matrix were accepted as one candidate batch")
	}
}

func TestEveryControlReasonIsPreAdmissionOrPostAdmissionIneligible(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	binding := fixture.binding(t, 1)
	tests := []struct {
		name   string
		reason domain.ControlReason
		route  string
	}{
		{"materialization", domain.ControlMaterializationError, "post-admission-primary"},
		{"setup", domain.ControlSetupError, "post-admission-primary"},
		{"start", domain.ControlStartError, "post-admission-primary"},
		{"readiness", domain.ControlReadinessError, "post-admission-primary"},
		{"probe-transport", domain.ControlProbeTransportError, "post-admission-primary"},
		{"timeout", domain.ControlTimeout, "post-admission-primary"},
		{"cancelled", domain.ControlCancelled, "post-admission-primary"},
		{"output-limit", domain.ControlOutputLimit, "post-admission-primary"},
		{"projection", domain.ControlProjectionRejected, "post-admission-primary"},
		{"orphan-risk", domain.ControlOrphanRisk, "post-admission-teardown"},
		{"teardown", domain.ControlTeardownError, "post-admission-teardown"},
		{"envelope", domain.ControlEnvelopeRejected, "pre-admission"},
		{"git-mode", domain.ControlUnsupportedGitMode, "post-admission-primary"},
		{"missing-object", domain.ControlMissingObject, "post-admission-primary"},
		{"budget", domain.ControlBudgetExhausted, "post-admission-primary"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.reason.Valid() {
				t.Fatalf("table contains invalid control %q", test.reason)
			}
			if test.route == "pre-admission" {
				attemptDigest := digest(40000 + index)
				attempt := controlledFinalizedAttempt(t, attemptDigest, domain.AttemptDiscovery, test.reason)
				admitted := worldFor(t, 900+index, binding, domain.AttemptDiscovery, index, attemptDigest, fixture)
				if _, err := NewControlledTrial(admitted.world, attempt, admitted.token); err == nil {
					t.Fatal("pre-admission envelope rejection manufactured a TrialFact")
				}
				return
			}

			var trial TrialFact
			if test.route == "post-admission-teardown" {
				trial = controlledTrial(
					t, 900+index, binding, domain.AttemptDiscovery, index, fixture,
					domain.ControlTimeout, test.reason,
				)
			} else {
				trial = controlledTrial(t, 900+index, binding, domain.AttemptDiscovery, index, fixture, test.reason)
			}
			eligibility := Eligible(trial)
			if eligibility.IsEligible() || !containsReason(eligibility.Reasons(), test.reason) {
				t.Fatalf("control eligibility = %#v, want ineligible with %s", eligibility, test.reason)
			}
			batch, err := Classify(inputFor(t, trial))
			if err != nil {
				t.Fatal(err)
			}
			status := batch.Classification().Status()
			if status != Uncomparable || status == ObservedStable || status == Unstable {
				t.Fatalf("control reached projection classification %s", status)
			}
		})
	}
}

func containsReason(reasons []domain.ControlReason, target domain.ControlReason) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func FuzzBatchControlsNeverPromote(f *testing.F) {
	f.Add(uint8(3), uint8(0))
	f.Add(uint8(5), uint8(1))
	f.Fuzz(func(t *testing.T, countByte, reasonByte uint8) {
		count := int(countByte%5) + 1
		reasons := []domain.ControlReason{domain.ControlTimeout, domain.ControlCancelled, domain.ControlOutputLimit}
		fixture := newExecutionFixture(t, count, count)
		binding := fixture.binding(t, 1)
		trials := make([]TrialFact, count)
		for index := range trials {
			trials[index] = controlledTrial(
				t, 1000+index, binding, domain.AttemptDiscovery, index, fixture,
				reasons[int(reasonByte)%len(reasons)],
			)
		}
		input := inputFor(t, trials...)
		batch, err := Classify(input)
		if err != nil {
			return
		}
		status := batch.Classification().Status()
		if status == ObservedStable || status == Unstable {
			t.Fatalf("controls promoted to %s", status)
		}
	})
}
