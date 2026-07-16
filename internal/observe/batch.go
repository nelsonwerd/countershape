package observe

import (
	"context"
	"errors"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
)

type preparedTrialKind string

const (
	preparedCaptured           preparedTrialKind = "CAPTURED"
	preparedAttemptControlled  preparedTrialKind = "ATTEMPT_CONTROLLED"
	preparedProjectionRejected preparedTrialKind = "PROJECTION_REJECTED"
)

// PreparedTrial is the adapter's pre-admission handoff. It contains one exact
// scheduled world, its complete measurement row, and enough typed evidence to
// construct a TrialFact only after the complete repetition matrix is admitted.
// It deliberately carries no AdmissionToken supplied by an adapter.
type PreparedTrial struct {
	slot         ScheduledTrial
	kind         preparedTrialKind
	world        domain.WorldInstance
	attempt      domain.FinalizedAttempt
	measurements domain.InstanceMeasurements
	capture      StructuralCapture
	rejection    ProjectionRejectionEvidence
}

func NewPreparedCapturedTrial(
	slot ScheduledTrial,
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	measurements domain.InstanceMeasurements,
	capture StructuralCapture,
) (PreparedTrial, error) {
	prepared := PreparedTrial{
		slot: slot, kind: preparedCaptured, world: world, attempt: attempt,
		measurements: measurements, capture: capture,
	}
	if err := validatePreparedCommon(prepared); err != nil {
		return PreparedTrial{}, err
	}
	if attempt.HasControls() || !capture.observationDigest.Valid() || !capture.projectionDigest.Valid() ||
		!capture.derivationDigest.Valid() || !capture.fingerprint.Valid() || capture.worldDigest != world.Digest() ||
		capture.attemptDigest != attempt.ArtifactDigest() ||
		capture.capturePolicyDigest != world.CapturePolicyDigest() ||
		capture.projectionDefinitionDigest != world.ProjectionDefinitionDigest() {
		return PreparedTrial{}, &domain.Error{Code: "INVALID_PREPARED_CAPTURE", Detail: "capture or attempt lineage is incomplete"}
	}
	return prepared, nil
}

func NewPreparedControlledTrial(
	slot ScheduledTrial,
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	measurements domain.InstanceMeasurements,
) (PreparedTrial, error) {
	prepared := PreparedTrial{
		slot: slot, kind: preparedAttemptControlled, world: world,
		attempt: attempt, measurements: measurements,
	}
	if err := validatePreparedCommon(prepared); err != nil {
		return PreparedTrial{}, err
	}
	if !attempt.HasControls() {
		return PreparedTrial{}, &domain.Error{Code: "INVALID_PREPARED_CONTROL", Detail: "attempt has no control evidence"}
	}
	if primary, ok := attempt.PrimaryControl(); ok && primary == domain.ControlEnvelopeRejected {
		return PreparedTrial{}, &domain.Error{Code: "ENVELOPE_REJECTION_HAS_NO_PREPARED_TRIAL"}
	}
	return prepared, nil
}

// NewPreparedProjectionRejectedTrial is the one post-attempt control edge in
// U3. Projection is necessarily evaluated after a clean finalized process
// attempt, so PROJECTION_REJECTED cannot be forged into that earlier attempt's
// state history. The central observe package attaches the typed control only
// after matrix admission; it can never become a fingerprint.
func NewPreparedProjectionRejectedTrial(
	slot ScheduledTrial,
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	measurements domain.InstanceMeasurements,
	rejection ProjectionRejectionEvidence,
) (PreparedTrial, error) {
	prepared := PreparedTrial{
		slot: slot, kind: preparedProjectionRejected, world: world,
		attempt: attempt, measurements: measurements, rejection: rejection,
	}
	if err := validatePreparedCommon(prepared); err != nil {
		return PreparedTrial{}, err
	}
	if attempt.HasControls() {
		return PreparedTrial{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION", Detail: "attempt already has a process control"}
	}
	if !rejection.Valid() || rejection.worldDigest != world.Digest() ||
		rejection.attemptArtifactDigest != attempt.ArtifactDigest() ||
		rejection.candidateKey != world.CandidateKey() ||
		rejection.capturePolicyDigest != world.CapturePolicyDigest() ||
		rejection.definitionDigest != world.ProjectionDefinitionDigest() {
		return PreparedTrial{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION_EVIDENCE"}
	}
	return prepared, nil
}

func validatePreparedCommon(prepared PreparedTrial) error {
	if !prepared.slot.candidateKey.Valid() || prepared.slot.repetition < 0 ||
		prepared.slot.position < 0 || prepared.slot.ordinal < 0 ||
		!prepared.world.Digest().Valid() || !prepared.attempt.ArtifactDigest().Valid() ||
		!prepared.measurements.Digest().Valid() {
		return &domain.Error{Code: "INVALID_PREPARED_TRIAL", Detail: "scheduled evidence is incomplete"}
	}
	if prepared.world.CandidateKey() != prepared.slot.candidateKey ||
		prepared.world.ScheduleOrdinal() != prepared.slot.ordinal ||
		prepared.world.AttemptArtifactDigest() != prepared.attempt.ArtifactDigest() ||
		prepared.world.Purpose() != prepared.attempt.Purpose() ||
		prepared.measurements.SubjectDigest() != prepared.world.Digest() {
		return &domain.Error{Code: "PREPARED_TRIAL_LINEAGE_MISMATCH"}
	}
	return nil
}

func (p PreparedTrial) admit(comparison domain.AdmittedComparison) (TrialFact, error) {
	token, err := comparison.AdmissionFor(p.measurements)
	if err != nil {
		return TrialFact{}, err
	}
	switch p.kind {
	case preparedCaptured:
		return NewCapturedTrial(p.world, p.attempt, token, p.capture)
	case preparedAttemptControlled:
		return NewControlledTrial(p.world, p.attempt, token)
	case preparedProjectionRejected:
		return NewProjectionRejectedTrial(p.world, p.attempt, token, p.rejection)
	default:
		return TrialFact{}, &domain.Error{Code: "UNKNOWN_PREPARED_TRIAL_KIND"}
	}
}

type ObservationRunStatus string

const (
	ObservationComplete   ObservationRunStatus = "COMPLETE"
	ObservationIncomplete ObservationRunStatus = "INCOMPLETE"
	ObservationRejected   ObservationRunStatus = "COMPARISON_REJECTED"
)

type ObservationConfig struct {
	Plan            domain.WorldPlan
	Purpose         domain.AttemptPurpose
	Envelope        domain.ComparisonEnvelope
	CandidateRoster []domain.CandidateExecutionKey
	Repetitions     int
	Budget          TrialBudget
}

type TrialProducer func(context.Context, ScheduledTrial) (PreparedTrial, error)

// OutcomeMapInput is the cycle-safe handoff to internal/compare. Compare
// imports observe to validate StableBatch, so observe cannot import compare in
// return. Only a non-rejected run with at least one complete admitted matrix
// can expose this capability; a higher package passes its exact getters to
// compare.NewCandidateOutcomeMap.
type OutcomeMapInput struct {
	stimulusDigest domain.Digest
	envelopeDigest domain.Digest
	roster         []domain.CandidateExecutionKey
	batches        []StableBatch
}

func (i OutcomeMapInput) StimulusDigest() domain.Digest { return i.stimulusDigest }
func (i OutcomeMapInput) EnvelopeDigest() domain.Digest { return i.envelopeDigest }
func (i OutcomeMapInput) Roster() []domain.CandidateExecutionKey {
	return append([]domain.CandidateExecutionKey(nil), i.roster...)
}
func (i OutcomeMapInput) Batches() []StableBatch {
	return append([]StableBatch(nil), i.batches...)
}

type ObservationRun struct {
	status            ObservationRunStatus
	schedule          RotatedSchedule
	completedMatrices int
	batches           []StableBatch
	rejection         domain.RejectedComparison
	hasRejection      bool
	budget            BudgetSnapshot
	mapInput          OutcomeMapInput
	hasMapInput       bool
}

func (r ObservationRun) Status() ObservationRunStatus { return r.status }
func (r ObservationRun) Schedule() RotatedSchedule    { return cloneSchedule(r.schedule) }
func (r ObservationRun) CompletedMatrices() int       { return r.completedMatrices }
func (r ObservationRun) Batches() []StableBatch {
	return append([]StableBatch(nil), r.batches...)
}
func (r ObservationRun) Rejection() (domain.RejectedComparison, bool) {
	return r.rejection, r.hasRejection
}
func (r ObservationRun) Budget() BudgetSnapshot { return r.budget }
func (r ObservationRun) OutcomeMapInput() (OutcomeMapInput, bool) {
	if !r.hasMapInput {
		return OutcomeMapInput{}, false
	}
	return cloneOutcomeMapInput(r.mapInput), true
}

func cloneSchedule(input RotatedSchedule) RotatedSchedule {
	return RotatedSchedule{
		digest:      input.digest,
		canonical:   append([]byte(nil), input.canonical...),
		roster:      append([]domain.CandidateExecutionKey(nil), input.roster...),
		repetitions: input.repetitions,
		phase:       input.phase,
		startOffset: input.startOffset,
		trials:      append([]ScheduledTrial(nil), input.trials...),
	}
}

func cloneOutcomeMapInput(input OutcomeMapInput) OutcomeMapInput {
	return OutcomeMapInput{
		stimulusDigest: input.stimulusDigest, envelopeDigest: input.envelopeDigest,
		roster:  append([]domain.CandidateExecutionKey(nil), input.roster...),
		batches: append([]StableBatch(nil), input.batches...),
	}
}

// RunObservation executes the adapter producer strictly in schedule order. One
// and only one AssessComparison call is made after each complete repetition
// matrix. A rejected matrix stops the whole run before any StableBatch is
// exposed. Partial matrices are never assessed and never produce tokens.
func RunObservation(
	ctx context.Context,
	config ObservationConfig,
	producer TrialProducer,
) (ObservationRun, error) {
	return runObservation(ctx, config, producer, time.Now)
}

func runObservation(
	ctx context.Context,
	config ObservationConfig,
	producer TrialProducer,
	now func() time.Time,
) (ObservationRun, error) {
	if ctx == nil || producer == nil || now == nil || !config.Envelope.Digest().Valid() ||
		!config.Plan.Digest().Valid() || len(config.Plan.CanonicalBytes()) == 0 || !config.Purpose.Valid() ||
		config.Plan.ComparisonEnvelopeDigest() != config.Envelope.Digest() {
		return ObservationRun{}, &domain.Error{Code: "INVALID_OBSERVATION_CONFIG"}
	}
	schedule, err := NewPhaseRotatedSchedule(config.CandidateRoster, config.Repetitions, config.Purpose)
	if err != nil {
		return ObservationRun{}, err
	}
	expectedRepeats := config.Plan.RepeatSchedule().DiscoveryRepeats
	if config.Purpose == domain.AttemptFinalSweep || config.Purpose == domain.AttemptConfirmation ||
		config.Purpose == domain.AttemptConformance {
		expectedRepeats = config.Plan.RepeatSchedule().ConfirmationRepeats
	}
	// MUTATION_ANCHOR: observation-schedule-must-match-world-plan
	if schedule.CandidateCount() != config.Plan.Budgets().CandidateCount ||
		schedule.Repetitions() != expectedRepeats ||
		schedule.Rotation() != config.Plan.ScheduleRotation() {
		return ObservationRun{}, &domain.Error{Code: "OBSERVATION_SCHEDULE_PLAN_MISMATCH"}
	}
	if err := config.Budget.validate(schedule.CandidateCount()); err != nil {
		return ObservationRun{}, err
	}

	startedAt := now()
	tracker := newBudgetTracker(config.Budget, startedAt)
	deadline := startedAt.Add(config.Budget.WallBudget)
	runContext, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	trialsByCandidate := make(map[domain.CandidateExecutionKey][]TrialFact, schedule.CandidateCount())
	seenAttempts := make(map[domain.Digest]struct{}, schedule.TotalTrials())
	seenWorlds := make(map[domain.Digest]struct{}, schedule.TotalTrials())
	seenObservations := make(map[domain.Digest]struct{}, schedule.TotalTrials())
	completedMatrices := 0

	for repetition := 0; repetition < schedule.Repetitions(); repetition++ {
		if ctx.Err() != nil {
			return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
		}
		if !tracker.canStartMatrix(now(), schedule.CandidateCount()) {
			return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
		}
		slots, _ := schedule.TrialsForRepetition(repetition)
		prepared := make([]PreparedTrial, 0, len(slots))
		measurements := make([]domain.InstanceMeasurements, 0, len(slots))
		for _, slot := range slots {
			if ctx.Err() != nil {
				return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
			}
			if !tracker.beginTrial(now()) {
				return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
			}
			result, produceErr := producer(runContext, slot)
			finishedAt := now()
			completedWithinBudget := tracker.completeTrial(finishedAt)
			if produceErr != nil {
				if observationContextEnded(ctx, produceErr) ||
					(errors.Is(produceErr, context.DeadlineExceeded) && !finishedAt.Before(deadline)) {
					return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(finishedAt), ObservationIncomplete)
				}
				return ObservationRun{}, produceErr
			}
			if !completedWithinBudget {
				return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(finishedAt), ObservationIncomplete)
			}
			if err := validateProducedSlot(result, slot, schedule, config.Plan, config.Purpose); err != nil {
				return ObservationRun{}, err
			}
			if err := registerFreshEvidence(result, seenAttempts, seenWorlds, seenObservations); err != nil {
				return ObservationRun{}, err
			}
			// A producer may ignore cancellation and still return a structurally
			// valid success. Parent cancellation observed after that call must stop
			// before the partial matrix is admitted, including on the final slot of
			// the final repetition; otherwise the run can falsely become COMPLETE.
			if ctx.Err() != nil {
				return finishObservation(
					schedule, completedMatrices, trialsByCandidate,
					tracker.snapshot(finishedAt), ObservationIncomplete,
				)
			}
			prepared = append(prepared, result)
			measurements = append(measurements, result.measurements)
		}

		// MUTATION_ANCHOR: observation-wall-budget-covers-assessment-and-admission
		if !tracker.withinWall(now()) {
			return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
		}
		assessment, err := domain.AssessComparison(config.Envelope, measurements)
		if err != nil {
			return ObservationRun{}, err
		}
		if !tracker.withinWall(now()) {
			return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
		}
		switch comparison := assessment.(type) {
		case domain.RejectedComparison:
			rejectedAt := now()
			if !tracker.withinWall(rejectedAt) {
				return finishObservation(
					schedule, completedMatrices, trialsByCandidate,
					tracker.snapshot(rejectedAt), ObservationIncomplete,
				)
			}
			return ObservationRun{
				status: ObservationRejected, schedule: cloneSchedule(schedule),
				completedMatrices: completedMatrices, rejection: comparison, hasRejection: true,
				budget: tracker.snapshot(rejectedAt),
			}, nil
		case domain.AdmittedComparison:
			matrixTrials := make([]TrialFact, len(prepared))
			for index, candidateTrial := range prepared {
				trial, trialErr := candidateTrial.admit(comparison)
				if trialErr != nil {
					return ObservationRun{}, trialErr
				}
				matrixTrials[index] = trial
			}
			if !tracker.withinWall(now()) {
				return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
			}
			for index, candidateTrial := range prepared {
				candidate := candidateTrial.slot.candidateKey
				trialsByCandidate[candidate] = append(trialsByCandidate[candidate], matrixTrials[index])
			}
			completedMatrices++
		default:
			return ObservationRun{}, &domain.Error{Code: "UNKNOWN_COMPARISON_ASSESSMENT"}
		}
	}

	if !tracker.withinWall(now()) {
		return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
	}
	finished, err := finishObservation(
		schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationComplete,
	)
	if err != nil {
		return ObservationRun{}, err
	}
	if !tracker.withinWall(now()) {
		return finishObservation(schedule, completedMatrices, trialsByCandidate, tracker.snapshot(now()), ObservationIncomplete)
	}
	return finished, nil
}

func observationContextEnded(parent context.Context, producerErr error) bool {
	parentErr := parent.Err()
	return parentErr != nil && errors.Is(producerErr, parentErr)
}

func validateProducedSlot(
	result PreparedTrial,
	slot ScheduledTrial,
	schedule RotatedSchedule,
	plan domain.WorldPlan,
	purpose domain.AttemptPurpose,
) error {
	if result.slot != slot || !slot.valid(schedule.CandidateCount(), schedule.Repetitions()) {
		return &domain.Error{Code: "PRODUCER_SCHEDULE_MISMATCH"}
	}
	if err := validatePreparedCommon(result); err != nil {
		return err
	}
	if result.world.PlanDigest() != plan.Digest() || result.world.Purpose() != purpose ||
		result.world.RequiredFreshTrials() != schedule.Repetitions() {
		return &domain.Error{Code: "PREPARED_REPEAT_REQUIREMENT_MISMATCH"}
	}
	return nil
}

func registerFreshEvidence(
	prepared PreparedTrial,
	seenAttempts, seenWorlds, seenObservations map[domain.Digest]struct{},
) error {
	attemptDigest := prepared.attempt.ArtifactDigest()
	// MUTATION_ANCHOR: prior-attempt-reuse-must-be-refused
	if _, reused := seenAttempts[attemptDigest]; reused {
		return &domain.Error{Code: "REUSED_ORCHESTRATION_ATTEMPT_EVIDENCE", Detail: attemptDigest.String()}
	}
	seenAttempts[attemptDigest] = struct{}{}
	worldDigest := prepared.world.Digest()
	if _, reused := seenWorlds[worldDigest]; reused {
		return &domain.Error{Code: "REUSED_ORCHESTRATION_WORLD_EVIDENCE", Detail: worldDigest.String()}
	}
	seenWorlds[worldDigest] = struct{}{}
	if prepared.kind == preparedCaptured || prepared.kind == preparedProjectionRejected {
		observationDigest := prepared.capture.observationDigest
		if prepared.kind == preparedProjectionRejected {
			observationDigest = prepared.rejection.observationDigest
		}
		if _, reused := seenObservations[observationDigest]; reused {
			return &domain.Error{Code: "REUSED_ORCHESTRATION_OBSERVATION_EVIDENCE", Detail: observationDigest.String()}
		}
		seenObservations[observationDigest] = struct{}{}
	}
	return nil
}

func finishObservation(
	schedule RotatedSchedule,
	completedMatrices int,
	trialsByCandidate map[domain.CandidateExecutionKey][]TrialFact,
	budget BudgetSnapshot,
	status ObservationRunStatus,
) (ObservationRun, error) {
	result := ObservationRun{
		status: status, schedule: cloneSchedule(schedule),
		completedMatrices: completedMatrices, budget: budget,
	}
	if completedMatrices == 0 {
		return result, nil
	}
	batches := make([]StableBatch, 0, schedule.CandidateCount())
	for _, candidate := range schedule.roster {
		trials := trialsByCandidate[candidate]
		if len(trials) != completedMatrices {
			return ObservationRun{}, &domain.Error{Code: "INCOMPLETE_ADMITTED_MATRIX_SET"}
		}
		// MUTATION_ANCHOR: alternating-candidate-must-use-all-trials-never-majority-or-last
		batch, err := ClassifyScheduled(ScheduledBatchInput{
			Schedule:      schedule,
			CandidateKey:  candidate,
			Trials:        append([]TrialFact(nil), trials...),
			RunIncomplete: status == ObservationIncomplete,
		})
		if err != nil {
			return ObservationRun{}, err
		}
		batches = append(batches, batch)
	}
	result.batches = append([]StableBatch(nil), batches...)
	input := OutcomeMapInput{
		stimulusDigest: batches[0].StimulusDigest(),
		envelopeDigest: batches[0].EnvelopeDigest(),
		roster:         append([]domain.CandidateExecutionKey(nil), schedule.roster...),
		batches:        append([]StableBatch(nil), batches...),
	}
	result.mapInput = input
	result.hasMapInput = true
	return result, nil
}
