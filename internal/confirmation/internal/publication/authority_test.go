package publication

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestIssueBindsFreshConfirmationBodyToExactReductionPredecessor(t *testing.T) {
	predecessor := publicationTestDigest(t, "reduction-a")
	other := publicationTestDigest(t, "reduction-b")
	digest, canonical := freshConfirmationBody(t, predecessor)

	authority, err := Issue(digest, predecessor, canonical)
	if err != nil || !authority.Valid() {
		t.Fatalf("exact body authority was refused: %v", err)
	}
	if _, err := Issue(digest, other, canonical); err == nil {
		t.Fatal("FreshConfirmation body was sealed to a detached reduction predecessor")
	}
	authority.predecessor = other
	if authority.Valid() {
		t.Fatal("mutated FreshConfirmation predecessor remained valid")
	}
}

func TestIssueRejectsWrongKindAndNoncanonicalFreshConfirmationBytes(t *testing.T) {
	predecessor := publicationTestDigest(t, "reduction-a")
	wrongKindDigest, wrongKind := publicationBody(t, "FreshConfirmation", struct {
		Kind               string `json:"kind"`
		ReductionRunDigest string `json:"reduction_run_digest"`
	}{"Choicepoint", predecessor.String()})
	if _, err := Issue(wrongKindDigest, predecessor, wrongKind); err == nil {
		t.Fatal("wrong-kind body received FreshConfirmation publication authority")
	}

	_, canonical := freshConfirmationBody(t, predecessor)
	noncanonical := append([]byte(" "), canonical...)
	raw, err := canon.DigestBytes("FreshConfirmation", noncanonical)
	if err != nil {
		t.Fatal(err)
	}
	noncanonicalDigest, err := domain.ParseDigest(raw.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Issue(noncanonicalDigest, predecessor, noncanonical); err == nil {
		t.Fatal("noncanonical bytes received FreshConfirmation publication authority")
	}
}

func TestAuthorityOwnsCanonicalBytesAndRejectsPrivateFieldTampering(t *testing.T) {
	predecessor := publicationTestDigest(t, "reduction-a")
	other := publicationTestDigest(t, "reduction-b")
	digest, source := freshConfirmationBody(t, predecessor)
	want := append([]byte(nil), source...)
	authority, err := Issue(digest, predecessor, source)
	if err != nil {
		t.Fatal(err)
	}
	source[0] ^= 0xff
	returned := authority.CanonicalBytes()
	returned[0] ^= 0xff
	if !authority.Valid() || !bytes.Equal(authority.CanonicalBytes(), want) {
		t.Fatal("publication authority did not retain immutable canonical bytes")
	}

	attacks := []struct {
		name   string
		mutate func(*Authority)
	}{
		{"digest", func(candidate *Authority) { candidate.digest = domain.Digest("") }},
		{"predecessor", func(candidate *Authority) { candidate.predecessor = other }},
		{"canonical", func(candidate *Authority) { candidate.canonical[0] ^= 0xff }},
		{"nil seal", func(candidate *Authority) { candidate.seal = nil }},
		{"seal marker", func(candidate *Authority) { candidate.seal.marker = 0 }},
	}
	for _, attack := range attacks {
		t.Run(attack.name, func(t *testing.T) {
			candidate, issueErr := Issue(digest, predecessor, want)
			if issueErr != nil {
				t.Fatal(issueErr)
			}
			attack.mutate(&candidate)
			if candidate.Valid() {
				t.Fatal("tampered FreshConfirmation authority remained valid")
			}
		})
	}
}

func freshConfirmationBody(t testing.TB, predecessor domain.Digest) (domain.Digest, []byte) {
	t.Helper()
	return publicationBody(t, "FreshConfirmation", struct {
		Kind               string `json:"kind"`
		ReductionRunDigest string `json:"reduction_run_digest"`
	}{"FreshConfirmation", predecessor.String()})
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
