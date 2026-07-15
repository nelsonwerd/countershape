// Package compare owns exact projection fingerprints, complete expected
// candidate rosters, and labeled preservation maps. Display clustering is a
// derived view with no semantic authority.
package compare

import (
	"sort"
	"strconv"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

type Entry struct {
	CandidateKey          domain.CandidateExecutionKey
	ProjectionFingerprint domain.ProjectionFingerprint
	StableBatchDigest     domain.Digest
}

type Exclusion struct {
	CandidateKey      domain.CandidateExecutionKey
	StableBatchDigest domain.Digest
	Classification    observe.BatchStatus
}

// OutcomeArtifactDigest and PreservationMapDigest are distinct authority
// domains. Neither can be substituted for the other at compile time.
type OutcomeArtifactDigest struct{ digest domain.Digest }

func (d OutcomeArtifactDigest) Valid() bool                 { return d.digest.Valid() }
func (d OutcomeArtifactDigest) String() string              { return d.digest.String() }
func (d OutcomeArtifactDigest) DomainDigest() domain.Digest { return d.digest }

type PreservationMapDigest struct{ digest domain.Digest }

func (d PreservationMapDigest) Valid() bool                 { return d.digest.Valid() }
func (d PreservationMapDigest) String() string              { return d.digest.String() }
func (d PreservationMapDigest) DomainDigest() domain.Digest { return d.digest }

// CandidateOutcomeMap covers exactly one declared candidate roster. It may be
// nondivergent or contain exclusions; a separate DivergentBaseline authority
// is required to enter baseline/reduction semantics.
type CandidateOutcomeMap struct {
	artifactDigest             OutcomeArtifactDigest
	preservationDigest         PreservationMapDigest
	planDigest                 domain.Digest
	stimulusDigest             domain.Digest
	comparisonEnvelopeDigest   domain.Digest
	capturePolicyDigest        domain.Digest
	projectionDefinitionDigest domain.Digest
	comparisonAdmissionDigests []domain.Digest
	comparisonBasisDigest      domain.Digest
	phase                      domain.AttemptPurpose
	roster                     []domain.CandidateExecutionKey
	entries                    []Entry
	exclusions                 []Exclusion
	distinctProjectionCount    int
	attemptDigests             []domain.Digest
	worldDigests               []domain.Digest
	observationDigests         []domain.Digest
	canonicalBytes             []byte
}

func NewCandidateOutcomeMap(
	stimulusDigest domain.Digest,
	comparisonEnvelopeDigest domain.Digest,
	expectedRoster []domain.CandidateExecutionKey,
	batches []observe.StableBatch,
) (CandidateOutcomeMap, error) {
	if !stimulusDigest.Valid() || !comparisonEnvelopeDigest.Valid() {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_IDENTITY"}
	}
	roster, err := validateRoster(expectedRoster)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	if len(batches) != len(roster) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "CANDIDATE_ROSTER_BATCH_MISMATCH"}
	}

	expected := make(map[domain.CandidateExecutionKey]struct{}, len(roster))
	for _, key := range roster {
		expected[key] = struct{}{}
	}
	seen := map[domain.CandidateExecutionKey]struct{}{}
	var admissionDigests []domain.Digest
	var planDigest domain.Digest
	var basisDigest domain.Digest
	var capturePolicyDigest domain.Digest
	var projectionDefinitionDigest domain.Digest
	var phase domain.AttemptPurpose
	seenAttempts := map[domain.Digest]struct{}{}
	seenWorlds := map[domain.Digest]struct{}{}
	seenObservations := map[domain.Digest]struct{}{}
	attemptDigests := make([]domain.Digest, 0)
	worldDigests := make([]domain.Digest, 0)
	observationDigests := make([]domain.Digest, 0)
	entries := make([]Entry, 0, len(batches))
	exclusions := make([]Exclusion, 0, len(batches))
	for _, batch := range batches {
		key := batch.CandidateKey()
		if !key.Valid() || !batch.Digest().Valid() || batch.StimulusDigest() != stimulusDigest ||
			batch.EnvelopeDigest() != comparisonEnvelopeDigest {
			return CandidateOutcomeMap{}, &domain.Error{Code: "BATCH_LINEAGE_MISMATCH"}
		}
		batchAdmissions := batch.AdmissionDigests()
		if invalidOrDuplicateDigests(batchAdmissions) || len(batchAdmissions) != len(batch.Trials()) || !batch.PlanDigest().Valid() ||
			!batch.CapturePolicyDigest().Valid() || !batch.ProjectionDefinitionDigest().Valid() ||
			!batch.ComparisonBasisDigest().Valid() || !batch.Phase().Valid() {
			return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_BATCH_ADMISSION_OR_PHASE"}
		}
		if !sameRoster(batch.AdmissionRoster(), roster) {
			return CandidateOutcomeMap{}, &domain.Error{Code: "COMPARISON_ADMISSION_ROSTER_MISMATCH"}
		}
		if len(admissionDigests) == 0 {
			admissionDigests = append([]domain.Digest(nil), batchAdmissions...)
			planDigest = batch.PlanDigest()
			capturePolicyDigest = batch.CapturePolicyDigest()
			projectionDefinitionDigest = batch.ProjectionDefinitionDigest()
			basisDigest = batch.ComparisonBasisDigest()
			phase = batch.Phase()
			// MUTANT_U1_COMPARE_IGNORE_SHARED_ADMISSION_SET: every candidate must come from the same concrete matrices.
		} else if !sameDigestList(batchAdmissions, admissionDigests) || batch.PlanDigest() != planDigest ||
			batch.CapturePolicyDigest() != capturePolicyDigest || batch.ProjectionDefinitionDigest() != projectionDefinitionDigest ||
			batch.ComparisonBasisDigest() != basisDigest || batch.Phase() != phase {
			return CandidateOutcomeMap{}, &domain.Error{Code: "MIXED_BATCH_ADMISSION_OR_PHASE"}
		}
		for _, digest := range batch.AttemptDigests() {
			if !digest.Valid() {
				return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_ATTEMPT_EVIDENCE"}
			}
			if _, reused := seenAttempts[digest]; reused {
				return CandidateOutcomeMap{}, &domain.Error{Code: "REUSED_OUTCOME_MAP_ATTEMPT_EVIDENCE", Detail: digest.String()}
			}
			seenAttempts[digest] = struct{}{}
			attemptDigests = append(attemptDigests, digest)
		}
		for _, digest := range batch.WorldDigests() {
			if !digest.Valid() {
				return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_WORLD_EVIDENCE"}
			}
			if _, reused := seenWorlds[digest]; reused {
				return CandidateOutcomeMap{}, &domain.Error{Code: "REUSED_OUTCOME_MAP_WORLD_EVIDENCE", Detail: digest.String()}
			}
			seenWorlds[digest] = struct{}{}
			worldDigests = append(worldDigests, digest)
		}
		for _, digest := range batch.ObservationDigests() {
			if !digest.Valid() {
				return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_OBSERVATION_EVIDENCE"}
			}
			if _, reused := seenObservations[digest]; reused {
				return CandidateOutcomeMap{}, &domain.Error{Code: "REUSED_OUTCOME_MAP_OBSERVATION_EVIDENCE", Detail: digest.String()}
			}
			seenObservations[digest] = struct{}{}
			observationDigests = append(observationDigests, digest)
		}
		if _, belongs := expected[key]; !belongs {
			return CandidateOutcomeMap{}, &domain.Error{Code: "UNEXPECTED_CANDIDATE_BATCH", Detail: key.String()}
		}
		if _, duplicate := seen[key]; duplicate {
			return CandidateOutcomeMap{}, &domain.Error{Code: domain.ErrDuplicateCandidateKey, Detail: key.String()}
		}
		seen[key] = struct{}{}
		classification := batch.Classification()
		if classification.Status() == observe.ObservedStable {
			fingerprint, ok := classification.Fingerprint()
			if !ok || !fingerprint.Valid() {
				return CandidateOutcomeMap{}, &domain.Error{Code: "CONTROL_AS_OUTCOME"}
			}
			entries = append(entries, Entry{
				CandidateKey:          key,
				ProjectionFingerprint: fingerprint,
				StableBatchDigest:     batch.Digest(),
			})
			continue
		}
		switch classification.Status() {
		case observe.Unstable, observe.Uncomparable, observe.Incomplete:
			exclusions = append(exclusions, Exclusion{
				CandidateKey:      key,
				StableBatchDigest: batch.Digest(),
				Classification:    classification.Status(),
			})
		default:
			return CandidateOutcomeMap{}, &domain.Error{Code: "UNKNOWN_BATCH_CLASSIFICATION"}
		}
	}
	if len(seen) != len(expected) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "MISSING_CANDIDATE_BATCH"}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CandidateKey.String() < entries[j].CandidateKey.String()
	})
	sort.Slice(exclusions, func(i, j int) bool {
		return exclusions[i].CandidateKey.String() < exclusions[j].CandidateKey.String()
	})
	sort.Slice(attemptDigests, func(i, j int) bool { return attemptDigests[i].String() < attemptDigests[j].String() })
	sort.Slice(worldDigests, func(i, j int) bool { return worldDigests[i].String() < worldDigests[j].String() })
	sort.Slice(observationDigests, func(i, j int) bool { return observationDigests[i].String() < observationDigests[j].String() })

	preservationDigest, err := digestPreservationMap(entries)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	artifactDigest, canonicalBytes, err := digestOutcomeArtifact(
		planDigest,
		stimulusDigest,
		comparisonEnvelopeDigest,
		capturePolicyDigest,
		projectionDefinitionDigest,
		admissionDigests,
		basisDigest,
		phase,
		roster,
		entries,
		exclusions,
		attemptDigests,
		worldDigests,
		observationDigests,
	)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	return CandidateOutcomeMap{
		artifactDigest:             artifactDigest,
		preservationDigest:         preservationDigest,
		planDigest:                 planDigest,
		stimulusDigest:             stimulusDigest,
		comparisonEnvelopeDigest:   comparisonEnvelopeDigest,
		capturePolicyDigest:        capturePolicyDigest,
		projectionDefinitionDigest: projectionDefinitionDigest,
		comparisonAdmissionDigests: append([]domain.Digest(nil), admissionDigests...),
		comparisonBasisDigest:      basisDigest,
		phase:                      phase,
		roster:                     append([]domain.CandidateExecutionKey(nil), roster...),
		entries:                    append([]Entry(nil), entries...),
		exclusions:                 append([]Exclusion(nil), exclusions...),
		distinctProjectionCount:    distinctFingerprintCount(entries),
		attemptDigests:             append([]domain.Digest(nil), attemptDigests...),
		worldDigests:               append([]domain.Digest(nil), worldDigests...),
		observationDigests:         append([]domain.Digest(nil), observationDigests...),
		canonicalBytes:             canonicalBytes,
	}, nil
}

func validateRoster(input []domain.CandidateExecutionKey) ([]domain.CandidateExecutionKey, error) {
	if len(input) < 2 || len(input) > 4 {
		return nil, &domain.Error{Code: "INVALID_EXPECTED_CANDIDATE_ROSTER", Detail: "candidate count outside 2..4"}
	}
	result := append([]domain.CandidateExecutionKey(nil), input...)
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	for index, key := range result {
		if !key.Valid() {
			return nil, &domain.Error{Code: "INVALID_EXPECTED_CANDIDATE_ROSTER"}
		}
		if index > 0 && key == result[index-1] {
			return nil, &domain.Error{Code: domain.ErrDuplicateCandidateKey, Detail: key.String()}
		}
	}
	return result, nil
}

// DivergentBaseline is sealed authority for a complete-roster map containing
// at least two eligible candidates and two distinct exact fingerprints.
type DivergentBaseline struct{ outcome CandidateOutcomeMap }

func RequireDivergence(outcome CandidateOutcomeMap) (DivergentBaseline, error) {
	if !outcome.artifactDigest.Valid() || len(outcome.roster) < 2 || len(outcome.entries) < 2 ||
		outcome.distinctProjectionCount < 2 {
		return DivergentBaseline{}, &domain.Error{Code: "NO_ELIGIBLE_DIVERGENCE"}
	}
	return DivergentBaseline{outcome: outcome}, nil
}

func (b DivergentBaseline) OutcomeMap() CandidateOutcomeMap { return b.outcome }

func (b DivergentBaseline) Valid() bool {
	return b.outcome.artifactDigest.Valid() && len(b.outcome.entries) >= 2 && b.outcome.distinctProjectionCount >= 2
}

// ConfirmedOutcomeMap is a complete, still-divergent map whose every batch is
// structurally bound to a finalized CONFIRMATION attempt. Physical freshness is
// a later execution/store claim; U1 does not infer it from this wrapper.
type ConfirmedOutcomeMap struct{ outcome CandidateOutcomeMap }

func RequireConfirmedOutcomeMap(outcome CandidateOutcomeMap) (ConfirmedOutcomeMap, error) {
	if !outcome.artifactDigest.Valid() || outcome.phase != domain.AttemptConfirmation ||
		len(outcome.entries) < 2 || outcome.distinctProjectionCount < 2 {
		return ConfirmedOutcomeMap{}, &domain.Error{Code: "INVALID_CONFIRMED_OUTCOME_MAP"}
	}
	return ConfirmedOutcomeMap{outcome: outcome}, nil
}

func (m ConfirmedOutcomeMap) Valid() bool {
	return m.outcome.artifactDigest.Valid() && m.outcome.phase == domain.AttemptConfirmation &&
		len(m.outcome.entries) >= 2 && m.outcome.distinctProjectionCount >= 2
}

func (m ConfirmedOutcomeMap) OutcomeMap() CandidateOutcomeMap { return m.outcome }

// ConfirmedProjectionEntry is a defensive DTO. Possessing a copied entry is
// not authority; Verify must be called on its originating opaque roster.
type ConfirmedProjectionEntry struct {
	candidate   domain.CandidateExecutionKey
	fingerprint domain.ProjectionFingerprint
}

func (e ConfirmedProjectionEntry) CandidateExecutionKey() domain.CandidateExecutionKey {
	return e.candidate
}

func (e ConfirmedProjectionEntry) ProjectionFingerprint() domain.ProjectionFingerprint {
	return e.fingerprint
}

// ConfirmedProjectionRoster is the map-derived capability consumed by choice.
// There is no public constructor or parser.
type ConfirmedProjectionRoster struct {
	mapDigest                  OutcomeArtifactDigest
	preservationDigest         PreservationMapDigest
	projectionDefinitionDigest domain.Digest
	entries                    []ConfirmedProjectionEntry
}

func (m ConfirmedOutcomeMap) ProjectionRoster() ConfirmedProjectionRoster {
	entries := make([]ConfirmedProjectionEntry, len(m.outcome.entries))
	for index, entry := range m.outcome.entries {
		entries[index] = ConfirmedProjectionEntry{
			candidate:   entry.CandidateKey,
			fingerprint: entry.ProjectionFingerprint,
		}
	}
	return ConfirmedProjectionRoster{
		mapDigest:                  m.outcome.artifactDigest,
		preservationDigest:         m.outcome.preservationDigest,
		projectionDefinitionDigest: m.outcome.projectionDefinitionDigest,
		entries:                    entries,
	}
}

func (r ConfirmedProjectionRoster) Valid() bool {
	return r.mapDigest.Valid() && r.preservationDigest.Valid() && r.projectionDefinitionDigest.Valid() && len(r.entries) >= 2
}

func (r ConfirmedProjectionRoster) OutcomeMapDigest() OutcomeArtifactDigest { return r.mapDigest }

func (r ConfirmedProjectionRoster) PreservationDigest() PreservationMapDigest {
	return r.preservationDigest
}

func (r ConfirmedProjectionRoster) ProjectionDefinitionDigest() domain.Digest {
	return r.projectionDefinitionDigest
}

func (r ConfirmedProjectionRoster) Entries() []ConfirmedProjectionEntry {
	return append([]ConfirmedProjectionEntry(nil), r.entries...)
}

func (r ConfirmedProjectionRoster) Verify(
	candidate domain.CandidateExecutionKey,
	canonicalProjection []byte,
) (domain.ProjectionFingerprint, error) {
	if !r.Valid() || !candidate.Valid() {
		return domain.ProjectionFingerprint{}, &domain.Error{Code: "INVALID_CONFIRMED_PROJECTION_ROSTER"}
	}
	computed, err := domain.NewProjectionFingerprint(canonicalProjection)
	if err != nil {
		return domain.ProjectionFingerprint{}, &domain.Error{Code: "NONCANONICAL_CONFIRMED_PROJECTION", Detail: err.Error()}
	}
	for _, entry := range r.entries {
		if entry.candidate == candidate {
			if entry.fingerprint != computed {
				return domain.ProjectionFingerprint{}, &domain.Error{Code: "CONFIRMED_PROJECTION_FINGERPRINT_MISMATCH"}
			}
			return computed, nil
		}
	}
	return domain.ProjectionFingerprint{}, &domain.Error{Code: "CANDIDATE_NOT_ELIGIBLE_IN_CONFIRMED_MAP"}
}

type labeledEntry struct {
	CandidateExecutionKey string `json:"candidate_execution_key"`
	ProjectionFingerprint string `json:"projection_fingerprint"`
}

func semanticCandidateKey(entry Entry) string {
	// MUTATION_ANCHOR: candidate-key-is-part-of-labeled-map-identity
	return entry.CandidateKey.String()
}

func digestPreservationMap(entries []Entry) (PreservationMapDigest, error) {
	identity := struct {
		IdentityBasis string         `json:"identity_basis"`
		Entries       []labeledEntry `json:"entries"`
	}{
		IdentityBasis: "COMPLETE_SORTED_ELIGIBLE_CANDIDATE_TO_PROJECTION_MAP",
		Entries:       make([]labeledEntry, len(entries)),
	}
	for index, entry := range entries {
		identity.Entries[index] = labeledEntry{
			CandidateExecutionKey: semanticCandidateKey(entry),
			ProjectionFingerprint: entry.ProjectionFingerprint.String(),
		}
	}
	digest, err := digestCanonical("OutcomeMapPreservation", identity)
	if err != nil {
		return PreservationMapDigest{}, err
	}
	return PreservationMapDigest{digest: digest}, nil
}

func digestOutcomeArtifact(
	planDigest domain.Digest,
	stimulusDigest domain.Digest,
	envelopeDigest domain.Digest,
	capturePolicyDigest domain.Digest,
	projectionDefinitionDigest domain.Digest,
	admissionDigests []domain.Digest,
	basisDigest domain.Digest,
	phase domain.AttemptPurpose,
	roster []domain.CandidateExecutionKey,
	entries []Entry,
	exclusions []Exclusion,
	attemptDigests []domain.Digest,
	worldDigests []domain.Digest,
	observationDigests []domain.Digest,
) (OutcomeArtifactDigest, []byte, error) {
	type artifactEntry struct {
		CandidateExecutionKey string `json:"candidate_execution_key"`
		ProjectionFingerprint string `json:"projection_fingerprint"`
		StableBatchDigest     string `json:"stable_batch_digest"`
	}
	type artifactExclusion struct {
		CandidateExecutionKey string `json:"candidate_execution_key"`
		StableBatchDigest     string `json:"stable_batch_digest"`
		Classification        string `json:"classification"`
	}
	identity := struct {
		SchemaVersion                  string              `json:"schema_version"`
		Kind                           string              `json:"kind"`
		WorldPlanDigest                string              `json:"world_plan_digest"`
		StimulusDigest                 string              `json:"stimulus_digest"`
		ComparisonEnvelopeDigest       string              `json:"comparison_envelope_digest"`
		CapturePolicyDigest            string              `json:"capture_policy_digest"`
		ProjectionDefinitionDigest     string              `json:"projection_definition_digest"`
		ComparisonAdmissionDigests     []string            `json:"comparison_admission_digests"`
		ComparisonBasisDigest          string              `json:"comparison_basis_digest"`
		Phase                          string              `json:"phase"`
		ExpectedCandidateRoster        []string            `json:"expected_candidate_roster"`
		Entries                        []artifactEntry     `json:"entries"`
		Exclusions                     []artifactExclusion `json:"excluded_candidates"`
		AttemptArtifactDigests         []string            `json:"attempt_artifact_digests"`
		WorldInstanceDigests           []string            `json:"world_instance_digests"`
		CapturedObservationDigests     []string            `json:"captured_observation_digests"`
		DistinctProjectionCount        int                 `json:"distinct_projection_count"`
		Divergence                     bool                `json:"divergence"`
		EntryOrder                     string              `json:"entry_order"`
		PreservationIdentityBasis      string              `json:"preservation_identity_basis"`
		DisplayGroupsSemanticAuthority bool                `json:"display_groups_semantic_authority"`
		MajoritySemanticAuthority      bool                `json:"majority_semantic_authority"`
	}{
		SchemaVersion:                  domain.SchemaVersion,
		Kind:                           "CandidateOutcomeMap",
		WorldPlanDigest:                planDigest.String(),
		StimulusDigest:                 stimulusDigest.String(),
		ComparisonEnvelopeDigest:       envelopeDigest.String(),
		CapturePolicyDigest:            capturePolicyDigest.String(),
		ProjectionDefinitionDigest:     projectionDefinitionDigest.String(),
		ComparisonAdmissionDigests:     make([]string, len(admissionDigests)),
		ComparisonBasisDigest:          basisDigest.String(),
		Phase:                          string(phase),
		ExpectedCandidateRoster:        make([]string, len(roster)),
		Entries:                        make([]artifactEntry, len(entries)),
		Exclusions:                     make([]artifactExclusion, len(exclusions)),
		AttemptArtifactDigests:         make([]string, len(attemptDigests)),
		WorldInstanceDigests:           make([]string, len(worldDigests)),
		CapturedObservationDigests:     make([]string, len(observationDigests)),
		DistinctProjectionCount:        distinctFingerprintCount(entries),
		Divergence:                     len(entries) >= 2 && distinctFingerprintCount(entries) >= 2,
		EntryOrder:                     "CANDIDATE_EXECUTION_KEY_UTF8_LEXICOGRAPHIC",
		PreservationIdentityBasis:      "COMPLETE_SORTED_ELIGIBLE_CANDIDATE_TO_PROJECTION_MAP",
		DisplayGroupsSemanticAuthority: false,
		MajoritySemanticAuthority:      false,
	}
	for index, key := range roster {
		identity.ExpectedCandidateRoster[index] = key.String()
	}
	for index, digest := range admissionDigests {
		identity.ComparisonAdmissionDigests[index] = digest.String()
	}
	for index, entry := range entries {
		identity.Entries[index] = artifactEntry{
			CandidateExecutionKey: semanticCandidateKey(entry),
			ProjectionFingerprint: entry.ProjectionFingerprint.String(),
			StableBatchDigest:     entry.StableBatchDigest.String(),
		}
	}
	for index, exclusion := range exclusions {
		identity.Exclusions[index].CandidateExecutionKey = exclusion.CandidateKey.String()
		identity.Exclusions[index].StableBatchDigest = exclusion.StableBatchDigest.String()
		identity.Exclusions[index].Classification = string(exclusion.Classification)
	}
	for index, digest := range attemptDigests {
		identity.AttemptArtifactDigests[index] = digest.String()
	}
	for index, digest := range worldDigests {
		identity.WorldInstanceDigests[index] = digest.String()
	}
	for index, digest := range observationDigests {
		identity.CapturedObservationDigests[index] = digest.String()
	}
	digest, canonicalBytes, err := digestCanonicalBytes("CandidateOutcomeMap", identity)
	if err != nil {
		return OutcomeArtifactDigest{}, nil, err
	}
	return OutcomeArtifactDigest{digest: digest}, canonicalBytes, nil
}

func distinctFingerprintCount(entries []Entry) int {
	distinct := map[domain.ProjectionFingerprint]struct{}{}
	for _, entry := range entries {
		distinct[entry.ProjectionFingerprint] = struct{}{}
	}
	return len(distinct)
}

func digestCanonical(kind string, value any) (domain.Digest, error) {
	digest, _, err := digestCanonicalBytes(kind, value)
	return digest, err
}

func digestCanonicalBytes(kind string, value any) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	return parsed, canonicalBytes, err
}

func SamePreservationMap(left, right CandidateOutcomeMap) bool {
	// MUTATION_ANCHOR: preservation-is-exact-labeled-map
	return ComparableForPreservation(left, right) &&
		left.preservationDigest.Valid() && left.preservationDigest == right.preservationDigest
}

// ComparableForPreservation requires the same policy, exact expected roster,
// and the same eligible/excluded candidate disposition. A changed eligibility
// set is UNRESOLVED, not a different product outcome.
func ComparableForPreservation(baseline, observed CandidateOutcomeMap) bool {
	if !baseline.artifactDigest.Valid() || !observed.artifactDigest.Valid() ||
		baseline.planDigest != observed.planDigest ||
		baseline.comparisonEnvelopeDigest != observed.comparisonEnvelopeDigest ||
		baseline.comparisonBasisDigest != observed.comparisonBasisDigest ||
		!sameRoster(baseline.roster, observed.roster) || len(baseline.entries) != len(observed.entries) ||
		len(baseline.exclusions) != len(observed.exclusions) {
		return false
	}
	for index := range baseline.entries {
		if baseline.entries[index].CandidateKey != observed.entries[index].CandidateKey {
			return false
		}
	}
	for index := range baseline.exclusions {
		if baseline.exclusions[index].CandidateKey != observed.exclusions[index].CandidateKey ||
			baseline.exclusions[index].Classification != observed.exclusions[index].Classification {
			return false
		}
	}
	return true
}

func sameRoster(left, right []domain.CandidateExecutionKey) bool {
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

func sameDigestList(left, right []domain.Digest) bool {
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

func invalidOrDuplicateDigests(digests []domain.Digest) bool {
	if len(digests) == 0 {
		return true
	}
	seen := make(map[domain.Digest]struct{}, len(digests))
	for _, digest := range digests {
		if !digest.Valid() {
			return true
		}
		if _, duplicate := seen[digest]; duplicate {
			return true
		}
		seen[digest] = struct{}{}
	}
	return false
}

func samePartitionShape(left, right CandidateOutcomeMap) bool {
	return partitionShape(left.entries) == partitionShape(right.entries)
}

func partitionShape(entries []Entry) string {
	counts := map[domain.ProjectionFingerprint]int{}
	for _, entry := range entries {
		counts[entry.ProjectionFingerprint]++
	}
	shape := make([]int, 0, len(counts))
	for _, count := range counts {
		shape = append(shape, count)
	}
	sort.Ints(shape)
	parts := make([]string, len(shape))
	for index, count := range shape {
		parts[index] = strconv.Itoa(count)
	}
	return strings.Join(parts, ":")
}

type DisplayGroup struct {
	Fingerprint domain.ProjectionFingerprint
	Members     []domain.CandidateExecutionKey
}

func (m CandidateOutcomeMap) DisplayGroups() []DisplayGroup {
	grouped := map[domain.ProjectionFingerprint][]domain.CandidateExecutionKey{}
	for _, entry := range m.entries {
		grouped[entry.ProjectionFingerprint] = append(grouped[entry.ProjectionFingerprint], entry.CandidateKey)
	}
	groups := make([]DisplayGroup, 0, len(grouped))
	for fingerprint, members := range grouped {
		sort.Slice(members, func(i, j int) bool { return members[i].String() < members[j].String() })
		groups = append(groups, DisplayGroup{Fingerprint: fingerprint, Members: members})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Fingerprint.String() < groups[j].Fingerprint.String() })
	return groups
}

func (m CandidateOutcomeMap) ArtifactDigest() OutcomeArtifactDigest     { return m.artifactDigest }
func (m CandidateOutcomeMap) PreservationDigest() PreservationMapDigest { return m.preservationDigest }
func (m CandidateOutcomeMap) PlanDigest() domain.Digest                 { return m.planDigest }
func (m CandidateOutcomeMap) StimulusDigest() domain.Digest             { return m.stimulusDigest }
func (m CandidateOutcomeMap) EnvelopeDigest() domain.Digest             { return m.comparisonEnvelopeDigest }
func (m CandidateOutcomeMap) CapturePolicyDigest() domain.Digest        { return m.capturePolicyDigest }
func (m CandidateOutcomeMap) ProjectionDefinitionDigest() domain.Digest {
	return m.projectionDefinitionDigest
}
func (m CandidateOutcomeMap) ComparisonBasisDigest() domain.Digest { return m.comparisonBasisDigest }
func (m CandidateOutcomeMap) AdmissionDigests() []domain.Digest {
	return append([]domain.Digest(nil), m.comparisonAdmissionDigests...)
}
func (m CandidateOutcomeMap) Phase() domain.AttemptPurpose { return m.phase }

func (m CandidateOutcomeMap) CandidateRoster() []domain.CandidateExecutionKey {
	return append([]domain.CandidateExecutionKey(nil), m.roster...)
}

func (m CandidateOutcomeMap) Entries() []Entry { return append([]Entry(nil), m.entries...) }
func (m CandidateOutcomeMap) Exclusions() []Exclusion {
	return append([]Exclusion(nil), m.exclusions...)
}
func (m CandidateOutcomeMap) DistinctProjectionCount() int { return m.distinctProjectionCount }

func (m CandidateOutcomeMap) Divergence() bool {
	return len(m.entries) >= 2 && m.distinctProjectionCount >= 2
}

func (m CandidateOutcomeMap) BatchDigests() []domain.Digest {
	result := make([]domain.Digest, 0, len(m.roster))
	for _, entry := range m.entries {
		result = append(result, entry.StableBatchDigest)
	}
	for _, exclusion := range m.exclusions {
		result = append(result, exclusion.StableBatchDigest)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func (m CandidateOutcomeMap) EvidenceAttemptDigests() []domain.Digest {
	return append([]domain.Digest(nil), m.attemptDigests...)
}

func (m CandidateOutcomeMap) EvidenceWorldDigests() []domain.Digest {
	return append([]domain.Digest(nil), m.worldDigests...)
}

func (m CandidateOutcomeMap) EvidenceObservationDigests() []domain.Digest {
	return append([]domain.Digest(nil), m.observationDigests...)
}

func (m CandidateOutcomeMap) CanonicalBytes() []byte {
	return append([]byte(nil), m.canonicalBytes...)
}
