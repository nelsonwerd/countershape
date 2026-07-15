package cli

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

// PrepareStructuralCapture is the strict adapter-to-generic handoff. It
// revalidates the exact observation, definition, behavior bytes, and complete
// adapter derivation transcript before observe can construct projection truth.
func PrepareStructuralCapture(
	world domain.WorldInstance,
	attempt domain.FinalizedAttempt,
	observation CLICapturedObservation,
	result CLIProjectionResult,
) (observe.StructuralCapture, error) {
	if !observation.Valid() || !result.derivation.Valid() ||
		result.derivation.observationDigest != observation.digest ||
		result.derivation.definitionDigest != observation.projectionDefinitionDigest ||
		!bytes.Equal(result.canonicalBytes, result.derivation.canonicalBytes) ||
		world.Digest() != observation.worldDigest || world.PlanDigest() != observation.planDigest ||
		world.StimulusDigest() != observation.stimulusDigest ||
		world.AttemptArtifactDigest() != observation.attemptDigest ||
		attempt.ArtifactDigest() != observation.attemptDigest {
		return observe.StructuralCapture{}, refuse(CodeCaptureLineageMismatch, "projection result cannot be paired with this world observation")
	}
	projectionDigest, _, err := cliDigestBytes("CLIProjectionCanonicalBytes", result.projectionBytes)
	if err != nil || projectionDigest != result.derivation.projectionByteDigest {
		return observe.StructuralCapture{}, refuse(CodeCaptureLineageMismatch, "projection behavior bytes differ from adapter derivation")
	}
	derivation, err := observe.NewProjectionDerivation(
		observation.digest,
		observation.projectionDefinitionDigest,
		result.projectionBytes,
		result.canonicalBytes,
	)
	if err != nil {
		return observe.StructuralCapture{}, err
	}
	return observe.NewStructuralCapture(world, attempt, observation.digest, result.projectionBytes, derivation)
}

// PrepareProjectionRejectionEvidence is the only trusted CLI rejection
// handoff. A rejection from another observation or definition cannot be paired
// with this capture, even if its machine code happens to match.
func PrepareProjectionRejectionEvidence(
	observation CLICapturedObservation,
	rejection *ProjectionRejection,
) (observe.ProjectionRejectionEvidence, error) {
	if !observation.Valid() || rejection == nil || !rejection.Valid() ||
		rejection.observationDigest != observation.digest ||
		rejection.definitionDigest != observation.projectionDefinitionDigest || len(rejection.canonicalBytes) == 0 {
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
