// Package reduce defines pure, logical, tri-valued reduction evidence. Durable
// completion and ONE_MINIMAL_UNDER authority intentionally do not exist in U1;
// they require the store-backed transcript and CAS lineage introduced later.
package reduce

import (
	"bytes"
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

// UnresolvedReason is the closed set of edge/control outcomes that may prevent
// a product comparison. Exact-map incomparability has its own internally
// derived reason and is never caller-authored through this type.
type UnresolvedReason string

const (
	ReasonTimeout             UnresolvedReason = "TIMEOUT"
	ReasonCancelled           UnresolvedReason = "CANCELLED"
	ReasonStale               UnresolvedReason = "STALE"
	ReasonUnstable            UnresolvedReason = "UNSTABLE"
	ReasonIncomplete          UnresolvedReason = "INCOMPLETE"
	ReasonTeardownError       UnresolvedReason = "TEARDOWN_ERROR"
	ReasonEvaluatorError      UnresolvedReason = "EVALUATOR_ERROR"
	ReasonWallBudgetExhausted UnresolvedReason = "WALL_BUDGET_EXHAUSTED"
	ReasonReductionCancelled  UnresolvedReason = "REDUCTION_CANCELLED"
	ReasonNoObservation       UnresolvedReason = "NO_OBSERVATION"

	// These reasons are terminal reducer/evaluator protocol refusals. They make
	// the attempted proposal visible without upgrading malformed, over-budget,
	// or reused evaluator output into product evidence.
	ReasonEvaluatorExceededBudget   UnresolvedReason = "EVALUATOR_EXCEEDED_CANDIDATE_TRIAL_BUDGET"
	ReasonObservedMapWithoutTrials  UnresolvedReason = "OBSERVED_MAP_WITHOUT_CANDIDATE_TRIALS"
	ReasonUnresolvedEvidenceMissing UnresolvedReason = "UNRESOLVED_EVALUATION_MISSING_EVIDENCE"
	ReasonInvalidEvaluatorResult    UnresolvedReason = "INVALID_EVALUATOR_RESULT"
	ReasonReusedEvaluationEvidence  UnresolvedReason = "REUSED_EVALUATION_EVIDENCE"
)

func (r UnresolvedReason) Valid() bool {
	switch r {
	case ReasonTimeout, ReasonCancelled, ReasonStale, ReasonUnstable, ReasonIncomplete,
		ReasonTeardownError, ReasonEvaluatorError, ReasonWallBudgetExhausted,
		ReasonReductionCancelled, ReasonNoObservation, ReasonEvaluatorExceededBudget,
		ReasonObservedMapWithoutTrials, ReasonUnresolvedEvidenceMissing,
		ReasonInvalidEvaluatorResult, ReasonReusedEvaluationEvidence:
		return true
	default:
		return false
	}
}

func isTerminalProtocolReason(reason UnresolvedReason) bool {
	switch reason {
	case ReasonEvaluatorExceededBudget, ReasonObservedMapWithoutTrials,
		ReasonUnresolvedEvidenceMissing, ReasonInvalidEvaluatorResult,
		ReasonReusedEvaluationEvidence:
		return true
	default:
		return false
	}
}

// UnresolvedEvidence retains fresh control-path lineage when no complete
// CandidateOutcomeMap exists. It is immutable and construction-safe; every
// evidence domain must be present together.
type UnresolvedEvidence struct {
	batchDigests       []domain.Digest
	attemptDigests     []domain.Digest
	worldDigests       []domain.Digest
	observationDigests []domain.Digest
}

func NewUnresolvedEvidence(batch, attempt, world, observation []domain.Digest) (UnresolvedEvidence, error) {
	if len(batch) == 0 || len(attempt) == 0 || len(world) == 0 || len(observation) == 0 ||
		hasInvalidOrDuplicate(batch) || hasInvalidOrDuplicate(attempt) ||
		hasInvalidOrDuplicate(world) || hasInvalidOrDuplicate(observation) {
		return UnresolvedEvidence{}, &domain.Error{Code: "INVALID_UNRESOLVED_EVIDENCE"}
	}
	return UnresolvedEvidence{
		batchDigests: append([]domain.Digest(nil), batch...), attemptDigests: append([]domain.Digest(nil), attempt...),
		worldDigests: append([]domain.Digest(nil), world...), observationDigests: append([]domain.Digest(nil), observation...),
	}, nil
}

func (e UnresolvedEvidence) Present() bool {
	return len(e.batchDigests) != 0 || len(e.attemptDigests) != 0 || len(e.worldDigests) != 0 || len(e.observationDigests) != 0
}
func (e UnresolvedEvidence) Valid() bool {
	if !e.Present() {
		return false
	}
	rebuilt, err := NewUnresolvedEvidence(e.batchDigests, e.attemptDigests, e.worldDigests, e.observationDigests)
	return err == nil && slicesEqualDigests(rebuilt.batchDigests, e.batchDigests) &&
		slicesEqualDigests(rebuilt.attemptDigests, e.attemptDigests) &&
		slicesEqualDigests(rebuilt.worldDigests, e.worldDigests) &&
		slicesEqualDigests(rebuilt.observationDigests, e.observationDigests)
}
func (e UnresolvedEvidence) BatchDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.batchDigests...)
}
func (e UnresolvedEvidence) AttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.attemptDigests...)
}
func (e UnresolvedEvidence) WorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.worldDigests...)
}
func (e UnresolvedEvidence) ObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observationDigests...)
}

func slicesEqualDigests(left, right []domain.Digest) bool {
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

func unresolvedDecision() Decision {
	// MUTATION_ANCHOR: unresolved-is-not-changes
	return Unresolved
}

// Neighbor is an opaque direct-neighbor proposal bound to the exact current
// stimulus, well-founded measure, and reducer-set identity.
type Neighbor struct {
	currentStimulusDigest domain.Digest
	currentMeasure        Measure
	stimulusDigest        domain.Digest
	measure               Measure
	rule                  ReducerRule
	locus                 string
	transformPriority     uint64
	reducerSetDigest      domain.Digest
	digest                domain.Digest
	canonicalBytes        []byte
}

type NeighborInput struct {
	CurrentStimulus   domain.Digest
	CurrentMeasure    Measure
	Stimulus          domain.Digest
	Measure           Measure
	Rule              ReducerRule
	Locus             string
	TransformPriority uint64
	ReducerSet        ReducerSet
}

type neighborIdentity struct {
	SchemaVersion         string `json:"schema_version"`
	Kind                  string `json:"kind"`
	CurrentStimulusDigest string `json:"current_stimulus_digest"`
	CurrentMeasureDigest  string `json:"current_measure_digest"`
	StimulusDigest        string `json:"stimulus_digest"`
	MeasureDigest         string `json:"measure_digest"`
	RuleName              string `json:"rule_name"`
	RuleVersion           string `json:"rule_version"`
	RuleDigest            string `json:"rule_digest"`
	Locus                 string `json:"locus"`
	TransformPriority     int64  `json:"transform_priority"`
	ReducerSetDigest      string `json:"reducer_set_digest"`
}

const maxReducerTransformPriority = 1024

func NewNeighbor(input NeighborInput) (Neighbor, error) {
	// MUTATION_ANCHOR: direct-neighbor-measure-must-strictly-decrease
	if !input.CurrentStimulus.Valid() || !input.Stimulus.Valid() || input.CurrentStimulus == input.Stimulus ||
		!input.CurrentMeasure.Valid() || !input.Measure.Valid() || !strictlyDecreases(input.CurrentMeasure, input.Measure) ||
		!input.Rule.Valid() || !reducerLocusPattern.MatchString(input.Locus) || !input.ReducerSet.Valid() ||
		input.TransformPriority > maxReducerTransformPriority || !input.ReducerSet.Contains(input.Rule) ||
		input.CurrentMeasure.DefinitionDigest() != input.ReducerSet.MeasureDefinitionDigest() {
		return Neighbor{}, &domain.Error{Code: "INVALID_DIRECT_NEIGHBOR"}
	}
	digest, canonicalBytes, err := digestTyped("ReductionProposal", neighborIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReductionProposal",
		CurrentStimulusDigest: input.CurrentStimulus.String(), CurrentMeasureDigest: input.CurrentMeasure.Digest().String(),
		StimulusDigest: input.Stimulus.String(), MeasureDigest: input.Measure.Digest().String(),
		RuleName: input.Rule.Name(), RuleVersion: input.Rule.Version(), RuleDigest: input.Rule.Digest().String(),
		Locus: input.Locus, TransformPriority: int64(input.TransformPriority), ReducerSetDigest: input.ReducerSet.Digest().String(),
	})
	if err != nil {
		return Neighbor{}, err
	}
	return Neighbor{
		currentStimulusDigest: input.CurrentStimulus, currentMeasure: input.CurrentMeasure,
		stimulusDigest: input.Stimulus, measure: input.Measure, rule: input.Rule, locus: input.Locus,
		transformPriority: input.TransformPriority, reducerSetDigest: input.ReducerSet.Digest(), digest: digest, canonicalBytes: canonicalBytes,
	}, nil
}

func (n Neighbor) CurrentStimulusDigest() domain.Digest { return n.currentStimulusDigest }
func (n Neighbor) CurrentMeasure() Measure              { return n.currentMeasure }
func (n Neighbor) StimulusDigest() domain.Digest        { return n.stimulusDigest }
func (n Neighbor) Measure() Measure                     { return n.measure }
func (n Neighbor) Rule() ReducerRule                    { return n.rule }
func (n Neighbor) Locus() string                        { return n.locus }
func (n Neighbor) TransformPriority() uint64            { return n.transformPriority }
func (n Neighbor) ReducerSetDigest() domain.Digest      { return n.reducerSetDigest }
func (n Neighbor) Digest() domain.Digest                { return n.digest }
func (n Neighbor) CanonicalBytes() []byte               { return append([]byte(nil), n.canonicalBytes...) }

type Evaluation struct {
	id                          string
	neighbor                    Neighbor
	purpose                     domain.AttemptPurpose
	observedBatchDigests        []domain.Digest
	observedAttemptDigests      []domain.Digest
	observedWorldDigests        []domain.Digest
	observedObservationDigests  []domain.Digest
	observedOutcomeMapDigest    *compare.OutcomeArtifactDigest
	observedPreservationDigest  *compare.PreservationMapDigest
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
	UnresolvedReason   UnresolvedReason
	UnresolvedEvidence UnresolvedEvidence
}

func NewEvaluation(input EvaluationInput) (Evaluation, error) {
	return newEvaluation(input, domain.AttemptReduction, false)
}

// NewFinalSweepEvaluation is the only constructor whose observed evidence may
// carry FINAL_SWEEP purpose. Ordinary reduction and final-sweep evidence are
// deliberately distinct even when all other structural facts match.
func NewFinalSweepEvaluation(input EvaluationInput) (Evaluation, error) {
	return newEvaluation(input, domain.AttemptFinalSweep, false)
}

func newProtocolRefusalEvaluation(input EvaluationInput, requiredPurpose domain.AttemptPurpose) (Evaluation, error) {
	if !isTerminalProtocolReason(input.UnresolvedReason) {
		return Evaluation{}, &domain.Error{Code: "INVALID_REDUCTION_PROTOCOL_REFUSAL_REASON"}
	}
	return newEvaluation(input, requiredPurpose, true)
}

func newEvaluation(input EvaluationInput, requiredPurpose domain.AttemptPurpose, allowTerminalProtocolReason bool) (Evaluation, error) {
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
		if !input.UnresolvedReason.Valid() {
			return Evaluation{}, &domain.Error{Code: "MISSING_UNRESOLVED_REASON"}
		}
		if isTerminalProtocolReason(input.UnresolvedReason) && !allowTerminalProtocolReason {
			return Evaluation{}, &domain.Error{Code: "RESERVED_REDUCTION_PROTOCOL_REFUSAL_REASON"}
		}
		if input.UnresolvedEvidence.Present() {
			if !input.UnresolvedEvidence.Valid() {
				return Evaluation{}, &domain.Error{Code: "INVALID_UNRESOLVED_EVIDENCE"}
			}
			if intersects(baseline.BatchDigests(), input.UnresolvedEvidence.batchDigests) ||
				intersects(baseline.EvidenceAttemptDigests(), input.UnresolvedEvidence.attemptDigests) ||
				intersects(baseline.EvidenceWorldDigests(), input.UnresolvedEvidence.worldDigests) ||
				intersects(baseline.EvidenceObservationDigests(), input.UnresolvedEvidence.observationDigests) {
				return Evaluation{}, &domain.Error{Code: "REUSED_BASELINE_UNRESOLVED_EVIDENCE"}
			}
			evaluation.observedBatchDigests = input.UnresolvedEvidence.BatchDigests()
			evaluation.observedAttemptDigests = input.UnresolvedEvidence.AttemptDigests()
			evaluation.observedWorldDigests = input.UnresolvedEvidence.WorldDigests()
			evaluation.observedObservationDigests = input.UnresolvedEvidence.ObservationDigests()
			evaluation.logicalNonReuseWithBaseline = true
		}
		evaluation.decision = unresolvedDecision()
		evaluation.reasonCode = string(input.UnresolvedReason)
		return evaluation, nil
	}
	if input.UnresolvedReason != "" || input.UnresolvedEvidence.Present() {
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
	preservationValue := observed.PreservationDigest()
	evaluation.observedOutcomeMapDigest = &value
	evaluation.observedPreservationDigest = &preservationValue
	evaluation.observedBatchDigests = append([]domain.Digest(nil), batchDigests...)
	evaluation.observedAttemptDigests = append([]domain.Digest(nil), attemptDigests...)
	evaluation.observedWorldDigests = append([]domain.Digest(nil), worldDigests...)
	evaluation.observedObservationDigests = append([]domain.Digest(nil), observationDigests...)
	evaluation.logicalNonReuseWithBaseline = true
	assessment := compare.AssessPreservation(baseline, observed)
	if !assessment.Valid() {
		return Evaluation{}, &domain.Error{Code: "INVALID_PRESERVATION_ASSESSMENT"}
	}
	switch assessment.Relation() {
	case compare.PreservationUnresolved:
		evaluation.decision = unresolvedDecision()
	case compare.PreservationEqual:
		evaluation.decision = Preserves
	case compare.PreservationDifferent:
		evaluation.decision = Changes
	default:
		return Evaluation{}, &domain.Error{Code: "INVALID_PRESERVATION_ASSESSMENT"}
	}
	evaluation.reasonCode = assessment.ReasonCode()
	return evaluation, nil
}

func validNeighbor(neighbor Neighbor) bool {
	return neighbor.currentStimulusDigest.Valid() && neighbor.stimulusDigest.Valid() && neighbor.reducerSetDigest.Valid() &&
		neighbor.currentStimulusDigest != neighbor.stimulusDigest && neighbor.currentMeasure.Valid() && neighbor.measure.Valid() &&
		strictlyDecreases(neighbor.currentMeasure, neighbor.measure) && neighbor.rule.Valid() &&
		reducerLocusPattern.MatchString(neighbor.locus) && neighbor.transformPriority <= maxReducerTransformPriority &&
		neighbor.digest.Valid() && len(neighbor.canonicalBytes) > 0
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
func (e Evaluation) BaselineOutcomeMapDigest() compare.OutcomeArtifactDigest {
	return e.baselineOutcomeMapDigest
}
func (e Evaluation) BaselinePreservationDigest() compare.PreservationMapDigest {
	return e.baselinePreservationDigest
}
func (e Evaluation) ComparisonEnvelopeDigest() domain.Digest { return e.comparisonEnvelopeDigest }
func (e Evaluation) CandidateRoster() []domain.CandidateExecutionKey {
	return append([]domain.CandidateExecutionKey(nil), e.candidateRoster...)
}
func (e Evaluation) ObservedOutcomeMapDigest() (compare.OutcomeArtifactDigest, bool) {
	if e.observedOutcomeMapDigest == nil {
		return compare.OutcomeArtifactDigest{}, false
	}
	return *e.observedOutcomeMapDigest, true
}
func (e Evaluation) ObservedPreservationDigest() (compare.PreservationMapDigest, bool) {
	if e.observedPreservationDigest == nil {
		return compare.PreservationMapDigest{}, false
	}
	return *e.observedPreservationDigest, true
}

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
	CurrentMeasure      Measure
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
	currentStimulus            domain.Digest
	currentMeasure             Measure
	reducerSetDigest           domain.Digest
	neighborDigests            []domain.Digest
	baselineMapDigest          compare.OutcomeArtifactDigest
	baselinePreservationDigest compare.PreservationMapDigest
}

func NewLogicalCompleteSweep(input LogicalSweepInput) (LogicalCompleteSweep, error) {
	if !sweepStateQualifies(input.State) {
		return LogicalCompleteSweep{}, &domain.Error{Code: "INCOMPLETE_FINAL_SWEEP"}
	}
	if input.Cancelled || input.BudgetExhausted || input.BaselineStale {
		return LogicalCompleteSweep{}, &domain.Error{Code: "FINAL_SWEEP_NOT_CURRENT_AND_COMPLETE"}
	}
	if !input.Baseline.Valid() || !input.CurrentStimulus.Valid() || !input.ReducerSetDigest.Valid() || !input.CurrentMeasure.Valid() {
		return LogicalCompleteSweep{}, &domain.Error{Code: "INVALID_LOGICAL_SWEEP_BASELINE"}
	}
	baseline := input.Baseline.OutcomeMap()
	if baseline.StimulusDigest() != input.CurrentStimulus {
		return LogicalCompleteSweep{}, &domain.Error{Code: "LOGICAL_SWEEP_BASELINE_MISMATCH"}
	}
	enumerated := make([]domain.Digest, 0, len(input.EnumeratedNeighbors))
	neighborByDigest := map[domain.Digest]Neighbor{}
	for _, neighbor := range input.EnumeratedNeighbors {
		if !validNeighbor(neighbor) || neighbor.currentStimulusDigest != input.CurrentStimulus ||
			neighbor.currentMeasure.digest != input.CurrentMeasure.digest || neighbor.reducerSetDigest != input.ReducerSetDigest {
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
		currentStimulus:            input.CurrentStimulus,
		currentMeasure:             input.CurrentMeasure,
		reducerSetDigest:           input.ReducerSetDigest,
		neighborDigests:            append([]domain.Digest(nil), enumerated...),
		baselineMapDigest:          baseline.ArtifactDigest(),
		baselinePreservationDigest: baseline.PreservationDigest(),
	}, nil
}

func sweepStateQualifies(state SweepState) bool {
	// MUTATION_ANCHOR: incomplete-sweep-cannot-prove-one-minimal
	return state == SweepComplete
}

func sameNeighbor(left, right Neighbor) bool {
	return left.currentStimulusDigest == right.currentStimulusDigest && left.currentMeasure.digest == right.currentMeasure.digest &&
		left.stimulusDigest == right.stimulusDigest && left.measure.digest == right.measure.digest &&
		left.rule.digest == right.rule.digest && left.locus == right.locus && left.transformPriority == right.transformPriority &&
		left.reducerSetDigest == right.reducerSetDigest &&
		left.digest == right.digest && bytes.Equal(left.canonicalBytes, right.canonicalBytes)
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

func (s LogicalCompleteSweep) CurrentStimulusDigest() domain.Digest { return s.currentStimulus }
func (s LogicalCompleteSweep) CurrentMeasure() Measure              { return s.currentMeasure }
func (s LogicalCompleteSweep) ReducerSetDigest() domain.Digest      { return s.reducerSetDigest }
func (s LogicalCompleteSweep) BaselineMapDigest() compare.OutcomeArtifactDigest {
	return s.baselineMapDigest
}
func (s LogicalCompleteSweep) BaselinePreservationMapDigest() compare.PreservationMapDigest {
	return s.baselinePreservationDigest
}
