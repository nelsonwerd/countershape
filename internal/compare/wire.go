package compare

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

const candidateOutcomeMapMemberCount = 25

// ParseCandidateOutcomeMap strictly reconstructs a durable map from its exact
// canonical bytes. It is intentionally narrower than a generic JSON decoder:
// every member, closed enum, order, derived count, and domain-separated digest
// is checked before the typed map is returned.
func ParseCandidateOutcomeMap(exact []byte) (CandidateOutcomeMap, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_WIRE", Detail: err.Error()}
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "NONEXACT_OUTCOME_MAP_WIRE"}
	}
	members, object := value.Members()
	if !object || len(members) != candidateOutcomeMapMemberCount {
		return CandidateOutcomeMap{}, &domain.Error{Code: "UNKNOWN_OR_MISSING_OUTCOME_MAP_FIELD"}
	}

	schema, ok := wireText(value, "schema_version")
	if !ok || schema != domain.SchemaVersion {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_SCHEMA"}
	}
	kind, ok := wireText(value, "kind")
	if !ok || kind != "CandidateOutcomeMap" {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_KIND"}
	}
	plan, err := wireDigest(value, "world_plan_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	stimulus, err := wireDigest(value, "stimulus_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	envelope, err := wireDigest(value, "comparison_envelope_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	capture, err := wireDigest(value, "capture_policy_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	projection, err := wireDigest(value, "projection_definition_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	basis, err := wireDigest(value, "comparison_basis_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	schedule, err := wireDigest(value, "schedule_digest")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}

	phaseText, ok := wireText(value, "phase")
	phase := domain.AttemptPurpose(phaseText)
	rotation, rotationOK := wireText(value, "rotation")
	startOffset64, offsetOK := wireInteger(value, "schedule_start_offset")
	if !ok || !phase.Valid() || !rotationOK || rotation == "" || !offsetOK || startOffset64 < 0 || startOffset64 > 1 ||
		(phase == domain.AttemptConfirmation && startOffset64 != 1) ||
		(phase != domain.AttemptConfirmation && startOffset64 != 0) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_SCHEDULE"}
	}
	admissions, err := wireDigestArray(value, "comparison_admission_digests", 1, 5)
	if err != nil || invalidOrDuplicateDigests(admissions) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_ADMISSIONS"}
	}
	roster, err := wireRoster(value, "expected_candidate_roster")
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	entries, err := wireEntries(value)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	exclusions, err := wireExclusions(value)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	if err := validateWireDisposition(roster, entries, exclusions); err != nil {
		return CandidateOutcomeMap{}, err
	}
	attempts, err := wireDigestArray(value, "attempt_artifact_digests", len(roster), len(roster)*5)
	if err != nil || invalidOrDuplicateDigests(attempts) || !strictDigestOrder(attempts) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_ATTEMPT_EVIDENCE"}
	}
	worlds, err := wireDigestArray(value, "world_instance_digests", len(roster), len(roster)*5)
	if err != nil || invalidOrDuplicateDigests(worlds) || !strictDigestOrder(worlds) || len(worlds) != len(attempts) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_WORLD_EVIDENCE"}
	}
	observations, err := wireDigestArray(value, "captured_observation_digests", 0, len(attempts))
	if err != nil || (len(observations) > 0 && (invalidOrDuplicateDigests(observations) || !strictDigestOrder(observations))) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_OBSERVATION_EVIDENCE"}
	}

	distinct64, distinctOK := wireInteger(value, "distinct_projection_count")
	divergence, divergenceOK := wireBool(value, "divergence")
	entryOrder, entryOrderOK := wireText(value, "entry_order")
	preservationBasis, preservationBasisOK := wireText(value, "preservation_identity_basis")
	displayAuthority, displayOK := wireBool(value, "display_groups_semantic_authority")
	majorityAuthority, majorityOK := wireBool(value, "majority_semantic_authority")
	distinct := distinctFingerprintCount(entries)
	if !distinctOK || int64(distinct) != distinct64 || !divergenceOK ||
		divergence != (len(entries) >= 2 && distinct >= 2) ||
		!entryOrderOK || entryOrder != "CANDIDATE_EXECUTION_KEY_UTF8_LEXICOGRAPHIC" ||
		!preservationBasisOK || preservationBasis != "COMPLETE_SORTED_ELIGIBLE_CANDIDATE_TO_PROJECTION_MAP" ||
		!displayOK || displayAuthority || !majorityOK || majorityAuthority {
		return CandidateOutcomeMap{}, &domain.Error{Code: "INVALID_OUTCOME_MAP_DERIVED_FACTS"}
	}
	preservationDigest, err := digestPreservationMap(entries)
	if err != nil {
		return CandidateOutcomeMap{}, err
	}
	artifactDigest, rebuiltBytes, err := digestOutcomeArtifact(
		plan, stimulus, envelope, capture, projection, admissions, basis, phase, schedule, rotation,
		int(startOffset64), roster, entries, exclusions, attempts, worlds, observations,
	)
	if err != nil || !bytes.Equal(rebuiltBytes, exact) {
		return CandidateOutcomeMap{}, &domain.Error{Code: "OUTCOME_MAP_WIRE_IDENTITY_MISMATCH"}
	}
	return CandidateOutcomeMap{
		artifactDigest: artifactDigest, preservationDigest: preservationDigest, planDigest: plan,
		stimulusDigest: stimulus, comparisonEnvelopeDigest: envelope, capturePolicyDigest: capture,
		projectionDefinitionDigest: projection, comparisonAdmissionDigests: admissions,
		comparisonBasisDigest: basis, phase: phase, scheduleDigest: schedule, rotation: rotation,
		scheduleStartOffset: int(startOffset64), roster: roster, entries: entries, exclusions: exclusions,
		distinctProjectionCount: distinct, attemptDigests: attempts, worldDigests: worlds,
		observationDigests: observations, canonicalBytes: append([]byte(nil), exact...),
	}, nil
}

func wireText(object canon.Value, name string) (string, bool) {
	value, present := object.LookupMember(name)
	if !present {
		return "", false
	}
	return value.Text()
}

func wireInteger(object canon.Value, name string) (int64, bool) {
	value, present := object.LookupMember(name)
	if !present {
		return 0, false
	}
	return value.Int64()
}

func wireBool(object canon.Value, name string) (bool, bool) {
	value, present := object.LookupMember(name)
	if !present {
		return false, false
	}
	return value.Boolean()
}

func wireDigest(object canon.Value, name string) (domain.Digest, error) {
	raw, ok := wireText(object, name)
	if !ok {
		return "", &domain.Error{Code: "INVALID_OUTCOME_MAP_DIGEST", Detail: name}
	}
	digest, err := domain.ParseDigest(raw)
	if err != nil {
		return "", &domain.Error{Code: "INVALID_OUTCOME_MAP_DIGEST", Detail: name}
	}
	return digest, nil
}

func wireDigestArray(object canon.Value, name string, minimum, maximum int) ([]domain.Digest, error) {
	value, present := object.LookupMember(name)
	elements, array := value.Elements()
	if !present || !array || len(elements) < minimum || len(elements) > maximum {
		return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_DIGEST_ARRAY", Detail: name}
	}
	result := make([]domain.Digest, len(elements))
	for index, element := range elements {
		raw, text := element.Text()
		if !text {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_DIGEST_ARRAY", Detail: name}
		}
		digest, err := domain.ParseDigest(raw)
		if err != nil {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_DIGEST_ARRAY", Detail: name}
		}
		result[index] = digest
	}
	return result, nil
}

func wireRoster(object canon.Value, name string) ([]domain.CandidateExecutionKey, error) {
	value, present := object.LookupMember(name)
	elements, array := value.Elements()
	if !present || !array || len(elements) < 2 || len(elements) > 4 {
		return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ROSTER"}
	}
	result := make([]domain.CandidateExecutionKey, len(elements))
	for index, element := range elements {
		raw, text := element.Text()
		if !text {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ROSTER"}
		}
		key, err := domain.ParseCandidateExecutionKey(raw)
		if err != nil || index > 0 && key.String() <= result[index-1].String() {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ROSTER"}
		}
		result[index] = key
	}
	return result, nil
}

func wireEntries(object canon.Value) ([]Entry, error) {
	value, present := object.LookupMember("entries")
	elements, array := value.Elements()
	if !present || !array || len(elements) > 4 {
		return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ENTRIES"}
	}
	result := make([]Entry, len(elements))
	for index, element := range elements {
		members, object := element.Members()
		candidateRaw, candidateOK := wireText(element, "candidate_execution_key")
		fingerprintRaw, fingerprintOK := wireText(element, "projection_fingerprint")
		batchRaw, batchOK := wireText(element, "stable_batch_digest")
		if !object || len(members) != 3 || !candidateOK || !fingerprintOK || !batchOK {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ENTRIES"}
		}
		candidate, candidateErr := domain.ParseCandidateExecutionKey(candidateRaw)
		fingerprint, fingerprintErr := domain.ParseProjectionFingerprint(fingerprintRaw)
		batch, batchErr := domain.ParseDigest(batchRaw)
		if candidateErr != nil || fingerprintErr != nil || batchErr != nil ||
			index > 0 && candidate.String() <= result[index-1].CandidateKey.String() {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_ENTRIES"}
		}
		result[index] = Entry{CandidateKey: candidate, ProjectionFingerprint: fingerprint, StableBatchDigest: batch}
	}
	return result, nil
}

func wireExclusions(object canon.Value) ([]Exclusion, error) {
	value, present := object.LookupMember("excluded_candidates")
	elements, array := value.Elements()
	if !present || !array || len(elements) > 4 {
		return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_EXCLUSIONS"}
	}
	result := make([]Exclusion, len(elements))
	for index, element := range elements {
		members, object := element.Members()
		candidateRaw, candidateOK := wireText(element, "candidate_execution_key")
		batchRaw, batchOK := wireText(element, "stable_batch_digest")
		classificationRaw, classificationOK := wireText(element, "classification")
		classification := observe.BatchStatus(classificationRaw)
		if !object || len(members) != 3 || !candidateOK || !batchOK || !classificationOK ||
			(classification != observe.Unstable && classification != observe.Uncomparable && classification != observe.Incomplete) {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_EXCLUSIONS"}
		}
		candidate, candidateErr := domain.ParseCandidateExecutionKey(candidateRaw)
		batch, batchErr := domain.ParseDigest(batchRaw)
		if candidateErr != nil || batchErr != nil ||
			index > 0 && candidate.String() <= result[index-1].CandidateKey.String() {
			return nil, &domain.Error{Code: "INVALID_OUTCOME_MAP_EXCLUSIONS"}
		}
		result[index] = Exclusion{CandidateKey: candidate, StableBatchDigest: batch, Classification: classification}
	}
	return result, nil
}

func validateWireDisposition(roster []domain.CandidateExecutionKey, entries []Entry, exclusions []Exclusion) error {
	if len(entries)+len(exclusions) != len(roster) {
		return &domain.Error{Code: "OUTCOME_MAP_DISPOSITION_MISMATCH"}
	}
	seen := make(map[domain.CandidateExecutionKey]struct{}, len(roster))
	batchSeen := make(map[domain.Digest]struct{}, len(roster))
	for _, entry := range entries {
		seen[entry.CandidateKey] = struct{}{}
		if _, duplicate := batchSeen[entry.StableBatchDigest]; duplicate {
			return &domain.Error{Code: "REUSED_OUTCOME_MAP_BATCH_EVIDENCE"}
		}
		batchSeen[entry.StableBatchDigest] = struct{}{}
	}
	for _, exclusion := range exclusions {
		if _, duplicate := seen[exclusion.CandidateKey]; duplicate {
			return &domain.Error{Code: "DUPLICATE_OUTCOME_MAP_DISPOSITION"}
		}
		seen[exclusion.CandidateKey] = struct{}{}
		if _, duplicate := batchSeen[exclusion.StableBatchDigest]; duplicate {
			return &domain.Error{Code: "REUSED_OUTCOME_MAP_BATCH_EVIDENCE"}
		}
		batchSeen[exclusion.StableBatchDigest] = struct{}{}
	}
	for _, candidate := range roster {
		if _, present := seen[candidate]; !present {
			return &domain.Error{Code: "OUTCOME_MAP_DISPOSITION_MISMATCH"}
		}
	}
	return nil
}

func strictDigestOrder(values []domain.Digest) bool {
	for index := 1; index < len(values); index++ {
		if values[index].String() <= values[index-1].String() {
			return false
		}
	}
	return true
}
