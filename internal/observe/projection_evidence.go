package observe

import (
	"bytes"
	"encoding/base64"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const maxProjectionEvidenceCanonicalBytes = 1 << 20

// ProjectionDerivation is generic sealed lineage for an adapter-owned,
// validated operation/source-link transcript. The adapter remains responsible
// for proving its closed pipeline is complete; observe binds that exact
// transcript to the observation, definition, and projected bytes.
type ProjectionDerivation struct {
	digest                domain.Digest
	observationDigest     domain.Digest
	definitionDigest      domain.Digest
	projectionFingerprint domain.ProjectionFingerprint
	adapterEvidence       []byte
	canonicalBytes        []byte
}

type projectionDerivationIdentity struct {
	SchemaVersion              string `json:"schema_version"`
	Kind                       string `json:"kind"`
	CapturedObservationDigest  string `json:"captured_observation_digest"`
	ProjectionDefinitionDigest string `json:"projection_definition_digest"`
	ProjectionFingerprint      string `json:"projection_fingerprint"`
	AdapterEvidenceBase64      string `json:"adapter_evidence_base64"`
}

func NewProjectionDerivation(
	observationDigest domain.Digest,
	definitionDigest domain.Digest,
	canonicalProjection []byte,
	adapterEvidenceCanonical []byte,
) (ProjectionDerivation, error) {
	if !observationDigest.Valid() || !definitionDigest.Valid() {
		return ProjectionDerivation{}, &domain.Error{Code: "INVALID_PROJECTION_DERIVATION_LINEAGE"}
	}
	fingerprint, err := domain.NewProjectionFingerprint(canonicalProjection)
	if err != nil {
		return ProjectionDerivation{}, &domain.Error{Code: "NONCANONICAL_PROJECTION_DERIVATION_RESULT", Detail: err.Error()}
	}
	if err := requireExactCanonicalEvidence(adapterEvidenceCanonical); err != nil {
		return ProjectionDerivation{}, err
	}
	identity := projectionDerivationIdentity{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ProjectionDerivation",
		CapturedObservationDigest:  observationDigest.String(),
		ProjectionDefinitionDigest: definitionDigest.String(),
		ProjectionFingerprint:      fingerprint.String(),
		AdapterEvidenceBase64:      base64.StdEncoding.EncodeToString(adapterEvidenceCanonical),
	}
	digest, canonicalBytes, err := canon.DigestTyped("ProjectionDerivation", identity)
	if err != nil {
		return ProjectionDerivation{}, &domain.Error{Code: "INVALID_PROJECTION_DERIVATION_IDENTITY", Detail: err.Error()}
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return ProjectionDerivation{}, &domain.Error{Code: "INVALID_PROJECTION_DERIVATION_IDENTITY", Detail: err.Error()}
	}
	return ProjectionDerivation{
		digest:                parsed,
		observationDigest:     observationDigest,
		definitionDigest:      definitionDigest,
		projectionFingerprint: fingerprint,
		adapterEvidence:       append([]byte(nil), adapterEvidenceCanonical...),
		canonicalBytes:        append([]byte(nil), canonicalBytes...),
	}, nil
}

func (d ProjectionDerivation) Valid() bool {
	if !d.digest.Valid() || !d.observationDigest.Valid() || !d.definitionDigest.Valid() ||
		!d.projectionFingerprint.Valid() || len(d.canonicalBytes) == 0 ||
		requireExactCanonicalEvidence(d.adapterEvidence) != nil {
		return false
	}
	identity := projectionDerivationIdentity{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ProjectionDerivation",
		CapturedObservationDigest:  d.observationDigest.String(),
		ProjectionDefinitionDigest: d.definitionDigest.String(),
		ProjectionFingerprint:      d.projectionFingerprint.String(),
		AdapterEvidenceBase64:      base64.StdEncoding.EncodeToString(d.adapterEvidence),
	}
	digest, canonicalBytes, err := canon.DigestTyped("ProjectionDerivation", identity)
	return err == nil && digest.String() == d.digest.String() && bytes.Equal(canonicalBytes, d.canonicalBytes)
}

func (d ProjectionDerivation) Digest() domain.Digest { return d.digest }

func (d ProjectionDerivation) CanonicalBytes() []byte {
	return append([]byte(nil), d.canonicalBytes...)
}

// ProjectionRejectionEvidence binds adapter-owned typed rejection evidence to
// the captured observation and exact projection definition. It carries no
// fingerprint and can only enter a control trial.
type ProjectionRejectionEvidence struct {
	digest                domain.Digest
	worldDigest           domain.Digest
	attemptArtifactDigest domain.Digest
	candidateKey          domain.CandidateExecutionKey
	capturePolicyDigest   domain.Digest
	observationDigest     domain.Digest
	definitionDigest      domain.Digest
	adapterEvidence       []byte
	canonicalBytes        []byte
}

type projectionRejectionIdentity struct {
	SchemaVersion              string `json:"schema_version"`
	Kind                       string `json:"kind"`
	WorldDigest                string `json:"world_instance_digest"`
	AttemptArtifactDigest      string `json:"attempt_artifact_digest"`
	CandidateExecutionKey      string `json:"candidate_execution_key"`
	CapturePolicyDigest        string `json:"capture_policy_digest"`
	CapturedObservationDigest  string `json:"captured_observation_digest"`
	ProjectionDefinitionDigest string `json:"projection_definition_digest"`
	AdapterEvidenceBase64      string `json:"adapter_evidence_base64"`
}

// ProjectionRejectionLineage is the adapter-established capture authority
// which a scheduled world must match before rejection evidence can enter an
// admitted trial. The observation digest alone is intentionally insufficient:
// it cannot be cross-paired with another world, attempt, candidate, or capture
// policy even when the projection definition happens to be shared.
type ProjectionRejectionLineage struct {
	WorldDigest                domain.Digest
	AttemptArtifactDigest      domain.Digest
	CandidateKey               domain.CandidateExecutionKey
	CapturePolicyDigest        domain.Digest
	ObservationDigest          domain.Digest
	ProjectionDefinitionDigest domain.Digest
}

func NewProjectionRejectionEvidence(
	lineage ProjectionRejectionLineage,
	adapterEvidenceCanonical []byte,
) (ProjectionRejectionEvidence, error) {
	if !lineage.WorldDigest.Valid() || !lineage.AttemptArtifactDigest.Valid() ||
		!lineage.CandidateKey.Valid() || !lineage.CapturePolicyDigest.Valid() ||
		!lineage.ObservationDigest.Valid() || !lineage.ProjectionDefinitionDigest.Valid() {
		return ProjectionRejectionEvidence{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION_LINEAGE"}
	}
	if err := requireExactCanonicalEvidence(adapterEvidenceCanonical); err != nil {
		return ProjectionRejectionEvidence{}, err
	}
	identity := projectionRejectionIdentity{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ProjectionRejectionEvidence",
		WorldDigest:                lineage.WorldDigest.String(),
		AttemptArtifactDigest:      lineage.AttemptArtifactDigest.String(),
		CandidateExecutionKey:      lineage.CandidateKey.String(),
		CapturePolicyDigest:        lineage.CapturePolicyDigest.String(),
		CapturedObservationDigest:  lineage.ObservationDigest.String(),
		ProjectionDefinitionDigest: lineage.ProjectionDefinitionDigest.String(),
		AdapterEvidenceBase64:      base64.StdEncoding.EncodeToString(adapterEvidenceCanonical),
	}
	digest, canonicalBytes, err := canon.DigestTyped("ProjectionRejectionEvidence", identity)
	if err != nil {
		return ProjectionRejectionEvidence{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION_IDENTITY", Detail: err.Error()}
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return ProjectionRejectionEvidence{}, &domain.Error{Code: "INVALID_PROJECTION_REJECTION_IDENTITY", Detail: err.Error()}
	}
	return ProjectionRejectionEvidence{
		digest:                parsed,
		worldDigest:           lineage.WorldDigest,
		attemptArtifactDigest: lineage.AttemptArtifactDigest,
		candidateKey:          lineage.CandidateKey,
		capturePolicyDigest:   lineage.CapturePolicyDigest,
		observationDigest:     lineage.ObservationDigest,
		definitionDigest:      lineage.ProjectionDefinitionDigest,
		adapterEvidence:       append([]byte(nil), adapterEvidenceCanonical...),
		canonicalBytes:        append([]byte(nil), canonicalBytes...),
	}, nil
}

func (e ProjectionRejectionEvidence) Valid() bool {
	if !e.digest.Valid() || !e.worldDigest.Valid() || !e.attemptArtifactDigest.Valid() ||
		!e.candidateKey.Valid() || !e.capturePolicyDigest.Valid() ||
		!e.observationDigest.Valid() || !e.definitionDigest.Valid() ||
		len(e.canonicalBytes) == 0 || requireExactCanonicalEvidence(e.adapterEvidence) != nil {
		return false
	}
	identity := projectionRejectionIdentity{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ProjectionRejectionEvidence",
		WorldDigest:                e.worldDigest.String(),
		AttemptArtifactDigest:      e.attemptArtifactDigest.String(),
		CandidateExecutionKey:      e.candidateKey.String(),
		CapturePolicyDigest:        e.capturePolicyDigest.String(),
		CapturedObservationDigest:  e.observationDigest.String(),
		ProjectionDefinitionDigest: e.definitionDigest.String(),
		AdapterEvidenceBase64:      base64.StdEncoding.EncodeToString(e.adapterEvidence),
	}
	digest, canonicalBytes, err := canon.DigestTyped("ProjectionRejectionEvidence", identity)
	return err == nil && digest.String() == e.digest.String() && bytes.Equal(canonicalBytes, e.canonicalBytes)
}

func (e ProjectionRejectionEvidence) Digest() domain.Digest      { return e.digest }
func (e ProjectionRejectionEvidence) WorldDigest() domain.Digest { return e.worldDigest }
func (e ProjectionRejectionEvidence) AttemptArtifactDigest() domain.Digest {
	return e.attemptArtifactDigest
}
func (e ProjectionRejectionEvidence) CandidateKey() domain.CandidateExecutionKey {
	return e.candidateKey
}
func (e ProjectionRejectionEvidence) CapturePolicyDigest() domain.Digest {
	return e.capturePolicyDigest
}
func (e ProjectionRejectionEvidence) ObservationDigest() domain.Digest { return e.observationDigest }
func (e ProjectionRejectionEvidence) DefinitionDigest() domain.Digest  { return e.definitionDigest }
func (e ProjectionRejectionEvidence) CanonicalBytes() []byte {
	return append([]byte(nil), e.canonicalBytes...)
}

func requireExactCanonicalEvidence(value []byte) error {
	if len(value) == 0 || len(value) > maxProjectionEvidenceCanonicalBytes {
		return &domain.Error{Code: "INVALID_PROJECTION_EVIDENCE_SIZE"}
	}
	canonical, err := canon.Canonicalize(value)
	if err != nil || !bytes.Equal(canonical, value) {
		detail := "adapter evidence is not exact canonical JSON"
		if err != nil {
			detail = err.Error()
		}
		return &domain.Error{Code: "NONCANONICAL_PROJECTION_EVIDENCE", Detail: detail}
	}
	return nil
}
