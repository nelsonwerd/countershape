//go:build darwin && cgo

package http_invoices

import (
	"bytes"
	"context"
	"encoding/base64"
	"path/filepath"
	"slices"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
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
	baselineCheckpoint, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery ||
		len(baselineResult.OutcomeMap.Entries()) != 3 || len(baselineResult.OutcomeMap.Exclusions()) != 1 {
		t.Fatal("physical HTTP baseline did not retain the stable A/B/C map plus excluded D")
	}
	if !baselineCheckpoint.HasOutcomeMap || baselineCheckpoint.Plan.Digest() != baselineResult.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineResult.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(baselineCheckpoint.OutcomeMap.CanonicalBytes(), baselineResult.OutcomeMap.CanonicalBytes()) {
		t.Fatal("physical HTTP baseline checkpoint did not retain fresh equivalent evidence")
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
	httpFields := []string{
		string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldContentType),
		string(counterhttp.HTTPFieldBodyKind), string(counterhttp.HTTPFieldBodyMetadata),
	}
	blindDTO := blind.DTO()
	if blindDTO.ProjectionMode() != "ADAPTER_BOUND_PORTABLE_FIELDS_V1" ||
		!slices.Equal(blindDTO.SelectableFields(), httpFields) ||
		!slices.Equal(blindDTO.DifferingFields(), []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
			string(counterhttp.HTTPFieldBodyMetadata),
		}) {
		t.Fatalf("physical HTTP Choicepoint lacks exact portable profile order or separation: selectable=%#v differing=%#v mode=%q",
			blindDTO.SelectableFields(), blindDTO.DifferingFields(), blindDTO.ProjectionMode())
	}
	for _, card := range blindDTO.Cards() {
		if len(card.Fields) != len(httpFields) {
			t.Fatalf("HTTP blind card has %d fields, want %d", len(card.Fields), len(httpFields))
		}
		for index, field := range card.Fields {
			if field.FieldID != httpFields[index] {
				t.Fatalf("HTTP blind card field %d = %q, want %q", index, field.FieldID, httpFields[index])
			}
		}
		contentType := card.Fields[1]
		contentTypeBytes, decodeErr := base64.StdEncoding.Strict().DecodeString(contentType.CanonicalJSONBase64)
		if decodeErr != nil || base64.StdEncoding.EncodeToString(contentTypeBytes) != contentType.CanonicalJSONBase64 ||
			contentType.Tag != string(choice.ValueOrderedStringList) || contentType.Text != "" || contentType.Boolean ||
			!bytes.Equal(contentTypeBytes, []byte(`["application/json"]`)) {
			t.Fatalf("HTTP blind card lost exact ordered content-type evidence: %#v", contentType)
		}
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

	promotionRoot := filepath.Join(t.TempDir(), "promotion-store")
	promotionStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	studyID, err := store.NewStudyID("physical HTTP choicepoint promotion")
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
	head, err = promotionStore.AdvanceReduction(context.Background(), head, run)
	if err != nil {
		t.Fatal(err)
	}
	storedConfirmation, err := promotion.PersistConfirmation(context.Background(), promotionStore, head, confirmationDraft)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "Which exact invoice response behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || ready.Record().Digest() != choicepoint.Digest() {
		t.Fatalf("physical HTTP store-bound Choicepoint promotion failed: %v", err)
	}
	restartedStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || reopenedReady.Record().Digest() != ready.Record().Digest() {
		t.Fatalf("physical HTTP CHOICEPOINT_READY did not survive restart: %v", err)
	}
	readyRecord := reopenedReady.Record()

	smuggledContentType, err := choice.OrderedStringListValue([]string{"application/json"})
	if err != nil {
		t.Fatal(err)
	}
	if _, smuggleErr := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldStatus)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldContentType), Value: smuggledContentType}},
	); !choice.IsRefusal(smuggleErr, choice.CodeIncompleteTuple) {
		t.Fatalf("same-cardinality unselected HTTP field smuggled into custom expectation: %v", smuggleErr)
	}

	customContentTypes := []string{"application/problem+json", "", "application/problem+json"}
	customContentType, err := choice.OrderedStringListValue(customContentTypes)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeExpectation, err := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldContentType)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldContentType), Value: customContentType}},
	)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeInput := choice.RulingDraftInput{
		Action: choice.ActionCustomExpectation, SelectedFields: []string{string(counterhttp.HTTPFieldContentType)},
		AllowedAliases: []string{}, CustomExpectation: &contentTypeExpectation,
		CustomReviewer: "physical-http-content-type-reviewer", CustomReviewEvidence: confirmationDraft.Digest(),
	}
	contentTypeSession, err := choice.NewSession(readyRecord)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeSession, _, err = contentTypeSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		contentTypeSession, err = contentTypeSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	contentTypeSession, err = contentTypeSession.Revise(contentTypeInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, contentTypeDecision, err := contentTypeSession.Finalize(
		"local-test-operator", "Preserve exact ordered content-type members.", []domain.ReceiptReference{},
	)
	if err != nil || !contentTypeDecision.EarlyReveal() ||
		!slices.Equal(contentTypeDecision.SelectedFields(), []string{string(counterhttp.HTTPFieldContentType)}) ||
		!slices.Equal(contentTypeDecision.NonassertedFields(), []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
			string(counterhttp.HTTPFieldBodyMetadata),
		}) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord lost its selected-only partition: %v", err)
	}
	contentTypeCompiled, ok := contentTypeDecision.CompilableRuling()
	if !ok {
		t.Fatal("portable HTTP ordered-list DecisionRecord was not compilable")
	}
	contentTypeAllowed := contentTypeCompiled.AllowedTuples()
	if len(contentTypeAllowed) != 1 || len(contentTypeAllowed[0].Fields) != 1 ||
		contentTypeAllowed[0].Fields[0].FieldID != string(counterhttp.HTTPFieldContentType) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord changed tuple shape: %#v", contentTypeAllowed)
	}
	ordered, ok := contentTypeAllowed[0].Fields[0].Value.OrderedStrings()
	if !ok || !slices.Equal(ordered, customContentTypes) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord changed order or duplicates: %#v", ordered)
	}
	parsedContentType, err := choice.ParseDecisionRecord(contentTypeDecision.CanonicalBytes(), readyRecord)
	if err != nil || parsedContentType.Digest() != contentTypeDecision.Digest() {
		t.Fatalf("portable HTTP ordered-list DecisionRecord did not round trip strictly: %v", err)
	}

	status401, err := choice.IntegerValue("401")
	if err != nil {
		t.Fatal(err)
	}
	custom401, err := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldStatus)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldStatus), Value: status401}},
	)
	if err != nil {
		t.Fatal(err)
	}
	customFields := custom401.Fields()
	if len(customFields) != 1 || customFields[0].FieldID != string(counterhttp.HTTPFieldStatus) ||
		customFields[0].Value.Tag() != choice.ValueInteger || customFields[0].Value.Text() != "401" {
		t.Fatalf("custom HTTP expectation retained unselected context: %#v", customFields)
	}
	customInput := choice.RulingDraftInput{
		Action: choice.ActionCustomExpectation, SelectedFields: []string{string(counterhttp.HTTPFieldStatus)},
		AllowedAliases: []string{}, CustomExpectation: &custom401,
		CustomReviewer: "physical-http-contract-reviewer", CustomReviewEvidence: confirmationDraft.Digest(),
	}
	customSession, err := choice.NewSession(readyRecord)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
	} {
		customSession, err = customSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	customSession, err = customSession.Propose(customInput)
	if err != nil {
		t.Fatal(err)
	}
	customSession, _, err = customSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	customSession, err = customSession.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	customSession, err = customSession.Revise(customInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, customDecision, err := customSession.Finalize(
		"local-test-operator", "Accept only the separately reviewed HTTP 401 status.", []domain.ReceiptReference{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if customDecision.Action() != choice.ActionCustomExpectation ||
		!slices.Equal(customDecision.SelectedFields(), httpFields[:1]) ||
		!slices.Equal(customDecision.NonassertedFields(), httpFields[1:]) ||
		len(customDecision.ConfirmedAllowedOutcomeIDs()) != 0 ||
		len(customDecision.ConfirmedDisallowedOutcomeIDs()) != len(confirmedStudy.OutcomeMap.Entries()) {
		t.Fatalf("custom HTTP DecisionRecord lost its selected-only partition")
	}
	compiled, compilable := customDecision.CompilableRuling()
	if !compilable {
		t.Fatal("custom HTTP 401 decision was not compilable")
	}
	allowed := compiled.AllowedTuples()
	if len(allowed) != 1 || len(allowed[0].Fields) != 1 ||
		allowed[0].Fields[0].FieldID != string(counterhttp.HTTPFieldStatus) ||
		allowed[0].Fields[0].Value.Text() != "401" {
		t.Fatalf("compiled HTTP 401 predicate smuggled profile context: %#v", allowed)
	}
	parsedDecision, err := choice.ParseDecisionRecord(customDecision.CanonicalBytes(), readyRecord)
	if err != nil || parsedDecision.Digest() != customDecision.Digest() {
		t.Fatalf("custom HTTP DecisionRecord did not round trip strictly: %v", err)
	}
	durableRuling, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, customDecision)
	if err != nil || durableRuling.Record().Digest() != customDecision.Digest() {
		t.Fatalf("physical HTTP DecisionRecord promotion failed: %v", err)
	}
	rulingRestart, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedRuling, err := promotion.OpenRuling(context.Background(), rulingRestart, studyID)
	if err != nil || reopenedRuling.Record().Digest() != customDecision.Digest() {
		t.Fatalf("physical HTTP RULING did not survive restart: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), rulingRestart, reopenedRuling)
	if err != nil || !preparation.Valid() || !preparation.ProfileDigest().Valid() ||
		preparation.DecisionDigest() != parsedDecision.Digest() ||
		!slices.Equal(preparation.SelectedFields(), httpFields[:1]) ||
		promotion.ValidatePortableRulingPreparation(context.Background(), rulingRestart, preparation) != nil {
		t.Fatalf("current portable HTTP ruling preparation failed: %#v, %v", preparation, err)
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
