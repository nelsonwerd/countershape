package projectiontranslate

import (
	"bytes"
	"sort"

	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	maxConfirmedProjectionBytes = 600 * 1024
	maxConfirmedTupleBytes      = 512 * 1024
)

// ProjectionProof supplies exact historical bytes for one confirmed
// candidate. It is not authority until TranslateConfirmed verifies it against
// the opaque map-derived roster.
type ProjectionProof struct {
	CandidateExecutionKey domain.CandidateExecutionKey
	CanonicalProjection   []byte
}

// TranslatedOutcome is a defensive view of one proof-first result.
type TranslatedOutcome struct {
	candidate           domain.CandidateExecutionKey
	fingerprint         domain.ProjectionFingerprint
	canonicalProjection []byte
	tuple               Tuple
}

func (o TranslatedOutcome) CandidateExecutionKey() domain.CandidateExecutionKey { return o.candidate }
func (o TranslatedOutcome) ProjectionFingerprint() domain.ProjectionFingerprint { return o.fingerprint }
func (o TranslatedOutcome) CanonicalProjection() []byte {
	return append([]byte(nil), o.canonicalProjection...)
}
func (o TranslatedOutcome) Tuple() Tuple {
	return Tuple{profileDigest: o.tuple.profileDigest, fields: append([]Field(nil), o.tuple.fields...)}
}

// ConfirmedTranslations is the only portable tuple authority intended for
// Choice construction. It proves complete roster coverage and verification
// happened before any adapter translation.
type ConfirmedTranslations struct {
	resolved           Resolved
	outcomes           []TranslatedOutcome
	outcomeMapDigest   compare.OutcomeArtifactDigest
	preservationDigest compare.PreservationMapDigest
	seal               *confirmedSeal
}

type confirmedSeal struct{}

var confirmedAuthority = &confirmedSeal{}

// TranslateConfirmed verifies all proof candidates/fingerprints first, then
// resolves the exact profile, then translates those same copied proof bytes.
func TranslateConfirmed(
	binding domain.ProjectionDefinitionBinding,
	roster compare.ConfirmedProjectionRoster,
	proofs []ProjectionProof,
) (ConfirmedTranslations, error) {
	if !binding.Valid() || !roster.Valid() || roster.ProjectionDefinitionDigest() != binding.Digest() {
		return ConfirmedTranslations{}, refuse(CodeInvalidBinding, "", "confirmed roster and projection binding differ")
	}
	entries := roster.Entries()
	if len(proofs) == 0 || len(proofs) != len(entries) {
		return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proofs do not cover the confirmed roster exactly")
	}
	expected := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		expected[entry.CandidateExecutionKey().String()] = struct{}{}
	}
	verified := make([]TranslatedOutcome, len(proofs))
	seen := make(map[string]struct{}, len(proofs))
	projectionBytes := 0
	for index, proof := range proofs {
		candidateKey := proof.CandidateExecutionKey.String()
		if !proof.CandidateExecutionKey.Valid() {
			return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof has an invalid candidate key")
		}
		if _, duplicate := seen[candidateKey]; duplicate {
			return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof candidate occurs more than once")
		}
		if _, member := expected[candidateKey]; !member {
			return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof candidate is not in the confirmed roster")
		}
		frozenProjection := append([]byte(nil), proof.CanonicalProjection...)
		projectionBytes += len(frozenProjection)
		if projectionBytes > maxConfirmedProjectionBytes {
			return ConfirmedTranslations{}, refuse(CodeTranslationLimit, "", "confirmed projection proofs exceed the aggregate retained-byte ceiling")
		}
		fingerprint, err := roster.Verify(proof.CandidateExecutionKey, frozenProjection) // MUTANT_P07A_VERIFY_BEFORE_TRANSLATE
		if err != nil {
			return ConfirmedTranslations{}, refuse(CodeRosterMismatch, candidateKey, "projection proof does not match its confirmed fingerprint")
		}
		seen[candidateKey] = struct{}{}
		verified[index] = TranslatedOutcome{
			candidate: proof.CandidateExecutionKey, fingerprint: fingerprint,
			canonicalProjection: frozenProjection,
		}
	}
	if len(seen) != len(expected) {
		return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof coverage differs from the confirmed roster")
	}
	// Resolution deliberately follows complete proof/fingerprint verification.
	resolved, err := Resolve(binding)
	if err != nil {
		return ConfirmedTranslations{}, err
	}
	tupleBytes := 0
	for index := range verified {
		tuple, translateErr := resolved.Translate(verified[index].canonicalProjection)
		if translateErr != nil {
			return ConfirmedTranslations{}, translateErr
		}
		verified[index].tuple = tuple
		tupleBytes += tuple.RetainedBytes() + tuple.CompatibilityEncodedBytes()
		if tupleBytes > maxConfirmedTupleBytes {
			return ConfirmedTranslations{}, refuse(CodeTranslationLimit, "", "confirmed portable tuples exceed the aggregate retained/encoded ceiling")
		}
	}
	sort.Slice(verified, func(i, j int) bool {
		return verified[i].candidate.String() < verified[j].candidate.String()
	})
	return ConfirmedTranslations{
		resolved: resolved, outcomes: verified, outcomeMapDigest: roster.OutcomeMapDigest(),
		preservationDigest: roster.PreservationDigest(), seal: confirmedAuthority,
	}, nil
}

func (c ConfirmedTranslations) Valid() bool {
	if c.seal != confirmedAuthority || !c.resolved.Valid() || !c.outcomeMapDigest.Valid() ||
		!c.preservationDigest.Valid() || len(c.outcomes) < 2 {
		return false
	}
	projectionBytes := 0
	tupleBytes := 0
	for index, outcome := range c.outcomes {
		if !outcome.candidate.Valid() || !outcome.fingerprint.Valid() || !outcome.tuple.Valid() ||
			outcome.tuple.ProfileDigest() != c.resolved.profile.Digest() || len(outcome.canonicalProjection) == 0 ||
			(index > 0 && outcome.candidate.String() <= c.outcomes[index-1].candidate.String()) {
			return false
		}
		computed, err := domain.NewProjectionFingerprint(outcome.canonicalProjection)
		if err != nil || computed != outcome.fingerprint {
			return false
		}
		projectionBytes += len(outcome.canonicalProjection)
		tupleBytes += outcome.tuple.RetainedBytes() + outcome.tuple.CompatibilityEncodedBytes()
	}
	return projectionBytes <= maxConfirmedProjectionBytes && tupleBytes <= maxConfirmedTupleBytes
}

func (c ConfirmedTranslations) Profile() projectionprofile.Profile { return c.resolved.Profile() }
func (c ConfirmedTranslations) ExpectedStimulusKind() string {
	return c.resolved.ExpectedStimulusKind()
}
func (c ConfirmedTranslations) OutcomeMapDigest() compare.OutcomeArtifactDigest {
	return c.outcomeMapDigest
}
func (c ConfirmedTranslations) PreservationDigest() compare.PreservationMapDigest {
	return c.preservationDigest
}
func (c ConfirmedTranslations) Outcomes() []TranslatedOutcome {
	result := make([]TranslatedOutcome, len(c.outcomes))
	for index, outcome := range c.outcomes {
		result[index] = TranslatedOutcome{
			candidate: outcome.candidate, fingerprint: outcome.fingerprint,
			canonicalProjection: append([]byte(nil), outcome.canonicalProjection...), tuple: outcome.Tuple(),
		}
	}
	return result
}

// EqualProofBytes exists only for tests/consumers comparing an independently
// retained proof preimage. It never equates a tuple or digest with source bytes.
func (o TranslatedOutcome) EqualProofBytes(exact []byte) bool {
	return bytes.Equal(o.canonicalProjection, exact)
}
