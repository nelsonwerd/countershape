package reduce

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

// TranscriptRecord is a validated inert archive. It can drive a fresh replay
// plan, but it cannot reconstruct Evaluation or execution-evidence authority.
type TranscriptRecord struct {
	digest               domain.Digest
	canonicalBytes       []byte
	reducerSetDigest     domain.Digest
	measureDefinition    domain.Digest
	scopeDigest          domain.Digest
	baselineOutcome      compare.OutcomeArtifactDigest
	baselinePreservation compare.PreservationMapDigest
	baselineBatch        []domain.Digest
	baselineAttempt      []domain.Digest
	baselineWorld        []domain.Digest
	baselineObservation  []domain.Digest
	proposalLimit        uint64
	candidateTrialLimit  uint64
	wallLimitMS          int64
	entries              []TranscriptRecordEntry
	accepted             []domain.Digest
	limitations          []string
	finalState           FinalSweepState
	finalNeighbors       []domain.Digest
}

type TranscriptRecordEntry struct {
	evaluationID         string
	proposalDigest       domain.Digest
	parentStimulus       domain.Digest
	neighborStimulus     domain.Digest
	beforeMeasure        Measure
	afterMeasure         Measure
	ruleName             string
	ruleVersion          string
	locus                string
	transformPriority    uint64
	purpose              domain.AttemptPurpose
	decision             Decision
	reasonCode           string
	observedOutcome      *compare.OutcomeArtifactDigest
	observedPreservation *compare.PreservationMapDigest
	batchDigests         []domain.Digest
	attemptDigests       []domain.Digest
	worldDigests         []domain.Digest
	observationDigests   []domain.Digest
	proposalCount        uint64
	totalCandidateTrials uint64
	candidateTrials      uint64
}

func (e TranscriptRecordEntry) EvaluationID() string                  { return e.evaluationID }
func (e TranscriptRecordEntry) ProposalDigest() domain.Digest         { return e.proposalDigest }
func (e TranscriptRecordEntry) ParentStimulusDigest() domain.Digest   { return e.parentStimulus }
func (e TranscriptRecordEntry) NeighborStimulusDigest() domain.Digest { return e.neighborStimulus }
func (e TranscriptRecordEntry) BeforeMeasure() Measure                { return e.beforeMeasure }
func (e TranscriptRecordEntry) AfterMeasure() Measure                 { return e.afterMeasure }
func (e TranscriptRecordEntry) RuleName() string                      { return e.ruleName }
func (e TranscriptRecordEntry) RuleVersion() string                   { return e.ruleVersion }
func (e TranscriptRecordEntry) Locus() string                         { return e.locus }
func (e TranscriptRecordEntry) TransformPriority() uint64             { return e.transformPriority }
func (e TranscriptRecordEntry) Purpose() domain.AttemptPurpose        { return e.purpose }
func (e TranscriptRecordEntry) Decision() Decision                    { return e.decision }
func (e TranscriptRecordEntry) ReasonCode() string                    { return e.reasonCode }
func (e TranscriptRecordEntry) ProposalCount() uint64                 { return e.proposalCount }
func (e TranscriptRecordEntry) TotalCandidateTrials() uint64          { return e.totalCandidateTrials }
func (e TranscriptRecordEntry) CandidateTrials() uint64               { return e.candidateTrials }
func (e TranscriptRecordEntry) BatchDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.batchDigests...)
}
func (e TranscriptRecordEntry) AttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.attemptDigests...)
}
func (e TranscriptRecordEntry) WorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.worldDigests...)
}
func (e TranscriptRecordEntry) ObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), e.observationDigests...)
}
func (e TranscriptRecordEntry) ObservedOutcomeMapDigest() (compare.OutcomeArtifactDigest, bool) {
	if e.observedOutcome == nil {
		return compare.OutcomeArtifactDigest{}, false
	}
	return *e.observedOutcome, true
}
func (e TranscriptRecordEntry) ObservedPreservationMapDigest() (compare.PreservationMapDigest, bool) {
	if e.observedPreservation == nil {
		return compare.PreservationMapDigest{}, false
	}
	return *e.observedPreservation, true
}

func (r TranscriptRecord) Digest() domain.Digest                  { return r.digest }
func (r TranscriptRecord) CanonicalBytes() []byte                 { return append([]byte(nil), r.canonicalBytes...) }
func (r TranscriptRecord) ReducerSetDigest() domain.Digest        { return r.reducerSetDigest }
func (r TranscriptRecord) MeasureDefinitionDigest() domain.Digest { return r.measureDefinition }
func (r TranscriptRecord) ScopeDigest() domain.Digest             { return r.scopeDigest }
func (r TranscriptRecord) BaselineOutcomeMapDigest() compare.OutcomeArtifactDigest {
	return r.baselineOutcome
}
func (r TranscriptRecord) BaselinePreservationMapDigest() compare.PreservationMapDigest {
	return r.baselinePreservation
}
func (r TranscriptRecord) BaselineBatchDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.baselineBatch...)
}
func (r TranscriptRecord) BaselineAttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.baselineAttempt...)
}
func (r TranscriptRecord) BaselineWorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.baselineWorld...)
}
func (r TranscriptRecord) BaselineObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.baselineObservation...)
}
func (r TranscriptRecord) ProposalLimit() uint64       { return r.proposalLimit }
func (r TranscriptRecord) CandidateTrialLimit() uint64 { return r.candidateTrialLimit }
func (r TranscriptRecord) WallLimitMS() int64          { return r.wallLimitMS }
func (r TranscriptRecord) Entries() []TranscriptRecordEntry {
	return append([]TranscriptRecordEntry(nil), r.entries...)
}
func (r TranscriptRecord) AcceptedProposalDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.accepted...)
}
func (r TranscriptRecord) Limitations() []string            { return append([]string(nil), r.limitations...) }
func (r TranscriptRecord) FinalSweepState() FinalSweepState { return r.finalState }
func (r TranscriptRecord) FinalNeighborDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.finalNeighbors...)
}
func (r TranscriptRecord) Valid() bool {
	parsed, err := ParseTranscriptRecord(r.canonicalBytes)
	return err == nil && parsed.digest == r.digest && bytes.Equal(parsed.canonicalBytes, r.canonicalBytes)
}

func ParseTranscriptRecord(exactCanonicalBytes []byte) (TranscriptRecord, error) {
	if err := requireCanonical(exactCanonicalBytes); err != nil {
		return TranscriptRecord{}, err
	}
	var identity transcriptIdentity
	if err := json.Unmarshal(exactCanonicalBytes, &identity); err != nil {
		return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_WIRE", Detail: err.Error()}
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "ReductionTranscript" ||
		identity.ProposalLimit < 1 || identity.ProposalLimit > maxProposalLimit ||
		identity.CandidateTrialLimit < 1 || identity.CandidateTrialLimit > maxTrialLimit ||
		identity.WallLimitMS < 1 || identity.WallLimitMS > maxWallLimit.Milliseconds() ||
		len(identity.Entries) > int(identity.ProposalLimit) || identity.Entries == nil || identity.AcceptedProposalDigests == nil ||
		identity.Limitations == nil || identity.FinalNeighborDigests == nil ||
		identity.BaselineBatchDigests == nil || identity.BaselineAttemptDigests == nil ||
		identity.BaselineWorldDigests == nil || identity.BaselineObservationDigests == nil {
		return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_STATE"}
	}
	reducerSetDigest, err := domain.ParseDigest(identity.ReducerSetDigest)
	if err != nil {
		return TranscriptRecord{}, err
	}
	measureDefinition, err := domain.ParseDigest(identity.MeasureDefinitionDigest)
	if err != nil {
		return TranscriptRecord{}, err
	}
	scopeDigest, err := domain.ParseDigest(identity.ScopeDigest)
	if err != nil {
		return TranscriptRecord{}, err
	}
	baselineOutcome, err := compare.ParseOutcomeArtifactDigest(identity.BaselineOutcomeMapDigest)
	if err != nil {
		return TranscriptRecord{}, err
	}
	baselinePreservation, err := compare.ParsePreservationMapDigest(identity.BaselinePreservationDigest)
	if err != nil {
		return TranscriptRecord{}, err
	}
	finalState := FinalSweepState(identity.FinalSweepState)
	if finalState != FinalSweepComplete && finalState != FinalSweepIncomplete && finalState != FinalSweepNotRun {
		return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_FINAL_SWEEP_STATE"}
	}
	seenLimitations := make(map[string]struct{}, len(identity.Limitations))
	for _, limitation := range identity.Limitations {
		if limitation == "" {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_LIMITATION"}
		}
		if _, duplicate := seenLimitations[limitation]; duplicate {
			return TranscriptRecord{}, &domain.Error{Code: "DUPLICATE_REDUCTION_TRANSCRIPT_LIMITATION"}
		}
		seenLimitations[limitation] = struct{}{}
	}
	accepted, err := parseDigestList(identity.AcceptedProposalDigests, false)
	if err != nil {
		return TranscriptRecord{}, err
	}
	finalNeighbors, err := parseDigestList(identity.FinalNeighborDigests, true)
	if err != nil {
		return TranscriptRecord{}, err
	}
	baselineLists := [][]string{
		identity.BaselineBatchDigests, identity.BaselineAttemptDigests,
		identity.BaselineWorldDigests, identity.BaselineObservationDigests,
	}
	parsedBaseline := make([][]domain.Digest, len(baselineLists))
	for index, values := range baselineLists {
		parsed, parseErr := parseDigestList(values, true)
		if parseErr != nil || len(parsed) == 0 {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_BASELINE_EVIDENCE"}
		}
		parsedBaseline[index] = parsed
	}
	entries := make([]TranscriptRecordEntry, len(identity.Entries))
	seenEvaluation := map[string]struct{}{}
	seenEvidence := []map[domain.Digest]struct{}{
		{}, {}, {}, {},
	}
	for index, values := range parsedBaseline {
		for _, digest := range values {
			seenEvidence[index][digest] = struct{}{}
		}
	}
	expectedAccepted := make([]domain.Digest, 0, len(accepted))
	finalStimuli := make([]domain.Digest, 0, len(finalNeighbors))
	finalSweepStarted := false
	var priorProposalCount int64
	var priorTrialCount int64
	var expectedParent domain.Digest
	var expectedMeasure Measure
	haveExpectedParent := false
	var terminalEntryReason UnresolvedReason
	terminalFinalSweepLimitation := ""
	for index, entry := range identity.Entries {
		if index > 0 && priorTrialCount >= identity.CandidateTrialLimit {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_ENTRY_AFTER_CANDIDATE_TRIAL_BUDGET"}
		}
		overConfiguredTrialBudget := entry.TotalCandidateTrials > identity.CandidateTrialLimit
		if entry.EvaluationID == "" || entry.ReasonCode == "" || entry.ProposalCount != int64(index+1) ||
			entry.ProposalCount <= priorProposalCount || entry.TotalCandidateTrials < priorTrialCount || entry.CandidateTrials < 0 ||
			entry.TotalCandidateTrials-priorTrialCount != entry.CandidateTrials ||
			entry.TotalCandidateTrials > maxTrialLimit ||
			entry.TransformPriority < 0 || entry.TransformPriority > maxReducerTransformPriority {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_ENTRY"}
		}
		if _, duplicate := seenEvaluation[entry.EvaluationID]; duplicate {
			return TranscriptRecord{}, &domain.Error{Code: "DUPLICATE_REDUCTION_EVALUATION_ID"}
		}
		seenEvaluation[entry.EvaluationID] = struct{}{}
		proposalDigest, err := domain.ParseDigest(entry.ProposalDigest)
		if err != nil {
			return TranscriptRecord{}, err
		}
		parent, err := domain.ParseDigest(entry.ParentStimulusDigest)
		if err != nil {
			return TranscriptRecord{}, err
		}
		neighbor, err := domain.ParseDigest(entry.NeighborStimulusDigest)
		if err != nil {
			return TranscriptRecord{}, err
		}
		if parent == neighbor {
			return TranscriptRecord{}, &domain.Error{Code: "SELF_REDUCTION_TRANSCRIPT_NEIGHBOR"}
		}
		before, err := parseWireMeasure(measureDefinition, entry.BeforeMeasureDigest, entry.BeforeMeasureComponents)
		if err != nil {
			return TranscriptRecord{}, err
		}
		after, err := parseWireMeasure(measureDefinition, entry.AfterMeasureDigest, entry.AfterMeasureComponents)
		if err != nil {
			return TranscriptRecord{}, err
		}
		if !strictlyDecreases(before, after) {
			return TranscriptRecord{}, &domain.Error{Code: "NONDECREASING_REDUCTION_TRANSCRIPT_ENTRY"}
		}
		if haveExpectedParent && (parent != expectedParent || before.digest != expectedMeasure.digest) {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_TRANSCRIPT_PARENT_CHAIN_MISMATCH"}
		}
		if !haveExpectedParent {
			expectedParent, expectedMeasure, haveExpectedParent = parent, before, true
		}
		rule, err := NewReducerRule(entry.RuleName, entry.RuleVersion)
		if err != nil || !reducerLocusPattern.MatchString(entry.Locus) {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_PROPOSAL_IDENTITY"}
		}
		expectedProposal, _, err := digestTyped("ReductionProposal", neighborIdentity{
			SchemaVersion: domain.SchemaVersion, Kind: "ReductionProposal",
			CurrentStimulusDigest: parent.String(), CurrentMeasureDigest: before.digest.String(),
			StimulusDigest: neighbor.String(), MeasureDigest: after.digest.String(),
			RuleName: rule.name, RuleVersion: rule.version, RuleDigest: rule.digest.String(),
			Locus: entry.Locus, TransformPriority: entry.TransformPriority, ReducerSetDigest: reducerSetDigest.String(),
		})
		if err != nil || expectedProposal != proposalDigest {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_TRANSCRIPT_PROPOSAL_DIGEST_MISMATCH"}
		}
		purpose := domain.AttemptPurpose(entry.Purpose)
		if purpose != domain.AttemptReduction && purpose != domain.AttemptFinalSweep {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_PURPOSE"}
		}
		if purpose == domain.AttemptFinalSweep {
			finalSweepStarted = true
			finalStimuli = append(finalStimuli, neighbor)
		} else if finalSweepStarted {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_ENTRY_AFTER_FINAL_SWEEP"}
		}
		decision := Decision(entry.Decision)
		if decision != Preserves && decision != Changes && decision != Unresolved {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_DECISION"}
		}
		outcome, preservation, err := parseOptionalMapDigests(entry.ObservedOutcomeMapDigest, entry.ObservedPreservationMapDigest)
		if err != nil {
			return TranscriptRecord{}, err
		}
		if (decision == Preserves || decision == Changes) && (outcome == nil || preservation == nil) {
			return TranscriptRecord{}, &domain.Error{Code: "MISSING_REDUCTION_TRANSCRIPT_MAP_DIGEST"}
		}
		if preservation != nil {
			matchesBaseline := *preservation == baselinePreservation
			if (decision == Preserves && !matchesBaseline) || (decision == Changes && matchesBaseline) {
				return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_TRANSCRIPT_PRESERVATION_DECISION_MISMATCH"}
			}
		}
		switch {
		case decision == Preserves && entry.ReasonCode != "EXACT_PRESERVATION_MAP_MATCH":
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_REASON"}
		case decision == Changes && entry.ReasonCode != "EXACT_PRESERVATION_MAP_CHANGED":
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_REASON"}
		case decision == Unresolved && outcome != nil && entry.ReasonCode != "CANDIDATE_ELIGIBILITY_OR_ADMISSION_CHANGED":
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_REASON"}
		case decision == Unresolved && outcome == nil && !UnresolvedReason(entry.ReasonCode).Valid():
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_TRANSCRIPT_REASON"}
		}
		reason := UnresolvedReason(entry.ReasonCode)
		terminalProtocolRefusal := isTerminalProtocolReason(reason)
		if overConfiguredTrialBudget != (reason == ReasonEvaluatorExceededBudget) {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_REDUCTION_EVALUATOR_BUDGET_BREACH"}
		}
		if terminalProtocolRefusal &&
			(decision != Unresolved || outcome != nil || index != len(identity.Entries)-1 ||
				finalState == FinalSweepComplete || !slices.Contains(identity.Limitations, entry.ReasonCode)) {
			return TranscriptRecord{}, &domain.Error{Code: "INVALID_TERMINAL_REDUCTION_PROTOCOL_REFUSAL"}
		}
		if terminalProtocolRefusal {
			terminalEntryReason = reason
		}
		if purpose == domain.AttemptReduction && decision == Preserves {
			expectedAccepted = append(expectedAccepted, proposalDigest)
			expectedParent, expectedMeasure = neighbor, after
		}
		if finalState == FinalSweepComplete && purpose == domain.AttemptFinalSweep && decision != Changes {
			return TranscriptRecord{}, &domain.Error{Code: "NONCHANGING_COMPLETE_FINAL_SWEEP_ENTRY"}
		}
		if entry.BatchDigests == nil || entry.AttemptDigests == nil || entry.WorldDigests == nil || entry.ObservationDigests == nil {
			return TranscriptRecord{}, &domain.Error{Code: "MISSING_REDUCTION_TRANSCRIPT_EVIDENCE_ARRAY"}
		}
		evidenceLists := [][]string{entry.BatchDigests, entry.AttemptDigests, entry.WorldDigests, entry.ObservationDigests}
		parsedEvidence := make([][]domain.Digest, len(evidenceLists))
		nonemptyEvidenceDomains := 0
		for evidenceIndex, values := range evidenceLists {
			parsed, err := parseDigestList(values, false)
			if err != nil {
				return TranscriptRecord{}, err
			}
			if outcome != nil && len(parsed) == 0 {
				return TranscriptRecord{}, &domain.Error{Code: "MISSING_REDUCTION_TRANSCRIPT_EVIDENCE"}
			}
			if len(parsed) != 0 {
				nonemptyEvidenceDomains++
			}
			for _, digest := range parsed {
				if _, duplicate := seenEvidence[evidenceIndex][digest]; duplicate {
					return TranscriptRecord{}, &domain.Error{Code: "REUSED_REDUCTION_TRANSCRIPT_EVIDENCE"}
				}
				seenEvidence[evidenceIndex][digest] = struct{}{}
			}
			parsedEvidence[evidenceIndex] = parsed
		}
		if nonemptyEvidenceDomains != 0 && nonemptyEvidenceDomains != len(evidenceLists) {
			return TranscriptRecord{}, &domain.Error{Code: "PARTIAL_REDUCTION_TRANSCRIPT_EVIDENCE"}
		}
		if outcome != nil && entry.CandidateTrials == 0 {
			return TranscriptRecord{}, &domain.Error{Code: "OBSERVED_REDUCTION_MAP_WITHOUT_CANDIDATE_TRIAL"}
		}
		if outcome != nil && entry.CandidateTrials != int64(len(parsedEvidence[1])) {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_TRANSCRIPT_MAP_TRIAL_COUNT_MISMATCH"}
		}
		if outcome == nil && !terminalProtocolRefusal && entry.CandidateTrials > 0 && nonemptyEvidenceDomains == 0 {
			return TranscriptRecord{}, &domain.Error{Code: "CHARGED_REDUCTION_ENTRY_WITHOUT_EVIDENCE"}
		}
		if terminalProtocolRefusal {
			switch reason {
			case ReasonObservedMapWithoutTrials:
				if entry.CandidateTrials != 0 {
					return TranscriptRecord{}, &domain.Error{Code: "INVALID_TERMINAL_REDUCTION_PROTOCOL_ACCOUNTING"}
				}
			case ReasonUnresolvedEvidenceMissing:
				if entry.CandidateTrials <= 0 || nonemptyEvidenceDomains != 0 {
					return TranscriptRecord{}, &domain.Error{Code: "INVALID_TERMINAL_REDUCTION_PROTOCOL_ACCOUNTING"}
				}
			case ReasonReusedEvaluationEvidence:
				if nonemptyEvidenceDomains != 0 {
					return TranscriptRecord{}, &domain.Error{Code: "INVALID_TERMINAL_REDUCTION_PROTOCOL_ACCOUNTING"}
				}
			}
		}
		if purpose == domain.AttemptFinalSweep && decision != Changes {
			if index != len(identity.Entries)-1 || finalState != FinalSweepIncomplete {
				return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_ENTRY_AFTER_TERMINAL_FINAL_SWEEP_DECISION"}
			}
			if !terminalProtocolRefusal {
				expectedLimitation := "FINAL_SWEEP_PRESERVING_NEIGHBOR"
				if decision == Unresolved {
					expectedLimitation = "FINAL_SWEEP_UNRESOLVED_" + entry.ReasonCode
				}
				if !slices.Contains(identity.Limitations, expectedLimitation) {
					return TranscriptRecord{}, &domain.Error{Code: "MISSING_TERMINAL_FINAL_SWEEP_LIMITATION"}
				}
				terminalFinalSweepLimitation = expectedLimitation
			}
		}
		entries[index] = TranscriptRecordEntry{
			evaluationID: entry.EvaluationID, proposalDigest: proposalDigest, parentStimulus: parent,
			neighborStimulus: neighbor, beforeMeasure: before, afterMeasure: after,
			ruleName: entry.RuleName, ruleVersion: entry.RuleVersion, locus: entry.Locus,
			transformPriority: uint64(entry.TransformPriority), purpose: purpose, decision: decision,
			reasonCode: entry.ReasonCode, observedOutcome: outcome, observedPreservation: preservation,
			batchDigests: parsedEvidence[0], attemptDigests: parsedEvidence[1], worldDigests: parsedEvidence[2],
			observationDigests: parsedEvidence[3], proposalCount: uint64(entry.ProposalCount),
			totalCandidateTrials: uint64(entry.TotalCandidateTrials), candidateTrials: uint64(entry.CandidateTrials),
		}
		priorProposalCount, priorTrialCount = entry.ProposalCount, entry.TotalCandidateTrials
	}
	for limitation := range seenLimitations {
		reason := UnresolvedReason(limitation)
		if isTerminalProtocolReason(reason) && reason != terminalEntryReason {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_TERMINAL_LIMITATION_WITHOUT_ENTRY"}
		}
		if (limitation == "FINAL_SWEEP_PRESERVING_NEIGHBOR" || strings.HasPrefix(limitation, "FINAL_SWEEP_UNRESOLVED_")) &&
			limitation != terminalFinalSweepLimitation {
			return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_FINAL_SWEEP_LIMITATION_WITHOUT_ENTRY"}
		}
	}
	if !slices.Equal(accepted, expectedAccepted) {
		return TranscriptRecord{}, &domain.Error{Code: "REDUCTION_ACCEPTED_PATH_MISMATCH"}
	}
	slices.SortFunc(finalStimuli, func(left, right domain.Digest) int {
		return bytes.Compare([]byte(left.String()), []byte(right.String()))
	})
	switch finalState {
	case FinalSweepComplete:
		if !slices.Equal(finalNeighbors, finalStimuli) {
			return TranscriptRecord{}, &domain.Error{Code: "COMPLETE_FINAL_SWEEP_SET_MISMATCH"}
		}
	case FinalSweepIncomplete:
		if len(finalNeighbors) != 0 {
			return TranscriptRecord{}, &domain.Error{Code: "INCOMPLETE_FINAL_SWEEP_HAS_COMPLETED_SET"}
		}
	case FinalSweepNotRun:
		if finalSweepStarted || len(finalNeighbors) != 0 {
			return TranscriptRecord{}, &domain.Error{Code: "UNRUN_FINAL_SWEEP_HAS_EVIDENCE"}
		}
	}
	digest, canonicalBytes, err := digestTyped("ReductionTranscript", identity)
	if err != nil {
		return TranscriptRecord{}, err
	}
	if !bytes.Equal(canonicalBytes, exactCanonicalBytes) {
		return TranscriptRecord{}, &domain.Error{Code: "UNKNOWN_OR_NONEXACT_REDUCTION_TRANSCRIPT_FIELD"}
	}
	return TranscriptRecord{
		digest: digest, canonicalBytes: append([]byte(nil), exactCanonicalBytes...), reducerSetDigest: reducerSetDigest,
		measureDefinition: measureDefinition, scopeDigest: scopeDigest,
		baselineOutcome: baselineOutcome, baselinePreservation: baselinePreservation,
		baselineBatch: parsedBaseline[0], baselineAttempt: parsedBaseline[1],
		baselineWorld: parsedBaseline[2], baselineObservation: parsedBaseline[3],
		proposalLimit:       uint64(identity.ProposalLimit),
		candidateTrialLimit: uint64(identity.CandidateTrialLimit), wallLimitMS: identity.WallLimitMS, entries: entries, accepted: accepted,
		limitations: append([]string(nil), identity.Limitations...), finalState: finalState, finalNeighbors: finalNeighbors,
	}, nil
}

type ReductionRunRecord struct {
	digest               domain.Digest
	canonicalBytes       []byte
	baselineOutcome      compare.OutcomeArtifactDigest
	baselinePreservation compare.PreservationMapDigest
	originalStimulus     domain.Digest
	minimizedStimulus    domain.Digest
	reducerSetDigest     domain.Digest
	measureDefinition    domain.Digest
	scopeDigest          domain.Digest
	grade                DraftGrade
	transcript           TranscriptRecord
}

func ParseReductionRunRecord(exactRunBytes, exactTranscriptBytes []byte) (ReductionRunRecord, error) {
	if err := requireCanonical(exactRunBytes); err != nil {
		return ReductionRunRecord{}, err
	}
	var identity reductionRunIdentity
	if err := json.Unmarshal(exactRunBytes, &identity); err != nil {
		return ReductionRunRecord{}, err
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "ReductionRun" || identity.GlobalMinimumClaimed || identity.RootCauseClaimed || identity.Limitations == nil {
		return ReductionRunRecord{}, &domain.Error{Code: "INVALID_REDUCTION_RUN_RECORD"}
	}
	transcript, err := ParseTranscriptRecord(exactTranscriptBytes)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	transcriptDigest, err := domain.ParseDigest(identity.TranscriptDigest)
	if err != nil || transcriptDigest != transcript.digest {
		return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_RUN_TRANSCRIPT_MISMATCH"}
	}
	if identity.ProposalLimit != int64(transcript.proposalLimit) ||
		identity.CandidateTrialLimit != int64(transcript.candidateTrialLimit) ||
		identity.WallLimitMS != transcript.wallLimitMS ||
		!slices.Equal(identity.Limitations, transcript.limitations) {
		return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_RUN_TRANSCRIPT_POLICY_MISMATCH"}
	}
	baselineOutcome, err := compare.ParseOutcomeArtifactDigest(identity.BaselineOutcomeMapDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	baselinePreservation, err := compare.ParsePreservationMapDigest(identity.BaselinePreservationMapDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	if baselineOutcome != transcript.baselineOutcome || baselinePreservation != transcript.baselinePreservation {
		return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_RUN_BASELINE_MISMATCH"}
	}
	original, err := domain.ParseDigest(identity.OriginalStimulusDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	minimized, err := domain.ParseDigest(identity.MinimizedStimulusDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	reducerSet, err := domain.ParseDigest(identity.ReducerSetDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	measureDefinition, err := domain.ParseDigest(identity.MeasureDefinitionDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	scope, err := domain.ParseDigest(identity.ScopeDigest)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	if reducerSet != transcript.reducerSetDigest || measureDefinition != transcript.measureDefinition || scope != transcript.scopeDigest {
		return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_RUN_SCOPE_MISMATCH"}
	}
	originalMeasure, err := parseWireMeasure(measureDefinition, identity.OriginalMeasureDigest, identity.OriginalMeasureComponents)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	minimizedMeasure, err := parseWireMeasure(measureDefinition, identity.MinimizedMeasureDigest, identity.MinimizedMeasureComponents)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	comparison, err := minimizedMeasure.Compare(originalMeasure)
	if err != nil || comparison > 0 {
		return ReductionRunRecord{}, &domain.Error{Code: "INVALID_REDUCTION_RUN_MEASURE"}
	}
	grade := DraftGrade(identity.Grade)
	if grade != GradeUnchanged && grade != GradeBestKnown {
		return ReductionRunRecord{}, &domain.Error{Code: "INVALID_REDUCTION_RUN_GRADE"}
	}
	expectedMinimized := original
	expectedMeasure := originalMeasure
	for _, entry := range transcript.entries {
		if entry.parentStimulus != expectedMinimized || entry.beforeMeasure.digest != expectedMeasure.digest {
			return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_TRANSCRIPT_PARENT_CHAIN_MISMATCH"}
		}
		if entry.purpose == domain.AttemptReduction && entry.decision == Preserves {
			expectedMinimized = entry.neighborStimulus
			expectedMeasure = entry.afterMeasure
		}
	}
	if expectedMinimized != minimized || expectedMeasure.digest != minimizedMeasure.digest {
		return ReductionRunRecord{}, &domain.Error{Code: "REDUCTION_RUN_MINIMIZED_STIMULUS_MISMATCH"}
	}
	if len(transcript.accepted) == 0 {
		if grade != GradeUnchanged || original != minimized || comparison != 0 {
			return ReductionRunRecord{}, &domain.Error{Code: "INVALID_UNCHANGED_REDUCTION_RUN"}
		}
	} else if grade != GradeBestKnown || comparison >= 0 {
		return ReductionRunRecord{}, &domain.Error{Code: "INVALID_BEST_KNOWN_REDUCTION_RUN"}
	}
	digest, canonicalBytes, err := digestTyped("ReductionRun", identity)
	if err != nil {
		return ReductionRunRecord{}, err
	}
	if !bytes.Equal(canonicalBytes, exactRunBytes) {
		return ReductionRunRecord{}, &domain.Error{Code: "UNKNOWN_OR_NONEXACT_REDUCTION_RUN_FIELD"}
	}
	return ReductionRunRecord{
		digest: digest, canonicalBytes: append([]byte(nil), exactRunBytes...), baselineOutcome: baselineOutcome,
		baselinePreservation: baselinePreservation, originalStimulus: original, minimizedStimulus: minimized,
		reducerSetDigest: reducerSet, measureDefinition: measureDefinition, scopeDigest: scope, grade: grade, transcript: transcript,
	}, nil
}

func (r ReductionRunRecord) Digest() domain.Digest  { return r.digest }
func (r ReductionRunRecord) CanonicalBytes() []byte { return append([]byte(nil), r.canonicalBytes...) }
func (r ReductionRunRecord) BaselineOutcomeMapDigest() compare.OutcomeArtifactDigest {
	return r.baselineOutcome
}
func (r ReductionRunRecord) BaselinePreservationMapDigest() compare.PreservationMapDigest {
	return r.baselinePreservation
}
func (r ReductionRunRecord) OriginalStimulusDigest() domain.Digest  { return r.originalStimulus }
func (r ReductionRunRecord) MinimizedStimulusDigest() domain.Digest { return r.minimizedStimulus }
func (r ReductionRunRecord) ReducerSetDigest() domain.Digest        { return r.reducerSetDigest }
func (r ReductionRunRecord) MeasureDefinitionDigest() domain.Digest { return r.measureDefinition }
func (r ReductionRunRecord) ScopeDigest() domain.Digest             { return r.scopeDigest }
func (r ReductionRunRecord) Grade() DraftGrade                      { return r.grade }
func (r ReductionRunRecord) Transcript() TranscriptRecord           { return r.transcript }

func requireCanonical(raw []byte) error {
	value, err := canon.Parse(raw)
	if err != nil {
		return &domain.Error{Code: "INVALID_CANONICAL_REDUCTION_WIRE", Detail: err.Error()}
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, raw) {
		return &domain.Error{Code: "NONCANONICAL_REDUCTION_WIRE"}
	}
	return nil
}

func parseWireMeasure(definition domain.Digest, digestRaw string, rawComponents []int64) (Measure, error) {
	if rawComponents == nil {
		return Measure{}, &domain.Error{Code: "MISSING_REDUCTION_MEASURE_COMPONENTS"}
	}
	components := make([]uint64, len(rawComponents))
	for index, value := range rawComponents {
		if value < 0 {
			return Measure{}, &domain.Error{Code: "INVALID_REDUCTION_MEASURE_COMPONENT"}
		}
		components[index] = uint64(value)
	}
	measure, err := NewMeasure(definition, components)
	if err != nil || measure.digest.String() != digestRaw {
		return Measure{}, &domain.Error{Code: "REDUCTION_MEASURE_DIGEST_MISMATCH"}
	}
	return measure, nil
}

func parseOptionalMapDigests(outcomeRaw, preservationRaw string) (*compare.OutcomeArtifactDigest, *compare.PreservationMapDigest, error) {
	if outcomeRaw == "" && preservationRaw == "" {
		return nil, nil, nil
	}
	if outcomeRaw == "" || preservationRaw == "" {
		return nil, nil, &domain.Error{Code: "PARTIAL_REDUCTION_MAP_DIGESTS"}
	}
	outcome, err := compare.ParseOutcomeArtifactDigest(outcomeRaw)
	if err != nil {
		return nil, nil, err
	}
	preservation, err := compare.ParsePreservationMapDigest(preservationRaw)
	if err != nil {
		return nil, nil, err
	}
	return &outcome, &preservation, nil
}

func parseDigestList(raw []string, requireSorted bool) ([]domain.Digest, error) {
	if raw == nil {
		return nil, &domain.Error{Code: "MISSING_REDUCTION_DIGEST_LIST"}
	}
	result := make([]domain.Digest, len(raw))
	seen := map[domain.Digest]struct{}{}
	for index, value := range raw {
		digest, err := domain.ParseDigest(value)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[digest]; duplicate {
			return nil, &domain.Error{Code: "DUPLICATE_REDUCTION_DIGEST"}
		}
		if requireSorted && index > 0 && result[index-1].String() >= digest.String() {
			return nil, &domain.Error{Code: "UNSORTED_REDUCTION_DIGEST_LIST"}
		}
		seen[digest] = struct{}{}
		result[index] = digest
	}
	return result, nil
}
