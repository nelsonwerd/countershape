//go:build darwin && cgo

package http_invoices

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

func TestHTTPPhysicalTenantSeedNeighborChangesExactLabeledMapWithStableRoster(t *testing.T) {
	referenceStimulus, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	tenantlessStimulus, err := newInvoiceStimulusWithSeed(httpfixture.TenantlessSeedJSON())
	if err != nil {
		t.Fatal(err)
	}
	referenceWire, err := counterhttp.EncodeRequest(referenceStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	tenantlessWire, err := counterhttp.EncodeRequest(tenantlessStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(referenceWire.Bytes(), tenantlessWire.Bytes()) {
		t.Fatal("tenant-seed neighbor changed request bytes instead of only fixture authority")
	}

	baselineConfig := referenceConfig(t)
	baselineConfig.Repetitions = 1
	baselineConfig.MaxTotalTrials = 4
	baselineConfig.StimulusOverride = &referenceStimulus
	reference, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	tenantlessConfig := referenceConfig(t)
	tenantlessConfig.Repetitions = 1
	tenantlessConfig.MaxTotalTrials = 4
	tenantlessConfig.Purpose = domain.AttemptReduction
	tenantlessConfig.StimulusOverride = &tenantlessStimulus
	tenantless, err := Run(context.Background(), tenantlessConfig)
	if err != nil {
		t.Fatal(err)
	}
	for name, result := range map[string]StudyResult{"reference": reference, "tenantless": tenantless} {
		if !result.HasOutcomeMap || len(result.OutcomeMap.Entries()) != 4 ||
			len(result.OutcomeMap.Exclusions()) != 0 || !result.OutcomeMap.Divergence() {
			t.Fatalf("%s stable-roster map entries=%d exclusions=%d divergence=%t",
				name, len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()), result.OutcomeMap.Divergence())
		}
	}
	assessment := compare.AssessPreservation(reference.OutcomeMap, tenantless.OutcomeMap)
	if !assessment.Valid() || assessment.Relation() != compare.PreservationDifferent ||
		assessment.BaselinePreservationDigest().String() == assessment.ObservedPreservationDigest().String() {
		t.Fatalf("tenant seed shape trap was not exact labeled-map CHANGES: relation=%s reason=%s",
			assessment.Relation(), assessment.ReasonCode())
	}
}

func TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence(t *testing.T) {
	base, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	noise, err := counterhttp.NewSeedFile("z-noise.txt", []byte("not-consumed\n"), counterhttp.SeedMode0644)
	if err != nil {
		t.Fatal(err)
	}
	noisySeeds := append(base.Seeds(), noise)
	noisy, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: base.Method(), Path: base.Path(), Query: base.Query(), Headers: base.Headers(),
		Body: base.Body(), Seeds: noisySeeds,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := httpPhysicalReductionConfig(t)
	baselineConfig.StimulusOverride = &noisy
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery ||
		len(baselineResult.OutcomeMap.Entries()) != 3 || len(baselineResult.OutcomeMap.Exclusions()) != 1 {
		t.Fatal("physical HTTP baseline did not retain the stable A/B/C map plus excluded D")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := counterhttp.NewHTTPReductionPolicy(counterhttp.HTTPReductionPolicyConfig{
		Anchor: noisy, PinnedSeedPaths: []string{httpfixture.SeedFilename},
		EnabledRules: []counterhttp.HTTPReducerID{counterhttp.HTTPSeedRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if budget.ProposalLimit() != 2 || budget.CandidateTrialLimit() != 12 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled HTTP reduction budget = %d/%d/%s", budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	var evaluatorErr error
	var minimizedStudy StudyResult
	run, err := reducer.Run(context.Background(), reducer.RunInput[counterhttp.HTTPStimulus]{
		Original: noisy,
		Reference: func(stimulus counterhttp.HTTPStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := counterhttp.MeasureHTTPStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus counterhttp.HTTPStimulus) ([]reducer.TypedProposal[counterhttp.HTTPStimulus], error) {
			neighbors, enumerateErr := counterhttp.EnumerateHTTPNeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[counterhttp.HTTPStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[counterhttp.HTTPStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus counterhttp.HTTPStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := httpPhysicalReductionConfig(t)
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
		t.Fatalf("physical HTTP reducer evaluator failed: %v", evaluatorErr)
	}
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepComplete || len(run.Transcript().Entries()) != 1 ||
		len(run.Transcript().AcceptedPath()) != 1 {
		t.Fatalf("unexpected physical HTTP reduction: grade=%s accepted=%t sweep=%s entries=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), run.Transcript().FinalSweepState(),
			len(run.Transcript().Entries()), run.Transcript().Limitations())
	}
	evaluation := run.Transcript().Entries()[0].Evaluation()
	if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != reducer.Preserves ||
		!evaluation.LogicalNonReuseWithBaseline() || len(evaluation.ObservedAttemptDigests()) != 12 {
		t.Fatalf("physical HTTP preserving evidence is incomplete: purpose=%s decision=%s attempts=%d",
			evaluation.Purpose(), evaluation.Decision(), len(evaluation.ObservedAttemptDigests()))
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 ||
		draft.CurrentStimulusDigest() != run.MinimizedStimulusDigest() {
		t.Fatalf("physical HTTP empty-neighbor completed sweep was not retained: present=%t err=%v", present, err)
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
		t.Fatalf("physical HTTP durable grade was not quantified: status=%s err=%v",
			finalized.Grade().Status(), err)
	}
	if !minimizedStudy.HasOutcomeMap || minimizedStudy.Stimulus.Digest() != run.MinimizedStimulusDigest() {
		t.Fatal("physical HTTP reducer did not retain the exact minimized preserving study")
	}
	reducedBaseline, err := compare.RequireDivergence(minimizedStudy.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	confirmationConfig := httpPhysicalReductionConfig(t)
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
		confirmedStudy.OutcomeMap.ScheduleStartOffset() != 1 || len(confirmedStudy.OutcomeMap.Exclusions()) != 1 ||
		confirmedStudy.OutcomeMap.Exclusions()[0].Classification != observe.Unstable ||
		len(confirmedStudy.Confirmation.Draft().PhysicalFacts()) != 12 {
		t.Fatal("physical HTTP confirmation did not reproduce the complete A/B/C plus unstable-D disposition")
	}
	confirmationDraft := confirmedStudy.Confirmation.Draft()
	if parsed, parseErr := confirmation.ParseRecord(confirmationDraft.CanonicalBytes()); parseErr != nil || parsed.Digest() != confirmationDraft.Digest() {
		t.Fatalf("physical HTTP confirmation did not round trip strictly: %v", parseErr)
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
		"HTTPStimulus", baselineResult.Stimulus.Digest(), baselineResult.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact(
		"HTTPStimulus", confirmedStudy.Stimulus.Digest(), confirmedStudy.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reveals := make([]choice.CandidateReveal, len(confirmedStudy.CandidateBindings))
	for index, binding := range confirmedStudy.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(confirmedStudy.CandidateRoles[binding.Key()]),
			ProducerMetadata: "local deterministic HTTP fixture",
		}
	}
	choicepoint, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: "Which exact invoice response behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, Confirmation: confirmationDraft.Record(), CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || !choicepoint.Valid() {
		t.Fatalf("physical HTTP Choicepoint construction failed: %v", err)
	}
	parsedChoicepoint, err := choice.ParseChoicepointRecord(choicepoint.CanonicalBytes())
	if err != nil || parsedChoicepoint.Digest() != choicepoint.Digest() {
		t.Fatalf("physical HTTP Choicepoint did not round trip strictly: %v", err)
	}
	blind, err := choice.NewBlindView(parsedChoicepoint)
	if err != nil || len(blind.DTO().Cards()) != confirmedStudy.OutcomeMap.DistinctProjectionCount() {
		t.Fatalf("physical HTTP blind DTO did not group exact eligible outcomes: %v", err)
	}
	excluded := confirmedStudy.OutcomeMap.Exclusions()[0]
	var excludedBinding domain.CandidateExecutionBinding
	var excludedReveal choice.CandidateReveal
	for index, binding := range confirmedStudy.CandidateBindings {
		if binding.Key() == excluded.CandidateKey {
			excludedBinding = binding
			excludedReveal = reveals[index]
			break
		}
	}
	if !excludedBinding.Valid() {
		t.Fatal("physical HTTP excluded candidate lacks its exact binding")
	}
	blindBytes := blind.DTO().CanonicalBytes()
	for _, forbidden := range []string{
		excluded.CandidateKey.String(), excludedBinding.Identity().TreeIdentityDigest.String(),
		excludedReveal.DisplayRef, excludedReveal.ProducerMetadata,
	} {
		if forbidden != "" && bytes.Contains(blindBytes, []byte(forbidden)) {
			t.Fatalf("physical HTTP blind DTO leaked excluded-candidate provenance %q", forbidden)
		}
	}
	decisionSession, err := choice.NewSession(parsedChoicepoint)
	if err != nil {
		t.Fatal(err)
	}
	_, reveal, err := decisionSession.Reveal()
	if err != nil {
		t.Fatalf("physical HTTP early reveal failed: %v", err)
	}
	revealedExclusions := reveal.Exclusions()
	if len(revealedExclusions) != 1 ||
		revealedExclusions[0].Candidate.CandidateExecutionKey != excluded.CandidateKey.String() ||
		revealedExclusions[0].Candidate.TreeIdentityDigest != excludedBinding.Identity().TreeIdentityDigest.String() ||
		revealedExclusions[0].Candidate.DisplayRef != excludedReveal.DisplayRef ||
		revealedExclusions[0].Candidate.ProducerMetadata != excludedReveal.ProducerMetadata ||
		revealedExclusions[0].Classification != string(observe.Unstable) {
		t.Fatalf("physical HTTP reveal did not restore exactly one unstable exclusion with provenance: %#v", revealedExclusions)
	}
	choiceObject, err := store.NewSemanticObject("Choicepoint", choicepoint.Digest(), choicepoint.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	choiceAuthority, err := confirmationStore.Publish(context.Background(), choiceObject)
	if err != nil || confirmationStore.Validate(context.Background(), choiceObject, choiceAuthority) != nil {
		t.Fatalf("physical HTTP Choicepoint did not persist immutably: %v", err)
	}
}

func httpPhysicalReductionConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.ReductionProposalLimit = 2
	// The plan reserves 24 trials for discovery+confirmation and 12 for U5.
	config.ReductionTotalCandidateTrials = 36
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}
