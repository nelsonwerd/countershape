package reduce

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func wireCompleteRun(t *testing.T) ReductionRun {
	t.Helper()
	set := engineReducerSet(t)
	original := engineNode(t, 1, 3)
	smaller := engineNode(t, 2, 2)
	finalNeighbor := engineNode(t, 3, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, smaller, "node.2")},
		smaller.digest:  {engineProposal(t, set, smaller, finalNeighbor, "node.3")},
	}
	budget, err := NewBudget(10, 20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "wire-complete/v1"), nextOffset: 1710,
		preserve: map[domain.Digest]bool{smaller.digest: true},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original = original
	input.ReducerSet = set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if run.Transcript().FinalSweepState() != FinalSweepComplete || len(run.Transcript().Entries()) != 3 {
		t.Fatalf("wire fixture did not include search plus fresh final sweep: %#v", run.Transcript())
	}
	return run
}

func wireProtocolRefusalRun(t *testing.T, overBudget bool) ReductionRun {
	t.Helper()
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, err := NewBudget(10, 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "wire-protocol-refusal/v1"), nextOffset: 1760,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	input.Evaluate = func(context.Context, engineStimulus, Neighbor, domain.AttemptPurpose, EvaluationAllowance) (EvaluationObservation, error) {
		if overBudget {
			return EvaluationObservation{
				UnresolvedReason: ReasonIncomplete, UnresolvedEvidence: engineUnresolvedEvidence(t, 1770), CandidateTrials: 3,
			}, nil
		}
		return EvaluationObservation{UnresolvedReason: UnresolvedReason("NOT_A_CLOSED_REASON")}, nil
	}
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func wireIdentities(t *testing.T, run ReductionRun) (reductionRunIdentity, transcriptIdentity) {
	t.Helper()
	var runIdentity reductionRunIdentity
	if err := json.Unmarshal(run.CanonicalBytes(), &runIdentity); err != nil {
		t.Fatal(err)
	}
	var transcript transcriptIdentity
	if err := json.Unmarshal(run.Transcript().CanonicalBytes(), &transcript); err != nil {
		t.Fatal(err)
	}
	return runIdentity, transcript
}

func wireCanonical(t *testing.T, value any) []byte {
	t.Helper()
	bytes, err := canon.CanonicalizeTyped(value)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func wireWithCanonicalTrailingUnknown(t *testing.T, exact []byte) []byte {
	t.Helper()
	if len(exact) < 2 || exact[len(exact)-1] != '}' {
		t.Fatal("wire fixture is not a canonical object")
	}
	mutated := append([]byte(nil), exact[:len(exact)-1]...)
	mutated = append(mutated, []byte(`,"zz_unknown_authority":true}`)...)
	value, err := canon.Parse(mutated)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, mutated) {
		t.Fatalf("unknown-field mutation is not canonical: %v", err)
	}
	return mutated
}

func wireRecomputeProposalDigest(t *testing.T, transcript *transcriptIdentity, index int) {
	t.Helper()
	entry := &transcript.Entries[index]
	rule, err := NewReducerRule(entry.RuleName, entry.RuleVersion)
	if err != nil {
		t.Fatal(err)
	}
	proposal, _, err := digestTyped("ReductionProposal", neighborIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReductionProposal",
		CurrentStimulusDigest: entry.ParentStimulusDigest, CurrentMeasureDigest: entry.BeforeMeasureDigest,
		StimulusDigest: entry.NeighborStimulusDigest, MeasureDigest: entry.AfterMeasureDigest,
		RuleName: entry.RuleName, RuleVersion: entry.RuleVersion, RuleDigest: rule.Digest().String(),
		Locus: entry.Locus, TransformPriority: entry.TransformPriority, ReducerSetDigest: transcript.ReducerSetDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	entry.ProposalDigest = proposal.String()
}

func TestReductionWireRoundTripRetainsExactMapDigestsAndRepeatedFinalProposal(t *testing.T) {
	run := wireCompleteRun(t)
	transcript, err := ParseTranscriptRecord(run.Transcript().CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	record, err := ParseReductionRunRecord(run.CanonicalBytes(), run.Transcript().CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	if transcript.Digest() != run.Transcript().Digest() || record.Digest() != run.Digest() ||
		record.Transcript().Digest() != transcript.Digest() || record.Grade() != GradeBestKnown ||
		len(transcript.Entries()) != 3 {
		t.Fatalf("wire round trip lost exact reduction identity: %#v %#v", transcript, record)
	}

	liveEntries := run.Transcript().Entries()
	parsedEntries := transcript.Entries()
	for index := range liveEntries {
		liveOutcome, liveOutcomePresent := liveEntries[index].Evaluation().ObservedOutcomeMapDigest()
		livePreservation, livePreservationPresent := liveEntries[index].Evaluation().ObservedPreservationDigest()
		parsedOutcome, parsedOutcomePresent := parsedEntries[index].ObservedOutcomeMapDigest()
		parsedPreservation, parsedPreservationPresent := parsedEntries[index].ObservedPreservationMapDigest()
		if liveOutcomePresent != parsedOutcomePresent || livePreservationPresent != parsedPreservationPresent ||
			liveOutcome.String() != parsedOutcome.String() || livePreservation.String() != parsedPreservation.String() {
			t.Fatalf("entry %d lost exact map digests", index)
		}
	}

	// The same direct neighbor is evaluated once during search and again with
	// fresh FINAL_SWEEP evidence. Repeated proposal identity is legitimate;
	// repeated execution evidence is not.
	if parsedEntries[1].ProposalDigest() != parsedEntries[2].ProposalDigest() ||
		parsedEntries[1].Purpose() != domain.AttemptReduction ||
		parsedEntries[2].Purpose() != domain.AttemptFinalSweep {
		t.Fatal("wire fixture did not retain the legitimate search/final proposal repetition")
	}
}

func TestTranscriptWireRejectsAuthorityChangingMutations(t *testing.T) {
	run := wireCompleteRun(t)
	_, original := wireIdentities(t, run)

	testCases := []struct {
		name   string
		mutate func(*transcriptIdentity)
	}{
		{
			name: "missing-map-digest",
			mutate: func(value *transcriptIdentity) {
				value.Entries[0].ObservedOutcomeMapDigest = ""
			},
		},
		{
			name: "preserves-with-changed-preservation-map",
			mutate: func(value *transcriptIdentity) {
				value.Entries[0].ObservedPreservationMapDigest = digest(1778).String()
			},
		},
		{
			name: "changes-with-matching-preservation-map",
			mutate: func(value *transcriptIdentity) {
				value.Entries[1].ObservedPreservationMapDigest = value.BaselinePreservationDigest
			},
		},
		{
			name: "final-sweep-changes-with-matching-preservation-map",
			mutate: func(value *transcriptIdentity) {
				value.Entries[2].ObservedPreservationMapDigest = value.BaselinePreservationDigest
			},
		},
		{
			name: "trial-delta-mismatch",
			mutate: func(value *transcriptIdentity) {
				value.Entries[1].CandidateTrials++
			},
		},
		{
			name: "proposal-fields-disagree-with-digest",
			mutate: func(value *transcriptIdentity) {
				value.Entries[0].Locus = "node.other"
			},
		},
		{
			name: "execution-evidence-reuse",
			mutate: func(value *transcriptIdentity) {
				value.Entries[1].BatchDigests = append([]string(nil), value.Entries[0].BatchDigests...)
			},
		},
		{
			name: "complete-final-sweep-set-mismatch",
			mutate: func(value *transcriptIdentity) {
				value.FinalNeighborDigests[0] = digest(1777).String()
			},
		},
		{
			name: "not-run-with-final-evidence",
			mutate: func(value *transcriptIdentity) {
				value.FinalSweepState = string(FinalSweepNotRun)
			},
		},
		{
			name: "accepted-path-mismatch",
			mutate: func(value *transcriptIdentity) {
				value.AcceptedProposalDigests = []string{}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := original
			mutated.Entries = append([]transcriptEvaluationIdentity(nil), original.Entries...)
			mutated.AcceptedProposalDigests = append([]string(nil), original.AcceptedProposalDigests...)
			mutated.FinalNeighborDigests = append([]string(nil), original.FinalNeighborDigests...)
			for index := range mutated.Entries {
				mutated.Entries[index].BatchDigests = append([]string(nil), original.Entries[index].BatchDigests...)
				mutated.Entries[index].AttemptDigests = append([]string(nil), original.Entries[index].AttemptDigests...)
				mutated.Entries[index].WorldDigests = append([]string(nil), original.Entries[index].WorldDigests...)
				mutated.Entries[index].ObservationDigests = append([]string(nil), original.Entries[index].ObservationDigests...)
			}
			testCase.mutate(&mutated)
			if _, err := ParseTranscriptRecord(wireCanonical(t, mutated)); err == nil {
				t.Fatal("authority-changing transcript mutation was accepted")
			}
		})
	}

	noncanonical := append(run.Transcript().CanonicalBytes(), '\n')
	if _, err := ParseTranscriptRecord(noncanonical); err == nil {
		t.Fatal("noncanonical transcript bytes were accepted")
	}
}

func TestReductionRunWireRejectsTranscriptPolicyAndGradeMutations(t *testing.T) {
	run := wireCompleteRun(t)
	original, _ := wireIdentities(t, run)
	testCases := []struct {
		name   string
		mutate func(*reductionRunIdentity)
	}{
		{
			name: "proposal-budget-mismatch",
			mutate: func(value *reductionRunIdentity) {
				value.ProposalLimit++
			},
		},
		{
			name: "limitations-mismatch",
			mutate: func(value *reductionRunIdentity) {
				value.Limitations = append(value.Limitations, "FORGED_LIMITATION")
			},
		},
		{
			name: "grade-does-not-follow-accepted-path",
			mutate: func(value *reductionRunIdentity) {
				value.Grade = string(GradeUnchanged)
			},
		},
		{
			name: "minimized-stimulus-does-not-follow-accepted-path",
			mutate: func(value *reductionRunIdentity) {
				value.MinimizedStimulusDigest = value.OriginalStimulusDigest
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := original
			mutated.Limitations = append([]string(nil), original.Limitations...)
			testCase.mutate(&mutated)
			if _, err := ParseReductionRunRecord(wireCanonical(t, mutated), run.Transcript().CanonicalBytes()); err == nil {
				t.Fatal("authority-changing run mutation was accepted")
			}
		})
	}
}

func TestReductionWireRejectsUnknownCanonicalFields(t *testing.T) {
	run := wireCompleteRun(t)
	if _, err := ParseTranscriptRecord(wireWithCanonicalTrailingUnknown(t, run.Transcript().CanonicalBytes())); err == nil {
		t.Fatal("unknown canonical transcript field was silently dropped")
	}

	if _, err := ParseReductionRunRecord(wireWithCanonicalTrailingUnknown(t, run.CanonicalBytes()), run.Transcript().CanonicalBytes()); err == nil {
		t.Fatal("unknown canonical run field was silently dropped")
	}

	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present {
		t.Fatalf("wire fixture lacks completed draft: present=%t err=%v", present, err)
	}
	if _, err := ParseCompletedSweepDraft(wireWithCanonicalTrailingUnknown(t, draft.CanonicalBytes())); err == nil {
		t.Fatal("unknown canonical completed-sweep field was silently dropped")
	}
}

func TestCompletedSweepDraftRejectsNullEnumerationArrays(t *testing.T) {
	run := wireCompleteRun(t)
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present {
		t.Fatalf("wire fixture lacks completed draft: present=%t err=%v", present, err)
	}
	for _, field := range []string{"enumerated_neighbor_digests", "evaluations"} {
		var body completedSweepDraftIdentity
		if err := json.Unmarshal(draft.CanonicalBytes(), &body); err != nil {
			t.Fatal(err)
		}
		if field == "enumerated_neighbor_digests" {
			body.EnumeratedNeighborDigests = nil
		} else {
			body.Evaluations = nil
		}
		if _, err := ParseCompletedSweepDraft(wireCanonical(t, body)); err == nil {
			t.Fatalf("null %s was accepted as completed enumeration", field)
		}
	}
}

func TestTranscriptParserEnforcesTerminalProtocolRefusalSemantics(t *testing.T) {
	assertRejected := func(t *testing.T, identity transcriptIdentity) {
		t.Helper()
		if _, err := ParseTranscriptRecord(wireCanonical(t, identity)); err == nil {
			t.Fatal("forged terminal protocol-refusal transcript was accepted")
		}
	}

	overBudgetRun := wireProtocolRefusalRun(t, true)
	var overBudget transcriptIdentity
	if err := json.Unmarshal(overBudgetRun.Transcript().CanonicalBytes(), &overBudget); err != nil {
		t.Fatal(err)
	}
	if len(overBudget.Entries) != 1 || overBudget.Entries[0].ReasonCode != string(ReasonEvaluatorExceededBudget) {
		t.Fatal("wire fixture did not produce the over-budget terminal refusal")
	}

	t.Run("budget reason requires actual overage", func(t *testing.T) {
		mutated := overBudget
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), overBudget.Entries...)
		mutated.Entries[0].CandidateTrials = mutated.CandidateTrialLimit
		mutated.Entries[0].TotalCandidateTrials = mutated.CandidateTrialLimit
		assertRejected(t, mutated)
	})
	t.Run("actual overage requires budget reason", func(t *testing.T) {
		mutated := overBudget
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), overBudget.Entries...)
		mutated.Entries[0].ReasonCode = string(ReasonInvalidEvaluatorResult)
		mutated.Limitations = []string{string(ReasonInvalidEvaluatorResult)}
		assertRejected(t, mutated)
	})
	t.Run("terminal reason requires matching limitation", func(t *testing.T) {
		mutated := overBudget
		mutated.Limitations = []string{}
		assertRejected(t, mutated)
	})
	t.Run("terminal reason cannot claim complete sweep", func(t *testing.T) {
		mutated := overBudget
		mutated.FinalSweepState = string(FinalSweepComplete)
		assertRejected(t, mutated)
	})
	t.Run("terminal limitation cannot name a second refusal", func(t *testing.T) {
		mutated := overBudget
		mutated.Limitations = append([]string(nil), overBudget.Limitations...)
		mutated.Limitations = append(mutated.Limitations, string(ReasonInvalidEvaluatorResult))
		assertRejected(t, mutated)
	})

	invalidRun := wireProtocolRefusalRun(t, false)
	var invalid transcriptIdentity
	if err := json.Unmarshal(invalidRun.Transcript().CanonicalBytes(), &invalid); err != nil {
		t.Fatal(err)
	}
	t.Run("terminal reason must be final entry", func(t *testing.T) {
		mutated := invalid
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), invalid.Entries...)
		continuation := invalid.Entries[0]
		continuation.EvaluationID = "eval:000002-reduction"
		continuation.ProposalCount = 2
		continuation.ReasonCode = string(ReasonIncomplete)
		mutated.Entries = append(mutated.Entries, continuation)
		assertRejected(t, mutated)
	})
	t.Run("observed-map-without-trials reason requires zero trials", func(t *testing.T) {
		mutated := invalid
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), invalid.Entries...)
		mutated.Entries[0].ReasonCode = string(ReasonObservedMapWithoutTrials)
		mutated.Entries[0].CandidateTrials = 1
		mutated.Entries[0].TotalCandidateTrials = 1
		mutated.Limitations = []string{string(ReasonObservedMapWithoutTrials)}
		assertRejected(t, mutated)
	})
	t.Run("missing-evidence reason requires a charged trial", func(t *testing.T) {
		mutated := invalid
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), invalid.Entries...)
		mutated.Entries[0].ReasonCode = string(ReasonUnresolvedEvidenceMissing)
		mutated.Limitations = []string{string(ReasonUnresolvedEvidenceMissing)}
		assertRejected(t, mutated)
	})
	t.Run("reused-evidence refusal cannot retain the reused identities", func(t *testing.T) {
		mutated := invalid
		mutated.Entries = append([]transcriptEvaluationIdentity(nil), invalid.Entries...)
		mutated.Entries[0].ReasonCode = string(ReasonReusedEvaluationEvidence)
		mutated.Entries[0].BatchDigests = []string{digest(188_011).String()}
		mutated.Entries[0].AttemptDigests = []string{digest(188_012).String()}
		mutated.Entries[0].WorldDigests = []string{digest(188_013).String()}
		mutated.Entries[0].ObservationDigests = []string{digest(188_014).String()}
		mutated.Limitations = []string{string(ReasonReusedEvaluationEvidence)}
		assertRejected(t, mutated)
	})
}

func TestTranscriptParserRejectsMidstreamTerminalProtocolRefusal(t *testing.T) {
	run := wireProtocolRefusalRun(t, false)
	_, transcript := wireIdentities(t, run)
	continuation := transcript.Entries[0]
	continuation.EvaluationID = "eval:000002-reduction"
	continuation.ProposalCount = 2
	continuation.ReasonCode = string(ReasonIncomplete)
	transcript.Entries = append(transcript.Entries, continuation)
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript accepted a terminal protocol refusal before a later entry")
	}
}

func TestStandaloneTranscriptParserRejectsDisconnectedParentChain(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	if len(transcript.Entries) < 2 {
		t.Fatal("wire fixture lacks a multi-step chain")
	}
	entry := &transcript.Entries[1]
	entry.ParentStimulusDigest = digest(188_001).String()
	wireRecomputeProposalDigest(t, &transcript, 1)
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("standalone transcript accepted individually valid but disconnected proposal intents")
	}
}

func TestTranscriptParserRejectsPartialUnresolvedEvidenceDomains(t *testing.T) {
	run := wireProtocolRefusalRun(t, false)
	_, transcript := wireIdentities(t, run)
	transcript.Entries[0].BatchDigests = []string{digest(188_020).String()}
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript accepted only one of four unresolved evidence domains")
	}
}

func TestTranscriptParserRejectsImpossibleObservationAccounting(t *testing.T) {
	t.Run("observed map has no candidate trial", func(t *testing.T) {
		run := wireCompleteRun(t)
		_, transcript := wireIdentities(t, run)
		transcript.Entries = append([]transcriptEvaluationIdentity(nil), transcript.Entries[:1]...)
		entry := &transcript.Entries[0]
		entry.Decision = string(Changes)
		entry.ReasonCode = "EXACT_PRESERVATION_MAP_CHANGED"
		entry.CandidateTrials = 0
		entry.TotalCandidateTrials = 0
		transcript.AcceptedProposalDigests = []string{}
		transcript.Limitations = []string{}
		transcript.FinalSweepState = string(FinalSweepNotRun)
		transcript.FinalNeighborDigests = []string{}
		if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
			t.Fatal("transcript accepted an observed map without a candidate trial")
		}
	})

	t.Run("charged ordinary unresolved entry has no evidence", func(t *testing.T) {
		run := wireProtocolRefusalRun(t, false)
		_, transcript := wireIdentities(t, run)
		entry := &transcript.Entries[0]
		entry.ReasonCode = string(ReasonIncomplete)
		entry.CandidateTrials = 1
		entry.TotalCandidateTrials = 1
		transcript.Limitations = []string{"UNRESOLVED_" + string(ReasonIncomplete)}
		if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
			t.Fatal("transcript accepted charged unresolved evaluation without all four evidence domains")
		}
	})
}

func TestTranscriptParserRejectsMapBackedTrialCountMismatch(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	if len(transcript.Entries) == 0 || transcript.Entries[0].CandidateTrials != 2 ||
		len(transcript.Entries[0].AttemptDigests) != 2 {
		t.Fatal("wire fixture no longer starts with two typed attempts")
	}
	transcript.Entries = append([]transcriptEvaluationIdentity(nil), transcript.Entries...)
	transcript.Entries[0].CandidateTrials = 1
	for index := range transcript.Entries {
		transcript.Entries[index].TotalCandidateTrials--
	}
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript accepted a map-backed count below its typed attempt evidence")
	}
}

func TestTranscriptParserRejectsOrphanTerminalProtocolLimitation(t *testing.T) {
	run := wireCompleteRun(t)
	_, base := wireIdentities(t, run)
	for _, limitation := range []string{
		string(ReasonInvalidEvaluatorResult),
		"FINAL_SWEEP_PRESERVING_NEIGHBOR",
		"FINAL_SWEEP_UNRESOLVED_" + string(ReasonIncomplete),
	} {
		t.Run(limitation, func(t *testing.T) {
			transcript := base
			transcript.Limitations = append([]string(nil), base.Limitations...)
			transcript.Limitations = append(transcript.Limitations, limitation)
			if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
				t.Fatal("transcript accepted a terminal limitation with no matching terminal entry")
			}
		})
	}
}

func TestTranscriptParserRejectsEntryAfterCandidateTrialBudgetReached(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	if len(transcript.Entries) < 2 || transcript.Entries[0].TotalCandidateTrials < 1 {
		t.Fatal("wire fixture lacks a charged multi-entry prefix")
	}
	transcript.Entries = append([]transcriptEvaluationIdentity(nil), transcript.Entries[:2]...)
	transcript.CandidateTrialLimit = transcript.Entries[0].TotalCandidateTrials
	continuation := &transcript.Entries[1]
	continuation.CandidateTrials = 0
	continuation.TotalCandidateTrials = transcript.CandidateTrialLimit
	continuation.Decision = string(Unresolved)
	continuation.ReasonCode = string(ReasonIncomplete)
	continuation.ObservedOutcomeMapDigest = ""
	continuation.ObservedPreservationMapDigest = ""
	continuation.BatchDigests = []string{}
	continuation.AttemptDigests = []string{}
	continuation.WorldDigests = []string{}
	continuation.ObservationDigests = []string{}
	transcript.Limitations = []string{"UNRESOLVED_" + string(ReasonIncomplete)}
	transcript.FinalSweepState = string(FinalSweepNotRun)
	transcript.FinalNeighborDigests = []string{}
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript continued after the prior entry consumed the exact candidate-trial budget")
	}
}

func TestTranscriptParserRejectsSelfNeighborProposal(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	transcript.Entries = append([]transcriptEvaluationIdentity(nil), transcript.Entries[:1]...)
	entry := &transcript.Entries[0]
	entry.NeighborStimulusDigest = entry.ParentStimulusDigest
	entry.Decision = string(Changes)
	entry.ReasonCode = "EXACT_PRESERVATION_MAP_CHANGED"
	wireRecomputeProposalDigest(t, &transcript, 0)
	transcript.AcceptedProposalDigests = []string{}
	transcript.Limitations = []string{}
	transcript.FinalSweepState = string(FinalSweepNotRun)
	transcript.FinalNeighborDigests = []string{}
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript accepted one stimulus as both proposal parent and neighbor")
	}
}

func TestTranscriptParserRejectsContinuationAfterFinalSweepNonChanges(t *testing.T) {
	run := wireCompleteRun(t)
	_, base := wireIdentities(t, run)
	for _, decision := range []Decision{Preserves, Unresolved} {
		t.Run(string(decision), func(t *testing.T) {
			transcript := base
			transcript.Entries = append([]transcriptEvaluationIdentity(nil), base.Entries...)
			entry := &transcript.Entries[1]
			entry.Purpose = string(domain.AttemptFinalSweep)
			if decision == Preserves {
				entry.Decision = string(Preserves)
				entry.ReasonCode = "EXACT_PRESERVATION_MAP_MATCH"
				transcript.Limitations = []string{"FINAL_SWEEP_PRESERVING_NEIGHBOR"}
			} else {
				entry.Decision = string(Unresolved)
				entry.ReasonCode = string(ReasonIncomplete)
				entry.ObservedOutcomeMapDigest = ""
				entry.ObservedPreservationMapDigest = ""
				transcript.Limitations = []string{"FINAL_SWEEP_UNRESOLVED_" + string(ReasonIncomplete)}
			}
			transcript.FinalSweepState = string(FinalSweepIncomplete)
			transcript.FinalNeighborDigests = []string{}
			if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
				t.Fatal("transcript continued after a terminal final-sweep decision")
			}
		})
	}
}

func TestTranscriptParserRequiresSortedBaselineEvidence(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	if len(transcript.BaselineBatchDigests) < 2 {
		t.Fatal("wire fixture lacks a sortable baseline batch set")
	}
	transcript.BaselineBatchDigests[0], transcript.BaselineBatchDigests[1] =
		transcript.BaselineBatchDigests[1], transcript.BaselineBatchDigests[0]
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("transcript accepted a noncanonical baseline evidence order")
	}
}

func TestReductionRunParserRejectsTranscriptBaselineMismatch(t *testing.T) {
	run := wireCompleteRun(t)
	runIdentity, transcript := wireIdentities(t, run)
	transcript.BaselineOutcomeMapDigest = digest(188_030).String()
	transcript.BaselinePreservationDigest = digest(188_031).String()
	transcriptDigest, transcriptBytes, err := digestTyped("ReductionTranscript", transcript)
	if err != nil {
		t.Fatal(err)
	}
	runIdentity.TranscriptDigest = transcriptDigest.String()
	if _, err := ParseReductionRunRecord(wireCanonical(t, runIdentity), transcriptBytes); err == nil {
		t.Fatal("run accepted a transcript bound to a different baseline identity")
	}
}

func TestTranscriptParserRejectsEvaluationEvidenceReusedFromBaseline(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	if len(transcript.Entries) == 0 {
		t.Fatal("wire fixture lacks evaluation evidence")
	}
	transcript.Entries[0].BatchDigests = append([]string(nil), transcript.BaselineBatchDigests...)
	transcript.Entries[0].AttemptDigests = append([]string(nil), transcript.BaselineAttemptDigests...)
	transcript.Entries[0].WorldDigests = append([]string(nil), transcript.BaselineWorldDigests...)
	transcript.Entries[0].ObservationDigests = append([]string(nil), transcript.BaselineObservationDigests...)
	if _, err := ParseTranscriptRecord(wireCanonical(t, transcript)); err == nil {
		t.Fatal("serialized evaluation reused original baseline evidence")
	}
}

func TestSerializedTranscriptReplayRequiresTrustedBaselineAndNewNoncollidingEvidence(t *testing.T) {
	run := wireCompleteRun(t)
	record, err := ParseTranscriptRecord(run.Transcript().CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	baselineMap := run.Baseline().OutcomeMap()
	if record.BaselineOutcomeMapDigest() != baselineMap.ArtifactDigest() ||
		record.BaselinePreservationMapDigest() != baselineMap.PreservationDigest() {
		t.Fatal("serialized transcript lost its typed baseline map identities")
	}
	plan := record.ReplayPlan()
	if len(plan) != len(run.Transcript().Entries()) || len(plan) == 0 {
		t.Fatalf("serialized replay plan has %d steps", len(plan))
	}
	for index, step := range plan {
		if step.EvaluationID == "" || !step.ProposalDigest.Valid() || !step.ParentDigest.Valid() ||
			!step.NeighborDigest.Valid() || !step.BeforeMeasure.Valid() || !step.AfterMeasure.Valid() ||
			!step.ReducerSetDigest.Valid() || step.ReducerSetDigest != record.ReducerSetDigest() ||
			step.MeasureDefinitionDigest != record.MeasureDefinitionDigest() || step.ScopeDigest != record.ScopeDigest() ||
			step.RuleName == "" || step.RuleVersion == "" || step.Locus == "" || !step.Purpose.Valid() ||
			(step.RecordedDecision != Preserves && step.RecordedDecision != Changes && step.RecordedDecision != Unresolved) ||
			step.RecordedReason == "" || step.RecordedProposalCount != uint64(index+1) ||
			step.RecordedTotalTrials < step.RecordedCandidateTrials {
			t.Fatalf("serialized replay intent is incomplete: %#v", step)
		}
		outcome, outcomePresent := step.ArchivedOutcomeMapDigest()
		preservation, preservationPresent := step.ArchivedPreservationMapDigest()
		if outcomePresent != preservationPresent ||
			(step.RecordedDecision != Unresolved && (!outcome.Valid() || !preservation.Valid())) {
			t.Fatalf("serialized replay step %d lost observed map identities", index)
		}
	}

	executions := 0
	receipts, err := ExecuteReplay(context.Background(), record, run.Baseline(), func(_ context.Context, step ReplayStep) (ReplayEvidence, error) {
		executions++
		base := 200_000 + executions*10
		return NewReplayEvidence(
			[]domain.Digest{digest(base + 1)}, []domain.Digest{digest(base + 2)},
			[]domain.Digest{digest(base + 3)}, []domain.Digest{digest(base + 4)},
		)
	})
	if err != nil || executions != len(plan) || len(receipts) != len(plan) {
		t.Fatalf("serialized replay did not invoke fresh execution per step: calls=%d receipts=%d err=%v", executions, len(receipts), err)
	}
	for index, receipt := range receipts {
		if receipt.Step().ProposalDigest != plan[index].ProposalDigest || !receipt.Evidence().Valid() {
			t.Fatalf("replay receipt %d lost intent/fresh evidence", index)
		}
	}

	archived := plan[0]
	reused, err := NewReplayEvidence(
		archived.ArchivedBatchDigests(), archived.ArchivedAttemptDigests(),
		archived.ArchivedWorldDigests(), archived.ArchivedObservationDigests(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExecuteReplay(context.Background(), record, run.Baseline(), func(context.Context, ReplayStep) (ReplayEvidence, error) {
		return reused, nil
	}); err == nil {
		t.Fatal("serialized replay reused archived captures instead of requiring execution")
	}

	baselineReused, err := NewReplayEvidence(
		record.BaselineBatchDigests(), record.BaselineAttemptDigests(),
		record.BaselineWorldDigests(), record.BaselineObservationDigests(),
	)
	if err != nil {
		t.Fatal(err)
	}
	baselineReplayCalls := 0
	if _, err := ExecuteReplay(context.Background(), record, run.Baseline(), func(context.Context, ReplayStep) (ReplayEvidence, error) {
		baselineReplayCalls++
		if baselineReplayCalls == 1 {
			return baselineReused, nil
		}
		base := 300_000 + baselineReplayCalls*10
		return NewReplayEvidence(
			[]domain.Digest{digest(base + 1)}, []domain.Digest{digest(base + 2)},
			[]domain.Digest{digest(base + 3)}, []domain.Digest{digest(base + 4)},
		)
	}); err == nil {
		t.Fatal("serialized replay reused original baseline captures")
	}

	callbackInvoked := false
	wrongBaseline := wireProtocolRefusalRun(t, false).Baseline()
	if _, err := ExecuteReplay(context.Background(), record, wrongBaseline, func(context.Context, ReplayStep) (ReplayEvidence, error) {
		callbackInvoked = true
		return ReplayEvidence{}, nil
	}); err == nil || callbackInvoked {
		t.Fatalf("replay accepted a different typed baseline: callback_invoked=%t err=%v", callbackInvoked, err)
	}

	_, alteredIdentity := wireIdentities(t, run)
	alteredIdentity.BaselineBatchDigests = []string{digest(300_999).String()}
	alteredRecord, err := ParseTranscriptRecord(wireCanonical(t, alteredIdentity))
	if err != nil {
		t.Fatalf("inert transcript declaration did not parse: %v", err)
	}
	callbackInvoked = false
	if _, err := ExecuteReplay(context.Background(), alteredRecord, run.Baseline(), func(context.Context, ReplayStep) (ReplayEvidence, error) {
		callbackInvoked = true
		return ReplayEvidence{}, nil
	}); err == nil || callbackInvoked {
		t.Fatalf("typed replay trusted altered transcript baseline evidence: callback_invoked=%t err=%v", callbackInvoked, err)
	}
}

func TestSerializedTranscriptReplayRejectsUnrelatedFirstParentBeforeCallback(t *testing.T) {
	run := wireCompleteRun(t)
	_, transcript := wireIdentities(t, run)
	transcript.Entries = append([]transcriptEvaluationIdentity(nil), transcript.Entries...)
	transcript.Entries[0].ParentStimulusDigest = digest(200_100).String()
	wireRecomputeProposalDigest(t, &transcript, 0)
	transcript.AcceptedProposalDigests = append([]string(nil), transcript.AcceptedProposalDigests...)
	transcript.AcceptedProposalDigests[0] = transcript.Entries[0].ProposalDigest
	record, err := ParseTranscriptRecord(wireCanonical(t, transcript))
	if err != nil {
		t.Fatalf("standalone archive should remain inertly parseable: %v", err)
	}
	executions := 0
	_, err = ExecuteReplay(context.Background(), record, run.Baseline(), func(context.Context, ReplayStep) (ReplayEvidence, error) {
		executions++
		return ReplayEvidence{}, nil
	})
	if err == nil || executions != 0 {
		t.Fatalf("typed replay accepted an unrelated first parent: callbacks=%d err=%v", executions, err)
	}
}
