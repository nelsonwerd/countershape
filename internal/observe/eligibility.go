// Package observe owns the single generic eligibility gate and finite batch
// classifier. Domain-specific HTTP and CLI facts stop at this boundary.
package observe

import (
	"fmt"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe/eligibilitycore"
)

const maxProjectionResultCanonicalBytes = canon.MaxInputBytes - 16*1024

// StructuralCapture is a trusted adapter handoff whose projection fingerprint
// is computed from exact canonical bytes. It is not I/O or freshness proof.
type StructuralCapture struct {
	observationDigest          domain.Digest
	projectionDigest           domain.Digest
	worldDigest                domain.Digest
	attemptDigest              domain.Digest
	capturePolicyDigest        domain.Digest
	projectionDefinitionDigest domain.Digest
	derivationDigest           domain.Digest
	fingerprint                domain.ProjectionFingerprint
	canonicalProjection        []byte
}

func NewStructuralCapture(
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	observationDigest domain.Digest,
	canonicalProjection []byte,
	derivation ProjectionDerivation,
) (StructuralCapture, error) {
	if !world.Digest().Valid() || !attempt.ArtifactDigest().Valid() ||
		world.AttemptArtifactDigest() != attempt.ArtifactDigest() || world.Purpose() != attempt.Purpose() ||
		!world.CapturePolicyDigest().Valid() || !world.ProjectionDefinitionDigest().Valid() || !observationDigest.Valid() ||
		!derivation.Valid() || derivation.observationDigest != observationDigest ||
		derivation.definitionDigest != world.ProjectionDefinitionDigest() {
		return StructuralCapture{}, &domain.Error{Code: "INCOMPLETE_CAPTURE_PROVENANCE"}
	}
	if len(canonicalProjection) > maxProjectionResultCanonicalBytes {
		return StructuralCapture{}, &domain.Error{Code: "CAPTURE_PROJECTION_TOO_LARGE", Detail: "projection result reserves canonical profile headroom for lineage metadata"}
	}
	fingerprint, err := domain.NewProjectionFingerprint(canonicalProjection)
	if err != nil {
		return StructuralCapture{}, &domain.Error{Code: "NONCANONICAL_CAPTURE_PROJECTION", Detail: err.Error()}
	}
	if derivation.projectionFingerprint != fingerprint {
		return StructuralCapture{}, &domain.Error{Code: "PROJECTION_DERIVATION_RESULT_MISMATCH"}
	}
	// A fingerprint groups byte-identical canonical projections. A projection
	// result is a different artifact: it binds those bytes to the exact world,
	// attempt, observation, capture policy, and projection definition that
	// produced them. Accepting either digest from an adapter would let metadata
	// disagree with the bytes and lineage that actually drive comparison.
	resultIdentity := struct {
		SchemaVersion              string `json:"schema_version"`
		Kind                       string `json:"kind"`
		WorldInstanceDigest        string `json:"world_instance_digest"`
		AttemptArtifactDigest      string `json:"attempt_artifact_digest"`
		CapturedObservationDigest  string `json:"captured_observation_digest"`
		CapturePolicyDigest        string `json:"capture_policy_digest"`
		ProjectionDefinitionDigest string `json:"projection_definition_digest"`
		ProjectionFingerprint      string `json:"projection_fingerprint"`
		ProjectionDerivationDigest string `json:"projection_derivation_digest"`
		CanonicalProjection        string `json:"canonical_projection"`
	}{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ProjectionResult",
		WorldInstanceDigest:        world.Digest().String(),
		AttemptArtifactDigest:      attempt.ArtifactDigest().String(),
		CapturedObservationDigest:  observationDigest.String(),
		CapturePolicyDigest:        world.CapturePolicyDigest().String(),
		ProjectionDefinitionDigest: world.ProjectionDefinitionDigest().String(),
		ProjectionFingerprint:      fingerprint.String(),
		ProjectionDerivationDigest: derivation.Digest().String(),
		CanonicalProjection:        string(canonicalProjection),
	}
	resultDigest, _, err := canon.DigestTyped("ProjectionResult", resultIdentity)
	if err != nil {
		return StructuralCapture{}, &domain.Error{Code: "INVALID_PROJECTION_RESULT_DIGEST", Detail: err.Error()}
	}
	projectionDigest, err := domain.ParseDigest(resultDigest.String())
	if err != nil {
		return StructuralCapture{}, &domain.Error{Code: "INVALID_PROJECTION_RESULT_DIGEST", Detail: err.Error()}
	}
	return StructuralCapture{
		observationDigest:          observationDigest,
		projectionDigest:           projectionDigest,
		worldDigest:                world.Digest(),
		attemptDigest:              attempt.ArtifactDigest(),
		capturePolicyDigest:        world.CapturePolicyDigest(),
		projectionDefinitionDigest: world.ProjectionDefinitionDigest(),
		derivationDigest:           derivation.Digest(),
		fingerprint:                fingerprint,
		canonicalProjection:        append([]byte(nil), canonicalProjection...),
	}, nil
}

func (c StructuralCapture) ProjectionFingerprint() domain.ProjectionFingerprint { return c.fingerprint }

func (c StructuralCapture) ProjectionResultDigest() domain.Digest { return c.projectionDigest }

func (c StructuralCapture) ProjectionDerivationDigest() domain.Digest { return c.derivationDigest }

func (c StructuralCapture) CanonicalProjection() []byte {
	return append([]byte(nil), c.canonicalProjection...)
}

type trialKind string

const (
	trialCaptured   trialKind = "BEHAVIOR_CAPTURED"
	trialControlled trialKind = "CONTROL_INELIGIBLE"
)

// TrialFact is a tagged sum. Its fields are deliberately private: captured
// behavior and control ineligibility cannot coexist in one value.
type TrialFact struct {
	kind      trialKind
	world     domain.WorldInstance
	attempt   domain.FinalizedAttempt
	admission domain.AdmissionToken
	capture   StructuralCapture
	controls  []domain.ControlReason
	rejection ProjectionRejectionEvidence
}

func NewCapturedTrial(
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	admission domain.AdmissionToken,
	capture StructuralCapture,
) (TrialFact, error) {
	if err := validateTrialLineage(world, attempt, admission); err != nil {
		return TrialFact{}, err
	}
	if attempt.HasControls() {
		return TrialFact{}, &domain.Error{Code: "CAPTURED_TRIAL_HAS_CONTROL"}
	}
	if !capture.observationDigest.Valid() || !capture.projectionDigest.Valid() ||
		!capture.derivationDigest.Valid() || !capture.fingerprint.Valid() {
		return TrialFact{}, &domain.Error{Code: "INCOMPLETE_CAPTURE_PROVENANCE"}
	}
	if capture.worldDigest != world.Digest() || capture.attemptDigest != attempt.ArtifactDigest() ||
		capture.capturePolicyDigest != world.CapturePolicyDigest() ||
		capture.projectionDefinitionDigest != world.ProjectionDefinitionDigest() {
		return TrialFact{}, &domain.Error{Code: "CAPTURE_LINEAGE_MISMATCH"}
	}
	return TrialFact{kind: trialCaptured, world: world, attempt: attempt, admission: admission, capture: capture}, nil
}

func NewControlledTrial(world domain.WorldInstance, attempt domain.FinalizedAttempt, admission domain.AdmissionToken) (TrialFact, error) {
	if err := validateTrialLineage(world, attempt, admission); err != nil {
		return TrialFact{}, err
	}
	controls := finalizedControls(attempt)
	if len(controls) == 0 {
		return TrialFact{}, &domain.Error{Code: "CONTROL_TRIAL_HAS_NO_CONTROL"}
	}
	if containsControl(controls, domain.ControlEnvelopeRejected) {
		return TrialFact{}, &domain.Error{Code: "ENVELOPE_REJECTION_HAS_NO_ADMISSION_TOKEN"}
	}
	return TrialFact{
		kind:      trialControlled,
		world:     world,
		attempt:   attempt,
		admission: admission,
		controls:  append([]domain.ControlReason(nil), controls...),
	}, nil
}

// NewProjectionRejectedTrial records the one control that can arise only
// after an otherwise clean finalized process has crossed the typed adapter
// boundary. Projection rejection is not allowed to rewrite lifecycle history,
// and it is never represented by a projection fingerprint.
//
// Keeping this constructor narrower than a general caller-authored control
// prevents adapters from laundering timeouts, output limits, or teardown
// failures around the FinalizedAttempt authority.
func NewProjectionRejectedTrial(
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	admission domain.AdmissionToken,
	rejection ProjectionRejectionEvidence,
) (TrialFact, error) {
	if err := validateTrialLineage(world, attempt, admission); err != nil {
		return TrialFact{}, err
	}
	if attempt.HasControls() {
		return TrialFact{}, &domain.Error{Code: "PROJECTION_REJECTION_HAS_LIFECYCLE_CONTROL"}
	}
	if !rejection.Valid() || rejection.worldDigest != world.Digest() ||
		rejection.attemptArtifactDigest != attempt.ArtifactDigest() ||
		rejection.candidateKey != world.CandidateKey() ||
		rejection.capturePolicyDigest != world.CapturePolicyDigest() ||
		rejection.definitionDigest != world.ProjectionDefinitionDigest() {
		return TrialFact{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION_EVIDENCE"}
	}
	return TrialFact{
		kind:      trialControlled,
		world:     world,
		attempt:   attempt,
		admission: admission,
		controls:  []domain.ControlReason{domain.ControlProjectionRejected},
		rejection: rejection,
	}, nil
}

func validateTrialLineage(world domain.WorldInstance, attempt domain.FinalizedAttempt, admission domain.AdmissionToken) error {
	if !world.Digest().Valid() || !attempt.ArtifactDigest().Valid() ||
		world.AttemptArtifactDigest() != attempt.ArtifactDigest() ||
		world.Purpose() != attempt.Purpose() || !admission.Valid() ||
		admission.SubjectDigest() != world.Digest() || admission.PlanDigest() != world.PlanDigest() ||
		admission.StimulusDigest() != world.StimulusDigest() || admission.EnvelopeDigest() != world.EnvelopeDigest() ||
		admission.Purpose() != world.Purpose() || admission.RequiredFreshTrials() != world.RequiredFreshTrials() {
		return &domain.Error{Code: "TRIAL_LINEAGE_MISMATCH"}
	}
	return nil
}

func finalizedControls(attempt domain.FinalizedAttempt) []domain.ControlReason {
	result := make([]domain.ControlReason, 0, 3)
	if primary, ok := attempt.PrimaryControl(); ok {
		result = append(result, primary)
	}
	result = append(result, attempt.TeardownControls()...)
	return result
}

func containsControl(controls []domain.ControlReason, target domain.ControlReason) bool {
	for _, control := range controls {
		if control == target {
			return true
		}
	}
	return false
}

type Eligibility struct {
	eligible    bool
	fingerprint domain.ProjectionFingerprint
	reasons     []domain.ControlReason
}

func Eligible(fact TrialFact) Eligibility {
	if fact.kind == trialControlled {
		decision, err := eligibilitycore.Select(eligibilitycore.ControlIneligible, fact.controls)
		if err != nil {
			return Eligibility{reasons: []domain.ControlReason{domain.ControlProjectionRejected}}
		}
		return Eligibility{eligible: decision.IsEligible(), reasons: decision.Reasons()}
	}
	if !behaviorWasCaptured(fact.kind) || !fact.world.Digest().Valid() ||
		!fact.attempt.ArtifactDigest().Valid() || fact.attempt.HasControls() ||
		!fact.admission.Valid() || fact.admission.SubjectDigest() != fact.world.Digest() ||
		!fact.capture.observationDigest.Valid() ||
		!fact.capture.projectionDigest.Valid() || !fact.capture.fingerprint.Valid() {
		return Eligibility{reasons: []domain.ControlReason{domain.ControlProjectionRejected}}
	}
	decision, err := eligibilitycore.Select(eligibilitycore.BehaviorCaptured, nil)
	if err != nil || !decision.IsEligible() {
		return Eligibility{reasons: []domain.ControlReason{domain.ControlProjectionRejected}}
	}
	return Eligibility{eligible: true, fingerprint: fact.capture.fingerprint}
}

func behaviorWasCaptured(kind trialKind) bool {
	// MUTATION_ANCHOR: control-failure-must-remain-ineligible
	return kind == trialCaptured
}

func (e Eligibility) IsEligible() bool { return e.eligible }

func (e Eligibility) Fingerprint() (domain.ProjectionFingerprint, bool) {
	return e.fingerprint, e.eligible
}

func (e Eligibility) Reasons() []domain.ControlReason {
	if e.eligible {
		return nil
	}
	return append([]domain.ControlReason(nil), e.reasons...)
}

func (e Eligibility) Reason() (domain.ControlReason, bool) {
	if e.eligible || len(e.reasons) == 0 {
		return "", false
	}
	return e.reasons[0], true
}

type BatchStatus string

const (
	ObservedStable BatchStatus = "OBSERVED_STABLE"
	Unstable       BatchStatus = "UNSTABLE"
	Uncomparable   BatchStatus = "UNCOMPARABLE"
	Incomplete     BatchStatus = "INCOMPLETE"
)

type HistogramBin struct {
	Fingerprint domain.ProjectionFingerprint
	Count       int
}

type Classification struct {
	status         BatchStatus
	eligibleTrials int
	requiredTrials int
	fingerprint    domain.ProjectionFingerprint
	histogram      []HistogramBin
	reasons        []domain.ControlReason
}

func (c Classification) Status() BatchStatus { return c.status }
func (c Classification) EligibleTrials() int { return c.eligibleTrials }
func (c Classification) RequiredTrials() int { return c.requiredTrials }

func (c Classification) Fingerprint() (domain.ProjectionFingerprint, bool) {
	return c.fingerprint, c.status == ObservedStable
}

func (c Classification) Histogram() []HistogramBin {
	return append([]HistogramBin(nil), c.histogram...)
}

func (c Classification) Reasons() []domain.ControlReason {
	return append([]domain.ControlReason(nil), c.reasons...)
}

type BatchInput struct {
	Trials []TrialFact
}

// ScheduledBatchInput is the U3 authority-bearing classifier input. Unlike
// legacy BatchInput, it proves the exact canonical rotation across the full
// candidate roster and records whether orchestration ended before its declared
// repetition count. CandidateKey keeps incomplete runs explicit even when a
// future schema permits zero-trial batch artifacts.
type ScheduledBatchInput struct {
	Schedule      RotatedSchedule
	CandidateKey  domain.CandidateExecutionKey
	Trials        []TrialFact
	RunIncomplete bool
}

type StableBatch struct {
	digest                     domain.Digest
	candidateKey               domain.CandidateExecutionKey
	planDigest                 domain.Digest
	stimulusDigest             domain.Digest
	comparisonEnvelopeDigest   domain.Digest
	capturePolicyDigest        domain.Digest
	projectionDefinitionDigest domain.Digest
	comparisonAdmissionDigests []domain.Digest
	comparisonBasisDigest      domain.Digest
	comparisonAdmissionRoster  []domain.CandidateExecutionKey
	requiredFreshTrials        int
	phase                      domain.AttemptPurpose
	scheduleDigest             domain.Digest
	rotation                   string
	scheduleStartOffset        int
	scheduleOrdinals           []int
	trials                     []TrialFact
	observationDigests         []domain.Digest
	classification             Classification
	canonicalBytes             []byte
}

func Classify(input BatchInput) (StableBatch, error) {
	return classifyBatch(input.Trials, domain.Digest(""), "NOT_ESTABLISHED_IN_U1", 0, false)
}

// ClassifyScheduled is the only U3 batch constructor. It proves that every
// supplied trial occupies this candidate's slot in the exact canonical rotated
// schedule. A run-level stop remains INCOMPLETE when the admitted prefix only
// agrees; observed agreement is not promoted to stability. Two eligible
// fingerprints, however, establish UNSTABLE before that stop.
func ClassifyScheduled(input ScheduledBatchInput) (StableBatch, error) {
	if !input.Schedule.Valid() || !input.CandidateKey.Valid() {
		return StableBatch{}, &domain.Error{Code: "INVALID_SCHEDULED_BATCH_AUTHORITY"}
	}
	found := false
	for _, candidate := range input.Schedule.roster {
		if candidate == input.CandidateKey {
			found = true
			break
		}
	}
	if !found {
		return StableBatch{}, &domain.Error{Code: "SCHEDULED_CANDIDATE_NOT_IN_ROSTER"}
	}
	if len(input.Trials) == 0 {
		return StableBatch{}, &domain.Error{Code: "MISSING_BATCH_EVIDENCE"}
	}
	if len(input.Trials) > input.Schedule.repetitions || (!input.RunIncomplete && len(input.Trials) != input.Schedule.repetitions) {
		return StableBatch{}, &domain.Error{Code: "SCHEDULED_BATCH_REPEAT_MISMATCH"}
	}
	if input.Trials[0].world.RequiredFreshTrials() != input.Schedule.repetitions {
		return StableBatch{}, &domain.Error{Code: "SCHEDULED_BATCH_REPEAT_AUTHORITY_MISMATCH"}
	}
	for repetition, trial := range input.Trials {
		slot, ok := input.Schedule.Slot(input.CandidateKey, repetition)
		if !ok || trial.world.CandidateKey() != input.CandidateKey || trial.world.ScheduleOrdinal() != slot.ordinal {
			return StableBatch{}, &domain.Error{Code: "SCHEDULED_BATCH_SLOT_MISMATCH"}
		}
	}
	return classifyBatch(
		input.Trials,
		input.Schedule.digest,
		string(input.Schedule.Rotation()),
		input.Schedule.startOffset,
		input.RunIncomplete,
	)
}

func classifyBatch(
	trials []TrialFact,
	scheduleDigest domain.Digest,
	rotation string,
	scheduleStartOffset int,
	forceIncomplete bool,
) (StableBatch, error) {
	if len(trials) == 0 {
		return StableBatch{}, &domain.Error{Code: "MISSING_BATCH_EVIDENCE"}
	}
	first := trials[0]
	candidateKey := first.world.CandidateKey()
	planDigest := first.world.PlanDigest()
	stimulusDigest := first.world.StimulusDigest()
	envelopeDigest := first.world.EnvelopeDigest()
	capturePolicyDigest := first.world.CapturePolicyDigest()
	projectionDefinitionDigest := first.world.ProjectionDefinitionDigest()
	basisDigest := first.admission.ComparisonBasisDigest()
	requiredFreshTrials := first.world.RequiredFreshTrials()
	phase := first.world.Purpose()
	if !candidateKey.Valid() || !planDigest.Valid() || !stimulusDigest.Valid() || !envelopeDigest.Valid() ||
		!capturePolicyDigest.Valid() || !projectionDefinitionDigest.Valid() ||
		!basisDigest.Valid() || !phase.Valid() {
		return StableBatch{}, &domain.Error{Code: "INVALID_BATCH_IDENTITY", Detail: "trial-derived identity is incomplete"}
	}
	if requiredFreshTrials < 1 || requiredFreshTrials > 5 {
		return StableBatch{}, &domain.Error{Code: "INVALID_REPEAT_COUNT", Detail: "plan-derived required trials outside 1..5"}
	}
	if len(trials) > requiredFreshTrials {
		return StableBatch{}, &domain.Error{Code: "EXTRA_TRIALS_REQUIRE_NEW_BATCH"}
	}
	scheduleOrdinals := make([]int, len(trials))
	seenOrdinals := map[int]struct{}{}
	for index, trial := range trials {
		ordinal := trial.world.ScheduleOrdinal()
		if ordinal < 0 {
			return StableBatch{}, &domain.Error{Code: "NEGATIVE_SCHEDULE_ORDINAL"}
		}
		if _, duplicate := seenOrdinals[ordinal]; duplicate {
			return StableBatch{}, &domain.Error{Code: "DUPLICATE_SCHEDULE_ORDINAL"}
		}
		if index > 0 && ordinal <= scheduleOrdinals[index-1] {
			return StableBatch{}, &domain.Error{Code: "NONCANONICAL_SCHEDULE_ORDER"}
		}
		seenOrdinals[ordinal] = struct{}{}
		scheduleOrdinals[index] = ordinal
	}

	seenAttempts := map[domain.Digest]struct{}{}
	seenWorlds := map[domain.Digest]struct{}{}
	seenAdmissions := map[domain.Digest]struct{}{}
	seenObservations := map[domain.Digest]struct{}{}
	observationDigests := make([]domain.Digest, 0, len(trials))
	admissionDigests := make([]domain.Digest, len(trials))
	counts := map[domain.ProjectionFingerprint]int{}
	reasonSet := map[domain.ControlReason]struct{}{}
	eligibleCount := 0
	var admissionRoster []domain.CandidateExecutionKey
	for index, trial := range trials {
		world := trial.world
		attemptDigest := trial.attempt.ArtifactDigest()
		if !world.Digest().Valid() || !attemptDigest.Valid() {
			return StableBatch{}, &domain.Error{Code: "INVALID_TRIAL_IDENTITY"}
		}
		// MUTANT_U1_OBSERVE_IGNORE_STIMULUS_BINDING: stimulus equality is authority-bearing.
		if world.CandidateKey() != candidateKey || world.PlanDigest() != planDigest ||
			world.StimulusDigest() != stimulusDigest || world.EnvelopeDigest() != envelopeDigest ||
			world.CapturePolicyDigest() != capturePolicyDigest || world.ProjectionDefinitionDigest() != projectionDefinitionDigest ||
			world.Purpose() != phase || world.RequiredFreshTrials() != requiredFreshTrials ||
			trial.admission.EnvelopeDigest() != envelopeDigest || trial.admission.PlanDigest() != planDigest ||
			trial.admission.StimulusDigest() != stimulusDigest || trial.admission.Purpose() != phase ||
			trial.admission.RequiredFreshTrials() != requiredFreshTrials ||
			trial.admission.ComparisonBasisDigest() != basisDigest ||
			world.AttemptArtifactDigest() != attemptDigest {
			return StableBatch{}, &domain.Error{Code: "TRIAL_BATCH_LINEAGE_MISMATCH"}
		}
		trialRoster := trial.admission.CandidateRoster()
		if index == 0 {
			admissionRoster = trialRoster
			if !candidateRosterContains(admissionRoster, candidateKey) {
				return StableBatch{}, &domain.Error{Code: "CANDIDATE_NOT_IN_COMPARISON_ADMISSION"}
			}
		} else if !sameCandidateRoster(admissionRoster, trialRoster) {
			return StableBatch{}, &domain.Error{Code: "MIXED_COMPARISON_ADMISSION_ROSTER"}
		}
		if _, duplicate := seenAttempts[attemptDigest]; duplicate {
			return StableBatch{}, &domain.Error{Code: "REUSED_ATTEMPT_EVIDENCE"}
		}
		seenAttempts[attemptDigest] = struct{}{}
		if _, duplicate := seenWorlds[world.Digest()]; duplicate {
			return StableBatch{}, &domain.Error{Code: "REUSED_WORLD_INSTANCE"}
		}
		seenWorlds[world.Digest()] = struct{}{}
		admissionDigest := trial.admission.ComparisonAdmissionDigest()
		if !admissionDigest.Valid() {
			return StableBatch{}, &domain.Error{Code: "INVALID_COMPARISON_ADMISSION_DIGEST"}
		}
		if _, duplicate := seenAdmissions[admissionDigest]; duplicate {
			return StableBatch{}, &domain.Error{Code: "REUSED_COMPARISON_ADMISSION"}
		}
		seenAdmissions[admissionDigest] = struct{}{}
		admissionDigests[index] = admissionDigest
		if trial.kind == trialCaptured || trial.rejection.Valid() {
			observationDigest := trial.capture.observationDigest
			if trial.rejection.Valid() {
				observationDigest = trial.rejection.observationDigest
			}
			if !observationDigest.Valid() {
				return StableBatch{}, &domain.Error{Code: "INVALID_CAPTURED_OBSERVATION_DIGEST"}
			}
			if _, duplicate := seenObservations[observationDigest]; duplicate { // MUTANT_U1_OBSERVE_ALLOW_REUSED_CAPTURE
				return StableBatch{}, &domain.Error{Code: "REUSED_CAPTURED_OBSERVATION"}
			}
			seenObservations[observationDigest] = struct{}{}
			observationDigests = append(observationDigests, observationDigest)
		}
		eligibility := Eligible(trial)
		if !eligibility.IsEligible() {
			for _, reason := range eligibility.Reasons() {
				if !reason.Valid() {
					return StableBatch{}, &domain.Error{Code: "INVALID_TRIAL_CONTROL"}
				}
				reasonSet[reason] = struct{}{}
			}
			continue
		}
		fingerprint, _ := eligibility.Fingerprint()
		counts[fingerprint]++
		eligibleCount++
	}

	classification := Classification{eligibleTrials: eligibleCount, requiredTrials: requiredFreshTrials}
	if len(reasonSet) > 0 {
		classification.status = Uncomparable
		classification.reasons = sortedReasons(reasonSet)
	} else if len(counts) >= 2 {
		// Once two eligible fingerprints exist, instability is established
		// evidence. A later orchestration stop can prevent stability from being
		// established, but cannot erase an observed disagreement.
		classification.status = Unstable
		classification.histogram = sortedHistogram(counts)
	} else if forceIncomplete {
		classification.status = Incomplete
		classification.reasons = []domain.ControlReason{domain.ControlBudgetExhausted}
	} else if eligibleCount < requiredFreshTrials {
		classification.status = Incomplete
		classification.reasons = []domain.ControlReason{domain.ControlBudgetExhausted}
	} else if eligibleCount == requiredFreshTrials && len(counts) == 1 {
		classification.status = ObservedStable
		for fingerprint := range counts {
			classification.fingerprint = fingerprint
		}
	} else {
		return StableBatch{}, &domain.Error{Code: "INVALID_BATCH_CLASSIFICATION"}
	}

	admissionDigestStrings := make([]string, len(admissionDigests))
	for index, digest := range admissionDigests {
		admissionDigestStrings[index] = digest.String()
	}
	identity := batchIdentity{
		CandidateExecutionKey:      candidateKey.String(),
		WorldPlanDigest:            planDigest.String(),
		StimulusDigest:             stimulusDigest.String(),
		ComparisonEnvelopeDigest:   envelopeDigest.String(),
		CapturePolicyDigest:        capturePolicyDigest.String(),
		ProjectionDefinitionDigest: projectionDefinitionDigest.String(),
		ComparisonAdmissionDigests: admissionDigestStrings,
		ComparisonBasisDigest:      basisDigest.String(),
		ComparisonAdmissionRoster:  candidateRosterStrings(admissionRoster),
		RequiredFreshTrials:        requiredFreshTrials,
		Schedule: scheduleIdentity{
			Phase:          string(phase),
			Rotation:       rotation,
			StartOffset:    scheduleStartOffset,
			ScheduleDigest: scheduleDigest.String(),
			Ordinals:       append([]int(nil), scheduleOrdinals...),
		},
		Trials:                       make([]trialIdentity, len(trials)),
		CapturedObservationDigests:   digestStrings(observationDigests),
		Classification:               classificationIdentity(classification),
		DuplicateEvidenceWithinBatch: false,
	}
	for index, trial := range trials {
		identity.Trials[index] = trialIdentityOf(trial)
	}
	digest, canonicalBytes, err := digestBatch(identity)
	if err != nil {
		return StableBatch{}, err
	}
	return StableBatch{
		digest:                     digest,
		candidateKey:               candidateKey,
		planDigest:                 planDigest,
		stimulusDigest:             stimulusDigest,
		comparisonEnvelopeDigest:   envelopeDigest,
		capturePolicyDigest:        capturePolicyDigest,
		projectionDefinitionDigest: projectionDefinitionDigest,
		comparisonAdmissionDigests: append([]domain.Digest(nil), admissionDigests...),
		comparisonBasisDigest:      basisDigest,
		comparisonAdmissionRoster:  append([]domain.CandidateExecutionKey(nil), admissionRoster...),
		requiredFreshTrials:        requiredFreshTrials,
		phase:                      phase,
		scheduleDigest:             scheduleDigest,
		rotation:                   rotation,
		scheduleStartOffset:        scheduleStartOffset,
		scheduleOrdinals:           append([]int(nil), scheduleOrdinals...),
		trials:                     append([]TrialFact(nil), trials...),
		observationDigests:         append([]domain.Digest(nil), observationDigests...),
		classification:             classification,
		canonicalBytes:             canonicalBytes,
	}, nil
}

func sortedReasons(set map[domain.ControlReason]struct{}) []domain.ControlReason {
	result := make([]domain.ControlReason, 0, len(set))
	for reason := range set {
		result = append(result, reason)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func candidateRosterContains(roster []domain.CandidateExecutionKey, target domain.CandidateExecutionKey) bool {
	for _, candidate := range roster {
		if candidate == target {
			return true
		}
	}
	return false
}

func sameCandidateRoster(left, right []domain.CandidateExecutionKey) bool {
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

func candidateRosterStrings(roster []domain.CandidateExecutionKey) []string {
	result := make([]string, len(roster))
	for index, candidate := range roster {
		result[index] = candidate.String()
	}
	return result
}

func sortedHistogram(counts map[domain.ProjectionFingerprint]int) []HistogramBin {
	result := make([]HistogramBin, 0, len(counts))
	for fingerprint, count := range counts {
		result = append(result, HistogramBin{Fingerprint: fingerprint, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Fingerprint.String() < result[j].Fingerprint.String()
	})
	return result
}

func (b StableBatch) Digest() domain.Digest                      { return b.digest }
func (b StableBatch) CandidateKey() domain.CandidateExecutionKey { return b.candidateKey }
func (b StableBatch) PlanDigest() domain.Digest                  { return b.planDigest }
func (b StableBatch) StimulusDigest() domain.Digest              { return b.stimulusDigest }
func (b StableBatch) EnvelopeDigest() domain.Digest              { return b.comparisonEnvelopeDigest }
func (b StableBatch) CapturePolicyDigest() domain.Digest         { return b.capturePolicyDigest }
func (b StableBatch) ProjectionDefinitionDigest() domain.Digest  { return b.projectionDefinitionDigest }
func (b StableBatch) ComparisonBasisDigest() domain.Digest       { return b.comparisonBasisDigest }
func (b StableBatch) AdmissionDigests() []domain.Digest {
	return append([]domain.Digest(nil), b.comparisonAdmissionDigests...)
}
func (b StableBatch) AdmissionRoster() []domain.CandidateExecutionKey {
	return append([]domain.CandidateExecutionKey(nil), b.comparisonAdmissionRoster...)
}
func (b StableBatch) Phase() domain.AttemptPurpose   { return b.phase }
func (b StableBatch) ScheduleDigest() domain.Digest  { return b.scheduleDigest }
func (b StableBatch) Rotation() string               { return b.rotation }
func (b StableBatch) ScheduleStartOffset() int       { return b.scheduleStartOffset }
func (b StableBatch) RequiredFreshTrials() int       { return b.requiredFreshTrials }
func (b StableBatch) Classification() Classification { return b.classification }
func (b StableBatch) Trials() []TrialFact            { return append([]TrialFact(nil), b.trials...) }

func (b StableBatch) ScheduleOrdinals() []int {
	return append([]int(nil), b.scheduleOrdinals...)
}

func (b StableBatch) AttemptDigests() []domain.Digest {
	result := make([]domain.Digest, len(b.trials))
	for index, trial := range b.trials {
		result[index] = trial.attempt.ArtifactDigest()
	}
	return result
}

func (b StableBatch) WorldDigests() []domain.Digest {
	result := make([]domain.Digest, len(b.trials))
	for index, trial := range b.trials {
		result[index] = trial.world.Digest()
	}
	return result
}

func (b StableBatch) ObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), b.observationDigests...)
}

func digestStrings(digests []domain.Digest) []string {
	result := make([]string, len(digests))
	for index, digest := range digests {
		result[index] = digest.String()
	}
	return result
}

func (b StableBatch) CanonicalBytes() []byte {
	return append([]byte(nil), b.canonicalBytes...)
}

func (c Classification) BoundedLabel() string {
	if c.status != ObservedStable {
		return ""
	}
	return fmt.Sprintf("OBSERVED_STABLE(%d/%d,%s)", c.eligibleTrials, c.requiredTrials, c.fingerprint.String())
}
