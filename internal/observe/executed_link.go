package observe

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/world"
)

// PreparedExecutionLink proves that an adapter's PreparedTrial was derived
// from the exact opaque world.Result returned beside it. It prevents a fresh
// physical result from being paired with a copied prepared observation.
type PreparedExecutionLink struct {
	worldDigest         domain.Digest
	attemptDigest       domain.Digest
	observationDigest   domain.Digest
	hasObservation      bool
	candidateKey        domain.CandidateExecutionKey
	fingerprint         domain.ProjectionFingerprint
	canonicalProjection []byte
	hasProjection       bool
}

func (p PreparedTrial) BindExecuted(result world.Result) (PreparedExecutionLink, error) {
	worldInstance := result.World()
	finalized := result.FinalizedAttempt()
	if err := validatePreparedCommon(p); err != nil {
		return PreparedExecutionLink{}, err
	}
	if p.world.Digest() != worldInstance.Digest() ||
		p.attempt.ArtifactDigest() != finalized.ArtifactDigest() ||
		p.world.AttemptArtifactDigest() != finalized.ArtifactDigest() ||
		p.world.Purpose() != finalized.Purpose() {
		return PreparedExecutionLink{}, &domain.Error{Code: "PREPARED_EXECUTION_RESULT_MISMATCH"}
	}
	link := PreparedExecutionLink{
		worldDigest: worldInstance.Digest(), attemptDigest: finalized.ArtifactDigest(),
		candidateKey: worldInstance.CandidateKey(),
	}
	switch p.kind {
	case preparedCaptured:
		link.observationDigest = p.capture.observationDigest
		link.hasObservation = true
		link.fingerprint = p.capture.fingerprint
		link.canonicalProjection = p.capture.CanonicalProjection()
		link.hasProjection = true
	case preparedProjectionRejected:
		link.observationDigest = p.rejection.observationDigest
		link.hasObservation = true
	case preparedAttemptControlled:
	default:
		return PreparedExecutionLink{}, &domain.Error{Code: "UNKNOWN_PREPARED_TRIAL_KIND"}
	}
	return link, nil
}

func (l PreparedExecutionLink) Valid() bool {
	if !l.worldDigest.Valid() || !l.attemptDigest.Valid() || !l.candidateKey.Valid() ||
		l.hasObservation != l.observationDigest.Valid() || l.hasProjection != l.fingerprint.Valid() {
		return false
	}
	if !l.hasProjection {
		return len(l.canonicalProjection) == 0
	}
	computed, err := domain.NewProjectionFingerprint(l.canonicalProjection)
	return err == nil && computed == l.fingerprint
}

func (l PreparedExecutionLink) WorldDigest() domain.Digest   { return l.worldDigest }
func (l PreparedExecutionLink) AttemptDigest() domain.Digest { return l.attemptDigest }
func (l PreparedExecutionLink) ObservationDigest() (domain.Digest, bool) {
	return l.observationDigest, l.hasObservation
}

func (l PreparedExecutionLink) CandidateExecutionKey() domain.CandidateExecutionKey {
	return l.candidateKey
}

func (l PreparedExecutionLink) Projection() (domain.ProjectionFingerprint, []byte, bool) {
	return l.fingerprint, append([]byte(nil), l.canonicalProjection...), l.hasProjection
}

func (l PreparedExecutionLink) SameProjection(other PreparedExecutionLink) bool {
	return l.hasProjection && other.hasProjection && l.candidateKey == other.candidateKey &&
		l.fingerprint == other.fingerprint && bytes.Equal(l.canonicalProjection, other.canonicalProjection)
}
