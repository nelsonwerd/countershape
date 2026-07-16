package publication

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestIssueChoicepointBindsCanonicalBodyToExactConfirmation(t *testing.T) {
	predecessor := publicationTestDigest(t, "confirmation-a")
	other := publicationTestDigest(t, "confirmation-b")
	digest, canonical := choicepointBody(t, predecessor)

	authority, err := IssueChoicepoint(digest, predecessor, canonical)
	if err != nil || !authority.Valid() {
		t.Fatalf("exact Choicepoint authority was refused: %v", err)
	}
	if _, err := IssueChoicepoint(digest, other, canonical); err == nil {
		t.Fatal("Choicepoint body was sealed to a detached confirmation predecessor")
	}
	authority.predecessor = other
	if authority.Valid() {
		t.Fatal("mutated Choicepoint predecessor remained valid")
	}
}

func TestIssueChoicepointRejectsWrongKindAndNoncanonicalBytes(t *testing.T) {
	predecessor := publicationTestDigest(t, "confirmation-a")
	wrongKindDigest, wrongKind := publicationBody(t, "Choicepoint", struct {
		FreshConfirmationDigest string `json:"fresh_confirmation_digest"`
		Kind                    string `json:"kind"`
	}{predecessor.String(), "DecisionRecord"})
	if _, err := IssueChoicepoint(wrongKindDigest, predecessor, wrongKind); err == nil {
		t.Fatal("wrong-kind body received Choicepoint publication authority")
	}

	_, canonical := choicepointBody(t, predecessor)
	noncanonical := append([]byte("\n"), canonical...)
	raw, err := canon.DigestBytes("Choicepoint", noncanonical)
	if err != nil {
		t.Fatal(err)
	}
	noncanonicalDigest, err := domain.ParseDigest(raw.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := IssueChoicepoint(noncanonicalDigest, predecessor, noncanonical); err == nil {
		t.Fatal("noncanonical bytes received Choicepoint publication authority")
	}
}

func TestIssueRulingBindsCanonicalBodyPredecessorAndDurableAction(t *testing.T) {
	predecessor := publicationTestDigest(t, "choicepoint-a")
	other := publicationTestDigest(t, "choicepoint-b")
	digest, canonical := decisionBody(t, predecessor, "DEFER")

	authority, err := IssueRuling(digest, predecessor, "DEFER", canonical)
	if err != nil || !authority.Valid() {
		t.Fatalf("exact DecisionRecord authority was refused: %v", err)
	}
	for _, attack := range []struct {
		name        string
		predecessor domain.Digest
		action      string
	}{
		{"detached predecessor", other, "DEFER"},
		{"detached action", predecessor, "REJECT_ALL"},
	} {
		t.Run(attack.name, func(t *testing.T) {
			if _, err := IssueRuling(digest, attack.predecessor, attack.action, canonical); err == nil {
				t.Fatal("detached DecisionRecord authority was issued")
			}
		})
	}
	refineDigest, refineCanonical := decisionBody(t, predecessor, "REFINE")
	if _, err := IssueRuling(refineDigest, predecessor, "DEFER", refineCanonical); err == nil {
		t.Fatal("REFINE DecisionRecord bytes were relabeled as DEFER")
	}
	if _, err := IssueRuling(refineDigest, predecessor, "REFINE", refineCanonical); err == nil {
		t.Fatal("REFINE DecisionRecord received durable U6 ruling authority")
	}
	authority.action = "REJECT_ALL"
	if authority.Valid() {
		t.Fatal("mutated DecisionRecord action remained valid")
	}
}

func TestIssueRulingRejectsWrongKindAndNoncanonicalBytes(t *testing.T) {
	predecessor := publicationTestDigest(t, "choicepoint-a")
	wrongKindDigest, wrongKind := publicationBody(t, "DecisionRecord", struct {
		Action            string `json:"action"`
		ChoicepointDigest string `json:"choicepoint_digest"`
		Kind              string `json:"kind"`
	}{"DEFER", predecessor.String(), "Choicepoint"})
	if _, err := IssueRuling(wrongKindDigest, predecessor, "DEFER", wrongKind); err == nil {
		t.Fatal("wrong-kind body received DecisionRecord publication authority")
	}

	_, canonical := decisionBody(t, predecessor, "DEFER")
	noncanonical := append(append([]byte(nil), canonical...), '\n')
	raw, err := canon.DigestBytes("DecisionRecord", noncanonical)
	if err != nil {
		t.Fatal(err)
	}
	noncanonicalDigest, err := domain.ParseDigest(raw.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := IssueRuling(noncanonicalDigest, predecessor, "DEFER", noncanonical); err == nil {
		t.Fatal("noncanonical bytes received DecisionRecord publication authority")
	}
}

func TestIssueRulingAcceptsExactlyTheFourDurableActions(t *testing.T) {
	predecessor := publicationTestDigest(t, "choicepoint-a")
	for _, action := range []string{"ALLOW_OBSERVED", "CUSTOM_EXPECTATION", "REJECT_ALL", "DEFER"} {
		t.Run(action, func(t *testing.T) {
			digest, canonical := decisionBody(t, predecessor, action)
			authority, err := IssueRuling(digest, predecessor, action, canonical)
			if err != nil || !authority.Valid() {
				t.Fatalf("durable action was refused: %v", err)
			}
		})
	}
	for _, action := range []string{"", "REFINE", "UNKNOWN"} {
		t.Run("refuse-"+action, func(t *testing.T) {
			digest, canonical := decisionBody(t, predecessor, action)
			if _, err := IssueRuling(digest, predecessor, action, canonical); err == nil {
				t.Fatal("nondurable action received ruling publication authority")
			}
		})
	}
}

func TestChoicepointAndRulingAuthoritiesOwnBytesAndRejectSealTampering(t *testing.T) {
	predecessor := publicationTestDigest(t, "predecessor-a")
	t.Run("choicepoint", func(t *testing.T) {
		digest, source := choicepointBody(t, predecessor)
		want := append([]byte(nil), source...)
		authority, err := IssueChoicepoint(digest, predecessor, source)
		if err != nil {
			t.Fatal(err)
		}
		source[0] ^= 0xff
		returned := authority.CanonicalBytes()
		returned[0] ^= 0xff
		if !authority.Valid() || !bytes.Equal(authority.CanonicalBytes(), want) {
			t.Fatal("Choicepoint authority did not retain immutable canonical bytes")
		}
		authority.seal.marker = 0
		if authority.Valid() {
			t.Fatal("Choicepoint authority with a mutated seal remained valid")
		}
	})

	t.Run("ruling", func(t *testing.T) {
		digest, source := decisionBody(t, predecessor, "DEFER")
		want := append([]byte(nil), source...)
		authority, err := IssueRuling(digest, predecessor, "DEFER", source)
		if err != nil {
			t.Fatal(err)
		}
		source[0] ^= 0xff
		returned := authority.CanonicalBytes()
		returned[0] ^= 0xff
		if !authority.Valid() || !bytes.Equal(authority.CanonicalBytes(), want) {
			t.Fatal("Ruling authority did not retain immutable canonical bytes")
		}
		authority.seal = nil
		if authority.Valid() {
			t.Fatal("Ruling authority with a nil seal remained valid")
		}
	})
}

func choicepointBody(t testing.TB, predecessor domain.Digest) (domain.Digest, []byte) {
	t.Helper()
	return publicationBody(t, "Choicepoint", struct {
		Kind                    string `json:"kind"`
		FreshConfirmationDigest string `json:"fresh_confirmation_digest"`
	}{"Choicepoint", predecessor.String()})
}

func decisionBody(t testing.TB, predecessor domain.Digest, action string) (domain.Digest, []byte) {
	t.Helper()
	return publicationBody(t, "DecisionRecord", struct {
		Action            string `json:"action"`
		ChoicepointDigest string `json:"choicepoint_digest"`
		Kind              string `json:"kind"`
	}{action, predecessor.String(), "DecisionRecord"})
}

func publicationBody(t testing.TB, kind string, identity any) (domain.Digest, []byte) {
	t.Helper()
	raw, canonical, err := canon.DigestTyped(kind, identity)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(raw.String())
	if err != nil {
		t.Fatal(err)
	}
	return digest, canonical
}

func publicationTestDigest(t testing.TB, label string) domain.Digest {
	t.Helper()
	raw, _, err := canon.DigestTyped("PublicationTestPredecessor", struct {
		Label string `json:"label"`
	}{label})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(raw.String())
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
