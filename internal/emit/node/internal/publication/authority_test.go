package publication

import (
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/compiler"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

func TestAuthorityBindsEveryTerminalPublicationJoin(t *testing.T) {
	bundle := publicationTestBundle(t)
	study := "study:" + strings.Repeat("a", 64)
	head := publicationTestDigest("b")
	lineage := bundle.PortableSource().Plan().Digest()
	authority, err := Issue(bundle, study, head, bundle.DecisionRecordDigest(), lineage)
	if err != nil || !authority.Valid() || authority.StudyID() != study ||
		authority.ExpectedHeadDigest() != head || authority.Digest() != bundle.Digest() ||
		authority.Predecessor() != bundle.DecisionRecordDigest() ||
		authority.Choicepoint() != bundle.ChoicepointDigest() || authority.LineageRoot() != lineage {
		t.Fatalf("exact publication authority was not issued: %#v, %v", authority, err)
	}
	canonical := authority.CanonicalBytes()
	canonical[0] ^= 0xff
	if !authority.Valid() || string(canonical) == string(authority.CanonicalBytes()) {
		t.Fatal("publication authority canonical getter was not defensive")
	}
	copyAuthority, err := Issue(bundle, study, head, bundle.DecisionRecordDigest(), lineage)
	if err != nil || !authority.Equal(copyAuthority) {
		t.Fatalf("equal exact publication authority differs: %v", err)
	}

	other := publicationTestDigest("e")
	contextMutations := []struct {
		name   string
		mutate func(*Authority)
	}{
		{"study", func(value *Authority) { value.study = "study:" + strings.Repeat("f", 64) }},
		{"expected-head", func(value *Authority) { value.expectedHead = other }},
	}
	for _, mutation := range contextMutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := authority
			changed.canonical = append([]byte(nil), authority.canonical...)
			mutation.mutate(&changed)
			if !changed.Valid() || changed.Equal(authority) {
				t.Fatalf("changed %s binding was not distinguished", mutation.name)
			}
		})
	}
	bodyMutations := []struct {
		name   string
		mutate func(*Authority)
	}{
		{"bundle-digest", func(value *Authority) { value.digest = other }},
		{"decision-predecessor", func(value *Authority) { value.predecessor = other }},
		{"choicepoint", func(value *Authority) { value.choicepoint = other }},
		{"lineage-root", func(value *Authority) { value.lineageRoot = other }},
		{"canonical-body", func(value *Authority) { value.canonical[0] ^= 0xff }},
		{"seal", func(value *Authority) { value.seal = nil }},
	}
	for _, mutation := range bodyMutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := authority
			changed.canonical = append([]byte(nil), authority.canonical...)
			mutation.mutate(&changed)
			if changed.Valid() || changed.Equal(authority) {
				t.Fatalf("mutated %s authority remained valid", mutation.name)
			}
		})
	}
}

func TestIssueRejectsIncompleteTerminalPublicationJoins(t *testing.T) {
	bundle := publicationTestBundle(t)
	study := "study:" + strings.Repeat("a", 64)
	head := publicationTestDigest("b")
	predecessor := bundle.DecisionRecordDigest()
	lineage := bundle.PortableSource().Plan().Digest()
	for _, test := range []struct {
		name        string
		bundle      model.ContractBundle
		study       string
		head        domain.Digest
		predecessor domain.Digest
		lineage     domain.Digest
	}{
		{name: "zero-bundle", study: study, head: head, predecessor: predecessor, lineage: lineage},
		{name: "malformed-study", bundle: bundle, study: "study:invalid", head: head, predecessor: predecessor, lineage: lineage},
		{name: "zero-head", bundle: bundle, study: study, predecessor: predecessor, lineage: lineage},
		{name: "wrong-predecessor", bundle: bundle, study: study, head: head, predecessor: publicationTestDigest("e"), lineage: lineage},
		{name: "wrong-lineage", bundle: bundle, study: study, head: head, predecessor: predecessor, lineage: publicationTestDigest("e")},
	} {
		t.Run(test.name, func(t *testing.T) {
			authority, err := Issue(test.bundle, test.study, test.head, test.predecessor, test.lineage)
			if err == nil || authority.Valid() || authority.StudyID() != "" || authority.ExpectedHeadDigest().Valid() ||
				authority.Digest().Valid() || authority.Predecessor().Valid() || authority.Choicepoint().Valid() ||
				authority.LineageRoot().Valid() || len(authority.CanonicalBytes()) != 0 {
				t.Fatalf("incomplete publication join issued authority: %#v, %v", authority, err)
			}
		})
	}
}

func publicationTestBundle(t testing.TB) model.ContractBundle {
	t.Helper()
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		t.Fatal(err)
	}
	value, err := portablevalue.Bytes([]byte("ok\n"))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := model.NewExactValue(value)
	if err != nil {
		t.Fatal(err)
	}
	field, err := model.NewExactField("cli.stdout.bytes", exact)
	if err != nil {
		t.Fatal(err)
	}
	tuple, err := model.NewExactTuple([]model.ExactField{field})
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := model.NewPredicate(
		source.StimulusDigest(), source.Profile(), []string{"cli.stdout.bytes"}, []model.ExactTuple{tuple},
	)
	if err != nil {
		t.Fatal(err)
	}
	input, err := compilation.New(
		publicationTestDigest("d"), publicationTestDigest("c"), compilation.ActionAllowObserved,
		source, profile, predicate,
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := compiler.Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func publicationTestDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}
