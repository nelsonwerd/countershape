//go:build darwin && cgo

package http_invoices

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
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
