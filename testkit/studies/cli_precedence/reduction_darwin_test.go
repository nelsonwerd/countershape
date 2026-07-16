//go:build darwin && cgo

package cli_precedence

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
)

func TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence(t *testing.T) {
	appMode, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	noise, err := countercli.PresentEnvironment("Z_NOISE", "ignored")
	if err != nil {
		t.Fatal(err)
	}
	configFixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	noisyStimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{clifixture.Entrypoint}, Argv: []string{"--mode", "argv"},
		Stdin: countercli.AbsentStdin(), Environment: []countercli.CLIEnvironmentBinding{appMode, noise},
		Fixtures: []countercli.CLIFixtureFile{configFixture}, CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := cliPhysicalReductionConfig(t)
	baselineConfig.StimulusOverride = &noisyStimulus
	baselineCheckpoint, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatal("physical CLI baseline did not produce a discovery outcome map")
	}
	if !baselineCheckpoint.HasOutcomeMap || baselineCheckpoint.Plan.Digest() != baselineResult.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineResult.OutcomeMap).Relation() != compare.PreservationEqual {
		t.Fatal("physical CLI pre-divergence baseline checkpoint changed semantic plan or labeled behavior")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baselineResult.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if budget.ProposalLimit() != 5 || budget.CandidateTrialLimit() != 36 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled CLI reduction budget = %d/%d/%s", budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	var evaluatorErr error
	var minimizedStudy StudyResult
	run, err := reducer.Run(context.Background(), reducer.RunInput[countercli.CLIStimulus]{
		Original: baselineResult.Stimulus,
		Reference: func(stimulus countercli.CLIStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := countercli.MeasureCLIStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus countercli.CLIStimulus) ([]reducer.TypedProposal[countercli.CLIStimulus], error) {
			neighbors, enumerateErr := countercli.EnumerateCLINeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[countercli.CLIStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[countercli.CLIStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus countercli.CLIStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := cliPhysicalReductionConfig(t)
			requiredTrials := uint64(config.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			config.Purpose = purpose
			config.StimulusOverride = &stimulus
			evaluationContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := Run(evaluationContext, config)
			if runErr != nil {
				evaluatorErr = runErr
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				evaluatorErr = context.Canceled
				return reducer.EvaluationObservation{}, evaluatorErr
			}
			outcome := result.OutcomeMap
			if compare.AssessPreservation(baseline.OutcomeMap(), outcome).Relation() == compare.PreservationEqual {
				minimizedStudy = result
			}
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluatorErr != nil {
		t.Fatalf("physical CLI reducer evaluator failed: %v", evaluatorErr)
	}
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepComplete || len(run.Transcript().Entries()) != 4 ||
		len(run.Transcript().AcceptedPath()) != 1 || run.MinimizedStimulusDigest() == baselineResult.Stimulus.Digest() {
		t.Fatalf("unexpected physical CLI reduction: grade=%s accepted=%t sweep=%s entries=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), run.Transcript().FinalSweepState(),
			len(run.Transcript().Entries()), run.Transcript().Limitations())
	}
	entries := run.Transcript().Entries()
	preserving := entries[1].Evaluation()
	finalSweep := entries[len(entries)-1].Evaluation()
	if preserving.Purpose() != domain.AttemptReduction || preserving.Decision() != reducer.Preserves ||
		!preserving.LogicalNonReuseWithBaseline() || len(preserving.ObservedAttemptDigests()) != 9 ||
		finalSweep.Purpose() != domain.AttemptFinalSweep || finalSweep.Decision() != reducer.Changes ||
		!finalSweep.LogicalNonReuseWithBaseline() || len(finalSweep.ObservedAttemptDigests()) != 9 {
		t.Fatalf("physical CLI evidence is incomplete: preserving=%s/%s/%d final=%s/%s/%d",
			preserving.Purpose(), preserving.Decision(), len(preserving.ObservedAttemptDigests()),
			finalSweep.Purpose(), finalSweep.Decision(), len(finalSweep.ObservedAttemptDigests()))
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 1 ||
		draft.CurrentStimulusDigest() != run.MinimizedStimulusDigest() {
		t.Fatalf("physical CLI empty-neighbor completed sweep was not retained: present=%t err=%v", present, err)
	}
	if parsed, parseErr := reducer.ParseCompletedSweepDraft(draft.CanonicalBytes()); parseErr != nil || parsed.Digest() != draft.Digest() {
		t.Fatalf("physical CLI completed sweep did not round trip exactly: %v", parseErr)
	}
	sweepStore, err := store.OpenReductionSweepStore(filepath.Join(t.TempDir(), "sweep-store"))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := sweepStore.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := grade.Finalize(context.Background(), run, &grade.SweepCompletion{
		Store: sweepStore, Draft: draft, Authority: authority,
	})
	if err != nil || !finalized.Valid() || finalized.Grade().Status() != grade.StatusOneMinimalUnder {
		t.Fatalf("physical CLI durable grade was not quantified: status=%s err=%v",
			finalized.Grade().Status(), err)
	}
	if !minimizedStudy.HasOutcomeMap || minimizedStudy.Stimulus.Digest() != run.MinimizedStimulusDigest() {
		t.Fatal("physical CLI reducer did not retain the exact minimized preserving study")
	}
	reducedBaseline, err := compare.RequireDivergence(minimizedStudy.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	confirmationConfig := cliPhysicalReductionConfig(t)
	confirmationConfig.Purpose = domain.AttemptConfirmation
	confirmationConfig.StimulusOverride = &minimizedStudy.Stimulus
	confirmationConfig.Confirmation = &ConfirmationInput{
		ReducedBaseline: reducedBaseline, ReductionRun: run, ReductionResult: finalized,
	}
	confirmedStudy, err := Run(context.Background(), confirmationConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmedStudy.HasConfirmation || !confirmedStudy.Confirmation.Valid() || !confirmedStudy.HasOutcomeMap ||
		confirmedStudy.OutcomeMap.Phase() != domain.AttemptConfirmation ||
		confirmedStudy.OutcomeMap.ScheduleStartOffset() != 1 || len(confirmedStudy.Confirmation.Draft().PhysicalFacts()) != 9 {
		t.Fatal("physical CLI confirmation did not produce one complete fresh phase-bound matrix")
	}
	confirmationDraft := confirmedStudy.Confirmation.Draft()
	if parsed, parseErr := confirmation.ParseRecord(confirmationDraft.CanonicalBytes()); parseErr != nil || parsed.Digest() != confirmationDraft.Digest() {
		t.Fatalf("physical CLI confirmation did not round trip strictly: %v", parseErr)
	}
	confirmationWire := confirmationDraft.CanonicalBytes()
	unknownConfirmation := append([]byte(nil), confirmationWire[:len(confirmationWire)-1]...)
	unknownConfirmation = append(unknownConfirmation, []byte(`,"zz_unknown":true}`)...)
	if _, parseErr := confirmation.ParseRecord(unknownConfirmation); parseErr == nil {
		t.Fatal("FreshConfirmation parser accepted an unknown canonical member")
	}
	confirmationStore, err := store.OpenObjectStore(filepath.Join(t.TempDir(), "confirmation-store"))
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.NewSemanticObject("FreshConfirmation", confirmationDraft.Digest(), confirmationDraft.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	confirmationAuthority, err := confirmationStore.Publish(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmationStore.Validate(context.Background(), object, confirmationAuthority); err != nil {
		t.Fatal(err)
	}
	originalArtifact, err := choice.NewCanonicalArtifact(
		"CLIStimulus", baselineResult.Stimulus.Digest(), baselineResult.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact(
		"CLIStimulus", confirmedStudy.Stimulus.Digest(), confirmedStudy.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reveals := make([]choice.CandidateReveal, len(confirmedStudy.CandidateBindings))
	for index, binding := range confirmedStudy.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(confirmedStudy.CandidateRoles[binding.Key()]),
			ProducerMetadata: "local deterministic CLI fixture",
		}
	}
	choicepoint, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: "Which exact CLI precedence behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, Confirmation: confirmationDraft.Record(), CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || !choicepoint.Valid() {
		t.Fatalf("physical CLI Choicepoint construction failed: %v", err)
	}
	parsedChoicepoint, err := choice.ParseChoicepointRecord(choicepoint.CanonicalBytes())
	if err != nil || parsedChoicepoint.Digest() != choicepoint.Digest() {
		t.Fatalf("physical CLI Choicepoint did not round trip strictly: %v", err)
	}
	blind, err := choice.NewBlindView(choicepoint)
	if err != nil || len(blind.DTO().Cards()) != confirmedStudy.OutcomeMap.DistinctProjectionCount() {
		t.Fatalf("physical CLI blind DTO did not group exact outcomes: %v", err)
	}
	blindBytes := blind.DTO().CanonicalBytes()
	for _, binding := range confirmedStudy.CandidateBindings {
		for _, forbidden := range []string{
			binding.Key().String(), binding.Identity().TreeIdentityDigest.String(),
			string(confirmedStudy.CandidateRoles[binding.Key()]), "local deterministic CLI fixture",
		} {
			if forbidden != "" && bytes.Contains(blindBytes, []byte(forbidden)) {
				t.Fatalf("blind DTO leaked reveal or candidate identity %q", forbidden)
			}
		}
	}
	for _, entry := range confirmedStudy.OutcomeMap.Entries() {
		if bytes.Contains(blindBytes, []byte(entry.ProjectionFingerprint.String())) {
			t.Fatal("blind DTO leaked a raw projection fingerprint")
		}
	}
	choiceObject, err := store.NewSemanticObject("Choicepoint", choicepoint.Digest(), choicepoint.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	choiceAuthority, err := confirmationStore.Publish(context.Background(), choiceObject)
	if err != nil || confirmationStore.Validate(context.Background(), choiceObject, choiceAuthority) != nil {
		t.Fatalf("physical CLI Choicepoint did not persist immutably: %v", err)
	}
	promotionRoot := filepath.Join(t.TempDir(), "promotion-store")
	promotionStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	studyID, err := store.NewStudyID("physical CLI choicepoint promotion")
	if err != nil {
		t.Fatal(err)
	}
	head, err := promotionStore.CreateStudy(context.Background(), studyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceBaseline(context.Background(), head, baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceDivergence(context.Background(), head, baseline)
	if err != nil {
		t.Fatal(err)
	}
	wrongDivergence, err := compare.RequireDivergence(baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	wrongStudyID, err := store.NewStudyID("physical CLI mismatched reduction lineage")
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err := promotionStore.CreateStudy(context.Background(), wrongStudyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err = promotionStore.AdvanceBaseline(context.Background(), wrongHead, baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err = promotionStore.AdvanceDivergence(context.Background(), wrongHead, wrongDivergence)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := promotionStore.AdvanceReduction(context.Background(), wrongHead, run); err == nil {
		t.Fatal("ReductionRun advanced beneath a different exact divergence predecessor")
	}
	unchangedWrongHead, err := promotionStore.OpenHead(context.Background(), wrongStudyID)
	if err != nil || unchangedWrongHead.HeadDigest() != wrongHead.HeadDigest() ||
		unchangedWrongHead.Stage() != store.StageDivergence {
		t.Fatalf("rejected reduction lineage changed its head: %v", err)
	}
	if _, _, err := promotionStore.Read(context.Background(), "ReductionRun", run.Digest()); err == nil {
		t.Fatal("rejected reduction lineage published its successor object")
	}
	head, err = promotionStore.AdvanceReduction(context.Background(), head, run)
	if err != nil {
		t.Fatal(err)
	}
	storedConfirmation, err := promotion.PersistConfirmation(context.Background(), promotionStore, head, confirmationDraft)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "Which exact CLI precedence behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || ready.Record().Digest() != choicepoint.Digest() {
		t.Fatalf("physical CLI store-bound Choicepoint promotion failed: %v", err)
	}
	restartedStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || reopenedReady.Record().Digest() != ready.Record().Digest() {
		t.Fatalf("physical CLI CHOICEPOINT_READY did not survive restart: %v", err)
	}
	if _, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "stale confirmation must not promote again", Plan: confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals, EvidenceReceipts: []domain.ReceiptReference{},
	}); err == nil {
		t.Fatal("superseded confirmation authority promoted a second Choicepoint")
	}

	decisionSession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	cards := decisionSession.BlindDTO().Cards()
	if len(cards) < 2 {
		t.Fatal("divergent CLI Choicepoint did not expose at least two blind outcome cards")
	}
	standardInput := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: []string{choice.WholeProjectionFieldID},
		AllowedAliases: []string{cards[0].Alias},
	}
	if _, err := decisionSession.Propose(standardInput); !choice.IsRefusal(err, choice.CodeRequiredSurfaceNotVisited) {
		t.Fatalf("standard blind proposal bypassed evidence presentation: %v", err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
	} {
		decisionSession, err = decisionSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	decisionSession, err = decisionSession.Propose(standardInput)
	if err != nil {
		t.Fatal(err)
	}
	decisionSession, reveal, err := decisionSession.Reveal()
	if err != nil || len(reveal.Groups()) != len(cards) {
		t.Fatalf("standard decision reveal failed: %v", err)
	}
	groups := reveal.Groups()
	if len(groups[0].Candidates) == 0 {
		t.Fatal("reveal group omitted its exact supporting candidates")
	}
	originalDisplayRef := groups[0].Candidates[0].DisplayRef
	groups[0].Candidates[0].DisplayRef = "mutated caller copy"
	if reveal.Groups()[0].Candidates[0].DisplayRef != originalDisplayRef {
		t.Fatal("RevealDTO nested candidate slices were not defensively copied")
	}
	decisionSession, err = decisionSession.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decisionSession.Revise(standardInput, "unnecessary"); !choice.IsRefusal(err, choice.CodeUnnecessaryChangeRationale) {
		t.Fatalf("semantic no-op post-reveal ruling accepted a rationale: %v", err)
	}
	changedInput := standardInput
	changedInput.AllowedAliases = []string{cards[1].Alias}
	if _, err := decisionSession.Revise(changedInput, ""); !choice.IsRefusal(err, choice.CodeChangeRationaleRequired) {
		t.Fatalf("changed post-reveal ruling omitted its rationale: %v", err)
	}
	const changedRationale = "Provenance changed which exact outcome I accept."
	changedSession, changeErr := decisionSession.Revise(changedInput, changedRationale)
	if changeErr != nil || changedSession.State() != choice.SessionPostRevealRecorded {
		t.Fatalf("rationalized post-reveal change was refused: %v", changeErr)
	}
	_, changedDecision, changeErr := changedSession.Finalize(
		"local-test-operator", "Changed exact CLI witness after reveal.", []domain.ReceiptReference{},
	)
	if changeErr != nil || !changedDecision.ChangedAfterReveal() ||
		changedDecision.PostRevealChangeRationale() != changedRationale {
		t.Fatalf("rationalized post-reveal change was not retained in DecisionRecord: %v", changeErr)
	}
	parsedChanged, changeErr := choice.ParseDecisionRecord(changedDecision.CanonicalBytes(), reopenedReady.Record())
	if changeErr != nil || parsedChanged.Digest() != changedDecision.Digest() || !parsedChanged.ChangedAfterReveal() {
		t.Fatalf("rationalized DecisionRecord did not round trip strictly: %v", changeErr)
	}
	decisionSession, err = decisionSession.Revise(standardInput, "")
	if err != nil {
		t.Fatal(err)
	}
	finalizedSession, decision, err := decisionSession.Finalize("local-test-operator", "Exact CLI witness only.", []domain.ReceiptReference{})
	if err != nil || !decision.Valid() || decision.EarlyReveal() || decision.ChangedAfterReveal() ||
		decision.ReceiptStatusWhenEmpty() != domain.Unreceipted() || len(decision.Receipts()) != 0 {
		t.Fatalf("standard DecisionRecord did not preserve exact blind/reveal facts: %v", err)
	}
	if _, ok := decision.CompilableRuling(); !ok {
		t.Fatal("validated ALLOW_OBSERVED DecisionRecord lost sealed compile eligibility")
	}
	parsedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), reopenedReady.Record())
	if err != nil || parsedDecision.Digest() != decision.Digest() {
		t.Fatalf("DecisionRecord did not round trip strictly: %v", err)
	}
	tamperedCompilable := bytes.Replace(decision.CanonicalBytes(), []byte(`"compilable":true`), []byte(`"compilable":false`), 1)
	if bytes.Equal(tamperedCompilable, decision.CanonicalBytes()) {
		t.Fatal("DecisionRecord test did not locate the derived compilable projection")
	}
	if _, err := choice.ParseDecisionRecord(tamperedCompilable, reopenedReady.Record()); err == nil {
		t.Fatal("DecisionRecord parser trusted a tampered derived compilable projection")
	}
	if _, err := finalizedSession.Visit(choice.SurfaceOriginalWitness); !choice.IsRefusal(err, choice.CodeInvalidSessionState) {
		t.Fatalf("finalized decision session remained mutable: %v", err)
	}

	earlySession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	earlySession, _, err = earlySession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		earlySession, err = earlySession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	earlyInput := choice.RulingDraftInput{
		Action: choice.ActionRejectAll, SelectedFields: []string{}, AllowedAliases: []string{},
	}
	earlySession, err = earlySession.Revise(earlyInput, "")
	if err != nil {
		t.Fatal(err)
	}
	firstReceipt, err := domain.NewDidrunReceipt(
		confirmationDraft.Digest().String()+"\x00opaque-grade", "test-commit", choicepoint.Digest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondReceipt, err := domain.NewDidrunReceipt(
		"opaque-grade", "test-commit\x00"+choicepoint.Digest().String(), confirmationDraft.Digest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, rejectedDecision, err := earlySession.Finalize(
		"local-test-operator", "No positive oracle.", []domain.ReceiptReference{firstReceipt, secondReceipt},
	)
	if err != nil || !rejectedDecision.Valid() || !rejectedDecision.EarlyReveal() ||
		rejectedDecision.ChangedAfterReveal() || len(rejectedDecision.Receipts()) != 2 {
		t.Fatalf("early-reveal noncompilable DecisionRecord was invalid: %v", err)
	}
	if _, ok := rejectedDecision.CompilableRuling(); ok {
		t.Fatal("REJECT_ALL DecisionRecord was cast to a compilable ruling")
	}
	reparsedRejected, err := choice.ParseDecisionRecord(rejectedDecision.CanonicalBytes(), reopenedReady.Record())
	if err != nil || len(reparsedRejected.Receipts()) != 2 ||
		reparsedRejected.Receipts()[0].GradeVerbatim() == reparsedRejected.Receipts()[1].GradeVerbatim() {
		t.Fatalf("opaque DecisionRecord receipts did not remain distinct and verbatim: %v", err)
	}

	refineSession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	refineSession, _, err = refineSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		refineSession, err = refineSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	refineSession, err = refineSession.Revise(choice.RulingDraftInput{
		Action: choice.ActionRefine, SelectedFields: []string{}, AllowedAliases: []string{},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, refineDecision, err := refineSession.Finalize(
		"local-test-operator", "A successor study is required.", []domain.ReceiptReference{},
	)
	if err != nil || refineDecision.Action() != choice.ActionRefine {
		t.Fatalf("semantic REFINE DecisionRecord was invalid: %v", err)
	}
	if _, ok := refineDecision.CompilableRuling(); ok {
		t.Fatal("REFINE DecisionRecord was cast to a compilable ruling")
	}
	_, err = promotion.Finalize(context.Background(), restartedStore, reopenedReady, refineDecision)
	var promotionErr *promotion.Error
	if !errors.As(err, &promotionErr) || promotionErr.Code != promotion.CodeRefineRequiresSuccessorStudy {
		t.Fatalf("durable REFINE did not fail with %s: %v", promotion.CodeRefineRequiresSuccessorStudy, err)
	}
	if _, _, err := restartedStore.Read(context.Background(), "DecisionRecord", refineDecision.Digest()); err == nil {
		t.Fatal("refused REFINE DecisionRecord was published without a successor study")
	}
	stillReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || stillReady.Record().Digest() != reopenedReady.Record().Digest() {
		t.Fatalf("refused REFINE mutated the current CHOICEPOINT_READY head: %v", err)
	}

	finalizedRuling, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, decision)
	if err != nil || finalizedRuling.Record().Digest() != decision.Digest() {
		t.Fatalf("current Choicepoint did not promote its exact DecisionRecord: %v", err)
	}
	rulingRestart, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedRuling, err := promotion.OpenRuling(context.Background(), rulingRestart, studyID)
	if err != nil || reopenedRuling.Record().Digest() != decision.Digest() {
		t.Fatalf("durable RULING did not survive strict restart reconstruction: %v", err)
	}
	if _, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, rejectedDecision); err == nil {
		t.Fatal("stale CHOICEPOINT_READY authority published a second DecisionRecord")
	}
	if _, _, err := restartedStore.Read(context.Background(), "DecisionRecord", rejectedDecision.Digest()); err == nil {
		t.Fatal("losing stale DecisionRecord was published before the head compare-and-swap")
	}
}

func TestCLIPhysicalReducerBudgetFenceRetainsOnlyBestKnown(t *testing.T) {
	appMode, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	noise, err := countercli.PresentEnvironment("Z_NOISE", "ignored")
	if err != nil {
		t.Fatal(err)
	}
	configFixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	noisyStimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{clifixture.Entrypoint}, Argv: []string{"--mode", "argv"},
		Stdin: countercli.AbsentStdin(), Environment: []countercli.CLIEnvironmentBinding{appMode, noise},
		Fixtures: []countercli.CLIFixtureFile{configFixture}, CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := cliPhysicalBestKnownConfig(t)
	baselineConfig.StimulusOverride = &noisyStimulus
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatal("physical CLI budget-fence baseline did not produce a discovery outcome map")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baselineResult.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	planDigest, planBound := budget.WorldPlanDigest()
	if !planBound || planDigest != baselineResult.Plan.Digest() || budget.ProposalLimit() != 5 ||
		budget.CandidateTrialLimit() != 27 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled CLI budget fence lost plan authority: bound=%t plan=%s budget=%d/%d/%s",
			planBound, planDigest, budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	physicalEvaluations := 0
	var evaluatorErr error
	run, err := reducer.Run(context.Background(), reducer.RunInput[countercli.CLIStimulus]{
		Original: baselineResult.Stimulus,
		Reference: func(stimulus countercli.CLIStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := countercli.MeasureCLIStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus countercli.CLIStimulus) ([]reducer.TypedProposal[countercli.CLIStimulus], error) {
			neighbors, enumerateErr := countercli.EnumerateCLINeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[countercli.CLIStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[countercli.CLIStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus countercli.CLIStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := cliPhysicalBestKnownConfig(t)
			requiredTrials := uint64(config.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			physicalEvaluations++
			config.Purpose = purpose
			config.StimulusOverride = &stimulus
			evaluationContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := Run(evaluationContext, config)
			if runErr != nil {
				evaluatorErr = runErr
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				evaluatorErr = context.Canceled
				return reducer.EvaluationObservation{}, evaluatorErr
			}
			outcome := result.OutcomeMap
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluatorErr != nil {
		t.Fatalf("physical CLI budget-fence evaluator failed: %v", evaluatorErr)
	}
	transcript := run.Transcript()
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		transcript.FinalSweepState() != reducer.FinalSweepIncomplete || len(transcript.Entries()) != 3 ||
		len(transcript.AcceptedPath()) != 1 || run.MinimizedStimulusDigest() == baselineResult.Stimulus.Digest() ||
		physicalEvaluations != 3 || !slices.Contains(transcript.Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") {
		t.Fatalf("physical CLI budget fence was promoted or lost its accepted reduction: grade=%s accepted=%t sweep=%s entries=%d evaluations=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), transcript.FinalSweepState(), len(transcript.Entries()),
			physicalEvaluations, transcript.Limitations())
	}
	entries := transcript.Entries()
	wantDecisions := []reducer.Decision{reducer.Changes, reducer.Preserves, reducer.Changes}
	for index, entry := range entries {
		evaluation := entry.Evaluation()
		if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != wantDecisions[index] ||
			entry.ProposalCount() != uint64(index+1) || entry.CandidateTrials() != 9 ||
			entry.TrialCount() != uint64((index+1)*9) || !evaluation.LogicalNonReuseWithBaseline() {
			t.Fatalf("physical CLI budget-fence entry %d = purpose=%s decision=%s proposals=%d trials=%d/%d nonreuse=%t",
				index, evaluation.Purpose(), evaluation.Decision(), entry.ProposalCount(), entry.CandidateTrials(),
				entry.TrialCount(), evaluation.LogicalNonReuseWithBaseline())
		}
	}
	if draft, present, draftErr := run.CompletedSweepDraft(); draftErr != nil || present || draft.Valid() {
		t.Fatalf("incomplete physical CLI sweep exposed a completion draft: present=%t valid=%t err=%v",
			present, draft.Valid(), draftErr)
	}
	finalized, err := grade.Finalize(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	finalGrade := finalized.Grade()
	if !finalized.Valid() || !finalGrade.Valid() || finalGrade.Status() != grade.StatusBestKnown ||
		!slices.Contains(finalGrade.Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") ||
		!slices.Contains(finalGrade.Limitations(), "DURABLE_SWEEP_AUTHORITY_ABSENT") {
		t.Fatalf("physical CLI budget fence did not retain an honest BEST_KNOWN grade: status=%s limits=%v",
			finalGrade.Status(), finalGrade.Limitations())
	}
	if _, quantified := finalGrade.QuantifiedReducerSetDigest(); quantified {
		t.Fatal("BEST_KNOWN physical CLI budget fence exposed a local-minimality quantifier")
	}
	if _, completed := finalGrade.CompletedSweepDigest(); completed {
		t.Fatal("BEST_KNOWN physical CLI budget fence exposed a completed-sweep digest")
	}
}

func cliPhysicalReductionConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.ReductionProposalLimit = 5
	// The plan reserves 18 trials for discovery+confirmation and 36 for U5.
	config.ReductionTotalCandidateTrials = 54
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}

func cliPhysicalBestKnownConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.ReductionProposalLimit = 5
	// The plan reserves 18 trials for discovery+confirmation and only 27 for U5:
	// exactly three physical evaluations, with no authority left for the final sweep.
	config.ReductionTotalCandidateTrials = 45
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}
