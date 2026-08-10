package httpstudy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/contractmaterialize"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

const tenantlessHTTPSeed = `{"invoice_id":"inv-204","metadata":{"amount_cents":4200,"currency":"USD"}}`

type completedHTTPStudySeal struct{ marker byte }

var issuedCompletedHTTPStudy = &completedHTTPStudySeal{marker: 1}

type standaloneTrial struct {
	attemptDigest          domain.Digest
	targetDigest           domain.Digest
	targetCanonicalSHA256  string
	bundleDigest           domain.Digest
	residueHeadDigest      domain.Digest
	processDigest          domain.Digest
	processCanonicalSHA256 string
	invocationSHA256       string
	testFileSHA256         string
	stdoutSHA256           string
	stderrSHA256           string
	stdoutBytes            int
	stderrBytes            int
	exitCode               int
	observedDiagnostic     string
	status                 string
	candidateRoot          string
}

type standaloneTargetSnapshot struct {
	attemptDigest  domain.Digest
	targetDigest   domain.Digest
	bundleDigest   domain.Digest
	canonicalBytes []byte
	candidateRoot  string
	roots          [12]string
}

type standaloneMaterializationSnapshot struct {
	destination  string
	bundleDigest domain.Digest
	headDigest   domain.Digest
}

// CompletedHTTPStudy is the only value that authorizes U7 HTTP evidence
// publication. Its seal is issued after every physical, reduction, choice,
// compilation, fresh-target, and direct bundle-test gate has completed.
type CompletedHTTPStudy struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	discovery            Result
	recoveryOne          Result
	recoveryTwo          Result
	shapeReference       Result
	shapeChanged         Result
	baselineCheckpoint   Result
	baseline             Result
	minimized            Result
	confirmed            Result
	shapeAssessment      compare.PreservationAssessment
	reductionRun         reducer.ReductionRun
	weakReduction        grade.Result
	strongReduction      grade.Result
	choicepoint          choice.ChoicepointRecord
	decision             choice.DecisionRecord
	durableRuling        promotion.Ruling
	portableSource       contractsource.PortableSource
	bundle               emitmodel.ContractBundle
	residue              nodeemit.Residue
	standaloneTrials     []standaloneTrial
	seal                 *completedHTTPStudySeal
}

// Valid reports construction integrity and exact joins. It does not claim
// currentness beyond the retained physical run.
func (completed CompletedHTTPStudy) Valid() bool {
	return validateCompletedHTTPStudy(completed) == nil
}

// CompleteHTTPStudy executes the entire admitted HTTP workflow. No evidence
// payload can be constructed from a discovery-only Result.
func CompleteHTTPStudy(ctx context.Context, config Config) (CompletedHTTPStudy, error) {
	if ctx == nil {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_CONFIG_REFUSED")
	}
	discovery, err := runPhase(ctx, config, phaseRunSpec{
		label: "decisive-discovery", purpose: domain.AttemptDiscovery, repetitions: 3,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if err := validateDiscovery(discovery); err != nil {
		return CompletedHTTPStudy{}, err
	}
	recoveryOne, err := runPhase(ctx, config, phaseRunSpec{
		label: "fresh-recovery-one", purpose: domain.AttemptDiscovery, repetitions: 3,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	recoveryTwo, err := runPhase(ctx, config, phaseRunSpec{
		label: "fresh-recovery-two", purpose: domain.AttemptDiscovery, repetitions: 3,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if validateDiscovery(recoveryOne) != nil || validateDiscovery(recoveryTwo) != nil ||
		compare.AssessPreservation(discovery.OutcomeMap, recoveryOne.OutcomeMap).Relation() != compare.PreservationEqual ||
		compare.AssessPreservation(discovery.OutcomeMap, recoveryTwo.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(discovery.OutcomeMap.CanonicalBytes(), recoveryOne.OutcomeMap.CanonicalBytes()) ||
		bytes.Equal(recoveryOne.OutcomeMap.CanonicalBytes(), recoveryTwo.OutcomeMap.CanonicalBytes()) ||
		validateFreshDisjoint(discovery, recoveryOne, recoveryTwo) != nil {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_FRESH_RECOVERY_REFUSED")
	}

	shapeReference, err := runPhase(ctx, config, phaseRunSpec{
		label: "shape-reference", purpose: domain.AttemptDiscovery, repetitions: 1,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	tenantless, err := invoiceStimulus([]byte(tenantlessHTTPSeed))
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	shapeChanged, err := runPhase(ctx, config, phaseRunSpec{
		label: "shape-tenantless", purpose: domain.AttemptReduction, repetitions: 1, stimulus: &tenantless,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	shapeAssessment, err := validateShapeTrap(shapeReference, shapeChanged)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}

	noisy, err := noisyHTTPStimulus(discovery.Stimulus)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	baselineCheckpoint, err := runPhase(ctx, config, phaseRunSpec{
		label: "baseline-checkpoint", purpose: domain.AttemptDiscovery, repetitions: 3, stimulus: &noisy,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	baselineStudy, err := runPhase(ctx, config, phaseRunSpec{
		label: "baseline-divergence", purpose: domain.AttemptDiscovery, repetitions: 3, stimulus: &noisy,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if !baselineCheckpoint.HasOutcomeMap || !baselineStudy.HasOutcomeMap ||
		baselineCheckpoint.Plan.Digest() != baselineStudy.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineStudy.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(baselineCheckpoint.OutcomeMap.CanonicalBytes(), baselineStudy.OutcomeMap.CanonicalBytes()) {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_FRESH_BASELINE_REFUSED")
	}
	baseline, err := compare.RequireDivergence(baselineStudy.OutcomeMap)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	policy, err := counterhttp.NewHTTPReductionPolicy(counterhttp.HTTPReductionPolicyConfig{
		Anchor: noisy, PinnedSeedPaths: []string{reference.HTTPSeedFilename},
		EnabledRules: []counterhttp.HTTPReducerID{counterhttp.HTTPSeedRemove},
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineStudy.Plan)
	if err != nil || budget.ProposalLimit() != 2 || budget.CandidateTrialLimit() != 16 || budget.WallLimit() != 3*time.Minute {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_REDUCTION_BUDGET_REFUSED: %v", err)
	}

	evaluationIndex := 0
	var minimized Result
	reductionRun, err := reducer.Run(ctx, reducer.RunInput[counterhttp.HTTPStimulus]{
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
		Evaluate: func(runContext context.Context, stimulus counterhttp.HTTPStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			evaluationIndex++
			if allowance.RemainingCandidateTrials < 12 || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			evaluationContext, cancel := context.WithDeadline(runContext, allowance.WallDeadline)
			defer cancel()
			result, runErr := runPhase(evaluationContext, config, phaseRunSpec{
				label: fmt.Sprintf("reduction-eval-%02d", evaluationIndex), purpose: purpose,
				repetitions: 3, planDiscoveryRepeats: 3, planConfirmationRepeats: 2, stimulus: &stimulus,
			})
			if runErr != nil {
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap || len(result.Trials) != 12 {
				return reducer.EvaluationObservation{}, fmt.Errorf("HTTP_STUDY_REDUCTION_MAP_REFUSED")
			}
			outcome := result.OutcomeMap
			if compare.AssessPreservation(baseline.OutcomeMap(), outcome).Relation() == compare.PreservationEqual {
				minimized = result
			}
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if !reductionRun.Valid() || reductionRun.DraftGrade() != reducer.GradeBestKnown ||
		!reductionRun.HasAcceptedReduction() || reductionRun.Transcript().FinalSweepState() != reducer.FinalSweepComplete ||
		len(reductionRun.Transcript().Entries()) != 1 || len(reductionRun.Transcript().AcceptedPath()) != 1 ||
		!minimized.HasOutcomeMap || minimized.Stimulus.Digest() != reductionRun.MinimizedStimulusDigest() {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_REDUCTION_REFUSED")
	}
	evaluation := reductionRun.Transcript().Entries()[0].Evaluation()
	if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != reducer.Preserves ||
		!evaluation.LogicalNonReuseWithBaseline() || len(evaluation.ObservedAttemptDigests()) != 12 {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_REDUCTION_EVIDENCE_REFUSED")
	}
	weakReduction, err := grade.Finalize(ctx, reductionRun, nil)
	if err != nil || !weakReduction.Valid() || weakReduction.Grade().Status() != grade.StatusBestKnown {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_WEAK_GRADE_REFUSED: %v", err)
	}
	draft, present, err := reductionRun.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_SWEEP_REFUSED: %v", err)
	}
	authorityRoot, err := privateDirectory(config.ScratchRoot, fmt.Sprintf("http-authorities-%d", config.Ordinal))
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	sweepStore, err := store.OpenReductionSweepStore(authorityRoot + "/sweep-store")
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	sweepAuthority, err := sweepStore.Publish(ctx, draft)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	strongReduction, err := grade.Finalize(ctx, reductionRun, &grade.SweepCompletion{
		Store: sweepStore, Draft: draft, Authority: sweepAuthority,
	})
	if err != nil || !strongReduction.Valid() || strongReduction.Grade().Status() != grade.StatusOneMinimalUnder {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_STRONG_GRADE_REFUSED: %v", err)
	}

	reducedBaseline, err := compare.RequireDivergence(minimized.OutcomeMap)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	confirmed, err := runPhase(ctx, config, phaseRunSpec{
		label: "rotated-confirmation", purpose: domain.AttemptConfirmation, repetitions: 2,
		planDiscoveryRepeats: 3, planConfirmationRepeats: 2,
		stimulus: &minimized.Stimulus,
		confirmation: &confirmationRunInput{
			reducedBaseline: reducedBaseline, reductionRun: reductionRun, reductionResult: strongReduction,
		},
	})
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if err := validateConfirmation(minimized, confirmed); err != nil {
		return CompletedHTTPStudy{}, err
	}

	choicepoint, decision, durableRuling, source, bundle, residue, objectStore, err :=
		completeChoiceAndBundle(ctx, authorityRoot, baselineCheckpoint, baseline, baselineStudy, reductionRun, confirmed)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	standaloneTrials, err := executeStandaloneTrials(
		ctx, config, authorityRoot, objectStore, residue, bundle,
	)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	if err := ctx.Err(); err != nil {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_CONTEXT_ENDED_BEFORE_COMPLETION: %w", err)
	}

	completed := CompletedHTTPStudy{
		ordinal:   config.Ordinal,
		discovery: discovery, recoveryOne: recoveryOne, recoveryTwo: recoveryTwo,
		shapeReference: shapeReference, shapeChanged: shapeChanged,
		baselineCheckpoint: baselineCheckpoint, baseline: baselineStudy, minimized: minimized, confirmed: confirmed,
		shapeAssessment: shapeAssessment, reductionRun: reductionRun,
		weakReduction: weakReduction, strongReduction: strongReduction,
		choicepoint: choicepoint, decision: decision, durableRuling: durableRuling,
		portableSource: source, bundle: bundle, residue: residue,
		standaloneTrials: standaloneTrials,
	}
	completed.physicalRunAuthority, err = physicalRunAuthority(completed)
	if err != nil {
		return CompletedHTTPStudy{}, err
	}
	completed.seal = issuedCompletedHTTPStudy
	if err := validateCompletedHTTPStudy(completed); err != nil {
		return CompletedHTTPStudy{}, err
	}
	if err := ctx.Err(); err != nil {
		return CompletedHTTPStudy{}, fmt.Errorf("HTTP_STUDY_CONTEXT_ENDED_AT_COMPLETION: %w", err)
	}
	return completed, nil
}

// validateDiscovery makes the decisive physical discovery facts load-bearing
// before any later phase may execute.
func validateDiscovery(result Result) error {
	if !result.HasOutcomeMap || result.Observation.Status() != observe.ObservationComplete ||
		result.Observation.CompletedMatrices() != 3 || len(result.Trials) != 12 ||
		len(result.OutcomeMap.Entries()) != 3 || len(result.OutcomeMap.Exclusions()) != 1 ||
		result.OutcomeMap.DistinctProjectionCount() != 3 || !result.OutcomeMap.Divergence() ||
		result.OutcomeMap.Phase() != domain.AttemptDiscovery {
		return fmt.Errorf("HTTP_STUDY_DISCOVERY_SHAPE_REFUSED")
	}

	want := map[reference.CandidateRole]struct {
		status   int64
		kind     string
		metadata string
	}{
		reference.Forbidden:          {403, "forbidden", `{}`},
		reference.ConcealNotFound:    {404, "not_found", `{}`},
		reference.MetadataDisclosure: {200, "authorized_metadata", `{"amount_cents":4200,"currency":"USD","owner_tenant":"tenant-b"}`},
	}
	counts := make(map[reference.CandidateRole]int, 4)
	stableProjection := make(map[reference.CandidateRole][]byte, 3)
	seenWorlds := make(map[domain.Digest]struct{}, 12)
	seenAttempts := make(map[domain.Digest]struct{}, 12)
	seenObservations := make(map[domain.Digest]struct{}, 12)
	alternating := make(map[string]int, 2)
	for _, trial := range result.Trials {
		if !trial.Admitted || !trial.Projected || trial.ProjectionRejection != nil ||
			!trial.Result.World().Digest().Valid() || !trial.Result.FinalizedAttempt().ArtifactDigest().Valid() ||
			!trial.Observation.Digest().Valid() || !trial.Projection.Digest().Valid() {
			return fmt.Errorf("HTTP_STUDY_TRIAL_ADMISSION_REFUSED")
		}
		for digest, ledger := range map[domain.Digest]map[domain.Digest]struct{}{
			trial.Result.World().Digest():                    seenWorlds,
			trial.Result.FinalizedAttempt().ArtifactDigest(): seenAttempts,
			trial.Observation.Digest():                       seenObservations,
		} {
			if _, duplicate := ledger[digest]; duplicate {
				return fmt.Errorf("HTTP_STUDY_PHYSICAL_REUSE_REFUSED")
			}
			ledger[digest] = struct{}{}
		}
		counts[trial.Role]++
		if trial.Role == reference.Alternating {
			alternating[string(trial.Projection.ProjectionBytes())]++
			continue
		}
		expected, present := want[trial.Role]
		if !present || !projectionMatches(trial.Projection, expected.status, expected.kind, expected.metadata) {
			return fmt.Errorf("HTTP_STUDY_EXACT_PROJECTION_REFUSED")
		}
		projection := trial.Projection.ProjectionBytes()
		if prior, present := stableProjection[trial.Role]; present && !bytes.Equal(prior, projection) {
			return fmt.Errorf("HTTP_STUDY_STABLE_PROJECTION_REFUSED")
		}
		stableProjection[trial.Role] = append([]byte(nil), projection...)
	}
	for _, role := range reference.HTTPRoles() {
		if counts[role] != 3 {
			return fmt.Errorf("HTTP_STUDY_REPEAT_COUNT_REFUSED")
		}
	}
	if len(alternating) != 2 ||
		alternating[string(stableProjection[reference.ConcealNotFound])] != 2 ||
		alternating[string(stableProjection[reference.MetadataDisclosure])] != 1 {
		return fmt.Errorf("HTTP_STUDY_FALSIFICATION_SPLIT_REFUSED")
	}
	for _, batch := range result.Observation.Batches() {
		role, present := result.CandidateRoles[batch.CandidateKey()]
		if !present || batch.RequiredFreshTrials() != 3 || batch.Classification().EligibleTrials() != 3 ||
			batch.Classification().RequiredTrials() != 3 {
			return fmt.Errorf("HTTP_STUDY_CLASSIFICATION_COUNT_REFUSED")
		}
		if role == reference.Alternating {
			if batch.Classification().Status() != observe.Unstable || len(batch.Classification().Histogram()) != 2 {
				return fmt.Errorf("HTTP_STUDY_UNSTABLE_EXCLUSION_REFUSED")
			}
		} else if batch.Classification().Status() != observe.ObservedStable || batch.Classification().BoundedLabel() == "" {
			return fmt.Errorf("HTTP_STUDY_STABLE_CLASSIFICATION_REFUSED")
		}
	}
	return nil
}

func projectionMatches(projection counterhttp.HTTPProjectionResult, status int64, kind, metadata string) bool {
	fields := projection.Fields()
	if len(fields) != 4 || fields[0].ID() != counterhttp.HTTPFieldStatus ||
		fields[1].ID() != counterhttp.HTTPFieldContentType || fields[2].ID() != counterhttp.HTTPFieldBodyKind ||
		fields[3].ID() != counterhttp.HTTPFieldBodyMetadata {
		return false
	}
	observedStatus, statusOK := fields[0].Integer()
	contentTypes, contentOK := fields[1].Strings()
	observedKind, kindOK := fields[2].String()
	observedMetadata, metadataOK := fields[3].CanonicalJSON()
	return statusOK && observedStatus == status && contentOK &&
		slices.Equal(contentTypes, []string{"application/json"}) && kindOK && observedKind == kind &&
		metadataOK && observedMetadata == metadata
}

func validateShapeTrap(referenceResult, changed Result) (compare.PreservationAssessment, error) {
	if !referenceResult.HasOutcomeMap || !changed.HasOutcomeMap ||
		len(referenceResult.Trials) != 4 || len(changed.Trials) != 4 ||
		len(referenceResult.OutcomeMap.Entries()) != 4 || len(changed.OutcomeMap.Entries()) != 4 ||
		len(referenceResult.OutcomeMap.Exclusions()) != 0 || len(changed.OutcomeMap.Exclusions()) != 0 ||
		!referenceResult.OutcomeMap.Divergence() || !changed.OutcomeMap.Divergence() {
		return compare.PreservationAssessment{}, fmt.Errorf("HTTP_STUDY_SHAPE_TRAP_MAP_REFUSED")
	}
	referenceWire, err := counterhttp.EncodeRequest(referenceResult.Stimulus, 43210)
	if err != nil {
		return compare.PreservationAssessment{}, err
	}
	changedWire, err := counterhttp.EncodeRequest(changed.Stimulus, 43210)
	if err != nil {
		return compare.PreservationAssessment{}, err
	}
	if !bytes.Equal(referenceWire.Bytes(), changedWire.Bytes()) {
		return compare.PreservationAssessment{}, fmt.Errorf("HTTP_STUDY_SHAPE_TRAP_WIRE_REFUSED")
	}
	assessment := compare.AssessPreservation(referenceResult.OutcomeMap, changed.OutcomeMap)
	if !assessment.Valid() || assessment.Relation() != compare.PreservationDifferent ||
		assessment.ReasonCode() != "EXACT_PRESERVATION_MAP_CHANGED" ||
		assessment.BaselinePreservationDigest().String() == assessment.ObservedPreservationDigest().String() {
		return compare.PreservationAssessment{}, fmt.Errorf("HTTP_STUDY_SHAPE_TRAP_CHANGES_REFUSED")
	}
	return assessment, nil
}

func noisyHTTPStimulus(base counterhttp.HTTPStimulus) (counterhttp.HTTPStimulus, error) {
	noise, err := counterhttp.NewSeedFile("z-noise.txt", []byte("not-consumed\n"), counterhttp.SeedMode0644)
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	return counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: base.Method(), Path: base.Path(), Query: base.Query(), Headers: base.Headers(),
		Body: base.Body(), Seeds: append(base.Seeds(), noise),
	})
}

func validateConfirmation(minimized, confirmed Result) error {
	if !confirmed.HasConfirmation || !confirmed.Confirmation.Valid() || !confirmed.HasOutcomeMap ||
		confirmed.OutcomeMap.Phase() != domain.AttemptConfirmation || confirmed.OutcomeMap.ScheduleStartOffset() != 1 ||
		confirmed.OutcomeMap.ScheduleDigest() == minimized.OutcomeMap.ScheduleDigest() ||
		len(confirmed.OutcomeMap.Entries()) != 3 || len(confirmed.OutcomeMap.Exclusions()) != 1 ||
		confirmed.OutcomeMap.Exclusions()[0].Classification != observe.Unstable ||
		confirmed.Confirmation.Assessment().Relation() != compare.PreservationEqual ||
		len(confirmed.Trials) != 8 || len(confirmed.Confirmation.Draft().PhysicalFacts()) != 8 {
		return fmt.Errorf("HTTP_STUDY_CONFIRMATION_REFUSED")
	}
	parsed, err := confirmation.ParseRecord(confirmed.Confirmation.Draft().CanonicalBytes())
	if err != nil || parsed.Digest() != confirmed.Confirmation.Draft().Digest() ||
		len(parsed.ExecutionBindingDigests()) != 8 {
		return fmt.Errorf("HTTP_STUDY_CONFIRMATION_PARSE_REFUSED: %v", err)
	}
	return validateFreshDisjoint(minimized, confirmed)
}

func validateFreshDisjoint(studies ...Result) error {
	ledgers := []map[domain.Digest]struct{}{{}, {}, {}}
	for _, study := range studies {
		if !study.HasOutcomeMap {
			return fmt.Errorf("HTTP_STUDY_FRESH_MAP_REFUSED")
		}
		for index, roster := range [][]domain.Digest{
			study.OutcomeMap.EvidenceWorldDigests(), study.OutcomeMap.EvidenceAttemptDigests(),
			study.OutcomeMap.EvidenceObservationDigests(),
		} {
			for _, digest := range roster {
				if !digest.Valid() {
					return fmt.Errorf("HTTP_STUDY_FRESH_DIGEST_REFUSED")
				}
				if _, duplicate := ledgers[index][digest]; duplicate {
					return fmt.Errorf("HTTP_STUDY_FRESH_REUSE_REFUSED")
				}
				ledgers[index][digest] = struct{}{}
			}
		}
	}
	return nil
}

func completeChoiceAndBundle(
	ctx context.Context,
	authorityRoot string,
	baselineCheckpoint Result,
	baseline compare.DivergentBaseline,
	original Result,
	reductionRun reducer.ReductionRun,
	confirmed Result,
) (choice.ChoicepointRecord, choice.DecisionRecord, promotion.Ruling, contractsource.PortableSource, emitmodel.ContractBundle, nodeemit.Residue, *store.ObjectStore, error) {
	objectStore, err := store.OpenObjectStore(authorityRoot + "/object-store")
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	studyID, err := store.NewStudyID("u7 physical http status-only contract")
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	head, err := objectStore.CreateStudy(ctx, studyID, confirmed.Plan)
	if err == nil {
		head, err = objectStore.AdvanceBaseline(ctx, head, baselineCheckpoint.OutcomeMap)
	}
	if err == nil {
		head, err = objectStore.AdvanceDivergence(ctx, head, baseline)
	}
	if err == nil {
		head, err = objectStore.AdvanceReduction(ctx, head, reductionRun)
	}
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	stored, err := promotion.PersistConfirmation(ctx, objectStore, head, confirmed.Confirmation.Draft())
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	originalArtifact, err := choice.NewCanonicalArtifact("HTTPStimulus", original.Stimulus.Digest(), original.Stimulus.CanonicalBytes())
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact("HTTPStimulus", confirmed.Stimulus.Digest(), confirmed.Stimulus.CanonicalBytes())
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	reveals := make([]choice.CandidateReveal, len(confirmed.CandidateBindings))
	for index, binding := range confirmed.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(confirmed.CandidateRoles[binding.Key()]),
			ProducerMetadata: "outer deterministic HTTP driver fixture",
		}
	}
	ready, err := promotion.Promote(ctx, objectStore, stored, promotion.ChoicepointRequest{
		Scenario: "Which exact invoice response status should become the accepted contract?",
		Plan:     confirmed.Plan, Envelope: confirmed.Envelope, CandidateBindings: confirmed.CandidateBindings,
		OriginalStimulus: originalArtifact, MinimizedStimulus: minimizedArtifact,
		CandidateReveals: reveals, EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	record := ready.Record()
	parsed, err := choice.ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || parsed.Digest() != record.Digest() {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_CHOICEPOINT_PARSE_REFUSED: %v", err)
	}
	blind, err := choice.NewBlindView(parsed)
	if err != nil || len(blind.DTO().Cards()) != 3 {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_BLIND_VIEW_REFUSED: %v", err)
	}
	alias404 := ""
	for _, card := range blind.DTO().Cards() {
		for _, field := range card.Fields {
			if field.FieldID == string(counterhttp.HTTPFieldStatus) && field.Tag == string(choice.ValueInteger) && field.Text == "404" {
				alias404 = card.Alias
			}
		}
	}
	if alias404 == "" {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_404_ALIAS_REFUSED")
	}
	visitBlindSurfaces := func(session choice.Session) (choice.Session, error) {
		var visitErr error
		for _, surface := range []choice.ReviewSurface{
			choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
			choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
		} {
			session, visitErr = session.Visit(surface)
			if visitErr != nil {
				return choice.Session{}, visitErr
			}
		}
		return session, nil
	}
	hostileSession, err := choice.NewSession(parsed)
	if err == nil {
		hostileSession, err = visitBlindSurfaces(hostileSession)
	}
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	ambiguous := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: []string{string(counterhttp.HTTPFieldContentType)},
		AllowedAliases: []string{alias404},
	}
	if _, ambiguousErr := hostileSession.Propose(ambiguous); !choice.IsRefusal(ambiguousErr, choice.CodeAmbiguousScope) {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_AMBIGUOUS_SCOPE_NOT_REFUSED: %v", ambiguousErr)
	}
	session, err := choice.NewSession(parsed)
	if err == nil {
		session, err = visitBlindSurfaces(session)
	}
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	statusOnly := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: []string{string(counterhttp.HTTPFieldStatus)},
		AllowedAliases: []string{alias404},
	}
	session, err = session.Propose(statusOnly)
	if err == nil {
		session, _, err = session.Reveal()
	}
	if err == nil {
		session, err = session.Visit(choice.SurfaceProvenance)
	}
	if err == nil {
		session, err = session.Revise(statusOnly, "")
	}
	var decision choice.DecisionRecord
	if err == nil {
		_, decision, err = session.Finalize(
			"u7-reference-operator", "Accept only the blind-selected observed HTTP 404 status.",
			[]domain.ReceiptReference{},
		)
	}
	if err != nil || decision.EarlyReveal() || decision.Action() != choice.ActionAllowObserved ||
		!slices.Equal(decision.SelectedFields(), []string{string(counterhttp.HTTPFieldStatus)}) ||
		!slices.Equal(decision.NonassertedFields(), []string{
			string(counterhttp.HTTPFieldContentType), string(counterhttp.HTTPFieldBodyKind), string(counterhttp.HTTPFieldBodyMetadata),
		}) {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_STATUS_RULING_REFUSED: %v", err)
	}
	compiled, present := decision.CompilableRuling()
	if !present || len(compiled.AllowedTuples()) != 1 || len(compiled.AllowedTuples()[0].Fields) != 1 ||
		compiled.AllowedTuples()[0].Fields[0].FieldID != string(counterhttp.HTTPFieldStatus) ||
		compiled.AllowedTuples()[0].Fields[0].Value.Tag() != choice.ValueInteger ||
		compiled.AllowedTuples()[0].Fields[0].Value.Text() != "404" {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_STATUS_PREDICATE_REFUSED")
	}
	parsedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), parsed)
	if err != nil || parsedDecision.Digest() != decision.Digest() {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_DECISION_PARSE_REFUSED: %v", err)
	}
	durableRuling, err := promotion.Finalize(ctx, objectStore, ready, decision)
	if err != nil || durableRuling.Record().Digest() != decision.Digest() {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_DURABLE_RULING_REFUSED: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(ctx, objectStore, durableRuling)
	if err != nil || !preparation.Valid() || !slices.Equal(preparation.SelectedFields(), []string{string(counterhttp.HTTPFieldStatus)}) {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_RULING_PREPARATION_REFUSED: %v", err)
	}
	resolved, err := projectiontranslate.Resolve(confirmed.ProjectionDefinition.Binding())
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	projectionAuthority, err := httpmodel.ResolveHTTPProjectionAuthority(
		confirmed.ProjectionDefinition.Digest(), confirmed.ProjectionDefinition.CanonicalBytes(),
		confirmed.ProjectionDefinition.Binding(),
	)
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	source, err := contractsource.NewHTTPSource(contractsource.HTTPInput{
		Plan: confirmed.Plan, Stimulus: confirmed.Stimulus, Start: confirmed.StartSpec,
		Capture: confirmed.CapturePolicy, Readiness: confirmed.Readiness,
		Profile: resolved.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, err
	}
	prepared, err := nodeemit.PrepareCompilation(ctx, objectStore, preparation, source)
	if err != nil || !prepared.Valid() || prepared.DecisionRecordDigest() != decision.Digest() ||
		prepared.ChoicepointDigest() != parsed.Digest() || prepared.SourceDigest() != source.Digest() {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_COMPILATION_PREPARATION_REFUSED: %v", err)
	}
	preparedBundle, err := nodeemit.CompilePrepared(prepared)
	if err != nil || !preparedBundle.Valid() || len(preparedBundle.Bundle().Files()) != 6 {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_CONTRACT_BUNDLE_REFUSED: %v", err)
	}
	bundle := preparedBundle.Bundle()
	parsedBundle, err := emitmodel.ParseContractBundle(bundle.CanonicalBytes(), bundle.Digest())
	if err != nil || parsedBundle.Digest() != bundle.Digest() ||
		!slices.Equal(parsedBundle.Predicate().SelectedFields(), []string{string(counterhttp.HTTPFieldStatus)}) {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_CONTRACT_PARSE_REFUSED: %v", err)
	}
	published, err := nodeemit.PublishPrepared(ctx, objectStore, preparedBundle)
	if err != nil || published.Disposition != nodeemit.PublicationCreated || !published.Residue.Valid() ||
		published.Residue.BundleDigest() != bundle.Digest() {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{}, emitmodel.ContractBundle{}, nodeemit.Residue{}, nil, fmt.Errorf("HTTP_STUDY_CONTRACT_PUBLICATION_REFUSED: %v", err)
	}
	return parsed, parsedDecision, durableRuling, source, parsedBundle, published.Residue, objectStore, nil
}

func executeStandaloneTrials(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
) (trials []standaloneTrial, returnErr error) {
	if config.runStandalone == nil {
		return nil, fmt.Errorf("HTTP_STUDY_STANDALONE_RUNNER_REFUSED")
	}
	bundleParent, err := privateDirectory(config.ScratchRoot, fmt.Sprintf("http-standalone-bundle-%d", config.Ordinal))
	if err != nil {
		return nil, err
	}
	bundleRoot := filepath.Join(bundleParent, "bundle")
	materialized, err := contractmaterialize.Materialize(ctx, objectStore, residue, bundleRoot)
	materializedSnapshot, materializedErr := standaloneMaterializationSnapshotFor(
		materialized, contractmaterialize.Created, contractmaterialize.StateCreated, bundleRoot, bundle.Digest(),
	)
	if err != nil || materializedErr != nil {
		return nil, fmt.Errorf("HTTP_STUDY_STANDALONE_BUNDLE_REFUSED: %w", errors.Join(err, materializedErr))
	}
	gitScratch, err := privateDirectory(authorityRoot, "contract-git-scratch")
	if err != nil {
		return nil, err
	}
	fixture, err := reference.OpenHTTPFixture(ctx, reference.HTTPFixtureConfig{
		Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
	})
	if err != nil {
		return nil, err
	}
	fixtureOpen := true
	defer func() {
		if fixtureOpen {
			returnErr = errors.Join(returnErr, fixture.Close())
		}
	}()
	displayRef, err := fixture.Ref(reference.ConcealNotFound)
	if err != nil {
		return nil, err
	}
	trials = make([]standaloneTrial, 0, 10)
	seenRoots := make(map[string]struct{}, 10)
	for trial := 1; trial <= 10; trial++ {
		standalone, trialErr := executeStandaloneTrial(
			ctx, config, objectStore, residue, bundle, fixture.Repository(), displayRef,
			bundleRoot, materializedSnapshot,
		)
		if trialErr != nil {
			return nil, fmt.Errorf("HTTP_STUDY_STANDALONE_%02d_REFUSED: %w", trial, trialErr)
		}
		if _, duplicate := seenRoots[standalone.candidateRoot]; duplicate {
			return nil, fmt.Errorf("HTTP_STUDY_TARGET_ROOT_REUSE_%02d_REFUSED", trial)
		}
		seenRoots[standalone.candidateRoot] = struct{}{}
		trials = append(trials, standalone)
	}
	if err := fixture.Close(); err != nil {
		return nil, fmt.Errorf("HTTP_STUDY_STANDALONE_FIXTURE_CLOSE_REFUSED: %v", err)
	}
	fixtureOpen = false
	terminalContext, terminalCancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
	defer terminalCancel()
	terminal, err := contractmaterialize.Materialize(terminalContext, objectStore, residue, bundleRoot)
	terminalSnapshot, terminalErr := standaloneMaterializationSnapshotFor(
		terminal, contractmaterialize.AlreadyExact, contractmaterialize.StateAlreadyExact, bundleRoot, bundle.Digest(),
	)
	if err != nil || terminalErr != nil || terminalSnapshot != materializedSnapshot {
		return nil, fmt.Errorf("HTTP_STUDY_STANDALONE_BUNDLE_TERMINAL_REFUSED: %v", err)
	}
	if err := validateStandaloneTrials(trials, bundle.Digest()); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("HTTP_STUDY_CONTEXT_ENDED_AFTER_STANDALONE_CLOSURE: %w", err)
	}
	return trials, nil
}

func executeStandaloneTrial(
	ctx context.Context,
	config Config,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
	repository gitobj.Repository,
	displayRef string,
	bundleRoot string,
	materialized standaloneMaterializationSnapshot,
) (trial standaloneTrial, returnErr error) {
	preRun, err := contractmaterialize.Materialize(ctx, objectStore, residue, bundleRoot)
	preRunSnapshot, preRunErr := standaloneMaterializationSnapshotFor(
		preRun, contractmaterialize.AlreadyExact, contractmaterialize.StateAlreadyExact, bundleRoot, bundle.Digest(),
	)
	if err != nil || preRunErr != nil || preRunSnapshot != materialized {
		return standaloneTrial{}, fmt.Errorf("bundle pre-run currentness failed: %w", errors.Join(err, preRunErr))
	}

	published, err := contractexec.PublishOfficialTarget(ctx, contractexec.PublishOfficialTargetRequest{
		Store: objectStore, Residue: residue, Repository: repository,
		DisplayRef: displayRef, NodeExecutable: config.NodeExecutable,
	})
	if err != nil || !published.Valid() {
		return standaloneTrial{}, errors.Join(err, fmt.Errorf("official target publication failed"))
	}
	publishedOpen := true
	defer func() {
		if publishedOpen {
			returnErr = errors.Join(returnErr, published.Close())
		}
	}()

	fresh, err := contractexec.ReopenOfficialTarget(ctx, published)
	if err != nil || !fresh.Valid() {
		return standaloneTrial{}, errors.Join(err, fmt.Errorf("fresh official target reopen failed"))
	}
	freshOpen := true
	defer func() {
		if freshOpen {
			returnErr = errors.Join(returnErr, fresh.Close())
		}
	}()
	freshSnapshot, err := snapshotStandaloneTarget(fresh, bundle.Digest())
	if err != nil {
		return standaloneTrial{}, err
	}

	observed, runErr := config.runStandalone(ctx, standaloneRunInput{
		workingDirectory:   freshSnapshot.candidateRoot,
		testFile:           filepath.Join(bundleRoot, "contract.test.mjs"),
		homeDirectory:      freshSnapshot.roots[3],
		temporaryDirectory: freshSnapshot.roots[4],
		timeout:            30 * time.Second,
	})
	observationErr := validateStandaloneObservation(observed)

	closureContext, closureCancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
	defer closureCancel()
	postRun, postRunMaterializeErr := contractmaterialize.Materialize(closureContext, objectStore, residue, bundleRoot)
	postRunSnapshot, postRunSnapshotErr := standaloneMaterializationSnapshotFor(
		postRun, contractmaterialize.AlreadyExact, contractmaterialize.StateAlreadyExact, bundleRoot, bundle.Digest(),
	)
	terminal, reopenErr := contractexec.ReopenOfficialTarget(closureContext, fresh)
	terminalOpen := reopenErr == nil && terminal.Valid()
	defer func() {
		if terminalOpen {
			returnErr = errors.Join(returnErr, terminal.Close())
		}
	}()
	var terminalSnapshot standaloneTargetSnapshot
	var terminalSnapshotErr error
	if terminalOpen {
		terminalSnapshot, terminalSnapshotErr = snapshotStandaloneTarget(terminal, bundle.Digest())
	}
	if runErr != nil || observationErr != nil || postRunMaterializeErr != nil || postRunSnapshotErr != nil ||
		postRunSnapshot != materialized || reopenErr != nil || !terminalOpen || terminalSnapshotErr != nil ||
		!sameStandaloneTargetSnapshot(freshSnapshot, terminalSnapshot) {
		return standaloneTrial{}, errors.Join(
			runErr, observationErr, postRunMaterializeErr, postRunSnapshotErr, reopenErr, terminalSnapshotErr,
			fmt.Errorf("direct standalone run or terminal authority check failed"),
		)
	}
	if err := ctx.Err(); err != nil {
		return standaloneTrial{}, fmt.Errorf("direct standalone context ended before observation admission: %w", err)
	}

	stdoutSHA256 := digestBytesSHA256(observed.stdout)
	stderrSHA256 := digestBytesSHA256(observed.stderr)
	processDigest, processCanonicalSHA256, digestErr := standaloneIdentity(
		"U7HTTPStandaloneProcessObservation",
		struct {
			SchemaVersion      string `json:"schema_version"`
			AttemptDigest      string `json:"attempt_digest"`
			TargetDigest       string `json:"target_digest"`
			BundleDigest       string `json:"bundle_digest"`
			ResidueHeadDigest  string `json:"residue_head_digest"`
			InvocationSHA256   string `json:"invocation_sha256"`
			ContractTestSHA256 string `json:"contract_test_sha256"`
			StdoutSHA256       string `json:"stdout_sha256"`
			StdoutBytes        int    `json:"stdout_bytes"`
			StderrSHA256       string `json:"stderr_sha256"`
			StderrBytes        int    `json:"stderr_bytes"`
			ExitCode           int    `json:"exit_code"`
			ObservedDiagnostic string `json:"observed_diagnostic"`
			Status             string `json:"status"`
			TerminalState      string `json:"terminal_state"`
		}{
			SchemaVersion: "countershape/u7-http-standalone-process/v1",
			AttemptDigest: freshSnapshot.attemptDigest.String(), TargetDigest: freshSnapshot.targetDigest.String(),
			BundleDigest: bundle.Digest().String(), ResidueHeadDigest: materialized.headDigest.String(),
			InvocationSHA256:   observed.invocationSHA256,
			ContractTestSHA256: observed.testFileSHA256,
			StdoutSHA256:       stdoutSHA256, StdoutBytes: len(observed.stdout),
			StderrSHA256: stderrSHA256, StderrBytes: len(observed.stderr),
			ExitCode: observed.exitCode, ObservedDiagnostic: "CONFORMS|NONE",
			Status: "STANDALONE_BUNDLE_TEST_PASSED", TerminalState: "CLEAN_NORMAL_RETURN",
		},
	)
	if digestErr != nil {
		return standaloneTrial{}, digestErr
	}
	return standaloneTrial{
		attemptDigest: freshSnapshot.attemptDigest, targetDigest: freshSnapshot.targetDigest,
		targetCanonicalSHA256: digestBytesSHA256(freshSnapshot.canonicalBytes), bundleDigest: bundle.Digest(),
		residueHeadDigest: materialized.headDigest, processDigest: processDigest,
		processCanonicalSHA256: processCanonicalSHA256, invocationSHA256: observed.invocationSHA256,
		testFileSHA256: observed.testFileSHA256,
		stdoutSHA256:   stdoutSHA256, stderrSHA256: stderrSHA256,
		stdoutBytes: len(observed.stdout), stderrBytes: len(observed.stderr), exitCode: observed.exitCode,
		observedDiagnostic: "CONFORMS|NONE", status: "STANDALONE_BUNDLE_TEST_PASSED",
		candidateRoot: freshSnapshot.candidateRoot,
	}, nil
}

func standaloneMaterializationSnapshotFor(
	materialized contractmaterialize.MaterializedContract,
	disposition contractmaterialize.Disposition,
	state string,
	destination string,
	bundleDigest domain.Digest,
) (standaloneMaterializationSnapshot, error) {
	if !materialized.Valid() || materialized.Disposition() != disposition || materialized.State() != state ||
		materialized.Destination() != destination || materialized.BundleDigest() != bundleDigest ||
		!materialized.ResidueHeadDigest().Valid() {
		return standaloneMaterializationSnapshot{}, fmt.Errorf("HTTP_STUDY_STANDALONE_MATERIALIZATION_REFUSED")
	}
	return standaloneMaterializationSnapshot{
		destination: materialized.Destination(), bundleDigest: materialized.BundleDigest(),
		headDigest: materialized.ResidueHeadDigest(),
	}, nil
}

func snapshotStandaloneTarget(
	capability contractexec.OfficialTarget,
	bundleDigest domain.Digest,
) (standaloneTargetSnapshot, error) {
	if !capability.Valid() || !bundleDigest.Valid() {
		return standaloneTargetSnapshot{}, fmt.Errorf("HTTP_STUDY_TARGET_SNAPSHOT_REFUSED")
	}
	target := capability.Model()
	record := capability.TargetRecord()
	roots := capability.Roots()
	candidateRoot := capability.CandidateRoot()
	contractBundle := capability.ContractBundle()
	rootRoster := [12]string{
		roots.AttemptRoot(), roots.CandidateParent(), roots.FixtureRoot(), roots.HomeRoot(), roots.TemporaryRoot(),
		roots.XDGConfigRoot(), roots.XDGCacheRoot(), roots.XDGDataRoot(), roots.XDGStateRoot(), roots.StateRoot(),
		roots.EvidenceRoot(), roots.MarkerPath(),
	}
	if !target.Valid() || target.Digest() != capability.Digest() || target.ContractBundleDigest() != bundleDigest ||
		!record.Valid() || record.Digest() != target.Digest() || !record.AttemptDigest().Valid() ||
		!contractBundle.Valid() || contractBundle.Digest() != bundleDigest ||
		candidateRoot != filepath.Join(roots.CandidateParent(), "candidate") {
		return standaloneTargetSnapshot{}, fmt.Errorf("HTTP_STUDY_TARGET_SNAPSHOT_JOIN_REFUSED")
	}
	for _, root := range rootRoster {
		if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return standaloneTargetSnapshot{}, fmt.Errorf("HTTP_STUDY_TARGET_ROOT_REFUSED")
		}
	}
	return standaloneTargetSnapshot{
		attemptDigest: record.AttemptDigest(), targetDigest: target.Digest(), bundleDigest: bundleDigest,
		canonicalBytes: target.CanonicalBytes(), candidateRoot: candidateRoot, roots: rootRoster,
	}, nil
}

func sameStandaloneTargetSnapshot(left, right standaloneTargetSnapshot) bool {
	return left.attemptDigest.Valid() && right.attemptDigest.Valid() &&
		left.attemptDigest == right.attemptDigest && left.targetDigest == right.targetDigest &&
		left.bundleDigest == right.bundleDigest && left.candidateRoot == right.candidateRoot &&
		left.roots == right.roots && bytes.Equal(left.canonicalBytes, right.canonicalBytes)
}

func validateStandaloneObservation(observed standaloneRunObservation) error {
	want := []byte("# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\n")
	if observed.exitCode != 0 || len(observed.stderr) != 0 || len(observed.stdout) == 0 ||
		len(observed.stdout) > 512<<10 || !utf8.Valid(observed.stdout) ||
		strings.ContainsAny(string(observed.stdout), "\r\x00\x1b") ||
		bytes.Count(observed.stdout, []byte("COUNTERSHAPE_RESULT_V1|")) != 1 ||
		bytes.Count(observed.stdout, want) != 1 ||
		bytes.Count(observed.stdout, []byte("# pass 1")) != 1 ||
		bytes.Count(observed.stdout, []byte("# fail 0")) != 1 ||
		!validAuthoritySHA256(observed.invocationSHA256) {
		return fmt.Errorf("HTTP_STUDY_STANDALONE_DIAGNOSTIC_REFUSED")
	}
	return nil
}

func standaloneIdentity(kind string, identity any) (domain.Digest, string, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, identity)
	if err != nil {
		return domain.Digest(""), "", err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return domain.Digest(""), "", err
	}
	return parsed, digestBytesSHA256(canonicalBytes), nil
}

func validateStandaloneTrials(trials []standaloneTrial, bundleDigest domain.Digest) error {
	if len(trials) != 10 || !bundleDigest.Valid() {
		return fmt.Errorf("HTTP_STUDY_STANDALONE_COUNT_REFUSED")
	}
	seenAttempts := make(map[domain.Digest]struct{}, len(trials))
	seenTargets := make(map[domain.Digest]struct{}, len(trials))
	seenProcesses := make(map[domain.Digest]struct{}, len(trials))
	seenInvocations := make(map[string]struct{}, len(trials))
	seenRoots := make(map[string]struct{}, len(trials))
	for _, trial := range trials {
		if !trial.attemptDigest.Valid() || !trial.targetDigest.Valid() || !trial.processDigest.Valid() ||
			trial.bundleDigest != bundleDigest || !trial.residueHeadDigest.Valid() ||
			trial.observedDiagnostic != "CONFORMS|NONE" || trial.status != "STANDALONE_BUNDLE_TEST_PASSED" ||
			trial.candidateRoot == "" || !filepath.IsAbs(trial.candidateRoot) || filepath.Clean(trial.candidateRoot) != trial.candidateRoot ||
			!validAuthoritySHA256(trial.targetCanonicalSHA256) ||
			!validAuthoritySHA256(trial.processCanonicalSHA256) ||
			!validAuthoritySHA256(trial.invocationSHA256) || !validAuthoritySHA256(trial.testFileSHA256) ||
			!validAuthoritySHA256(trial.stdoutSHA256) ||
			!validAuthoritySHA256(trial.stderrSHA256) || trial.stdoutBytes <= 0 || trial.stderrBytes != 0 || trial.exitCode != 0 {
			return fmt.Errorf("HTTP_STUDY_STANDALONE_TRIAL_REFUSED")
		}
		for _, item := range []struct {
			digest domain.Digest
			seen   map[domain.Digest]struct{}
		}{
			{trial.attemptDigest, seenAttempts},
			{trial.targetDigest, seenTargets},
			{trial.processDigest, seenProcesses},
		} {
			if _, duplicate := item.seen[item.digest]; duplicate {
				return fmt.Errorf("HTTP_STUDY_STANDALONE_AUTHORITY_REUSE_REFUSED")
			}
			item.seen[item.digest] = struct{}{}
		}
		if _, duplicate := seenInvocations[trial.invocationSHA256]; duplicate {
			return fmt.Errorf("HTTP_STUDY_STANDALONE_INVOCATION_REUSE_REFUSED")
		}
		seenInvocations[trial.invocationSHA256] = struct{}{}
		if _, duplicate := seenRoots[trial.candidateRoot]; duplicate {
			return fmt.Errorf("HTTP_STUDY_STANDALONE_ROOT_REUSE_REFUSED")
		}
		seenRoots[trial.candidateRoot] = struct{}{}
	}
	return nil
}

func phaseTrialReceipts(completed CompletedHTTPStudy) ([]phaseTrialReceipt, error) {
	receipts := make([]phaseTrialReceipt, 0, 98)
	seen := make(map[domain.Digest]struct{}, 98)
	appendAuthority := func(phase string, authority domain.Digest) error {
		if _, err := rawSHA256(authority); err != nil {
			return err
		}
		if _, duplicate := seen[authority]; duplicate {
			return fmt.Errorf("HTTP_STUDY_PHASE_AUTHORITY_REUSE_REFUSED")
		}
		seen[authority] = struct{}{}
		trial := 1
		if len(receipts) > 0 && receipts[len(receipts)-1].phase == phase {
			trial = receipts[len(receipts)-1].trial + 1
		}
		receipts = append(receipts, phaseTrialReceipt{phase: phase, trial: trial, authority: authority})
		return nil
	}

	searchStudies := []Result{
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint,
		completed.baseline, completed.minimized,
	}
	for _, study := range searchStudies {
		for _, trial := range study.Trials {
			attempt := trial.Result.FinalizedAttempt()
			if !trial.Admitted || attempt.HasControls() || !trial.Result.World().Digest().Valid() ||
				!attempt.ArtifactDigest().Valid() || !trial.Observation.Digest().Valid() {
				return nil, fmt.Errorf("HTTP_STUDY_SEARCH_TRIAL_AUTHORITY_REFUSED")
			}
			if err := appendAuthority("search", attempt.ArtifactDigest()); err != nil {
				return nil, err
			}
		}
	}
	if len(receipts) != 80 {
		return nil, fmt.Errorf("HTTP_STUDY_SEARCH_PHYSICAL_COUNT_REFUSED")
	}

	facts := completed.confirmed.Confirmation.Draft().PhysicalFacts()
	if len(facts) != 8 {
		return nil, fmt.Errorf("HTTP_STUDY_CONFIRMATION_FACT_COUNT_REFUSED")
	}
	for _, fact := range facts {
		if !fact.Valid() {
			return nil, fmt.Errorf("HTTP_STUDY_CONFIRMATION_FACT_REFUSED")
		}
		if err := appendAuthority("confirm", fact.Digest()); err != nil {
			return nil, err
		}
	}

	if len(completed.standaloneTrials) != 10 {
		return nil, fmt.Errorf("HTTP_STUDY_CONTRACT_RECEIPT_COUNT_REFUSED")
	}
	for _, trial := range completed.standaloneTrials {
		if trial.status != "STANDALONE_BUNDLE_TEST_PASSED" ||
			trial.observedDiagnostic != "CONFORMS|NONE" || !trial.processDigest.Valid() {
			return nil, fmt.Errorf("HTTP_STUDY_CONTRACT_RECEIPT_REFUSED")
		}
		if err := appendAuthority("contract", trial.processDigest); err != nil {
			return nil, err
		}
	}
	if len(receipts) != 98 {
		return nil, fmt.Errorf("HTTP_STUDY_PHASE_RECEIPT_COUNT_REFUSED")
	}
	return receipts, nil
}

func rawSHA256(digest domain.Digest) (string, error) {
	if !digest.Valid() {
		return "", fmt.Errorf("HTTP_STUDY_PHASE_AUTHORITY_REFUSED")
	}
	raw := strings.TrimPrefix(digest.String(), "sha256:")
	if len(raw) != 64 || raw == strings.Repeat("0", 64) {
		return "", fmt.Errorf("HTTP_STUDY_PHASE_AUTHORITY_REFUSED")
	}
	for _, character := range raw {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return "", fmt.Errorf("HTTP_STUDY_PHASE_AUTHORITY_REFUSED")
		}
	}
	return raw, nil
}

func physicalRunAuthority(completed CompletedHTTPStudy) (domain.Digest, error) {
	receipts, err := phaseTrialReceipts(completed)
	if err != nil {
		return domain.Digest(""), err
	}
	parts := make([]string, 0, len(receipts)+1)
	parts = append(parts, "countershape/u7-http-physical-run/v1")
	ordinalPrefix := fmt.Sprintf("u7-http-%d-", completed.ordinal)
	for _, study := range []Result{
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint,
		completed.baseline, completed.minimized,
	} {
		for _, trial := range study.Trials {
			world := trial.Result.World()
			if !world.Digest().Valid() || !strings.HasPrefix(world.InstanceNonce(), ordinalPrefix) {
				return domain.Digest(""), fmt.Errorf("HTTP_STUDY_ORDINAL_WORLD_AUTHORITY_REFUSED")
			}
			parts = append(parts, world.InstanceNonce(), world.Digest().String())
		}
	}
	for _, receipt := range receipts {
		parts = append(parts, receipt.phase, fmt.Sprintf("%d", receipt.trial), receipt.authority.String())
	}
	return domain.ParseDigest("sha256:" + hashStrings(parts))
}

func validateCompletedHTTPStudy(completed CompletedHTTPStudy) error {
	if completed.seal != issuedCompletedHTTPStudy || completed.ordinal < 1 || completed.ordinal > 3 ||
		!completed.physicalRunAuthority.Valid() || validateDiscovery(completed.discovery) != nil ||
		validateDiscovery(completed.recoveryOne) != nil || validateDiscovery(completed.recoveryTwo) != nil ||
		compare.AssessPreservation(completed.discovery.OutcomeMap, completed.recoveryOne.OutcomeMap).Relation() != compare.PreservationEqual ||
		compare.AssessPreservation(completed.discovery.OutcomeMap, completed.recoveryTwo.OutcomeMap).Relation() != compare.PreservationEqual ||
		!completed.shapeAssessment.Valid() || completed.shapeAssessment.Relation() != compare.PreservationDifferent ||
		!completed.reductionRun.Valid() || !completed.weakReduction.Valid() || !completed.strongReduction.Valid() ||
		completed.weakReduction.Grade().Status() != grade.StatusBestKnown ||
		completed.strongReduction.Grade().Status() != grade.StatusOneMinimalUnder ||
		validateConfirmation(completed.minimized, completed.confirmed) != nil ||
		!completed.choicepoint.Valid() || !completed.decision.Valid() || !completed.portableSource.Valid() ||
		!completed.bundle.Valid() || !completed.residue.Valid() || len(completed.bundle.Files()) != 6 ||
		completed.bundle.DecisionRecordDigest() != completed.decision.Digest() ||
		completed.bundle.ChoicepointDigest() != completed.choicepoint.Digest() ||
		completed.bundle.PortableSource().Digest() != completed.portableSource.Digest() ||
		!slices.Equal(completed.bundle.Predicate().SelectedFields(), []string{string(counterhttp.HTTPFieldStatus)}) ||
		validateStandaloneTrials(completed.standaloneTrials, completed.bundle.Digest()) != nil {
		return fmt.Errorf("HTTP_STUDY_COMPLETION_REFUSED")
	}
	if validateFreshDisjoint(
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint,
		completed.baseline, completed.minimized, completed.confirmed,
	) != nil {
		return fmt.Errorf("HTTP_STUDY_COMPLETE_FRESHNESS_REFUSED")
	}
	authority, err := physicalRunAuthority(completed)
	if err != nil || authority != completed.physicalRunAuthority {
		return fmt.Errorf("HTTP_STUDY_PHYSICAL_RUN_AUTHORITY_REFUSED")
	}
	return nil
}

func evidenceInput(completed CompletedHTTPStudy, root string) (evidencePublicationInput, error) {
	if err := validateCompletedHTTPStudy(completed); err != nil {
		return evidencePublicationInput{}, err
	}
	worlds, attempts, captures, measurements, projections := freshStudyEvidence(
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged,
		completed.baselineCheckpoint, completed.baseline, completed.minimized, completed.confirmed,
	)
	confirmationFacts := completed.confirmed.Confirmation.Draft().PhysicalFacts()
	confirmationDigests := make([]string, len(confirmationFacts))
	for index, fact := range confirmationFacts {
		confirmationDigests[index] = fact.Digest().String()
	}
	targetDigests := make([]string, 10)
	targetCanonical := make([]string, 10)
	attemptDigests := make([]string, 10)
	residueHeadDigests := make([]string, 10)
	processDigests := make([]string, 10)
	processCanonical := make([]string, 10)
	invocationSHA256 := make([]string, 10)
	testFileSHA256 := make([]string, 10)
	stdoutSHA256 := make([]string, 10)
	stderrSHA256 := make([]string, 10)
	stdoutBytes := make([]int, 10)
	stderrBytes := make([]int, 10)
	exitCodes := make([]int, 10)
	for index, trial := range completed.standaloneTrials {
		attemptDigests[index] = trial.attemptDigest.String()
		targetDigests[index] = trial.targetDigest.String()
		targetCanonical[index] = trial.targetCanonicalSHA256
		residueHeadDigests[index] = trial.residueHeadDigest.String()
		processDigests[index] = trial.processDigest.String()
		processCanonical[index] = trial.processCanonicalSHA256
		invocationSHA256[index] = trial.invocationSHA256
		testFileSHA256[index] = trial.testFileSHA256
		stdoutSHA256[index] = trial.stdoutSHA256
		stderrSHA256[index] = trial.stderrSHA256
		stdoutBytes[index] = trial.stdoutBytes
		stderrBytes[index] = trial.stderrBytes
		exitCodes[index] = trial.exitCode
	}
	phaseTrials, err := phaseTrialReceipts(completed)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	searchAttemptGroups, confirmationAttemptDigests, err := phaseAttemptPartitions(completed, attempts, phaseTrials)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	predicateSHA := digestBytesSHA256(completed.bundle.Predicate().CanonicalBytes())
	semanticDecisionSHA := hashStrings([]string{
		string(completed.decision.Action()), strings.Join(completed.decision.SelectedFields(), "\x00"), predicateSHA,
	})
	semanticBundleSHA := hashStrings([]string{
		completed.portableSource.Digest().String(), predicateSHA, "six-files", string(completed.decision.Action()),
	})
	return evidencePublicationInput{
		Root: root, Domain: evidenceDomainHTTP, Ordinal: completed.ordinal,
		PhysicalRunAuthority: completed.physicalRunAuthority, PhaseTrials: phaseTrials,
		Deterministic: DeterministicEvidencePayloads{
			SourceSpec: EvidencePayload{
				"authority": "STRICT_SOURCE_SPEC_COMPILED_BY_SEALED_CORE", "canonical_sha256": digestBytesSHA256(completed.discovery.SourceSpecBytes),
				"digest": completed.discovery.SourceSpecDigest.String(),
			},
			WorldPlan: EvidencePayload{
				"authority": "WORLD_PLAN_COMPILED_FROM_STRICT_SOURCE", "canonical_sha256": digestBytesSHA256(completed.discovery.Plan.CanonicalBytes()),
				"digest": completed.discovery.Plan.Digest().String(),
			},
			Ruling: EvidencePayload{
				"action": string(completed.decision.Action()), "allowed_status": 404,
				"authority": "DETERMINISTIC_STATUS_ONLY_RULING_PROJECTION_FROM_SEALED_DECISION", "predicate_sha256": predicateSHA,
				"projection_kind": "SEMANTIC_REGRESSION_PROJECTION",
				"selected_fields": []string{string(counterhttp.HTTPFieldStatus)},
			},
			DecisionRecord: EvidencePayload{
				"authority": "DETERMINISTIC_DECISION_SEMANTIC_PROJECTION_FROM_STRICT_RECORD", "canonical_record_valid": true,
				"early_reveal": false, "semantic_sha256": semanticDecisionSHA,
				"projection_kind": "SEMANTIC_REGRESSION_PROJECTION",
				"selected_fields": []string{string(counterhttp.HTTPFieldStatus)},
			},
			ContractBundle: EvidencePayload{
				"authority": "DETERMINISTIC_CONTRACT_SEMANTIC_PROJECTION_FROM_STRICT_SIX_FILE_BUNDLE", "canonical_bundle_valid": true,
				"member_count": 6, "predicate_sha256": predicateSHA, "semantic_sha256": semanticBundleSHA,
				"projection_kind": "SEMANTIC_REGRESSION_PROJECTION",
				"selected_fields": []string{string(counterhttp.HTTPFieldStatus)},
			},
		},
		Fresh: FreshEvidencePayloads{
			WorldInstance: EvidencePayload{
				"authority": "FRESH_WORLD_INSTANCE_DIGESTS", "digests": worlds,
				"physical_run_authority": completed.physicalRunAuthority.String(),
			},
			Attempts: EvidencePayload{
				"authority": "FRESH_ATTEMPT_ARTIFACT_DIGESTS", "digests": attempts,
				"confirmation_attempt_digests": confirmationAttemptDigests,
				"physical_run_authority":       completed.physicalRunAuthority.String(),
				"search_attempt_groups":        searchAttemptGroups,
				"search_phase_receipts":        phaseReceiptPayload(phaseTrials[:80]),
			},
			Measurements: EvidencePayload{
				"authority": "FRESH_INSTANCE_MEASUREMENT_DIGESTS", "digests": measurements,
				"envelope_digest":        completed.discovery.Envelope.Digest().String(),
				"physical_run_authority": completed.physicalRunAuthority.String(),
			},
			Captures: EvidencePayload{
				"authority": "FRESH_CAPTURE_AND_PROJECTION_DIGESTS", "observation_digests": captures,
				"physical_run_authority": completed.physicalRunAuthority.String(),
				"projection_digests":     projections,
			},
			Confirmation: EvidencePayload{
				"authority": "PHYSICAL_ROTATED_FRESH_CONFIRMATION", "canonical_sha256": digestBytesSHA256(completed.confirmed.Confirmation.Draft().CanonicalBytes()),
				"digest": completed.confirmed.Confirmation.Draft().Digest().String(), "physical_fact_digests": confirmationDigests,
				"physical_run_authority": completed.physicalRunAuthority.String(),
				"phase_receipts":         phaseReceiptPayload(phaseTrials[80:88]),
			},
			TargetProjection: EvidencePayload{
				"authority": "TEN_FRESH_PRESPAWN_OFFICIAL_TARGETS", "attempt_digests": attemptDigests,
				"canonical_sha256": targetCanonical, "digests": targetDigests,
				"contract_bundle_digest": completed.bundle.Digest().String(),
				"physical_run_authority": completed.physicalRunAuthority.String(), "residue_head_digests": residueHeadDigests,
			},
			ProcessObservation: EvidencePayload{
				"authority": "FINALIZED_RUN_AUTHORITY_ABSENT_DIRECT_PROCESS_FACTS_ONLY", "availability": "ABSENT",
				"contract_bundle_digest": completed.bundle.Digest().String(), "contract_test_sha256": testFileSHA256,
				"exit_codes":        exitCodes,
				"invocation_sha256": invocationSHA256, "physical_run_authority": completed.physicalRunAuthority.String(),
				"process_fact_canonical_sha256": processCanonical, "process_fact_digests": processDigests,
				"stderr_bytes": stderrBytes, "stderr_sha256": stderrSHA256,
				"stdout_bytes": stdoutBytes, "stdout_sha256": stdoutSHA256, "target_digests": targetDigests,
			},
			StandaloneResult: EvidencePayload{
				"authority": "CLASSIFICATION_AUTHORITY_ABSENT_DIRECT_BUNDLE_RESULTS_ONLY", "availability": "ABSENT",
				"observed_diagnostic": "CONFORMS|NONE", "outcome": "STANDALONE_BUNDLE_TEST_PASSED",
				"phase_receipts":         phaseReceiptPayload(phaseTrials[88:]),
				"physical_run_authority": completed.physicalRunAuthority.String(), "process_fact_digests": processDigests,
			},
		},
	}, nil
}

func phaseAttemptPartitions(
	completed CompletedHTTPStudy,
	allAttemptDigests []string,
	phaseTrials []phaseTrialReceipt,
) ([]map[string]any, []string, error) {
	searchStudies := []struct {
		name  string
		count int
		study Result
	}{
		{name: "decisive", count: 12, study: completed.discovery},
		{name: "recovery_one", count: 12, study: completed.recoveryOne},
		{name: "recovery_two", count: 12, study: completed.recoveryTwo},
		{name: "shape_reference", count: 4, study: completed.shapeReference},
		{name: "shape_changed", count: 4, study: completed.shapeChanged},
		{name: "baseline_checkpoint", count: 12, study: completed.baselineCheckpoint},
		{name: "divergent_baseline", count: 12, study: completed.baseline},
		{name: "accepted_reducer_evaluation", count: 12, study: completed.minimized},
	}
	groups := make([]map[string]any, 0, len(searchStudies))
	searchDigests := make([]string, 0, 80)
	for _, item := range searchStudies {
		digests, err := attemptDigestsInTrialOrder(item.study)
		if err != nil || len(digests) != item.count {
			return nil, nil, fmt.Errorf("HTTP_STUDY_SEARCH_ATTEMPT_PARTITION_REFUSED")
		}
		groups = append(groups, map[string]any{"name": item.name, "digests": digests})
		searchDigests = append(searchDigests, digests...)
	}
	if len(searchDigests) != 80 || len(phaseTrials) != 98 {
		return nil, nil, fmt.Errorf("HTTP_STUDY_ATTEMPT_PARTITION_COUNT_REFUSED")
	}
	for index, digest := range searchDigests {
		if phaseTrials[index].phase != "search" || phaseTrials[index].trial != index+1 ||
			phaseTrials[index].authority.String() != digest {
			return nil, nil, fmt.Errorf("HTTP_STUDY_SEARCH_ATTEMPT_RECEIPT_JOIN_REFUSED")
		}
	}
	confirmationDigests, err := attemptDigestsInTrialOrder(completed.confirmed)
	if err != nil || len(confirmationDigests) != 8 {
		return nil, nil, fmt.Errorf("HTTP_STUDY_CONFIRMATION_ATTEMPT_PARTITION_REFUSED")
	}
	partition := append(append([]string(nil), searchDigests...), confirmationDigests...)
	sort.Strings(partition)
	if !slices.Equal(partition, allAttemptDigests) {
		return nil, nil, fmt.Errorf("HTTP_STUDY_ATTEMPT_PARTITION_CLOSURE_REFUSED")
	}
	return groups, confirmationDigests, nil
}

func attemptDigestsInTrialOrder(study Result) ([]string, error) {
	result := make([]string, len(study.Trials))
	for index, trial := range study.Trials {
		attempt := trial.Result.FinalizedAttempt()
		if !trial.Admitted || attempt.HasControls() || !attempt.ArtifactDigest().Valid() {
			return nil, fmt.Errorf("HTTP_STUDY_ATTEMPT_PARTITION_REFUSED")
		}
		result[index] = attempt.ArtifactDigest().String()
	}
	return result, nil
}

func phaseReceiptPayload(receipts []phaseTrialReceipt) []map[string]any {
	result := make([]map[string]any, len(receipts))
	for index, receipt := range receipts {
		raw, err := rawSHA256(receipt.authority)
		if err != nil {
			return nil
		}
		result[index] = map[string]any{
			"authority_sha256": raw, "phase": receipt.phase, "trial": receipt.trial,
		}
	}
	return result
}

func freshStudyEvidence(studies ...Result) (worlds, attempts, captures, measurements, projections []string) {
	for _, study := range studies {
		worlds = append(worlds, digestStrings(study.OutcomeMap.EvidenceWorldDigests())...)
		attempts = append(attempts, digestStrings(study.OutcomeMap.EvidenceAttemptDigests())...)
		captures = append(captures, digestStrings(study.OutcomeMap.EvidenceObservationDigests())...)
		for _, trial := range study.Trials {
			measurements = append(measurements, trial.Measurements.Digest().String())
			projections = append(projections, trial.Projection.Digest().String())
		}
	}
	for _, values := range [][]string{worlds, attempts, captures, measurements, projections} {
		sort.Strings(values)
	}
	return worlds, attempts, captures, measurements, projections
}

func digestStrings(input []domain.Digest) []string {
	result := make([]string, len(input))
	for index, digest := range input {
		result[index] = digest.String()
	}
	sort.Strings(result)
	return result
}

func hashStrings(input []string) string {
	hash := sha256.New()
	for _, value := range input {
		hash.Write([]byte(value))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func digestBytesSHA256(input []byte) string {
	digest := sha256.Sum256(input)
	return hex.EncodeToString(digest[:])
}
