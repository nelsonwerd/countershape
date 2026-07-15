// Package reduce defines pure, logical, tri-valued reduction evidence. Durable
// completion and ONE_MINIMAL_UNDER authority intentionally do not exist in U1;
// they require the store-backed transcript and CAS lineage introduced later.
package reduce

import (
	"sort"

	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type Decision string

const (
	Preserves  Decision = "PRESERVES"
	Changes    Decision = "CHANGES"
	Unresolved Decision = "UNRESOLVED"
)

func unresolvedDecision() Decision {
	// MUTATION_ANCHOR: unresolved-is-not-changes
	return Unresolved
}

// Neighbor is an opaque direct-neighbor proposal bound to the exact current
// stimulus, well-founded measure, and reducer-set identity.
type Neighbor struct {
	currentStimulusDigest domain.Digest
	currentMeasure        int
	stimulusDigest        domain.Digest
	measure               int
	reducerSetDigest      domain.Digest
}

func NewNeighbor(
	currentStimulusDigest domain.Digest,
	currentMeasure int,
	stimulusDigest domain.Digest,
	measure int,
	reducerSetDigest domain.Digest,
) (Neighbor, error) {
	if !currentStimulusDigest.Valid() || !stimulusDigest.Valid() || !reducerSetDigest.Valid() ||
		currentStimulusDigest == stimulusDigest || currentMeasure <= 0 || measure < 0 || measure >= currentMeasure {
		return Neighbor{}, &domain.Error{Code: "INVALID_DIRECT_NEIGHBOR"}
	}
	return Neighbor{
		currentStimulusDigest: currentStimulusDigest,
		currentMeasure:        currentMeasure,
		stimulusDigest:        stimulusDigest,
		measure:               measure,
		reducerSetDigest:      reducerSetDigest,
	}, nil
}

func (n Neighbor) CurrentStimulusDigest() domain.Digest { return n.currentStimulusDigest }
func (n Neighbor) CurrentMeasure() int                  { return n.currentMeasure }
func (n Neighbor) StimulusDigest() domain.Digest        { return n.stimulusDigest }
func (n Neighbor) Measure() int                         { return n.measure }
func (n Neighbor) ReducerSetDigest() domain.Digest      { return n.reducerSetDigest }

type Evaluation struct {
	id                          string
	neighbor                    Neighbor
	purpose                     domain.AttemptPurpose
	observedBatchDigests        []domain.Digest
	observedAttemptDigests      []domain.Digest
	observedWorldDigests        []domain.Digest
	observedObservationDigests  []domain.Digest
	observedOutcomeMapDigest    *compare.OutcomeArtifactDigest
	decision                    Decision
	reasonCode                  string
	logicalNonReuseWithBaseline bool
	baselineOutcomeMapDigest    compare.OutcomeArtifactDigest
	baselinePreservationDigest  compare.PreservationMapDigest
	comparisonEnvelopeDigest    domain.Digest
	candidateRoster             []domain.CandidateExecutionKey
}

type EvaluationInput struct {
	ID                 string
	Baseline           compare.DivergentBaseline
	Neighbor           Neighbor
	ObservedOutcomeMap *compare.CandidateOutcomeMap
	UnresolvedReason   string
}

func NewEvaluation(input EvaluationInput) (Evaluation, error) {
	return newEvaluation(input, domain.AttemptReduction)
}

// NewFinalSweepEvaluation is the only constructor whose observed evidence may
// carry FINAL_SWEEP purpose. Ordinary reduction and final-sweep evidence are
// deliberately distinct even when all other structural facts match.
func NewFinalSweepEvaluation(input EvaluationInput) (Evaluation, error) {
	return newEvaluation(input, domain.AttemptFinalSweep)
}

func newEvaluation(input EvaluationInput, requiredPurpose domain.AttemptPurpose) (Evaluation, error) {
	if input.ID == "" || !input.Baseline.Valid() || !validNeighbor(input.Neighbor) {
		return Evaluation{}, &domain.Error{Code: "INVALID_REDUCTION_EVALUATION"}
	}
	baseline := input.Baseline.OutcomeMap()
	if baseline.StimulusDigest() != input.Neighbor.currentStimulusDigest {
		return Evaluation{}, &domain.Error{Code: "EVALUATION_BASELINE_STIMULUS_MISMATCH"}
	}
	evaluation := Evaluation{
		id:                         input.ID,
		neighbor:                   input.Neighbor,
		purpose:                    requiredPurpose,
		baselineOutcomeMapDigest:   baseline.ArtifactDigest(),
		baselinePreservationDigest: baseline.PreservationDigest(),
		comparisonEnvelopeDigest:   baseline.EnvelopeDigest(),
		candidateRoster:            baseline.CandidateRoster(),
	}
	if input.ObservedOutcomeMap == nil {
		if input.UnresolvedReason == "" {
			return Evaluation{}, &domain.Error{Code: "MISSING_UNRESOLVED_REASON"}
		}
		evaluation.decision = unresolvedDecision()
		evaluation.reasonCode = input.UnresolvedReason
		return evaluation, nil
	}
	if input.UnresolvedReason != "" {
		return Evaluation{}, &domain.Error{Code: "OBSERVED_EVALUATION_HAS_UNRESOLVED_REASON"}
	}
	observed := *input.ObservedOutcomeMap
	if !observed.ArtifactDigest().Valid() || !observed.PreservationDigest().Valid() {
		return Evaluation{}, &domain.Error{Code: "INVALID_OBSERVED_OUTCOME_MAP"}
	}
	// MUTANT_U1_REDUCTION_PHASE_BINDING: phase is authority, not a display label.
	if observed.Phase() != requiredPurpose {
		return Evaluation{}, &domain.Error{Code: "EVALUATION_ATTEMPT_PURPOSE_MISMATCH"}
	}
	if observed.StimulusDigest() != input.Neighbor.stimulusDigest {
		return Evaluation{}, &domain.Error{Code: "EVALUATION_NEIGHBOR_STIMULUS_MISMATCH"}
	}
	if observed.EnvelopeDigest() != baseline.EnvelopeDigest() {
		return Evaluation{}, &domain.Error{Code: "EVALUATION_ENVELOPE_MISMATCH"}
	}
	if !sameRoster(baseline.CandidateRoster(), observed.CandidateRoster()) {
		return Evaluation{}, &domain.Error{Code: "EVALUATION_CANDIDATE_ROSTER_MISMATCH"}
	}
	batchDigests := observed.BatchDigests()
	if len(batchDigests) != len(observed.CandidateRoster()) || hasInvalidOrDuplicate(batchDigests) {
		return Evaluation{}, &domain.Error{Code: "INVALID_OBSERVED_BATCH_SET"}
	}
	if intersects(baseline.BatchDigests(), batchDigests) {
		return Evaluation{}, &domain.Error{Code: "REUSED_BASELINE_BATCH_EVIDENCE"}
	}
	attemptDigests := observed.EvidenceAttemptDigests()
	worldDigests := observed.EvidenceWorldDigests()
	observationDigests := observed.EvidenceObservationDigests()
	if len(attemptDigests) == 0 || hasInvalidOrDuplicate(attemptDigests) ||
		len(worldDigests) == 0 || hasInvalidOrDuplicate(worldDigests) ||
		len(observationDigests) == 0 || hasInvalidOrDuplicate(observationDigests) {
		return Evaluation{}, &domain.Error{Code: "INVALID_OBSERVED_UNDERLYING_EVIDENCE_SET"}
	}
	if intersects(baseline.EvidenceAttemptDigests(), attemptDigests) {
		return Evaluation{}, &domain.Error{Code: "REUSED_BASELINE_ATTEMPT_EVIDENCE"}
	}
	if intersects(baseline.EvidenceWorldDigests(), worldDigests) {
		return Evaluation{}, &domain.Error{Code: "REUSED_BASELINE_WORLD_EVIDENCE"}
	}
	if intersects(baseline.EvidenceObservationDigests(), observationDigests) {
		return Evaluation{}, &domain.Error{Code: "REUSED_BASELINE_OBSERVATION_EVIDENCE"}
	}
	value := observed.ArtifactDigest()
	evaluation.observedOutcomeMapDigest = &value
	evaluation.observedBatchDigests = append([]domain.Digest(nil), batchDigests...)
	evaluation.observedAttemptDigests = append([]domain.Digest(nil), attemptDigests...)
	evaluation.observedWorldDigests = append([]domain.Digest(nil), worldDigests...)
	evaluation.observedObservationDigests = append([]domain.Digest(nil), observationDigests...)
	evaluation.logicalNonReuseWithBaseline = true
	if !compare.ComparableForPreservation(baseline, observed) {
		evaluation.decision = unresolvedDecision()
		evaluation.reasonCode = "CANDIDATE_ELIGIBILITY_CHANGED"
		return evaluation, nil
	}
	if compare.SamePreservationMap(baseline, observed) {
		evaluation.decision = Preserves
		evaluation.reasonCode = "EXACT_PRESERVATION_MAP_MATCH"
		return evaluation, nil
	}
	evaluation.decision = Changes
	evaluation.reasonCode = "EXACT_PRESERVATION_MAP_CHANGED"
	return evaluation, nil
}

func validNeighbor(neighbor Neighbor) bool {
	return neighbor.currentStimulusDigest.Valid() && neighbor.stimulusDigest.Valid() &&
		neighbor.reducerSetDigest.Valid() && neighbor.currentStimulusDigest != neighbor.stimulusDigest &&
		neighbor.currentMeasure > 0 && neighbor.measure >= 0 && neighbor.measure < neighbor.currentMeasure
}

func sameRoster(left, right []domain.CandidateExecutionKey) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func hasInvalidOrDuplicate(digests []domain.Digest) bool {
	seen := map[domain.Digest]struct{}{}
	for _, digest := range digests {
		if !digest.Valid() {
			return true
		}
		if _, duplicate := seen[digest]; duplicate {
			return true
		}
		seen[digest] = struct{}{}
	}
	return false
}

func intersects(left, right []domain.Digest) bool {
	seen := make(map[domain.Digest]struct{}, len(left))
	for _, digest := range left {
		seen[digest] = struct{}{}
	}
	for _, digest := range right {
		if _, exists := seen[digest]; exists {
			return true
		}
	}
	return false
}

func (e Evaluation) ID() string                     { return e.id }
func (e Evaluation) Neighbor() Neighbor             { return e.neighbor }
func (e Evaluation) Purpose() domain.AttemptPurpose { return e.purpose }
func (e Evaluation) Decision() Decision             { return e.decision }
func (e Evaluation) ReasonCode() string             { return e.reasonCode }

// LogicalNonReuseWithBaseline reports only content-addressed non-overlap with
// the baseline batch set. It is not a claim that execution was physically fresh.
func (e Evaluation) LogicalNonReuseWithBaseline() bool {
	return e.logicalNonReuseWithBaseline
}

func (e Evaluation) ObservedBatchDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observedBatchDigests...)
}

func (e Evaluation) ObservedAttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observedAttemptDigests...)
}

func (e Evaluation) ObservedWorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observedWorldDigests...)
}

func (e Evaluation) ObservedObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observedObservationDigests...)
}

type SweepState string

const (
	SweepComplete   SweepState = "COMPLETE"
	SweepIncomplete SweepState = "INCOMPLETE"
	SweepNotRun     SweepState = "NOT_RUN"
)

type LogicalSweepInput struct {
	State               SweepState
	Baseline            compare.DivergentBaseline
	CurrentStimulus     domain.Digest
	CurrentMeasure      int
	ReducerSetDigest    domain.Digest
	EnumeratedNeighbors []Neighbor
	Evaluations         []Evaluation
	Cancelled           bool
	BudgetExhausted     bool
	BaselineStale       bool
}

// LogicalCompleteSweep proves only an in-memory set equality and decision
// relation. It has no persistence/completion digest and no conversion to a
// reduction grade. U5/U6 must re-establish this from durable store authority.
type LogicalCompleteSweep struct {
	currentStimulus   domain.Digest
	currentMeasure    int
	reducerSetDigest  domain.Digest
	neighborDigests   []domain.Digest
	baselineMapDigest compare.OutcomeArtifactDigest
}

func NewLogicalCompleteSweep(input LogicalSweepInput) (LogicalCompleteSweep, error) {
	if !sweepStateQualifies(input.State) {
		return LogicalCompleteSweep{}, &domain.Error{Code: "INCOMPLETE_FINAL_SWEEP"}
	}
	if input.Cancelled || input.BudgetExhausted || input.BaselineStale {
		return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_NOT_CURRENT_AND_COMPLETE"}
	}
	if !input.Baseline.Valid() || !input.CurrentStimulus.Valid() || !input.ReducerSetDigest.Valid() || input.CurrentMeasure <= 0 {
		return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_LOGICAL_SWEEP_BASELINE"}
	}
	baseline := input.Baseline.OutcomeMap()
	if baseline.StimulusDigest() != input.CurrentStimulus {
		return LogicalCompleteSweep{}, &domain.Error{Code: "LOGICAL_SWEEP_BASELINE_MISMATCH"}
	}
	if len(input.EnumeratedNeighbors) == 0 {
		return LogicalCompleteSweep{}, &domain.Error{Code: "EMPTY_FINAL_SWEEP"}
	}

	enumerated := make([]domain.Digest, 0, len(input.EnumeratedNeighbors))
	neighborByDigest := map[domain.Digest]Neighbor{}
	for _, neighbor := range input.EnumeratedNeighbors {
		if !validNeighbor(neighbor) || neighbor.currentStimulusDigest != input.CurrentStimulus ||
			neighbor.currentMeasure != input.CurrentMeasure || neighbor.reducerSetDigest != input.ReducerSetDigest {
			return LogicalCompleteSweep{}, &domain.Error{Code: "LOGICAL_SWEEP_NEIGHBOR_BINDING_MISMATCH"}
		}
		if _, duplicate := neighborByDigest[neighbor.stimulusDigest]; duplicate {
			return LogicalCompleteSweep{}, &domain.Error{Code: "DUPLICATE_ENUMERATED_NEIGHBOR"}
		}
		neighborByDigest[neighbor.stimulusDigest] = neighbor
		enumerated = append(enumerated, neighbor.stimulusDigest)
	}
	sort.Slice(enumerated, func(i, j int) bool { return enumerated[i].String() < enumerated[j].String() })

	changed := make([]domain.Digest, 0, len(input.Evaluations))
	seenEvaluationIDs := map[string]struct{}{}
	seenAttempts := map[domain.Digest]struct{}{}
	seenWorlds := map[domain.Digest]struct{}{}
	seenObservations := map[domain.Digest]struct{}{}
	for _, evaluation := range input.Evaluations {
		if evaluation.id == "" {
			return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_SWEEP_EVALUATION"}
		}
		if _, duplicate := seenEvaluationIDs[evaluation.id]; duplicate {
			return LogicalCompleteSweep{}, &domain.Error{Code: "DUPLICATE_SWEEP_EVALUATION"}
		}
		seenEvaluationIDs[evaluation.id] = struct{}{}
		if evaluation.purpose != domain.AttemptFinalSweep {
			return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_EVALUATION_PURPOSE_MISMATCH"}
		}
		neighbor, exists := neighborByDigest[evaluation.neighbor.stimulusDigest]
		if !exists || !sameNeighbor(neighbor, evaluation.neighbor) {
			return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_NEIGHBOR_SET_MISMATCH"}
		}
		if evaluation.baselineOutcomeMapDigest != baseline.ArtifactDigest() ||
			evaluation.baselinePreservationDigest != baseline.PreservationDigest() ||
			evaluation.comparisonEnvelopeDigest != baseline.EnvelopeDigest() ||
			!sameRoster(evaluation.candidateRoster, baseline.CandidateRoster()) {
			return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_BASELINE_MISMATCH"}
		}
		if evaluation.decision != Changes {
			if evaluation.decision == Unresolved {
				return LogicalCompleteSweep{}, &domain.Error{Code: "UNRESOLVED_FINAL_SWEEP"}
			}
			return LogicalCompleteSweep{}, &domain.Error{Code: "PRESERVING_DIRECT_NEIGHBOR"}
		}
		if !evaluation.logicalNonReuseWithBaseline {
			return LogicalCompleteSweep{}, &domain.Error{Code: "UNESTABLISHED_LOGICAL_BATCH_NONREUSE"}
		}
		for _, digest := range evaluation.observedAttemptDigests {
			if !digest.Valid() {
				return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_FINAL_SWEEP_ATTEMPT_EVIDENCE"}
			}
			if _, reused := seenAttempts[digest]; reused {
				return LogicalCompleteSweep{}, &domain.Error{Code: "REUSED_FINAL_SWEEP_ATTEMPT_EVIDENCE"}
			}
			seenAttempts[digest] = struct{}{}
		}
		for _, digest := range evaluation.observedWorldDigests {
			if !digest.Valid() {
				return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_FINAL_SWEEP_WORLD_EVIDENCE"}
			}
			if _, reused := seenWorlds[digest]; reused {
				return LogicalCompleteSweep{}, &domain.Error{Code: "REUSED_FINAL_SWEEP_WORLD_EVIDENCE"}
			}
			seenWorlds[digest] = struct{}{}
		}
		for _, digest := range evaluation.observedObservationDigests {
			if !digest.Valid() {
				return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_FINAL_SWEEP_OBSERVATION_EVIDENCE"}
			}
			if _, reused := seenObservations[digest]; reused {
				return LogicalCompleteSweep{}, &domain.Error{Code: "REUSED_FINAL_SWEEP_OBSERVATION_EVIDENCE"}
			}
			seenObservations[digest] = struct{}{}
		}
		changed = append(changed, evaluation.neighbor.stimulusDigest)
	}
	sort.Slice(changed, func(i, j int) bool { return changed[i].String() < changed[j].String() })
	if !sameDigestSet(enumerated, changed) {
		return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_NEIGHBOR_SET_MISMATCH"}
	}
	return LogicalCompleteSweep{
		currentStimulus:   input.CurrentStimulus,
		currentMeasure:    input.CurrentMeasure,
		reducerSetDigest:  input.ReducerSetDigest,
		neighborDigests:   append([]domain.Digest(nil), enumerated...),
		baselineMapDigest: baseline.ArtifactDigest(),
	}, nil
}

func sweepStateQualifies(state SweepState) bool {
	// MUTATION_ANCHOR: incomplete-sweep-cannot-prove-one-minimal
	return state == SweepComplete
}

func sameNeighbor(left, right Neighbor) bool {
	return left.currentStimulusDigest == right.currentStimulusDigest && left.currentMeasure == right.currentMeasure &&
		left.stimulusDigest == right.stimulusDigest && left.measure == right.measure &&
		left.reducerSetDigest == right.reducerSetDigest
}

func sameDigestSet(left, right []domain.Digest) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (s LogicalCompleteSweep) NeighborDigests() []domain.Digest {
	return append([]domain.Digest(nil), s.neighborDigests...)
}

func (s LogicalCompleteSweep) ReducerSetDigest() domain.Digest { return s.reducerSetDigest }
func (s LogicalCompleteSweep) BaselineMapDigest() compare.OutcomeArtifactDigest {
	return s.baselineMapDigest
}
