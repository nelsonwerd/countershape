//go:build darwin && cgo

package cli_precedence

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/compare"
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
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatal("physical CLI baseline did not produce a discovery outcome map")
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
