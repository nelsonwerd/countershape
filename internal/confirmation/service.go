// Package confirmation owns the live, physical transition from a minimized
// divergent witness to durable FreshConfirmation evidence.
package confirmation

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	confirmationauthority "github.com/nelsonwerd/countershape/internal/confirmation/authority"
	confirmationpublication "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/world"
)

const (
	challengeBytes          = 32
	maxProjectionProofBytes = 512 * 1024
	confirmationScope       = "EXACT_MINIMIZED_WITNESS_AND_COMPLETE_LABELED_OUTCOME_MAP_ONLY"
	invocationTrustNonclaim = "CANDIDATE_WRITTEN_FIXTURE_EVIDENCE_IS_NOT_HOSTILE_PROCESS_ATTESTATION"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

// TrialRequest is issued by the orchestrator for one exact scheduled slot.
// Callers cannot author its nonce or change its CONFIRMATION purpose.
type TrialRequest struct {
	slot          observe.ScheduledTrial
	instanceNonce string
}

func (r TrialRequest) Slot() observe.ScheduledTrial   { return r.slot }
func (r TrialRequest) InstanceNonce() string          { return r.instanceNonce }
func (r TrialRequest) Purpose() domain.AttemptPurpose { return domain.AttemptConfirmation }

// TrialExecutor must cross a concrete world adapter and return the exact
// PreparedTrial derived from that same opaque result. Run binds the pair.
type TrialExecutor func(context.Context, TrialRequest) (world.Result, observe.PreparedTrial, error)

type Request struct {
	Plan            domain.WorldPlan
	Envelope        domain.ComparisonEnvelope
	ReducedBaseline compare.DivergentBaseline
	ReductionRun    reduce.ReductionRun
	ReductionResult reduction.Result
	WallBudget      time.Duration
	Execute         TrialExecutor
}

type executionSeal struct{ marker byte }

// ProjectionProof is an exact, confirmation-derived projection retained for
// durable Choicepoint promotion. It is issued only from executed PreparedTrial
// links and is independently rechecked by the strict confirmation parser.
type ProjectionProof struct {
	candidate   domain.CandidateExecutionKey
	fingerprint domain.ProjectionFingerprint
	canonical   []byte
}

func (p ProjectionProof) CandidateExecutionKey() domain.CandidateExecutionKey { return p.candidate }
func (p ProjectionProof) ProjectionFingerprint() domain.ProjectionFingerprint { return p.fingerprint }
func (p ProjectionProof) CanonicalProjection() []byte                         { return append([]byte(nil), p.canonical...) }

func (p ProjectionProof) Valid() bool {
	computed, err := domain.NewProjectionFingerprint(p.canonical)
	return p.candidate.Valid() && p.fingerprint.Valid() && err == nil && computed == p.fingerprint
}

// Draft is live freshness authority. Its canonical bytes can be parsed into an
// inert Record, but serialization cannot reconstruct this private seal.
type Draft struct {
	digest         domain.Digest
	canonicalBytes []byte
	record         Record
	confirmed      compare.ConfirmedOutcomeMap
	reduced        compare.DivergentBaseline
	assessment     compare.PreservationAssessment
	facts          []world.FreshExecutionFact
	publication    confirmationauthority.Publication
	liveSeal       *executionSeal
}

type Completed struct {
	draft      Draft
	confirmed  compare.ConfirmedOutcomeMap
	assessment compare.PreservationAssessment
}

func (c Completed) Valid() bool {
	return c.draft.Valid() && c.confirmed.Valid() && c.assessment.Valid() &&
		c.draft.confirmed.OutcomeMap().ArtifactDigest() == c.confirmed.OutcomeMap().ArtifactDigest() &&
		c.draft.assessment.ObservedArtifactDigest() == c.assessment.ObservedArtifactDigest()
}

func (c Completed) Draft() Draft                                     { return cloneDraft(c.draft) }
func (c Completed) ConfirmedOutcomeMap() compare.ConfirmedOutcomeMap { return c.confirmed }
func (c Completed) Assessment() compare.PreservationAssessment       { return c.assessment }

func Run(ctx context.Context, request Request) (Completed, error) {
	return runWithEntropy(ctx, request, rand.Reader)
}

func runWithEntropy(ctx context.Context, request Request, entropy io.Reader) (Completed, error) {
	if err := validateRequestBeforeExecution(ctx, request); err != nil {
		return Completed{}, err
	}
	challenge := make([]byte, challengeBytes)
	if _, err := io.ReadFull(entropy, challenge); err != nil {
		return Completed{}, refuse("CONFIRMATION_ENTROPY_UNAVAILABLE", "run challenge could not be generated", err)
	}
	challengeRaw, err := canon.DigestBytes("ConfirmationRunChallenge", challenge)
	if err != nil {
		return Completed{}, err
	}
	challengeDigest, err := domain.ParseDigest(challengeRaw.String())
	if err != nil {
		return Completed{}, err
	}

	reducedMap := request.ReducedBaseline.OutcomeMap()
	roster := reducedMap.CandidateRoster()
	repetitions := request.Plan.RepeatSchedule().ConfirmationRepeats
	facts := make([]world.FreshExecutionFact, 0, len(roster)*repetitions)
	links := make([]observe.PreparedExecutionLink, 0, len(roster)*repetitions)
	seenPhysical := newPhysicalLedger()
	run, err := observe.RunObservation(ctx, observe.ObservationConfig{
		Plan: request.Plan, Purpose: domain.AttemptConfirmation, Envelope: request.Envelope,
		CandidateRoster: roster, Repetitions: repetitions,
		Budget: observe.TrialBudget{MaxTotalTrials: len(roster) * repetitions, WallBudget: request.WallBudget},
	}, func(trialContext context.Context, slot observe.ScheduledTrial) (observe.PreparedTrial, error) {
		nonce, nonceErr := confirmationNonce(challengeDigest, slot.Ordinal())
		if nonceErr != nil {
			return observe.PreparedTrial{}, nonceErr
		}
		result, prepared, executeErr := request.Execute(trialContext, TrialRequest{slot: slot, instanceNonce: nonce})
		if executeErr != nil {
			return observe.PreparedTrial{}, executeErr
		}
		fact, factErr := result.FreshConfirmationEvidence()
		if factErr != nil {
			return observe.PreparedTrial{}, refuse("PHYSICAL_CONFIRMATION_REFUSED", "world result did not prove the confirmation execution profile", factErr)
		}
		if fact.PlanDigest() != request.Plan.Digest() || fact.CandidateKey() != slot.CandidateKey() ||
			fact.StimulusDigest() != reducedMap.StimulusDigest() || fact.Purpose() != domain.AttemptConfirmation ||
			fact.InstanceNonce() != nonce || fact.ScheduleOrdinal() != slot.Ordinal() {
			return observe.PreparedTrial{}, refuse("CONFIRMATION_EXECUTION_BINDING_MISMATCH", "physical fact disagrees with its service-issued slot", nil)
		}
		link, linkErr := prepared.BindExecuted(result)
		if linkErr != nil {
			return observe.PreparedTrial{}, refuse("PREPARED_EXECUTION_BINDING_MISMATCH", "prepared trial was not derived from the returned world", linkErr)
		}
		if !link.Valid() || link.WorldDigest() != fact.WorldDigest() || link.AttemptDigest() != fact.AttemptArtifactDigest() {
			return observe.PreparedTrial{}, refuse("PREPARED_EXECUTION_BINDING_MISMATCH", "prepared link disagrees with physical fact", nil)
		}
		if err := seenPhysical.admit(fact); err != nil {
			return observe.PreparedTrial{}, err
		}
		facts = append(facts, fact)
		links = append(links, link)
		return prepared, nil
	})
	if err != nil {
		return Completed{}, err
	}
	if run.Status() != observe.ObservationComplete || run.CompletedMatrices() != repetitions {
		return Completed{}, refuse("CONFIRMATION_INCOMPLETE", "the complete confirmation matrix did not finish", nil)
	}
	mapInput, present := run.OutcomeMapInput()
	if !present {
		return Completed{}, refuse("CONFIRMATION_MAP_ABSENT", "complete observation exposed no map input", nil)
	}
	confirmedMap, err := compare.NewCandidateOutcomeMap(
		mapInput.StimulusDigest(), mapInput.EnvelopeDigest(), mapInput.Roster(), mapInput.Batches(),
	)
	if err != nil {
		return Completed{}, err
	}
	if confirmedMap.Phase() != domain.AttemptConfirmation || confirmedMap.ScheduleStartOffset() != 1 ||
		confirmedMap.ScheduleDigest() == reducedMap.ScheduleDigest() {
		return Completed{}, refuse("CONFIRMATION_SCHEDULE_NOT_PHASE_DISTINCT", "confirmation did not use its phase-bound rotated schedule", nil)
	}
	confirmed, err := compare.RequireConfirmedOutcomeMap(confirmedMap)
	if err != nil {
		return Completed{}, err
	}
	assessment, err := requireExactLabeledConfirmation(reducedMap, confirmedMap)
	if err != nil {
		return Completed{}, err
	}
	if err := requireExactPhysicalCoverage(confirmedMap, facts, links); err != nil {
		return Completed{}, err
	}
	proofs, err := collectProjectionProofs(confirmed, links, repetitions)
	if err != nil {
		return Completed{}, err
	}
	priorLedger, err := buildPriorEvidenceLedger(request.ReductionRun, reducedMap)
	if err != nil {
		return Completed{}, err
	}
	if err := priorLedger.rejectMap(confirmedMap); err != nil {
		return Completed{}, err
	}
	draft, err := newDraft(request, challengeDigest, priorLedger.digest, confirmed, assessment, facts, proofs)
	if err != nil {
		return Completed{}, err
	}
	return Completed{draft: draft, confirmed: confirmed, assessment: assessment}, nil
}

func collectProjectionProofs(
	confirmed compare.ConfirmedOutcomeMap,
	links []observe.PreparedExecutionLink,
	repetitions int,
) ([]ProjectionProof, error) {
	if !confirmed.Valid() || repetitions < 1 {
		return nil, refuse("INVALID_CONFIRMATION_PROJECTION_PROOFS", "confirmed map or repetition count is invalid", nil)
	}
	entries := confirmed.ProjectionRoster().Entries()
	proofs := make([]ProjectionProof, 0, len(entries))
	retained := 0
	for _, entry := range entries {
		var proof ProjectionProof
		matches := 0
		for _, link := range links {
			if link.CandidateExecutionKey() != entry.CandidateExecutionKey() {
				continue
			}
			fingerprint, canonicalProjection, present := link.Projection()
			if !present || fingerprint != entry.ProjectionFingerprint() {
				return nil, refuse("INVALID_CONFIRMATION_PROJECTION_PROOFS", "eligible candidate lacks its exact stable projection", nil)
			}
			candidateProof := ProjectionProof{candidate: entry.CandidateExecutionKey(), fingerprint: fingerprint, canonical: canonicalProjection}
			if !candidateProof.Valid() {
				return nil, refuse("INVALID_CONFIRMATION_PROJECTION_PROOFS", "projection proof failed exact fingerprint reconstruction", nil)
			}
			if matches == 0 {
				proof = candidateProof
			} else if proof.fingerprint != candidateProof.fingerprint || !bytes.Equal(proof.canonical, candidateProof.canonical) {
				return nil, refuse("INVALID_CONFIRMATION_PROJECTION_PROOFS", "stable candidate projection bytes changed across confirmation repeats", nil)
			}
			matches++
		}
		if matches != repetitions {
			return nil, refuse("INVALID_CONFIRMATION_PROJECTION_PROOFS", "projection proof matrix is incomplete", nil)
		}
		retained += len(proof.canonical)
		if retained > maxProjectionProofBytes {
			return nil, refuse("CONFIRMATION_RESOURCE_LIMIT", "aggregate canonical projections exceed the durable v1 proof budget", nil)
		}
		proofs = append(proofs, proof)
	}
	return proofs, nil
}

func validateRequestBeforeExecution(ctx context.Context, request Request) error {
	if ctx == nil || ctx.Err() != nil || !request.Plan.Digest().Valid() || len(request.Plan.CanonicalBytes()) == 0 ||
		!request.Envelope.Digest().Valid() || request.Plan.ComparisonEnvelopeDigest() != request.Envelope.Digest() ||
		!request.ReducedBaseline.Valid() || !request.ReductionRun.Valid() || !request.ReductionResult.Valid() ||
		request.WallBudget <= 0 || request.Execute == nil {
		return refuse("INVALID_CONFIRMATION_REQUEST", "request lacks live plan, lineage, budget, or executor authority", ctxErr(ctx))
	}
	run := request.ReductionRun
	result := request.ReductionResult
	original := run.Baseline().OutcomeMap()
	reduced := request.ReducedBaseline.OutcomeMap()
	if original.ArtifactDigest() != run.Baseline().OutcomeMap().ArtifactDigest() ||
		original.PlanDigest() != request.Plan.Digest() || reduced.PlanDigest() != request.Plan.Digest() ||
		original.EnvelopeDigest() != request.Envelope.Digest() || reduced.EnvelopeDigest() != request.Envelope.Digest() ||
		reduced.StimulusDigest() != run.MinimizedStimulusDigest() ||
		result.RunDigest() != run.Digest() || result.TranscriptDigest() != run.Transcript().Digest() ||
		result.Grade().RunDigest() != run.Digest() {
		return refuse("CONFIRMATION_REDUCTION_LINEAGE_MISMATCH", "reduced baseline, live run, grade, plan, or envelope disagree", nil)
	}
	record, err := reduce.ParseReductionRunRecord(run.CanonicalBytes(), run.Transcript().CanonicalBytes())
	if err != nil || record.Digest() != run.Digest() || record.Transcript().Digest() != run.Transcript().Digest() {
		return refuse("CONFIRMATION_REDUCTION_LINEAGE_MISMATCH", "live reduction run failed its strict wire parser", err)
	}
	if err := bindReducedMapToRun(run, reduced); err != nil {
		return err
	}
	return nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func bindReducedMapToRun(run reduce.ReductionRun, reduced compare.CandidateOutcomeMap) error {
	accepted := run.Transcript().AcceptedPath()
	if len(accepted) == 0 {
		if reduced.ArtifactDigest() != run.Baseline().OutcomeMap().ArtifactDigest() {
			return refuse("REDUCED_BASELINE_NOT_EARNED", "unchanged run requires the original exact baseline map", nil)
		}
		return nil
	}
	last := accepted[len(accepted)-1]
	matches := 0
	for _, entry := range run.Transcript().Entries() {
		evaluation := entry.Evaluation()
		if evaluation.Neighbor().Digest() != last {
			continue
		}
		observedMap, hasMap := evaluation.ObservedOutcomeMapDigest()
		observedPreservation, hasPreservation := evaluation.ObservedPreservationDigest()
		if evaluation.Decision() == reduce.Preserves && hasMap && hasPreservation &&
			observedMap == reduced.ArtifactDigest() && observedPreservation == reduced.PreservationDigest() &&
			evaluation.Neighbor().StimulusDigest() == run.MinimizedStimulusDigest() {
			matches++
		}
	}
	if matches != 1 {
		return refuse("REDUCED_BASELINE_NOT_EARNED", "minimized map is not the unique last accepted preserving evaluation", nil)
	}
	return nil
}

func confirmationNonce(challenge domain.Digest, ordinal int) (string, error) {
	if !challenge.Valid() || ordinal < 0 {
		return "", refuse("INVALID_CONFIRMATION_NONCE_INPUT", "challenge or ordinal is invalid", nil)
	}
	digest, _, err := canon.DigestTyped("ConfirmationInstanceNonce", struct {
		SchemaVersion   string `json:"schema_version"`
		Kind            string `json:"kind"`
		ChallengeDigest string `json:"challenge_digest"`
		Ordinal         int    `json:"schedule_ordinal"`
	}{domain.SchemaVersion, "ConfirmationInstanceNonce", challenge.String(), ordinal})
	if err != nil {
		return "", err
	}
	return "u6-confirmation:" + strings.TrimPrefix(digest.String(), "sha256:"), nil
}

func requireExactLabeledConfirmation(
	reduced compare.CandidateOutcomeMap,
	confirmed compare.CandidateOutcomeMap,
) (compare.PreservationAssessment, error) {
	assessment := compare.AssessPreservation(reduced, confirmed) // MUTANT_U6_CONFIRM_PARTITION_SHAPE
	if !assessment.Valid() || assessment.Relation() != compare.PreservationEqual {
		return compare.PreservationAssessment{}, refuse(
			"CONFIRMATION_DOES_NOT_PRESERVE_EXACT_MAP", assessment.ReasonCode(), nil,
		)
	}
	return assessment, nil
}

type physicalEvidenceIDs struct {
	attempt, world, process, root, invocation, file domain.Digest
}

type physicalLedger struct {
	attempt    map[domain.Digest]struct{}
	world      map[domain.Digest]struct{}
	process    map[domain.Digest]struct{}
	root       map[domain.Digest]struct{}
	invocation map[domain.Digest]struct{}
	file       map[domain.Digest]struct{}
}

func newPhysicalLedger() physicalLedger {
	return physicalLedger{
		attempt: map[domain.Digest]struct{}{}, world: map[domain.Digest]struct{}{},
		process: map[domain.Digest]struct{}{}, root: map[domain.Digest]struct{}{},
		invocation: map[domain.Digest]struct{}{}, file: map[domain.Digest]struct{}{},
	}
}

func (l physicalLedger) admit(fact world.FreshExecutionFact) error {
	return l.admitIDs(physicalEvidenceIDs{
		attempt: fact.AttemptArtifactDigest(), world: fact.WorldDigest(), process: fact.ProcessDigest(),
		root: fact.RootLayoutDigest(), invocation: fact.InvocationReceiptDigest(), file: fact.InvocationFileDigest(),
	})
}

func (l physicalLedger) admitIDs(ids physicalEvidenceIDs) error {
	sets := []struct {
		name   string
		set    map[domain.Digest]struct{}
		digest domain.Digest
	}{
		{"attempt", l.attempt, ids.attempt}, {"world", l.world, ids.world},
		{"process", l.process, ids.process}, {"root", l.root, ids.root},
		{"invocation receipt", l.invocation, ids.invocation}, {"invocation file", l.file, ids.file},
	}
	for _, item := range sets {
		if _, duplicate := item.set[item.digest]; duplicate {
			return refuse("REUSED_CONFIRMATION_PHYSICAL_EVIDENCE", item.name, nil)
		}
	}
	for _, item := range sets {
		item.set[item.digest] = struct{}{}
	}
	return nil
}

func requireExactPhysicalCoverage(
	confirmed compare.CandidateOutcomeMap,
	facts []world.FreshExecutionFact,
	links []observe.PreparedExecutionLink,
) error {
	if len(facts) == 0 || len(facts) != len(links) || len(facts) != len(confirmed.EvidenceAttemptDigests()) ||
		len(facts) != len(confirmed.EvidenceWorldDigests()) {
		return refuse("CONFIRMATION_PHYSICAL_COVERAGE_MISMATCH", "fact cardinality differs from admitted map evidence", nil)
	}
	attempts := make([]domain.Digest, len(facts))
	worlds := make([]domain.Digest, len(facts))
	observations := make([]domain.Digest, 0, len(links))
	for index, fact := range facts {
		if fact.ScheduleOrdinal() != index {
			return refuse("CONFIRMATION_PHYSICAL_COVERAGE_MISMATCH", "physical facts are not the exact ordinal matrix", nil)
		}
		attempts[index] = fact.AttemptArtifactDigest()
		worlds[index] = fact.WorldDigest()
		if observation, present := links[index].ObservationDigest(); present {
			observations = append(observations, observation)
		}
	}
	sortDigests(attempts)
	sortDigests(worlds)
	sortDigests(observations)
	if !sameDigests(attempts, confirmed.EvidenceAttemptDigests()) ||
		!sameDigests(worlds, confirmed.EvidenceWorldDigests()) ||
		!sameDigests(observations, confirmed.EvidenceObservationDigests()) {
		return refuse("CONFIRMATION_PHYSICAL_COVERAGE_MISMATCH", "physical and admitted evidence sets differ", nil)
	}
	return nil
}

type priorEvidenceLedger struct {
	batch, attempt, world, observation map[domain.Digest]struct{}
	digest                             domain.Digest
}

func buildPriorEvidenceLedger(run reduce.ReductionRun, reduced compare.CandidateOutcomeMap) (priorEvidenceLedger, error) {
	ledger := priorEvidenceLedger{
		batch: map[domain.Digest]struct{}{}, attempt: map[domain.Digest]struct{}{},
		world: map[domain.Digest]struct{}{}, observation: map[domain.Digest]struct{}{},
	}
	ledger.addMap(run.Baseline().OutcomeMap())
	for _, entry := range run.Transcript().Entries() {
		evaluation := entry.Evaluation()
		addDigests(ledger.batch, evaluation.ObservedBatchDigests())
		addDigests(ledger.attempt, evaluation.ObservedAttemptDigests())
		addDigests(ledger.world, evaluation.ObservedWorldDigests())
		addDigests(ledger.observation, evaluation.ObservedObservationDigests())
	}
	ledger.addMap(reduced)
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Batch         []string `json:"batch_digests"`
		Attempt       []string `json:"attempt_digests"`
		World         []string `json:"world_digests"`
		Observation   []string `json:"observation_digests"`
	}{domain.SchemaVersion, "PriorEvidenceLedger", setStrings(ledger.batch), setStrings(ledger.attempt), setStrings(ledger.world), setStrings(ledger.observation)}
	digestRaw, _, err := canon.DigestTyped("PriorEvidenceLedger", identity)
	if err != nil {
		return priorEvidenceLedger{}, err
	}
	ledger.digest, err = domain.ParseDigest(digestRaw.String())
	return ledger, err
}

func (l priorEvidenceLedger) addMap(value compare.CandidateOutcomeMap) {
	addDigests(l.batch, value.BatchDigests())
	addDigests(l.attempt, value.EvidenceAttemptDigests())
	addDigests(l.world, value.EvidenceWorldDigests())
	addDigests(l.observation, value.EvidenceObservationDigests())
}

func (l priorEvidenceLedger) rejectMap(value compare.CandidateOutcomeMap) error {
	return l.rejectEvidence(
		value.BatchDigests(), value.EvidenceAttemptDigests(), value.EvidenceWorldDigests(), value.EvidenceObservationDigests(),
	)
}

func (l priorEvidenceLedger) rejectEvidence(batch, attempt, worldDigests, observation []domain.Digest) error {
	sets := []struct {
		name   string
		prior  map[domain.Digest]struct{}
		values []domain.Digest
	}{
		{"batch", l.batch, batch}, {"attempt", l.attempt, attempt},
		{"world", l.world, worldDigests}, {"observation", l.observation, observation},
	}
	for _, set := range sets {
		for _, digest := range set.values {
			if _, reused := set.prior[digest]; reused {
				return refuse("REUSED_CONFIRMATION_LINEAGE_EVIDENCE", set.name+":"+digest.String(), nil)
			}
		}
	}
	return nil
}

func addDigests(target map[domain.Digest]struct{}, values []domain.Digest) {
	for _, value := range values {
		target[value] = struct{}{}
	}
}

func setStrings(values map[domain.Digest]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value.String())
	}
	sort.Strings(result)
	return result
}

func sortDigests(values []domain.Digest) {
	sort.Slice(values, func(i, j int) bool { return values[i].String() < values[j].String() })
}

func sameDigests(left, right []domain.Digest) bool {
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

type freshConfirmationIdentity struct {
	SchemaVersion               string                    `json:"schema_version"`
	Kind                        string                    `json:"kind"`
	WorldPlanDigest             string                    `json:"world_plan_digest"`
	OriginalBaselineMapBase64   string                    `json:"original_baseline_map_base64"`
	ReducedBaselineMapBase64    string                    `json:"reduced_baseline_map_base64"`
	ReducedBaselineArtifact     string                    `json:"reduced_baseline_artifact_digest"`
	ReducedBaselinePreservation string                    `json:"reduced_baseline_preservation_digest"`
	ReductionRunBase64          string                    `json:"reduction_run_base64"`
	ReductionTranscriptBase64   string                    `json:"reduction_transcript_base64"`
	ReductionRunDigest          string                    `json:"reduction_run_digest"`
	ReductionTranscriptDigest   string                    `json:"reduction_transcript_digest"`
	ReductionGradeBase64        string                    `json:"reduction_grade_base64"`
	ReductionGradeDigest        string                    `json:"reduction_grade_digest"`
	ReductionGradeStatus        string                    `json:"reduction_grade_status"`
	ConfirmedMapBase64          string                    `json:"confirmed_map_base64"`
	ConfirmedArtifact           string                    `json:"confirmed_artifact_digest"`
	ConfirmedPreservation       string                    `json:"confirmed_preservation_digest"`
	ProjectionProofs            []projectionProofIdentity `json:"projection_proofs"`
	PreservationRelation        string                    `json:"preservation_relation"`
	PreservationReason          string                    `json:"preservation_reason"`
	ChallengeDigest             string                    `json:"challenge_digest"`
	ConfirmationPhase           string                    `json:"confirmation_phase"`
	ConfirmationScheduleDigest  string                    `json:"confirmation_schedule_digest"`
	ConfirmationScheduleOffset  int                       `json:"confirmation_schedule_offset"`
	ConfirmationTrialCount      int                       `json:"confirmation_trial_count"`
	PhysicalFactBytesBase64     []string                  `json:"physical_fact_bytes_base64"`
	PhysicalFactDigests         []string                  `json:"physical_fact_digests"`
	ProcessDigests              []string                  `json:"process_digests"`
	RootLayoutDigests           []string                  `json:"root_layout_digests"`
	InvocationReceiptDigests    []string                  `json:"invocation_receipt_digests"`
	InvocationFileDigests       []string                  `json:"invocation_file_digests"`
	BatchDigests                []string                  `json:"batch_digests"`
	AttemptDigests              []string                  `json:"attempt_digests"`
	WorldDigests                []string                  `json:"world_digests"`
	ObservationDigests          []string                  `json:"observation_digests"`
	PriorEvidenceLedgerDigest   string                    `json:"prior_evidence_ledger_digest"`
	Scope                       string                    `json:"scope"`
	InvocationTrustNonclaim     string                    `json:"invocation_trust_nonclaim"`
}

type projectionProofIdentity struct {
	CandidateExecutionKey     string `json:"candidate_execution_key"`
	ProjectionFingerprint     string `json:"projection_fingerprint"`
	CanonicalProjectionBase64 string `json:"canonical_projection_base64"`
}

func newDraft(
	request Request,
	challengeDigest, priorLedgerDigest domain.Digest,
	confirmed compare.ConfirmedOutcomeMap,
	assessment compare.PreservationAssessment,
	facts []world.FreshExecutionFact,
	proofs []ProjectionProof,
) (Draft, error) {
	reduced := request.ReducedBaseline.OutcomeMap()
	confirmedMap := confirmed.OutcomeMap()
	identity := freshConfirmationIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "FreshConfirmation", WorldPlanDigest: request.Plan.Digest().String(),
		OriginalBaselineMapBase64: base64.StdEncoding.EncodeToString(request.ReductionRun.Baseline().OutcomeMap().CanonicalBytes()),
		ReducedBaselineMapBase64:  base64.StdEncoding.EncodeToString(reduced.CanonicalBytes()),
		ReducedBaselineArtifact:   reduced.ArtifactDigest().String(), ReducedBaselinePreservation: reduced.PreservationDigest().String(),
		ReductionRunBase64:        base64.StdEncoding.EncodeToString(request.ReductionRun.CanonicalBytes()),
		ReductionTranscriptBase64: base64.StdEncoding.EncodeToString(request.ReductionRun.Transcript().CanonicalBytes()),
		ReductionRunDigest:        request.ReductionRun.Digest().String(), ReductionTranscriptDigest: request.ReductionRun.Transcript().Digest().String(),
		ReductionGradeBase64: base64.StdEncoding.EncodeToString(request.ReductionResult.Grade().CanonicalBytes()),
		ReductionGradeDigest: request.ReductionResult.Grade().Digest().String(), ReductionGradeStatus: string(request.ReductionResult.Grade().Status()),
		ConfirmedMapBase64: base64.StdEncoding.EncodeToString(confirmedMap.CanonicalBytes()),
		ConfirmedArtifact:  confirmedMap.ArtifactDigest().String(), ConfirmedPreservation: confirmedMap.PreservationDigest().String(),
		ProjectionProofs:     make([]projectionProofIdentity, len(proofs)),
		PreservationRelation: string(assessment.Relation()), PreservationReason: assessment.ReasonCode(),
		ChallengeDigest: challengeDigest.String(), ConfirmationPhase: string(domain.AttemptConfirmation),
		ConfirmationScheduleDigest: confirmedMap.ScheduleDigest().String(), ConfirmationScheduleOffset: confirmedMap.ScheduleStartOffset(),
		ConfirmationTrialCount: len(facts), PhysicalFactBytesBase64: make([]string, len(facts)),
		PhysicalFactDigests: make([]string, len(facts)), ProcessDigests: make([]string, len(facts)),
		RootLayoutDigests: make([]string, len(facts)), InvocationReceiptDigests: make([]string, len(facts)),
		InvocationFileDigests: make([]string, len(facts)), BatchDigests: digestStrings(confirmedMap.BatchDigests()),
		AttemptDigests: digestStrings(confirmedMap.EvidenceAttemptDigests()), WorldDigests: digestStrings(confirmedMap.EvidenceWorldDigests()),
		ObservationDigests:        digestStrings(confirmedMap.EvidenceObservationDigests()),
		PriorEvidenceLedgerDigest: priorLedgerDigest.String(), Scope: confirmationScope, InvocationTrustNonclaim: invocationTrustNonclaim,
	}
	for index, proof := range proofs {
		identity.ProjectionProofs[index] = projectionProofIdentity{
			CandidateExecutionKey: proof.candidate.String(), ProjectionFingerprint: proof.fingerprint.String(),
			CanonicalProjectionBase64: base64.StdEncoding.EncodeToString(proof.canonical),
		}
	}
	for index, fact := range facts {
		identity.PhysicalFactBytesBase64[index] = base64.StdEncoding.EncodeToString(fact.CanonicalBytes())
		identity.PhysicalFactDigests[index] = fact.Digest().String()
		identity.ProcessDigests[index] = fact.ProcessDigest().String()
		identity.RootLayoutDigests[index] = fact.RootLayoutDigest().String()
		identity.InvocationReceiptDigests[index] = fact.InvocationReceiptDigest().String()
		identity.InvocationFileDigests[index] = fact.InvocationFileDigest().String()
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("FreshConfirmation", identity)
	if err != nil {
		return Draft{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return Draft{}, err
	}
	record, err := parseRecordIdentity(identity, canonicalBytes, digest)
	if err != nil {
		return Draft{}, err
	}
	publicationAuthority, err := confirmationpublication.Issue(digest, request.ReductionRun.Digest(), canonicalBytes)
	if err != nil {
		return Draft{}, refuse("INVALID_FRESH_CONFIRMATION_DRAFT", "publication authority could not be issued", err)
	}
	draft := Draft{
		digest: digest, canonicalBytes: canonicalBytes, record: record, confirmed: confirmed,
		reduced: request.ReducedBaseline, assessment: assessment, facts: append([]world.FreshExecutionFact(nil), facts...),
		publication: publicationAuthority, liveSeal: &executionSeal{marker: 1},
	}
	if !draft.Valid() {
		return Draft{}, refuse("INVALID_FRESH_CONFIRMATION_DRAFT", "constructed draft failed self-validation", nil)
	}
	return draft, nil
}

func digestStrings(values []domain.Digest) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.String()
	}
	return result
}

func cloneDraft(input Draft) Draft {
	input.canonicalBytes = append([]byte(nil), input.canonicalBytes...)
	input.facts = append([]world.FreshExecutionFact(nil), input.facts...)
	return input
}

func (d Draft) Valid() bool {
	if d.liveSeal == nil || d.liveSeal.marker != 1 || !d.digest.Valid() || len(d.canonicalBytes) == 0 ||
		!d.publication.Valid() || d.publication.Digest() != d.digest ||
		d.publication.Predecessor() != d.record.ReductionRunDigest() ||
		!d.record.Valid() || !d.confirmed.Valid() || !d.reduced.Valid() || !d.assessment.Valid() || len(d.facts) == 0 {
		return false
	}
	parsed, err := ParseRecord(d.canonicalBytes)
	return err == nil && parsed.digest == d.digest && parsed.confirmedArtifact == d.confirmed.OutcomeMap().ArtifactDigest() &&
		parsed.reducedArtifact == d.reduced.OutcomeMap().ArtifactDigest() &&
		d.assessment.Relation() == compare.PreservationEqual
}

func (d Draft) Digest() domain.Digest  { return d.digest }
func (d Draft) CanonicalBytes() []byte { return append([]byte(nil), d.canonicalBytes...) }
func (d Draft) Record() Record         { return cloneRecord(d.record) }
func (d Draft) PublicationAuthority() confirmationauthority.Publication {
	return d.publication
}
func (d Draft) ConfirmedOutcomeMap() compare.ConfirmedOutcomeMap { return d.confirmed }
func (d Draft) ReducedBaseline() compare.DivergentBaseline       { return d.reduced }
func (d Draft) Assessment() compare.PreservationAssessment       { return d.assessment }
func (d Draft) PhysicalFacts() []world.FreshExecutionFact {
	return append([]world.FreshExecutionFact(nil), d.facts...)
}
