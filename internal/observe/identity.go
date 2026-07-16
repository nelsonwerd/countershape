package observe

import (
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type scheduleIdentity struct {
	Phase          string `json:"phase"`
	Rotation       string `json:"rotation"`
	StartOffset    int    `json:"start_offset"`
	ScheduleDigest string `json:"schedule_digest"`
	Ordinals       []int  `json:"ordinals"`
}

type capturedTrialIdentity struct {
	ObservationDigest          string `json:"captured_observation_digest"`
	CapturePolicyDigest        string `json:"capture_policy_digest"`
	ProjectionDefinitionDigest string `json:"projection_definition_digest"`
	ProjectionResult           string `json:"projection_result_digest"`
	ProjectionFingerprint      string `json:"projection_fingerprint"`
	ProjectionDerivationDigest string `json:"projection_derivation_digest"`
}

type controlTrialIdentity struct {
	Reasons                   []domain.ControlReason `json:"reason_codes"`
	CapturedObservationDigest string                 `json:"captured_observation_digest"`
	ProjectionRejectionDigest string                 `json:"projection_rejection_evidence_digest"`
}

type trialIdentity struct {
	Disposition               string                 `json:"disposition"`
	WorldInstanceDigest       string                 `json:"world_instance_digest"`
	AttemptArtifactDigest     string                 `json:"attempt_artifact_digest"`
	ComparisonAdmissionDigest string                 `json:"comparison_admission_digest"`
	Captured                  *capturedTrialIdentity `json:"captured"`
	Control                   *controlTrialIdentity  `json:"control"`
	AdmissionTokenDigest      string                 `json:"admission_token_digest"`
}

type batchIdentity struct {
	SchemaVersion                string           `json:"schema_version"`
	Kind                         string           `json:"kind"`
	CandidateExecutionKey        string           `json:"candidate_execution_key"`
	WorldPlanDigest              string           `json:"world_plan_digest"`
	StimulusDigest               string           `json:"stimulus_digest"`
	ComparisonEnvelopeDigest     string           `json:"comparison_envelope_digest"`
	CapturePolicyDigest          string           `json:"capture_policy_digest"`
	ProjectionDefinitionDigest   string           `json:"projection_definition_digest"`
	ComparisonAdmissionDigests   []string         `json:"comparison_admission_digests"`
	ComparisonBasisDigest        string           `json:"comparison_basis_digest"`
	ComparisonAdmissionRoster    []string         `json:"comparison_admission_candidate_roster"`
	RequiredFreshTrials          int              `json:"required_fresh_trials"`
	Schedule                     scheduleIdentity `json:"schedule"`
	Trials                       []trialIdentity  `json:"trials"`
	CapturedObservationDigests   []string         `json:"captured_observation_digests"`
	Classification               any              `json:"classification"`
	DuplicateEvidenceWithinBatch bool             `json:"duplicate_evidence_within_batch"`
}

func trialIdentityOf(trial TrialFact) trialIdentity {
	identity := trialIdentity{
		Disposition:               string(trial.kind),
		WorldInstanceDigest:       trial.world.Digest().String(),
		AttemptArtifactDigest:     trial.attempt.ArtifactDigest().String(),
		ComparisonAdmissionDigest: trial.admission.ComparisonAdmissionDigest().String(),
		AdmissionTokenDigest:      trial.admission.Digest().String(),
	}
	if trial.kind == trialCaptured {
		identity.Captured = &capturedTrialIdentity{
			ObservationDigest:          trial.capture.observationDigest.String(),
			CapturePolicyDigest:        trial.capture.capturePolicyDigest.String(),
			ProjectionDefinitionDigest: trial.capture.projectionDefinitionDigest.String(),
			ProjectionResult:           trial.capture.projectionDigest.String(),
			ProjectionFingerprint:      trial.capture.fingerprint.String(),
			ProjectionDerivationDigest: trial.capture.derivationDigest.String(),
		}
	} else {
		identity.Control = &controlTrialIdentity{Reasons: append([]domain.ControlReason(nil), trial.controls...)}
		if trial.rejection.Valid() {
			identity.Control.CapturedObservationDigest = trial.rejection.observationDigest.String()
			identity.Control.ProjectionRejectionDigest = trial.rejection.digest.String()
		}
	}
	return identity
}

func classificationIdentity(classification Classification) any {
	switch classification.status {
	case ObservedStable:
		return struct {
			Status      string `json:"status"`
			Eligible    int    `json:"eligible_trials"`
			Required    int    `json:"required_trials"`
			Fingerprint string `json:"projection_fingerprint"`
			Bounded     string `json:"bounded_label"`
		}{string(classification.status), classification.eligibleTrials, classification.requiredTrials, classification.fingerprint.String(), classification.BoundedLabel()}
	case Unstable:
		histogram := make([]struct {
			Fingerprint string `json:"projection_fingerprint"`
			Count       int    `json:"count"`
		}, len(classification.histogram))
		for index, bin := range classification.histogram {
			histogram[index].Fingerprint = bin.Fingerprint.String()
			histogram[index].Count = bin.Count
		}
		return struct {
			Status    string `json:"status"`
			Eligible  int    `json:"eligible_trials"`
			Histogram any    `json:"histogram"`
		}{string(classification.status), classification.eligibleTrials, histogram}
	case Uncomparable:
		return struct {
			Status  string                 `json:"status"`
			Reasons []domain.ControlReason `json:"reason_codes"`
		}{string(classification.status), classification.reasons}
	default:
		return struct {
			Status   string `json:"status"`
			Eligible int    `json:"eligible_trials"`
			Required int    `json:"required_trials"`
			Reason   string `json:"reason"`
		}{string(classification.status), classification.eligibleTrials, classification.requiredTrials, "BUDGET_EXHAUSTED"}
	}
}

func digestBatch(identity batchIdentity) (domain.Digest, []byte, error) {
	identity.SchemaVersion = domain.SchemaVersion
	identity.Kind = "StableBatch"
	digest, canonicalBytes, err := canon.DigestTyped("StableBatch", identity)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	return parsed, canonicalBytes, err
}
