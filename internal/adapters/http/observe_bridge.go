package http

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

// PrepareStructuralCapture is the sole HTTP-to-observe admission seam. The
// adapter derivation remains independently reconstructable; observe receives
// no mutable HTTP policy or parser state.
func PrepareStructuralCapture(world domain.WorldInstance, attempt domain.FinalizedAttempt, observation HTTPCapturedObservation, result HTTPProjectionResult) (observe.StructuralCapture, error) {
	if !observation.Valid() || !result.Valid() || result.observationDigest != observation.digest || result.definitionDigest != observation.projectionDefinitionDigest || world.Digest() != observation.worldDigest || world.PlanDigest() != observation.planDigest || world.StimulusDigest() != observation.stimulusDigest || world.AttemptArtifactDigest() != observation.attemptDigest || attempt.ArtifactDigest() != observation.attemptDigest {
		return observe.StructuralCapture{}, refuse(CodeCaptureLineageMismatch, "projection result cannot be paired with this world observation")
	}
	projectionDigest, err := digestBytes("HTTPProjectionCanonicalBytes", result.projectionBytes)
	if err != nil {
		return observe.StructuralCapture{}, err
	}
	traceIDs := traceIdentities(result.transcript)
	identity := struct {
		SchemaVersion, Kind, ObservationDigest, DefinitionDigest, ProjectionByteDigest string
		Transcript                                                                     any
	}{domain.SchemaVersion, "HTTPProjectionDerivation", observation.digest.String(), observation.projectionDefinitionDigest.String(), projectionDigest.String(), traceIDs}
	expectedDigest, expectedCanonical, err := digestTyped("HTTPProjectionDerivation", identity)
	if err != nil || expectedDigest != result.digest || !bytes.Equal(expectedCanonical, result.canonicalBytes) {
		return observe.StructuralCapture{}, refuse(CodeCaptureLineageMismatch, "projection behavior bytes differ from adapter derivation")
	}
	derivation, err := observe.NewProjectionDerivation(observation.digest, observation.projectionDefinitionDigest, result.projectionBytes, result.canonicalBytes)
	if err != nil {
		return observe.StructuralCapture{}, err
	}
	return observe.NewStructuralCapture(world, attempt, observation.digest, result.projectionBytes, derivation)
}

// PrepareProjectionRejectionEvidence preserves the same lineage as a
// successful capture while keeping controlled projection failures out of the
// behavior-comparison path.
func PrepareProjectionRejectionEvidence(observation HTTPCapturedObservation, rejection *ProjectionRejection) (observe.ProjectionRejectionEvidence, error) {
	if !observation.Valid() || rejection == nil || !rejection.Valid() || rejection.observationDigest != observation.digest || rejection.definitionDigest != observation.projectionDefinitionDigest || len(rejection.canonicalBytes) == 0 {
		return observe.ProjectionRejectionEvidence{}, refuse(CodeCaptureLineageMismatch, "projection rejection cannot be paired with this observation")
	}
	canonical, err := canon.Canonicalize(rejection.canonicalBytes)
	if err != nil || !bytes.Equal(canonical, rejection.canonicalBytes) {
		return observe.ProjectionRejectionEvidence{}, refuse(CodeCaptureEvidenceInvalid, "projection rejection evidence is not exact canonical bytes")
	}
	return observe.NewProjectionRejectionEvidence(observe.ProjectionRejectionLineage{
		WorldDigest:                observation.worldDigest,
		AttemptArtifactDigest:      observation.attemptDigest,
		CandidateKey:               observation.candidateKey,
		CapturePolicyDigest:        observation.capturePolicyDigest,
		ObservationDigest:          observation.digest,
		ProjectionDefinitionDigest: observation.projectionDefinitionDigest,
	}, rejection.canonicalBytes)
}
