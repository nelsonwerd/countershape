package choice

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

func TestTranslateConfirmedBindsVerifiedProofsBeforePortableTuples(t *testing.T) {
	binding := p07CLIProjectionBinding(t, cli.CLIFieldStdoutBytes)
	firstBytes := p07CLIBytesProjection(t, 0x00)
	secondBytes := p07CLIBytesProjection(t, 0xff)
	inputs := []ProjectionProofInput{
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-proof-a", firstBytes),
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-proof-b", secondBytes),
	}
	roster := testProjectionRoster(t, inputs)
	proofs := []projectiontranslate.ProjectionProof{
		{CandidateExecutionKey: inputs[1].CandidateExecutionKey, CanonicalProjection: append([]byte(nil), inputs[1].CanonicalProjection...)},
		{CandidateExecutionKey: inputs[0].CandidateExecutionKey, CanonicalProjection: append([]byte(nil), inputs[0].CanonicalProjection...)},
	}
	confirmed, err := projectiontranslate.TranslateConfirmed(binding, roster, proofs)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed.Valid() || confirmed.ExpectedStimulusKind() != "CLIStimulus" ||
		confirmed.OutcomeMapDigest() != roster.OutcomeMapDigest() || confirmed.PreservationDigest() != roster.PreservationDigest() {
		t.Fatal("confirmed translations omitted exact roster/profile authority")
	}
	outcomes := confirmed.Outcomes()
	if len(outcomes) != 2 || outcomes[0].CandidateExecutionKey().String() >= outcomes[1].CandidateExecutionKey().String() {
		t.Fatalf("confirmed outcomes are not in canonical candidate order: %#v", outcomes)
	}
	for _, outcome := range outcomes {
		if !outcome.Tuple().Valid() || outcome.Tuple().ProfileDigest() != confirmed.Profile().Digest() {
			t.Fatal("confirmed outcome tuple lost its resolved profile")
		}
		fingerprint, fingerprintErr := domain.NewProjectionFingerprint(outcome.CanonicalProjection())
		if fingerprintErr != nil || fingerprint != outcome.ProjectionFingerprint() {
			t.Fatalf("confirmed outcome fingerprint = (%s,%v)", outcome.ProjectionFingerprint().String(), fingerprintErr)
		}
	}
	original := append([]byte(nil), proofs[0].CanonicalProjection...)
	proofs[0].CanonicalProjection[0] ^= 1
	var retained bool
	for _, outcome := range confirmed.Outcomes() {
		if outcome.CandidateExecutionKey() == proofs[0].CandidateExecutionKey {
			retained = outcome.EqualProofBytes(original)
		}
	}
	if !retained {
		t.Fatal("confirmed translation retained caller-owned proof storage")
	}

	reversed, err := projectiontranslate.TranslateConfirmed(binding, roster, []projectiontranslate.ProjectionProof{
		{CandidateExecutionKey: inputs[0].CandidateExecutionKey, CanonicalProjection: inputs[0].CanonicalProjection},
		{CandidateExecutionKey: inputs[1].CandidateExecutionKey, CanonicalProjection: inputs[1].CanonicalProjection},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := range outcomes {
		other := reversed.Outcomes()[index]
		if outcomes[index].CandidateExecutionKey() != other.CandidateExecutionKey() ||
			outcomes[index].ProjectionFingerprint() != other.ProjectionFingerprint() ||
			!outcomes[index].Tuple().Equal(other.Tuple()) {
			t.Fatal("proof input order changed confirmed translation identity")
		}
	}
}

func TestTranslateConfirmedVerifiesCompleteRosterBeforeAnyTranslation(t *testing.T) {
	binding := p07CLIProjectionBinding(t, cli.CLIFieldStdoutBytes)
	malformedForTranslator := []byte(`{}`)
	secondBytes := p07CLIBytesProjection(t, 0x02)
	inputs := []ProjectionProofInput{
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-order-a", malformedForTranslator),
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-order-b", secondBytes),
	}
	roster := testProjectionRoster(t, inputs)
	_, err := projectiontranslate.TranslateConfirmed(binding, roster, []projectiontranslate.ProjectionProof{
		{CandidateExecutionKey: inputs[0].CandidateExecutionKey, CanonicalProjection: malformedForTranslator},
		{CandidateExecutionKey: inputs[1].CandidateExecutionKey, CanonicalProjection: p07CLIBytesProjection(t, 0x03)},
	})
	if !projectiontranslate.IsCode(err, projectiontranslate.CodeRosterMismatch) {
		t.Fatalf("TranslateConfirmed() error = %v; proof verification must precede the earlier translation failure", err)
	}
}

func TestTranslateConfirmedRefusesCoverageAndBindingSubstitution(t *testing.T) {
	binding := p07CLIProjectionBinding(t, cli.CLIFieldStdoutBytes)
	inputs := []ProjectionProofInput{
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-coverage-a", p07CLIBytesProjection(t, 0x10)),
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-coverage-b", p07CLIBytesProjection(t, 0x11)),
	}
	roster := testProjectionRoster(t, inputs)
	valid := []projectiontranslate.ProjectionProof{
		{CandidateExecutionKey: inputs[0].CandidateExecutionKey, CanonicalProjection: inputs[0].CanonicalProjection},
		{CandidateExecutionKey: inputs[1].CandidateExecutionKey, CanonicalProjection: inputs[1].CanonicalProjection},
	}
	foreign := trustedBytesInputForProjection(t, binding.Digest(), "candidate:p07-coverage-foreign", p07CLIBytesProjection(t, 0x12))
	tests := []struct {
		name   string
		proofs []projectiontranslate.ProjectionProof
	}{
		{name: "missing", proofs: valid[:1]},
		{name: "duplicate", proofs: []projectiontranslate.ProjectionProof{valid[0], valid[0]}},
		{name: "foreign", proofs: []projectiontranslate.ProjectionProof{valid[0], {CandidateExecutionKey: foreign.CandidateExecutionKey, CanonicalProjection: foreign.CanonicalProjection}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := projectiontranslate.TranslateConfirmed(binding, roster, test.proofs); !projectiontranslate.IsCode(err, projectiontranslate.CodeRosterMismatch) {
				t.Fatalf("TranslateConfirmed() error = %v", err)
			}
		})
	}
	other := p07CLIProjectionBinding(t, cli.CLIFieldStderrText)
	if _, err := projectiontranslate.TranslateConfirmed(other, roster, valid); !projectiontranslate.IsCode(err, projectiontranslate.CodeInvalidBinding) {
		t.Fatalf("TranslateConfirmed(binding substitution) error = %v", err)
	}
}

func p07CLIProjectionBinding(t testing.TB, fields ...cli.CLIFieldID) domain.ProjectionDefinitionBinding {
	t.Helper()
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	binding := definition.Binding()
	if previous, loaded := choiceTestProjectionBindings.LoadOrStore(binding.Digest().String(), binding); loaded {
		stored, ok := previous.(domain.ProjectionDefinitionBinding)
		if !ok || stored.Digest() != binding.Digest() || !bytes.Equal(stored.CanonicalBytes(), binding.CanonicalBytes()) {
			t.Fatalf("P07 projection binding fixture collision for %s", binding.Digest())
		}
	}
	return binding
}

func p07CLIBytesProjection(t testing.TB, value byte) []byte {
	t.Helper()
	exact, err := canon.CanonicalizeTyped(map[string]any{
		"fields": []map[string]any{{
			"field_id": string(cli.CLIFieldStdoutBytes),
			"value":    map[string]any{"base64": []string{"AA==", "AQ==", "Ag==", "Aw=="}[int(value)%4], "tag": "BYTES"},
		}},
		"kind": "CLIProjection", "schema_version": "cli-projection/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return exact
}
