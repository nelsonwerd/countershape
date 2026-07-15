package observe

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type preparedFixture struct {
	execution executionFixture
	bindings  map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding
	roster    []domain.CandidateExecutionKey
}

type inheritedDeadlineContext struct {
	done chan struct{}
}

func newInheritedDeadlineContext() *inheritedDeadlineContext {
	return &inheritedDeadlineContext{done: make(chan struct{})}
}

func (*inheritedDeadlineContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *inheritedDeadlineContext) Done() <-chan struct{}     { return c.done }
func (c *inheritedDeadlineContext) Err() error {
	select {
	case <-c.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}
func (*inheritedDeadlineContext) Value(any) any { return nil }
func (c *inheritedDeadlineContext) expire()     { close(c.done) }

func newPreparedFixture(t *testing.T, repeats int) preparedFixture {
	t.Helper()
	execution := newExecutionFixture(t, repeats, repeats)
	bindings := map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding{}
	for _, number := range []int{1, 2} {
		binding := execution.binding(t, number)
		bindings[binding.Key()] = binding
	}
	schedule, err := NewRotatedSchedule([]domain.CandidateExecutionKey{
		execution.binding(t, 2).Key(), execution.binding(t, 1).Key(),
	}, repeats)
	if err != nil {
		t.Fatal(err)
	}
	return preparedFixture{execution: execution, bindings: bindings, roster: schedule.Roster()}
}

func (f preparedFixture) config(maxTrials int) ObservationConfig {
	return ObservationConfig{
		Plan: f.execution.plan, Purpose: domain.AttemptDiscovery,
		Envelope: f.execution.envelope, CandidateRoster: append([]domain.CandidateExecutionKey(nil), f.roster...),
		Repetitions: f.execution.plan.RepeatSchedule().DiscoveryRepeats,
		Budget:      TrialBudget{MaxTotalTrials: maxTrials, WallBudget: time.Second},
	}
}

func (f preparedFixture) captured(
	t *testing.T,
	slot ScheduledTrial,
	attemptNumber int,
	projectionValue int,
	measurementValue string,
) PreparedTrial {
	t.Helper()
	attemptDigest := digest(attemptNumber)
	world, err := domain.NewWorldInstance(f.execution.plan, f.bindings[slot.CandidateKey()], domain.WorldInstanceConfig{
		StimulusDigest: f.execution.stimulus, AttemptArtifactDigest: attemptDigest,
		Purpose:         domain.AttemptDiscovery,
		InstanceNonce:   fmt.Sprintf("prepared:%d:%d", attemptNumber, slot.Ordinal()),
		ScheduleOrdinal: slot.Ordinal(),
	})
	if err != nil {
		t.Fatal(err)
	}
	measurements, err := domain.NewInstanceMeasurements(
		f.execution.envelope, world, f.measurements(t, measurementValue),
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt := cleanFinalizedAttempt(t, attemptDigest, domain.AttemptDiscovery)
	observationDigest := digest(60000 + slot.Ordinal())
	projection := projectionBytes(projectionValue)
	capture, err := NewStructuralCapture(
		world, attempt, observationDigest, projection,
		projectionDerivation(t, world, observationDigest, projection),
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := NewPreparedCapturedTrial(slot, world, attempt, measurements, capture)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func (f preparedFixture) projectionRejected(
	t *testing.T,
	slot ScheduledTrial,
	attemptNumber int,
) PreparedTrial {
	t.Helper()
	attemptDigest := digest(attemptNumber)
	world, err := domain.NewWorldInstance(f.execution.plan, f.bindings[slot.CandidateKey()], domain.WorldInstanceConfig{
		StimulusDigest: f.execution.stimulus, AttemptArtifactDigest: attemptDigest,
		Purpose:         domain.AttemptDiscovery,
		InstanceNonce:   fmt.Sprintf("projection-rejected:%d", slot.Ordinal()),
		ScheduleOrdinal: slot.Ordinal(),
	})
	if err != nil {
		t.Fatal(err)
	}
	measurements, err := domain.NewInstanceMeasurements(
		f.execution.envelope, world, f.measurements(t, "darwin"),
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt := cleanFinalizedAttempt(t, attemptDigest, domain.AttemptDiscovery)
	observationDigest := digest(61000 + slot.Ordinal())
	rejection, err := NewProjectionRejectionEvidence(ProjectionRejectionLineage{
		WorldDigest: world.Digest(), AttemptArtifactDigest: attempt.ArtifactDigest(),
		CandidateKey: world.CandidateKey(), CapturePolicyDigest: world.CapturePolicyDigest(),
		ObservationDigest: observationDigest, ProjectionDefinitionDigest: world.ProjectionDefinitionDigest(),
	},
		[]byte(`{"code":"CLI_PROJECTION_INVALID_STRICT_JSON","kind":"TEST_PROJECTION_REJECTION","operation":"strict-json"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := NewPreparedProjectionRejectedTrial(slot, world, attempt, measurements, rejection)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func TestPreparedProjectionRejectionRejectsEveryCrossPairedLineage(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	schedule, err := NewRotatedSchedule(fixture.roster, 3)
	if err != nil {
		t.Fatal(err)
	}
	slot := schedule.Trials()[0]
	prepared := fixture.projectionRejected(t, slot, 62000)
	base := ProjectionRejectionLineage{
		WorldDigest: prepared.world.Digest(), AttemptArtifactDigest: prepared.attempt.ArtifactDigest(),
		CandidateKey: prepared.world.CandidateKey(), CapturePolicyDigest: prepared.world.CapturePolicyDigest(),
		ObservationDigest:          prepared.rejection.ObservationDigest(),
		ProjectionDefinitionDigest: prepared.world.ProjectionDefinitionDigest(),
	}
	otherCandidate := fixture.roster[0]
	if otherCandidate == base.CandidateKey {
		otherCandidate = fixture.roster[1]
	}
	tests := map[string]func(*ProjectionRejectionLineage){
		"world":          func(lineage *ProjectionRejectionLineage) { lineage.WorldDigest = digest(62001) },
		"attempt":        func(lineage *ProjectionRejectionLineage) { lineage.AttemptArtifactDigest = digest(62002) },
		"candidate":      func(lineage *ProjectionRejectionLineage) { lineage.CandidateKey = otherCandidate },
		"capture-policy": func(lineage *ProjectionRejectionLineage) { lineage.CapturePolicyDigest = digest(62003) },
		"definition":     func(lineage *ProjectionRejectionLineage) { lineage.ProjectionDefinitionDigest = digest(62004) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			lineage := base
			mutate(&lineage)
			rejection, err := NewProjectionRejectionEvidence(lineage, prepared.rejection.adapterEvidence)
			if err != nil || !rejection.Valid() {
				t.Fatalf("could not construct independently valid cross-paired evidence: %v", err)
			}
			if rejection.Digest() == prepared.rejection.Digest() {
				t.Fatal("lineage mutation did not change rejection evidence identity")
			}
			if _, err := NewPreparedProjectionRejectedTrial(
				prepared.slot, prepared.world, prepared.attempt, prepared.measurements, rejection,
			); err == nil {
				t.Fatal("cross-paired projection rejection entered a prepared trial")
			}
		})
	}
}

func (f preparedFixture) measurements(t *testing.T, value string) []domain.MeasurementValue {
	t.Helper()
	result := measurementValues(t, f.execution.envelope)
	canonical, err := canon.String(value)
	if err != nil {
		t.Fatal(err)
	}
	result[0].Value = canonical
	return result
}

func batchForCandidate(t *testing.T, batches []StableBatch, candidate domain.CandidateExecutionKey) StableBatch {
	t.Helper()
	for _, batch := range batches {
		if batch.CandidateKey() == candidate {
			return batch
		}
	}
	t.Fatalf("missing batch for %s", candidate.String())
	return StableBatch{}
}

func TestRunObservationRotatesFreshAttemptsAndNeverMajorityClassifiesAlternation(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	seen := make([]ScheduledTrial, 0, 6)
	run, err := RunObservation(context.Background(), fixture.config(6), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		seen = append(seen, slot)
		value := 90
		if slot.CandidateKey() == fixture.roster[0] {
			value = 10 + slot.Repetition()%2
		}
		return fixture.captured(t, slot, 40000+slot.Ordinal(), value, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status() != ObservationComplete || run.CompletedMatrices() != 3 || len(seen) != 6 {
		t.Fatalf("run status=%s matrices=%d calls=%d", run.Status(), run.CompletedMatrices(), len(seen))
	}
	if !run.Schedule().Valid() || !run.Schedule().Digest().Valid() ||
		run.Schedule().Rotation() != domain.ScheduleRotationStartByRepetitionV1 {
		t.Fatal("run did not retain sealed rotation authority")
	}
	for index, slot := range run.Schedule().Trials() {
		if seen[index] != slot {
			t.Fatalf("producer call %d = %#v, want %#v", index, seen[index], slot)
		}
	}
	alternating := batchForCandidate(t, run.Batches(), fixture.roster[0]).Classification()
	// MUTATION_ANCHOR: alternating-majority-or-last-must-not-be-stable
	if alternating.Status() != Unstable || len(alternating.Histogram()) != 2 {
		t.Fatalf("alternating classification = %s %#v", alternating.Status(), alternating.Histogram())
	}
	if _, ok := alternating.Fingerprint(); ok {
		t.Fatal("alternating candidate acquired a majority/last fingerprint")
	}
	constant := batchForCandidate(t, run.Batches(), fixture.roster[1]).Classification()
	if constant.Status() != ObservedStable || constant.BoundedLabel() == "" {
		t.Fatalf("constant classification = %s", constant.Status())
	}
	for _, batch := range run.Batches() {
		if batch.ScheduleDigest() != run.Schedule().Digest() ||
			batch.Rotation() != string(domain.ScheduleRotationStartByRepetitionV1) {
			t.Fatal("scheduled batch dropped canonical rotation authority")
		}
	}
	mapInput, ok := run.OutcomeMapInput()
	if !ok || mapInput.StimulusDigest() != fixture.execution.stimulus ||
		mapInput.EnvelopeDigest() != fixture.execution.envelope.Digest() ||
		len(mapInput.Roster()) != 2 || len(mapInput.Batches()) != 2 {
		t.Fatal("complete admitted run did not expose exact map inputs")
	}
}

func TestRunObservationBindsScheduleAndProducedWorldToExactPlanMutationGuard(t *testing.T) {
	fixture := newPreparedFixture(t, 2)
	wrongCount := fixture.config(6)
	wrongCount.CandidateRoster = append(
		wrongCount.CandidateRoster,
		fixture.execution.binding(t, 3).Key(),
	)
	calls := 0
	_, err := RunObservation(context.Background(), wrongCount, func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		return fixture.captured(t, slot, 49000+slot.Ordinal(), 7, "darwin"), nil
	})
	var mismatch *domain.Error
	if !errors.As(err, &mismatch) || mismatch.Code != "OBSERVATION_SCHEDULE_PLAN_MISMATCH" || calls != 0 {
		t.Fatalf("plan candidate-count mismatch reached producer: calls=%d err=%v", calls, err)
	}

	foreignPlan := newExecutionFixtureWithMarker(t, 2, 2, 49999).plan
	wrongPlan := fixture.config(4)
	wrongPlan.Plan = foreignPlan
	calls = 0
	_, err = RunObservation(context.Background(), wrongPlan, func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		return fixture.captured(t, slot, 49100+slot.Ordinal(), 7, "darwin"), nil
	})
	mismatch = nil
	if !errors.As(err, &mismatch) || mismatch.Code != "PREPARED_REPEAT_REQUIREMENT_MISMATCH" || calls != 1 {
		t.Fatalf("foreign world-plan evidence crossed observation authority: calls=%d err=%v", calls, err)
	}
}

func TestRunObservationBudgetEndsWithExplicitIncompleteBatches(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	run, err := RunObservation(context.Background(), fixture.config(3), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		return fixture.captured(t, slot, 41000+slot.Ordinal(), 7, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status() != ObservationIncomplete || run.CompletedMatrices() != 1 {
		t.Fatalf("run status=%s matrices=%d", run.Status(), run.CompletedMatrices())
	}
	if run.Budget().StartedTrials() != 2 ||
		run.Budget().Exhaustion() != "TOTAL_TRIAL_BUDGET_CANNOT_COMPLETE_MATRIX" {
		t.Fatalf("budget = %#v", run.Budget())
	}
	for _, batch := range run.Batches() {
		classification := batch.Classification()
		if classification.Status() != Incomplete || classification.EligibleTrials() != 1 ||
			classification.RequiredTrials() != 3 {
			t.Fatalf("classification = %s %d/%d", classification.Status(), classification.EligibleTrials(), classification.RequiredTrials())
		}
	}
	if _, ok := run.OutcomeMapInput(); !ok {
		t.Fatal("one complete admitted matrix should expose a complete roster of explicit INCOMPLETE exclusions")
	}
}

func TestRunObservationWallBudgetCoversPostProducerAdmissionMutationGuard(t *testing.T) {
	fixture := newPreparedFixture(t, 1)
	config := fixture.config(2)
	config.Budget.WallBudget = time.Second
	started := time.Unix(100, 0)
	afterFinalProducer := false
	readsAfterFinalProducer := 0
	now := func() time.Time {
		if afterFinalProducer {
			readsAfterFinalProducer++
			if readsAfterFinalProducer >= 2 {
				return started.Add(time.Second)
			}
		}
		return started
	}
	calls := 0
	run, err := runObservation(context.Background(), config, func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		trial := fixture.captured(t, slot, 49500+slot.Ordinal(), 7, "darwin")
		if calls == 2 {
			afterFinalProducer = true
		}
		return trial, nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 0 ||
		len(run.Batches()) != 0 || run.Budget().Exhaustion() != "WALL_BUDGET_EXHAUSTED" {
		t.Fatalf("post-producer wall expiry acquired admission authority: calls=%d status=%s matrices=%d batches=%d budget=%#v",
			calls, run.Status(), run.CompletedMatrices(), len(run.Batches()), run.Budget())
	}
	if _, present := run.OutcomeMapInput(); present {
		t.Fatal("post-deadline matrix exposed outcome-map authority")
	}
}

func TestRunObservationBudgetStopCannotEraseEstablishedInstability(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	run, err := RunObservation(context.Background(), fixture.config(5), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		value := 90
		if slot.CandidateKey() == fixture.roster[0] {
			value = 10 + slot.Repetition()
		}
		return fixture.captured(t, slot, 41200+slot.Ordinal(), value, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status() != ObservationIncomplete || run.CompletedMatrices() != 2 {
		t.Fatalf("run status=%s matrices=%d", run.Status(), run.CompletedMatrices())
	}
	unstable := batchForCandidate(t, run.Batches(), fixture.roster[0]).Classification()
	// MUTATION_ANCHOR: established-instability-precedes-budget-incomplete
	if unstable.Status() != Unstable || unstable.EligibleTrials() != 2 || len(unstable.Histogram()) != 2 {
		t.Fatalf("budget-stopped disagreement = %s %#v", unstable.Status(), unstable.Histogram())
	}
	if _, ok := unstable.Fingerprint(); ok || len(unstable.Reasons()) != 0 {
		t.Fatal("established disagreement was relabeled as stable or budget-incomplete")
	}
	constant := batchForCandidate(t, run.Batches(), fixture.roster[1]).Classification()
	if constant.Status() != Incomplete || constant.EligibleTrials() != 2 || constant.RequiredTrials() != 3 {
		t.Fatalf("agreement-only prefix = %s %d/%d", constant.Status(), constant.EligibleTrials(), constant.RequiredTrials())
	}
}

func TestRunObservationParentCancellationPreservesEvidenceAndStrongerControl(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	run, err := RunObservation(parent, fixture.config(6), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		if slot.Repetition() == 1 {
			cancel()
			return PreparedTrial{}, context.Canceled
		}
		if slot.CandidateKey() == fixture.roster[0] {
			return fixture.projectionRejected(t, slot, 41300+slot.Ordinal()), nil
		}
		return fixture.captured(t, slot, 41300+slot.Ordinal(), 8, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 1 {
		t.Fatalf("calls=%d status=%s matrices=%d", calls, run.Status(), run.CompletedMatrices())
	}
	controlled := batchForCandidate(t, run.Batches(), fixture.roster[0]).Classification()
	if controlled.Status() != Uncomparable || len(controlled.Reasons()) != 1 ||
		controlled.Reasons()[0] != domain.ControlProjectionRejected {
		t.Fatalf("stronger admitted control lost to cancellation: %s %#v", controlled.Status(), controlled.Reasons())
	}
	if run.Budget().Exhaustion() != "AVAILABLE" {
		t.Fatalf("parent cancellation was mislabeled as budget exhaustion: %#v", run.Budget())
	}
}

func TestRunObservationInheritedDeadlinePreservesPriorMatrices(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	parent := newInheritedDeadlineContext()
	calls := 0
	run, err := RunObservation(parent, fixture.config(6), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		if slot.Repetition() == 1 {
			parent.expire()
			return PreparedTrial{}, context.DeadlineExceeded
		}
		return fixture.captured(t, slot, 41400+slot.Ordinal(), 8, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 1 || len(run.Batches()) != 2 {
		t.Fatalf("calls=%d status=%s matrices=%d batches=%d", calls, run.Status(), run.CompletedMatrices(), len(run.Batches()))
	}
	for _, batch := range run.Batches() {
		if classification := batch.Classification(); classification.Status() != Incomplete || classification.EligibleTrials() != 1 {
			t.Fatalf("inherited deadline discarded or promoted prior evidence: %#v", classification)
		}
	}
	if run.Budget().Exhaustion() != "AVAILABLE" {
		t.Fatalf("inherited deadline was mislabeled as local budget exhaustion: %#v", run.Budget())
	}
}

func TestRunObservationFinalSuccessfulProducerCannotHideParentCancellation(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	run, err := RunObservation(parent, fixture.config(6), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		result := fixture.captured(t, slot, 41450+slot.Ordinal(), 8, "darwin")
		if slot.Ordinal() == 5 {
			// Deliberately ignore the producer context and return a valid success
			// after canceling the parent on the last possible producer call.
			cancel()
		}
		return result, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 6 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 2 || len(run.Batches()) != 2 {
		t.Fatalf("calls=%d status=%s matrices=%d batches=%d", calls, run.Status(), run.CompletedMatrices(), len(run.Batches()))
	}
	for _, batch := range run.Batches() {
		classification := batch.Classification()
		if classification.Status() != Incomplete || classification.EligibleTrials() != 2 ||
			classification.RequiredTrials() != 3 {
			t.Fatalf("post-success cancellation promoted admitted prefix: %#v", classification)
		}
	}
	if run.Budget().Exhaustion() != "AVAILABLE" {
		t.Fatalf("parent cancellation was mislabeled as local budget exhaustion: %#v", run.Budget())
	}
}

func TestRunObservationBudgetTooSmallForFirstMatrixIsIncompleteWithoutTokens(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	calls := 0
	run, err := RunObservation(context.Background(), fixture.config(1), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		return fixture.captured(t, slot, 41500+slot.Ordinal(), 7, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 0 || len(run.Batches()) != 0 {
		t.Fatalf("calls=%d status=%s matrices=%d batches=%d", calls, run.Status(), run.CompletedMatrices(), len(run.Batches()))
	}
	if run.Budget().Exhaustion() != "TOTAL_TRIAL_BUDGET_CANNOT_COMPLETE_MATRIX" {
		t.Fatalf("budget = %#v", run.Budget())
	}
	if _, ok := run.OutcomeMapInput(); ok {
		t.Fatal("zero admitted matrices exposed outcome-map inputs")
	}
}

func TestRunObservationRejectsCompleteMeasurementMatrixBeforeBatching(t *testing.T) {
	fixture := newPreparedFixture(t, 3)
	run, err := RunObservation(context.Background(), fixture.config(6), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		measured := "darwin"
		if slot.CandidateKey() == fixture.roster[1] {
			measured = "linux"
		}
		return fixture.captured(t, slot, 42000+slot.Ordinal(), 5, measured), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status() != ObservationRejected || len(run.Batches()) != 0 {
		t.Fatalf("rejected run status=%s batches=%d", run.Status(), len(run.Batches()))
	}
	rejection, ok := run.Rejection()
	if !ok || len(rejection.ReasonCodes()) != 1 ||
		rejection.ReasonCodes()[0] != "REQUIRED_EQUAL_VARIANCE:operating system" {
		t.Fatalf("rejection = %#v", rejection.ReasonCodes())
	}
	if _, ok := run.OutcomeMapInput(); ok {
		t.Fatal("rejected comparison exposed outcome-map inputs")
	}
}

func TestRunObservationWallBudgetCannotExpireWhilePublishingRejectedMatrix(t *testing.T) {
	fixture := newPreparedFixture(t, 1)
	config := fixture.config(2)
	config.Budget.WallBudget = time.Second
	started := time.Unix(200, 0)
	afterFinalProducer := false
	readsAfterFinalProducer := 0
	now := func() time.Time {
		if afterFinalProducer {
			readsAfterFinalProducer++
			if readsAfterFinalProducer >= 4 {
				return started.Add(time.Second)
			}
		}
		return started
	}
	calls := 0
	run, err := runObservation(context.Background(), config, func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		calls++
		measured := "darwin"
		if slot.CandidateKey() == fixture.roster[1] {
			measured = "linux"
		}
		trial := fixture.captured(t, slot, 49600+slot.Ordinal(), 5, measured)
		if calls == 2 {
			afterFinalProducer = true
		}
		return trial, nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || run.Status() != ObservationIncomplete || run.CompletedMatrices() != 0 ||
		len(run.Batches()) != 0 || run.Budget().Exhaustion() != "WALL_BUDGET_EXHAUSTED" {
		t.Fatalf("post-assessment wall expiry published rejection authority: calls=%d reads=%d status=%s matrices=%d batches=%d budget=%#v",
			calls, readsAfterFinalProducer, run.Status(), run.CompletedMatrices(), len(run.Batches()), run.Budget())
	}
	if _, present := run.Rejection(); present {
		t.Fatal("post-deadline comparison rejection escaped the whole-run wall budget")
	}
}

func TestRunObservationRefusesPriorAttemptReuseAcrossCandidates(t *testing.T) {
	fixture := newPreparedFixture(t, 1)
	_, err := RunObservation(context.Background(), fixture.config(2), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		// Each world is distinct, but the attempt artifact is deliberately copied.
		return fixture.captured(t, slot, 43000, 5, "darwin"), nil
	})
	if err == nil {
		t.Fatal("prior attempt reuse was accepted")
	}
	var domainErr *domain.Error
	if !errorsAs(err, &domainErr) || domainErr.Code != "REUSED_ORCHESTRATION_ATTEMPT_EVIDENCE" {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectionRejectionIsPostAttemptControlNeverOutcome(t *testing.T) {
	fixture := newPreparedFixture(t, 1)
	run, err := RunObservation(context.Background(), fixture.config(2), func(_ context.Context, slot ScheduledTrial) (PreparedTrial, error) {
		if slot.CandidateKey() == fixture.roster[0] {
			return fixture.projectionRejected(t, slot, 44000+slot.Ordinal()), nil
		}
		return fixture.captured(t, slot, 44000+slot.Ordinal(), 8, "darwin"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rejected := batchForCandidate(t, run.Batches(), fixture.roster[0]).Classification()
	if rejected.Status() != Uncomparable || len(rejected.Reasons()) != 1 ||
		rejected.Reasons()[0] != domain.ControlProjectionRejected {
		t.Fatalf("projection rejection classification = %s %#v", rejected.Status(), rejected.Reasons())
	}
	if _, ok := rejected.Fingerprint(); ok {
		t.Fatal("projection rejection became an outcome")
	}
}

// errorsAs is a tiny local indirection so mutation tooling can target evidence
// reuse behavior without rewriting imports or test setup.
func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
