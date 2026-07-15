package reduce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxProposalLimit = 200
	maxTrialLimit    = 2000
	maxWallLimit     = time.Hour
)

type Budget struct {
	proposalLimit uint64
	trialLimit    uint64
	wallLimit     time.Duration
	worldPlan     domain.Digest
}

func NewBudget(proposalLimit, candidateTrialLimit uint64, wallLimit time.Duration) (Budget, error) {
	if proposalLimit == 0 || proposalLimit > maxProposalLimit || candidateTrialLimit == 0 ||
		candidateTrialLimit > maxTrialLimit || wallLimit <= 0 || wallLimit > maxWallLimit {
		return Budget{}, &domain.Error{Code: "INVALID_REDUCTION_BUDGET"}
	}
	return Budget{proposalLimit: proposalLimit, trialLimit: candidateTrialLimit, wallLimit: wallLimit}, nil
}
func (b Budget) Valid() bool {
	rebuilt, err := NewBudget(b.proposalLimit, b.trialLimit, b.wallLimit)
	return err == nil && (b.worldPlan == rebuilt.worldPlan || b.worldPlan.Valid())
}
func (b Budget) ProposalLimit() uint64       { return b.proposalLimit }
func (b Budget) CandidateTrialLimit() uint64 { return b.trialLimit }
func (b Budget) WallLimit() time.Duration    { return b.wallLimit }
func (b Budget) WorldPlanDigest() (domain.Digest, bool) {
	return b.worldPlan, b.Valid() && b.worldPlan.Valid()
}

// NewBudgetFromWorldPlan materializes the exact compiled shrink defaults and
// retains the opaque plan's identity in live memory. Serialized run data remains
// inert; only the live plan-bound value can participate in a strong finalization.
func NewBudgetFromWorldPlan(plan domain.WorldPlan) (Budget, error) {
	if !plan.Digest().Valid() {
		return Budget{}, &domain.Error{Code: "INVALID_REDUCTION_BUDGET_PLAN"}
	}
	budgets := plan.Budgets()
	schedule := plan.RepeatSchedule()
	reservedTrials := budgets.CandidateCount * (schedule.DiscoveryRepeats + schedule.ConfirmationRepeats)
	shrinkTrials := budgets.TotalCandidateTrials - reservedTrials
	if shrinkTrials <= 0 {
		return Budget{}, &domain.Error{
			Code: "INVALID_REDUCTION_BUDGET_PLAN", Detail: "compiled plan reserves no candidate trials for reduction",
		}
	}
	result, err := NewBudget(
		uint64(budgets.ProposedShrinkStimuli),
		uint64(shrinkTrials),
		time.Duration(budgets.ShrinkWallMS)*time.Millisecond,
	)
	if err != nil {
		return Budget{}, &domain.Error{Code: "INVALID_REDUCTION_BUDGET_PLAN", Detail: err.Error()}
	}
	result.worldPlan = plan.Digest()
	return result, nil
}

type TypedProposal[S any] struct {
	Stimulus S
	Neighbor Neighbor
}

type ReferenceFunc[S any] func(S) (domain.Digest, Measure, bool)
type EnumerateFunc[S any] func(context.Context, S) ([]TypedProposal[S], error)

type EvaluationAllowance struct {
	RemainingCandidateTrials uint64
	RemainingProposals       uint64
	WallDeadline             time.Time
}

type EvaluationObservation struct {
	OutcomeMap         *compare.CandidateOutcomeMap
	UnresolvedReason   UnresolvedReason
	UnresolvedEvidence UnresolvedEvidence
	CandidateTrials    uint64
	terminalRefusal    bool
}

type EvaluateFunc[S any] func(context.Context, S, Neighbor, domain.AttemptPurpose, EvaluationAllowance) (EvaluationObservation, error)

type Clock interface{ Now() time.Time }
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

type RunInput[S any] struct {
	Original   S
	Reference  ReferenceFunc[S]
	Enumerate  EnumerateFunc[S]
	Evaluate   EvaluateFunc[S]
	Baseline   compare.DivergentBaseline
	ReducerSet ReducerSet
	Budget     Budget
	Clock      Clock
}

type DraftGrade string

const (
	GradeUnchanged DraftGrade = "UNCHANGED"
	GradeBestKnown DraftGrade = "BEST_KNOWN"
)

type FinalSweepState string

const (
	FinalSweepComplete   FinalSweepState = "COMPLETE"
	FinalSweepIncomplete FinalSweepState = "INCOMPLETE"
	FinalSweepNotRun     FinalSweepState = "NOT_RUN"
)

type TranscriptEntry struct {
	evaluation      Evaluation
	proposalCount   uint64
	trialCount      uint64
	candidateTrials uint64
}

func (e TranscriptEntry) Evaluation() Evaluation  { return e.evaluation }
func (e TranscriptEntry) ProposalCount() uint64   { return e.proposalCount }
func (e TranscriptEntry) TrialCount() uint64      { return e.trialCount }
func (e TranscriptEntry) CandidateTrials() uint64 { return e.candidateTrials }

type Transcript struct {
	digest         domain.Digest
	canonicalBytes []byte
	entries        []TranscriptEntry
	acceptedPath   []domain.Digest
	limitations    []string
	finalState     FinalSweepState
	finalNeighbors []domain.Digest
}

func (t Transcript) Valid() bool {
	return t.digest.Valid() && len(t.canonicalBytes) > 0 && len(t.limitations) <= maxProposalLimit+4
}
func (t Transcript) Digest() domain.Digest      { return t.digest }
func (t Transcript) CanonicalBytes() []byte     { return append([]byte(nil), t.canonicalBytes...) }
func (t Transcript) Entries() []TranscriptEntry { return append([]TranscriptEntry(nil), t.entries...) }
func (t Transcript) AcceptedPath() []domain.Digest {
	return append([]domain.Digest(nil), t.acceptedPath...)
}
func (t Transcript) Limitations() []string            { return append([]string{}, t.limitations...) }
func (t Transcript) FinalSweepState() FinalSweepState { return t.finalState }
func (t Transcript) FinalNeighborDigests() []domain.Digest {
	return append([]domain.Digest(nil), t.finalNeighbors...)
}

type ReductionRun struct {
	digest            domain.Digest
	canonicalBytes    []byte
	baseline          compare.DivergentBaseline
	originalStimulus  domain.Digest
	minimizedStimulus domain.Digest
	originalMeasure   Measure
	minimizedMeasure  Measure
	reducerSet        ReducerSet
	budget            Budget
	transcript        Transcript
	draftGrade        DraftGrade
	finalSweep        *LogicalCompleteSweep
	finalNeighbors    []Neighbor
	finalEvaluations  []Evaluation
}

func (r ReductionRun) Valid() bool {
	return r.digest.Valid() && len(r.canonicalBytes) > 0 && r.baseline.Valid() && r.originalStimulus.Valid() &&
		r.minimizedStimulus.Valid() && r.originalMeasure.Valid() && r.minimizedMeasure.Valid() &&
		r.reducerSet.Valid() && r.budget.Valid() && r.transcript.Valid() &&
		(r.draftGrade == GradeUnchanged || r.draftGrade == GradeBestKnown)
}
func (r ReductionRun) Digest() domain.Digest                  { return r.digest }
func (r ReductionRun) CanonicalBytes() []byte                 { return append([]byte(nil), r.canonicalBytes...) }
func (r ReductionRun) OriginalStimulusDigest() domain.Digest  { return r.originalStimulus }
func (r ReductionRun) MinimizedStimulusDigest() domain.Digest { return r.minimizedStimulus }
func (r ReductionRun) OriginalMeasure() Measure               { return r.originalMeasure }
func (r ReductionRun) MinimizedMeasure() Measure              { return r.minimizedMeasure }
func (r ReductionRun) ReducerSet() ReducerSet                 { return r.reducerSet }
func (r ReductionRun) Budget() Budget                         { return r.budget }
func (r ReductionRun) BaselinePlanDigest() domain.Digest      { return r.baseline.OutcomeMap().PlanDigest() }
func (r ReductionRun) Baseline() compare.DivergentBaseline    { return r.baseline }
func (r ReductionRun) Transcript() Transcript                 { return r.transcript }
func (r ReductionRun) DraftGrade() DraftGrade                 { return r.draftGrade }
func (r ReductionRun) HasAcceptedReduction() bool             { return r.originalStimulus != r.minimizedStimulus }

func (r ReductionRun) CompletedSweepDraft() (CompletedSweepDraft, bool, error) {
	// A complete-looking final sweep cannot erase an earlier unresolved,
	// cancelled, stale, or budget-limited observation. Any recorded limitation
	// keeps the run at its honest weak grade and outside durable sweep authority.
	if !r.Valid() || r.finalSweep == nil || len(r.transcript.limitations) != 0 {
		return CompletedSweepDraft{}, false, nil
	}
	draft, err := NewCompletedSweepDraft(r.digest, r.transcript.digest, *r.finalSweep, r.finalNeighbors, r.finalEvaluations)
	return draft, err == nil, err
}

type evidenceLedger struct {
	batch       map[domain.Digest]struct{}
	attempt     map[domain.Digest]struct{}
	world       map[domain.Digest]struct{}
	observation map[domain.Digest]struct{}
}

func newEvidenceLedger(baseline compare.CandidateOutcomeMap) evidenceLedger {
	ledger := evidenceLedger{batch: map[domain.Digest]struct{}{}, attempt: map[domain.Digest]struct{}{}, world: map[domain.Digest]struct{}{}, observation: map[domain.Digest]struct{}{}}
	ledger.addAll(ledger.batch, baseline.BatchDigests())
	ledger.addAll(ledger.attempt, baseline.EvidenceAttemptDigests())
	ledger.addAll(ledger.world, baseline.EvidenceWorldDigests())
	ledger.addAll(ledger.observation, baseline.EvidenceObservationDigests())
	return ledger
}
func (l *evidenceLedger) addAll(target map[domain.Digest]struct{}, values []domain.Digest) {
	for _, value := range values {
		target[value] = struct{}{}
	}
}
func (l *evidenceLedger) admit(evaluation Evaluation) bool {
	sets := []struct {
		target map[domain.Digest]struct{}
		values []domain.Digest
	}{
		{l.batch, evaluation.observedBatchDigests}, {l.attempt, evaluation.observedAttemptDigests},
		{l.world, evaluation.observedWorldDigests}, {l.observation, evaluation.observedObservationDigests},
	}
	for _, set := range sets {
		for _, value := range set.values {
			if _, duplicate := set.target[value]; duplicate {
				return false
			}
		}
	}
	for _, set := range sets {
		l.addAll(set.target, set.values)
	}
	return true
}

type runState[S any] struct {
	input            RunInput[S]
	parentContext    context.Context
	clock            Clock
	deadline         time.Time
	proposalsUsed    uint64
	trialsUsed       uint64
	current          S
	currentDigest    domain.Digest
	currentMeasure   Measure
	currentBaseline  compare.DivergentBaseline
	originalDigest   domain.Digest
	originalMeasure  Measure
	entries          []TranscriptEntry
	accepted         []domain.Digest
	limitations      []string
	ledger           evidenceLedger
	finalState       FinalSweepState
	finalNeighbors   []Neighbor
	finalEvaluations []Evaluation
	completeSweep    *LogicalCompleteSweep
	fatalErr         error
}

func Run[S any](ctx context.Context, input RunInput[S]) (ReductionRun, error) {
	if ctx == nil || input.Reference == nil || input.Enumerate == nil || input.Evaluate == nil ||
		!input.Baseline.Valid() || !input.ReducerSet.Valid() || !input.Budget.Valid() {
		return ReductionRun{}, &domain.Error{Code: "INVALID_REDUCTION_RUN_INPUT"}
	}
	digest, measure, ok := input.Reference(input.Original)
	if !ok || !digest.Valid() || !measure.Valid() || measure.DefinitionDigest() != input.ReducerSet.MeasureDefinitionDigest() ||
		input.Baseline.OutcomeMap().StimulusDigest() != digest {
		return ReductionRun{}, &domain.Error{Code: "REDUCTION_ORIGINAL_BINDING_MISMATCH"}
	}
	if planDigest, bound := input.Budget.WorldPlanDigest(); bound && planDigest != input.Baseline.OutcomeMap().PlanDigest() {
		return ReductionRun{}, &domain.Error{Code: "REDUCTION_BUDGET_PLAN_MISMATCH"}
	}
	clock := input.Clock
	if clock == nil {
		clock = wallClock{}
	}
	started := clock.Now()
	state := &runState[S]{
		input: input, parentContext: ctx, clock: clock, deadline: started.Add(input.Budget.wallLimit),
		current: input.Original, currentDigest: digest, currentMeasure: measure, currentBaseline: input.Baseline,
		originalDigest: digest, originalMeasure: measure, ledger: newEvidenceLedger(input.Baseline.OutcomeMap()),
		finalState: FinalSweepNotRun,
	}
	wallContext, cancel := context.WithTimeout(ctx, input.Budget.wallLimit)
	defer cancel()
	state.search(wallContext)
	return state.finish()
}

func (s *runState[S]) search(ctx context.Context) {
	for {
		if reason := s.stopReason(ctx); reason != "" {
			s.limit(string(reason))
			return
		}
		proposals, err := s.enumerate(ctx)
		if reason := s.stopReason(ctx); reason != "" {
			s.limit(string(reason))
			return
		}
		if err != nil {
			s.limit("NEIGHBOR_ENUMERATION_FAILED")
			return
		}
		accepted := false
		for _, proposal := range proposals {
			evaluation, observation, ok := s.evaluate(ctx, proposal, domain.AttemptReduction)
			if !ok {
				return
			}
			s.record(evaluation, observation.CandidateTrials)
			if observation.terminalRefusal {
				s.limit(evaluation.reasonCode)
				return
			}
			s.noteDecisionLimitation(evaluation)
			if evaluation.decision != Preserves {
				continue
			}
			if observation.OutcomeMap == nil {
				s.limit("PRESERVING_EVALUATION_MISSING_MAP")
				return
			}
			baseline, err := compare.RequireDivergence(*observation.OutcomeMap)
			if err != nil {
				s.limit("PRESERVING_EVALUATION_LOST_DIVERGENCE")
				return
			}
			s.current = proposal.Stimulus
			s.currentDigest = proposal.Neighbor.stimulusDigest
			s.currentMeasure = proposal.Neighbor.measure
			s.currentBaseline = baseline
			s.accepted = append(s.accepted, proposal.Neighbor.digest)
			accepted = true
			break
		}
		if accepted {
			continue
		}
		s.finalSweepRun(ctx, proposals)
		return
	}
}

func (s *runState[S]) enumerate(ctx context.Context) ([]TypedProposal[S], error) {
	proposals, err := s.input.Enumerate(ctx, s.current)
	if err != nil {
		return nil, err
	}
	result := append([]TypedProposal[S](nil), proposals...)
	seenProposal := map[domain.Digest]struct{}{}
	seenStimulus := map[domain.Digest]struct{}{}
	for _, proposal := range result {
		digest, measure, ok := s.input.Reference(proposal.Stimulus)
		neighbor := proposal.Neighbor
		if !ok || !validNeighbor(neighbor) || neighbor.currentStimulusDigest != s.currentDigest ||
			neighbor.currentMeasure.digest != s.currentMeasure.digest || neighbor.reducerSetDigest != s.input.ReducerSet.digest ||
			digest != neighbor.stimulusDigest || measure.digest != neighbor.measure.digest {
			return nil, &domain.Error{Code: "INVALID_TYPED_REDUCTION_PROPOSAL"}
		}
		if _, duplicate := seenProposal[neighbor.digest]; duplicate {
			return nil, &domain.Error{Code: "DUPLICATE_REDUCTION_PROPOSAL"}
		}
		if _, duplicate := seenStimulus[neighbor.stimulusDigest]; duplicate {
			return nil, &domain.Error{Code: "DUPLICATE_REDUCTION_NEIGHBOR"}
		}
		seenProposal[neighbor.digest] = struct{}{}
		seenStimulus[neighbor.stimulusDigest] = struct{}{}
	}
	sort.Slice(result, func(left, right int) bool {
		l, r := result[left].Neighbor, result[right].Neighbor
		leftPriority, _ := s.input.ReducerSet.rulePriority(l.rule)
		rightPriority, _ := s.input.ReducerSet.rulePriority(r.rule)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if l.locus != r.locus {
			return l.locus < r.locus
		}
		if l.transformPriority != r.transformPriority {
			return l.transformPriority < r.transformPriority
		}
		return l.stimulusDigest.String() < r.stimulusDigest.String()
	})
	return result, nil
}

func (s *runState[S]) evaluate(ctx context.Context, proposal TypedProposal[S], purpose domain.AttemptPurpose) (Evaluation, EvaluationObservation, bool) {
	if reason := s.stopReason(ctx); reason != "" {
		s.limit(string(reason))
		return Evaluation{}, EvaluationObservation{}, false
	}
	// MUTATION_ANCHOR: proposal-budget-equality-is-exhausted
	if s.proposalsUsed >= s.input.Budget.proposalLimit {
		s.limit("PROPOSAL_BUDGET_EXHAUSTED")
		return Evaluation{}, EvaluationObservation{}, false
	}
	// MUTATION_ANCHOR: candidate-trial-budget-equality-is-exhausted
	if s.trialsUsed >= s.input.Budget.trialLimit {
		s.limit("CANDIDATE_TRIAL_BUDGET_EXHAUSTED")
		return Evaluation{}, EvaluationObservation{}, false
	}
	s.proposalsUsed++
	allowance := EvaluationAllowance{
		RemainingCandidateTrials: s.input.Budget.trialLimit - s.trialsUsed,
		RemainingProposals:       s.input.Budget.proposalLimit - s.proposalsUsed,
		WallDeadline:             s.deadline,
	}
	observation, err := s.input.Evaluate(ctx, proposal.Stimulus, proposal.Neighbor, purpose, allowance)
	if err != nil {
		// Preserve trials completed before an edge failure while discarding any
		// partial map that cannot be admitted as product evidence.
		evidence := observation.UnresolvedEvidence
		if observation.OutcomeMap != nil {
			evidence, _ = unresolvedEvidenceFromMap(*observation.OutcomeMap)
		}
		observation = EvaluationObservation{
			UnresolvedReason: ReasonEvaluatorError, UnresolvedEvidence: evidence,
			CandidateTrials: observation.CandidateTrials,
		}
	}
	if observation.CandidateTrials > maxTrialLimit || s.trialsUsed > maxTrialLimit-observation.CandidateTrials {
		s.fatalErr = &domain.Error{Code: "REDUCTION_EVALUATOR_TRIAL_COUNT_OVERFLOW"}
		return Evaluation{}, EvaluationObservation{}, false
	}
	s.trialsUsed += observation.CandidateTrials
	if observation.CandidateTrials > allowance.RemainingCandidateTrials {
		return s.protocolRefusal(proposal, purpose, ReasonEvaluatorExceededBudget, observation)
	}
	if reason := s.stopReason(ctx); reason != "" {
		evidence := observation.UnresolvedEvidence
		if observation.OutcomeMap != nil {
			evidence, _ = unresolvedEvidenceFromMap(*observation.OutcomeMap)
		}
		observation = EvaluationObservation{
			UnresolvedReason: reason, UnresolvedEvidence: evidence, CandidateTrials: observation.CandidateTrials,
		}
	}
	if observation.OutcomeMap != nil && observation.CandidateTrials == 0 {
		return s.protocolRefusal(proposal, purpose, ReasonObservedMapWithoutTrials, observation)
	}
	// MUTATION_ANCHOR: map-backed-trial-count-matches-attempt-evidence
	if observation.OutcomeMap != nil &&
		observation.CandidateTrials != uint64(len(observation.OutcomeMap.EvidenceAttemptDigests())) {
		return s.protocolRefusal(proposal, purpose, ReasonInvalidEvaluatorResult, observation)
	}
	if observation.OutcomeMap == nil && observation.CandidateTrials > 0 && !observation.UnresolvedEvidence.Valid() {
		return s.protocolRefusal(proposal, purpose, ReasonUnresolvedEvidenceMissing, observation)
	}
	id := fmt.Sprintf("eval:%06d-%s", len(s.entries)+1, purposeToken(purpose))
	input := EvaluationInput{ID: id, Baseline: s.currentBaseline, Neighbor: proposal.Neighbor,
		ObservedOutcomeMap: observation.OutcomeMap, UnresolvedReason: observation.UnresolvedReason,
		UnresolvedEvidence: observation.UnresolvedEvidence}
	var evaluation Evaluation
	if purpose == domain.AttemptFinalSweep {
		evaluation, err = NewFinalSweepEvaluation(input)
	} else {
		evaluation, err = NewEvaluation(input)
	}
	if err != nil {
		reason := ReasonInvalidEvaluatorResult
		var domainErr *domain.Error
		if errors.As(err, &domainErr) && len(domainErr.Code) >= len("REUSED_") && domainErr.Code[:len("REUSED_")] == "REUSED_" {
			reason = ReasonReusedEvaluationEvidence
		}
		return s.protocolRefusal(proposal, purpose, reason, observation)
	}
	if !s.ledger.admit(evaluation) {
		return s.protocolRefusal(proposal, purpose, ReasonReusedEvaluationEvidence, EvaluationObservation{CandidateTrials: observation.CandidateTrials})
	}
	return evaluation, observation, true
}

// protocolRefusal records the exact attempted proposal and honest charged trial
// count, but deliberately drops malformed map authority. Same-domain nonreused
// partial lineage is retained when it can be admitted; otherwise the refusal remains explicit
// with empty evidence rather than laundering reused captures.
func (s *runState[S]) protocolRefusal(
	proposal TypedProposal[S], purpose domain.AttemptPurpose, reason UnresolvedReason, observation EvaluationObservation,
) (Evaluation, EvaluationObservation, bool) {
	evidence := observation.UnresolvedEvidence
	if observation.OutcomeMap != nil {
		derived, err := unresolvedEvidenceFromMap(*observation.OutcomeMap)
		if err == nil {
			evidence = derived
		}
	}
	build := func(value UnresolvedEvidence) (Evaluation, error) {
		input := EvaluationInput{
			ID:       fmt.Sprintf("eval:%06d-%s", len(s.entries)+1, purposeToken(purpose)),
			Baseline: s.currentBaseline, Neighbor: proposal.Neighbor,
			UnresolvedReason: reason, UnresolvedEvidence: value,
		}
		return newProtocolRefusalEvaluation(input, purpose)
	}
	evaluation, err := build(evidence)
	if err != nil || !s.ledger.admit(evaluation) {
		evidence = UnresolvedEvidence{}
		evaluation, err = build(evidence)
		if err != nil || !s.ledger.admit(evaluation) {
			s.fatalErr = &domain.Error{Code: "REDUCTION_PROTOCOL_REFUSAL_RECORD_FAILED"}
			return Evaluation{}, EvaluationObservation{}, false
		}
	}
	return evaluation, EvaluationObservation{
		UnresolvedReason: reason, UnresolvedEvidence: evidence,
		CandidateTrials: observation.CandidateTrials, terminalRefusal: true,
	}, true
}

func purposeToken(purpose domain.AttemptPurpose) string {
	if purpose == domain.AttemptFinalSweep {
		return "final-sweep"
	}
	return "reduction"
}

func (s *runState[S]) record(evaluation Evaluation, candidateTrials uint64) {
	s.entries = append(s.entries, TranscriptEntry{
		evaluation: evaluation, proposalCount: s.proposalsUsed, trialCount: s.trialsUsed,
		candidateTrials: candidateTrials,
	})
}

func (s *runState[S]) noteDecisionLimitation(evaluation Evaluation) {
	if evaluation.decision == Unresolved {
		s.limit("UNRESOLVED_" + evaluation.reasonCode)
	}
}

func (s *runState[S]) finalSweepRun(ctx context.Context, searchProposals []TypedProposal[S]) {
	s.finalState = FinalSweepIncomplete
	proposals, err := s.enumerate(ctx)
	if reason := s.stopReason(ctx); reason != "" {
		s.limit(string(reason))
		return
	}
	if err != nil {
		s.limit("FINAL_SWEEP_ENUMERATION_FAILED")
		return
	}
	if !sameProposalSet(searchProposals, proposals) {
		s.limit("FINAL_SWEEP_ENUMERATION_STALE")
		return
	}
	neighbors := make([]Neighbor, 0, len(proposals))
	evaluations := make([]Evaluation, 0, len(proposals))
	for _, proposal := range proposals {
		evaluation, observation, ok := s.evaluate(ctx, proposal, domain.AttemptFinalSweep)
		if !ok {
			return
		}
		s.record(evaluation, observation.CandidateTrials)
		if observation.terminalRefusal {
			s.limit(evaluation.reasonCode)
			return
		}
		neighbors = append(neighbors, proposal.Neighbor)
		evaluations = append(evaluations, evaluation)
		if evaluation.decision != Changes {
			if evaluation.decision == Unresolved {
				s.limit("FINAL_SWEEP_UNRESOLVED_" + evaluation.reasonCode)
			} else {
				s.limit("FINAL_SWEEP_PRESERVING_NEIGHBOR")
			}
			return
		}
	}
	if reason := s.stopReason(ctx); reason != "" {
		s.limit(string(reason))
		return
	}
	sweep, err := NewLogicalCompleteSweep(LogicalSweepInput{
		State: SweepComplete, Baseline: s.currentBaseline, CurrentStimulus: s.currentDigest,
		CurrentMeasure: s.currentMeasure, ReducerSetDigest: s.input.ReducerSet.digest,
		EnumeratedNeighbors: neighbors, Evaluations: evaluations,
	})
	if err != nil {
		s.limit("LOGICAL_FINAL_SWEEP_REJECTED")
		return
	}
	s.finalState = FinalSweepComplete
	s.finalNeighbors = neighbors
	s.finalEvaluations = evaluations
	s.completeSweep = &sweep
}

func sameProposalSet[S any](left, right []TypedProposal[S]) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !sameNeighbor(left[index].Neighbor, right[index].Neighbor) {
			return false
		}
	}
	return true
}

func unresolvedEvidenceFromMap(outcome compare.CandidateOutcomeMap) (UnresolvedEvidence, error) {
	return NewUnresolvedEvidence(
		outcome.BatchDigests(), outcome.EvidenceAttemptDigests(),
		outcome.EvidenceWorldDigests(), outcome.EvidenceObservationDigests(),
	)
}

func (s *runState[S]) stopReason(ctx context.Context) UnresolvedReason {
	// MUTATION_ANCHOR: wall-budget-fencepost-is-exhausted
	if !s.clock.Now().Before(s.deadline) {
		return ReasonWallBudgetExhausted
	}
	select {
	case <-s.parentContext.Done():
		// MUTATION_ANCHOR: parent-cancellation-prevents-completion
		return ReasonReductionCancelled
	default:
	}
	select {
	case <-ctx.Done():
		// The derived context can expire while the parent remains live; this is
		// the real wall budget, not caller cancellation.
		return ReasonWallBudgetExhausted
	default:
	}
	return UnresolvedReason("")
}

func (s *runState[S]) limit(reason string) {
	if reason == "" {
		return
	}
	for _, existing := range s.limitations {
		if existing == reason {
			return
		}
	}
	s.limitations = append(s.limitations, reason)
}

func (s *runState[S]) finish() (ReductionRun, error) {
	if s.fatalErr != nil {
		return ReductionRun{}, s.fatalErr
	}
	finalDigests := make([]domain.Digest, len(s.finalNeighbors))
	for index, neighbor := range s.finalNeighbors {
		finalDigests[index] = neighbor.stimulusDigest
	}
	sort.Slice(finalDigests, func(i, j int) bool {
		return finalDigests[i].String() < finalDigests[j].String()
	})
	transcript, err := newTranscript(
		s.input.Baseline.OutcomeMap(), s.input.ReducerSet, s.input.Budget,
		s.entries, s.accepted, s.limitations, s.finalState, finalDigests,
	)
	if err != nil {
		return ReductionRun{}, err
	}
	grade := GradeUnchanged
	if len(s.accepted) > 0 {
		grade = GradeBestKnown
	}
	run, err := newReductionRun(runConstruction{
		baseline: s.input.Baseline, originalStimulus: s.originalDigest, minimizedStimulus: s.currentDigest,
		originalMeasure: s.originalMeasure, minimizedMeasure: s.currentMeasure, reducerSet: s.input.ReducerSet,
		budget: s.input.Budget, transcript: transcript, grade: grade,
	})
	if err != nil {
		return ReductionRun{}, err
	}
	run.finalSweep = s.completeSweep
	run.finalNeighbors = append([]Neighbor(nil), s.finalNeighbors...)
	run.finalEvaluations = append([]Evaluation(nil), s.finalEvaluations...)
	return run, nil
}

type transcriptEvaluationIdentity struct {
	EvaluationID                  string   `json:"evaluation_id"`
	ProposalDigest                string   `json:"proposal_digest"`
	ParentStimulusDigest          string   `json:"parent_stimulus_digest"`
	NeighborStimulusDigest        string   `json:"neighbor_stimulus_digest"`
	BeforeMeasureDigest           string   `json:"before_measure_digest"`
	BeforeMeasureComponents       []int64  `json:"before_measure_components"`
	AfterMeasureDigest            string   `json:"after_measure_digest"`
	AfterMeasureComponents        []int64  `json:"after_measure_components"`
	RuleName                      string   `json:"rule_name"`
	RuleVersion                   string   `json:"rule_version"`
	Locus                         string   `json:"locus"`
	TransformPriority             int64    `json:"transform_priority"`
	Purpose                       string   `json:"purpose"`
	Decision                      string   `json:"decision"`
	ReasonCode                    string   `json:"reason_code"`
	ObservedOutcomeMapDigest      string   `json:"observed_outcome_map_digest"`
	ObservedPreservationMapDigest string   `json:"observed_preservation_map_digest"`
	BatchDigests                  []string `json:"batch_digests"`
	AttemptDigests                []string `json:"attempt_digests"`
	WorldDigests                  []string `json:"world_digests"`
	ObservationDigests            []string `json:"observation_digests"`
	ProposalCount                 int64    `json:"proposal_count"`
	TotalCandidateTrials          int64    `json:"total_candidate_trials"`
	CandidateTrials               int64    `json:"candidate_trials"`
}

type transcriptIdentity struct {
	SchemaVersion              string                         `json:"schema_version"`
	Kind                       string                         `json:"kind"`
	ReducerSetDigest           string                         `json:"reducer_set_digest"`
	MeasureDefinitionDigest    string                         `json:"measure_definition_digest"`
	ScopeDigest                string                         `json:"scope_digest"`
	BaselineOutcomeMapDigest   string                         `json:"baseline_outcome_map_digest"`
	BaselinePreservationDigest string                         `json:"baseline_preservation_map_digest"`
	BaselineBatchDigests       []string                       `json:"baseline_batch_digests"`
	BaselineAttemptDigests     []string                       `json:"baseline_attempt_digests"`
	BaselineWorldDigests       []string                       `json:"baseline_world_digests"`
	BaselineObservationDigests []string                       `json:"baseline_observation_digests"`
	ProposalLimit              int64                          `json:"proposal_limit"`
	CandidateTrialLimit        int64                          `json:"candidate_trial_limit"`
	WallLimitMS                int64                          `json:"wall_limit_ms"`
	Entries                    []transcriptEvaluationIdentity `json:"entries"`
	AcceptedProposalDigests    []string                       `json:"accepted_proposal_digests"`
	Limitations                []string                       `json:"limitations"`
	FinalSweepState            string                         `json:"final_sweep_state"`
	FinalNeighborDigests       []string                       `json:"final_neighbor_digests"`
}

func newTranscript(
	baseline compare.CandidateOutcomeMap,
	reducerSet ReducerSet,
	budget Budget,
	entries []TranscriptEntry,
	accepted []domain.Digest,
	limitations []string,
	finalState FinalSweepState,
	finalNeighbors []domain.Digest,
) (Transcript, error) {
	identities := make([]transcriptEvaluationIdentity, len(entries))
	for index, entry := range entries {
		evaluation, neighbor := entry.evaluation, entry.evaluation.neighbor
		observedOutcome, _ := evaluation.ObservedOutcomeMapDigest()
		observedPreservation, _ := evaluation.ObservedPreservationDigest()
		identities[index] = transcriptEvaluationIdentity{
			EvaluationID: evaluation.id, ProposalDigest: neighbor.digest.String(), ParentStimulusDigest: neighbor.currentStimulusDigest.String(),
			NeighborStimulusDigest: neighbor.stimulusDigest.String(), BeforeMeasureDigest: neighbor.currentMeasure.digest.String(),
			BeforeMeasureComponents: int64Components(neighbor.currentMeasure.components), AfterMeasureDigest: neighbor.measure.digest.String(),
			AfterMeasureComponents: int64Components(neighbor.measure.components), RuleName: neighbor.rule.name, RuleVersion: neighbor.rule.version,
			Locus: neighbor.locus, TransformPriority: int64(neighbor.transformPriority), Purpose: string(evaluation.purpose),
			Decision: string(evaluation.decision), ReasonCode: evaluation.reasonCode,
			ObservedOutcomeMapDigest: observedOutcome.String(), ObservedPreservationMapDigest: observedPreservation.String(),
			BatchDigests: digestStrings(evaluation.observedBatchDigests), AttemptDigests: digestStrings(evaluation.observedAttemptDigests),
			WorldDigests: digestStrings(evaluation.observedWorldDigests), ObservationDigests: digestStrings(evaluation.observedObservationDigests),
			ProposalCount: int64(entry.proposalCount), TotalCandidateTrials: int64(entry.trialCount), CandidateTrials: int64(entry.candidateTrials),
		}
	}
	identity := transcriptIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReductionTranscript", ReducerSetDigest: reducerSet.digest.String(),
		MeasureDefinitionDigest: reducerSet.measureDefinition.String(), ScopeDigest: reducerSet.scopeDigest.String(),
		BaselineOutcomeMapDigest:   baseline.ArtifactDigest().String(),
		BaselinePreservationDigest: baseline.PreservationDigest().String(),
		BaselineBatchDigests:       digestStrings(baseline.BatchDigests()),
		BaselineAttemptDigests:     digestStrings(baseline.EvidenceAttemptDigests()),
		BaselineWorldDigests:       digestStrings(baseline.EvidenceWorldDigests()),
		BaselineObservationDigests: digestStrings(baseline.EvidenceObservationDigests()),
		ProposalLimit:              int64(budget.proposalLimit),
		CandidateTrialLimit:        int64(budget.trialLimit), WallLimitMS: budget.wallLimit.Milliseconds(), Entries: identities,
		AcceptedProposalDigests: digestStrings(accepted), Limitations: append([]string{}, limitations...),
		FinalSweepState: string(finalState), FinalNeighborDigests: digestStrings(finalNeighbors),
	}
	digest, canonicalBytes, err := digestTyped("ReductionTranscript", identity)
	if err != nil {
		return Transcript{}, err
	}
	return Transcript{digest: digest, canonicalBytes: canonicalBytes, entries: append([]TranscriptEntry(nil), entries...),
		acceptedPath: append([]domain.Digest(nil), accepted...), limitations: append([]string(nil), limitations...),
		finalState: finalState, finalNeighbors: append([]domain.Digest(nil), finalNeighbors...)}, nil
}

func int64Components(values []uint64) []int64 {
	result := make([]int64, len(values))
	for index, value := range values {
		result[index] = int64(value)
	}
	return result
}

type runConstruction struct {
	baseline                            compare.DivergentBaseline
	originalStimulus, minimizedStimulus domain.Digest
	originalMeasure, minimizedMeasure   Measure
	reducerSet                          ReducerSet
	budget                              Budget
	transcript                          Transcript
	grade                               DraftGrade
}

type reductionRunIdentity struct {
	SchemaVersion                 string   `json:"schema_version"`
	Kind                          string   `json:"kind"`
	BaselineOutcomeMapDigest      string   `json:"baseline_outcome_map_digest"`
	BaselinePreservationMapDigest string   `json:"baseline_preservation_map_digest"`
	OriginalStimulusDigest        string   `json:"original_stimulus_digest"`
	MinimizedStimulusDigest       string   `json:"minimized_stimulus_digest"`
	OriginalMeasureDigest         string   `json:"original_measure_digest"`
	OriginalMeasureComponents     []int64  `json:"original_measure_components"`
	MinimizedMeasureDigest        string   `json:"minimized_measure_digest"`
	MinimizedMeasureComponents    []int64  `json:"minimized_measure_components"`
	ReducerSetDigest              string   `json:"reducer_set_digest"`
	MeasureDefinitionDigest       string   `json:"measure_definition_digest"`
	ScopeDigest                   string   `json:"scope_digest"`
	ProposalLimit                 int64    `json:"proposal_limit"`
	CandidateTrialLimit           int64    `json:"candidate_trial_limit"`
	WallLimitMS                   int64    `json:"wall_limit_ms"`
	TranscriptDigest              string   `json:"transcript_digest"`
	Grade                         string   `json:"grade"`
	Limitations                   []string `json:"limitations"`
	GlobalMinimumClaimed          bool     `json:"global_minimum_claimed"`
	RootCauseClaimed              bool     `json:"root_cause_claimed"`
}

func newReductionRun(input runConstruction) (ReductionRun, error) {
	comparison, err := input.minimizedMeasure.Compare(input.originalMeasure)
	if err != nil || comparison > 0 || !input.baseline.Valid() || !input.originalStimulus.Valid() || !input.minimizedStimulus.Valid() ||
		!input.reducerSet.Valid() || !input.budget.Valid() || !input.transcript.Valid() ||
		(input.grade != GradeUnchanged && input.grade != GradeBestKnown) {
		return ReductionRun{}, &domain.Error{Code: "INVALID_REDUCTION_RUN"}
	}
	baselineMap := input.baseline.OutcomeMap()
	identity := reductionRunIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReductionRun", BaselineOutcomeMapDigest: baselineMap.ArtifactDigest().String(),
		BaselinePreservationMapDigest: baselineMap.PreservationDigest().String(), OriginalStimulusDigest: input.originalStimulus.String(),
		MinimizedStimulusDigest: input.minimizedStimulus.String(), OriginalMeasureDigest: input.originalMeasure.digest.String(),
		OriginalMeasureComponents: int64Components(input.originalMeasure.components), MinimizedMeasureDigest: input.minimizedMeasure.digest.String(),
		MinimizedMeasureComponents: int64Components(input.minimizedMeasure.components), ReducerSetDigest: input.reducerSet.digest.String(),
		MeasureDefinitionDigest: input.reducerSet.measureDefinition.String(), ScopeDigest: input.reducerSet.scopeDigest.String(),
		ProposalLimit: int64(input.budget.proposalLimit), CandidateTrialLimit: int64(input.budget.trialLimit),
		WallLimitMS: input.budget.wallLimit.Milliseconds(), TranscriptDigest: input.transcript.digest.String(), Grade: string(input.grade),
		Limitations: input.transcript.Limitations(), GlobalMinimumClaimed: false, RootCauseClaimed: false,
	}
	digest, canonicalBytes, err := digestTyped("ReductionRun", identity)
	if err != nil {
		return ReductionRun{}, err
	}
	return ReductionRun{
		digest: digest, canonicalBytes: canonicalBytes, baseline: input.baseline, originalStimulus: input.originalStimulus,
		minimizedStimulus: input.minimizedStimulus, originalMeasure: input.originalMeasure, minimizedMeasure: input.minimizedMeasure,
		reducerSet: input.reducerSet, budget: input.budget, transcript: input.transcript, draftGrade: input.grade,
	}, nil
}

// RenderTranscript is deliberately non-authoritative presentation. It never
// returns captured bytes, candidate aliases, support counts, or group ordinals.
func RenderTranscript(transcript Transcript) string {
	if !transcript.Valid() {
		return "invalid reduction transcript\n"
	}
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "reduction %s  final-sweep=%s\n", transcript.digest.String(), transcript.finalState)
	for _, entry := range transcript.entries {
		evaluation, neighbor := entry.evaluation, entry.evaluation.neighbor
		fmt.Fprintf(&buffer, "%s  %s@%s  %s -> %s  %s  trials=%d/%d\n",
			evaluation.id, neighbor.rule.name, neighbor.rule.version, neighbor.currentStimulusDigest.String(),
			neighbor.stimulusDigest.String(), evaluation.decision, entry.candidateTrials, entry.trialCount)
	}
	for _, limitation := range transcript.limitations {
		fmt.Fprintf(&buffer, "limit: %s\n", limitation)
	}
	return buffer.String()
}

// ReplayStep is inert, complete proposal/evaluation intent reconstructed from
// exact transcript bytes. Archived evidence is exposed for audit, never as an
// execution result or authority.
type ReplayStep struct {
	EvaluationID            string
	ProposalDigest          domain.Digest
	ParentDigest            domain.Digest
	NeighborDigest          domain.Digest
	BeforeMeasure           Measure
	AfterMeasure            Measure
	ReducerSetDigest        domain.Digest
	MeasureDefinitionDigest domain.Digest
	ScopeDigest             domain.Digest
	RuleName                string
	RuleVersion             string
	Locus                   string
	TransformPriority       uint64
	Purpose                 domain.AttemptPurpose
	RecordedDecision        Decision
	RecordedReason          string
	RecordedProposalCount   uint64
	RecordedTotalTrials     uint64
	RecordedCandidateTrials uint64
	archivedOutcome         *compare.OutcomeArtifactDigest
	archivedPreservation    *compare.PreservationMapDigest
	archivedBatch           []domain.Digest
	archivedAttempt         []domain.Digest
	archivedWorld           []domain.Digest
	archivedObservation     []domain.Digest
}

func (s ReplayStep) ArchivedOutcomeMapDigest() (compare.OutcomeArtifactDigest, bool) {
	if s.archivedOutcome == nil {
		return compare.OutcomeArtifactDigest{}, false
	}
	return *s.archivedOutcome, true
}
func (s ReplayStep) ArchivedPreservationMapDigest() (compare.PreservationMapDigest, bool) {
	if s.archivedPreservation == nil {
		return compare.PreservationMapDigest{}, false
	}
	return *s.archivedPreservation, true
}

func (s ReplayStep) ArchivedBatchDigests() []domain.Digest {
	return append([]domain.Digest(nil), s.archivedBatch...)
}
func (s ReplayStep) ArchivedAttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), s.archivedAttempt...)
}
func (s ReplayStep) ArchivedWorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), s.archivedWorld...)
}
func (s ReplayStep) ArchivedObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), s.archivedObservation...)
}

func (r TranscriptRecord) ReplayPlan() []ReplayStep {
	steps := make([]ReplayStep, len(r.entries))
	for index, entry := range r.entries {
		steps[index] = ReplayStep{
			EvaluationID: entry.evaluationID, ProposalDigest: entry.proposalDigest,
			ParentDigest: entry.parentStimulus, NeighborDigest: entry.neighborStimulus,
			BeforeMeasure: entry.beforeMeasure, AfterMeasure: entry.afterMeasure,
			ReducerSetDigest: r.reducerSetDigest, MeasureDefinitionDigest: r.measureDefinition,
			ScopeDigest: r.scopeDigest,
			RuleName:    entry.ruleName, RuleVersion: entry.ruleVersion, Locus: entry.locus,
			TransformPriority: entry.transformPriority, Purpose: entry.purpose,
			RecordedDecision: entry.decision, RecordedReason: entry.reasonCode,
			RecordedProposalCount: entry.proposalCount, RecordedTotalTrials: entry.totalCandidateTrials,
			RecordedCandidateTrials: entry.candidateTrials,
			archivedOutcome:         entry.observedOutcome, archivedPreservation: entry.observedPreservation,
			archivedBatch:       append([]domain.Digest(nil), entry.batchDigests...),
			archivedAttempt:     append([]domain.Digest(nil), entry.attemptDigests...),
			archivedWorld:       append([]domain.Digest(nil), entry.worldDigests...),
			archivedObservation: append([]domain.Digest(nil), entry.observationDigests...),
		}
	}
	return steps
}

func (r ReductionRun) ReplayPlan() []ReplayStep {
	record, err := ParseTranscriptRecord(r.transcript.canonicalBytes)
	if err != nil {
		return nil
	}
	return record.ReplayPlan()
}

// ReplayEvidence is newly executed lineage. The alias deliberately reuses the
// same construction-safe four-domain shape as unresolved control evidence.
type ReplayEvidence = UnresolvedEvidence

func NewReplayEvidence(batch, attempt, world, observation []domain.Digest) (ReplayEvidence, error) {
	return NewUnresolvedEvidence(batch, attempt, world, observation)
}

type ReplayExecutor func(context.Context, ReplayStep) (ReplayEvidence, error)

// ReplayReceipt is a non-authoritative record that the caller callback returned
// structurally valid evidence whose same-domain digests did not collide with
// the validated baseline, archive, or earlier replay steps. Digest noncollision
// cannot prove physical execution or freshness.
type ReplayReceipt struct {
	step     ReplayStep
	evidence ReplayEvidence
}

func (r ReplayReceipt) Step() ReplayStep         { return r.step }
func (r ReplayReceipt) Evidence() ReplayEvidence { return r.evidence }

// ExecuteReplay never returns archived captures. The caller must supply the
// trusted baseline map whose exact identity and four evidence domains are bound
// into the transcript; self-asserted transcript lineage is not sufficient. The
// executor then runs once per serialized step, and replay rejects evidence
// reused from that baseline, the archived run, or an earlier replay step.
// Re-evaluating product behavior remains the edge's job; receipts do not
// construct Evaluation, a reduction grade, or freshness authority beyond
// content-addressed nonreuse.
func ExecuteReplay(
	ctx context.Context,
	record TranscriptRecord,
	trustedBaseline compare.DivergentBaseline,
	executor ReplayExecutor,
) ([]ReplayReceipt, error) {
	if ctx == nil || !record.Valid() || !trustedBaseline.Valid() || executor == nil {
		return nil, &domain.Error{Code: "INVALID_REDUCTION_REPLAY_INPUT"}
	}
	baseline := trustedBaseline.OutcomeMap()
	if baseline.ArtifactDigest() != record.baselineOutcome ||
		baseline.PreservationDigest() != record.baselinePreservation ||
		!slices.Equal(baseline.BatchDigests(), record.baselineBatch) ||
		!slices.Equal(baseline.EvidenceAttemptDigests(), record.baselineAttempt) ||
		!slices.Equal(baseline.EvidenceWorldDigests(), record.baselineWorld) ||
		!slices.Equal(baseline.EvidenceObservationDigests(), record.baselineObservation) {
		return nil, &domain.Error{Code: "REDUCTION_REPLAY_BASELINE_MISMATCH"}
	}
	steps := record.ReplayPlan()
	if len(steps) > 0 && steps[0].ParentDigest != baseline.StimulusDigest() {
		return nil, &domain.Error{Code: "REDUCTION_REPLAY_BASELINE_STIMULUS_MISMATCH"}
	}
	seen := [4]map[domain.Digest]struct{}{{}, {}, {}, {}}
	for domainIndex, values := range [][]domain.Digest{
		baseline.BatchDigests(), baseline.EvidenceAttemptDigests(),
		baseline.EvidenceWorldDigests(), baseline.EvidenceObservationDigests(),
	} {
		for _, digest := range values {
			seen[domainIndex][digest] = struct{}{}
		}
	}
	for _, step := range steps {
		for domainIndex, values := range [][]domain.Digest{
			step.archivedBatch, step.archivedAttempt, step.archivedWorld, step.archivedObservation,
		} {
			for _, digest := range values {
				seen[domainIndex][digest] = struct{}{}
			}
		}
	}
	receipts := make([]ReplayReceipt, 0, len(steps))
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return nil, &domain.Error{Code: "REDUCTION_REPLAY_CANCELLED", Detail: err.Error()}
		}
		evidence, err := executor(ctx, step)
		if err != nil {
			return nil, &domain.Error{Code: "REDUCTION_REPLAY_EXECUTION_FAILED", Detail: err.Error()}
		}
		if !evidence.Valid() {
			return nil, &domain.Error{Code: "INVALID_REDUCTION_REPLAY_EVIDENCE"}
		}
		for domainIndex, values := range [][]domain.Digest{
			evidence.batchDigests, evidence.attemptDigests, evidence.worldDigests, evidence.observationDigests,
		} {
			for _, digest := range values {
				if _, reused := seen[domainIndex][digest]; reused {
					return nil, &domain.Error{Code: "REUSED_REDUCTION_REPLAY_EVIDENCE"}
				}
			}
			for _, digest := range values {
				seen[domainIndex][digest] = struct{}{}
			}
		}
		receipts = append(receipts, ReplayReceipt{step: step, evidence: evidence})
	}
	return receipts, nil
}

// Compile-time reminder that transcript integers remain within canon's exact
// interoperability range through the explicit U5 maxima.
var _ = canon.MaxSafeInteger
