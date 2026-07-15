package compare

import (
	"fmt"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

var testStimulus = digestNumber(1)

func digestNumber(number int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", number))
}

func projectionNumber(number int) domain.ProjectionFingerprint {
	value, err := domain.NewProjectionFingerprint(projectionBytes(number))
	if err != nil {
		panic(err)
	}
	return value
}

func projectionBytes(number int) []byte {
	value, _ := canon.Integer(int64(number))
	return value.Canonical()
}

func comparisonEnvelope(t *testing.T) domain.ComparisonEnvelope {
	t.Helper()
	value, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version:       "compare/v1",
		Measured:      []domain.MeasuredDimension{{Name: "os", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "os"}},
		Uncontrolled:  []string{"scheduler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func comparisonPlan(t *testing.T, envelopeValue domain.ComparisonEnvelope, salt int) domain.WorldPlan {
	t.Helper()
	base := 1000 + salt*100
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          digestNumber(base + 1),
		MaterializationPolicyDigest: digestNumber(base + 2),
		ComparisonEnvelopeDigest:    envelopeValue.Digest(),
		Adapter: domain.Adapter{
			Domain: domain.AdapterHTTP, AdapterVersion: "http/v1", RunnerDigest: digestNumber(base + 3),
		},
		ExecutionShape:      domain.OneLoopbackHTTPRequest,
		StartArgv:           []string{"node", "fixture/server.mjs"},
		SetupArgv:           []string{},
		Environment:         []domain.EnvironmentEntry{{Name: "NODE_NO_WARNINGS", Value: "1"}, {Name: "LANG", Value: "C"}},
		SecretSlots:         []domain.SecretSlot{},
		FixtureRecipeDigest: digestNumber(base + 4),
		Readiness:           domain.Readiness{Kind: domain.FixtureOwnedReadiness, SignalName: "ready-byte"},
		CapturePolicyDigest: digestNumber(base + 5),
		ProjectionDefinition: func() domain.ProjectionDefinitionBinding {
			binding, bindingErr := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
				AdapterDomain: domain.AdapterHTTP, ImplementationDigest: digestNumber(base + 6), ConfigurationDigest: digestNumber(base + 7),
				AcceptedChannels: []string{"http.body", "http.headers", "http.status"},
				Operations:       []domain.ProjectionOperationBinding{{Name: "test-projection", RuleDigest: digestNumber(base + 8)}},
				Comparator:       domain.ProjectionComparatorExact, FieldRegistryDigest: digestNumber(base + 9),
			})
			if bindingErr != nil {
				t.Fatal(bindingErr)
			}
			return binding
		}(),
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 3, ConfirmationRepeats: 3},
		RequiredTools:  []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount:            4,
			MaterializedEntryCount:    25000,
			MaterializedBytesPerWorld: 268435456,
			SingleBlobBytes:           33554432,
			ReadinessMS:               5000,
			ProbeMS:                   3000,
			TeardownMS:                3000,
			StdoutBytes:               1048576,
			StderrBytes:               1048576,
			HTTPBodyBytes:             1048576,
			ProposedShrinkStimuli:     40,
			TotalCandidateTrials:      300,
			ShrinkWallMS:              600000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

type comparisonFixture struct {
	envelope domain.ComparisonEnvelope
	plan     domain.WorldPlan
	bindings map[int]domain.CandidateExecutionBinding
}

func newComparisonFixture(t *testing.T, candidateNumbers ...int) comparisonFixture {
	t.Helper()
	return newComparisonFixtureWithPlanSalt(t, 0, candidateNumbers...)
}

func newComparisonFixtureWithPlanSalt(t *testing.T, salt int, candidateNumbers ...int) comparisonFixture {
	t.Helper()
	envelopeValue := comparisonEnvelope(t)
	plan := comparisonPlan(t, envelopeValue, salt)
	bindings := make(map[int]domain.CandidateExecutionBinding, len(candidateNumbers))
	for _, number := range candidateNumbers {
		binding, err := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
			TreeIdentityDigest:          digestNumber(10000 + number),
			MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
			WorldPlanDigest:             plan.Digest(),
			AdapterDigest:               plan.AdapterDigest(),
			RunnerDigest:                plan.Adapter().RunnerDigest,
			ProjectionDefinitionDigest:  plan.ProjectionDefinitionDigest(),
		})
		if err != nil {
			t.Fatal(err)
		}
		bindings[number] = binding
	}
	return comparisonFixture{envelope: envelopeValue, plan: plan, bindings: bindings}
}

func (f comparisonFixture) binding(t *testing.T, candidateNumber int) domain.CandidateExecutionBinding {
	t.Helper()
	binding, ok := f.bindings[candidateNumber]
	if !ok {
		t.Fatalf("missing candidate fixture %d", candidateNumber)
	}
	return binding
}

func (f comparisonFixture) key(t *testing.T, candidateNumber int) domain.CandidateExecutionKey {
	t.Helper()
	return f.binding(t, candidateNumber).Key()
}

func finalizedClean(t *testing.T, attemptDigest domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
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

func finalizedControl(t *testing.T, attemptDigest domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+attemptDigest.String(), attemptDigest, purpose)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = attempt.Fail(domain.ControlTimeout)
	if err != nil {
		t.Fatal(err)
	}
	result, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func measuredValues(t *testing.T, envelopeValue domain.ComparisonEnvelope, basisValue int) []domain.MeasurementValue {
	t.Helper()
	values := make([]domain.MeasurementValue, 0, len(envelopeValue.Config().Measured))
	for _, dimension := range envelopeValue.Config().Measured {
		value, err := canon.String(fmt.Sprintf("basis:%d:%s", basisValue, dimension.Name))
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, domain.MeasurementValue{Name: dimension.Name, Source: dimension.Source, Value: value})
	}
	return values
}

type candidateBatchSpec struct {
	candidateNumber int
	status          observe.BatchStatus
	projection      int
}

func observed(candidateNumber, projection int) candidateBatchSpec {
	return candidateBatchSpec{candidateNumber: candidateNumber, status: observe.ObservedStable, projection: projection}
}

func unstable(candidateNumber, projection int) candidateBatchSpec {
	return candidateBatchSpec{candidateNumber: candidateNumber, status: observe.Unstable, projection: projection}
}

func uncomparable(candidateNumber int) candidateBatchSpec {
	return candidateBatchSpec{candidateNumber: candidateNumber, status: observe.Uncomparable}
}

type trialCoordinate struct {
	candidateNumber int
	repetition      int
}

type batchRunOptions struct {
	evidenceSalt        int
	basisValue          int
	attemptOverride     map[trialCoordinate]domain.Digest
	observationOverride map[trialCoordinate]domain.Digest
}

type candidateBatchSet struct {
	roster   []domain.CandidateExecutionKey
	batches  []observe.StableBatch
	byNumber map[int]observe.StableBatch
}

func (s candidateBatchSet) batch(t *testing.T, candidateNumber int) observe.StableBatch {
	t.Helper()
	batch, ok := s.byNumber[candidateNumber]
	if !ok {
		t.Fatalf("missing batch fixture %d", candidateNumber)
	}
	return batch
}

func buildCandidateBatches(
	t *testing.T,
	fixture comparisonFixture,
	stimulus domain.Digest,
	purpose domain.AttemptPurpose,
	options batchRunOptions,
	specs ...candidateBatchSpec,
) candidateBatchSet {
	t.Helper()
	if len(specs) < 2 || len(specs) > 4 {
		t.Fatalf("fixture roster has %d candidates, want 2..4", len(specs))
	}
	trials := make(map[int][]observe.TrialFact, len(specs))
	roster := make([]domain.CandidateExecutionKey, len(specs))
	for index, spec := range specs {
		roster[index] = fixture.key(t, spec.candidateNumber)
	}

	for repetition := 0; repetition < 3; repetition++ {
		worlds := make([]domain.WorldInstance, len(specs))
		attempts := make([]domain.FinalizedAttempt, len(specs))
		measurements := make([]domain.InstanceMeasurements, len(specs))
		for index, spec := range specs {
			coordinate := trialCoordinate{candidateNumber: spec.candidateNumber, repetition: repetition}
			attemptDigest, overridden := options.attemptOverride[coordinate]
			if !overridden {
				attemptDigest = digestNumber(1000000 + options.evidenceSalt*10000 + repetition*100 + index*10 + spec.candidateNumber)
			}
			if spec.status == observe.Uncomparable {
				attempts[index] = finalizedControl(t, attemptDigest, purpose)
			} else {
				attempts[index] = finalizedClean(t, attemptDigest, purpose)
			}
			world, err := domain.NewWorldInstance(fixture.plan, fixture.binding(t, spec.candidateNumber), domain.WorldInstanceConfig{
				StimulusDigest:        stimulus,
				AttemptArtifactDigest: attemptDigest,
				Purpose:               purpose,
				InstanceNonce:         fmt.Sprintf("run:%d:repeat:%d:candidate:%d", options.evidenceSalt, repetition, spec.candidateNumber),
				ScheduleOrdinal:       options.evidenceSalt*1000 + repetition*10 + index,
			})
			if err != nil {
				t.Fatal(err)
			}
			row, err := domain.NewInstanceMeasurements(fixture.envelope, world, measuredValues(t, fixture.envelope, options.basisValue))
			if err != nil {
				t.Fatal(err)
			}
			worlds[index] = world
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
		for index, spec := range specs {
			token, err := admitted.AdmissionFor(measurements[index])
			if err != nil {
				t.Fatal(err)
			}
			var trial observe.TrialFact
			if spec.status == observe.Uncomparable {
				trial, err = observe.NewControlledTrial(worlds[index], attempts[index], token)
			} else {
				projection := spec.projection
				if spec.status == observe.Unstable && repetition == 1 {
					projection++
				}
				captureBase := 2000000 + options.evidenceSalt*10000 + repetition*100 + index*10 + spec.candidateNumber
				observationDigest, overridden := options.observationOverride[trialCoordinate{
					candidateNumber: spec.candidateNumber, repetition: repetition,
				}]
				if !overridden {
					observationDigest = digestNumber(captureBase + 1)
				}
				capture, captureErr := observe.NewStructuralCapture(
					worlds[index], attempts[index], observationDigest, projectionBytes(projection),
				)
				if captureErr != nil {
					t.Fatal(captureErr)
				}
				trial, err = observe.NewCapturedTrial(worlds[index], attempts[index], token, capture)
			}
			if err != nil {
				t.Fatal(err)
			}
			trials[spec.candidateNumber] = append(trials[spec.candidateNumber], trial)
		}
	}

	result := candidateBatchSet{
		roster: roster, batches: make([]observe.StableBatch, len(specs)), byNumber: make(map[int]observe.StableBatch, len(specs)),
	}
	for index, spec := range specs {
		batch, err := observe.Classify(observe.BatchInput{Trials: trials[spec.candidateNumber]})
		if err != nil {
			t.Fatal(err)
		}
		if batch.Classification().Status() != spec.status {
			t.Fatalf("candidate %d status = %s, want %s", spec.candidateNumber, batch.Classification().Status(), spec.status)
		}
		result.batches[index] = batch
		result.byNumber[spec.candidateNumber] = batch
	}
	for index := 1; index < len(result.batches); index++ {
		if !sameDigestList(result.batches[0].AdmissionDigests(), result.batches[index].AdmissionDigests()) {
			t.Fatalf("candidate batches do not share the same admitted matrix sequence")
		}
	}
	return result
}

func mapFrom(
	t *testing.T,
	stimulus domain.Digest,
	envelopeValue domain.ComparisonEnvelope,
	roster []domain.CandidateExecutionKey,
	batches ...observe.StableBatch,
) CandidateOutcomeMap {
	t.Helper()
	result, err := NewCandidateOutcomeMap(stimulus, envelopeValue.Digest(), roster, batches)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requireErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	construction, ok := err.(*domain.Error)
	if !ok || construction.Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func TestOutcomeMapCandidatePermutationPreservesArtifactIdentity(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3, 4)
	set := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 1},
		observed(1, 10), observed(2, 11), observed(3, 12),
	)
	a, b, c := set.batch(t, 1), set.batch(t, 2), set.batch(t, 3)
	left := mapFrom(t, testStimulus, fixture.envelope, set.roster, a, b, c)
	right := mapFrom(t, testStimulus, fixture.envelope,
		[]domain.CandidateExecutionKey{set.roster[2], set.roster[0], set.roster[1]}, c, a, b,
	)
	if left.ArtifactDigest() != right.ArtifactDigest() || left.PreservationDigest() != right.PreservationDigest() {
		t.Fatal("candidate input permutation changed sorted map identity")
	}
	entries := left.Entries()
	entries[0].CandidateKey = fixture.key(t, 4)
	if left.Entries()[0].CandidateKey == fixture.key(t, 4) {
		t.Fatal("entry accessor exposed mutable storage")
	}
}

func TestCandidateKeyIsPartOfLabeledMapIdentity(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3, 4)
	leftSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 2},
		observed(1, 10), observed(2, 11),
	)
	rightSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 3},
		observed(3, 10), observed(4, 11),
	)
	left := mapFrom(t, testStimulus, fixture.envelope, leftSet.roster, leftSet.batches...)
	right := mapFrom(t, testStimulus, fixture.envelope, rightSet.roster, rightSet.batches...)
	if left.PreservationDigest() == right.PreservationDigest() || SamePreservationMap(left, right) {
		t.Fatal("candidate keys were dropped from preservation identity")
	}
}

func TestPreservationUsesLabeledMapNotPartitionShape(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	leftSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 4},
		observed(1, 10), observed(2, 11),
	)
	rightStimulus := digestNumber(2)
	rightSet := buildCandidateBatches(t, fixture, rightStimulus, domain.AttemptReduction, batchRunOptions{evidenceSalt: 5},
		observed(1, 11), observed(2, 10),
	)
	left := mapFrom(t, testStimulus, fixture.envelope, leftSet.roster, leftSet.batches...)
	right := mapFrom(t, rightStimulus, fixture.envelope, rightSet.roster, rightSet.batches...)
	if !samePartitionShape(left, right) {
		t.Fatal("test fixture does not preserve partition shape")
	}
	if SamePreservationMap(left, right) {
		t.Fatal("partition shape replaced exact labeled-map equality")
	}
}

func TestExpectedRosterRejectsOmissionAdditionAndDuplication(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3)
	set := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 6},
		observed(1, 10), observed(2, 11), observed(3, 12),
	)
	a, b, c := set.batch(t, 1), set.batch(t, 2), set.batch(t, 3)
	if _, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, []observe.StableBatch{a, b}); err == nil {
		t.Fatal("omitted candidate vanished from map")
	}
	if _, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster[:2], []observe.StableBatch{a, c}); err == nil {
		t.Fatal("unexpected candidate entered map")
	}
	if _, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, []observe.StableBatch{a, b, b}); err == nil {
		t.Fatal("duplicate candidate batch entered map")
	}
}

func TestCandidateOutcomeMapAndDivergentBaselineAreSeparateAuthorities(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3)
	equalSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 7},
		observed(1, 10), observed(2, 10), unstable(3, 20),
	)
	equal := mapFrom(t, testStimulus, fixture.envelope, equalSet.roster, equalSet.batches...)
	if equal.Divergence() || equal.DistinctProjectionCount() != 1 {
		t.Fatal("equal candidate map reported divergence")
	}
	if _, err := RequireDivergence(equal); err == nil {
		t.Fatal("equal map constructed divergent authority")
	}
	divergentSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 8},
		observed(1, 10), observed(2, 11), unstable(3, 20),
	)
	divergent := mapFrom(t, testStimulus, fixture.envelope, divergentSet.roster, divergentSet.batches...)
	if baseline, err := RequireDivergence(divergent); err != nil || !baseline.Valid() {
		t.Fatalf("divergent baseline = %#v, err = %v", baseline, err)
	}
}

func TestEligibilityChangeBlocksPreservationComparability(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3)
	baselineSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 9},
		observed(1, 10), observed(2, 11), unstable(3, 20),
	)
	baseline := mapFrom(t, testStimulus, fixture.envelope, baselineSet.roster, baselineSet.batches...)
	neighborStimulus := digestNumber(2)
	changedSet := buildCandidateBatches(t, fixture, neighborStimulus, domain.AttemptReduction, batchRunOptions{evidenceSalt: 10},
		observed(1, 10), observed(2, 11), uncomparable(3),
	)
	changedEligibility := mapFrom(t, neighborStimulus, fixture.envelope, changedSet.roster, changedSet.batches...)
	if baseline.PreservationDigest() != changedEligibility.PreservationDigest() {
		t.Fatal("fixture should keep eligible preservation map equal")
	}
	if SamePreservationMap(baseline, changedEligibility) {
		t.Fatal("public preservation predicate ignored changed exclusion disposition")
	}
	if ComparableForPreservation(baseline, changedEligibility) {
		t.Fatal("changed exclusion classification remained preservation-comparable")
	}
}

func TestConfirmedProjectionRosterIsMapDerivedAndExact(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2, 3)
	confirmedSet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptConfirmation, batchRunOptions{evidenceSalt: 11},
		observed(1, 10), observed(2, 11),
	)
	confirmedMap := mapFrom(t, testStimulus, fixture.envelope, confirmedSet.roster, confirmedSet.batches...)
	confirmed, err := RequireConfirmedOutcomeMap(confirmedMap)
	if err != nil {
		t.Fatal(err)
	}
	authority := confirmed.ProjectionRoster()
	if !authority.Valid() || authority.OutcomeMapDigest() != confirmedMap.ArtifactDigest() ||
		authority.PreservationDigest() != confirmedMap.PreservationDigest() {
		t.Fatal("confirmed projection authority lost exact map binding")
	}
	if got, err := authority.Verify(fixture.key(t, 1), projectionBytes(10)); err != nil || got != projectionNumber(10) {
		t.Fatalf("exact confirmed projection = %v, %v", got, err)
	}
	if _, err := authority.Verify(fixture.key(t, 1), projectionBytes(11)); err == nil {
		t.Fatal("candidate accepted another confirmed candidate's projection")
	}
	if _, err := authority.Verify(fixture.key(t, 3), projectionBytes(10)); err == nil {
		t.Fatal("unconfirmed candidate entered projection roster")
	}
	discoverySet := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 12},
		observed(1, 10), observed(2, 11),
	)
	discoveryMap := mapFrom(t, testStimulus, fixture.envelope, discoverySet.roster, discoverySet.batches...)
	if _, err := RequireConfirmedOutcomeMap(discoveryMap); err == nil {
		t.Fatal("discovery evidence constructed confirmed projection authority")
	}
}

func TestCandidateOutcomeMapRejectsDifferentAdmissionSetsEvenWhenBasisMatches(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	left := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 13},
		observed(1, 10), observed(2, 11),
	)
	right := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 14},
		observed(1, 10), observed(2, 11),
	)
	if left.batch(t, 1).ComparisonBasisDigest() != right.batch(t, 2).ComparisonBasisDigest() {
		t.Fatal("fixture unexpectedly changed its declared comparison basis")
	}
	if sameDigestList(left.batch(t, 1).AdmissionDigests(), right.batch(t, 2).AdmissionDigests()) {
		t.Fatal("independent admitted matrices reused admission identity")
	}
	_, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), left.roster,
		[]observe.StableBatch{left.batch(t, 1), right.batch(t, 2)},
	)
	requireErrorCode(t, err, "MIXED_BATCH_ADMISSION_OR_PHASE")
}

func TestCandidateOutcomeMapRejectsCrossCandidateRepeatedAttemptEvidence(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	reused := digestNumber(9000000)
	set := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{
		evidenceSalt: 15,
		attemptOverride: map[trialCoordinate]domain.Digest{
			{candidateNumber: 1, repetition: 0}: reused,
			{candidateNumber: 2, repetition: 1}: reused,
		},
	}, observed(1, 10), observed(2, 11))
	_, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, set.batches)
	requireErrorCode(t, err, "REUSED_OUTCOME_MAP_ATTEMPT_EVIDENCE")
}

func TestCandidateOutcomeMapRejectsCrossCandidateRepeatedObservationEvidence(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	reused := digestNumber(9000002)
	set := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{
		evidenceSalt: 16,
		observationOverride: map[trialCoordinate]domain.Digest{
			{candidateNumber: 1, repetition: 0}: reused,
			{candidateNumber: 2, repetition: 0}: reused,
		},
	}, observed(1, 10), observed(2, 11))
	_, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, set.batches)
	requireErrorCode(t, err, "REUSED_OUTCOME_MAP_OBSERVATION_EVIDENCE")
}

func TestWorldEvidenceIdentityBindsCandidateBeforeOutcomeMapping(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	// Exact cross-candidate WorldDigest reuse is not representable through the
	// sealed constructors: CandidateExecutionKey is part of WorldInstance
	// identity. This is the upstream invariant behind that outcome-map guard.
	config := domain.WorldInstanceConfig{
		StimulusDigest: testStimulus, AttemptArtifactDigest: digestNumber(9000001), Purpose: domain.AttemptDiscovery,
		InstanceNonce: "same-structural-allocation", ScheduleOrdinal: 0,
	}
	left, err := domain.NewWorldInstance(fixture.plan, fixture.binding(t, 1), config)
	if err != nil {
		t.Fatal(err)
	}
	right, err := domain.NewWorldInstance(fixture.plan, fixture.binding(t, 2), config)
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest() == right.Digest() {
		t.Fatal("cross-candidate worlds shared identity despite distinct execution bindings")
	}
}

func TestCandidateOutcomeMapRejectsMixedBasisAndPhase(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	baseline := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 16},
		observed(1, 10), observed(2, 11),
	)
	basisChanged := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 17, basisValue: 1},
		observed(1, 10), observed(2, 11),
	)
	phaseChanged := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptReduction, batchRunOptions{evidenceSalt: 18},
		observed(1, 10), observed(2, 11),
	)
	if baseline.batch(t, 1).ComparisonBasisDigest() == basisChanged.batch(t, 2).ComparisonBasisDigest() {
		t.Fatal("changed measured equality basis retained comparison-basis identity")
	}
	for name, second := range map[string]observe.StableBatch{
		"basis": basisChanged.batch(t, 2),
		"phase": phaseChanged.batch(t, 2),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), baseline.roster,
				[]observe.StableBatch{baseline.batch(t, 1), second},
			)
			requireErrorCode(t, err, "MIXED_BATCH_ADMISSION_OR_PHASE")
		})
	}
}

func TestCandidateOutcomeMapRejectsMixedPlanEvidence(t *testing.T) {
	leftFixture := newComparisonFixtureWithPlanSalt(t, 20, 1, 2)
	rightFixture := newComparisonFixtureWithPlanSalt(t, 21, 1, 2)
	left := buildCandidateBatches(t, leftFixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 19},
		observed(1, 10), observed(2, 11),
	)
	right := buildCandidateBatches(t, rightFixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 20},
		observed(1, 10), observed(2, 11),
	)
	if left.batch(t, 1).PlanDigest() == right.batch(t, 2).PlanDigest() {
		t.Fatal("fixture did not produce distinct plan identity")
	}
	mixedRoster := []domain.CandidateExecutionKey{left.batch(t, 1).CandidateKey(), right.batch(t, 2).CandidateKey()}
	_, err := NewCandidateOutcomeMap(testStimulus, leftFixture.envelope.Digest(), mixedRoster,
		[]observe.StableBatch{left.batch(t, 1), right.batch(t, 2)},
	)
	if err == nil {
		t.Fatal("batches bound to distinct world plans entered one outcome map")
	}
}

func TestComparisonBasisIgnoresStimulusAndDiscoveryReductionPhase(t *testing.T) {
	fixture := newComparisonFixture(t, 1, 2)
	discovery := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 21},
		observed(1, 10), observed(2, 11),
	)
	reductionStimulus := digestNumber(2)
	reduction := buildCandidateBatches(t, fixture, reductionStimulus, domain.AttemptReduction, batchRunOptions{evidenceSalt: 22},
		observed(1, 10), observed(2, 11),
	)
	discoveryMap := mapFrom(t, testStimulus, fixture.envelope, discovery.roster, discovery.batches...)
	reductionMap := mapFrom(t, reductionStimulus, fixture.envelope, reduction.roster, reduction.batches...)
	if discoveryMap.ComparisonBasisDigest() != reductionMap.ComparisonBasisDigest() {
		t.Fatal("stimulus or DISCOVERY/REDUCTION phase contaminated declared comparison-basis identity")
	}
	if sameDigestList(discoveryMap.AdmissionDigests(), reductionMap.AdmissionDigests()) {
		t.Fatal("per-run admissions ignored changed stimulus and phase")
	}
	if discoveryMap.PlanDigest() != reductionMap.PlanDigest() {
		t.Fatal("fixture changed plan while testing comparison-basis independence")
	}
}

func FuzzCandidatePermutationIdentity(f *testing.F) {
	f.Add(uint8(0), uint8(1), true)
	f.Add(uint8(3), uint8(3), false)
	f.Fuzz(func(t *testing.T, leftValue, rightValue uint8, reverse bool) {
		fixture := newComparisonFixture(t, 1, 2)
		leftProjection := int(leftValue%16) + 10
		rightProjection := int(rightValue%16) + 30
		set := buildCandidateBatches(t, fixture, testStimulus, domain.AttemptDiscovery, batchRunOptions{evidenceSalt: 23},
			observed(1, leftProjection), observed(2, rightProjection),
		)
		a, b := set.batch(t, 1), set.batch(t, 2)
		ordered := []observe.StableBatch{a, b}
		if reverse {
			ordered[0], ordered[1] = ordered[1], ordered[0]
		}
		first, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, []observe.StableBatch{a, b})
		if err != nil {
			t.Fatal(err)
		}
		second, err := NewCandidateOutcomeMap(testStimulus, fixture.envelope.Digest(), set.roster, ordered)
		if err != nil {
			t.Fatal(err)
		}
		if first.ArtifactDigest() != second.ArtifactDigest() || !SamePreservationMap(first, second) {
			t.Fatal("candidate permutation changed identity")
		}
	})
}
